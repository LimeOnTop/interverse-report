package entity

import "time"

type Report struct {
	ID                  string
	InterviewID         string
	CandidateID         string
	InterviewerID       string
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
