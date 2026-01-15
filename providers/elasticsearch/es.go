package elasticsearch

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/google/uuid"

	"goqkview/interfaces"
)

type ElasticsearchIndexer struct {
	client    *elasticsearch.Client
	indexName string
}

func New(cfg interfaces.IndexerConfig) (*ElasticsearchIndexer, error) {
	esCfg := elasticsearch.Config{
		Addresses: cfg.Addresses,
	}
	if cfg.Password != "" {
		esCfg.Password = cfg.Password
	}
	if cfg.Username != "" {
		esCfg.Username = cfg.Username
	}

	client, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		return nil, fmt.Errorf("elasticsearch: failed to create client: %w", err)
	}

	return &ElasticsearchIndexer{
		client:    client,
		indexName: cfg.IndexName,
	}, nil
}

func (e *ElasticsearchIndexer) Index(ctx context.Context, entry interfaces.LogEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("elasticsearch: failed to marshal entry: %w", err)
	}

	docID := uuid.New().String()
	req := esapi.IndexRequest{
		Index:      e.indexName,
		DocumentID: docID,
		Body:       strings.NewReader(string(data)),
		Refresh:    "true",
	}

	res, err := req.Do(ctx, e.client)
	if err != nil {
		return fmt.Errorf("elasticsearch: failed to index document: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("elasticsearch: index request failed: %s", res.Status())
	}
	return nil
}

func (e *ElasticsearchIndexer) IndexBatch(ctx context.Context, entries []interfaces.LogEntry) error {
	for _, entry := range entries {
		if err := e.Index(ctx, entry); err != nil {
			return err
		}
	}
	return nil
}

func (e *ElasticsearchIndexer) Close() error {
	return nil
}

var _ interfaces.LogIndexer = (*ElasticsearchIndexer)(nil)
