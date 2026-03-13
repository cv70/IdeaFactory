package infra

import (
	"backend/config"
	"context"

	"github.com/IBM/sarama"
)

func NewKafkaSyncProducer(ctx context.Context, c *config.KafkaConfig) (sarama.SyncProducer, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Producer.Return.Successes = true
	return sarama.NewSyncProducer([]string{c.Host}, saramaConfig)
}

func NewKafkaAsyncProducer(ctx context.Context, c *config.KafkaConfig) (sarama.AsyncProducer, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Producer.Return.Successes = true
	return sarama.NewAsyncProducer([]string{c.Host}, saramaConfig)
}

func NewKafkaConsumer(ctx context.Context, c *config.KafkaConfig) (sarama.Consumer, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Consumer.Return.Errors = true
	return sarama.NewConsumer([]string{c.Host}, saramaConfig)
}
