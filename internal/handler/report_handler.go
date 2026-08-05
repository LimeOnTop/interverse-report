package handler

import (
	"context"
	"time"

	pb "github.com/LimeOnTop/interverse-report/gen"
	"github.com/LimeOnTop/interverse-report/internal/models"
	"github.com/LimeOnTop/interverse-report/internal/service"
)

type ReportHandler struct {
	pb.UnimplementedReportServiceServer
	reportService *service.ReportService
}

func NewReportHandler(reportService *service.ReportService) *ReportHandler {
	return &ReportHandler{
		reportService: reportService,
	}
}

func (h *ReportHandler) CreateReport(ctx context.Context, req *pb.CreateReportRequest) (*pb.CreateReportResponse, error) {
	report := &models.Report{
		InterviewID:         req.InterviewId,
		CandidateID:         req.CandidateId,
		InterviewerID:       req.InterviewerId,
		OverallRating:       req.OverallRating,
		TechnicalSkills:     req.TechnicalSkills,
		CommunicationSkills: req.CommunicationSkills,
		ProblemSolving:      req.ProblemSolving,
		Strengths:           req.Strengths,
		Weaknesses:          req.Weaknesses,
		Recommendations:     req.Recommendations,
		Notes:               req.Notes,
	}

	createdReport, err := h.reportService.CreateReport(report)
	if err != nil {
		return &pb.CreateReportResponse{
			Response: &pb.Response{
				Success: false,
				Error:   err.Error(),
			},
		}, nil
	}

	return &pb.CreateReportResponse{
		Response: &pb.Response{
			Success: true,
			Message: "Report created successfully",
		},
		Report: &pb.Report{
			Id:                  createdReport.ID,
			InterviewId:         createdReport.InterviewID,
			CandidateId:         createdReport.CandidateID,
			InterviewerId:       createdReport.InterviewerID,
			OverallRating:       createdReport.OverallRating,
			TechnicalSkills:     createdReport.TechnicalSkills,
			CommunicationSkills: createdReport.CommunicationSkills,
			ProblemSolving:      createdReport.ProblemSolving,
			Strengths:           createdReport.Strengths,
			Weaknesses:          createdReport.Weaknesses,
			Recommendations:     createdReport.Recommendations,
			Notes:               createdReport.Notes,
			CreatedAt:           createdReport.CreatedAt.Format(time.RFC3339),
			UpdatedAt:           createdReport.UpdatedAt.Format(time.RFC3339),
		},
	}, nil
}

func (h *ReportHandler) GetReport(ctx context.Context, req *pb.GetReportRequest) (*pb.GetReportResponse, error) {
	report, err := h.reportService.GetReportByID(req.ReportId)
	if err != nil {
		return &pb.GetReportResponse{
			Response: &pb.Response{
				Success: false,
				Error:   err.Error(),
			},
		}, nil
	}

	return &pb.GetReportResponse{
		Response: &pb.Response{
			Success: true,
		},
		Report: &pb.Report{
			Id:                  report.ID,
			InterviewId:         report.InterviewID,
			CandidateId:         report.CandidateID,
			InterviewerId:       report.InterviewerID,
			OverallRating:       report.OverallRating,
			TechnicalSkills:     report.TechnicalSkills,
			CommunicationSkills: report.CommunicationSkills,
			ProblemSolving:      report.ProblemSolving,
			Strengths:           report.Strengths,
			Weaknesses:          report.Weaknesses,
			Recommendations:     report.Recommendations,
			Notes:               report.Notes,
			CreatedAt:           report.CreatedAt.Format(time.RFC3339),
			UpdatedAt:           report.UpdatedAt.Format(time.RFC3339),
		},
	}, nil
}

func (h *ReportHandler) GetReports(ctx context.Context, req *pb.GetReportsRequest) (*pb.GetReportsResponse, error) {
	limit := 10
	offset := 0

	if req.Pagination != nil {
		limit = int(req.Pagination.Limit)
		offset = int(req.Pagination.Page-1) * int(req.Pagination.Limit)
	}

	reports, err := h.reportService.GetReportsByInterviewer(req.InterviewerId, limit, offset)
	if err != nil {
		return &pb.GetReportsResponse{
			Response: &pb.Response{
				Success: false,
				Error:   err.Error(),
			},
		}, nil
	}

	var pbReports []*pb.Report
	for _, report := range reports {
		pbReports = append(pbReports, &pb.Report{
			Id:                  report.ID,
			InterviewId:         report.InterviewID,
			CandidateId:         report.CandidateID,
			InterviewerId:       report.InterviewerID,
			OverallRating:       report.OverallRating,
			TechnicalSkills:     report.TechnicalSkills,
			CommunicationSkills: report.CommunicationSkills,
			ProblemSolving:      report.ProblemSolving,
			Strengths:           report.Strengths,
			Weaknesses:          report.Weaknesses,
			Recommendations:     report.Recommendations,
			Notes:               report.Notes,
			CreatedAt:           report.CreatedAt.Format(time.RFC3339),
			UpdatedAt:           report.UpdatedAt.Format(time.RFC3339),
		})
	}

	return &pb.GetReportsResponse{
		Response: &pb.Response{
			Success: true,
		},
		Reports: pbReports,
		Pagination: &pb.Pagination{
			Page:  req.Pagination.Page,
			Limit: req.Pagination.Limit,
			Total: int32(len(pbReports)),
		},
	}, nil
}

func (h *ReportHandler) UpdateReport(ctx context.Context, req *pb.UpdateReportRequest) (*pb.UpdateReportResponse, error) {
	report := &models.Report{
		ID:                  req.ReportId,
		OverallRating:       req.OverallRating,
		TechnicalSkills:     req.TechnicalSkills,
		CommunicationSkills: req.CommunicationSkills,
		ProblemSolving:      req.ProblemSolving,
		Strengths:           req.Strengths,
		Weaknesses:          req.Weaknesses,
		Recommendations:     req.Recommendations,
		Notes:               req.Notes,
	}

	updatedReport, err := h.reportService.UpdateReport(report)
	if err != nil {
		return &pb.UpdateReportResponse{
			Response: &pb.Response{
				Success: false,
				Error:   err.Error(),
			},
		}, nil
	}

	return &pb.UpdateReportResponse{
		Response: &pb.Response{
			Success: true,
			Message: "Report updated successfully",
		},
		Report: &pb.Report{
			Id:                  updatedReport.ID,
			InterviewId:         updatedReport.InterviewID,
			CandidateId:         updatedReport.CandidateID,
			InterviewerId:       updatedReport.InterviewerID,
			OverallRating:       updatedReport.OverallRating,
			TechnicalSkills:     updatedReport.TechnicalSkills,
			CommunicationSkills: updatedReport.CommunicationSkills,
			ProblemSolving:      updatedReport.ProblemSolving,
			Strengths:           updatedReport.Strengths,
			Weaknesses:          updatedReport.Weaknesses,
			Recommendations:     updatedReport.Recommendations,
			Notes:               updatedReport.Notes,
			CreatedAt:           updatedReport.CreatedAt.Format(time.RFC3339),
			UpdatedAt:           updatedReport.UpdatedAt.Format(time.RFC3339),
		},
	}, nil
}

func (h *ReportHandler) DeleteReport(ctx context.Context, req *pb.DeleteReportRequest) (*pb.Response, error) {
	err := h.reportService.DeleteReport(req.ReportId)
	if err != nil {
		return &pb.Response{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &pb.Response{
		Success: true,
		Message: "Report deleted successfully",
	}, nil
}
