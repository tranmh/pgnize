-- +goose Up
-- +goose StatementBegin
-- job_images holds the EXTRA images of a multi-image job (idx >= 1). The PRIMARY (first)
-- image stays on recognition_jobs.upload_id; this table only records the additional uploads
-- so one submission can be fed to the recognizer as a single multi-image request.
CREATE TABLE job_images (
    job_id    uuid NOT NULL REFERENCES recognition_jobs(id) ON DELETE CASCADE,
    upload_id uuid NOT NULL REFERENCES uploads(id) ON DELETE CASCADE,
    idx       int  NOT NULL,
    PRIMARY KEY (job_id, idx)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS job_images;
-- +goose StatementEnd
