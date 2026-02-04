package mailer

import (
	"context"
	"fmt"

	"github.com/Seraf-seraf/mkk_test/internal/pkg/breaker"
)

// BreakerMailer оборачивает отправку писем в circuit breaker.
type BreakerMailer struct {
	breaker breaker.Breaker
	next    Mailer
}

// NewBreakerMailer создает обертку mailer-а с circuit breaker.
func NewBreakerMailer(next Mailer, breaker breaker.Breaker) (*BreakerMailer, error) {
	const methodCtx = "mailer.NewBreakerMailer"

	if next == nil {
		return nil, fmt.Errorf("%s: mailer не задан", methodCtx)
	}
	if breaker == nil {
		return nil, fmt.Errorf("%s: breaker не задан", methodCtx)
	}

	return &BreakerMailer{
		breaker: breaker,
		next:    next,
	}, nil
}

// Send выполняет отправку письма через circuit breaker.
func (m *BreakerMailer) Send(ctx context.Context, msg Message) error {
	const methodCtx = "mailer.BreakerMailer.Send"

	if m == nil || m.next == nil || m.breaker == nil {
		return fmt.Errorf("%s: mailer не инициализирован", methodCtx)
	}

	err := m.breaker.Execute(func() error {
		return m.next.Send(ctx, msg)
	})
	if err != nil {
		return fmt.Errorf("%s: %w", methodCtx, err)
	}

	return nil
}
