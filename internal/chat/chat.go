// Package chat is the conversational coach: a multi-turn LLM loop that answers free-form
// questions about a chess position ("What's the best move?", "Why is Nf3 bad?", "Is there a
// mate?"). The LLM decides when to consult the engine via function-calling; this package
// owns the tool definitions, the validate-then-analyze dispatch, and the turn loop.
//
// Correctness boundary: every FEN/move the model passes to a tool is validated through
// chesskit BEFORE the engine sees it (illegal input becomes an error result the model can
// recover from — the engine is never called with an illegal position). The engine itself
// (internal/engine) knows no chess rules. Coaching is advisory and never affects saved PGN.
package chat

import "context"

// LangDefault mirrors the rest of PGNize: German-first.
const LangDefault = "de"

// Role is a conversation turn author.
type Role string

const (
	RoleUser Role = "user"
	RoleCoach Role = "model"
)

// Message is one stored conversation turn. ToolTrace records the engine calls a coach turn
// made (nil for plain user turns); it is persisted and surfaced to the UI as the facts the
// answer was grounded in.
type Message struct {
	Role      Role
	Text      string
	ToolTrace []ToolCall
}

// ToolCall is one engine consultation: the function the model invoked, the (validated) args,
// the result fed back, and an error string when validation/engine failed.
type ToolCall struct {
	Name   string         `json:"name"`
	Args   map[string]any `json:"args"`
	Result map[string]any `json:"result,omitempty"`
	Err    string         `json:"err,omitempty"`
}

// Context grounds a conversation in a concrete, already-normalized position.
type Context struct {
	FEN    string
	Side   string // "white" | "black"
	GameID string // optional
	Ply    *int   // optional
	Lang   string // "" -> LangDefault
}

// Reply is the coach's answer for one user turn.
type Reply struct {
	Text  string     `json:"text"`
	Model string     `json:"model"`
	Calls []ToolCall `json:"calls"`
}

// Chatter answers one user utterance given prior history and the position context.
type Chatter interface {
	Respond(ctx context.Context, history []Message, user string, pctx Context) (Reply, error)
	Name() string
}

func normLang(l string) string {
	if l == "" {
		return LangDefault
	}
	return l
}
