-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS pgcrypto;   -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pg_trgm;     -- trigram library search
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE users (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name          text NOT NULL,
    email         text NOT NULL UNIQUE,
    password_hash text NOT NULL,
    role          text NOT NULL DEFAULT 'user' CHECK (role IN ('user','admin')),
    created_at    timestamptz NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE sessions (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash text NOT NULL UNIQUE,
    expires_at timestamptz NOT NULL,
    ip         text,
    user_agent text,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_sessions_user ON sessions(user_id);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE players (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    full_name       text NOT NULL,
    normalized_name text NOT NULL,
    club            text NOT NULL DEFAULT '',
    fide_id         text NOT NULL DEFAULT '',
    times_used      int  NOT NULL DEFAULT 0,
    created_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, normalized_name)
);
CREATE INDEX idx_players_lookup ON players(user_id, normalized_name);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE uploads (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          uuid REFERENCES users(id) ON DELETE CASCADE,
    storage_key      text NOT NULL,
    mime_type        text NOT NULL DEFAULT '',
    byte_size        bigint NOT NULL DEFAULT 0,
    sha256           text NOT NULL DEFAULT '',
    layout           text NOT NULL DEFAULT 'unknown',
    consent_training boolean NOT NULL DEFAULT false,
    created_at       timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_uploads_ttl ON uploads(created_at) WHERE user_id IS NULL OR consent_training = false;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE games (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         uuid REFERENCES users(id) ON DELETE CASCADE,
    upload_id       uuid REFERENCES uploads(id) ON DELETE SET NULL,
    source          text NOT NULL CHECK (source IN ('manual','recognized')),
    status          text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','reviewing','saved')),
    event           text NOT NULL DEFAULT '',
    site            text NOT NULL DEFAULT '',
    event_date      text NOT NULL DEFAULT '',
    round           text NOT NULL DEFAULT '',
    board           text NOT NULL DEFAULT '',
    white_player    text NOT NULL DEFAULT '',
    black_player    text NOT NULL DEFAULT '',
    white_player_id uuid REFERENCES players(id) ON DELETE SET NULL,
    black_player_id uuid REFERENCES players(id) ON DELETE SET NULL,
    result          text NOT NULL DEFAULT '*',
    start_fen       text NOT NULL DEFAULT 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1',
    confidence      real NOT NULL DEFAULT 0,
    final_pgn       text,
    created_at      timestamptz NOT NULL DEFAULT now(),
    saved_at        timestamptz
);
CREATE INDEX idx_games_library ON games(user_id, saved_at DESC) WHERE status = 'saved';
CREATE INDEX idx_games_search ON games USING gin ((white_player || ' ' || black_player || ' ' || event) gin_trgm_ops);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE moves (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    game_id         uuid NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    ply             int  NOT NULL,
    side            text NOT NULL CHECK (side IN ('white','black')),
    san             text NOT NULL DEFAULT '',
    fen_after       text NOT NULL DEFAULT '',
    clock_sec       int,
    is_legal        boolean NOT NULL DEFAULT false,
    recognized_text text NOT NULL DEFAULT '',
    corrected       boolean NOT NULL DEFAULT false,
    UNIQUE (game_id, ply)
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE recognition_jobs (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    upload_id       uuid NOT NULL REFERENCES uploads(id) ON DELETE CASCADE,
    user_id         uuid REFERENCES users(id) ON DELETE CASCADE,
    game_id         uuid REFERENCES games(id) ON DELETE SET NULL,
    status          text NOT NULL DEFAULT 'queued' CHECK (status IN ('queued','running','done','failed','canceled')),
    recognizer_name text NOT NULL DEFAULT '',
    attempts        int  NOT NULL DEFAULT 0,
    error           text NOT NULL DEFAULT '',
    result_raw_json jsonb,
    confidence      real,
    queued_at       timestamptz NOT NULL DEFAULT now(),
    started_at      timestamptz,
    finished_at     timestamptz
);
CREATE INDEX idx_jobs_claim ON recognition_jobs(status, queued_at) WHERE status = 'queued';
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE feedback_corrections (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    upload_id       uuid REFERENCES uploads(id) ON DELETE SET NULL,
    game_id         uuid REFERENCES games(id) ON DELETE CASCADE,
    user_id         uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    recognizer_name text NOT NULL DEFAULT '',
    before_json     jsonb NOT NULL,
    after_json      jsonb NOT NULL,
    edit_distance   int   NOT NULL DEFAULT 0,
    created_at      timestamptz NOT NULL DEFAULT now(),
    exported_at     timestamptz
);
CREATE INDEX idx_feedback_export ON feedback_corrections(created_at) WHERE exported_at IS NULL;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE rate_limit_entries (
    key          text PRIMARY KEY,
    count        int NOT NULL DEFAULT 0,
    window_start timestamptz NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS rate_limit_entries;
DROP TABLE IF EXISTS feedback_corrections;
DROP TABLE IF EXISTS recognition_jobs;
DROP TABLE IF EXISTS moves;
DROP TABLE IF EXISTS games;
DROP TABLE IF EXISTS uploads;
DROP TABLE IF EXISTS players;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;
-- +goose StatementEnd
