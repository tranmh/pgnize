package recognition

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/tranmh/pgnize/internal/domain"
)

// Ollama is a Recognizer backed by a local Ollama server running a vision model.
type Ollama struct {
	Host   string
	Model  string
	Client *http.Client
}

// NewOllama builds an Ollama-backed recognizer.
func NewOllama(host, model string) *Ollama {
	return &Ollama{
		Host:   host,
		Model:  model,
		Client: &http.Client{Timeout: 10 * time.Minute}, // CPU inference is slow
	}
}

func (o *Ollama) Name() string { return "ollama:" + o.Model }

type ollamaRequest struct {
	Model    string         `json:"model"`
	Prompt   string         `json:"prompt"`
	Images   []string       `json:"images,omitempty"`
	Format   map[string]any `json:"format,omitempty"`
	Stream   bool           `json:"stream"`
	Options  map[string]any `json:"options,omitempty"`
}

type ollamaResponse struct {
	Response string `json:"response"`
}

type modelOutput struct {
	Header domain.Header `json:"header"`
	Moves  []struct {
		No    int    `json:"no"`
		White string `json:"white"`
		Black string `json:"black"`
	} `json:"moves"`
}

func (o *Ollama) Recognize(ctx context.Context, in ScoreSheetInput) (RecognitionResult, error) {
	reqBody := ollamaRequest{
		Model:   o.Model,
		Prompt:  buildPrompt(in),
		Images:  []string{base64.StdEncoding.EncodeToString(in.Image)},
		Format:  jsonSchema,
		Stream:  false,
		Options: map[string]any{"temperature": 0.1},
	}
	buf, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.Host+"/api/generate", bytes.NewReader(buf))
	if err != nil {
		return RecognitionResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := o.Client.Do(req)
	if err != nil {
		return RecognitionResult{}, fmt.Errorf("ollama request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return RecognitionResult{}, fmt.Errorf("ollama status %d", resp.StatusCode)
	}
	var or ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&or); err != nil {
		return RecognitionResult{}, fmt.Errorf("decode ollama envelope: %w", err)
	}
	var mo modelOutput
	if err := json.Unmarshal([]byte(or.Response), &mo); err != nil {
		return RecognitionResult{}, fmt.Errorf("decode model json: %w", err)
	}
	return RecognitionResult{
		Header:     mo.Header,
		MoveTokens: flattenMoves(mo),
		Confidence: 0.5, // local models do not self-report; the review loop is the safety net
		RawJSON:    or.Response,
	}, nil
}

func flattenMoves(mo modelOutput) []MoveToken {
	var out []MoveToken
	ply := 0
	for _, row := range mo.Moves {
		if row.White != "" {
			ply++
			out = append(out, MoveToken{Ply: ply, Side: SideWhite, Text: row.White, Confidence: 0.5})
		}
		if row.Black != "" {
			ply++
			out = append(out, MoveToken{Ply: ply, Side: SideBlack, Text: row.Black, Confidence: 0.5})
		}
	}
	return out
}
