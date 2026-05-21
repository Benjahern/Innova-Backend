package repository

import (
	"context"
	"fmt"

	"turno-papa/internal/db"
	"turno-papa/internal/models"
)

type AttendanceLogRepository interface {
	Create(log *models.AttendanceLog) error
	GetByID(id string) (*models.AttendanceLog, error)
	GetByUser(userID string) ([]*models.AttendanceLog, error)
	GetByUserAndDateRange(userID, startDate, endDate string) ([]*models.AttendanceLog, error)
	GetByBranchAndDate(branchID, date string) ([]*models.AttendanceLog, error)
	GetByCompany(companyID string, limit, offset int) ([]*models.AttendanceLog, int, error)
	GetByBranch(companyID, branchID string, limit, offset int) ([]*models.AttendanceLog, int, error)
	GetByUserPaginated(userID string, limit, offset int) ([]*models.AttendanceLog, int, error)
	GetLastCheckin(userID string) (*models.AttendanceLog, error)
	Delete(id string) error
}

type AttendanceLogRepositoryImpl struct {
	db *db.DB
}

func NewAttendanceLogRepository(database *db.DB) *AttendanceLogRepositoryImpl {
	return &AttendanceLogRepositoryImpl{db: database}
}

func (r *AttendanceLogRepositoryImpl) Create(log *models.AttendanceLog) error {
	query := `
		INSERT INTO attendance_logs (log_id, user_id, branch_id, type, timestamp, is_late, source)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.Pool.Exec(context.Background(), query,
		log.LogID, log.UserID, log.BranchID, log.Type, log.Timestamp, log.IsLate, log.Source,
	)
	if err != nil {
		return fmt.Errorf("CreateAttendanceLog: %w", err)
	}
	return nil
}

func (r *AttendanceLogRepositoryImpl) GetByID(id string) (*models.AttendanceLog, error) {
	query := `SELECT log_id, user_id, branch_id, type, timestamp, is_late, source FROM attendance_logs WHERE log_id = $1`
	row := r.db.Pool.QueryRow(context.Background(), query, id)

	var log models.AttendanceLog
	err := row.Scan(&log.LogID, &log.UserID, &log.BranchID, &log.Type, &log.Timestamp, &log.IsLate, &log.Source)
	if err != nil {
		return nil, fmt.Errorf("GetAttendanceLogByID: %w", err)
	}
	return &log, nil
}

func (r *AttendanceLogRepositoryImpl) GetByUser(userID string) ([]*models.AttendanceLog, error) {
	query := `SELECT log_id, user_id, branch_id, type, timestamp, is_late, source FROM attendance_logs WHERE user_id = $1 ORDER BY timestamp DESC`
	rows, err := r.db.Pool.Query(context.Background(), query, userID)
	if err != nil {
		return nil, fmt.Errorf("GetAttendanceLogsByUser: %w", err)
	}
	defer rows.Close()

	var logs []*models.AttendanceLog
	for rows.Next() {
		var log models.AttendanceLog
		err := rows.Scan(&log.LogID, &log.UserID, &log.BranchID, &log.Type, &log.Timestamp, &log.IsLate, &log.Source)
		if err != nil {
			return nil, fmt.Errorf("GetAttendanceLogsByUser scan: %w", err)
		}
		logs = append(logs, &log)
	}

	return logs, nil
}

func (r *AttendanceLogRepositoryImpl) GetByUserAndDateRange(userID, startDate, endDate string) ([]*models.AttendanceLog, error) {
	query := `SELECT log_id, user_id, branch_id, type, timestamp, is_late, source FROM attendance_logs WHERE user_id = $1 AND timestamp >= $2 AND timestamp <= $3 ORDER BY timestamp`
	rows, err := r.db.Pool.Query(context.Background(), query, userID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("GetAttendanceLogsByUserAndDateRange: %w", err)
	}
	defer rows.Close()

	var logs []*models.AttendanceLog
	for rows.Next() {
		var log models.AttendanceLog
		err := rows.Scan(&log.LogID, &log.UserID, &log.BranchID, &log.Type, &log.Timestamp, &log.IsLate, &log.Source)
		if err != nil {
			return nil, fmt.Errorf("GetAttendanceLogsByUserAndDateRange scan: %w", err)
		}
		logs = append(logs, &log)
	}

	return logs, nil
}

func (r *AttendanceLogRepositoryImpl) GetByBranchAndDate(branchID, date string) ([]*models.AttendanceLog, error) {
	query := `SELECT log_id, user_id, branch_id, type, timestamp, is_late, source FROM attendance_logs WHERE branch_id = $1 AND DATE(timestamp) = $2 ORDER BY timestamp`
	rows, err := r.db.Pool.Query(context.Background(), query, branchID, date)
	if err != nil {
		return nil, fmt.Errorf("GetAttendanceLogsByBranchAndDate: %w", err)
	}
	defer rows.Close()

	var logs []*models.AttendanceLog
	for rows.Next() {
		var log models.AttendanceLog
		err := rows.Scan(&log.LogID, &log.UserID, &log.BranchID, &log.Type, &log.Timestamp, &log.IsLate, &log.Source)
		if err != nil {
			return nil, fmt.Errorf("GetAttendanceLogsByBranchAndDate scan: %w", err)
		}
		logs = append(logs, &log)
	}

	return logs, nil
}

func (r *AttendanceLogRepositoryImpl) GetByCompany(companyID string, limit, offset int) ([]*models.AttendanceLog, int, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	if offset < 0 {
		offset = 0
	}

	countQuery := `SELECT COUNT(*) FROM attendance_logs al JOIN users u ON al.user_id = u.user_id WHERE u.company_id = $1`
	var total int
	r.db.Pool.QueryRow(context.Background(), countQuery, companyID).Scan(&total)

	query := `
		SELECT al.log_id, al.user_id, al.branch_id, al.type, al.timestamp, al.is_late, al.source
		FROM attendance_logs al
		JOIN users u ON al.user_id = u.user_id
		WHERE u.company_id = $1
		ORDER BY al.timestamp DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Pool.Query(context.Background(), query, companyID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("GetAttendanceLogsByCompany: %w", err)
	}
	defer rows.Close()

	var logs []*models.AttendanceLog
	for rows.Next() {
		var log models.AttendanceLog
		err := rows.Scan(&log.LogID, &log.UserID, &log.BranchID, &log.Type, &log.Timestamp, &log.IsLate, &log.Source)
		if err != nil {
			return nil, 0, fmt.Errorf("GetAttendanceLogsByCompany scan: %w", err)
		}
		logs = append(logs, &log)
	}

	return logs, total, nil
}

func (r *AttendanceLogRepositoryImpl) GetByBranch(companyID, branchID string, limit, offset int) ([]*models.AttendanceLog, int, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	if offset < 0 {
		offset = 0
	}

	countQuery := `SELECT COUNT(*) FROM attendance_logs al JOIN users u ON al.user_id = u.user_id WHERE u.company_id = $1 AND u.branch_id = $2`
	var total int
	r.db.Pool.QueryRow(context.Background(), countQuery, companyID, branchID).Scan(&total)

	query := `
		SELECT al.log_id, al.user_id, al.branch_id, al.type, al.timestamp, al.is_late, al.source
		FROM attendance_logs al
		JOIN users u ON al.user_id = u.user_id
		WHERE u.company_id = $1 AND u.branch_id = $2
		ORDER BY al.timestamp DESC
		LIMIT $3 OFFSET $4
	`
	rows, err := r.db.Pool.Query(context.Background(), query, companyID, branchID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("GetAttendanceLogsByBranch: %w", err)
	}
	defer rows.Close()

	var logs []*models.AttendanceLog
	for rows.Next() {
		var log models.AttendanceLog
		err := rows.Scan(&log.LogID, &log.UserID, &log.BranchID, &log.Type, &log.Timestamp, &log.IsLate, &log.Source)
		if err != nil {
			return nil, 0, fmt.Errorf("GetAttendanceLogsByBranch scan: %w", err)
		}
		logs = append(logs, &log)
	}

	return logs, total, nil
}

func (r *AttendanceLogRepositoryImpl) GetByUserPaginated(userID string, limit, offset int) ([]*models.AttendanceLog, int, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	if offset < 0 {
		offset = 0
	}

	var total int
	countQuery := `SELECT COUNT(*) FROM attendance_logs WHERE user_id = $1`
	r.db.Pool.QueryRow(context.Background(), countQuery, userID).Scan(&total)

	query := `SELECT log_id, user_id, branch_id, type, timestamp, is_late, source FROM attendance_logs WHERE user_id = $1 ORDER BY timestamp DESC LIMIT $2 OFFSET $3`
	rows, err := r.db.Pool.Query(context.Background(), query, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("GetAttendanceLogsByUserPaginated: %w", err)
	}
	defer rows.Close()

	var logs []*models.AttendanceLog
	for rows.Next() {
		var log models.AttendanceLog
		err := rows.Scan(&log.LogID, &log.UserID, &log.BranchID, &log.Type, &log.Timestamp, &log.IsLate, &log.Source)
		if err != nil {
			return nil, 0, fmt.Errorf("GetAttendanceLogsByUserPaginated scan: %w", err)
		}
		logs = append(logs, &log)
	}

	return logs, total, nil
}

func (r *AttendanceLogRepositoryImpl) GetLastCheckin(userID string) (*models.AttendanceLog, error) {
	query := `SELECT log_id, user_id, branch_id, type, timestamp, is_late, source FROM attendance_logs WHERE user_id = $1 AND type = 'checkin' ORDER BY timestamp DESC LIMIT 1`
	row := r.db.Pool.QueryRow(context.Background(), query, userID)

	var log models.AttendanceLog
	err := row.Scan(&log.LogID, &log.UserID, &log.BranchID, &log.Type, &log.Timestamp, &log.IsLate, &log.Source)
	if err != nil {
		return nil, fmt.Errorf("GetLastCheckin: %w", err)
	}
	return &log, nil
}

func (r *AttendanceLogRepositoryImpl) Delete(id string) error {
	query := `DELETE FROM attendance_logs WHERE log_id = $1`
	_, err := r.db.Pool.Exec(context.Background(), query, id)
	if err != nil {
		return fmt.Errorf("DeleteAttendanceLog: %w", err)
	}
	return nil
}