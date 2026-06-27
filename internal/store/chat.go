package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// ChatSession is one conversational-coach thread. UserID is nil for anonymous (not persisted
// in practice). It is returned by GetChatSession so the handler can enforce ownership.
type ChatSession struct {
	ID     string
	UserID *string
	GameID *string
	Ply    *int
	FEN    string
	Lang   string
	Model  string
}

// ChatMessage is one stored turn. ToolTrace is the raw jsonb engine-call trace (nil for
// plain user turns).
type ChatMessage struct {
	Seq       int
	Role      string // "user" | "model"
	Content   string
	ToolTrace []byte
}

// CreateChatSession inserts a new conversation and returns its id.
func (s *Store) CreateChatSession(ctx context.Context, userID, gameID *string, ply *int, fen, lang, model string) (string, error) {
	var id string
	err := s.Pool.QueryRow(ctx,
		`INSERT INTO chat_sessions (user_id, game_id, ply, fen, lang, model)
		 VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
		userID, gameID, ply, fen, lang, model).Scan(&id)
	return id, err
}

// GetChatSession returns a session by id (for ownership checks), or ErrNotFound.
func (s *Store) GetChatSession(ctx context.Context, id string) (ChatSession, error) {
	var cs ChatSession
	err := s.Pool.QueryRow(ctx,
		`SELECT id, user_id, game_id, ply, fen, lang, model FROM chat_sessions WHERE id=$1`,
		id).Scan(&cs.ID, &cs.UserID, &cs.GameID, &cs.Ply, &cs.FEN, &cs.Lang, &cs.Model)
	if errors.Is(err, pgx.ErrNoRows) {
		return ChatSession{}, ErrNotFound
	}
	return cs, err
}

// LatestChatSession returns the most recently updated session for (userID, gameID), or
// ErrNotFound. Used to re-hydrate a game's conversation on page load.
func (s *Store) LatestChatSession(ctx context.Context, userID, gameID string) (ChatSession, error) {
	var cs ChatSession
	err := s.Pool.QueryRow(ctx,
		`SELECT id, user_id, game_id, ply, fen, lang, model FROM chat_sessions
		 WHERE user_id=$1 AND game_id=$2 ORDER BY updated_at DESC LIMIT 1`,
		userID, gameID).Scan(&cs.ID, &cs.UserID, &cs.GameID, &cs.Ply, &cs.FEN, &cs.Lang, &cs.Model)
	if errors.Is(err, pgx.ErrNoRows) {
		return ChatSession{}, ErrNotFound
	}
	return cs, err
}

// AppendChatMessage adds a turn, auto-assigning the next seq within the session, and touches
// the session's updated_at. toolTrace may be nil (stored as SQL NULL).
func (s *Store) AppendChatMessage(ctx context.Context, sessionID, role, content string, toolTrace []byte) (int, error) {
	var seq int
	err := s.Pool.QueryRow(ctx,
		`INSERT INTO chat_messages (session_id, seq, role, content, tool_trace)
		 VALUES ($1, (SELECT COALESCE(MAX(seq)+1, 0) FROM chat_messages WHERE session_id=$1), $2, $3, $4)
		 RETURNING seq`,
		sessionID, role, content, toolTrace).Scan(&seq)
	if err != nil {
		return 0, err
	}
	_, err = s.Pool.Exec(ctx, `UPDATE chat_sessions SET updated_at = now() WHERE id=$1`, sessionID)
	return seq, err
}

// ChatHistory returns a session's turns ordered by seq.
func (s *Store) ChatHistory(ctx context.Context, sessionID string) ([]ChatMessage, error) {
	rows, err := s.Pool.Query(ctx,
		`SELECT seq, role, content, tool_trace FROM chat_messages WHERE session_id=$1 ORDER BY seq`,
		sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ChatMessage
	for rows.Next() {
		var m ChatMessage
		if err := rows.Scan(&m.Seq, &m.Role, &m.Content, &m.ToolTrace); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
