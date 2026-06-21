package chesskit

import (
	"fmt"
	"strings"

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

// NormalizeFEN parses and canonicalizes a FEN, layering on the sanity checks that
// notnil/chess does not perform: exactly one king of each color, and no pawn on the
// back ranks (rank 8 or rank 1). It returns the canonical FEN on success.
func NormalizeFEN(f FEN) (FEN, error) {
	pos, err := positionFromFEN(f)
	if err != nil {
		return "", err
	}
	board := strings.Split(strings.Fields(pos.String())[0], "/")
	if len(board) != 8 {
		return "", fmt.Errorf("chesskit: FEN board must have 8 ranks, got %d", len(board))
	}
	whiteKings, blackKings := 0, 0
	for _, rank := range board {
		for _, r := range rank {
			switch r {
			case 'K':
				whiteKings++
			case 'k':
				blackKings++
			}
		}
	}
	if whiteKings != 1 {
		return "", fmt.Errorf("chesskit: FEN must have exactly one white king, got %d", whiteKings)
	}
	if blackKings != 1 {
		return "", fmt.Errorf("chesskit: FEN must have exactly one black king, got %d", blackKings)
	}
	if strings.ContainsAny(board[0], "Pp") {
		return "", fmt.Errorf("chesskit: FEN has a pawn on rank 8")
	}
	if strings.ContainsAny(board[7], "Pp") {
		return "", fmt.Errorf("chesskit: FEN has a pawn on rank 1")
	}
	return FEN(pos.String()), nil
}
