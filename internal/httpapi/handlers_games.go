package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/tranmh/pgnize/internal/domain"
	"github.com/tranmh/pgnize/internal/store"
)

func (s *Server) imageURL(uploadID *string) string {
	if uploadID == nil {
		return ""
	}
	return "/api/images/" + *uploadID
}

// handleCreateManual creates an empty draft for manual entry.
func (s *Server) handleCreateManual(w http.ResponseWriter, r *http.Request) {
	u := s.user(r)
	id, err := s.Store.CreateDraftGame(r.Context(), store.NewDraft{
		UserID: &u.ID,
		Source: domain.SourceManual,
		Header: domain.Header{Result: "*"},
	})
	if err != nil {
		s.writeErr(w, http.StatusInternalServerError, "internal", "could not create draft")
		return
	}
	g, meta, err := s.Store.GetGame(r.Context(), id)
	if err != nil {
		s.writeErr(w, http.StatusInternalServerError, "internal", "reload failed")
		return
	}
	g.ImageURL = s.imageURL(meta.UploadID)
	s.writeJSON(w, http.StatusCreated, map[string]any{"game": g})
}

func (s *Server) handleGetGame(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	g, meta, err := s.Store.GetGame(r.Context(), id)
	if isNotFound(err) {
		s.writeErr(w, http.StatusNotFound, "not_found", "game not found")
		return
	}
	if err != nil {
		s.writeErr(w, http.StatusInternalServerError, "internal", "load failed")
		return
	}
	if !s.ownsGame(r, meta.OwnerID) {
		s.writeErr(w, http.StatusNotFound, "not_found", "game not found")
		return
	}
	g.ImageURL = s.imageURL(meta.UploadID)
	s.writeJSON(w, http.StatusOK, g)
}

type saveRequest struct {
	Header   domain.Header `json:"header"`
	Moves    []moveInput   `json:"moves"`
	StartFEN string        `json:"startFen"`
}

func (s *Server) handleSaveGame(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	u := s.user(r)
	g, meta, err := s.Store.GetGame(r.Context(), id)
	if isNotFound(err) || (err == nil && !s.ownsGame(r, meta.OwnerID)) {
		s.writeErr(w, http.StatusNotFound, "not_found", "game not found")
		return
	}
	if err != nil {
		s.writeErr(w, http.StatusInternalServerError, "internal", "load failed")
		return
	}
	var req saveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeErr(w, http.StatusBadRequest, "bad_request", "invalid json")
		return
	}
	if req.StartFEN == "" {
		req.StartFEN = g.StartFEN
	}
	vg, verr := buildVerifiedGame(req.Header, req.StartFEN, req.Moves)
	if verr != nil {
		fa := vg.FailedAt
		s.writeJSON(w, http.StatusUnprocessableEntity, apiError{
			Error: "illegal_move", Message: verr.Error(), FailedAt: &fa,
		})
		return
	}
	// Upsert players into the user's autocomplete pool.
	whiteID, _ := s.Store.UpsertPlayer(r.Context(), u.ID, req.Header.White)
	blackID, _ := s.Store.UpsertPlayer(r.Context(), u.ID, req.Header.Black)
	wp, bp := nilIfEmpty(whiteID), nilIfEmpty(blackID)

	if err := s.Store.SaveGame(r.Context(), id, req.Header, req.StartFEN, vg.Moves, vg.PGN, wp, bp); err != nil {
		s.writeErr(w, http.StatusInternalServerError, "internal", "save failed")
		return
	}
	s.recordFeedback(r, id, u.ID, req)

	saved, meta2, err := s.Store.GetGame(r.Context(), id)
	if err != nil {
		s.writeErr(w, http.StatusInternalServerError, "internal", "reload failed")
		return
	}
	saved.ImageURL = s.imageURL(meta2.UploadID)
	s.writeJSON(w, http.StatusOK, map[string]any{"game": saved})
}

func (s *Server) handleDeleteGame(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	_, meta, err := s.Store.GetGame(r.Context(), id)
	if isNotFound(err) || (err == nil && !s.ownsGame(r, meta.OwnerID)) {
		s.writeErr(w, http.StatusNotFound, "not_found", "game not found")
		return
	}
	if err != nil {
		s.writeErr(w, http.StatusInternalServerError, "internal", "load failed")
		return
	}
	if err := s.Store.DeleteGame(r.Context(), id); err != nil {
		s.writeErr(w, http.StatusInternalServerError, "internal", "delete failed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// recordFeedback best-effort captures the model-before vs human-after pair for training.
func (s *Server) recordFeedback(r *http.Request, gameID, userID string, req saveRequest) {
	recognizer, beforeJSON, uploadID, err := s.Store.JobRawByGame(r.Context(), gameID)
	if err != nil || beforeJSON == "" {
		return // manual entry or no recognition job: nothing to learn from
	}
	sans := make([]string, len(req.Moves))
	for i, m := range req.Moves {
		sans[i] = m.San
	}
	after, _ := json.Marshal(map[string]any{"header": req.Header, "sans": sans})
	_ = s.Store.CreateFeedback(r.Context(), uploadID, gameID, userID, recognizer, beforeJSON, string(after), len(req.Moves))
}

func (s *Server) ownsGame(r *http.Request, ownerID *string) bool {
	u := s.user(r)
	if u == nil || ownerID == nil {
		return false
	}
	return *ownerID == u.ID
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
