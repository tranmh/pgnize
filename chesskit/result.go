// Package chesskit is a standalone chess core that wraps github.com/notnil/chess
// and exposes only JSON-friendly value types. No notnil/chess type appears in any
// exported signature.
package chesskit

import "strings"

// Result is a PGN game result token.
type Result string

const (
	ResultWhiteWin Result = "1-0"
	ResultBlackWin Result = "0-1"
	ResultDraw     Result = "1/2-1/2"
	ResultOngoing  Result = "*"
)

// NormalizeResult maps loose result strings ("1:0", "½-½", "", "remis", ...) to a Result.
func NormalizeResult(s string) Result {
	t := strings.TrimSpace(s)
	if t == "" {
		return ResultOngoing
	}
	// Normalize common separators and casing for matching.
	lower := strings.ToLower(t)
	// Collapse whitespace.
	lower = strings.Join(strings.Fields(lower), " ")

	switch lower {
	case "1-0", "1:0", "1 - 0", "1 0", "white", "white wins", "1":
		return ResultWhiteWin
	case "0-1", "0:1", "0 - 1", "0 1", "black", "black wins":
		return ResultBlackWin
	case "1/2-1/2", "1/2", "1/2 1/2", "½-½", "½", "0.5-0.5", "0.5",
		"draw", "drawn", "remis", "patt", "stalemate", "=", "½-½ ", "1/2 - 1/2":
		return ResultDraw
	case "*", "ongoing", "unknown", "open", "?", "in progress":
		return ResultOngoing
	}

	// Heuristic fallbacks for fuzzier inputs.
	switch {
	case strings.Contains(lower, "½") || strings.Contains(lower, "1/2") ||
		strings.Contains(lower, "remis") || strings.Contains(lower, "draw"):
		return ResultDraw
	case lower == "1-0" || strings.HasPrefix(lower, "1-0"):
		return ResultWhiteWin
	case lower == "0-1" || strings.HasPrefix(lower, "0-1"):
		return ResultBlackWin
	}
	return ResultOngoing
}
