package minio

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"goqkview/interfaces"
)

type MinIOStorage struct {
	client *minio.Client
}

func New(cfg interfaces.StorageConfig) (*MinIOStorage, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio: failed to create client: %w", err)
	}
	return &MinIOStorage{client: client}, nil
}

func (m *MinIOStorage) Download(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	obj, err := m.client.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("minio: failed to get object %s/%s: %w", bucket, key, err)
	}
	return obj, nil
}

func (m *MinIOStorage) DownloadToFile(ctx context.Context, bucket, key, destPath string) error {
	err := m.client.FGetObject(ctx, bucket, key, destPath, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("minio: failed to download %s/%s to %s: %w", bucket, key, destPath, err)
	}
	return nil
}

func (m *MinIOStorage) Close() error {
	return nil
}

var _ interfaces.StorageBackend = (*MinIOStorage)(nil)
