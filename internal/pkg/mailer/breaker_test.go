package mailer

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeBreaker struct {
	err    error
	called bool
}

func (b *fakeBreaker) Execute(fn func() error) error {
	b.called = true
	if b.err != nil {
		return b.err
	}
	return fn()
}

func TestNewBreakerMailerValidation(t *testing.T) {
	const methodCtx = "mailer.TestNewBreakerMailerValidation"

	_, err := NewBreakerMailer(nil, &fakeBreaker{})
	require.Error(t, err, methodCtx)

	_, err = NewBreakerMailer(NewMockMailer(), nil)
	require.Error(t, err, methodCtx)
}

func TestBreakerMailerSend(t *testing.T) {
	const methodCtx = "mailer.TestBreakerMailerSend"

	mock := NewMockMailer()
	breaker := &fakeBreaker{}
	bm, err := NewBreakerMailer(mock, breaker)
	require.NoError(t, err, methodCtx)

	err = bm.Send(context.Background(), Message{To: "test@example.com"})
	require.NoError(t, err, methodCtx)
	require.True(t, breaker.called, methodCtx)
	require.Len(t, mock.Messages(), 1, methodCtx)
}

func TestBreakerMailerSendError(t *testing.T) {
	const methodCtx = "mailer.TestBreakerMailerSendError"

	mock := NewMockMailer()
	breaker := &fakeBreaker{err: errors.New("boom")}
	bm, err := NewBreakerMailer(mock, breaker)
	require.NoError(t, err, methodCtx)

	err = bm.Send(context.Background(), Message{To: "fail@example.com"})
	require.Error(t, err, methodCtx)
}

func TestMockMailerReset(t *testing.T) {
	const methodCtx = "mailer.TestMockMailerReset"

	mock := NewMockMailer()
	mock.SetError(errors.New("send error"))
	err := mock.Send(context.Background(), Message{To: "x@example.com"})
	require.Error(t, err, methodCtx)

	mock.Reset()
	err = mock.Send(context.Background(), Message{To: "y@example.com"})
	require.NoError(t, err, methodCtx)
	require.Len(t, mock.Messages(), 1, methodCtx)
}
