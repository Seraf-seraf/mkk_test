package cache

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Seraf-seraf/mkk_test/internal/api"
	"github.com/Seraf-seraf/mkk_test/internal/tests/redistest"
)

func TestTasksCacheTTL(t *testing.T) {
	const methodCtx = "cache.TestTasksCacheTTL"

	client, cleanup := redistest.Start(t)
	t.Cleanup(cleanup)

	cache, err := NewTasksCache(client)
	require.NoError(t, err, methodCtx)

	key := "tasks:ttl:test"
	items := []api.Task{
		{
			Id:        api.UUID(uuid.New()),
			TeamId:    api.UUID(uuid.New()),
			Title:     "title",
			Status:    api.TaskStatus("todo"),
			CreatedBy: api.UUID(uuid.New()),
			CreatedAt: time.Now().UTC(),
		},
	}

	err = cache.SetTeamTasks(context.Background(), uuid.New(), key, items)
	require.NoError(t, err, methodCtx)

	ttl, err := client.TTL(context.Background(), key).Result()
	require.NoError(t, err, methodCtx)
	require.Greater(t, ttl, 4*time.Minute, methodCtx)
	require.LessOrEqual(t, ttl, 5*time.Minute, methodCtx)
}
