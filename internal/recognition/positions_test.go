package recognition

import (
	"context"
	"testing"
)

func startGrid() []string {
	return []string{
		"rnbqkbnr",
		"pppppppp",
		"........",
		"........",
		"........",
		"........",
		"PPPPPPPP",
		"RNBQKBNR",
	}
}

func TestAssembleFENStartingPosition(t *testing.T) {
	got, err := AssembleFEN(PositionResult{Grid: startGrid(), SideToMove: SideWhite, Orientation: "white_bottom"})
	if err != nil {
		t.Fatalf("AssembleFEN: %v", err)
	}
	want := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w - - 0 1"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestAssembleFENSparseEndgame(t *testing.T) {
	grid := []string{
		"....k...",
		"........",
		"........",
		"........",
		"........",
		"........",
		"........",
		"....K..R",
	}
	got, err := AssembleFEN(PositionResult{Grid: grid, SideToMove: SideWhite, Orientation: "white_bottom"})
	if err != nil {
		t.Fatalf("AssembleFEN: %v", err)
	}
	want := "4k3/8/8/8/8/8/8/4K2R w - - 0 1"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestAssembleFENBlackToMove(t *testing.T) {
	got, err := AssembleFEN(PositionResult{Grid: startGrid(), SideToMove: SideBlack, Orientation: "white_bottom"})
	if err != nil {
		t.Fatalf("AssembleFEN: %v", err)
	}
	want := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR b - - 0 1"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestAssembleFENBlackBottomFlip(t *testing.T) {
	// A black-bottom photo of the K+R vs k endgame: the same physical board seen rotated
	// 180°. Flipping must recover the canonical white-bottom FEN.
	grid := []string{
		"R..K....",
		"........",
		"........",
		"........",
		"........",
		"........",
		"........",
		"...k....",
	}
	got, err := AssembleFEN(PositionResult{Grid: grid, SideToMove: SideWhite, Orientation: "black_bottom"})
	if err != nil {
		t.Fatalf("AssembleFEN: %v", err)
	}
	want := "4k3/8/8/8/8/8/8/4K2R w - - 0 1"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestAssembleFENRejectsWrongRowCount(t *testing.T) {
	seven := startGrid()[:7]
	if _, err := AssembleFEN(PositionResult{Grid: seven}); err == nil {
		t.Fatal("expected error for 7-row grid")
	}
}

func TestAssembleFENRejectsWrongRowLength(t *testing.T) {
	grid := startGrid()
	grid[2] = "........." // 9 cells
	if _, err := AssembleFEN(PositionResult{Grid: grid}); err == nil {
		t.Fatal("expected error for 9-char row")
	}
}

func TestAssembleFENSanitizesUnknownChars(t *testing.T) {
	// 'x' and '?' are unknown glyphs and must be treated as empty squares; a spurious space
	// is stripped, and a run-length digit ('8') is expanded.
	grid := []string{
		"4k3", // digits expand to ".......": 4 empties + k + 3 empties
		"8",
		"8",
		"8",
		"8",
		"8",
		"8",
		"4K2R",
	}
	got, err := AssembleFEN(PositionResult{Grid: grid, SideToMove: SideWhite, Orientation: "white_bottom"})
	if err != nil {
		t.Fatalf("AssembleFEN: %v", err)
	}
	want := "4k3/8/8/8/8/8/8/4K2R w - - 0 1"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestAssembleFENMapsUnknownGlyphsToEmpty(t *testing.T) {
	grid := []string{
		"xxxxkxxx", // unknown glyphs → empty, leaving a lone king
		"........",
		"........",
		"........",
		"........",
		"........",
		"........",
		"xxxxKxxR", // unknowns → empty, leaving K and R
	}
	got, err := AssembleFEN(PositionResult{Grid: grid, SideToMove: SideWhite, Orientation: "white_bottom"})
	if err != nil {
		t.Fatalf("AssembleFEN: %v", err)
	}
	want := "4k3/8/8/8/8/8/8/4K2R w - - 0 1"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestFakeRecognizePosition(t *testing.T) {
	res, err := NewFake().RecognizePosition(context.Background(), PositionInput{})
	if err != nil {
		t.Fatalf("RecognizePosition: %v", err)
	}
	if res.Confidence != 0.9 {
		t.Fatalf("confidence = %v, want 0.9", res.Confidence)
	}
	fen, err := AssembleFEN(res)
	if err != nil {
		t.Fatalf("AssembleFEN: %v", err)
	}
	want := "4k3/8/8/8/8/8/8/4K2R w - - 0 1"
	if fen != want {
		t.Fatalf("fake FEN = %q, want %q", fen, want)
	}
}
