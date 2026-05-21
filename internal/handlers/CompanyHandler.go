package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"turno-papa/internal/models"
	"turno-papa/internal/repository"
)

type CompanyHandler struct {
	companyRepo repository.CompanyRepository
}

func NewCompanyHandler(companyRepo repository.CompanyRepository) *CompanyHandler {
	return &CompanyHandler{companyRepo: companyRepo}
}

// GetCompany returns the company info for the current user
// GET /api/v1/company
func (h *CompanyHandler) GetCompany(c echo.Context) error {
	role := c.Get("role").(string)
	if role != "admin" && role != "manager" {
		return echo.NewHTTPError(http.StatusForbidden, "only admins and managers can view company")
	}

	companyID := c.Get("company_id").(string)
	company, err := h.companyRepo.GetByID(companyID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "company not found")
	}

	return c.JSON(http.StatusOK, company)
}

// UpdateCompany updates the company settings
// PUT /api/v1/company
func (h *CompanyHandler) UpdateCompany(c echo.Context) error {
	role := c.Get("role").(string)
	if role != "admin" && role != "manager" {
		return echo.NewHTTPError(http.StatusForbidden, "only admins and managers can update company")
	}

	companyID := c.Get("company_id").(string)
	company, err := h.companyRepo.GetByID(companyID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "company not found")
	}

	var req struct {
		DefaultStartTime      string  `json:"default_start_time"`
		DefaultEndTime        *string `json:"default_end_time"`
		WorkHoursPerWeek     float64 `json:"work_hours_per_week"`
		LunchStart           *string `json:"lunch_start"`
		LunchEnd             *string `json:"lunch_end"`
		LogoURL              *string `json:"logo_url"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	if req.DefaultStartTime != "" {
		company.DefaultStartTime = req.DefaultStartTime
	}
	if req.DefaultEndTime != nil {
		company.DefaultEndTime = req.DefaultEndTime
	}
	if req.WorkHoursPerWeek > 0 {
		company.WorkHoursPerWeek = req.WorkHoursPerWeek
	}
	if req.LunchStart != nil {
		company.LunchStart = req.LunchStart
	}
	if req.LunchEnd != nil {
		company.LunchEnd = req.LunchEnd
	}
	if req.LogoURL != nil {
		company.LogoURL = req.LogoURL
	}

	if err := h.companyRepo.Update(company); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update company")
	}

	return c.JSON(http.StatusOK, company)
}

// UploadLogo uploads a logo for the company
// POST /api/v1/company/logo
func (h *CompanyHandler) UploadLogo(c echo.Context) error {
	role := c.Get("role").(string)
	if role != "admin" && role != "manager" {
		return echo.NewHTTPError(http.StatusForbidden, "only admins and managers can upload logo")
	}

	companyID := c.Get("company_id").(string)
	company, err := h.companyRepo.GetByID(companyID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "company not found")
	}

	logoURL, err := uploadFile(c, "logo", "logos")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	company.LogoURL = &logoURL
	if err := h.companyRepo.Update(company); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update logo")
	}

	return c.JSON(http.StatusOK, map[string]string{"logo_url": logoURL})
}

// GetCompanyPublic returns company info for branded login (public)
// GET /api/v1/public/companies/:name
func (h *CompanyHandler) GetCompanyPublic(c echo.Context) error {
	name := c.Param("name")
	if name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "company name required")
	}

	company, err := h.companyRepo.GetByName(name)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "company not found")
	}

	var config models.CompanyConfig
	if company.Config != nil && *company.Config != "" {
		// Config is already JSON, return as-is
		config = models.CompanyConfig{}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"company": company,
		"config":  config,
	})
}