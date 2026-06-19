package chesskit

import (
	"errors"
	"testing"
)

func TestStartingFEN(t *testing.T) {
	want := FEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")
	if StartingFEN() != want {
		t.Fatalf("StartingFEN() = %q, want %q", StartingFEN(), want)
	}
}

func TestFENRoundTrip(t *testing.T) {
	// Validate a move and confirm the FEN it produces re-parses and replays.
	to, err := Validate(StartingFEN(), "e4")
	if err != nil {
		t.Fatalf("Validate e4: %v", err)
	}
	want := FEN("rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1")
	if to != want {
		t.Fatalf("after e4 = %q, want %q", to, want)
	}
	// FEN re-parses: legal moves from it should be non-empty (black to move).
	moves, err := LegalMovesSAN(to)
	if err != nil {
		t.Fatalf("LegalMovesSAN: %v", err)
	}
	if len(moves) != 20 {
		t.Fatalf("legal moves after e4 = %d, want 20", len(moves))
	}
}

func TestParseSAN_PiecesAndSpecials(t *testing.T) {
	cases := []struct {
		name string
		fen  FEN
		san  SAN
		want SAN // canonical SAN expected back
	}{
		{"pawn double", StartingFEN(), "e4", "e4"},
		{"knight", StartingFEN(), "Nf3", "Nf3"},
		{"knight queenside", StartingFEN(), "Nc3", "Nc3"},
		// Bishop after 1.e4 e5: Bc4
		{"bishop", "rnbqkbnr/pppp1ppp/8/4p3/4P3/8/PPPP1PPP/RNBQKBNR w KQkq e6 0 2", "Bc4", "Bc4"},
		// White short castle ready position
		{"castle short", "rnbqk2r/pppp1ppp/5n2/2b1p3/2B1P3/5N2/PPPP1PPP/RNBQK2R w KQkq - 4 4", "O-O", "O-O"},
		{"castle short zeros", "rnbqk2r/pppp1ppp/5n2/2b1p3/2B1P3/5N2/PPPP1PPP/RNBQK2R w KQkq - 4 4", "0-0", "O-O"},
		// Long castle position for white
		{"castle long", "r3kbnr/pppqpppp/2np4/8/3PP1b1/2N1BN2/PPPQ1PPP/R3KB1R w KQkq - 4 6", "O-O-O", "O-O-O"},
		{"castle long zeros", "r3kbnr/pppqpppp/2np4/8/3PP1b1/2N1BN2/PPPQ1PPP/R3KB1R w KQkq - 4 6", "0-0-0", "O-O-O"},
		// En passant: 1.e4 a6 2.e5 d5 3.exd6
		{"en passant", "rnbqkbnr/1pp1pppp/p7/3pP3/8/8/PPPP1PPP/RNBQKBNR w KQkq d6 0 3", "exd6", "exd6"},
		// Promotion to queen with check, white pawn a7 -> a8
		{"promotion queen", "4k3/P7/8/8/8/8/8/4K3 w - - 0 1", "a8=Q", "a8=Q+"},
		{"underpromotion knight", "4k3/P7/8/8/8/8/8/4K3 w - - 0 1", "a8=N", "a8=N"},
		// Capture by pawn: position with black pawn on d5, white pawn e4
		{"pawn capture", "rnbqkbnr/ppp1pppp/8/3p4/4P3/8/PPPP1PPP/RNBQKBNR w KQkq d6 0 2", "exd5", "exd5"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m, err := ParseSAN(c.fen, c.san)
			if err != nil {
				t.Fatalf("ParseSAN(%q) error: %v", c.san, err)
			}
			if m.SAN != c.want {
				t.Fatalf("canonical SAN = %q, want %q", m.SAN, c.want)
			}
			if m.FromFEN != c.fen {
				t.Errorf("FromFEN = %q, want %q", m.FromFEN, c.fen)
			}
			if m.ToFEN == "" {
				t.Errorf("ToFEN empty")
			}
		})
	}
}

func TestParseSAN_CheckAndMate(t *testing.T) {
	// Scholar's mate final move: Qxf7# from the position before the mate.
	fenBefore := FEN("r1bqkb1r/pppp1ppp/2n2n2/4p2Q/2B1P3/8/PPPP1PPP/RNB1K1NR w KQkq - 4 4")
	m, err := ParseSAN(fenBefore, "Qxf7#")
	if err != nil {
		t.Fatalf("Qxf7# error: %v", err)
	}
	if m.SAN != "Qxf7#" {
		t.Fatalf("canonical = %q, want Qxf7#", m.SAN)
	}

	// A checking (non-mate) move: Bb5+ in a Ruy-Lopez-style line. After
	// 1.e4 d6 2.d4 e5 it is White to move and Bb5 gives check.
	priorFEN := FEN("rnbqkbnr/ppp2ppp/3p4/4p3/3PP3/8/PPP2PPP/RNBQKBNR w KQkq - 0 3")
	mm, err := ParseSAN(priorFEN, "Bb5+")
	if err != nil {
		t.Fatalf("Bb5+ error: %v", err)
	}
	if mm.SAN != "Bb5+" {
		t.Fatalf("canonical = %q, want Bb5+", mm.SAN)
	}
}

func TestParseSAN_Illegal(t *testing.T) {
	_, err := ParseSAN(StartingFEN(), "e5")
	if !errors.Is(err, ErrIllegalMove) {
		t.Fatalf("expected ErrIllegalMove, got %v", err)
	}
	_, err = ParseSAN(StartingFEN(), "Qd5")
	if !errors.Is(err, ErrIllegalMove) {
		t.Fatalf("expected ErrIllegalMove for Qd5, got %v", err)
	}
}

// twoKnightsD2FEN is a position where knights on b1 and f3 both reach d2,
// so bare "Nd2" is ambiguous and Nbd2 / Nfd2 disambiguate.
const twoKnightsD2FEN = FEN("rnbqk2r/ppp1bppp/4pn2/3p4/8/3P1NP1/PPP1PPBP/RNBQK2R w KQkq - 1 5")

func TestParseSAN_Ambiguous(t *testing.T) {
	fen := twoKnightsD2FEN
	_, err := ParseSAN(fen, "Nd2")
	if !errors.Is(err, ErrAmbiguousMove) {
		t.Fatalf("expected ErrAmbiguousMove for Nd2, got %v", err)
	}
	// Disambiguated forms must work.
	if _, err := ParseSAN(fen, "Nfd2"); err != nil {
		t.Fatalf("Nfd2 should be legal: %v", err)
	}
	if _, err := ParseSAN(fen, "Nbd2"); err != nil {
		t.Fatalf("Nbd2 should be legal: %v", err)
	}
}

func TestLegalMovesSAN_StartCount(t *testing.T) {
	moves, err := LegalMovesSAN(StartingFEN())
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(moves) != 20 {
		t.Fatalf("legal moves from start = %d, want 20", len(moves))
	}
}

func TestLegalMovesSAN_Disambiguation(t *testing.T) {
	// Position where both knights can reach d2 -> expect Nbd2/Nfd2 disambiguation.
	moves, err := LegalMovesSAN(twoKnightsD2FEN)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	hasFd2, hasBd2 := false, false
	for _, m := range moves {
		if m == "Nfd2" {
			hasFd2 = true
		}
		if m == "Nbd2" {
			hasBd2 = true
		}
		// Bare "Nd2" must NOT appear; it would be ambiguous.
		if m == "Nd2" {
			t.Errorf("found ambiguous bare Nd2 in legal moves")
		}
	}
	if !hasFd2 || !hasBd2 {
		t.Fatalf("expected disambiguated Nfd2 and Nbd2, got %v", moves)
	}
}

func TestApplyMoves_ScholarsMate(t *testing.T) {
	sans := []SAN{"e4", "e5", "Bc4", "Nc6", "Qh5", "Nf6", "Qxf7#"}
	positions, err, failedAt := ApplyMoves(StartingFEN(), sans)
	if err != nil {
		t.Fatalf("Scholar's mate should be legal, err=%v failedAt=%d", err, failedAt)
	}
	if failedAt != -1 {
		t.Fatalf("failedAt = %d, want -1", failedAt)
	}
	if len(positions) != len(sans) {
		t.Fatalf("positions len = %d, want %d", len(positions), len(sans))
	}
	// Final position should be the FEN after Qxf7#.
	last := positions[len(positions)-1]
	if last == "" {
		t.Fatalf("empty final position")
	}
}

func TestApplyMoves_IllegalStopsWithFailedAt(t *testing.T) {
	// Same game but corrupt move index 4 (Qh5 -> Qh6 is illegal here).
	sans := []SAN{"e4", "e5", "Bc4", "Nc6", "Qh6", "Nf6"}
	positions, err, failedAt := ApplyMoves(StartingFEN(), sans)
	if err == nil {
		t.Fatalf("expected error on illegal move")
	}
	if !errors.Is(err, ErrIllegalMove) {
		t.Fatalf("expected ErrIllegalMove, got %v", err)
	}
	if failedAt != 4 {
		t.Fatalf("failedAt = %d, want 4", failedAt)
	}
	if len(positions) != 4 {
		t.Fatalf("positions len = %d, want 4 (moves applied before failure)", len(positions))
	}
}

func TestApplyMoves_AmbiguousStops(t *testing.T) {
	// Reach a position where knights on b1 and f3 both reach d2, then issue ambiguous Nd2.
	sans := []SAN{"Nf3", "d5", "g3", "e6", "Bg2", "Nf6", "d3", "Be7", "Nd2"}
	_, err, failedAt := ApplyMoves(StartingFEN(), sans)
	if !errors.Is(err, ErrAmbiguousMove) {
		t.Fatalf("expected ErrAmbiguousMove, got %v", err)
	}
	if failedAt != 8 {
		t.Fatalf("failedAt = %d, want 8", failedAt)
	}
}

func TestApplyMoves_Empty(t *testing.T) {
	positions, err, failedAt := ApplyMoves(StartingFEN(), nil)
	if err != nil || failedAt != -1 {
		t.Fatalf("empty: err=%v failedAt=%d", err, failedAt)
	}
	if len(positions) != 0 {
		t.Fatalf("expected no positions, got %d", len(positions))
	}
}
