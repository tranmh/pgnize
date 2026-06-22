package tts

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// These thin clients only do a single synthesis call; they do not reuse the recognition
// or coaching clients (those are shaped for image/text completions and keep their helpers
// unexported).

const ttsDefaultTimeout = 60 * time.Second

// Gemini TTS returns headerless signed 16-bit little-endian mono PCM at 24kHz.
const (
	geminiPCMSampleRate    = 24000
	geminiPCMBitsPerSample = 16
	geminiPCMChannels      = 1
)

func envStr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return n
		}
	}
	return def
}

// GeminiTTS is a Synthesizer backed by Google's Gemini API (the same generateContent
// endpoint the coach uses), requesting the AUDIO modality.
type GeminiTTS struct {
	Host         string
	Model        string
	APIKey       string
	DefaultVoice string
	Client       *http.Client
}

// NewGeminiTTS builds a Gemini-backed synthesizer. voice is the default prebuilt voice.
func NewGeminiTTS(host, model, apiKey, voice string) *GeminiTTS {
	timeout := time.Duration(envInt("TTS_TIMEOUT_SEC", int(ttsDefaultTimeout.Seconds()))) * time.Second
	return &GeminiTTS{
		Host:         strings.TrimRight(host, "/"),
		Model:        model,
		APIKey:       apiKey,
		DefaultVoice: voice,
		Client:       &http.Client{Timeout: timeout},
	}
}

func (g *GeminiTTS) Name() string { return "gemini-tts:" + g.Model }

// Voice ignores lang: Gemini's prebuilt voices are multilingual.
func (g *GeminiTTS) Voice(string) string { return g.DefaultVoice }

type geminiPart struct {
	Text string `json:"text"`
}
type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}
type geminiPrebuiltVoiceConfig struct {
	VoiceName string `json:"voiceName"`
}
type geminiVoiceConfig struct {
	PrebuiltVoiceConfig geminiPrebuiltVoiceConfig `json:"prebuiltVoiceConfig"`
}
type geminiSpeechConfig struct {
	VoiceConfig geminiVoiceConfig `json:"voiceConfig"`
}
type geminiTTSGenConfig struct {
	ResponseModalities []string           `json:"responseModalities"`
	SpeechConfig       geminiSpeechConfig `json:"speechConfig"`
}
type geminiTTSRequest struct {
	Contents         []geminiContent    `json:"contents"`
	GenerationConfig geminiTTSGenConfig `json:"generationConfig"`
}
type geminiTTSResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				InlineData struct {
					MimeType string `json:"mimeType"`
					Data     string `json:"data"`
				} `json:"inlineData"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	PromptFeedback struct {
		BlockReason string `json:"blockReason"`
	} `json:"promptFeedback"`
}

func (g *GeminiTTS) Synthesize(ctx context.Context, in SpeakInput) (Audio, error) {
	voice := in.Voice
	if voice == "" {
		voice = g.DefaultVoice
	}
	reqBody := geminiTTSRequest{
		Contents: []geminiContent{{Role: "user", Parts: []geminiPart{{Text: in.Text}}}},
		GenerationConfig: geminiTTSGenConfig{
			ResponseModalities: []string{"AUDIO"},
			SpeechConfig: geminiSpeechConfig{
				VoiceConfig: geminiVoiceConfig{
					PrebuiltVoiceConfig: geminiPrebuiltVoiceConfig{VoiceName: voice},
				},
			},
		},
	}
	buf, _ := json.Marshal(reqBody)

	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent", g.Host, g.Model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		return Audio{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Goog-Api-Key", g.APIKey)

	resp, err := g.Client.Do(req)
	if err != nil {
		return Audio{}, fmt.Errorf("gemini tts request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return Audio{}, fmt.Errorf("gemini tts status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var gr geminiTTSResponse
	if err := json.NewDecoder(resp.Body).Decode(&gr); err != nil {
		return Audio{}, fmt.Errorf("decode gemini tts envelope: %w", err)
	}
	if len(gr.Candidates) == 0 {
		if reason := gr.PromptFeedback.BlockReason; reason != "" {
			return Audio{}, fmt.Errorf("gemini tts blocked: %s", reason)
		}
		return Audio{}, fmt.Errorf("gemini tts returned no candidates")
	}
	var data string
	for _, p := range gr.Candidates[0].Content.Parts {
		if p.InlineData.Data != "" {
			data = p.InlineData.Data
			break
		}
	}
	if data == "" {
		return Audio{}, fmt.Errorf("gemini tts returned no audio data")
	}
	pcm, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return Audio{}, fmt.Errorf("decode gemini tts pcm: %w", err)
	}
	wav := pcmToWAV(pcm, geminiPCMSampleRate, geminiPCMBitsPerSample, geminiPCMChannels)
	return Audio{Bytes: wav, ContentType: "audio/wav"}, nil
}
