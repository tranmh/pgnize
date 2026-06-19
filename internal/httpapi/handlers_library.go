package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/tranmh/pgnize/internal/domain"
	"github.com/tranmh/pgnize/internal/store"
)

func (s *Server) handleListGames(w http.ResponseWriter, r *http.Request) {
	u := s.user(r)
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	pageSize, _ := strconv.Atoi(q.Get("pageSize"))
	games, total, err := s.Store.ListGames(r.Context(), u.ID, store.GameFilter{
		Q: q.Get("q"), Player: q.Get("player"), Event: q.Get("event"),
		From: q.Get("from"), To: q.Get("to"), Page: page, PageSize: pageSize,
	})
	if err != nil {
		s.writeErr(w, http.StatusInternalServerError, "internal", "list failed")
		return
	}
	if games == nil {
		games = []domain.GameSummary{}
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"games": games, "total": total, "page": page, "pageSize": pageSize,
	})
}

func (s *Server) handleGamePGN(w http.ResponseWriter, r *http.Request) {
	u := s.user(r)
	pgn, err := s.Store.GamePGN(r.Context(), u.ID, chi.URLParam(r, "id"))
	if isNotFound(err) {
		s.writeErr(w, http.StatusNotFound, "not_found", "game not found or not saved")
		return
	}
	if err != nil {
		s.writeErr(w, http.StatusInternalServerError, "internal", "load failed")
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="game.pgn"`)
	_, _ = w.Write([]byte(pgn))
}

func (s *Server) handleExportBundle(w http.ResponseWriter, r *http.Request) {
	u := s.user(r)
	var req struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.IDs) == 0 {
		s.writeErr(w, http.StatusBadRequest, "bad_request", "expected non-empty ids")
		return
	}
	pgns, err := s.Store.GamesPGN(r.Context(), u.ID, req.IDs)
	if err != nil {
		s.writeErr(w, http.StatusInternalServerError, "internal", "export failed")
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="games.pgn"`)
	_, _ = w.Write([]byte(strings.Join(pgns, "\n\n")))
}

func (s *Server) handlePlayers(w http.ResponseWriter, r *http.Request) {
	u := s.user(r)
	players, err := s.Store.SearchPlayers(r.Context(), u.ID, r.URL.Query().Get("q"), 10)
	if err != nil {
		s.writeErr(w, http.StatusInternalServerError, "internal", "search failed")
		return
	}
	if players == nil {
		players = []domain.Player{}
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"players": players})
}
