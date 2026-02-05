package middlewares

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/Seraf-seraf/mkk_test/internal/config"
)

func TestJWTMiddlewareMissingToken(t *testing.T) {
	const methodCtx = "middlewares.TestJWTMiddlewareMissingToken"

	gin.SetMode(gin.TestMode)

	validator, err := NewJWTValidator(config.JWTConfig{Secret: "test-secret"})
	require.NoError(t, err, methodCtx)

	router := gin.New()
	router.GET("/protected", JWT(validator), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusUnauthorized, resp.Code, methodCtx)
}

func TestJWTMiddlewareInvalidFormat(t *testing.T) {
	const methodCtx = "middlewares.TestJWTMiddlewareInvalidFormat"

	gin.SetMode(gin.TestMode)

	validator, err := NewJWTValidator(config.JWTConfig{Secret: "test-secret"})
	require.NoError(t, err, methodCtx)

	router := gin.New()
	router.GET("/protected", JWT(validator), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Token abc")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusUnauthorized, resp.Code, methodCtx)
}

func TestJWTMiddlewareInvalidToken(t *testing.T) {
	const methodCtx = "middlewares.TestJWTMiddlewareInvalidToken"

	gin.SetMode(gin.TestMode)

	validator, err := NewJWTValidator(config.JWTConfig{Secret: "test-secret"})
	require.NoError(t, err, methodCtx)

	badToken := buildToken(t, "other-secret", "user-1", "member")

	router := gin.New()
	router.GET("/protected", JWT(validator), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+badToken)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusUnauthorized, resp.Code, methodCtx)
}

func TestJWTMiddlewareValidToken(t *testing.T) {
	const methodCtx = "middlewares.TestJWTMiddlewareValidToken"

	gin.SetMode(gin.TestMode)

	validator, err := NewJWTValidator(config.JWTConfig{Secret: "test-secret"})
	require.NoError(t, err, methodCtx)

	token := buildToken(t, "test-secret", "user-42", "admin")

	router := gin.New()
	router.GET("/protected", JWT(validator), func(c *gin.Context) {
		user, _ := c.Get(ContextUserKey)
		role, _ := c.Get(ContextRoleKey)
		c.JSON(http.StatusOK, gin.H{"user": user, "role": role})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code, methodCtx)

	var payload map[string]string
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload), methodCtx)
	require.Equal(t, "user-42", payload["user"], methodCtx)
	require.Equal(t, "admin", payload["role"], methodCtx)
}
