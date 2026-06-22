// Package coaching turns engine evaluations into human teaching prose — the "why"
// behind the numbers. Model-specific code sits behind the Coach interface, mirroring the
// recognition package. The engine evaluation is computed elsewhere (the browser Stockfish)
// and every position/move is validated via chesskit before it reaches a Coach, so the
// coach only ever sees legal positions. Coaching is ADVISORY: it never affects the
// correctness of a saved PGN.
package coaching

import (
	"context"

	"github.com/tranmh/pgnize/internal/domain"
)

// LangDefault is the default coaching language. PGNize is a German-first product
// (handwritten German Partieformular), so coaching prose defaults to German.
const LangDefault = "de"

// Eval is a White-POV engine evaluation; at most one of Cp/Mate is set. Cp is centipawns
// (positive = good for White); Mate is signed moves-to-mate (positive = White mates).
type Eval struct {
	Cp   *int `json:"cp"`
	Mate *int `json:"mate"`
}

// MoveInput is one per-move coaching request. FEN is the position BEFORE the move. The
// caller (the HTTP handler) has already validated FEN, PlayedSAN and BestSAN via chesskit.
type MoveInput struct {
	FEN        string
	Side       string // "white" | "black"
	PlayedSAN  string
	BestSAN    string
	BestLine   []string // engine principal variation in SAN (optional)
	EvalBefore Eval
	EvalAfter  Eval
	Quality    string // "blunder" | "mistake" | "inaccuracy" | ""
	Lang       string // "" -> LangDefault
}

// PositionInput is a single-position coaching request (e.g. a pasted FEN with no moves):
// explain whose move it is, how the engine evaluates it, and the recommended plan.
type PositionInput struct {
	FEN      string
	Side     string // side to move: "white" | "black"
	BestSAN  string // engine's recommended move (optional)
	BestLine []string
	Eval     Eval // White-POV engine evaluation of the position
	Lang     string
}

// GameMove summarizes one ply for whole-game coaching.
type GameMove struct {
	Ply       int
	Side      string
	SAN       string
	EvalAfter Eval
	Quality   string
}

// GameInput is a whole-game coaching request.
type GameInput struct {
	StartFEN string
	Header   domain.Header
	Moves    []GameMove
	Lang     string
}

// Coaching is the produced teaching text.
type Coaching struct {
	Text  string `json:"text"`
	Model string `json:"model"`
	Lang  string `json:"lang"`
}

// Coach turns engine numbers into teaching prose. Implementations: fake (tests/CI),
// gemini and ollama (LLM). A Coach must never be treated as the correctness authority.
type Coach interface {
	CoachMove(ctx context.Context, in MoveInput) (Coaching, error)
	CoachGame(ctx context.Context, in GameInput) (Coaching, error)
	CoachPosition(ctx context.Context, in PositionInput) (Coaching, error)
	Name() string
}

// normLang returns the effective language code, defaulting to German.
func normLang(l string) string {
	if l == "" {
		return LangDefault
	}
	return l
}
