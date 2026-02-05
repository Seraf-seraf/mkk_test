package middlewares

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestJWTOptionalSetsContext(t *testing.T) {
	const methodCtx = "middlewares.TestJWTOptionalSetsContext"

	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer ok-token")
	ctx.Request = req

	mw := JWTOptional(func(token string) (interface{}, string, error) {
		require.Equal(t, "ok-token", token, methodCtx)
		return "user-1", "member", nil
	})

	mw(ctx)

	user, ok := ctx.Get(ContextUserKey)
	require.True(t, ok, methodCtx)
	require.Equal(t, "user-1", user, methodCtx)

	role, ok := ctx.Get(ContextRoleKey)
	require.True(t, ok, methodCtx)
	require.Equal(t, "member", role, methodCtx)
}

func TestJWTOptionalIgnoresInvalidToken(t *testing.T) {
	const methodCtx = "middlewares.TestJWTOptionalIgnoresInvalidToken"

	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer bad-token")
	ctx.Request = req

	mw := JWTOptional(func(token string) (interface{}, string, error) {
		return nil, "", errors.New("bad token")
	})

	mw(ctx)

	_, ok := ctx.Get(ContextUserKey)
	require.False(t, ok, methodCtx)
	_, ok = ctx.Get(ContextRoleKey)
	require.False(t, ok, methodCtx)
}

func TestJWTOptionalNoValidator(t *testing.T) {
	const methodCtx = "middlewares.TestJWTOptionalNoValidator"

	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("GET", "/", nil)

	mw := JWTOptional(nil)
	mw(ctx)

	_, ok := ctx.Get(ContextUserKey)
	require.False(t, ok, methodCtx)
}
