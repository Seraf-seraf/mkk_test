package tests

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func (s *IntegrationSuite) CreateUser(email string) uuid.UUID {
	const methodCtx = "tests.IntegrationSuite.CreateUser"

	id := uuid.New()
	now := time.Now().UTC()

	hash, err := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.DefaultCost)
	s.Require().NoError(err, methodCtx)

	_, err = s.DB.ExecContext(
		s.ctx,
		"INSERT INTO users (id, email, password_hash, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		id.String(),
		email,
		string(hash),
		now,
		now,
	)
	s.Require().NoError(err, methodCtx)

	return id
}

func (s *IntegrationSuite) CreateTeam(name string, createdBy uuid.UUID) uuid.UUID {
	const methodCtx = "tests.IntegrationSuite.CreateTeam"

	id := uuid.New()
	now := time.Now().UTC()

	_, err := s.DB.ExecContext(
		s.ctx,
		"INSERT INTO teams (id, name, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		id.String(),
		name,
		createdBy.String(),
		now,
		now,
	)
	s.Require().NoError(err, methodCtx)

	return id
}

func (s *IntegrationSuite) AddTeamMember(teamID uuid.UUID, userID uuid.UUID, role string) {
	const methodCtx = "tests.IntegrationSuite.AddTeamMember"

	now := time.Now().UTC()

	_, err := s.DB.ExecContext(
		s.ctx,
		"INSERT INTO team_members (team_id, user_id, role, created_at) VALUES (?, ?, ?, ?)",
		teamID.String(),
		userID.String(),
		role,
		now,
	)
	s.Require().NoError(err, methodCtx)
}

func (s *IntegrationSuite) CreateInvite(teamID uuid.UUID, inviterID uuid.UUID, email string, code string) (uuid.UUID, string) {
	const methodCtx = "tests.IntegrationSuite.CreateInvite"

	id := uuid.New()
	if code == "" {
		code = uuid.NewString()
	}
	now := time.Now().UTC()

	_, err := s.DB.ExecContext(
		s.ctx,
		"INSERT INTO team_invites (id, team_id, email, inviter_id, code, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		id.String(),
		teamID.String(),
		email,
		inviterID.String(),
		code,
		now,
	)
	s.Require().NoError(err, methodCtx)

	return id, code
}

func (s *IntegrationSuite) CreateTask(teamID uuid.UUID, createdBy uuid.UUID, assigneeID *uuid.UUID, status string, title string, description string) uuid.UUID {
	const methodCtx = "tests.IntegrationSuite.CreateTask"

	id := uuid.New()
	now := time.Now().UTC()

	var assignee interface{}
	if assigneeID != nil {
		assignee = assigneeID.String()
	}

	var completedAt sql.NullTime
	if status == "done" {
		completedAt = sql.NullTime{Time: now, Valid: true}
	}

	_, err := s.DB.ExecContext(
		s.ctx,
		"INSERT INTO tasks (id, team_id, title, description, status, assignee_id, created_by, created_at, updated_at, completed_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		id.String(),
		teamID.String(),
		title,
		description,
		status,
		assignee,
		createdBy.String(),
		now,
		now,
		completedAt,
	)
	s.Require().NoError(err, methodCtx)

	return id
}

func (s *IntegrationSuite) CreateTaskHistory(taskID uuid.UUID, changedBy uuid.UUID, changes string) uuid.UUID {
	const methodCtx = "tests.IntegrationSuite.CreateTaskHistory"

	id := uuid.New()
	now := time.Now().UTC()
	if changes == "" {
		changes = "{}"
	}

	_, err := s.DB.ExecContext(
		s.ctx,
		"INSERT INTO task_history (id, task_id, changed_by, changes, changed_at) VALUES (?, ?, ?, ?, ?)",
		id.String(),
		taskID.String(),
		changedBy.String(),
		changes,
		now,
	)
	s.Require().NoError(err, methodCtx)

	return id
}

func (s *IntegrationSuite) CreateComment(taskID uuid.UUID, userID uuid.UUID, body string) uuid.UUID {
	const methodCtx = "tests.IntegrationSuite.CreateComment"

	id := uuid.New()
	now := time.Now().UTC()

	_, err := s.DB.ExecContext(
		s.ctx,
		"INSERT INTO task_comments (id, task_id, user_id, body, created_at) VALUES (?, ?, ?, ?, ?)",
		id.String(),
		taskID.String(),
		userID.String(),
		body,
		now,
	)
	s.Require().NoError(err, methodCtx)

	return id
}
