package usecase

import (
	"context"

	"github.com/LimeOnTop/interverse-report/internal/entity"
)

type Report interface {
	Create(ctx context.Context, report entity.Report) (ReportDTO, error)
	GetByID(ctx context.Context, id string) (ReportDTO, error)
	GetByUser(ctx context.Context, userID int64, limit, offset int64) ([]ReportDTO, error)
	Update(ctx context.Context, report entity.Report) (ReportDTO, error)
	Delete(ctx context.Context, id string) error
}
