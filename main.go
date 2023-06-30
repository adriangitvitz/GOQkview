package main

import (
	"minioconsumer/minioconsumer"
	"minioconsumer/repositories"
	"minioconsumer/storage"
	"os"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	cfg := elasticsearch.Config{
		Password: os.Getenv("ELASTIC_PASSWORD"),
		Addresses: []string{
			os.Getenv("ELASTIC_ENDPOINT"),
		},
	}
	m := minioconsumer.Consumer{
		Bootstrap: os.Getenv("BOOTSTRAP"),
		Mechanism: os.Getenv("MECHANISM"),
		User:      os.Getenv("KAFKAUSER"),
		Password:  os.Getenv("PASSWORD"),
		Topic:     os.Getenv("TOPIC"),
		Elastic:   cfg,
	}
	s := storage.Storage{
		Endpoint:  os.Getenv("ENDPOINT"),
		Accesskey: os.Getenv("ACCESSKEY"),
		Secretkey: os.Getenv("SECRETKEY"),
	}
    repositories.Connectpostgres()
	m.Consume(&s)
}
