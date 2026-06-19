package httpapi

import (
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// handleImage streams a stored upload. Account-owned images require the owner;
// anonymous uploads are served to anyone holding the (random) upload id.
func (s *Server) handleImage(w http.ResponseWriter, r *http.Request) {
	up, err := s.Store.GetUpload(r.Context(), chi.URLParam(r, "uploadID"))
	if isNotFound(err) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		s.writeErr(w, http.StatusInternalServerError, "internal", "load failed")
		return
	}
	if up.UserID != nil {
		u := s.user(r)
		if u == nil || *up.UserID != u.ID {
			http.NotFound(w, r)
			return
		}
	}
	body, ct, err := s.Storage.Get(r.Context(), up.StorageKey)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer body.Close()
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "private, max-age=300")
	_, _ = io.Copy(w, body)
}
