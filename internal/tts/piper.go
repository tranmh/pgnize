package tts

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// PiperTTS is a Synthesizer backed by a self-hosted Piper HTTP server, which returns a WAV
// body directly. It is the local fallback when no Gemini key is configured.
type PiperTTS struct {
	Host         string
	DefaultVoice string
	Client       *http.Client
}

// NewPiperTTS builds a Piper-backed synthesizer. voice names the bundled Piper voice model.
func NewPiperTTS(host, voice string) *PiperTTS {
	timeout := time.Duration(envInt("TTS_TIMEOUT_SEC", int(ttsDefaultTimeout.Seconds()))) * time.Second
	return &PiperTTS{
		Host:         strings.TrimRight(host, "/"),
		DefaultVoice: voice,
		Client:       &http.Client{Timeout: timeout},
	}
}

func (p *PiperTTS) Name() string { return "piper:" + p.DefaultVoice }

// Voice ignores lang: a Piper server is configured with a single voice model.
func (p *PiperTTS) Voice(string) string { return p.DefaultVoice }

func (p *PiperTTS) Synthesize(ctx context.Context, in SpeakInput) (Audio, error) {
	// The Piper HTTP server reads the text to speak from the request body and returns WAV.
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.Host, bytes.NewReader([]byte(in.Text)))
	if err != nil {
		return Audio{}, err
	}
	req.Header.Set("Content-Type", "text/plain")

	resp, err := p.Client.Do(req)
	if err != nil {
		return Audio{}, fmt.Errorf("piper tts request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return Audio{}, fmt.Errorf("piper tts status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Audio{}, fmt.Errorf("read piper tts body: %w", err)
	}
	if len(body) == 0 {
		return Audio{}, fmt.Errorf("piper tts returned empty audio")
	}
	return Audio{Bytes: body, ContentType: "audio/wav"}, nil
}
