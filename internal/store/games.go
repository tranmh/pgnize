package store

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/tranmh/pgnize/internal/domain"
)

// GameMeta carries ownership/source info alongside a draft.
type GameMeta struct {
	UploadID *string
	OwnerID  *string // user_id; nil for anonymous
}

// NewDraft is the input for creating a draft game.
type NewDraft struct {
	UserID     *string
	UploadID   *string
	Source     string
	Header     domain.Header
	StartFEN   string
	Confidence float64
	Moves      []domain.Move
}

// CreateDraftGame inserts a draft game and its moves in one transaction.
func (s *Store) CreateDraftGame(ctx context.Context, d NewDraft) (string, error) {
	if d.StartFEN == "" {
		d.StartFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	}
	if d.Header.Result == "" {
		d.Header.Result = "*"
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var id string
	if err := tx.QueryRow(ctx,
		`INSERT INTO games (user_id, upload_id, source, status, event, site, event_date, round, board,
		                    white_player, black_player, result, start_fen, confidence)
		 VALUES ($1,$2,$3,'draft',$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) RETURNING id`,
		d.UserID, d.UploadID, d.Source, d.Header.Event, d.Header.Site, d.Header.Date, d.Header.Round,
		d.Header.Board, d.Header.White, d.Header.Black, d.Header.Result, d.StartFEN, d.Confidence,
	).Scan(&id); err != nil {
		return "", err
	}
	if err := insertMoves(ctx, tx, id, d.Moves); err != nil {
		return "", err
	}
	return id, tx.Commit(ctx)
}

func insertMoves(ctx context.Context, tx pgx.Tx, gameID string, moves []domain.Move) error {
	for _, m := range moves {
		if _, err := tx.Exec(ctx,
			`INSERT INTO moves (game_id, ply, side, san, fen_after, clock_sec, is_legal, recognized_text, corrected, confidence)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
			gameID, m.Ply, m.Side, m.SAN, m.FenAfter, m.ClockSec, m.IsLegal, m.RecognizedText, m.Corrected, m.Confidence,
		); err != nil {
			return err
		}
	}
	return nil
}

// GetGame returns a draft plus ownership metadata.
func (s *Store) GetGame(ctx context.Context, id string) (domain.GameDraft, GameMeta, error) {
	var g domain.GameDraft
	var meta GameMeta
	err := s.Pool.QueryRow(ctx,
		`SELECT id, user_id, upload_id, source, status, event, site, event_date, round, board,
		        white_player, black_player, result, start_fen, confidence
		   FROM games WHERE id = $1`, id,
	).Scan(&g.ID, &meta.OwnerID, &meta.UploadID, &g.Source, &g.Status, &g.Header.Event, &g.Header.Site,
		&g.Header.Date, &g.Header.Round, &g.Header.Board, &g.Header.White, &g.Header.Black,
		&g.Header.Result, &g.StartFEN, &g.Confidence)
	if errors.Is(err, pgx.ErrNoRows) {
		return g, meta, ErrNotFound
	}
	if err != nil {
		return g, meta, err
	}
	rows, err := s.Pool.Query(ctx,
		`SELECT ply, side, san, fen_after, clock_sec, is_legal, recognized_text, corrected, confidence
		   FROM moves WHERE game_id = $1 ORDER BY ply`, id)
	if err != nil {
		return g, meta, err
	}
	defer rows.Close()
	g.Moves = []domain.Move{} // never nil: the API contract is moves: Move[]
	for rows.Next() {
		var m domain.Move
		if err := rows.Scan(&m.Ply, &m.Side, &m.SAN, &m.FenAfter, &m.ClockSec, &m.IsLegal,
			&m.RecognizedText, &m.Corrected, &m.Confidence); err != nil {
			return g, meta, err
		}
		g.Moves = append(g.Moves, m)
	}
	return g, meta, rows.Err()
}

// SaveGame replaces header + moves, marks the game saved, and stores the canonical PGN.
func (s *Store) SaveGame(ctx context.Context, id string, h domain.Header, startFEN string, moves []domain.Move, finalPGN string, whitePlayerID, blackPlayerID *string) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		`UPDATE games SET event=$2, site=$3, event_date=$4, round=$5, board=$6,
		        white_player=$7, black_player=$8, result=$9, start_fen=$10, final_pgn=$11,
		        white_player_id=$12, black_player_id=$13, status='saved', saved_at=now()
		  WHERE id=$1`,
		id, h.Event, h.Site, h.Date, h.Round, h.Board, h.White, h.Black, h.Result, startFEN, finalPGN,
		whitePlayerID, blackPlayerID,
	); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM moves WHERE game_id = $1`, id); err != nil {
		return err
	}
	if err := insertMoves(ctx, tx, id, moves); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// GameOwner returns the owning user id (nil for anonymous) and whether the game exists.
func (s *Store) GameOwner(ctx context.Context, id string) (*string, error) {
	var owner *string
	err := s.Pool.QueryRow(ctx, `SELECT user_id FROM games WHERE id=$1`, id).Scan(&owner)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return owner, err
}

// DeleteGame removes a game (and its moves via cascade).
func (s *Store) DeleteGame(ctx context.Context, id string) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM games WHERE id=$1`, id)
	return err
}

// GameFilter narrows a library listing.
type GameFilter struct {
	Q, Player, Event, From, To string
	Page, PageSize             int
}

// ListGames returns saved games for a user with optional search/filters.
func (s *Store) ListGames(ctx context.Context, userID string, f GameFilter) ([]domain.GameSummary, int, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 || f.PageSize > 100 {
		f.PageSize = 20
	}
	where := []string{"user_id = $1", "status = 'saved'"}
	args := []any{userID}
	add := func(cond string, val any) {
		args = append(args, val)
		where = append(where, strings.Replace(cond, "?", "$"+itoa(len(args)), 1))
	}
	if f.Q != "" {
		add("(white_player || ' ' || black_player || ' ' || event) ILIKE ?", "%"+f.Q+"%")
	}
	if f.Player != "" {
		add("(white_player ILIKE ? OR black_player ILIKE ?)", "%"+f.Player+"%")
		// second placeholder reuses the same arg
		where[len(where)-1] = strings.Replace(where[len(where)-1], "?", "$"+itoa(len(args)), 1)
	}
	if f.Event != "" {
		add("event ILIKE ?", "%"+f.Event+"%")
	}
	if f.From != "" {
		add("event_date >= ?", f.From)
	}
	if f.To != "" {
		add("event_date <= ?", f.To)
	}
	clause := strings.Join(where, " AND ")

	var total int
	if err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM games WHERE `+clause, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)
	rows, err := s.Pool.Query(ctx,
		`SELECT g.id, g.white_player, g.black_player, g.event, g.event_date, g.result, g.saved_at,
		        (SELECT count(*) FROM moves m WHERE m.game_id = g.id)
		   FROM games g WHERE `+clause+
			` ORDER BY g.saved_at DESC LIMIT $`+itoa(len(args)-1)+` OFFSET $`+itoa(len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []domain.GameSummary
	for rows.Next() {
		var s domain.GameSummary
		var savedAt *time.Time
		if err := rows.Scan(&s.ID, &s.White, &s.Black, &s.Event, &s.Date, &s.Result, &savedAt, &s.MoveCount); err != nil {
			return nil, 0, err
		}
		s.SavedAt = savedAt
		out = append(out, s)
	}
	return out, total, rows.Err()
}

// GamePGN returns the stored canonical PGN for one saved game owned by userID.
func (s *Store) GamePGN(ctx context.Context, userID, id string) (string, error) {
	var pgn *string
	err := s.Pool.QueryRow(ctx,
		`SELECT final_pgn FROM games WHERE id=$1 AND user_id=$2`, id, userID).Scan(&pgn)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if pgn == nil {
		return "", ErrNotFound
	}
	return *pgn, nil
}

// GamesPGN returns the stored PGNs for the given ids owned by userID, in any order.
func (s *Store) GamesPGN(ctx context.Context, userID string, ids []string) ([]string, error) {
	rows, err := s.Pool.Query(ctx,
		`SELECT final_pgn FROM games WHERE user_id=$1 AND id = ANY($2) AND final_pgn IS NOT NULL`,
		userID, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
