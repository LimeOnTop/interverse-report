package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/LimeOnTop/interverse-report/internal/entity"
	"github.com/LimeOnTop/interverse-report/internal/usecase"
)

type OutboxRepository struct {
	db *sql.DB
}

func NewOutboxRepository(db *sql.DB) *OutboxRepository {
	return &OutboxRepository{db: db}
}

var _ usecase.OutboxRepository = (*OutboxRepository)(nil)

func (r *OutboxRepository) Enqueue(ctx context.Context, message entity.OutboxMessage) (entity.OutboxMessage, error) {
	if message.Status == "" {
		message.Status = entity.OutboxStatusPending
	}

	query := `
		INSERT INTO outbox_messages (
			topic, partition_key, payload, event_id, status, attempts, created_at
		) VALUES ($1, $2, $3, $4, $5, 0, NOW())
		RETURNING id, topic, partition_key, payload, event_id, status, attempts, last_error, created_at, published_at
	`

	var lastError sql.NullString
	var publishedAt sql.NullTime
	err := r.querier(ctx).QueryRowContext(
		ctx,
		query,
		message.Topic,
		message.PartitionKey,
		message.Payload,
		message.EventID,
		message.Status,
	).Scan(
		&message.ID,
		&message.Topic,
		&message.PartitionKey,
		&message.Payload,
		&message.EventID,
		&message.Status,
		&message.Attempts,
		&lastError,
		&message.CreatedAt,
		&publishedAt,
	)
	if err != nil {
		return entity.OutboxMessage{}, fmt.Errorf("enqueue outbox message: %w", err)
	}
	if lastError.Valid {
		message.LastError = lastError.String
	}
	if publishedAt.Valid {
		t := publishedAt.Time
		message.PublishedAt = &t
	}
	return message, nil
}

func (r *OutboxRepository) ClaimPending(ctx context.Context, limit int64) ([]entity.OutboxMessage, error) {
	if limit <= 0 {
		limit = 50
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin outbox claim tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	selectQuery := `
		SELECT id
		FROM outbox_messages
		WHERE status = $1
		ORDER BY created_at ASC, id ASC
		LIMIT $2
		FOR UPDATE SKIP LOCKED
	`
	rows, err := tx.QueryContext(ctx, selectQuery, entity.OutboxStatusPending, limit)
	if err != nil {
		return nil, fmt.Errorf("select pending outbox: %w", err)
	}

	ids := make([]int64, 0, limit)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan outbox id: %w", err)
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate outbox ids: %w", err)
	}
	if len(ids) == 0 {
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("commit empty outbox claim: %w", err)
		}
		return nil, nil
	}

	updateQuery := `
		UPDATE outbox_messages
		SET status = $1, attempts = attempts + 1
		WHERE id = $2
		RETURNING id, topic, partition_key, payload, event_id, status, attempts, last_error, created_at, published_at
	`

	messages := make([]entity.OutboxMessage, 0, len(ids))
	for _, id := range ids {
		row := tx.QueryRowContext(ctx, updateQuery, entity.OutboxStatusPublishing, id)
		msg, err := scanOutboxMessage(row)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit outbox claim: %w", err)
	}
	return messages, nil
}

func (r *OutboxRepository) MarkPublished(ctx context.Context, id int64) error {
	query := `
		UPDATE outbox_messages
		SET status = $1, published_at = NOW(), last_error = NULL
		WHERE id = $2
	`
	if _, err := r.querier(ctx).ExecContext(ctx, query, entity.OutboxStatusPublished, id); err != nil {
		return fmt.Errorf("mark outbox published: %w", err)
	}
	return nil
}

func (r *OutboxRepository) MarkFailed(ctx context.Context, id int64, lastError string) error {
	query := `
		UPDATE outbox_messages
		SET status = $1, last_error = $2
		WHERE id = $3
	`
	if _, err := r.querier(ctx).ExecContext(ctx, query, entity.OutboxStatusFailed, lastError, id); err != nil {
		return fmt.Errorf("mark outbox failed: %w", err)
	}
	return nil
}

func (r *OutboxRepository) Requeue(ctx context.Context, id int64, lastError string) error {
	query := `
		UPDATE outbox_messages
		SET status = $1, last_error = $2
		WHERE id = $3
	`
	if _, err := r.querier(ctx).ExecContext(ctx, query, entity.OutboxStatusPending, lastError, id); err != nil {
		return fmt.Errorf("requeue outbox message: %w", err)
	}
	return nil
}

func (r *OutboxRepository) querier(ctx context.Context) DBTX {
	return DBTXFromContext(ctx, r.db)
}

func scanOutboxMessage(row *sql.Row) (entity.OutboxMessage, error) {
	var msg entity.OutboxMessage
	var lastError sql.NullString
	var publishedAt sql.NullTime
	if err := row.Scan(
		&msg.ID,
		&msg.Topic,
		&msg.PartitionKey,
		&msg.Payload,
		&msg.EventID,
		&msg.Status,
		&msg.Attempts,
		&lastError,
		&msg.CreatedAt,
		&publishedAt,
	); err != nil {
		return entity.OutboxMessage{}, fmt.Errorf("scan outbox message: %w", err)
	}
	if lastError.Valid {
		msg.LastError = lastError.String
	}
	if publishedAt.Valid {
		t := publishedAt.Time
		msg.PublishedAt = &t
	}
	return msg, nil
}
