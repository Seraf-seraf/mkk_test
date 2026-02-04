package app

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pressly/goose/v3"

	appmw "github.com/Seraf-seraf/mkk_test/internal/app/middlewares"
	"github.com/Seraf-seraf/mkk_test/internal/config"
	httpserver "github.com/Seraf-seraf/mkk_test/internal/pkg/http"
	mysqlpkg "github.com/Seraf-seraf/mkk_test/internal/pkg/mysql"
	redispkg "github.com/Seraf-seraf/mkk_test/internal/pkg/redis"
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

	publicMW, apiMW, err := buildAPIMiddlewares(cfg)
	if err != nil {
		_ = redisClient.Close()
		_ = db.Close()
		return nil, nil, fmt.Errorf("%s: %w", methodCtx, err)
	}

	server, err := httpserver.New(cfg, httpserver.Deps{Redis: redisClient}, httpserver.ServerOptions{
		PublicGroupMW: publicMW,
		APIGroupMW:    apiMW,
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

func buildAPIMiddlewares(cfg *config.Config) ([]gin.HandlerFunc, []gin.HandlerFunc, error) {
	const methodCtx = "app.buildAPIMiddlewares"

	validator, err := appmw.OapiRequestValidator("api/openapi.yml")
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", methodCtx, err)
	}

	jwtValidator, err := appmw.NewJWTValidator(cfg.Auth.JWT)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", methodCtx, err)
	}

	publicMW := []gin.HandlerFunc{validator}
	apiMW := []gin.HandlerFunc{validator, appmw.JWT(jwtValidator)}

	return publicMW, apiMW, nil
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
