package repository

import (
	"context"
	"fmt"

	"turno-papa/internal/db"
	"turno-papa/internal/models"
)

type AdminUserRepository interface {
	Create(admin *models.AdminUser) error
	GetByEmail(email string) (*models.AdminUser, error)
	GetByID(id string) (*models.AdminUser, error)
}

type AdminUserRepositoryImpl struct {
	db *db.DB
}

func NewAdminUserRepository(database *db.DB) *AdminUserRepositoryImpl {
	return &AdminUserRepositoryImpl{db: database}
}

func (r *AdminUserRepositoryImpl) Create(admin *models.AdminUser) error {
	query := `
		INSERT INTO admin_users (admin_id, email, password, name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
	`
	_, err := r.db.Pool.Exec(context.Background(), query, admin.AdminID, admin.Email, admin.Password, admin.Name)
	if err != nil {
		return fmt.Errorf("CreateAdminUser: %w", err)
	}
	return nil
}

func (r *AdminUserRepositoryImpl) GetByEmail(email string) (*models.AdminUser, error) {
	query := `SELECT admin_id, email, password, name, created_at, updated_at FROM admin_users WHERE email = $1`
	row := r.db.Pool.QueryRow(context.Background(), query, email)

	var admin models.AdminUser
	err := row.Scan(&admin.AdminID, &admin.Email, &admin.Password, &admin.Name, &admin.CreatedAt, &admin.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("GetAdminUserByEmail: %w", err)
	}
	return &admin, nil
}

func (r *AdminUserRepositoryImpl) GetByID(id string) (*models.AdminUser, error) {
	query := `SELECT admin_id, email, password, name, created_at, updated_at FROM admin_users WHERE admin_id = $1`
	row := r.db.Pool.QueryRow(context.Background(), query, id)

	var admin models.AdminUser
	err := row.Scan(&admin.AdminID, &admin.Email, &admin.Password, &admin.Name, &admin.CreatedAt, &admin.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("GetAdminUserByID: %w", err)
	}
	return &admin, nil
}