package httpapi

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/tranmh/pgnize/internal/domain"
)

// maxImagesPerJob caps how many images one anonymous submission may carry. The whole
// submission is fed to the recognizer as a single multi-image request (one job, one
// rate-limit unit), so the cap bounds per-request cost.
const maxImagesPerJob = 5

// storeImages reads ALL "image" multipart fields (1..maxImagesPerJob), stores each blob,
// and returns the upload IDs in submission order. The first is the primary image; the rest
// are the extras of the same job. On any failure it writes the appropriate error and returns
// false.
func (s *Server) storeImages(w http.ResponseWriter, r *http.Request, owner *string) ([]string, bool) {
	if err := r.ParseMultipartForm(s.Cfg.UploadMaxBytes); err != nil {
		s.writeErr(w, http.StatusBadRequest, "bad_request", "could not parse upload")
		return nil, false
	}
	files := r.MultipartForm.File["image"]
	if len(files) == 0 {
		s.writeErr(w, http.StatusBadRequest, "missing_image", "expected at least one 'image' file field")
		return nil, false
	}
	if len(files) > maxImagesPerJob {
		s.writeErr(w, http.StatusBadRequest, "bad_request",
			fmt.Sprintf("too many images: %d (max %d)", len(files), maxImagesPerJob))
		return nil, false
	}
	ids := make([]string, 0, len(files))
	for _, hdr := range files {
		id, err := s.storeFileHeader(r, owner, hdr)
		if err != nil {
			s.writeStoreErr(w, err)
			return nil, false
		}
		ids = append(ids, id)
	}
	return ids, true
}

// storeErr is a sentinel carrying the HTTP status and error code for a failed file store, so
// the per-file helper can report failures uniformly to either single- or multi-image callers.
type storeErr struct {
	status int
	code   string
	msg    string
}

func (e storeErr) Error() string { return e.msg }

func (s *Server) writeStoreErr(w http.ResponseWriter, err error) {
	if se, ok := err.(storeErr); ok {
		s.writeErr(w, se.status, se.code, se.msg)
		return
	}
	s.writeErr(w, http.StatusInternalServerError, "internal", "could not store upload")
}

// storeFileHeader reads, size-limits, hashes, stores, and records one uploaded file. It is
// the single-file primitive used by storeImages; failures come back as a storeErr carrying
// the status/code the handler must surface.
func (s *Server) storeFileHeader(r *http.Request, owner *string, hdr *multipart.FileHeader) (string, error) {
	file, err := hdr.Open()
	if err != nil {
		return "", storeErr{http.StatusBadRequest, "bad_request", "could not open uploaded file"}
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, s.Cfg.UploadMaxBytes+1))
	if err != nil {
		return "", storeErr{http.StatusBadRequest, "bad_request", "read failed"}
	}
	if int64(len(data)) > s.Cfg.UploadMaxBytes {
		return "", storeErr{http.StatusRequestEntityTooLarge, "too_large", "image exceeds size limit"}
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
		return "", storeErr{http.StatusInternalServerError, "internal", "storage failed"}
	}
	consent := r.FormValue("consentTraining") == "true"
	up, err := s.Store.CreateUpload(r.Context(), domain.Upload{
		UserID: owner, StorageKey: key, MimeType: mime, ByteSize: int64(len(data)),
		SHA256: hex.EncodeToString(sum[:]), ConsentTraining: consent,
	})
	if err != nil {
		return "", storeErr{http.StatusInternalServerError, "internal", "could not record upload"}
	}
	return up.ID, nil
}

// handleUpload (account): store one or more images, enqueue a single recognition job.
// Like the anonymous endpoints, multiple "image" fields are fed to the recognizer as one
// multi-image request (one job): ids[0] is the primary, ids[1:] the extras of the same job.
func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	u := s.user(r)
	if !s.rateLimit(w, r, "upload:"+u.ID, 60, time.Hour) {
		return
	}
	ids, ok := s.storeImages(w, r, &u.ID)
	if !ok {
		return
	}
	backend, recName, ok := s.jobBackend(w, r)
	if !ok {
		return
	}
	// kind selects the recognition pipeline; an unrecognized value falls back to scoresheet.
	kind := r.FormValue("kind")
	if kind != "scoresheet" && kind != "position" {
		kind = "scoresheet"
	}
	jobID, err := s.Store.CreateJob(r.Context(), ids[0], &u.ID, recName, backend, kind, ids[1:])
	if err != nil {
		s.writeErr(w, http.StatusInternalServerError, "internal", "could not enqueue job")
		return
	}
	s.writeJSON(w, http.StatusAccepted, map[string]string{"uploadId": ids[0], "jobId": jobID})
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
