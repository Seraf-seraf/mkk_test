-- +goose Up
CREATE TABLE task_history (
  id CHAR(36) NOT NULL,
  task_id CHAR(36) NOT NULL,
  changed_by CHAR(36) NOT NULL,
  changes JSON NOT NULL,
  changed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  CONSTRAINT fk_task_history_task FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE,
  CONSTRAINT fk_task_history_changed_by FOREIGN KEY (changed_by) REFERENCES users(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
DROP TABLE IF EXISTS task_history;
