package engine

import (
	"context"
	"hash/fnv"
)

// Fake is a deterministic Engine for tests and CI: no subprocess, no binary, no network.
// Its output is a pure function of the FEN string (so tests can assert stable values), and
// specific positions can be seeded to return a chosen Analysis (e.g. a known mate).
type Fake struct {
	overrides map[string]Analysis
}

var _ Engine = (*Fake)(nil)

// NewFake returns a Fake engine.
func NewFake() *Fake { return &Fake{overrides: map[string]Analysis{}} }

// Seed makes Analyze return a fixed Analysis for an exact FEN string. Useful for driving
// the "is there a mate" path deterministically in tests.
func (f *Fake) Seed(fen string, a Analysis) { f.overrides[fen] = a }

func (f *Fake) Name() string { return "fake" }

func (f *Fake) Close() error { return nil }

// Analyze returns a seeded Analysis when present, otherwise a deterministic one derived
// from the FEN. The eval spans roughly [-150, +149] cp and never reports a mate (so the
// default path is unambiguous); seed a FEN to exercise mate handling.
func (f *Fake) Analyze(_ context.Context, fen string, opts Options) (Analysis, error) {
	if a, ok := f.overrides[fen]; ok {
		return a, nil
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(fen))
	sum := h.Sum32()
	cp := int(sum%300) - 150
	depth := 12 + int(sum%9)

	n := opts.MultiPV
	if n <= 0 {
		n = 1
	}
	lines := make([]Line, 0, n)
	for i := 0; i < n; i++ {
		c := cp - i*15 // weaker alternatives, still deterministic
		lines = append(lines, Line{
			Cp:       &c,
			Depth:    depth,
			PV:       []string{"e2e4", "e7e5"},
			BestMove: "e2e4",
		})
	}
	return Analysis{FEN: fen, Lines: lines}, nil
}
