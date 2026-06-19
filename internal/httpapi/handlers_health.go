package httpapi

import "net/http"

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	if err := s.Store.Ping(r.Context()); err != nil {
		s.writeErr(w, http.StatusServiceUnavailable, "not_ready", err.Error())
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
