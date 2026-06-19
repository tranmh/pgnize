// Package storage abstracts blob storage for raw score-sheet images.
// Driver is chosen via STORAGE_DRIVER=auto|s3|filesystem.
package storage

import (
	"context"
	"io"
)

// Storage stores and retrieves opaque objects by key.
type Storage interface {
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
	Get(ctx context.Context, key string) (io.ReadCloser, string, error) // body, contentType
	Delete(ctx context.Context, key string) error
}
