package recognition

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/tranmh/chesskit"
	"github.com/tranmh/pgnize/internal/domain"
)

// TestRealWorldConfidence runs the committed real handwritten/printed score-sheet fixtures
// through the actual Ollama VLM + Reconcile and prints a per-move confidence table. It is a
// demonstration harness, gated behind RUN_REAL_RECOGNITION=1 (and a reachable Ollama), so it
// never runs in `make test`/CI. It still asserts the confidence invariants on whatever the
// model returns.
//
//	RUN_REAL_RECOGNITION=1 go test ./internal/recognition/ -run RealWorld -v
//
// Overrides: OLLAMA_HOST (default http://localhost:11434), OLLAMA_MODEL (default minicpm-v).
func TestRealWorldConfidence(t *testing.T) {
	if os.Getenv("RUN_REAL_RECOGNITION") == "" {
		t.Skip("set RUN_REAL_RECOGNITION=1 (and run a local Ollama) to exercise real recognition")
	}

	host := envStr("OLLAMA_HOST", "http://localhost:11434")
	model := envStr("OLLAMA_MODEL", "minicpm-v")
	rec := NewOllama(host, model)

	files, err := filepath.Glob("../../testdata/scoresheets/*.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no fixtures found in testdata/scoresheets/*.jpg")
	}
	sort.Strings(files)

	startFEN := string(chesskit.StartingFEN())
	var grand strings.Builder
	fmt.Fprintf(&grand, "\nReal-world recognition confidence — model=%s, %d fixtures\n", rec.Name(), len(files))

	for _, f := range files {
		img, err := os.ReadFile(f)
		if err != nil {
			t.Errorf("%s: read: %v", f, err)
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
		res, err := rec.Recognize(ctx, ScoreSheetInput{Image: img, MimeType: "image/jpeg"})
		cancel()
		if err != nil {
			fmt.Fprintf(&grand, "\n=== %s ===\n  recognition error: %v\n", filepath.Base(f), err)
			t.Logf("%s: recognition error (non-fatal for demo): %v", filepath.Base(f), err)
			continue
		}

		moves := Reconcile(startFEN, res.MoveTokens)
		grand.WriteString(renderTable(filepath.Base(f), res, moves))

		// Invariants: confidence in [0,1]; clean legal moves are confident; illegal are zero.
		for _, m := range moves {
			if m.Confidence < 0 || m.Confidence > 1 {
				t.Errorf("%s ply %d: confidence %.2f out of range", filepath.Base(f), m.Ply, m.Confidence)
			}
			if !m.IsLegal && m.Confidence != 0 {
				t.Errorf("%s ply %d: illegal move must have zero confidence, got %.2f", filepath.Base(f), m.Ply, m.Confidence)
			}
			if m.IsLegal && !m.Corrected && m.Confidence < confThreshold {
				t.Errorf("%s ply %d: clean legal move should be >= %.2f, got %.2f", filepath.Base(f), m.Ply, confThreshold, m.Confidence)
			}
		}
	}

	t.Log(grand.String())
	// Also write the report to a file so the full tables survive regardless of -v truncation.
	_ = os.WriteFile("../../testdata/scoresheets/RESULTS.txt", []byte(grand.String()), 0o644)
}

func renderTable(name string, res RecognitionResult, moves []domain.Move) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\n=== %s ===\n", name)
	fmt.Fprintf(&b, "header: White=%q Black=%q Event=%q  overall=%.2f\n",
		res.Header.White, res.Header.Black, res.Header.Event, res.Confidence)
	fmt.Fprintf(&b, "  %-4s %-6s %-18s %-9s %-5s %s\n", "ply", "side", "read -> san", "legality", "conf", "state")
	var ok, verify, illegal int
	var sum float64
	for _, m := range moves {
		readToSan := m.RecognizedText
		if m.SAN != m.RecognizedText {
			readToSan = fmt.Sprintf("%s -> %s", m.RecognizedText, m.SAN)
		}
		leg := "legal"
		if !m.IsLegal {
			leg = "illegal"
		}
		st := moveState(m)
		switch st {
		case "ok":
			ok++
		case "verify":
			verify++
		default:
			illegal++
		}
		sum += m.Confidence
		fmt.Fprintf(&b, "  %-4d %-6s %-18s %-9s %.2f  %s\n", m.Ply, m.Side, trunc(readToSan, 18), leg, m.Confidence, st)
	}
	mean := 0.0
	if len(moves) > 0 {
		mean = sum / float64(len(moves))
	}
	fmt.Fprintf(&b, "  summary: %d plies · %d ok · %d verify · %d illegal · mean conf %.2f\n",
		len(moves), ok, verify, illegal, mean)
	return b.String()
}

// moveState mirrors the frontend reviewState: a legal move below the threshold is "verify".
func moveState(m domain.Move) string {
	if !m.IsLegal {
		return "illegal"
	}
	if m.Confidence < confThreshold {
		return "verify"
	}
	return "ok"
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
