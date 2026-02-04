package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Seraf-seraf/mkk_test/internal/app"
	"github.com/Seraf-seraf/mkk_test/internal/config"
	"github.com/Seraf-seraf/mkk_test/internal/pkg/logger"
)

func main() {
	const methodCtx = "main.main"

	cfg, err := config.LoadDefault()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	logg, err := logger.New(cfg.Slogger)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	slog.SetDefault(logg)

	server, shutdown, err := app.New(cfg)
	if err != nil {
		logg.Error("ошибка инициализации приложения", slog.String("context", methodCtx), slog.String("error", err.Error()))
		os.Exit(1)
	}

	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case <-signalCtx.Done():
		logg.Info("останавливаем сервер...", slog.String("context", methodCtx))
	case err := <-serverErr:
		logg.Error("сервер остановлен с ошибкой", slog.String("context", methodCtx), slog.String("error", err.Error()))
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := shutdown(shutdownCtx); err != nil {
		logg.Error("ошибка остановки сервера", slog.String("context", methodCtx), slog.String("error", err.Error()))
	}
}
