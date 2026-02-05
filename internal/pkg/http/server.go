package http

import (
	"encoding/json"
	"fmt"
	"net"
	nethttp "net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/yaml.v3"

	"github.com/Seraf-seraf/mkk_test/internal/config"
	"github.com/Seraf-seraf/mkk_test/internal/pkg/http/middlewares"
	"github.com/redis/go-redis/v9"
)

const openAPISpecPath = "api/openapi.yml"

// ServerOptions описывает дополнительные параметры создания HTTP сервера.
type ServerOptions struct {
	Middlewares    []gin.HandlerFunc
	APIPrefix      string
	Register       func(*gin.Engine) error
	RegisterPublic func(*gin.RouterGroup) error
	RegisterAPI    func(*gin.RouterGroup) error
	PublicGroupMW  []gin.HandlerFunc
	APIGroupMW     []gin.HandlerFunc
}

// Deps содержит зависимости для HTTP сервера.
type Deps struct {
	Redis *redis.Client
}

// New создает HTTP сервер и настраивает базовые middleware и маршруты.
func New(cfg *config.Config, deps Deps, opts ServerOptions) (*nethttp.Server, error) {
	const methodCtx = "http.New"

	if cfg == nil {
		return nil, fmt.Errorf("%s: конфигурация не задана", methodCtx)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.RemoveExtraSlash = true

	if cfg.Metrics.Enabled {
		router.Use(middlewares.Metrics())
	}

	router.Use(middlewares.CORS())

	if len(opts.Middlewares) > 0 {
		router.Use(opts.Middlewares...)
	}

	limiter, err := middlewares.NewRateLimiter(deps.Redis)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", methodCtx, err)
	}
	router.Use(middlewares.RateLimit(limiter))

	if cfg.Metrics.Enabled {
		metricsPath := cfg.Metrics.Path
		if metricsPath == "" {
			metricsPath = "/metrics"
		}
		router.GET(metricsPath, gin.WrapH(promhttp.Handler()))
	}

	if err := registerOpenAPI(router); err != nil {
		return nil, fmt.Errorf("%s: %w", methodCtx, err)
	}

	apiPrefix := opts.APIPrefix
	if apiPrefix == "" {
		apiPrefix = "/api/v1"
	}

	if opts.RegisterPublic != nil || len(opts.PublicGroupMW) > 0 {
		publicGroup := router.Group(apiPrefix, opts.PublicGroupMW...)
		if opts.RegisterPublic != nil {
			if err := opts.RegisterPublic(publicGroup); err != nil {
				return nil, fmt.Errorf("%s: %w", methodCtx, err)
			}
		}
	}

	if opts.RegisterAPI != nil || len(opts.APIGroupMW) > 0 {
		apiGroup := router.Group(apiPrefix, opts.APIGroupMW...)
		if opts.RegisterAPI != nil {
			if err := opts.RegisterAPI(apiGroup); err != nil {
				return nil, fmt.Errorf("%s: %w", methodCtx, err)
			}
		}
	}

	if opts.Register != nil {
		if err := opts.Register(router); err != nil {
			return nil, fmt.Errorf("%s: %w", methodCtx, err)
		}
	}

	addr := net.JoinHostPort(cfg.Server.Host, strconv.Itoa(cfg.Server.Port))
	server := &nethttp.Server{
		Addr:    addr,
		Handler: router,
	}

	return server, nil
}

func registerOpenAPI(router *gin.Engine) error {
	const methodCtx = "http.registerOpenAPI"

	spec, err := loadOpenAPISpec(openAPISpecPath)
	if err != nil {
		return fmt.Errorf("%s: %w", methodCtx, err)
	}

	router.GET("/openapi.json", func(c *gin.Context) {
		c.Header("Content-Type", "application/json; charset=utf-8")
		c.Writer.WriteHeader(nethttp.StatusOK)
		_, _ = c.Writer.Write(spec)
	})

	return nil
}

func loadOpenAPISpec(path string) ([]byte, error) {
	const methodCtx = "http.loadOpenAPISpec"

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
