package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/tranmh/pgnize/internal/domain"
)

// CreateJob enqueues a recognition job for an upload. backend is the selectable
// recognizer key ("" = server default) the worker resolves at claim time; recognizerName
// is the resolved Recognizer.Name() recorded for provenance.
func (s *Store) CreateJob(ctx context.Context, uploadID string, userID *string, recognizerName, backend string) (string, error) {
	var id string
	err := s.Pool.QueryRow(ctx,
		`INSERT INTO recognition_jobs (upload_id, user_id, recognizer_name, backend) VALUES ($1, $2, $3, $4) RETURNING id`,
		uploadID, userID, recognizerName, backend).Scan(&id)
	return id, err
}

// ClaimedJob is a job atomically claimed by a worker, with its image location.
type ClaimedJob struct {
	JobID      string
	UploadID   string
	UserID     *string
	StorageKey string
	Backend    string
}

// ClaimNextJob atomically claims one queued job (FOR UPDATE SKIP LOCKED) and marks it running.
// Returns ErrNotFound when the queue is empty.
func (s *Store) ClaimNextJob(ctx context.Context) (ClaimedJob, error) {
	var c ClaimedJob
	err := s.Pool.QueryRow(ctx,
		`UPDATE recognition_jobs j
		    SET status = 'running', started_at = now(), attempts = attempts + 1
		  FROM uploads u
		  WHERE j.id = (
		        SELECT id FROM recognition_jobs
		         WHERE status = 'queued'
		         ORDER BY queued_at
		         FOR UPDATE SKIP LOCKED
		         LIMIT 1)
		    AND u.id = j.upload_id
		 RETURNING j.id, j.upload_id, j.user_id, u.storage_key, j.backend`,
	).Scan(&c.JobID, &c.UploadID, &c.UserID, &c.StorageKey, &c.Backend)
	if errors.Is(err, pgx.ErrNoRows) {
		return c, ErrNotFound
	}
	return c, err
}

// MarkJobDone links the produced draft game and marks the job done.
func (s *Store) MarkJobDone(ctx context.Context, jobID, gameID string, confidence float64, rawJSON string) error {
	_, err := s.Pool.Exec(ctx,
		`UPDATE recognition_jobs
		    SET status='done', game_id=$2, confidence=$3, result_raw_json=$4::jsonb, finished_at=now()
		  WHERE id=$1`,
		jobID, gameID, confidence, rawJSON)
	return err
}

// MarkJobFailed records an error and marks the job failed.
func (s *Store) MarkJobFailed(ctx context.Context, jobID, errMsg string) error {
	_, err := s.Pool.Exec(ctx,
		`UPDATE recognition_jobs SET status='failed', error=$2, finished_at=now() WHERE id=$1`,
		jobID, errMsg)
	return err
}

// GetJob returns a job's status view.
func (s *Store) GetJob(ctx context.Context, id string) (domain.Job, error) {
	var j domain.Job
	err := s.Pool.QueryRow(ctx,
		`SELECT id, upload_id, user_id, status, recognizer_name, attempts, error, game_id, confidence
		   FROM recognition_jobs WHERE id = $1`, id,
	).Scan(&j.ID, &j.UploadID, &j.UserID, &j.Status, &j.RecognizerName, &j.Attempts, &j.Error, &j.GameID, &j.Confidence)
	if errors.Is(err, pgx.ErrNoRows) {
		return j, ErrNotFound
	}
	return j, err
}
