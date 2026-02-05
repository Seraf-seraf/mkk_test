package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRBACNoRole(t *testing.T) {
	const methodCtx = "middlewares.TestRBACNoRole"

	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/protected", RBAC("admin"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusForbidden, resp.Code, methodCtx)
}

func TestRBACForbiddenRole(t *testing.T) {
	const methodCtx = "middlewares.TestRBACForbiddenRole"

	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/protected",
		func(c *gin.Context) {
			c.Set(ContextRoleKey, "member")
			c.Next()
		},
		RBAC("admin"),
		func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		},
	)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusForbidden, resp.Code, methodCtx)
}

func TestRBACAllowedRole(t *testing.T) {
	const methodCtx = "middlewares.TestRBACAllowedRole"

	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/protected",
		func(c *gin.Context) {
			c.Set(ContextRoleKey, "admin")
			c.Next()
		},
		RBAC("admin"),
		func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		},
	)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code, methodCtx)
}
