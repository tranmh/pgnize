package httpapi

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tranmh/pgnize/internal/chat"
	"github.com/tranmh/pgnize/internal/config"
	"github.com/tranmh/pgnize/internal/engine"
	"github.com/tranmh/pgnize/internal/stt"
)

// chatTestServer is a DB-free server with fake chat/engine/STT. Anonymous turns never touch
// the (nil) Store, so the handler can be exercised without a database.
func chatTestServer() *Server {
	return &Server{
		Cfg:  config.Config{RateLimitDisabled: true, STTMaxBytes: 1 << 20},
		Chat: chat.NewFake(engine.NewFake()),
		STT:  stt.NewFake(),
	}
}

func postChatJSON(s *Server, body chatTurnRequest) *httptest.ResponseRecorder {
	buf, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/coach/chat", bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.handleChatTurn(rec, req)
	return rec
}

func TestChatTurnText(t *testing.T) {
	s := chatTestServer()
	rec := postChatJSON(s, chatTurnRequest{FEN: startFENForTest, Side: "white", Text: "What is the best move?", Lang: "en"})
	if rec.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp chatTurnResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Reply == "" {
		t.Error("expected a reply")
	}
	if resp.UserText != "What is the best move?" {
		t.Errorf("userText = %q", resp.UserText)
	}
	if resp.ConversationID != "" {
		t.Error("anonymous turn must not get a persisted conversationId")
	}
	if len(resp.EngineFacts) == 0 {
		t.Error("expected engine facts from the tool loop")
	}
}

func TestChatTurnIllegalFEN(t *testing.T) {
	s := chatTestServer()
	rec := postChatJSON(s, chatTurnRequest{FEN: "not-a-fen", Text: "hi"})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code=%d, want 400", rec.Code)
	}
}

func TestChatTurnNoInput(t *testing.T) {
	s := chatTestServer()
	rec := postChatJSON(s, chatTurnRequest{FEN: startFENForTest})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code=%d, want 400 (no question)", rec.Code)
	}
}

func TestChatTurnTranscriptMode(t *testing.T) {
	s := chatTestServer()
	rec := postChatJSON(s, chatTurnRequest{FEN: startFENForTest, Transcript: "Gibt es ein Matt?", Lang: "de"})
	if rec.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp chatTurnResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.UserText != "Gibt es ein Matt?" {
		t.Errorf("userText = %q, want the transcript", resp.UserText)
	}
}

func TestChatTurnAudioInlineUsesSTT(t *testing.T) {
	s := chatTestServer()
	rec := postChatJSON(s, chatTurnRequest{
		FEN:   startFENForTest,
		Audio: &chatAudioB64{Data: base64.StdEncoding.EncodeToString([]byte("voice")), MimeType: "audio/webm"},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp chatTurnResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	// Fake STT returns a fixed German question.
	if resp.UserText == "" {
		t.Error("expected a transcript as userText")
	}
}

func TestChatTurnAudioMultipart(t *testing.T) {
	s := chatTestServer()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	_ = mw.WriteField("fen", startFENForTest)
	_ = mw.WriteField("side", "white")
	fw, _ := mw.CreateFormFile("audio", "turn.webm")
	_, _ = fw.Write([]byte("fake-audio-bytes"))
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/coach/chat", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	s.handleChatTurn(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp chatTurnResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Reply == "" || resp.UserText == "" {
		t.Errorf("expected reply+userText, got %+v", resp)
	}
}

func TestChatTurnAudioWithoutSTTBackend(t *testing.T) {
	s := chatTestServer()
	s.STT = nil // no server STT configured
	rec := postChatJSON(s, chatTurnRequest{
		FEN:   startFENForTest,
		Audio: &chatAudioB64{Data: base64.StdEncoding.EncodeToString([]byte("voice")), MimeType: "audio/webm"},
	})
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("code=%d, want 503 so the client falls back to browser STT", rec.Code)
	}
}

func TestChatTurnConversationIDRequiresAuth(t *testing.T) {
	s := chatTestServer()
	rec := postChatJSON(s, chatTurnRequest{FEN: startFENForTest, Text: "hi", ConversationID: "some-id"})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d, want 401 (anon cannot continue a stored conversation)", rec.Code)
	}
}

func TestChatHistoryAnonymousEmpty(t *testing.T) {
	s := chatTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/coach/chat/history?gameId=abc", nil)
	rec := httptest.NewRecorder()
	s.handleChatHistory(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code=%d", rec.Code)
	}
	var resp chatHistoryResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if len(resp.Messages) != 0 {
		t.Errorf("anonymous history should be empty, got %d", len(resp.Messages))
	}
}

func TestChatTurnUnavailableWhenNoChatter(t *testing.T) {
	s := &Server{Cfg: config.Config{RateLimitDisabled: true}}
	rec := postChatJSON(s, chatTurnRequest{FEN: startFENForTest, Text: "hi"})
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("code=%d, want 503", rec.Code)
	}
}
