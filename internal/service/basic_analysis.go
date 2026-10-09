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

func basicSummary(level string, mcqCorrect, mcqTotal, solved, taskTotal, theory, coding int) string {
	sections := make([]int, 0, 2)
	if mcqTotal > 0 {
		sections = append(sections, theory)
	}
	if taskTotal > 0 {
		sections = append(sections, coding)
	}
	overall := 0
	for _, score := range sections {
		overall += score
	}
	if len(sections) > 0 {
		overall /= len(sections)
	}

	levelName := strings.TrimSpace(level)
	if levelName == "" {
		levelName = "выбранного"
	} else {
		levelName = capitalize(levelName)
	}

	var verdict string
	switch {
	case overall >= 80:
		verdict = fmt.Sprintf("Уровень %s подтверждён: результат уверенный.", levelName)
	case overall >= passScoreThreshold:
		verdict = fmt.Sprintf("Уровень %s почти достигнут: база есть, но остались пробелы.", levelName)
	default:
		verdict = fmt.Sprintf("Уровень %s пока не достигнут: стоит подтянуть теорию и практику.", levelName)
	}

	details := make([]string, 0, 2)
	if mcqTotal > 0 {
		details = append(details, fmt.Sprintf("верных ответов %d из %d", mcqCorrect, mcqTotal))
	}
	if taskTotal > 0 {
		details = append(details, fmt.Sprintf("задач решено %d из %d", solved, taskTotal))
	}
	if len(details) == 0 {
		return verdict
	}
	return verdict + " " + capitalize(strings.Join(details, ", ")) + "."
}

func capitalize(text string) string {
	runes := []rune(text)
	if len(runes) == 0 {
		return text
	}
	return strings.ToUpper(string(runes[0])) + string(runes[1:])
}
