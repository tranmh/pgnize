package recognition

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewOllamaDefaultTimeoutAndKeepAlive(t *testing.T) {
	o := NewOllama("h", "m")
	if o.Client.Timeout != defaultTimeout {
		t.Fatalf("default client timeout = %s, want %s", o.Client.Timeout, defaultTimeout)
	}
	if o.KeepAlive != defaultKeepAlive {
		t.Fatalf("default keep-alive = %q, want %q", o.KeepAlive, defaultKeepAlive)
	}
}

func TestNewOllamaTimeoutFromEnv(t *testing.T) {
	t.Setenv("OLLAMA_TIMEOUT_SEC", "720")
	t.Setenv("OLLAMA_KEEP_ALIVE", "1h")
	o := NewOllama("h", "m")
	if o.Client.Timeout != 720*time.Second {
		t.Fatalf("client timeout = %s, want 720s", o.Client.Timeout)
	}
	if o.KeepAlive != "1h" {
		t.Fatalf("keep-alive = %q, want %q", o.KeepAlive, "1h")
	}
}

func TestOllamaRequestCarriesKeepAlive(t *testing.T) {
	// keep_alive must be serialized so the model stays resident between sheets
	// (a cold reload of a multi-GB VLM otherwise blows the recognition budget).
	o := NewOllama("h", "m")
	body, err := json.Marshal(ollamaRequest{Model: o.Model, KeepAlive: o.KeepAlive})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(body, []byte(`"keep_alive":"30m"`)) {
		t.Fatalf("keep_alive missing from request body: %s", body)
	}
}

func TestSalvageMovesFromTruncatedJSON(t *testing.T) {
	// Mid-output truncation (num_predict cap): header + two complete moves, then a
	// third move object cut off. We must recover the four complete half-moves.
	raw := `{
	  "header": {"event": "Club", "white": "A", "black": "B"},
	  "moves": [
	    {"no": 1, "White": "e4", "Black": "e5"},
	    {"no": 2, "White": "Sf3", "Black": "Sc6"},
	    {"no": 3, "White": "Lb`
	tokens := salvageMoves(raw)
	if len(tokens) != 4 {
		t.Fatalf("want 4 salvaged half-moves, got %d: %+v", len(tokens), tokens)
	}
	wantText := []string{"e4", "e5", "Sf3", "Sc6"}
	wantSide := []string{SideWhite, SideBlack, SideWhite, SideBlack}
	for i, tok := range tokens {
		if tok.Text != wantText[i] || tok.Side != wantSide[i] {
			t.Errorf("token %d = {%q,%s}, want {%q,%s}", i, tok.Text, tok.Side, wantText[i], wantSide[i])
		}
	}
}

func TestSalvageMovesIgnoresHeaderOnlyJSON(t *testing.T) {
	// No move objects with white/black -> nothing to salvage.
	if got := salvageMoves(`{"header":{"event":"X"},"moves":[`); len(got) != 0 {
		t.Fatalf("expected 0 tokens, got %d", len(got))
	}
}

func TestOllamaRecognizePositionParsesGrid(t *testing.T) {
	var gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		inner := `{"grid":["....k...","........","........","........","........","........","........","....K..R"],` +
			`"sideToMove":"black","orientation":"white_bottom"}`
		resp := map[string]any{"response": inner}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	o := NewOllama(srv.URL, "minicpm-v")
	o.MaxDim = 0 // skip image decode/resize
	res, err := o.RecognizePosition(context.Background(), PositionInput{Image: []byte("img"), MimeType: "image/png"})
	if err != nil {
		t.Fatalf("RecognizePosition: %v", err)
	}
	if gotPath != "/api/generate" {
		t.Errorf("path = %q", gotPath)
	}
	if !strings.Contains(gotBody, "single chess position") {
		t.Errorf("request missing position prompt: %s", gotBody)
	}
	if len(res.Grid) != 8 || res.Grid[0] != "....k..." || res.Grid[7] != "....K..R" {
		t.Fatalf("grid = %+v", res.Grid)
	}
	if res.SideToMove != "black" {
		t.Errorf("sideToMove = %q", res.SideToMove)
	}
	fen, err := AssembleFEN(res)
	if err != nil || fen != "4k3/8/8/8/8/8/8/4K2R b - - 0 1" {
		t.Fatalf("AssembleFEN = %q (%v)", fen, err)
	}
}

// TestOllamaRecognizeSendsExtraImages proves a multi-image submission reaches Ollama as a
// multi-element images[] array: primary + each Extra blob, in order.
func TestOllamaRecognizeSendsExtraImages(t *testing.T) {
	var gotImages []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req ollamaRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		gotImages = req.Images
		resp := map[string]any{"response": `{"header":{},"moves":[]}`}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	o := NewOllama(srv.URL, "minicpm-v")
	o.MaxDim = 0 // skip downscale; bytes pass through verbatim
	_, err := o.Recognize(context.Background(), ScoreSheetInput{
		Image:    []byte("primary"),
		MimeType: "image/png",
		Extra: []ImageBlob{
			{Data: []byte("extra-1"), MimeType: "image/png"},
			{Data: []byte("extra-2"), MimeType: "image/jpeg"},
		},
	})
	if err != nil {
		t.Fatalf("Recognize: %v", err)
	}
	if len(gotImages) != 3 { // primary + 2 extras
		t.Fatalf("images = %d, want 3: %v", len(gotImages), gotImages)
	}
	// Order must be preserved: primary first, then extras as submitted.
	want := []string{
		base64.StdEncoding.EncodeToString([]byte("primary")),
		base64.StdEncoding.EncodeToString([]byte("extra-1")),
		base64.StdEncoding.EncodeToString([]byte("extra-2")),
	}
	for i, w := range want {
		if gotImages[i] != w {
			t.Errorf("image[%d] mismatch", i)
		}
	}
}

// TestOllamaRecognizePositionSendsExtraImages is the position-pipeline counterpart.
func TestOllamaRecognizePositionSendsExtraImages(t *testing.T) {
	var gotImages []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req ollamaRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		gotImages = req.Images
		resp := map[string]any{"response": `{"grid":[],"sideToMove":"white","orientation":"white_bottom"}`}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	o := NewOllama(srv.URL, "minicpm-v")
	o.MaxDim = 0
	_, err := o.RecognizePosition(context.Background(), PositionInput{
		Image:    []byte("primary"),
		MimeType: "image/png",
		Extra:    []ImageBlob{{Data: []byte("extra-1"), MimeType: "image/png"}},
	})
	if err != nil {
		t.Fatalf("RecognizePosition: %v", err)
	}
	if len(gotImages) != 2 { // primary + 1 extra
		t.Fatalf("images = %d, want 2: %v", len(gotImages), gotImages)
	}
}

func TestDownscaleShrinksLargeImageOnly(t *testing.T) {
	o := NewOllama("", "")

	// A 4000x3000 image must be shrunk so its longest edge <= MaxDim (1600).
	big := image.NewRGBA(image.Rect(0, 0, 4000, 3000))
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, big, nil); err != nil {
		t.Fatal(err)
	}
	out := o.downscale(buf.Bytes())
	img, _, err := image.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("decode downscaled: %v", err)
	}
	if w := img.Bounds().Dx(); w != 1600 {
		t.Fatalf("downscaled width = %d, want 1600", w)
	}

	// A small image is returned unchanged (same bytes).
	small := image.NewRGBA(image.Rect(0, 0, 800, 600))
	var sbuf bytes.Buffer
	_ = jpeg.Encode(&sbuf, small, nil)
	if got := o.downscale(sbuf.Bytes()); !bytes.Equal(got, sbuf.Bytes()) {
		t.Fatal("small image should be returned unchanged")
	}
}
