package handler

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/Seraf-seraf/mkk_test/internal/api"
)

// AuthService описывает методы сервиса аутентификации.
type AuthService interface {
	Register(ctx context.Context, req api.RegisterRequest) (api.User, error)
	Login(ctx context.Context, req api.LoginRequest) (api.AuthResponse, error)
}

// TeamsService описывает методы сервиса команд.
type TeamsService interface {
	CreateTeam(ctx context.Context, userID uuid.UUID, req api.CreateTeamRequest) (api.Team, error)
	ListTeams(ctx context.Context, userID uuid.UUID) (api.TeamsListResponse, error)
	Invite(ctx context.Context, inviterID uuid.UUID, teamID uuid.UUID, req api.InviteRequest) (api.Invite, error)
	AcceptInvite(ctx context.Context, userID uuid.UUID, req api.AcceptInviteRequest) (api.TeamMember, error)
}

// TasksService описывает методы сервиса задач.
type TasksService interface {
	Create(ctx context.Context, userID uuid.UUID, req api.CreateTaskRequest) (api.Task, error)
	List(ctx context.Context, userID uuid.UUID, teamID uuid.UUID, status *api.TaskStatus, assigneeID *uuid.UUID, page int, perPage int) (api.TasksListResponse, error)
	Update(ctx context.Context, userID uuid.UUID, taskID uuid.UUID, req api.UpdateTaskRequest) (api.Task, error)
	History(ctx context.Context, userID uuid.UUID, taskID uuid.UUID) (api.TaskHistoryListResponse, error)
}

// CommentsService описывает методы сервиса комментариев.
type CommentsService interface {
	Create(ctx context.Context, userID uuid.UUID, taskID uuid.UUID, req api.CreateCommentRequest) (api.Comment, error)
	List(ctx context.Context, userID uuid.UUID, taskID uuid.UUID, page int, perPage int) (api.CommentsListResponse, error)
	Update(ctx context.Context, userID uuid.UUID, taskID uuid.UUID, commentID uuid.UUID, req api.UpdateCommentRequest) (api.Comment, error)
	Delete(ctx context.Context, userID uuid.UUID, taskID uuid.UUID, commentID uuid.UUID) error
}

// ReportsService описывает методы сервиса отчетов.
type ReportsService interface {
	TeamSummary(ctx context.Context, userID uuid.UUID) ([]api.TeamSummary, error)
	TopCreators(ctx context.Context, userID uuid.UUID, month string) ([]api.TeamTopCreators, error)
	InvalidAssignees(ctx context.Context, userID uuid.UUID) ([]api.InvalidAssignee, error)
}

// Handler реализует HTTP-обработчики по контракту OpenAPI.
type Handler struct {
	auth     AuthService
	teams    TeamsService
	tasks    TasksService
	comments CommentsService
	reports  ReportsService
}

// New создает новый набор обработчиков.
func New(auth AuthService, teams TeamsService, tasks TasksService, comments CommentsService, reports ReportsService) (*Handler, error) {
	const methodCtx = "handler.New"

	slog.Debug("инициализация HTTP-обработчиков", slog.String("context", methodCtx))

	if auth == nil {
		return nil, fmt.Errorf("%s: auth сервис не задан", methodCtx)
	}
	if teams == nil {
		return nil, fmt.Errorf("%s: teams сервис не задан", methodCtx)
	}
	if tasks == nil {
		return nil, fmt.Errorf("%s: tasks сервис не задан", methodCtx)
	}
	if comments == nil {
		return nil, fmt.Errorf("%s: comments сервис не задан", methodCtx)
	}
	if reports == nil {
		return nil, fmt.Errorf("%s: reports сервис не задан", methodCtx)
	}

	return &Handler{auth: auth, teams: teams, tasks: tasks, comments: comments, reports: reports}, nil
}
