package stt

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFakeTranscribe(t *testing.T) {
	f := NewFake()
	out, err := f.Transcribe(context.Background(), TranscribeInput{Audio: []byte("ignored"), MimeType: "audio/webm"})
	if err != nil {
		t.Fatal(err)
	}
	if out.Text == "" {
		t.Error("expected non-empty transcript")
	}
	if out.Lang != "de" {
		t.Errorf("lang = %q, want de (default)", out.Lang)
	}
	if out.Model != "fake" {
		t.Errorf("model = %q, want fake", out.Model)
	}
}

func TestGeminiTranscribeSendsInlineAudio(t *testing.T) {
	var captured geminiRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Goog-Api-Key"); got != "k" {
			t.Errorf("api key header = %q", got)
		}
		_ = json.NewDecoder(r.Body).Decode(&captured)
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"Wie steht die Partie?"}]}}]}`))
	}))
	defer srv.Close()

	g := NewGemini(srv.URL, "m", "k")
	out, err := g.Transcribe(context.Background(), TranscribeInput{Audio: []byte("voice"), MimeType: "audio/ogg", Lang: "de"})
	if err != nil {
		t.Fatal(err)
	}
	if out.Text != "Wie steht die Partie?" {
		t.Errorf("text = %q", out.Text)
	}
	// The request must carry the audio inline with its mime type.
	if len(captured.Contents) != 1 || len(captured.Contents[0].Parts) != 2 {
		t.Fatalf("unexpected request shape: %+v", captured)
	}
	var inline *geminiInlineData
	for _, p := range captured.Contents[0].Parts {
		if p.InlineData != nil {
			inline = p.InlineData
		}
	}
	if inline == nil || inline.MimeType != "audio/ogg" {
		t.Fatalf("expected inline audio/ogg, got %+v", inline)
	}
	if inline.Data == "" {
		t.Error("expected base64 audio data")
	}
}

func TestGeminiTranscribeErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("upstream boom"))
	}))
	defer srv.Close()
	g := NewGemini(srv.URL, "m", "k")
	_, err := g.Transcribe(context.Background(), TranscribeInput{Audio: []byte("x"), MimeType: "audio/webm"})
	if err == nil || !strings.Contains(err.Error(), "502") {
		t.Errorf("expected 502 error, got %v", err)
	}
}
