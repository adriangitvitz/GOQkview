package interfaces

import (
	"context"
	"time"
)

type LogEntry struct {
	Path      string    `json:"path"`
	Line      string    `json:"line"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Source    string    `json:"source,omitempty"` // qkview filename
}

type LogIndexer interface {
	Index(ctx context.Context, entry LogEntry) error
	IndexBatch(ctx context.Context, entries []LogEntry) error
	Close() error
}

type IndexerConfig struct {
	Addresses []string
	Username  string
	Password  string
	IndexName string
	BatchSize int // For batch operations
}
