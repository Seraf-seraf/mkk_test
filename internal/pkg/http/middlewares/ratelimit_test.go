package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/Seraf-seraf/mkk_test/internal/tests/redistest"
)

func TestRateLimitPerMinute(t *testing.T) {
	const methodCtx = "middlewares.TestRateLimitPerMinute"

	gin.SetMode(gin.TestMode)

	client, cleanup := redistest.Start(t)
	t.Cleanup(cleanup)

	limiter, err := NewRateLimiter(client)
	require.NoError(t, err, methodCtx)

	router := gin.New()
	router.Use(RateLimit(limiter))
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	for i := 0; i < rateLimitPerMinute; i++ {
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		require.Equal(t, http.StatusOK, resp.Code, methodCtx)
	}

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	require.Equal(t, http.StatusTooManyRequests, resp.Code, methodCtx)

	reqOther := httptest.NewRequest(http.MethodGet, "/ping", nil)
	reqOther.RemoteAddr = "5.6.7.8:1234"
	respOther := httptest.NewRecorder()
	router.ServeHTTP(respOther, reqOther)
	require.Equal(t, http.StatusOK, respOther.Code, methodCtx)
}
