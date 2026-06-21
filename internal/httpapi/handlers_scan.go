package httpapi

import (
	"net/http"
	"time"
)

// handleScan (anonymous): store a board photo + enqueue a position-recognition job. It
// mirrors handleConvert but uses its own rate-limit bucket and the "position" job kind.
func (s *Server) handleScan(w http.ResponseWriter, r *http.Request) {
	if !s.rateLimit(w, r, "scan:"+clientIP(r), 10, time.Hour) {
		return
	}
	uploadID, ok := s.storeImage(w, r, nil)
	if !ok {
		return
	}
	backend, recName, ok := s.jobBackend(w, r)
	if !ok {
		return
	}
	jobID, err := s.Store.CreateJob(r.Context(), uploadID, nil, recName, backend, "position")
	if err != nil {
		s.writeErr(w, http.StatusInternalServerError, "internal", "could not enqueue job")
		return
	}
	s.writeJSON(w, http.StatusAccepted, map[string]string{"jobId": jobID})
}
