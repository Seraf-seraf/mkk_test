package reports

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	repomysql "github.com/Seraf-seraf/mkk_test/internal/repo/mysql"
	"github.com/Seraf-seraf/mkk_test/internal/tests"
)

type ReportsSuite struct {
	tests.IntegrationSuite
	service  *Service
	ownerID  uuid.UUID
	teamAID  uuid.UUID
	teamBID  uuid.UUID
	memberID uuid.UUID
}

func TestReportsSuite(t *testing.T) {
	const methodCtx = "reports.TestReportsSuite"

	t.Log(methodCtx)
	suite.Run(t, new(ReportsSuite))
}

func (s *ReportsSuite) SetupTest() {
	const methodCtx = "reports.ReportsSuite.SetupTest"

	s.TruncateTables(
		"task_comments",
		"task_history",
		"tasks",
		"team_invites",
		"team_members",
		"teams",
		"users",
	)

	s.ownerID = s.CreateUser("owner-report@example.com")
	s.memberID = s.CreateUser("member-report@example.com")

	s.teamAID = s.CreateTeam("Alpha", s.ownerID)
	s.teamBID = s.CreateTeam("Beta", s.ownerID)

	s.AddTeamMember(s.teamAID, s.ownerID, "owner")
	s.AddTeamMember(s.teamAID, s.memberID, "member")
	s.AddTeamMember(s.teamBID, s.ownerID, "owner")

	repo := repomysql.NewReportsRepo(s.DB)
	service, err := NewService(repo)
	s.Require().NoError(err, methodCtx)
	s.service = service
}

func (s *ReportsSuite) TestTeamSummary() {
	const methodCtx = "reports.ReportsSuite.TestTeamSummary"

	ctx := context.Background()
	now := time.Now().UTC()

	done1 := now.Add(-24 * time.Hour)
	done2 := now.Add(-3 * 24 * time.Hour)
	done3 := now.Add(-10 * 24 * time.Hour)
	done4 := now.Add(-2 * 24 * time.Hour)

	s.insertTaskWithTimes(s.teamAID, s.ownerID, "done", now.Add(-24*time.Hour), &done1)
	s.insertTaskWithTimes(s.teamAID, s.ownerID, "done", now.Add(-3*24*time.Hour), &done2)
	s.insertTaskWithTimes(s.teamAID, s.ownerID, "done", now.Add(-10*24*time.Hour), &done3)
	s.insertTaskWithTimes(s.teamBID, s.ownerID, "done", now.Add(-2*24*time.Hour), &done4)

	resp, err := s.service.TeamSummary(ctx, s.ownerID)
	s.Require().NoError(err, methodCtx)

	teamCounts := map[string]struct {
		members int
		done7d  int
	}{}
	for _, item := range resp {
		teamCounts[item.TeamName] = struct {
			members int
			done7d  int
		}{members: item.MembersCount, done7d: item.DoneLast7d}
	}

	s.Require().Contains(teamCounts, "Alpha", methodCtx)
	s.Equal(2, teamCounts["Alpha"].members)
	s.Equal(2, teamCounts["Alpha"].done7d)

	s.Require().Contains(teamCounts, "Beta", methodCtx)
	s.Equal(1, teamCounts["Beta"].members)
	s.Equal(1, teamCounts["Beta"].done7d)
}

func (s *ReportsSuite) TestTopCreators() {
	const methodCtx = "reports.ReportsSuite.TestTopCreators"

	ctx := context.Background()
	month := "2025-01"

	creatorA := s.CreateUser("creator-a@example.com")
	creatorB := s.CreateUser("creator-b@example.com")
	creatorC := s.CreateUser("creator-c@example.com")
	creatorD := s.CreateUser("creator-d@example.com")

	s.AddTeamMember(s.teamAID, creatorA, "member")
	s.AddTeamMember(s.teamAID, creatorB, "member")
	s.AddTeamMember(s.teamAID, creatorC, "member")
	s.AddTeamMember(s.teamAID, creatorD, "member")

	jan := time.Date(2025, 1, 5, 10, 0, 0, 0, time.UTC)
	feb := time.Date(2025, 2, 5, 10, 0, 0, 0, time.UTC)

	s.insertTaskWithTimes(s.teamAID, creatorA, "todo", jan, nil)
	s.insertTaskWithTimes(s.teamAID, creatorA, "todo", jan, nil)
	s.insertTaskWithTimes(s.teamAID, creatorB, "todo", jan, nil)
	s.insertTaskWithTimes(s.teamAID, creatorB, "todo", jan, nil)
	s.insertTaskWithTimes(s.teamAID, creatorB, "todo", jan, nil)
	s.insertTaskWithTimes(s.teamAID, creatorC, "todo", jan, nil)
	s.insertTaskWithTimes(s.teamAID, creatorD, "todo", feb, nil)

	resp, err := s.service.TopCreators(ctx, s.ownerID, month)
	s.Require().NoError(err, methodCtx)
	s.Require().NotEmpty(resp)

	var alpha *struct {
		creators map[uuid.UUID]int
	}

	for _, item := range resp {
		if item.TeamId.String() == s.teamAID.String() {
			counts := map[uuid.UUID]int{}
			for _, creator := range item.Creators {
				counts[creator.UserId] = creator.TasksCreated
			}
			alpha = &struct{ creators map[uuid.UUID]int }{creators: counts}
			break
		}
	}

	s.Require().NotNil(alpha, methodCtx)
	s.Equal(3, len(alpha.creators))
	s.Equal(3, alpha.creators[creatorB])
	s.Equal(2, alpha.creators[creatorA])
	s.Equal(1, alpha.creators[creatorC])
}

func (s *ReportsSuite) TestInvalidAssignees() {
	const methodCtx = "reports.ReportsSuite.TestInvalidAssignees"

	ctx := context.Background()
	assignee := s.CreateUser("invalid-assignee@example.com")
	taskID := s.CreateTask(s.teamAID, s.ownerID, &assignee, "todo", "Bad", "")

	resp, err := s.service.InvalidAssignees(ctx, s.ownerID)
	s.Require().NoError(err, methodCtx)

	found := false
	for _, item := range resp {
		if item.TaskId.String() == taskID.String() {
			found = true
			break
		}
	}

	s.True(found, methodCtx)
}

func (s *ReportsSuite) insertTaskWithTimes(teamID uuid.UUID, creatorID uuid.UUID, status string, createdAt time.Time, completedAt *time.Time) {
	const methodCtx = "reports.ReportsSuite.insertTaskWithTimes"

	id := uuid.New()

	var completedValue sql.NullTime
	if completedAt != nil {
		completedValue = sql.NullTime{Time: *completedAt, Valid: true}
	}

	_, err := s.DB.ExecContext(
		context.Background(),
		"INSERT INTO tasks (id, team_id, title, description, status, assignee_id, created_by, created_at, updated_at, completed_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		id.String(),
		teamID.String(),
		"title",
		"",
		status,
		interface{}(nil),
		creatorID.String(),
		createdAt,
		createdAt,
		completedValue,
	)
	if err != nil {
		s.T().Fatalf("%s: %v", methodCtx, err)
	}
}
