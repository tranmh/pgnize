package recognition

import (
	"encoding/json"
	"fmt"
	"strings"
)

// systemPrompt steers the VLM. German piece-letter translation is described, but the
// authoritative translation happens deterministically in postprocess.go.
const systemPrompt = `You read photographs of handwritten German chess score sheets (Partieformular).
The sheet lists moves in two columns: White's move then Black's move per numbered row.
German piece letters: K=King, D=Queen, T=Rook, L=Bishop, S=Knight; pawns have no letter.
Castling may be written 0-0 or O-O. Transcribe EXACTLY what is written, one entry per half-move,
in reading order. Also read the header fields (players, event, site, date, round, board, result).
If a cell is illegible, output "?" for that move. Do not invent moves. Return ONLY JSON.`

// jsonSchema constrains the model output (Ollama "format" field).
var jsonSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"header": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"white": map[string]any{"type": "string"}, "black": map[string]any{"type": "string"},
				"event": map[string]any{"type": "string"}, "site": map[string]any{"type": "string"},
				"date": map[string]any{"type": "string"}, "round": map[string]any{"type": "string"},
				"board": map[string]any{"type": "string"}, "result": map[string]any{"type": "string"},
			},
		},
		"moves": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"no":    map[string]any{"type": "integer"},
					"white": map[string]any{"type": "string"},
					"black": map[string]any{"type": "string"},
				},
			},
		},
	},
	"required": []string{"moves"},
}

// buildPrompt assembles the user prompt text, weaving in few-shot examples.
func buildPrompt(in ScoreSheetInput) string {
	var b strings.Builder
	b.WriteString(systemPrompt)
	if in.Hint != nil && len(in.Hint.KnownPlayers) > 0 {
		fmt.Fprintf(&b, "\nLikely player names: %s.", strings.Join(in.Hint.KnownPlayers, ", "))
	}
	for i, ex := range in.FewShot {
		exJSON, _ := json.Marshal(map[string]any{"header": ex.Header, "sans": ex.SANs})
		fmt.Fprintf(&b, "\nExample %d (a previously corrected sheet): %s", i+1, string(exJSON))
	}
	b.WriteString("\nNow transcribe the attached sheet as JSON {header:{...}, moves:[{no,white,black}]}.")
	return b.String()
}
