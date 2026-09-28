CREATE TABLE IF NOT EXISTS outbox_messages (
    id BIGSERIAL PRIMARY KEY,
    topic VARCHAR(255) NOT NULL,
    partition_key VARCHAR(255) NOT NULL DEFAULT '',
    payload JSONB NOT NULL,
    event_id UUID NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    attempts INT NOT NULL DEFAULT 0,
    last_error TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    published_at TIMESTAMP NULL,
    CONSTRAINT outbox_messages_status_check
        CHECK (status IN ('pending', 'publishing', 'published', 'failed')),
    CONSTRAINT outbox_messages_event_id_unique UNIQUE (event_id)
);

CREATE INDEX IF NOT EXISTS idx_outbox_messages_pending
    ON outbox_messages (created_at ASC)
    WHERE status = 'pending';

CREATE INDEX IF NOT EXISTS idx_outbox_messages_status_created
    ON outbox_messages (status, created_at);
