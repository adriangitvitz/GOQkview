# GOQkview

A pluggable system for processing **qkview** diagnostic files. Supports both local file processing and distributed mode with Kafka/MinIO/Elasticsearch.

## Quick Start

### Local Mode (No Dependencies)

Process a qkview file directly without any external services:

```bash
# Build
go build -o goqkview .

# Process a file - outputs metadata.json in same directory
./goqkview --file /path/to/qkview.tar.gz

# Output to stdout (for piping)
./goqkview --file /path/to/qkview.tar.gz --stdout

# Custom output path
./goqkview --file /path/to/qkview.tar.gz --output /path/to/output.json
```

### Output Format

Local mode generates a `metadata.json` with:

```json
{
  "summary": {
    "critical": 3,
    "warning": 7,
    "healthy": 12,
    "certsExpiringSoon": 2
  },
  "errorTimeline": [
    { "date": "2026-01-05", "errors": 847 }
  ],
  "sslFindings": [
    {
      "severity": "critical",
      "type": "certificate",
      "message": "Certificate expiration detected",
      "detail": "...",
      "affectedVS": ["vs_production_443"]
    }
  ],
  "topErrors": [
    {
      "message": "SSL handshake failed",
      "count": 847,
      "lastOccurred": "2026-01-08T12:34:00Z"
    }
  ],
  "recommendations": [
    {
      "priority": "critical",
      "title": "Certificate renewal required",
      "description": "...",
      "impact": "..."
    }
  ]
}
```

## Architecture

GOQkview uses a pluggable architecture with three core interfaces:

- **StorageBackend** - Object storage (MinIO, local filesystem, or custom)
- **EventSource** - Event consumption (Kafka, local file, or custom)
- **LogIndexer** - Log indexing (Elasticsearch, memory, or custom)

### Project Structure

```
goqkview/
├── main.go                      # Entry point with mode routing
├── cmd/cli.go                   # CLI flag parsing
├── interfaces/                  # Core interfaces
├── providers/
│   ├── minio/                   # MinIO storage
│   ├── kafka/                   # Kafka events
│   ├── elasticsearch/           # ES indexer
│   └── local/                   # Local mode providers
├── analyzer/                    # Analysis engine
│   ├── analyzer.go              # Orchestrator
│   ├── ssl.go                   # SSL/TLS analysis
│   ├── errors.go                # Error grouping
│   ├── timeline.go              # Timeline aggregation
│   └── recommendations.go       # Recommendations
├── output/writer.go             # JSON output
├── parser/                      # Log parsing
├── processor/                   # Processing orchestration
└── repositories/                # PostgreSQL (optional)
```

## Installation

```bash
go mod download
go build -o goqkview .
```

## Usage Modes

### Local Mode

Process files locally without external services:

```bash
./goqkview --file /path/to/qkview.tar.gz
```

### Distributed Mode

Run with Kafka/MinIO/Elasticsearch (requires environment variables):

```bash
./goqkview
```

## Configuration (Distributed Mode)

### Environment Variables

**Storage (MinIO):**

```bash
ENDPOINT=minio:9000
ACCESSKEY=minioadmin
SECRETKEY=minioadmin
```

**Event Source (Kafka):**

```bash
BOOTSTRAP=kafka:9092
TOPIC=minio-events
KAFKAUSER=user
PASSWORD=password
MECHANISM=PLAIN
```

**Indexer (Elasticsearch):**

```bash
ELASTIC_ENDPOINT=http://elasticsearch:9200
ELASTIC_PASSWORD=elastic
ELASTIC_INDEX=qkview-logs
```

**Database (PostgreSQL) - Optional:**

```bash
POSTGRES_HOST=postgres
POSTGRES_PORT=5432
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres
POSTGRES_DB=qkview
```

### Docker Services

**Kafka** (`.docker/kafka`):

- Create a `.env` file:
  ```
  KAFKA_BROKER_HEAP_OPTS="-XX:MaxRAMPercentage=70.0"
  DOCKER_HOST_IP="your-ip"
  ```

**MinIO** (`.docker/minio`):

- Create a `.env` file:
  ```
  MINIO_USER="miniouser"
  MINIO_PASSWORD="miniopassword"
  ```
- Create a Kafka event destination in the MinIO web console
- Subscribe your bucket to the event destination

**Elasticsearch** (`.docker/elastic`):

- Create a `.env` file:
  ```
  ELASTIC_PASSWORD="password"
  MEM_LIMIT="memorylimit"
  ```

## Features

- **Local Mode** - Process files without external services
- **Distributed Mode** - Scale with Kafka/MinIO/Elasticsearch
- **Full Analysis** - SSL/TLS findings, error grouping, timeline, recommendations
- **Pluggable Architecture** - Swap storage, event sources, and indexers
- **Multiple Date Formats** - Smart year inference for logs without year
- **Graceful Shutdown** - Handles SIGINT/SIGTERM signals
- **Error Recovery** - Continues processing on non-fatal errors

## Analysis Features

### SSL/TLS Analysis

Detects:
- Certificate expiration warnings
- Weak cipher suites (RC4, DES, NULL, EXPORT, MD5)
- Obsolete protocols (TLS 1.0, TLS 1.1, SSLv2, SSLv3)
- Handshake failures

### Error Analysis

- Groups similar errors (normalizes IPs, ports)
- Counts occurrences
- Tracks last occurrence
- Returns top 10 most frequent

### Recommendations

Generates prioritized action items based on:
- SSL/TLS findings
- Error frequency
- Summary statistics

## Supported Log Formats

Date formats:
- `Oct 14 13:00:00 2020` (with year)
- `2023-10-24 13:00:00` (ISO format)
- `2023-10-24T13:00:00Z` (RFC3339)
- `Oct 14 13:00:00` (without year - infers from context)

Status levels: `WARNING`, `ERROR`, `SEVERE`, `CRITICAL`, `NOTICE`

## Custom Implementations

Implement your own backends by satisfying the interfaces:

```go
package main

import (
    "context"
    "goqkview/interfaces"
    "goqkview/processor"
)

type MyStorage struct{}
func (s *MyStorage) Download(ctx context.Context, bucket, key string) (io.ReadCloser, error) { ... }
func (s *MyStorage) DownloadToFile(ctx context.Context, bucket, key, destPath string) error { ... }
func (s *MyStorage) Close() error { return nil }

type MyEventSource struct{}
func (e *MyEventSource) Subscribe(ctx context.Context, handler interfaces.EventHandler) error { ... }
func (e *MyEventSource) Close() error { return nil }

type MyIndexer struct{}
func (i *MyIndexer) Index(ctx context.Context, entry interfaces.LogEntry) error { ... }
func (i *MyIndexer) IndexBatch(ctx context.Context, entries []interfaces.LogEntry) error { ... }
func (i *MyIndexer) Close() error { return nil }

func main() {
    proc, _ := processor.New(processor.Config{
        Storage: &MyStorage{},
        Events:  &MyEventSource{},
        Indexer: &MyIndexer{},
    })
    defer proc.Close()
    proc.Run(context.Background())
}
```

## Dependencies

Core dependencies (installed via `go mod download`):

- `github.com/Shopify/sarama` - Kafka client (distributed mode)
- `github.com/minio/minio-go/v7` - MinIO client (distributed mode)
- `github.com/elastic/go-elasticsearch/v8` - Elasticsearch client (distributed mode)
- `github.com/google/uuid` - UUID generation
- `gorm.io/gorm` - PostgreSQL ORM (optional)

**Note:** Local mode uses only Go standard library for file processing.
