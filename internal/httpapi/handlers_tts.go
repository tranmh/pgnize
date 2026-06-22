package httpapi

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/tranmh/pgnize/internal/tts"
)

// ttsMaxTextLen caps synthesis input. The coach's prose is short; a hard cap keeps a single
// request from billing a large TTS call.
const ttsMaxTextLen = 4000

type speakRequest struct {
	Text  string `json:"text"`
	Lang  string `json:"lang"`
	Voice string `json:"voice"`
}

type speakResponse struct {
	AudioURL string `json:"audioUrl"`
	Cached   bool   `json:"cached"`
	Provider string `json:"provider"`
	Voice    string `json:"voice"`
}

// handleSpeak synthesizes (or serves cached) speech for a piece of coaching text. Audio is
// content-addressed by hash(provider|voice|lang|text) and shared across all callers — there
// is no ownership, the same posture as anonymous uploads.
func (s *Server) handleSpeak(w http.ResponseWriter, r *http.Request) {
	if !s.rateLimit(w, r, "tts:"+clientIP(r), 60, time.Hour) {
		return
	}
	var req speakRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeErr(w, http.StatusBadRequest, "bad_request", "invalid json")
		return
	}
	if req.Text == "" {
		s.writeErr(w, http.StatusBadRequest, "bad_request", "text is required")
		return
	}
	if len(req.Text) > ttsMaxTextLen {
		s.writeErr(w, http.StatusBadRequest, "text_too_long", "text exceeds the maximum length")
		return
	}

	lang := coachLang(req.Lang)
	voice := req.Voice
	if voice == "" {
		voice = s.TTS.Voice(lang)
	}
	provider := s.TTS.Name()
	hash := sha256hex(provider + "|" + voice + "|" + lang + "|" + req.Text)

	// Cache hit: serve the stored audio with no TTS call.
	if _, _, err := s.Store.GetTTSAudio(r.Context(), hash); err == nil {
		s.writeJSON(w, http.StatusOK, speakResponse{
			AudioURL: "/api/coach/audio/" + hash,
			Cached:   true,
			Provider: provider,
			Voice:    voice,
		})
		return
	}

	audio, err := s.TTS.Synthesize(r.Context(), tts.SpeakInput{Text: req.Text, Lang: lang, Voice: voice})
	if err != nil {
		// 503 signals the client to fall back to browser speech synthesis.
		s.writeErr(w, http.StatusServiceUnavailable, "tts_unavailable", "speech synthesis is unavailable")
		return
	}

	key := "tts/" + hash + ".wav"
	if err := s.Storage.Put(r.Context(), key, bytes.NewReader(audio.Bytes), int64(len(audio.Bytes)), audio.ContentType); err != nil {
		s.writeErr(w, http.StatusInternalServerError, "internal", "store failed")
		return
	}
	// The audio GET reads this row, so a failed upsert must not be ignored: without the row
	// the just-stored blob is unreachable.
	if err := s.Store.UpsertTTSAudio(r.Context(), hash, provider, voice, lang, key, audio.ContentType, len(audio.Bytes)); err != nil {
		s.writeErr(w, http.StatusInternalServerError, "internal", "record failed")
		return
	}

	s.writeJSON(w, http.StatusOK, speakResponse{
		AudioURL: "/api/coach/audio/" + hash,
		Cached:   false,
		Provider: provider,
		Voice:    voice,
	})
}

// handleSpeakAudio streams cached synthesized audio by content hash. No ownership check:
// the hash is content-addressed and non-enumerable (same posture as handleImage's anon path).
func (s *Server) handleSpeakAudio(w http.ResponseWriter, r *http.Request) {
	key, ct, err := s.Store.GetTTSAudio(r.Context(), chi.URLParam(r, "hash"))
	if isNotFound(err) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		s.writeErr(w, http.StatusInternalServerError, "internal", "load failed")
		return
	}
	body, storedCT, err := s.Storage.Get(r.Context(), key)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer body.Close()
	if storedCT != "" {
		ct = storedCT
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "private, max-age=86400")
	_, _ = io.Copy(w, body)
}

func sha256hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
