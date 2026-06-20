package recognition

import (
	"errors"
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

// longAlgRe matches long algebraic notation (e.g. "Sf3-e5", "e2-e4", "e4:d5") which the
// recognition models were not trained on. Groups: piece, from-square, separator, to-square,
// optional promotion. Reduced to short SAN in GermanToSAN; chesskit then disambiguates.
var longAlgRe = regexp.MustCompile(`^([KQRBNDTLS]?)([a-h][1-8])([-x:])([a-h][1-8])(=?[QRBNqrbnDTLS])?$`)

// transPiece translates a German piece letter to English SAN; English letters pass through.
func transPiece(b byte) byte {
	if eng, ok := germanPiece[upper(b)]; ok {
		return eng
	}
	return upper(b)
}

// Per-ply recognition confidence. Confidence is a deterministic signal separate from legality:
// a move can be legal yet uncertain (auto-corrected, a guessed disambiguation), which the review
// UI surfaces as a "verify" (yellow) state. Models do not self-report reliably, so these come
// from signals we trust. A move at/above confThreshold renders green; below it renders yellow.
const (
	confThreshold     = 0.6
	confClean         = 0.90 // cleanly validated legal read
	confAutoCorrected = 0.40 // edit-distance auto-fix of a misread
	confAmbiguousPick = 0.30 // guessed disambiguation (the "knight problem")
	confIllegal       = 0.0  // illegal / illegible / blocked downstream
)

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
	s = strings.TrimSpace(s)

	// Long algebraic ("Sf3-e5", "e2-e4", "e4:d5") -> short SAN. Done before the capture-colon
	// strip below so the separator (which marks a capture) is still visible.
	if m := longAlgRe.FindStringSubmatch(s); m != nil {
		piece, from, sep, dest, promo := m[1], m[2], m[3], m[4], m[5]
		capture := sep == "x" || sep == ":"
		var out strings.Builder
		if piece != "" {
			out.WriteByte(transPiece(piece[0]))
			if capture {
				out.WriteByte('x')
			}
			out.WriteString(dest)
		} else {
			if capture {
				out.WriteByte(from[0]) // pawn capture keeps the origin file: e4:d5 -> exd5
				out.WriteByte('x')
			}
			out.WriteString(dest)
			if promo != "" {
				out.WriteString("=")
				out.WriteByte(transPiece(promo[len(promo)-1]))
			}
		}
		return out.String() + suffix
	}

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
// per-ply moves with legality, resulting FEN, and a recognition-confidence score. The first
// unrecoverable move and everything after it are marked illegal (fenAfter empty), mirroring
// the review-loop semantics. Confidence is independent of legality: a legal move may still be
// flagged for verification (auto-corrected misread, or a guessed disambiguation).
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
			Confidence:     confIllegal,
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
			m.Confidence = cleanConfidence(t.Confidence)
			cur = to
			out = append(out, m)
			continue
		}
		legal := toStrings(mustLegal(cur))
		// The "knight problem": an under-disambiguated read (e.g. bare "Nd2" when two knights
		// reach d2) is ambiguous, not wrong. Auto-pick a deterministic disambiguation so the
		// reviewer still sees the whole game, but flag it low-confidence with the alternatives.
		if errors.Is(err, chesskit.ErrAmbiguousMove) {
			choices := disambiguations(san, legal)
			if len(choices) >= 2 {
				sort.Strings(choices)
				if to2, err2 := chesskit.Validate(cur, chesskit.SAN(choices[0])); err2 == nil {
					m.SAN = choices[0]
					m.Corrected = true
					m.IsLegal = true
					m.FenAfter = string(to2)
					m.Suggestions = choices
					m.Confidence = confAmbiguousPick
					cur = to2
					out = append(out, m)
					continue
				}
			}
		}
		// Illegal read: use legality as a prior. Rank the legal moves by similarity to
		// the recognized SAN; auto-adjust a confident single-character misread so the
		// game can continue (flagged Corrected for review), otherwise offer ranked
		// suggestions and stop here for the human to resolve.
		best, ranked, confident := matchLegal(san, legal)
		m.Suggestions = ranked
		if confident {
			if to2, err2 := chesskit.Validate(cur, chesskit.SAN(best)); err2 == nil {
				m.SAN = best
				m.Corrected = true
				m.IsLegal = true
				m.FenAfter = string(to2)
				m.Confidence = confAutoCorrected
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

// cleanConfidence scores a cleanly-validated legal move. The model's own per-token score is
// used only when it is a meaningful low signal (a future per-cell uncertainty hook); the flat
// defaults today (0.5/0.9) carry no real information, so a clean read scores confClean.
func cleanConfidence(modelConf float64) float64 {
	if modelConf > 0 && modelConf < confThreshold {
		return modelConf
	}
	return confClean
}

func mustLegal(fen chesskit.FEN) []chesskit.SAN {
	legal, _ := chesskit.LegalMovesSAN(fen)
	return legal
}

// disambiguations returns the legal SANs that move the same piece to the same destination as
// san but from a different origin — i.e. san is under-disambiguated. Only piece moves (KQRBN)
// can be ambiguous this way; pawn captures already carry their origin file.
func disambiguations(san string, legal []string) []string {
	p, d, ok := sanPieceDest(san)
	if !ok {
		return nil
	}
	var out []string
	for _, l := range legal {
		if lp, ld, lok := sanPieceDest(l); lok && lp == p && ld == d {
			out = append(out, l)
		}
	}
	return out
}

// sanPieceDest extracts the piece letter and destination square from a piece move's SAN
// (origin disambiguation, capture mark, and check/mate suffix ignored). Reports ok=false for
// pawn moves, castling, and anything without a trailing square.
func sanPieceDest(s string) (piece byte, dest string, ok bool) {
	s = strings.TrimRight(s, "+#")
	if len(s) < 3 {
		return 0, "", false
	}
	switch s[0] {
	case 'K', 'Q', 'R', 'B', 'N':
	default:
		return 0, "", false
	}
	dest = s[len(s)-2:]
	if dest[0] < 'a' || dest[0] > 'h' || dest[1] < '1' || dest[1] > '8' {
		return 0, "", false
	}
	return s[0], dest, true
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
