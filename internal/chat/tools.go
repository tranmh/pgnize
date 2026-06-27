package chat

import (
	"context"
	"fmt"

	"github.com/tranmh/chesskit"
	"github.com/tranmh/pgnize/internal/engine"
)

// defaultMultiPV is how many candidate lines find_best_line returns when unspecified.
const defaultMultiPV = 3

// toolDeclarations are the function declarations advertised to the model. Gemini's Schema
// uses upper-case type names (OBJECT/STRING/INTEGER), matching recognition.geminiResponseSchema.
func toolDeclarations(lang string) []fnDecl {
	de := lang != "en"
	desc := func(d, e string) string {
		if de {
			return d
		}
		return e
	}
	fenProp := map[string]any{"type": "STRING", "description": desc(
		"Die FEN-Stellung. Standard: die aktuelle Stellung aus dem Gespräch.",
		"The FEN position. Defaults to the current position from the conversation.")}
	return []fnDecl{
		{
			Name: "analyze_position",
			Description: desc(
				"Bewerte eine Stellung mit der Engine und gib den besten Zug, die Bewertung und die Hauptvariante zurück.",
				"Evaluate a position with the engine; returns the best move, evaluation, and principal variation."),
			Parameters: map[string]any{
				"type":       "OBJECT",
				"properties": map[string]any{"fen": fenProp},
			},
		},
		{
			Name: "evaluate_move",
			Description: desc(
				"Bewerte einen konkreten Zug gegen den besten Zug der Engine — nutze dies, um zu erklären, warum ein Zug gut oder schlecht ist.",
				"Evaluate a specific move against the engine's best move — use this to explain why a move is good or bad."),
			Parameters: map[string]any{
				"type": "OBJECT",
				"properties": map[string]any{
					"fen": fenProp,
					"move": map[string]any{"type": "STRING", "description": desc(
						"Der zu bewertende Zug in SAN, z. B. Sf3, exd5, O-O.",
						"The move to evaluate in SAN, e.g. Nf3, exd5, O-O.")},
				},
				"required": []string{"move"},
			},
		},
		{
			Name: "find_best_line",
			Description: desc(
				"Gib die besten Kandidatenzüge der Stellung mit Bewertungen zurück.",
				"Return the top candidate moves for the position with evaluations."),
			Parameters: map[string]any{
				"type": "OBJECT",
				"properties": map[string]any{
					"fen":     fenProp,
					"multipv": map[string]any{"type": "INTEGER", "description": desc("Anzahl der Kandidaten (1–5).", "Number of candidates (1-5).")},
				},
			},
		},
		{
			Name: "find_mate",
			Description: desc(
				"Prüfe, ob die Seite am Zug ein erzwungenes Matt hat.",
				"Check whether the side to move has a forced mate."),
			Parameters: map[string]any{
				"type":       "OBJECT",
				"properties": map[string]any{"fen": fenProp},
			},
		},
	}
}

// dispatch validates the model's tool arguments via chesskit and runs the engine. It never
// returns an error: validation/engine failures are reported as {"error": ...} in the result
// map so the model can self-correct on the next turn. fallbackFEN is the conversation's
// current position, used when the model omits/empties the fen argument.
func dispatch(ctx context.Context, eng engine.Engine, name string, args map[string]any, fallbackFEN string) map[string]any {
	fen, err := resolveFEN(args, fallbackFEN)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}

	switch name {
	case "analyze_position":
		a, err := eng.Analyze(ctx, fen, engine.Options{MultiPV: 1})
		if err != nil {
			return map[string]any{"error": "engine unavailable"}
		}
		return lineResult(fen, a.Best())

	case "evaluate_move":
		moveStr, _ := args["move"].(string)
		if moveStr == "" {
			return map[string]any{"error": "missing move"}
		}
		childFEN, verr := chesskit.Validate(chesskit.FEN(fen), chesskit.SAN(moveStr))
		if verr != nil {
			return map[string]any{"error": fmt.Sprintf("illegal move %q: %v", moveStr, verr)}
		}
		best, played, delta, eerr := engine.EvalMove(ctx, eng, fen, string(childFEN), engine.Options{MultiPV: 1})
		if eerr != nil {
			return map[string]any{"error": "engine unavailable"}
		}
		res := map[string]any{
			"move":       canonicalSAN(fen, moveStr),
			"move_eval":  formatEval(played),
			"best_move":  sanOfUCI(fen, best.BestMove),
			"best_eval":  formatEval(best),
			"delta_cp":   delta,
			"pv":         pvToSAN(fen, best.PV, 6),
		}
		addScore(res, "move", played)
		return res

	case "find_best_line":
		n := defaultMultiPV
		if v, ok := args["multipv"]; ok {
			if iv, ok2 := toInt(v); ok2 && iv >= 1 && iv <= 5 {
				n = iv
			}
		}
		a, err := eng.Analyze(ctx, fen, engine.Options{MultiPV: n})
		if err != nil {
			return map[string]any{"error": "engine unavailable"}
		}
		lines := make([]map[string]any, 0, len(a.Lines))
		for _, l := range a.Lines {
			lines = append(lines, lineResult(fen, l))
		}
		return map[string]any{"lines": lines}

	case "find_mate":
		mateIn, pv, err := engine.FindMate(ctx, eng, fen, engine.Options{MultiPV: 1})
		if err != nil {
			return map[string]any{"error": "engine unavailable"}
		}
		if mateIn == nil {
			return map[string]any{"mate": false}
		}
		return map[string]any{"mate": true, "mate_in": *mateIn, "pv": pvToSAN(fen, pv, 8)}

	default:
		return map[string]any{"error": "unknown tool"}
	}
}

// resolveFEN picks the fen argument (or the fallback) and validates it via chesskit.
func resolveFEN(args map[string]any, fallback string) (string, error) {
	fen, _ := args["fen"].(string)
	if fen == "" {
		fen = fallback
	}
	norm, err := chesskit.NormalizeFEN(chesskit.FEN(fen))
	if err != nil {
		return "", fmt.Errorf("illegal fen")
	}
	return string(norm), nil
}

// lineResult renders one engine Line (best move + eval + pv) as a tool-result map.
func lineResult(fen string, l engine.Line) map[string]any {
	res := map[string]any{
		"best_move": sanOfUCI(fen, l.BestMove),
		"eval":      formatEval(l),
		"depth":     l.Depth,
		"pv":        pvToSAN(fen, l.PV, 6),
	}
	addScore(res, "", l)
	return res
}

// addScore attaches the numeric cp/mate (side-to-move POV) under an optional prefix.
func addScore(res map[string]any, prefix string, l engine.Line) {
	key := func(k string) string {
		if prefix == "" {
			return k
		}
		return prefix + "_" + k
	}
	if l.Cp != nil {
		res[key("cp")] = *l.Cp
	}
	if l.Mate != nil {
		res[key("mate_in")] = *l.Mate
	}
}

func formatEval(l engine.Line) string {
	switch {
	case l.Mate != nil:
		if *l.Mate > 0 {
			return fmt.Sprintf("#%d", *l.Mate)
		}
		return fmt.Sprintf("-#%d", -*l.Mate)
	case l.Cp != nil:
		return fmt.Sprintf("%+.2f", float64(*l.Cp)/100)
	default:
		return "?"
	}
}

// sanOfUCI converts a UCI move to SAN for nicer prose; falls back to the UCI on failure.
func sanOfUCI(fen, uci string) string {
	if uci == "" {
		return ""
	}
	san, err := chesskit.UCItoSAN(chesskit.FEN(fen), uci)
	if err != nil {
		return uci
	}
	return string(san)
}

// canonicalSAN normalizes the model's SAN spelling (e.g. casing/castling) via chesskit.
func canonicalSAN(fen, san string) string {
	mv, err := chesskit.ParseSAN(chesskit.FEN(fen), chesskit.SAN(san))
	if err != nil {
		return san
	}
	return string(mv.SAN)
}

// pvToSAN converts a UCI principal variation to SAN, replaying through chesskit. It stops at
// the first move it cannot convert (best-effort) and caps the length.
func pvToSAN(fen string, uciPV []string, max int) []string {
	out := make([]string, 0, len(uciPV))
	cur := chesskit.FEN(fen)
	for i, uci := range uciPV {
		if i >= max {
			break
		}
		san, err := chesskit.UCItoSAN(cur, uci)
		if err != nil {
			break
		}
		out = append(out, string(san))
		next, err := chesskit.Validate(cur, san)
		if err != nil {
			break
		}
		cur = next
	}
	return out
}

func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	case int64:
		return int(n), true
	default:
		return 0, false
	}
}
