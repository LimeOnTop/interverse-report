ALTER TABLE reports
    DROP COLUMN IF EXISTS error_message,
    DROP COLUMN IF EXISTS status;
