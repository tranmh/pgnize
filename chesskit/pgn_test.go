package chesskit

import (
	"strings"
	"testing"
)

func TestParsePGN_SingleGame(t *testing.T) {
	pgn := `[Event "Test Event"]
[Site "Berlin"]
[Date "2026.06.19"]
[Round "1"]
[White "Alice"]
[Black "Bob"]
[Result "1-0"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 1-0
`
	games, err := ParsePGN(pgn)
	if err != nil {
		t.Fatalf("ParsePGN: %v", err)
	}
	if len(games) != 1 {
		t.Fatalf("games = %d, want 1", len(games))
	}
	g := games[0]
	if g.Header.Event != "Test Event" || g.Header.White != "Alice" || g.Header.Black != "Bob" {
		t.Errorf("header mismatch: %+v", g.Header)
	}
	if g.Header.Result != ResultWhiteWin {
		t.Errorf("result = %q, want 1-0", g.Header.Result)
	}
	if len(g.Moves) != 6 {
		t.Fatalf("moves = %d, want 6", len(g.Moves))
	}
	if g.Moves[0].SAN != "e4" || g.Moves[4].SAN != "Bb5" {
		t.Errorf("moves wrong: %v", sansOf(g.Moves))
	}
}

func TestParsePGN_TwoGameBundle(t *testing.T) {
	pgn := `[Event "G1"]
[White "A"]
[Black "B"]
[Result "1-0"]

1. e4 e5 1-0

[Event "G2"]
[White "C"]
[Black "D"]
[Result "0-1"]

1. d4 d5 0-1
`
	games, err := ParsePGN(pgn)
	if err != nil {
		t.Fatalf("ParsePGN: %v", err)
	}
	if len(games) != 2 {
		t.Fatalf("games = %d, want 2", len(games))
	}
	if games[0].Header.Event != "G1" || games[1].Header.Event != "G2" {
		t.Errorf("events: %q %q", games[0].Header.Event, games[1].Header.Event)
	}
	if games[0].Header.Result != ResultWhiteWin || games[1].Header.Result != ResultBlackWin {
		t.Errorf("results: %q %q", games[0].Header.Result, games[1].Header.Result)
	}
}

func TestParsePGN_ClockComments(t *testing.T) {
	pgn := `[Result "*"]

1. e4 {[%clk 1:02:03]} e5 {[%clk 0:59:59]} 2. Nf3 {[%clk 1:01:00]} *
`
	games, err := ParsePGN(pgn)
	if err != nil {
		t.Fatalf("ParsePGN: %v", err)
	}
	if len(games) != 1 {
		t.Fatalf("games = %d, want 1", len(games))
	}
	mv := games[0].Moves
	if len(mv) != 3 {
		t.Fatalf("moves = %d, want 3", len(mv))
	}
	if mv[0].ClockSec == nil || *mv[0].ClockSec != 1*3600+2*60+3 {
		t.Errorf("move0 clock = %v, want %d", mv[0].ClockSec, 1*3600+2*60+3)
	}
	if mv[1].ClockSec == nil || *mv[1].ClockSec != 59*60+59 {
		t.Errorf("move1 clock = %v", mv[1].ClockSec)
	}
	if mv[2].ClockSec == nil || *mv[2].ClockSec != 3600+60 {
		t.Errorf("move2 clock = %v", mv[2].ClockSec)
	}
}

func TestParsePGN_IllegalMoveTruncates(t *testing.T) {
	// 2... e5 after 2.e5 is illegal (e5 already occupied path); craft a clear illegal move.
	pgn := `[Result "*"]

1. e4 e5 2. Nf3 Nc6 3. QZ9 Bc5 *
`
	games, err := ParsePGN(pgn)
	if err != nil {
		t.Fatalf("ParsePGN should not fail: %v", err)
	}
	if len(games) != 1 {
		t.Fatalf("games = %d, want 1", len(games))
	}
	// 4 legal moves before the garbage token "QZ9".
	if len(games[0].Moves) != 4 {
		t.Fatalf("moves = %d, want 4 (truncated), got %v", len(games[0].Moves), sansOf(games[0].Moves))
	}
}

func TestParsePGN_IllegalChessMoveTruncates(t *testing.T) {
	// A syntactically-valid but illegal move mid-game must truncate.
	pgn := `[Result "*"]

1. e4 e5 2. Ke2 Nf6 3. Kd4 *
`
	games, err := ParsePGN(pgn)
	if err != nil {
		t.Fatalf("ParsePGN: %v", err)
	}
	// e4 e5 Ke2 Nf6 are legal (4); Kd4 (king to d4) is illegal -> truncate.
	if len(games[0].Moves) != 4 {
		t.Fatalf("moves = %d, want 4, got %v", len(games[0].Moves), sansOf(games[0].Moves))
	}
}

func TestParsePGN_PreservesExtraTags(t *testing.T) {
	pgn := `[Event "E"]
[White "A"]
[Black "B"]
[Result "*"]
[ECO "C50"]
[WhiteElo "2400"]

1. e4 e5 *
`
	games, _ := ParsePGN(pgn)
	g := games[0]
	if g.Header.Extra["ECO"] != "C50" || g.Header.Extra["WhiteElo"] != "2400" {
		t.Fatalf("extra tags not preserved: %v", g.Header.Extra)
	}
}

func TestWritePGN_RoundTrip(t *testing.T) {
	in := Game{
		Header: Header{
			Event:  "Round Trip",
			Site:   "Nowhere",
			Date:   "2026.01.01",
			Round:  "2",
			White:  "Winner",
			Black:  "Loser",
			Result: ResultWhiteWin,
			Extra:  map[string]string{"ECO": "C20"},
		},
		StartFEN: StartingFEN(),
	}
	// Build moves via ApplyMoves so FENs are correct.
	sans := []SAN{"e4", "e5", "Nf3", "Nc6", "Bb5", "a6"}
	positions, err, _ := ApplyMoves(StartingFEN(), sans)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	from := StartingFEN()
	for i, s := range sans {
		in.Moves = append(in.Moves, Move{SAN: s, FromFEN: from, ToFEN: positions[i]})
		from = positions[i]
	}

	out, err := WritePGN(in)
	if err != nil {
		t.Fatalf("WritePGN: %v", err)
	}
	games, err := ParsePGN(out)
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	if len(games) != 1 {
		t.Fatalf("re-parse games = %d", len(games))
	}
	rt := games[0]

	// Seven-Tag Roster preserved.
	if rt.Header.Event != in.Header.Event || rt.Header.Site != in.Header.Site ||
		rt.Header.Date != in.Header.Date || rt.Header.Round != in.Header.Round ||
		rt.Header.White != in.Header.White || rt.Header.Black != in.Header.Black ||
		rt.Header.Result != in.Header.Result {
		t.Errorf("roster mismatch:\nin=%+v\nrt=%+v", in.Header, rt.Header)
	}
	// Moves preserved.
	if len(rt.Moves) != len(in.Moves) {
		t.Fatalf("moves len %d != %d", len(rt.Moves), len(in.Moves))
	}
	for i := range in.Moves {
		if rt.Moves[i].SAN != in.Moves[i].SAN {
			t.Errorf("move %d: %q != %q", i, rt.Moves[i].SAN, in.Moves[i].SAN)
		}
	}
}

func TestWritePGN_ClockEmitted(t *testing.T) {
	sec := 3661 // 1:01:01
	g := Game{
		Header:   Header{White: "A", Black: "B", Result: ResultOngoing},
		StartFEN: StartingFEN(),
		Moves: []Move{
			{SAN: "e4", ClockSec: &sec},
		},
	}
	out, err := WritePGN(g)
	if err != nil {
		t.Fatalf("WritePGN: %v", err)
	}
	if !strings.Contains(out, "{[%clk 1:01:01]}") {
		t.Fatalf("clock annotation missing in:\n%s", out)
	}
	// Round-trip the clock.
	games, _ := ParsePGN(out)
	if games[0].Moves[0].ClockSec == nil || *games[0].Moves[0].ClockSec != sec {
		t.Fatalf("clock round-trip failed: %v", games[0].Moves[0].ClockSec)
	}
}

func TestWritePGN_StableRosterOrder(t *testing.T) {
	g := Game{Header: Header{White: "A", Black: "B", Result: ResultDraw}, StartFEN: StartingFEN()}
	out, _ := WritePGN(g)
	order := []string{"[Event ", "[Site ", "[Date ", "[Round ", "[White ", "[Black ", "[Result "}
	idx := -1
	for _, tag := range order {
		i := strings.Index(out, tag)
		if i < 0 {
			t.Fatalf("missing tag %q in:\n%s", tag, out)
		}
		if i < idx {
			t.Fatalf("tag %q out of order", tag)
		}
		idx = i
	}
}

func TestWriteBundlePGN_TwoGames(t *testing.T) {
	mk := func(ev string, res Result) Game {
		return Game{Header: Header{Event: ev, White: "A", Black: "B", Result: res}, StartFEN: StartingFEN()}
	}
	g1 := mk("First", ResultWhiteWin)
	g2 := mk("Second", ResultBlackWin)
	out, err := WriteBundlePGN([]Game{g1, g2})
	if err != nil {
		t.Fatalf("WriteBundlePGN: %v", err)
	}
	games, err := ParsePGN(out)
	if err != nil {
		t.Fatalf("re-parse bundle: %v", err)
	}
	if len(games) != 2 {
		t.Fatalf("bundle games = %d, want 2", len(games))
	}
	if games[0].Header.Event != "First" || games[1].Header.Event != "Second" {
		t.Errorf("bundle events: %q %q", games[0].Header.Event, games[1].Header.Event)
	}
}

func TestParsePGN_BlackToMoveStart(t *testing.T) {
	// Game starting from a FEN where Black is to move; write then re-parse.
	startFEN := FEN("rnbqkbnr/pppp1ppp/8/4p3/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1")
	positions, err, _ := ApplyMoves(startFEN, []SAN{"Nc6", "Nf3"})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	g := Game{
		Header:   Header{White: "A", Black: "B", Result: ResultOngoing},
		StartFEN: startFEN,
		Moves: []Move{
			{SAN: "Nc6", FromFEN: startFEN, ToFEN: positions[0]},
			{SAN: "Nf3", FromFEN: positions[0], ToFEN: positions[1]},
		},
	}
	out, err := WritePGN(g)
	if err != nil {
		t.Fatalf("WritePGN: %v", err)
	}
	if !strings.Contains(out, "1...") {
		t.Errorf("expected black-to-move marker '1...' in:\n%s", out)
	}
	games, err := ParsePGN(out)
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	if len(games[0].Moves) != 2 || games[0].Moves[0].SAN != "Nc6" {
		t.Fatalf("black-start round trip failed: %v", sansOf(games[0].Moves))
	}
}

func sansOf(moves []Move) []SAN {
	out := make([]SAN, len(moves))
	for i, m := range moves {
		out[i] = m.SAN
	}
	return out
}
