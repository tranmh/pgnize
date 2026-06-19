package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/tranmh/pgnize/internal/domain"
)

// CreateUpload records a stored image.
func (s *Store) CreateUpload(ctx context.Context, u domain.Upload) (domain.Upload, error) {
	err := s.Pool.QueryRow(ctx,
		`INSERT INTO uploads (user_id, storage_key, mime_type, byte_size, sha256, consent_training)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, created_at`,
		u.UserID, u.StorageKey, u.MimeType, u.ByteSize, u.SHA256, u.ConsentTraining,
	).Scan(&u.ID, &u.CreatedAt)
	return u, err
}

// GetUpload fetches an upload by id.
func (s *Store) GetUpload(ctx context.Context, id string) (domain.Upload, error) {
	var u domain.Upload
	err := s.Pool.QueryRow(ctx,
		`SELECT id, user_id, storage_key, mime_type, byte_size, sha256, consent_training, created_at
		   FROM uploads WHERE id = $1`, id,
	).Scan(&u.ID, &u.UserID, &u.StorageKey, &u.MimeType, &u.ByteSize, &u.SHA256, &u.ConsentTraining, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, ErrNotFound
	}
	return u, err
}

// PurgeExpiredUploads deletes anonymous + non-consented uploads older than ttlDays,
// returning the storage keys that were removed (so the caller can delete blobs).
func (s *Store) PurgeExpiredUploads(ctx context.Context, ttlDays int) ([]string, error) {
	rows, err := s.Pool.Query(ctx,
		`DELETE FROM uploads
		  WHERE (user_id IS NULL OR consent_training = false)
		    AND created_at < now() - make_interval(days => $1)
		 RETURNING storage_key`, ttlDays)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}
