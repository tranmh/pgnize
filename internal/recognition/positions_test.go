package recognition

import (
	"context"
	"strings"
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

func TestAssembleFENFlipsWhiteOnTop(t *testing.T) {
	// The observed model failure: White's army is reported at the TOP of the grid with
	// files already in a→h order. Orientation is inferred from the piece distribution and
	// corrected with a vertical rank flip only (files are NOT mirrored, as that would
	// corrupt the already-correct file order). White King+Rook on top must end on rank 1.
	grid := []string{
		"....K..R", // white K+R at top, correct file order
		"........",
		"........",
		"........",
		"........",
		"........",
		"........",
		"....k...", // black king at bottom
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

// The model's self-reported orientation is unreliable, so it must be ignored: even when
// the model claims black_bottom, a grid with White clearly on the bottom must NOT flip.
func TestAssembleFENIgnoresWrongModelOrientationFlag(t *testing.T) {
	got, err := AssembleFEN(PositionResult{
		Grid:        startGrid(), // White on bottom (canonical)
		SideToMove:  SideWhite,
		Orientation: "black_bottom", // wrong flag — must be overridden by inference
	})
	if err != nil {
		t.Fatalf("AssembleFEN: %v", err)
	}
	want := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w - - 0 1"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// And the converse: a start position reported upside-down (White on top) is corrected to
// the canonical view by inference, regardless of the (here absent) model flag.
func TestAssembleFENCorrectsUpsideDownStart(t *testing.T) {
	grid := []string{
		"RNBQKBNR", // white back rank at top, files a→h
		"PPPPPPPP",
		"........",
		"........",
		"........",
		"........",
		"pppppppp",
		"rnbqkbnr", // black back rank at bottom
	}
	got, err := AssembleFEN(PositionResult{Grid: grid, SideToMove: SideWhite, Orientation: ""})
	if err != nil {
		t.Fatalf("AssembleFEN: %v", err)
	}
	want := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w - - 0 1"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// A short grid must be padded with empty ranks and assembled best-effort, never
// rejected — rejecting discards the read and resets the editor to the start.
func TestAssembleFENRepairsShortGrid(t *testing.T) {
	// 7 rows of a legal K vs k+R layout; the missing 8th rank is padded empty.
	grid := []string{
		"....k...",
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
	want := "4k3/8/8/8/8/8/4K2R/8 w - - 0 1"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// An over-long row must be truncated to 8 cells, never rejected.
func TestAssembleFENRepairsLongRow(t *testing.T) {
	grid := startGrid()
	grid[2] = "........." // 9 cells -> truncate to 8 empties
	got, err := AssembleFEN(PositionResult{Grid: grid, SideToMove: SideWhite, Orientation: "white_bottom"})
	if err != nil {
		t.Fatalf("AssembleFEN: %v", err)
	}
	want := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w - - 0 1"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// The core production bug: a well-formed but chess-illegal read (here two black kings
// and no white king) must still yield the RECOGNIZED board — not the starting position.
// AssembleFEN returns the best-effort FEN together with a non-nil error so the caller
// can flag low confidence, but the FEN must be usable and must not be the start.
func TestAssembleFENKeepsIllegalRecognizedPosition(t *testing.T) {
	grid := []string{
		"....k...",
		"....k...", // two black kings, no white king: illegal
		"........",
		"........",
		"........",
		"........",
		"....q...",
		"........",
	}
	got, err := AssembleFEN(PositionResult{Grid: grid, SideToMove: SideWhite, Orientation: "white_bottom"})
	if err == nil {
		t.Fatal("expected an error flagging the illegal position")
	}
	if got == "" {
		t.Fatal("illegal position must still return a best-effort FEN, got empty")
	}
	start := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR"
	if strings.HasPrefix(got, start) {
		t.Fatalf("must not reset to the starting position, got %q", got)
	}
	wantBoard := "4k3/4k3/8/8/8/8/4q3/8"
	if gotBoard := got[:len(got)-len(" w - - 0 1")]; gotBoard != wantBoard {
		t.Fatalf("board field = %q, want %q", gotBoard, wantBoard)
	}
}

// A completely empty grid (the model read nothing) returns no FEN so the caller may
// fall back to a sensible default.
func TestAssembleFENEmptyGrid(t *testing.T) {
	got, err := AssembleFEN(PositionResult{Grid: nil})
	if err == nil {
		t.Fatal("expected error for empty grid")
	}
	if got != "" {
		t.Fatalf("empty grid must return empty FEN, got %q", got)
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
