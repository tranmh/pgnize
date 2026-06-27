package engine

import (
	"strconv"
	"strings"
)

// parseInfo parses a UCI "info" line into a Line plus its MultiPV index (1-based; 1 when
// the engine omits "multipv"). ok is false for info lines without a score (e.g. "info
// depth 1 currmove ...") and for non-info lines. Example inputs:
//
//	info depth 18 seldepth 24 multipv 1 score cp 23 nodes 12345 pv e2e4 e7e5 g1f3
//	info depth 5 score mate 3 pv d1h5 e8e7 h5e5
func parseInfo(line string) (multipv int, l Line, ok bool) {
	f := strings.Fields(line)
	if len(f) == 0 || f[0] != "info" {
		return 0, Line{}, false
	}
	multipv = 1
	hasScore := false
	for i := 1; i < len(f); i++ {
		switch f[i] {
		case "depth":
			if i+1 < len(f) {
				l.Depth, _ = strconv.Atoi(f[i+1])
			}
		case "multipv":
			if i+1 < len(f) {
				if n, err := strconv.Atoi(f[i+1]); err == nil {
					multipv = n
				}
			}
		case "score":
			if i+2 < len(f) {
				v, err := strconv.Atoi(f[i+2])
				if err == nil {
					switch f[i+1] {
					case "cp":
						cp := v
						l.Cp = &cp
						hasScore = true
					case "mate":
						mate := v
						l.Mate = &mate
						hasScore = true
					}
				}
			}
		case "pv":
			// "pv" is always last; the remaining fields are the variation.
			if i+1 < len(f) {
				l.PV = append([]string(nil), f[i+1:]...)
				l.BestMove = l.PV[0]
			}
			return multipv, l, hasScore
		}
	}
	return multipv, l, hasScore
}

// parseBestMove extracts the move from a UCI "bestmove" line, e.g. "bestmove e2e4 ponder
// e7e5" -> "e2e4". It returns ("", false) for any other line. "bestmove (none)" (no legal
// move — mate/stalemate) yields ("", true).
func parseBestMove(line string) (move string, ok bool) {
	f := strings.Fields(line)
	if len(f) < 2 || f[0] != "bestmove" {
		return "", false
	}
	if f[1] == "(none)" {
		return "", true
	}
	return f[1], true
}
