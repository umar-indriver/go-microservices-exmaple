package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var (
	ErrUserNotFound          = errors.New("user not found")
	ErrUserAlreadyExists     = errors.New("user with this email already exists")
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
)

type User struct {
	ID           string
	Email        string
	PasswordHash string
	Name         string
	Role         string
	CreatedAt    time.Time
}

type RefreshToken struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	Revoked   bool
	CreatedAt time.Time
}

type UserRepository interface {
	CreateUser(ctx context.Context, email, passwordHash, name, role string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id string) (*User, error)

	SaveRefreshToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) (*RefreshToken, error)
	GetRefreshToken(ctx context.Context, tokenHash string) (*RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, tokenHash, userID string) error
}

type postgresUserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepository {
	return &postgresUserRepository{db: db}
}

func (r *postgresUserRepository) CreateUser(ctx context.Context, email, passwordHash, name, role string) (*User, error) {
	if role == "" {
		role = "user"
	}

	id := uuid.New().String()
	now := time.Now().UTC()

	query := `
		INSERT INTO users (id, email, password_hash, name, role, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, email, password_hash, name, role, created_at
	`

	var user User
	err := r.db.QueryRowContext(ctx, query, id, email, passwordHash, name, role, now).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Name,
		&user.Role,
		&user.CreatedAt,
	)

	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrUserAlreadyExists
		}
		return nil, fmt.Errorf("failed to insert user: %w", err)
	}

	return &user, nil
}

func (r *postgresUserRepository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT id, email, password_hash, name, role, created_at
		FROM users
		WHERE email = $1
	`

	var user User
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Name,
		&user.Role,
		&user.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to query user by email: %w", err)
	}

	return &user, nil
}

func (r *postgresUserRepository) GetUserByID(ctx context.Context, id string) (*User, error) {
	query := `
		SELECT id, email, password_hash, name, role, created_at
		FROM users
		WHERE id = $1
	`

	var user User
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Name,
		&user.Role,
		&user.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to query user by id: %w", err)
	}

	return &user, nil
}

func (r *postgresUserRepository) SaveRefreshToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) (*RefreshToken, error) {
	id := uuid.New().String()
	now := time.Now().UTC()

	query := `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, revoked, created_at)
		VALUES ($1, $2, $3, $4, FALSE, $5)
		RETURNING id, user_id, token_hash, expires_at, revoked, created_at
	`

	var rt RefreshToken
	err := r.db.QueryRowContext(ctx, query, id, userID, tokenHash, expiresAt, now).Scan(
		&rt.ID,
		&rt.UserID,
		&rt.TokenHash,
		&rt.ExpiresAt,
		&rt.Revoked,
		&rt.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to insert refresh token: %w", err)
	}

	return &rt, nil
}

func (r *postgresUserRepository) GetRefreshToken(ctx context.Context, tokenHash string) (*RefreshToken, error) {
	query := `
		SELECT id, user_id, token_hash, expires_at, revoked, created_at
		FROM refresh_tokens
		WHERE token_hash = $1
	`

	var rt RefreshToken
	err := r.db.QueryRowContext(ctx, query, tokenHash).Scan(
		&rt.ID,
		&rt.UserID,
		&rt.TokenHash,
		&rt.ExpiresAt,
		&rt.Revoked,
		&rt.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRefreshTokenNotFound
		}
		return nil, fmt.Errorf("failed to query refresh token: %w", err)
	}

	return &rt, nil
}

func (r *postgresUserRepository) RevokeRefreshToken(ctx context.Context, tokenHash, userID string) error {
	var query string
	var args []interface{}

	if tokenHash != "" {
		query = `UPDATE refresh_tokens SET revoked = TRUE WHERE token_hash = $1`
		args = append(args, tokenHash)
	} else if userID != "" {
		query = `UPDATE refresh_tokens SET revoked = TRUE WHERE user_id = $1`
		args = append(args, userID)
	} else {
		return errors.New("either tokenHash or userID must be provided to revoke refresh token")
	}

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}

	return nil
}

func isUniqueViolation(err error) bool {
	return err != nil && (contains(err.Error(), "unique constraint") || contains(err.Error(), "duplicate key"))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > 0 && len(substr) > 0 && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	return len(s) >= len(substr) && stringIndex(s, substr) >= 0
}

func stringIndex(s, substr string) int {
	n := len(substr)
	if n == 0 {
		return 0
	}
	for i := 0; i+n <= len(s); i++ {
		if s[i:i+n] == substr {
			return i
		}
	}
	return -1
}
