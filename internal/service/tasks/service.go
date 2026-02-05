package tasks

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Seraf-seraf/mkk_test/internal/api"
	repomysql "github.com/Seraf-seraf/mkk_test/internal/repo/mysql"
)

// Service реализует бизнес-логику задач.
type Service struct {
	db      *sql.DB
	tasks   TasksRepository
	members MembersRepository
	history HistoryRepository
	cache   Cache
}

// TasksRepository описывает работу с задачами.
type TasksRepository interface {
	Create(ctx context.Context, exec repomysql.DBTX, record repomysql.TaskRecord) error
	List(ctx context.Context, filter repomysql.TaskFilter) ([]repomysql.TaskRecord, error)
	Count(ctx context.Context, filter repomysql.TaskFilter) (int, error)
	GetForUpdate(ctx context.Context, tx *sql.Tx, taskID uuid.UUID) (repomysql.TaskRecord, error)
	Update(ctx context.Context, tx *sql.Tx, record repomysql.TaskRecord) error
	GetTeamID(ctx context.Context, taskID uuid.UUID) (uuid.UUID, error)
}

// MembersRepository описывает доступ к участникам команды.
type MembersRepository interface {
	IsMember(ctx context.Context, teamID uuid.UUID, userID uuid.UUID) (bool, error)
}

// HistoryRepository описывает доступ к истории задач.
type HistoryRepository interface {
	Add(ctx context.Context, exec repomysql.DBTX, record repomysql.TaskHistoryRecord) error
	ListByTask(ctx context.Context, taskID uuid.UUID) ([]repomysql.TaskHistoryRecord, error)
}

// Cache описывает кэширование задач.
type Cache interface {
	GetTeamTasks(ctx context.Context, teamID uuid.UUID, key string) ([]api.Task, bool, error)
	SetTeamTasks(ctx context.Context, teamID uuid.UUID, key string, tasks []api.Task) error
}

// NewService создает сервис задач.
func NewService(db *sql.DB, tasks TasksRepository, members MembersRepository, history HistoryRepository, cache Cache) (*Service, error) {
	const methodCtx = "tasks.NewService"

	slog.Debug("инициализация сервиса задач", slog.String("context", methodCtx))

	if db == nil {
		return nil, fmt.Errorf("%s: db не задан", methodCtx)
	}
	if tasks == nil {
		return nil, fmt.Errorf("%s: tasks repo не задан", methodCtx)
	}
	if members == nil {
		return nil, fmt.Errorf("%s: members repo не задан", methodCtx)
	}
	if history == nil {
		return nil, fmt.Errorf("%s: history repo не задан", methodCtx)
	}

	return &Service{db: db, tasks: tasks, members: members, history: history, cache: cache}, nil
}

// Create создает задачу.
func (s *Service) Create(ctx context.Context, userID uuid.UUID, req api.CreateTaskRequest) (api.Task, error) {
	const methodCtx = "tasks.Service.Create"

	slog.Debug("вызов создания задачи", slog.String("context", methodCtx))

	if strings.TrimSpace(req.Title) == "" {
		return api.Task{}, fmt.Errorf("%s: заголовок не задан", methodCtx)
	}

	member, err := s.members.IsMember(ctx, req.TeamId, userID)
	if err != nil {
		return api.Task{}, fmt.Errorf("%s: %w", methodCtx, err)
	}
	if !member {
		return api.Task{}, fmt.Errorf("%s: %w", methodCtx, ErrForbidden)
	}

	var assigneePtr *uuid.UUID
	if req.AssigneeId != nil {
		assigneeID := *req.AssigneeId
		ok, err := s.members.IsMember(ctx, req.TeamId, assigneeID)
		if err != nil {
			return api.Task{}, fmt.Errorf("%s: %w", methodCtx, err)
		}
		if !ok {
			return api.Task{}, fmt.Errorf("%s: %w", methodCtx, ErrInvalidAssignee)
		}
		assigneePtr = &assigneeID
	}

	status := api.TaskStatus("todo")
	if req.Status != nil {
		status = *req.Status
	}

	now := time.Now().UTC()
	taskID := uuid.New()

	var completedAt *time.Time
	if status == "done" {
		completedAt = &now
	}

	record := repomysql.TaskRecord{
		ID:          taskID,
		TeamID:      req.TeamId,
		Title:       req.Title,
		Description: req.Description,
		Status:      string(status),
		AssigneeID:  assigneePtr,
		CreatedBy:   userID,
		CreatedAt:   now,
		UpdatedAt:   &now,
		CompletedAt: completedAt,
	}

	if err := s.tasks.Create(ctx, nil, record); err != nil {
		return api.Task{}, fmt.Errorf("%s: %w", methodCtx, err)
	}

	return taskToAPI(record), nil
}

// List возвращает список задач с фильтрами и пагинацией.
func (s *Service) List(ctx context.Context, userID uuid.UUID, teamID uuid.UUID, status *api.TaskStatus, assigneeID *uuid.UUID, page int, perPage int) (api.TasksListResponse, error) {
	const methodCtx = "tasks.Service.List"

	slog.Debug("вызов списка задач", slog.String("context", methodCtx))

	member, err := s.members.IsMember(ctx, teamID, userID)
	if err != nil {
		return api.TasksListResponse{}, fmt.Errorf("%s: %w", methodCtx, err)
	}
	if !member {
		return api.TasksListResponse{}, fmt.Errorf("%s: %w", methodCtx, ErrForbidden)
	}

	page, perPage = normalizePagination(page, perPage)
	cacheKey := buildCacheKey(teamID, status, assigneeID, page, perPage)

	if s.cache != nil {
		items, hit, err := s.cache.GetTeamTasks(ctx, teamID, cacheKey)
		if err == nil && hit {
			total, err := s.tasks.Count(ctx, buildFilter(teamID, status, assigneeID, page, perPage))
			if err != nil {
				return api.TasksListResponse{}, fmt.Errorf("%s: %w", methodCtx, err)
			}
			return api.TasksListResponse{Items: items, Page: page, PerPage: perPage, Total: total}, nil
		}
	}

	filter := buildFilter(teamID, status, assigneeID, page, perPage)
	records, err := s.tasks.List(ctx, filter)
	if err != nil {
		return api.TasksListResponse{}, fmt.Errorf("%s: %w", methodCtx, err)
	}

	items := make([]api.Task, 0, len(records))
	for _, record := range records {
		items = append(items, taskToAPI(record))
	}

	total, err := s.tasks.Count(ctx, filter)
	if err != nil {
		return api.TasksListResponse{}, fmt.Errorf("%s: %w", methodCtx, err)
	}

	if s.cache != nil {
		_ = s.cache.SetTeamTasks(ctx, teamID, cacheKey, items)
	}

	return api.TasksListResponse{
		Items:   items,
		Page:    page,
		PerPage: perPage,
		Total:   total,
	}, nil
}

// Update обновляет задачу.
func (s *Service) Update(ctx context.Context, userID uuid.UUID, taskID uuid.UUID, req api.UpdateTaskRequest) (api.Task, error) {
	const methodCtx = "tasks.Service.Update"

	slog.Debug("вызов обновления задачи", slog.String("context", methodCtx))

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return api.Task{}, fmt.Errorf("%s: %w", methodCtx, err)
	}
	defer func() { _ = tx.Rollback() }()

	current, err := s.tasks.GetForUpdate(ctx, tx, taskID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return api.Task{}, fmt.Errorf("%s: %w", methodCtx, ErrNotFound)
		}
		return api.Task{}, fmt.Errorf("%s: %w", methodCtx, err)
	}

	if current.CreatedBy != userID {
		return api.Task{}, fmt.Errorf("%s: %w", methodCtx, ErrForbidden)
	}

	newTitle := current.Title
	if req.Title != nil {
		newTitle = *req.Title
	}

	newDescription := current.Description
	if req.Description != nil {
		newDescription = req.Description
	}

	newStatus := api.TaskStatus(current.Status)
	if req.Status != nil {
		newStatus = *req.Status
	}

	newAssignee := current.AssigneeID
	if req.AssigneeId != nil {
		assigneeUUID := *req.AssigneeId
		ok, err := s.members.IsMember(ctx, current.TeamID, assigneeUUID)
		if err != nil {
			return api.Task{}, fmt.Errorf("%s: %w", methodCtx, err)
		}
		if !ok {
			return api.Task{}, fmt.Errorf("%s: %w", methodCtx, ErrInvalidAssignee)
		}
		newAssignee = &assigneeUUID
	}

	now := time.Now().UTC()
	var completedAt *time.Time
	if newStatus == "done" {
		completedAt = &now
	}

	changes := map[string]interface{}{}
	if newTitle != current.Title {
		changes["title"] = map[string]interface{}{"from": current.Title, "to": newTitle}
	}
	if !stringPtrEqual(newDescription, current.Description) {
		changes["description"] = map[string]interface{}{"from": current.Description, "to": newDescription}
	}
	if newStatus != api.TaskStatus(current.Status) {
		changes["status"] = map[string]interface{}{"from": current.Status, "to": newStatus}
	}
	if !uuidPtrEqual(newAssignee, current.AssigneeID) {
		changes["assignee_id"] = map[string]interface{}{"from": current.AssigneeID, "to": newAssignee}
	}

	current.Title = newTitle
	current.Description = newDescription
	current.Status = string(newStatus)
	current.AssigneeID = newAssignee
	current.UpdatedAt = &now
	current.CompletedAt = completedAt

	if err := s.tasks.Update(ctx, tx, current); err != nil {
		return api.Task{}, fmt.Errorf("%s: %w", methodCtx, err)
	}

	if err := s.history.Add(ctx, tx, repomysql.TaskHistoryRecord{
		ID:        uuid.New(),
		TaskID:    current.ID,
		ChangedBy: userID,
		Changes:   changes,
		ChangedAt: now,
	}); err != nil {
		return api.Task{}, fmt.Errorf("%s: %w", methodCtx, err)
	}

	if err := tx.Commit(); err != nil {
		return api.Task{}, fmt.Errorf("%s: %w", methodCtx, err)
	}

	return taskToAPI(current), nil
}

// History возвращает историю изменений задачи.
func (s *Service) History(ctx context.Context, userID uuid.UUID, taskID uuid.UUID) (api.TaskHistoryListResponse, error) {
	const methodCtx = "tasks.Service.History"

	slog.Debug("вызов истории задачи", slog.String("context", methodCtx))

	teamID, err := s.tasks.GetTeamID(ctx, taskID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return api.TaskHistoryListResponse{}, fmt.Errorf("%s: %w", methodCtx, ErrNotFound)
		}
		return api.TaskHistoryListResponse{}, fmt.Errorf("%s: %w", methodCtx, err)
	}

	member, err := s.members.IsMember(ctx, teamID, userID)
	if err != nil {
		return api.TaskHistoryListResponse{}, fmt.Errorf("%s: %w", methodCtx, err)
	}
	if !member {
		return api.TaskHistoryListResponse{}, fmt.Errorf("%s: %w", methodCtx, ErrForbidden)
	}

	records, err := s.history.ListByTask(ctx, taskID)
	if err != nil {
		return api.TaskHistoryListResponse{}, fmt.Errorf("%s: %w", methodCtx, err)
	}

	items := make([]api.TaskHistory, 0, len(records))
	for _, record := range records {
		items = append(items, api.TaskHistory{
			Id:        api.UUID(record.ID),
			TaskId:    api.UUID(record.TaskID),
			ChangedBy: api.UUID(record.ChangedBy),
			Changes:   record.Changes,
			ChangedAt: record.ChangedAt,
		})
	}

	return api.TaskHistoryListResponse{Items: items}, nil
}

func buildFilter(teamID uuid.UUID, status *api.TaskStatus, assigneeID *uuid.UUID, page int, perPage int) repomysql.TaskFilter {
	var statusPtr *string
	if status != nil {
		value := string(*status)
		statusPtr = &value
	}

	var assigneePtr *uuid.UUID
	if assigneeID != nil {
		value := *assigneeID
		assigneePtr = &value
	}

	return repomysql.TaskFilter{
		TeamID:     teamID,
		Status:     statusPtr,
		AssigneeID: assigneePtr,
		Page:       page,
		PerPage:    perPage,
	}
}

func taskToAPI(record repomysql.TaskRecord) api.Task {
	return api.Task{
		Id:          record.ID,
		TeamId:      record.TeamID,
		Title:       record.Title,
		Description: record.Description,
		Status:      api.TaskStatus(record.Status),
		AssigneeId:  toAPUUIDPtr(record.AssigneeID),
		CreatedBy:   record.CreatedBy,
		CreatedAt:   record.CreatedAt,
		UpdatedAt:   record.UpdatedAt,
		CompletedAt: record.CompletedAt,
	}
}

func toAPUUIDPtr(id *uuid.UUID) *api.UUID {
	if id == nil {
		return nil
	}
	tmp := api.UUID(*id)
	return &tmp
}

func normalizePagination(page int, perPage int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}
	return page, perPage
}

func buildCacheKey(teamID uuid.UUID, status *api.TaskStatus, assigneeID *uuid.UUID, page int, perPage int) string {
	statusValue := ""
	if status != nil {
		statusValue = string(*status)
	}
	assigneeValue := ""
	if assigneeID != nil {
		assigneeValue = assigneeID.String()
	}
	return fmt.Sprintf("tasks:%s:status=%s:assignee=%s:page=%d:per=%d",
		teamID.String(),
		statusValue,
		assigneeValue,
		page,
		perPage,
	)
}

func stringPtrEqual(a *string, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func uuidPtrEqual(a *uuid.UUID, b *uuid.UUID) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}
