package qkviewparse

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type QKviewparser struct {
	Path string
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
		log.Printf("File extracted to: %s", destinationpath)
	}
}

func (q QKviewparser) Read() {
	q.extract()
}
