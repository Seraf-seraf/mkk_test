package middlewares

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORS разрешает любые источники, методы и базовые заголовки.
func CORS() gin.HandlerFunc {
	const methodCtx = "middlewares.CORS"

	slog.Debug("инициализация CORS middleware", slog.String("context", methodCtx))

	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization,Content-Type,Accept,Origin")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
