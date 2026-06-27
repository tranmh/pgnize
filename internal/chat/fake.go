package chat

import (
	"context"
	"fmt"
	"strings"

	"github.com/tranmh/pgnize/internal/engine"
)

// Fake is a deterministic Chatter for tests/CI. It exercises the REAL tool path — it picks a
// tool from the question, runs it through dispatch() (chesskit validation + engine), and
// returns templated prose embedding the result — but never calls a network LLM. This means a
// unit test covers validate -> engine -> feed-back -> final-text without any model.
type Fake struct {
	Engine engine.Engine
}

var _ Chatter = (*Fake)(nil)

// NewFake builds a Fake chatter over the given engine.
func NewFake(eng engine.Engine) *Fake { return &Fake{Engine: eng} }

func (f *Fake) Name() string { return "fake" }

func (f *Fake) Respond(ctx context.Context, _ []Message, user string, pctx Context) (Reply, error) {
	lang := normLang(pctx.Lang)
	lower := strings.ToLower(user)

	// Choose a tool from the question, mirroring how a real model would route.
	var name string
	args := map[string]any{"fen": pctx.FEN}
	switch {
	case strings.Contains(lower, "mate") || strings.Contains(lower, "matt"):
		name = "find_mate"
	default:
		name = "analyze_position"
	}

	result := dispatch(ctx, f.Engine, name, args, pctx.FEN)
	tc := ToolCall{Name: name, Args: args, Result: result}
	if e, ok := result["error"].(string); ok {
		tc.Err = e
	}

	text := fakeProse(lang, name, result)
	return Reply{Text: text, Model: f.Name(), Calls: []ToolCall{tc}}, nil
}

func fakeProse(lang, tool string, result map[string]any) string {
	de := lang != "en"
	if e, ok := result["error"].(string); ok {
		if de {
			return fmt.Sprintf("Ich konnte die Stellung nicht auswerten (%s).", e)
		}
		return fmt.Sprintf("I could not evaluate the position (%s).", e)
	}
	switch tool {
	case "find_mate":
		if m, _ := result["mate"].(bool); m {
			if de {
				return fmt.Sprintf("Ja — es gibt ein Matt in %v. Variante: %v.", result["mate_in"], result["pv"])
			}
			return fmt.Sprintf("Yes — there is a mate in %v. Line: %v.", result["mate_in"], result["pv"])
		}
		if de {
			return "Nein, ich sehe kein erzwungenes Matt in dieser Stellung."
		}
		return "No, I do not see a forced mate in this position."
	default:
		if de {
			return fmt.Sprintf("Der beste Zug ist %v (Bewertung %v). Empfohlene Fortsetzung: %v.",
				result["best_move"], result["eval"], result["pv"])
		}
		return fmt.Sprintf("The best move is %v (evaluation %v). Suggested continuation: %v.",
			result["best_move"], result["eval"], result["pv"])
	}
}
