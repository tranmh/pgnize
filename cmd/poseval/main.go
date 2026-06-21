// Command poseval is a quality-evaluation harness for the "board photo → FEN"
// feature. It runs every image in a labeled corpus through one or more position
// recognizers (Ollama and/or Gemini), assembles a FEN, and scores the resulting
// board field against the hand-labeled ground truth in manifest.json.
//
// It is a developer tool, not part of CI: it needs a running Ollama server and/or
// a GEMINI_API_KEY. Gemini is skipped (with a printed notice) when no key is set.
//
//	go run ./cmd/poseval -backend=ollama
//	go run ./cmd/poseval -backend=both          # needs GEMINI_API_KEY for the gemini column
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/tranmh/pgnize/internal/recognition"
)

// manifestEntry is one labeled corpus image. The full FEN may be given; only the
// board field (the part before the first space) is compared.
type manifestEntry struct {
	File        string `json:"file"`
	Category    string `json:"category"`
	FEN         string `json:"fen"`
	Orientation string `json:"orientation"`
}

// result holds one (image × backend) evaluation outcome.
type result struct {
	entry    manifestEntry
	backend  string
	gotBoard string  // assembled board field ("" on error)
	exact    bool    // got board field == truth board field (clean assembly only)
	accuracy float64 // fraction of the 64 squares that match
	err      error   // hard error: recognition failed or no usable grid
	repaired bool    // AssembleFEN rejected the grid; scored best-effort from the raw grid
	rawGrid  string  // the recognizer's raw 8-row grid, "/"-joined (for inspection)
	elapsed  time.Duration
}

func main() {
	dir := flag.String("dir", "testdata/positions", "corpus directory containing manifest.json")
	backend := flag.String("backend", "both", "recognizer backend: ollama|gemini|both")
	ollamaHost := flag.String("ollama-host", "http://localhost:11434", "Ollama server host")
	ollamaModel := flag.String("ollama-model", "minicpm-v:latest", "Ollama vision model")
	geminiModel := flag.String("gemini-model", "gemini-2.5-flash", "Gemini model")
	flag.Parse()

	entries, err := loadManifest(*dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "poseval: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Loaded %d corpus images from %s\n", len(entries), *dir)

	backends := selectBackends(*backend, *ollamaHost, *ollamaModel, *geminiModel)
	if len(backends) == 0 {
		fmt.Fprintln(os.Stderr, "poseval: no backends available to run; nothing to do")
		os.Exit(1)
	}

	// Run every backend across every image. A single failure is recorded and the
	// run continues. results[backendName] is keyed by backend.
	results := make(map[string][]result, len(backends))
	for _, b := range backends {
		fmt.Printf("\n=== Running backend %q over %d images ===\n", b.name, len(entries))
		for i, e := range entries {
			r := evalOne(b, *dir, e)
			results[b.name] = append(results[b.name], r)
			status := "ok"
			switch {
			case r.err != nil:
				status = "ERR: " + r.err.Error()
			case r.exact:
				status = "exact"
			}
			fmt.Printf("  [%2d/%2d] %-14s acc=%5.1f%% %-6s (%s)\n",
				i+1, len(entries), e.File, r.accuracy*100, statusTag(r), elapsedOrStatus(r, status))
		}
	}

	report := renderReport(entries, backends, results)
	fmt.Print("\n" + report)

	outPath := filepath.Join(*dir, "..", "..", "poseval-report.md")
	if abs, err := filepath.Abs(outPath); err == nil {
		outPath = abs
	}
	if err := os.WriteFile(outPath, []byte(report), 0o644); err != nil {
		// Fall back to the corpus dir if the repo root is not writable.
		alt := filepath.Join(*dir, "poseval-report.md")
		if err2 := os.WriteFile(alt, []byte(report), 0o644); err2 != nil {
			fmt.Fprintf(os.Stderr, "poseval: could not write report: %v / %v\n", err, err2)
		} else {
			fmt.Printf("\nReport written to %s\n", alt)
		}
	} else {
		fmt.Printf("\nReport written to %s\n", outPath)
	}
}

// backend pairs a recognizer with the display name used in the report columns.
type backend struct {
	name string
	rec  recognition.Recognizer
}

// selectBackends builds the requested recognizers. Ollama is always attempted (a
// dead server surfaces as per-image errors). Gemini is only added when
// GEMINI_API_KEY is set; otherwise it is skipped with a printed notice.
func selectBackends(which, ollamaHost, ollamaModel, geminiModel string) []backend {
	var out []backend
	wantOllama := which == "ollama" || which == "both"
	wantGemini := which == "gemini" || which == "both"

	if wantOllama {
		out = append(out, backend{name: "ollama", rec: recognition.NewOllama(ollamaHost, ollamaModel)})
	}
	if wantGemini {
		key := os.Getenv("GEMINI_API_KEY")
		if key == "" {
			fmt.Println("NOTICE: GEMINI_API_KEY is not set — skipping the Gemini backend. " +
				"Set it and re-run (e.g. `make poseval`) to add the Gemini column.")
		} else {
			host := os.Getenv("GEMINI_HOST")
			if host == "" {
				host = "https://generativelanguage.googleapis.com"
			}
			out = append(out, backend{name: "gemini", rec: recognition.NewGemini(host, geminiModel, key)})
		}
	}
	return out
}

// evalOne runs a single image through a backend and scores it. It never panics or
// aborts: read/recognition/assembly failures become a 0-accuracy result with the
// error attached so the caller can keep going.
func evalOne(b backend, dir string, e manifestEntry) result {
	r := result{entry: e, backend: b.name}
	truthBoard := boardField(e.FEN)

	data, err := os.ReadFile(filepath.Join(dir, e.File))
	if err != nil {
		r.err = fmt.Errorf("read image: %w", err)
		return r
	}

	// A generous ceiling on top of the recognizer's own HTTP timeout, so a single
	// stuck call cannot wedge the whole run.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	start := time.Now()
	res, err := b.rec.RecognizePosition(ctx, recognition.PositionInput{
		Image:    data,
		MimeType: mimeForFile(e.File),
	})
	r.elapsed = time.Since(start)
	r.rawGrid = strings.Join(res.Grid, "/")
	if err != nil {
		r.err = fmt.Errorf("recognize: %w", err)
		return r
	}

	fen, err := recognition.AssembleFEN(res)
	if err != nil {
		// AssembleFEN is strict (production feeds the editor a clean FEN or falls
		// back to the empty board). For the eval we still want to know how many
		// squares the model got, so score best-effort from the raw grid.
		cells := bestEffortCells(res.Grid, e.Orientation)
		if cells == nil {
			r.err = fmt.Errorf("assemble: %w (no usable grid)", err)
			return r
		}
		r.repaired = true
		r.gotBoard = cellsToBoardField(cells)
		r.accuracy = squareAccuracyCells(expandBoard(truthBoard), cells)
		return r
	}
	r.gotBoard = boardField(fen)
	r.accuracy = squareAccuracy(truthBoard, r.gotBoard)
	r.exact = r.gotBoard == truthBoard
	return r
}

// bestEffortCells builds a 64-cell board (rank 8 a→h … rank 1) from a raw, possibly
// malformed recognizer grid: it takes up to 8 rows, normalizes each to exactly 8
// cells (FEN digits expanded, unknown glyphs → empty, over-long rows truncated,
// short rows padded), and applies the 180° rotation for a black_bottom image so it
// aligns with the white-relative ground truth. Returns nil when the grid is empty.
func bestEffortCells(grid []string, orientation string) []byte {
	if len(grid) == 0 {
		return nil
	}
	rows := make([]string, 8)
	for i := 0; i < 8; i++ {
		if i < len(grid) {
			rows[i] = normRow(grid[i])
		} else {
			rows[i] = "........"
		}
	}
	if orientation == "black_bottom" {
		for i, j := 0, 7; i < j; i, j = i+1, j-1 {
			rows[i], rows[j] = rows[j], rows[i]
		}
		for i := range rows {
			rows[i] = reverseString(rows[i])
		}
	}
	cells := make([]byte, 0, 64)
	for _, row := range rows {
		cells = append(cells, []byte(row)...)
	}
	return cells
}

// normRow turns one raw grid row into exactly 8 cells (FEN piece letters or '.').
func normRow(row string) string {
	var cells []byte
	for i := 0; i < len(row) && len(cells) < 8; i++ {
		c := row[i]
		switch {
		case c >= '1' && c <= '8':
			for n := 0; n < int(c-'0') && len(cells) < 8; n++ {
				cells = append(cells, '.')
			}
		case strings.IndexByte("KQRBNPkqrbnp", c) >= 0:
			cells = append(cells, c)
		case c == '.' || c == ' ' || c == '_' || c == '-':
			cells = append(cells, '.')
			// any other glyph (coordinate letters, punctuation) is skipped
		}
	}
	for len(cells) < 8 {
		cells = append(cells, '.')
	}
	return string(cells)
}

func reverseString(s string) string {
	b := []byte(s)
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	return string(b)
}

// cellsToBoardField run-length-encodes 64 cells back into a FEN board field.
func cellsToBoardField(cells []byte) string {
	var b strings.Builder
	for rank := 0; rank < 8; rank++ {
		empty := 0
		for file := 0; file < 8; file++ {
			c := cells[rank*8+file]
			if c == '.' {
				empty++
				continue
			}
			if empty > 0 {
				b.WriteByte(byte('0' + empty))
				empty = 0
			}
			b.WriteByte(c)
		}
		if empty > 0 {
			b.WriteByte(byte('0' + empty))
		}
		if rank < 7 {
			b.WriteByte('/')
		}
	}
	return b.String()
}

// squareAccuracyCells scores pre-expanded truth cells against a 64-cell candidate.
func squareAccuracyCells(truth []byte, got []byte) float64 {
	match := 0
	for i := 0; i < 64; i++ {
		if i < len(truth) && i < len(got) && truth[i] == got[i] {
			match++
		}
	}
	return float64(match) / 64.0
}

// boardField returns the placement field (before the first space) of a FEN string.
func boardField(fen string) string {
	if i := strings.IndexByte(fen, ' '); i >= 0 {
		return fen[:i]
	}
	return fen
}

// expandBoard turns a FEN board field into 64 cells (rank 8 a→h, then rank 7, …,
// rank 1). Empty squares are '.'. A malformed field yields a short/long slice,
// which squareAccuracy handles by comparing position-by-position.
func expandBoard(board string) []byte {
	cells := make([]byte, 0, 64)
	for _, rank := range strings.Split(board, "/") {
		for i := 0; i < len(rank); i++ {
			c := rank[i]
			if c >= '1' && c <= '8' {
				for n := 0; n < int(c-'0'); n++ {
					cells = append(cells, '.')
				}
				continue
			}
			cells = append(cells, c)
		}
	}
	return cells
}

// squareAccuracy is the fraction of the 64 squares that match between the truth
// and candidate board fields. Both are expanded to cells. A valid FEN board field
// always expands to exactly 64 cells; any position past the end of either slice is
// treated as a mismatch, so a short/degenerate output cannot inflate the score.
func squareAccuracy(truth, got string) float64 {
	t := expandBoard(truth)
	g := expandBoard(got)
	match := 0
	for i := 0; i < 64; i++ {
		if i < len(t) && i < len(g) && t[i] == g[i] {
			match++
		}
	}
	return float64(match) / 64.0
}

func mimeForFile(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	default:
		return "image/jpeg"
	}
}

func loadManifest(dir string) ([]manifestEntry, error) {
	path := filepath.Join(dir, "manifest.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	var entries []manifestEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("manifest %s is empty", path)
	}
	return entries, nil
}

// ---- aggregation + reporting -------------------------------------------------

// agg is the aggregate for one (backend × category) bucket.
type agg struct {
	count    int
	exact    int
	repaired int
	errs     int
	accSum   float64
}

func (a agg) meanAcc() float64 {
	if a.count == 0 {
		return 0
	}
	return a.accSum / float64(a.count)
}

func (a agg) exactRate() float64 {
	if a.count == 0 {
		return 0
	}
	return float64(a.exact) / float64(a.count)
}

// aggregate buckets results by category (plus an "overall" bucket).
func aggregate(results []result) map[string]agg {
	out := map[string]agg{}
	add := func(key string, r result) {
		a := out[key]
		a.count++
		a.accSum += r.accuracy
		if r.exact {
			a.exact++
		}
		if r.repaired {
			a.repaired++
		}
		if r.err != nil {
			a.errs++
		}
		out[key] = a
	}
	for _, r := range results {
		add(r.entry.Category, r)
		add("overall", r)
	}
	return out
}

func categories(entries []manifestEntry) []string {
	seen := map[string]bool{}
	var cats []string
	for _, e := range entries {
		if !seen[e.Category] {
			seen[e.Category] = true
			cats = append(cats, e.Category)
		}
	}
	sort.Strings(cats)
	cats = append(cats, "overall")
	return cats
}

// renderReport builds the Markdown report: a side-by-side aggregate table (one
// backend column group each) followed by a per-image breakdown.
func renderReport(entries []manifestEntry, backends []backend, results map[string][]result) string {
	var b strings.Builder
	b.WriteString("# poseval — board photo → FEN quality report\n\n")
	b.WriteString(fmt.Sprintf("_Generated %s • %d images_\n\n", time.Now().Format("2006-01-02 15:04"), len(entries)))

	names := make([]string, 0, len(backends))
	for _, bk := range backends {
		names = append(names, bk.name)
	}
	b.WriteString("Backends: " + strings.Join(names, ", ") + "\n\n")

	aggByBackend := map[string]map[string]agg{}
	for _, bk := range backends {
		aggByBackend[bk.name] = aggregate(results[bk.name])
	}

	// Aggregate table: rows are categories, columns are per-backend metrics.
	b.WriteString("## Aggregate (mean per-square accuracy · exact-match rate · n · repaired · errors)\n\n")
	b.WriteString("_sq-acc = mean fraction of the 64 squares correct. exact = clean FEN equals truth. " +
		"repaired = model grid was malformed (rejected by AssembleFEN) and scored best-effort. " +
		"err = no usable grid at all._\n\n")
	b.WriteString("| Category |")
	for _, n := range names {
		b.WriteString(fmt.Sprintf(" %s sq-acc | %s exact | %s n | %s repaired | %s err |", n, n, n, n, n))
	}
	b.WriteString("\n|---|")
	for range names {
		b.WriteString("---|---|---|---|---|")
	}
	b.WriteString("\n")
	for _, cat := range categories(entries) {
		b.WriteString(fmt.Sprintf("| %s |", cat))
		for _, n := range names {
			a := aggByBackend[n][cat]
			b.WriteString(fmt.Sprintf(" %.1f%% | %.0f%% (%d/%d) | %d | %d | %d |",
				a.meanAcc()*100, a.exactRate()*100, a.exact, a.count, a.count, a.repaired, a.errs))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Per-image breakdown.
	b.WriteString("## Per-image breakdown\n\n")
	b.WriteString("| Image | Category |")
	for _, n := range names {
		b.WriteString(fmt.Sprintf(" %s sq-acc | %s result |", n, n))
	}
	b.WriteString("\n|---|---|")
	for range names {
		b.WriteString("---|---|")
	}
	b.WriteString("\n")
	for i, e := range entries {
		b.WriteString(fmt.Sprintf("| %s | %s |", e.File, e.Category))
		for _, n := range names {
			r := results[n][i]
			b.WriteString(fmt.Sprintf(" %.1f%% | %s |", r.accuracy*100, cellStatus(r)))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Ground truth vs. observed, for the curious / for debugging mislabels.
	b.WriteString("## Ground truth vs. recognized board field\n\n")
	for _, bk := range backends {
		b.WriteString("### " + bk.name + "\n\n")
		b.WriteString("| Image | Truth | Got (board field) | Raw grid (repaired/error) |\n|---|---|---|---|\n")
		for i, e := range entries {
			r := results[bk.name][i]
			got := r.gotBoard
			if r.err != nil {
				got = "ERROR: " + r.err.Error()
			}
			raw := ""
			if r.repaired || r.err != nil {
				raw = "`" + r.rawGrid + "`"
			}
			b.WriteString(fmt.Sprintf("| %s | `%s` | `%s` | %s |\n", e.File, boardField(e.FEN), got, raw))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func statusTag(r result) string {
	switch {
	case r.err != nil:
		return "err"
	case r.exact:
		return "EXACT"
	case r.repaired:
		return "rep"
	default:
		return ""
	}
}

func cellStatus(r result) string {
	switch {
	case r.err != nil:
		return "error"
	case r.exact:
		return "exact"
	case r.repaired:
		return "repaired"
	default:
		return "partial"
	}
}

func elapsedOrStatus(r result, status string) string {
	if r.elapsed > 0 {
		return fmt.Sprintf("%s, %s", r.elapsed.Round(time.Second), status)
	}
	return status
}
