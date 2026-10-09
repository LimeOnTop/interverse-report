package usecase

import "context"

type SessionItem struct {
	ID         string
	QuestionID string
	ItemType   string
	SortOrder  int32
	Text       string
	Technology string
	Difficulty string
	Category   string
	Options    []string
}

type InterviewSummary struct {
	ID             string
	UserID         int64
	Title          string
	Description    string
	Status         string
	ScheduledAt    string
	Level          string
	Specialization string
}

type QuestionOption struct {
	Text      string
	IsCorrect bool
	SortOrder int32
}

type QuestionDetails struct {
	ID         string
	Text       string
	Category   string
	Difficulty string
	Technology string
	Answer     string
	Options    []QuestionOption
}

type GeminiAnalysis struct {
	OverallScore      int    `json:"overall_score"`
	AlgorithmScore    int    `json:"algorithm_score"`
	ArchitectureScore int    `json:"architecture_score"`
	CodingScore       int    `json:"coding_score"`
	SoftSkillsScore   int    `json:"soft_skills_score"`
	Comments          string `json:"comments"`
	SummaryPublic     string `json:"summary_public"`
	Strengths         string `json:"strengths"`
	Weaknesses        string `json:"weaknesses"`
	Recommendations   string `json:"recommendations"`
}

type AnswerInput struct {
	StepID              string
	QuestionID          string
	ItemType            string
	SelectedOptionIndex *int32
	TaskAnswer          string
}
type Analysis interface {
	Generate(context.Context, string, int64, []AnswerInput, bool) (ReportDTO, AnalysisScoresDTO, error)
}
type InterviewReader interface {
	GetSessionContent(context.Context, string, int64) (InterviewSummary, []SessionItem, []SessionItem, error)
	CompleteInterview(context.Context, InterviewSummary) error
}
type QuestionReader interface {
	GetByID(context.Context, string) (QuestionDetails, error)
}
type Analyzer interface {
	Analyze(context.Context, string) (GeminiAnalysis, error)
}
type JSONCompleter interface {
	CompleteJSON(context.Context, string, string) (string, error)
}
type AnswerCache interface {
	Save(context.Context, string, []AnswerInput) error
	Load(context.Context, string) ([]AnswerInput, error)
}
