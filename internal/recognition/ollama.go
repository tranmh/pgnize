package recognition

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	xdraw "golang.org/x/image/draw"

	"github.com/tranmh/pgnize/internal/domain"
)

// Defaults tuned from CPU benchmarking (minicpm-v): strict JSON-schema decoding caused
// runaway generation (>10 min); simple json mode + a token cap returns in ~30-60s.
const (
	// On CPU, minicpm-v emits ~3 tokens/sec, so the token cap dominates latency.
	// 512 tokens (~30 move pairs of JSON) returns in ~2-3 min and the model usually
	// stops earlier on its own; 1024 overran a 4-min timeout.
	defaultNumPredict = 512
	defaultTimeout    = 5 * time.Minute
	defaultMaxDim     = 1600 // longest image edge sent to the model
)

// Ollama is a Recognizer backed by a local Ollama server running a vision model.
type Ollama struct {
	Host       string
	Model      string
	Client     *http.Client
	NumPredict int
	MaxDim     int
}

// NewOllama builds an Ollama-backed recognizer with CPU-friendly defaults.
// OLLAMA_NUM_PREDICT and OLLAMA_MAX_DIM override the token cap and downscale size.
func NewOllama(host, model string) *Ollama {
	return &Ollama{
		Host:       host,
		Model:      model,
		Client:     &http.Client{Timeout: defaultTimeout},
		NumPredict: envInt("OLLAMA_NUM_PREDICT", defaultNumPredict),
		MaxDim:     envInt("OLLAMA_MAX_DIM", defaultMaxDim),
	}
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func (o *Ollama) Name() string { return "ollama:" + o.Model }

type ollamaRequest struct {
	Model   string         `json:"model"`
	Prompt  string         `json:"prompt"`
	Images  []string       `json:"images,omitempty"`
	Format  string         `json:"format,omitempty"` // "json": fast + stops naturally (see const block)
	Stream  bool           `json:"stream"`
	Options map[string]any `json:"options,omitempty"`
}

type ollamaResponse struct {
	Response string `json:"response"`
}

// modelMove keys off white/black move text. We deliberately omit the "no" field: the
// model returns it inconsistently as int or string, which would break struct decoding.
type modelMove struct {
	White string `json:"white"`
	Black string `json:"black"`
}

type modelOutput struct {
	Header domain.Header `json:"header"`
	Moves  []modelMove   `json:"moves"`
}

func (o *Ollama) Recognize(ctx context.Context, in ScoreSheetInput) (RecognitionResult, error) {
	img := o.downscale(in.Image)
	reqBody := ollamaRequest{
		Model:   o.Model,
		Prompt:  buildPrompt(in),
		Images:  []string{base64.StdEncoding.EncodeToString(img)},
		Format:  "json",
		Stream:  false,
		Options: map[string]any{"temperature": 0.1, "num_predict": o.NumPredict},
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
		// A num_predict cap can truncate the JSON mid-output. Rather than lose the
		// whole transcription, salvage the complete move objects that did arrive.
		if tokens := salvageMoves(or.Response); len(tokens) > 0 {
			return RecognitionResult{MoveTokens: tokens, RawJSON: or.Response, Confidence: 0.3}, nil
		}
		return RecognitionResult{RawJSON: or.Response}, fmt.Errorf("decode model json: %w", err)
	}
	return RecognitionResult{
		Header:     mo.Header,
		MoveTokens: flattenModelMoves(mo),
		Confidence: 0.5, // local models do not self-report; the review loop is the safety net
		RawJSON:    or.Response,
	}, nil
}

func flattenModelMoves(mo modelOutput) []MoveToken {
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

// salvageMoves recovers move tokens from truncated/invalid JSON by extracting every
// flat {...} object (no nested braces) and keeping those that decode to a move with a
// white/black field. The trailing incomplete object is simply skipped.
func salvageMoves(raw string) []MoveToken {
	// Only scan within the "moves" array; the header object (which also carries
	// white/black player names) precedes it and must not be mistaken for a move.
	if mi := strings.Index(raw, `"moves"`); mi >= 0 {
		raw = raw[mi:]
	}
	type frame struct {
		start int
		child bool
	}
	var stack []frame
	var objs []string
	for i := 0; i < len(raw); i++ {
		switch raw[i] {
		case '{':
			if n := len(stack); n > 0 {
				stack[n-1].child = true
			}
			stack = append(stack, frame{start: i})
		case '}':
			if n := len(stack); n > 0 {
				f := stack[n-1]
				stack = stack[:n-1]
				if !f.child { // innermost (leaf) object
					objs = append(objs, raw[f.start:i+1])
				}
			}
		}
	}
	var moves []modelMove
	for _, o := range objs {
		var m modelMove
		if json.Unmarshal([]byte(o), &m) == nil && (m.White != "" || m.Black != "") {
			moves = append(moves, m)
		}
	}
	return flattenModelMoves(modelOutput{Moves: moves})
}

// downscale shrinks images whose longest edge exceeds MaxDim, re-encoding as JPEG.
// On any decode error it returns the original bytes unchanged.
func (o *Ollama) downscale(data []byte) []byte {
	maxDim := o.MaxDim
	if maxDim <= 0 {
		return data
	}
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return data
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	longest := w
	if h > w {
		longest = h
	}
	if longest <= maxDim {
		return data
	}
	scale := float64(maxDim) / float64(longest)
	nw, nh := int(float64(w)*scale), int(float64(h)*scale)
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, b, xdraw.Over, nil)
	var out bytes.Buffer
	if err := jpeg.Encode(&out, dst, &jpeg.Options{Quality: 85}); err != nil {
		return data
	}
	return out.Bytes()
}
