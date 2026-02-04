package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pressly/goose/v3"
	"gopkg.in/yaml.v3"

	"github.com/Seraf-seraf/mkk_test/internal/config"
	mysqlpkg "github.com/Seraf-seraf/mkk_test/internal/pkg/mysql"
	redispkg "github.com/Seraf-seraf/mkk_test/internal/pkg/redis"
)

const (
	openAPISpecPath       = "api/openapi.yml"
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

	router := gin.New()
	router.Use(gin.Recovery())

	if cfg.Metrics.Enabled {
		registerMetrics(router, cfg.Metrics.Path)
	}

	if err := registerOpenAPI(router); err != nil {
		return nil, nil, fmt.Errorf("%s: %w", methodCtx, err)
	}

	addr := net.JoinHostPort(cfg.Server.Host, strconv.Itoa(cfg.Server.Port))
	server := &http.Server{
		Addr:    addr,
		Handler: router,
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

func registerMetrics(router *gin.Engine, path string) {
	const methodCtx = "app.registerMetrics"

	if path == "" {
		path = "/metrics"
	}

	slog.Debug("регистрация метрик", slog.String("context", methodCtx), slog.String("path", path))

	router.GET(path, func(c *gin.Context) {
		c.Header("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		c.String(http.StatusOK, "app_up 1\n")
	})
}

func registerOpenAPI(router *gin.Engine) error {
	const methodCtx = "app.registerOpenAPI"

	spec, err := loadOpenAPISpec(openAPISpecPath)
	if err != nil {
		return fmt.Errorf("%s: %w", methodCtx, err)
	}

	slog.Debug("регистрация openapi.json", slog.String("context", methodCtx))

	router.GET("/openapi.json", func(c *gin.Context) {
		c.Header("Content-Type", "application/json; charset=utf-8")
		c.Writer.WriteHeader(http.StatusOK)
		_, _ = c.Writer.Write(spec)
	})

	return nil
}

func loadOpenAPISpec(path string) ([]byte, error) {
	const methodCtx = "app.loadOpenAPISpec"

	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%s: спецификация OpenAPI не найдена: %s", methodCtx, path)
		}
		return nil, fmt.Errorf("%s: ошибка проверки файла спецификации: %w", methodCtx, err)
	}

	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("%s: ошибка чтения спецификации: %w", methodCtx, err)
	}

	var doc interface{}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("%s: ошибка разбора спецификации: %w", methodCtx, err)
	}

	jsonData, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("%s: ошибка сериализации спецификации: %w", methodCtx, err)
	}

	return jsonData, nil
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
