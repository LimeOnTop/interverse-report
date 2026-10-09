package service

import (
	"testing"

	"github.com/LimeOnTop/interverse-report/internal/usecase"
)

func TestIsSelectedOptionCorrectMatchesByTextAfterShuffle(t *testing.T) {
	bank := []usecase.QuestionOption{
		{Text: "right", IsCorrect: true},
		{Text: "wrong 1"},
		{Text: "wrong 2"},
	}
	session := []string{"wrong 2", "right", "wrong 1"}

	if !isSelectedOptionCorrect(session, bank, 1) {
		t.Fatal("choosing the shuffled correct option must be scored as correct")
	}
	if isSelectedOptionCorrect(session, bank, 0) {
		t.Fatal("choosing the first option must not be scored as correct when it is wrong")
	}
	if isSelectedOptionCorrect(session, bank, 5) {
		t.Fatal("out-of-range choice must not be correct")
	}
}

func TestIsSelectedOptionCorrectLegacySessionWithoutOptions(t *testing.T) {
	bank := []usecase.QuestionOption{{Text: "right", IsCorrect: true}, {Text: "wrong"}}
	if !isSelectedOptionCorrect(nil, bank, 0) || isSelectedOptionCorrect(nil, bank, 1) {
		t.Fatal("legacy sessions fall back to bank order")
	}
}
