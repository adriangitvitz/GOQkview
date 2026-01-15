package local

import (
	"context"
	"path/filepath"

	"goqkview/interfaces"
)

type LocalEventSource struct {
	filePath string
	fired    bool
}

func NewLocalEventSource(filePath string) *LocalEventSource {
	return &LocalEventSource{filePath: filePath}
}

func (l *LocalEventSource) Subscribe(ctx context.Context, handler interfaces.EventHandler) error {
	if l.fired {
		return nil
	}
	l.fired = true

	event := interfaces.Event{
		Bucket: "local",
		Key:    l.filePath,
		Metadata: map[string]string{
			"filename": filepath.Base(l.filePath),
			"mode":     "local",
		},
	}

	return handler(ctx, event)
}

func (l *LocalEventSource) Close() error {
	return nil
}

var _ interfaces.EventSource = (*LocalEventSource)(nil)
