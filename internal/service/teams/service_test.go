package teams

import (
	"context"
	"testing"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/suite"

	"github.com/Seraf-seraf/mkk_test/internal/api"
	"github.com/Seraf-seraf/mkk_test/internal/pkg/breaker"
	"github.com/Seraf-seraf/mkk_test/internal/pkg/mailer"
	repomysql "github.com/Seraf-seraf/mkk_test/internal/repo/mysql"
	"github.com/Seraf-seraf/mkk_test/internal/tests"
)

type TeamsSuite struct {
	tests.IntegrationSuite
	service    *Service
	mailer     *mailer.MockMailer
	ownerID    uuid.UUID
	adminID    uuid.UUID
	memberID   uuid.UUID
	outsiderID uuid.UUID
	teamID     uuid.UUID
}

func TestTeamsSuite(t *testing.T) {
	const methodCtx = "teams.TestTeamsSuite"

	t.Log(methodCtx)
	suite.Run(t, new(TeamsSuite))
}

func (s *TeamsSuite) SetupTest() {
	const methodCtx = "teams.TeamsSuite.SetupTest"

	s.TruncateTables(
		"task_comments",
		"task_history",
		"tasks",
		"team_invites",
		"team_members",
		"teams",
		"users",
	)

	s.ownerID = s.CreateUser("owner@example.com")
	s.adminID = s.CreateUser("admin@example.com")
	s.memberID = s.CreateUser("member@example.com")
	s.outsiderID = s.CreateUser("outsider@example.com")

	s.teamID = s.CreateTeam("Core Team", s.ownerID)
	s.AddTeamMember(s.teamID, s.ownerID, "owner")
	s.AddTeamMember(s.teamID, s.adminID, "admin")
	s.AddTeamMember(s.teamID, s.memberID, "member")

	s.mailer = mailer.NewMockMailer()
	cb, err := breaker.New("mailer")
	s.Require().NoError(err, methodCtx)

	teamsRepo := repomysql.NewTeamsRepo(s.DB)
	membersRepo := repomysql.NewTeamMembersRepo(s.DB)
	invitesRepo := repomysql.NewTeamInvitesRepo(s.DB)
	usersRepo := repomysql.NewUsersRepo(s.DB)

	s.service, err = NewService(s.DB, teamsRepo, membersRepo, invitesRepo, usersRepo, s.mailer, cb)
	s.Require().NoError(err, methodCtx)
}

func (s *TeamsSuite) TestCreateTeam() {
	const methodCtx = "teams.TeamsSuite.TestCreateTeam"

	ctx := context.Background()
	resp, err := s.service.CreateTeam(ctx, s.ownerID, api.CreateTeamRequest{Name: "New Team"})
	s.Require().NoError(err, methodCtx)
	s.Equal("New Team", resp.Name)
	s.Equal(s.ownerID, resp.CreatedBy)

	var role string
	err = s.DB.QueryRowContext(
		ctx,
		"SELECT role FROM team_members WHERE team_id = ? AND user_id = ?",
		resp.Id.String(),
		s.ownerID.String(),
	).Scan(&role)
	s.Require().NoError(err, methodCtx)
	s.Equal("owner", role)
}

func (s *TeamsSuite) TestListTeams() {
	const methodCtx = "teams.TeamsSuite.TestListTeams"

	ctx := context.Background()
	secondTeam := s.CreateTeam("Second Team", s.memberID)
	s.AddTeamMember(secondTeam, s.memberID, "owner")

	resp, err := s.service.ListTeams(ctx, s.memberID)
	s.Require().NoError(err, methodCtx)
	s.Len(resp.Items, 2)

	ids := map[uuid.UUID]struct{}{}
	for _, team := range resp.Items {
		ids[team.Id] = struct{}{}
	}

	_, hasFirst := ids[s.teamID]
	_, hasSecond := ids[secondTeam]
	s.True(hasFirst, methodCtx)
	s.True(hasSecond, methodCtx)
}

func (s *TeamsSuite) TestInviteByOwner() {
	const methodCtx = "teams.TeamsSuite.TestInviteByOwner"

	ctx := context.Background()
	resp, err := s.service.Invite(ctx, s.ownerID, s.teamID, api.InviteRequest{Email: "invitee@example.com"})
	s.Require().NoError(err, methodCtx)
	s.NotEmpty(resp.Code)
	s.Equal(openapi_types.Email("invitee@example.com"), resp.Email)

	var count int
	err = s.DB.QueryRowContext(
		ctx,
		"SELECT COUNT(*) FROM team_invites WHERE code = ?",
		resp.Code,
	).Scan(&count)
	s.Require().NoError(err, methodCtx)
	s.Equal(1, count)

	msgs := s.mailer.Messages()
	s.Require().Len(msgs, 1, methodCtx)
	s.Equal("invitee@example.com", msgs[0].To)
}

func (s *TeamsSuite) TestInviteForbidden() {
	const methodCtx = "teams.TeamsSuite.TestInviteForbidden"

	ctx := context.Background()
	_, err := s.service.Invite(ctx, s.memberID, s.teamID, api.InviteRequest{Email: "new@example.com"})
	s.Require().Error(err, methodCtx)
	s.ErrorIs(err, ErrForbidden)
}

func (s *TeamsSuite) TestInviteForbiddenOutsider() {
	const methodCtx = "teams.TeamsSuite.TestInviteForbiddenOutsider"

	ctx := context.Background()
	_, err := s.service.Invite(ctx, s.outsiderID, s.teamID, api.InviteRequest{Email: "new@example.com"})
	s.Require().Error(err, methodCtx)
	s.ErrorIs(err, ErrForbidden)
}

func (s *TeamsSuite) TestInviteTeamNotFound() {
	const methodCtx = "teams.TeamsSuite.TestInviteTeamNotFound"

	ctx := context.Background()
	_, err := s.service.Invite(ctx, s.ownerID, uuid.New(), api.InviteRequest{Email: "new@example.com"})
	s.Require().Error(err, methodCtx)
	s.ErrorIs(err, ErrNotFound)
}

func (s *TeamsSuite) TestInviteAlreadyMember() {
	const methodCtx = "teams.TeamsSuite.TestInviteAlreadyMember"

	ctx := context.Background()
	_, err := s.service.Invite(ctx, s.ownerID, s.teamID, api.InviteRequest{Email: "member@example.com"})
	s.Require().Error(err, methodCtx)
	s.ErrorIs(err, ErrAlreadyMember)
}

func (s *TeamsSuite) TestAcceptInvite() {
	const methodCtx = "teams.TeamsSuite.TestAcceptInvite"

	ctx := context.Background()
	inviteUserID := s.CreateUser("invitee@example.com")
	_, code := s.CreateInvite(s.teamID, s.ownerID, "invitee@example.com", "")

	resp, err := s.service.AcceptInvite(ctx, inviteUserID, api.AcceptInviteRequest{Code: code})
	s.Require().NoError(err, methodCtx)
	s.Equal(s.teamID, resp.TeamId)
	s.Equal(inviteUserID, resp.UserId)
	s.Equal(api.TeamMemberRole("member"), resp.Role)

	var count int
	err = s.DB.QueryRowContext(
		ctx,
		"SELECT COUNT(*) FROM team_members WHERE team_id = ? AND user_id = ?",
		s.teamID.String(),
		inviteUserID.String(),
	).Scan(&count)
	s.Require().NoError(err, methodCtx)
	s.Equal(1, count)

	err = s.DB.QueryRowContext(
		ctx,
		"SELECT COUNT(*) FROM team_invites WHERE code = ?",
		code,
	).Scan(&count)
	s.Require().NoError(err, methodCtx)
	s.Equal(0, count)
}

func (s *TeamsSuite) TestAcceptInviteWrongEmail() {
	const methodCtx = "teams.TeamsSuite.TestAcceptInviteWrongEmail"

	ctx := context.Background()
	_, code := s.CreateInvite(s.teamID, s.ownerID, "invitee@example.com", "")

	_, err := s.service.AcceptInvite(ctx, s.outsiderID, api.AcceptInviteRequest{Code: code})
	s.Require().Error(err, methodCtx)
	s.ErrorIs(err, ErrInviteEmailMismatch)
}

func (s *TeamsSuite) TestAcceptInviteNotFound() {
	const methodCtx = "teams.TeamsSuite.TestAcceptInviteNotFound"

	ctx := context.Background()
	_, err := s.service.AcceptInvite(ctx, s.outsiderID, api.AcceptInviteRequest{Code: "missing"})
	s.Require().Error(err, methodCtx)
	s.ErrorIs(err, ErrInviteNotFound)
}

func (s *TeamsSuite) TestInviteEmailExistsButNotUser() {
	const methodCtx = "teams.TeamsSuite.TestInviteEmailExistsButNotUser"

	ctx := context.Background()
	_, err := s.service.Invite(ctx, s.adminID, s.teamID, api.InviteRequest{Email: "nonexistent@example.com"})
	s.Require().NoError(err, methodCtx)

	var count int
	err = s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM team_invites WHERE email = ?", "nonexistent@example.com").Scan(&count)
	s.Require().NoError(err, methodCtx)
	s.Equal(1, count)
}

func (s *TeamsSuite) TestListTeamsEmpty() {
	const methodCtx = "teams.TeamsSuite.TestListTeamsEmpty"

	ctx := context.Background()
	userID := s.CreateUser("empty@example.com")

	resp, err := s.service.ListTeams(ctx, userID)
	s.Require().NoError(err, methodCtx)
	s.Empty(resp.Items)
}

func (s *TeamsSuite) TearDownTest() {
	const methodCtx = "teams.TeamsSuite.TearDownTest"

	if s.T() != nil {
		s.T().Log(methodCtx)
	}
}
