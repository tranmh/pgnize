package recognition

import (
	"regexp"
	"sort"
	"strings"

	"github.com/tranmh/chesskit"
	"github.com/tranmh/pgnize/internal/domain"
)

// germanPiece maps German piece letters to English SAN piece letters.
var germanPiece = map[byte]byte{
	'K': 'K', // König  -> King
	'D': 'Q', // Dame   -> Queen
	'T': 'R', // Turm   -> Rook
	'L': 'B', // Läufer -> Bishop
	'S': 'N', // Springer -> Knight
}

var promoSuffix = regexp.MustCompile(`^([a-h][18])[=]?([QRBNqrbnDTLS])$`)

// GermanToSAN converts a single handwritten move token (German or mixed notation) to
// canonical-ish English SAN. It does NOT check legality — that is chesskit's job.
func GermanToSAN(token string) string {
	s := strings.TrimSpace(token)
	if s == "" {
		return ""
	}
	// Castling: 0-0 / O-O / o-o, with optional check/mate marker.
	low := strings.ToLower(strings.NewReplacer("–", "-", "—", "-").Replace(s))
	suffix := ""
	for strings.HasSuffix(low, "+") || strings.HasSuffix(low, "#") {
		suffix = string(low[len(low)-1]) + suffix
		low = low[:len(low)-1]
		s = s[:len(s)-1]
	}
	switch low {
	case "0-0", "o-o":
		return "O-O" + suffix
	case "0-0-0", "o-o-o":
		return "O-O-O" + suffix
	}
	// Result / draw / resign words are not moves.
	switch low {
	case "remis", "rem.", "1/2", "½", "aufg.", "aufgegeben", "1-0", "0-1":
		return ""
	}

	s = strings.TrimSuffix(s, "e.p.")
	s = strings.TrimSuffix(s, "ep")
	s = strings.ReplaceAll(s, ":", "") // German capture colon (e.g. "Sf3:")
	s = strings.TrimSpace(s)

	// Promotion: e8D, e8=D, e8Q -> e8=Q
	if m := promoSuffix.FindStringSubmatch(s); m != nil {
		p := m[2][0]
		if eng, ok := germanPiece[upper(p)]; ok {
			return m[1] + "=" + string(eng) + suffix
		}
		return m[1] + "=" + string(upper(p)) + suffix
	}

	// Translate a leading German piece letter.
	if len(s) > 0 {
		if eng, ok := germanPiece[s[0]]; ok {
			s = string(eng) + s[1:]
		}
	}
	return s + suffix
}

func upper(b byte) byte {
	if b >= 'a' && b <= 'z' {
		return b - 32
	}
	return b
}

// Reconcile translates raw tokens to SAN and replays them through chesskit, producing
// per-ply moves with legality + resulting FEN. The first illegal move and everything
// after it are marked illegal (fenAfter empty), mirroring the review-loop semantics.
func Reconcile(startFEN string, tokens []MoveToken) []domain.Move {
	if startFEN == "" {
		startFEN = string(chesskit.StartingFEN())
	}
	cur := chesskit.FEN(startFEN)
	blocked := false
	out := make([]domain.Move, 0, len(tokens))
	for i, t := range tokens {
		san := GermanToSAN(t.Text)
		m := domain.Move{
			Ply:            i + 1,
			Side:           sideToMove(string(cur)),
			SAN:            san,
			RecognizedText: t.Text,
		}
		if blocked || san == "" {
			m.IsLegal = false
			out = append(out, m)
			if san == "" {
				blocked = true
			}
			continue
		}
		to, err := chesskit.Validate(cur, chesskit.SAN(san))
		if err == nil {
			m.IsLegal = true
			m.FenAfter = string(to)
			cur = to
			out = append(out, m)
			continue
		}
		// Illegal read: use legality as a prior. Rank the legal moves by similarity to
		// the recognized SAN; auto-adjust a confident single-character misread so the
		// game can continue (flagged Corrected for review), otherwise offer ranked
		// suggestions and stop here for the human to resolve.
		legal, _ := chesskit.LegalMovesSAN(cur)
		best, ranked, confident := matchLegal(san, toStrings(legal))
		m.Suggestions = ranked
		if confident {
			if to2, err2 := chesskit.Validate(cur, chesskit.SAN(best)); err2 == nil {
				m.SAN = best
				m.Corrected = true
				m.IsLegal = true
				m.FenAfter = string(to2)
				cur = to2
				out = append(out, m)
				continue
			}
		}
		m.IsLegal = false
		blocked = true
		out = append(out, m)
	}
	return out
}

func toStrings(s []chesskit.SAN) []string {
	out := make([]string, len(s))
	for i, v := range s {
		out[i] = string(v)
	}
	return out
}

// matchLegal ranks legal moves by edit distance to the recognized SAN. It returns the
// closest move, the top-ranked suggestions, and whether the match is confident enough to
// auto-apply: the best candidate is within one edit AND uniquely closest (no tie).
func matchLegal(cand string, legal []string) (best string, ranked []string, confident bool) {
	if len(legal) == 0 || cand == "" {
		return "", nil, false
	}
	type scored struct {
		san  string
		dist int
	}
	scoredAll := make([]scored, len(legal))
	for i, l := range legal {
		scoredAll[i] = scored{l, editDistance(cand, l)}
	}
	sort.Slice(scoredAll, func(i, j int) bool {
		if scoredAll[i].dist != scoredAll[j].dist {
			return scoredAll[i].dist < scoredAll[j].dist
		}
		return scoredAll[i].san < scoredAll[j].san
	})
	const maxSuggestions = 5
	for i := 0; i < len(scoredAll) && i < maxSuggestions; i++ {
		ranked = append(ranked, scoredAll[i].san)
	}
	best = scoredAll[0].san
	unique := len(scoredAll) == 1 || scoredAll[1].dist > scoredAll[0].dist
	confident = scoredAll[0].dist <= 1 && unique
	return best, ranked, confident
}

// editDistance is the Levenshtein distance between two short SAN strings.
func editDistance(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min3(curr[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}

func min3(a, b, c int) int {
	if b < a {
		a = b
	}
	if c < a {
		a = c
	}
	return a
}

// sideToMove reads the active-colour field of a FEN.
func sideToMove(fen string) string {
	parts := strings.Fields(fen)
	if len(parts) >= 2 && parts[1] == "b" {
		return SideBlack
	}
	return SideWhite
}
