package chat

import (
	"context"
	"testing"

	"github.com/tranmh/pgnize/internal/engine"
)

const startFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

// spyEngine records whether Analyze was called and can fail the test if it is — used to
// prove illegal tool args never reach the engine.
type spyEngine struct {
	t            *testing.T
	calls        int
	failIfCalled bool
}

func (s *spyEngine) Name() string  { return "spy" }
func (s *spyEngine) Close() error  { return nil }
func (s *spyEngine) Analyze(_ context.Context, fen string, opts engine.Options) (engine.Analysis, error) {
	s.calls++
	if s.failIfCalled {
		s.t.Errorf("engine.Analyze called with %q — illegal input must be rejected before the engine", fen)
	}
	cp := 30
	return engine.Analysis{FEN: fen, Lines: []engine.Line{{Cp: &cp, PV: []string{"e2e4"}, BestMove: "e2e4"}}}, nil
}

func TestDispatchAnalyzePosition(t *testing.T) {
	res := dispatch(context.Background(), engine.NewFake(), "analyze_position", map[string]any{"fen": startFEN}, startFEN)
	if _, bad := res["error"]; bad {
		t.Fatalf("unexpected error: %v", res)
	}
	if res["best_move"] == "" {
		t.Errorf("expected a best move, got %v", res)
	}
}

func TestDispatchIllegalFENNeverCallsEngine(t *testing.T) {
	spy := &spyEngine{t: t, failIfCalled: true}
	res := dispatch(context.Background(), spy, "analyze_position", map[string]any{"fen": "not-a-fen"}, "not-a-fen")
	if res["error"] == nil {
		t.Fatalf("expected error for illegal fen, got %v", res)
	}
	if spy.calls != 0 {
		t.Errorf("engine called %d times for illegal fen, want 0", spy.calls)
	}
}

func TestDispatchIllegalMoveNeverCallsEngine(t *testing.T) {
	spy := &spyEngine{t: t, failIfCalled: true}
	res := dispatch(context.Background(), spy, "evaluate_move", map[string]any{"fen": startFEN, "move": "Qd5"}, startFEN)
	if res["error"] == nil {
		t.Fatalf("expected error for illegal move, got %v", res)
	}
	if spy.calls != 0 {
		t.Errorf("engine called %d times for illegal move, want 0", spy.calls)
	}
}

func TestDispatchEvaluateLegalMove(t *testing.T) {
	res := dispatch(context.Background(), engine.NewFake(), "evaluate_move", map[string]any{"fen": startFEN, "move": "e4"}, startFEN)
	if res["error"] != nil {
		t.Fatalf("unexpected error: %v", res)
	}
	if res["move"] != "e4" {
		t.Errorf("move = %v, want e4", res["move"])
	}
	if _, ok := res["delta_cp"]; !ok {
		t.Errorf("expected delta_cp in result, got %v", res)
	}
}

func TestDispatchFindMate(t *testing.T) {
	f := engine.NewFake()
	mate := 2
	f.Seed(startFEN, engine.Analysis{FEN: startFEN, Lines: []engine.Line{{Mate: &mate, PV: []string{"d1h5"}, BestMove: "d1h5"}}})
	res := dispatch(context.Background(), f, "find_mate", map[string]any{"fen": startFEN}, startFEN)
	if m, _ := res["mate"].(bool); !m {
		t.Fatalf("expected mate=true, got %v", res)
	}
	if res["mate_in"] != 2 {
		t.Errorf("mate_in = %v, want 2", res["mate_in"])
	}
}

func TestDispatchFallbackFEN(t *testing.T) {
	// Empty fen arg should fall back to the conversation FEN.
	res := dispatch(context.Background(), engine.NewFake(), "analyze_position", map[string]any{}, startFEN)
	if res["error"] != nil {
		t.Fatalf("unexpected error using fallback fen: %v", res)
	}
}
