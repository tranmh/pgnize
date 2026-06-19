package auth

import (
	"context"
	"testing"

	"github.com/tranmh/pgnize/internal/domain"
)

func TestPasswordHashRoundTrip(t *testing.T) {
	h, err := HashPassword("s3cret-pw")
	if err != nil {
		t.Fatal(err)
	}
	if !CheckPassword(h, "s3cret-pw") {
		t.Fatal("correct password rejected")
	}
	if CheckPassword(h, "wrong") {
		t.Fatal("wrong password accepted")
	}
}

func TestSessionTokenHashStable(t *testing.T) {
	raw, hash, err := NewSessionToken()
	if err != nil {
		t.Fatal(err)
	}
	if raw == "" || hash == "" || raw == hash {
		t.Fatal("token/hash invalid")
	}
	if HashToken(raw) != hash {
		t.Fatal("HashToken not stable")
	}
}

func TestUserContext(t *testing.T) {
	ctx := context.Background()
	if UserFrom(ctx) != nil {
		t.Fatal("expected nil user")
	}
	u := &domain.User{ID: "abc"}
	ctx = WithUser(ctx, u)
	if got := UserFrom(ctx); got == nil || got.ID != "abc" {
		t.Fatal("user not retrieved from context")
	}
}
