package reports

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	"github.com/Seraf-seraf/mkk_test/internal/api"
	repomysql "github.com/Seraf-seraf/mkk_test/internal/repo/mysql"
)

// Service реализует бизнес-логику отчетов.
type Service struct {
	reports ReportsRepository
}

// ReportsRepository описывает запросы отчетов.
type ReportsRepository interface {
	TeamSummary(ctx context.Context) ([]repomysql.TeamSummaryRecord, error)
	TopCreators(ctx context.Context, month string) ([]repomysql.TopCreatorRecord, error)
	InvalidAssignees(ctx context.Context) ([]repomysql.InvalidAssigneeRecord, error)
}

// NewService создает сервис отчетов.
func NewService(reports ReportsRepository) (*Service, error) {
	const methodCtx = "reports.NewService"

	slog.Debug("инициализация сервиса отчетов", slog.String("context", methodCtx))

	if reports == nil {
		return nil, fmt.Errorf("%s: reports repo не задан", methodCtx)
	}

	return &Service{reports: reports}, nil
}

// TeamSummary возвращает отчет по командам.
func (s *Service) TeamSummary(ctx context.Context, _ uuid.UUID) ([]api.TeamSummary, error) {
	const methodCtx = "reports.Service.TeamSummary"

	slog.Debug("вызов отчета по командам", slog.String("context", methodCtx))

	records, err := s.reports.TeamSummary(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", methodCtx, err)
	}

	items := make([]api.TeamSummary, 0, len(records))
	for _, record := range records {
		items = append(items, api.TeamSummary{
			TeamId:       api.UUID(record.TeamID),
			TeamName:     record.TeamName,
			MembersCount: record.MembersCount,
			DoneLast7d:   record.DoneLast7d,
		})
	}

	return items, nil
}

// TopCreators возвращает топ создателей задач за месяц.
func (s *Service) TopCreators(ctx context.Context, _ uuid.UUID, month string) ([]api.TeamTopCreators, error) {
	const methodCtx = "reports.Service.TopCreators"

	slog.Debug("вызов отчета top creators", slog.String("context", methodCtx))

	if strings.TrimSpace(month) == "" {
		return nil, fmt.Errorf("%s: месяц не задан", methodCtx)
	}

	records, err := s.reports.TopCreators(ctx, month)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", methodCtx, err)
	}

	grouped := map[uuid.UUID]*api.TeamTopCreators{}
	for _, record := range records {
		entry, exists := grouped[record.TeamID]
		if !exists {
			entry = &api.TeamTopCreators{
				TeamId:   api.UUID(record.TeamID),
				TeamName: record.TeamName,
			}
			grouped[record.TeamID] = entry
		}
		entry.Creators = append(entry.Creators, api.UserTaskCount{
			UserId:       api.UUID(record.UserID),
			TasksCreated: record.TasksCreated,
		})
	}

	result := make([]api.TeamTopCreators, 0, len(grouped))
	for _, entry := range grouped {
		result = append(result, *entry)
	}

	return result, nil
}

// InvalidAssignees возвращает задачи с некорректными исполнителями.
func (s *Service) InvalidAssignees(ctx context.Context, _ uuid.UUID) ([]api.InvalidAssignee, error) {
	const methodCtx = "reports.Service.InvalidAssignees"

	slog.Debug("вызов отчета invalid assignees", slog.String("context", methodCtx))

	records, err := s.reports.InvalidAssignees(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", methodCtx, err)
	}

	items := make([]api.InvalidAssignee, 0, len(records))
	for _, record := range records {
		items = append(items, api.InvalidAssignee{
			TaskId:     api.UUID(record.TaskID),
			TeamId:     api.UUID(record.TeamID),
			AssigneeId: api.UUID(record.AssigneeID),
		})
	}

	return items, nil
}
