package repository

import (
	"context"
	"fmt"

	"turno-papa/internal/db"
	"turno-papa/internal/models"
)

type PatternRepository interface {
	Create(pattern *models.ShiftPattern) error
	GetByID(id string) (*models.ShiftPattern, error)
	GetByCompany(companyID string) ([]*models.ShiftPattern, error)
	Update(pattern *models.ShiftPattern) error
	Delete(id string) error
}

type PatternRepositoryImpl struct {
	db *db.DB
}

func NewPatternRepository(database *db.DB) *PatternRepositoryImpl {
	return &PatternRepositoryImpl{db: database}
}

func (r *PatternRepositoryImpl) Create(pattern *models.ShiftPattern) error {
	query := `
		INSERT INTO shift_patterns (pattern_id, company_id, name, work_days, off_days, is_legal_modality, legal_reference)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.Pool.Exec(context.Background(), query,
		pattern.PatternID, pattern.CompanyID, pattern.Name,
		pattern.WorkDays, pattern.OffDays, pattern.IsLegalModality, pattern.LegalReference,
	)
	if err != nil {
		return fmt.Errorf("CreatePattern: %w", err)
	}
	return nil
}

func (r *PatternRepositoryImpl) GetByID(id string) (*models.ShiftPattern, error) {
	query := `SELECT pattern_id, company_id, name, work_days, off_days, is_legal_modality, legal_reference, created_at FROM shift_patterns WHERE pattern_id = $1`
	row := r.db.Pool.QueryRow(context.Background(), query, id)

	var p models.ShiftPattern
	err := row.Scan(&p.PatternID, &p.CompanyID, &p.Name, &p.WorkDays, &p.OffDays, &p.IsLegalModality, &p.LegalReference, &p.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("GetPatternByID: %w", err)
	}
	return &p, nil
}

func (r *PatternRepositoryImpl) GetByCompany(companyID string) ([]*models.ShiftPattern, error) {
	query := `SELECT pattern_id, company_id, name, work_days, off_days, is_legal_modality, legal_reference, created_at FROM shift_patterns WHERE company_id = $1`
	rows, err := r.db.Pool.Query(context.Background(), query, companyID)
	if err != nil {
		return nil, fmt.Errorf("GetPatternsByCompany: %w", err)
	}
	defer rows.Close()

	var patterns []*models.ShiftPattern
	for rows.Next() {
		var p models.ShiftPattern
		err := rows.Scan(&p.PatternID, &p.CompanyID, &p.Name, &p.WorkDays, &p.OffDays, &p.IsLegalModality, &p.LegalReference, &p.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan pattern: %w", err)
		}
		patterns = append(patterns, &p)
	}
	return patterns, nil
}

func (r *PatternRepositoryImpl) Update(pattern *models.ShiftPattern) error {
	query := `UPDATE shift_patterns SET name = $1, work_days = $2, off_days = $3, is_legal_modality = $4, legal_reference = $5 WHERE pattern_id = $6`
	_, err := r.db.Pool.Exec(context.Background(), query, pattern.Name, pattern.WorkDays, pattern.OffDays, pattern.IsLegalModality, pattern.LegalReference, pattern.PatternID)
	if err != nil {
		return fmt.Errorf("UpdatePattern: %w", err)
	}
	return nil
}

func (r *PatternRepositoryImpl) Delete(id string) error {
	_, err := r.db.Pool.Exec(context.Background(), `DELETE FROM shift_patterns WHERE pattern_id = $1`, id)
	if err != nil {
		return fmt.Errorf("DeletePattern: %w", err)
	}
	return nil
}