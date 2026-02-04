-- +goose Up
CREATE TABLE task_comments (
  id CHAR(36) NOT NULL,
  task_id CHAR(36) NOT NULL,
  user_id CHAR(36) NOT NULL,
  body TEXT NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  CONSTRAINT fk_task_comments_task FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE,
  CONSTRAINT fk_task_comments_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
DROP TABLE IF EXISTS task_comments;
