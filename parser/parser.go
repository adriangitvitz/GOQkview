package parser

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"goqkview/interfaces"
)

type Parser struct {
	binaryChars map[byte]bool
	dateOpts DateParseOptions
}

func NewParser(opts DateParseOptions) *Parser {
	return &Parser{
		binaryChars: buildBinaryCharMap(),
		dateOpts:    opts,
	}
}

// Based on file/file library encoding detection.
// https://github.com/file/file/blob/f2a6e7cb7db9b5fd86100403df6b2f830c7f22ba/src/encoding.c#L151-L228
func buildBinaryCharMap() map[byte]bool {
	charArray := []byte{7, 8, 9, 10, 12, 13, 27}
	for i := 0x20; i < 0x100; i++ {
		if i != 0x7F {
			charArray = append(charArray, byte(i))
		}
	}
	charMap := make(map[byte]bool)
	for _, b := range charArray {
		charMap[b] = true
	}
	return charMap
}

type ProcessResult struct {
	EntriesFound   int
	EntriesIndexed int
	Errors         []error
	BigIPConfig    *BigIPConfig
}

func (p *Parser) ProcessFile(ctx context.Context, filePath string, indexer interfaces.LogIndexer) (*ProcessResult, error) {
	log.Printf("ProcessFile called with: %s", filePath)
	result := &ProcessResult{}

	extractDir, err := p.extract(filePath)
	log.Printf("Extraction complete, extractDir: %s", extractDir)
	if err != nil {
		return nil, fmt.Errorf("parser: extraction failed: %w", err)
	}

	configPath := filepath.Join(extractDir, "config", "bigip.conf")
	log.Printf("Looking for BigIP config at: %s", configPath)
	if _, statErr := os.Stat(configPath); statErr == nil {
		log.Printf("Parsing BigIP configuration: %s", configPath)
		bigipConfig, parseErr := ParseBigIPConfig(configPath)
		if parseErr != nil {
			log.Printf("Warning: failed to parse BigIP config: %v", parseErr)
			result.Errors = append(result.Errors, fmt.Errorf("bigip config parse: %w", parseErr))
		} else {
			result.BigIPConfig = bigipConfig
			log.Printf("Found %d virtual servers and %d pools",
				len(bigipConfig.VirtualServers), len(bigipConfig.Pools))
		}
	} else {
		log.Printf("BigIP config not found at %s: %v", configPath, statErr)
	}

	logPath := filepath.Join(extractDir, "var", "log")
	if _, statErr := os.Stat(logPath); os.IsNotExist(statErr) {
		return nil, fmt.Errorf("parser: log directory not found: %s", logPath)
	}

	err = filepath.Walk(logPath, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			result.Errors = append(result.Errors, fmt.Errorf("walk error for %s: %w", path, walkErr))
			return nil // Continue walking
		}

		if info.IsDir() {
			if info.Name() == "journal" {
				return filepath.SkipDir
			}
			return nil
		}

		if strings.Contains(info.Name(), "audit") || info.Size() == 0 {
			return nil
		}

		isBinary, error := p.isBinaryFile(path)
		if error != nil {
			result.Errors = append(result.Errors, fmt.Errorf("binary check failed for %s: %w", path, err))
			return nil
		}
		if isBinary {
			return nil
		}

		entries, errs := p.parseLogFile(ctx, path, filePath)
		result.EntriesFound += len(entries)
		result.Errors = append(result.Errors, errs...)

		for _, entry := range entries {
			if error := indexer.Index(ctx, entry); err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("indexing failed: %w", error))
			} else {
				result.EntriesIndexed++
			}
		}

		return nil
	})

	if err != nil {
		return result, fmt.Errorf("parser: walk failed: %w", err)
	}

	return result, nil
}

func (p *Parser) extract(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return "", fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	destDir := strings.TrimSuffix(filePath, filepath.Ext(filePath))
	destDir = strings.TrimSuffix(destDir, ".tar") // Handle .tar.gz

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("failed to read tar header: %w", err)
		}

		destPath := filepath.Join(destDir, header.Name)
		destDirPath := filepath.Dir(destPath)

		if err := os.MkdirAll(destDirPath, os.ModePerm); err != nil {
			return "", fmt.Errorf("failed to create directory %s: %w", destDirPath, err)
		}

		if header.Typeflag == tar.TypeReg {
			dest, err := os.Create(destPath)
			if err != nil {
				return "", fmt.Errorf("failed to create file %s: %w", destPath, err)
			}
			if _, err := io.Copy(dest, tarReader); err != nil {
				dest.Close()
				return "", fmt.Errorf("failed to write file %s: %w", destPath, err)
			}
			dest.Close()
		}
	}

	return destDir, nil
}

func (p *Parser) isBinaryFile(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()

	buffer := make([]byte, 1024)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return false, err
	}

	for i := range n {
		if !p.binaryChars[buffer[i]] {
			return true, nil
		}
	}
	return false, nil
}

func GetExtractDir(filePath string) string {
	destDir := strings.TrimSuffix(filePath, filepath.Ext(filePath))
	destDir = strings.TrimSuffix(destDir, ".tar") // Handle .tar.gz
	return destDir
}

func (p *Parser) parseLogFile(ctx context.Context, path, source string) ([]interfaces.LogEntry, []error) {
	var entries []interfaces.LogEntry
	var errors []error

	log.Printf("Processing file: %s", path)

	file, err := os.Open(path)
	if err != nil {
		return nil, []error{fmt.Errorf("failed to open %s: %w", path, err)}
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return entries, append(errors, ctx.Err())
		default:
		}

		line := scanner.Text()

		status, hasStatus := ParseStatus(line)
		if !hasStatus {
			continue
		}

		timestamp, hasDate := ParseDate(line, p.dateOpts)
		if !hasDate {
			continue
		}

		entries = append(entries, interfaces.LogEntry{
			Path:      path,
			Line:      line,
			Status:    status,
			Timestamp: timestamp,
			Source:    source,
		})
	}

	if err := scanner.Err(); err != nil {
		errors = append(errors, fmt.Errorf("scanner error for %s: %w", path, err))
	}

	return entries, errors
}
