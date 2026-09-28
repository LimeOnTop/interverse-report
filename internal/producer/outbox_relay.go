package producer

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/LimeOnTop/interverse-report/internal/usecase"
)

// OutboxRelayDispatcher polls the outbox and publishes claimed rows to the broker.
type OutboxRelayDispatcher struct {
	outbox       usecase.OutboxRepository
	broker       usecase.BrokerPublisher
	batchSize    int64
	pollInterval time.Duration
	maxAttempts  int
}

type RelayConfig struct {
	BatchSize    int64
	PollInterval time.Duration
	MaxAttempts  int
}

func NewOutboxRelay(
	outbox usecase.OutboxRepository,
	broker usecase.BrokerPublisher,
	cfg RelayConfig,
) *OutboxRelayDispatcher {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 50
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = time.Second
	}
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 10
	}
	return &OutboxRelayDispatcher{
		outbox:       outbox,
		broker:       broker,
		batchSize:    cfg.BatchSize,
		pollInterval: cfg.PollInterval,
		maxAttempts:  cfg.MaxAttempts,
	}
}

var _ usecase.OutboxRelay = (*OutboxRelayDispatcher)(nil)

func (r *OutboxRelayDispatcher) Run(ctx context.Context) error {
	ticker := time.NewTicker(r.pollInterval)
	defer ticker.Stop()

	for {
		if err := r.dispatchBatch(ctx); err != nil {
			log.Printf("outbox relay batch error: %v", err)
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func (r *OutboxRelayDispatcher) dispatchBatch(ctx context.Context) error {
	messages, err := r.outbox.ClaimPending(ctx, r.batchSize)
	if err != nil {
		return fmt.Errorf("claim pending outbox: %w", err)
	}
	if len(messages) == 0 {
		return nil
	}

	for _, msg := range messages {
		if err := ctx.Err(); err != nil {
			return err
		}

		headers := map[string]string{
			"event_id":     msg.EventID,
			"content_type": "application/json",
		}

		publishErr := r.broker.PublishBytes(ctx, msg.Topic, msg.PartitionKey, msg.Payload, headers)
		if publishErr == nil {
			if err := r.outbox.MarkPublished(ctx, msg.ID); err != nil {
				return fmt.Errorf("mark outbox published id=%d: %w", msg.ID, err)
			}
			continue
		}

		errText := truncateErr(publishErr.Error(), 1024)
		if msg.Attempts >= r.maxAttempts {
			if err := r.outbox.MarkFailed(ctx, msg.ID, errText); err != nil {
				return fmt.Errorf("mark outbox failed id=%d: %w", msg.ID, err)
			}
			log.Printf("outbox message id=%d moved to failed after %d attempts: %v", msg.ID, msg.Attempts, publishErr)
			continue
		}

		if err := r.outbox.Requeue(ctx, msg.ID, errText); err != nil {
			return fmt.Errorf("requeue outbox id=%d: %w", msg.ID, err)
		}
		log.Printf("outbox message id=%d requeued attempt=%d: %v", msg.ID, msg.Attempts, publishErr)
	}

	return nil
}

func truncateErr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
