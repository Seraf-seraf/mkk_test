package auth

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/suite"
	"golang.org/x/crypto/bcrypt"

	"github.com/Seraf-seraf/mkk_test/internal/api"
	"github.com/Seraf-seraf/mkk_test/internal/pkg/breaker"
	"github.com/Seraf-seraf/mkk_test/internal/pkg/mailer"
	repomysql "github.com/Seraf-seraf/mkk_test/internal/repo/mysql"
	"github.com/Seraf-seraf/mkk_test/internal/tests"
)

type AuthSuite struct {
	tests.IntegrationSuite
	repo    *repomysql.UsersRepo
	service *Service
	mailer  *mailer.MockMailer
}

func TestAuthSuite(t *testing.T) {
	const methodCtx = "auth.TestAuthSuite"

	t.Log(methodCtx)
	suite.Run(t, new(AuthSuite))
}

func (s *AuthSuite) SetupTest() {
	const methodCtx = "auth.AuthSuite.SetupTest"

	s.TruncateTables(
		"task_comments",
		"task_history",
		"tasks",
		"team_invites",
		"team_members",
		"teams",
		"users",
	)

	s.mailer = mailer.NewMockMailer()
	cb, err := breaker.New("mailer")
	s.Require().NoError(err, methodCtx)

	s.repo = repomysql.NewUsersRepo(s.DB)
	membersRepo := repomysql.NewTeamMembersRepo(s.DB)
	s.service, err = NewService(s.repo, membersRepo, s.mailer, cb, s.Config.Auth.JWT)
	s.Require().NoError(err, methodCtx)
}

func (s *AuthSuite) TestRegisterSuccess() {
	const methodCtx = "auth.AuthSuite.TestRegisterSuccess"

	ctx := context.Background()
	req := api.RegisterRequest{Email: "user@example.com", Password: "secret123"}

	user, err := s.service.Register(ctx, req)
	s.Require().NoError(err, methodCtx)
	s.Equal(req.Email, user.Email)
	s.NotZero(user.Id)
	s.NotZero(user.CreatedAt)

	record, err := s.repo.GetByEmail(ctx, string(req.Email))
	s.Require().NoError(err, methodCtx)
	s.NotEqual(req.Password, record.PasswordHash)
	s.Require().NoError(bcrypt.CompareHashAndPassword([]byte(record.PasswordHash), []byte(req.Password)), methodCtx)

	msgs := s.mailer.Messages()
	s.Require().Len(msgs, 1, methodCtx)
	s.Equal(string(req.Email), msgs[0].To)
}

func (s *AuthSuite) TestRegisterDuplicateEmail() {
	const methodCtx = "auth.AuthSuite.TestRegisterDuplicateEmail"

	ctx := context.Background()
	req := api.RegisterRequest{Email: "dup@example.com", Password: "secret123"}

	_, err := s.service.Register(ctx, req)
	s.Require().NoError(err, methodCtx)

	_, err = s.service.Register(ctx, req)
	s.Require().Error(err, methodCtx)
	s.ErrorIs(err, ErrUserExists)
}

func (s *AuthSuite) TestLoginSuccess() {
	const methodCtx = "auth.AuthSuite.TestLoginSuccess"

	ctx := context.Background()
	req := api.RegisterRequest{Email: "login@example.com", Password: "secret123"}

	user, err := s.service.Register(ctx, req)
	s.Require().NoError(err, methodCtx)

	resp, err := s.service.Login(ctx, api.LoginRequest{Email: req.Email, Password: req.Password})
	s.Require().NoError(err, methodCtx)
	s.NotEmpty(resp.Token)
	s.Equal(user.Email, resp.User.Email)

	claims := struct {
		Role string `json:"role"`
		jwt.RegisteredClaims
	}{}

	parsed, err := jwt.ParseWithClaims(resp.Token, &claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(s.Config.Auth.JWT.Secret), nil
	})
	s.Require().NoError(err, methodCtx)
	s.True(parsed.Valid)
	s.Equal(user.Id.String(), claims.Subject)
	s.Equal("member", claims.Role)

	now := time.Now().UTC()
	s.Require().NotNil(claims.ExpiresAt)
	exp := claims.ExpiresAt.Time
	s.True(exp.After(now.Add(29*time.Minute)), methodCtx)
	s.True(exp.Before(now.Add(31*time.Minute)), methodCtx)
}

func (s *AuthSuite) TestLoginInvalidPassword() {
	const methodCtx = "auth.AuthSuite.TestLoginInvalidPassword"

	ctx := context.Background()
	req := api.RegisterRequest{Email: "badpass@example.com", Password: "secret123"}

	_, err := s.service.Register(ctx, req)
	s.Require().NoError(err, methodCtx)

	_, err = s.service.Login(ctx, api.LoginRequest{Email: req.Email, Password: "wrong"})
	s.Require().Error(err, methodCtx)
	s.ErrorIs(err, ErrInvalidCredentials)
}

func (s *AuthSuite) TestLoginUnknownEmail() {
	const methodCtx = "auth.AuthSuite.TestLoginUnknownEmail"

	ctx := context.Background()
	_, err := s.service.Login(ctx, api.LoginRequest{Email: "none@example.com", Password: "secret123"})
	s.Require().Error(err, methodCtx)
	s.ErrorIs(err, ErrInvalidCredentials)
}
