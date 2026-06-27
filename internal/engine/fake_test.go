package engine

import (
	"context"
	"testing"
)

func TestFakeDeterministic(t *testing.T) {
	f := NewFake()
	fen := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	a1, err := f.Analyze(context.Background(), fen, Options{})
	if err != nil {
		t.Fatal(err)
	}
	a2, _ := f.Analyze(context.Background(), fen, Options{})
	if len(a1.Lines) != 1 || len(a2.Lines) != 1 {
		t.Fatalf("want 1 line each, got %d/%d", len(a1.Lines), len(a2.Lines))
	}
	if *a1.Best().Cp != *a2.Best().Cp {
		t.Errorf("non-deterministic: %d vs %d", *a1.Best().Cp, *a2.Best().Cp)
	}
}

func TestFakeMultiPV(t *testing.T) {
	f := NewFake()
	a, _ := f.Analyze(context.Background(), "fen-x", Options{MultiPV: 3})
	if len(a.Lines) != 3 {
		t.Fatalf("want 3 lines, got %d", len(a.Lines))
	}
	// Alternatives are weakly ordered (best first).
	if a.Lines[0].score() < a.Lines[2].score() {
		t.Errorf("expected line 0 >= line 2, got %d < %d", a.Lines[0].score(), a.Lines[2].score())
	}
}

func TestFakeSeedMate(t *testing.T) {
	f := NewFake()
	fen := "1r6/8/8/8/8/8/5PPP/6K1 w - - 0 1"
	mate := 2
	f.Seed(fen, Analysis{FEN: fen, Lines: []Line{{Mate: &mate, PV: []string{"d1d8"}, BestMove: "d1d8"}}})

	mateIn, pv, err := FindMate(context.Background(), f, fen, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if mateIn == nil || *mateIn != 2 {
		t.Fatalf("mateIn = %v, want 2", mateIn)
	}
	if len(pv) != 1 || pv[0] != "d1d8" {
		t.Errorf("pv = %v, want [d1d8]", pv)
	}
}

func TestFindMateNoneWhenCpOnly(t *testing.T) {
	f := NewFake()
	mateIn, _, err := FindMate(context.Background(), f, "no-mate-here", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if mateIn != nil {
		t.Errorf("mateIn = %v, want nil", mateIn)
	}
}

func TestEvalMoveDeltaSign(t *testing.T) {
	f := NewFake()
	parent := "parent-fen"
	// Seed the parent so the best line is clearly +100 (mover POV).
	best := 100
	f.Seed(parent, Analysis{FEN: parent, Lines: []Line{{Cp: &best, PV: []string{"g1f3"}, BestMove: "g1f3"}}})
	// The child (opponent to move) evaluates at +80 for the opponent -> -80 for the mover.
	child := "child-fen"
	oppBest := 80
	f.Seed(child, Analysis{FEN: child, Lines: []Line{{Cp: &oppBest, PV: []string{"e7e5"}, BestMove: "e7e5"}}})

	b, played, delta, err := EvalMove(context.Background(), f, parent, child, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if *b.Cp != 100 {
		t.Errorf("best cp = %d, want 100", *b.Cp)
	}
	if *played.Cp != -80 {
		t.Errorf("played cp (mover POV) = %d, want -80", *played.Cp)
	}
	if delta != -180 {
		t.Errorf("delta = %d, want -180 (played - best)", delta)
	}
}

func TestLineScoreMateOrdering(t *testing.T) {
	m1, m5 := 1, 5
	mNeg1, mNeg5 := -1, -5
	if (Line{Mate: &m1}).score() <= (Line{Mate: &m5}).score() {
		t.Error("mate in 1 should outrank mate in 5")
	}
	if (Line{Mate: &mNeg1}).score() >= (Line{Mate: &mNeg5}).score() {
		t.Error("getting mated in 1 should be worse than in 5")
	}
	cp := 900
	if (Line{Mate: &m5}).score() <= (Line{Cp: &cp}).score() {
		t.Error("any mate should outrank a finite cp score")
	}
}
