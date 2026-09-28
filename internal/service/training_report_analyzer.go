package service

import (
	"context"

	"github.com/LimeOnTop/interverse-report/internal/apperr"
	"github.com/LimeOnTop/interverse-report/internal/entity"
	"github.com/LimeOnTop/interverse-report/internal/usecase"
)

// TrainingReportAnalyzer handles async analysis of a completed training session.
type TrainingReportAnalyzer struct {
	// analysis *AnalysisService // wire when pending→processing→ready path is ready
}

func NewTrainingReportAnalyzer() *TrainingReportAnalyzer {
	return &TrainingReportAnalyzer{}
}

var _ usecase.TrainingReportAnalyzer = (*TrainingReportAnalyzer)(nil)

func (a *TrainingReportAnalyzer) AnalyzeTrainingReport(ctx context.Context, msg entity.ReportMessage) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// TODO: set status=processing, load answers from cache, run analysis, set ready/failed.
	apperr.Logf(
		"analyze training report: event=%s report=%s interview=%s user=%d",
		msg.EventID, msg.ReportID, msg.InterviewID, msg.UserID,
	)
	return nil
}
