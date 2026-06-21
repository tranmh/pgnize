package jobs

import (
	"strings"
	"testing"

	"github.com/tranmh/pgnize/internal/recognition"
)

const startFENPrefix = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR"

// A clean legal read is stored normalized with its reported confidence.
func TestPositionDraftFENLegal(t *testing.T) {
	res := recognition.PositionResult{
		Grid: []string{
			"....k...", "........", "........", "........",
			"........", "........", "........", "....K..R",
		},
		SideToMove: recognition.SideWhite, Orientation: "white_bottom", Confidence: 0.5,
	}
	fen, conf := positionDraftFEN(res, "job-legal")
	if want := "4k3/8/8/8/8/8/8/4K2R w - - 0 1"; fen != want {
		t.Fatalf("fen = %q, want %q", fen, want)
	}
	if conf != 0.5 {
		t.Fatalf("conf = %v, want 0.5", conf)
	}
}

// The core regression: a readable-but-illegal recognized position must be KEPT (not reset
// to the starting position) so the user can fix it in the editor. This is exactly what the
// production "always starting position" bug was: ~1/3 of real photos hit this path.
func TestPositionDraftFENKeepsIllegalRead(t *testing.T) {
	res := recognition.PositionResult{
		Grid: []string{
			"....k...", "....k...", "........", "........", // two black kings, no white king
			"........", "........", "....q...", "........",
		},
		SideToMove: recognition.SideWhite, Orientation: "white_bottom", Confidence: 0.5,
	}
	fen, conf := positionDraftFEN(res, "job-illegal")
	if strings.HasPrefix(fen, startFENPrefix) {
		t.Fatalf("must not fall back to the starting position, got %q", fen)
	}
	if want := "4k3/4k3/8/8/8/8/4q3/8 w - - 0 1"; fen != want {
		t.Fatalf("fen = %q, want %q", fen, want)
	}
	if conf != 0 {
		t.Fatalf("illegal read should report 0 confidence, got %v", conf)
	}
}

// Only a truly empty read falls back to the starting position.
func TestPositionDraftFENEmptyFallsBackToStart(t *testing.T) {
	fen, conf := positionDraftFEN(recognition.PositionResult{Grid: nil, Confidence: 0.5}, "job-empty")
	if !strings.HasPrefix(fen, startFENPrefix) {
		t.Fatalf("empty read should fall back to start, got %q", fen)
	}
	if conf != 0 {
		t.Fatalf("conf = %v, want 0", conf)
	}
}

func TestSafeRawJSON(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"valid object", `{"a":1}`, `{"a":1}`},
		{"valid array", `[{"white":"e4"}]`, `[{"white":"e4"}]`},
		{"empty string", "", "{}"},
		// A num_predict cap truncates the model output mid-JSON: invalid -> "{}"
		// so the result_raw_json::jsonb cast cannot fail with SQLSTATE 22P02.
		{"truncated object", `{"moves":[{"white":"e4","black":`, "{}"},
		{"whitespace only", "   ", "{}"},
		{"garbage", "not json", "{}"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := safeRawJSON(c.in); got != c.want {
				t.Errorf("safeRawJSON(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}
