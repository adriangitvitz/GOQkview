package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/Shopify/sarama"

	"goqkview/interfaces"
)

type KafkaEventSource struct {
	consumer  sarama.Consumer
	partition sarama.PartitionConsumer
	topic     string
}

type minioEvent struct {
	Records []struct {
		S3 struct {
			Bucket struct {
				Name string `json:"name"`
			} `json:"bucket"`
			Object struct {
				Key      string `json:"key"`
				Metadata struct {
					Metauuid string `json:"X-Amz-Meta-Uuid"`
				} `json:"userMetadata"`
			} `json:"object"`
		} `json:"s3"`
	} `json:"Records"`
}

func New(cfg interfaces.EventSourceConfig) (*KafkaEventSource, error) {
	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true

	if cfg.Username != "" {
		config.Net.SASL.Enable = true
		config.Net.SASL.Mechanism = sarama.SASLMechanism(cfg.Mechanism)
		config.Net.SASL.User = cfg.Username
		config.Net.SASL.Password = cfg.Password
	}

	consumer, err := sarama.NewConsumer(cfg.Brokers, config)
	if err != nil {
		return nil, fmt.Errorf("kafka: failed to create consumer: %w", err)
	}

	partition, err := consumer.ConsumePartition(cfg.Topic, 0, sarama.OffsetNewest)
	if err != nil {
		consumer.Close()
		return nil, fmt.Errorf("kafka: failed to consume partition: %w", err)
	}

	return &KafkaEventSource{
		consumer:  consumer,
		partition: partition,
		topic:     cfg.Topic,
	}, nil
}

func (k *KafkaEventSource) Subscribe(ctx context.Context, handler interfaces.EventHandler) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-k.partition.Errors():
			log.Printf("kafka: partition error: %v", err)
		case msg := <-k.partition.Messages():
			events, err := k.parseMessage(msg.Value)
			if err != nil {
				log.Printf("kafka: failed to parse message: %v", err)
				continue
			}
			for _, event := range events {
				if err := handler(ctx, event); err != nil {
					log.Printf("kafka: handler error for %s/%s: %v", event.Bucket, event.Key, err)
				}
			}
		}
	}
}

func (k *KafkaEventSource) parseMessage(data []byte) ([]interfaces.Event, error) {
	var mEvent minioEvent
	if err := json.Unmarshal(data, &mEvent); err != nil {
		return nil, err
	}

	events := make([]interfaces.Event, 0, len(mEvent.Records))
	for _, r := range mEvent.Records {
		metadata := make(map[string]string)
		if r.S3.Object.Metadata.Metauuid != "" {
			metadata["X-Amz-Meta-Uuid"] = r.S3.Object.Metadata.Metauuid
		}
		events = append(events, interfaces.Event{
			Bucket:   r.S3.Bucket.Name,
			Key:      r.S3.Object.Key,
			Metadata: metadata,
		})
	}
	return events, nil
}

func (k *KafkaEventSource) Close() error {
	if err := k.partition.Close(); err != nil {
		return fmt.Errorf("kafka: failed to close partition: %w", err)
	}
	if err := k.consumer.Close(); err != nil {
		return fmt.Errorf("kafka: failed to close consumer: %w", err)
	}
	return nil
}

var _ interfaces.EventSource = (*KafkaEventSource)(nil)
