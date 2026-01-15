package cmd

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

type Mode int

const (
	ModeDistributed Mode = iota
	ModeLocal
)

type Config struct {
	Mode       Mode
	FilePath   string // For local mode: path to qkview file
	OutputPath string // Output path for metadata.json
	Stdout     bool   // Print to stdout instead of file
}

func ParseFlags() (*Config, error) {
	cfg := &Config{}

	file := flag.String("file", "", "Path to qkview.tar.gz file (enables local mode)")
	output := flag.String("output", "", "Output path for metadata.json (default: same directory as input)")
	stdout := flag.Bool("stdout", false, "Print JSON output to stdout instead of file")
	help := flag.Bool("help", false, "Show help message")

	flag.Parse()

	if *help {
		printUsage()
		os.Exit(0)
	}

	if *file != "" {
		cfg.Mode = ModeLocal

		absPath, err := filepath.Abs(*file)
		if err != nil {
			return nil, fmt.Errorf("invalid file path: %w", err)
		}

		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", absPath)
		}

		cfg.FilePath = absPath
		cfg.Stdout = *stdout

		if *output != "" {
			cfg.OutputPath = *output
		} else if !*stdout {
			dir := filepath.Dir(absPath)
			cfg.OutputPath = filepath.Join(dir, "metadata.json")
		}
	} else {
		cfg.Mode = ModeDistributed
	}

	return cfg, nil
}

func printUsage() {
	fmt.Println(`GOQkview - Qkview Diagnostic File Processor

Usage:
  goqkview                                    Run in distributed mode (Kafka/MinIO/ES)
  goqkview --file /path/to/qkview.tar.gz     Process a local file
  goqkview --file /path/to/file --stdout     Output JSON to stdout
  goqkview --file /path/to/file --output /custom/path/metadata.json

Options:
  --file     Path to qkview.tar.gz file (enables local mode)
  --output   Custom output path for metadata.json (default: same directory as input)
  --stdout   Print JSON to stdout instead of writing to file
  --help     Show this help message

Environment Variables (distributed mode only):
  ENDPOINT, ACCESSKEY, SECRETKEY      MinIO configuration
  BOOTSTRAP, TOPIC, KAFKAUSER, etc.   Kafka configuration
  ELASTIC_ENDPOINT, ELASTIC_PASSWORD  Elasticsearch configuration
  POSTGRES_HOST (optional)            PostgreSQL for tracking`)
}
