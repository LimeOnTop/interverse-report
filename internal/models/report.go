package models

import (
	"time"
)

type Report struct {
	ID                  string    `json:"id" db:"id"`
	InterviewID         string    `json:"interview_id" db:"interview_id"`
	CandidateID         string    `json:"candidate_id" db:"candidate_id"`
	InterviewerID       string    `json:"interviewer_id" db:"interviewer_id"`
	OverallRating       string    `json:"overall_rating" db:"overall_rating"`
	TechnicalSkills     string    `json:"technical_skills" db:"technical_skills"`
	CommunicationSkills string    `json:"communication_skills" db:"communication_skills"`
	ProblemSolving      string    `json:"problem_solving" db:"problem_solving"`
	Strengths           string    `json:"strengths" db:"strengths"`
	Weaknesses          string    `json:"weaknesses" db:"weaknesses"`
	Recommendations     string    `json:"recommendations" db:"recommendations"`
	Notes               string    `json:"notes" db:"notes"`
	CreatedAt           time.Time `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time `json:"updated_at" db:"updated_at"`
}

