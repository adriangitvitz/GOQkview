package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"goqkview/analyzer"
	"goqkview/cmd"
	"goqkview/interfaces"
	"goqkview/output"
	"goqkview/parser"
	"goqkview/processor"
	"goqkview/providers/elasticsearch"
	"goqkview/providers/kafka"
	"goqkview/providers/local"
	"goqkview/providers/minio"
	"goqkview/repositories"
)

func main() {
	cfg, err := cmd.ParseFlags()
	if err != nil {
		log.Printf("Configuration error: %v", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("Shutdown signal received, stopping...")
		cancel()
	}()

	switch cfg.Mode {
	case cmd.ModeLocal:
		if err := runLocalMode(ctx, cfg); err != nil {
			log.Printf("Local mode error: %v", err)
			os.Exit(1)
		}
	case cmd.ModeDistributed:
		if err := runDistributedMode(ctx); err != nil && err != context.Canceled {
			log.Printf("Distributed mode error: %v", err)
			os.Exit(1)
		}
	}
}

func runLocalMode(ctx context.Context, cfg *cmd.Config) error {
	log.Printf("Processing local file: %s", cfg.FilePath)

	storage := local.NewLocalStorage(cfg.FilePath)
	events := local.NewLocalEventSource(cfg.FilePath)
	indexer := local.NewMemoryIndexer()

	p := parser.NewParser(parser.DateParseOptions{})

	proc, err := processor.New(processor.Config{
		Storage: storage,
		Events:  events,
		Indexer: indexer,
		Parser:  p,
	})
	if err != nil {
		return err
	}
	defer proc.Close()

	if err := proc.Run(ctx); err != nil && err != context.Canceled {
		return err
	}

	entries := indexer.GetEntries()
	log.Printf("Collected %d log entries", len(entries))

	bigipConfig := proc.GetBigIPConfig()

	a := analyzer.New()
	result, err := a.Analyze(entries, bigipConfig)
	if err != nil {
		return err
	}

	writer := output.NewWriter(cfg.OutputPath, cfg.Stdout)
	if err := writer.Write(result); err != nil {
		return err
	}

	if !cfg.Stdout {
		log.Printf("Analysis written to: %s", cfg.OutputPath)
	}

	return nil
}

func runDistributedMode(ctx context.Context) error {
	storage, err := minio.New(interfaces.StorageConfig{
		Endpoint:  os.Getenv("ENDPOINT"),
		AccessKey: os.Getenv("ACCESSKEY"),
		SecretKey: os.Getenv("SECRETKEY"),
		UseSSL:    false,
	})
	if err != nil {
		return err
	}

	events, err := kafka.New(interfaces.EventSourceConfig{
		Brokers:   []string{os.Getenv("BOOTSTRAP")},
		Topic:     os.Getenv("TOPIC"),
		Username:  os.Getenv("KAFKAUSER"),
		Password:  os.Getenv("PASSWORD"),
		Mechanism: os.Getenv("MECHANISM"),
	})
	if err != nil {
		return err
	}

	indexer, err := elasticsearch.New(interfaces.IndexerConfig{
		Addresses: []string{os.Getenv("ELASTIC_ENDPOINT")},
		Password:  os.Getenv("ELASTIC_PASSWORD"),
		IndexName: os.Getenv("ELASTIC_INDEX"),
	})
	if err != nil {
		return err
	}

	var db *repositories.PostgresDB
	if os.Getenv("POSTGRES_HOST") != "" {
		db, err = repositories.NewPostgresDB(repositories.ConfigFromEnv())
		if err != nil {
			return err
		}
	}

	p := parser.NewParser(parser.DateParseOptions{})

	proc, err := processor.New(processor.Config{
		Storage:  storage,
		Events:   events,
		Indexer:  indexer,
		Parser:   p,
		Database: db,
	})
	if err != nil {
		return err
	}
	defer proc.Close()

	log.Println("Starting GOQkview processor in distributed mode...")
	return proc.Run(ctx)
}
