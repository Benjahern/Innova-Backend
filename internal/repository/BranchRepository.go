package repository

import (
	"context"
	"fmt"

	"turno-papa/internal/db"
	"turno-papa/internal/models"
)

type BranchRepository interface {
	Create(branch *models.Branch) error
	GetByID(id string) (*models.Branch, error)
	GetByCompany(companyID string) ([]*models.Branch, error)
	Update(branch *models.Branch) error
	Delete(id string) error
}

type BranchRepositoryImpl struct {
	db *db.DB
}

func NewBranchRepository(database *db.DB) *BranchRepositoryImpl {
	return &BranchRepositoryImpl{db: database}
}

func (r *BranchRepositoryImpl) Create(branch *models.Branch) error {
	query := `
		INSERT INTO branches (branch_id, company_id, name, address, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
	`
	_, err := r.db.Pool.Exec(context.Background(), query,
		branch.BranchID, branch.CompanyID, branch.Name, branch.Address,
	)
	if err != nil {
		return fmt.Errorf("CreateBranch: %w", err)
	}
	return nil
}

func (r *BranchRepositoryImpl) GetByID(id string) (*models.Branch, error) {
	query := `SELECT branch_id, company_id, name, address, created_at, updated_at FROM branches WHERE branch_id = $1`
	row := r.db.Pool.QueryRow(context.Background(), query, id)

	var branch models.Branch
	err := row.Scan(&branch.BranchID, &branch.CompanyID, &branch.Name, &branch.Address, &branch.CreatedAt, &branch.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("GetBranchByID: %w", err)
	}
	return &branch, nil
}

func (r *BranchRepositoryImpl) GetByCompany(companyID string) ([]*models.Branch, error) {
	query := `SELECT branch_id, company_id, name, address, created_at, updated_at FROM branches WHERE company_id = $1`
	rows, err := r.db.Pool.Query(context.Background(), query, companyID)
	if err != nil {
		return nil, fmt.Errorf("GetBranchesByCompany: %w", err)
	}
	defer rows.Close()

	var branches []*models.Branch
	for rows.Next() {
		var branch models.Branch
		err := rows.Scan(&branch.BranchID, &branch.CompanyID, &branch.Name, &branch.Address, &branch.CreatedAt, &branch.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("GetBranchesByCompany scan: %w", err)
		}
		branches = append(branches, &branch)
	}

	return branches, nil
}

func (r *BranchRepositoryImpl) Update(branch *models.Branch) error {
	query := `UPDATE branches SET name = $1, address = $2, updated_at = NOW() WHERE branch_id = $3`
	_, err := r.db.Pool.Exec(context.Background(), query, branch.Name, branch.Address, branch.BranchID)
	if err != nil {
		return fmt.Errorf("UpdateBranch: %w", err)
	}
	return nil
}

func (r *BranchRepositoryImpl) Delete(id string) error {
	query := `DELETE FROM branches WHERE branch_id = $1`
	_, err := r.db.Pool.Exec(context.Background(), query, id)
	if err != nil {
		return fmt.Errorf("DeleteBranch: %w", err)
	}
	return nil
}