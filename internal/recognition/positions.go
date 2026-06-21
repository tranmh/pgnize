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

// AssembleFEN turns a model-read grid into a validated FEN. It repairs the grid (expanding
// any run-length digits, stripping spaces, mapping unknown characters to empty), flips a
// black-bottom orientation back to the canonical white-bottom view, builds the board field
// with run-length encoding, and validates the result via chesskit.
func AssembleFEN(res PositionResult) (string, error) {
	if len(res.Grid) != 8 {
		return "", fmt.Errorf("grid must have 8 rows, got %d", len(res.Grid))
	}
	rows := make([]string, 8)
	for i, raw := range res.Grid {
		row, err := normalizeRow(raw)
		if err != nil {
			return "", fmt.Errorf("row %d: %w", i, err)
		}
		rows[i] = row
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
		return "", err
	}
	return string(norm), nil
}

// normalizeRow expands FEN run-length digits, strips spaces, maps unknown characters to
// empty squares, and returns an 8-character row (or an error if it cannot be made length 8).
func normalizeRow(raw string) (string, error) {
	var b strings.Builder
	for _, r := range raw {
		switch {
		case r == ' ':
			continue
		case r >= '1' && r <= '8':
			for n := 0; n < int(r-'0'); n++ {
				b.WriteByte('.')
			}
		case strings.ContainsRune(fenPieces, r):
			b.WriteRune(r)
		default:
			b.WriteByte('.') // unknown glyph → empty square
		}
	}
	row := b.String()
	if len(row) != 8 {
		return "", fmt.Errorf("row has length %d, want 8", len(row))
	}
	return row, nil
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
