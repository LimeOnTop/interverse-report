package usecase

import (
	"context"

	"github.com/LimeOnTop/interverse-report/internal/entity"
)

type TrainingReportAnalyzer interface {
	AnalyzeTrainingReport(ctx context.Context, msg entity.ReportMessage) error
}
