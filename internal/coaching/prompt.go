package coaching

import (
	"fmt"
	"strings"
)

// Prompt assembly is deterministic (byte-stable for a given input) so it is easy to test
// and reason about. German-notation/teaching specifics live here, never in chesskit.

// formatEval renders a White-POV evaluation: "+1.30", "-0.50", "0.00", "#3", "-#2", or
// "?" when unknown.
func formatEval(e Eval) string {
	if e.Mate != nil {
		m := *e.Mate
		switch {
		case m == 0:
			return "#"
		case m > 0:
			return fmt.Sprintf("#%d", m)
		default:
			return fmt.Sprintf("-#%d", -m)
		}
	}
	if e.Cp != nil {
		p := float64(*e.Cp) / 100.0
		if p == 0 {
			return "0.00"
		}
		return fmt.Sprintf("%+.2f", p)
	}
	return "?"
}

var qualityWords = map[string]map[string]string{
	"de": {
		"blunder":    "ein grober Fehler",
		"mistake":    "ein Fehler",
		"inaccuracy": "eine Ungenauigkeit",
	},
	"en": {
		"blunder":    "a blunder",
		"mistake":    "a mistake",
		"inaccuracy": "an inaccuracy",
	},
}

func qualityWord(quality, lang string) string {
	if m, ok := qualityWords[lang]; ok {
		if w, ok := m[quality]; ok {
			return w
		}
	}
	return quality
}

func sideWord(side, lang string) string {
	if lang == "en" {
		if side == "black" {
			return "Black"
		}
		return "White"
	}
	if side == "black" {
		return "Schwarz"
	}
	return "Weiß"
}

// systemInstruction is the role + language directive. German-first: any lang other than
// "en" yields the German instruction.
func systemInstruction(lang string) string {
	if lang == "en" {
		return "You are an experienced, encouraging chess coach. Use the engine evaluation as ground truth, " +
			"but translate the cold numbers into ideas and plans the player can learn from. " +
			"Answer in English in 2–4 short sentences. Refer to moves in standard algebraic notation (SAN)."
	}
	return "Du bist ein erfahrener, ermutigender Schachtrainer. Nutze die Engine-Bewertung als Wahrheit, " +
		"aber übersetze die kalten Zahlen in Ideen und Pläne, aus denen der Spieler lernen kann. " +
		"Antworte auf Deutsch in 2–4 kurzen Sätzen. Bezeichne Züge in der Standard-Notation (SAN)."
}

// buildMovePrompt assembles the per-move coaching prompt.
func buildMovePrompt(in MoveInput) string {
	lang := normLang(in.Lang)
	var b strings.Builder
	b.WriteString(systemInstruction(lang))
	b.WriteString("\n\n")

	if lang == "en" {
		fmt.Fprintf(&b, "Position (FEN): %s\n", in.FEN)
		fmt.Fprintf(&b, "Side to move: %s\n", sideWord(in.Side, lang))
		fmt.Fprintf(&b, "Move played: %s\n", in.PlayedSAN)
		fmt.Fprintf(&b, "Evaluation before the move: %s\n", formatEval(in.EvalBefore))
		fmt.Fprintf(&b, "Evaluation after the move: %s\n", formatEval(in.EvalAfter))
		if in.BestSAN != "" {
			fmt.Fprintf(&b, "Engine's best move: %s\n", in.BestSAN)
		}
		if len(in.BestLine) > 0 {
			fmt.Fprintf(&b, "Engine main line: %s\n", strings.Join(in.BestLine, " "))
		}
		if in.Quality != "" {
			fmt.Fprintf(&b, "This move is classified as %s.\n", qualityWord(in.Quality, lang))
		}
		if in.BestSAN != "" {
			b.WriteString("\nExplain why the played move is good or bad, and what the better idea is.")
		} else {
			b.WriteString("\nExplain why the played move is good or bad and what to keep in mind here.")
		}
	} else {
		fmt.Fprintf(&b, "Stellung (FEN): %s\n", in.FEN)
		fmt.Fprintf(&b, "Am Zug: %s\n", sideWord(in.Side, lang))
		fmt.Fprintf(&b, "Gespielter Zug: %s\n", in.PlayedSAN)
		fmt.Fprintf(&b, "Bewertung vor dem Zug: %s\n", formatEval(in.EvalBefore))
		fmt.Fprintf(&b, "Bewertung nach dem Zug: %s\n", formatEval(in.EvalAfter))
		if in.BestSAN != "" {
			fmt.Fprintf(&b, "Bester Zug der Engine: %s\n", in.BestSAN)
		}
		if len(in.BestLine) > 0 {
			fmt.Fprintf(&b, "Hauptvariante der Engine: %s\n", strings.Join(in.BestLine, " "))
		}
		if in.Quality != "" {
			fmt.Fprintf(&b, "Dieser Zug gilt als %s.\n", qualityWord(in.Quality, lang))
		}
		if in.BestSAN != "" {
			b.WriteString("\nErkläre, warum der gespielte Zug gut oder schlecht ist und was die bessere Idee wäre.")
		} else {
			b.WriteString("\nErkläre, warum der gespielte Zug gut oder schlecht ist und worauf es hier ankommt.")
		}
	}
	return b.String()
}

// buildGamePrompt assembles the whole-game coaching prompt.
func buildGamePrompt(in GameInput) string {
	lang := normLang(in.Lang)
	var b strings.Builder
	b.WriteString(systemInstruction(lang))
	b.WriteString("\n\n")

	if lang == "en" {
		fmt.Fprintf(&b, "Start position (FEN): %s\n", in.StartFEN)
		if in.Header.White != "" || in.Header.Black != "" {
			fmt.Fprintf(&b, "Players: %s vs %s\n", in.Header.White, in.Header.Black)
		}
		if in.Header.Result != "" {
			fmt.Fprintf(&b, "Result: %s\n", in.Header.Result)
		}
		b.WriteString("Moves (with White-POV eval after each, and quality flags):\n")
	} else {
		fmt.Fprintf(&b, "Startstellung (FEN): %s\n", in.StartFEN)
		if in.Header.White != "" || in.Header.Black != "" {
			fmt.Fprintf(&b, "Spieler: %s gegen %s\n", in.Header.White, in.Header.Black)
		}
		if in.Header.Result != "" {
			fmt.Fprintf(&b, "Ergebnis: %s\n", in.Header.Result)
		}
		b.WriteString("Züge (mit Bewertung aus Weiß-Sicht nach jedem Zug und Qualitätsmarkierung):\n")
	}

	for _, m := range in.Moves {
		num := (m.Ply + 1) / 2
		marker := ""
		if m.Quality != "" {
			marker = " [" + qualityWord(m.Quality, lang) + "]"
		}
		fmt.Fprintf(&b, "%d%s %s = %s%s\n", num, dotForSide(m.Side), m.SAN, formatEval(m.EvalAfter), marker)
	}

	if lang == "en" {
		b.WriteString("\nGive a short coaching summary: the opening, the critical turning points, and one or two things to work on.")
	} else {
		b.WriteString("\nGib eine kurze Trainer-Zusammenfassung: die Eröffnung, die kritischen Wendepunkte und ein bis zwei Dinge zum Üben.")
	}
	return b.String()
}

func dotForSide(side string) string {
	if side == "black" {
		return "..."
	}
	return "."
}
