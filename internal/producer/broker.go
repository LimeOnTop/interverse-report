package producer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/LimeOnTop/interverse-report/internal/usecase"
)

// MessageBroker publishes serialized payloads to the message broker (outbox relay transport).
type MessageBroker struct {
	producer sarama.SyncProducer
}

type BrokerConfig struct {
	Addresses []string
}

func NewMessageBroker(cfg BrokerConfig) (*MessageBroker, error) {
	if len(cfg.Addresses) == 0 {
		return nil, fmt.Errorf("broker addresses are required")
	}

	saramaCfg := sarama.NewConfig()
	saramaCfg.ClientID = "interverse-report-broker"
	saramaCfg.Producer.RequiredAcks = sarama.WaitForAll
	saramaCfg.Producer.Retry.Max = 5
	saramaCfg.Producer.Retry.Backoff = 100 * time.Millisecond
	saramaCfg.Producer.Return.Successes = true
	saramaCfg.Producer.Return.Errors = true
	saramaCfg.Producer.Compression = sarama.CompressionSnappy
	saramaCfg.Producer.Idempotent = true
	saramaCfg.Net.MaxOpenRequests = 1
	saramaCfg.Version = sarama.V3_6_0_0

	producer, err := sarama.NewSyncProducer(cfg.Addresses, saramaCfg)
	if err != nil {
		return nil, fmt.Errorf("create message broker: %w", err)
	}

	return &MessageBroker{producer: producer}, nil
}

var _ usecase.BrokerPublisher = (*MessageBroker)(nil)

func (b *MessageBroker) PublishBytes(
	ctx context.Context,
	topic, key string,
	payload []byte,
	headers map[string]string,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	recordHeaders := make([]sarama.RecordHeader, 0, len(headers))
	for k, v := range headers {
		recordHeaders = append(recordHeaders, sarama.RecordHeader{
			Key:   []byte(k),
			Value: []byte(v),
		})
	}

	msg := &sarama.ProducerMessage{
		Topic:     topic,
		Key:       sarama.StringEncoder(key),
		Value:     sarama.ByteEncoder(payload),
		Headers:   recordHeaders,
		Timestamp: time.Now().UTC(),
	}

	if _, _, err := b.producer.SendMessage(msg); err != nil {
		return fmt.Errorf("publish message: %w", err)
	}
	return nil
}

func (b *MessageBroker) Close() error {
	if b.producer == nil {
		return nil
	}
	return b.producer.Close()
}

func SplitAddresses(addresses string) []string {
	parts := strings.Split(addresses, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
