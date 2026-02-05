package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// TeamRecord содержит данные команды.
type TeamRecord struct {
	ID        uuid.UUID
	Name      string
	CreatedBy uuid.UUID
	CreatedAt time.Time
}

// TeamsRepo реализует доступ к командам.
type TeamsRepo struct {
	db *sql.DB
}

// NewTeamsRepo создает репозиторий команд.
func NewTeamsRepo(db *sql.DB) *TeamsRepo {
	const methodCtx = "repo.NewTeamsRepo"

	slog.Debug("инициализация репозитория команд", slog.String("context", methodCtx))

	return &TeamsRepo{db: db}
}

// Create создает команду.
func (r *TeamsRepo) Create(ctx context.Context, exec DBTX, id uuid.UUID, name string, createdBy uuid.UUID, createdAt time.Time) error {
	const methodCtx = "repo.TeamsRepo.Create"

	if r == nil || r.db == nil {
		return fmt.Errorf("%s: репозиторий не инициализирован", methodCtx)
	}
	if exec == nil {
		exec = r.db
	}

	_, err := exec.ExecContext(
		ctx,
		"INSERT INTO teams (id, name, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		id.String(),
		name,
		createdBy.String(),
		createdAt,
		createdAt,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", methodCtx, err)
	}

	return nil
}

// ListByUser возвращает команды пользователя.
func (r *TeamsRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]TeamRecord, error) {
	const methodCtx = "repo.TeamsRepo.ListByUser"

	if r == nil || r.db == nil {
		return nil, fmt.Errorf("%s: репозиторий не инициализирован", methodCtx)
	}

	rows, err := r.db.QueryContext(
		ctx,
		`SELECT t.id, t.name, t.created_by, t.created_at
		FROM teams t
		INNER JOIN team_members tm ON tm.team_id = t.id
		WHERE tm.user_id = ?
		ORDER BY t.created_at ASC`,
		userID.String(),
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", methodCtx, err)
	}
	defer rows.Close()

	var items []TeamRecord
	for rows.Next() {
		var idStr, createdByStr string
		var name string
		var createdAt time.Time
		if err := rows.Scan(&idStr, &name, &createdByStr, &createdAt); err != nil {
			return nil, fmt.Errorf("%s: %w", methodCtx, err)
		}
		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, fmt.Errorf("%s: некорректный id команды", methodCtx)
		}
		createdBy, err := uuid.Parse(createdByStr)
		if err != nil {
			return nil, fmt.Errorf("%s: некорректный created_by", methodCtx)
		}
		items = append(items, TeamRecord{
			ID:        id,
			Name:      name,
			CreatedBy: createdBy,
			CreatedAt: createdAt,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", methodCtx, err)
	}

	return items, nil
}

// Exists проверяет наличие команды.
func (r *TeamsRepo) Exists(ctx context.Context, teamID uuid.UUID) (bool, error) {
	const methodCtx = "repo.TeamsRepo.Exists"

	if r == nil || r.db == nil {
		return false, fmt.Errorf("%s: репозиторий не инициализирован", methodCtx)
	}

	var exists int
	err := r.db.QueryRowContext(ctx, "SELECT 1 FROM teams WHERE id = ? LIMIT 1", teamID.String()).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("%s: %w", methodCtx, err)
	}
	return true, nil
}
