-- +goose Up
-- +goose StatementBegin
ALTER TABLE recognition_jobs ADD COLUMN kind text NOT NULL DEFAULT 'scoresheet' CHECK (kind IN ('scoresheet','position'));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE recognition_jobs DROP COLUMN kind;
-- +goose StatementEnd
