package storage

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3 is an S3-compatible store (works against MinIO and AWS S3).
type S3 struct {
	client *minio.Client
	bucket string
}

// S3Options configures the S3 store.
type S3Options struct {
	Endpoint  string // http(s)://host:port
	Region    string
	AccessKey string
	SecretKey string
	Bucket    string
	PathStyle bool
}

// NewS3 builds an S3 store and ensures the bucket exists.
func NewS3(ctx context.Context, o S3Options) (*S3, error) {
	secure := strings.HasPrefix(o.Endpoint, "https://")
	host := strings.TrimPrefix(strings.TrimPrefix(o.Endpoint, "https://"), "http://")
	cl, err := minio.New(host, &minio.Options{
		Creds:        credentials.NewStaticV4(o.AccessKey, o.SecretKey, ""),
		Secure:       secure,
		Region:       o.Region,
		BucketLookup: bucketLookup(o.PathStyle),
	})
	if err != nil {
		return nil, fmt.Errorf("s3 client: %w", err)
	}
	exists, err := cl.BucketExists(ctx, o.Bucket)
	if err != nil {
		return nil, fmt.Errorf("s3 bucket check: %w", err)
	}
	if !exists {
		if err := cl.MakeBucket(ctx, o.Bucket, minio.MakeBucketOptions{Region: o.Region}); err != nil {
			return nil, fmt.Errorf("s3 make bucket: %w", err)
		}
	}
	return &S3{client: cl, bucket: o.Bucket}, nil
}

func bucketLookup(pathStyle bool) minio.BucketLookupType {
	if pathStyle {
		return minio.BucketLookupPath
	}
	return minio.BucketLookupAuto
}

func (s *S3) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, r, size, minio.PutObjectOptions{ContentType: contentType})
	return err
}

func (s *S3) Get(ctx context.Context, key string) (io.ReadCloser, string, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, "", err
	}
	info, err := obj.Stat()
	ct := "application/octet-stream"
	if err == nil && info.ContentType != "" {
		ct = info.ContentType
	}
	return obj, ct, nil
}

func (s *S3) Delete(ctx context.Context, key string) error {
	return s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}
