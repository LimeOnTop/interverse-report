ALTER TABLE reports ADD COLUMN IF NOT EXISTS candidate_id UUID;
ALTER TABLE reports ADD COLUMN IF NOT EXISTS interviewer_id BIGINT;

UPDATE reports
SET interviewer_id = user_id
WHERE interviewer_id IS NULL AND user_id IS NOT NULL;

UPDATE reports
SET candidate_id = '00000000-0000-0000-0000-000000000000'
WHERE candidate_id IS NULL;

ALTER TABLE reports ALTER COLUMN candidate_id SET NOT NULL;
ALTER TABLE reports ALTER COLUMN interviewer_id SET NOT NULL;

ALTER TABLE reports DROP COLUMN IF EXISTS user_id;
