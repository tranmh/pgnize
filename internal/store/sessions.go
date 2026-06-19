package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/tranmh/pgnize/internal/domain"
)

// CreateSession persists a session by its token hash.
func (s *Store) CreateSession(ctx context.Context, userID, tokenHash, ip, userAgent string, expires time.Time) error {
	_, err := s.Pool.Exec(ctx,
		`INSERT INTO sessions (user_id, token_hash, expires_at, ip, user_agent)
		 VALUES ($1, $2, $3, $4, $5)`,
		userID, tokenHash, expires, ip, userAgent)
	return err
}

// UserBySessionToken returns the user for a live (non-expired) session token hash.
func (s *Store) UserBySessionToken(ctx context.Context, tokenHash string) (domain.User, error) {
	var u domain.User
	err := s.Pool.QueryRow(ctx,
		`SELECT u.id, u.name, u.email, u.password_hash, u.role, u.created_at
		   FROM sessions s JOIN users u ON u.id = s.user_id
		  WHERE s.token_hash = $1 AND s.expires_at > now()`,
		tokenHash,
	).Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, ErrNotFound
	}
	return u, err
}

// DeleteSession removes a session by token hash (logout).
func (s *Store) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash)
	return err
}
