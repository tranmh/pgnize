-- +goose Up
-- +goose StatementBegin
-- chat_sessions is one conversational-coach thread, grounded in a position. user_id is NULL
-- for anonymous flows; those are NOT persisted by the handler (the column is nullable to
-- leave the option open), so in practice every stored session belongs to a user. game_id/ply
-- record the position context; fen is the grounding position at session start.
CREATE TABLE chat_sessions (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid REFERENCES users(id) ON DELETE CASCADE,
    game_id    uuid REFERENCES games(id) ON DELETE SET NULL,
    ply        int,
    fen        text NOT NULL,
    lang       text NOT NULL DEFAULT 'de',
    model      text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_chat_sessions_user ON chat_sessions (user_id, updated_at DESC);
-- +goose StatementEnd

-- +goose StatementBegin
-- chat_messages are the ordered turns of a session. seq is 0-based and unique per session.
-- role is 'user' or 'model' (the coach). tool_trace is the JSON array of engine calls a coach
-- turn made (NULL for plain user turns), kept so the UI can show what the answer was grounded in.
CREATE TABLE chat_messages (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id uuid NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
    seq        int  NOT NULL,
    role       text NOT NULL CHECK (role IN ('user','model')),
    content    text NOT NULL DEFAULT '',
    tool_trace jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (session_id, seq)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS chat_messages;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE IF EXISTS chat_sessions;
-- +goose StatementEnd
