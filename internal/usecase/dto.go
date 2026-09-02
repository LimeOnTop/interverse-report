package usecase

import "time"

type ReportDTO struct {
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

type AnalysisScoresDTO struct {
	OverallScore      int
	AlgorithmScore    int
	ArchitectureScore int
	CodingScore       int
	SoftSkillsScore   int
	Comments          string
}
