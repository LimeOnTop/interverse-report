package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/LimeOnTop/interverse-report/internal/entity"
	"github.com/google/uuid"
)

type ReportRepository struct {
	db *sql.DB
}

func NewReportRepository(db *sql.DB) *ReportRepository {
	return &ReportRepository{db: db}
}

func (r *ReportRepository) Create(ctx context.Context, report entity.Report) (entity.Report, error) {
	report.ID = uuid.New().String()
	report.CreatedAt = time.Now()
	report.UpdatedAt = time.Now()

	query := `
		INSERT INTO reports (
			id, interview_id, candidate_id, interviewer_id,
			overall_rating, technical_skills, communication_skills, problem_solving,
			strengths, weaknesses, recommendations, notes, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	_, err := r.db.ExecContext(ctx, query,
		report.ID, report.InterviewID, report.CandidateID, report.InterviewerID,
		report.OverallRating, report.TechnicalSkills, report.CommunicationSkills,
		report.ProblemSolving, report.Strengths, report.Weaknesses,
		report.Recommendations, report.Notes, report.CreatedAt, report.UpdatedAt,
	)
	if err != nil {
		return entity.Report{}, fmt.Errorf("create report: %w", err)
	}

	return report, nil
}

func (r *ReportRepository) GetByID(ctx context.Context, id string) (entity.Report, error) {
	query := `
		SELECT id, interview_id, candidate_id, interviewer_id,
		       overall_rating, technical_skills, communication_skills, problem_solving,
		       strengths, weaknesses, recommendations, notes, created_at, updated_at
		FROM reports WHERE id = $1
	`

	var report entity.Report
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&report.ID, &report.InterviewID, &report.CandidateID, &report.InterviewerID,
		&report.OverallRating, &report.TechnicalSkills, &report.CommunicationSkills,
		&report.ProblemSolving, &report.Strengths, &report.Weaknesses,
		&report.Recommendations, &report.Notes, &report.CreatedAt, &report.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return entity.Report{}, fmt.Errorf("get report: not found")
		}
		return entity.Report{}, fmt.Errorf("get report: %w", err)
	}

	return report, nil
}

func (r *ReportRepository) GetByInterviewer(ctx context.Context, interviewerID string, limit, offset int) ([]entity.Report, error) {
	query := `
		SELECT id, interview_id, candidate_id, interviewer_id,
		       overall_rating, technical_skills, communication_skills, problem_solving,
		       strengths, weaknesses, recommendations, notes, created_at, updated_at
		FROM reports
		WHERE interviewer_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, interviewerID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("get reports: %w", err)
	}
	defer rows.Close()

	return scanReports(rows)
}

func (r *ReportRepository) Update(ctx context.Context, report entity.Report) (entity.Report, error) {
	report.UpdatedAt = time.Now()

	query := `
		UPDATE reports
		SET overall_rating = $1, technical_skills = $2, communication_skills = $3,
		    problem_solving = $4, strengths = $5, weaknesses = $6,
		    recommendations = $7, notes = $8, updated_at = $9
		WHERE id = $10
	`

	_, err := r.db.ExecContext(ctx, query,
		report.OverallRating, report.TechnicalSkills, report.CommunicationSkills,
		report.ProblemSolving, report.Strengths, report.Weaknesses,
		report.Recommendations, report.Notes, report.UpdatedAt, report.ID,
	)
	if err != nil {
		return entity.Report{}, fmt.Errorf("update report: %w", err)
	}

	return report, nil
}

func (r *ReportRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM reports WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete report: %w", err)
	}

	return nil
}

func scanReports(rows *sql.Rows) ([]entity.Report, error) {
	var reports []entity.Report

	for rows.Next() {
		var report entity.Report
		if err := rows.Scan(
			&report.ID, &report.InterviewID, &report.CandidateID, &report.InterviewerID,
			&report.OverallRating, &report.TechnicalSkills, &report.CommunicationSkills,
			&report.ProblemSolving, &report.Strengths, &report.Weaknesses,
			&report.Recommendations, &report.Notes, &report.CreatedAt, &report.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan report: %w", err)
		}
		reports = append(reports, report)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate reports: %w", err)
	}

	return reports, nil
}
