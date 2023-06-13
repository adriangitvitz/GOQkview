package qkviewparse

import (
	"archive/tar"
	"bufio"
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

// https://github.com/file/file/blob/f2a6e7cb7db9b5fd86100403df6b2f830c7f22ba/src/encoding.c#L151-L228
func (q QKviewparser) charidentities() map[byte]bool {
	char_array := []byte{7, 8, 9, 10, 12, 13, 27}
	for i := 0x20; i < 0x100; i++ {
		if i != 0x7F {
			char_array = append(char_array, byte(i))
		}
	}
	charmap := make(map[byte]bool)
	for _, i := range char_array {
		charmap[i] = true
	}
	return charmap
}

func (q QKviewparser) checkifbinary(path string) (error, bool) {
	file, err := os.Open(path)
	if err != nil {
		return err, false
	}
	defer file.Close()
	chars := q.charidentities()
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

func (q QKviewparser) readlogs() {
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
		// if info.IsDir() {
		// 	fmt.Println(path)
		// }
		if !info.IsDir() {
			if !strings.Contains(info.Name(), "audit") {
				if info.Size() > 0 {
					err, ok := q.checkifbinary(path)
					if err != nil {
						log.Fatalf("Error in binary checks: %s", err.Error())
					}
					if !ok {
						fmt.Println(path)
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

func (q QKviewparser) Read() {
	q.extract()
	q.readlogs()
}
