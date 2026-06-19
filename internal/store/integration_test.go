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
		if _, err := st.CreateJob(ctx, up.ID, nil, "fake", "fake"); err != nil {
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
