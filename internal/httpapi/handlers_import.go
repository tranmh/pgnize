package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/tranmh/chesskit"
	"github.com/tranmh/pgnize/internal/domain"
	"github.com/tranmh/pgnize/internal/store"
)

const (
	lichessFetchTimeout = 10 * time.Second
	lichessMaxBytes     = 5 << 20 // 5 MiB cap on a fetched PGN
)

var lichessIDRe = regexp.MustCompile(`^[A-Za-z0-9]+$`)

// handlePasteFEN accepts a raw FEN, validates it via chesskit, and returns a position
// draft (no moves). Anonymous → inline draft; logged-in → also persisted to the library.
func (s *Server) handlePasteFEN(w http.ResponseWriter, r *http.Request) {
	if !s.rateLimit(w, r, "input:"+clientIP(r), 60, time.Hour) {
		return
	}
	var req struct {
		FEN string `json:"fen"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeErr(w, http.StatusBadRequest, "bad_request", "invalid json")
		return
	}
	fen, err := chesskit.NormalizeFEN(chesskit.FEN(strings.TrimSpace(req.FEN)))
	if err != nil {
		s.writeErr(w, http.StatusBadRequest, "bad_request", "illegal fen")
		return
	}
	draft, err := s.persistOrInlineDraft(r, domain.SourceManual, domain.Header{Result: "*"}, string(fen), nil)
	if err != nil {
		s.writeErr(w, http.StatusInternalServerError, "internal", "could not create draft")
		return
	}
	s.writeJSON(w, http.StatusOK, draft)
}

type importRequest struct {
	PGN string `json:"pgn"`
	URL string `json:"url"`
}

// handleImport ingests a raw PGN paste or a Lichess study/game URL and returns one verified
// draft per game. The server re-validates every move via chesskit, so an imported game can
// never carry an illegal move into the review UI.
func (s *Server) handleImport(w http.ResponseWriter, r *http.Request) {
	if !s.rateLimit(w, r, "input:"+clientIP(r), 60, time.Hour) {
		return
	}
	var req importRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeErr(w, http.StatusBadRequest, "bad_request", "invalid json")
		return
	}

	var pgnText string
	switch {
	case strings.TrimSpace(req.PGN) != "":
		pgnText = req.PGN
	case strings.TrimSpace(req.URL) != "":
		text, err := s.fetchLichessPGN(r.Context(), req.URL)
		if err != nil {
			s.writeErr(w, http.StatusBadRequest, "bad_request", "could not import from that URL: "+err.Error())
			return
		}
		pgnText = text
	default:
		s.writeErr(w, http.StatusBadRequest, "bad_request", "provide pgn or url")
		return
	}

	games, err := chesskit.ParsePGN(pgnText)
	if err != nil || len(games) == 0 {
		s.writeErr(w, http.StatusUnprocessableEntity, "no_games", "no games found in the PGN")
		return
	}

	drafts := make([]domain.GameDraft, 0, len(games))
	for _, g := range games {
		draft, derr := s.draftFromGame(r, g)
		if derr != nil {
			continue // skip a game we cannot replay legally rather than fail the whole import
		}
		drafts = append(drafts, draft)
	}
	if len(drafts) == 0 {
		s.writeErr(w, http.StatusUnprocessableEntity, "no_games", "no legal games found in the PGN")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"games": drafts})
}

// draftFromGame maps a parsed chesskit.Game to a verified draft (anonymous inline or persisted).
func (s *Server) draftFromGame(r *http.Request, g chesskit.Game) (domain.GameDraft, error) {
	h := fromChessHeader(g.Header)

	startFEN := strings.TrimSpace(string(g.StartFEN))
	if startFEN == "" {
		startFEN = string(chesskit.StartingFEN())
	} else if nf, err := chesskit.NormalizeFEN(chesskit.FEN(startFEN)); err == nil {
		startFEN = string(nf)
	} else {
		return domain.GameDraft{}, err
	}

	in := make([]moveInput, len(g.Moves))
	for i, m := range g.Moves {
		in[i] = moveInput{Ply: i + 1, San: string(m.SAN), ClockSec: m.ClockSec}
	}
	vg, err := buildVerifiedGame(h, startFEN, in)
	if err != nil {
		return domain.GameDraft{}, err
	}
	return s.persistOrInlineDraft(r, domain.SourceManual, h, startFEN, vg.Moves)
}

// persistOrInlineDraft returns a draft for the given content. For logged-in users it
// persists a games row (so it appears in their library and coaching can be cached); for
// anonymous users it returns the draft inline with no DB row.
func (s *Server) persistOrInlineDraft(r *http.Request, source string, h domain.Header, startFEN string, moves []domain.Move) (domain.GameDraft, error) {
	if h.Result == "" {
		h.Result = "*"
	}
	if moves == nil {
		moves = []domain.Move{}
	}
	u := s.user(r)
	if u == nil {
		return domain.GameDraft{
			Source:     source,
			Status:     domain.StatusDraft,
			Header:     h,
			StartFEN:   startFEN,
			Moves:      moves,
			Confidence: 1.0,
		}, nil
	}
	id, err := s.Store.CreateDraftGame(r.Context(), store.NewDraft{
		UserID:     &u.ID,
		Source:     source,
		Header:     h,
		StartFEN:   startFEN,
		Confidence: 1.0,
		Moves:      moves,
	})
	if err != nil {
		return domain.GameDraft{}, err
	}
	g, meta, err := s.Store.GetGame(r.Context(), id)
	if err != nil {
		return domain.GameDraft{}, err
	}
	g.ImageURL = s.imageURL(meta.UploadID)
	return g, nil
}

func fromChessHeader(h chesskit.Header) domain.Header {
	return domain.Header{
		White:  h.White,
		Black:  h.Black,
		Event:  h.Event,
		Site:   h.Site,
		Date:   h.Date,
		Round:  h.Round,
		Board:  h.Board,
		Result: string(h.Result),
	}
}

// lichessPGNEndpoint maps a public Lichess study/game URL to its PGN export endpoint.
// It is pure (no network) so the SSRF allowlist is unit-testable: only https lichess.org
// URLs with alphanumeric ids are accepted; everything else is rejected.
func lichessPGNEndpoint(rawURL string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return "", fmt.Errorf("invalid url")
	}
	if u.Scheme != "https" {
		return "", fmt.Errorf("only https URLs are allowed")
	}
	if u.Hostname() != "lichess.org" {
		return "", fmt.Errorf("only lichess.org URLs are allowed")
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		return "", fmt.Errorf("unrecognized lichess URL")
	}
	if parts[0] == "study" {
		if len(parts) < 2 || !lichessIDRe.MatchString(parts[1]) {
			return "", fmt.Errorf("invalid study id")
		}
		if len(parts) >= 3 && lichessIDRe.MatchString(parts[2]) {
			return "https://lichess.org/api/study/" + parts[1] + "/" + parts[2] + ".pgn", nil
		}
		return "https://lichess.org/api/study/" + parts[1] + ".pgn", nil
	}
	// Otherwise treat the first path segment as a game id (optionally /white|/black follows).
	if !lichessIDRe.MatchString(parts[0]) {
		return "", fmt.Errorf("invalid game id")
	}
	return "https://lichess.org/game/export/" + parts[0], nil
}

// fetchLichessPGN fetches a study/game PGN from lichess.org with an SSRF allowlist, a hard
// timeout, redirect re-validation, and a response-size cap.
func (s *Server) fetchLichessPGN(ctx context.Context, rawURL string) (string, error) {
	endpoint, err := lichessPGNEndpoint(rawURL)
	if err != nil {
		return "", err
	}
	client := &http.Client{
		Timeout: lichessFetchTimeout,
		CheckRedirect: func(req *http.Request, _ []*http.Request) error {
			if req.URL.Scheme != "https" || req.URL.Hostname() != "lichess.org" {
				return errors.New("redirect to a disallowed host")
			}
			return nil
		},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/x-chess-pgn")
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("lichess returned status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, lichessMaxBytes))
	if err != nil {
		return "", fmt.Errorf("read failed")
	}
	if len(body) == 0 {
		return "", fmt.Errorf("empty response")
	}
	return string(body), nil
}
