package storage

import (
	"context"
	"log"
	"minioconsumer/qkviewparse"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Storage struct {
	Endpoint  string `default:""`
	Accesskey string `default:""`
	Secretkey string `default:""`
}

func (s Storage) Getlog(bucket string, miniopath string, fname string, chars map[byte]bool, es *elasticsearch.Client) {
	// NOTE: Uncomment to use defaults
	// utils.Setdefault(&s, "default")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	minioclient, err := minio.New(s.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(s.Accesskey, s.Secretkey, ""),
		Secure: false,
	})
	if err != nil {
		log.Fatalf("Minio error: %s", err.Error())
	}

	err = minioclient.FGetObject(ctx, bucket, miniopath, fname, minio.GetObjectOptions{})
	if err != nil {
		log.Fatalf("Minio object: %s", err.Error())
	}
	log.Printf("File Downloaded: %s", fname)
	q := qkviewparse.QKviewparser{
		Path: fname,
	}
	q.Read(chars, es)
}
