package repository

import (
	"context"
	"fmt"

	"turno-papa/internal/db"
	"turno-papa/internal/models"
)

type UserRepository interface {
	Create(user *models.User) error
	GetByID(id string) (*models.User, error)
	GetByIDAndCompany(id, companyID string) (*models.User, error)
	GetByEmail(email string) (*models.User, error)
	GetByCompany(companyID string) ([]*models.User, error)
	Update(user *models.User) error
	Delete(id string) error
}

type UserRepositoryImpl struct {
	db *db.DB
}

func NewUserRepository(database *db.DB) *UserRepositoryImpl {
	return &UserRepositoryImpl{db: database}
}

func (r *UserRepositoryImpl) Create(user *models.User) error {
	query := `
		INSERT INTO users (user_id, company_id, branch_id, name, email, password, rut, rol, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
	`
	var branchID interface{}
	if user.BranchID != nil && *user.BranchID != "" {
		branchID = *user.BranchID
	} else {
		branchID = nil
	}
	_, err := r.db.Pool.Exec(context.Background(), query,
		user.UserID, user.CompanyID, branchID, user.Name,
		user.Email, user.Password, user.RUT, user.Rol,
	)
	if err != nil {
		return fmt.Errorf("CreateUser: %w", err)
	}
	return nil
}

func (r *UserRepositoryImpl) GetByID(id string) (*models.User, error) {
	query := `SELECT user_id, company_id, branch_id, name, email, password, rut, rol, worker_type, created_at, updated_at FROM users WHERE user_id = $1`
	row := r.db.Pool.QueryRow(context.Background(), query, id)

	var user models.User
	var branchID *string
	var workerType *string
	err := row.Scan(&user.UserID, &user.CompanyID, &branchID, &user.Name, &user.Email, &user.Password, &user.RUT, &user.Rol, &workerType, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("GetByID: %w", err)
	}
	user.BranchID = branchID
	if workerType != nil {
		user.WorkerType = *workerType
	}
	return &user, nil
}

func (r *UserRepositoryImpl) GetByIDAndCompany(id, companyID string) (*models.User, error) {
	query := `SELECT user_id, company_id, branch_id, name, email, password, rut, rol, worker_type, created_at, updated_at FROM users WHERE user_id = $1 AND company_id = $2`
	row := r.db.Pool.QueryRow(context.Background(), query, id, companyID)

	var user models.User
	var branchID *string
	var workerType *string
	err := row.Scan(&user.UserID, &user.CompanyID, &branchID, &user.Name, &user.Email, &user.Password, &user.RUT, &user.Rol, &workerType, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("GetByIDAndCompany: %w", err)
	}
	user.BranchID = branchID
	if workerType != nil {
		user.WorkerType = *workerType
	}
	return &user, nil
}

func (r *UserRepositoryImpl) GetByEmail(email string) (*models.User, error) {
	query := `SELECT user_id, company_id, branch_id, name, email, password, rut, rol, worker_type, created_at, updated_at FROM users WHERE email = $1`
	row := r.db.Pool.QueryRow(context.Background(), query, email)

	var user models.User
	var branchID *string
	var workerType *string
	err := row.Scan(&user.UserID, &user.CompanyID, &branchID, &user.Name, &user.Email, &user.Password, &user.RUT, &user.Rol, &workerType, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("GetByEmail: %w", err)
	}
	user.BranchID = branchID
	if workerType != nil {
		user.WorkerType = *workerType
	}
	return &user, nil
}

func (r *UserRepositoryImpl) GetByCompany(companyID string) ([]*models.User, error) {
	query := `
		SELECT u.user_id, u.company_id, u.branch_id, u.name, u.email, u.password, u.rut, u.rol, u.worker_type, u.created_at, u.updated_at,
			   s.shift_id, s.name as shift_name, s.start_time as shift_start, s.end_time as shift_end
		FROM users u
		LEFT JOIN user_shifts us ON u.user_id = us.user_id
		LEFT JOIN work_shifts s ON us.shift_id = s.shift_id
		WHERE u.company_id = $1 AND u.deleted_at IS NULL
	`
	rows, err := r.db.Pool.Query(context.Background(), query, companyID)
	if err != nil {
		return nil, fmt.Errorf("GetByCompany: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var user models.User
		var branchID, shiftID, shiftName, shiftStart, shiftEnd *string
		var workerType *string
		err := rows.Scan(&user.UserID, &user.CompanyID, &branchID, &user.Name, &user.Email, &user.Password, &user.RUT, &user.Rol, &workerType, &user.CreatedAt, &user.UpdatedAt,
			&shiftID, &shiftName, &shiftStart, &shiftEnd)
		if err != nil {
			return nil, fmt.Errorf("GetByCompany scan: %w", err)
		}
		user.BranchID = branchID
		if workerType != nil {
			user.WorkerType = *workerType
		}
		// Attach shift info as extended fields (for API response)
		if shiftID != nil {
			user.ShiftID = shiftID
			user.ShiftName = shiftName
			user.ShiftStart = shiftStart
			user.ShiftEnd = shiftEnd
		}
		users = append(users, &user)
	}

	return users, nil
}

func (r *UserRepositoryImpl) Update(user *models.User) error {
	query := `UPDATE users SET name = $1, email = $2, rut = $3, rol = $4, branch_id = $5, worker_type = $6, updated_at = NOW() WHERE user_id = $7`
	_, err := r.db.Pool.Exec(context.Background(), query, user.Name, user.Email, user.RUT, user.Rol, user.BranchID, user.WorkerType, user.UserID)
	if err != nil {
		return fmt.Errorf("UpdateUser: %w", err)
	}
	return nil
}

func (r *UserRepositoryImpl) Delete(id string) error {
	query := `DELETE FROM users WHERE user_id = $1`
	_, err := r.db.Pool.Exec(context.Background(), query, id)
	if err != nil {
		return fmt.Errorf("DeleteUser: %w", err)
	}
	return nil
}