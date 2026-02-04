package middlewares

import (
	"fmt"

	"github.com/golang-jwt/jwt/v4"

	"github.com/Seraf-seraf/mkk_test/internal/config"
)

// Claims описывает JWT claims, включая роль.
type Claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

// NewJWTValidator создает валидатор JWT на основе конфигурации.
func NewJWTValidator(cfg config.JWTConfig) (JWTValidator, error) {
	const methodCtx = "middlewares.NewJWTValidator"

	if cfg.Secret == "" {
		return nil, fmt.Errorf("%s: секрет JWT не задан", methodCtx)
	}

	secret := []byte(cfg.Secret)

	return func(token string) (interface{}, string, error) {
		claims := &Claims{}

		parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("%s: неподдерживаемый метод подписи", methodCtx)
			}
			return secret, nil
		})
		if err != nil {
			return nil, "", fmt.Errorf("%s: %w", methodCtx, err)
		}
		if !parsed.Valid {
			return nil, "", fmt.Errorf("%s: токен недействителен", methodCtx)
		}

		return claims.Subject, claims.Role, nil
	}, nil
}
