package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/Seraf-seraf/mkk_test/internal/config"
)

// New создает slog.Logger на основе конфигурации.
func New(cfg config.SloggerConfig) (*slog.Logger, error) {
	const methodCtx = "logger.New"

	level, err := parseLevel(cfg.Level)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", methodCtx, err)
	}

	writer, err := outputWriter(cfg.Output)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", methodCtx, err)
	}

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: cfg.AddSource,
	}

	var handler slog.Handler
	switch strings.ToLower(strings.TrimSpace(cfg.Format)) {
	case "", "json":
		handler = slog.NewJSONHandler(writer, opts)
	case "text":
		handler = slog.NewTextHandler(writer, opts)
	default:
		return nil, fmt.Errorf("%s: неподдерживаемый формат логов: %s", methodCtx, cfg.Format)
	}

	return slog.New(handler), nil
}

func parseLevel(level string) (slog.Level, error) {
	const methodCtx = "logger.parseLevel"

	switch strings.ToLower(strings.TrimSpace(level)) {
	case "", "info":
		return slog.LevelInfo, nil
	case "debug":
		return slog.LevelDebug, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("%s: неподдерживаемый уровень логов: %s", methodCtx, level)
	}
}

func outputWriter(output string) (io.Writer, error) {
	const methodCtx = "logger.outputWriter"

	switch strings.ToLower(strings.TrimSpace(output)) {
	case "", "stdout":
		return os.Stdout, nil
	case "stderr":
		return os.Stderr, nil
	default:
		return nil, fmt.Errorf("%s: неподдерживаемый вывод логов: %s", methodCtx, output)
	}
}
