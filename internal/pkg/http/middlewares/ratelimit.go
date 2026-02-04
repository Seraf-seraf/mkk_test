package middlewares

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	libredis "github.com/redis/go-redis/v9"
	limiterlib "github.com/ulule/limiter/v3"
	ginmiddleware "github.com/ulule/limiter/v3/drivers/middleware/gin"
	"github.com/ulule/limiter/v3/drivers/store/redis"
)

const (
	rateLimitPerMinute = 100
)

// NewRateLimiter создает лимитер запросов на основе Redis и лимита 100/мин.
func NewRateLimiter(client *libredis.Client) (*limiterlib.Limiter, error) {
	const methodCtx = "middlewares.NewRateLimiter"

	if client == nil {
		return nil, fmt.Errorf("%s: redis клиент не задан", methodCtx)
	}

	store, err := redis.NewStore(client)
	if err != nil {
		return nil, fmt.Errorf("%s: ошибка инициализации store: %w", methodCtx, err)
	}

	rate := limiterlib.Rate{
		Period: time.Minute,
		Limit:  rateLimitPerMinute,
	}

	return limiterlib.New(store, rate), nil
}

// RateLimit возвращает gin middleware для ограничения по IP.
func RateLimit(limiter *limiterlib.Limiter) gin.HandlerFunc {
	const methodCtx = "middlewares.RateLimit"

	slog.Debug("инициализация rate limit middleware", slog.String("context", methodCtx))

	return ginmiddleware.NewMiddleware(
		limiter,
		ginmiddleware.WithKeyGetter(func(c *gin.Context) string {
			return c.ClientIP()
		}),
		ginmiddleware.WithLimitReachedHandler(func(c *gin.Context) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "превышен лимит запросов",
			})
		}),
		ginmiddleware.WithErrorHandler(func(c *gin.Context, err error) {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "ошибка ограничения запросов",
			})
		}),
	)
}
