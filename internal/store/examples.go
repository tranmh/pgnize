package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/tranmh/pgnize/internal/domain"
)

// ExampleRow is a past corrected sheet used as few-shot context.
type ExampleRow struct {
	Header domain.Header
	SANs   []string
}

// RecentExamples returns a user's most recent saved, recognized games as few-shot examples.
func (s *Store) RecentExamples(ctx context.Context, userID string, limit int) ([]ExampleRow, error) {
	if limit <= 0 {
		limit = 3
	}
	rows, err := s.Pool.Query(ctx,
		`SELECT g.event, g.site, g.event_date, g.round, g.board, g.white_player, g.black_player, g.result,
		        array_remove(array_agg(m.san ORDER BY m.ply), '') AS sans
		   FROM games g JOIN moves m ON m.game_id = g.id
		  WHERE g.user_id = $1 AND g.status = 'saved' AND g.source = 'recognized'
		  GROUP BY g.id, g.saved_at
		  ORDER BY g.saved_at DESC
		  LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ExampleRow
	for rows.Next() {
		var e ExampleRow
		if err := rows.Scan(&e.Header.Event, &e.Header.Site, &e.Header.Date, &e.Header.Round,
			&e.Header.Board, &e.Header.White, &e.Header.Black, &e.Header.Result, &e.SANs); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// JobRawByGame returns the recognizer name, raw JSON, and upload id for the job that produced a game.
func (s *Store) JobRawByGame(ctx context.Context, gameID string) (recognizer, rawJSON string, uploadID *string, err error) {
	var raw *string
	err = s.Pool.QueryRow(ctx,
		`SELECT recognizer_name, result_raw_json::text, upload_id
		   FROM recognition_jobs WHERE game_id = $1 ORDER BY finished_at DESC LIMIT 1`, gameID,
	).Scan(&recognizer, &raw, &uploadID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", nil, ErrNotFound
	}
	if raw != nil {
		rawJSON = *raw
	}
	return recognizer, rawJSON, uploadID, err
}
