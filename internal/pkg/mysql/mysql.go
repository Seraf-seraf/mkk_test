package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/Seraf-seraf/mkk_test/internal/config"
)

const pingTimeout = 5 * time.Second

// Open открывает соединение с MySQL и применяет настройки пула.
func Open(cfg config.MySQLConfig) (*sql.DB, error) {
	const methodCtx = "mysql.Open"

	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?parseTime=true&multiStatements=true",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("%s: ошибка подключения к базе данных: %w", methodCtx, err)
	}

	if cfg.Pool.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.Pool.MaxOpenConns)
	}
	if cfg.Pool.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.Pool.MaxIdleConns)
	}
	if cfg.Pool.ConnMaxLifetimeSeconds > 0 {
		db.SetConnMaxLifetime(time.Duration(cfg.Pool.ConnMaxLifetimeSeconds) * time.Second)
	}
	if cfg.Pool.ConnMaxIdleTimeSeconds > 0 {
		db.SetConnMaxIdleTime(time.Duration(cfg.Pool.ConnMaxIdleTimeSeconds) * time.Second)
	}

	pingCtx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("%s: ошибка проверки подключения к базе данных: %w", methodCtx, err)
	}

	return db, nil
}
