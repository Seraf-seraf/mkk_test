package teams

import "errors"

var (
	ErrForbidden           = errors.New("доступ запрещен")
	ErrNotFound            = errors.New("не найдено")
	ErrAlreadyMember       = errors.New("пользователь уже состоит в команде")
	ErrInviteNotFound      = errors.New("приглашение не найдено")
	ErrInviteEmailMismatch = errors.New("email не соответствует приглашению")
	ErrNotImplemented      = errors.New("не реализовано")
)
