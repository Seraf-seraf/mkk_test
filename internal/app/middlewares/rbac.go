package middlewares

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RBAC проверяет роль пользователя, сохранённую в контексте.
func RBAC(allowed ...string) gin.HandlerFunc {
	const methodCtx = "middlewares.RBAC"

	allowedSet := map[string]struct{}{}
	for _, role := range allowed {
		allowedSet[role] = struct{}{}
	}

	return func(c *gin.Context) {
		roleVal, ok := c.Get(ContextRoleKey)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "роль не определена",
				"context": methodCtx,
			})
			return
		}

		role, ok := roleVal.(string)
		if !ok || role == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "роль не определена",
				"context": methodCtx,
			})
			return
		}

		if len(allowedSet) == 0 {
			c.Next()
			return
		}

		if _, exists := allowedSet[role]; !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "нет прав доступа",
				"context": methodCtx,
			})
			return
		}

		c.Next()
	}
}
