-- +goose Up
CREATE TABLE team_invites (
  id CHAR(36) NOT NULL,
  team_id CHAR(36) NOT NULL,
  email VARCHAR(255) NOT NULL,
  inviter_id CHAR(36) NOT NULL,
  code VARCHAR(128) NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_team_invites_code (code),
  CONSTRAINT fk_team_invites_team FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE,
  CONSTRAINT fk_team_invites_inviter FOREIGN KEY (inviter_id) REFERENCES users(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
DROP TABLE IF EXISTS team_invites;
