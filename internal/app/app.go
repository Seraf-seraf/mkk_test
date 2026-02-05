package app

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pressly/goose/v3"

	"github.com/Seraf-seraf/mkk_test/internal/api"
	appmw "github.com/Seraf-seraf/mkk_test/internal/app/middlewares"
	"github.com/Seraf-seraf/mkk_test/internal/config"
	"github.com/Seraf-seraf/mkk_test/internal/handler"
	"github.com/Seraf-seraf/mkk_test/internal/pkg/breaker"
	"github.com/Seraf-seraf/mkk_test/internal/pkg/cache"
	httpserver "github.com/Seraf-seraf/mkk_test/internal/pkg/http"
	"github.com/Seraf-seraf/mkk_test/internal/pkg/mailer"
	mysqlpkg "github.com/Seraf-seraf/mkk_test/internal/pkg/mysql"
	redispkg "github.com/Seraf-seraf/mkk_test/internal/pkg/redis"
	repomysql "github.com/Seraf-seraf/mkk_test/internal/repo/mysql"
	"github.com/Seraf-seraf/mkk_test/internal/service/auth"
	"github.com/Seraf-seraf/mkk_test/internal/service/comments"
	"github.com/Seraf-seraf/mkk_test/internal/service/reports"
	"github.com/Seraf-seraf/mkk_test/internal/service/tasks"
	"github.com/Seraf-seraf/mkk_test/internal/service/teams"
)

const (
	schemaMigrationsDir   = "internal/migrations/schema"
	dataMigrationsDir     = "internal/migrations/data"
	schemaMigrationsTable = "goose_db_version"
	dataMigrationsTable   = "goose_db_version_data"
)

// ShutdownFunc вызывается при graceful shutdown.
type ShutdownFunc func(ctx context.Context) error

// New инициализирует приложение и возвращает HTTP сервер и функцию завершения.
func New(cfg *config.Config) (*http.Server, ShutdownFunc, error) {
	const methodCtx = "app.New"

	if cfg == nil {
		return nil, nil, fmt.Errorf("%s: конфигурация не задана", methodCtx)
	}

	db, err := mysqlpkg.Open(cfg.MySQL)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", methodCtx, err)
	}

	redisClient, err := redispkg.New(cfg.Redis)
	if err != nil {
		_ = db.Close()
		return nil, nil, fmt.Errorf("%s: %w", methodCtx, err)
	}

	if cfg.Migrations.Auto {
		if err := runMigrations(db); err != nil {
			_ = redisClient.Close()
			_ = db.Close()
			return nil, nil, fmt.Errorf("%s: %w", methodCtx, err)
		}
	}

	publicMW, apiMW, optionalMW, err := buildAPIMiddlewares(cfg)
	if err != nil {
		_ = redisClient.Close()
		_ = db.Close()
		return nil, nil, fmt.Errorf("%s: %w", methodCtx, err)
	}

	tasksCache, err := cache.NewTasksCache(redisClient)
	if err != nil {
		_ = redisClient.Close()
		_ = db.Close()
		return nil, nil, fmt.Errorf("%s: %w", methodCtx, err)
	}

	usersRepo := repomysql.NewUsersRepo(db)
	teamsRepo := repomysql.NewTeamsRepo(db)
	membersRepo := repomysql.NewTeamMembersRepo(db)
	invitesRepo := repomysql.NewTeamInvitesRepo(db)
	tasksRepo := repomysql.NewTasksRepo(db)
	historyRepo := repomysql.NewTaskHistoryRepo(db)
	commentsRepo := repomysql.NewCommentsRepo(db)
	reportsRepo := repomysql.NewReportsRepo(db)

	cb, err := breaker.New("mailer")
	if err != nil {
		_ = redisClient.Close()
		_ = db.Close()
		return nil, nil, fmt.Errorf("%s: %w", methodCtx, err)
	}

	mailerSvc := mailer.NewMockMailer()

	authSvc, err := auth.NewService(usersRepo, membersRepo, mailerSvc, cb, cfg.Auth.JWT)
	if err != nil {
		_ = redisClient.Close()
		_ = db.Close()
		return nil, nil, fmt.Errorf("%s: %w", methodCtx, err)
	}

	teamsSvc, err := teams.NewService(db, teamsRepo, membersRepo, invitesRepo, usersRepo, mailerSvc, cb)
	if err != nil {
		_ = redisClient.Close()
		_ = db.Close()
		return nil, nil, fmt.Errorf("%s: %w", methodCtx, err)
	}

	tasksSvc, err := tasks.NewService(db, tasksRepo, membersRepo, historyRepo, tasksCache)
	if err != nil {
		_ = redisClient.Close()
		_ = db.Close()
		return nil, nil, fmt.Errorf("%s: %w", methodCtx, err)
	}

	commentsSvc, err := comments.NewService(commentsRepo, tasksRepo, membersRepo)
	if err != nil {
		_ = redisClient.Close()
		_ = db.Close()
		return nil, nil, fmt.Errorf("%s: %w", methodCtx, err)
	}

	reportsSvc, err := reports.NewService(reportsRepo)
	if err != nil {
		_ = redisClient.Close()
		_ = db.Close()
		return nil, nil, fmt.Errorf("%s: %w", methodCtx, err)
	}

	handlerSvc, err := handler.New(authSvc, teamsSvc, tasksSvc, commentsSvc, reportsSvc)
	if err != nil {
		_ = redisClient.Close()
		_ = db.Close()
		return nil, nil, fmt.Errorf("%s: %w", methodCtx, err)
	}

	errorHandler := func(c *gin.Context, err error, statusCode int) {
		c.JSON(statusCode, api.ErrorResponse{Error: err.Error()})
	}
	wrapper := api.ServerInterfaceWrapper{
		Handler:      handlerSvc,
		ErrorHandler: errorHandler,
	}

	server, err := httpserver.New(cfg, httpserver.Deps{Redis: redisClient}, httpserver.ServerOptions{
		Middlewares:   optionalMW,
		PublicGroupMW: publicMW,
		APIGroupMW:    apiMW,
		RegisterPublic: func(group *gin.RouterGroup) error {
			group.POST("/login", wrapper.PostApiV1Login)
			group.POST("/register", wrapper.PostApiV1Register)
			return nil
		},
		RegisterAPI: func(group *gin.RouterGroup) error {
			group.GET("/teams", wrapper.GetApiV1Teams)
			group.POST("/teams", wrapper.PostApiV1Teams)
			group.POST("/teams/invites/accept", wrapper.PostApiV1TeamsInvitesAccept)
			group.POST("/teams/:id/invite", appmw.RBAC("owner", "admin"), wrapper.PostApiV1TeamsIdInvite)

			group.GET("/tasks", wrapper.GetApiV1Tasks)
			group.POST("/tasks", appmw.RBAC("member", "admin", "owner"), wrapper.PostApiV1Tasks)
			group.PUT("/tasks/:id", wrapper.PutApiV1TasksId)
			group.GET("/tasks/:id/history", wrapper.GetApiV1TasksIdHistory)
			group.GET("/tasks/:id/comments", wrapper.GetApiV1TasksIdComments)
			group.POST("/tasks/:id/comments", wrapper.PostApiV1TasksIdComments)
			group.PUT("/tasks/:id/comments/:comment_id", wrapper.PutApiV1TasksIdCommentsCommentId)
			group.DELETE("/tasks/:id/comments/:comment_id", wrapper.DeleteApiV1TasksIdCommentsCommentId)

			group.GET("/reports/team-summary", wrapper.GetApiV1ReportsTeamSummary)
			group.GET("/reports/top-creators", wrapper.GetApiV1ReportsTopCreators)
			group.GET("/reports/invalid-assignees", wrapper.GetApiV1ReportsInvalidAssignees)

			return nil
		},
	})
	if err != nil {
		_ = redisClient.Close()
		_ = db.Close()
		return nil, nil, fmt.Errorf("%s: %w", methodCtx, err)
	}

	shutdown := func(ctx context.Context) error {
		var shutdownErr error
		if err := server.Shutdown(ctx); err != nil {
			shutdownErr = err
		}
		if err := redisClient.Close(); err != nil && shutdownErr == nil {
			shutdownErr = fmt.Errorf("ошибка закрытия Redis: %w", err)
		}
		if err := db.Close(); err != nil && shutdownErr == nil {
			shutdownErr = fmt.Errorf("ошибка закрытия базы данных: %w", err)
		}
		return shutdownErr
	}

	return server, shutdown, nil
}

func buildAPIMiddlewares(cfg *config.Config) ([]gin.HandlerFunc, []gin.HandlerFunc, []gin.HandlerFunc, error) {
	const methodCtx = "app.buildAPIMiddlewares"

	validator, err := appmw.OapiRequestValidator("api/openapi.yml")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("%s: %w", methodCtx, err)
	}

	jwtValidator, err := appmw.NewJWTValidator(cfg.Auth.JWT)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("%s: %w", methodCtx, err)
	}

	publicMW := []gin.HandlerFunc{validator}
	apiMW := []gin.HandlerFunc{validator, appmw.JWT(jwtValidator)}

	optionalMW := []gin.HandlerFunc{appmw.JWTOptional(jwtValidator)}

	return publicMW, apiMW, optionalMW, nil
}

func runMigrations(db *sql.DB) error {
	const methodCtx = "app.runMigrations"

	if err := goose.SetDialect("mysql"); err != nil {
		return fmt.Errorf("%s: ошибка настройки миграций: %w", methodCtx, err)
	}

	goose.SetTableName(schemaMigrationsTable)
	if err := goose.Up(db, schemaMigrationsDir); err != nil {
		return fmt.Errorf("%s: ошибка применения миграций схемы: %w", methodCtx, err)
	}

	goose.SetTableName(dataMigrationsTable)
	if err := goose.Up(db, dataMigrationsDir); err != nil {
		return fmt.Errorf("%s: ошибка применения миграций данных: %w", methodCtx, err)
	}

	return nil
}
