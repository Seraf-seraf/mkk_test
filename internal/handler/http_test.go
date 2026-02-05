package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/Seraf-seraf/mkk_test/internal/api"
	appmw "github.com/Seraf-seraf/mkk_test/internal/app/middlewares"
	"github.com/Seraf-seraf/mkk_test/internal/pkg/breaker"
	"github.com/Seraf-seraf/mkk_test/internal/pkg/cache"
	httpserver "github.com/Seraf-seraf/mkk_test/internal/pkg/http"
	"github.com/Seraf-seraf/mkk_test/internal/pkg/mailer"
	repomysql "github.com/Seraf-seraf/mkk_test/internal/repo/mysql"
	"github.com/Seraf-seraf/mkk_test/internal/service/auth"
	"github.com/Seraf-seraf/mkk_test/internal/service/comments"
	"github.com/Seraf-seraf/mkk_test/internal/service/reports"
	"github.com/Seraf-seraf/mkk_test/internal/service/tasks"
	"github.com/Seraf-seraf/mkk_test/internal/service/teams"
	"github.com/Seraf-seraf/mkk_test/internal/tests"
)

type HTTPSuite struct {
	tests.IntegrationSuite
}

func TestHTTPSuite(t *testing.T) {
	const methodCtx = "handler.TestHTTPSuite"

	t.Log(methodCtx)
	suite.Run(t, new(HTTPSuite))
}

func (s *HTTPSuite) SetupSuite() {
	const methodCtx = "handler.HTTPSuite.SetupSuite"

	s.IntegrationSuite.SetupSuite()
	s.startServer()
	s.waitForServer(methodCtx)
}

func (s *HTTPSuite) startServer() {
	const methodCtx = "handler.HTTPSuite.startServer"

	tasksCache, err := cache.NewTasksCache(s.Redis)
	require.NoError(s.T(), err, methodCtx)

	usersRepo := repomysql.NewUsersRepo(s.DB)
	teamsRepo := repomysql.NewTeamsRepo(s.DB)
	membersRepo := repomysql.NewTeamMembersRepo(s.DB)
	invitesRepo := repomysql.NewTeamInvitesRepo(s.DB)
	tasksRepo := repomysql.NewTasksRepo(s.DB)
	historyRepo := repomysql.NewTaskHistoryRepo(s.DB)
	commentsRepo := repomysql.NewCommentsRepo(s.DB)
	reportsRepo := repomysql.NewReportsRepo(s.DB)

	cb, err := breaker.New("mailer")
	require.NoError(s.T(), err, methodCtx)

	mailerSvc := mailer.NewMockMailer()

	authSvc, err := auth.NewService(usersRepo, membersRepo, mailerSvc, cb, s.Config.Auth.JWT)
	require.NoError(s.T(), err, methodCtx)

	teamsSvc, err := teams.NewService(s.DB, teamsRepo, membersRepo, invitesRepo, usersRepo, mailerSvc, cb)
	require.NoError(s.T(), err, methodCtx)

	tasksSvc, err := tasks.NewService(s.DB, tasksRepo, membersRepo, historyRepo, tasksCache)
	require.NoError(s.T(), err, methodCtx)

	commentsSvc, err := comments.NewService(commentsRepo, tasksRepo, membersRepo)
	require.NoError(s.T(), err, methodCtx)

	reportsSvc, err := reports.NewService(reportsRepo)
	require.NoError(s.T(), err, methodCtx)

	handlerSvc, err := New(authSvc, teamsSvc, tasksSvc, commentsSvc, reportsSvc)
	require.NoError(s.T(), err, methodCtx)

	validator, err := appmw.OapiRequestValidator("api/openapi.yml")
	require.NoError(s.T(), err, methodCtx)

	jwtValidator, err := appmw.NewJWTValidator(s.Config.Auth.JWT)
	require.NoError(s.T(), err, methodCtx)

	publicMW := []gin.HandlerFunc{validator}
	apiMW := []gin.HandlerFunc{validator, appmw.JWT(jwtValidator)}

	errorHandler := func(c *gin.Context, err error, statusCode int) {
		c.JSON(statusCode, api.ErrorResponse{Error: err.Error()})
	}

	wrapper := api.ServerInterfaceWrapper{
		Handler:      handlerSvc,
		ErrorHandler: errorHandler,
	}

	opts := httpserver.ServerOptions{
		PublicGroupMW: publicMW,
		APIGroupMW:    apiMW,
		RegisterPublic: func(group *gin.RouterGroup) error {
			group.POST("/login", wrapper.PostApiV1Login)
			group.POST("/register", wrapper.PostApiV1Register)
			return nil
		},
		RegisterAPI: func(group *gin.RouterGroup) error {
			group.GET("/teams", wrapper.GetApiV1Teams)
			group.POST("/teams", wrapper.PostApiV1Teams)
			group.POST("/teams/invites/accept", wrapper.PostApiV1TeamsInvitesAccept)
			group.POST("/teams/:id/invite", appmw.RBAC("owner", "admin"), wrapper.PostApiV1TeamsIdInvite)

			group.GET("/tasks", wrapper.GetApiV1Tasks)
			group.POST("/tasks", appmw.RBAC("member", "admin", "owner"), wrapper.PostApiV1Tasks)
			group.PUT("/tasks/:id", wrapper.PutApiV1TasksId)
			group.GET("/tasks/:id/history", wrapper.GetApiV1TasksIdHistory)
			group.GET("/tasks/:id/comments", wrapper.GetApiV1TasksIdComments)
			group.POST("/tasks/:id/comments", wrapper.PostApiV1TasksIdComments)
			group.PUT("/tasks/:id/comments/:comment_id", wrapper.PutApiV1TasksIdCommentsCommentId)
			group.DELETE("/tasks/:id/comments/:comment_id", wrapper.DeleteApiV1TasksIdCommentsCommentId)

			group.GET("/reports/team-summary", wrapper.GetApiV1ReportsTeamSummary)
			group.GET("/reports/top-creators", wrapper.GetApiV1ReportsTopCreators)
			group.GET("/reports/invalid-assignees", wrapper.GetApiV1ReportsInvalidAssignees)
			return nil
		},
	}

	s.StartHTTPServer(opts)
}

func (s *HTTPSuite) waitForServer(methodCtx string) {
	require.Eventually(s.T(), func() bool {
		resp, err := http.Get(s.ServerURL + "/openapi.json")
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 5*time.Second, 100*time.Millisecond, methodCtx)
}

func (s *HTTPSuite) doJSON(method string, path string, token string, body interface{}) (*http.Response, []byte) {
	const methodCtx = "handler.HTTPSuite.doJSON"

	var payload io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		require.NoError(s.T(), err, methodCtx)
		payload = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, s.ServerURL+path, payload)
	require.NoError(s.T(), err, methodCtx)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	require.NoError(s.T(), err, methodCtx)
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	require.NoError(s.T(), err, methodCtx)

	return resp, data
}

func (s *HTTPSuite) buildToken(subject string, role string) string {
	const methodCtx = "handler.HTTPSuite.buildToken"

	claims := struct {
		Role string `json:"role"`
		jwt.RegisteredClaims
	}{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.Config.Auth.JWT.Secret))
	require.NoError(s.T(), err, methodCtx)

	return signed
}

func (s *HTTPSuite) TestRegisterAndLogin() {
	const methodCtx = "handler.HTTPSuite.TestRegisterAndLogin"

	regReq := api.RegisterRequest{Email: "http-user@example.com", Password: "secret123"}
	resp, body := s.doJSON(http.MethodPost, "/api/v1/register", "", regReq)
	require.Equal(s.T(), http.StatusCreated, resp.StatusCode, methodCtx)

	var user api.User
	require.NoError(s.T(), json.Unmarshal(body, &user), methodCtx)
	require.Equal(s.T(), regReq.Email, user.Email, methodCtx)

	loginReq := api.LoginRequest{Email: regReq.Email, Password: regReq.Password}
	resp, body = s.doJSON(http.MethodPost, "/api/v1/login", "", loginReq)
	require.Equal(s.T(), http.StatusOK, resp.StatusCode, methodCtx)

	var authResp api.AuthResponse
	require.NoError(s.T(), json.Unmarshal(body, &authResp), methodCtx)
	require.NotEmpty(s.T(), authResp.Token, methodCtx)
}

func (s *HTTPSuite) TestTasksAndCommentsFlow() {
	const methodCtx = "handler.HTTPSuite.TestTasksAndCommentsFlow"

	s.TruncateTables(
		"task_comments",
		"task_history",
		"tasks",
		"team_invites",
		"team_members",
		"teams",
		"users",
	)

	userID := s.CreateUser("flow-user@example.com")
	teamID := s.CreateTeam("Flow Team", userID)
	s.AddTeamMember(teamID, userID, "owner")

	token := s.buildToken(userID.String(), "member")

	createTask := api.CreateTaskRequest{
		TeamId: api.UUID(teamID),
		Title:  "Task 1",
	}

	resp, body := s.doJSON(http.MethodPost, "/api/v1/tasks", token, createTask)
	require.Equal(s.T(), http.StatusCreated, resp.StatusCode, methodCtx)

	var task api.Task
	require.NoError(s.T(), json.Unmarshal(body, &task), methodCtx)
	require.Equal(s.T(), "Task 1", task.Title, methodCtx)

	listPath := fmt.Sprintf("/api/v1/tasks?team_id=%s", teamID.String())
	resp, body = s.doJSON(http.MethodGet, listPath, token, nil)
	require.Equal(s.T(), http.StatusOK, resp.StatusCode, methodCtx)

	var listResp api.TasksListResponse
	require.NoError(s.T(), json.Unmarshal(body, &listResp), methodCtx)
	require.GreaterOrEqual(s.T(), len(listResp.Items), 1, methodCtx)

	commentReq := api.CreateCommentRequest{Body: "Комментарий"}
	commentPath := fmt.Sprintf("/api/v1/tasks/%s/comments", task.Id.String())
	resp, body = s.doJSON(http.MethodPost, commentPath, token, commentReq)
	require.Equal(s.T(), http.StatusCreated, resp.StatusCode, methodCtx)

	var comment api.Comment
	require.NoError(s.T(), json.Unmarshal(body, &comment), methodCtx)
	require.Equal(s.T(), "Комментарий", comment.Body, methodCtx)
}

func (s *HTTPSuite) TestInviteRBAC() {
	const methodCtx = "handler.HTTPSuite.TestInviteRBAC"

	s.TruncateTables(
		"task_comments",
		"task_history",
		"tasks",
		"team_invites",
		"team_members",
		"teams",
		"users",
	)

	ownerID := s.CreateUser("owner-http@example.com")
	teamID := s.CreateTeam("Invite Team", ownerID)
	s.AddTeamMember(teamID, ownerID, "owner")

	memberToken := s.buildToken(ownerID.String(), "member")
	inviteReq := api.InviteRequest{Email: "invitee@example.com"}
	invitePath := fmt.Sprintf("/api/v1/teams/%s/invite", teamID.String())

	resp, _ := s.doJSON(http.MethodPost, invitePath, memberToken, inviteReq)
	require.Equal(s.T(), http.StatusForbidden, resp.StatusCode, methodCtx)

	ownerToken := s.buildToken(ownerID.String(), "owner")
	resp, _ = s.doJSON(http.MethodPost, invitePath, ownerToken, inviteReq)
	require.Equal(s.T(), http.StatusCreated, resp.StatusCode, methodCtx)
}

func (s *HTTPSuite) TestProtectedRequiresAuth() {
	const methodCtx = "handler.HTTPSuite.TestProtectedRequiresAuth"

	resp, _ := s.doJSON(http.MethodGet, "/api/v1/teams", "", nil)
	require.Equal(s.T(), http.StatusUnauthorized, resp.StatusCode, methodCtx)
}
