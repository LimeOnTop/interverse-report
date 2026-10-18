package service

import (
	"context"
	"fmt"
	"github.com/LimeOnTop/interverse-report/internal/usecase"
	"testing"
)

type firstCorrectBank struct{}

func (firstCorrectBank) GetByID(context.Context, string) (usecase.QuestionDetails, error) {
	return usecase.QuestionDetails{Options: []usecase.QuestionOption{{Text: "right", IsCorrect: true}, {Text: "wrong"}}}, nil
}
func TestNineSkipsAndOneWrongAreZero(t *testing.T) {
	s := &AnalysisService{questionClient: firstCorrectBank{}}
	items := []usecase.SessionItem{}
	answers := map[string]usecase.AnswerInput{}
	for i := 0; i < 10; i++ {
		id := fmt.Sprint(i)
		items = append(items, usecase.SessionItem{ID: id, QuestionID: id, ItemType: "question", Options: []string{"right", "wrong"}})
		answers[id] = usecase.AnswerInput{StepID: id}
	}
	wrong := int32(1)
	answers["9"] = usecase.AnswerInput{StepID: "9", SelectedOptionIndex: &wrong}
	_, correct, total := s.scoreMCQ(context.Background(), items, answers)
	if correct != 0 || total != 10 {
		t.Fatalf("score %d/%d, want 0/10", correct, total)
	}
	reviews := s.buildAnswerReviews(context.Background(), items, nil, answers)
	for i := 0; i < 9; i++ {
		r := reviews[i]
		if r.UserAnswer != "Нет ответа" || r.SelectedIndex != nil || r.IsCorrect == nil || *r.IsCorrect {
			t.Fatalf("skip is misrepresented: %+v", r)
		}
	}
	zero := int32(0)
	answers["0"] = usecase.AnswerInput{StepID: "0", SelectedOptionIndex: &zero}
	_, correct, _ = s.scoreMCQ(context.Background(), items, answers)
	if correct != 1 {
		t.Fatal("explicit first option must still count")
	}
}
