package tasks

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/Seraf-seraf/mkk_test/internal/api"
	repomysql "github.com/Seraf-seraf/mkk_test/internal/repo/mysql"
	"github.com/Seraf-seraf/mkk_test/internal/tests"
)

type TasksSuite struct {
	tests.IntegrationSuite
	service    *Service
	ownerID    uuid.UUID
	memberID   uuid.UUID
	outsiderID uuid.UUID
	teamID     uuid.UUID
	cache      *cacheSpy
}

func TestTasksSuite(t *testing.T) {
	const methodCtx = "tasks.TestTasksSuite"

	t.Log(methodCtx)
	suite.Run(t, new(TasksSuite))
}

func (s *TasksSuite) SetupTest() {
	const methodCtx = "tasks.TasksSuite.SetupTest"

	s.TruncateTables(
		"task_comments",
		"task_history",
		"tasks",
		"team_invites",
		"team_members",
		"teams",
		"users",
	)

	s.ownerID = s.CreateUser("owner-task@example.com")
	s.memberID = s.CreateUser("member-task@example.com")
	s.outsiderID = s.CreateUser("outsider-task@example.com")

	s.teamID = s.CreateTeam("Task Team", s.ownerID)
	s.AddTeamMember(s.teamID, s.ownerID, "owner")
	s.AddTeamMember(s.teamID, s.memberID, "member")

	s.cache = &cacheSpy{}
	tasksRepo := repomysql.NewTasksRepo(s.DB)
	membersRepo := repomysql.NewTeamMembersRepo(s.DB)
	historyRepo := repomysql.NewTaskHistoryRepo(s.DB)
	service, err := NewService(s.DB, tasksRepo, membersRepo, historyRepo, s.cache)
	s.Require().NoError(err, methodCtx)
	s.service = service
}

func (s *TasksSuite) TestCreateTaskMember() {
	const methodCtx = "tasks.TasksSuite.TestCreateTaskMember"

	ctx := context.Background()

	req := api.CreateTaskRequest{
		TeamId: s.teamID,
		Title:  "New Task",
	}

	resp, err := s.service.Create(ctx, s.memberID, req)
	s.Require().NoError(err, methodCtx)
	s.Equal("New Task", resp.Title)
	s.Equal(s.teamID, resp.TeamId)
	s.Equal(s.memberID, resp.CreatedBy)

	var status string
	var createdBy string
	err = s.DB.QueryRowContext(ctx, "SELECT status, created_by FROM tasks WHERE id = ?", resp.Id.String()).Scan(&status, &createdBy)
	s.Require().NoError(err, methodCtx)
	s.Equal("todo", status)
	s.Equal(s.memberID.String(), createdBy)
}

func (s *TasksSuite) TestCreateTaskForbidden() {
	const methodCtx = "tasks.TasksSuite.TestCreateTaskForbidden"

	ctx := context.Background()
	req := api.CreateTaskRequest{TeamId: s.teamID, Title: "Forbidden"}

	_, err := s.service.Create(ctx, s.outsiderID, req)
	s.Require().Error(err, methodCtx)
	s.ErrorIs(err, ErrForbidden)
}

func (s *TasksSuite) TestCreateTaskInvalidAssignee() {
	const methodCtx = "tasks.TasksSuite.TestCreateTaskInvalidAssignee"

	ctx := context.Background()
	assignee := api.UUID(s.outsiderID)
	req := api.CreateTaskRequest{
		TeamId:     s.teamID,
		Title:      "Task",
		AssigneeId: &assignee,
	}

	_, err := s.service.Create(ctx, s.ownerID, req)
	s.Require().Error(err, methodCtx)
	s.ErrorIs(err, ErrInvalidAssignee)
}

func (s *TasksSuite) TestCreateTaskWithAssignee() {
	const methodCtx = "tasks.TasksSuite.TestCreateTaskWithAssignee"

	ctx := context.Background()
	assignee := api.UUID(s.memberID)
	status := api.TaskStatus("in_progress")
	req := api.CreateTaskRequest{
		TeamId:     s.teamID,
		Title:      "Assigned",
		AssigneeId: &assignee,
		Status:     &status,
	}

	resp, err := s.service.Create(ctx, s.ownerID, req)
	s.Require().NoError(err, methodCtx)
	s.NotNil(resp.AssigneeId)
	s.Equal(s.memberID, *resp.AssigneeId)
}

func (s *TasksSuite) TestListTasksFilters() {
	const methodCtx = "tasks.TasksSuite.TestListTasksFilters"

	ctx := context.Background()
	assignee := s.memberID

	s.CreateTask(s.teamID, s.ownerID, &assignee, "todo", "todo-1", "")
	s.CreateTask(s.teamID, s.ownerID, &assignee, "done", "done-1", "")
	s.CreateTask(s.teamID, s.ownerID, nil, "todo", "todo-2", "")

	status := api.TaskStatus("todo")
	resp, err := s.service.List(ctx, s.memberID, s.teamID, &status, &assignee, 1, 10)
	s.Require().NoError(err, methodCtx)
	s.Len(resp.Items, 1)
	s.Equal("todo-1", resp.Items[0].Title)
}

func (s *TasksSuite) TestListTasksUsesCache() {
	const methodCtx = "tasks.TasksSuite.TestListTasksUsesCache"

	ctx := context.Background()

	s.CreateTask(s.teamID, s.ownerID, nil, "todo", "cached-1", "")
	s.CreateTask(s.teamID, s.ownerID, nil, "todo", "cached-2", "")

	resp, err := s.service.List(ctx, s.memberID, s.teamID, nil, nil, 1, 10)
	s.Require().NoError(err, methodCtx)
	s.Require().Greater(len(resp.Items), 0, methodCtx)

	s.Equal(1, s.cache.getCalls)
	s.Equal(1, s.cache.setCalls)
	s.NotEmpty(s.cache.lastKey)

	s.cache.hit = true
	s.cache.data = []api.Task{resp.Items[0]}

	respCached, err := s.service.List(ctx, s.memberID, s.teamID, nil, nil, 1, 10)
	s.Require().NoError(err, methodCtx)
	s.Len(respCached.Items, 1)
	s.Equal(resp.Items[0].Title, respCached.Items[0].Title)
	s.Equal(2, s.cache.getCalls)
	s.Equal(1, s.cache.setCalls)
}

func (s *TasksSuite) TestListTasksCacheKeyIncludesFilters() {
	const methodCtx = "tasks.TasksSuite.TestListTasksCacheKeyIncludesFilters"

	ctx := context.Background()

	s.CreateTask(s.teamID, s.ownerID, nil, "todo", "todo-1", "")
	s.CreateTask(s.teamID, s.ownerID, nil, "done", "done-1", "")

	statusTodo := api.TaskStatus("todo")
	_, err := s.service.List(ctx, s.memberID, s.teamID, &statusTodo, nil, 1, 10)
	s.Require().NoError(err, methodCtx)
	keyTodo := s.cache.lastKey

	statusDone := api.TaskStatus("done")
	_, err = s.service.List(ctx, s.memberID, s.teamID, &statusDone, nil, 1, 10)
	s.Require().NoError(err, methodCtx)
	keyDone := s.cache.lastKey

	s.NotEmpty(keyTodo)
	s.NotEmpty(keyDone)
	s.NotEqual(keyTodo, keyDone)
}

func (s *TasksSuite) TestListTasksForbidden() {
	const methodCtx = "tasks.TasksSuite.TestListTasksForbidden"

	ctx := context.Background()

	status := api.TaskStatus("todo")
	_, err := s.service.List(ctx, s.outsiderID, s.teamID, &status, nil, 1, 10)
	s.Require().Error(err, methodCtx)
	s.ErrorIs(err, ErrForbidden)
}

func (s *TasksSuite) TestUpdateTaskAndHistory() {
	const methodCtx = "tasks.TasksSuite.TestUpdateTaskAndHistory"

	ctx := context.Background()
	taskID := s.CreateTask(s.teamID, s.memberID, nil, "todo", "old", "old-desc")

	newStatus := api.TaskStatus("done")
	req := api.UpdateTaskRequest{
		Title:       ptrString("new"),
		Description: ptrString("new-desc"),
		Status:      &newStatus,
	}

	resp, err := s.service.Update(ctx, s.memberID, taskID, req)
	s.Require().NoError(err, methodCtx)
	s.Equal("new", resp.Title)
	s.Equal(api.TaskStatus("done"), resp.Status)

	var count int
	err = s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM task_history WHERE task_id = ?", taskID.String()).Scan(&count)
	s.Require().NoError(err, methodCtx)
	s.Equal(1, count)

	var completedAt sql.NullTime
	err = s.DB.QueryRowContext(ctx, "SELECT completed_at FROM tasks WHERE id = ?", taskID.String()).Scan(&completedAt)
	s.Require().NoError(err, methodCtx)
	s.True(completedAt.Valid, methodCtx)
	s.WithinDuration(time.Now().UTC(), completedAt.Time, time.Minute)
}

func (s *TasksSuite) TestUpdateTaskForbidden() {
	const methodCtx = "tasks.TasksSuite.TestUpdateTaskForbidden"

	ctx := context.Background()
	taskID := s.CreateTask(s.teamID, s.memberID, nil, "todo", "old", "")

	req := api.UpdateTaskRequest{Title: ptrString("new")}
	_, err := s.service.Update(ctx, s.outsiderID, taskID, req)
	s.Require().Error(err, methodCtx)
	s.ErrorIs(err, ErrForbidden)
}

func (s *TasksSuite) TestUpdateTaskNotFound() {
	const methodCtx = "tasks.TasksSuite.TestUpdateTaskNotFound"

	ctx := context.Background()
	req := api.UpdateTaskRequest{Title: ptrString("new")}

	_, err := s.service.Update(ctx, s.memberID, uuid.New(), req)
	s.Require().Error(err, methodCtx)
	s.ErrorIs(err, ErrNotFound)
}

func (s *TasksSuite) TestHistory() {
	const methodCtx = "tasks.TasksSuite.TestHistory"

	ctx := context.Background()
	taskID := s.CreateTask(s.teamID, s.memberID, nil, "todo", "old", "")
	s.CreateTaskHistory(taskID, s.memberID, "{}")
	s.CreateTaskHistory(taskID, s.ownerID, "{\"title\":{\"from\":\"old\",\"to\":\"new\"}}")

	resp, err := s.service.History(ctx, s.memberID, taskID)
	s.Require().NoError(err, methodCtx)
	s.Len(resp.Items, 2)
}

func (s *TasksSuite) TestHistoryForbidden() {
	const methodCtx = "tasks.TasksSuite.TestHistoryForbidden"

	ctx := context.Background()
	taskID := s.CreateTask(s.teamID, s.memberID, nil, "todo", "old", "")

	_, err := s.service.History(ctx, s.outsiderID, taskID)
	s.Require().Error(err, methodCtx)
	s.ErrorIs(err, ErrForbidden)
}

func ptrString(value string) *string {
	return &value
}

type cacheSpy struct {
	getCalls int
	setCalls int
	hit      bool
	data     []api.Task
	lastKey  string
}

func (c *cacheSpy) GetTeamTasks(ctx context.Context, teamID uuid.UUID, key string) ([]api.Task, bool, error) {
	c.getCalls++
	c.lastKey = key
	if c.hit {
		return c.data, true, nil
	}
	return nil, false, nil
}

func (c *cacheSpy) SetTeamTasks(ctx context.Context, teamID uuid.UUID, key string, tasks []api.Task) error {
	c.setCalls++
	c.lastKey = key
	c.data = tasks
	return nil
}
