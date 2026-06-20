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
		// Long algebraic notation the model was not trained on -> reduce to short SAN.
		"Sf3-e5":  "Ne5",   // piece long move
		"Sf3xe5":  "Nxe5",  // piece long capture
		"Td1-d8":  "Rd8",   // rook long move
		"e2-e4":   "e4",    // pawn long move
		"e4:d5":   "exd5",  // pawn long capture (German colon)
		"e4xd5":   "exd5",  // pawn long capture (x)
	}
	for in, want := range cases {
		if got := GermanToSAN(in); got != want {
			t.Errorf("GermanToSAN(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestReconcileConfidenceForCleanGame(t *testing.T) {
	tokens := []MoveToken{{Text: "e4"}, {Text: "e5"}, {Text: "Sf3"}}
	moves := Reconcile("", tokens)
	for i, m := range moves {
		if m.Confidence < confThreshold {
			t.Errorf("clean legal move %d (%s) confidence=%.2f, want >= %.2f", i, m.SAN, m.Confidence, confThreshold)
		}
	}
}

func TestReconcileConfidenceForAutoCorrected(t *testing.T) {
	// "Nf6" after 1.e4 e5 auto-corrects to Nf3 -> legal but low confidence (verify).
	moves := Reconcile("", []MoveToken{{Text: "e4"}, {Text: "e5"}, {Text: "Nf6"}})
	m := moves[2]
	if !m.IsLegal || !m.Corrected {
		t.Fatalf("expected auto-corrected legal move, got legal=%v corrected=%v", m.IsLegal, m.Corrected)
	}
	if m.Confidence >= confThreshold {
		t.Fatalf("auto-corrected move confidence=%.2f, want < %.2f (flagged for verify)", m.Confidence, confThreshold)
	}
}

func TestReconcileAmbiguousAutoPickContinues(t *testing.T) {
	// Reach a position where knights on b1 and f3 both reach d2, then a bare ambiguous "Nd2".
	// The game must CONTINUE on a deterministic disambiguation, flagged low-confidence with
	// the alternatives offered as suggestions.
	tokens := []MoveToken{
		{Text: "Sf3"}, {Text: "d5"}, {Text: "g3"}, {Text: "e6"}, {Text: "Lg2"},
		{Text: "Sf6"}, {Text: "d3"}, {Text: "Le7"}, {Text: "Sd2"}, // ambiguous knight to d2
		{Text: "0-0"}, // must still be reachable -> proves the game wasn't blocked
	}
	moves := Reconcile("", tokens)
	amb := moves[8]
	if !amb.IsLegal {
		t.Fatalf("ambiguous Nd2 should be auto-resolved to a legal move, got illegal (san=%q)", amb.SAN)
	}
	if !amb.Corrected {
		t.Fatalf("auto-picked disambiguation must be flagged corrected")
	}
	if amb.SAN != "Nbd2" && amb.SAN != "Nfd2" {
		t.Fatalf("expected a disambiguated knight move, got %q", amb.SAN)
	}
	if amb.Confidence >= confThreshold {
		t.Fatalf("ambiguous auto-pick confidence=%.2f, want < %.2f", amb.Confidence, confThreshold)
	}
	if len(amb.Suggestions) < 2 {
		t.Fatalf("expected >=2 disambiguation suggestions, got %v", amb.Suggestions)
	}
	if !moves[9].IsLegal {
		t.Fatalf("move after the ambiguous one (O-O) must be legal: the game must not be blocked")
	}
}

func TestReconcileIllegalHasZeroConfidence(t *testing.T) {
	moves := Reconcile("", []MoveToken{{Text: "e4"}, {Text: "e5"}, {Text: "zzz"}, {Text: "Sc6"}})
	if moves[2].Confidence != 0 {
		t.Errorf("illegal move confidence=%.2f, want 0", moves[2].Confidence)
	}
	if moves[3].Confidence != 0 {
		t.Errorf("blocked-downstream move confidence=%.2f, want 0", moves[3].Confidence)
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
	// "zzz" is garbage with no near-legal move, so it can't be auto-corrected;
	// it and everything after it must block.
	tokens := []MoveToken{
		{Text: "e4"}, {Text: "e5"}, {Text: "zzz"}, {Text: "Sc6"},
	}
	moves := Reconcile("", tokens)
	if !moves[0].IsLegal || !moves[1].IsLegal {
		t.Fatal("first two moves should be legal")
	}
	if moves[2].IsLegal {
		t.Fatal("garbage move should be illegal")
	}
	if moves[3].IsLegal {
		t.Fatal("fourth move must be blocked after an illegal move")
	}
}

func TestReconcileAutoCorrectsConfidentMisread(t *testing.T) {
	// After 1.e4 e5, White "Nf6" is illegal (digit misread); the only legal move within
	// one edit is Nf3, so it is auto-corrected and the game continues.
	tokens := []MoveToken{{Text: "e4"}, {Text: "e5"}, {Text: "Nf6"}}
	moves := Reconcile("", tokens)
	m := moves[2]
	if !m.IsLegal || !m.Corrected || m.SAN != "Nf3" {
		t.Fatalf("expected auto-correct to Nf3 (legal,corrected), got san=%q legal=%v corrected=%v",
			m.SAN, m.IsLegal, m.Corrected)
	}
	if m.RecognizedText != "Nf6" {
		t.Fatalf("recognizedText must preserve the original read, got %q", m.RecognizedText)
	}
}

func TestReconcileSuggestsWhenAmbiguous(t *testing.T) {
	// "Qh4" after 1.e4 e5 is illegal with two equally-close legal moves (Qg4, Qh5),
	// so it is NOT auto-applied but ranked suggestions are offered.
	tokens := []MoveToken{{Text: "e4"}, {Text: "e5"}, {Text: "Qh4"}}
	moves := Reconcile("", tokens)
	m := moves[2]
	if m.IsLegal || m.Corrected {
		t.Fatalf("ambiguous misread must not be auto-applied, got legal=%v corrected=%v", m.IsLegal, m.Corrected)
	}
	if len(m.Suggestions) == 0 {
		t.Fatal("expected ranked legal-move suggestions")
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

func TestFakeRecognizerFlagsAmbiguousMove(t *testing.T) {
	res, err := NewFake().Recognize(context.Background(), ScoreSheetInput{})
	if err != nil {
		t.Fatal(err)
	}
	moves := Reconcile(string(chesskit.StartingFEN()), res.MoveTokens)
	var verify int
	for _, m := range moves {
		if !m.IsLegal {
			t.Fatalf("fake move %d (%s) should be legal", m.Ply, m.SAN)
		}
		if m.IsLegal && m.Confidence < confThreshold {
			verify++
			if !m.Corrected || len(m.Suggestions) < 2 {
				t.Errorf("low-confidence fake move %d should be corrected with suggestions, got corrected=%v suggestions=%v",
					m.Ply, m.Corrected, m.Suggestions)
			}
		}
	}
	if verify != 1 {
		t.Fatalf("expected exactly one low-confidence (verify) move in the fake game, got %d", verify)
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
