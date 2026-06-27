package stt

import "context"

// Fake is a deterministic Transcriber for tests/CI: it returns a fixed German question
// regardless of the audio bytes, mirroring how recognition.Fake ignores image pixels. This
// keeps the audio path exercised end-to-end without a real speech model.
type Fake struct{}

var _ Transcriber = (*Fake)(nil)

// NewFake returns a Fake transcriber.
func NewFake() *Fake { return &Fake{} }

func (f *Fake) Name() string { return "fake" }

func (f *Fake) Transcribe(_ context.Context, in TranscribeInput) (Transcription, error) {
	return Transcription{
		Text:  "Was ist der beste Zug in dieser Stellung?",
		Model: f.Name(),
		Lang:  normLang(in.Lang),
	}, nil
}
