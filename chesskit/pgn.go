package chesskit

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Move is a single played ply with the positions before and after it.
type Move struct {
	SAN      SAN  `json:"san"`
	FromFEN  FEN  `json:"fromFen"`
	ToFEN    FEN  `json:"toFen"`
	ClockSec *int `json:"clockSec,omitempty"`
}

// Header holds PGN metadata. The Seven-Tag Roster fields are explicit; any other
// tags are preserved in Extra for round-tripping.
type Header struct {
	Event, Site, Date, Round, Board string
	White, Black                    string
	Result                          Result
	Extra                           map[string]string
}

// Game is a parsed chess game.
type Game struct {
	Header   Header `json:"header"`
	Moves    []Move `json:"moves"`
	StartFEN FEN    `json:"startFen"`
}

// sevenTagRoster lists the standard PGN tags that get explicit Header fields and
// are emitted first in a stable order.
var sevenTagRoster = []string{"Event", "Site", "Date", "Round", "White", "Black", "Result"}

// reTagPair matches a PGN tag pair line: [Key "Value"].
var reTagPair = regexp.MustCompile(`^\s*\[\s*([A-Za-z0-9_]+)\s+"((?:[^"\\]|\\.)*)"\s*\]\s*$`)

// reClk matches a %clk annotation inside a comment: [%clk h:mm:ss] (hours optional).
var reClk = regexp.MustCompile(`\[%clk\s+(\d+):([0-5]?\d):([0-5]?\d)\]`)

// reResultToken matches the trailing game result token.
var reResultToken = regexp.MustCompile(`^(1-0|0-1|1/2-1/2|\*)$`)

// ParsePGN tolerantly parses one or more games from PGN text. It extracts %clk
// comments into Move.ClockSec. A game whose movetext contains an illegal move is
// truncated at that move (not dropped). Non-standard header tags are preserved in
// Header.Extra.
func ParsePGN(text string) ([]Game, error) {
	chunks := splitGames(text)
	var games []Game
	for _, c := range chunks {
		g, ok := parseOneGame(c)
		if ok {
			games = append(games, g)
		}
	}
	return games, nil
}

// splitGames divides multi-game PGN text into per-game chunks. A new game starts
// at a tag-pair block that follows a previous game's movetext.
func splitGames(text string) []string {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	var chunks []string
	var cur []string
	seenMovetext := false

	flush := func() {
		if len(cur) > 0 {
			joined := strings.TrimSpace(strings.Join(cur, "\n"))
			if joined != "" {
				chunks = append(chunks, joined)
			}
		}
		cur = nil
		seenMovetext = false
	}

	for _, ln := range lines {
		isTag := reTagPair.MatchString(ln)
		if isTag && seenMovetext {
			// A tag line after we've seen movetext begins a new game.
			flush()
		}
		if !isTag && strings.TrimSpace(ln) != "" {
			seenMovetext = true
		}
		cur = append(cur, ln)
	}
	flush()
	return chunks
}

// parseOneGame parses a single game chunk (tags + movetext). It returns ok=false
// only for an entirely empty chunk.
func parseOneGame(chunk string) (Game, bool) {
	lines := strings.Split(chunk, "\n")
	hdr := Header{Result: ResultOngoing, Extra: map[string]string{}}
	var movetextLines []string

	for _, ln := range lines {
		if m := reTagPair.FindStringSubmatch(ln); m != nil {
			key, val := m[1], unescapePGN(m[2])
			applyTag(&hdr, key, val)
			continue
		}
		movetextLines = append(movetextLines, ln)
	}

	movetext := strings.TrimSpace(strings.Join(movetextLines, " "))
	if movetext == "" && len(hdr.Extra) == 0 &&
		hdr.Event == "" && hdr.White == "" && hdr.Black == "" {
		// Nothing useful in this chunk.
		// Still emit if it had a result tag etc; guard only the truly empty.
	}

	start := hdr.startFEN()
	moves, res := parseMovetext(movetext, start)
	if res != "" {
		hdr.Result = res
	}

	g := Game{Header: hdr, Moves: moves, StartFEN: start}
	if len(hdr.Extra) == 0 {
		g.Header.Extra = nil
	}
	// Treat a chunk with no tags and no moves as not-a-game.
	if len(moves) == 0 && hdr.Event == "" && hdr.White == "" && hdr.Black == "" &&
		hdr.Site == "" && len(hdr.Extra) == 0 && res == "" {
		return Game{}, false
	}
	return g, true
}

// applyTag sets a Seven-Tag-Roster (or Board/FEN) field, or stores into Extra.
func applyTag(h *Header, key, val string) {
	switch key {
	case "Event":
		h.Event = val
	case "Site":
		h.Site = val
	case "Date":
		h.Date = val
	case "Round":
		h.Round = val
	case "White":
		h.White = val
	case "Black":
		h.Black = val
	case "Board":
		h.Board = val
	case "Result":
		h.Result = NormalizeResult(val)
	default:
		if h.Extra == nil {
			h.Extra = map[string]string{}
		}
		h.Extra[key] = val
	}
}

// startFEN returns the starting position for the game, honoring a FEN/SetUp tag in Extra.
func (h Header) startFEN() FEN {
	if h.Extra != nil {
		if f, ok := h.Extra["FEN"]; ok && strings.TrimSpace(f) != "" {
			return FEN(strings.TrimSpace(f))
		}
	}
	return StartingFEN()
}

// token is one parsed movetext element.
type token struct {
	san     string
	comment string
}

// parseMovetext tokenizes movetext, replays moves from start, attaches %clk clocks,
// and truncates at the first illegal/ambiguous move. It returns the moves and the
// result token if present.
func parseMovetext(movetext string, start FEN) ([]Move, Result) {
	toks, result := tokenizeMovetext(movetext)

	pos, err := positionFromFEN(start)
	if err != nil {
		return nil, result
	}

	var moves []Move
	for _, t := range toks {
		mv, canonical, mErr := resolveMove(pos, SAN(t.san))
		if mErr != nil {
			// Truncate the game at the first illegal/ambiguous move.
			break
		}
		from := FEN(pos.String())
		pos = pos.Update(mv)
		m := Move{
			SAN:     SAN(canonical),
			FromFEN: from,
			ToFEN:   FEN(pos.String()),
		}
		if sec, ok := clockFromComment(t.comment); ok {
			m.ClockSec = &sec
		}
		moves = append(moves, m)
	}
	return moves, result
}

// tokenizeMovetext walks movetext extracting SAN tokens and the comment that
// follows each move, plus a trailing result token.
func tokenizeMovetext(movetext string) ([]token, Result) {
	var toks []token
	var result Result

	i := 0
	n := len(movetext)
	for i < n {
		c := movetext[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			i++
		case c == '{':
			// Comment: attach to the most recent move token.
			end := strings.IndexByte(movetext[i:], '}')
			var body string
			if end < 0 {
				body = movetext[i+1:]
				i = n
			} else {
				body = movetext[i+1 : i+end]
				i = i + end + 1
			}
			if len(toks) > 0 {
				if toks[len(toks)-1].comment != "" {
					toks[len(toks)-1].comment += " "
				}
				toks[len(toks)-1].comment += strings.TrimSpace(body)
			}
		case c == ';':
			// Rest-of-line comment.
			end := strings.IndexByte(movetext[i:], '\n')
			if end < 0 {
				i = n
			} else {
				i = i + end + 1
			}
		case c == '(':
			// Variation: skip the whole balanced parenthesized block.
			depth := 0
			for i < n {
				if movetext[i] == '(' {
					depth++
				} else if movetext[i] == ')' {
					depth--
					if depth == 0 {
						i++
						break
					}
				}
				i++
			}
		case c == '$':
			// NAG annotation glyph: skip the token.
			i++
			for i < n && movetext[i] >= '0' && movetext[i] <= '9' {
				i++
			}
		default:
			// Read a whitespace/brace-delimited word.
			j := i
			for j < n {
				ch := movetext[j]
				if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' ||
					ch == '{' || ch == '(' || ch == ';' {
					break
				}
				j++
			}
			word := movetext[i:j]
			i = j
			word = strings.TrimSpace(word)
			if word == "" {
				continue
			}
			if reResultToken.MatchString(word) {
				result = Result(word)
				continue
			}
			san := stripMoveNumber(word)
			if san == "" {
				continue
			}
			toks = append(toks, token{san: san})
		}
	}
	return toks, result
}

// stripMoveNumber removes a leading move number like "12." or "12..." and returns
// the SAN portion. Returns "" if the word is purely a move number.
func stripMoveNumber(word string) string {
	w := word
	// Strip leading digits.
	k := 0
	for k < len(w) && w[k] >= '0' && w[k] <= '9' {
		k++
	}
	if k > 0 {
		// Strip following dots.
		for k < len(w) && w[k] == '.' {
			k++
		}
		w = w[k:]
	}
	w = strings.TrimSpace(w)
	if w == "" {
		return ""
	}
	// Bare dots (continuation) or stray punctuation.
	if strings.Trim(w, ".") == "" {
		return ""
	}
	return w
}

// clockFromComment extracts seconds from a %clk annotation if present.
func clockFromComment(comment string) (int, bool) {
	m := reClk.FindStringSubmatch(comment)
	if m == nil {
		return 0, false
	}
	h, _ := strconv.Atoi(m[1])
	mins, _ := strconv.Atoi(m[2])
	secs, _ := strconv.Atoi(m[3])
	return h*3600 + mins*60 + secs, true
}

// WritePGN renders a single game: Seven-Tag Roster first (stable order), then
// sorted Extra tags, a blank line, then movetext with move numbers, %clk comments,
// and the trailing result token.
func WritePGN(g Game) (string, error) {
	var b strings.Builder

	res := g.Header.Result
	if res == "" {
		res = ResultOngoing
	}

	// Seven-Tag Roster in stable order.
	for _, key := range sevenTagRoster {
		var val string
		switch key {
		case "Event":
			val = g.Header.Event
		case "Site":
			val = g.Header.Site
		case "Date":
			val = g.Header.Date
		case "Round":
			val = g.Header.Round
		case "White":
			val = g.Header.White
		case "Black":
			val = g.Header.Black
		case "Result":
			val = string(res)
		}
		if val == "" {
			val = defaultTag(key)
		}
		b.WriteString(fmt.Sprintf("[%s \"%s\"]\n", key, escapePGN(val)))
	}

	// Board is not part of the Seven-Tag Roster; emit it (and any FEN setup) and
	// Extra tags in sorted order for stability.
	extra := map[string]string{}
	for k, v := range g.Header.Extra {
		extra[k] = v
	}
	if g.Header.Board != "" {
		extra["Board"] = g.Header.Board
	}
	// If the game starts from a non-standard position, record SetUp/FEN.
	if g.StartFEN != "" && g.StartFEN != StartingFEN() {
		if _, ok := extra["FEN"]; !ok {
			extra["FEN"] = string(g.StartFEN)
		}
		if _, ok := extra["SetUp"]; !ok {
			extra["SetUp"] = "1"
		}
	}
	keys := make([]string, 0, len(extra))
	for k := range extra {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		b.WriteString(fmt.Sprintf("[%s \"%s\"]\n", k, escapePGN(extra[k])))
	}

	b.WriteString("\n")

	// Movetext.
	movetext := buildMovetext(g, res)
	b.WriteString(movetext)
	b.WriteString("\n")

	return b.String(), nil
}

// buildMovetext renders the moves with move numbers, clock comments, and the result.
func buildMovetext(g Game, res Result) string {
	// Determine the side to move and move number of the start position.
	startNum, whiteToMove := startCounters(g.StartFEN)

	var parts []string
	num := startNum
	white := whiteToMove

	for i, mv := range g.Moves {
		if white {
			parts = append(parts, fmt.Sprintf("%d.", num))
		} else if i == 0 {
			// Game starts with Black to move.
			parts = append(parts, fmt.Sprintf("%d...", num))
		}
		parts = append(parts, string(mv.SAN))
		if mv.ClockSec != nil {
			parts = append(parts, fmt.Sprintf("{[%%clk %s]}", formatClock(*mv.ClockSec)))
		}
		if !white {
			num++
		}
		white = !white
	}
	parts = append(parts, string(res))
	return wrapMovetext(parts)
}

// startCounters returns the fullmove number and side-to-move from a FEN.
func startCounters(f FEN) (num int, whiteToMove bool) {
	s := string(f)
	if s == "" {
		return 1, true
	}
	fields := strings.Fields(s)
	num = 1
	whiteToMove = true
	if len(fields) >= 2 {
		whiteToMove = fields[1] != "b"
	}
	if len(fields) >= 6 {
		if n, err := strconv.Atoi(fields[5]); err == nil && n > 0 {
			num = n
		}
	}
	return num, whiteToMove
}

// wrapMovetext joins tokens with spaces, wrapping at ~80 columns like standard PGN.
func wrapMovetext(parts []string) string {
	var b strings.Builder
	lineLen := 0
	for i, p := range parts {
		add := len(p)
		if i > 0 {
			add++ // space
		}
		if lineLen > 0 && lineLen+add > 80 {
			b.WriteString("\n")
			b.WriteString(p)
			lineLen = len(p)
			continue
		}
		if i > 0 {
			b.WriteString(" ")
		}
		b.WriteString(p)
		lineLen += add
	}
	return b.String()
}

// formatClock renders seconds as h:mm:ss.
func formatClock(sec int) string {
	if sec < 0 {
		sec = 0
	}
	h := sec / 3600
	m := (sec % 3600) / 60
	s := sec % 60
	return fmt.Sprintf("%d:%02d:%02d", h, m, s)
}

// WriteBundlePGN renders many games separated by a blank line (library export).
func WriteBundlePGN(games []Game) (string, error) {
	parts := make([]string, 0, len(games))
	for _, g := range games {
		s, err := WritePGN(g)
		if err != nil {
			return "", err
		}
		parts = append(parts, strings.TrimRight(s, "\n"))
	}
	return strings.Join(parts, "\n\n") + "\n", nil
}

// defaultTag returns the PGN default value for a Seven-Tag-Roster field.
func defaultTag(key string) string {
	switch key {
	case "Date":
		return "????.??.??"
	case "Result":
		return string(ResultOngoing)
	default:
		return "?"
	}
}

// escapePGN escapes backslashes and quotes for a PGN tag value.
func escapePGN(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

// unescapePGN reverses escapePGN.
func unescapePGN(s string) string {
	s = strings.ReplaceAll(s, `\"`, `"`)
	s = strings.ReplaceAll(s, `\\`, `\`)
	return s
}
