package chat

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

	"github.com/tranmh/pgnize/internal/engine"
)

const (
	chatDefaultMaxTokens = 1024
	chatDefaultTimeout   = 90 * time.Second
	chatDefaultMaxIters  = 5
	chatTemperature      = 0.4
)

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return n
		}
	}
	return def
}

// GeminiChatter is a Chatter backed by Gemini function-calling. It runs the multi-turn loop:
// the model emits functionCalls, this client validates+runs them against the engine, feeds
// the results back, and repeats until the model returns a text answer (or the iteration cap).
type GeminiChatter struct {
	Host     string
	Model    string
	APIKey   string
	Client   *http.Client
	Engine   engine.Engine
	MaxTokens int
	MaxIters  int
}

var _ Chatter = (*GeminiChatter)(nil)

// NewGemini builds a Gemini-backed conversational coach over the given engine.
func NewGemini(host, model, apiKey string, eng engine.Engine) *GeminiChatter {
	return &GeminiChatter{
		Host:      strings.TrimRight(host, "/"),
		Model:     model,
		APIKey:    apiKey,
		Client:    &http.Client{Timeout: chatDefaultTimeout},
		Engine:    eng,
		MaxTokens: envInt("CHAT_MAX_TOKENS", chatDefaultMaxTokens),
		MaxIters:  envInt("CHAT_MAX_TOOL_ITERS", chatDefaultMaxIters),
	}
}

func (c *GeminiChatter) Name() string { return "gemini:" + c.Model }

// ---- Gemini function-calling wire types (chat-local) ----

type fnCall struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}
type fnResponse struct {
	Name     string         `json:"name"`
	Response map[string]any `json:"response"`
}
type part struct {
	Text             string      `json:"text,omitempty"`
	FunctionCall     *fnCall     `json:"functionCall,omitempty"`
	FunctionResponse *fnResponse `json:"functionResponse,omitempty"`
}
type content struct {
	Role  string `json:"role,omitempty"`
	Parts []part `json:"parts"`
}
type fnDecl struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}
type tool struct {
	FunctionDeclarations []fnDecl `json:"functionDeclarations"`
}
type thinking struct {
	ThinkingBudget int `json:"thinkingBudget"`
}
type genConfig struct {
	Temperature     float64   `json:"temperature"`
	MaxOutputTokens int       `json:"maxOutputTokens,omitempty"`
	ThinkingConfig  *thinking `json:"thinkingConfig,omitempty"`
}
type geminiChatRequest struct {
	SystemInstruction *content   `json:"systemInstruction,omitempty"`
	Contents          []content  `json:"contents"`
	Tools             []tool     `json:"tools,omitempty"`
	GenerationConfig  genConfig  `json:"generationConfig"`
}
type geminiChatResponse struct {
	Candidates []struct {
		Content      content `json:"content"`
		FinishReason string  `json:"finishReason"`
	} `json:"candidates"`
	PromptFeedback struct {
		BlockReason string `json:"blockReason"`
	} `json:"promptFeedback"`
}

func (c *GeminiChatter) Respond(ctx context.Context, history []Message, user string, pctx Context) (Reply, error) {
	lang := normLang(pctx.Lang)

	contents := make([]content, 0, len(history)+8)
	for _, m := range history {
		contents = append(contents, content{Role: string(m.Role), Parts: []part{{Text: m.Text}}})
	}
	contents = append(contents, content{
		Role:  "user",
		Parts: []part{{Text: userPrompt(lang, pctx.FEN, pctx.Side, user)}},
	})

	sys := &content{Parts: []part{{Text: systemPrompt(lang)}}}
	tools := []tool{{FunctionDeclarations: toolDeclarations(lang)}}

	var trace []ToolCall
	for i := 0; i < c.MaxIters; i++ {
		resp, err := c.generate(ctx, sys, contents, tools)
		if err != nil {
			return Reply{}, err
		}
		if len(resp.Candidates) == 0 {
			if r := resp.PromptFeedback.BlockReason; r != "" {
				return Reply{}, fmt.Errorf("chat blocked: %s", r)
			}
			return Reply{}, fmt.Errorf("chat returned no candidates")
		}
		parts := resp.Candidates[0].Content.Parts

		// Collect any function calls in this turn (Gemini may emit several).
		var calls []part
		for _, p := range parts {
			if p.FunctionCall != nil {
				calls = append(calls, p)
			}
		}

		if len(calls) == 0 {
			// Final text answer.
			var text strings.Builder
			for _, p := range parts {
				text.WriteString(p.Text)
			}
			out := strings.TrimSpace(text.String())
			if out == "" {
				return Reply{}, fmt.Errorf("chat returned empty text")
			}
			return Reply{Text: out, Model: c.Name(), Calls: trace}, nil
		}

		// Echo the model's function-call turn, then answer with function responses.
		contents = append(contents, content{Role: "model", Parts: calls})
		respParts := make([]part, 0, len(calls))
		for _, p := range calls {
			result := dispatch(ctx, c.Engine, p.FunctionCall.Name, p.FunctionCall.Args, pctx.FEN)
			tc := ToolCall{Name: p.FunctionCall.Name, Args: p.FunctionCall.Args, Result: result}
			if e, ok := result["error"].(string); ok {
				tc.Err = e
			}
			trace = append(trace, tc)
			respParts = append(respParts, part{FunctionResponse: &fnResponse{Name: p.FunctionCall.Name, Response: result}})
		}
		contents = append(contents, content{Role: "user", Parts: respParts})
	}

	// Iteration cap hit: force a final text answer with tools disabled.
	resp, err := c.generate(ctx, sys, contents, nil)
	if err != nil {
		return Reply{}, err
	}
	if len(resp.Candidates) == 0 {
		return Reply{}, fmt.Errorf("chat returned no final answer")
	}
	var text strings.Builder
	for _, p := range resp.Candidates[0].Content.Parts {
		text.WriteString(p.Text)
	}
	out := strings.TrimSpace(text.String())
	if out == "" {
		return Reply{}, fmt.Errorf("chat returned empty text after tool loop")
	}
	return Reply{Text: out, Model: c.Name(), Calls: trace}, nil
}

func (c *GeminiChatter) generate(ctx context.Context, sys *content, contents []content, tools []tool) (geminiChatResponse, error) {
	reqBody := geminiChatRequest{
		SystemInstruction: sys,
		Contents:          contents,
		Tools:             tools,
		GenerationConfig: genConfig{
			Temperature:     chatTemperature,
			MaxOutputTokens: c.MaxTokens,
			ThinkingConfig:  &thinking{ThinkingBudget: 0},
		},
	}
	buf, _ := json.Marshal(reqBody)

	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent", c.Host, c.Model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		return geminiChatResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Goog-Api-Key", c.APIKey)

	resp, err := c.Client.Do(req)
	if err != nil {
		return geminiChatResponse{}, fmt.Errorf("gemini chat request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return geminiChatResponse{}, fmt.Errorf("gemini chat status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var gr geminiChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&gr); err != nil {
		return geminiChatResponse{}, fmt.Errorf("decode gemini chat envelope: %w", err)
	}
	return gr, nil
}
