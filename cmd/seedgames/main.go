// Command seedgames populates the demo user's library with sample saved games
// spread across a few players, for local manual testing. Throwaway dev tool.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/tranmh/chesskit"
	"github.com/tranmh/pgnize/internal/config"
	"github.com/tranmh/pgnize/internal/domain"
	"github.com/tranmh/pgnize/internal/recognition"
	"github.com/tranmh/pgnize/internal/store"
)

// Three players to spread the games across.
var players = []string{"Anna Schmidt", "Bernd Müller", "Clara Weber"}

// Real, legal opening lines (SAN). Reconcile validates them and computes FENs.
var lines = []string{
	"e4 e5 Nf3 Nc6 Bc4 Bc5 c3 Nf6 d3 d6 O-O O-O Re1 a6 Bb3 Ba7",
	"e4 e5 Nf3 Nc6 Bb5 a6 Ba4 Nf6 O-O Be7 Re1 b5 Bb3 d6 c3 O-O h3 Na5",
	"e4 c5 Nf3 d6 d4 cxd4 Nxd4 Nf6 Nc3 a6 Be2 e5 Nb3 Be7 O-O O-O",
	"d4 d5 c4 e6 Nc3 Nf6 Bg5 Be7 e3 O-O Nf3 h6 Bh4 b6 cxd5 Nxd5",
	"e4 e6 d4 d5 Nc3 Bb4 e5 c5 a3 Bxc3+ bxc3 Ne7 Qg4 O-O",
	"e4 c6 d4 d5 Nc3 dxe4 Nxe4 Bf5 Ng3 Bg6 h4 h6 Nf3 Nd7",
	"d4 Nf6 c4 g6 Nc3 Bg7 e4 d6 Nf3 O-O Be2 e5 O-O Nc6 d5 Ne7",
	"c4 e5 Nc3 Nf6 Nf3 Nc6 g3 d5 cxd5 Nxd5 Bg2 Nb6 O-O Be7",
}

var events = []string{
	"Vereinsmeisterschaft Trier",
	"Bezirksliga Mosel",
	"Stadtpokal Koblenz",
	"Schnellschach-Open Mainz",
}

var results = []string{"1-0", "0-1", "1/2-1/2", "1-0", "0-1"}

func main() {
	n := flag.Int("n", 100, "number of games to seed")
	email := flag.String("email", "demo@pgnize.local", "owner account email")
	reset := flag.Bool("reset", true, "delete the owner's existing games first (idempotent re-seed)")
	flag.Parse()

	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		fatal("config", err)
	}
	st, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		fatal("store", err)
	}
	defer st.Close()

	user, err := st.GetUserByEmail(ctx, *email)
	if err != nil {
		fatal("lookup user", err)
	}
	uid := user.ID

	if *reset {
		if _, err := st.Pool.Exec(ctx, `DELETE FROM games WHERE user_id=$1`, uid); err != nil {
			fatal("reset games", err)
		}
	}

	startFEN := string(chesskit.StartingFEN())

	created := 0
	for i := 0; i < *n; i++ {
		line := lines[i%len(lines)]
		sans := strings.Fields(line)

		// Rotate white/black across the 3 players so all pairings appear.
		w := players[i%len(players)]
		b := players[(i/len(players)+1+i)%len(players)]
		if w == b {
			b = players[(i+1)%len(players)]
		}
		result := results[i%len(results)]

		// Vary the date across a couple of years.
		year := 2024 + (i % 2)
		month := (i % 12) + 1
		day := (i % 27) + 1
		date := fmt.Sprintf("%04d.%02d.%02d", year, month, day)

		header := domain.Header{
			White:  w,
			Black:  b,
			Event:  events[i%len(events)],
			Site:   "Trier",
			Date:   date,
			Round:  fmt.Sprintf("%d", (i%9)+1),
			Board:  fmt.Sprintf("%d", (i%8)+1),
			Result: result,
		}

		moves := recognition.Reconcile(startFEN, toTokens(sans))

		id, err := st.CreateDraftGame(ctx, store.NewDraft{
			UserID:     &uid,
			Source:     "manual",
			Header:     header,
			StartFEN:   startFEN,
			Confidence: 1.0,
			Moves:      moves,
		})
		if err != nil {
			fatal("create draft", err)
		}

		pgn := buildPGN(header, sans, result)
		// Mark saved (the library only lists status='saved'); stagger saved_at so
		// the newest-first ordering looks natural.
		if _, err := st.Pool.Exec(ctx,
			`UPDATE games SET status='saved', final_pgn=$2,
			        saved_at = now() - make_interval(mins => $3)
			  WHERE id=$1`,
			id, pgn, i,
		); err != nil {
			fatal("mark saved", err)
		}
		created++
	}
	fmt.Printf("seeded %d saved games for %s across %d players\n", created, *email, len(players))
}

func toTokens(sans []string) []recognition.MoveToken {
	out := make([]recognition.MoveToken, 0, len(sans))
	for i, s := range sans {
		side := recognition.SideWhite
		if i%2 == 1 {
			side = recognition.SideBlack
		}
		out = append(out, recognition.MoveToken{Ply: i + 1, Side: side, Text: s, Confidence: 1.0})
	}
	return out
}

func buildPGN(h domain.Header, sans []string, result string) string {
	var b strings.Builder
	tag := func(k, v string) { fmt.Fprintf(&b, "[%s \"%s\"]\n", k, v) }
	tag("Event", h.Event)
	tag("Site", h.Site)
	tag("Date", h.Date)
	tag("Round", h.Round)
	tag("White", h.White)
	tag("Black", h.Black)
	tag("Result", result)
	b.WriteString("\n")
	for i, s := range sans {
		if i%2 == 0 {
			fmt.Fprintf(&b, "%d. ", i/2+1)
		}
		b.WriteString(s)
		b.WriteString(" ")
	}
	b.WriteString(result)
	b.WriteString("\n")
	return b.String()
}

func fatal(stage string, err error) {
	fmt.Fprintf(os.Stderr, "fatal: %s: %v\n", stage, err)
	os.Exit(1)
}
