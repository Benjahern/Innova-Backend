package repository

import (
	"context"
	"fmt"

	"turno-papa/internal/db"
	"turno-papa/internal/models"
)

type MonthlyArrearsRepository interface {
	Upsert(summary *models.MonthlyArrearsSummary) error
	GetByUserAndMonth(userID string, year, month int) (*models.MonthlyArrearsSummary, error)
	GetByCompanyAndMonth(companyID string, year, month int) ([]*models.MonthlyArrearsSummary, error)
}

type MonthlyArrearsRepositoryImpl struct {
	db *db.DB
}

func NewMonthlyArrearsRepository(database *db.DB) *MonthlyArrearsRepositoryImpl {
	return &MonthlyArrearsRepositoryImpl{db: database}
}

func (r *MonthlyArrearsRepositoryImpl) Upsert(summary *models.MonthlyArrearsSummary) error {
	query := `
		INSERT INTO monthly_arrears_summary (summary_id, user_id, year, month, total_arrears_minutes, days_with_arrears)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id, year, month) DO UPDATE SET
			total_arrears_minutes = EXCLUDED.total_arrears_minutes,
			days_with_arrears = EXCLUDED.days_with_arrears
	`
	_, err := r.db.Pool.Exec(context.Background(), query,
		summary.SummaryID, summary.UserID, summary.Year, summary.Month, summary.TotalArrearsMin, summary.DaysWithArrears,
	)
	if err != nil {
		return fmt.Errorf("UpsertMonthlyArrears: %w", err)
	}
	return nil
}

func (r *MonthlyArrearsRepositoryImpl) GetByUserAndMonth(userID string, year, month int) (*models.MonthlyArrearsSummary, error) {
	query := `SELECT summary_id, user_id, year, month, total_arrears_minutes, days_with_arrears FROM monthly_arrears_summary WHERE user_id = $1 AND year = $2 AND month = $3`
	row := r.db.Pool.QueryRow(context.Background(), query, userID, year, month)

	var summary models.MonthlyArrearsSummary
	err := row.Scan(&summary.SummaryID, &summary.UserID, &summary.Year, &summary.Month, &summary.TotalArrearsMin, &summary.DaysWithArrears)
	if err != nil {
		return nil, fmt.Errorf("GetMonthlyArrearsByUserAndMonth: %w", err)
	}
	return &summary, nil
}

func (r *MonthlyArrearsRepositoryImpl) GetByCompanyAndMonth(companyID string, year, month int) ([]*models.MonthlyArrearsSummary, error) {
	query := `
		SELECT mas.summary_id, mas.user_id, mas.year, mas.month, mas.total_arrears_minutes, mas.days_with_arrears
		FROM monthly_arrears_summary mas
		JOIN users u ON mas.user_id = u.user_id
		WHERE u.company_id = $1 AND mas.year = $2 AND mas.month = $3
	`
	rows, err := r.db.Pool.Query(context.Background(), query, companyID, year, month)
	if err != nil {
		return nil, fmt.Errorf("GetMonthlyArrearsByCompanyAndMonth: %w", err)
	}
	defer rows.Close()

	var summaries []*models.MonthlyArrearsSummary
	for rows.Next() {
		var s models.MonthlyArrearsSummary
		err := rows.Scan(&s.SummaryID, &s.UserID, &s.Year, &s.Month, &s.TotalArrearsMin, &s.DaysWithArrears)
		if err != nil {
			return nil, fmt.Errorf("GetMonthlyArrearsByCompanyAndMonth scan: %w", err)
		}
		summaries = append(summaries, &s)
	}

	return summaries, nil
}