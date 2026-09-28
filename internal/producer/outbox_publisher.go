package producer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/LimeOnTop/interverse-report/internal/entity"
	"github.com/LimeOnTop/interverse-report/internal/usecase"
	"github.com/google/uuid"
)

// OutboxPublisher enqueues training-report analysis jobs into the outbox table.
type OutboxPublisher struct {
	outbox usecase.OutboxRepository
	topic  string
}

func NewOutboxPublisher(outbox usecase.OutboxRepository, topic string) *OutboxPublisher {
	return &OutboxPublisher{
		outbox: outbox,
		topic:  topic,
	}
}

var _ usecase.ReportProducer = (*OutboxPublisher)(nil)

func (p *OutboxPublisher) Publish(ctx context.Context, report entity.ReportMessage) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if p.topic == "" {
		return fmt.Errorf("outbox topic is required")
	}

	if report.EventID == "" {
		report.EventID = uuid.NewString()
	}

	payload, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("marshal report message: %w", err)
	}

	_, err = p.outbox.Enqueue(ctx, entity.OutboxMessage{
		Topic:        p.topic,
		PartitionKey: report.InterviewID,
		Payload:      payload,
		EventID:      report.EventID,
		Status:       entity.OutboxStatusPending,
	})
	if err != nil {
		return fmt.Errorf("enqueue training report analysis: %w", err)
	}
	return nil
}
