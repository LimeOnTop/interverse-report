package usecase

import (
	"context"

	"github.com/LimeOnTop/interverse-report/internal/entity"
)

// ReportProducer enqueues a training-report analysis job (outbox write).
// It must not talk to Kafka directly — the outbox relay publishes later.
type ReportProducer interface {
	Publish(ctx context.Context, report entity.ReportMessage) error
}

// OutboxRepository persists and claims outbox rows.
type OutboxRepository interface {
	Enqueue(ctx context.Context, message entity.OutboxMessage) (entity.OutboxMessage, error)
	ClaimPending(ctx context.Context, limit int64) ([]entity.OutboxMessage, error)
	MarkPublished(ctx context.Context, id int64) error
	MarkFailed(ctx context.Context, id int64, lastError string) error
	Requeue(ctx context.Context, id int64, lastError string) error
}

// BrokerPublisher sends already-serialized messages to the message broker (Kafka).
// Used only by the outbox relay, not by business use cases.
type BrokerPublisher interface {
	PublishBytes(ctx context.Context, topic, key string, payload []byte, headers map[string]string) error
	Close() error
}

// OutboxRelay polls pending outbox rows and publishes them through BrokerPublisher.
type OutboxRelay interface {
	Run(ctx context.Context) error
}

// TxManager runs fn inside a DB transaction; repositories honor tx from context.
type TxManager interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
