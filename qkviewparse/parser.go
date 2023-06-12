package qkviewparse

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"
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

func (q QKviewparser) readlogs() {
	logpath := fmt.Sprintf("%s/var/log", q.Path)
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
		if info.IsDir() {
			fmt.Println(path)
		} else {
			if !strings.Contains(info.Name(), "audit") {
				// TODO: Check if file is binary
				fmt.Println(path)
			}
		}
		return nil
	})
	if error != nil {
		log.Fatal(error.Error())
	}
}

func (q QKviewparser) Read() {
	q.extract()
	q.readlogs()
}
