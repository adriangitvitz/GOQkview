package main

import (
	"minioconsumer/minioconsumer"
	"minioconsumer/storage"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	m := minioconsumer.Consumer{
		Bootstrap: os.Getenv("BOOTSTRAP"),
		Mechanism: os.Getenv("MECHANISM"),
		User:      os.Getenv("KAFKAUSER"),
		Password:  os.Getenv("PASSWORD"),
		Topic:     os.Getenv("TOPIC"),
	}
	s := storage.Storage{
		Endpoint:  os.Getenv("ENDPOINT"),
		Accesskey: os.Getenv("ACCESSKEY"),
		Secretkey: os.Getenv("SECRETKEY"),
	}
	m.Consume(&s)
}
