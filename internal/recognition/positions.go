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

	// A black-bottom photo is the board seen rotated 180°; reverse rank order and each
	// rank's file order to recover the canonical white-bottom view.
	if res.Orientation == "black_bottom" {
		for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
			rows[i], rows[j] = rows[j], rows[i]
		}
		for i := range rows {
			rows[i] = reverseString(rows[i])
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

func reverseString(s string) string {
	b := []byte(s)
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	return string(b)
}
