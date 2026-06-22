package tts

import (
	"bytes"
	"context"
	"testing"
)

func TestFakeProducesValidWAV(t *testing.T) {
	f := NewFake()
	if f.Name() != "fake" {
		t.Errorf("Name = %q, want fake", f.Name())
	}
	a, err := f.Synthesize(context.Background(), SpeakInput{Text: "Hallo", Lang: "de"})
	if err != nil {
		t.Fatalf("synthesize: %v", err)
	}
	if a.ContentType != "audio/wav" {
		t.Errorf("content type = %q, want audio/wav", a.ContentType)
	}
	if len(a.Bytes) <= 44 {
		t.Fatalf("expected non-trivial WAV, got %d bytes", len(a.Bytes))
	}
	if !bytes.Equal(a.Bytes[0:4], []byte("RIFF")) || !bytes.Equal(a.Bytes[8:12], []byte("WAVE")) {
		t.Errorf("fake did not emit a WAV container")
	}
}

func TestFakeDeterministic(t *testing.T) {
	f := NewFake()
	a1, _ := f.Synthesize(context.Background(), SpeakInput{Text: "x"})
	a2, _ := f.Synthesize(context.Background(), SpeakInput{Text: "y"})
	if !bytes.Equal(a1.Bytes, a2.Bytes) {
		t.Errorf("fake output must be deterministic regardless of input text")
	}
}
