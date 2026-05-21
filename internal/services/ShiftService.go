package services

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"turno-papa/internal/models"
	"turno-papa/internal/repository"
)

type ShiftService struct {
	shiftRepo repository.WorkShiftRepository
	userShiftRepo repository.UserShiftRepository
}

func NewShiftService(
	shiftRepo repository.WorkShiftRepository,
	userShiftRepo repository.UserShiftRepository,
) *ShiftService {
	return &ShiftService{
		shiftRepo:    shiftRepo,
		userShiftRepo: userShiftRepo,
	}
}

func (s *ShiftService) CreateShift(companyID, name string, days []string, startTime, endTime string) (*models.WorkShift, error) {
	shift := &models.WorkShift{
		ShiftID:   uuid.New().String(),
		CompanyID: companyID,
		Name:      name,
		Days:      days,
		StartTime: startTime,
		EndTime:   endTime,
	}

	if err := s.shiftRepo.Create(shift); err != nil {
		return nil, fmt.Errorf("create shift: %w", err)
	}

	return shift, nil
}

func (s *ShiftService) GetShift(id string) (*models.WorkShift, error) {
	return s.shiftRepo.GetByID(id)
}

func (s *ShiftService) GetByCompany(companyID string) ([]*models.WorkShift, error) {
	return s.shiftRepo.GetByCompany(companyID)
}

func (s *ShiftService) UpdateShift(shift *models.WorkShift) error {
	return s.shiftRepo.Update(shift)
}

func (s *ShiftService) DeleteShift(id string) error {
	// Remove all user assignments first
	if err := s.userShiftRepo.DeleteByUser(id); err != nil {
		return fmt.Errorf("delete user shifts: %w", err)
	}
	return s.shiftRepo.Delete(id)
}

func (s *ShiftService) AssignToUser(userID, shiftID string) error {
	userShift := &models.UserShift{
		UserShiftID: uuid.New().String(),
		UserID:      userID,
		ShiftID:     shiftID,
	}
	return s.userShiftRepo.Create(userShift)
}

func (s *ShiftService) GetUsersByShift(shiftID string) ([]*models.UserShift, error) {
	return s.userShiftRepo.GetByShift(shiftID)
}

func (s *ShiftService) RemoveUserFromShift(userShiftID string) error {
	return s.userShiftRepo.Delete(userShiftID)
}

// GetUserShiftForDay returns the shift assignment for a user on a specific day
func (s *ShiftService) GetUserShiftForDay(userID string, date time.Time) (*models.WorkShift, error) {
	userShifts, err := s.userShiftRepo.GetByUser(userID)
	if err != nil {
		return nil, err
	}

	if len(userShifts) == 0 {
		return nil, fmt.Errorf("no shift assigned to user")
	}

	shift, err := s.shiftRepo.GetByID(userShifts[0].ShiftID)
	if err != nil {
		return nil, err
	}

	// Check if the day is in the shift's days
	weekday := date.Weekday().String()
	for _, day := range shift.Days {
		if day == weekday {
			return shift, nil
		}
	}

	return nil, fmt.Errorf("user does not work on this day")
}