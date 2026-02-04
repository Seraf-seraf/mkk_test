package mailer

import (
	"context"
	"log/slog"
	"sync"
)

// MockMailer - простой мок для тестов.
type MockMailer struct {
	mu       sync.Mutex
	messages []Message
	err      error
}

// NewMockMailer создает новый мок.
func NewMockMailer() *MockMailer {
	const methodCtx = "mailer.NewMockMailer"

	slog.Debug("инициализация mock mailer", slog.String("context", methodCtx))

	return &MockMailer{}
}

func (m *MockMailer) Send(_ context.Context, msg Message) error {
	const methodCtx = "mailer.MockMailer.Send"

	slog.Debug("mock отправка письма", slog.String("context", methodCtx))

	m.mu.Lock()
	defer m.mu.Unlock()

	m.messages = append(m.messages, msg)
	return m.err
}

// Messages возвращает копию всех сообщений.
func (m *MockMailer) Messages() []Message {
	const methodCtx = "mailer.MockMailer.Messages"

	slog.Debug("получение сообщений mock mailer", slog.String("context", methodCtx))

	m.mu.Lock()
	defer m.mu.Unlock()

	out := make([]Message, len(m.messages))
	copy(out, m.messages)
	return out
}

// SetError задает ошибку, которую будет возвращать Send.
func (m *MockMailer) SetError(err error) {
	const methodCtx = "mailer.MockMailer.SetError"

	slog.Debug("установка ошибки mock mailer", slog.String("context", methodCtx))

	m.mu.Lock()
	defer m.mu.Unlock()

	m.err = err
}

// Reset очищает состояние мок-объекта.
func (m *MockMailer) Reset() {
	const methodCtx = "mailer.MockMailer.Reset"

	slog.Debug("сброс состояния mock mailer", slog.String("context", methodCtx))

	m.mu.Lock()
	defer m.mu.Unlock()

	m.messages = nil
	m.err = nil
}
