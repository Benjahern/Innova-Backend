package repository

import (
	"context"
	"fmt"

	"turno-papa/internal/db"
	"turno-papa/internal/models"
)

type WorkShiftRepository interface {
	Create(shift *models.WorkShift) error
	GetByID(id string) (*models.WorkShift, error)
	GetByCompany(companyID string) ([]*models.WorkShift, error)
	Update(shift *models.WorkShift) error
	Delete(id string) error
}

type WorkShiftRepositoryImpl struct {
	db *db.DB
}

func NewWorkShiftRepository(database *db.DB) *WorkShiftRepositoryImpl {
	return &WorkShiftRepositoryImpl{db: database}
}

func (r *WorkShiftRepositoryImpl) Create(shift *models.WorkShift) error {
	query := `
		INSERT INTO work_shifts (shift_id, company_id, name, days, start_time, end_time, shift_type, pattern_id, lunch_start, lunch_end, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := r.db.Pool.Exec(context.Background(), query,
		shift.ShiftID, shift.CompanyID, shift.Name, shift.Days, shift.StartTime, shift.EndTime,
		shift.ShiftType, shift.PatternID, shift.LunchStart, shift.LunchEnd, shift.IsActive,
	)
	if err != nil {
		return fmt.Errorf("CreateWorkShift: %w", err)
	}
	return nil
}

func (r *WorkShiftRepositoryImpl) GetByID(id string) (*models.WorkShift, error) {
	query := `SELECT shift_id, company_id, name, days, start_time, end_time, shift_type, pattern_id, lunch_start, lunch_end, is_active FROM work_shifts WHERE shift_id = $1`
	row := r.db.Pool.QueryRow(context.Background(), query, id)

	var shift models.WorkShift
	err := row.Scan(&shift.ShiftID, &shift.CompanyID, &shift.Name, &shift.Days, &shift.StartTime, &shift.EndTime,
		&shift.ShiftType, &shift.PatternID, &shift.LunchStart, &shift.LunchEnd, &shift.IsActive)
	if err != nil {
		return nil, fmt.Errorf("GetWorkShiftByID: %w", err)
	}
	return &shift, nil
}

func (r *WorkShiftRepositoryImpl) GetByCompany(companyID string) ([]*models.WorkShift, error) {
	query := `
		SELECT s.shift_id, s.company_id, s.name, s.days, s.start_time, s.end_time, s.shift_type, s.pattern_id, s.lunch_start, s.lunch_end, s.is_active,
			   p.name as pattern_name
		FROM work_shifts s
		LEFT JOIN shift_patterns p ON s.pattern_id = p.pattern_id
		WHERE s.company_id = $1 AND s.deleted_at IS NULL`
	rows, err := r.db.Pool.Query(context.Background(), query, companyID)
	if err != nil {
		return nil, fmt.Errorf("GetWorkShiftsByCompany: %w", err)
	}
	defer rows.Close()

	var shifts []*models.WorkShift
	for rows.Next() {
		var shift models.WorkShift
		var patternName *string
		err := rows.Scan(&shift.ShiftID, &shift.CompanyID, &shift.Name, &shift.Days, &shift.StartTime, &shift.EndTime,
			&shift.ShiftType, &shift.PatternID, &shift.LunchStart, &shift.LunchEnd, &shift.IsActive, &patternName)
		if err != nil {
			return nil, fmt.Errorf("GetWorkShiftsByCompany scan: %w", err)
		}
		shift.PatternName = patternName
		shifts = append(shifts, &shift)
	}

	return shifts, nil
}

func (r *WorkShiftRepositoryImpl) Update(shift *models.WorkShift) error {
	query := `UPDATE work_shifts SET name = $1, days = $2, start_time = $3, end_time = $4, shift_type = $5, pattern_id = $6, lunch_start = $7, lunch_end = $8, is_active = $9 WHERE shift_id = $10`
	_, err := r.db.Pool.Exec(context.Background(), query, shift.Name, shift.Days, shift.StartTime, shift.EndTime,
		shift.ShiftType, shift.PatternID, shift.LunchStart, shift.LunchEnd, shift.IsActive, shift.ShiftID)
	if err != nil {
		return fmt.Errorf("UpdateWorkShift: %w", err)
	}
	return nil
}

func (r *WorkShiftRepositoryImpl) Delete(id string) error {
	query := `DELETE FROM work_shifts WHERE shift_id = $1`
	_, err := r.db.Pool.Exec(context.Background(), query, id)
	if err != nil {
		return fmt.Errorf("DeleteWorkShift: %w", err)
	}
	return nil
}