//go:build ollama

// Opt-in harness that runs the REAL Ollama vision recognizer against real score-sheet
// images. Not part of CI. Run with:
//
//	docker compose --profile vlm up -d ollama && docker exec ... ollama pull minicpm-v
//	PGNIZE_SHEETS_DIR=./testdata/real-sheets OLLAMA_HOST=http://localhost:11434 \
//	  RECOGNIZER_MODEL=minicpm-v go test -tags=ollama -run TestOllamaRealSheets -v -timeout 60m ./internal/recognition/
package recognition

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/tranmh/chesskit"
)

func TestOllamaRealSheets(t *testing.T) {
	dir := os.Getenv("PGNIZE_SHEETS_DIR")
	if dir == "" {
		t.Skip("PGNIZE_SHEETS_DIR not set")
	}
	host := envOr("OLLAMA_HOST", "http://localhost:11434")
	model := envOr("RECOGNIZER_MODEL", "minicpm-v")
	rec := NewOllama(host, model)

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	var images []string
	for _, e := range entries {
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".webp" {
			images = append(images, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(images)
	if len(images) == 0 {
		t.Skipf("no images in %s", dir)
	}
	if lim := envOr("PGNIZE_SHEETS_LIMIT", ""); lim != "" {
		if n, err := strconv.Atoi(lim); err == nil && n < len(images) {
			images = images[:n]
		}
	}
	t.Logf("recognizer=%s model=%s images=%d", rec.Name(), model, len(images))

	var totalDur time.Duration
	for _, img := range images {
		img := img
		name := filepath.Base(img)
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(img)
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
			defer cancel()
			start := time.Now()
			res, rerr := rec.Recognize(ctx, ScoreSheetInput{Image: data, MimeType: "image/jpeg"})
			dur := time.Since(start)
			totalDur += dur

			t.Logf("size=%d bytes latency=%s", len(data), dur.Round(time.Millisecond))
			if rerr != nil {
				t.Logf("RECOGNIZE ERROR (finding, not failure): %v", rerr)
				return
			}
			t.Logf("header: White=%q Black=%q Event=%q Date=%q Result=%q",
				res.Header.White, res.Header.Black, res.Header.Event, res.Header.Date, res.Header.Result)
			t.Logf("raw model JSON: %s", truncate(res.RawJSON, 1500))
			t.Logf("token count=%d", len(res.MoveTokens))

			moves := Reconcile(string(chesskit.StartingFEN()), res.MoveTokens)
			legal, firstIllegal := 0, -1
			var sb strings.Builder
			for i, m := range moves {
				if m.IsLegal {
					legal++
				} else if firstIllegal < 0 {
					firstIllegal = i
				}
				if i < 50 {
					sb.WriteString(m.RecognizedText + "→" + m.SAN)
					if !m.IsLegal {
						sb.WriteString("(X)")
					}
					sb.WriteString(" ")
				}
			}
			t.Logf("reconciled: %d/%d legal, firstIllegal=%d", legal, len(moves), firstIllegal)
			t.Logf("moves: %s", sb.String())
		})
	}
	if n := len(images); n > 0 {
		t.Logf("######## total latency=%s, avg=%s/image ########",
			totalDur.Round(time.Millisecond), (totalDur / time.Duration(n)).Round(time.Millisecond))
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…(truncated)"
}
