package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/LimeOnTop/interverse-report/internal/entity"
	"github.com/LimeOnTop/interverse-report/internal/usecase"
)

type interviewStub struct {
	owner     int64
	err       error
	completed int
}

func (s *interviewStub) GetSessionContent(context.Context, string, int64) (usecase.InterviewSummary, []usecase.SessionItem, []usecase.SessionItem, error) {
	return usecase.InterviewSummary{UserID: s.owner}, nil, nil, s.err
}
func (s *interviewStub) CompleteInterview(context.Context, usecase.InterviewSummary) error {
	s.completed++
	return nil
}

type reportStub struct {
	usecase.ReportRepository
	existing                         entity.Report
	lookupErr, updateErr             error
	reads, updates, creates, deletes int
}

func (s *reportStub) GetByInterviewID(context.Context, string) (entity.Report, error) {
	s.reads++
	return s.existing, s.lookupErr
}
func (s *reportStub) Update(_ context.Context, r entity.Report) (entity.Report, error) {
	s.updates++
	if s.updateErr != nil {
		return entity.Report{}, s.updateErr
	}
	s.existing = r
	return r, nil
}
func (s *reportStub) Create(_ context.Context, r entity.Report) (entity.Report, error) {
	s.creates++
	return r, nil
}
func (s *reportStub) Delete(context.Context, string) error {
	s.deletes++
	s.existing = entity.Report{}
	return nil
}

type answersStub struct{ saves, loads int }

func (s *answersStub) Save(context.Context, string, []usecase.AnswerInput) error {
	s.saves++
	return nil
}
func (s *answersStub) Load(context.Context, string) ([]usecase.AnswerInput, error) {
	s.loads++
	return nil, nil
}

type analyzerStub struct{}

func (analyzerStub) Analyze(context.Context, string) (usecase.GeminiAnalysis, error) {
	return usecase.GeminiAnalysis{OverallScore: 75}, nil
}

func TestGenerateChecksOwnerBeforeReportOrCache(t *testing.T) {
	for _, tc := range []struct {
		name  string
		owner int64
		err   error
	}{{"wrong owner", 2, nil}, {"upstream denial", 1, errors.New("forbidden")}} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &reportStub{}
			cache := &answersStub{}
			interview := &interviewStub{owner: tc.owner, err: tc.err}
			s := NewAnalysisService(repo, interview, nil, nil, nil, cache)
			_, _, err := s.Generate(context.Background(), "victim", 1, []usecase.AnswerInput{{StepID: "step"}}, false)
			if err == nil {
				t.Fatal("expected rejection")
			}
			if repo.reads+repo.updates+repo.creates+repo.deletes+cache.saves+cache.loads != 0 {
				t.Fatal("unauthorized request touched report or cache")
			}
		})
	}
}
func TestGenerateDoesNotTreatDatabaseFailureAsMissing(t *testing.T) {
	repo := &reportStub{lookupErr: errors.New("database unavailable")}
	cache := &answersStub{}
	s := NewAnalysisService(repo, &interviewStub{owner: 1}, nil, nil, nil, cache)
	_, _, err := s.Generate(context.Background(), "interview", 1, []usecase.AnswerInput{{StepID: "step"}}, false)
	if err == nil || !strings.Contains(err.Error(), "database unavailable") {
		t.Fatalf("err=%v", err)
	}
	if cache.saves != 0 || repo.creates != 0 {
		t.Fatal("lookup failure caused side effects")
	}
}
func TestGenerateReplacementPreservesOldReportOnFailure(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "successful update", true: "failed update"}[fail], func(t *testing.T) {
			old := entity.Report{ID: "existing", UserID: 1, InterviewID: "interview", Notes: "invalid old metadata"}
			repo := &reportStub{existing: old}
			if fail {
				repo.updateErr = errors.New("write failed")
			}
			interview := &interviewStub{owner: 1}
			s := NewAnalysisService(repo, interview, nil, analyzerStub{}, nil, &answersStub{})
			result, _, err := s.Generate(context.Background(), "interview", 1, []usecase.AnswerInput{{StepID: "step"}}, false)
			if repo.deletes != 0 || repo.creates != 0 || repo.updates != 1 {
				t.Fatalf("replacement used wrong persistence operation: %+v", repo)
			}
			if fail {
				if err == nil || repo.existing != old || interview.completed != 0 {
					t.Fatal("failed update destroyed existing report or completed interview")
				}
			} else {
				if err != nil || result.ID != old.ID || interview.completed != 1 {
					t.Fatalf("result=%+v err=%v", result, err)
				}
			}
		})
	}
}

func TestGenerateCachedReportRequiresMatchingOwner(t *testing.T) {
	for _, owner := range []int64{1, 2} {
		t.Run(map[int64]string{1: "owner", 2: "foreign persisted report"}[owner], func(t *testing.T) {
			repo := &reportStub{existing: entity.Report{ID: "existing", UserID: owner, Notes: `{"overall_score":80,"comments":"ready"}`}}
			cache := &answersStub{}
			s := NewAnalysisService(repo, &interviewStub{owner: 1}, nil, nil, nil, cache)
			result, _, err := s.Generate(context.Background(), "interview", 1, []usecase.AnswerInput{{StepID: "step"}}, false)
			if owner == 1 {
				if err != nil || result.ID != "existing" {
					t.Fatalf("cached result: %+v err=%v", result, err)
				}
			} else {
				if err == nil {
					t.Fatal("foreign report was returned")
				}
			}
			if cache.saves+cache.loads+repo.updates+repo.creates+repo.deletes != 0 {
				t.Fatal("cached lookup changed state")
			}
		})
	}
}
