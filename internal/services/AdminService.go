package services

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"turno-papa/internal/models"
	"turno-papa/internal/repository"
)

type AdminService struct {
	companyRepo  repository.CompanyRepository
	userRepo     repository.UserRepository
	branchRepo   repository.BranchRepository
	adminUserRep repository.AdminUserRepository
}

func NewAdminService(
	companyRepo repository.CompanyRepository,
	userRepo repository.UserRepository,
	branchRepo repository.BranchRepository,
	adminUserRep repository.AdminUserRepository,
) *AdminService {
	return &AdminService{
		companyRepo:   companyRepo,
		userRepo:      userRepo,
		branchRepo:    branchRepo,
		adminUserRep: adminUserRep,
	}
}

type CompanyListResult struct {
	Companies []*models.Company `json:"companies"`
	Total     int               `json:"total"`
	Limit     int               `json:"limit"`
	Offset    int               `json:"offset"`
}

func (s *AdminService) ListCompanies(limit, offset int) (*CompanyListResult, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	companies, total, err := s.companyRepo.ListAll(limit, offset)
	if err != nil {
		return nil, err
	}
	return &CompanyListResult{
		Companies: companies,
		Total:    total,
		Limit:    limit,
		Offset:   offset,
	}, nil
}

func (s *AdminService) CreateCompany(req *models.CreateCompanyRequest) (*models.Company, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("company name is required")
	}
	if req.Branch.Name == "" {
		return nil, fmt.Errorf("at least one branch is required during company creation")
	}

	companyID := uuid.New().String()
	configJSON, _ := json.Marshal(req.Config)
	configStr := string(configJSON)

	company := &models.Company{
		CompanyID:         companyID,
		Name:              req.Name,
		LogoURL:           req.LogoURL,
		Config:            &configStr,
		DefaultStartTime:  req.DefaultStartTime,
		DefaultEndTime:    req.DefaultEndTime,
		WorkHoursPerWeek:  req.WorkHoursPerWeek,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	// Set defaults if not provided
	if company.WorkHoursPerWeek == 0 {
		company.WorkHoursPerWeek = 42.0
	}

	if err := s.companyRepo.Create(company); err != nil {
		return nil, fmt.Errorf("create company: %w", err)
	}

	// Create admin user for this company
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.AdminUser.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	adminUser := &models.User{
		UserID:    uuid.New().String(),
		CompanyID: companyID,
		Name:      req.AdminUser.Name,
		Email:     req.AdminUser.Email,
		Password:  string(hashedPassword),
		RUT:       req.AdminUser.RUT,
		Rol:       "admin",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.userRepo.Create(adminUser); err != nil {
		return nil, fmt.Errorf("create admin user: %w", err)
	}

	// Create required branch
	branch := &models.Branch{
		BranchID:  uuid.New().String(),
		CompanyID: companyID,
		Name:      req.Branch.Name,
		Address:   req.Branch.Address,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.branchRepo.Create(branch); err != nil {
		return nil, fmt.Errorf("create branch: %w", err)
	}

	return company, nil
}

func (s *AdminService) UpdateCompany(companyID string, req *models.CreateCompanyRequest) (*models.Company, error) {
	company, err := s.companyRepo.GetByID(companyID)
	if err != nil {
		return nil, fmt.Errorf("company not found: %w", err)
	}

	company.Name = req.Name
	company.LogoURL = req.LogoURL
	configJSON, _ := json.Marshal(req.Config)
	configStr := string(configJSON)
	company.Config = &configStr
	company.DefaultStartTime = req.DefaultStartTime
	if req.DefaultStartTime == "" {
		company.DefaultStartTime = "08:00" // Default fallback
	}
	
	company.DefaultEndTime = req.DefaultEndTime
	if company.DefaultEndTime != nil && *company.DefaultEndTime == "" {
		company.DefaultEndTime = nil
	}

	company.WorkHoursPerWeek = req.WorkHoursPerWeek
	if company.WorkHoursPerWeek == 0 {
		company.WorkHoursPerWeek = 42.0
	}
	company.UpdatedAt = time.Now()

	if err := s.companyRepo.Update(company); err != nil {
		return nil, err
	}

	return company, nil
}

func (s *AdminService) DeleteCompany(companyID string) error {
	return s.companyRepo.Delete(companyID)
}

func (s *AdminService) GetCompanyConfig(companyName string) (*models.Company, *models.CompanyConfig, error) {
	company, err := s.companyRepo.GetByName(companyName)
	if err != nil {
		// Try by ID as fallback
		company, err = s.companyRepo.GetByID(companyName)
		if err != nil {
			// Log the actual error for debugging
			fmt.Printf("GetCompanyConfig: looking for '%s', error: %v\n", companyName, err)
			return nil, nil, err
		}
	}

	var config models.CompanyConfig
	if company.Config != nil && *company.Config != "" {
		json.Unmarshal([]byte(*company.Config), &config)
	}
	return company, &config, nil
}

type AdminLoginResult struct {
	AdminID string `json:"admin_id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Role    string `json:"role"`
	Token   string `json:"token"`
}

func (s *AdminService) AdminLogin(email, password string) (*AdminLoginResult, error) {
	admin, err := s.adminUserRep.GetByEmail(email)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Generate a special JWT for super-admin (not using userRepo)
	// This will be handled in the handler since it needs special claims

	return &AdminLoginResult{
		AdminID: admin.AdminID,
		Email:   admin.Email,
		Name:    admin.Name,
		Role:    "super_admin",
	}, nil
}