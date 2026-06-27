package engine

import "testing"

func TestParseInfoCp(t *testing.T) {
	mpv, l, ok := parseInfo("info depth 18 seldepth 24 multipv 1 score cp 23 nodes 12345 pv e2e4 e7e5 g1f3")
	if !ok {
		t.Fatal("expected ok")
	}
	if mpv != 1 {
		t.Errorf("multipv = %d, want 1", mpv)
	}
	if l.Cp == nil || *l.Cp != 23 {
		t.Errorf("cp = %v, want 23", l.Cp)
	}
	if l.Mate != nil {
		t.Errorf("mate = %v, want nil", l.Mate)
	}
	if l.Depth != 18 {
		t.Errorf("depth = %d, want 18", l.Depth)
	}
	if l.BestMove != "e2e4" {
		t.Errorf("bestmove = %q, want e2e4", l.BestMove)
	}
	if len(l.PV) != 3 {
		t.Errorf("pv len = %d, want 3", len(l.PV))
	}
}

func TestParseInfoMateAndMultiPV(t *testing.T) {
	mpv, l, ok := parseInfo("info depth 5 multipv 2 score mate 3 pv d1h5 e8e7 h5e5")
	if !ok {
		t.Fatal("expected ok")
	}
	if mpv != 2 {
		t.Errorf("multipv = %d, want 2", mpv)
	}
	if l.Mate == nil || *l.Mate != 3 {
		t.Errorf("mate = %v, want 3", l.Mate)
	}
	if l.Cp != nil {
		t.Errorf("cp = %v, want nil", l.Cp)
	}
}

func TestParseInfoNegativeCpNoPV(t *testing.T) {
	_, l, ok := parseInfo("info depth 1 score cp -45")
	if !ok {
		t.Fatal("expected ok")
	}
	if l.Cp == nil || *l.Cp != -45 {
		t.Errorf("cp = %v, want -45", l.Cp)
	}
	if l.BestMove != "" {
		t.Errorf("bestmove = %q, want empty", l.BestMove)
	}
}

func TestParseInfoNoScore(t *testing.T) {
	if _, _, ok := parseInfo("info depth 1 currmove e2e4 currmovenumber 1"); ok {
		t.Error("expected !ok for scoreless info line")
	}
	if _, _, ok := parseInfo("bestmove e2e4"); ok {
		t.Error("expected !ok for non-info line")
	}
}

func TestParseBestMove(t *testing.T) {
	m, ok := parseBestMove("bestmove e2e4 ponder e7e5")
	if !ok || m != "e2e4" {
		t.Errorf("got (%q,%v), want (e2e4,true)", m, ok)
	}
	m, ok = parseBestMove("bestmove (none)")
	if !ok || m != "" {
		t.Errorf("got (%q,%v), want (\"\",true)", m, ok)
	}
	if _, ok := parseBestMove("info depth 1"); ok {
		t.Error("expected !ok for non-bestmove line")
	}
}

func TestAssembleOrdersByMultiPV(t *testing.T) {
	cp1, cp2, cp3 := 50, 30, 10
	lines := map[int]Line{
		3: {Cp: &cp3},
		1: {Cp: &cp1},
		2: {Cp: &cp2},
	}
	a := assemble("fen", lines)
	if len(a.Lines) != 3 {
		t.Fatalf("len = %d, want 3", len(a.Lines))
	}
	if *a.Lines[0].Cp != 50 || *a.Lines[1].Cp != 30 || *a.Lines[2].Cp != 10 {
		t.Errorf("lines out of order: %v", a.Lines)
	}
}
