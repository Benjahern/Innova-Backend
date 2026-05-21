package repository

import (
	"context"
	"fmt"

	"turno-papa/internal/db"
	"turno-papa/internal/models"
)

type CompanyRepository interface {
	Create(company *models.Company) error
	GetByID(id string) (*models.Company, error)
	GetByName(name string) (*models.Company, error)
	Update(company *models.Company) error
	Delete(id string) error
	ListAll(limit, offset int) ([]*models.Company, int, error)
}

type CompanyRepositoryImpl struct {
	db *db.DB
}

func NewCompanyRepository(database *db.DB) *CompanyRepositoryImpl {
	return &CompanyRepositoryImpl{db: database}
}

func (r *CompanyRepositoryImpl) Create(company *models.Company) error {
	query := `
		INSERT INTO companies (company_id, name, logo_url, config, default_start_time, default_end_time, work_hours_per_week, lunch_start, lunch_end, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
	`
	_, err := r.db.Pool.Exec(context.Background(), query,
		company.CompanyID, company.Name, company.LogoURL, company.Config, company.DefaultStartTime, company.DefaultEndTime, company.WorkHoursPerWeek, company.LunchStart, company.LunchEnd,
	)
	if err != nil {
		return fmt.Errorf("CreateCompany: %w", err)
	}
	return nil
}

func (r *CompanyRepositoryImpl) GetByID(id string) (*models.Company, error) {
	query := `SELECT company_id, name, logo_url, config, default_start_time, default_end_time, work_hours_per_week, lunch_start, lunch_end, created_at, updated_at FROM companies WHERE company_id = $1`
	row := r.db.Pool.QueryRow(context.Background(), query, id)

	var company models.Company
	var logoURL, config, defaultEndTime, lunchStart, lunchEnd *string
	err := row.Scan(&company.CompanyID, &company.Name, &logoURL, &config, &company.DefaultStartTime, &defaultEndTime, &company.WorkHoursPerWeek, &lunchStart, &lunchEnd, &company.CreatedAt, &company.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("GetCompanyByID: %w", err)
	}
	company.LogoURL = logoURL
	company.Config = config
	company.DefaultEndTime = defaultEndTime
	company.LunchStart = lunchStart
	company.LunchEnd = lunchEnd
	return &company, nil
}

func (r *CompanyRepositoryImpl) GetByName(name string) (*models.Company, error) {
	query := `SELECT company_id, name, logo_url, config, default_start_time, default_end_time, work_hours_per_week, lunch_start, lunch_end, created_at, updated_at FROM companies WHERE LOWER(name) = LOWER($1)`
	row := r.db.Pool.QueryRow(context.Background(), query, name)

	var company models.Company
	var logoURL, config, defaultEndTime, lunchStart, lunchEnd *string
	err := row.Scan(&company.CompanyID, &company.Name, &logoURL, &config, &company.DefaultStartTime, &defaultEndTime, &company.WorkHoursPerWeek, &lunchStart, &lunchEnd, &company.CreatedAt, &company.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("GetCompanyByName: %w", err)
	}
	company.LogoURL = logoURL
	company.Config = config
	company.DefaultEndTime = defaultEndTime
	company.LunchStart = lunchStart
	company.LunchEnd = lunchEnd
	return &company, nil
}

func (r *CompanyRepositoryImpl) Update(company *models.Company) error {
	query := `UPDATE companies SET name = $1, logo_url = $2, config = $3, default_start_time = $4, default_end_time = $5, work_hours_per_week = $6, lunch_start = $7, lunch_end = $8, updated_at = NOW() WHERE company_id = $9`
	_, err := r.db.Pool.Exec(context.Background(), query, company.Name, company.LogoURL, company.Config, company.DefaultStartTime, company.DefaultEndTime, company.WorkHoursPerWeek, company.LunchStart, company.LunchEnd, company.CompanyID)
	if err != nil {
		return fmt.Errorf("UpdateCompany: %w", err)
	}
	return nil
}

func (r *CompanyRepositoryImpl) Delete(id string) error {
	query := `DELETE FROM companies WHERE company_id = $1`
	_, err := r.db.Pool.Exec(context.Background(), query, id)
	if err != nil {
		return fmt.Errorf("DeleteCompany: %w", err)
	}
	return nil
}

func (r *CompanyRepositoryImpl) ListAll(limit, offset int) ([]*models.Company, int, error) {
	query := `SELECT company_id, name, logo_url, config, default_start_time, default_end_time, work_hours_per_week, created_at, updated_at
	          FROM companies ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := r.db.Pool.Query(context.Background(), query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("ListAllCompanies: %w", err)
	}
	defer rows.Close()

	var companies []*models.Company
	for rows.Next() {
		var c models.Company
		var logoURL, config, defaultEndTime *string
		if err := rows.Scan(&c.CompanyID, &c.Name, &logoURL, &config, &c.DefaultStartTime, &defaultEndTime, &c.WorkHoursPerWeek, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, 0, err
		}
		c.LogoURL = logoURL
		c.Config = config
		c.DefaultEndTime = defaultEndTime
		companies = append(companies, &c)
	}

	var total int
	countQuery := `SELECT COUNT(*) FROM companies`
	r.db.Pool.QueryRow(context.Background(), countQuery).Scan(&total)

	return companies, total, nil
}