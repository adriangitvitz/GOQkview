package qkviewparse

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/google/uuid"
)

type QKviewparser struct {
	Path string
}

type ElasticIndex struct {
	Path   string
	Line   string
	Status string
	Date   time.Time
}

func (q QKviewparser) extract() {
	file, err := os.Open(q.Path)
	if err != nil {
		log.Fatalf("Error extracting: %s", err.Error())
	}
	defer file.Close()
	gzip_reader, err := gzip.NewReader(file)
	if err != nil {
		log.Fatalf("Error extracting: %s", err.Error())
	}
	defer gzip_reader.Close()
	tar_reader := tar.NewReader(gzip_reader)
	dirname := strings.Split(q.Path, ".")[0]
	for {
		header, err := tar_reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("Error reading content: %s", err.Error())
			return
		}
		destinationpath := filepath.Join(dirname, header.Name)
		destinationdir := filepath.Dir(destinationpath)
		if err := os.MkdirAll(destinationdir, os.ModePerm); err != nil {
			log.Fatalf("Error reading content: %s", err.Error())
			return
		}
		destination, err := os.Create(destinationpath)
		if err != nil {
			log.Fatalf("Error saving: %s", err.Error())
			return
		}
		defer destination.Close()
		_, err = io.Copy(destination, tar_reader)
		if err != nil {
			log.Fatalf("Error saving: %s", err.Error())
			return
		}
	}
}

func (q QKviewparser) checkifexists(path string) error {
	_, err := os.Stat(path)
	if err == nil {
		return nil
	} else if err.(*os.PathError).Err == syscall.ENOTDIR {
		// WARNING: Path is not a directory
		return err
	} else {
		return err
	}
}

func (q QKviewparser) checkifbinary(path string, chars map[byte]bool) (error, bool) {
	file, err := os.Open(path)
	if err != nil {
		return err, false
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	buffer := make([]byte, 1024)
	_, err = reader.Read(buffer)
	if err != nil {
		return err, false
	}
	for _, b := range buffer {
		if _, ok := chars[b]; !ok {
			return nil, true
		}
	}
	return nil, false
}

func (q QKviewparser) saveindex(es *elasticsearch.Client, index ElasticIndex) {
	index_str, err := json.Marshal(index)
	if err != nil {
		log.Fatalf("Error parsing index: %s", err.Error())
	}
	docid := uuid.New().String()
	if docid == "" {
		log.Fatalf("Error processing uuid")
	}
	req := esapi.IndexRequest{
		Index:      os.Getenv("ELASTIC_INDEX"),
		DocumentID: docid,
		Body:       strings.NewReader(string(index_str)),
		Refresh:    "true",
	}
	res, err := req.Do(context.Background(), es)
	if err != nil {
		log.Fatalf("Error sending index: %s", err.Error())
	}
	defer res.Body.Close()
	if res.IsError() {
		log.Fatalf("Error sending index: %s", res.Status())
	}
}

func (q QKviewparser) readlines(path string, es *elasticsearch.Client) {
	log.Printf("Processing file: %s\n", path)
	f, err := os.Open(path)
	if err != nil {
		log.Fatalf("Error reading file: %s", err.Error())
	}
	defer f.Close()
	f_scanner := bufio.NewScanner(f)
	// Oct 14 13:00:00 2020 or 2023-10-24 13:00:00 or Doc 14 13:00:00
	re := regexp.MustCompile(`(\b(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s+\d{1,2}\s+\d{2}:\d{2}:\d{2}\s+\d{4}\b)|(\b\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\b)|(\b(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s+\d{1,2}\s+\d{2}:\d{2}:\d{2}\b)`)
	// Case sensitive status codes
	re_status := regexp.MustCompile(`(?i)(warning|error|severe|critical|notice)`)
	ansic_layout := "Jan _2 15:04:05 2006"
	without_year_ansic := "Jan _2 15:04:05 2006"
	date_layout := "2006-01-02 15:04:05"
	for f_scanner.Scan() {
		line := f_scanner.Text()
		match_status := re_status.FindStringSubmatch(line)
		if len(match_status) > 0 {
			matches := re.FindStringSubmatch(line)
			if len(matches) > 0 {
				if matches[1] != "" {
					d, error := time.Parse(ansic_layout, matches[1])
					if error != nil {
						log.Fatal(error.Error())
					}
					q.saveindex(es, ElasticIndex{
						Path:   path,
						Line:   line,
						Date:   d,
						Status: strings.ToUpper(match_status[0]),
					})
				} else if matches[3] != "" {
					d, error := time.Parse(date_layout, matches[3])
					if error != nil {
						log.Fatal(error.Error())
					}
					q.saveindex(es, ElasticIndex{
						Path:   path,
						Line:   line,
						Date:   d,
						Status: strings.ToUpper(match_status[0]),
					})
				} else if matches[4] != "" {
					d, error := time.Parse(without_year_ansic, matches[4]+" "+strconv.Itoa(time.Now().Year()))
					if error != nil {
						log.Fatal(error.Error())
					}
					q.saveindex(es, ElasticIndex{
						Path:   path,
						Line:   line,
						Date:   d,
						Status: strings.ToUpper(match_status[0]),
					})
				}
			}
		}
	}
	if err := f_scanner.Err(); err != nil {
		log.Fatalf("Error reading file lines: %s", err.Error())
	}
}

func (q QKviewparser) readlogs(chars map[byte]bool, es *elasticsearch.Client) {
	logpath := fmt.Sprintf("%s/var/log", strings.Split(q.Path, ".")[0])
	err := q.checkifexists(logpath)
	if err != nil {
		log.Fatalf("Error searching path: %s", err.Error())
	}
	error := filepath.Walk(logpath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && info.Name() == "journal" {
			return filepath.SkipDir
		}
		if !info.IsDir() {
			if !strings.Contains(info.Name(), "audit") {
				if info.Size() > 0 {
					err, ok := q.checkifbinary(path, chars)
					if err != nil {
						log.Fatalf("Error in binary checks: %s", err.Error())
					}
					if !ok {
						q.readlines(path, es)
					}
				}
			}
		}
		return nil
	})
	if error != nil {
		log.Fatal(error.Error())
	}
}

func (q QKviewparser) Read(chars map[byte]bool, es *elasticsearch.Client) {
	q.extract()
	q.readlogs(chars, es)
}
