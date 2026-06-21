package coaching

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// The coach is text-only and needs free-form prose, so it does NOT reuse the recognition
// package's clients (those are image + JSON-schema shaped, and their env helpers are
// unexported). These thin clients only do a single text completion.

const (
	coachDefaultMaxTokens = 512
	coachDefaultTimeout   = 60 * time.Second
	// A little warmth produces better teaching prose than the near-deterministic
	// transcription temperature, while staying grounded in the supplied numbers.
	coachTemperature = 0.4
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

// ---- Gemini ----

// GeminiCoach is a Coach backed by Google's Gemini API (text generation, no image).
type GeminiCoach struct {
	Host      string
	Model     string
	APIKey    string
	Client    *http.Client
	MaxTokens int
}

// NewGeminiCoach builds a Gemini-backed coach. COACH_MAX_TOKENS overrides the budget.
func NewGeminiCoach(host, model, apiKey string) *GeminiCoach {
	return &GeminiCoach{
		Host:      strings.TrimRight(host, "/"),
		Model:     model,
		APIKey:    apiKey,
		Client:    &http.Client{Timeout: coachDefaultTimeout},
		MaxTokens: envInt("COACH_MAX_TOKENS", coachDefaultMaxTokens),
	}
}

func (c *GeminiCoach) Name() string { return "gemini:" + c.Model }

type geminiTextPart struct {
	Text string `json:"text"`
}
type geminiTextContent struct {
	Role  string           `json:"role,omitempty"`
	Parts []geminiTextPart `json:"parts"`
}
type geminiThinking struct {
	ThinkingBudget int `json:"thinkingBudget"`
}
type geminiTextGenConfig struct {
	Temperature     float64         `json:"temperature"`
	MaxOutputTokens int             `json:"maxOutputTokens,omitempty"`
	ThinkingConfig  *geminiThinking `json:"thinkingConfig,omitempty"`
}
type geminiTextRequest struct {
	Contents         []geminiTextContent `json:"contents"`
	GenerationConfig geminiTextGenConfig `json:"generationConfig"`
}
type geminiTextResponse struct {
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

func (c *GeminiCoach) generate(ctx context.Context, prompt string) (string, error) {
	reqBody := geminiTextRequest{
		Contents: []geminiTextContent{{Role: "user", Parts: []geminiTextPart{{Text: prompt}}}},
		GenerationConfig: geminiTextGenConfig{
			Temperature:     coachTemperature,
			MaxOutputTokens: c.MaxTokens,
			// No chain-of-thought needed for a short explanation.
			ThinkingConfig: &geminiThinking{ThinkingBudget: 0},
		},
	}
	buf, _ := json.Marshal(reqBody)

	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent", c.Host, c.Model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Goog-Api-Key", c.APIKey)

	resp, err := c.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("gemini coach request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return "", fmt.Errorf("gemini coach status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var gr geminiTextResponse
	if err := json.NewDecoder(resp.Body).Decode(&gr); err != nil {
		return "", fmt.Errorf("decode gemini coach envelope: %w", err)
	}
	if len(gr.Candidates) == 0 {
		if reason := gr.PromptFeedback.BlockReason; reason != "" {
			return "", fmt.Errorf("gemini coach blocked: %s", reason)
		}
		return "", fmt.Errorf("gemini coach returned no candidates")
	}
	var text strings.Builder
	for _, p := range gr.Candidates[0].Content.Parts {
		text.WriteString(p.Text)
	}
	out := strings.TrimSpace(text.String())
	if out == "" {
		return "", fmt.Errorf("gemini coach returned empty text")
	}
	return out, nil
}

func (c *GeminiCoach) CoachMove(ctx context.Context, in MoveInput) (Coaching, error) {
	lang := normLang(in.Lang)
	text, err := c.generate(ctx, buildMovePrompt(in))
	if err != nil {
		return Coaching{}, err
	}
	return Coaching{Text: text, Model: c.Name(), Lang: lang}, nil
}

func (c *GeminiCoach) CoachGame(ctx context.Context, in GameInput) (Coaching, error) {
	lang := normLang(in.Lang)
	text, err := c.generate(ctx, buildGamePrompt(in))
	if err != nil {
		return Coaching{}, err
	}
	return Coaching{Text: text, Model: c.Name(), Lang: lang}, nil
}

// ---- Ollama ----

// OllamaCoach is a Coach backed by a local Ollama server (text generation).
type OllamaCoach struct {
	Host       string
	Model      string
	Client     *http.Client
	NumPredict int
	KeepAlive  string
}

// NewOllamaCoach builds an Ollama-backed coach.
func NewOllamaCoach(host, model string) *OllamaCoach {
	timeout := time.Duration(envInt("COACH_TIMEOUT_SEC", int(coachDefaultTimeout.Seconds()))) * time.Second
	return &OllamaCoach{
		Host:       strings.TrimRight(host, "/"),
		Model:      model,
		Client:     &http.Client{Timeout: timeout},
		NumPredict: envInt("COACH_MAX_TOKENS", coachDefaultMaxTokens),
		KeepAlive:  envStr("OLLAMA_KEEP_ALIVE", "30m"),
	}
}

func (c *OllamaCoach) Name() string { return "ollama:" + c.Model }

type ollamaTextRequest struct {
	Model     string         `json:"model"`
	Prompt    string         `json:"prompt"`
	Stream    bool           `json:"stream"`
	KeepAlive string         `json:"keep_alive,omitempty"`
	Options   map[string]any `json:"options,omitempty"`
}
type ollamaTextResponse struct {
	Response string `json:"response"`
}

func (c *OllamaCoach) generate(ctx context.Context, prompt string) (string, error) {
	reqBody := ollamaTextRequest{
		Model:     c.Model,
		Prompt:    prompt,
		Stream:    false,
		KeepAlive: c.KeepAlive,
		Options:   map[string]any{"temperature": coachTemperature, "num_predict": c.NumPredict},
	}
	buf, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Host+"/api/generate", bytes.NewReader(buf))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ollama coach request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama coach status %d", resp.StatusCode)
	}
	var or ollamaTextResponse
	if err := json.NewDecoder(resp.Body).Decode(&or); err != nil {
		return "", fmt.Errorf("decode ollama coach envelope: %w", err)
	}
	out := strings.TrimSpace(or.Response)
	if out == "" {
		return "", fmt.Errorf("ollama coach returned empty text")
	}
	return out, nil
}

func (c *OllamaCoach) CoachMove(ctx context.Context, in MoveInput) (Coaching, error) {
	lang := normLang(in.Lang)
	text, err := c.generate(ctx, buildMovePrompt(in))
	if err != nil {
		return Coaching{}, err
	}
	return Coaching{Text: text, Model: c.Name(), Lang: lang}, nil
}

func (c *OllamaCoach) CoachGame(ctx context.Context, in GameInput) (Coaching, error) {
	lang := normLang(in.Lang)
	text, err := c.generate(ctx, buildGamePrompt(in))
	if err != nil {
		return Coaching{}, err
	}
	return Coaching{Text: text, Model: c.Name(), Lang: lang}, nil
}
