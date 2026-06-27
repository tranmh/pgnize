//go:build integration

package store_test

import (
	"context"
	"testing"

	"github.com/tranmh/pgnize/internal/store"
)

func TestChatSessionAndMessages(t *testing.T) {
	st := newStore(t)
	ctx := context.Background()

	u, err := st.CreateUser(ctx, "Chat User", "chat@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	gid, err := st.CreateDraftGame(ctx, store.NewDraft{UserID: &u.ID, Source: "manual"})
	if err != nil {
		t.Fatal(err)
	}

	ply := 4
	sid, err := st.CreateChatSession(ctx, &u.ID, &gid, &ply, "fen-here", "de", "gemini:test")
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	cs, err := st.GetChatSession(ctx, sid)
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if cs.UserID == nil || *cs.UserID != u.ID {
		t.Errorf("session user = %v, want %s", cs.UserID, u.ID)
	}
	if cs.GameID == nil || *cs.GameID != gid {
		t.Errorf("session game = %v, want %s", cs.GameID, gid)
	}

	// Append a user turn (no trace) and a model turn (with a jsonb trace).
	if _, err := st.AppendChatMessage(ctx, sid, "user", "Bester Zug?", nil); err != nil {
		t.Fatal(err)
	}
	trace := []byte(`[{"name":"analyze_position","result":{"best_move":"e4"}}]`)
	seq, err := st.AppendChatMessage(ctx, sid, "model", "Der beste Zug ist e4.", trace)
	if err != nil {
		t.Fatal(err)
	}
	if seq != 1 {
		t.Errorf("second seq = %d, want 1", seq)
	}

	hist, err := st.ChatHistory(ctx, sid)
	if err != nil {
		t.Fatal(err)
	}
	if len(hist) != 2 {
		t.Fatalf("history len = %d, want 2", len(hist))
	}
	if hist[0].Seq != 0 || hist[0].Role != "user" {
		t.Errorf("first message = %+v", hist[0])
	}
	if hist[1].Role != "model" || len(hist[1].ToolTrace) == 0 {
		t.Errorf("second message should carry a tool trace: %+v", hist[1])
	}
}

func TestGetChatSessionNotFound(t *testing.T) {
	st := newStore(t)
	_, err := st.GetChatSession(context.Background(), "00000000-0000-0000-0000-000000000000")
	if err != store.ErrNotFound {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}
