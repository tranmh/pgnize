package recognition

import (
	"context"

	"github.com/tranmh/pgnize/internal/domain"
)

// Fake is a deterministic recognizer for tests and CI (RECOGNIZER=fake). It ignores the
// image and returns a fixed German-notation opening that postprocessing turns into legal SAN.
type Fake struct{}

// NewFake returns a deterministic recognizer.
func NewFake() *Fake { return &Fake{} }

func (f *Fake) Name() string { return "fake" }

func (f *Fake) Recognize(_ context.Context, _ ScoreSheetInput) (RecognitionResult, error) {
	// Ruy Lopez opening written in German piece letters (S=N, L=B).
	tokens := []string{"e4", "e5", "Sf3", "Sc6", "Lb5", "a6", "La4", "Sf6", "0-0", "Le7"}
	mt := make([]MoveToken, len(tokens))
	for i, t := range tokens {
		side := SideWhite
		if i%2 == 1 {
			side = SideBlack
		}
		mt[i] = MoveToken{Ply: i + 1, Side: side, Text: t, Confidence: 0.9}
	}
	return RecognitionResult{
		Header: domain.Header{
			White:  "Doe, John",
			Black:  "Roe, Jane",
			Event:  "Club Championship",
			Date:   "2026.06.19",
			Result: "*",
		},
		MoveTokens: mt,
		Confidence: 0.9,
		RawJSON:    `{"recognizer":"fake"}`,
	}, nil
}
