ALTER TABLE reports ADD COLUMN IF NOT EXISTS user_id BIGINT;

UPDATE reports
SET user_id = interviewer_id
WHERE user_id IS NULL AND interviewer_id IS NOT NULL;

ALTER TABLE reports ALTER COLUMN user_id SET NOT NULL;

ALTER TABLE reports DROP COLUMN IF EXISTS candidate_id;
ALTER TABLE reports DROP COLUMN IF EXISTS interviewer_id;
