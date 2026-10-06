package service

import (
	"strings"
	"testing"

	"github.com/LimeOnTop/interverse-report/internal/client"
)

func boolPtr(v bool) *bool { return &v }

func TestOverallScoreIsRealMeanOfSections(t *testing.T) {
	got := finalizeAnalysisScores(client.GeminiAnalysis{AlgorithmScore: 21, CodingScore: 0}, 19, 0, 1)
	if got.Analysis.OverallScore != 11 {
		t.Fatalf("overall = %d, want 11 (mean of 21 and 0)", got.Analysis.OverallScore)
	}
	if got.Passed.Algorithm || got.Passed.Coding {
		t.Fatal("sections below threshold must not pass")
	}

	onlyTheory := finalizeAnalysisScores(client.GeminiAnalysis{AlgorithmScore: 21}, 19, 0, 0)
	if onlyTheory.Analysis.OverallScore != 21 {
		t.Fatalf("overall without tasks = %d, want 21", onlyTheory.Analysis.OverallScore)
	}
}

func TestBuildWeakPoints(t *testing.T) {
	reviews := []answerReviewItem{
		{StepID: "q1", ItemType: "question", Prompt: "Что такое замыкание?", IsCorrect: boolPtr(true)},
		{StepID: "q2", ItemType: "question", Prompt: "Что такое горутина?", Technology: "Go", IsCorrect: boolPtr(false)},
		{StepID: "q3", ItemType: "question", Prompt: "Без ответа", UserAnswer: "Нет ответа"},
		{StepID: "t1", ItemType: "task", Prompt: "Решено", UserAnswer: "func a() {}"},
		{StepID: "t2", ItemType: "task", Prompt: "Не решено", UserAnswer: "Нет ответа"},
	}
	candidates := weakPointCandidates(reviews)
	if len(candidates) != 4 {
		t.Fatalf("candidates = %d, want 4 (correct question excluded)", len(candidates))
	}

	verdicts := map[string]weakPointVerdict{
		"q2": {ID: "q2", Explanation: "Потому что", Topic: "горутины Go"},
		"t1": {ID: "t1", IsWeak: boolPtr(false)},
		"t2": {ID: "t2", IsWeak: boolPtr(false)},
	}
	points := buildWeakPoints(candidates, verdicts, false)
	ids := []string{}
	for _, p := range points {
		ids = append(ids, p.StepID)
		if !strings.HasPrefix(p.SourceURL, "https://habr.com/ru/search/?q=") {
			t.Fatalf("source url %q is not a Habr search", p.SourceURL)
		}
	}
	if strings.Join(ids, ",") != "q2,q3,t2" {
		t.Fatalf("weak points = %v, want q2,q3,t2 (solved task dropped, unanswered task kept)", ids)
	}
	if points[0].Explanation != "Потому что" || !strings.Contains(points[0].SourceURL, "%D0%B3%D0%BE%D1%80%D1%83%D1%82%D0%B8%D0%BD%D1%8B") {
		t.Fatalf("unexpected first point: %+v", points[0])
	}
}
