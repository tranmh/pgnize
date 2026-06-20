package recognition

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/tranmh/chesskit"
	"github.com/tranmh/pgnize/internal/domain"
)

// expectedBrennerSAN is the human-verified move list for testdata/brenner_tran.jpg —
// a real handwritten German Partieformular (Schach Niggemann) of the over-the-board game
// Markus Brenner (White) vs Minh Cuong Tran (Black), board 1, 2026-03-15, drawn (½–½).
//
// It was transcribed by reading the score-sheet image directly (Claude Code "viewing" the
// game, column by column at high zoom) and every ply was then replayed through a chess
// engine, so this prefix is GUARANTEED to be a legal, coherent game — it is the ground
// truth the recognition backends are scored against below.
//
// Only the first 48 moves (96 plies) are committed: the remaining moves 49–60 are a
// drawn queen-and-bishop endgame whose handwriting is not legible enough to transcribe
// with confidence, so they are deliberately left out rather than guessed. The sheet's
// final "(=)" mark records the draw. Scoring against a verified prefix is the right call:
// it covers the entire opening and middlegame — exactly where backends diverge — without
// asserting moves we could not verify.
var expectedBrennerSAN = []string{
	"e4", "c6", "c4", "d5", "cxd5", "cxd5", "exd5", "Nf6", "Qa4+", "Bd7",
	"Qb3", "Qc7", "Nc3", "Na6", "d4", "Rc8", "Nf3", "e6", "dxe6", "Bxe6",
	"Bb5+", "Nfd7", "Qd1", "Bb4", "Bd2", "O-O", "O-O", "Nf6", "a3", "Be7",
	"Rfe1", "Rfd8", "Qe2", "Nb8", "h3", "a6", "Bd3", "Nc6", "Be3", "b5",
	"Rac1", "Qd6", "Red1", "Bb3", "Rde1", "Bf8", "Ne4", "Nxe4", "Bxe4", "Bd5",
	"Bxd5", "Qxd5", "Red1", "Be7", "Rc2", "Na5", "Rdc1", "Rxc2", "Rxc2", "Bf6",
	"a4", "Nb3", "axb5", "axb5", "Qd1", "Ra8", "Kh2", "g6", "Rc3", "Qd6+",
	"g3", "Na5", "Ra3", "Qd5", "Qa1", "Qxf3", "Rxa5", "Rd8", "Qh1", "Qe2",
	"Ra8", "Kg7", "Rxd8", "Bxd8", "Qd5", "Bf6", "b3", "h5", "h4", "Qd3",
	"Qf3", "Qc2", "Kg2", "Be7", "Qb7", "Bf6",
}

const brennerImage = "../../testdata/brenner_tran.jpg"

// TestBrennerTranOllama runs the single 60-move sheet through the local Ollama VLM and
// scores its reconciled output against expectedBrennerSAN. Gated like the other real
// recognition tests, so it never runs in `make test`/CI.
//
//	RUN_REAL_RECOGNITION=1 go test ./internal/recognition/ -run BrennerTranOllama -v
//
// Overrides: OLLAMA_HOST (default http://localhost:11434), OLLAMA_MODEL (default minicpm-v).
func TestBrennerTranOllama(t *testing.T) {
	if os.Getenv("RUN_REAL_RECOGNITION") == "" {
		t.Skip("set RUN_REAL_RECOGNITION=1 (and run a local Ollama) to exercise real recognition")
	}
	host := envStr("OLLAMA_HOST", "http://localhost:11434")
	model := envStr("OLLAMA_MODEL", "minicpm-v")
	runBrenner(t, NewOllama(host, model), "../../testdata/brenner_tran_RESULTS_OLLAMA.txt")
}

// TestBrennerTranGemini runs the same sheet through Gemini Flash and scores it the same way,
// so the two backends can be compared on a single known-answer game. Gated behind
// RUN_REAL_RECOGNITION=1 and a GEMINI_API_KEY.
//
//	RUN_REAL_RECOGNITION=1 GEMINI_API_KEY=... go test ./internal/recognition/ -run BrennerTranGemini -v
//
// Overrides: GEMINI_HOST, GEMINI_MODEL (default gemini-2.5-flash).
func TestBrennerTranGemini(t *testing.T) {
	if os.Getenv("RUN_REAL_RECOGNITION") == "" {
		t.Skip("set RUN_REAL_RECOGNITION=1 to exercise real recognition")
	}
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("set GEMINI_API_KEY to exercise the Gemini backend")
	}
	host := envStr("GEMINI_HOST", "https://generativelanguage.googleapis.com")
	model := envStr("GEMINI_MODEL", "gemini-2.5-flash")
	runBrenner(t, NewGemini(host, model, apiKey), "../../testdata/brenner_tran_RESULTS_GEMINI.txt")
}

// runBrenner recognizes the brenner_tran sheet with rec, reconciles to legal SAN, prints a
// per-ply comparison against expectedBrennerSAN with an accuracy score and timing, writes the
// report to outPath, and asserts the confidence invariants on whatever the model returned.
func runBrenner(t *testing.T, rec Recognizer, outPath string) {
	t.Helper()

	img, err := os.ReadFile(brennerImage)
	if err != nil {
		t.Fatalf("read %s: %v", brennerImage, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
	defer cancel()
	t0 := time.Now()
	res, err := rec.Recognize(ctx, ScoreSheetInput{Image: img, MimeType: "image/jpeg"})
	elapsed := time.Since(t0)
	if err != nil {
		t.Fatalf("%s: recognition failed after %s: %v", rec.Name(), elapsed.Round(time.Millisecond), err)
	}

	moves := Reconcile(string(chesskit.StartingFEN()), res.MoveTokens)
	report := renderBrennerComparison(rec.Name(), res, moves, elapsed)
	t.Log(report)
	if werr := os.WriteFile(outPath, []byte(report), 0o644); werr != nil {
		t.Errorf("write %s: %v", outPath, werr)
	}

	// Invariants (identical to the multi-fixture harness): confidence in [0,1]; clean legal
	// moves are confident; illegal moves carry zero confidence.
	for _, m := range moves {
		if m.Confidence < 0 || m.Confidence > 1 {
			t.Errorf("ply %d: confidence %.2f out of range", m.Ply, m.Confidence)
		}
		if !m.IsLegal && m.Confidence != 0 {
			t.Errorf("ply %d: illegal move must have zero confidence, got %.2f", m.Ply, m.Confidence)
		}
		if m.IsLegal && !m.Corrected && m.Confidence < confThreshold {
			t.Errorf("ply %d: clean legal move should be >= %.2f, got %.2f", m.Ply, confThreshold, m.Confidence)
		}
	}
}

// renderBrennerComparison builds the per-ply table: expected SAN (ground truth) vs the model's
// reconciled SAN, marking matches. Accuracy is reported over the verified prefix.
func renderBrennerComparison(name string, res RecognitionResult, moves []domain.Move, elapsed time.Duration) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\nbrenner_tran.jpg — model=%s (%.1fs)\n", name, elapsed.Seconds())
	fmt.Fprintf(&b, "header: White=%q Black=%q  overall=%.2f\n", res.Header.White, res.Header.Black, res.Confidence)
	fmt.Fprintf(&b, "reference: %d plies verified (game is a draw; moves 49-60 not transcribed)\n", len(expectedBrennerSAN))
	fmt.Fprintf(&b, "  %-4s %-6s %-16s %-8s %-5s %-3s %s\n", "ply", "side", "read -> san", "expected", "conf", "ok", "state")

	n := len(moves)
	if len(expectedBrennerSAN) > n {
		n = len(expectedBrennerSAN)
	}
	var compared, correct int
	for i := 0; i < n; i++ {
		var exp string
		if i < len(expectedBrennerSAN) {
			exp = expectedBrennerSAN[i]
		}
		ply := i + 1
		side := "white"
		if i%2 == 1 {
			side = "black"
		}
		readToSan, conf, st := "—", 0.0, "—"
		if i < len(moves) {
			m := moves[i]
			ply, side = m.Ply, m.Side
			readToSan = m.RecognizedText
			if m.SAN != m.RecognizedText {
				readToSan = fmt.Sprintf("%s -> %s", m.RecognizedText, m.SAN)
			}
			conf, st = m.Confidence, moveState(m)
		}
		mark := ""
		if exp != "" && i < len(moves) {
			compared++
			if moves[i].SAN == exp {
				correct++
				mark = "✓"
			} else {
				mark = "✗"
			}
		}
		fmt.Fprintf(&b, "  %-4d %-6s %-16s %-8s %.2f  %-3s %s\n",
			ply, side, trunc(readToSan, 16), exp, conf, mark, st)
	}

	acc := 0.0
	if compared > 0 {
		acc = 100 * float64(correct) / float64(compared)
	}
	fmt.Fprintf(&b, "  accuracy: %d/%d plies match reference (%.0f%% over %d compared) · model emitted %d plies\n",
		correct, len(expectedBrennerSAN), acc, compared, len(moves))
	return b.String()
}
