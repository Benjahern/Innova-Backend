package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"turno-papa/internal/models"
	"turno-papa/internal/repository"
	"turno-papa/internal/services"
)

type AttendanceHandler struct {
	attendanceService *services.AttendanceService
	patternService    *services.PatternService
	userRepo          repository.UserRepository
	shiftRepo         repository.WorkShiftRepository
	userShiftRepo     repository.UserShiftRepository
}

func NewAttendanceHandler(attendanceService *services.AttendanceService, patternService *services.PatternService, userRepo repository.UserRepository, shiftRepo repository.WorkShiftRepository, userShiftRepo repository.UserShiftRepository) *AttendanceHandler {
	return &AttendanceHandler{
		attendanceService: attendanceService,
		patternService:    patternService,
		userRepo:          userRepo,
		shiftRepo:         shiftRepo,
		userShiftRepo:     userShiftRepo,
	}
}

// RecordAttendance marks a checkin or checkout for a worker
// POST /api/v1/attendance
func (h *AttendanceHandler) RecordAttendance(c echo.Context) error {
	userID := c.Get("user_id").(string)
	companyID := c.Get("company_id").(string)
	role := c.Get("role").(string)
	managerBranchID, _ := c.Get("branch_id").(string)

	var req struct {
		WorkerID string `json:"user_id"`
		Type     string `json:"type"` // checkin or checkout - if empty, auto-detect
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	// Validate type if provided
	if req.Type != "" && req.Type != "checkin" && req.Type != "checkout" {
		return echo.NewHTTPError(http.StatusBadRequest, "type must be 'checkin' or 'checkout'")
	}

	// Determine target worker
	targetUserID := userID
	if req.WorkerID != "" {
		targetUserID = req.WorkerID
	}

	// Get target worker's info to find company and branch
	targetWorker, err := h.userRepo.GetByID(targetUserID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "trabajador no encontrado")
	}

	// Verify permission: manager can only record for workers in their branch (or themselves)
	if role == "manager" {
		isSelf := targetUserID == userID
		if targetWorker.BranchID == nil || (*targetWorker.BranchID != managerBranchID && !isSelf) {
			return echo.NewHTTPError(http.StatusForbidden, "solo puedes registrar asistencia de trabajadores de tu sucursal")
		}
	}

	if targetWorker.BranchID == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "el trabajador no tiene sucursal asignada")
	}
	branchID := *targetWorker.BranchID

	// For super-admin, company_id is empty - get it from the worker
	if companyID == "" {
		companyID = targetWorker.CompanyID
	}

	source := c.Request().Header.Get("X-Source")
	if source == "" {
		source = "mobile"
	}

	result, err := h.attendanceService.RecordAttendance(targetUserID, companyID, branchID, source, req.Type)
	if err != nil {
		c.Logger().Errorf("RecordAttendance error: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to record attendance")
	}

	return c.JSON(http.StatusCreated, result)
}

// GetMyAttendance returns attendance logs for the authenticated user
// GET /api/v1/attendance/me
func (h *AttendanceHandler) GetMyAttendance(c echo.Context) error {
	userID := c.Get("user_id").(string)
	startDate := c.QueryParam("start")
	endDate := c.QueryParam("end")

	var logs []*models.AttendanceLog
	var err error

	if startDate != "" && endDate != "" {
		logs, err = h.attendanceService.GetUserAttendanceByDateRange(userID, startDate, endDate)
	} else {
		logs, err = h.attendanceService.GetUserAttendance(userID)
	}

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get attendance")
	}

	return c.JSON(http.StatusOK, logs)
}

// GetAttendance returns paginated attendance logs for the company (admin/manager only)
// GET /api/v1/attendance?limit=50&offset=0
func (h *AttendanceHandler) GetAttendance(c echo.Context) error {
	role := c.Get("role").(string)
	companyID := c.Get("company_id").(string)
	managerBranchID, _ := c.Get("branch_id").(string)

	// Only admin and manager can view all attendance
	if role != "admin" && role != "manager" && role != "super_admin" {
		return echo.NewHTTPError(http.StatusForbidden, "only admins and managers can view all attendance")
	}

	// Parse pagination params
	var limit, offset int
	if l := c.QueryParam("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if o := c.QueryParam("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	var logs []*models.AttendanceLog
	var total int
	var err error

	if role == "manager" {
		logs, total, err = h.attendanceService.GetBranchAttendance(companyID, managerBranchID, limit, offset)
	} else {
		logs, total, err = h.attendanceService.GetCompanyAttendance(companyID, limit, offset)
	}
	if err != nil {
		c.Logger().Errorf("GetAttendance error: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get attendance")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"logs":   logs,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// ExportAttendance exports attendance logs as CSV (admin/manager only)
// GET /api/v1/attendance/export
func (h *AttendanceHandler) ExportAttendance(c echo.Context) error {
	role := c.Get("role").(string)

	if role != "admin" && role != "manager" && role != "super_admin" {
		return echo.NewHTTPError(http.StatusForbidden, "only admins and managers can export attendance")
	}

	companyID := c.Get("company_id").(string)

	// Get all logs for company (no limit for export)
	logs, _, err := h.attendanceService.GetCompanyAttendance(companyID, 10000, 0)
	if err != nil {
		c.Logger().Errorf("ExportAttendance error: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to export attendance")
	}

	// Load users to resolve names
	users, err := h.userRepo.GetByCompany(companyID)
	if err != nil {
		c.Logger().Errorf("ExportAttendance error loading users: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to load users for export")
	}
	userNames := make(map[string]string)
	for _, u := range users {
		userNames[u.UserID] = u.Name
	}

	// Generate CSV - user_id resolved to name, no log_id
	csv := "usuario,branch,tipo,fecha_hora,atraso,fuente\n"
	for _, log := range logs {
		userName := log.UserID
		if name, ok := userNames[log.UserID]; ok {
			userName = name
		}
		csv += fmt.Sprintf("%s,%s,%s,%s,%v,%s\n",
			userName, log.BranchID, log.Type, log.Timestamp.Format("2006-01-02 15:04:05"), log.IsLate, log.Source)
	}

	c.Response().Header().Set("Content-Type", "text/csv")
	c.Response().Header().Set("Content-Disposition", "attachment; filename=attendance_export.csv")

	return c.String(http.StatusOK, csv)
}

// ExportUserAttendance exports attendance logs for a specific user as CSV
// GET /api/v1/attendance/export/:userId
func (h *AttendanceHandler) ExportUserAttendance(c echo.Context) error {
	role := c.Get("role").(string)

	if role != "admin" && role != "manager" && role != "super_admin" {
		return echo.NewHTTPError(http.StatusForbidden, "only admins and managers can export attendance")
	}

	userID := c.Param("userId")
	if userID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "user_id is required")
	}

	logs, _, err := h.attendanceService.GetUserAttendancePaginated(userID, 10000, 0)
	if err != nil {
		c.Logger().Errorf("ExportUserAttendance error: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to export user attendance")
	}

	// Load user to get name
	user, err := h.userRepo.GetByID(userID)
	userName := userID
	if err == nil && user != nil {
		userName = user.Name
	}

	// Generate CSV - user name resolved, no log_id
	csv := "usuario,branch,tipo,fecha_hora,atraso,fuente\n"
	for _, log := range logs {
		csv += fmt.Sprintf("%s,%s,%s,%s,%v,%s\n",
			userName, log.BranchID, log.Type, log.Timestamp.Format("2006-01-02 15:04:05"), log.IsLate, log.Source)
	}

	c.Response().Header().Set("Content-Type", "text/csv")
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=attendance_user_%s.csv", userID))

	return c.String(http.StatusOK, csv)
}

// GetWeeklyAudit returns detailed audit for a worker's week
// GET /api/v1/attendance/audit?user_id=X&week_start=YYYY-MM-DD
func (h *AttendanceHandler) GetWeeklyAudit(c echo.Context) error {
	role := c.Get("role").(string)

	if role != "admin" && role != "manager" && role != "super_admin" {
		return echo.NewHTTPError(http.StatusForbidden, "only admins and managers can view audit")
	}

	userID := c.QueryParam("user_id")
	weekStart := c.QueryParam("week_start")

	if userID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "user_id is required")
	}
	if weekStart == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "week_start is required")
	}

	// Get user to find company
	user, err := h.userRepo.GetByID(userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	}

	// Get company for schedule settings
	var defaultStart, defaultEnd *string
	var lunchStart, lunchEnd *string // From shift if assigned, else from company
	lunchDurationMinutes := 60 // default

	company, err := h.attendanceService.GetCompanyByID(user.CompanyID)
	if err == nil && company != nil {
		defaultStart = &company.DefaultStartTime
		defaultEnd = company.DefaultEndTime
		// Company-level lunch defaults (fallback)
		lunchStart = company.LunchStart
		lunchEnd = company.LunchEnd

		// Try to get user's assigned shift for shift-specific lunch window
		userShifts, err := h.userShiftRepo.GetByUser(userID)
		if err == nil && len(userShifts) > 0 {
			// Get the most recent active assignment
			for _, us := range userShifts {
				shift, err := h.shiftRepo.GetByID(us.ShiftID)
				if err == nil && shift != nil && shift.IsActive {
					// Use shift-specific lunch if configured
					if shift.LunchStart != nil && *shift.LunchStart != "" {
						lunchStart = shift.LunchStart
					}
					if shift.LunchEnd != nil && *shift.LunchEnd != "" {
						lunchEnd = shift.LunchEnd
					}
					// Shift also overrides default start/end if configured
					if shift.StartTime != "" {
						defaultStart = &shift.StartTime
					}
					if shift.EndTime != "" {
						defaultEnd = &shift.EndTime
					}
					break // Use first active shift
				}
			}
		}

		// Calculate lunch duration if both times are set
		if lunchStart != nil && lunchEnd != nil {
			startParts := parseTimeParts(*lunchStart)
			endParts := parseTimeParts(*lunchEnd)
			lunchStartDt := time.Date(0, 0, 0, startParts.hour, startParts.min, 0, 0, time.UTC)
			lunchEndDt := time.Date(0, 0, 0, endParts.hour, endParts.min, 0, 0, time.UTC)
			lunchDurationMinutes = int(lunchEndDt.Sub(lunchStartDt).Minutes())
		}
	}

	// Get logs for the week (Mon-Sun = 6 days)
	logs, err := h.attendanceService.GetUserAttendanceByDateRange(userID, weekStart, weekStart+" +6 days")
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get attendance")
	}

	// Build daily breakdown
	type DayAudit struct {
		Date        string `json:"date"`
		IsWorkDay   bool   `json:"is_work_day"`
		Checkin     string `json:"checkin,omitempty"`
		Checkout    string `json:"checkout,omitempty"`
		WorkedMin   int    `json:"worked_minutes"`
		ExpectedMin int    `json:"expected_minutes"`
		LateMin     int    `json:"late_minutes"`
		HasLunch    bool   `json:"has_lunch"`
	}

	var audit []DayAudit
	var totalWorked, totalExpected int

	// Parse week_start as date
	weekStartDate, _ := parseDate(weekStart)

	for i := 0; i < 7; i++ {
		currentDate := weekStartDate.AddDate(0, 0, i)
		dateStr := currentDate.Format("2006-01-02")
		dayName := currentDate.Weekday().String()

		day := DayAudit{
			Date: fmt.Sprintf("%s (%s)", dateStr, dayName[:3]),
		}

		// Check if this is a work day for the user (for rotating shifts)
		isWorkDay := true
		if h.patternService != nil {
			isWorkDay, _ = h.patternService.IsWorkDay(userID, currentDate)
		}
		day.IsWorkDay = isWorkDay

		// If not a work day, skip attendance processing but still add to audit
		if !isWorkDay {
			audit = append(audit, day)
			continue
		}

		// Find checkin and checkout for this day
		var checkinTime, checkoutTime *string
		for _, log := range logs {
			logDate := log.Timestamp.Format("2006-01-02")
			if logDate != dateStr {
				continue
			}
			if log.Type == "checkin" {
				t := log.Timestamp.Format("15:04")
				checkinTime = &t
			} else if log.Type == "checkout" {
				t := log.Timestamp.Format("15:04")
				checkoutTime = &t
			}
		}

		if checkinTime != nil {
			day.Checkin = *checkinTime
		}
		if checkoutTime != nil {
			day.Checkout = *checkoutTime
		}

		// If we have both checkin and checkout, calculate worked time
		if checkinTime != nil && checkoutTime != nil {
			checkinDt, _ := parseDateTime(dateStr + " " + *checkinTime)
			checkoutDt, _ := parseDateTime(dateStr + " " + *checkoutTime)
			workedMin := int(checkoutDt.Sub(checkinDt).Minutes())

			// Calculate expected time and lunch detection
			hasLunch := false
			expectedMin := 0

			if defaultEnd != nil && defaultStart != nil {
				defaultEndDt, _ := parseDateTime(dateStr + " " + *defaultEnd)
				startDt, _ := parseDateTime(dateStr + " " + *defaultStart)

				// Calculate gross expected (start to end)
				grossExpected := int(defaultEndDt.Sub(startDt).Minutes())

				// Default lunch window: 13:00-14:00
				defaultLunchStart := time.Date(checkinDt.Year(), checkinDt.Month(), checkinDt.Day(), 13, 0, 0, 0, time.UTC)
				defaultLunchEnd := time.Date(checkinDt.Year(), checkinDt.Month(), checkinDt.Day(), 14, 0, 0, 0, time.UTC)

				// If company has lunch_start/lunch_end config, use it
				lunchStartDt := defaultLunchStart
				lunchEndDt := defaultLunchEnd
				if lunchStart != nil && lunchEnd != nil {
					startParts := parseTimeParts(*lunchStart)
					endParts := parseTimeParts(*lunchEnd)
					lunchStartDt = time.Date(checkinDt.Year(), checkinDt.Month(), checkinDt.Day(), startParts.hour, startParts.min, 0, 0, time.UTC)
					lunchEndDt = time.Date(checkinDt.Year(), checkinDt.Month(), checkinDt.Day(), endParts.hour, endParts.min, 0, 0, time.UTC)
				}

				// Calculate overlap with lunch window
				overlapStart := checkinDt
				if lunchStartDt.After(checkinDt) {
					overlapStart = lunchStartDt
				}
				overlapEnd := checkoutDt
				if lunchEndDt.Before(checkoutDt) {
					overlapEnd = lunchEndDt
				}

				// Calculate lunch duration for subtraction
				lunchDurationMins := int(lunchEndDt.Sub(lunchStartDt).Minutes())

				// If there's at least 30 minutes overlap with lunch window, count as lunch taken
				if overlapEnd.After(overlapStart) && overlapEnd.Sub(overlapStart).Minutes() >= 30 {
					hasLunch = true
					expectedMin = grossExpected - lunchDurationMins
				} else {
					// No lunch taken - count all worked time
					hasLunch = false
					expectedMin = workedMin
				}
			} else {
				// No schedule configured - use 8hr day default
				expectedMin = 480
				if workedMin >= 420 {
					hasLunch = true
					expectedMin = 480 - 60
				} else {
					hasLunch = false
					expectedMin = workedMin
				}
			}

			day.WorkedMin = workedMin
			day.ExpectedMin = expectedMin
			day.HasLunch = hasLunch

			totalWorked += workedMin
			totalExpected += expectedMin
		}

		audit = append(audit, day)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"user_id":       userID,
		"week_start":    weekStart,
		"user_name":     user.Name,
		"default_start": defaultStart,
		"default_end":   defaultEnd,
		"lunch_minutes": lunchDurationMinutes,
		"days":          audit,
		"totals": map[string]int{
			"worked_minutes":   totalWorked,
			"expected_minutes": totalExpected,
			"deficit_minutes":  totalExpected - totalWorked,
		},
	})
}

func parseDate(s string) (time.Time, error) {
	return time.Parse("2006-01-02", s)
}

func parseTime(s string) (time.Time, error) {
	return time.Parse("15:04", s)
}

func parseDateTime(s string) (time.Time, error) {
	return time.Parse("2006-01-02 15:04", s)
}

type timeParts struct {
	hour int
	min  int
}

func parseTimeParts(s string) timeParts {
	s = strings.TrimSpace(s)
	parts := strings.Split(s, ":")
	h := 13
	m := 0
	if len(parts) >= 2 {
		h, _ = strconv.Atoi(parts[0])
		m, _ = strconv.Atoi(parts[1])
	}
	return timeParts{hour: h, min: m}
}