package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Seraf-seraf/mkk_test/internal/api"
)

// GetApiV1TasksIdComments возвращает комментарии задачи.
func (h *Handler) GetApiV1TasksIdComments(c *gin.Context, id api.TaskId, params api.GetApiV1TasksIdCommentsParams) {
	const methodCtx = "handler.GetApiV1TasksIdComments"

	userID, err := getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, api.ErrorResponse{Error: err.Error()})
		return
	}

	page := 0
	if params.Page != nil {
		page = *params.Page
	}
	perPage := 0
	if params.PerPage != nil {
		perPage = *params.PerPage
	}

	resp, err := h.comments.List(c.Request.Context(), userID, id, page, perPage)
	if err != nil {
		writeError(c, err, methodCtx)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// PostApiV1TasksIdComments добавляет комментарий.
func (h *Handler) PostApiV1TasksIdComments(c *gin.Context, id api.TaskId) {
	const methodCtx = "handler.PostApiV1TasksIdComments"

	userID, err := getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, api.ErrorResponse{Error: err.Error()})
		return
	}

	var req api.CreateCommentRequest
	if err := bindJSON(c, &req, methodCtx); err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{Error: err.Error()})
		return
	}

	resp, err := h.comments.Create(c.Request.Context(), userID, id, req)
	if err != nil {
		writeError(c, err, methodCtx)
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// PutApiV1TasksIdCommentsCommentId обновляет комментарий.
func (h *Handler) PutApiV1TasksIdCommentsCommentId(c *gin.Context, id api.TaskId, commentId api.CommentId) {
	const methodCtx = "handler.PutApiV1TasksIdCommentsCommentId"

	userID, err := getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, api.ErrorResponse{Error: err.Error()})
		return
	}

	var req api.UpdateCommentRequest
	if err := bindJSON(c, &req, methodCtx); err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{Error: err.Error()})
		return
	}

	resp, err := h.comments.Update(c.Request.Context(), userID, id, commentId, req)
	if err != nil {
		writeError(c, err, methodCtx)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// DeleteApiV1TasksIdCommentsCommentId удаляет комментарий.
func (h *Handler) DeleteApiV1TasksIdCommentsCommentId(c *gin.Context, id api.TaskId, commentId api.CommentId) {
	const methodCtx = "handler.DeleteApiV1TasksIdCommentsCommentId"

	userID, err := getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, api.ErrorResponse{Error: err.Error()})
		return
	}

	if err := h.comments.Delete(c.Request.Context(), userID, id, commentId); err != nil {
		writeError(c, err, methodCtx)
		return
	}

	c.Status(http.StatusNoContent)
}
