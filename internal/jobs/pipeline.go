// Package jobs runs the asynchronous recognition pipeline: claim queued jobs from
// Postgres (FOR UPDATE SKIP LOCKED), run the recognizer, reconcile against chesskit,
// and persist a draft game for review.
package jobs

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"

	"github.com/tranmh/chesskit"
	"github.com/tranmh/pgnize/internal/domain"
	"github.com/tranmh/pgnize/internal/recognition"
	"github.com/tranmh/pgnize/internal/store"
)

// Deps are the collaborators the pipeline needs.
type Deps struct {
	Store      *store.Store
	Storage    storageGetter
	Registry   *recognition.Registry
	FewShotMax int
}

type storageGetter interface {
	Get(ctx context.Context, key string) (io.ReadCloser, string, error)
}

// Process runs one claimed job to completion, marking it done or failed.
func Process(ctx context.Context, d Deps, job store.ClaimedJob) error {
	if job.Kind == "position" {
		return processPosition(ctx, d, job)
	}

	img, mime, err := loadImage(ctx, d, job)
	if err != nil {
		return err
	}

	rec, ok := d.Registry.Resolve(job.Backend)
	if !ok {
		return d.Store.MarkJobFailed(ctx, job.JobID, "unknown recognition backend: "+job.Backend)
	}

	in := recognition.ScoreSheetInput{Image: img, MimeType: mime}
	if job.UserID != nil {
		if rows, err := d.Store.RecentExamples(ctx, *job.UserID, d.FewShotMax); err == nil {
			in.FewShot = toExamples(rows)
		}
	}

	res, err := rec.Recognize(ctx, in)
	if err != nil {
		return d.Store.MarkJobFailed(ctx, job.JobID, "recognize: "+err.Error())
	}

	startFEN := string(chesskit.StartingFEN())
	moves := recognition.Reconcile(startFEN, res.MoveTokens)
	if res.Header.Result == "" {
		res.Header.Result = "*"
	}

	gameID, err := d.Store.CreateDraftGame(ctx, store.NewDraft{
		UserID:     job.UserID,
		UploadID:   &job.UploadID,
		Source:     "recognized",
		Header:     res.Header,
		StartFEN:   startFEN,
		Confidence: res.Confidence,
		Moves:      moves,
	})
	if err != nil {
		return d.Store.MarkJobFailed(ctx, job.JobID, "create draft: "+err.Error())
	}
	return d.Store.MarkJobDone(ctx, job.JobID, gameID, res.Confidence, safeRawJSON(res.RawJSON))
}

// loadImage fetches the upload bytes and MIME type. On a storage/read error it marks the
// job failed (preserving the graceful behavior) and returns a non-nil error so the caller
// stops; the returned MarkJobFailed error, if any, is what Process surfaces.
func loadImage(ctx context.Context, d Deps, job store.ClaimedJob) ([]byte, string, error) {
	rc, mime, err := d.Storage.Get(ctx, job.StorageKey)
	if err != nil {
		return nil, "", d.Store.MarkJobFailed(ctx, job.JobID, "load image: "+err.Error())
	}
	img, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		return nil, "", d.Store.MarkJobFailed(ctx, job.JobID, "read image: "+err.Error())
	}
	return img, mime, nil
}

// processPosition recognizes a single board position and stores it as a draft game whose
// StartFEN is the recognized position and whose move list is empty. A failed FEN assembly
// falls back to the standard start (the editable board is the correction path), so the job
// never fails on assembly alone.
func processPosition(ctx context.Context, d Deps, job store.ClaimedJob) error {
	img, mime, err := loadImage(ctx, d, job)
	if err != nil {
		return err
	}

	rec, ok := d.Registry.Resolve(job.Backend)
	if !ok {
		return d.Store.MarkJobFailed(ctx, job.JobID, "unknown recognition backend: "+job.Backend)
	}

	res, err := rec.RecognizePosition(ctx, recognition.PositionInput{Image: img, MimeType: mime})
	if err != nil {
		return d.Store.MarkJobFailed(ctx, job.JobID, "recognize position: "+err.Error())
	}

	fen, err := recognition.AssembleFEN(res)
	conf := res.Confidence
	if err != nil {
		slog.Warn("position FEN assembly failed; falling back to starting position",
			"jobID", job.JobID, "err", err)
		fen = string(chesskit.StartingFEN())
		conf = 0
	}

	gameID, err := d.Store.CreateDraftGame(ctx, store.NewDraft{
		UserID:     job.UserID,
		UploadID:   &job.UploadID,
		Source:     "recognized",
		Header:     domain.Header{Result: "*"},
		StartFEN:   fen,
		Confidence: conf,
		Moves:      nil,
	})
	if err != nil {
		return d.Store.MarkJobFailed(ctx, job.JobID, "create draft: "+err.Error())
	}
	return d.Store.MarkJobDone(ctx, job.JobID, gameID, conf, safeRawJSON(res.RawJSON))
}

// safeRawJSON returns raw when it is valid JSON, else "{}". result_raw_json is a
// jsonb column, and a num_predict cap can truncate the model output mid-JSON —
// the moves are still salvaged, but storing the invalid raw would fail the
// ::jsonb cast with SQLSTATE 22P02.
func safeRawJSON(raw string) string {
	if raw != "" && json.Valid([]byte(raw)) {
		return raw
	}
	return "{}"
}

func toExamples(rows []store.ExampleRow) []recognition.Example {
	out := make([]recognition.Example, 0, len(rows))
	for _, r := range rows {
		out = append(out, recognition.Example{Header: r.Header, SANs: r.SANs})
	}
	return out
}
