package minioconsumer

import (
	"encoding/json"
	"log"
	"minioconsumer/models"
	"minioconsumer/storage"
	"os"
	"strings"

	"github.com/Shopify/sarama"
	"github.com/elastic/go-elasticsearch/v8"
)

type Consumer struct {
	Bootstrap string
	Mechanism string
	User      string
	Password  string
	Topic     string
	Elastic   elasticsearch.Config
}

// https://github.com/file/file/blob/f2a6e7cb7db9b5fd86100403df6b2f830c7f22ba/src/encoding.c#L151-L228
func (c Consumer) charidentities() map[byte]bool {
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

func (c Consumer) Consume(s *storage.Storage) {
	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true
	config.Net.SASL.Enable = true
	config.Net.SASL.Mechanism = sarama.SASLMechanism(c.Mechanism)
	config.Net.SASL.User = c.User
	config.Net.SASL.Password = c.Password

	consumer, err := sarama.NewConsumer([]string{c.Bootstrap}, config)
	if err != nil {
		log.Fatalf("%s", err.Error())
	}
	defer consumer.Close()
	partitiionconsumer, err := consumer.ConsumePartition(c.Topic, 0, sarama.OffsetNewest)
	if err != nil {
		log.Fatalf("%s", err.Error())
	}
	defer partitiionconsumer.Close()
	chars := c.charidentities()
	es, err := elasticsearch.NewClient(c.Elastic)
	log.Println("Connecting to Elastic")
	if err != nil {
		log.Fatalf("Error connecting to elastic: %s", err.Error())
	}
	for msg := range partitiionconsumer.Messages() {
		mresponse := models.Minioresponse{}
		err := json.Unmarshal(msg.Value, &mresponse)
		if err != nil {
			log.Fatalf("%s", err.Error())
		}
		for _, v := range mresponse.Records {
			log.Printf("Getting file: %s", v.Bucketinfo.Object.Key)
			fnameparts := strings.Split(v.Bucketinfo.Object.Key, "/")
			fname := fnameparts[0]
			if len(fnameparts) > 1 {
				fname = fnameparts[len(fnameparts)-1]
			}
			dirname := strings.Split(fname, ".")[0]
			s.Getlog(v.Bucketinfo.Bucket.Name, v.Bucketinfo.Object.Key, fname, chars, es)
			err := os.Remove(fname)
			if err != nil {
				log.Fatalf("Error removing file: %s", err.Error())
			}
			err = os.RemoveAll(dirname)
			if err != nil {
				log.Fatalf("Error removing file: %s", err.Error())
			}
		}
	}
}
