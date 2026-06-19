package httpapi

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/tranmh/pgnize/internal/domain"
)

// storeImage reads the "image" multipart field, stores the blob, and records the upload row.
func (s *Server) storeImage(w http.ResponseWriter, r *http.Request, owner *string) (string, bool) {
	if err := r.ParseMultipartForm(s.Cfg.UploadMaxBytes); err != nil {
		s.writeErr(w, http.StatusBadRequest, "bad_request", "could not parse upload")
		return "", false
	}
	file, hdr, err := r.FormFile("image")
	if err != nil {
		s.writeErr(w, http.StatusBadRequest, "missing_image", "expected an 'image' file field")
		return "", false
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, s.Cfg.UploadMaxBytes+1))
	if err != nil {
		s.writeErr(w, http.StatusBadRequest, "bad_request", "read failed")
		return "", false
	}
	if int64(len(data)) > s.Cfg.UploadMaxBytes {
		s.writeErr(w, http.StatusRequestEntityTooLarge, "too_large", "image exceeds size limit")
		return "", false
	}
	sum := sha256.Sum256(data)
	mime := hdr.Header.Get("Content-Type")
	if mime == "" {
		mime = "application/octet-stream"
	}
	ext := strings.ToLower(filepath.Ext(hdr.Filename))
	if ext == "" {
		ext = ".jpg"
	}
	id := uuid.NewString()
	scope := "anon"
	if owner != nil {
		scope = *owner
	}
	now := time.Now().UTC()
	key := fmt.Sprintf("uploads/%s/%04d/%02d/%s%s", scope, now.Year(), int(now.Month()), id, ext)

	if err := s.Storage.Put(r.Context(), key, strings.NewReader(string(data)), int64(len(data)), mime); err != nil {
		s.writeErr(w, http.StatusInternalServerError, "internal", "storage failed")
		return "", false
	}
	consent := r.FormValue("consentTraining") == "true"
	up, err := s.Store.CreateUpload(r.Context(), domain.Upload{
		UserID: owner, StorageKey: key, MimeType: mime, ByteSize: int64(len(data)),
		SHA256: hex.EncodeToString(sum[:]), ConsentTraining: consent,
	})
	if err != nil {
		s.writeErr(w, http.StatusInternalServerError, "internal", "could not record upload")
		return "", false
	}
	return up.ID, true
}

// handleUpload (account): store image, enqueue a recognition job.
func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	u := s.user(r)
	if !s.rateLimit(w, r, "upload:"+u.ID, 60, time.Hour) {
		return
	}
	uploadID, ok := s.storeImage(w, r, &u.ID)
	if !ok {
		return
	}
	jobID, err := s.Store.CreateJob(r.Context(), uploadID, &u.ID, s.Recognizer.Name())
	if err != nil {
		s.writeErr(w, http.StatusInternalServerError, "internal", "could not enqueue job")
		return
	}
	s.writeJSON(w, http.StatusAccepted, map[string]string{"uploadId": uploadID, "jobId": jobID})
}

// handleJobStatus (account): poll a job owned by the caller.
func (s *Server) handleJobStatus(w http.ResponseWriter, r *http.Request) {
	u := s.user(r)
	j, err := s.Store.GetJob(r.Context(), chi.URLParam(r, "jobID"))
	if isNotFound(err) {
		s.writeErr(w, http.StatusNotFound, "not_found", "job not found")
		return
	}
	if err != nil {
		s.writeErr(w, http.StatusInternalServerError, "internal", "load failed")
		return
	}
	if j.UserID == nil || *j.UserID != u.ID {
		s.writeErr(w, http.StatusNotFound, "not_found", "job not found")
		return
	}
	s.writeJobStatus(w, j)
}

func (s *Server) writeJobStatus(w http.ResponseWriter, j domain.Job) {
	resp := map[string]any{"status": j.Status}
	if j.GameID != nil {
		resp["gameId"] = *j.GameID
	}
	if j.Error != "" {
		resp["error"] = j.Error
	}
	s.writeJSON(w, http.StatusOK, resp)
}
