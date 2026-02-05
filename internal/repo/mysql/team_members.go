package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// TeamMembersRepo реализует доступ к участникам команды.
type TeamMembersRepo struct {
	db *sql.DB
}

// NewTeamMembersRepo создает репозиторий участников.
func NewTeamMembersRepo(db *sql.DB) *TeamMembersRepo {
	const methodCtx = "repo.NewTeamMembersRepo"

	slog.Debug("инициализация репозитория участников", slog.String("context", methodCtx))

	return &TeamMembersRepo{db: db}
}

// Add добавляет участника в команду.
func (r *TeamMembersRepo) Add(ctx context.Context, exec DBTX, teamID uuid.UUID, userID uuid.UUID, role string, createdAt time.Time) error {
	const methodCtx = "repo.TeamMembersRepo.Add"

	if r == nil || r.db == nil {
		return fmt.Errorf("%s: репозиторий не инициализирован", methodCtx)
	}
	if exec == nil {
		exec = r.db
	}

	_, err := exec.ExecContext(
		ctx,
		"INSERT INTO team_members (team_id, user_id, role, created_at) VALUES (?, ?, ?, ?)",
		teamID.String(),
		userID.String(),
		role,
		createdAt,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", methodCtx, err)
	}
	return nil
}

// GetRole возвращает роль пользователя в команде.
func (r *TeamMembersRepo) GetRole(ctx context.Context, teamID uuid.UUID, userID uuid.UUID) (string, bool, error) {
	const methodCtx = "repo.TeamMembersRepo.GetRole"

	if r == nil || r.db == nil {
		return "", false, fmt.Errorf("%s: репозиторий не инициализирован", methodCtx)
	}

	var role string
	err := r.db.QueryRowContext(
		ctx,
		"SELECT role FROM team_members WHERE team_id = ? AND user_id = ?",
		teamID.String(),
		userID.String(),
	).Scan(&role)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("%s: %w", methodCtx, err)
	}
	return role, true, nil
}

// IsMember проверяет членство пользователя в команде.
func (r *TeamMembersRepo) IsMember(ctx context.Context, teamID uuid.UUID, userID uuid.UUID) (bool, error) {
	const methodCtx = "repo.TeamMembersRepo.IsMember"

	if r == nil || r.db == nil {
		return false, fmt.Errorf("%s: репозиторий не инициализирован", methodCtx)
	}

	var exists int
	err := r.db.QueryRowContext(
		ctx,
		"SELECT 1 FROM team_members WHERE team_id = ? AND user_id = ? LIMIT 1",
		teamID.String(),
		userID.String(),
	).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("%s: %w", methodCtx, err)
	}
	return true, nil
}

// GetHighestRoleByUser возвращает максимальную роль пользователя среди команд.
func (r *TeamMembersRepo) GetHighestRoleByUser(ctx context.Context, userID uuid.UUID) (string, bool, error) {
	const methodCtx = "repo.TeamMembersRepo.GetHighestRoleByUser"

	if r == nil || r.db == nil {
		return "", false, fmt.Errorf("%s: репозиторий не инициализирован", methodCtx)
	}

	var role string
	err := r.db.QueryRowContext(
		ctx,
		"SELECT role FROM team_members WHERE user_id = ? ORDER BY FIELD(role, 'owner', 'admin', 'member') LIMIT 1",
		userID.String(),
	).Scan(&role)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("%s: %w", methodCtx, err)
	}

	return role, true, nil
}
