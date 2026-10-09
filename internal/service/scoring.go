package service

import (
	"strings"
	"time"

	"github.com/LimeOnTop/interverse-report/internal/usecase"
)

const passScoreThreshold = 60

// llmBudget keeps report generation under the 120s gateway/nginx timeout.
const llmBudget = 95 * time.Second

type sectionPassFlags struct {
	Algorithm    bool `json:"algorithm_passed"`
	Architecture bool `json:"architecture_passed"`
	Coding       bool `json:"coding_passed"`
	SoftSkills   bool `json:"soft_skills_passed"`
}

type finalizedScores struct {
	Analysis usecase.GeminiAnalysis
	Passed   sectionPassFlags
}

func countAnsweredTasks(tasks []usecase.SessionItem, answers map[string]usecase.AnswerInput) int {
	count := 0
	for _, item := range tasks {
		answer, ok := answers[item.ID]
		if ok && strings.TrimSpace(answer.TaskAnswer) != "" {
			count++
		}
	}
	return count
}

func finalizeAnalysisScores(
	analysis usecase.GeminiAnalysis,
	mcqTotal int,
	answeredTasks int,
	taskTotal int,
) finalizedScores {
	hasMCQ := mcqTotal > 0
	hasTasks := answeredTasks > 0

	// Theory score comes only from theoretical MCQ questions.
	theoryScore := 0
	theoryPassed := false
	if hasMCQ {
		theoryScore = analysis.AlgorithmScore
		theoryPassed = theoryScore >= passScoreThreshold
	}

	// Architecture / soft skills stages are not live yet.
	architectureScore := 0
	architecturePassed := false
	softSkillsScore := 0
	softSkillsPassed := false

	// Coding score comes only from practical tasks.
	codingScore := 0
	codingPassed := false
	if hasTasks {
		codingScore = analysis.CodingScore
		codingPassed = codingScore >= passScoreThreshold
	}

	// Overall is the real result: the mean of every section the session had,
	// failed ones included (a 21% theory must not show up as 0% overall).
	sectionScores := make([]int, 0, 2)
	if hasMCQ {
		sectionScores = append(sectionScores, theoryScore)
	}
	if taskTotal > 0 {
		sectionScores = append(sectionScores, codingScore)
	}

	overallScore := 0
	if len(sectionScores) > 0 {
		sum := 0
		for _, score := range sectionScores {
			sum += score
		}
		overallScore = int(float64(sum)/float64(len(sectionScores)) + 0.5)
	}

	return finalizedScores{
		Analysis: usecase.GeminiAnalysis{
			OverallScore:      overallScore,
			AlgorithmScore:    theoryScore,
			ArchitectureScore: architectureScore,
			CodingScore:       codingScore,
			SoftSkillsScore:   softSkillsScore,
			Comments:          analysis.Comments,
			SummaryPublic:     analysis.SummaryPublic,
			Strengths:         analysis.Strengths,
			Weaknesses:        analysis.Weaknesses,
			Recommendations:   analysis.Recommendations,
		},
		Passed: sectionPassFlags{
			Algorithm:    theoryPassed,
			Architecture: architecturePassed,
			Coding:       codingPassed,
			SoftSkills:   softSkillsPassed,
		},
	}
}
