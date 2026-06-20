package httpapi

import (
	"strings"

	"github.com/tranmh/chesskit"
	"github.com/tranmh/pgnize/internal/domain"
)

// moveInput is one ply supplied by the client on save/export.
type moveInput struct {
	Ply      int    `json:"ply"`
	San      string `json:"san"`
	ClockSec *int   `json:"clockSec"`
}

// verifiedGame is the result of replaying client moves through chesskit.
type verifiedGame struct {
	Moves    []domain.Move
	PGN      string
	FailedAt int // -1 when all legal
}

// buildVerifiedGame replays the supplied SAN moves from startFEN. On the first illegal
// move it returns FailedAt >= 0 and an error; otherwise it returns the per-ply moves
// (with resulting FENs) and the canonical PGN. The server is authoritative here.
func buildVerifiedGame(h domain.Header, startFEN string, in []moveInput) (verifiedGame, error) {
	if startFEN == "" {
		startFEN = string(chesskit.StartingFEN())
	}
	if h.Result == "" {
		h.Result = "*"
	}
	sans := make([]chesskit.SAN, len(in))
	for i, m := range in {
		sans[i] = chesskit.SAN(strings.TrimSpace(m.San))
	}
	positions, err, failedAt := chesskit.ApplyMoves(chesskit.FEN(startFEN), sans)
	if err != nil {
		return verifiedGame{FailedAt: failedAt}, err
	}

	moves := make([]domain.Move, len(in))
	ckMoves := make([]chesskit.Move, len(in))
	prev := chesskit.FEN(startFEN)
	for i, m := range in {
		moves[i] = domain.Move{
			Ply:        i + 1,
			Side:       sideToMove(string(prev)),
			SAN:        string(sans[i]),
			FenAfter:   string(positions[i]),
			ClockSec:   m.ClockSec,
			IsLegal:    true,
			Confidence: 1.0, // human-verified on save/export
		}
		ckMoves[i] = chesskit.Move{SAN: sans[i], FromFEN: prev, ToFEN: positions[i], ClockSec: m.ClockSec}
		prev = positions[i]
	}

	pgn, err := chesskit.WritePGN(chesskit.Game{
		Header:   toChessHeader(h),
		Moves:    ckMoves,
		StartFEN: chesskit.FEN(startFEN),
	})
	if err != nil {
		return verifiedGame{FailedAt: -1}, err
	}
	return verifiedGame{Moves: moves, PGN: pgn, FailedAt: -1}, nil
}

func toChessHeader(h domain.Header) chesskit.Header {
	return chesskit.Header{
		Event:  h.Event,
		Site:   h.Site,
		Date:   h.Date,
		Round:  h.Round,
		Board:  h.Board,
		White:  h.White,
		Black:  h.Black,
		Result: chesskit.NormalizeResult(h.Result),
	}
}

func sideToMove(fen string) string {
	parts := strings.Fields(fen)
	if len(parts) >= 2 && parts[1] == "b" {
		return domain.SideBlack
	}
	return domain.SideWhite
}
