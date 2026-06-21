//go:build integration

package jobs_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/tranmh/pgnize/internal/domain"
	"github.com/tranmh/pgnize/internal/jobs"
	"github.com/tranmh/pgnize/internal/recognition"
	"github.com/tranmh/pgnize/internal/storage"
	"github.com/tranmh/pgnize/internal/store"
	"github.com/tranmh/pgnize/migrations"
)

// spyRecognizer captures the input it is handed so a test can assert the pipeline wired the
// extra images through. It otherwise returns a fixed legal result.
type spyRecognizer struct {
	gotScore    recognition.ScoreSheetInput
	gotPosition recognition.PositionInput
}

func (s *spyRecognizer) Name() string { return "spy" }

func (s *spyRecognizer) Recognize(_ context.Context, in recognition.ScoreSheetInput) (recognition.RecognitionResult, error) {
	s.gotScore = in
	return recognition.RecognitionResult{
		Header:     domain.Header{White: "A", Black: "B", Result: "*"},
		MoveTokens: []recognition.MoveToken{{Ply: 1, Side: recognition.SideWhite, Text: "e4", Confidence: 0.9}},
		Confidence: 0.9,
		RawJSON:    `{"recognizer":"spy"}`,
	}, nil
}

func (s *spyRecognizer) RecognizePosition(_ context.Context, in recognition.PositionInput) (recognition.PositionResult, error) {
	s.gotPosition = in
	return recognition.PositionResult{
		Grid: []string{
			"....k...", "........", "........", "........",
			"........", "........", "........", "....K..R",
		},
		SideToMove: recognition.SideWhite, Orientation: "white_bottom",
		Confidence: 0.9, RawJSON: `{"recognizer":"spy","kind":"position"}`,
	}, nil
}

func newJobsStore(t *testing.T) *store.Store {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	if err := store.Migrate(url, migrations.FS, false); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	st, err := store.New(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	st.Pool.Exec(context.Background(), `TRUNCATE users, sessions, players, uploads, games, moves,
		recognition_jobs, feedback_corrections, rate_limit_entries RESTART IDENTITY CASCADE`)
	t.Cleanup(st.Close)
	return st
}

// putUpload writes a blob to storage and records its upload row, returning the upload id.
func putUpload(t *testing.T, st *store.Store, blob *storage.FS, key string, data []byte) string {
	t.Helper()
	ctx := context.Background()
	if err := blob.Put(ctx, key, strings.NewReader(string(data)), int64(len(data)), "image/jpeg"); err != nil {
		t.Fatalf("put: %v", err)
	}
	up, err := st.CreateUpload(ctx, domain.Upload{StorageKey: key, MimeType: "image/jpeg", ByteSize: int64(len(data))})
	if err != nil {
		t.Fatalf("create upload: %v", err)
	}
	return up.ID
}

// TestProcessLoadsExtraImages proves a multi-image scoresheet job feeds every extra image
// (from job_images) into the recognizer alongside the primary, in order.
func TestProcessLoadsExtraImages(t *testing.T) {
	st := newJobsStore(t)
	ctx := context.Background()
	blob, err := storage.NewFS(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	primary := putUpload(t, st, blob, "k/primary.jpg", []byte("primary-bytes"))
	extra1 := putUpload(t, st, blob, "k/extra1.jpg", []byte("extra-1-bytes"))
	extra2 := putUpload(t, st, blob, "k/extra2.jpg", []byte("extra-2-bytes"))

	if _, err := st.CreateJob(ctx, primary, nil, "spy", "spy", "scoresheet", []string{extra1, extra2}); err != nil {
		t.Fatalf("create job: %v", err)
	}

	spy := &spyRecognizer{}
	reg := recognition.NewRegistry()
	reg.Register("spy", "spy", false, spy)
	reg.SetDefault("spy")
	deps := jobs.Deps{Store: st, Storage: blob, Registry: reg, FewShotMax: 0}

	job, err := st.ClaimNextJob(ctx)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if err := jobs.Process(ctx, deps, job); err != nil {
		t.Fatalf("process: %v", err)
	}

	if string(spy.gotScore.Image) != "primary-bytes" {
		t.Fatalf("primary image = %q", spy.gotScore.Image)
	}
	if len(spy.gotScore.Extra) != 2 {
		t.Fatalf("extra images = %d, want 2", len(spy.gotScore.Extra))
	}
	if string(spy.gotScore.Extra[0].Data) != "extra-1-bytes" || string(spy.gotScore.Extra[1].Data) != "extra-2-bytes" {
		t.Fatalf("extra image order wrong: %q, %q", spy.gotScore.Extra[0].Data, spy.gotScore.Extra[1].Data)
	}
}

// TestProcessPositionLoadsExtraImages is the position-pipeline counterpart.
func TestProcessPositionLoadsExtraImages(t *testing.T) {
	st := newJobsStore(t)
	ctx := context.Background()
	blob, err := storage.NewFS(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	primary := putUpload(t, st, blob, "k/p.jpg", []byte("primary-bytes"))
	extra1 := putUpload(t, st, blob, "k/e1.jpg", []byte("extra-1-bytes"))

	if _, err := st.CreateJob(ctx, primary, nil, "spy", "spy", "position", []string{extra1}); err != nil {
		t.Fatalf("create job: %v", err)
	}

	spy := &spyRecognizer{}
	reg := recognition.NewRegistry()
	reg.Register("spy", "spy", false, spy)
	reg.SetDefault("spy")
	deps := jobs.Deps{Store: st, Storage: blob, Registry: reg}

	job, err := st.ClaimNextJob(ctx)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if err := jobs.Process(ctx, deps, job); err != nil {
		t.Fatalf("process: %v", err)
	}

	if len(spy.gotPosition.Extra) != 1 {
		t.Fatalf("position extra images = %d, want 1", len(spy.gotPosition.Extra))
	}
	if string(spy.gotPosition.Extra[0].Data) != "extra-1-bytes" {
		t.Fatalf("position extra = %q", spy.gotPosition.Extra[0].Data)
	}
}
