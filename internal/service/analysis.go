package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/LimeOnTop/interverse-report/internal/client"
	"github.com/LimeOnTop/interverse-report/internal/entity"
	"github.com/LimeOnTop/interverse-report/internal/usecase"
)

type analysisMetadata struct {
	OverallScore            int                `json:"overall_score"`
	AlgorithmScore          int                `json:"algorithm_score"`
	ArchitectureScore       int                `json:"architecture_score"`
	CodingScore             int                `json:"coding_score"`
	SoftSkillsScore         int                `json:"soft_skills_score"`
	AlgorithmPassed         bool               `json:"algorithm_passed"`
	ArchitecturePassed      bool               `json:"architecture_passed"`
	CodingPassed            bool               `json:"coding_passed"`
	SoftSkillsPassed        bool               `json:"soft_skills_passed"`
	Comments                string             `json:"comments"`
	InterviewTitle          string             `json:"interview_title"`
	InterviewLevel          string             `json:"interview_level"`
	InterviewSpecialization string             `json:"interview_specialization"`
	InterviewScheduledAt    string             `json:"interview_scheduled_at"`
	MCQCorrect              int                `json:"mcq_correct"`
	MCQTotal                int                `json:"mcq_total"`
	AnswerReviews           []answerReviewItem `json:"answer_reviews,omitempty"`
}

// answerReviewItem is persisted so the report UI can show user vs correct/reference answers.
type answerReviewItem struct {
	StepID        string   `json:"step_id"`
	QuestionID    string   `json:"question_id"`
	ItemType      string   `json:"item_type"`
	SortOrder     int      `json:"sort_order"`
	Label         string   `json:"label"`
	Prompt        string   `json:"prompt"`
	Technology    string   `json:"technology,omitempty"`
	Options       []string `json:"options,omitempty"`
	SelectedIndex *int     `json:"selected_index,omitempty"`
	CorrectIndex  *int     `json:"correct_index,omitempty"`
	UserAnswer    string   `json:"user_answer"`
	CorrectAnswer string   `json:"correct_answer"`
	IsCorrect     *bool    `json:"is_correct,omitempty"`
}

type AnswerInput struct {
	StepID              string
	QuestionID          string
	ItemType            string
	SelectedOptionIndex *int32
	TaskAnswer          string
}

type AnalysisService struct {
	repository      usecase.ReportRepository
	interviewClient *client.InterviewClient
	questionClient  *client.QuestionClient
	geminiClient    client.Analyzer
	answerCache     *AnswerCache
}

func NewAnalysisService(
	repository usecase.ReportRepository,
	interviewClient *client.InterviewClient,
	questionClient *client.QuestionClient,
	geminiClient client.Analyzer,
	answerCache *AnswerCache,
) *AnalysisService {
	return &AnalysisService{
		repository:      repository,
		interviewClient: interviewClient,
		questionClient:  questionClient,
		geminiClient:    geminiClient,
		answerCache:     answerCache,
	}
}

func (s *AnalysisService) Generate(
	ctx context.Context,
	interviewID string,
	userID int64,
	answers []AnswerInput,
) (usecase.ReportDTO, usecase.AnalysisScoresDTO, error) {
	answers, err := s.resolveAnswers(ctx, interviewID, answers)
	if err != nil {
		return usecase.ReportDTO{}, usecase.AnalysisScoresDTO{}, err
	}

	existing, err := s.repository.GetByInterviewID(ctx, interviewID)
	if err == nil {
		scores, parseErr := scoresFromNotes(existing.Notes)
		if parseErr == nil && !isFallbackReport(scores.Comments) {
			return toDTO(existing), scores, nil
		}
		if deleteErr := s.repository.Delete(ctx, existing.ID); deleteErr != nil {
			return usecase.ReportDTO{}, usecase.AnalysisScoresDTO{}, fmt.Errorf("replace fallback report: %w", deleteErr)
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
	answeredTasks := countAnsweredTasks(tasks, answerByStep)
	answerReviews := s.buildAnswerReviews(ctx, questions, tasks, answerByStep)

	prompt := buildAnalysisPrompt(interview, mcqSummary, mcqCorrect, mcqTotal, taskSummary, answeredTasks)
	analysis, err := s.geminiClient.Analyze(ctx, prompt)
	if err != nil {
		log.Printf("gemini analysis failed: %v", err)
		analysis = fallbackAnalysis(mcqCorrect, mcqTotal, taskSummary, answeredTasks)
	}

	if mcqTotal > 0 && analysis.AlgorithmScore == 0 {
		analysis.AlgorithmScore = int(float64(mcqCorrect) / float64(mcqTotal) * 100)
	}

	finalized := finalizeAnalysisScores(analysis, mcqTotal, answeredTasks)
	analysis = finalized.Analysis

	metadata := analysisMetadata{
		OverallScore:            analysis.OverallScore,
		AlgorithmScore:          analysis.AlgorithmScore,
		ArchitectureScore:       analysis.ArchitectureScore,
		CodingScore:             analysis.CodingScore,
		SoftSkillsScore:         analysis.SoftSkillsScore,
		AlgorithmPassed:         finalized.Passed.Algorithm,
		ArchitecturePassed:      finalized.Passed.Architecture,
		CodingPassed:            finalized.Passed.Coding,
		SoftSkillsPassed:        finalized.Passed.SoftSkills,
		Comments:                analysis.Comments,
		InterviewTitle:          interview.Title,
		InterviewLevel:          interview.Level,
		InterviewSpecialization: interview.Specialization,
		InterviewScheduledAt:    interview.ScheduledAt,
		MCQCorrect:              mcqCorrect,
		MCQTotal:                mcqTotal,
		AnswerReviews:           answerReviews,
	}

	notesJSON, err := json.Marshal(metadata)
	if err != nil {
		return usecase.ReportDTO{}, usecase.AnalysisScoresDTO{}, fmt.Errorf("marshal analysis metadata: %w", err)
	}

	created, err := s.repository.Create(ctx, entity.Report{
		InterviewID:         interviewID,
		UserID:              userID,
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
		userAnswer := ""
		if ok {
			userAnswer = strings.TrimSpace(answer.TaskAnswer)
		}
		if userAnswer == "" {
			userAnswer = "(no answer)"
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
			"Task %d:\nQuestion: %s\nTechnology: %s\nCategory: %s\nReference answer: %s\nUser answer:\n%s\n\n",
			idx+1,
			item.Text,
			item.Technology,
			item.Category,
			referenceAnswer,
			userAnswer,
		))
	}

	return builder.String()
}

func (s *AnalysisService) buildAnswerReviews(
	ctx context.Context,
	questions, tasks []client.SessionItem,
	answers map[string]AnswerInput,
) []answerReviewItem {
	reviews := make([]answerReviewItem, 0, len(questions)+len(tasks))

	for index, item := range questions {
		answer, hasAnswer := answers[item.ID]
		questionID := item.QuestionID
		if questionID == "" && hasAnswer {
			questionID = answer.QuestionID
		}

		options := append([]string(nil), item.Options...)
		var selectedIndex *int
		var correctIndex *int
		userAnswer := ""
		correctAnswer := ""
		var isCorrect *bool

		if hasAnswer && answer.SelectedOptionIndex != nil {
			selected := int(*answer.SelectedOptionIndex)
			selectedIndex = &selected
			if selected >= 0 && selected < len(options) {
				userAnswer = options[selected]
			} else {
				userAnswer = fmt.Sprintf("Вариант #%d", selected+1)
			}
		}

		if questionID != "" {
			details, err := s.questionClient.GetByID(ctx, questionID)
			if err == nil {
				if len(options) == 0 && len(details.Options) > 0 {
					options = make([]string, 0, len(details.Options))
					for _, option := range details.Options {
						options = append(options, option.Text)
					}
				}
				for idx, option := range details.Options {
					if !option.IsCorrect {
						continue
					}
					correct := idx
					correctIndex = &correct
					correctAnswer = option.Text
					break
				}
				if correctAnswer == "" {
					correctAnswer = details.Answer
				}
			}
		}

		if selectedIndex != nil && correctIndex != nil {
			ok := *selectedIndex == *correctIndex
			isCorrect = &ok
		} else if selectedIndex != nil && correctAnswer != "" && userAnswer != "" {
			ok := strings.TrimSpace(userAnswer) == strings.TrimSpace(correctAnswer)
			isCorrect = &ok
		}

		if userAnswer == "" {
			userAnswer = "Нет ответа"
		}
		if correctAnswer == "" {
			correctAnswer = "Эталон недоступен"
		}

		reviews = append(reviews, answerReviewItem{
			StepID:        item.ID,
			QuestionID:    questionID,
			ItemType:      "question",
			SortOrder:     int(item.SortOrder),
			Label:         fmt.Sprintf("Вопрос %d", index+1),
			Prompt:        item.Text,
			Technology:    item.Technology,
			Options:       options,
			SelectedIndex: selectedIndex,
			CorrectIndex:  correctIndex,
			UserAnswer:    userAnswer,
			CorrectAnswer: correctAnswer,
			IsCorrect:     isCorrect,
		})
	}

	for index, item := range tasks {
		answer, hasAnswer := answers[item.ID]
		questionID := item.QuestionID
		if questionID == "" && hasAnswer {
			questionID = answer.QuestionID
		}

		userAnswer := ""
		if hasAnswer {
			userAnswer = strings.TrimSpace(answer.TaskAnswer)
		}
		correctAnswer := ""
		if questionID != "" {
			details, err := s.questionClient.GetByID(ctx, questionID)
			if err == nil {
				correctAnswer = strings.TrimSpace(details.Answer)
			}
		}

		if userAnswer == "" {
			userAnswer = "Нет ответа"
		}
		if correctAnswer == "" {
			correctAnswer = "Эталонное решение недоступно"
		}

		reviews = append(reviews, answerReviewItem{
			StepID:        item.ID,
			QuestionID:    questionID,
			ItemType:      "task",
			SortOrder:     int(item.SortOrder),
			Label:         fmt.Sprintf("Задача %d", index+1),
			Prompt:        item.Text,
			Technology:    item.Technology,
			UserAnswer:    userAnswer,
			CorrectAnswer: correctAnswer,
		})
	}

	return reviews
}

func buildAnalysisPrompt(
	interview client.InterviewSummary,
	mcqSummary string,
	mcqCorrect, mcqTotal int,
	taskSummary string,
	answeredTasks int,
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

Practical tasks (answered %d):
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
- Pass threshold for each section is %d points
- algorithm_score: THEORY score from theoretical MCQ only (map MCQ accuracy here; ignore tasks)
- architecture_score: always 0 — architecture stage is not available yet
- coding_score: practical task solutions only (ignore MCQ)
- soft_skills_score: always 0 — soft skills stage is not available yet
- overall_score will be recalculated server-side from passed theory/coding only; still provide your best estimate
- Sections below %d or not attempted are treated as failed and excluded from overall score
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
		answeredTasks,
		taskSummary,
		passScoreThreshold,
		passScoreThreshold,
	)
}

func (s *AnalysisService) resolveAnswers(
	ctx context.Context,
	interviewID string,
	answers []AnswerInput,
) ([]AnswerInput, error) {
	if len(answers) > 0 {
		if s.answerCache != nil {
			if err := s.answerCache.Save(ctx, interviewID, answers); err != nil {
				return nil, fmt.Errorf("cache interview answers: %w", err)
			}
		}
		return answers, nil
	}

	if s.answerCache == nil {
		return nil, fmt.Errorf("generate report: answers are required")
	}

	cached, err := s.answerCache.Load(ctx, interviewID)
	if err != nil {
		return nil, fmt.Errorf("generate report: answers not found or expired")
	}

	if len(cached) == 0 {
		return nil, fmt.Errorf("generate report: answers not found or expired")
	}

	return cached, nil
}

func isFallbackReport(comments string) bool {
	return strings.Contains(comments, "AI-анализ временно недоступен")
}

func fallbackAnalysis(mcqCorrect, mcqTotal int, taskSummary string, answeredTasks int) client.GeminiAnalysis {
	mcqScore := 0
	if mcqTotal > 0 {
		mcqScore = int(float64(mcqCorrect) / float64(mcqTotal) * 100)
	}

	codingScore := 0
	if answeredTasks > 0 {
		codingScore = 50
	}

	raw := client.GeminiAnalysis{
		OverallScore:      0,
		AlgorithmScore:    mcqScore,
		ArchitectureScore: 0,
		CodingScore:       codingScore,
		SoftSkillsScore:   0,
		Comments:          "Автоматическая оценка на основе результатов теста. AI-анализ временно недоступен.",
		Strengths:         "Ответы на теоретические вопросы зафиксированы.",
		Weaknesses:        "Требуется ручная проверка практических задач.",
		Recommendations:   "Повторите темы с ошибками и пересдайте тренировку.",
	}
	_ = taskSummary

	return finalizeAnalysisScores(raw, mcqTotal, answeredTasks).Analysis
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
