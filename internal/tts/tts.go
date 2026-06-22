// Package tts turns coaching prose into spoken audio — "the coach really talks to you".
// Model/provider-specific code sits behind the Synthesizer interface, mirroring the
// coaching and recognition packages. TTS is ADVISORY: it only voices already-rendered
// coaching text and never affects the correctness of a saved PGN. Generated audio is
// content-addressed and cached, so the same text is synthesized at most once per
// (provider, voice, lang).
package tts

import "context"

// LangDefault is the default speech language. PGNize is a German-first product, so the
// coach speaks German unless another language is requested.
const LangDefault = "de"

// SpeakInput is one synthesis request. The caller (the HTTP handler) has already
// validated and length-capped the text and normalized the language.
type SpeakInput struct {
	Text  string
	Lang  string
	Voice string
}

// Audio is synthesized speech plus its MIME type (e.g. "audio/wav").
type Audio struct {
	Bytes       []byte
	ContentType string
}

// Synthesizer turns text into speech. Implementations: fake (tests/CI), gemini and piper.
// No provider/HTTP types leak through this interface — only the value types above.
type Synthesizer interface {
	Synthesize(ctx context.Context, in SpeakInput) (Audio, error)
	Name() string             // "gemini-tts:<model>" / "piper:<voice>" / "fake"
	Voice(lang string) string // default voice per provider/lang
}

// normLang returns the effective language code, defaulting to German.
func normLang(l string) string {
	if l == "" {
		return LangDefault
	}
	return l
}

// Compile-time assertions that every synthesizer satisfies the interface.
var (
	_ Synthesizer = (*Fake)(nil)
	_ Synthesizer = (*GeminiTTS)(nil)
	_ Synthesizer = (*PiperTTS)(nil)
	_ Synthesizer = (*Chain)(nil)
)
