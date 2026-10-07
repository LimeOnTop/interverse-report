package service

import (
	"strings"
	"testing"

	"github.com/LimeOnTop/interverse-report/internal/client"
)

func TestTaskCoverageScoreUsesReferenceCode(t *testing.T) {
	reference := "```go\nfunc Sum(values []int) int {\n\ttotal := 0\n\tfor _, v := range values {\n\t\ttotal += v\n\t}\n\treturn total\n}\n```\nКлючевое: один проход по срезу."

	if got := taskCoverageScore("Нет ответа", reference); got != 0 {
		t.Fatalf("unanswered task scored %d, want 0", got)
	}
	full := "func Sum(values []int) int {\n total := 0\n for _, v := range values { total += v }\n return total\n}"
	if got := taskCoverageScore(full, reference); got != 100 {
		t.Fatalf("matching solution scored %d, want 100", got)
	}
	if got := taskCoverageScore("не знаю", reference); got >= passScoreThreshold {
		t.Fatalf("unrelated answer scored %d, want below pass threshold", got)
	}
}

func TestBasicAnalysisHasNoTopics(t *testing.T) {
	correct, wrong := true, false
	reviews := []answerReviewItem{
		{ItemType: "question", IsCorrect: &correct},
		{ItemType: "question", IsCorrect: &wrong, Prompt: "Что такое горутина?"},
		{ItemType: "task", UserAnswer: "Нет ответа", CorrectAnswer: "return a + b", Prompt: "Сложите числа"},
	}
	analysis := basicAnalysis(client.InterviewSummary{Level: "junior"}, reviews, 1, 2)

	if analysis.AlgorithmScore != 50 || analysis.CodingScore != 0 {
		t.Fatalf("scores = %d/%d, want 50/0", analysis.AlgorithmScore, analysis.CodingScore)
	}
	if analysis.SummaryPublic != analysis.Comments || !strings.Contains(analysis.Comments, "Junior") {
		t.Fatalf("unexpected summary: %q", analysis.Comments)
	}
	if strings.Contains(analysis.Comments, "горутин") || strings.Contains(analysis.Comments, "AI-анализ") {
		t.Fatalf("basic summary must not name topics or look like a fallback: %q", analysis.Comments)
	}
	t.Log(analysis.Comments)
}
