//go:build integration

package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func (h *harness) chatSessionCount(t *testing.T) int {
	t.Helper()
	var n int
	if err := h.st.Pool.QueryRow(context.Background(), `SELECT count(*) FROM chat_sessions`).Scan(&n); err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	return n
}

func (h *harness) chatMessageCount(t *testing.T) int {
	t.Helper()
	var n int
	if err := h.st.Pool.QueryRow(context.Background(), `SELECT count(*) FROM chat_messages`).Scan(&n); err != nil {
		t.Fatalf("count messages: %v", err)
	}
	return n
}

func TestChatAnonymousNoPersistence(t *testing.T) {
	h := setup(t)
	resp, body := h.json(t, "POST", "/api/coach/chat", map[string]any{
		"fen": startFEN, "side": "white", "text": "What is the best move?", "lang": "en",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("chat %d: %s", resp.StatusCode, body)
	}
	var cr struct {
		ConversationID string `json:"conversationId"`
		Reply          string `json:"reply"`
	}
	json.Unmarshal(body, &cr)
	if cr.Reply == "" {
		t.Error("expected a reply")
	}
	if cr.ConversationID != "" {
		t.Error("anonymous chat must not return a conversationId")
	}
	if n := h.chatSessionCount(t); n != 0 {
		t.Fatalf("anonymous chat must not persist a session, got %d", n)
	}
}

func TestChatRegisteredPersistsAndContinues(t *testing.T) {
	h := setup(t)
	h.register(t, "Dana", "dana@example.com")

	// First turn: creates a session + two messages.
	resp, body := h.json(t, "POST", "/api/coach/chat", map[string]any{
		"fen": startFEN, "side": "white", "text": "Best move?", "lang": "en",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("turn 1 %d: %s", resp.StatusCode, body)
	}
	var first struct {
		ConversationID string `json:"conversationId"`
	}
	json.Unmarshal(body, &first)
	if first.ConversationID == "" {
		t.Fatalf("registered chat should return a conversationId: %s", body)
	}
	if n := h.chatSessionCount(t); n != 1 {
		t.Fatalf("want 1 session, got %d", n)
	}
	if n := h.chatMessageCount(t); n != 2 {
		t.Fatalf("want 2 messages after turn 1, got %d", n)
	}

	// Second turn continues the same conversation.
	resp, body = h.json(t, "POST", "/api/coach/chat", map[string]any{
		"conversationId": first.ConversationID,
		"fen":            startFEN, "side": "white", "text": "And why?", "lang": "en",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("turn 2 %d: %s", resp.StatusCode, body)
	}
	if n := h.chatSessionCount(t); n != 1 {
		t.Fatalf("second turn must reuse the session, got %d sessions", n)
	}
	if n := h.chatMessageCount(t); n != 4 {
		t.Fatalf("want 4 messages after turn 2, got %d", n)
	}

	// History GET requires a gameId; without one it returns empty (this session has no game).
	resp, body = h.do(t, "GET", "/api/coach/chat/history?gameId=", "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("history %d: %s", resp.StatusCode, body)
	}
}

func TestChatForeignConversationForbidden(t *testing.T) {
	h := setup(t)
	h.register(t, "Eve", "eve@example.com")
	_, body := h.json(t, "POST", "/api/coach/chat", map[string]any{
		"fen": startFEN, "side": "white", "text": "Best move?",
	})
	var first struct {
		ConversationID string `json:"conversationId"`
	}
	json.Unmarshal(body, &first)

	// Log out and register a different user, then try to continue Eve's conversation.
	h.json(t, "POST", "/api/auth/logout", map[string]any{})
	h.register(t, "Frank", "frank@example.com")
	resp, _ := h.json(t, "POST", "/api/coach/chat", map[string]any{
		"conversationId": first.ConversationID,
		"fen":            startFEN, "side": "white", "text": "steal",
	})
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("foreign conversation status %d, want 403", resp.StatusCode)
	}
}

func TestChatHistoryReHydratesByGame(t *testing.T) {
	h := setup(t)
	h.register(t, "Gina", "gina@example.com")

	// Persisted position draft gives us a game id.
	_, body := h.json(t, "POST", "/api/positions", map[string]string{"fen": startFEN})
	var draft struct {
		ID string `json:"id"`
	}
	json.Unmarshal(body, &draft)
	if draft.ID == "" {
		t.Fatalf("expected a draft id: %s", body)
	}

	resp, body := h.json(t, "POST", "/api/coach/chat", map[string]any{
		"gameId": draft.ID, "fen": startFEN, "side": "white", "text": "Best move?",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("chat %d: %s", resp.StatusCode, body)
	}

	resp, body = h.do(t, "GET", "/api/coach/chat/history?gameId="+draft.ID, "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("history %d: %s", resp.StatusCode, body)
	}
	var hist struct {
		ConversationID string `json:"conversationId"`
		Messages       []struct {
			Role string `json:"role"`
			Text string `json:"text"`
		} `json:"messages"`
	}
	json.Unmarshal(body, &hist)
	if len(hist.Messages) != 2 {
		t.Fatalf("expected 2 messages re-hydrated, got %d: %s", len(hist.Messages), body)
	}
	if hist.Messages[0].Role != "user" || hist.Messages[1].Role != "coach" {
		t.Errorf("unexpected roles: %+v", hist.Messages)
	}
}
