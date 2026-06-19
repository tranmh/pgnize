# chesskit public API (authoritative)

`chesskit` is a standalone Go module (`github.com/tranmh/chesskit`) — the reusable chess core. It wraps
`github.com/notnil/chess` but **exposes only the value types below; no `notnil/chess` type may appear in
any exported signature.** It MUST NOT import any pgnize `internal/` package and MUST pass
`cd chesskit && GOWORK=off go test ./...` standalone.

```go
package chesskit

type FEN string
type SAN string
type Result string // "1-0" | "0-1" | "1/2-1/2" | "*"

const (
    ResultWhiteWin Result = "1-0"
    ResultBlackWin Result = "0-1"
    ResultDraw     Result = "1/2-1/2"
    ResultOngoing  Result = "*"
)

type Move struct {
    SAN      SAN  `json:"san"`
    FromFEN  FEN  `json:"fromFen"`
    ToFEN    FEN  `json:"toFen"`
    ClockSec *int `json:"clockSec,omitempty"`
}

type Header struct {
    Event, Site, Date, Round, Board string
    White, Black                    string
    Result                          Result
    Extra                           map[string]string // preserved round-trip
}

type Game struct {
    Header   Header `json:"header"`
    Moves    []Move `json:"moves"`
    StartFEN FEN    `json:"startFen"`
}

var (
    ErrIllegalMove   = errors.New("chesskit: illegal move")
    ErrAmbiguousMove = errors.New("chesskit: ambiguous move")
)

// Starting position FEN.
func StartingFEN() FEN

// ParseSAN parses one SAN move from a position. Returns ErrIllegalMove / ErrAmbiguousMove.
func ParseSAN(from FEN, san SAN) (Move, error)

// Validate reports the resulting FEN after playing san from `from`, or an error.
func Validate(from FEN, san SAN) (to FEN, err error)

// LegalMovesSAN lists every legal move from a position in canonical SAN (for correction dropdowns).
func LegalMovesSAN(from FEN) ([]SAN, error)

// ApplyMoves replays sans from start. positions[i] is the FEN AFTER sans[i].
// On the first illegal/ambiguous move it stops: err != nil and failedAt is that move's index
// (failedAt == -1 when err == nil). This is the load-bearing primitive for the review loop.
func ApplyMoves(start FEN, sans []SAN) (positions []FEN, err error, failedAt int)

// ParsePGN tolerantly parses one or more games from PGN text. Extracts %clk comments into Move.ClockSec.
// A game whose movetext contains an illegal move is truncated at that move (not dropped).
func ParsePGN(text string) ([]Game, error)

// WritePGN renders a single game: Seven-Tag Roster first (stable order), then Extra tags, then movetext
// with the result token; %clk comments emitted when ClockSec set.
func WritePGN(g Game) (string, error)

// WriteBundlePGN renders many games separated by a blank line (library export).
func WriteBundlePGN(games []Game) (string, error)

// NormalizeResult maps loose result strings ("1:0", "½-½", "", "remis", ...) to a Result.
func NormalizeResult(s string) Result
```

Notes:
- `ParseSAN`/`Validate` accept already-English SAN (e.g. `Nf3`, `O-O`, `exd6`, `e8=Q`). German→English
  translation is NOT chesskit's job — it happens in `internal/recognition/postprocess.go` before calling.
- `ApplyMoves` is what the API uses on save: feed `[]SAN`, reject with `failedAt` on the first illegal ply.
- Keep an optional `chesskit/httpsvc/` package later (thin HTTP adapter over these funcs) — not required now.
