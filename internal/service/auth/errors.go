package auth

import "errors"

var (
	ErrInvalidCredentials = errors.New("неправильный логин или пароль")
	ErrUserExists         = errors.New("пользователь уже существует")
)
