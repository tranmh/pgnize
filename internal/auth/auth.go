// Package auth provides password hashing, opaque session tokens, and request context.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"

	"github.com/tranmh/pgnize/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

// CookieName is the session cookie name.
const CookieName = "pgnize_session"

// HashPassword returns a bcrypt hash.
func HashPassword(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(b), err
}

// CheckPassword verifies a password against a bcrypt hash.
func CheckPassword(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

// NewSessionToken returns a random opaque token and its storage hash.
// The raw token goes in the cookie; only the hash is persisted.
func NewSessionToken() (raw, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	raw = hex.EncodeToString(b)
	return raw, HashToken(raw), nil
}

// HashToken hashes a raw session token for storage/lookup.
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

type ctxKey struct{}

// WithUser stores the authenticated user in the context.
func WithUser(ctx context.Context, u *domain.User) context.Context {
	return context.WithValue(ctx, ctxKey{}, u)
}

// UserFrom returns the authenticated user, or nil if anonymous.
func UserFrom(ctx context.Context) *domain.User {
	u, _ := ctx.Value(ctxKey{}).(*domain.User)
	return u
}
