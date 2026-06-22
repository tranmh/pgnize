-- +goose Up
-- +goose StatementBegin
-- tts_audio caches synthesized speech so identical requests do not re-bill the TTS provider.
-- Audio is content-addressed by content_hash = sha256(provider|voice|lang|text); the blob is
-- stored in MinIO/S3 under storage_key. The cache is shared across anonymous and logged-in
-- users (broader than the per-game coaching text cache) because the hash is non-enumerable and
-- ties to no game/user.
CREATE TABLE tts_audio (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    content_hash text NOT NULL UNIQUE,
    provider     text NOT NULL,
    voice        text NOT NULL,
    lang         text NOT NULL,
    storage_key  text NOT NULL,
    content_type text NOT NULL,
    bytes        int  NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS tts_audio;
-- +goose StatementEnd
