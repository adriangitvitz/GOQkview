package storage

import (
	"context"
	"log"
	"minioconsumer/qkviewparse"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Storage struct {
	Endpoint  string `default:""`
	Accesskey string `default:""`
	Secretkey string `default:""`
}

func (s Storage) Getlog(bucket string, miniopath string, chars map[byte]bool, es *elasticsearch.Client) {
	// NOTE: Uncomment to use defaults
	// utils.Setdefault(&s, "default")
	minioclient, err := minio.New(s.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(s.Accesskey, s.Secretkey, ""),
		Secure: false,
	})
	if err != nil {
		log.Fatalf("Minio error: %s", err.Error())
	}

	fnameparts := strings.Split(miniopath, "/")
	fname := fnameparts[0]
	if len(fnameparts) > 1 {
		fname = fnameparts[len(fnameparts)-1]
	}

	err = minioclient.FGetObject(context.Background(), bucket, miniopath, fname, minio.GetObjectOptions{})
	if err != nil {
		log.Fatalf("Minio object: %s", err.Error())
	}
	log.Printf("File Downloaded: %s", fname)
	q := qkviewparse.QKviewparser{
		Path: fname,
	}
	q.Read(chars, es)
}
