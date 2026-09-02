package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/LimeOnTop/interverse-report/internal/client"
	"github.com/LimeOnTop/interverse-report/internal/entity"
	"github.com/LimeOnTop/interverse-report/internal/usecase"
)

const trainingCandidateID = "00000000-0000-0000-0000-000000000000"

type analysisMetadata struct {
	OverallScore           int    `json:"overall_score"`
	AlgorithmScore         int    `json:"algorithm_score"`
	ArchitectureScore      int    `json:"architecture_score"`
	CodingScore            int    `json:"coding_score"`
	SoftSkillsScore        int    `json:"soft_skills_score"`
	Comments               string `json:"comments"`
	InterviewTitle         string `json:"interview_title"`
	InterviewLevel         string `json:"interview_level"`
	InterviewSpecialization string `json:"interview_specialization"`
	InterviewScheduledAt   string `json:"interview_scheduled_at"`
	MCQCorrect             int    `json:"mcq_correct"`
	MCQTotal               int    `json:"mcq_total"`
}

type AnswerInput struct {
	StepID               string
	QuestionID           string
	ItemType             string
	SelectedOptionIndex  *int32
	TaskAnswer           string
}

type AnalysisService struct {
	repository      usecase.ReportRepository
	interviewClient *client.InterviewClient
	questionClient  *client.QuestionClient
	geminiClient    *client.GeminiClient
}

func NewAnalysisService(
	repository usecase.ReportRepository,
	interviewClient *client.InterviewClient,
	questionClient *client.QuestionClient,
	geminiClient *client.GeminiClient,
) *AnalysisService {
	return &AnalysisService{
		repository:      repository,
		interviewClient: interviewClient,
		questionClient:  questionClient,
		geminiClient:    geminiClient,
	}
}

func (s *AnalysisService) Generate(
	ctx context.Context,
	interviewID, userID string,
	answers []AnswerInput,
) (usecase.ReportDTO, usecase.AnalysisScoresDTO, error) {
	existing, err := s.repository.GetByInterviewID(ctx, interviewID)
	if err == nil {
		scores, parseErr := scoresFromNotes(existing.Notes)
		if parseErr == nil {
			return toDTO(existing), scores, nil
		}
	}

	interview, questions, tasks, err := s.interviewClient.GetSessionContent(ctx, interviewID, userID)
	if err != nil {
		return usecase.ReportDTO{}, usecase.AnalysisScoresDTO{}, fmt.Errorf("load session: %w", err)
	}

	if interview.UserID != userID {
		return usecase.ReportDTO{}, usecase.AnalysisScoresDTO{}, fmt.Errorf("generate report: forbidden")
	}

	answerByStep := make(map[string]AnswerInput, len(answers))
	for _, answer := range answers {
		answerByStep[answer.StepID] = answer
	}

	mcqSummary, mcqCorrect, mcqTotal := s.scoreMCQ(ctx, questions, answerByStep)
	taskSummary := s.buildTaskSummary(ctx, tasks, answerByStep)

	prompt := buildAnalysisPrompt(interview, mcqSummary, mcqCorrect, mcqTotal, taskSummary)
	analysis, err := s.geminiClient.Analyze(ctx, prompt)
	if err != nil {
		analysis = fallbackAnalysis(mcqCorrect, mcqTotal, taskSummary)
	}

	if mcqTotal > 0 && analysis.AlgorithmScore == 0 {
		analysis.AlgorithmScore = int(float64(mcqCorrect) / float64(mcqTotal) * 100)
	}

	metadata := analysisMetadata{
		OverallScore:            analysis.OverallScore,
		AlgorithmScore:          analysis.AlgorithmScore,
		ArchitectureScore:       analysis.ArchitectureScore,
		CodingScore:             analysis.CodingScore,
		SoftSkillsScore:         analysis.SoftSkillsScore,
		Comments:                analysis.Comments,
		InterviewTitle:          interview.Title,
		InterviewLevel:          interview.Level,
		InterviewSpecialization: interview.Specialization,
		InterviewScheduledAt:    interview.ScheduledAt,
		MCQCorrect:              mcqCorrect,
		MCQTotal:                mcqTotal,
	}

	notesJSON, err := json.Marshal(metadata)
	if err != nil {
		return usecase.ReportDTO{}, usecase.AnalysisScoresDTO{}, fmt.Errorf("marshal analysis metadata: %w", err)
	}

	created, err := s.repository.Create(ctx, entity.Report{
		InterviewID:         interviewID,
		CandidateID:         trainingCandidateID,
		InterviewerID:       userID,
		OverallRating:       fmt.Sprintf("%d", analysis.OverallScore),
		TechnicalSkills:     fmt.Sprintf("%d", analysis.AlgorithmScore),
		CommunicationSkills: fmt.Sprintf("%d", analysis.SoftSkillsScore),
		ProblemSolving:      fmt.Sprintf("%d", analysis.CodingScore),
		Strengths:           analysis.Strengths,
		Weaknesses:          analysis.Weaknesses,
		Recommendations:     analysis.Recommendations,
		Notes:               string(notesJSON),
	})
	if err != nil {
		return usecase.ReportDTO{}, usecase.AnalysisScoresDTO{}, fmt.Errorf("save report: %w", err)
	}

	if err := s.interviewClient.CompleteInterview(ctx, interview); err != nil {
		return usecase.ReportDTO{}, usecase.AnalysisScoresDTO{}, fmt.Errorf("complete interview: %w", err)
	}

	scores := usecase.AnalysisScoresDTO{
		OverallScore:      analysis.OverallScore,
		AlgorithmScore:    analysis.AlgorithmScore,
		ArchitectureScore: analysis.ArchitectureScore,
		CodingScore:       analysis.CodingScore,
		SoftSkillsScore:   analysis.SoftSkillsScore,
		Comments:          analysis.Comments,
	}

	return toDTO(created), scores, nil
}

func (s *AnalysisService) scoreMCQ(
	ctx context.Context,
	questions []client.SessionItem,
	answers map[string]AnswerInput,
) (string, int, int) {
	var builder strings.Builder
	correct := 0
	total := 0

	for _, item := range questions {
		answer, ok := answers[item.ID]
		if !ok || answer.SelectedOptionIndex == nil {
			builder.WriteString(fmt.Sprintf("- [NO ANSWER] %s (%s)\n", item.Text, item.Category))
			total++
			continue
		}

		questionID := item.QuestionID
		if questionID == "" {
			questionID = answer.QuestionID
		}

		isCorrect := false
		if questionID != "" {
			details, err := s.questionClient.GetByID(ctx, questionID)
			if err == nil {
				selected := int(*answer.SelectedOptionIndex)
				for idx, option := range details.Options {
					if idx == selected && option.IsCorrect {
						isCorrect = true
						break
					}
				}
			}
		}

		total++
		if isCorrect {
			correct++
		}

		status := "INCORRECT"
		if isCorrect {
			status = "CORRECT"
		}

		builder.WriteString(fmt.Sprintf(
			"- [%s] %s (%s, technology=%s)\n",
			status,
			item.Text,
			item.Category,
			item.Technology,
		))
	}

	return builder.String(), correct, total
}

func (s *AnalysisService) buildTaskSummary(
	ctx context.Context,
	tasks []client.SessionItem,
	answers map[string]AnswerInput,
) string {
	var builder strings.Builder

	for idx, item := range tasks {
		answer, ok := answers[item.ID]
		candidateAnswer := ""
		if ok {
			candidateAnswer = strings.TrimSpace(answer.TaskAnswer)
		}
		if candidateAnswer == "" {
			candidateAnswer = "(no answer)"
		}

		referenceAnswer := ""
		questionID := item.QuestionID
		if questionID == "" && ok {
			questionID = answer.QuestionID
		}
		if questionID != "" {
			details, err := s.questionClient.GetByID(ctx, questionID)
			if err == nil {
				referenceAnswer = details.Answer
			}
		}

		builder.WriteString(fmt.Sprintf(
			"Task %d:\nQuestion: %s\nTechnology: %s\nCategory: %s\nReference answer: %s\nCandidate answer:\n%s\n\n",
			idx+1,
			item.Text,
			item.Technology,
			item.Category,
			referenceAnswer,
			candidateAnswer,
		))
	}

	return builder.String()
}

func buildAnalysisPrompt(
	interview client.InterviewSummary,
	mcqSummary string,
	mcqCorrect, mcqTotal int,
	taskSummary string,
) string {
	mcqPercent := 0
	if mcqTotal > 0 {
		mcqPercent = int(float64(mcqCorrect) / float64(mcqTotal) * 100)
	}

	return fmt.Sprintf(`You are a senior technical interviewer evaluating a training session.

Interview context:
- Title: %s
- Level: %s
- Specialization: %s
- Description: %s

MCQ results (%d/%d correct, %d%%):
%s

Practical tasks:
%s

Return ONLY valid JSON with this exact schema:
{
  "overall_score": 0,
  "algorithm_score": 0,
  "architecture_score": 0,
  "coding_score": 0,
  "soft_skills_score": 0,
  "comments": "detailed feedback in Russian",
  "strengths": "strengths in Russian",
  "weaknesses": "weaknesses in Russian",
  "recommendations": "recommendations in Russian"
}

Scoring rules:
- overall_score: weighted summary 0-100
- algorithm_score: MCQ accuracy and algorithmic thinking
- architecture_score: system design and architecture knowledge
- coding_score: quality of practical task answers
- soft_skills_score: clarity and completeness of explanations
- Use Russian for all text fields
- Be constructive and specific`,
		interview.Title,
		interview.Level,
		interview.Specialization,
		interview.Description,
		mcqCorrect,
		mcqTotal,
		mcqPercent,
		mcqSummary,
		taskSummary,
	)
}

func fallbackAnalysis(mcqCorrect, mcqTotal int, taskSummary string) client.GeminiAnalysis {
	mcqScore := 0
	if mcqTotal > 0 {
		mcqScore = int(float64(mcqCorrect) / float64(mcqTotal) * 100)
	}

	hasTasks := strings.TrimSpace(taskSummary) != ""
	codingScore := 50
	if !hasTasks {
		codingScore = 0
	}

	overall := mcqScore
	if hasTasks {
		overall = (mcqScore + codingScore) / 2
	}

	return client.GeminiAnalysis{
		OverallScore:      overall,
		AlgorithmScore:    mcqScore,
		ArchitectureScore: mcqScore,
		CodingScore:       codingScore,
		SoftSkillsScore:   max(mcqScore-5, 40),
		Comments:          "Автоматическая оценка на основе результатов теста. AI-анализ временно недоступен.",
		Strengths:         "Ответы на вопросы с вариантами зафиксированы.",
		Weaknesses:          "Требуется ручная проверка практических задач.",
		Recommendations:     "Повторите темы с ошибками и пересдайте тренировку.",
	}
}

func scoresFromNotes(notes string) (usecase.AnalysisScoresDTO, error) {
	var metadata analysisMetadata
	if err := json.Unmarshal([]byte(notes), &metadata); err != nil {
		return usecase.AnalysisScoresDTO{}, err
	}

	return usecase.AnalysisScoresDTO{
		OverallScore:      metadata.OverallScore,
		AlgorithmScore:    metadata.AlgorithmScore,
		ArchitectureScore: metadata.ArchitectureScore,
		CodingScore:       metadata.CodingScore,
		SoftSkillsScore:   metadata.SoftSkillsScore,
		Comments:          metadata.Comments,
	}, nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
