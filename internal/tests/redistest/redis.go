package redistest

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	rediscontainer "github.com/testcontainers/testcontainers-go/modules/redis"
)

// Start поднимает Redis контейнер и возвращает клиента и функцию очистки.
func Start(t *testing.T) (*redis.Client, func()) {
	const methodCtx = "redistest.Start"

	testcontainers.SkipIfProviderIsNotHealthy(t)

	ctx := context.Background()

	container, err := rediscontainer.Run(ctx, "redis:7")
	require.NoError(t, err, methodCtx)

	redisURL, err := container.ConnectionString(ctx)
	require.NoError(t, err, methodCtx)

	opts, err := redis.ParseURL(redisURL)
	require.NoError(t, err, methodCtx)

	client := redis.NewClient(opts)
	require.NoError(t, client.Ping(ctx).Err(), methodCtx)

	cleanup := func() {
		_ = client.Close()
		_ = container.Terminate(ctx)
	}

	return client, cleanup
}
