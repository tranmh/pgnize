// Package jobs runs the asynchronous recognition pipeline: claim queued jobs from
// Postgres (FOR UPDATE SKIP LOCKED), run the recognizer, reconcile against chesskit,
// and persist a draft game for review.
package jobs

import (
	"context"
	"io"

	"github.com/tranmh/chesskit"
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
	rc, mime, err := d.Storage.Get(ctx, job.StorageKey)
	if err != nil {
		return d.Store.MarkJobFailed(ctx, job.JobID, "load image: "+err.Error())
	}
	img, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		return d.Store.MarkJobFailed(ctx, job.JobID, "read image: "+err.Error())
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
	raw := res.RawJSON
	if raw == "" {
		raw = "{}"
	}
	return d.Store.MarkJobDone(ctx, job.JobID, gameID, res.Confidence, raw)
}

func toExamples(rows []store.ExampleRow) []recognition.Example {
	out := make([]recognition.Example, 0, len(rows))
	for _, r := range rows {
		out = append(out, recognition.Example{Header: r.Header, SANs: r.SANs})
	}
	return out
}
