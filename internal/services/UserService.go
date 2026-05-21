package services

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"turno-papa/internal/models"
	"turno-papa/internal/repository"
)

type UserService struct {
	userRepo    repository.UserRepository
	shiftRepo   repository.WorkShiftRepository
	userShiftRepo repository.UserShiftRepository
}

func NewUserService(
	userRepo repository.UserRepository,
	shiftRepo repository.WorkShiftRepository,
	userShiftRepo repository.UserShiftRepository,
) *UserService {
	return &UserService{
		userRepo:    userRepo,
		shiftRepo:   shiftRepo,
		userShiftRepo: userShiftRepo,
	}
}

func (s *UserService) CreateUser(companyID, name, email, password, rut, role, branchID string) (*models.User, error) {
	// Password must be pre-hashed by caller (bcrypt)
	// Valid bcrypt hash: 60 chars, starts with $2a$ or $2b$
	if len(password) != 60 || (password[:4] != "$2a$" && password[:4] != "$2b$") {
		return nil, fmt.Errorf("password must be hashed with bcrypt before calling CreateUser")
	}

	// Check if email exists
	existing, _ := s.userRepo.GetByEmail(email)
	if existing != nil {
		return nil, fmt.Errorf("email already in use")
	}

	user := &models.User{
		UserID:     uuid.New().String(),
		CompanyID:  companyID,
		Name:       name,
		Email:      email,
		Password:   password,
		RUT:        rut,
		Rol:        role,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if branchID != "" {
		user.BranchID = &branchID
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}

func (s *UserService) GetUser(id string) (*models.User, error) {
	return s.userRepo.GetByID(id)
}

func (s *UserService) GetUserByIDAndCompany(id, companyID string) (*models.User, error) {
	return s.userRepo.GetByIDAndCompany(id, companyID)
}

func (s *UserService) GetWorkers(companyID string) ([]*models.User, error) {
	users, err := s.userRepo.GetByCompany(companyID)
	if err != nil {
		return nil, err
	}

	return users, nil
}

func (s *UserService) UpdateUser(user *models.User) error {
	user.UpdatedAt = time.Now()
	return s.userRepo.Update(user)
}

func (s *UserService) DeleteUser(id string) error {
	return s.userRepo.Delete(id)
}

func (s *UserService) AssignShift(userID, shiftID string, startDate *string) error {
	// Verify shift exists
	_, err := s.shiftRepo.GetByID(shiftID)
	if err != nil {
		return fmt.Errorf("shift not found: %w", err)
	}

	userShift := &models.UserShift{
		UserShiftID: uuid.New().String(),
		UserID:      userID,
		ShiftID:     shiftID,
		StartDate:   startDate,
	}

	return s.userShiftRepo.Create(userShift)
}

func (s *UserService) GetUserShifts(userID string) ([]*models.WorkShift, error) {
	userShifts, err := s.userShiftRepo.GetByUser(userID)
	if err != nil {
		return nil, err
	}

	var shifts []*models.WorkShift
	for _, us := range userShifts {
		shift, err := s.shiftRepo.GetByID(us.ShiftID)
		if err == nil {
			shifts = append(shifts, shift)
		}
	}

	return shifts, nil
}

func (s *UserService) RemoveShift(userShiftID string) error {
	return s.userShiftRepo.Delete(userShiftID)
}