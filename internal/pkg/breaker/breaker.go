package breaker

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/sony/gobreaker"
)

const (
	defaultMinRequests = 20
	defaultFailureRate = 0.5
)

// Breaker интерфейс circuit breaker.
type Breaker interface {
	Execute(fn func() error) error
}

// GoBreaker реализует Breaker на основе gobreaker.
type GoBreaker struct {
	cb *gobreaker.CircuitBreaker
}

// DefaultSettings возвращает базовые настройки circuit breaker.
func DefaultSettings(name string) gobreaker.Settings {
	const methodCtx = "breaker.DefaultSettings"

	slog.Debug("инициализация настроек circuit breaker", slog.String("context", methodCtx))

	return gobreaker.Settings{
		Name:        name,
		MaxRequests: 3,
		Interval:    time.Minute,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			if counts.Requests < defaultMinRequests {
				return false
			}
			failureRate := float64(counts.TotalFailures) / float64(counts.Requests)
			return failureRate >= defaultFailureRate
		},
	}
}

// New создает circuit breaker с настройками по умолчанию.
func New(name string) (*GoBreaker, error) {
	const methodCtx = "breaker.New"

	if name == "" {
		return nil, fmt.Errorf("%s: name не задан", methodCtx)
	}

	return NewWithSettings(DefaultSettings(name))
}

// NewWithSettings создает circuit breaker с указанными настройками.
func NewWithSettings(settings gobreaker.Settings) (*GoBreaker, error) {
	const methodCtx = "breaker.NewWithSettings"

	if settings.Name == "" {
		return nil, fmt.Errorf("%s: name не задан", methodCtx)
	}

	return &GoBreaker{cb: gobreaker.NewCircuitBreaker(settings)}, nil
}

// Execute выполняет функцию через circuit breaker.
func (b *GoBreaker) Execute(fn func() error) error {
	const methodCtx = "breaker.GoBreaker.Execute"

	if b == nil || b.cb == nil {
		return fmt.Errorf("%s: breaker не инициализирован", methodCtx)
	}

	_, err := b.cb.Execute(func() (interface{}, error) {
		return nil, fn()
	})
	if err != nil {
		return fmt.Errorf("%s: %w", methodCtx, err)
	}

	return nil
}
