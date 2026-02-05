package handler

import (
	"net/http"

	"github.com/Seraf-seraf/mkk_test/internal/api"
	"github.com/gin-gonic/gin"
)

// GetApiV1Teams возвращает список команд пользователя.
func (h *Handler) GetApiV1Teams(c *gin.Context) {
	const methodCtx = "handler.GetApiV1Teams"

	userID, err := getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, api.ErrorResponse{Error: err.Error()})
		return
	}

	resp, err := h.teams.ListTeams(c.Request.Context(), userID)
	if err != nil {
		writeError(c, err, methodCtx)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// PostApiV1Teams создает команду.
func (h *Handler) PostApiV1Teams(c *gin.Context) {
	const methodCtx = "handler.PostApiV1Teams"

	userID, err := getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, api.ErrorResponse{Error: err.Error()})
		return
	}

	var req api.CreateTeamRequest
	if err := bindJSON(c, &req, methodCtx); err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{Error: err.Error()})
		return
	}

	team, err := h.teams.CreateTeam(c.Request.Context(), userID, req)
	if err != nil {
		writeError(c, err, methodCtx)
		return
	}

	c.JSON(http.StatusCreated, team)
}

// PostApiV1TeamsIdInvite приглашает пользователя в команду.
func (h *Handler) PostApiV1TeamsIdInvite(c *gin.Context, id api.TeamId) {
	const methodCtx = "handler.PostApiV1TeamsIdInvite"

	userID, err := getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, api.ErrorResponse{Error: err.Error()})
		return
	}

	var req api.InviteRequest
	if err := bindJSON(c, &req, methodCtx); err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{Error: err.Error()})
		return
	}

	teamID := id
	resp, err := h.teams.Invite(c.Request.Context(), userID, teamID, req)
	if err != nil {
		writeError(c, err, methodCtx)
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// PostApiV1TeamsInvitesAccept принимает приглашение.
func (h *Handler) PostApiV1TeamsInvitesAccept(c *gin.Context) {
	const methodCtx = "handler.PostApiV1TeamsInvitesAccept"

	userID, err := getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, api.ErrorResponse{Error: err.Error()})
		return
	}

	var req api.AcceptInviteRequest
	if err := bindJSON(c, &req, methodCtx); err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{Error: err.Error()})
		return
	}

	resp, err := h.teams.AcceptInvite(c.Request.Context(), userID, req)
	if err != nil {
		writeError(c, err, methodCtx)
		return
	}

	c.JSON(http.StatusOK, resp)
}
