package tts

import "context"

// fakeSampleRate / fakeDurationSamples produce ~0.1s of 16-bit mono silence.
const (
	fakeSampleRate     = 24000
	fakeDurationSecond = 0.1
)

// Fake is a deterministic Synthesizer for tests and CI. It ignores any model and returns a
// tiny valid WAV (~0.1s of silence) so the speak/audio flow can be exercised without a
// network call (mirrors RECOGNIZER=fake / COACH=fake).
type Fake struct{}

// NewFake returns a deterministic synthesizer.
func NewFake() *Fake { return &Fake{} }

func (f *Fake) Name() string { return "fake" }

// Voice ignores lang: the fake always uses a single deterministic voice.
func (f *Fake) Voice(string) string { return "fake" }

func (f *Fake) Synthesize(_ context.Context, _ SpeakInput) (Audio, error) {
	// 16-bit mono → 2 bytes per sample; all zeros = silence.
	samples := int(fakeSampleRate * fakeDurationSecond)
	pcm := make([]byte, samples*2)
	wav := pcmToWAV(pcm, fakeSampleRate, 16, 1)
	return Audio{Bytes: wav, ContentType: "audio/wav"}, nil
}
