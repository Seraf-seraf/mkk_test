package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// CommentRecord описывает комментарий.
type CommentRecord struct {
	ID        uuid.UUID
	TaskID    uuid.UUID
	UserID    uuid.UUID
	Body      string
	CreatedAt time.Time
}

// CommentsRepo реализует доступ к комментариям.
type CommentsRepo struct {
	db *sql.DB
}

// NewCommentsRepo создает репозиторий комментариев.
func NewCommentsRepo(db *sql.DB) *CommentsRepo {
	const methodCtx = "repo.NewCommentsRepo"

	slog.Debug("инициализация репозитория комментариев", slog.String("context", methodCtx))

	return &CommentsRepo{db: db}
}

// Create создает комментарий.
func (r *CommentsRepo) Create(ctx context.Context, record CommentRecord) error {
	const methodCtx = "repo.CommentsRepo.Create"

	if r == nil || r.db == nil {
		return fmt.Errorf("%s: репозиторий не инициализирован", methodCtx)
	}

	_, err := r.db.ExecContext(
		ctx,
		"INSERT INTO task_comments (id, task_id, user_id, body, created_at) VALUES (?, ?, ?, ?, ?)",
		record.ID.String(),
		record.TaskID.String(),
		record.UserID.String(),
		record.Body,
		record.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", methodCtx, err)
	}
	return nil
}

// List возвращает комментарии задачи.
func (r *CommentsRepo) List(ctx context.Context, taskID uuid.UUID, page int, perPage int) ([]CommentRecord, error) {
	const methodCtx = "repo.CommentsRepo.List"

	if r == nil || r.db == nil {
		return nil, fmt.Errorf("%s: репозиторий не инициализирован", methodCtx)
	}

	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id, task_id, user_id, body, created_at
		FROM task_comments
		WHERE task_id = ?
		ORDER BY created_at ASC
		LIMIT ? OFFSET ?`,
		taskID.String(),
		perPage,
		(page-1)*perPage,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", methodCtx, err)
	}
	defer rows.Close()

	var items []CommentRecord
	for rows.Next() {
		var idStr, taskIDStr, userIDStr string
		var body string
		var createdAt time.Time
		if err := rows.Scan(&idStr, &taskIDStr, &userIDStr, &body, &createdAt); err != nil {
			return nil, fmt.Errorf("%s: %w", methodCtx, err)
		}
		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, fmt.Errorf("%s: некорректный id комментария", methodCtx)
		}
		taskUUID, err := uuid.Parse(taskIDStr)
		if err != nil {
			return nil, fmt.Errorf("%s: некорректный task_id", methodCtx)
		}
		userUUID, err := uuid.Parse(userIDStr)
		if err != nil {
			return nil, fmt.Errorf("%s: некорректный user_id", methodCtx)
		}
		items = append(items, CommentRecord{
			ID:        id,
			TaskID:    taskUUID,
			UserID:    userUUID,
			Body:      body,
			CreatedAt: createdAt,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", methodCtx, err)
	}

	return items, nil
}

// Count возвращает количество комментариев задачи.
func (r *CommentsRepo) Count(ctx context.Context, taskID uuid.UUID) (int, error) {
	const methodCtx = "repo.CommentsRepo.Count"

	if r == nil || r.db == nil {
		return 0, fmt.Errorf("%s: репозиторий не инициализирован", methodCtx)
	}

	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM task_comments WHERE task_id = ?", taskID.String()).Scan(&total); err != nil {
		return 0, fmt.Errorf("%s: %w", methodCtx, err)
	}
	return total, nil
}

// Get возвращает комментарий по id.
func (r *CommentsRepo) Get(ctx context.Context, taskID uuid.UUID, commentID uuid.UUID) (CommentRecord, error) {
	const methodCtx = "repo.CommentsRepo.Get"

	if r == nil || r.db == nil {
		return CommentRecord{}, fmt.Errorf("%s: репозиторий не инициализирован", methodCtx)
	}

	var record CommentRecord
	var idStr, taskIDStr, userIDStr string
	if err := r.db.QueryRowContext(
		ctx,
		"SELECT id, task_id, user_id, body, created_at FROM task_comments WHERE id = ? AND task_id = ?",
		commentID.String(),
		taskID.String(),
	).Scan(&idStr, &taskIDStr, &userIDStr, &record.Body, &record.CreatedAt); err != nil {
		return CommentRecord{}, err
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return CommentRecord{}, fmt.Errorf("%s: некорректный id комментария", methodCtx)
	}
	taskUUID, err := uuid.Parse(taskIDStr)
	if err != nil {
		return CommentRecord{}, fmt.Errorf("%s: некорректный task_id", methodCtx)
	}
	userUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		return CommentRecord{}, fmt.Errorf("%s: некорректный user_id", methodCtx)
	}

	record.ID = id
	record.TaskID = taskUUID
	record.UserID = userUUID

	return record, nil
}

// Update обновляет текст комментария.
func (r *CommentsRepo) Update(ctx context.Context, commentID uuid.UUID, body string) error {
	const methodCtx = "repo.CommentsRepo.Update"

	if r == nil || r.db == nil {
		return fmt.Errorf("%s: репозиторий не инициализирован", methodCtx)
	}

	_, err := r.db.ExecContext(ctx, "UPDATE task_comments SET body = ? WHERE id = ?", body, commentID.String())
	if err != nil {
		return fmt.Errorf("%s: %w", methodCtx, err)
	}
	return nil
}

// Delete удаляет комментарий.
func (r *CommentsRepo) Delete(ctx context.Context, commentID uuid.UUID) error {
	const methodCtx = "repo.CommentsRepo.Delete"

	if r == nil || r.db == nil {
		return fmt.Errorf("%s: репозиторий не инициализирован", methodCtx)
	}

	_, err := r.db.ExecContext(ctx, "DELETE FROM task_comments WHERE id = ?", commentID.String())
	if err != nil {
		return fmt.Errorf("%s: %w", methodCtx, err)
	}
	return nil
}
