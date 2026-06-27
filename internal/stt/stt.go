// Package stt is speech-to-text for the conversational coach: it turns a recorded voice
// question into text the LLM can read. Like recognition/coaching/tts it sits behind an
// interface with a deterministic Fake (tests/CI) and a real Gemini multimodal backend.
package stt

import "context"

// LangDefault mirrors the rest of PGNize: German-first.
const LangDefault = "de"

// MaxBytesDefault caps the uploaded audio size (5 MiB) unless overridden via STT_MAX_BYTES.
const MaxBytesDefault = 5 << 20

// TranscribeInput is one voice utterance to transcribe.
type TranscribeInput struct {
	Audio    []byte
	MimeType string // e.g. "audio/webm", "audio/ogg", "audio/wav", "audio/mp4"
	Lang     string // "" -> LangDefault
}

// Transcription is the recognized text.
type Transcription struct {
	Text  string `json:"text"`
	Model string `json:"model"`
	Lang  string `json:"lang"`
}

// Transcriber converts speech audio to text.
type Transcriber interface {
	Transcribe(ctx context.Context, in TranscribeInput) (Transcription, error)
	Name() string
}

func normLang(l string) string {
	if l == "" {
		return LangDefault
	}
	return l
}
