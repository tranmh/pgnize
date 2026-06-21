package coaching

import (
	"context"
	"fmt"
)

// Compile-time assertions that every coach satisfies the interface.
var (
	_ Coach = (*Fake)(nil)
	_ Coach = (*GeminiCoach)(nil)
	_ Coach = (*OllamaCoach)(nil)
)

// Fake is a deterministic coach for tests and CI (COACH=fake). It ignores the LLM and
// returns templated prose so the review/coach UI can be exercised without a model.
type Fake struct{}

// NewFake returns a deterministic coach.
func NewFake() *Fake { return &Fake{} }

func (f *Fake) Name() string { return "fake" }

func (f *Fake) CoachMove(_ context.Context, in MoveInput) (Coaching, error) {
	lang := normLang(in.Lang)
	var text string
	if lang == "en" {
		text = fmt.Sprintf("The engine prefers %s (eval %s); your %s leaves the evaluation at %s.",
			in.BestSAN, formatEval(in.EvalBefore), in.PlayedSAN, formatEval(in.EvalAfter))
		if in.Quality != "" {
			text += fmt.Sprintf(" That move is %s.", qualityWord(in.Quality, lang))
		}
	} else {
		text = fmt.Sprintf("Die Engine bevorzugt %s (Bewertung %s); dein %s führt zur Bewertung %s.",
			in.BestSAN, formatEval(in.EvalBefore), in.PlayedSAN, formatEval(in.EvalAfter))
		if in.Quality != "" {
			text += fmt.Sprintf(" Dieser Zug ist %s.", qualityWord(in.Quality, lang))
		}
	}
	return Coaching{Text: text, Model: f.Name(), Lang: lang}, nil
}

func (f *Fake) CoachGame(_ context.Context, in GameInput) (Coaching, error) {
	lang := normLang(in.Lang)
	var text string
	if lang == "en" {
		text = fmt.Sprintf("This game lasted %d half-moves. Review the flagged turning points and keep your king safe.", len(in.Moves))
	} else {
		text = fmt.Sprintf("Diese Partie dauerte %d Halbzüge. Sieh dir die markierten Wendepunkte an und achte auf die Königssicherheit.", len(in.Moves))
	}
	return Coaching{Text: text, Model: f.Name(), Lang: lang}, nil
}
