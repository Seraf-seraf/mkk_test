package middlewares

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	ContextUserKey = "user"
	ContextRoleKey = "role"
)

// JWTValidator проверяет токен и возвращает данные пользователя и роль.
type JWTValidator func(token string) (user interface{}, role string, err error)

// JWT проверяет наличие токена и валидирует его.
func JWT(validator JWTValidator) gin.HandlerFunc {
	const methodCtx = "middlewares.JWT"

	return func(c *gin.Context) {
		if validator == nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error":   "jwt validator не настроен",
				"context": methodCtx,
			})
			return
		}

		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "отсутствует токен",
				"context": methodCtx,
			})
			return
		}

		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "некорректный формат токена",
				"context": methodCtx,
			})
			return
		}

		user, role, err := validator(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "токен недействителен",
				"context": methodCtx,
			})
			return
		}

		c.Set(ContextUserKey, user)
		c.Set(ContextRoleKey, role)

		c.Next()
	}
}

// JWTOptional пытается разобрать токен и заполнить контекст, не прерывая запрос при ошибках.
func JWTOptional(validator JWTValidator) gin.HandlerFunc {
	const methodCtx = "middlewares.JWTOptional"

	return func(c *gin.Context) {
		if validator == nil {
			c.Next()
			return
		}

		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.Next()
			return
		}

		user, role, err := validator(parts[1])
		if err != nil {
			slog.Debug("не удалось разобрать JWT", slog.String("context", methodCtx))
			c.Next()
			return
		}

		c.Set(ContextUserKey, user)
		c.Set(ContextRoleKey, role)
		c.Next()
	}
}
