package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// GetCoaching returns cached coaching text for (gameID, ply, model, lang), or ErrNotFound
// when nothing is cached. ply = -1 is the whole-game summary.
func (s *Store) GetCoaching(ctx context.Context, gameID string, ply int, model, lang string) (string, error) {
	var text string
	err := s.Pool.QueryRow(ctx,
		`SELECT text FROM game_coaching WHERE game_id=$1 AND ply=$2 AND model=$3 AND lang=$4`,
		gameID, ply, model, lang).Scan(&text)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return text, err
}

// UpsertCoaching stores (or refreshes) coaching text for (gameID, ply, model, lang).
func (s *Store) UpsertCoaching(ctx context.Context, gameID string, ply int, model, lang, text string) error {
	_, err := s.Pool.Exec(ctx,
		`INSERT INTO game_coaching (game_id, ply, model, lang, text)
		 VALUES ($1,$2,$3,$4,$5)
		 ON CONFLICT (game_id, ply, model, lang)
		 DO UPDATE SET text = EXCLUDED.text, created_at = now()`,
		gameID, ply, model, lang, text)
	return err
}
