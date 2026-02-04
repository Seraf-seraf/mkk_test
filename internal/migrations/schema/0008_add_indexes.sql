-- +goose Up
CREATE INDEX idx_teams_created_by ON teams (created_by);

CREATE INDEX idx_team_members_user_team ON team_members (user_id, team_id);
CREATE INDEX idx_team_members_team_user ON team_members (team_id, user_id);

CREATE INDEX idx_team_invites_team_id ON team_invites (team_id);
CREATE INDEX idx_team_invites_email ON team_invites (email);

CREATE INDEX idx_tasks_team_status_assignee_created ON tasks (team_id, status, assignee_id, created_at);
CREATE INDEX idx_tasks_team_creator_created ON tasks (team_id, created_by, created_at);
CREATE INDEX idx_tasks_team_assignee ON tasks (team_id, assignee_id);
CREATE INDEX idx_tasks_team_status_completed ON tasks (team_id, status, completed_at);

CREATE INDEX idx_task_history_task_changed ON task_history (task_id, changed_at);

CREATE INDEX idx_task_comments_task_created ON task_comments (task_id, created_at);

-- +goose Down
DROP INDEX idx_task_comments_task_created ON task_comments;

DROP INDEX idx_task_history_task_changed ON task_history;

DROP INDEX idx_tasks_team_status_completed ON tasks;
DROP INDEX idx_tasks_team_assignee ON tasks;
DROP INDEX idx_tasks_team_creator_created ON tasks;
DROP INDEX idx_tasks_team_status_assignee_created ON tasks;

DROP INDEX idx_team_invites_email ON team_invites;
DROP INDEX idx_team_invites_team_id ON team_invites;

DROP INDEX idx_team_members_team_user ON team_members;
DROP INDEX idx_team_members_user_team ON team_members;

DROP INDEX idx_teams_created_by ON teams;
