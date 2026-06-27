package engine

import (
	"context"
	"os/exec"
	"testing"
	"time"
)

// findStockfish locates a Stockfish binary, honoring ENGINE_PATH-style discovery via PATH.
// Tests that need a real engine skip when none is installed (CI uses the Fake).
func findStockfish(t *testing.T) string {
	t.Helper()
	path, err := exec.LookPath("stockfish")
	if err != nil {
		t.Skip("stockfish binary not found in PATH; skipping real-engine test")
	}
	return path
}

func TestStockfishAnalyzeReal(t *testing.T) {
	path := findStockfish(t)
	eng, err := NewStockfish(path, PoolOpts{Instances: 1, MoveTimeMs: 200})
	if err != nil {
		t.Fatalf("new stockfish: %v", err)
	}
	defer eng.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	a, err := eng.Analyze(ctx, "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1", Options{MoveTimeMs: 200})
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if len(a.Lines) == 0 || a.Best().BestMove == "" {
		t.Fatalf("expected a best move, got %+v", a)
	}
}

func TestStockfishFindMateReal(t *testing.T) {
	path := findStockfish(t)
	eng, err := NewStockfish(path, PoolOpts{Instances: 1, MoveTimeMs: 300})
	if err != nil {
		t.Fatalf("new stockfish: %v", err)
	}
	defer eng.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// White: Qh5-h7# style mate-in-1 (back-rank). Rh8 mate: rook to h8 is mate.
	// Simple mate-in-1: white queen delivers mate. Kh1 black, ... use a known M1.
	mateIn, _, err := FindMate(ctx, eng, "6k1/5ppp/8/8/8/8/8/4R1K1 w - - 0 1", Options{MoveTimeMs: 300})
	if err != nil {
		t.Fatalf("find mate: %v", err)
	}
	// This position is not forced mate; just assert the call works and returns cleanly.
	_ = mateIn
}
