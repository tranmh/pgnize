//go:build integration

package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func (h *harness) ttsRows(t *testing.T) int {
	t.Helper()
	var n int
	if err := h.st.Pool.QueryRow(context.Background(), `SELECT count(*) FROM tts_audio`).Scan(&n); err != nil {
		t.Fatalf("count tts_audio: %v", err)
	}
	return n
}

func TestSpeakAndFetchAudio(t *testing.T) {
	h := setup(t)

	resp, body := h.json(t, "POST", "/api/coach/speak", map[string]any{
		"text": "Die Engine bevorzugt d4.",
		"lang": "de",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("speak %d: %s", resp.StatusCode, body)
	}
	var first struct {
		AudioURL string `json:"audioUrl"`
		Cached   bool   `json:"cached"`
		Provider string `json:"provider"`
		Voice    string `json:"voice"`
	}
	if err := json.Unmarshal(body, &first); err != nil {
		t.Fatal(err)
	}
	if first.Cached {
		t.Error("first synthesis must not be cached")
	}
	if !strings.HasPrefix(first.AudioURL, "/api/coach/audio/") {
		t.Errorf("unexpected audioUrl %q", first.AudioURL)
	}
	if first.Provider == "" || first.Voice == "" {
		t.Errorf("expected provider+voice, got %s", body)
	}
	if n := h.ttsRows(t); n != 1 {
		t.Fatalf("expected 1 tts_audio row after first speak, got %d", n)
	}

	// Fetch the audio blob.
	resp, audio := h.do(t, "GET", first.AudioURL, "", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get audio %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "audio/") {
		t.Errorf("content type = %q, want audio/*", ct)
	}
	if len(audio) == 0 {
		t.Error("audio body must be non-empty")
	}

	// Identical repeat → cached, no new row.
	resp, body = h.json(t, "POST", "/api/coach/speak", map[string]any{
		"text": "Die Engine bevorzugt d4.",
		"lang": "de",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("speak repeat %d: %s", resp.StatusCode, body)
	}
	var second struct {
		AudioURL string `json:"audioUrl"`
		Cached   bool   `json:"cached"`
	}
	json.Unmarshal(body, &second)
	if !second.Cached {
		t.Error("identical repeat should be cached")
	}
	if second.AudioURL != first.AudioURL {
		t.Errorf("cached audioUrl differs: %q vs %q", second.AudioURL, first.AudioURL)
	}
	if n := h.ttsRows(t); n != 1 {
		t.Fatalf("repeat must not write a second row, got %d", n)
	}
}

func TestSpeakEmptyText(t *testing.T) {
	h := setup(t)
	resp, body := h.json(t, "POST", "/api/coach/speak", map[string]any{"text": ""})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("empty text status %d want 400: %s", resp.StatusCode, body)
	}
	var ae struct {
		Error string `json:"error"`
	}
	json.Unmarshal(body, &ae)
	if ae.Error != "bad_request" {
		t.Errorf("error = %q, want bad_request", ae.Error)
	}
}

func TestSpeakTextTooLong(t *testing.T) {
	h := setup(t)
	resp, body := h.json(t, "POST", "/api/coach/speak", map[string]any{
		"text": strings.Repeat("a", 4001),
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("over-long text status %d want 400: %s", resp.StatusCode, body)
	}
	var ae struct {
		Error string `json:"error"`
	}
	json.Unmarshal(body, &ae)
	if ae.Error != "text_too_long" {
		t.Errorf("error = %q, want text_too_long", ae.Error)
	}
}
