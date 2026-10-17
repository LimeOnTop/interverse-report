package service

import (
	"github.com/LimeOnTop/interverse-report/internal/usecase"
	"strings"
	"testing"
)

func TestCoachingSeparatesAbsentAndWeakPractice(t *testing.T) {
	for _, tc := range []struct {
		name                string
		theory, coding      int
		hasTheory, hasTasks bool
		want                string
	}{
		{"weak practice", 90, 20, true, true, "«Много задач» (4–6)"},
		{"weak theory", 20, 90, true, true, "«Только теория»"},
		{"absent practice", 90, 0, true, false, "практику мы пока не оценивали"},
		{"absent theory", 0, 90, false, true, "теоретическими вопросами"},
		{"strong both", 90, 90, true, true, "уровнем выше"},
		{"weak both", 20, 20, true, true, "1–3 задачами"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := personalReportSummary(tc.theory, tc.coding, tc.hasTheory, tc.hasTasks)
			if !strings.Contains(got, tc.want) || strings.Contains(got, "Кандидат") {
				t.Fatalf("summary: %s", got)
			}
		})
	}
}

func TestDeepSeekPromptUsesCoachingAndTaskPresence(t *testing.T) {
	got := buildAnalysisPrompt(usecase.InterviewSummary{Level: "senior"}, "", 2, 3, "", 0, 4)
	for _, want := range []string{usecase.ReportCoachingInstructions, "Practical tasks (answered 0 of 4)", "summary_public", "submitted answers"} {
		if !strings.Contains(got, want) {
			t.Fatalf("prompt missing %q", want)
		}
	}
	if weakPointsSystemPrompt != usecase.ReportCoachingInstructions {
		t.Fatal("weak point prompt must share coaching policy")
	}
}

func TestFallbackDoesNotJudgeAbsentTasks(t *testing.T) {
	got := fallbackAnalysis(3, 4, "", 0, 0)
	if !strings.Contains(got.Comments, "AI-анализ временно недоступен") || !strings.Contains(got.Comments, "не оценивали") || got.Weaknesses != "" {
		t.Fatalf("fallback: %+v", got)
	}
}
