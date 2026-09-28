ALTER TABLE reports
    ADD COLUMN IF NOT EXISTS status VARCHAR(32) NOT NULL DEFAULT 'ready',
    ADD COLUMN IF NOT EXISTS error_message TEXT;

UPDATE reports SET status = 'ready' WHERE status IS NULL OR status = '';
