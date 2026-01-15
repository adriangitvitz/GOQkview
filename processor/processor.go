package processor

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"goqkview/interfaces"
	"goqkview/parser"
	"goqkview/repositories"
)

type Processor struct {
	storage interfaces.StorageBackend
	events  interfaces.EventSource
	indexer interfaces.LogIndexer
	parser  *parser.Parser
	db *repositories.PostgresDB
	bigipConfig *parser.BigIPConfig
}

type Config struct {
	Storage  interfaces.StorageBackend
	Events   interfaces.EventSource
	Indexer  interfaces.LogIndexer
	Parser   *parser.Parser
	Database *repositories.PostgresDB // Optional
}

func New(cfg Config) (*Processor, error) {
	if cfg.Storage == nil {
		return nil, fmt.Errorf("processor: storage backend is required")
	}
	if cfg.Events == nil {
		return nil, fmt.Errorf("processor: event source is required")
	}
	if cfg.Indexer == nil {
		return nil, fmt.Errorf("processor: log indexer is required")
	}

	p := cfg.Parser
	if p == nil {
		p = parser.NewParser(parser.DateParseOptions{})
	}

	return &Processor{
		storage: cfg.Storage,
		events:  cfg.Events,
		indexer: cfg.Indexer,
		parser:  p,
		db:      cfg.Database,
	}, nil
}

func (p *Processor) Run(ctx context.Context) error {
	log.Println("Processor: starting event consumption...")

	return p.events.Subscribe(ctx, func(ctx context.Context, event interfaces.Event) error {
		return p.handleEvent(ctx, event)
	})
}

func (p *Processor) handleEvent(ctx context.Context, event interfaces.Event) error {
	log.Printf("Processor: handling event for %s/%s", event.Bucket, event.Key)
	if p.db != nil {
		uuid := event.Metadata["X-Amz-Meta-Uuid"]
		if uuid == "" {
			uuid = event.Metadata["x-amz-meta-uuid"]
		}

		upload, err := p.db.FindUnprocessedUpload(uuid, "logs")
		if err != nil {
			log.Printf("Processor: skipping %s (not found in tracking DB or already processed)", event.Key)
			return nil // Not an error, just skip
		}

		event.Bucket = upload.Bucket
	}

	keyParts := strings.Split(event.Key, "/")
	filename := keyParts[len(keyParts)-1]
	localPath := filepath.Join(os.TempDir(), filename)

	if err := p.storage.DownloadToFile(ctx, event.Bucket, event.Key, localPath); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	log.Printf("Processor: downloaded %s to %s", event.Key, localPath)

	result, err := p.parser.ProcessFile(ctx, localPath, p.indexer)
	if err != nil {
		p.cleanup(localPath)
		return fmt.Errorf("processing failed: %w", err)
	}

	if result.BigIPConfig != nil {
		p.bigipConfig = result.BigIPConfig
	}

	log.Printf("Processor: processed %s - found %d entries, indexed %d, errors: %d",
		filename, result.EntriesFound, result.EntriesIndexed, len(result.Errors))

	for _, e := range result.Errors {
		log.Printf("Processor: non-fatal error: %v", e)
	}

	if p.db != nil {
		uuid := event.Metadata["X-Amz-Meta-Uuid"]
		if uuid == "" {
			uuid = event.Metadata["x-amz-meta-uuid"]
		}
		if err := p.db.MarkProcessed(uuid); err != nil {
			log.Printf("Processor: failed to mark as processed: %v", err)
		}
	}

	p.cleanup(localPath)

	return nil
}

func (p *Processor) cleanup(localPath string) {
	if err := os.Remove(localPath); err != nil && !os.IsNotExist(err) {
		log.Printf("Processor: failed to remove %s: %v", localPath, err)
	}

	extractDir := strings.TrimSuffix(localPath, filepath.Ext(localPath))
	extractDir = strings.TrimSuffix(extractDir, ".tar")
	if err := os.RemoveAll(extractDir); err != nil && !os.IsNotExist(err) {
		log.Printf("Processor: failed to remove directory %s: %v", extractDir, err)
	}
}

func (p *Processor) GetBigIPConfig() *parser.BigIPConfig {
	return p.bigipConfig
}

func (p *Processor) Close() error {
	var errs []error

	if err := p.events.Close(); err != nil {
		errs = append(errs, fmt.Errorf("events close: %w", err))
	}
	if err := p.storage.Close(); err != nil {
		errs = append(errs, fmt.Errorf("storage close: %w", err))
	}
	if err := p.indexer.Close(); err != nil {
		errs = append(errs, fmt.Errorf("indexer close: %w", err))
	}
	if p.db != nil {
		if err := p.db.Close(); err != nil {
			errs = append(errs, fmt.Errorf("database close: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}
	return nil
}
