//go:build integration

package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/tranmh/pgnize/internal/config"
	"github.com/tranmh/pgnize/internal/httpapi"
	"github.com/tranmh/pgnize/internal/jobs"
	"github.com/tranmh/pgnize/internal/recognition"
	"github.com/tranmh/pgnize/internal/storage"
	"github.com/tranmh/pgnize/internal/store"
	"github.com/tranmh/pgnize/migrations"
)

func testDBURL(t *testing.T) string {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	return url
}

type harness struct {
	ts     *httptest.Server
	st     *store.Store
	deps   jobs.Deps
	client *http.Client
}

func setup(t *testing.T) *harness {
	t.Helper()
	ctx := context.Background()
	url := testDBURL(t)
	if err := store.Migrate(url, migrations.FS, false); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	st, err := store.New(ctx, url)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	// Fresh slate.
	_, err = st.Pool.Exec(ctx, `TRUNCATE users, sessions, players, uploads, games, moves,
		recognition_jobs, feedback_corrections, rate_limit_entries RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Fatalf("truncate: %v", err)
	}
	blob, err := storage.NewFS(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{AuthSecret: "test-secret-32-bytes-xxxxxxxxxxxx", PublicBase: "http://x", UploadMaxBytes: 5 << 20, FewShotMax: 3}
	rec := recognition.NewFake()
	srv := &httpapi.Server{Cfg: cfg, Store: st, Storage: blob, Recognizer: rec}
	jar, _ := cookiejar.New(nil)
	h := &harness{
		ts:     httptest.NewServer(srv.Routes()),
		st:     st,
		deps:   jobs.Deps{Store: st, Storage: blob, Recognizer: rec, FewShotMax: 3},
		client: &http.Client{Jar: jar},
	}
	t.Cleanup(func() { h.ts.Close(); st.Close() })
	return h
}

func (h *harness) do(t *testing.T, method, path, ctype string, body io.Reader) (*http.Response, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, h.ts.URL+path, body)
	if err != nil {
		t.Fatal(err)
	}
	if ctype != "" {
		req.Header.Set("Content-Type", ctype)
	}
	resp, err := h.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp, b
}

func (h *harness) json(t *testing.T, method, path string, v any) (*http.Response, []byte) {
	buf, _ := json.Marshal(v)
	return h.do(t, method, path, "application/json", bytes.NewReader(buf))
}

func (h *harness) drainJob(t *testing.T) {
	t.Helper()
	job, err := h.st.ClaimNextJob(context.Background())
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if err := jobs.Process(context.Background(), h.deps, job); err != nil {
		t.Fatalf("process: %v", err)
	}
}

func uploadBody(t *testing.T, field string) (string, *bytes.Buffer) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile(field, "sheet.jpg")
	_, _ = fw.Write([]byte("fake-image-bytes"))
	mw.Close()
	return mw.FormDataContentType(), &buf
}

func TestAccountJourney(t *testing.T) {
	h := setup(t)

	// Register
	resp, body := h.json(t, "POST", "/api/auth/register",
		map[string]string{"name": "Magnus", "email": "m@example.com", "password": "hunter2hunter"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register status %d: %s", resp.StatusCode, body)
	}

	// Upload a sheet
	ct, buf := uploadBody(t, "image")
	resp, body = h.do(t, "POST", "/api/uploads", ct, buf)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("upload status %d: %s", resp.StatusCode, body)
	}
	var up struct{ JobID, UploadID string }
	json.Unmarshal(body, &up)
	if up.JobID == "" {
		t.Fatal("no jobId")
	}

	// Worker processes it
	h.drainJob(t)

	// Poll
	resp, body = h.do(t, "GET", "/api/jobs/"+up.JobID, "", nil)
	var js struct{ Status, GameID string }
	json.Unmarshal(body, &js)
	if js.Status != "done" || js.GameID == "" {
		t.Fatalf("job not done: %s", body)
	}

	// Get draft — fake recognizer yields a legal Ruy Lopez
	resp, body = h.do(t, "GET", "/api/games/"+js.GameID, "", nil)
	if resp.StatusCode != 200 {
		t.Fatalf("get game %d: %s", resp.StatusCode, body)
	}
	var draft struct {
		Moves []struct {
			San     string `json:"san"`
			IsLegal bool   `json:"isLegal"`
		} `json:"moves"`
		Header map[string]string `json:"header"`
		ImageURL string `json:"imageUrl"`
	}
	json.Unmarshal(body, &draft)
	if len(draft.Moves) == 0 || !draft.Moves[0].IsLegal {
		t.Fatalf("expected legal moves, got %s", body)
	}
	if draft.ImageURL == "" {
		t.Fatal("expected imageUrl on recognized draft")
	}

	// Save with corrected SAN list
	sans := []map[string]any{}
	for i, m := range draft.Moves {
		sans = append(sans, map[string]any{"ply": i + 1, "san": m.San})
	}
	resp, body = h.json(t, "PATCH", "/api/games/"+js.GameID, map[string]any{
		"header": map[string]string{"white": "Carlsen", "black": "Nepo", "result": "1-0"},
		"moves":  sans,
	})
	if resp.StatusCode != 200 {
		t.Fatalf("save status %d: %s", resp.StatusCode, body)
	}

	// Library shows it
	resp, body = h.do(t, "GET", "/api/games?q=Carlsen", "", nil)
	var lib struct {
		Total int `json:"total"`
		Games []struct{ ID, White string } `json:"games"`
	}
	json.Unmarshal(body, &lib)
	if lib.Total != 1 || lib.Games[0].White != "Carlsen" {
		t.Fatalf("library wrong: %s", body)
	}

	// PGN export contains the players + a known move
	resp, body = h.do(t, "GET", "/api/games/"+js.GameID+"/pgn", "", nil)
	pgn := string(body)
	if !strings.Contains(pgn, "Carlsen") || !strings.Contains(pgn, "Nf3") || !strings.Contains(pgn, "1-0") {
		t.Fatalf("pgn unexpected: %s", pgn)
	}

	// Feedback row was recorded (recognized + saved)
	var fbCount int
	h.st.Pool.QueryRow(context.Background(), `SELECT count(*) FROM feedback_corrections`).Scan(&fbCount)
	if fbCount != 1 {
		t.Fatalf("expected 1 feedback row, got %d", fbCount)
	}
}

func TestSaveRejectsIllegalMove(t *testing.T) {
	h := setup(t)
	h.json(t, "POST", "/api/auth/register",
		map[string]string{"name": "A", "email": "a@b.c", "password": "password12"})
	resp, body := h.json(t, "POST", "/api/games", map[string]string{"source": "manual"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("manual create %d: %s", resp.StatusCode, body)
	}
	var created struct {
		Game struct{ ID string } `json:"game"`
	}
	json.Unmarshal(body, &created)

	// e4 (legal) then Ke7 (illegal: e7 is occupied by Black's pawn)
	resp, body = h.json(t, "PATCH", "/api/games/"+created.Game.ID, map[string]any{
		"header": map[string]string{"white": "X", "black": "Y", "result": "*"},
		"moves":  []map[string]any{{"ply": 1, "san": "e4"}, {"ply": 2, "san": "Ke7"}},
	})
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d: %s", resp.StatusCode, body)
	}
	var ae struct {
		Error    string `json:"error"`
		FailedAt *int   `json:"failedAt"`
	}
	json.Unmarshal(body, &ae)
	if ae.Error != "illegal_move" || ae.FailedAt == nil || *ae.FailedAt != 1 {
		t.Fatalf("expected illegal_move failedAt=1, got %s", body)
	}
}

func TestAnonymousConvert(t *testing.T) {
	h := setup(t)
	ct, buf := uploadBody(t, "image")
	resp, body := h.do(t, "POST", "/api/convert", ct, buf)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("convert %d: %s", resp.StatusCode, body)
	}
	var c struct{ JobID string }
	json.Unmarshal(body, &c)
	h.drainJob(t)

	resp, body = h.do(t, "GET", "/api/convert/"+c.JobID, "", nil)
	var js struct{ Status string }
	json.Unmarshal(body, &js)
	if js.Status != "done" {
		t.Fatalf("anon job not done: %s", body)
	}
	resp, body = h.do(t, "GET", "/api/convert/"+c.JobID+"/game", "", nil)
	if resp.StatusCode != 200 {
		t.Fatalf("anon game %d: %s", resp.StatusCode, body)
	}
	var draft struct {
		Moves []map[string]any `json:"moves"`
	}
	json.Unmarshal(body, &draft)
	export := []map[string]any{}
	for i, m := range draft.Moves {
		export = append(export, map[string]any{"ply": i + 1, "san": m["san"]})
	}
	resp, body = h.json(t, "POST", "/api/convert/"+c.JobID+"/export", map[string]any{
		"header": map[string]string{"white": "Anon", "black": "Mouse", "result": "*"},
		"moves":  export,
	})
	if resp.StatusCode != 200 || !strings.Contains(string(body), "Anon") {
		t.Fatalf("anon export failed %d: %s", resp.StatusCode, body)
	}
}

func TestRateLimitConvert(t *testing.T) {
	h := setup(t)
	var got429 bool
	for i := 0; i < 12; i++ {
		ct, buf := uploadBody(t, "image")
		resp, _ := h.do(t, "POST", "/api/convert", ct, buf)
		if resp.StatusCode == http.StatusTooManyRequests {
			got429 = true
			break
		}
	}
	if !got429 {
		t.Fatal("expected a 429 within 12 anonymous convert requests (limit 10/hour)")
	}
	_ = time.Second
}
