// Package domain holds shared application types used across stores and the API.
package domain

import "time"

// JobStatus values for a recognition job.
const (
	JobQueued   = "queued"
	JobRunning  = "running"
	JobDone     = "done"
	JobFailed   = "failed"
	JobCanceled = "canceled"
)

// Move sides.
const (
	SideWhite = "white"
	SideBlack = "black"
)

// Game source + status.
const (
	SourceManual     = "manual"
	SourceRecognized = "recognized"

	StatusDraft     = "draft"
	StatusReviewing = "reviewing"
	StatusSaved     = "saved"
)

// User is an account holder.
type User struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"-"`
	CreatedAt    time.Time `json:"-"`
}

// Player is an entry in a user's autocomplete pool.
type Player struct {
	ID        string `json:"id"`
	UserID    string `json:"-"`
	FullName  string `json:"fullName"`
	Club      string `json:"club,omitempty"`
	FideID    string `json:"fideId,omitempty"`
	TimesUsed int    `json:"-"`
}

// Header mirrors the PGN seven-tag roster plus board/site.
type Header struct {
	White  string `json:"white"`
	Black  string `json:"black"`
	Event  string `json:"event"`
	Site   string `json:"site"`
	Date   string `json:"date"`
	Round  string `json:"round"`
	Board  string `json:"board"`
	Result string `json:"result"`
}

// Move is one ply in a draft, carrying review metadata.
type Move struct {
	Ply            int    `json:"ply"`
	Side           string `json:"side"` // white | black
	SAN            string `json:"san"`
	FenAfter       string `json:"fenAfter"`
	ClockSec       *int   `json:"clockSec"`
	IsLegal        bool   `json:"isLegal"`
	RecognizedText string `json:"recognizedText"`
	Corrected      bool   `json:"corrected"`
	// Confidence (0..1) is a deterministic recognition-confidence score, independent of
	// legality: a legal move below the review threshold (auto-corrected, guessed
	// disambiguation) is surfaced as a "verify" state. Defaults to 1.0 for human-entered moves.
	Confidence float64 `json:"confidence"`
	// Suggestions are legal moves in this position ranked by similarity to the
	// recognized text — populated when the read move is illegal/ambiguous, so the
	// review UI can offer ranked corrections instead of a raw legal-move list.
	Suggestions []string `json:"suggestions,omitempty"`
}

// GameDraft is the full reviewable game returned by the API.
type GameDraft struct {
	ID         string  `json:"id"`
	Source     string  `json:"source"`
	Status     string  `json:"status"`
	Header     Header  `json:"header"`
	StartFEN   string  `json:"startFen"`
	Moves      []Move  `json:"moves"`
	ImageURL   string  `json:"imageUrl"`
	Confidence float64 `json:"confidence"`
}

// GameSummary is a library list item.
type GameSummary struct {
	ID        string     `json:"id"`
	White     string     `json:"white"`
	Black     string     `json:"black"`
	Event     string     `json:"event"`
	Date      string     `json:"date"`
	Result    string     `json:"result"`
	MoveCount int        `json:"moveCount"`
	SavedAt   *time.Time `json:"savedAt"`
}

// Upload is a stored raw image.
type Upload struct {
	ID              string
	UserID          *string
	StorageKey      string
	MimeType        string
	ByteSize        int64
	SHA256          string
	ConsentTraining bool
	CreatedAt       time.Time
}

// Job is a recognition job row.
type Job struct {
	ID             string
	UploadID       string
	UserID         *string
	Status         string
	RecognizerName string
	Attempts       int
	Error          string
	GameID         *string
	Confidence     *float64
}
