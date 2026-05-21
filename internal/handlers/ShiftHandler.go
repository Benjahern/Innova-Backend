package handlers

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"turno-papa/internal/models"
	"turno-papa/internal/repository"
)

var time24hRegex = regexp.MustCompile(`^([01]?\d|2[0-3]):([0-5]\d)$`)

// parseTime24h parses a time string and returns HH:mm format (24h)
func parseTime24h(input string) (string, error) {
	// Normalize whitespace
	re := regexp.MustCompile(`\s+`)
	input = re.ReplaceAllString(input, " ")
	input = regexp.MustCompile(`^\s+|\s+$`).ReplaceAllString(input, "")
	input = strings.ToLower(input)

	// Already in valid 24h format: normalize zero-padding
	if time24hRegex.MatchString(input) {
		parts := time24hRegex.FindStringSubmatch(input)
		hour, _ := strconv.Atoi(parts[1])
		min, _ := strconv.Atoi(parts[2])
		return fmt.Sprintf("%02d:%02d", hour, min), nil
	}

	// Try parsing as 12h with AM/PM
	for _, layout := range []string{"1:04 PM", "1:04PM", "1:04  PM"} {
		t, err := time.Parse(layout, input)
		if err == nil {
			return t.Format("15:04"), nil
		}
	}

	return "", fmt.Errorf("invalid time format: %s (expected HH:mm in 24h, e.g. 09:00 or 17:00)", input)
}

type ShiftHandler struct {
	shiftRepo   repository.WorkShiftRepository
	patternRepo repository.PatternRepository
}

func NewShiftHandler(shiftRepo repository.WorkShiftRepository, patternRepo repository.PatternRepository) *ShiftHandler {
	return &ShiftHandler{shiftRepo: shiftRepo, patternRepo: patternRepo}
}

// CreateShift creates a new work shift
// POST /api/v1/shifts
func (h *ShiftHandler) CreateShift(c echo.Context) error {
	companyID := c.Get("company_id").(string)
	role := c.Get("role").(string)

	if role != "admin" && role != "manager" && role != "super_admin" {
		return echo.NewHTTPError(http.StatusForbidden, "only admins and managers can create shifts")
	}

	var req struct {
		Name      string   `json:"name"`
		Days      []string `json:"days"`
		StartTime string   `json:"start_time"`
		EndTime   string   `json:"end_time"`
		ShiftType string   `json:"shift_type"`
		PatternID *string  `json:"pattern_id"`
		LunchStart *string `json:"lunch_start"`
		LunchEnd   *string `json:"lunch_end"`
		IsActive  bool     `json:"is_active"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Name == "" || req.StartTime == "" || req.EndTime == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name, start_time and end_time are required")
	}

	// Normalize to 24h format
	startTime, err := parseTime24h(req.StartTime)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	endTime, err := parseTime24h(req.EndTime)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Validate end > start
	if startTime >= endTime {
		return echo.NewHTTPError(http.StatusBadRequest, "end_time must be after start_time")
	}

	// Normalize lunch times if provided
	var lunchStart, lunchEnd *string
	if req.LunchStart != nil && *req.LunchStart != "" {
		normalized, err := parseTime24h(*req.LunchStart)
		if err == nil {
			lunchStart = &normalized
		}
	}
	if req.LunchEnd != nil && *req.LunchEnd != "" {
		normalized, err := parseTime24h(*req.LunchEnd)
		if err == nil {
			lunchEnd = &normalized
		}
	}

	// Default shift_type to "fixed"
	shiftType := req.ShiftType
	if shiftType == "" {
		shiftType = "fixed"
	}

	shift := &models.WorkShift{
		ShiftID:    uuid.New().String(),
		CompanyID:  companyID,
		Name:       req.Name,
		Days:       req.Days,
		StartTime:  startTime,
		EndTime:    endTime,
		ShiftType:  shiftType,
		PatternID:  req.PatternID,
		LunchStart: lunchStart,
		LunchEnd:   lunchEnd,
		IsActive:   req.IsActive,
	}

	if err := h.shiftRepo.Create(shift); err != nil {
		c.Logger().Errorf("CreateShift error: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create shift")
	}

	return c.JSON(http.StatusCreated, shift)
}

// GetShifts returns all shifts for the company
// GET /api/v1/shifts
func (h *ShiftHandler) GetShifts(c echo.Context) error {
	companyID := c.Get("company_id").(string)

	shifts, err := h.shiftRepo.GetByCompany(companyID)
	if err != nil {
		c.Logger().Errorf("GetShifts error: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get shifts")
	}

	return c.JSON(http.StatusOK, shifts)
}

// UpdateShift updates a work shift
// PUT /api/v1/shifts/:id
func (h *ShiftHandler) UpdateShift(c echo.Context) error {
	role := c.Get("role").(string)

	if role != "admin" && role != "manager" && role != "super_admin" {
		return echo.NewHTTPError(http.StatusForbidden, "only admins and managers can update shifts")
	}

	shiftID := c.Param("id")
	if shiftID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "shift id is required")
	}

	var req struct {
		Name      *string   `json:"name,omitempty"`
		Days      *[]string `json:"days,omitempty"`
		StartTime *string   `json:"start_time,omitempty"`
		EndTime   *string   `json:"end_time,omitempty"`
		ShiftType *string   `json:"shift_type,omitempty"`
		PatternID *string   `json:"pattern_id,omitempty"`
		LunchStart *string  `json:"lunch_start,omitempty"`
		LunchEnd   *string  `json:"lunch_end,omitempty"`
		IsActive  *bool     `json:"is_active,omitempty"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	shift, err := h.shiftRepo.GetByID(shiftID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "shift not found")
	}

	if req.Name != nil {
		shift.Name = *req.Name
	}
	if req.Days != nil {
		shift.Days = *req.Days
	}
	if req.StartTime != nil {
		normalized, err := parseTime24h(*req.StartTime)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		shift.StartTime = normalized
	}
	if req.EndTime != nil {
		normalized, err := parseTime24h(*req.EndTime)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		shift.EndTime = normalized
	}
	if req.ShiftType != nil {
		shift.ShiftType = *req.ShiftType
	}
	if req.PatternID != nil {
		if *req.PatternID == "" {
			shift.PatternID = nil
		} else {
			shift.PatternID = req.PatternID
		}
	}
	if req.LunchStart != nil {
		if *req.LunchStart == "" {
			shift.LunchStart = nil
		} else {
			normalized, err := parseTime24h(*req.LunchStart)
			if err == nil {
				shift.LunchStart = &normalized
			}
		}
	}
	if req.LunchEnd != nil {
		if *req.LunchEnd == "" {
			shift.LunchEnd = nil
		} else {
			normalized, err := parseTime24h(*req.LunchEnd)
			if err == nil {
				shift.LunchEnd = &normalized
			}
		}
	}
	if req.IsActive != nil {
		shift.IsActive = *req.IsActive
	}

	if shift.StartTime >= shift.EndTime {
		return echo.NewHTTPError(http.StatusBadRequest, "end_time must be after start_time")
	}

	if err := h.shiftRepo.Update(shift); err != nil {
		c.Logger().Errorf("UpdateShift error: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update shift")
	}

	return c.JSON(http.StatusOK, shift)
}

// GetShift returns a single shift by ID
// GET /api/v1/shifts/:id
func (h *ShiftHandler) GetShift(c echo.Context) error {
	shiftID := c.Param("id")
	if shiftID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "shift id is required")
	}

	shift, err := h.shiftRepo.GetByID(shiftID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "shift not found")
	}

	return c.JSON(http.StatusOK, shift)
}

// DeleteShift removes a work shift
// DELETE /api/v1/shifts/:id
func (h *ShiftHandler) DeleteShift(c echo.Context) error {
	role := c.Get("role").(string)

	if role != "admin" && role != "manager" && role != "super_admin" {
		return echo.NewHTTPError(http.StatusForbidden, "only admins and managers can delete shifts")
	}

	shiftID := c.Param("id")
	if shiftID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "shift id is required")
	}

	if err := h.shiftRepo.Delete(shiftID); err != nil {
		c.Logger().Errorf("DeleteShift error: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete shift")
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "shift deleted"})
}

// CreatePattern creates a new shift pattern (4x3, 3x2, etc)
// POST /api/v1/patterns
func (h *ShiftHandler) CreatePattern(c echo.Context) error {
	companyID := c.Get("company_id").(string)
	role := c.Get("role").(string)

	if role != "admin" && role != "manager" && role != "super_admin" {
		return echo.NewHTTPError(http.StatusForbidden, "only admins and managers can create patterns")
	}

	var req struct {
		Name            string `json:"name"`
		WorkDays        int    `json:"work_days"`
		OffDays         int    `json:"off_days"`
		IsLegalModality bool   `json:"is_legal_modality"`
		LegalReference  string `json:"legal_reference,omitempty"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Name == "" || req.WorkDays < 1 || req.OffDays < 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "name, work_days (>=1) and off_days (>=0) are required")
	}

	pattern := &models.ShiftPattern{
		PatternID:       uuid.New().String(),
		CompanyID:       companyID,
		Name:            req.Name,
		WorkDays:        req.WorkDays,
		OffDays:         req.OffDays,
		IsLegalModality: req.IsLegalModality,
		LegalReference:  req.LegalReference,
	}

	if err := h.patternRepo.Create(pattern); err != nil {
		c.Logger().Errorf("CreatePattern error: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create pattern")
	}

	return c.JSON(http.StatusCreated, pattern)
}

// GetPatterns returns all patterns for the company
// GET /api/v1/patterns
func (h *ShiftHandler) GetPatterns(c echo.Context) error {
	companyID := c.Get("company_id").(string)

	patterns, err := h.patternRepo.GetByCompany(companyID)
	if err != nil {
		c.Logger().Errorf("GetPatterns error: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get patterns")
	}

	return c.JSON(http.StatusOK, patterns)
}

// UpdatePattern updates a shift pattern
// PUT /api/v1/patterns/:id
func (h *ShiftHandler) UpdatePattern(c echo.Context) error {
	role := c.Get("role").(string)

	if role != "admin" && role != "manager" && role != "super_admin" {
		return echo.NewHTTPError(http.StatusForbidden, "only admins and managers can update patterns")
	}

	patternID := c.Param("id")
	if patternID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "pattern id is required")
	}

	var req struct {
		Name            *string `json:"name,omitempty"`
		WorkDays        *int    `json:"work_days,omitempty"`
		OffDays         *int    `json:"off_days,omitempty"`
		IsLegalModality *bool   `json:"is_legal_modality,omitempty"`
		LegalReference  *string `json:"legal_reference,omitempty"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	pattern, err := h.patternRepo.GetByID(patternID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "pattern not found")
	}

	if req.Name != nil {
		pattern.Name = *req.Name
	}
	if req.WorkDays != nil {
		pattern.WorkDays = *req.WorkDays
	}
	if req.OffDays != nil {
		pattern.OffDays = *req.OffDays
	}
	if req.IsLegalModality != nil {
		pattern.IsLegalModality = *req.IsLegalModality
	}
	if req.LegalReference != nil {
		pattern.LegalReference = *req.LegalReference
	}

	if err := h.patternRepo.Update(pattern); err != nil {
		c.Logger().Errorf("UpdatePattern error: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update pattern")
	}

	return c.JSON(http.StatusOK, pattern)
}

// DeletePattern removes a shift pattern
// DELETE /api/v1/patterns/:id
func (h *ShiftHandler) DeletePattern(c echo.Context) error {
	role := c.Get("role").(string)

	if role != "admin" && role != "manager" && role != "super_admin" {
		return echo.NewHTTPError(http.StatusForbidden, "only admins and managers can delete patterns")
	}

	patternID := c.Param("id")
	if patternID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "pattern id is required")
	}

	if err := h.patternRepo.Delete(patternID); err != nil {
		c.Logger().Errorf("DeletePattern error: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete pattern")
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "pattern deleted"})
}