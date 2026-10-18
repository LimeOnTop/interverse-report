package controller

import (
	"context"
	pb "github.com/LimeOnTop/interverse-contracts/report/gen"
	"github.com/LimeOnTop/interverse-report/internal/entity"
	"github.com/LimeOnTop/interverse-report/internal/usecase"
	"testing"
)

type createReportStub struct {
	usecase.Report
	calls int
}

func (s *createReportStub) Create(_ context.Context, r entity.Report) (usecase.ReportDTO, error) {
	s.calls++
	return usecase.ReportDTO{}, nil
}
func TestCreateReportRejectsInvalidUserBeforeUsecase(t *testing.T) {
	for _, id := range []string{"abc", "", "0", "-1", "9223372036854775808"} {
		t.Run(id, func(t *testing.T) {
			s := &createReportStub{}
			c := NewReportController(s, nil)
			resp, err := c.CreateReport(context.Background(), &pb.CreateReportRequest{UserId: id})
			if err != nil || resp.GetResponse().GetSuccess() || s.calls != 0 {
				t.Fatalf("invalid user reached usecase: resp=%v err=%v calls=%d", resp, err, s.calls)
			}
		})
	}
}

type answerCapture struct{ answers []usecase.AnswerInput }

func (s *answerCapture) Generate(_ context.Context, _ string, _ int64, a []usecase.AnswerInput, _ bool) (usecase.ReportDTO, usecase.AnalysisScoresDTO, error) {
	s.answers = a
	return usecase.ReportDTO{}, usecase.AnalysisScoresDTO{}, nil
}
func TestGeneratePreservesSkippedAnswer(t *testing.T) {
	zero := int32(0)
	spy := &answerCapture{}
	c := NewReportController(nil, spy)
	_, err := c.GenerateReport(context.Background(), &pb.GenerateReportRequest{UserId: "1", Answers: []*pb.SessionAnswerInput{{ItemType: "question", StepId: "skip"}, {ItemType: "question", StepId: "first", SelectedOptionIndex: &zero}}})
	if err != nil {
		t.Fatal(err)
	}
	if spy.answers[0].SelectedOptionIndex != nil {
		t.Fatal("skip converted to first option")
	}
	if spy.answers[1].SelectedOptionIndex == nil || *spy.answers[1].SelectedOptionIndex != 0 {
		t.Fatal("first option dropped")
	}
}
