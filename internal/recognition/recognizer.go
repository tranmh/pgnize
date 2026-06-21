// Package recognition turns score-sheet images into draft move lists.
// Model-specific code sits behind the Recognizer interface; orchestration and the
// German-notation postprocessing live here (never in chesskit).
package recognition

import (
	"context"

	"github.com/tranmh/pgnize/internal/domain"
)

// Side of a move.
const (
	SideWhite = "white"
	SideBlack = "black"
)

// Layout hint for a score sheet.
type Layout string

const (
	LayoutUnknown   Layout = "unknown"
	LayoutTwoColumn Layout = "two_column"
)

// Example is a prior corrected sheet used as few-shot context.
type Example struct {
	Header     domain.Header `json:"header"`
	SANs       []string      `json:"sans"`
	ImageBase64 string       `json:"-"` // optional thumbnail; omitted from prompt text
}

// Hint carries optional context to steer recognition.
type Hint struct {
	Layout       Layout
	KnownPlayers []string
}

// ScoreSheetInput is one recognition request.
type ScoreSheetInput struct {
	Image    []byte
	MimeType string
	FewShot  []Example
	Hint     *Hint
}

// MoveToken is a raw move as read by the model, NOT yet legality-checked.
type MoveToken struct {
	Ply        int     `json:"ply"`
	Side       string  `json:"side"`
	Text       string  `json:"text"`
	Confidence float64 `json:"confidence"`
}

// RecognitionResult is the model's raw output.
type RecognitionResult struct {
	Header     domain.Header `json:"header"`
	MoveTokens []MoveToken   `json:"moveTokens"`
	Confidence float64       `json:"confidence"`
	RawJSON    string        `json:"-"`
}

// PositionInput is one board-position recognition request.
type PositionInput struct {
	Image    []byte
	MimeType string
}

// PositionResult is the model's raw read of a single board position.
type PositionResult struct {
	Grid        []string `json:"grid"`        // 8 strings, ranks 8→1, files a→h, FEN letters, '.'=empty
	SideToMove  string   `json:"sideToMove"`  // "white"|"black"|""
	Orientation string   `json:"orientation"` // "white_bottom"|"black_bottom"
	Confidence  float64  `json:"-"`
	RawJSON     string   `json:"-"`
}

// Recognizer reads a score sheet or a single board position. Implementations: fake
// (tests), ollama (local VLM), gemini (cloud VLM).
type Recognizer interface {
	Recognize(ctx context.Context, in ScoreSheetInput) (RecognitionResult, error)
	RecognizePosition(ctx context.Context, in PositionInput) (PositionResult, error)
	Name() string
}
