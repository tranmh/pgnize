-- +goose Up
-- +goose StatementBegin
-- game_coaching caches LLM coaching prose so identical requests do not re-bill the model.
-- ply = -1 denotes the whole-game summary; ply >= 0 is a per-move explanation. Keyed by
-- (game_id, ply, model, lang) because coaching differs by model and by language (German-first,
-- but the same move can be coached in another language). Anonymous flows pass no game_id and
-- therefore never cache (the row requires a real games row, FK + NOT NULL).
CREATE TABLE game_coaching (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    game_id    uuid NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    ply        int  NOT NULL DEFAULT -1,
    model      text NOT NULL DEFAULT '',
    lang       text NOT NULL DEFAULT 'de',
    text       text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (game_id, ply, model, lang)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS game_coaching;
-- +goose StatementEnd
