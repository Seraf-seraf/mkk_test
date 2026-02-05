package tasks

import "errors"

var (
	ErrForbidden       = errors.New("доступ запрещен")
	ErrNotFound        = errors.New("не найдено")
	ErrInvalidAssignee = errors.New("исполнитель не состоит в команде")
	ErrNotImplemented  = errors.New("не реализовано")
)
