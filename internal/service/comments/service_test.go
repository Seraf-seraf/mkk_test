package comments

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/Seraf-seraf/mkk_test/internal/api"
	repomysql "github.com/Seraf-seraf/mkk_test/internal/repo/mysql"
	"github.com/Seraf-seraf/mkk_test/internal/tests"
)

type CommentsSuite struct {
	tests.IntegrationSuite
	service    *Service
	ownerID    uuid.UUID
	memberID   uuid.UUID
	outsiderID uuid.UUID
	teamID     uuid.UUID
	taskID     uuid.UUID
}

func TestCommentsSuite(t *testing.T) {
	const methodCtx = "comments.TestCommentsSuite"

	t.Log(methodCtx)
	suite.Run(t, new(CommentsSuite))
}

func (s *CommentsSuite) SetupTest() {
	const methodCtx = "comments.CommentsSuite.SetupTest"

	s.TruncateTables(
		"task_comments",
		"task_history",
		"tasks",
		"team_invites",
		"team_members",
		"teams",
		"users",
	)

	s.ownerID = s.CreateUser("owner-comment@example.com")
	s.memberID = s.CreateUser("member-comment@example.com")
	s.outsiderID = s.CreateUser("outsider-comment@example.com")

	s.teamID = s.CreateTeam("Comment Team", s.ownerID)
	s.AddTeamMember(s.teamID, s.ownerID, "owner")
	s.AddTeamMember(s.teamID, s.memberID, "member")

	s.taskID = s.CreateTask(s.teamID, s.ownerID, nil, "todo", "Task", "")

	commentsRepo := repomysql.NewCommentsRepo(s.DB)
	tasksRepo := repomysql.NewTasksRepo(s.DB)
	membersRepo := repomysql.NewTeamMembersRepo(s.DB)
	service, err := NewService(commentsRepo, tasksRepo, membersRepo)
	s.Require().NoError(err, methodCtx)
	s.service = service
}

func (s *CommentsSuite) TestCreateCommentMember() {
	const methodCtx = "comments.CommentsSuite.TestCreateCommentMember"

	ctx := context.Background()
	req := api.CreateCommentRequest{Body: "hello"}

	resp, err := s.service.Create(ctx, s.memberID, s.taskID, req)
	s.Require().NoError(err, methodCtx)
	s.Equal("hello", resp.Body)
	s.Equal(s.taskID, uuid.UUID(resp.TaskId))
	s.Equal(s.memberID, uuid.UUID(resp.UserId))
}

func (s *CommentsSuite) TestCreateCommentForbidden() {
	const methodCtx = "comments.CommentsSuite.TestCreateCommentForbidden"

	ctx := context.Background()
	_, err := s.service.Create(ctx, s.outsiderID, s.taskID, api.CreateCommentRequest{Body: "nope"})
	s.Require().Error(err, methodCtx)
	s.ErrorIs(err, ErrForbidden)
}

func (s *CommentsSuite) TestListComments() {
	const methodCtx = "comments.CommentsSuite.TestListComments"

	ctx := context.Background()
	s.CreateComment(s.taskID, s.ownerID, "first")
	s.CreateComment(s.taskID, s.memberID, "second")

	resp, err := s.service.List(ctx, s.memberID, s.taskID, 1, 10)
	s.Require().NoError(err, methodCtx)
	s.Len(resp.Items, 2)
}

func (s *CommentsSuite) TestListCommentsForbidden() {
	const methodCtx = "comments.CommentsSuite.TestListCommentsForbidden"

	ctx := context.Background()

	_, err := s.service.List(ctx, s.outsiderID, s.taskID, 1, 10)
	s.Require().Error(err, methodCtx)
	s.ErrorIs(err, ErrForbidden)
}

func (s *CommentsSuite) TestUpdateCommentAuthor() {
	const methodCtx = "comments.CommentsSuite.TestUpdateCommentAuthor"

	ctx := context.Background()
	commentID := s.CreateComment(s.taskID, s.memberID, "old")

	resp, err := s.service.Update(ctx, s.memberID, s.taskID, commentID, api.UpdateCommentRequest{Body: "new"})
	s.Require().NoError(err, methodCtx)
	s.Equal("new", resp.Body)
}

func (s *CommentsSuite) TestUpdateCommentNotAuthor() {
	const methodCtx = "comments.CommentsSuite.TestUpdateCommentNotAuthor"

	ctx := context.Background()
	commentID := s.CreateComment(s.taskID, s.ownerID, "old")

	_, err := s.service.Update(ctx, s.memberID, s.taskID, commentID, api.UpdateCommentRequest{Body: "new"})
	s.Require().Error(err, methodCtx)
	s.ErrorIs(err, ErrForbidden)
}

func (s *CommentsSuite) TestUpdateCommentForbidden() {
	const methodCtx = "comments.CommentsSuite.TestUpdateCommentForbidden"

	ctx := context.Background()
	commentID := s.CreateComment(s.taskID, s.memberID, "old")

	_, err := s.service.Update(ctx, s.outsiderID, s.taskID, commentID, api.UpdateCommentRequest{Body: "new"})
	s.Require().Error(err, methodCtx)
	s.ErrorIs(err, ErrForbidden)
}

func (s *CommentsSuite) TestDeleteCommentAuthor() {
	const methodCtx = "comments.CommentsSuite.TestDeleteCommentAuthor"

	ctx := context.Background()
	commentID := s.CreateComment(s.taskID, s.ownerID, "old")

	err := s.service.Delete(ctx, s.ownerID, s.taskID, commentID)
	s.Require().NoError(err, methodCtx)

	var count int
	err = s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM task_comments WHERE id = ?", commentID.String()).Scan(&count)
	s.Require().NoError(err, methodCtx)
	s.Equal(0, count)
}

func (s *CommentsSuite) TestDeleteCommentNotAuthor() {
	const methodCtx = "comments.CommentsSuite.TestDeleteCommentNotAuthor"

	ctx := context.Background()
	commentID := s.CreateComment(s.taskID, s.ownerID, "old")

	err := s.service.Delete(ctx, s.memberID, s.taskID, commentID)
	s.Require().Error(err, methodCtx)
	s.ErrorIs(err, ErrForbidden)
}

func (s *CommentsSuite) TestDeleteCommentForbidden() {
	const methodCtx = "comments.CommentsSuite.TestDeleteCommentForbidden"

	ctx := context.Background()
	commentID := s.CreateComment(s.taskID, s.ownerID, "old")

	err := s.service.Delete(ctx, s.outsiderID, s.taskID, commentID)
	s.Require().Error(err, methodCtx)
	s.ErrorIs(err, ErrForbidden)
}
