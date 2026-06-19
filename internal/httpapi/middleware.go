package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/tranmh/pgnize/internal/auth"
	"github.com/tranmh/pgnize/internal/store"
)

// authContext attaches the authenticated user (if any) to the request context.
func (s *Server) authContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(auth.CookieName)
		if err == nil && c.Value != "" {
			u, err := s.Store.UserBySessionToken(r.Context(), auth.HashToken(c.Value))
			if err == nil {
				r = r.WithContext(auth.WithUser(r.Context(), &u))
			}
		}
		next.ServeHTTP(w, r)
	})
}

// requireAuth rejects anonymous requests with 401.
func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.user(r) == nil {
			s.writeErr(w, http.StatusUnauthorized, "unauthorized", "login required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// rateLimit enforces a per-key window; on exceed it writes 429 and returns false.
func (s *Server) rateLimit(w http.ResponseWriter, r *http.Request, key string, limit int, window time.Duration) bool {
	ok, err := s.Store.ConsumeRateLimit(r.Context(), key, limit, window)
	if err != nil {
		// Fail open on limiter errors, but never crash the request.
		return true
	}
	if !ok {
		s.writeErr(w, http.StatusTooManyRequests, "rate_limited", "too many requests, slow down")
		return false
	}
	return true
}

func isNotFound(err error) bool { return errors.Is(err, store.ErrNotFound) }
