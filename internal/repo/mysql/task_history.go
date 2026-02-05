package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// TaskHistoryRecord описывает запись истории.
type TaskHistoryRecord struct {
	ID        uuid.UUID
	TaskID    uuid.UUID
	ChangedBy uuid.UUID
	Changes   map[string]interface{}
	ChangedAt time.Time
}

// TaskHistoryRepo реализует доступ к истории задач.
type TaskHistoryRepo struct {
	db *sql.DB
}

// NewTaskHistoryRepo создает репозиторий истории.
func NewTaskHistoryRepo(db *sql.DB) *TaskHistoryRepo {
	const methodCtx = "repo.NewTaskHistoryRepo"

	slog.Debug("инициализация репозитория истории", slog.String("context", methodCtx))

	return &TaskHistoryRepo{db: db}
}

// Add добавляет запись истории.
func (r *TaskHistoryRepo) Add(ctx context.Context, exec DBTX, record TaskHistoryRecord) error {
	const methodCtx = "repo.TaskHistoryRepo.Add"

	if r == nil || r.db == nil {
		return fmt.Errorf("%s: репозиторий не инициализирован", methodCtx)
	}
	if exec == nil {
		exec = r.db
	}

	payload, err := json.Marshal(record.Changes)
	if err != nil {
		return fmt.Errorf("%s: ошибка сериализации изменений", methodCtx)
	}

	_, err = exec.ExecContext(
		ctx,
		"INSERT INTO task_history (id, task_id, changed_by, changes, changed_at) VALUES (?, ?, ?, ?, ?)",
		record.ID.String(),
		record.TaskID.String(),
		record.ChangedBy.String(),
		payload,
		record.ChangedAt,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", methodCtx, err)
	}
	return nil
}

// ListByTask возвращает историю по задаче.
func (r *TaskHistoryRepo) ListByTask(ctx context.Context, taskID uuid.UUID) ([]TaskHistoryRecord, error) {
	const methodCtx = "repo.TaskHistoryRepo.ListByTask"

	if r == nil || r.db == nil {
		return nil, fmt.Errorf("%s: репозиторий не инициализирован", methodCtx)
	}

	rows, err := r.db.QueryContext(
		ctx,
		"SELECT id, task_id, changed_by, changes, changed_at FROM task_history WHERE task_id = ? ORDER BY changed_at ASC",
		taskID.String(),
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", methodCtx, err)
	}
	defer rows.Close()

	var items []TaskHistoryRecord
	for rows.Next() {
		var idStr, taskIDStr, changedByStr string
		var changesData []byte
		var changedAt time.Time
		if err := rows.Scan(&idStr, &taskIDStr, &changedByStr, &changesData, &changedAt); err != nil {
			return nil, fmt.Errorf("%s: %w", methodCtx, err)
		}

		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, fmt.Errorf("%s: некорректный id истории", methodCtx)
		}
		taskUUID, err := uuid.Parse(taskIDStr)
		if err != nil {
			return nil, fmt.Errorf("%s: некорректный task_id", methodCtx)
		}
		changedBy, err := uuid.Parse(changedByStr)
		if err != nil {
			return nil, fmt.Errorf("%s: некорректный changed_by", methodCtx)
		}

		changes := map[string]interface{}{}
		if len(changesData) > 0 {
			if err := json.Unmarshal(changesData, &changes); err != nil {
				return nil, fmt.Errorf("%s: ошибка разбора изменений", methodCtx)
			}
		}

		items = append(items, TaskHistoryRecord{
			ID:        id,
			TaskID:    taskUUID,
			ChangedBy: changedBy,
			Changes:   changes,
			ChangedAt: changedAt,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", methodCtx, err)
	}

	return items, nil
}
