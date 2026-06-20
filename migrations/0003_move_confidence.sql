-- +goose Up
-- Per-ply recognition confidence (0..1), independent of legality. Default 1.0 means
-- "verified/confident unless flagged": manual and saved moves need no extra wiring; only the
-- recognition pipeline writes lower values to mark moves for human verification (yellow state).
ALTER TABLE moves ADD COLUMN confidence real NOT NULL DEFAULT 1.0;

-- +goose Down
ALTER TABLE moves DROP COLUMN confidence;
