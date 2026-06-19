package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// handleConvert (anonymous): store image + enqueue a job, no account required.
func (s *Server) handleConvert(w http.ResponseWriter, r *http.Request) {
	if !s.rateLimit(w, r, "convert:"+clientIP(r), 10, time.Hour) {
		return
	}
	uploadID, ok := s.storeImage(w, r, nil)
	if !ok {
		return
	}
	jobID, err := s.Store.CreateJob(r.Context(), uploadID, nil, s.Recognizer.Name())
	if err != nil {
		s.writeErr(w, http.StatusInternalServerError, "internal", "could not enqueue job")
		return
	}
	s.writeJSON(w, http.StatusAccepted, map[string]string{"jobId": jobID})
}

func (s *Server) handleConvertStatus(w http.ResponseWriter, r *http.Request) {
	j, err := s.Store.GetJob(r.Context(), chi.URLParam(r, "jobID"))
	if isNotFound(err) {
		s.writeErr(w, http.StatusNotFound, "not_found", "job not found")
		return
	}
	if err != nil {
		s.writeErr(w, http.StatusInternalServerError, "internal", "load failed")
		return
	}
	if j.UserID != nil { // anonymous endpoint only serves anonymous jobs
		s.writeErr(w, http.StatusNotFound, "not_found", "job not found")
		return
	}
	s.writeJobStatus(w, j)
}

func (s *Server) handleConvertGame(w http.ResponseWriter, r *http.Request) {
	j, err := s.Store.GetJob(r.Context(), chi.URLParam(r, "jobID"))
	if isNotFound(err) || (err == nil && (j.UserID != nil || j.GameID == nil)) {
		s.writeErr(w, http.StatusNotFound, "not_found", "game not ready")
		return
	}
	if err != nil {
		s.writeErr(w, http.StatusInternalServerError, "internal", "load failed")
		return
	}
	g, meta, err := s.Store.GetGame(r.Context(), *j.GameID)
	if err != nil {
		s.writeErr(w, http.StatusInternalServerError, "internal", "load failed")
		return
	}
	g.ImageURL = s.imageURL(meta.UploadID)
	s.writeJSON(w, http.StatusOK, g)
}

// handleConvertExport replays the reviewed anonymous game and returns PGN (no save).
func (s *Server) handleConvertExport(w http.ResponseWriter, r *http.Request) {
	var req saveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeErr(w, http.StatusBadRequest, "bad_request", "invalid json")
		return
	}
	startFEN := req.StartFEN
	if startFEN == "" {
		startFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	}
	vg, verr := buildVerifiedGame(req.Header, startFEN, req.Moves)
	if verr != nil {
		fa := vg.FailedAt
		s.writeJSON(w, http.StatusUnprocessableEntity, apiError{Error: "illegal_move", Message: verr.Error(), FailedAt: &fa})
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="game.pgn"`)
	_, _ = w.Write([]byte(vg.PGN))
}
