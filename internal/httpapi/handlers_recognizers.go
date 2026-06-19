package httpapi

import (
	"net/http"

	"github.com/tranmh/pgnize/internal/recognition"
)

// handleRecognizers lists the recognition backends a client may select. Only advertised
// (configured/available) backends are returned, so unconfigured backends — e.g. Gemini
// without an API key — are hidden from the picker entirely.
func (s *Server) handleRecognizers(w http.ResponseWriter, _ *http.Request) {
	list := s.Recognizers.Advertised()
	if list == nil {
		list = []recognition.BackendInfo{}
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"recognizers": list,
		"default":     s.Recognizers.Default(),
	})
}

// jobBackend reads and validates the requested recognition backend from the (already
// parsed) form. An empty value means "use the server default". It returns the backend key
// and its resolved Recognizer.Name(); on an unavailable choice it writes a 400 and ok=false.
func (s *Server) jobBackend(w http.ResponseWriter, r *http.Request) (backend, recName string, ok bool) {
	backend = r.FormValue("backend")
	if !s.Recognizers.Available(backend) {
		s.writeErr(w, http.StatusBadRequest, "unknown_backend", "unknown or unavailable recognition backend")
		return "", "", false
	}
	return backend, s.Recognizers.Name(backend), true
}
