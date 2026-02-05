package comments

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

// Service реализует бизнес-логику комментариев.
type Service struct {
	comments CommentsRepository
	tasks    TasksRepository
	members  MembersRepository
}

// CommentsRepository описывает работу с комментариями.
type CommentsRepository interface {
	Create(ctx context.Context, record repomysql.CommentRecord) error
	List(ctx context.Context, taskID uuid.UUID, page int, perPage int) ([]repomysql.CommentRecord, error)
	Count(ctx context.Context, taskID uuid.UUID) (int, error)
	Get(ctx context.Context, taskID uuid.UUID, commentID uuid.UUID) (repomysql.CommentRecord, error)
	Update(ctx context.Context, commentID uuid.UUID, body string) error
	Delete(ctx context.Context, commentID uuid.UUID) error
}

// TasksRepository описывает доступ к задачам.
type TasksRepository interface {
	GetTeamID(ctx context.Context, taskID uuid.UUID) (uuid.UUID, error)
}

// MembersRepository описывает доступ к участникам команды.
type MembersRepository interface {
	IsMember(ctx context.Context, teamID uuid.UUID, userID uuid.UUID) (bool, error)
}

// NewService создает сервис комментариев.
func NewService(comments CommentsRepository, tasks TasksRepository, members MembersRepository) (*Service, error) {
	const methodCtx = "comments.NewService"

	slog.Debug("инициализация сервиса комментариев", slog.String("context", methodCtx))

	if comments == nil {
		return nil, fmt.Errorf("%s: comments repo не задан", methodCtx)
	}
	if tasks == nil {
		return nil, fmt.Errorf("%s: tasks repo не задан", methodCtx)
	}
	if members == nil {
		return nil, fmt.Errorf("%s: members repo не задан", methodCtx)
	}

	return &Service{comments: comments, tasks: tasks, members: members}, nil
}

// Create добавляет комментарий к задаче.
func (s *Service) Create(ctx context.Context, userID uuid.UUID, taskID uuid.UUID, req api.CreateCommentRequest) (api.Comment, error) {
	const methodCtx = "comments.Service.Create"

	slog.Debug("вызов создания комментария", slog.String("context", methodCtx))

	if strings.TrimSpace(req.Body) == "" {
		return api.Comment{}, fmt.Errorf("%s: текст комментария не задан", methodCtx)
	}

	teamID, err := s.tasks.GetTeamID(ctx, taskID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return api.Comment{}, fmt.Errorf("%s: %w", methodCtx, ErrNotFound)
		}
		return api.Comment{}, fmt.Errorf("%s: %w", methodCtx, err)
	}

	isMember, err := s.members.IsMember(ctx, teamID, userID)
	if err != nil {
		return api.Comment{}, fmt.Errorf("%s: %w", methodCtx, err)
	}
	if !isMember {
		return api.Comment{}, fmt.Errorf("%s: %w", methodCtx, ErrForbidden)
	}

	now := time.Now().UTC()
	commentID := uuid.New()

	if err := s.comments.Create(ctx, repomysql.CommentRecord{
		ID:        commentID,
		TaskID:    taskID,
		UserID:    userID,
		Body:      req.Body,
		CreatedAt: now,
	}); err != nil {
		return api.Comment{}, fmt.Errorf("%s: %w", methodCtx, err)
	}

	return api.Comment{
		Id:        api.UUID(commentID),
		TaskId:    api.UUID(taskID),
		UserId:    api.UUID(userID),
		Body:      req.Body,
		CreatedAt: now,
	}, nil
}

// List возвращает список комментариев задачи.
func (s *Service) List(ctx context.Context, userID uuid.UUID, taskID uuid.UUID, page int, perPage int) (api.CommentsListResponse, error) {
	const methodCtx = "comments.Service.List"

	slog.Debug("вызов списка комментариев", slog.String("context", methodCtx))

	teamID, err := s.tasks.GetTeamID(ctx, taskID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return api.CommentsListResponse{}, fmt.Errorf("%s: %w", methodCtx, ErrNotFound)
		}
		return api.CommentsListResponse{}, fmt.Errorf("%s: %w", methodCtx, err)
	}

	isMember, err := s.members.IsMember(ctx, teamID, userID)
	if err != nil {
		return api.CommentsListResponse{}, fmt.Errorf("%s: %w", methodCtx, err)
	}
	if !isMember {
		return api.CommentsListResponse{}, fmt.Errorf("%s: %w", methodCtx, ErrForbidden)
	}

	page, perPage = normalizePagination(page, perPage)

	records, err := s.comments.List(ctx, taskID, page, perPage)
	if err != nil {
		return api.CommentsListResponse{}, fmt.Errorf("%s: %w", methodCtx, err)
	}

	items := make([]api.Comment, 0, len(records))
	for _, record := range records {
		items = append(items, api.Comment{
			Id:        api.UUID(record.ID),
			TaskId:    api.UUID(record.TaskID),
			UserId:    api.UUID(record.UserID),
			Body:      record.Body,
			CreatedAt: record.CreatedAt,
		})
	}

	total, err := s.comments.Count(ctx, taskID)
	if err != nil {
		return api.CommentsListResponse{}, fmt.Errorf("%s: %w", methodCtx, err)
	}

	return api.CommentsListResponse{Items: items, Page: page, PerPage: perPage, Total: total}, nil
}

// Update обновляет комментарий.
func (s *Service) Update(ctx context.Context, userID uuid.UUID, taskID uuid.UUID, commentID uuid.UUID, req api.UpdateCommentRequest) (api.Comment, error) {
	const methodCtx = "comments.Service.Update"

	slog.Debug("вызов обновления комментария", slog.String("context", methodCtx))

	if strings.TrimSpace(req.Body) == "" {
		return api.Comment{}, fmt.Errorf("%s: текст комментария не задан", methodCtx)
	}

	comment, err := s.comments.Get(ctx, taskID, commentID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return api.Comment{}, fmt.Errorf("%s: %w", methodCtx, ErrNotFound)
		}
		return api.Comment{}, fmt.Errorf("%s: %w", methodCtx, err)
	}

	if comment.UserID != userID {
		return api.Comment{}, fmt.Errorf("%s: %w", methodCtx, ErrForbidden)
	}

	if err := s.comments.Update(ctx, commentID, req.Body); err != nil {
		return api.Comment{}, fmt.Errorf("%s: %w", methodCtx, err)
	}

	comment.Body = req.Body

	return api.Comment{
		Id:        api.UUID(comment.ID),
		TaskId:    api.UUID(taskID),
		UserId:    api.UUID(comment.UserID),
		Body:      comment.Body,
		CreatedAt: comment.CreatedAt,
	}, nil
}

// Delete удаляет комментарий.
func (s *Service) Delete(ctx context.Context, userID uuid.UUID, taskID uuid.UUID, commentID uuid.UUID) error {
	const methodCtx = "comments.Service.Delete"

	slog.Debug("вызов удаления комментария", slog.String("context", methodCtx))

	comment, err := s.comments.Get(ctx, taskID, commentID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%s: %w", methodCtx, ErrNotFound)
		}
		return fmt.Errorf("%s: %w", methodCtx, err)
	}

	if comment.UserID != userID {
		return fmt.Errorf("%s: %w", methodCtx, ErrForbidden)
	}

	if err := s.comments.Delete(ctx, commentID); err != nil {
		return fmt.Errorf("%s: %w", methodCtx, err)
	}

	return nil
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
