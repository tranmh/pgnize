package chat

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tranmh/pgnize/internal/engine"
)

func TestFakeChatterAnalyzeLoop(t *testing.T) {
	c := NewFake(engine.NewFake())
	reply, err := c.Respond(context.Background(), nil, "What is the best move?", Context{FEN: startFEN, Side: "white", Lang: "en"})
	if err != nil {
		t.Fatal(err)
	}
	if reply.Text == "" {
		t.Error("expected non-empty reply")
	}
	if len(reply.Calls) != 1 || reply.Calls[0].Name != "analyze_position" {
		t.Fatalf("expected one analyze_position call, got %+v", reply.Calls)
	}
	if !strings.Contains(reply.Text, "e4") {
		t.Errorf("expected best move e4 in prose, got %q", reply.Text)
	}
}

func TestFakeChatterMateLoop(t *testing.T) {
	f := engine.NewFake()
	mate := 2
	f.Seed(startFEN, engine.Analysis{FEN: startFEN, Lines: []engine.Line{{Mate: &mate, PV: []string{"d1h5"}, BestMove: "d1h5"}}})
	c := NewFake(f)
	reply, err := c.Respond(context.Background(), nil, "Is there a mate combination?", Context{FEN: startFEN, Side: "white", Lang: "en"})
	if err != nil {
		t.Fatal(err)
	}
	if reply.Calls[0].Name != "find_mate" {
		t.Fatalf("expected find_mate call, got %s", reply.Calls[0].Name)
	}
	if !strings.Contains(reply.Text, "mate in 2") {
		t.Errorf("expected 'mate in 2' in prose, got %q", reply.Text)
	}
}

// TestGeminiChatterToolLoop drives the real multi-turn loop against a mock Gemini server:
// the model first returns a functionCall, then (after the tool result) a text answer.
func TestGeminiChatterToolLoop(t *testing.T) {
	var bodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(buf))
		w.Header().Set("Content-Type", "application/json")
		if len(bodies) == 1 {
			// First call: ask to analyze the position (omit fen -> exercises fallback).
			_, _ = w.Write([]byte(`{"candidates":[{"content":{"role":"model","parts":[{"functionCall":{"name":"analyze_position","args":{}}}]}}]}`))
			return
		}
		// Second call: final text answer.
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"role":"model","parts":[{"text":"Der beste Zug ist e4."}]}}]}`))
	}))
	defer srv.Close()

	c := NewGemini(srv.URL, "test-model", "key", engine.NewFake())
	reply, err := c.Respond(context.Background(), nil, "Bester Zug?", Context{FEN: startFEN, Side: "white", Lang: "de"})
	if err != nil {
		t.Fatal(err)
	}
	if len(bodies) != 2 {
		t.Fatalf("expected 2 LLM round-trips, got %d", len(bodies))
	}
	if len(reply.Calls) != 1 || reply.Calls[0].Name != "analyze_position" {
		t.Fatalf("expected one analyze_position tool call, got %+v", reply.Calls)
	}
	if reply.Calls[0].Result["best_move"] == nil {
		t.Errorf("tool result missing best_move: %v", reply.Calls[0].Result)
	}
	if reply.Text != "Der beste Zug ist e4." {
		t.Errorf("reply = %q", reply.Text)
	}
	// The second request must carry the functionResponse turn back to the model.
	if !strings.Contains(bodies[1], "functionResponse") {
		t.Errorf("second request should include functionResponse, got: %s", bodies[1])
	}
}

func TestGeminiChatterFirstAnswerNoTool(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"role":"model","parts":[{"text":"Hallo!"}]}}]}`))
	}))
	defer srv.Close()
	c := NewGemini(srv.URL, "m", "k", engine.NewFake())
	reply, err := c.Respond(context.Background(), nil, "Hi", Context{FEN: startFEN, Lang: "de"})
	if err != nil {
		t.Fatal(err)
	}
	if reply.Text != "Hallo!" || len(reply.Calls) != 0 {
		t.Errorf("got text=%q calls=%d", reply.Text, len(reply.Calls))
	}
}

// Ensure the request actually advertises the four tools.
func TestGeminiRequestAdvertisesTools(t *testing.T) {
	var captured geminiChatRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&captured)
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"ok"}]}}]}`))
	}))
	defer srv.Close()
	c := NewGemini(srv.URL, "m", "k", engine.NewFake())
	_, _ = c.Respond(context.Background(), nil, "Hi", Context{FEN: startFEN})
	if len(captured.Tools) != 1 || len(captured.Tools[0].FunctionDeclarations) != 4 {
		t.Fatalf("expected 4 tool declarations, got %+v", captured.Tools)
	}
	if captured.SystemInstruction == nil {
		t.Error("expected a system instruction")
	}
}
