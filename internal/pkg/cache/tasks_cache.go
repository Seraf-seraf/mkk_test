package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/Seraf-seraf/mkk_test/internal/api"
)

const tasksListTTL = 5 * time.Minute

// TasksCache хранит списки задач в Redis.
type TasksCache struct {
	client *redis.Client
}

// NewTasksCache создает Redis кеш задач.
func NewTasksCache(client *redis.Client) (*TasksCache, error) {
	const methodCtx = "cache.NewTasksCache"

	slog.Debug("инициализация кеша задач", slog.String("context", methodCtx))

	if client == nil {
		return nil, fmt.Errorf("%s: redis клиент не задан", methodCtx)
	}

	return &TasksCache{client: client}, nil
}

// GetTeamTasks возвращает список задач из кеша.
func (c *TasksCache) GetTeamTasks(ctx context.Context, _ uuid.UUID, key string) ([]api.Task, bool, error) {
	const methodCtx = "cache.TasksCache.GetTeamTasks"

	if c == nil || c.client == nil {
		return nil, false, fmt.Errorf("%s: кеш не инициализирован", methodCtx)
	}

	value, err := c.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("%s: %w", methodCtx, err)
	}

	var items []api.Task
	if err := json.Unmarshal([]byte(value), &items); err != nil {
		return nil, false, fmt.Errorf("%s: ошибка разбора кеша", methodCtx)
	}

	return items, true, nil
}

// SetTeamTasks сохраняет список задач в кеш.
func (c *TasksCache) SetTeamTasks(ctx context.Context, _ uuid.UUID, key string, tasks []api.Task) error {
	const methodCtx = "cache.TasksCache.SetTeamTasks"

	if c == nil || c.client == nil {
		return fmt.Errorf("%s: кеш не инициализирован", methodCtx)
	}

	data, err := json.Marshal(tasks)
	if err != nil {
		return fmt.Errorf("%s: ошибка сериализации кеша", methodCtx)
	}

	return c.client.Set(ctx, key, data, tasksListTTL).Err()
}
