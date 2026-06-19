package store

import (
	"context"
	"time"
)

// ConsumeRateLimit increments the counter for key within a rolling window and reports
// whether the action is allowed (count <= limit). Mirrors swiss-manager's consumeRateLimit:
// the window resets once `window` has elapsed since window_start.
func (s *Store) ConsumeRateLimit(ctx context.Context, key string, limit int, window time.Duration) (allowed bool, err error) {
	var count int
	secs := window.Seconds()
	err = s.Pool.QueryRow(ctx,
		`INSERT INTO rate_limit_entries (key, count, window_start)
		 VALUES ($1, 1, now())
		 ON CONFLICT (key) DO UPDATE SET
		   count = CASE WHEN rate_limit_entries.window_start < now() - make_interval(secs => $2)
		                THEN 1 ELSE rate_limit_entries.count + 1 END,
		   window_start = CASE WHEN rate_limit_entries.window_start < now() - make_interval(secs => $2)
		                       THEN now() ELSE rate_limit_entries.window_start END
		 RETURNING count`,
		key, secs,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count <= limit, nil
}
