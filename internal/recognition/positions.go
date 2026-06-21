package recognition

import (
	"fmt"
	"strings"

	"github.com/tranmh/chesskit"
)

// modelPosition is the shared JSON shape both the Ollama and Gemini position recognizers
// decode the model's answer into.
type modelPosition struct {
	Grid        []string `json:"grid"`
	SideToMove  string   `json:"sideToMove"`
	Orientation string   `json:"orientation"`
}

// fenPieces is the set of valid FEN piece letters; every other grid character is treated
// as an empty square.
const fenPieces = "KQRBNPkqrbnp"

// AssembleFEN turns a model-read grid into a FEN for the editable review board. It repairs
// the grid best-effort (taking the first 8 rows, padding missing ones; per row: expanding
// run-length digits, stripping spaces, mapping unknown glyphs to empty, truncating long and
// padding short rows to 8 cells), flips a black-bottom orientation back to the canonical
// white-bottom view, and run-length-encodes the board field.
//
// Return contract:
//   - legal position → (normalized FEN, nil)
//   - readable but chess-illegal position (e.g. the model misread a king or a back-rank
//     pawn) → (best-effort FEN, error). The FEN is still usable: we return the recognized
//     board so the editor shows what was read instead of discarding every correct square.
//     Falling back to the starting position here is exactly the bug this avoids.
//   - nothing usable read (empty grid) → ("", error)
func AssembleFEN(res PositionResult) (string, error) {
	rows := repairGrid(res.Grid)
	if rows == nil {
		return "", fmt.Errorf("empty grid: no rows to assemble")
	}

	// Recover the canonical white-bottom view. The model's self-reported orientation is
	// unreliable (it mislabels white_bottom boards as black_bottom and vice versa), so we
	// ignore it and infer orientation from where the armies sit. In practice this model
	// also emits files in the correct a→h order even when it inverts the ranks, so the
	// correction is a vertical rank flip only — mirroring files (a full 180°) would corrupt
	// the already-correct file order. Measured over the eval corpus this beats both
	// trusting the flag and the full-180° flip.
	if whiteOnTop(rows) {
		for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
			rows[i], rows[j] = rows[j], rows[i]
		}
	}

	fields := make([]string, 8)
	for i, row := range rows {
		fields[i] = encodeRank(row)
	}
	boardField := strings.Join(fields, "/")

	stm := "w"
	if res.SideToMove == SideBlack {
		stm = "b"
	}
	assembled := boardField + " " + stm + " - - 0 1"

	norm, err := chesskit.NormalizeFEN(chesskit.FEN(assembled))
	if err != nil {
		// Readable grid, illegal position: keep the recognized board for the editor.
		return assembled, fmt.Errorf("recognized position is not legal: %w", err)
	}
	return string(norm), nil
}

// repairGrid coerces a raw recognizer grid into exactly 8 rows of 8 cells: it keeps the
// first 8 rows (padding any missing trailing rows with empties) and normalizes each row.
// It returns nil only when the grid carries no rows at all.
func repairGrid(grid []string) []string {
	if len(grid) == 0 {
		return nil
	}
	rows := make([]string, 8)
	for i := 0; i < 8; i++ {
		if i < len(grid) {
			rows[i] = normalizeRow(grid[i])
		} else {
			rows[i] = "........"
		}
	}
	return rows
}

// normalizeRow coerces one raw grid row into exactly 8 cells. It expands FEN run-length
// digits, strips spaces, maps every unknown glyph to an empty square, truncates an
// over-long row and pads a short one. It never fails: the editable board is the correction
// path, so a best-effort row always beats discarding the whole read.
func normalizeRow(raw string) string {
	cells := make([]byte, 0, 8)
	for _, r := range raw {
		if len(cells) >= 8 {
			break
		}
		switch {
		case r == ' ':
			continue
		case r >= '1' && r <= '8':
			for n := 0; n < int(r-'0') && len(cells) < 8; n++ {
				cells = append(cells, '.')
			}
		case strings.ContainsRune(fenPieces, r):
			cells = append(cells, byte(r))
		default:
			cells = append(cells, '.') // unknown glyph → empty square
		}
	}
	for len(cells) < 8 {
		cells = append(cells, '.')
	}
	return string(cells)
}

// encodeRank collapses consecutive empties ('.') in an 8-char row into FEN count digits.
func encodeRank(row string) string {
	var b strings.Builder
	empties := 0
	for _, r := range row {
		if r == '.' {
			empties++
			continue
		}
		if empties > 0 {
			b.WriteString(fmt.Sprintf("%d", empties))
			empties = 0
		}
		b.WriteRune(r)
	}
	if empties > 0 {
		b.WriteString(fmt.Sprintf("%d", empties))
	}
	return b.String()
}

// whiteOnTop reports whether the grid is vertically inverted — White's army sitting in
// the top half and Black's in the bottom half — which means it must be flipped to recover
// the canonical white-bottom view. It scores the four quadrant masses; a positive score
// means White is (correctly) on the bottom, negative means it is on top. A tie (sparse or
// symmetric material with no vertical signal) defaults to no flip, since white_bottom is by
// far the most common real orientation and the editable board is the final correction path.
func whiteOnTop(rows []string) bool {
	var whiteTop, whiteBottom, blackTop, blackBottom int
	for i, row := range rows {
		top := i < len(rows)/2
		for _, c := range row {
			switch {
			case c >= 'A' && c <= 'Z': // white piece
				if top {
					whiteTop++
				} else {
					whiteBottom++
				}
			case c >= 'a' && c <= 'z': // black piece
				if top {
					blackTop++
				} else {
					blackBottom++
				}
			}
		}
	}
	score := (whiteBottom - whiteTop) + (blackTop - blackBottom)
	return score < 0
}
