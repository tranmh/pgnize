package chesskit

import "testing"

func TestNormalizeFENValidStart(t *testing.T) {
	got, err := NormalizeFEN(StartingFEN())
	if err != nil {
		t.Fatalf("NormalizeFEN(start): %v", err)
	}
	if got != StartingFEN() {
		t.Fatalf("normalized start = %q, want %q", got, StartingFEN())
	}
}

func TestNormalizeFENMissingWhiteKing(t *testing.T) {
	// Black king present, white king replaced by a rook → must be rejected.
	if _, err := NormalizeFEN("4k3/8/8/8/8/8/8/4R3 w - - 0 1"); err == nil {
		t.Fatal("expected error for missing white king")
	}
}

func TestNormalizeFENPawnOnRank8(t *testing.T) {
	if _, err := NormalizeFEN("P3k3/8/8/8/8/8/8/4K3 w - - 0 1"); err == nil {
		t.Fatal("expected error for pawn on rank 8")
	}
}

func TestNormalizeFENMalformed(t *testing.T) {
	if _, err := NormalizeFEN("not a fen at all"); err == nil {
		t.Fatal("expected error for malformed FEN")
	}
}

func TestNormalizeFENCanonicalRoundTrip(t *testing.T) {
	in := FEN("4k3/8/8/8/8/8/8/4K2R w - - 0 1")
	got, err := NormalizeFEN(in)
	if err != nil {
		t.Fatalf("NormalizeFEN: %v", err)
	}
	if got != in {
		t.Fatalf("round-trip = %q, want %q", got, in)
	}
	// Idempotent: normalizing the output yields the same value.
	again, err := NormalizeFEN(got)
	if err != nil || again != got {
		t.Fatalf("not idempotent: %q (%v)", again, err)
	}
}
