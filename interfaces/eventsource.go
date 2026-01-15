package interfaces

import (
	"context"
)

type Event struct {
	Bucket   string
	Key      string
	Metadata map[string]string
}

type EventHandler func(ctx context.Context, event Event) error

type EventSource interface {
	Subscribe(ctx context.Context, handler EventHandler) error
	Close() error
}

type EventSourceConfig struct {
	Brokers   []string
	Topic     string
	GroupID   string
	Username  string
	Password  string
	Mechanism string // SASL mechanism
}
