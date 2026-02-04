-- +goose Up
CREATE TABLE tasks (
  id CHAR(36) NOT NULL,
  team_id CHAR(36) NOT NULL,
  title VARCHAR(255) NOT NULL,
  description TEXT NULL,
  status ENUM('todo','in_progress','done') NOT NULL,
  assignee_id CHAR(36) NULL,
  created_by CHAR(36) NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  completed_at DATETIME NULL,
  PRIMARY KEY (id),
  CONSTRAINT fk_tasks_team FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE,
  CONSTRAINT fk_tasks_assignee FOREIGN KEY (assignee_id) REFERENCES users(id) ON DELETE SET NULL,
  CONSTRAINT fk_tasks_created_by FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
DROP TABLE IF EXISTS tasks;
