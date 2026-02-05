package reports

import "errors"

var (
	ErrForbidden      = errors.New("доступ запрещен")
	ErrNotImplemented = errors.New("не реализовано")
)
