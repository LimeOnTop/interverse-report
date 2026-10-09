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
