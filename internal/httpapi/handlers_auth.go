package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/tranmh/pgnize/internal/auth"
	"github.com/tranmh/pgnize/internal/domain"
	"github.com/tranmh/pgnize/internal/store"
)

const sessionTTL = 30 * 24 * time.Hour

type userDTO struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func toUserDTO(u domain.User) userDTO { return userDTO{ID: u.ID, Name: u.Name, Email: u.Email} }

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if !s.rateLimit(w, r, "register:"+clientIP(r), 5, 15*time.Minute) {
		return
	}
	var req struct{ Name, Email, Password string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeErr(w, http.StatusBadRequest, "bad_request", "invalid json")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Email == "" || len(req.Password) < 8 || strings.TrimSpace(req.Name) == "" {
		s.writeErr(w, http.StatusBadRequest, "invalid_input", "name, email and an 8+ char password are required")
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		s.writeErr(w, http.StatusInternalServerError, "internal", "hash failure")
		return
	}
	u, err := s.Store.CreateUser(r.Context(), strings.TrimSpace(req.Name), req.Email, hash)
	if err == store.ErrEmailTaken {
		s.writeErr(w, http.StatusConflict, "email_taken", "email already registered")
		return
	}
	if err != nil {
		s.writeErr(w, http.StatusInternalServerError, "internal", "could not create user")
		return
	}
	if err := s.startSession(w, r, u); err != nil {
		s.writeErr(w, http.StatusInternalServerError, "internal", "session failure")
		return
	}
	s.writeJSON(w, http.StatusCreated, map[string]any{"user": toUserDTO(u)})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct{ Email, Password string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeErr(w, http.StatusBadRequest, "bad_request", "invalid json")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if !s.rateLimit(w, r, "login:"+clientIP(r)+":"+req.Email, 10, 15*time.Minute) {
		return
	}
	u, err := s.Store.GetUserByEmail(r.Context(), req.Email)
	if err != nil || !auth.CheckPassword(u.PasswordHash, req.Password) {
		s.writeErr(w, http.StatusUnauthorized, "invalid_credentials", "wrong email or password")
		return
	}
	if err := s.startSession(w, r, u); err != nil {
		s.writeErr(w, http.StatusInternalServerError, "internal", "session failure")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"user": toUserDTO(u)})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(auth.CookieName); err == nil && c.Value != "" {
		_ = s.Store.DeleteSession(r.Context(), auth.HashToken(c.Value))
	}
	http.SetCookie(w, &http.Cookie{
		Name: auth.CookieName, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, SameSite: http.SameSiteLaxMode,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	u := s.user(r)
	if u == nil {
		s.writeErr(w, http.StatusUnauthorized, "unauthorized", "not logged in")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"user": toUserDTO(*u)})
}

func (s *Server) startSession(w http.ResponseWriter, r *http.Request, u domain.User) error {
	raw, hash, err := auth.NewSessionToken()
	if err != nil {
		return err
	}
	exp := time.Now().Add(sessionTTL)
	if err := s.Store.CreateSession(r.Context(), u.ID, hash, clientIP(r), r.UserAgent(), exp); err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name: auth.CookieName, Value: raw, Path: "/", Expires: exp,
		HttpOnly: true, SameSite: http.SameSiteLaxMode,
		Secure: strings.HasPrefix(s.Cfg.PublicBase, "https://"),
	})
	return nil
}
