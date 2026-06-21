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

// positionSystemPrompt steers the VLM to read a single chess position. FEN assembly and
// orientation correction happen deterministically in positions.go.
const positionSystemPrompt = `You read a single chess position from a photograph of a board OR a 2D diagram.
Report the piece standing on every square as 8 strings, one per rank, ordered rank 8 (top) down to rank 1 (bottom).
Each string is exactly 8 characters left-to-right for files a through h.
White pieces are UPPERCASE: K Q R B N P. Black pieces are lowercase: k q r b n p. An empty square is '.'.
Do NOT use FEN run-length digits; write one character per square.
If White's pieces are at the TOP of the image, report the ranks top-to-bottom exactly as you see them and set orientation="black_bottom"; otherwise set orientation="white_bottom".
Report who is to move as sideToMove="white" or "black"; if you cannot tell, set sideToMove="".
Return ONLY JSON.`

// positionJSONSchema constrains the position output (Ollama "format" field).
var positionJSONSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"grid":        map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		"sideToMove":  map[string]any{"type": "string"},
		"orientation": map[string]any{"type": "string"},
	},
	"required": []string{"grid"},
}

// geminiPositionSchema is the Gemini-style (upper-case type names) position schema.
var geminiPositionSchema = map[string]any{
	"type": "OBJECT",
	"properties": map[string]any{
		"grid":        map[string]any{"type": "ARRAY", "items": map[string]any{"type": "STRING"}},
		"sideToMove":  map[string]any{"type": "STRING"},
		"orientation": map[string]any{"type": "STRING"},
	},
	"required": []string{"grid"},
}

// buildPositionPrompt assembles the position prompt text.
func buildPositionPrompt(_ PositionInput) string {
	var b strings.Builder
	b.WriteString(positionSystemPrompt)
	b.WriteString("\nReturn JSON {grid:[8 strings of 8 chars], sideToMove, orientation}.")
	return b.String()
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
