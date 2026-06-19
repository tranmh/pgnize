package recognition

import (
	"context"
	"testing"

	"github.com/tranmh/chesskit"
)

func TestGermanToSAN(t *testing.T) {
	cases := map[string]string{
		"e4":      "e4",
		"Sf3":     "Nf3",   // Springer -> Knight
		"Lb5":     "Bb5",   // Läufer -> Bishop
		"Dh5":     "Qh5",   // Dame -> Queen
		"Td1":     "Rd1",   // Turm -> Rook
		"Kf1":     "Kf1",   // König -> King (same letter)
		"0-0":     "O-O",
		"0-0-0":   "O-O-O",
		"O-O":     "O-O",
		"Sxf3":    "Nxf3",
		"Sf3:":    "Nf3",   // German capture colon
		"exd6e.p.": "exd6",
		"e8D":     "e8=Q",  // promotion, German queen
		"e8=D":    "e8=Q",
		"e8Q":     "e8=Q",
		"Dxf7#":   "Qxf7#",
		"Lc4+":    "Bc4+",
		"remis":   "",      // result word, not a move
		"aufg.":   "",      // ressignation word
	}
	for in, want := range cases {
		if got := GermanToSAN(in); got != want {
			t.Errorf("GermanToSAN(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestReconcileLegalGame(t *testing.T) {
	tokens := []MoveToken{
		{Text: "e4"}, {Text: "e5"}, {Text: "Sf3"}, {Text: "Sc6"}, {Text: "Lb5"},
	}
	moves := Reconcile("", tokens)
	if len(moves) != 5 {
		t.Fatalf("want 5 moves, got %d", len(moves))
	}
	wantSAN := []string{"e4", "e5", "Nf3", "Nc6", "Bb5"}
	wantSide := []string{"white", "black", "white", "black", "white"}
	for i, m := range moves {
		if !m.IsLegal {
			t.Errorf("move %d (%s) marked illegal", i, m.SAN)
		}
		if m.SAN != wantSAN[i] {
			t.Errorf("move %d SAN=%q want %q", i, m.SAN, wantSAN[i])
		}
		if m.Side != wantSide[i] {
			t.Errorf("move %d side=%q want %q", i, m.Side, wantSide[i])
		}
		if m.FenAfter == "" {
			t.Errorf("move %d has empty fenAfter", i)
		}
	}
}

func TestReconcileIllegalBlocksDownstream(t *testing.T) {
	// 3rd move "Sf6" is illegal for White here (no knight reaches f6); everything after must block.
	tokens := []MoveToken{
		{Text: "e4"}, {Text: "e5"}, {Text: "Sf6"}, {Text: "Sc6"},
	}
	moves := Reconcile("", tokens)
	if !moves[0].IsLegal || !moves[1].IsLegal {
		t.Fatal("first two moves should be legal")
	}
	if moves[2].IsLegal {
		t.Fatal("third move (Sf6) should be illegal")
	}
	if moves[3].IsLegal {
		t.Fatal("fourth move must be blocked after an illegal move")
	}
}

func TestReconcileIllegibleBlocks(t *testing.T) {
	tokens := []MoveToken{{Text: "e4"}, {Text: "?"}, {Text: "Sf3"}}
	moves := Reconcile("", tokens)
	if !moves[0].IsLegal {
		t.Fatal("e4 should be legal")
	}
	if moves[1].IsLegal || moves[2].IsLegal {
		t.Fatal("an illegible '?' move and everything after must be blocked")
	}
}

func TestFakeRecognizerProducesLegalGame(t *testing.T) {
	res, err := NewFake().Recognize(context.Background(), ScoreSheetInput{})
	if err != nil {
		t.Fatal(err)
	}
	moves := Reconcile(string(chesskit.StartingFEN()), res.MoveTokens)
	for i, m := range moves {
		if !m.IsLegal {
			t.Fatalf("fake recognizer move %d (%s) should be legal", i, m.SAN)
		}
	}
}

func TestSelectFewShot(t *testing.T) {
	ex := []Example{{}, {}, {}, {}}
	if got := SelectFewShot(ex, 2); len(got) != 2 {
		t.Fatalf("want 2, got %d", len(got))
	}
	if got := SelectFewShot(ex, 0); got != nil {
		t.Fatal("max 0 should return nil")
	}
}
