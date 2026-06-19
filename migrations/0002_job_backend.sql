-- +goose Up
-- +goose StatementBegin
-- Per-request recognition backend key ("" = server default), resolved by the worker at
-- claim time. recognizer_name still records the resolved Recognizer.Name() for provenance.
ALTER TABLE recognition_jobs ADD COLUMN backend text NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE recognition_jobs DROP COLUMN backend;
-- +goose StatementEnd
