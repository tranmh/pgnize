package storage

import (
	"context"
	"fmt"

	"github.com/tranmh/pgnize/internal/config"
)

// New builds the configured storage driver.
func New(ctx context.Context, c config.Config) (Storage, error) {
	switch c.ResolveStorageDriver() {
	case "filesystem":
		return NewFS(c.StorageDir)
	case "s3":
		return NewS3(ctx, S3Options{
			Endpoint:  c.S3Endpoint,
			Region:    c.S3Region,
			AccessKey: c.S3AccessKey,
			SecretKey: c.S3SecretKey,
			Bucket:    c.S3Bucket,
			PathStyle: c.S3PathStyle,
		})
	default:
		return nil, fmt.Errorf("unknown storage driver %q", c.StorageDriver)
	}
}
