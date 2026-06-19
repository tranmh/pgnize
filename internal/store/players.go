package store

import (
	"context"
	"strings"

	"github.com/tranmh/pgnize/internal/domain"
)

// NormalizeName lowercases and collapses whitespace for matching.
func NormalizeName(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(s))), " ")
}

// SearchPlayers returns a user's autocomplete matches for prefix/substring q.
func (s *Store) SearchPlayers(ctx context.Context, userID, q string, limit int) ([]domain.Player, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.Pool.Query(ctx,
		`SELECT id, full_name, club, fide_id FROM players
		  WHERE user_id = $1 AND normalized_name LIKE $2
		  ORDER BY times_used DESC, full_name ASC LIMIT $3`,
		userID, "%"+NormalizeName(q)+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Player
	for rows.Next() {
		var p domain.Player
		if err := rows.Scan(&p.ID, &p.FullName, &p.Club, &p.FideID); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// UpsertPlayer creates or bumps a player in the user's pool, returning its id.
func (s *Store) UpsertPlayer(ctx context.Context, userID, fullName string) (string, error) {
	fullName = strings.TrimSpace(fullName)
	if fullName == "" {
		return "", nil
	}
	var id string
	err := s.Pool.QueryRow(ctx,
		`INSERT INTO players (user_id, full_name, normalized_name, times_used)
		 VALUES ($1, $2, $3, 1)
		 ON CONFLICT (user_id, normalized_name)
		 DO UPDATE SET times_used = players.times_used + 1, full_name = EXCLUDED.full_name
		 RETURNING id`,
		userID, fullName, NormalizeName(fullName)).Scan(&id)
	return id, err
}
