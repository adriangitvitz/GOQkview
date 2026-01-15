package local

import (
	"context"
	"fmt"
	"io"
	"os"

	"goqkview/interfaces"
)

type LocalStorage struct {
	basePath string
}

func NewLocalStorage(filePath string) *LocalStorage {
	return &LocalStorage{basePath: filePath}
}

func (l *LocalStorage) Download(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	file, err := os.Open(l.basePath)
	if err != nil {
		return nil, fmt.Errorf("local: failed to open file %s: %w", l.basePath, err)
	}
	return file, nil
}

func (l *LocalStorage) DownloadToFile(ctx context.Context, bucket, key, destPath string) error {
	if l.basePath == destPath {
		return nil
	}

	src, err := os.Open(l.basePath)
	if err != nil {
		return fmt.Errorf("local: failed to open source: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("local: failed to create destination: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("local: failed to copy file: %w", err)
	}

	return nil
}

func (l *LocalStorage) Close() error {
	return nil
}

var _ interfaces.StorageBackend = (*LocalStorage)(nil)
