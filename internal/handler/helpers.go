package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Seraf-seraf/mkk_test/internal/api"
	appmw "github.com/Seraf-seraf/mkk_test/internal/app/middlewares"
	"github.com/Seraf-seraf/mkk_test/internal/service/auth"
	"github.com/Seraf-seraf/mkk_test/internal/service/comments"
	"github.com/Seraf-seraf/mkk_test/internal/service/tasks"
	"github.com/Seraf-seraf/mkk_test/internal/service/teams"
)

func getUserID(c *gin.Context) (uuid.UUID, error) {
	const methodCtx = "handler.getUserID"

	val, ok := c.Get(appmw.ContextUserKey)
	if !ok {
		return uuid.UUID{}, fmt.Errorf("%s: пользователь не найден", methodCtx)
	}

	switch v := val.(type) {
	case string:
		id, err := uuid.Parse(v)
		if err != nil {
			return uuid.UUID{}, fmt.Errorf("%s: некорректный id пользователя", methodCtx)
		}
		return id, nil
	case uuid.UUID:
		return v, nil
	default:
		return uuid.UUID{}, fmt.Errorf("%s: некорректный тип пользователя", methodCtx)
	}
}

func bindJSON(c *gin.Context, target interface{}, methodCtx string) error {
	if err := c.ShouldBindJSON(target); err != nil {
		return fmt.Errorf("%s: ошибка разбора запроса", methodCtx)
	}
	return nil
}

func mapError(err error) (int, api.ErrorResponse) {
	if err == nil {
		return http.StatusOK, api.ErrorResponse{}
	}

	switch {
	case errors.Is(err, auth.ErrInvalidCredentials):
		return http.StatusUnauthorized, api.ErrorResponse{Error: err.Error()}
	case errors.Is(err, auth.ErrUserExists):
		return http.StatusBadRequest, api.ErrorResponse{Error: err.Error()}
	case errors.Is(err, teams.ErrForbidden), errors.Is(err, tasks.ErrForbidden), errors.Is(err, comments.ErrForbidden), errors.Is(err, teams.ErrInviteEmailMismatch):
		return http.StatusForbidden, api.ErrorResponse{Error: err.Error()}
	case errors.Is(err, teams.ErrNotFound), errors.Is(err, tasks.ErrNotFound), errors.Is(err, comments.ErrNotFound), errors.Is(err, teams.ErrInviteNotFound):
		return http.StatusNotFound, api.ErrorResponse{Error: err.Error()}
	case errors.Is(err, teams.ErrAlreadyMember), errors.Is(err, tasks.ErrInvalidAssignee):
		return http.StatusBadRequest, api.ErrorResponse{Error: err.Error()}
	default:
		return http.StatusInternalServerError, api.ErrorResponse{Error: "внутренняя ошибка сервера"}
	}
}

func writeError(c *gin.Context, err error, methodCtx string) {
	status, resp := mapError(err)
	if status == http.StatusInternalServerError {
		slog.Error("ошибка обработки запроса", slog.String("context", methodCtx), slog.String("error", err.Error()))
	}
	c.JSON(status, resp)
}
