package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/tranmh/pgnize/internal/domain"
)

// ErrNotFound is returned when a row does not exist.
var ErrNotFound = errors.New("not found")

// CreateUser inserts a new user. Returns ErrEmailTaken on unique violation.
var ErrEmailTaken = errors.New("email already registered")

func (s *Store) CreateUser(ctx context.Context, name, email, passwordHash string) (domain.User, error) {
	var u domain.User
	err := s.Pool.QueryRow(ctx,
		`INSERT INTO users (name, email, password_hash) VALUES ($1, $2, $3)
		 RETURNING id, name, email, password_hash, role, created_at`,
		name, email, passwordHash,
	).Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return u, ErrEmailTaken
		}
		return u, err
	}
	return u, nil
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	return s.scanUser(ctx,
		`SELECT id, name, email, password_hash, role, created_at FROM users WHERE email = $1`, email)
}

func (s *Store) GetUserByID(ctx context.Context, id string) (domain.User, error) {
	return s.scanUser(ctx,
		`SELECT id, name, email, password_hash, role, created_at FROM users WHERE id = $1`, id)
}

func (s *Store) scanUser(ctx context.Context, q string, args ...any) (domain.User, error) {
	var u domain.User
	err := s.Pool.QueryRow(ctx, q, args...).
		Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, ErrNotFound
	}
	return u, err
}
