package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/tranmh/chesskit"
	"github.com/tranmh/pgnize/internal/coaching"
	"github.com/tranmh/pgnize/internal/domain"
)

// coachMoveRequest carries one position plus the engine output the browser already
// computed. gameId/ply are optional: when both are present AND the requester owns the
// game, the result is cached (anonymous callers omit them and get a stateless response).
type coachMoveRequest struct {
	GameID     string        `json:"gameId"`
	Ply        *int          `json:"ply"`
	FEN        string        `json:"fen"`
	Side       string        `json:"side"`
	PlayedSan  string        `json:"playedSan"`
	BestSan    string        `json:"bestSan"`
	BestLine   []string      `json:"bestLine"`
	EvalBefore coaching.Eval `json:"evalBefore"`
	EvalAfter  coaching.Eval `json:"evalAfter"`
	Quality    string        `json:"quality"`
	Lang       string        `json:"lang"`
}

type coachResponse struct {
	Text   string `json:"text"`
	Model  string `json:"model"`
	Lang   string `json:"lang"`
	Cached bool   `json:"cached"`
}

// coachLang normalizes the requested language; German is the default (German-first product).
func coachLang(l string) string {
	if l == "" {
		return coaching.LangDefault
	}
	return l
}

func (s *Server) handleCoachMove(w http.ResponseWriter, r *http.Request) {
	if !s.rateLimit(w, r, "coach:"+clientIP(r), 60, time.Hour) {
		return
	}
	var req coachMoveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeErr(w, http.StatusBadRequest, "bad_request", "invalid json")
		return
	}
	// The coach must only ever see legal positions: validate the FEN and both moves via
	// chesskit before sending anything to the model.
	fen, err := chesskit.NormalizeFEN(chesskit.FEN(req.FEN))
	if err != nil {
		s.writeErr(w, http.StatusBadRequest, "bad_request", "illegal fen")
		return
	}
	if req.PlayedSan == "" || req.BestSan == "" {
		s.writeErr(w, http.StatusBadRequest, "bad_request", "playedSan and bestSan are required")
		return
	}
	if _, err := chesskit.Validate(fen, chesskit.SAN(req.PlayedSan)); err != nil {
		s.writeErr(w, http.StatusBadRequest, "bad_request", "playedSan is not legal in this position")
		return
	}
	if _, err := chesskit.Validate(fen, chesskit.SAN(req.BestSan)); err != nil {
		s.writeErr(w, http.StatusBadRequest, "bad_request", "bestSan is not legal in this position")
		return
	}

	lang := coachLang(req.Lang)
	model := s.Coach.Name()
	cacheable := req.GameID != "" && req.Ply != nil && s.ownsGameID(r, req.GameID)

	if cacheable {
		if txt, err := s.Store.GetCoaching(r.Context(), req.GameID, *req.Ply, model, lang); err == nil {
			s.writeJSON(w, http.StatusOK, coachResponse{Text: txt, Model: model, Lang: lang, Cached: true})
			return
		}
	}

	out, err := s.Coach.CoachMove(r.Context(), coaching.MoveInput{
		FEN:        string(fen),
		Side:       req.Side,
		PlayedSAN:  req.PlayedSan,
		BestSAN:    req.BestSan,
		BestLine:   req.BestLine,
		EvalBefore: req.EvalBefore,
		EvalAfter:  req.EvalAfter,
		Quality:    req.Quality,
		Lang:       lang,
	})
	if err != nil {
		s.writeErr(w, http.StatusBadGateway, "coach_failed", "coaching is unavailable")
		return
	}
	if cacheable {
		_ = s.Store.UpsertCoaching(r.Context(), req.GameID, *req.Ply, model, lang, out.Text)
	}
	s.writeJSON(w, http.StatusOK, coachResponse{Text: out.Text, Model: out.Model, Lang: out.Lang, Cached: false})
}

type coachGameMove struct {
	San       string        `json:"san"`
	Side      string        `json:"side"`
	EvalAfter coaching.Eval `json:"evalAfter"`
	Quality   string        `json:"quality"`
}

type coachGameRequest struct {
	GameID   string          `json:"gameId"`
	StartFEN string          `json:"startFen"`
	Header   domain.Header   `json:"header"`
	Moves    []coachGameMove `json:"moves"`
	Lang     string          `json:"lang"`
}

// gameSummaryPly is the cache ply for the whole-game summary.
const gameSummaryPly = -1

func (s *Server) handleCoachGame(w http.ResponseWriter, r *http.Request) {
	if !s.rateLimit(w, r, "coach:"+clientIP(r), 60, time.Hour) {
		return
	}
	var req coachGameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeErr(w, http.StatusBadRequest, "bad_request", "invalid json")
		return
	}
	fen, err := chesskit.NormalizeFEN(chesskit.FEN(req.StartFEN))
	if err != nil {
		s.writeErr(w, http.StatusBadRequest, "bad_request", "illegal start fen")
		return
	}

	lang := coachLang(req.Lang)
	model := s.Coach.Name()
	cacheable := req.GameID != "" && s.ownsGameID(r, req.GameID)

	if cacheable {
		if txt, err := s.Store.GetCoaching(r.Context(), req.GameID, gameSummaryPly, model, lang); err == nil {
			s.writeJSON(w, http.StatusOK, coachResponse{Text: txt, Model: model, Lang: lang, Cached: true})
			return
		}
	}

	moves := make([]coaching.GameMove, len(req.Moves))
	for i, m := range req.Moves {
		moves[i] = coaching.GameMove{Ply: i + 1, Side: m.Side, SAN: m.San, EvalAfter: m.EvalAfter, Quality: m.Quality}
	}
	out, err := s.Coach.CoachGame(r.Context(), coaching.GameInput{
		StartFEN: string(fen),
		Header:   req.Header,
		Moves:    moves,
		Lang:     lang,
	})
	if err != nil {
		s.writeErr(w, http.StatusBadGateway, "coach_failed", "coaching is unavailable")
		return
	}
	if cacheable {
		_ = s.Store.UpsertCoaching(r.Context(), req.GameID, gameSummaryPly, model, lang, out.Text)
	}
	s.writeJSON(w, http.StatusOK, coachResponse{Text: out.Text, Model: out.Model, Lang: out.Lang, Cached: false})
}

// ownsGameID reports whether the current (logged-in) user owns gameID. Anonymous callers,
// missing games, and games owned by someone else all return false — so caching is confined
// to the owner's own library rows.
func (s *Server) ownsGameID(r *http.Request, gameID string) bool {
	owner, err := s.Store.GameOwner(r.Context(), gameID)
	if err != nil {
		return false
	}
	return s.ownsGame(r, owner)
}
