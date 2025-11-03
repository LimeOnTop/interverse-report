package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/inter-verse/report-service/internal/models"
)

type ReportRepository struct {
	db *sql.DB
}

func NewReportRepository(db *sql.DB) *ReportRepository {
	return &ReportRepository{db: db}
}

func (r *ReportRepository) CreateReport(report *models.Report) error {
	report.ID = uuid.New().String()
	report.CreatedAt = time.Now()
	report.UpdatedAt = time.Now()

	query := `
		INSERT INTO reports (id, interview_id, candidate_id, interviewer_id, overall_rating, technical_skills, communication_skills, problem_solving, strengths, weaknesses, recommendations, notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	_, err := r.db.Exec(query,
		report.ID, report.InterviewID, report.CandidateID, report.InterviewerID,
		report.OverallRating, report.TechnicalSkills, report.CommunicationSkills,
		report.ProblemSolving, report.Strengths, report.Weaknesses,
		report.Recommendations, report.Notes, report.CreatedAt, report.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create report: %w", err)
	}

	return nil
}

func (r *ReportRepository) GetReportByID(id string) (*models.Report, error) {
	query := `
		SELECT id, interview_id, candidate_id, interviewer_id, overall_rating, technical_skills, communication_skills, problem_solving, strengths, weaknesses, recommendations, notes, created_at, updated_at
		FROM reports WHERE id = $1
	`

	report := &models.Report{}
	err := r.db.QueryRow(query, id).Scan(
		&report.ID, &report.InterviewID, &report.CandidateID, &report.InterviewerID,
		&report.OverallRating, &report.TechnicalSkills, &report.CommunicationSkills,
		&report.ProblemSolving, &report.Strengths, &report.Weaknesses,
		&report.Recommendations, &report.Notes, &report.CreatedAt, &report.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("report not found")
		}
		return nil, fmt.Errorf("failed to get report: %w", err)
	}

	return report, nil
}

func (r *ReportRepository) GetReportsByInterviewer(interviewerID string, limit, offset int) ([]*models.Report, error) {
	query := `
		SELECT id, interview_id, candidate_id, interviewer_id, overall_rating, technical_skills, communication_skills, problem_solving, strengths, weaknesses, recommendations, notes, created_at, updated_at
		FROM reports 
		WHERE interviewer_id = $1 
		ORDER BY created_at DESC 
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(query, interviewerID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get reports: %w", err)
	}
	defer rows.Close()

	var reports []*models.Report
	for rows.Next() {
		report := &models.Report{}
		err := rows.Scan(
			&report.ID, &report.InterviewID, &report.CandidateID, &report.InterviewerID,
			&report.OverallRating, &report.TechnicalSkills, &report.CommunicationSkills,
			&report.ProblemSolving, &report.Strengths, &report.Weaknesses,
			&report.Recommendations, &report.Notes, &report.CreatedAt, &report.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan report: %w", err)
		}
		reports = append(reports, report)
	}

	return reports, nil
}

func (r *ReportRepository) UpdateReport(report *models.Report) error {
	report.UpdatedAt = time.Now()

	query := `
		UPDATE reports 
		SET overall_rating = $1, technical_skills = $2, communication_skills = $3, problem_solving = $4, strengths = $5, weaknesses = $6, recommendations = $7, notes = $8, updated_at = $9
		WHERE id = $10
	`

	_, err := r.db.Exec(query,
		report.OverallRating, report.TechnicalSkills, report.CommunicationSkills,
		report.ProblemSolving, report.Strengths, report.Weaknesses,
		report.Recommendations, report.Notes, report.UpdatedAt, report.ID)
	if err != nil {
		return fmt.Errorf("failed to update report: %w", err)
	}

	return nil
}

func (r *ReportRepository) DeleteReport(id string) error {
	query := `DELETE FROM reports WHERE id = $1`

	_, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete report: %w", err)
	}

	return nil
}

