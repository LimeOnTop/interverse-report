package controller

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	pb "github.com/LimeOnTop/interverse-contracts/report/gen"
	"github.com/LimeOnTop/interverse-report/internal/apperr"
	"github.com/LimeOnTop/interverse-report/internal/entity"
	"github.com/LimeOnTop/interverse-report/internal/service"
	"github.com/LimeOnTop/interverse-report/internal/usecase"
)

type ReportController struct {
	pb.UnimplementedReportServiceServer
	report   usecase.Report
	analysis *service.AnalysisService
}

func NewReportController(report usecase.Report, analysis *service.AnalysisService) *ReportController {
	return &ReportController{
		report:   report,
		analysis: analysis,
	}
}

func (c *ReportController) CreateReport(ctx context.Context, req *pb.CreateReportRequest) (*pb.CreateReportResponse, error) {
	userID, err := strconv.ParseInt(req.GetUserId(), 10, 64)
	created, err := c.report.Create(ctx, entity.Report{
		InterviewID:         req.InterviewId,
		UserID:              userID,
		OverallRating:       req.OverallRating,
		TechnicalSkills:     req.TechnicalSkills,
		CommunicationSkills: req.CommunicationSkills,
		ProblemSolving:      req.ProblemSolving,
		Strengths:           req.Strengths,
		Weaknesses:          req.Weaknesses,
		Recommendations:     req.Recommendations,
		Notes:               req.Notes,
	})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, fmt.Errorf("context canceled: %w", err)
		}
		return &pb.CreateReportResponse{
			Response: &pb.Response{
				Success: false,
				Error:   apperr.Message(err, "request failed"),
			},
		}, nil
	}

	return &pb.CreateReportResponse{
		Response: &pb.Response{
			Success: true,
			Message: "Report created successfully",
		},
		Report: toProtoReport(created),
	}, nil
}

func (c *ReportController) GenerateReport(ctx context.Context, req *pb.GenerateReportRequest) (*pb.GenerateReportResponse, error) {
	userID, err := strconv.ParseInt(req.GetUserId(), 10, 64)
	if err != nil {
		return &pb.GenerateReportResponse{
			Response: &pb.Response{
				Success: false,
				Error:   "invalid user id",
			},
		}, nil
	}

	answers := make([]service.AnswerInput, 0, len(req.GetAnswers()))
	for _, answer := range req.GetAnswers() {
		input := service.AnswerInput{
			StepID:     answer.GetStepId(),
			QuestionID: answer.GetQuestionId(),
			ItemType:   answer.GetItemType(),
			TaskAnswer: answer.GetTaskAnswer(),
		}
		if answer.GetItemType() == "question" {
			selected := answer.GetSelectedOptionIndex()
			input.SelectedOptionIndex = &selected
		}
		answers = append(answers, input)
	}

	created, scores, err := c.analysis.Generate(ctx, req.GetInterviewId(), userID, answers)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, fmt.Errorf("context canceled: %w", err)
		}
		return &pb.GenerateReportResponse{
			Response: &pb.Response{
				Success: false,
				Error:   generateReportError(err),
			},
		}, nil
	}

	return &pb.GenerateReportResponse{
		Response: &pb.Response{
			Success: true,
			Message: "Report generated successfully",
		},
		Report: toProtoReport(created),
		Scores: &pb.AnalysisScores{
			OverallScore:      int32(scores.OverallScore),
			AlgorithmScore:    int32(scores.AlgorithmScore),
			ArchitectureScore: int32(scores.ArchitectureScore),
			CodingScore:       int32(scores.CodingScore),
			SoftSkillsScore:   int32(scores.SoftSkillsScore),
			Comments:          scores.Comments,
		},
	}, nil
}

func (c *ReportController) GetReport(ctx context.Context, req *pb.GetReportRequest) (*pb.GetReportResponse, error) {
	report, err := c.report.GetByID(ctx, req.ReportId)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, fmt.Errorf("context canceled: %w", err)
		}
		return &pb.GetReportResponse{
			Response: &pb.Response{
				Success: false,
				Error:   apperr.Message(err, "request failed"),
			},
		}, nil
	}

	return &pb.GetReportResponse{
		Response: &pb.Response{
			Success: true,
		},
		Report: toProtoReport(report),
	}, nil
}

func (c *ReportController) GetReports(ctx context.Context, req *pb.GetReportsRequest) (*pb.GetReportsResponse, error) {
	userID, err := strconv.ParseInt(req.GetUserId(), 10, 64)
	if err != nil {
		return &pb.GetReportsResponse{
			Response: &pb.Response{
				Success: false,
				Error:   "invalid user id",
			},
		}, nil
	}

	limit := int64(10)
	offset := int64(0)

	if req.Pagination != nil {
		limit = int64(req.Pagination.Limit)
		offset = int64(req.Pagination.Page-1) * int64(req.Pagination.Limit)
	}

	reports, err := c.report.GetByUser(ctx, userID, limit, offset)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, fmt.Errorf("context canceled: %w", err)
		}
		return &pb.GetReportsResponse{
			Response: &pb.Response{
				Success: false,
				Error:   apperr.Message(err, "request failed"),
			},
		}, nil
	}

	return &pb.GetReportsResponse{
		Response: &pb.Response{
			Success: true,
		},
		Reports: toProtoReports(reports),
		Pagination: &pb.Pagination{
			Page:  req.Pagination.GetPage(),
			Limit: req.Pagination.GetLimit(),
			Total: int32(len(reports)),
		},
	}, nil
}

func (c *ReportController) UpdateReport(ctx context.Context, req *pb.UpdateReportRequest) (*pb.UpdateReportResponse, error) {
	updated, err := c.report.Update(ctx, entity.Report{
		ID:                  req.ReportId,
		OverallRating:       req.OverallRating,
		TechnicalSkills:     req.TechnicalSkills,
		CommunicationSkills: req.CommunicationSkills,
		ProblemSolving:      req.ProblemSolving,
		Strengths:           req.Strengths,
		Weaknesses:          req.Weaknesses,
		Recommendations:     req.Recommendations,
		Notes:               req.Notes,
	})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, fmt.Errorf("context canceled: %w", err)
		}
		return &pb.UpdateReportResponse{
			Response: &pb.Response{
				Success: false,
				Error:   apperr.Message(err, "request failed"),
			},
		}, nil
	}

	return &pb.UpdateReportResponse{
		Response: &pb.Response{
			Success: true,
			Message: "Report updated successfully",
		},
		Report: toProtoReport(updated),
	}, nil
}

func (c *ReportController) DeleteReport(ctx context.Context, req *pb.DeleteReportRequest) (*pb.Response, error) {
	err := c.report.Delete(ctx, req.ReportId)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, fmt.Errorf("context canceled: %w", err)
		}
		return &pb.Response{
			Success: false,
			Error:   apperr.Message(err, "request failed"),
		}, nil
	}

	return &pb.Response{
		Success: true,
		Message: "Report deleted successfully",
	}, nil
}

func generateReportError(err error) string {
	lower := strings.ToLower(err.Error())
	switch {
	case strings.Contains(lower, "not found") || strings.Contains(lower, "не найдена"):
		return apperr.Message(err, "Тренировка не найдена")
	case strings.Contains(lower, "forbidden") || strings.Contains(lower, "нет доступа"):
		return apperr.Message(err, "Нет доступа к тренировке")
	default:
		return apperr.Message(err, "Не удалось сформировать отчёт")
	}
}

func toProtoReport(report usecase.ReportDTO) *pb.Report {
	return &pb.Report{
		Id:                  report.ID,
		InterviewId:         report.InterviewID,
		UserId:              strconv.FormatInt(report.UserID, 10),
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
	}
}

func toProtoReports(reports []usecase.ReportDTO) []*pb.Report {
	result := make([]*pb.Report, 0, len(reports))
	for _, report := range reports {
		result = append(result, toProtoReport(report))
	}
	return result
}
