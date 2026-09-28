package entity

import "time"

type Report struct {
	ID                  string
	InterviewID         string
	UserID              int64
	OverallRating       string
	TechnicalSkills     string
	CommunicationSkills string
	ProblemSolving      string
	Strengths           string
	Weaknesses          string
	Recommendations     string
	Notes               string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// ReportMessage is the payload for training.report.analyze (analyze finished training session).
type ReportMessage struct {
	EventID     string `json:"event_id"` // idempotency key (UUID)
	ReportID    string `json:"report_id"`
	InterviewID string `json:"interview_id"`
	UserID      int64  `json:"user_id"`
}
