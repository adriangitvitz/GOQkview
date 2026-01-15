package interfaces

import (
	"context"
	"io"
)

type StorageBackend interface {
	Download(ctx context.Context, bucket, key string) (io.ReadCloser, error)
	DownloadToFile(ctx context.Context, bucket, key, destPath string) error
	Close() error
}

type StorageConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	UseSSL    bool
	Region    string // Optional, for S3-compatible services
}
