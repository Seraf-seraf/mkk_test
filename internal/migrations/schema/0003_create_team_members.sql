-- +goose Up
CREATE TABLE team_members (
  team_id CHAR(36) NOT NULL,
  user_id CHAR(36) NOT NULL,
  role ENUM('owner','admin','member') NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (team_id, user_id),
  CONSTRAINT fk_team_members_team FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE,
  CONSTRAINT fk_team_members_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
DROP TABLE IF EXISTS team_members;
