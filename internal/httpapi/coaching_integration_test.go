//go:build integration

package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

const startFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

const samplePGN = `[Event "Test"]
[White "Alice"]
[Black "Bob"]
[Result "1-0"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 1-0
`

func (h *harness) coachingRows(t *testing.T) int {
	t.Helper()
	var n int
	if err := h.st.Pool.QueryRow(context.Background(), `SELECT count(*) FROM game_coaching`).Scan(&n); err != nil {
		t.Fatalf("count coaching: %v", err)
	}
	return n
}

func (h *harness) register(t *testing.T, name, email string) {
	t.Helper()
	resp, body := h.json(t, "POST", "/api/auth/register",
		map[string]string{"name": name, "email": email, "password": "password12"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register %d: %s", resp.StatusCode, body)
	}
}

func TestPasteFENAnonymous(t *testing.T) {
	h := setup(t)

	resp, body := h.json(t, "POST", "/api/positions", map[string]string{"fen": startFEN})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("paste fen %d: %s", resp.StatusCode, body)
	}
	var draft struct {
		StartFEN string `json:"startFen"`
		Moves    []any  `json:"moves"`
	}
	if err := json.Unmarshal(body, &draft); err != nil {
		t.Fatal(err)
	}
	if draft.StartFEN == "" {
		t.Errorf("expected startFen set, got empty: %s", body)
	}
	if draft.Moves == nil {
		t.Errorf("moves must be a (possibly empty) array, not null: %s", body)
	}

	// Illegal FEN is rejected.
	resp, _ = h.json(t, "POST", "/api/positions", map[string]string{"fen": "totally-bogus"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("illegal fen status %d want 400", resp.StatusCode)
	}
}

func TestImportPGNAnonymous(t *testing.T) {
	h := setup(t)

	resp, body := h.json(t, "POST", "/api/import", map[string]string{"pgn": samplePGN})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("import %d: %s", resp.StatusCode, body)
	}
	var out struct {
		Games []struct {
			Moves []struct {
				SAN     string `json:"san"`
				IsLegal bool   `json:"isLegal"`
			} `json:"moves"`
		} `json:"games"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Games) != 1 {
		t.Fatalf("expected 1 game, got %d: %s", len(out.Games), body)
	}
	if len(out.Games[0].Moves) != 6 {
		t.Fatalf("expected 6 plies, got %d", len(out.Games[0].Moves))
	}
	for _, m := range out.Games[0].Moves {
		if !m.IsLegal {
			t.Errorf("imported move %s not legal", m.SAN)
		}
	}
}

func TestCoachMoveAnonymousNoCache(t *testing.T) {
	h := setup(t)

	ply := 0
	resp, body := h.json(t, "POST", "/api/coach/move", map[string]any{
		"fen":        startFEN,
		"side":       "white",
		"playedSan":  "e4",
		"bestSan":    "d4",
		"ply":        ply,
		"evalBefore": map[string]any{"cp": 20},
		"evalAfter":  map[string]any{"cp": 15},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("coach %d: %s", resp.StatusCode, body)
	}
	var cr struct {
		Text   string `json:"text"`
		Cached bool   `json:"cached"`
	}
	json.Unmarshal(body, &cr)
	if cr.Text == "" {
		t.Error("expected coaching text")
	}
	if cr.Cached {
		t.Error("anonymous coaching (no gameId) must not be cached")
	}
	if n := h.coachingRows(t); n != 0 {
		t.Fatalf("anonymous coaching must not write a cache row, got %d", n)
	}
}

func TestCoachMoveCachingRegistered(t *testing.T) {
	h := setup(t)
	h.register(t, "Carla", "carla@example.com")

	// A persisted position draft (logged-in) gives us a game id to cache against.
	resp, body := h.json(t, "POST", "/api/positions", map[string]string{"fen": startFEN})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("paste fen %d: %s", resp.StatusCode, body)
	}
	var draft struct {
		ID string `json:"id"`
	}
	json.Unmarshal(body, &draft)
	if draft.ID == "" {
		t.Fatalf("expected persisted draft id for logged-in user: %s", body)
	}

	req := map[string]any{
		"gameId":     draft.ID,
		"ply":        0,
		"fen":        startFEN,
		"side":       "white",
		"playedSan":  "e4",
		"bestSan":    "d4",
		"evalBefore": map[string]any{"cp": 20},
		"evalAfter":  map[string]any{"cp": 15},
	}

	resp, body = h.json(t, "POST", "/api/coach/move", req)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("coach 1 %d: %s", resp.StatusCode, body)
	}
	var first struct {
		Text   string `json:"text"`
		Cached bool   `json:"cached"`
	}
	json.Unmarshal(body, &first)
	if first.Cached {
		t.Error("first call should not be cached")
	}

	resp, body = h.json(t, "POST", "/api/coach/move", req)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("coach 2 %d: %s", resp.StatusCode, body)
	}
	var second struct {
		Text   string `json:"text"`
		Cached bool   `json:"cached"`
	}
	json.Unmarshal(body, &second)
	if !second.Cached {
		t.Error("second identical call should be cached")
	}
	if second.Text != first.Text {
		t.Errorf("cached text differs: %q vs %q", second.Text, first.Text)
	}
	if n := h.coachingRows(t); n != 1 {
		t.Fatalf("expected exactly 1 cache row, got %d", n)
	}
}

func TestRateLimitCoach(t *testing.T) {
	h := setup(t)
	req := map[string]any{
		"fen": startFEN, "side": "white", "playedSan": "e4", "bestSan": "d4",
		"evalBefore": map[string]any{"cp": 20}, "evalAfter": map[string]any{"cp": 15},
	}
	limited := false
	for i := 0; i < 65; i++ {
		resp, _ := h.json(t, "POST", "/api/coach/move", req)
		if resp.StatusCode == http.StatusTooManyRequests {
			limited = true
			break
		}
	}
	if !limited {
		t.Fatal("expected a 429 after exceeding the coach rate limit")
	}
}
