package services

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"turno-papa/internal/models"
	"turno-papa/internal/repository"
)

type AttendanceService struct {
	attendanceRepo repository.AttendanceLogRepository
	companyRepo    repository.CompanyRepository
	userRepo       repository.UserRepository
	patternService *PatternService  // NEW: to get shift-specific times
}

func NewAttendanceService(
	attendanceRepo repository.AttendanceLogRepository,
	companyRepo repository.CompanyRepository,
	userRepo repository.UserRepository,
	patternService *PatternService,  // NEW
) *AttendanceService {
	return &AttendanceService{
		attendanceRepo: attendanceRepo,
		companyRepo:    companyRepo,
		userRepo:       userRepo,
		patternService: patternService,  // NEW
	}
}

type AttendanceResult struct {
	LogID     string `json:"log_id"`
	UserID    string `json:"user_id"`
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
	IsLate    int    `json:"is_late"` // minutes late (0 if early or checkout)
}

// RecordAttendance records checkin or checkout. If type is empty, auto-detects.
// Auto-detect logic: if no checkin today → checkin, if checkin without checkout → checkout, else checkin
func (s *AttendanceService) RecordAttendance(userID, companyID, branchID, source string, attendanceType string) (*AttendanceResult, error) {
	log.Printf("=== RecordAttendance START ===")
	log.Printf("Input: userID=%s, companyID=%s, branchID=%s, source=%s, type=%s", userID, companyID, branchID, source, attendanceType)

	now := time.Now()
	today := now.Format("2006-01-02")
	todayStart := today + "T00:00:00Z"
	todayEnd := today + "T23:59:59Z"
	log.Printf("Today range: %s to %s", todayStart, todayEnd)

	// Get today's logs for this user
	todayLogs, err := s.attendanceRepo.GetByUserAndDateRange(userID, todayStart, todayEnd)
	if err != nil {
		log.Printf("ERROR get today logs: %v", err)
		return nil, fmt.Errorf("get today logs: %w", err)
	}
	log.Printf("Found %d logs for today", len(todayLogs))

	// Find last checkin and checkout today
	var lastCheckinToday, lastCheckoutToday *models.AttendanceLog
	for _, log := range todayLogs {
		if log.Type == "checkin" {
			lastCheckinToday = log
		} else if log.Type == "checkout" {
			lastCheckoutToday = log
		}
	}

	// Determine type: if type provided, use it; otherwise auto-detect
	if attendanceType == "" {
		if lastCheckinToday == nil {
			attendanceType = "checkin"
		} else if lastCheckinToday != nil && lastCheckoutToday == nil {
			attendanceType = "checkout"
		} else {
			attendanceType = "checkin"
		}
	}
	log.Printf("Using attendance type: %s (lastCheckin=%v, lastCheckout=%v)", attendanceType, lastCheckinToday != nil, lastCheckoutToday != nil)

	// Check if late (only for checkin, and only for fixed workers)
	lateMinutes := 0
	if attendanceType == "checkin" {
		// Get user to check worker type
		user, err := s.userRepo.GetByID(userID)
		if err == nil && user.WorkerType != "flexible" && user.WorkerType != "external" {
			lateInfo, err := s.GetLateInfo(companyID, userID, now)
			if err != nil {
				log.Printf("warning: GetLateInfo failed: %v", err)
			} else if lateInfo.IsLate {
				lateMinutes = lateInfo.LateMinutes
			}
		}
		// For flexible/external workers, lateMinutes stays 0
	}

	// Create attendance log
	logEntry := &models.AttendanceLog{
		LogID:     uuid.New().String(),
		UserID:    userID,
		BranchID:  branchID,
		Type:      attendanceType,
		Timestamp: now,
		IsLate:    lateMinutes,
		Source:    source,
	}
	log.Printf("About to create attendance log: %+v", logEntry)

	if err := s.attendanceRepo.Create(logEntry); err != nil {
		log.Printf("ERROR creating attendance log: %v", err)
		return nil, fmt.Errorf("create attendance log: %w", err)
	}
	log.Printf("Successfully created attendance log with ID: %s", logEntry.LogID)

	return &AttendanceResult{
		LogID:     logEntry.LogID,
		UserID:    logEntry.UserID,
		Type:      logEntry.Type,
		Timestamp: logEntry.Timestamp.Format(time.RFC3339),
		IsLate:    logEntry.IsLate,
	}, nil
}

// LateInfo contains information about how late/early a user arrived or left
type LateInfo struct {
	IsLate         bool    `json:"is_late"`
	LateMinutes    int     `json:"late_minutes"`    // minutes late (0 if early)
	EarlyMinutes   int     `json:"early_minutes"`   // minutes early (0 if late)
	ExpectedEndTime string `json:"expected_end_time"` // HH:MM format
}

// IsLate checks if user is late based on company's default start time (or shift start time)
func (s *AttendanceService) IsLate(companyID, userID string, timestamp time.Time) (bool, error) {
	info, err := s.GetLateInfo(companyID, userID, timestamp)
	return info.IsLate, err
}

// GetLateInfo returns detailed lateness information
func (s *AttendanceService) GetLateInfo(companyID, userID string, timestamp time.Time) (*LateInfo, error) {
	company, err := s.companyRepo.GetByID(companyID)
	if err != nil {
		return nil, fmt.Errorf("get company: %w", err)
	}

	result := &LateInfo{}

	// NEW: Try to get shift-specific start time via PatternService
	startTimeStr := company.DefaultStartTime
	if s.patternService != nil && userID != "" {
		shiftStart, err := s.patternService.GetShiftStartTime(userID, timestamp, company.DefaultStartTime)
		if err == nil && shiftStart != "" {
			startTimeStr = shiftStart
		}
	}

	// Calculate expected end time from shift or company default
	expectedEndTime := ""
	if company.DefaultEndTime != nil && *company.DefaultEndTime != "" {
		expectedEndTime = *company.DefaultEndTime
	} else if startTimeStr != "" {
		// Default to start + 8 hours
		parsedStart, err := time.Parse("15:04", startTimeStr)
		if err == nil {
			endTime := parsedStart.Add(8 * time.Hour)
			expectedEndTime = endTime.Format("15:04")
		}
	}
	result.ExpectedEndTime = expectedEndTime

	if startTimeStr == "" {
		return result, nil // No start time configured
	}

	// Handle both HH:mm and HH:mm:ss formats
	if len(startTimeStr) > 5 {
		startTimeStr = startTimeStr[:5]
	}
	startTime, err := time.Parse("15:04", startTimeStr)
	if err != nil {
		return nil, fmt.Errorf("parse start time: %w", err)
	}

	userStart := time.Date(
		timestamp.Year(), timestamp.Month(), timestamp.Day(),
		startTime.Hour(), startTime.Minute(), 0, 0,
		timestamp.Location(),
	)

	if timestamp.After(userStart) {
		result.IsLate = true
		result.LateMinutes = int(timestamp.Sub(userStart).Minutes())
	} else {
		result.EarlyMinutes = int(userStart.Sub(timestamp).Minutes())
	}

	return result, nil
}

// GetUserAttendance returns all attendance logs for a user
func (s *AttendanceService) GetUserAttendance(userID string) ([]*models.AttendanceLog, error) {
	return s.attendanceRepo.GetByUser(userID)
}

// GetUserAttendanceByDateRange returns attendance logs for a user in a date range
func (s *AttendanceService) GetUserAttendanceByDateRange(userID, startDate, endDate string) ([]*models.AttendanceLog, error) {
	return s.attendanceRepo.GetByUserAndDateRange(userID, startDate, endDate)
}

// GetCompanyAttendance returns paginated attendance logs for a company
func (s *AttendanceService) GetCompanyAttendance(companyID string, limit, offset int) ([]*models.AttendanceLog, int, error) {
	return s.attendanceRepo.GetByCompany(companyID, limit, offset)
}

// GetBranchAttendance returns paginated attendance logs for a branch (manager view)
func (s *AttendanceService) GetBranchAttendance(companyID, branchID string, limit, offset int) ([]*models.AttendanceLog, int, error) {
	return s.attendanceRepo.GetByBranch(companyID, branchID, limit, offset)
}

// GetUserAttendancePaginated returns paginated attendance logs for a user
func (s *AttendanceService) GetUserAttendancePaginated(userID string, limit, offset int) ([]*models.AttendanceLog, int, error) {
	return s.attendanceRepo.GetByUserPaginated(userID, limit, offset)
}

// GetCompanyByID returns company by ID
func (s *AttendanceService) GetCompanyByID(companyID string) (*models.Company, error) {
	return s.companyRepo.GetByID(companyID)
}