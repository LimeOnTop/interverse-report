package service

import (
	"strings"

	"github.com/LimeOnTop/interverse-report/internal/client"
)

const passScoreThreshold = 60

type sectionPassFlags struct {
	Algorithm    bool `json:"algorithm_passed"`
	Architecture bool `json:"architecture_passed"`
	Coding       bool `json:"coding_passed"`
	SoftSkills   bool `json:"soft_skills_passed"`
}

type finalizedScores struct {
	Analysis client.GeminiAnalysis
	Passed   sectionPassFlags
}

func countAnsweredTasks(tasks []client.SessionItem, answers map[string]AnswerInput) int {
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
	analysis client.GeminiAnalysis,
	mcqTotal int,
	answeredTasks int,
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

	passedScores := make([]int, 0, 2)
	if theoryPassed {
		passedScores = append(passedScores, theoryScore)
	}
	if codingPassed {
		passedScores = append(passedScores, codingScore)
	}

	overallScore := 0
	if len(passedScores) > 0 {
		sum := 0
		for _, score := range passedScores {
			sum += score
		}
		overallScore = sum / len(passedScores)
	}

	return finalizedScores{
		Analysis: client.GeminiAnalysis{
			OverallScore:      overallScore,
			AlgorithmScore:    theoryScore,
			ArchitectureScore: architectureScore,
			CodingScore:       codingScore,
			SoftSkillsScore:   softSkillsScore,
			Comments:          analysis.Comments,
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
