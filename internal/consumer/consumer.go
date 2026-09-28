package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/LimeOnTop/interverse-report/internal/apperr"
	"github.com/LimeOnTop/interverse-report/internal/entity"
	"github.com/LimeOnTop/interverse-report/internal/usecase"
)

const (
	headerEventID       = "event_id"
	headerError         = "x-error"
	headerRetryCount    = "x-retry-count"
	headerOriginalTopic = "x-original-topic"
)

type Config struct {
	Brokers      []string
	Topic        string
	DLQTopic     string
	GroupID      string
	MaxRetries   int
	RetryBackoff time.Duration
}

type ReportConsumer struct {
	group        sarama.ConsumerGroup
	dlq          sarama.SyncProducer
	topic        string
	dlqTopic     string
	handler      usecase.TrainingReportAnalyzer
	idempotency  usecase.IdempotencyStore
	maxRetries   int
	retryBackoff time.Duration
}

func NewReportConsumer(
	cfg Config,
	handler usecase.TrainingReportAnalyzer,
	idempotency usecase.IdempotencyStore,
) (*ReportConsumer, error) {
	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("kafka brokers are required")
	}
	if cfg.Topic == "" || cfg.DLQTopic == "" || cfg.GroupID == "" {
		return nil, fmt.Errorf("kafka topic, dlq topic and group id are required")
	}
	if handler == nil {
		return nil, fmt.Errorf("training report analyzer is required")
	}
	if idempotency == nil {
		return nil, fmt.Errorf("idempotency store is required")
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 3
	}
	if cfg.RetryBackoff <= 0 {
		cfg.RetryBackoff = 500 * time.Millisecond
	}

	saramaCfg := sarama.NewConfig()
	saramaCfg.ClientID = "interverse-report-consumer"
	saramaCfg.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{
		sarama.NewBalanceStrategyRange(),
	}
	saramaCfg.Consumer.Offsets.Initial = sarama.OffsetNewest
	// At-least-once: mark/commit only after successful handling (or DLQ publish).
	saramaCfg.Consumer.Offsets.AutoCommit.Enable = false
	saramaCfg.Consumer.Return.Errors = true
	saramaCfg.Version = sarama.V3_6_0_0

	group, err := sarama.NewConsumerGroup(cfg.Brokers, cfg.GroupID, saramaCfg)
	if err != nil {
		return nil, fmt.Errorf("create consumer group: %w", err)
	}

	producerCfg := sarama.NewConfig()
	producerCfg.ClientID = "interverse-report-dlq-producer"
	producerCfg.Producer.RequiredAcks = sarama.WaitForAll
	producerCfg.Producer.Retry.Max = 5
	producerCfg.Producer.Return.Successes = true
	producerCfg.Producer.Idempotent = true
	producerCfg.Net.MaxOpenRequests = 1
	producerCfg.Version = sarama.V3_6_0_0

	dlq, err := sarama.NewSyncProducer(cfg.Brokers, producerCfg)
	if err != nil {
		_ = group.Close()
		return nil, fmt.Errorf("create dlq producer: %w", err)
	}

	return &ReportConsumer{
		group:        group,
		dlq:          dlq,
		topic:        cfg.Topic,
		dlqTopic:     cfg.DLQTopic,
		handler:      handler,
		idempotency:  idempotency,
		maxRetries:   cfg.MaxRetries,
		retryBackoff: cfg.RetryBackoff,
	}, nil
}

// Run blocks until ctx is cancelled or the consumer group fails fatally.
func (c *ReportConsumer) Run(ctx context.Context) error {
	h := &groupHandler{parent: c}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case err, ok := <-c.group.Errors():
				if !ok {
					return
				}
				log.Printf("kafka consumer group error: %v", err)
			}
		}
	}()

	for {
		if err := ctx.Err(); err != nil {
			return nil
		}
		if err := c.group.Consume(ctx, []string{c.topic}, h); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("consume: %w", err)
		}
		// Rebalance: Consume returns nil; loop to rejoin.
	}
}

func (c *ReportConsumer) Close() error {
	var firstErr error
	if c.group != nil {
		if err := c.group.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if c.dlq != nil {
		if err := c.dlq.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

type groupHandler struct {
	parent *ReportConsumer
}

func (h *groupHandler) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (h *groupHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (h *groupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case <-session.Context().Done():
			return nil
		case msg, ok := <-claim.Messages():
			if !ok {
				return nil
			}
			h.parent.handleMessage(session, msg)
		}
	}
}

func (c *ReportConsumer) handleMessage(session sarama.ConsumerGroupSession, msg *sarama.ConsumerMessage) {
	ctx := session.Context()

	var payload entity.ReportMessage
	if err := json.Unmarshal(msg.Value, &payload); err != nil {
		log.Printf("invalid report message at %s/%d@%d: %v", msg.Topic, msg.Partition, msg.Offset, err)
		if dlqErr := c.publishDLQ(ctx, msg, err, 0); dlqErr != nil {
			log.Printf("dlq publish failed: %v", dlqErr)
			return // do not mark — retry later
		}
		session.MarkMessage(msg, "")
		session.Commit()
		return
	}

	idemKey := payload.EventID
	if idemKey == "" {
		idemKey = headerValue(msg, headerEventID)
	}
	if idemKey == "" {
		idemKey = fmt.Sprintf("%s:%d:%d", msg.Topic, msg.Partition, msg.Offset)
	}

	claimed, err := c.idempotency.Claim(ctx, idemKey)
	if err != nil {
		log.Printf("idempotency claim failed event=%s: %v", idemKey, err)
		return // retry without mark
	}
	if !claimed {
		apperr.Logf("skip duplicate event=%s report=%s", idemKey, payload.ReportID)
		session.MarkMessage(msg, "")
		session.Commit()
		return
	}

	var handleErr error
	for attempt := 1; attempt <= c.maxRetries; attempt++ {
		handleErr = c.handler.AnalyzeTrainingReport(ctx, payload)
		if handleErr == nil {
			session.MarkMessage(msg, "")
			session.Commit()
			return
		}
		apperr.Logf(
			"handle report event=%s attempt=%d/%d: %v",
			idemKey, attempt, c.maxRetries, handleErr,
		)
		if attempt < c.maxRetries {
			select {
			case <-ctx.Done():
				_ = c.idempotency.Release(context.Background(), idemKey)
				return
			case <-time.After(c.retryBackoff * time.Duration(attempt)):
			}
		}
	}

	if err := c.idempotency.Release(ctx, idemKey); err != nil {
		log.Printf("idempotency release failed event=%s: %v", idemKey, err)
	}

	if dlqErr := c.publishDLQ(ctx, msg, handleErr, c.maxRetries); dlqErr != nil {
		log.Printf("dlq publish failed event=%s: %v", idemKey, dlqErr)
		return // keep offset uncommitted for another delivery
	}

	// Permanent failure path: committed after DLQ so the poison message does not block the partition.
	session.MarkMessage(msg, "")
	session.Commit()
}

func (c *ReportConsumer) publishDLQ(ctx context.Context, original *sarama.ConsumerMessage, cause error, retries int) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	headers := []sarama.RecordHeader{
		{Key: []byte(headerOriginalTopic), Value: []byte(original.Topic)},
		{Key: []byte(headerRetryCount), Value: []byte(fmt.Sprintf("%d", retries))},
	}
	if cause != nil {
		headers = append(headers, sarama.RecordHeader{
			Key:   []byte(headerError),
			Value: []byte(truncate(cause.Error(), 1024)),
		})
	}
	for _, h := range original.Headers {
		headers = append(headers, sarama.RecordHeader{Key: h.Key, Value: h.Value})
	}

	_, _, err := c.dlq.SendMessage(&sarama.ProducerMessage{
		Topic:     c.dlqTopic,
		Key:       sarama.ByteEncoder(original.Key),
		Value:     sarama.ByteEncoder(original.Value),
		Headers:   headers,
		Timestamp: time.Now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("send to dlq: %w", err)
	}
	return nil
}

func headerValue(msg *sarama.ConsumerMessage, key string) string {
	for _, h := range msg.Headers {
		if string(h.Key) == key {
			return string(h.Value)
		}
	}
	return ""
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func SplitBrokers(brokers string) []string {
	parts := strings.Split(brokers, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
