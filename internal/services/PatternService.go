package services

import (
	"fmt"
	"time"

	"turno-papa/internal/models"
	"turno-papa/internal/repository"
)

type PatternService struct {
	patternRepo  repository.PatternRepository
	shiftRepo    repository.WorkShiftRepository
	userShiftRepo repository.UserShiftRepository
}

func NewPatternService(
	patternRepo repository.PatternRepository,
	shiftRepo repository.WorkShiftRepository,
	userShiftRepo repository.UserShiftRepository,
) *PatternService {
	return &PatternService{
		patternRepo:   patternRepo,
		shiftRepo:     shiftRepo,
		userShiftRepo: userShiftRepo,
	}
}

// IsWorkDay returns true if the user should work on the given date
// For rotating shifts: uses (date - patternStart) % cycleLength
// For fixed shifts: uses day-of-week matching
func (s *PatternService) IsWorkDay(userID string, date time.Time) (bool, error) {
	userShifts, err := s.userShiftRepo.GetByUser(userID)
	if err != nil {
		return false, err
	}
	if len(userShifts) == 0 {
		return false, fmt.Errorf("no shift assigned to user")
	}

	// Get the most recent active assignment (with start_date)
	var activeShift *models.UserShift
	for _, us := range userShifts {
		if activeShift == nil {
			activeShift = us
		} else if us.StartDate != nil && activeShift.StartDate != nil {
			if *us.StartDate > *activeShift.StartDate {
				activeShift = us
			}
		}
	}

	if activeShift == nil {
		return false, fmt.Errorf("no active shift found")
	}

	shift, err := s.shiftRepo.GetByID(activeShift.ShiftID)
	if err != nil {
		return false, err
	}

	// If shift has a pattern, use rotation calculation
	if shift.PatternID != nil {
		return s.calculateIsWorkDayWithPattern(shift, activeShift, date)
	}

	// Otherwise use fixed day-of-week logic
	return s.isWorkDayFixed(shift, date), nil
}

// calculateIsWorkDayWithPattern calculates if date is a work day based on rotation pattern
func (s *PatternService) calculateIsWorkDayWithPattern(shift *models.WorkShift, userShift *models.UserShift, date time.Time) (bool, error) {
	if shift.PatternID == nil {
		return s.isWorkDayFixed(shift, date), nil
	}

	pattern, err := s.patternRepo.GetByID(*shift.PatternID)
	if err != nil {
		return false, err
	}

	// Need start_date to calculate cycle
	if userShift.StartDate == nil {
		// If no start_date, default to today as cycle start
		today := time.Now().Format("2006-01-02")
		userShift.StartDate = &today
	}

	patternStart, err := time.Parse("2006-01-02", *userShift.StartDate)
	if err != nil {
		return false, fmt.Errorf("invalid pattern_start_date: %w", err)
	}

	cycleLength := pattern.WorkDays + pattern.OffDays
	if cycleLength == 0 {
		return false, fmt.Errorf("cycle length is zero")
	}

	// Calculate days since pattern start
	daysSinceStart := int(date.Sub(patternStart).Hours() / 24)

	// Handle negative remainders for dates before pattern start
	dayInCycle := daysSinceStart % cycleLength
	if dayInCycle < 0 {
		dayInCycle += cycleLength
	}

	return dayInCycle < pattern.WorkDays, nil
}

// isWorkDayFixed returns true if the shift includes this day of week
func (s *PatternService) isWorkDayFixed(shift *models.WorkShift, date time.Time) bool {
	if len(shift.Days) == 0 {
		return true // No days specified means all days
	}

	weekday := date.Weekday().String()
	// Convert Go weekday (Monday, Tuesday, etc) to lowercase
	weekdayLower := make([]string, len(shift.Days))
	for i, d := range shift.Days {
		weekdayLower[i] = d
	}

	for _, day := range shift.Days {
		if day == weekday {
			return true
		}
	}
	return false
}

// GetUserShiftForDay returns the work shift if user should work on that day, nil if day off
func (s *PatternService) GetUserShiftForDay(userID string, date time.Time) (*models.WorkShift, error) {
	isWork, err := s.IsWorkDay(userID, date)
	if err != nil {
		return nil, err
	}
	if !isWork {
		return nil, nil // User has day off
	}

	userShifts, err := s.userShiftRepo.GetByUser(userID)
	if err != nil {
		return nil, err
	}
	if len(userShifts) == 0 {
		return nil, fmt.Errorf("no shift assigned")
	}

	// Get most recent active shift
	var latest *models.UserShift
	for _, us := range userShifts {
		if latest == nil {
			latest = us
		} else if us.StartDate != nil && latest.StartDate != nil && *us.StartDate > *latest.StartDate {
			latest = us
		}
	}

	return s.shiftRepo.GetByID(latest.ShiftID)
}

// GetShiftStartTime returns the start time for a user's shift on a given date
// Falls back to company default if no shift
func (s *PatternService) GetShiftStartTime(userID string, date time.Time, companyDefaultStart string) (string, error) {
	shift, err := s.GetUserShiftForDay(userID, date)
	if err != nil || shift == nil {
		return companyDefaultStart, nil
	}
	if shift.StartTime != "" {
		return shift.StartTime, nil
	}
	return companyDefaultStart, nil
}

// GetShiftEndTime returns the end time for a user's shift on a given date
func (s *PatternService) GetShiftEndTime(userID string, date time.Time, companyDefaultEnd string) (string, error) {
	shift, err := s.GetUserShiftForDay(userID, date)
	if err != nil || shift == nil {
		return companyDefaultEnd, nil
	}
	if shift.EndTime != "" {
		return shift.EndTime, nil
	}
	return companyDefaultEnd, nil
}