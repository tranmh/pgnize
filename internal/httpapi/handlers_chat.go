package httpapi

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/tranmh/chesskit"
	"github.com/tranmh/pgnize/internal/chat"
	"github.com/tranmh/pgnize/internal/store"
	"github.com/tranmh/pgnize/internal/stt"
)

// chatMaxQuestionLen caps a single typed/transcribed question.
const chatMaxQuestionLen = 2000

// chatTurnRequest is the JSON body for a text or browser-STT (transcript) turn. Audio turns
// arrive as multipart/form-data instead (see parseMultipartTurn).
type chatTurnRequest struct {
	ConversationID string        `json:"conversationId"`
	FEN            string        `json:"fen"`
	Side           string        `json:"side"`
	GameID         string        `json:"gameId"`
	Ply            *int          `json:"ply"`
	Lang           string        `json:"lang"`
	Text           string        `json:"text"`       // typed question
	Transcript     string        `json:"transcript"` // browser Web Speech result
	Audio          *chatAudioB64 `json:"audio"`      // optional inline base64 audio (server STT)
	History        []chatHistMsg `json:"history"`    // optional anon-continuity history
}

type chatAudioB64 struct {
	Data     string `json:"data"`
	MimeType string `json:"mimeType"`
}

type chatHistMsg struct {
	Role string `json:"role"` // "user" | "coach"
	Text string `json:"text"`
}

type chatTurnResponse struct {
	ConversationID string          `json:"conversationId"`
	UserText       string          `json:"userText"`
	Reply          string          `json:"reply"`
	Model          string          `json:"model"`
	EngineFacts    []chat.ToolCall `json:"engineFacts"`
}

// turnInput is the normalized result of decoding either a JSON or multipart request.
type turnInput struct {
	req       chatTurnRequest
	audio     []byte
	audioMime string
}

// handleChatTurn answers one user question about a position. Input may be typed text, a
// browser-STT transcript, or uploaded audio (server STT). The LLM decides when to consult the
// server-side engine; every FEN/move it references is validated via chesskit before the
// engine runs. History persists only for logged-in callers; anonymous turns are stateless.
func (s *Server) handleChatTurn(w http.ResponseWriter, r *http.Request) {
	if !s.rateLimit(w, r, "chat:"+clientIP(r), 60, time.Hour) {
		return
	}
	if s.Chat == nil {
		s.writeErr(w, http.StatusServiceUnavailable, "chat_unavailable", "the coach is unavailable")
		return
	}

	in, err := s.decodeTurn(r)
	if err != nil {
		s.writeErr(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}

	fen, err := chesskit.NormalizeFEN(chesskit.FEN(in.req.FEN))
	if err != nil {
		s.writeErr(w, http.StatusBadRequest, "bad_request", "illegal fen")
		return
	}

	// Resolve the user's utterance: audio -> STT, else transcript, else typed text.
	userText, err := s.resolveUtterance(r, in)
	if err != nil {
		switch err {
		case errNoInput:
			s.writeErr(w, http.StatusBadRequest, "bad_request", "a question (text, transcript, or audio) is required")
		case errSTTUnavailable:
			// 503 tells the client to fall back to browser speech recognition.
			s.writeErr(w, http.StatusServiceUnavailable, "stt_unavailable", "speech recognition is unavailable")
		default:
			s.writeErr(w, http.StatusBadGateway, "stt_failed", "could not transcribe the audio")
		}
		return
	}
	if len(userText) > chatMaxQuestionLen {
		userText = userText[:chatMaxQuestionLen]
	}

	lang := coachLang(in.req.Lang)
	side := in.req.Side
	if side == "" {
		side = sideToMove(string(fen))
	}

	// Load prior turns. A conversationId requires auth + ownership; anonymous callers may pass
	// inline history for continuity.
	user := s.user(r)
	var history []chat.Message
	var sessionID string
	if in.req.ConversationID != "" {
		if user == nil {
			s.writeErr(w, http.StatusUnauthorized, "unauthorized", "login required to continue a conversation")
			return
		}
		sess, err := s.Store.GetChatSession(r.Context(), in.req.ConversationID)
		if isNotFound(err) {
			s.writeErr(w, http.StatusNotFound, "not_found", "conversation not found")
			return
		}
		if err != nil {
			s.writeErr(w, http.StatusInternalServerError, "internal", "load failed")
			return
		}
		if sess.UserID == nil || *sess.UserID != user.ID {
			s.writeErr(w, http.StatusForbidden, "forbidden", "not your conversation")
			return
		}
		sessionID = sess.ID
		stored, err := s.Store.ChatHistory(r.Context(), sess.ID)
		if err != nil {
			s.writeErr(w, http.StatusInternalServerError, "internal", "load failed")
			return
		}
		history = toChatMessages(stored)
	} else {
		history = inlineHistory(in.req.History)
	}

	reply, err := s.Chat.Respond(r.Context(), history, userText, chat.Context{
		FEN:    string(fen),
		Side:   side,
		GameID: in.req.GameID,
		Ply:    in.req.Ply,
		Lang:   lang,
	})
	if err != nil {
		s.writeErr(w, http.StatusBadGateway, "chat_failed", "the coach is unavailable")
		return
	}

	// Persist only for logged-in callers.
	convID := ""
	if user != nil {
		convID = s.persistTurn(r, user.ID, sessionID, in.req, string(fen), lang, reply, userText)
	}

	s.writeJSON(w, http.StatusOK, chatTurnResponse{
		ConversationID: convID,
		UserText:       userText,
		Reply:          reply.Text,
		Model:          reply.Model,
		EngineFacts:    reply.Calls,
	})
}

// decodeTurn reads either a multipart (audio) or JSON (text/transcript) request.
func (s *Server) decodeTurn(r *http.Request) (turnInput, error) {
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		return s.parseMultipartTurn(r)
	}
	var req chatTurnRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return turnInput{}, errInvalidJSON
	}
	in := turnInput{req: req}
	if req.Audio != nil && req.Audio.Data != "" {
		raw, err := base64.StdEncoding.DecodeString(req.Audio.Data)
		if err != nil {
			return turnInput{}, errInvalidJSON
		}
		in.audio = raw
		in.audioMime = req.Audio.MimeType
	}
	return in, nil
}

func (s *Server) parseMultipartTurn(r *http.Request) (turnInput, error) {
	max := s.Cfg.STTMaxBytes
	if max <= 0 {
		max = stt.MaxBytesDefault
	}
	r.Body = http.MaxBytesReader(nil, r.Body, max+(1<<16)) // audio cap + form overhead
	if err := r.ParseMultipartForm(max + (1 << 16)); err != nil {
		return turnInput{}, errInvalidJSON
	}
	req := chatTurnRequest{
		ConversationID: r.FormValue("conversationId"),
		FEN:            r.FormValue("fen"),
		Side:           r.FormValue("side"),
		GameID:         r.FormValue("gameId"),
		Lang:           r.FormValue("lang"),
		Text:           r.FormValue("text"),
		Transcript:     r.FormValue("transcript"),
	}
	if v := r.FormValue("ply"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			req.Ply = &n
		}
	}
	in := turnInput{req: req}
	if file, hdr, err := r.FormFile("audio"); err == nil {
		defer file.Close()
		raw, err := io.ReadAll(io.LimitReader(file, max))
		if err != nil {
			return turnInput{}, errInvalidJSON
		}
		in.audio = raw
		in.audioMime = hdr.Header.Get("Content-Type")
	}
	return in, nil
}

var (
	errInvalidJSON    = chatErr("invalid request")
	errNoInput        = chatErr("no input")
	errSTTUnavailable = chatErr("stt unavailable")
)

type chatErr string

func (e chatErr) Error() string { return string(e) }

// resolveUtterance turns the request into the user's question text.
func (s *Server) resolveUtterance(r *http.Request, in turnInput) (string, error) {
	if len(in.audio) > 0 {
		if s.STT == nil {
			return "", errSTTUnavailable
		}
		out, err := s.STT.Transcribe(r.Context(), stt.TranscribeInput{
			Audio: in.audio, MimeType: in.audioMime, Lang: in.req.Lang,
		})
		if err != nil {
			return "", err
		}
		return out.Text, nil
	}
	if in.req.Transcript != "" {
		return in.req.Transcript, nil
	}
	if in.req.Text != "" {
		return in.req.Text, nil
	}
	return "", errNoInput
}

// persistTurn stores the user + coach turns, creating the session on the first turn. It
// returns the conversation id (empty on a storage failure, so the answer is still returned).
func (s *Server) persistTurn(r *http.Request, userID, sessionID string, req chatTurnRequest, fen, lang string, reply chat.Reply, userText string) string {
	if sessionID == "" {
		var gameID *string
		if req.GameID != "" && s.ownsGameID(r, req.GameID) {
			gameID = &req.GameID
		}
		id, err := s.Store.CreateChatSession(r.Context(), &userID, gameID, req.Ply, fen, lang, reply.Model)
		if err != nil {
			return ""
		}
		sessionID = id
	}
	if _, err := s.Store.AppendChatMessage(r.Context(), sessionID, "user", userText, nil); err != nil {
		return sessionID
	}
	var trace []byte
	if len(reply.Calls) > 0 {
		trace, _ = json.Marshal(reply.Calls)
	}
	_, _ = s.Store.AppendChatMessage(r.Context(), sessionID, "model", reply.Text, trace)
	return sessionID
}

// handleChatHistory re-hydrates the latest conversation for a game (logged-in owner only).
func (s *Server) handleChatHistory(w http.ResponseWriter, r *http.Request) {
	user := s.user(r)
	gameID := r.URL.Query().Get("gameId")
	if user == nil || gameID == "" {
		s.writeJSON(w, http.StatusOK, chatHistoryResponse{Messages: []chatHistoryItem{}})
		return
	}
	sess, err := s.Store.LatestChatSession(r.Context(), user.ID, gameID)
	if isNotFound(err) {
		s.writeJSON(w, http.StatusOK, chatHistoryResponse{Messages: []chatHistoryItem{}})
		return
	}
	if err != nil {
		s.writeErr(w, http.StatusInternalServerError, "internal", "load failed")
		return
	}
	stored, err := s.Store.ChatHistory(r.Context(), sess.ID)
	if err != nil {
		s.writeErr(w, http.StatusInternalServerError, "internal", "load failed")
		return
	}
	items := make([]chatHistoryItem, 0, len(stored))
	for _, m := range stored {
		role := "user"
		if m.Role == "model" {
			role = "coach"
		}
		items = append(items, chatHistoryItem{Role: role, Text: m.Content})
	}
	s.writeJSON(w, http.StatusOK, chatHistoryResponse{ConversationID: sess.ID, Messages: items})
}

type chatHistoryItem struct {
	Role string `json:"role"` // "user" | "coach"
	Text string `json:"text"`
}
type chatHistoryResponse struct {
	ConversationID string            `json:"conversationId"`
	Messages       []chatHistoryItem `json:"messages"`
}

// toChatMessages converts stored rows into the chat package's history shape.
func toChatMessages(rows []store.ChatMessage) []chat.Message {
	out := make([]chat.Message, 0, len(rows))
	for _, m := range rows {
		role := chat.RoleUser
		if m.Role == "model" {
			role = chat.RoleCoach
		}
		out = append(out, chat.Message{Role: role, Text: m.Content})
	}
	return out
}

// inlineHistory converts client-supplied (anonymous) history into chat messages.
func inlineHistory(rows []chatHistMsg) []chat.Message {
	out := make([]chat.Message, 0, len(rows))
	for _, m := range rows {
		role := chat.RoleUser
		if m.Role == "coach" || m.Role == "model" {
			role = chat.RoleCoach
		}
		out = append(out, chat.Message{Role: role, Text: m.Text})
	}
	return out
}
