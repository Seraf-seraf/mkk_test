package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/Seraf-seraf/mkk_test/internal/config"
)

func TestProtectedRouteRequiresJWT(t *testing.T) {
	const methodCtx = "middlewares.TestProtectedRouteRequiresJWT"

	gin.SetMode(gin.TestMode)

	validator, err := NewJWTValidator(config.JWTConfig{Secret: "test-secret"})
	require.NoError(t, err, methodCtx)

	router := gin.New()
	apiGroup := router.Group("/api/v1", JWT(validator))
	apiGroup.GET("/teams", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/teams", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusUnauthorized, resp.Code, methodCtx)
}

func TestPublicRoutesWithoutJWT(t *testing.T) {
	const methodCtx = "middlewares.TestPublicRoutesWithoutJWT"

	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/api/v1/login", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	router.POST("/api/v1/register", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	for _, path := range []string{"/api/v1/login", "/api/v1/register"} {
		req := httptest.NewRequest(http.MethodPost, path, nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		require.Equal(t, http.StatusOK, resp.Code, methodCtx)
	}
}

func TestProtectedRoutesRequireJWT(t *testing.T) {
	const methodCtx = "middlewares.TestProtectedRoutesRequireJWT"

	gin.SetMode(gin.TestMode)

	validator, err := NewJWTValidator(config.JWTConfig{Secret: "test-secret"})
	require.NoError(t, err, methodCtx)

	router := gin.New()
	apiGroup := router.Group("/api/v1", JWT(validator))
	apiGroup.GET("/teams", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	apiGroup.POST("/tasks", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	apiGroup.GET("/reports/team-summary", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	for _, tc := range []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/api/v1/teams"},
		{method: http.MethodPost, path: "/api/v1/tasks"},
		{method: http.MethodGet, path: "/api/v1/reports/team-summary"},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		require.Equal(t, http.StatusUnauthorized, resp.Code, methodCtx)
	}
}

func TestProtectedRoutesWithParamsRequireJWT(t *testing.T) {
	const methodCtx = "middlewares.TestProtectedRoutesWithParamsRequireJWT"

	gin.SetMode(gin.TestMode)

	validator, err := NewJWTValidator(config.JWTConfig{Secret: "test-secret"})
	require.NoError(t, err, methodCtx)

	router := gin.New()
	apiGroup := router.Group("/api/v1", JWT(validator))
	apiGroup.GET("/tasks/:id", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	apiGroup.PUT("/tasks/:id", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	apiGroup.GET("/tasks/:id/history", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	apiGroup.GET("/tasks/:id/comments", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	apiGroup.POST("/tasks/:id/comments", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	apiGroup.PUT("/tasks/:id/comments/:comment_id", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	apiGroup.DELETE("/tasks/:id/comments/:comment_id", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	apiGroup.POST("/teams/invites/accept", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	apiGroup.GET("/reports/top-creators", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	apiGroup.GET("/reports/invalid-assignees", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	for _, tc := range []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/api/v1/tasks/111"},
		{method: http.MethodPut, path: "/api/v1/tasks/111"},
		{method: http.MethodGet, path: "/api/v1/tasks/111/history"},
		{method: http.MethodGet, path: "/api/v1/tasks/111/comments"},
		{method: http.MethodPost, path: "/api/v1/tasks/111/comments"},
		{method: http.MethodPut, path: "/api/v1/tasks/111/comments/222"},
		{method: http.MethodDelete, path: "/api/v1/tasks/111/comments/222"},
		{method: http.MethodPost, path: "/api/v1/teams/invites/accept"},
		{method: http.MethodGet, path: "/api/v1/reports/top-creators"},
		{method: http.MethodGet, path: "/api/v1/reports/invalid-assignees"},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		require.Equal(t, http.StatusUnauthorized, resp.Code, methodCtx)
	}
}

func TestProtectedRouteWithJWT(t *testing.T) {
	const methodCtx = "middlewares.TestProtectedRouteWithJWT"

	gin.SetMode(gin.TestMode)

	validator, err := NewJWTValidator(config.JWTConfig{Secret: "test-secret"})
	require.NoError(t, err, methodCtx)

	router := gin.New()
	apiGroup := router.Group("/api/v1", JWT(validator))
	apiGroup.GET("/teams", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	token := buildToken(t, "test-secret", "user-1", "member")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/teams", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code, methodCtx)
}

func TestRBACForbiddenOnRoute(t *testing.T) {
	const methodCtx = "middlewares.TestRBACForbiddenOnRoute"

	gin.SetMode(gin.TestMode)

	validator, err := NewJWTValidator(config.JWTConfig{Secret: "test-secret"})
	require.NoError(t, err, methodCtx)

	router := gin.New()
	apiGroup := router.Group("/api/v1", JWT(validator))
	apiGroup.POST("/teams/:id/invite", RBAC("owner", "admin"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	token := buildToken(t, "test-secret", "user-2", "member")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/teams/111/invite", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusForbidden, resp.Code, methodCtx)
}

func TestRBACAllowedOnRoute(t *testing.T) {
	const methodCtx = "middlewares.TestRBACAllowedOnRoute"

	gin.SetMode(gin.TestMode)

	validator, err := NewJWTValidator(config.JWTConfig{Secret: "test-secret"})
	require.NoError(t, err, methodCtx)

	router := gin.New()
	apiGroup := router.Group("/api/v1", JWT(validator))
	apiGroup.POST("/teams/:id/invite", RBAC("owner", "admin"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	token := buildToken(t, "test-secret", "user-3", "admin")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/teams/111/invite", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code, methodCtx)
}

func TestRBACOwnerAllowedOnInvite(t *testing.T) {
	const methodCtx = "middlewares.TestRBACOwnerAllowedOnInvite"

	gin.SetMode(gin.TestMode)

	validator, err := NewJWTValidator(config.JWTConfig{Secret: "test-secret"})
	require.NoError(t, err, methodCtx)

	router := gin.New()
	apiGroup := router.Group("/api/v1", JWT(validator))
	apiGroup.POST("/teams/:id/invite", RBAC("owner", "admin"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	token := buildToken(t, "test-secret", "user-4", "owner")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/teams/111/invite", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code, methodCtx)
}

func TestRBACTasksCreate(t *testing.T) {
	const methodCtx = "middlewares.TestRBACTasksCreate"

	gin.SetMode(gin.TestMode)

	validator, err := NewJWTValidator(config.JWTConfig{Secret: "test-secret"})
	require.NoError(t, err, methodCtx)

	router := gin.New()
	apiGroup := router.Group("/api/v1", JWT(validator))
	apiGroup.POST("/tasks", RBAC("member", "admin", "owner"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	tests := []struct {
		role       string
		wantStatus int
	}{
		{role: "member", wantStatus: http.StatusOK},
		{role: "admin", wantStatus: http.StatusOK},
		{role: "owner", wantStatus: http.StatusOK},
		{role: "guest", wantStatus: http.StatusForbidden},
	}

	for _, tc := range tests {
		token := buildToken(t, "test-secret", "user-1", tc.role)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		require.Equal(t, tc.wantStatus, resp.Code, methodCtx)
	}
}
