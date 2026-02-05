package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
)

// TeamSummaryRecord описывает отчет по командам.
type TeamSummaryRecord struct {
	TeamID       uuid.UUID
	TeamName     string
	MembersCount int
	DoneLast7d   int
}

// TopCreatorRecord описывает топ пользователей.
type TopCreatorRecord struct {
	TeamID       uuid.UUID
	TeamName     string
	UserID       uuid.UUID
	TasksCreated int
}

// InvalidAssigneeRecord описывает задачу с некорректным исполнителем.
type InvalidAssigneeRecord struct {
	TaskID     uuid.UUID
	TeamID     uuid.UUID
	AssigneeID uuid.UUID
}

// ReportsRepo реализует запросы отчетов.
type ReportsRepo struct {
	db *sql.DB
}

// NewReportsRepo создает репозиторий отчетов.
func NewReportsRepo(db *sql.DB) *ReportsRepo {
	const methodCtx = "repo.NewReportsRepo"

	slog.Debug("инициализация репозитория отчетов", slog.String("context", methodCtx))

	return &ReportsRepo{db: db}
}

// TeamSummary возвращает отчет по командам.
func (r *ReportsRepo) TeamSummary(ctx context.Context) ([]TeamSummaryRecord, error) {
	const methodCtx = "repo.ReportsRepo.TeamSummary"

	if r == nil || r.db == nil {
		return nil, fmt.Errorf("%s: репозиторий не инициализирован", methodCtx)
	}

	rows, err := r.db.QueryContext(
		ctx,
		`SELECT t.id, t.name,
			COUNT(DISTINCT tm.user_id) AS members_count,
			COUNT(DISTINCT CASE WHEN tk.status = 'done' AND tk.completed_at >= DATE_SUB(NOW(), INTERVAL 7 DAY) THEN tk.id END) AS done_last_7d
		FROM teams t
		LEFT JOIN team_members tm ON tm.team_id = t.id
		LEFT JOIN tasks tk ON tk.team_id = t.id
		GROUP BY t.id, t.name
		ORDER BY t.name`,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", methodCtx, err)
	}
	defer rows.Close()

	var items []TeamSummaryRecord
	for rows.Next() {
		var idStr string
		var name string
		var membersCount int
		var doneLast7d int
		if err := rows.Scan(&idStr, &name, &membersCount, &doneLast7d); err != nil {
			return nil, fmt.Errorf("%s: %w", methodCtx, err)
		}
		teamID, err := uuid.Parse(idStr)
		if err != nil {
			return nil, fmt.Errorf("%s: некорректный id команды", methodCtx)
		}
		items = append(items, TeamSummaryRecord{
			TeamID:       teamID,
			TeamName:     name,
			MembersCount: membersCount,
			DoneLast7d:   doneLast7d,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", methodCtx, err)
	}

	return items, nil
}

// TopCreators возвращает топ пользователей по созданным задачам.
func (r *ReportsRepo) TopCreators(ctx context.Context, month string) ([]TopCreatorRecord, error) {
	const methodCtx = "repo.ReportsRepo.TopCreators"

	if r == nil || r.db == nil {
		return nil, fmt.Errorf("%s: репозиторий не инициализирован", methodCtx)
	}

	rows, err := r.db.QueryContext(
		ctx,
		`SELECT team_id, team_name, user_id, tasks_created
		FROM (
			SELECT t.id AS team_id, t.name AS team_name, tc.user_id, tc.tasks_created,
				ROW_NUMBER() OVER (PARTITION BY t.id ORDER BY tc.tasks_created DESC) AS rn
			FROM teams t
			JOIN (
				SELECT team_id, created_by AS user_id, COUNT(*) AS tasks_created
				FROM tasks
				WHERE DATE_FORMAT(created_at, '%Y-%m') = ?
				GROUP BY team_id, created_by
			) tc ON tc.team_id = t.id
		) ranked
		WHERE rn <= 3
		ORDER BY team_id, tasks_created DESC`,
		month,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", methodCtx, err)
	}
	defer rows.Close()

	var items []TopCreatorRecord
	for rows.Next() {
		var teamIDStr, userIDStr, teamName string
		var tasksCreated int
		if err := rows.Scan(&teamIDStr, &teamName, &userIDStr, &tasksCreated); err != nil {
			return nil, fmt.Errorf("%s: %w", methodCtx, err)
		}
		teamID, err := uuid.Parse(teamIDStr)
		if err != nil {
			return nil, fmt.Errorf("%s: некорректный id команды", methodCtx)
		}
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			return nil, fmt.Errorf("%s: некорректный id пользователя", methodCtx)
		}
		items = append(items, TopCreatorRecord{
			TeamID:       teamID,
			TeamName:     teamName,
			UserID:       userID,
			TasksCreated: tasksCreated,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", methodCtx, err)
	}

	return items, nil
}

// InvalidAssignees возвращает задачи с некорректными исполнителями.
func (r *ReportsRepo) InvalidAssignees(ctx context.Context) ([]InvalidAssigneeRecord, error) {
	const methodCtx = "repo.ReportsRepo.InvalidAssignees"

	if r == nil || r.db == nil {
		return nil, fmt.Errorf("%s: репозиторий не инициализирован", methodCtx)
	}

	rows, err := r.db.QueryContext(
		ctx,
		`SELECT t.id, t.team_id, t.assignee_id
		FROM tasks t
		LEFT JOIN team_members tm ON tm.team_id = t.team_id AND tm.user_id = t.assignee_id
		WHERE t.assignee_id IS NOT NULL AND tm.user_id IS NULL`,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", methodCtx, err)
	}
	defer rows.Close()

	var items []InvalidAssigneeRecord
	for rows.Next() {
		var taskIDStr, teamIDStr, assigneeIDStr string
		if err := rows.Scan(&taskIDStr, &teamIDStr, &assigneeIDStr); err != nil {
			return nil, fmt.Errorf("%s: %w", methodCtx, err)
		}
		taskID, err := uuid.Parse(taskIDStr)
		if err != nil {
			return nil, fmt.Errorf("%s: некорректный task_id", methodCtx)
		}
		teamID, err := uuid.Parse(teamIDStr)
		if err != nil {
			return nil, fmt.Errorf("%s: некорректный team_id", methodCtx)
		}
		assigneeID, err := uuid.Parse(assigneeIDStr)
		if err != nil {
			return nil, fmt.Errorf("%s: некорректный assignee_id", methodCtx)
		}
		items = append(items, InvalidAssigneeRecord{
			TaskID:     taskID,
			TeamID:     teamID,
			AssigneeID: assigneeID,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", methodCtx, err)
	}

	return items, nil
}
