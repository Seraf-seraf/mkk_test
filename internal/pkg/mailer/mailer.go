package mailer

import "context"

// Message описывает письмо.
type Message struct {
	To      string
	Subject string
	Body    string
}

// Mailer описывает интерфейс отправки писем.
type Mailer interface {
	Send(ctx context.Context, msg Message) error
}
