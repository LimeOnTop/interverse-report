package usecase

import (
	"context"

	"github.com/LimeOnTop/interverse-report/internal/entity"
)

type Report interface {
	Create(ctx context.Context, report entity.Report) (ReportDTO, error)
	GetByID(ctx context.Context, id string) (ReportDTO, error)
	GetByInterviewer(ctx context.Context, interviewerID string, limit, offset int) ([]ReportDTO, error)
	Update(ctx context.Context, report entity.Report) (ReportDTO, error)
	Delete(ctx context.Context, id string) error
}
