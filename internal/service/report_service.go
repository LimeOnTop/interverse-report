package service

import (
	"fmt"

	"github.com/LimeOnTop/interverse-report/internal/models"
	"github.com/LimeOnTop/interverse-report/internal/repository"
)

type ReportService struct {
	reportRepo          repository.ReportRepository
	userServiceURL      string
	interviewServiceURL string
}

func NewReportService(reportRepo repository.ReportRepository, userServiceURL, interviewServiceURL string) *ReportService {
	return &ReportService{
		reportRepo:          reportRepo,
		userServiceURL:      userServiceURL,
		interviewServiceURL: interviewServiceURL,
	}
}

func (s *ReportService) CreateReport(report *models.Report) (*models.Report, error) {
	if err := s.reportRepo.CreateReport(report); err != nil {
		return nil, fmt.Errorf("failed to create report: %w", err)
	}

	return report, nil
}

func (s *ReportService) GetReportByID(id string) (*models.Report, error) {
	report, err := s.reportRepo.GetReportByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get report: %w", err)
	}

	return report, nil
}

func (s *ReportService) GetReportsByInterviewer(interviewerID string, limit, offset int) ([]*models.Report, error) {
	reports, err := s.reportRepo.GetReportsByInterviewer(interviewerID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get reports: %w", err)
	}

	return reports, nil
}

func (s *ReportService) UpdateReport(report *models.Report) (*models.Report, error) {
	if err := s.reportRepo.UpdateReport(report); err != nil {
		return nil, fmt.Errorf("failed to update report: %w", err)
	}

	return report, nil
}

func (s *ReportService) DeleteReport(id string) error {
	return s.reportRepo.DeleteReport(id)
}

func (s *ReportService) ValidateUser(userID string) error {
	// In a real implementation, this would call the User Service via gRPC
	// For now, we'll assume the user is valid
	return nil
}

func (s *ReportService) ValidateInterview(interviewID string) error {
	// In a real implementation, this would call the Interview Service via gRPC
	// For now, we'll assume the interview is valid
	return nil
}

