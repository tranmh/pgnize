// Package engine is a server-side chess engine (UCI/Stockfish) used by the conversational
// coach to ground its answers in real analysis. It deliberately knows NOTHING about chess
// rules: it takes plain string FENs that the caller has ALREADY validated via chesskit, and
// returns plain analysis values. This keeps the correctness authority (chesskit) and the
// search engine cleanly separated — the engine never decides legality, and chesskit never
// shells out to a binary.
//
// Backends: a deterministic Fake (tests/CI, no binary required) and Stockfish (a UCI
// subprocess pool). Both satisfy Engine, so the EvalMove/FindMate helpers built on the
// interface work against either.
package engine

import (
	"context"
	"fmt"
)

// Line is one analysis result (one principal variation). At most one of Cp/Mate is set.
// Both are reported from the SIDE-TO-MOVE point of view (UCI convention): positive Cp /
// positive Mate is good for whoever is to move in the analyzed position. Callers convert
// to White-POV at the boundary using the position's side to move.
type Line struct {
	Cp       *int     `json:"cp,omitempty"`   // centipawns, side-to-move POV; nil when Mate set
	Mate     *int     `json:"mate,omitempty"` // signed moves-to-mate, side-to-move POV; nil when Cp set
	Depth    int      `json:"depth"`
	PV       []string `json:"pv"`       // principal variation in UCI long algebraic (e.g. "e2e4")
	BestMove string   `json:"bestMove"` // UCI; == PV[0] when present
}

// Options bound a single search. MoveTimeMs (or Depth) caps how long the engine thinks;
// MultiPV asks for the top-N candidate lines.
type Options struct {
	MoveTimeMs int // hard cap per search in ms; 0 -> backend default
	Depth      int // fixed depth; when >0 it takes precedence over MoveTimeMs
	MultiPV    int // number of PVs to return (best first); 0 -> 1
}

// Analysis is the result of one Analyze call: the analyzed FEN and its best lines.
type Analysis struct {
	FEN   string `json:"fen"`
	Lines []Line `json:"lines"` // best first; len matches the effective MultiPV
}

// Best returns the top line, or a zero Line when there are none.
func (a Analysis) Best() Line {
	if len(a.Lines) == 0 {
		return Line{}
	}
	return a.Lines[0]
}

// Engine runs searches on already-validated FEN strings.
type Engine interface {
	// Analyze searches a position. fen must be a legal, chesskit-normalized FEN.
	Analyze(ctx context.Context, fen string, opts Options) (Analysis, error)
	Name() string  // e.g. "stockfish:16" | "fake"
	Close() error  // tears down any subprocesses
}

// mateBase is large enough that any real centipawn score is dominated by a mate score,
// while keeping "mate in 1" strictly better than "mate in 5" (and "mated in 1" worst).
const mateBase = 1_000_000

// score collapses a Line to a single comparable integer from the side-to-move POV, so that
// mate and centipawn lines can be compared. Faster mates rank higher; faster losses lower.
func (l Line) score() int {
	switch {
	case l.Mate != nil:
		m := *l.Mate
		if m > 0 {
			return mateBase - m
		}
		return -mateBase - m // m<0: closer mate-against == more negative
	case l.Cp != nil:
		return *l.Cp
	default:
		return 0
	}
}

// negate flips a Line to the opposite POV (used to express the eval of a move from the
// mover's side, given the engine analyzes the resulting position from the opponent's side).
func negate(l Line) Line {
	out := Line{Depth: l.Depth, PV: l.PV, BestMove: l.BestMove}
	if l.Cp != nil {
		v := -*l.Cp
		out.Cp = &v
	}
	if l.Mate != nil {
		v := -*l.Mate
		out.Mate = &v
	}
	return out
}

// EvalMove answers "how good is this specific move, and how much does it lose versus the
// best move" — the primitive behind "why is X bad". fen is the position before the move;
// childFEN is the (already chesskit-validated) position after it. It returns the best line
// from the mover's POV, the played move's resulting line from the mover's POV, and the
// centipawn-equivalent delta (played - best; negative means the move is worse than best).
func EvalMove(ctx context.Context, eng Engine, fen, childFEN string, opts Options) (best, played Line, deltaCp int, err error) {
	a, err := eng.Analyze(ctx, fen, opts)
	if err != nil {
		return Line{}, Line{}, 0, fmt.Errorf("analyze parent: %w", err)
	}
	best = a.Best()

	c, err := eng.Analyze(ctx, childFEN, opts)
	if err != nil {
		return Line{}, Line{}, 0, fmt.Errorf("analyze child: %w", err)
	}
	// childFEN is the opponent to move; negate to express it from the mover's POV.
	played = negate(c.Best())
	return best, played, played.score() - best.score(), nil
}

// FindMate reports a forced mate for the side to move, if the engine sees one. It returns
// the signed moves-to-mate (positive) and the mating PV, or nil when there is none.
func FindMate(ctx context.Context, eng Engine, fen string, opts Options) (mateIn *int, pv []string, err error) {
	a, err := eng.Analyze(ctx, fen, opts)
	if err != nil {
		return nil, nil, err
	}
	for _, l := range a.Lines {
		if l.Mate != nil && *l.Mate > 0 {
			m := *l.Mate
			return &m, l.PV, nil
		}
	}
	return nil, nil, nil
}
