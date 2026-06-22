// Package httpapi wires the REST API (chi router, session-cookie auth).
package httpapi

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/tranmh/pgnize/internal/auth"
	"github.com/tranmh/pgnize/internal/coaching"
	"github.com/tranmh/pgnize/internal/config"
	"github.com/tranmh/pgnize/internal/domain"
	"github.com/tranmh/pgnize/internal/recognition"
	"github.com/tranmh/pgnize/internal/storage"
	"github.com/tranmh/pgnize/internal/store"
)

// Server holds API dependencies.
type Server struct {
	Cfg         config.Config
	Store       *store.Store
	Storage     storage.Storage
	Recognizers *recognition.Registry
	Coach       coaching.Coach
}

// Routes builds the HTTP handler.
func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(s.authContext) // attach user if a valid session cookie is present

	r.Get("/healthz", s.handleHealthz)
	r.Get("/readyz", s.handleReadyz)

	r.Route("/api", func(r chi.Router) {
		r.Post("/auth/register", s.handleRegister)
		r.Post("/auth/login", s.handleLogin)
		r.Post("/auth/logout", s.handleLogout)
		r.Get("/auth/me", s.handleMe)

		// Available recognition backends (public; the anonymous convert flow needs it).
		r.Get("/recognizers", s.handleRecognizers)

		// Anonymous convert (no auth required).
		r.Post("/convert", s.handleConvert)
		r.Get("/convert/{jobID}", s.handleConvertStatus)
		r.Get("/convert/{jobID}/game", s.handleConvertGame)
		r.Post("/convert/{jobID}/export", s.handleConvertExport)

		// Anonymous board-photo → position scan (no auth required). Status/game/export
		// reuse the convert handlers (they are job-kind agnostic).
		r.Post("/scan", s.handleScan)
		r.Get("/scan/{jobID}", s.handleConvertStatus)
		r.Get("/scan/{jobID}/game", s.handleConvertGame)
		r.Post("/scan/{jobID}/export", s.handleConvertExport)

		// Anonymous direct inputs that bypass photo recognition (no auth required).
		// They return the verified draft inline; when the requester is logged in the
		// draft is also persisted to their library.
		r.Post("/positions", s.handlePasteFEN)
		r.Post("/import", s.handleImport)

		// Engine→prose coaching (public; gameId optional — caches only for the owner).
		r.Post("/coach/move", s.handleCoachMove)
		r.Post("/coach/game", s.handleCoachGame)
		r.Post("/coach/position", s.handleCoachPosition)

		// Image streaming (authorized per object).
		r.Get("/images/{uploadID}", s.handleImage)

		// Account-only surface.
		r.Group(func(r chi.Router) {
			r.Use(s.requireAuth)
			r.Post("/uploads", s.handleUpload)
			r.Get("/jobs/{jobID}", s.handleJobStatus)
			r.Post("/games", s.handleCreateManual)
			r.Get("/games", s.handleListGames)
			r.Post("/games/export", s.handleExportBundle)
			r.Get("/games/{id}", s.handleGetGame)
			r.Patch("/games/{id}", s.handleSaveGame)
			r.Delete("/games/{id}", s.handleDeleteGame)
			r.Get("/games/{id}/pgn", s.handleGamePGN)
			r.Get("/players", s.handlePlayers)
		})
	})
	return r
}

// ---- helpers ----

func (s *Server) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

type apiError struct {
	Error    string `json:"error"`
	Message  string `json:"message"`
	FailedAt *int   `json:"failedAt,omitempty"`
}

func (s *Server) writeErr(w http.ResponseWriter, status int, code, msg string) {
	s.writeJSON(w, status, apiError{Error: code, Message: msg})
}

func (s *Server) user(r *http.Request) *domain.User { return auth.UserFrom(r.Context()) }

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.TrimSpace(strings.Split(xff, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
