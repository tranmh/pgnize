package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// FS stores objects as files under Root. Content types are kept in a sidecar .meta file.
type FS struct{ Root string }

// NewFS creates a filesystem-backed store rooted at dir.
func NewFS(dir string) (*FS, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &FS{Root: dir}, nil
}

func (f *FS) path(key string) (string, error) {
	clean := filepath.Clean("/" + key) // prevents traversal
	return filepath.Join(f.Root, clean), nil
}

func (f *FS) Put(_ context.Context, key string, r io.Reader, _ int64, contentType string) error {
	p, err := f.path(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	dst, err := os.Create(p)
	if err != nil {
		return err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, r); err != nil {
		return err
	}
	meta, _ := json.Marshal(map[string]string{"contentType": contentType})
	return os.WriteFile(p+".meta", meta, 0o644)
}

func (f *FS) Get(_ context.Context, key string) (io.ReadCloser, string, error) {
	p, err := f.path(key)
	if err != nil {
		return nil, "", err
	}
	file, err := os.Open(p)
	if err != nil {
		return nil, "", err
	}
	ct := "application/octet-stream"
	if b, err := os.ReadFile(p + ".meta"); err == nil {
		var m map[string]string
		if json.Unmarshal(b, &m) == nil && m["contentType"] != "" {
			ct = m["contentType"]
		}
	}
	return file, ct, nil
}

func (f *FS) Delete(_ context.Context, key string) error {
	p, err := f.path(key)
	if err != nil {
		return err
	}
	_ = os.Remove(p + ".meta")
	if err := os.Remove(p); err != nil && !strings.Contains(err.Error(), "no such file") {
		return fmt.Errorf("delete %s: %w", key, err)
	}
	return nil
}
