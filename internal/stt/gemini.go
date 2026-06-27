package stt

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const geminiDefaultTimeout = 60 * time.Second

// Gemini is a Transcriber backed by Google's Gemini multimodal API: it sends the audio
// inline (base64) with a transcription prompt and reads back plain text. Mirrors the inline
// data + X-Goog-Api-Key plumbing of recognition.Gemini, but for audio instead of an image.
type Gemini struct {
	Host   string
	Model  string
	APIKey string
	Client *http.Client
}

var _ Transcriber = (*Gemini)(nil)

// NewGemini builds a Gemini-backed transcriber.
func NewGemini(host, model, apiKey string) *Gemini {
	return &Gemini{
		Host:   strings.TrimRight(host, "/"),
		Model:  model,
		APIKey: apiKey,
		Client: &http.Client{Timeout: geminiDefaultTimeout},
	}
}

func (g *Gemini) Name() string { return "gemini-stt:" + g.Model }

type geminiInlineData struct {
	MimeType string `json:"mime_type"`
	Data     string `json:"data"`
}
type geminiPart struct {
	Text       string            `json:"text,omitempty"`
	InlineData *geminiInlineData `json:"inline_data,omitempty"`
}
type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}
type geminiThinking struct {
	ThinkingBudget int `json:"thinkingBudget"`
}
type geminiGenConfig struct {
	Temperature      float64         `json:"temperature"`
	ResponseMimeType string          `json:"responseMimeType"`
	ThinkingConfig   *geminiThinking `json:"thinkingConfig,omitempty"`
}
type geminiRequest struct {
	Contents         []geminiContent `json:"contents"`
	GenerationConfig geminiGenConfig `json:"generationConfig"`
}
type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	PromptFeedback struct {
		BlockReason string `json:"blockReason"`
	} `json:"promptFeedback"`
}

func transcribePrompt(lang string) string {
	if lang == "en" {
		return "Transcribe this voice recording verbatim. Return only the spoken text, no commentary."
	}
	return "Transkribiere diese Sprachaufnahme wörtlich. Gib nur den gesprochenen Text zurück, ohne Kommentar."
}

func (g *Gemini) Transcribe(ctx context.Context, in TranscribeInput) (Transcription, error) {
	lang := normLang(in.Lang)
	mime := in.MimeType
	if mime == "" {
		mime = "audio/webm"
	}
	reqBody := geminiRequest{
		Contents: []geminiContent{{
			Role: "user",
			Parts: []geminiPart{
				{Text: transcribePrompt(lang)},
				{InlineData: &geminiInlineData{MimeType: mime, Data: base64.StdEncoding.EncodeToString(in.Audio)}},
			},
		}},
		GenerationConfig: geminiGenConfig{
			Temperature:      0,
			ResponseMimeType: "text/plain",
			ThinkingConfig:   &geminiThinking{ThinkingBudget: 0},
		},
	}
	buf, _ := json.Marshal(reqBody)

	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent", g.Host, g.Model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		return Transcription{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Goog-Api-Key", g.APIKey)

	resp, err := g.Client.Do(req)
	if err != nil {
		return Transcription{}, fmt.Errorf("gemini stt request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return Transcription{}, fmt.Errorf("gemini stt status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var gr geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&gr); err != nil {
		return Transcription{}, fmt.Errorf("decode gemini stt envelope: %w", err)
	}
	if len(gr.Candidates) == 0 {
		if r := gr.PromptFeedback.BlockReason; r != "" {
			return Transcription{}, fmt.Errorf("gemini stt blocked: %s", r)
		}
		return Transcription{}, fmt.Errorf("gemini stt returned no candidates")
	}
	var text strings.Builder
	for _, p := range gr.Candidates[0].Content.Parts {
		text.WriteString(p.Text)
	}
	out := strings.TrimSpace(text.String())
	if out == "" {
		return Transcription{}, fmt.Errorf("gemini stt returned empty text")
	}
	return Transcription{Text: out, Model: g.Name(), Lang: lang}, nil
}
