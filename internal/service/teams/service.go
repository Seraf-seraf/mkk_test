package teams

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Seraf-seraf/mkk_test/internal/api"
	"github.com/Seraf-seraf/mkk_test/internal/pkg/breaker"
	"github.com/Seraf-seraf/mkk_test/internal/pkg/mailer"
	repomysql "github.com/Seraf-seraf/mkk_test/internal/repo/mysql"
)

// Service реализует бизнес-логику команд и приглашений.
type Service struct {
	db      *sql.DB
	teams   TeamsRepository
	members MembersRepository
	invites InvitesRepository
	users   UsersRepository
	mailer  mailer.Mailer
	breaker breaker.Breaker
}

// TeamsRepository описывает работу с командами.
type TeamsRepository interface {
	Create(ctx context.Context, exec repomysql.DBTX, id uuid.UUID, name string, createdBy uuid.UUID, createdAt time.Time) error
	ListByUser(ctx context.Context, userID uuid.UUID) ([]repomysql.TeamRecord, error)
	Exists(ctx context.Context, teamID uuid.UUID) (bool, error)
}

// MembersRepository описывает работу с участниками команды.
type MembersRepository interface {
	Add(ctx context.Context, exec repomysql.DBTX, teamID uuid.UUID, userID uuid.UUID, role string, createdAt time.Time) error
	GetRole(ctx context.Context, teamID uuid.UUID, userID uuid.UUID) (string, bool, error)
	IsMember(ctx context.Context, teamID uuid.UUID, userID uuid.UUID) (bool, error)
}

// InvitesRepository описывает работу с приглашениями.
type InvitesRepository interface {
	Create(ctx context.Context, exec repomysql.DBTX, record repomysql.TeamInviteRecord) error
	GetByCode(ctx context.Context, code string) (*repomysql.TeamInviteRecord, error)
	DeleteByCode(ctx context.Context, exec repomysql.DBTX, code string) error
}

// UsersRepository описывает доступ к пользователям.
type UsersRepository interface {
	FindIDByEmail(ctx context.Context, email string) (uuid.UUID, bool, error)
	GetEmailByID(ctx context.Context, id uuid.UUID) (string, bool, error)
}

// NewService создает сервис команд.
func NewService(db *sql.DB, teams TeamsRepository, members MembersRepository, invites InvitesRepository, users UsersRepository, mailer mailer.Mailer, breaker breaker.Breaker) (*Service, error) {
	const methodCtx = "teams.NewService"

	slog.Debug("инициализация сервиса команд", slog.String("context", methodCtx))

	if db == nil {
		return nil, fmt.Errorf("%s: db не задан", methodCtx)
	}
	if teams == nil {
		return nil, fmt.Errorf("%s: teams repo не задан", methodCtx)
	}
	if members == nil {
		return nil, fmt.Errorf("%s: members repo не задан", methodCtx)
	}
	if invites == nil {
		return nil, fmt.Errorf("%s: invites repo не задан", methodCtx)
	}
	if users == nil {
		return nil, fmt.Errorf("%s: users repo не задан", methodCtx)
	}

	return &Service{
		db:      db,
		teams:   teams,
		members: members,
		invites: invites,
		users:   users,
		mailer:  mailer,
		breaker: breaker,
	}, nil
}

// CreateTeam создает команду и добавляет создателя как owner.
func (s *Service) CreateTeam(ctx context.Context, userID uuid.UUID, req api.CreateTeamRequest) (api.Team, error) {
	const methodCtx = "teams.Service.CreateTeam"

	slog.Debug("вызов создания команды", slog.String("context", methodCtx))

	if strings.TrimSpace(req.Name) == "" {
		return api.Team{}, fmt.Errorf("%s: имя команды не задано", methodCtx)
	}

	now := time.Now().UTC()
	teamID := uuid.New()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return api.Team{}, fmt.Errorf("%s: %w", methodCtx, err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := s.teams.Create(ctx, tx, teamID, req.Name, userID, now); err != nil {
		return api.Team{}, fmt.Errorf("%s: %w", methodCtx, err)
	}
	if err := s.members.Add(ctx, tx, teamID, userID, "owner", now); err != nil {
		return api.Team{}, fmt.Errorf("%s: %w", methodCtx, err)
	}

	if err := tx.Commit(); err != nil {
		return api.Team{}, fmt.Errorf("%s: %w", methodCtx, err)
	}

	return api.Team{
		Id:        api.UUID(teamID),
		Name:      req.Name,
		CreatedBy: api.UUID(userID),
		CreatedAt: now,
	}, nil
}

// ListTeams возвращает список команд пользователя.
func (s *Service) ListTeams(ctx context.Context, userID uuid.UUID) (api.TeamsListResponse, error) {
	const methodCtx = "teams.Service.ListTeams"

	slog.Debug("вызов списка команд", slog.String("context", methodCtx))

	records, err := s.teams.ListByUser(ctx, userID)
	if err != nil {
		return api.TeamsListResponse{}, fmt.Errorf("%s: %w", methodCtx, err)
	}

	var items []api.Team
	for _, rec := range records {
		items = append(items, api.Team{
			Id:        api.UUID(rec.ID),
			Name:      rec.Name,
			CreatedBy: api.UUID(rec.CreatedBy),
			CreatedAt: rec.CreatedAt,
		})
	}

	return api.TeamsListResponse{Items: items}, nil
}

// Invite создает приглашение в команду.
func (s *Service) Invite(ctx context.Context, inviterID uuid.UUID, teamID uuid.UUID, req api.InviteRequest) (api.Invite, error) {
	const methodCtx = "teams.Service.Invite"

	slog.Debug("вызов приглашения в команду", slog.String("context", methodCtx))

	if strings.TrimSpace(string(req.Email)) == "" {
		return api.Invite{}, fmt.Errorf("%s: email не задан", methodCtx)
	}

	teamExists, err := s.teams.Exists(ctx, teamID)
	if err != nil {
		return api.Invite{}, fmt.Errorf("%s: %w", methodCtx, err)
	}
	if !teamExists {
		return api.Invite{}, fmt.Errorf("%s: %w", methodCtx, ErrNotFound)
	}

	role, ok, err := s.members.GetRole(ctx, teamID, inviterID)
	if err != nil {
		return api.Invite{}, fmt.Errorf("%s: %w", methodCtx, err)
	}
	if !ok || (role != "owner" && role != "admin") {
		return api.Invite{}, fmt.Errorf("%s: %w", methodCtx, ErrForbidden)
	}

	foundID, exists, err := s.users.FindIDByEmail(ctx, string(req.Email))
	if err != nil {
		return api.Invite{}, fmt.Errorf("%s: %w", methodCtx, err)
	}
	if exists {
		isMember, err := s.members.IsMember(ctx, teamID, foundID)
		if err != nil {
			return api.Invite{}, fmt.Errorf("%s: %w", methodCtx, err)
		}
		if isMember {
			return api.Invite{}, fmt.Errorf("%s: %w", methodCtx, ErrAlreadyMember)
		}
	}

	now := time.Now().UTC()
	inviteID := uuid.New()
	code := uuid.NewString()

	err = s.invites.Create(ctx, nil, repomysql.TeamInviteRecord{
		ID:        inviteID,
		TeamID:    teamID,
		Email:     string(req.Email),
		InviterID: inviterID,
		Code:      code,
		CreatedAt: now,
	})
	if err != nil {
		return api.Invite{}, fmt.Errorf("%s: %w", methodCtx, err)
	}

	if s.mailer != nil {
		send := func() error {
			return s.mailer.Send(ctx, mailer.Message{
				To:      string(req.Email),
				Subject: "Приглашение в команду",
				Body:    fmt.Sprintf("Ваш код приглашения: %s", code),
			})
		}
		if s.breaker != nil {
			err = s.breaker.Execute(send)
		} else {
			err = send()
		}
		if err != nil {
			return api.Invite{}, fmt.Errorf("%s: ошибка отправки письма: %w", methodCtx, err)
		}
	}

	return api.Invite{
		Id:        api.UUID(inviteID),
		TeamId:    api.UUID(teamID),
		Email:     req.Email,
		InviterId: api.UUID(inviterID),
		Code:      code,
		CreatedAt: now,
	}, nil
}

// AcceptInvite принимает приглашение по коду.
func (s *Service) AcceptInvite(ctx context.Context, userID uuid.UUID, req api.AcceptInviteRequest) (api.TeamMember, error) {
	const methodCtx = "teams.Service.AcceptInvite"

	slog.Debug("вызов принятия приглашения", slog.String("context", methodCtx))

	if strings.TrimSpace(req.Code) == "" {
		return api.TeamMember{}, fmt.Errorf("%s: код не задан", methodCtx)
	}

	invite, err := s.invites.GetByCode(ctx, req.Code)
	if err != nil {
		return api.TeamMember{}, fmt.Errorf("%s: %w", methodCtx, err)
	}
	if invite == nil {
		return api.TeamMember{}, fmt.Errorf("%s: %w", methodCtx, ErrInviteNotFound)
	}

	userEmail, ok, err := s.users.GetEmailByID(ctx, userID)
	if err != nil {
		return api.TeamMember{}, fmt.Errorf("%s: %w", methodCtx, err)
	}
	if !ok {
		return api.TeamMember{}, fmt.Errorf("%s: %w", methodCtx, ErrNotFound)
	}
	if !strings.EqualFold(invite.Email, userEmail) {
		return api.TeamMember{}, fmt.Errorf("%s: %w", methodCtx, ErrInviteEmailMismatch)
	}

	isMember, err := s.members.IsMember(ctx, invite.TeamID, userID)
	if err != nil {
		return api.TeamMember{}, fmt.Errorf("%s: %w", methodCtx, err)
	}
	if isMember {
		return api.TeamMember{}, fmt.Errorf("%s: %w", methodCtx, ErrAlreadyMember)
	}

	now := time.Now().UTC()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return api.TeamMember{}, fmt.Errorf("%s: %w", methodCtx, err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := s.members.Add(ctx, tx, invite.TeamID, userID, "member", now); err != nil {
		return api.TeamMember{}, fmt.Errorf("%s: %w", methodCtx, err)
	}
	if err := s.invites.DeleteByCode(ctx, tx, invite.Code); err != nil {
		return api.TeamMember{}, fmt.Errorf("%s: %w", methodCtx, err)
	}

	if err := tx.Commit(); err != nil {
		return api.TeamMember{}, fmt.Errorf("%s: %w", methodCtx, err)
	}

	return api.TeamMember{
		TeamId:    api.UUID(invite.TeamID),
		UserId:    api.UUID(userID),
		Role:      api.TeamMemberRole("member"),
		CreatedAt: now,
	}, nil
}
