package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// TaskRecord описывает запись задачи.
type TaskRecord struct {
	ID          uuid.UUID
	TeamID      uuid.UUID
	Title       string
	Description *string
	Status      string
	AssigneeID  *uuid.UUID
	CreatedBy   uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   *time.Time
	CompletedAt *time.Time
}

// TaskFilter описывает фильтры списка задач.
type TaskFilter struct {
	TeamID     uuid.UUID
	Status     *string
	AssigneeID *uuid.UUID
	Page       int
	PerPage    int
}

// TasksRepo реализует доступ к задачам.
type TasksRepo struct {
	db *sql.DB
}

// NewTasksRepo создает репозиторий задач.
func NewTasksRepo(db *sql.DB) *TasksRepo {
	const methodCtx = "repo.NewTasksRepo"

	slog.Debug("инициализация репозитория задач", slog.String("context", methodCtx))

	return &TasksRepo{db: db}
}

// Create создает задачу.
func (r *TasksRepo) Create(ctx context.Context, exec DBTX, record TaskRecord) error {
	const methodCtx = "repo.TasksRepo.Create"

	if r == nil || r.db == nil {
		return fmt.Errorf("%s: репозиторий не инициализирован", methodCtx)
	}
	if exec == nil {
		exec = r.db
	}

	var descValue interface{}
	if record.Description != nil {
		descValue = *record.Description
	}

	var assigneeValue interface{}
	if record.AssigneeID != nil {
		assigneeValue = record.AssigneeID.String()
	}

	var updatedValue interface{}
	if record.UpdatedAt != nil {
		updatedValue = *record.UpdatedAt
	}

	var completedValue interface{}
	if record.CompletedAt != nil {
		completedValue = *record.CompletedAt
	}

	_, err := exec.ExecContext(
		ctx,
		`INSERT INTO tasks (id, team_id, title, description, status, assignee_id, created_by, created_at, updated_at, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.ID.String(),
		record.TeamID.String(),
		record.Title,
		descValue,
		record.Status,
		assigneeValue,
		record.CreatedBy.String(),
		record.CreatedAt,
		updatedValue,
		completedValue,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", methodCtx, err)
	}
	return nil
}

// List возвращает список задач по фильтрам.
func (r *TasksRepo) List(ctx context.Context, filter TaskFilter) ([]TaskRecord, error) {
	const methodCtx = "repo.TasksRepo.List"

	if r == nil || r.db == nil {
		return nil, fmt.Errorf("%s: репозиторий не инициализирован", methodCtx)
	}

	query := `SELECT id, team_id, title, description, status, assignee_id, created_by, created_at, updated_at, completed_at
		FROM tasks WHERE team_id = ?`
	args := []interface{}{filter.TeamID.String()}

	if filter.Status != nil {
		query += " AND status = ?"
		args = append(args, *filter.Status)
	}
	if filter.AssigneeID != nil {
		query += " AND assignee_id = ?"
		args = append(args, filter.AssigneeID.String())
	}

	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, filter.PerPage, (filter.Page-1)*filter.PerPage)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", methodCtx, err)
	}
	defer rows.Close()

	var items []TaskRecord
	for rows.Next() {
		record, err := scanTaskRecord(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", methodCtx, err)
		}
		items = append(items, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", methodCtx, err)
	}

	return items, nil
}

// Count возвращает количество задач по фильтрам.
func (r *TasksRepo) Count(ctx context.Context, filter TaskFilter) (int, error) {
	const methodCtx = "repo.TasksRepo.Count"

	if r == nil || r.db == nil {
		return 0, fmt.Errorf("%s: репозиторий не инициализирован", methodCtx)
	}

	query := "SELECT COUNT(*) FROM tasks WHERE team_id = ?"
	args := []interface{}{filter.TeamID.String()}

	if filter.Status != nil {
		query += " AND status = ?"
		args = append(args, *filter.Status)
	}
	if filter.AssigneeID != nil {
		query += " AND assignee_id = ?"
		args = append(args, filter.AssigneeID.String())
	}

	var total int
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("%s: %w", methodCtx, err)
	}
	return total, nil
}

// GetForUpdate возвращает задачу для обновления с блокировкой.
func (r *TasksRepo) GetForUpdate(ctx context.Context, tx *sql.Tx, taskID uuid.UUID) (TaskRecord, error) {
	const methodCtx = "repo.TasksRepo.GetForUpdate"

	if r == nil || r.db == nil {
		return TaskRecord{}, fmt.Errorf("%s: репозиторий не инициализирован", methodCtx)
	}
	if tx == nil {
		return TaskRecord{}, fmt.Errorf("%s: транзакция не задана", methodCtx)
	}

	row := tx.QueryRowContext(
		ctx,
		"SELECT id, team_id, title, description, status, assignee_id, created_by, created_at, updated_at, completed_at FROM tasks WHERE id = ? FOR UPDATE",
		taskID.String(),
	)

	return scanTaskRecord(row)
}

// Update обновляет задачу.
func (r *TasksRepo) Update(ctx context.Context, tx *sql.Tx, record TaskRecord) error {
	const methodCtx = "repo.TasksRepo.Update"

	if r == nil || r.db == nil {
		return fmt.Errorf("%s: репозиторий не инициализирован", methodCtx)
	}
	if tx == nil {
		return fmt.Errorf("%s: транзакция не задана", methodCtx)
	}

	var descValue interface{}
	if record.Description != nil {
		descValue = *record.Description
	}

	var assigneeValue interface{}
	if record.AssigneeID != nil {
		assigneeValue = record.AssigneeID.String()
	}

	var completedValue interface{}
	if record.CompletedAt != nil {
		completedValue = *record.CompletedAt
	}

	_, err := tx.ExecContext(
		ctx,
		`UPDATE tasks
		SET title = ?, description = ?, status = ?, assignee_id = ?, updated_at = ?, completed_at = ?
		WHERE id = ?`,
		record.Title,
		descValue,
		record.Status,
		assigneeValue,
		record.UpdatedAt,
		completedValue,
		record.ID.String(),
	)
	if err != nil {
		return fmt.Errorf("%s: %w", methodCtx, err)
	}
	return nil
}

// GetTeamID возвращает team_id задачи.
func (r *TasksRepo) GetTeamID(ctx context.Context, taskID uuid.UUID) (uuid.UUID, error) {
	const methodCtx = "repo.TasksRepo.GetTeamID"

	if r == nil || r.db == nil {
		return uuid.UUID{}, fmt.Errorf("%s: репозиторий не инициализирован", methodCtx)
	}

	var teamIDStr string
	if err := r.db.QueryRowContext(ctx, "SELECT team_id FROM tasks WHERE id = ?", taskID.String()).Scan(&teamIDStr); err != nil {
		return uuid.UUID{}, err
	}

	teamID, err := uuid.Parse(teamIDStr)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("%s: некорректный team_id", methodCtx)
	}
	return teamID, nil
}

func scanTaskRecord(scanner interface{
	Scan(dest ...interface{}) error
}) (TaskRecord, error) {
	var record TaskRecord
	var idStr, teamIDStr, createdByStr string
	var description sql.NullString
	var assignee sql.NullString
	var updatedAt sql.NullTime
	var completedAt sql.NullTime

	if err := scanner.Scan(
		&idStr,
		&teamIDStr,
		&record.Title,
		&description,
		&record.Status,
		&assignee,
		&createdByStr,
		&record.CreatedAt,
		&updatedAt,
		&completedAt,
	); err != nil {
		return TaskRecord{}, err
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return TaskRecord{}, fmt.Errorf("некорректный id задачи")
	}
	teamID, err := uuid.Parse(teamIDStr)
	if err != nil {
		return TaskRecord{}, fmt.Errorf("некорректный id команды")
	}
	createdBy, err := uuid.Parse(createdByStr)
	if err != nil {
		return TaskRecord{}, fmt.Errorf("некорректный created_by")
	}

	record.ID = id
	record.TeamID = teamID
	record.CreatedBy = createdBy

	if description.Valid {
		record.Description = &description.String
	}
	if assignee.Valid {
		assigneeID, err := uuid.Parse(assignee.String)
		if err != nil {
			return TaskRecord{}, fmt.Errorf("некорректный assignee_id")
		}
		record.AssigneeID = &assigneeID
	}
	if updatedAt.Valid {
		record.UpdatedAt = &updatedAt.Time
	}
	if completedAt.Valid {
		record.CompletedAt = &completedAt.Time
	}

	return record, nil
}
