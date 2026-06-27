package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/tranmh/pgnize/internal/engine"
)

// OllamaChatter is a best-effort conversational coach over a local Ollama server using its
// /api/chat tool-calling support. Tool support is model-dependent; if the model never emits
// tool calls it simply answers from the prompt. Gemini is the primary backend.
type OllamaChatter struct {
	Host     string
	Model    string
	Client   *http.Client
	Engine   engine.Engine
	MaxIters int
}

var _ Chatter = (*OllamaChatter)(nil)

// NewOllama builds an Ollama-backed conversational coach over the given engine.
func NewOllama(host, model string, eng engine.Engine) *OllamaChatter {
	return &OllamaChatter{
		Host:     strings.TrimRight(host, "/"),
		Model:    model,
		Client:   &http.Client{Timeout: chatDefaultTimeout},
		Engine:   eng,
		MaxIters: envInt("CHAT_MAX_TOOL_ITERS", chatDefaultMaxIters),
	}
}

func (c *OllamaChatter) Name() string { return "ollama:" + c.Model }

type ollamaFn struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}
type ollamaToolCall struct {
	Function ollamaFn `json:"function"`
}
type ollamaMessage struct {
	Role      string           `json:"role"`
	Content   string           `json:"content"`
	ToolCalls []ollamaToolCall `json:"tool_calls,omitempty"`
}
type ollamaTool struct {
	Type     string `json:"type"`
	Function fnDecl `json:"function"`
}
type ollamaChatRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Tools    []ollamaTool    `json:"tools,omitempty"`
	Stream   bool            `json:"stream"`
	Options  map[string]any  `json:"options,omitempty"`
}
type ollamaChatResponse struct {
	Message ollamaMessage `json:"message"`
}

// ollamaTools renders the shared declarations with lower-case JSON-schema type names.
func ollamaTools(lang string) []ollamaTool {
	decls := toolDeclarations(lang)
	out := make([]ollamaTool, 0, len(decls))
	for _, d := range decls {
		out = append(out, ollamaTool{Type: "function", Function: fnDecl{
			Name:        d.Name,
			Description: d.Description,
			Parameters:  lowerCaseTypes(d.Parameters),
		}})
	}
	return out
}

// lowerCaseTypes deep-copies a Gemini-style schema (OBJECT/STRING/INTEGER) into the
// JSON-schema lower-case form Ollama expects (object/string/integer).
func lowerCaseTypes(v any) map[string]any {
	m, ok := v.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	out := make(map[string]any, len(m))
	for k, val := range m {
		switch {
		case k == "type":
			if s, ok := val.(string); ok {
				out[k] = strings.ToLower(s)
			} else {
				out[k] = val
			}
		case k == "properties":
			if props, ok := val.(map[string]any); ok {
				np := make(map[string]any, len(props))
				for pk, pv := range props {
					np[pk] = lowerCaseTypes(pv)
				}
				out[k] = np
			}
		default:
			out[k] = val
		}
	}
	return out
}

func (c *OllamaChatter) Respond(ctx context.Context, history []Message, user string, pctx Context) (Reply, error) {
	lang := normLang(pctx.Lang)
	msgs := []ollamaMessage{{Role: "system", Content: systemPrompt(lang)}}
	for _, m := range history {
		role := "user"
		if m.Role == RoleCoach {
			role = "assistant"
		}
		msgs = append(msgs, ollamaMessage{Role: role, Content: m.Text})
	}
	msgs = append(msgs, ollamaMessage{Role: "user", Content: userPrompt(lang, pctx.FEN, pctx.Side, user)})

	tools := ollamaTools(lang)
	var trace []ToolCall
	for i := 0; i < c.MaxIters; i++ {
		resp, err := c.chat(ctx, msgs, tools)
		if err != nil {
			return Reply{}, err
		}
		if len(resp.Message.ToolCalls) == 0 {
			out := strings.TrimSpace(resp.Message.Content)
			if out == "" {
				return Reply{}, fmt.Errorf("ollama chat returned empty text")
			}
			return Reply{Text: out, Model: c.Name(), Calls: trace}, nil
		}
		msgs = append(msgs, resp.Message)
		for _, tcall := range resp.Message.ToolCalls {
			result := dispatch(ctx, c.Engine, tcall.Function.Name, tcall.Function.Arguments, pctx.FEN)
			tc := ToolCall{Name: tcall.Function.Name, Args: tcall.Function.Arguments, Result: result}
			if e, ok := result["error"].(string); ok {
				tc.Err = e
			}
			trace = append(trace, tc)
			payload, _ := json.Marshal(result)
			msgs = append(msgs, ollamaMessage{Role: "tool", Content: string(payload)})
		}
	}

	resp, err := c.chat(ctx, msgs, nil)
	if err != nil {
		return Reply{}, err
	}
	out := strings.TrimSpace(resp.Message.Content)
	if out == "" {
		return Reply{}, fmt.Errorf("ollama chat returned empty text after tool loop")
	}
	return Reply{Text: out, Model: c.Name(), Calls: trace}, nil
}

func (c *OllamaChatter) chat(ctx context.Context, msgs []ollamaMessage, tools []ollamaTool) (ollamaChatResponse, error) {
	reqBody := ollamaChatRequest{
		Model:    c.Model,
		Messages: msgs,
		Tools:    tools,
		Stream:   false,
		Options:  map[string]any{"temperature": chatTemperature},
	}
	buf, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Host+"/api/chat", bytes.NewReader(buf))
	if err != nil {
		return ollamaChatResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Client.Do(req)
	if err != nil {
		return ollamaChatResponse{}, fmt.Errorf("ollama chat request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ollamaChatResponse{}, fmt.Errorf("ollama chat status %d", resp.StatusCode)
	}
	var or ollamaChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&or); err != nil {
		return ollamaChatResponse{}, fmt.Errorf("decode ollama chat envelope: %w", err)
	}
	return or, nil
}
