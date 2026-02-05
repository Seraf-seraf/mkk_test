package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Seraf-seraf/mkk_test/internal/api"
)

// GetApiV1ReportsTeamSummary возвращает отчет по командам.
func (h *Handler) GetApiV1ReportsTeamSummary(c *gin.Context) {
	const methodCtx = "handler.GetApiV1ReportsTeamSummary"

	userID, err := getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, api.ErrorResponse{Error: err.Error()})
		return
	}

	resp, err := h.reports.TeamSummary(c.Request.Context(), userID)
	if err != nil {
		writeError(c, err, methodCtx)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetApiV1ReportsTopCreators возвращает топ создателей задач.
func (h *Handler) GetApiV1ReportsTopCreators(c *gin.Context, params api.GetApiV1ReportsTopCreatorsParams) {
	const methodCtx = "handler.GetApiV1ReportsTopCreators"

	userID, err := getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, api.ErrorResponse{Error: err.Error()})
		return
	}

	month := ""
	if params.Month != nil {
		month = *params.Month
	}
	if month == "" {
		month = time.Now().UTC().Format("2006-01")
	}

	resp, err := h.reports.TopCreators(c.Request.Context(), userID, month)
	if err != nil {
		writeError(c, err, methodCtx)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetApiV1ReportsInvalidAssignees возвращает задачи с некорректными исполнителями.
func (h *Handler) GetApiV1ReportsInvalidAssignees(c *gin.Context) {
	const methodCtx = "handler.GetApiV1ReportsInvalidAssignees"

	userID, err := getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, api.ErrorResponse{Error: err.Error()})
		return
	}

	resp, err := h.reports.InvalidAssignees(c.Request.Context(), userID)
	if err != nil {
		writeError(c, err, methodCtx)
		return
	}

	c.JSON(http.StatusOK, resp)
}
