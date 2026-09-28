package entity

import "time"

const (
	OutboxStatusPending    = "pending"
	OutboxStatusPublishing = "publishing"
	OutboxStatusPublished  = "published"
	OutboxStatusFailed     = "failed"
)

// OutboxMessage is a durable broker intent stored in the same DB as domain writes.
type OutboxMessage struct {
	ID           int64
	Topic        string
	PartitionKey string
	Payload      []byte
	EventID      string
	Status       string
	Attempts     int
	LastError    string
	CreatedAt    time.Time
	PublishedAt  *time.Time
}
