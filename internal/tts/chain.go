package tts

import (
	"context"
	"fmt"
	"strings"
)

// Chain is a Synthesizer that tries each backend in order and returns the first success
// — this implements "Gemini → Piper". On the chain reporting a Name, it reports the
// primary's name (the configured intent); Synthesize errors only if every backend fails.
type Chain struct {
	Synths []Synthesizer
}

// NewChain builds a chain from the given synthesizers in priority order.
func NewChain(synths ...Synthesizer) *Chain { return &Chain{Synths: synths} }

func (c *Chain) Name() string {
	if len(c.Synths) == 0 {
		return "chain:empty"
	}
	return c.Synths[0].Name()
}

func (c *Chain) Voice(lang string) string {
	if len(c.Synths) == 0 {
		return ""
	}
	return c.Synths[0].Voice(lang)
}

func (c *Chain) Synthesize(ctx context.Context, in SpeakInput) (Audio, error) {
	if len(c.Synths) == 0 {
		return Audio{}, fmt.Errorf("tts chain is empty")
	}
	var errs []string
	for _, s := range c.Synths {
		audio, err := s.Synthesize(ctx, in)
		if err == nil {
			return audio, nil
		}
		errs = append(errs, fmt.Sprintf("%s: %v", s.Name(), err))
	}
	return Audio{}, fmt.Errorf("all tts backends failed: %s", strings.Join(errs, "; "))
}
