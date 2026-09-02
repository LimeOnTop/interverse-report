package usecase

import (
	"context"

	"github.com/LimeOnTop/interverse-report/internal/entity"
)

type ReportRepository interface {
	Create(ctx context.Context, report entity.Report) (entity.Report, error)
	GetByID(ctx context.Context, id string) (entity.Report, error)
	GetByInterviewID(ctx context.Context, interviewID string) (entity.Report, error)
	GetByInterviewer(ctx context.Context, interviewerID string, limit, offset int) ([]entity.Report, error)
	Update(ctx context.Context, report entity.Report) (entity.Report, error)
	Delete(ctx context.Context, id string) error
}
