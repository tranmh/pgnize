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
		// Thinking must be capped: 2.5 models otherwise spend the whole output-token
		// budget on internal reasoning and truncate the JSON after a single move.
		if !strings.Contains(string(body), `"thinkingConfig":{"thinkingBudget":0}`) {
			t.Errorf("request missing thinkingConfig with budget 0: %s", body)
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

// TestGeminiRecognizeSendsExtraImages proves a multi-image submission reaches the model as
// multiple inline_data parts: primary + each Extra blob (one combined request, one result).
func TestGeminiRecognizeSendsExtraImages(t *testing.T) {
	var inlineCount int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		inlineCount = strings.Count(string(body), "inline_data")
		inner := `{"header":{"white":"A","black":"B"},"moves":[{"no":1,"white":"e4","black":"e5"}]}`
		resp := map[string]any{"candidates": []map[string]any{{
			"content": map[string]any{"parts": []map[string]any{{"text": inner}}},
		}}}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	g := NewGemini(srv.URL, "gemini-2.5-flash", "key")
	g.MaxDim = 0 // skip image decode/resize; send bytes as-is
	_, err := g.Recognize(context.Background(), ScoreSheetInput{
		Image:    []byte("primary"),
		MimeType: "image/png",
		Extra: []ImageBlob{
			{Data: []byte("extra-1"), MimeType: "image/png"},
			{Data: []byte("extra-2"), MimeType: "image/jpeg"},
		},
	})
	if err != nil {
		t.Fatalf("Recognize: %v", err)
	}
	if inlineCount != 3 { // primary + 2 extras
		t.Fatalf("inline_data parts = %d, want 3", inlineCount)
	}
}

// TestGeminiRecognizePositionSendsExtraImages is the position-pipeline counterpart.
func TestGeminiRecognizePositionSendsExtraImages(t *testing.T) {
	var inlineCount int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		inlineCount = strings.Count(string(body), "inline_data")
		inner := `{"grid":["....k...","........","........","........","........","........","........","....K..R"],` +
			`"sideToMove":"white","orientation":"white_bottom"}`
		resp := map[string]any{"candidates": []map[string]any{{
			"content": map[string]any{"parts": []map[string]any{{"text": inner}}},
		}}}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	g := NewGemini(srv.URL, "gemini-2.5-flash", "key")
	g.MaxDim = 0
	_, err := g.RecognizePosition(context.Background(), PositionInput{
		Image:    []byte("primary"),
		MimeType: "image/png",
		Extra:    []ImageBlob{{Data: []byte("extra-1"), MimeType: "image/png"}},
	})
	if err != nil {
		t.Fatalf("RecognizePosition: %v", err)
	}
	if inlineCount != 2 { // primary + 1 extra
		t.Fatalf("inline_data parts = %d, want 2", inlineCount)
	}
}

func TestGeminiRecognizePositionParsesGrid(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		inner := `{"grid":["....k...","........","........","........","........","........","........","....K..R"],` +
			`"sideToMove":"white","orientation":"white_bottom"}`
		resp := map[string]any{
			"candidates": []map[string]any{{
				"content": map[string]any{"parts": []map[string]any{{"text": inner}}},
			}},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	g := NewGemini(srv.URL, "gemini-2.5-flash", "key")
	g.MaxDim = 0
	res, err := g.RecognizePosition(context.Background(), PositionInput{Image: []byte("img"), MimeType: "image/png"})
	if err != nil {
		t.Fatalf("RecognizePosition: %v", err)
	}
	// The request must carry the position schema and prompt.
	if !strings.Contains(gotBody, "orientation") {
		t.Errorf("request missing position schema: %s", gotBody)
	}
	if !strings.Contains(gotBody, "single chess position") {
		t.Errorf("request missing position prompt: %s", gotBody)
	}
	if len(res.Grid) != 8 || res.Grid[0] != "....k..." || res.Grid[7] != "....K..R" {
		t.Fatalf("grid = %+v", res.Grid)
	}
	if res.SideToMove != "white" || res.Orientation != "white_bottom" {
		t.Errorf("side/orientation = %q/%q", res.SideToMove, res.Orientation)
	}
	fen, err := AssembleFEN(res)
	if err != nil || fen != "4k3/8/8/8/8/8/8/4K2R w - - 0 1" {
		t.Fatalf("AssembleFEN = %q (%v)", fen, err)
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
