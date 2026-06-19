package chesskit

import (
	"fmt"

	"github.com/notnil/chess"
)

// FEN is a Forsyth–Edwards Notation position string.
type FEN string

const startingFEN FEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

// StartingFEN returns the FEN of the standard chess starting position.
func StartingFEN() FEN {
	return startingFEN
}

// positionFromFEN parses a FEN into a notnil chess.Position.
func positionFromFEN(f FEN) (*chess.Position, error) {
	s := string(f)
	if s == "" {
		s = string(startingFEN)
	}
	opt, err := chess.FEN(s)
	if err != nil {
		return nil, fmt.Errorf("chesskit: invalid FEN %q: %w", s, err)
	}
	g := chess.NewGame(opt)
	return g.Position(), nil
}
