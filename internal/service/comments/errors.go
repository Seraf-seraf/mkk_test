package comments

import "errors"

var (
	ErrForbidden      = errors.New("доступ запрещен")
	ErrNotFound       = errors.New("не найдено")
	ErrNotImplemented = errors.New("не реализовано")
)
