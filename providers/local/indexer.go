package local

import (
	"context"
	"sync"

	"goqkview/interfaces"
)

type MemoryIndexer struct {
	entries []interfaces.LogEntry
	mu      sync.Mutex
}

func NewMemoryIndexer() *MemoryIndexer {
	return &MemoryIndexer{
		entries: make([]interfaces.LogEntry, 0, 10000),
	}
}

func (m *MemoryIndexer) Index(ctx context.Context, entry interfaces.LogEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = append(m.entries, entry)
	return nil
}

func (m *MemoryIndexer) IndexBatch(ctx context.Context, entries []interfaces.LogEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = append(m.entries, entries...)
	return nil
}

func (m *MemoryIndexer) GetEntries() []interfaces.LogEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]interfaces.LogEntry, len(m.entries))
	copy(result, m.entries)
	return result
}

func (m *MemoryIndexer) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = m.entries[:0]
}

func (m *MemoryIndexer) Close() error {
	return nil
}

var _ interfaces.LogIndexer = (*MemoryIndexer)(nil)
