package service

import (
	"context"
	"github.com/LimeOnTop/interverse-report/internal/entity"
	"testing"
)

func TestManualReportRequiresInterviewOwner(t *testing.T) {
	repo := &reportStub{}
	s := NewReportService(repo, &interviewStub{owner: 2})
	if _, err := s.Create(context.Background(), entity.Report{UserID: 1, InterviewID: "foreign"}); err == nil {
		t.Fatal("foreign interview accepted")
	}
	if _, err := s.Create(context.Background(), entity.Report{UserID: 2, InterviewID: "owned"}); err != nil {
		t.Fatal("owner rejected", err)
	}
	s = NewReportService(repo)
	if _, err := s.Create(context.Background(), entity.Report{UserID: 1, InterviewID: "any"}); err == nil {
		t.Fatal("missing authorization accepted")
	}
}
