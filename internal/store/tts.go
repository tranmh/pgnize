package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// GetTTSAudio returns the stored blob key + content type for a content hash, or ErrNotFound
// when no audio is cached. The hash is sha256(provider|voice|lang|text).
func (s *Store) GetTTSAudio(ctx context.Context, hash string) (storageKey, contentType string, err error) {
	err = s.Pool.QueryRow(ctx,
		`SELECT storage_key, content_type FROM tts_audio WHERE content_hash=$1`,
		hash).Scan(&storageKey, &contentType)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", ErrNotFound
	}
	return storageKey, contentType, err
}

// UpsertTTSAudio records a synthesized-audio row. content_hash is unique; a concurrent
// duplicate is a no-op (the blob is identical), so DO NOTHING is correct.
func (s *Store) UpsertTTSAudio(ctx context.Context, hash, provider, voice, lang, storageKey, contentType string, bytes int) error {
	_, err := s.Pool.Exec(ctx,
		`INSERT INTO tts_audio (content_hash, provider, voice, lang, storage_key, content_type, bytes)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)
		 ON CONFLICT (content_hash) DO NOTHING`,
		hash, provider, voice, lang, storageKey, contentType, bytes)
	return err
}
