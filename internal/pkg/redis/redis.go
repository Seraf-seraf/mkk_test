package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/Seraf-seraf/mkk_test/internal/config"
)

const pingTimeout = 5 * time.Second

// New создает клиента Redis и проверяет соединение.
func New(cfg config.RedisConfig) (*redis.Client, error) {
	const methodCtx = "redis.New"

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	opts := &redis.Options{
		Addr: addr,
	}
	if cfg.Pool.Size > 0 {
		opts.PoolSize = cfg.Pool.Size
	}
	if cfg.Pool.TimeoutSeconds > 0 {
		opts.PoolTimeout = time.Duration(cfg.Pool.TimeoutSeconds) * time.Second
	}

	client := redis.NewClient(opts)
	pingCtx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()

	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("%s: ошибка подключения к Redis: %w", methodCtx, err)
	}

	return client, nil
}
