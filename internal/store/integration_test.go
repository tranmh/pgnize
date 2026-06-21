//go:build integration

package store_test

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/tranmh/pgnize/internal/domain"
	"github.com/tranmh/pgnize/internal/store"
	"github.com/tranmh/pgnize/migrations"
)

func newStore(t *testing.T) *store.Store {
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

func TestClaimNextJobNoDoubleClaim(t *testing.T) {
	st := newStore(t)
	ctx := context.Background()

	// Two uploads -> two queued jobs.
	for i := 0; i < 2; i++ {
		up, err := st.CreateUpload(ctx, domain.Upload{StorageKey: "k", MimeType: "image/jpeg"})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := st.CreateJob(ctx, up.ID, nil, "fake", "fake", "scoresheet", nil); err != nil {
			t.Fatal(err)
		}
	}

	// Claim concurrently from several goroutines; each job must be claimed at most once.
	var mu sync.Mutex
	seen := map[string]int{}
	var wg sync.WaitGroup
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c, err := st.ClaimNextJob(ctx)
			if err == store.ErrNotFound {
				return
			}
			if err != nil {
				t.Errorf("claim: %v", err)
				return
			}
			mu.Lock()
			seen[c.JobID]++
			mu.Unlock()
		}()
	}
	wg.Wait()

	if len(seen) != 2 {
		t.Fatalf("expected 2 distinct claimed jobs, got %d", len(seen))
	}
	for id, n := range seen {
		if n != 1 {
			t.Fatalf("job %s claimed %d times (double-claim)", id, n)
		}
	}
}

// TestCreateJobExtraImages proves a multi-image job records its extra uploads in job_images
// (idx 1..N) and that JobExtraStorageKeys returns their storage keys ordered by idx.
func TestCreateJobExtraImages(t *testing.T) {
	st := newStore(t)
	ctx := context.Background()

	mkUpload := func(key string) string {
		up, err := st.CreateUpload(ctx, domain.Upload{StorageKey: key, MimeType: "image/jpeg"})
		if err != nil {
			t.Fatal(err)
		}
		return up.ID
	}
	primary := mkUpload("primary-key")
	extra1 := mkUpload("extra1-key")
	extra2 := mkUpload("extra2-key")

	jobID, err := st.CreateJob(ctx, primary, nil, "fake", "fake", "scoresheet", []string{extra1, extra2})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}

	keys, err := st.JobExtraStorageKeys(ctx, jobID)
	if err != nil {
		t.Fatalf("extra keys: %v", err)
	}
	want := []string{"extra1-key", "extra2-key"}
	if len(keys) != len(want) {
		t.Fatalf("got %d extra keys, want %d: %v", len(keys), len(want), keys)
	}
	for i, w := range want {
		if keys[i] != w {
			t.Errorf("extra key[%d] = %q, want %q (idx ordering)", i, keys[i], w)
		}
	}

	// Single-image (nil extras) records no job_images rows.
	soloID, err := st.CreateJob(ctx, primary, nil, "fake", "fake", "scoresheet", nil)
	if err != nil {
		t.Fatalf("create solo job: %v", err)
	}
	solo, err := st.JobExtraStorageKeys(ctx, soloID)
	if err != nil {
		t.Fatalf("solo extra keys: %v", err)
	}
	if len(solo) != 0 {
		t.Fatalf("single-image job should have no extras, got %d", len(solo))
	}
}

func TestMoveConfidenceRoundTrip(t *testing.T) {
	st := newStore(t)
	ctx := context.Background()

	// A draft with a mix of confidences: a clean legal move, a flagged (verify) move, and an
	// illegal move — exactly what the recognition pipeline produces.
	id, err := st.CreateDraftGame(ctx, store.NewDraft{
		Source:     domain.SourceRecognized,
		Header:     domain.Header{White: "A", Black: "B"},
		Confidence: 0.5,
		Moves: []domain.Move{
			{Ply: 1, Side: "white", SAN: "e4", FenAfter: "x", IsLegal: true, Confidence: 0.9},
			{Ply: 2, Side: "black", SAN: "Nbd2", IsLegal: true, Corrected: true, Confidence: 0.3},
			{Ply: 3, Side: "white", SAN: "zzz", IsLegal: false, Confidence: 0.0},
		},
	})
	if err != nil {
		t.Fatalf("create draft: %v", err)
	}

	g, _, err := st.GetGame(ctx, id)
	if err != nil {
		t.Fatalf("get game: %v", err)
	}
	want := []float64{0.9, 0.3, 0.0}
	if len(g.Moves) != len(want) {
		t.Fatalf("got %d moves, want %d", len(g.Moves), len(want))
	}
	for i, w := range want {
		if diff := g.Moves[i].Confidence - w; diff > 0.001 || diff < -0.001 {
			t.Errorf("move %d confidence=%.3f, want %.3f", i+1, g.Moves[i].Confidence, w)
		}
	}

	// Saving replaces the moves with human-verified ones (confidence 1.0).
	saved := []domain.Move{
		{Ply: 1, Side: "white", SAN: "e4", FenAfter: "x", IsLegal: true, Confidence: 1.0},
		{Ply: 2, Side: "black", SAN: "e5", FenAfter: "y", IsLegal: true, Confidence: 1.0},
	}
	if err := st.SaveGame(ctx, id, g.Header, g.StartFEN, saved, "1. e4 e5 *", nil, nil); err != nil {
		t.Fatalf("save game: %v", err)
	}
	g2, _, err := st.GetGame(ctx, id)
	if err != nil {
		t.Fatalf("get saved game: %v", err)
	}
	if g2.Status != domain.StatusSaved {
		t.Fatalf("status=%q, want saved", g2.Status)
	}
	for _, m := range g2.Moves {
		if m.Confidence != 1.0 {
			t.Errorf("saved move %d confidence=%.3f, want 1.0 (human-verified)", m.Ply, m.Confidence)
		}
	}
}

// TestDirectInsertDefaultsConfidence proves the column default (1.0) applies to any INSERT that
// omits confidence — manual entries and raw seed inserts stay confident by default.
func TestDirectInsertDefaultsConfidence(t *testing.T) {
	st := newStore(t)
	ctx := context.Background()
	var conf float64
	err := st.Pool.QueryRow(ctx, `
		WITH g AS (
			INSERT INTO games (source, status, result, start_fen)
			VALUES ('manual','draft','*','startfen') RETURNING id
		), m AS (
			INSERT INTO moves (game_id, ply, side, san) SELECT id, 1, 'white', 'e4' FROM g
			RETURNING confidence
		) SELECT confidence FROM m`).Scan(&conf)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	if conf != 1.0 {
		t.Fatalf("default confidence=%.2f, want 1.0", conf)
	}
}

func TestRateLimitWindow(t *testing.T) {
	st := newStore(t)
	ctx := context.Background()
	key := "test:1.2.3.4"
	for i := 1; i <= 3; i++ {
		ok, err := st.ConsumeRateLimit(ctx, key, 3, time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Fatalf("request %d should be allowed under limit 3", i)
		}
	}
	ok, _ := st.ConsumeRateLimit(ctx, key, 3, time.Hour)
	if ok {
		t.Fatal("4th request should be rate-limited")
	}
}
