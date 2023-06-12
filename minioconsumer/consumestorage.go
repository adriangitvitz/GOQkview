package minioconsumer

import (
	"encoding/json"
	"log"
	"minioconsumer/models"
	"minioconsumer/storage"

	"github.com/Shopify/sarama"
)

type Consumer struct {
	Bootstrap string
	Mechanism string
	User      string
	Password  string
	Topic     string
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
	for msg := range partitiionconsumer.Messages() {
		mresponse := models.Minioresponse{}
		err := json.Unmarshal(msg.Value, &mresponse)
		if err != nil {
			log.Fatalf("%s", err.Error())
		}
		for _, v := range mresponse.Records {
			log.Printf("Getting file: %s", v.Bucketinfo.Object.Key)
			s.Getlog(v.Bucketinfo.Bucket.Name, v.Bucketinfo.Object.Key)
		}
	}
}
