package chesskit

import (
	"strings"

	"github.com/notnil/chess"
)

// SAN is a Standard Algebraic Notation move (English letters), e.g. "Nf3", "O-O", "exd6", "e8=Q".
type SAN string

var algebraic = chess.AlgebraicNotation{}

// stripCheck removes trailing check/mate markers from a SAN string.
func stripCheck(s string) string {
	return strings.TrimRight(strings.TrimSpace(s), "+#")
}

// normalizeSAN canonicalizes castling spellings (0-0 -> O-O) so that both
// zero and letter-O notations are accepted on input.
func normalizeSAN(s string) string {
	t := strings.TrimSpace(s)
	// Castling can be written with zeros or capital O's, optionally with check marks.
	core := stripCheck(t)
	suffix := t[len(core):]
	switch core {
	case "0-0-0", "O-O-O", "o-o-o":
		return "O-O-O" + suffix
	case "0-0", "O-O", "o-o":
		return "O-O" + suffix
	}
	return t
}

// sanSignature reduces a SAN to its disambiguation-independent core:
// piece letter (if any), capture flag, destination square, and promotion piece.
// This lets us match an under-specified or over-specified input against the
// canonical SAN of a legal move.
func sanSignature(s string) string {
	t := stripCheck(normalizeSAN(s))
	if t == "O-O" || t == "O-O-O" {
		return t
	}

	var piece byte
	i := 0
	if len(t) > 0 && t[0] >= 'A' && t[0] <= 'Z' && t[0] != 'O' {
		piece = t[0]
		i = 1
	}

	// Promotion suffix (=Q etc).
	var promo byte
	if idx := strings.IndexByte(t, '='); idx >= 0 {
		if idx+1 < len(t) {
			promo = t[idx+1]
		}
		t = t[:idx]
	}

	capture := strings.IndexByte(t, 'x') >= 0
	rest := t[i:]
	rest = strings.ReplaceAll(rest, "x", "")

	// Destination square is the trailing file+rank.
	dest := ""
	if len(rest) >= 2 {
		dest = rest[len(rest)-2:]
	}

	var b strings.Builder
	if piece != 0 {
		b.WriteByte(piece)
	}
	if capture {
		b.WriteByte('x')
	}
	b.WriteString(dest)
	if promo != 0 {
		b.WriteByte('=')
		b.WriteByte(promo)
	}
	return b.String()
}

// resolveMove finds the legal notnil move corresponding to san in pos, returning
// the canonical SAN encoding. It maps notnil decode failures onto
// ErrIllegalMove / ErrAmbiguousMove by inspecting the legal move set.
func resolveMove(pos *chess.Position, san SAN) (*chess.Move, string, error) {
	norm := normalizeSAN(string(san))
	if norm == "" {
		return nil, "", ErrIllegalMove
	}

	// Fast path: notnil decodes it directly.
	if mv, err := algebraic.Decode(pos, norm); err == nil {
		return mv, algebraic.Encode(pos, mv), nil
	}

	// notnil does not distinguish ambiguous from illegal, so decide ourselves by
	// matching the input signature against every legal move.
	wantSig := sanSignature(norm)
	wantStripped := stripCheck(norm)

	var matches []*chess.Move
	var exact []*chess.Move
	for _, mv := range pos.ValidMoves() {
		enc := algebraic.Encode(pos, mv)
		if stripCheck(enc) == wantStripped {
			exact = append(exact, mv)
		}
		if sanSignature(enc) == wantSig {
			matches = append(matches, mv)
		}
	}

	switch {
	case len(exact) == 1:
		return exact[0], algebraic.Encode(pos, exact[0]), nil
	case len(matches) == 1:
		return matches[0], algebraic.Encode(pos, matches[0]), nil
	case len(matches) > 1:
		return nil, "", ErrAmbiguousMove
	default:
		return nil, "", ErrIllegalMove
	}
}

// ParseSAN parses one SAN move from a position. Returns ErrIllegalMove / ErrAmbiguousMove.
func ParseSAN(from FEN, san SAN) (Move, error) {
	pos, err := positionFromFEN(from)
	if err != nil {
		return Move{}, err
	}
	mv, canonical, err := resolveMove(pos, san)
	if err != nil {
		return Move{}, err
	}
	next := pos.Update(mv)
	return Move{
		SAN:     SAN(canonical),
		FromFEN: FEN(pos.String()),
		ToFEN:   FEN(next.String()),
	}, nil
}

// Validate reports the resulting FEN after playing san from `from`, or an error.
func Validate(from FEN, san SAN) (to FEN, err error) {
	m, err := ParseSAN(from, san)
	if err != nil {
		return "", err
	}
	return m.ToFEN, nil
}

// UCItoSAN converts a UCI long-algebraic move (e.g. "e2e4", "e7e8q") played from `from`
// into canonical SAN (e.g. "e4", "e8=Q"). Returns ErrIllegalMove if no legal move matches.
// This lets callers that hold engine output (UCI) render it as human-readable SAN without
// leaking any notnil/chess types across the API.
func UCItoSAN(from FEN, uci string) (SAN, error) {
	pos, err := positionFromFEN(from)
	if err != nil {
		return "", err
	}
	for _, mv := range pos.ValidMoves() {
		if mv.String() == uci {
			return SAN(algebraic.Encode(pos, mv)), nil
		}
	}
	return "", ErrIllegalMove
}

// LegalMovesSAN lists every legal move from a position in canonical SAN
// (with disambiguation like Nbd2 / Rfd1 where needed).
func LegalMovesSAN(from FEN) ([]SAN, error) {
	pos, err := positionFromFEN(from)
	if err != nil {
		return nil, err
	}
	moves := pos.ValidMoves()
	out := make([]SAN, 0, len(moves))
	for _, mv := range moves {
		out = append(out, SAN(algebraic.Encode(pos, mv)))
	}
	return out, nil
}

// ApplyMoves replays sans from start. positions[i] is the FEN AFTER sans[i].
// On the first illegal/ambiguous move it stops: err != nil and failedAt is that
// move's index (failedAt == -1 when err == nil).
func ApplyMoves(start FEN, sans []SAN) (positions []FEN, err error, failedAt int) {
	pos, perr := positionFromFEN(start)
	if perr != nil {
		return nil, perr, 0
	}
	positions = make([]FEN, 0, len(sans))
	for i, san := range sans {
		mv, _, mErr := resolveMove(pos, san)
		if mErr != nil {
			return positions, mErr, i
		}
		pos = pos.Update(mv)
		positions = append(positions, FEN(pos.String()))
	}
	return positions, nil, -1
}
