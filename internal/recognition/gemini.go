package recognition

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"strings"
	"time"

	xdraw "golang.org/x/image/draw"
)

// Gemini defaults: cloud inference is fast, so we allow a generous token budget and a
// shorter timeout than the local Ollama path.
const (
	geminiDefaultMaxTokens = 4096
	geminiDefaultTimeout   = 2 * time.Minute
	geminiDefaultMaxDim    = 1600 // longest image edge sent to the model
	// Gemini 2.5 models "think" by default, and those thinking tokens are billed against
	// maxOutputTokens — on a long score sheet the model can burn the entire budget on
	// internal reasoning and truncate the JSON answer after a single move. Transcription
	// needs no chain-of-thought, so we disable thinking by default (0 = off, -1 = dynamic).
	geminiDefaultThinkingBudget = 0
)

// Gemini is a Recognizer backed by Google's Gemini API (AI Studio / generativelanguage).
// It uses structured-output (responseSchema + JSON mime) for reliable parsing and sends
// the score-sheet image inline. The review loop remains the correctness guarantee.
type Gemini struct {
	Host      string // e.g. https://generativelanguage.googleapis.com
	Model     string // e.g. gemini-2.5-flash
	APIKey    string
	Client    *http.Client
	MaxTokens int
	MaxDim    int
	// ThinkingBudget caps the model's internal reasoning tokens. 0 disables thinking
	// (the right default for transcription); -1 lets the model decide dynamically.
	ThinkingBudget int
}

// NewGemini builds a Gemini-backed recognizer. GEMINI_MAX_TOKENS and GEMINI_MAX_DIM
// override the output-token budget and the image downscale size.
func NewGemini(host, model, apiKey string) *Gemini {
	return &Gemini{
		Host:           strings.TrimRight(host, "/"),
		Model:          model,
		APIKey:         apiKey,
		Client:         &http.Client{Timeout: geminiDefaultTimeout},
		MaxTokens:      envInt("GEMINI_MAX_TOKENS", geminiDefaultMaxTokens),
		MaxDim:         envInt("GEMINI_MAX_DIM", geminiDefaultMaxDim),
		ThinkingBudget: envInt("GEMINI_THINKING_BUDGET", geminiDefaultThinkingBudget),
	}
}

func (g *Gemini) Name() string { return "gemini:" + g.Model }

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

type geminiThinkingConfig struct {
	ThinkingBudget int `json:"thinkingBudget"`
}

type geminiGenerationConfig struct {
	Temperature      float64               `json:"temperature"`
	ResponseMimeType string                `json:"responseMimeType"`
	ResponseSchema   map[string]any        `json:"responseSchema,omitempty"`
	MaxOutputTokens  int                   `json:"maxOutputTokens,omitempty"`
	ThinkingConfig   *geminiThinkingConfig `json:"thinkingConfig,omitempty"`
}

type geminiRequest struct {
	Contents         []geminiContent        `json:"contents"`
	GenerationConfig geminiGenerationConfig `json:"generationConfig"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	PromptFeedback struct {
		BlockReason string `json:"blockReason"`
	} `json:"promptFeedback"`
}

// geminiResponseSchema constrains the model to PGNize's {header, moves[]} shape. The
// Gemini Schema enum uses upper-case type names. We deliberately leave fields optional so
// a partly-illegible sheet still returns the rows it could read.
var geminiResponseSchema = map[string]any{
	"type": "OBJECT",
	"properties": map[string]any{
		"header": map[string]any{
			"type": "OBJECT",
			"properties": map[string]any{
				"white": map[string]any{"type": "STRING"}, "black": map[string]any{"type": "STRING"},
				"event": map[string]any{"type": "STRING"}, "site": map[string]any{"type": "STRING"},
				"date": map[string]any{"type": "STRING"}, "round": map[string]any{"type": "STRING"},
				"board": map[string]any{"type": "STRING"}, "result": map[string]any{"type": "STRING"},
			},
		},
		"moves": map[string]any{
			"type": "ARRAY",
			"items": map[string]any{
				"type": "OBJECT",
				"properties": map[string]any{
					"no":    map[string]any{"type": "INTEGER"},
					"white": map[string]any{"type": "STRING"},
					"black": map[string]any{"type": "STRING"},
				},
			},
		},
	},
}

func (g *Gemini) Recognize(ctx context.Context, in ScoreSheetInput) (RecognitionResult, error) {
	img, mime := g.prepImage(in.Image, in.MimeType)

	parts := []geminiPart{{Text: buildPrompt(in)}}
	// Optional few-shot example images (multi-image input). Today's few-shot rows carry
	// only text, but image examples are supported when present.
	for _, ex := range in.FewShot {
		if ex.ImageBase64 != "" {
			parts = append(parts, geminiPart{InlineData: &geminiInlineData{MimeType: "image/jpeg", Data: ex.ImageBase64}})
		}
	}
	parts = append(parts, geminiPart{InlineData: &geminiInlineData{MimeType: mime, Data: base64.StdEncoding.EncodeToString(img)}})

	reqBody := geminiRequest{
		Contents: []geminiContent{{Role: "user", Parts: parts}},
		GenerationConfig: geminiGenerationConfig{
			Temperature:      0.1,
			ResponseMimeType: "application/json",
			ResponseSchema:   geminiResponseSchema,
			MaxOutputTokens:  g.MaxTokens,
			// Explicitly cap thinking so reasoning tokens cannot starve the JSON answer.
			ThinkingConfig: &geminiThinkingConfig{ThinkingBudget: g.ThinkingBudget},
		},
	}
	buf, _ := json.Marshal(reqBody)

	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent", g.Host, g.Model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		return RecognitionResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Goog-Api-Key", g.APIKey)

	resp, err := g.Client.Do(req)
	if err != nil {
		return RecognitionResult{}, fmt.Errorf("gemini request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return RecognitionResult{}, fmt.Errorf("gemini status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var gr geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&gr); err != nil {
		return RecognitionResult{}, fmt.Errorf("decode gemini envelope: %w", err)
	}
	if len(gr.Candidates) == 0 {
		if reason := gr.PromptFeedback.BlockReason; reason != "" {
			return RecognitionResult{}, fmt.Errorf("gemini blocked the request: %s", reason)
		}
		return RecognitionResult{}, fmt.Errorf("gemini returned no candidates")
	}

	var text strings.Builder
	for _, p := range gr.Candidates[0].Content.Parts {
		text.WriteString(p.Text)
	}
	raw := text.String()

	var mo modelOutput
	if err := json.Unmarshal([]byte(raw), &mo); err != nil {
		// A token cap can truncate the JSON; salvage the complete move objects.
		if tokens := salvageMoves(raw); len(tokens) > 0 {
			return RecognitionResult{MoveTokens: tokens, RawJSON: raw, Confidence: 0.3}, nil
		}
		return RecognitionResult{RawJSON: raw}, fmt.Errorf("decode model json: %w", err)
	}
	return RecognitionResult{
		Header:     mo.Header,
		MoveTokens: flattenModelMoves(mo),
		Confidence: 0.5, // models do not reliably self-report; the review loop is the safety net
		RawJSON:    raw,
	}, nil
}

// prepImage downscales oversized images and reports the MIME type to send. It returns the
// original bytes (with the detected MIME) when no resize is needed, and re-encodes to JPEG
// when it must shrink the image. On any decode error it falls back to the input bytes and
// MIME (defaulting to image/jpeg when unknown).
func (g *Gemini) prepImage(data []byte, mime string) ([]byte, string) {
	if mime == "" || mime == "application/octet-stream" {
		mime = "image/jpeg"
	}
	if g.MaxDim <= 0 {
		return data, mime
	}
	src, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return data, mime
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	longest := w
	if h > w {
		longest = h
	}
	if longest <= g.MaxDim {
		return data, mimeForFormat(format, mime)
	}
	scale := float64(g.MaxDim) / float64(longest)
	nw, nh := int(float64(w)*scale), int(float64(h)*scale)
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, b, xdraw.Over, nil)
	var out bytes.Buffer
	if err := jpeg.Encode(&out, dst, &jpeg.Options{Quality: 85}); err != nil {
		return data, mimeForFormat(format, mime)
	}
	return out.Bytes(), "image/jpeg"
}

func mimeForFormat(format, fallback string) string {
	switch format {
	case "jpeg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "gif":
		return "image/gif"
	case "webp":
		return "image/webp"
	default:
		return fallback
	}
}
