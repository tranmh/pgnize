package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tranmh/pgnize/internal/coaching"
	"github.com/tranmh/pgnize/internal/config"
)

const startFENForTest = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

func coachIntPtr(n int) *int { return &n }

// coachTestServer is a DB-free server: rate limiting is disabled and the fake coach needs
// no model. Stateless coaching (no gameId) never touches the (nil) Store.
func coachTestServer() *Server {
	return &Server{Cfg: config.Config{RateLimitDisabled: true}, Coach: coaching.NewFake()}
}

func postCoach(s *Server, body coachMoveRequest) *httptest.ResponseRecorder {
	buf, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/coach/move", bytes.NewReader(buf))
	rec := httptest.NewRecorder()
	s.handleCoachMove(rec, req)
	return rec
}

func TestHandleCoachMoveStateless(t *testing.T) {
	s := coachTestServer()
	rec := postCoach(s, coachMoveRequest{
		FEN:        startFENForTest,
		Side:       "white",
		PlayedSan:  "e4",
		BestSan:    "d4",
		EvalBefore: coaching.Eval{Cp: coachIntPtr(20)},
		EvalAfter:  coaching.Eval{Cp: coachIntPtr(15)},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp coachResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Text == "" {
		t.Error("expected non-empty coaching text")
	}
	if resp.Cached {
		t.Error("must not be cached without a gameId")
	}
	if resp.Lang != "de" {
		t.Errorf("lang=%q want de (German-first default)", resp.Lang)
	}
}

func TestHandleCoachMoveIllegalFEN(t *testing.T) {
	s := coachTestServer()
	rec := postCoach(s, coachMoveRequest{FEN: "not-a-fen", Side: "white", PlayedSan: "e4", BestSan: "d4"})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code=%d want 400", rec.Code)
	}
}

func TestHandleCoachMoveMissingMoves(t *testing.T) {
	s := coachTestServer()
	rec := postCoach(s, coachMoveRequest{FEN: startFENForTest, Side: "white"})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code=%d want 400", rec.Code)
	}
}

func TestHandleCoachMoveIllegalPlayedMove(t *testing.T) {
	s := coachTestServer()
	// e5 is not legal for White from the starting position.
	rec := postCoach(s, coachMoveRequest{FEN: startFENForTest, Side: "white", PlayedSan: "e5", BestSan: "d4"})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code=%d want 400", rec.Code)
	}
}
