package service

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/LimeOnTop/interverse-report/internal/usecase"
)

// analysisModeBasic marks reports built without the LLM (Basic plan).
const analysisModeBasic = "basic"

var (
	codeFence = regexp.MustCompile("(?s)```[a-zA-Z0-9_+-]*\\n?(.*?)```")
	codeToken = regexp.MustCompile(`[\p{L}_][\p{L}\p{N}_]+`)
)

// basicAnalysis scores a session without the LLM: theory from MCQ accuracy,
// tasks by how much of the reference solution the answer covers.
func basicAnalysis(
	interview usecase.InterviewSummary,
	reviews []answerReviewItem,
	mcqCorrect, mcqTotal int,
) usecase.GeminiAnalysis {
	taskTotal, taskSum, solved := 0, 0, 0
	for _, item := range reviews {
		if item.ItemType != "task" {
			continue
		}
		taskTotal++
		score := taskCoverageScore(item.UserAnswer, item.CorrectAnswer)
		taskSum += score
		if score >= passScoreThreshold {
			solved++
		}
	}

	codingScore := 0
	if taskTotal > 0 {
		codingScore = int(float64(taskSum)/float64(taskTotal) + 0.5)
	}

	summary := basicSummary(interview.Level, mcqCorrect, mcqTotal, solved, taskTotal, percent(mcqCorrect, mcqTotal), codingScore)
	return usecase.GeminiAnalysis{
		AlgorithmScore: percent(mcqCorrect, mcqTotal),
		CodingScore:    codingScore,
		Comments:       summary,
		SummaryPublic:  summary,
	}
}

// taskCoverageScore is the share of identifiers from the reference code that
// appear in the answer; 70% coverage already counts as a full solution.
func taskCoverageScore(answer, reference string) int {
	answer = strings.TrimSpace(answer)
	if answer == "" || answer == "Нет ответа" {
		return 0
	}

	if blocks := codeFence.FindAllStringSubmatch(reference, -1); len(blocks) > 0 {
		parts := make([]string, 0, len(blocks))
		for _, block := range blocks {
			parts = append(parts, block[1])
		}
		reference = strings.Join(parts, "\n")
	}

	expected := tokenSet(reference)
	if len(expected) == 0 {
		return 50
	}
	got := tokenSet(answer)
	matched := 0
	for token := range expected {
		if _, ok := got[token]; ok {
			matched++
		}
	}

	score := int(float64(matched)/float64(len(expected))/0.7*100 + 0.5)
	if score > 100 {
		score = 100
	}
	return score
}

func tokenSet(text string) map[string]struct{} {
	set := make(map[string]struct{})
	for _, token := range codeToken.FindAllString(strings.ToLower(text), -1) {
		set[token] = struct{}{}
	}
	return set
}

func basicSummary(_ string, mcqCorrect, mcqTotal, solved, taskTotal, theory, coding int) string {
	summary := personalReportSummary(theory, coding, mcqTotal > 0, taskTotal > 0)
	if mcqTotal > 0 {
		summary += fmt.Sprintf(" Верных ответов: %d из %d.", mcqCorrect, mcqTotal)
	}
	if taskTotal > 0 {
		summary += fmt.Sprintf(" Решённых задач: %d из %d.", solved, taskTotal)
	}
	return summary
}

func personalReportSummary(theory, coding int, hasTheory, hasTasks bool) string {
	switch {
	case hasTheory && hasTasks:
		switch {
		case theory >= 80 && coding >= 80:
			return "Ты уверенно справляешься с теорией и практикой. Попробуй тренировку уровнем выше или выбери новую технологию."
		case theory >= 60 && coding >= 60:
			return "У тебя хорошая база в теории и практике. Разбери ошибки этой тренировки и повтори темы, в которых ещё есть сомнения."
		case theory >= 60:
			return "С теорией у тебя получается лучше, чем с практикой. Запусти тренировку в режиме «Много задач» (4–6) и разбирай решения шаг за шагом — это поможет закрепить знания в коде."
		case coding >= 60:
			return "С практическими задачами ты справляешься увереннее, чем с теорией. Выбери «Только теория» и повтори вопросы с ошибками, чтобы лучше объяснять свои решения."
		default:
			return "Пока есть трудности и в теории, и в практике. Начни с разбора ошибок, затем выбери тренировку с 1–3 задачами и постепенно увеличивай нагрузку."
		}
	case hasTheory:
		if theory >= 60 {
			return "С теоретической частью у тебя хороший результат. В этой тренировке не было задач, поэтому практику мы пока не оценивали. Добавь 1–3 задачи в следующую тренировку, чтобы проверить, как применяешь знания в коде."
		}
		return "Теорию стоит закрепить. Разбери вопросы с ошибками и попробуй ещё одну тренировку в режиме «Только теория». Практических задач здесь не было — эта часть ещё не оценивалась."
	case hasTasks:
		if coding >= 60 {
			return "Ты хорошо справляешься с практическими задачами. Пройди тренировку с теоретическими вопросами, чтобы проверить и эту часть подготовки."
		}
		return "Практические задачи пока вызывают трудности. Начни с 1–3 задач, сверяй решения с разбором и повторяй их без подсказок."
	default:
		return "В этой тренировке нет ответов для оценки. Запусти новую тренировку и ответь на вопросы, чтобы получить разбор."
	}
}

func capitalize(text string) string {
	runes := []rune(text)
	if len(runes) == 0 {
		return text
	}
	return strings.ToUpper(string(runes[0])) + string(runes[1:])
}
