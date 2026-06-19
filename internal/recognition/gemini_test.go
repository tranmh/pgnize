package recognition

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGeminiName(t *testing.T) {
	g := NewGemini("https://x", "gemini-2.5-flash", "key")
	if g.Name() != "gemini:gemini-2.5-flash" {
		t.Fatalf("Name() = %q", g.Name())
	}
}

func TestGeminiRecognizeParsesCandidate(t *testing.T) {
	var gotPath, gotKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotKey = r.Header.Get("X-Goog-Api-Key")
		body, _ := io.ReadAll(r.Body)
		// The request must carry the inline image and structured-output config.
		if !strings.Contains(string(body), "inline_data") {
			t.Errorf("request missing inline_data: %s", body)
		}
		if !strings.Contains(string(body), "responseSchema") {
			t.Errorf("request missing responseSchema: %s", body)
		}
		// Candidate text is the model's JSON answer.
		inner := `{"header":{"white":"Doe, John","black":"Roe, Jane","result":"1-0"},` +
			`"moves":[{"no":1,"white":"e4","black":"e5"},{"no":2,"white":"Sf3","black":"Sc6"}]}`
		resp := map[string]any{
			"candidates": []map[string]any{{
				"content": map[string]any{"parts": []map[string]any{{"text": inner}}},
			}},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	g := NewGemini(srv.URL, "gemini-2.5-flash", "secret-key")
	g.MaxDim = 0 // skip image decode/resize; send bytes as-is
	res, err := g.Recognize(context.Background(), ScoreSheetInput{Image: []byte("not-a-real-image"), MimeType: "image/png"})
	if err != nil {
		t.Fatalf("Recognize: %v", err)
	}
	if !strings.Contains(gotPath, "gemini-2.5-flash:generateContent") {
		t.Errorf("path = %q", gotPath)
	}
	if gotKey != "secret-key" {
		t.Errorf("api key header = %q", gotKey)
	}
	if res.Header.White != "Doe, John" || res.Header.Result != "1-0" {
		t.Errorf("header = %+v", res.Header)
	}
	if len(res.MoveTokens) != 4 {
		t.Fatalf("got %d move tokens, want 4: %+v", len(res.MoveTokens), res.MoveTokens)
	}
	if res.MoveTokens[0].Text != "e4" || res.MoveTokens[0].Side != SideWhite {
		t.Errorf("first token = %+v", res.MoveTokens[0])
	}
}

func TestGeminiRecognizeErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"quota"}}`))
	}))
	defer srv.Close()

	g := NewGemini(srv.URL, "gemini-2.5-flash", "key")
	g.MaxDim = 0
	if _, err := g.Recognize(context.Background(), ScoreSheetInput{Image: []byte("x")}); err == nil {
		t.Fatal("expected error on non-200 status")
	}
}

func TestGeminiRecognizeBlocked(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"candidates":[],"promptFeedback":{"blockReason":"SAFETY"}}`))
	}))
	defer srv.Close()

	g := NewGemini(srv.URL, "gemini-2.5-flash", "key")
	g.MaxDim = 0
	_, err := g.Recognize(context.Background(), ScoreSheetInput{Image: []byte("x")})
	if err == nil || !strings.Contains(err.Error(), "SAFETY") {
		t.Fatalf("expected block error mentioning SAFETY, got %v", err)
	}
}
