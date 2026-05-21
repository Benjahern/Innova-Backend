package repository

import (
	"context"
	"fmt"
	"time"

	"turno-papa/internal/db"
	"turno-papa/internal/models"
)

type WeeklyHoursRepository interface {
	Upsert(summary *models.WeeklyHoursSummary) error
	GetByUserAndWeek(userID string, weekStart time.Time) (*models.WeeklyHoursSummary, error)
	GetByCompanyAndWeek(companyID string, weekStart time.Time) ([]*models.WeeklyHoursSummary, error)
}

type WeeklyHoursRepositoryImpl struct {
	db *db.DB
}

func NewWeeklyHoursRepository(database *db.DB) *WeeklyHoursRepositoryImpl {
	return &WeeklyHoursRepositoryImpl{db: database}
}

func (r *WeeklyHoursRepositoryImpl) Upsert(summary *models.WeeklyHoursSummary) error {
	query := `
		INSERT INTO weekly_hours_summary (summary_id, user_id, week_start, total_hours, expected_hours)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, week_start) DO UPDATE SET
			total_hours = EXCLUDED.total_hours,
			expected_hours = EXCLUDED.expected_hours
	`
	_, err := r.db.Pool.Exec(context.Background(), query,
		summary.SummaryID, summary.UserID, summary.WeekStart, summary.TotalHours, summary.ExpectedHours,
	)
	if err != nil {
		return fmt.Errorf("UpsertWeeklyHours: %w", err)
	}
	return nil
}

func (r *WeeklyHoursRepositoryImpl) GetByUserAndWeek(userID string, weekStart time.Time) (*models.WeeklyHoursSummary, error) {
	query := `SELECT summary_id, user_id, week_start, total_hours, expected_hours FROM weekly_hours_summary WHERE user_id = $1 AND week_start = $2`
	row := r.db.Pool.QueryRow(context.Background(), query, userID, weekStart)

	var summary models.WeeklyHoursSummary
	err := row.Scan(&summary.SummaryID, &summary.UserID, &summary.WeekStart, &summary.TotalHours, &summary.ExpectedHours)
	if err != nil {
		return nil, fmt.Errorf("GetWeeklyHoursByUserAndWeek: %w", err)
	}
	return &summary, nil
}

func (r *WeeklyHoursRepositoryImpl) GetByCompanyAndWeek(companyID string, weekStart time.Time) ([]*models.WeeklyHoursSummary, error) {
	query := `
		SELECT whs.summary_id, whs.user_id, whs.week_start, whs.total_hours, whs.expected_hours
		FROM weekly_hours_summary whs
		JOIN users u ON whs.user_id = u.user_id
		WHERE u.company_id = $1 AND whs.week_start = $2
		ORDER BY whs.total_hours DESC
	`
	rows, err := r.db.Pool.Query(context.Background(), query, companyID, weekStart)
	if err != nil {
		return nil, fmt.Errorf("GetWeeklyHoursByCompanyAndWeek: %w", err)
	}
	defer rows.Close()

	var summaries []*models.WeeklyHoursSummary
	for rows.Next() {
		var s models.WeeklyHoursSummary
		err := rows.Scan(&s.SummaryID, &s.UserID, &s.WeekStart, &s.TotalHours, &s.ExpectedHours)
		if err != nil {
			return nil, fmt.Errorf("GetWeeklyHoursByCompanyAndWeek scan: %w", err)
		}
		summaries = append(summaries, &s)
	}

	return summaries, nil
}