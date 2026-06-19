// Package store holds Postgres-backed repositories (pgx; no ORM).
package store

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// Store is the data-access layer. One value is shared across the app.
type Store struct {
	Pool *pgxpool.Pool
}

// New opens a connection pool.
func New(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("pgx pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("db ping: %w", err)
	}
	return &Store{Pool: pool}, nil
}

// Close releases the pool.
func (s *Store) Close() { s.Pool.Close() }

// Ping checks DB liveness (used by /readyz).
func (s *Store) Ping(ctx context.Context) error { return s.Pool.Ping(ctx) }

// Migrate runs goose migrations from the given embedded FS against databaseURL.
func Migrate(databaseURL string, migrations fs.FS, down bool) error {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("open for migrate: %w", err)
	}
	defer db.Close()
	goose.SetBaseFS(migrations)
	defer goose.SetBaseFS(nil)
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	if down {
		return goose.Down(db, ".")
	}
	return goose.Up(db, ".")
}

var _ = stdlib.GetDefaultDriver // ensure the pgx stdlib driver is registered for goose
