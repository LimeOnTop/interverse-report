CREATE TABLE IF NOT EXISTS reports (
    id UUID PRIMARY KEY,
    interview_id UUID NOT NULL,
    candidate_id UUID NOT NULL,
    interviewer_id BIGINT NOT NULL,
    overall_rating VARCHAR(50),
    technical_skills TEXT,
    communication_skills TEXT,
    problem_solving TEXT,
    strengths TEXT,
    weaknesses TEXT,
    recommendations TEXT,
    notes TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
