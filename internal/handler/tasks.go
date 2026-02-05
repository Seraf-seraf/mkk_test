package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Seraf-seraf/mkk_test/internal/api"
)

// GetApiV1Tasks возвращает список задач.
func (h *Handler) GetApiV1Tasks(c *gin.Context, params api.GetApiV1TasksParams) {
	const methodCtx = "handler.GetApiV1Tasks"

	userID, err := getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, api.ErrorResponse{Error: err.Error()})
		return
	}

	teamID := params.TeamId

	var assigneeID *uuid.UUID
	if params.AssigneeId != nil {
		value := *params.AssigneeId
		assigneeID = &value
	}

	page := 0
	if params.Page != nil {
		page = *params.Page
	}
	perPage := 0
	if params.PerPage != nil {
		perPage = *params.PerPage
	}

	resp, err := h.tasks.List(c.Request.Context(), userID, teamID, params.Status, assigneeID, page, perPage)
	if err != nil {
		writeError(c, err, methodCtx)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// PostApiV1Tasks создает задачу.
func (h *Handler) PostApiV1Tasks(c *gin.Context) {
	const methodCtx = "handler.PostApiV1Tasks"

	userID, err := getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, api.ErrorResponse{Error: err.Error()})
		return
	}

	var req api.CreateTaskRequest
	if err := bindJSON(c, &req, methodCtx); err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{Error: err.Error()})
		return
	}

	resp, err := h.tasks.Create(c.Request.Context(), userID, req)
	if err != nil {
		writeError(c, err, methodCtx)
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// PutApiV1TasksId обновляет задачу.
func (h *Handler) PutApiV1TasksId(c *gin.Context, id api.TaskId) {
	const methodCtx = "handler.PutApiV1TasksId"

	userID, err := getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, api.ErrorResponse{Error: err.Error()})
		return
	}

	var req api.UpdateTaskRequest
	if err := bindJSON(c, &req, methodCtx); err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{Error: err.Error()})
		return
	}

	resp, err := h.tasks.Update(c.Request.Context(), userID, uuid.UUID(id), req)
	if err != nil {
		writeError(c, err, methodCtx)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetApiV1TasksIdHistory возвращает историю задачи.
func (h *Handler) GetApiV1TasksIdHistory(c *gin.Context, id api.TaskId) {
	const methodCtx = "handler.GetApiV1TasksIdHistory"

	userID, err := getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, api.ErrorResponse{Error: err.Error()})
		return
	}

	resp, err := h.tasks.History(c.Request.Context(), userID, id)
	if err != nil {
		writeError(c, err, methodCtx)
		return
	}

	c.JSON(http.StatusOK, resp)
}
