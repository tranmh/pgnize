package tts

import (
	"context"
	"errors"
	"testing"
)

// stubSynth is a programmable Synthesizer for chain tests.
type stubSynth struct {
	name  string
	audio Audio
	err   error
	calls *int
}

func (s *stubSynth) Name() string        { return s.name }
func (s *stubSynth) Voice(string) string { return s.name }
func (s *stubSynth) Synthesize(_ context.Context, _ SpeakInput) (Audio, error) {
	if s.calls != nil {
		*s.calls++
	}
	return s.audio, s.err
}

func TestChainFirstSuccessWins(t *testing.T) {
	var c1, c2 int
	primary := &stubSynth{name: "primary", err: errors.New("down"), calls: &c1}
	fallback := &stubSynth{name: "fallback", audio: Audio{Bytes: []byte("wav"), ContentType: "audio/wav"}, calls: &c2}

	chain := NewChain(primary, fallback)
	a, err := chain.Synthesize(context.Background(), SpeakInput{Text: "hi"})
	if err != nil {
		t.Fatalf("expected fallback to succeed: %v", err)
	}
	if string(a.Bytes) != "wav" {
		t.Errorf("got %q, want fallback audio", a.Bytes)
	}
	if c1 != 1 || c2 != 1 {
		t.Errorf("expected both tried once, got primary=%d fallback=%d", c1, c2)
	}
	// Name reports the primary (configured intent), not the one that produced audio.
	if chain.Name() != "primary" {
		t.Errorf("Name = %q, want primary", chain.Name())
	}
}

func TestChainStopsAtFirstSuccess(t *testing.T) {
	var c1, c2 int
	primary := &stubSynth{name: "primary", audio: Audio{Bytes: []byte("ok")}, calls: &c1}
	fallback := &stubSynth{name: "fallback", calls: &c2}

	chain := NewChain(primary, fallback)
	if _, err := chain.Synthesize(context.Background(), SpeakInput{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c1 != 1 || c2 != 0 {
		t.Errorf("fallback should not be called when primary succeeds: primary=%d fallback=%d", c1, c2)
	}
}

func TestChainAllFail(t *testing.T) {
	primary := &stubSynth{name: "primary", err: errors.New("a")}
	fallback := &stubSynth{name: "fallback", err: errors.New("b")}

	chain := NewChain(primary, fallback)
	if _, err := chain.Synthesize(context.Background(), SpeakInput{}); err == nil {
		t.Fatal("expected error when all backends fail")
	}
}

func TestChainEmpty(t *testing.T) {
	chain := NewChain()
	if _, err := chain.Synthesize(context.Background(), SpeakInput{}); err == nil {
		t.Fatal("expected error from empty chain")
	}
}
