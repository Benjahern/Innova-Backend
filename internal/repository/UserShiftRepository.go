package repository

import (
	"context"
	"fmt"

	"turno-papa/internal/db"
	"turno-papa/internal/models"
)

type UserShiftRepository interface {
	Create(userShift *models.UserShift) error
	GetByUser(userID string) ([]*models.UserShift, error)
	GetByShift(shiftID string) ([]*models.UserShift, error)
	Delete(userShiftID string) error
	DeleteByUser(userID string) error
}

type UserShiftRepositoryImpl struct {
	db *db.DB
}

func NewUserShiftRepository(database *db.DB) *UserShiftRepositoryImpl {
	return &UserShiftRepositoryImpl{db: database}
}

func (r *UserShiftRepositoryImpl) Create(userShift *models.UserShift) error {
	query := `
		INSERT INTO user_shifts (user_shift_id, user_id, shift_id, start_date, end_date)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.Pool.Exec(context.Background(), query,
		userShift.UserShiftID, userShift.UserID, userShift.ShiftID,
		userShift.StartDate, userShift.EndDate,
	)
	if err != nil {
		return fmt.Errorf("CreateUserShift: %w", err)
	}
	return nil
}

func (r *UserShiftRepositoryImpl) GetByUser(userID string) ([]*models.UserShift, error) {
	query := `SELECT user_shift_id, user_id, shift_id, start_date, end_date FROM user_shifts WHERE user_id = $1`
	rows, err := r.db.Pool.Query(context.Background(), query, userID)
	if err != nil {
		return nil, fmt.Errorf("GetUserShiftsByUser: %w", err)
	}
	defer rows.Close()

	var userShifts []*models.UserShift
	for rows.Next() {
		var us models.UserShift
		err := rows.Scan(&us.UserShiftID, &us.UserID, &us.ShiftID, &us.StartDate, &us.EndDate)
		if err != nil {
			return nil, fmt.Errorf("GetUserShiftsByUser scan: %w", err)
		}
		userShifts = append(userShifts, &us)
	}

	return userShifts, nil
}

func (r *UserShiftRepositoryImpl) GetByShift(shiftID string) ([]*models.UserShift, error) {
	query := `SELECT user_shift_id, user_id, shift_id, start_date, end_date FROM user_shifts WHERE shift_id = $1`
	rows, err := r.db.Pool.Query(context.Background(), query, shiftID)
	if err != nil {
		return nil, fmt.Errorf("GetUserShiftsByShift: %w", err)
	}
	defer rows.Close()

	var userShifts []*models.UserShift
	for rows.Next() {
		var us models.UserShift
		err := rows.Scan(&us.UserShiftID, &us.UserID, &us.ShiftID, &us.StartDate, &us.EndDate)
		if err != nil {
			return nil, fmt.Errorf("GetUserShiftsByShift scan: %w", err)
		}
		userShifts = append(userShifts, &us)
	}

	return userShifts, nil
}

func (r *UserShiftRepositoryImpl) Delete(userShiftID string) error {
	query := `DELETE FROM user_shifts WHERE user_shift_id = $1`
	_, err := r.db.Pool.Exec(context.Background(), query, userShiftID)
	if err != nil {
		return fmt.Errorf("DeleteUserShift: %w", err)
	}
	return nil
}

func (r *UserShiftRepositoryImpl) DeleteByUser(userID string) error {
	query := `DELETE FROM user_shifts WHERE user_id = $1`
	_, err := r.db.Pool.Exec(context.Background(), query, userID)
	if err != nil {
		return fmt.Errorf("DeleteUserShiftsByUser: %w", err)
	}
	return nil
}