package recognition

import (
	"bytes"
	"image"
	"image/jpeg"
	"testing"
)

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
