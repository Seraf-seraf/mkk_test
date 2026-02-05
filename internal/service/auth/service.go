package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"golang.org/x/crypto/bcrypt"

	"github.com/Seraf-seraf/mkk_test/internal/api"
	"github.com/Seraf-seraf/mkk_test/internal/config"
	"github.com/Seraf-seraf/mkk_test/internal/pkg/breaker"
	"github.com/Seraf-seraf/mkk_test/internal/pkg/mailer"
	repomysql "github.com/Seraf-seraf/mkk_test/internal/repo/mysql"
)

const defaultRole = "member"

// UserRepository описывает интерфейс работы с пользователями.
type UserRepository interface {
	Create(ctx context.Context, email, passwordHash string) (api.User, error)
	GetByEmail(ctx context.Context, email string) (repomysql.UserRecord, error)
}

// RoleRepository описывает доступ к ролям пользователя.
type RoleRepository interface {
	GetHighestRoleByUser(ctx context.Context, userID uuid.UUID) (string, bool, error)
}

// Service реализует регистрацию и вход.
type Service struct {
	repo      UserRepository
	roles     RoleRepository
	mailer    mailer.Mailer
	breaker   breaker.Breaker
	jwtSecret []byte
	accessTTL time.Duration
	now       func() time.Time
}

// NewService создает AuthService.
func NewService(repo UserRepository, roles RoleRepository, mailer mailer.Mailer, breaker breaker.Breaker, cfg config.JWTConfig) (*Service, error) {
	const methodCtx = "auth.NewService"

	if repo == nil {
		return nil, fmt.Errorf("%s: repo не задан", methodCtx)
	}
	if roles == nil {
		return nil, fmt.Errorf("%s: roles repo не задан", methodCtx)
	}
	if cfg.Secret == "" {
		return nil, fmt.Errorf("%s: secret не задан", methodCtx)
	}

	ttl := time.Duration(cfg.AccessTTLMinutes) * time.Minute
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}

	return &Service{
		repo:      repo,
		roles:     roles,
		mailer:    mailer,
		breaker:   breaker,
		jwtSecret: []byte(cfg.Secret),
		accessTTL: ttl,
		now:       time.Now,
	}, nil
}

// Register регистрирует пользователя и отправляет приветственное письмо.
func (s *Service) Register(ctx context.Context, req api.RegisterRequest) (api.User, error) {
	const methodCtx = "auth.Service.Register"

	if req.Email == "" || req.Password == "" {
		return api.User{}, fmt.Errorf("%s: email или пароль не задан", methodCtx)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return api.User{}, fmt.Errorf("%s: ошибка хэширования пароля: %w", methodCtx, err)
	}

	user, err := s.repo.Create(ctx, string(req.Email), string(hash))
	if err != nil {
		if errors.Is(err, repomysql.ErrUserExists) {
			return api.User{}, fmt.Errorf("%s: %w", methodCtx, ErrUserExists)
		}
		return api.User{}, fmt.Errorf("%s: %w", methodCtx, err)
	}

	if s.mailer != nil {
		send := func() error {
			return s.mailer.Send(ctx, mailer.Message{
				To:      string(req.Email),
				Subject: "Добро пожаловать",
				Body:    "Регистрация прошла успешно",
			})
		}
		if s.breaker != nil {
			err = s.breaker.Execute(send)
		} else {
			err = send()
		}
		if err != nil {
			return api.User{}, fmt.Errorf("%s: ошибка отправки письма: %w", methodCtx, err)
		}
	}

	return user, nil
}

// Login выполняет вход и возвращает JWT.
func (s *Service) Login(ctx context.Context, req api.LoginRequest) (api.AuthResponse, error) {
	const methodCtx = "auth.Service.Login"

	if req.Email == "" || req.Password == "" {
		return api.AuthResponse{}, fmt.Errorf("%s: email или пароль не задан", methodCtx)
	}

	record, err := s.repo.GetByEmail(ctx, string(req.Email))
	if err != nil {
		if errors.Is(err, repomysql.ErrUserNotFound) {
			return api.AuthResponse{}, fmt.Errorf("%s: %w", methodCtx, ErrInvalidCredentials)
		}
		return api.AuthResponse{}, fmt.Errorf("%s: %w", methodCtx, err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(record.PasswordHash), []byte(req.Password)); err != nil {
		return api.AuthResponse{}, fmt.Errorf("%s: %w", methodCtx, ErrInvalidCredentials)
	}

	token, err := s.generateToken(ctx, record)
	if err != nil {
		return api.AuthResponse{}, fmt.Errorf("%s: %w", methodCtx, err)
	}

	user := api.User{
		Id:        record.ID,
		Email:     openapi_types.Email(record.Email),
		CreatedAt: record.CreatedAt,
	}

	return api.AuthResponse{Token: token, User: user}, nil
}

func (s *Service) generateToken(ctx context.Context, user repomysql.UserRecord) (string, error) {
	const methodCtx = "auth.Service.generateToken"

	role := defaultRole
	if s.roles != nil {
		roleValue, ok, err := s.roles.GetHighestRoleByUser(ctx, user.ID)
		if err != nil {
			return "", fmt.Errorf("%s: %w", methodCtx, err)
		}
		if ok && roleValue != "" {
			role = roleValue
		}
	}

	now := s.now().UTC()
	claims := &jwt.RegisteredClaims{
		Subject:   user.ID.String(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTTL)),
	}

	custom := struct {
		Role string `json:"role"`
		*jwt.RegisteredClaims
	}{
		Role:             role,
		RegisteredClaims: claims,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, custom)
	signed, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("%s: ошибка подписи токена: %w", methodCtx, err)
	}

	return signed, nil
}
