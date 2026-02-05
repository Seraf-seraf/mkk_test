package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// TeamInviteRecord содержит данные приглашения.
type TeamInviteRecord struct {
	ID        uuid.UUID
	TeamID    uuid.UUID
	Email     string
	InviterID uuid.UUID
	Code      string
	CreatedAt time.Time
}

// TeamInvitesRepo реализует доступ к приглашениям.
type TeamInvitesRepo struct {
	db *sql.DB
}

// NewTeamInvitesRepo создает репозиторий приглашений.
func NewTeamInvitesRepo(db *sql.DB) *TeamInvitesRepo {
	const methodCtx = "repo.NewTeamInvitesRepo"

	slog.Debug("инициализация репозитория приглашений", slog.String("context", methodCtx))

	return &TeamInvitesRepo{db: db}
}

// Create создает приглашение.
func (r *TeamInvitesRepo) Create(ctx context.Context, exec DBTX, record TeamInviteRecord) error {
	const methodCtx = "repo.TeamInvitesRepo.Create"

	if r == nil || r.db == nil {
		return fmt.Errorf("%s: репозиторий не инициализирован", methodCtx)
	}
	if exec == nil {
		exec = r.db
	}

	_, err := exec.ExecContext(
		ctx,
		"INSERT INTO team_invites (id, team_id, email, inviter_id, code, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		record.ID.String(),
		record.TeamID.String(),
		record.Email,
		record.InviterID.String(),
		record.Code,
		record.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", methodCtx, err)
	}
	return nil
}

// GetByCode возвращает приглашение по коду.
func (r *TeamInvitesRepo) GetByCode(ctx context.Context, code string) (*TeamInviteRecord, error) {
	const methodCtx = "repo.TeamInvitesRepo.GetByCode"

	if r == nil || r.db == nil {
		return nil, fmt.Errorf("%s: репозиторий не инициализирован", methodCtx)
	}

	var rec TeamInviteRecord
	var idStr, teamIDStr, inviterIDStr string
	err := r.db.QueryRowContext(
		ctx,
		"SELECT id, team_id, email, inviter_id, code, created_at FROM team_invites WHERE code = ?",
		code,
	).Scan(&idStr, &teamIDStr, &rec.Email, &inviterIDStr, &rec.Code, &rec.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", methodCtx, err)
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("%s: некорректный id приглашения", methodCtx)
	}
	teamID, err := uuid.Parse(teamIDStr)
	if err != nil {
		return nil, fmt.Errorf("%s: некорректный id команды", methodCtx)
	}
	inviterID, err := uuid.Parse(inviterIDStr)
	if err != nil {
		return nil, fmt.Errorf("%s: некорректный id пригласившего", methodCtx)
	}

	rec.ID = id
	rec.TeamID = teamID
	rec.InviterID = inviterID

	return &rec, nil
}

// DeleteByCode удаляет приглашение по коду.
func (r *TeamInvitesRepo) DeleteByCode(ctx context.Context, exec DBTX, code string) error {
	const methodCtx = "repo.TeamInvitesRepo.DeleteByCode"

	if r == nil || r.db == nil {
		return fmt.Errorf("%s: репозиторий не инициализирован", methodCtx)
	}
	if exec == nil {
		exec = r.db
	}

	_, err := exec.ExecContext(ctx, "DELETE FROM team_invites WHERE code = ?", code)
	if err != nil {
		return fmt.Errorf("%s: %w", methodCtx, err)
	}
	return nil
}
