package service

import (
	"context"
	"fmt"

	"github.com/LimeOnTop/interverse-report/internal/entity"
	"github.com/LimeOnTop/interverse-report/internal/usecase"
)

type ReportService struct {
	repository usecase.ReportRepository
}

func NewReportService(repository usecase.ReportRepository) *ReportService {
	return &ReportService{repository: repository}
}

var _ usecase.Report = (*ReportService)(nil)

func (s *ReportService) Create(ctx context.Context, report entity.Report) (usecase.ReportDTO, error) {
	created, err := s.repository.Create(ctx, report)
	if err != nil {
		return usecase.ReportDTO{}, fmt.Errorf("create report: %w", err)
	}

	return toDTO(created), nil
}

func (s *ReportService) GetByID(ctx context.Context, id string) (usecase.ReportDTO, error) {
	report, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return usecase.ReportDTO{}, fmt.Errorf("get report: %w", err)
	}

	return toDTO(report), nil
}

func (s *ReportService) GetByUser(ctx context.Context, userID int64, limit, offset int64) ([]usecase.ReportDTO, error) {
	reports, err := s.repository.GetByUser(ctx, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("get reports: %w", err)
	}

	return toDTOs(reports), nil
}

func (s *ReportService) Update(ctx context.Context, report entity.Report) (usecase.ReportDTO, error) {
	existing, err := s.repository.GetByID(ctx, report.ID)
	if err != nil {
		return usecase.ReportDTO{}, fmt.Errorf("get report: %w", err)
	}

	existing.OverallRating = report.OverallRating
	existing.TechnicalSkills = report.TechnicalSkills
	existing.CommunicationSkills = report.CommunicationSkills
	existing.ProblemSolving = report.ProblemSolving
	existing.Strengths = report.Strengths
	existing.Weaknesses = report.Weaknesses
	existing.Recommendations = report.Recommendations
	existing.Notes = report.Notes

	updated, err := s.repository.Update(ctx, existing)
	if err != nil {
		return usecase.ReportDTO{}, fmt.Errorf("update report: %w", err)
	}

	return toDTO(updated), nil
}

func (s *ReportService) Delete(ctx context.Context, id string) error {
	if err := s.repository.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete report: %w", err)
	}
	return nil
}

func toDTO(report entity.Report) usecase.ReportDTO {
	return usecase.ReportDTO{
		ID:                  report.ID,
		InterviewID:         report.InterviewID,
		UserID:              report.UserID,
		OverallRating:       report.OverallRating,
		TechnicalSkills:     report.TechnicalSkills,
		CommunicationSkills: report.CommunicationSkills,
		ProblemSolving:      report.ProblemSolving,
		Strengths:           report.Strengths,
		Weaknesses:          report.Weaknesses,
		Recommendations:     report.Recommendations,
		Notes:               report.Notes,
		CreatedAt:           report.CreatedAt,
		UpdatedAt:           report.UpdatedAt,
	}
}

func toDTOs(reports []entity.Report) []usecase.ReportDTO {
	result := make([]usecase.ReportDTO, 0, len(reports))
	for _, report := range reports {
		result = append(result, toDTO(report))
	}
	return result
}
