package chesskit

import "errors"

var (
	// ErrIllegalMove is returned when a SAN move is not legal in the given position.
	ErrIllegalMove = errors.New("chesskit: illegal move")
	// ErrAmbiguousMove is returned when a SAN move could refer to more than one legal move.
	ErrAmbiguousMove = errors.New("chesskit: ambiguous move")
)
