package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	sqlmysql "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/Seraf-seraf/mkk_test/internal/api"
)

var (
	ErrUserExists   = errors.New("пользователь уже существует")
	ErrUserNotFound = errors.New("пользователь не найден")
)

// UserRecord содержит данные пользователя из БД.
type UserRecord struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// UsersRepo реализует работу с пользователями в MySQL.
type UsersRepo struct {
	db *sql.DB
}

// NewUsersRepo создает репозиторий пользователей.
func NewUsersRepo(db *sql.DB) *UsersRepo {
	const methodCtx = "repo.NewUsersRepo"

	slog.Debug("инициализация репозитория пользователей", slog.String("context", methodCtx))

	return &UsersRepo{db: db}
}

// Create создает пользователя.
func (r *UsersRepo) Create(ctx context.Context, email, passwordHash string) (api.User, error) {
	const methodCtx = "repo.UsersRepo.Create"

	if r == nil || r.db == nil {
		return api.User{}, fmt.Errorf("%s: репозиторий не инициализирован", methodCtx)
	}

	id := uuid.New()
	now := time.Now().UTC()

	query := `INSERT INTO users (id, email, password_hash, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, id.String(), email, passwordHash, now, now)
	if err != nil {
		if isDuplicate(err) {
			return api.User{}, fmt.Errorf("%s: %w", methodCtx, ErrUserExists)
		}
		return api.User{}, fmt.Errorf("%s: ошибка создания пользователя: %w", methodCtx, err)
	}

	return api.User{
		Id:        id,
		Email:     openapi_types.Email(email),
		CreatedAt: now,
	}, nil
}

// GetByEmail возвращает пользователя и хэш пароля по email.
func (r *UsersRepo) GetByEmail(ctx context.Context, email string) (UserRecord, error) {
	const methodCtx = "repo.UsersRepo.GetByEmail"

	if r == nil || r.db == nil {
		return UserRecord{}, fmt.Errorf("%s: репозиторий не инициализирован", methodCtx)
	}

	query := `SELECT id, email, password_hash, created_at, updated_at FROM users WHERE email = ? LIMIT 1`
	row := r.db.QueryRowContext(ctx, query, email)

	var idStr string
	var record UserRecord
	if err := row.Scan(&idStr, &record.Email, &record.PasswordHash, &record.CreatedAt, &record.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return UserRecord{}, fmt.Errorf("%s: %w", methodCtx, ErrUserNotFound)
		}
		return UserRecord{}, fmt.Errorf("%s: ошибка чтения пользователя: %w", methodCtx, err)
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return UserRecord{}, fmt.Errorf("%s: ошибка разбора id: %w", methodCtx, err)
	}

	record.ID = id
	return record, nil
}

// FindIDByEmail возвращает id пользователя по email.
func (r *UsersRepo) FindIDByEmail(ctx context.Context, email string) (uuid.UUID, bool, error) {
	const methodCtx = "repo.UsersRepo.FindIDByEmail"

	if r == nil || r.db == nil {
		return uuid.UUID{}, false, fmt.Errorf("%s: репозиторий не инициализирован", methodCtx)
	}

	var idStr string
	err := r.db.QueryRowContext(ctx, "SELECT id FROM users WHERE email = ? LIMIT 1", email).Scan(&idStr)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.UUID{}, false, nil
	}
	if err != nil {
		return uuid.UUID{}, false, fmt.Errorf("%s: ошибка чтения пользователя: %w", methodCtx, err)
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return uuid.UUID{}, false, fmt.Errorf("%s: ошибка разбора id: %w", methodCtx, err)
	}

	return id, true, nil
}

// GetEmailByID возвращает email пользователя по id.
func (r *UsersRepo) GetEmailByID(ctx context.Context, id uuid.UUID) (string, bool, error) {
	const methodCtx = "repo.UsersRepo.GetEmailByID"

	if r == nil || r.db == nil {
		return "", false, fmt.Errorf("%s: репозиторий не инициализирован", methodCtx)
	}

	var email string
	err := r.db.QueryRowContext(ctx, "SELECT email FROM users WHERE id = ? LIMIT 1", id.String()).Scan(&email)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("%s: ошибка чтения пользователя: %w", methodCtx, err)
	}

	return email, true, nil
}

func isDuplicate(err error) bool {
	const methodCtx = "repo.isDuplicate"

	slog.Debug("проверка дубликата", slog.String("context", methodCtx))

	var mysqlErr *sqlmysql.MySQLError
	if errors.As(err, &mysqlErr) {
		return mysqlErr.Number == 1062
	}
	return false
}
