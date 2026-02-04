package middlewares

import (
	"log/slog"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	requestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "mkk",
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "Количество HTTP запросов",
		},
		[]string{"method", "path", "status"},
	)
	requestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "mkk",
			Subsystem: "http",
			Name:      "request_duration_seconds",
			Help:      "Время обработки HTTP запросов",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)
)

// Metrics считает количество запросов и время ответа.
func Metrics() gin.HandlerFunc {
	const methodCtx = "middlewares.Metrics"

	slog.Debug("инициализация metrics middleware", slog.String("context", methodCtx))

	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		status := strconv.Itoa(c.Writer.Status())
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		requestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		requestDuration.WithLabelValues(c.Request.Method, path, status).Observe(time.Since(start).Seconds())
	}
}
