package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Seraf-seraf/mkk_test/internal/api"
)

// PostApiV1Register регистрирует пользователя.
func (h *Handler) PostApiV1Register(c *gin.Context) {
	const methodCtx = "handler.PostApiV1Register"

	var req api.RegisterRequest
	if err := bindJSON(c, &req, methodCtx); err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{Error: err.Error()})
		return
	}

	user, err := h.auth.Register(c.Request.Context(), req)
	if err != nil {
		writeError(c, err, methodCtx)
		return
	}

	c.JSON(http.StatusCreated, user)
}

// PostApiV1Login выполняет вход.
func (h *Handler) PostApiV1Login(c *gin.Context) {
	const methodCtx = "handler.PostApiV1Login"

	var req api.LoginRequest
	if err := bindJSON(c, &req, methodCtx); err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{Error: err.Error()})
		return
	}

	resp, err := h.auth.Login(c.Request.Context(), req)
	if err != nil {
		writeError(c, err, methodCtx)
		return
	}

	c.JSON(http.StatusOK, resp)
}
