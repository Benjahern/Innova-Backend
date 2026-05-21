package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"

	"turno-papa/internal/models"
	"turno-papa/internal/repository"
	"turno-papa/internal/services"
)

type AdminHandler struct {
	adminService *services.AdminService
	companyRepo  repository.CompanyRepository
	jwtSecret    string
}

func NewAdminHandler(adminService *services.AdminService, companyRepo repository.CompanyRepository, jwtSecret string) *AdminHandler {
	return &AdminHandler{adminService: adminService, companyRepo: companyRepo, jwtSecret: jwtSecret}
}

// ListCompanies returns all companies (super-admin only)
// GET /api/v1/admin/companies
func (h *AdminHandler) ListCompanies(c echo.Context) error {
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))

	result, err := h.adminService.ListCompanies(limit, offset)
	if err != nil {
		c.Logger().Errorf("ListCompanies error: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list companies")
	}

	return c.JSON(http.StatusOK, result)
}

// CreateCompany creates a new company with admin user and required branch
// POST /api/v1/admin/companies
func (h *AdminHandler) CreateCompany(c echo.Context) error {
	var req models.CreateCompanyRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	// Validation
	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "company name is required")
	}
	if req.AdminUser.Email == "" || req.AdminUser.Password == "" || req.AdminUser.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "admin user email, password, and name are required")
	}
	if req.Branch.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "at least one branch is required during company creation")
	}

	company, err := h.adminService.CreateCompany(&req)
	if err != nil {
		c.Logger().Errorf("CreateCompany error: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusCreated, company)
}

// UpdateCompany updates an existing company
// PUT /api/v1/admin/companies/:id
func (h *AdminHandler) UpdateCompany(c echo.Context) error {
	companyID := c.Param("id")
	if companyID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "company id is required")
	}

	var req models.CreateCompanyRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	company, err := h.adminService.UpdateCompany(companyID, &req)
	if err != nil {
		c.Logger().Errorf("UpdateCompany error: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update company")
	}

	return c.JSON(http.StatusOK, company)
}

// DeleteCompany removes a company
// DELETE /api/v1/admin/companies/:id
func (h *AdminHandler) DeleteCompany(c echo.Context) error {
	companyID := c.Param("id")
	if companyID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "company id is required")
	}

	if err := h.adminService.DeleteCompany(companyID); err != nil {
		c.Logger().Errorf("DeleteCompany error: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete company")
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "company deleted"})
}

// AdminLogin authenticates super-admin
// POST /api/v1/admin/login
func (h *AdminHandler) AdminLogin(c echo.Context) error {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	result, err := h.adminService.AdminLogin(req.Email, req.Password)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
	}

	// Generate JWT for super-admin with special claims
	claims := jwt.MapClaims{
		"admin_id": result.AdminID,
		"role":     "super_admin",
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(h.jwtSecret))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate token")
	}

	result.Token = tokenString

	return c.JSON(http.StatusOK, result)
}

// GetCompanyConfig returns company config for branded login (public endpoint)
// GET /api/v1/public/companies/:name/config
func (h *AdminHandler) GetCompanyConfig(c echo.Context) error {
	companyName := c.Param("name")
	if companyName == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "company name is required")
	}

	company, config, err := h.adminService.GetCompanyConfig(companyName)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "company not found")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"company": company,
		"config":  config,
	})
}

// UploadCompanyLogo handles logo file upload
// POST /api/v1/admin/companies/:id/logo
func (h *AdminHandler) UploadCompanyLogo(c echo.Context) error {
	companyID := c.Param("id")
	if companyID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "company id is required")
	}

	company, err := h.companyRepo.GetByID(companyID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "company not found")
	}

	logoURL, err := uploadFile(c, "logo", "logos")
	if err != nil {
		c.Logger().Errorf("UploadCompanyLogo error: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	company.LogoURL = &logoURL
	if err := h.companyRepo.Update(company); err != nil {
		c.Logger().Errorf("Update company logo error: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update logo")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":  "logo uploaded",
		"logo_url": logoURL,
	})
}