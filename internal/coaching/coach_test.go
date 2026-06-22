package coaching

import (
	"context"
	"strings"
	"testing"
)

func ptrInt(n int) *int { return &n }

func sampleMove() MoveInput {
	return MoveInput{
		FEN:        "r1bqkbnr/pppp1ppp/2n5/4p3/2B1P3/5N2/PPPP1PPP/RNBQK2R b KQkq - 3 3",
		Side:       "black",
		PlayedSAN:  "Nf6",
		BestSAN:    "Bc5",
		BestLine:   []string{"Bc5", "O-O", "Nf6"},
		EvalBefore: Eval{Cp: ptrInt(30)},
		EvalAfter:  Eval{Cp: ptrInt(45)},
		Quality:    "inaccuracy",
	}
}

func TestFakeCoachMoveDeterministic(t *testing.T) {
	f := NewFake()
	a, err := f.CoachMove(context.Background(), sampleMove())
	if err != nil {
		t.Fatal(err)
	}
	b, _ := f.CoachMove(context.Background(), sampleMove())
	if a.Text != b.Text {
		t.Fatalf("non-deterministic: %q vs %q", a.Text, b.Text)
	}
	if a.Text == "" {
		t.Fatal("empty text")
	}
	if a.Model != "fake" {
		t.Fatalf("model=%q want fake", a.Model)
	}
	if a.Lang != "de" {
		t.Fatalf("default lang=%q want de", a.Lang)
	}
	for _, want := range []string{"Bc5", "Nf6"} {
		if !strings.Contains(a.Text, want) {
			t.Errorf("text missing %q: %s", want, a.Text)
		}
	}
}

func TestFakeCoachMoveEnglish(t *testing.T) {
	f := NewFake()
	in := sampleMove()
	in.Lang = "en"
	c, _ := f.CoachMove(context.Background(), in)
	if c.Lang != "en" {
		t.Fatalf("lang=%q want en", c.Lang)
	}
	if !strings.Contains(strings.ToLower(c.Text), "engine") {
		t.Errorf("expected english prose mentioning engine: %s", c.Text)
	}
}

func TestFakeCoachGame(t *testing.T) {
	f := NewFake()
	c, err := f.CoachGame(context.Background(), GameInput{
		StartFEN: "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
		Moves:    []GameMove{{Ply: 1, Side: "white", SAN: "e4"}, {Ply: 2, Side: "black", SAN: "e5"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if c.Text == "" {
		t.Fatal("empty game summary")
	}
	if c.Lang != "de" {
		t.Fatalf("lang=%q want de", c.Lang)
	}
}

func TestBuildMovePromptStableAndComplete(t *testing.T) {
	in := sampleMove()
	p1 := buildMovePrompt(in)
	p2 := buildMovePrompt(in)
	if p1 != p2 {
		t.Fatal("prompt not byte-stable across calls")
	}
	for _, want := range []string{"Nf6", "Bc5", "Deutsch", in.FEN} {
		if !strings.Contains(p1, want) {
			t.Errorf("prompt missing %q:\n%s", want, p1)
		}
	}
}

func TestFakeCoachMoveNoBestMove(t *testing.T) {
	// The engine may not supply an alternative (BestSAN empty). The coach must still
	// produce sensible prose from the played move + evals, not a dangling "prefers  ".
	f := NewFake()
	in := sampleMove()
	in.BestSAN = ""
	in.BestLine = nil
	c, err := f.CoachMove(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if c.Text == "" {
		t.Fatal("empty text")
	}
	if strings.Contains(c.Text, "bevorzugt  ") || strings.Contains(c.Text, "bevorzugt (") {
		t.Errorf("dangling empty best move in prose: %q", c.Text)
	}
	if !strings.Contains(c.Text, in.PlayedSAN) {
		t.Errorf("text should mention the played move %q: %s", in.PlayedSAN, c.Text)
	}
}

func TestBuildMovePromptNoBestMove(t *testing.T) {
	in := sampleMove()
	in.BestSAN = ""
	in.BestLine = nil
	p := buildMovePrompt(in)
	if strings.Contains(p, "Bester Zug der Engine:") {
		t.Errorf("must not emit an empty best-move line:\n%s", p)
	}
	if !strings.Contains(p, in.PlayedSAN) {
		t.Errorf("prompt should still contain the played move:\n%s", p)
	}
}

func TestBuildMovePromptEnglish(t *testing.T) {
	in := sampleMove()
	in.Lang = "en"
	p := buildMovePrompt(in)
	if !strings.Contains(p, "English") {
		t.Errorf("expected English instruction:\n%s", p)
	}
}

func TestBuildGamePromptStable(t *testing.T) {
	in := GameInput{
		StartFEN: "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
		Moves:    []GameMove{{Ply: 1, Side: "white", SAN: "e4"}, {Ply: 2, Side: "black", SAN: "e5"}},
	}
	if buildGamePrompt(in) != buildGamePrompt(in) {
		t.Fatal("game prompt not byte-stable")
	}
}

func TestFormatEval(t *testing.T) {
	cases := []struct {
		want string
		e    Eval
	}{
		{"+0.30", Eval{Cp: ptrInt(30)}},
		{"-1.50", Eval{Cp: ptrInt(-150)}},
		{"0.00", Eval{Cp: ptrInt(0)}},
		{"#3", Eval{Mate: ptrInt(3)}},
		{"-#2", Eval{Mate: ptrInt(-2)}},
		{"?", Eval{}},
	}
	for _, c := range cases {
		if got := formatEval(c.e); got != c.want {
			t.Errorf("formatEval(%+v)=%q want %q", c.e, got, c.want)
		}
	}
}
