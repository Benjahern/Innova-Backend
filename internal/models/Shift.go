package models

import "time"

// ShiftPattern representa un patrón de turno rotativo (4x3, 3x3, 9x4)
type ShiftPattern struct {
	PatternID       string    `json:"pattern_id"`
	CompanyID       string    `json:"company_id"`
	Name            string    `json:"name"` // "4x3", "3x3", "9x4"
	WorkDays        int       `json:"work_days"`
	OffDays         int       `json:"off_days"`
	IsLegalModality bool      `json:"is_legal_modality"`
	LegalReference  string    `json:"legal_reference,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// WorkRules representa las reglas laborales de una empresa
type WorkRules struct {
	RuleID                   string    `json:"rule_id"`
	CompanyID                string    `json:"company_id"`
	Name                     string    `json:"name"`
	StandardHoursPerWeek     float64   `json:"standard_hours_per_week"`
	MaxHoursPerWeek          float64   `json:"max_hours_per_week"`
	MaxHoursPerDay           float64   `json:"max_hours_per_day"`
	MaxOvertimeHoursPerWeek  float64   `json:"max_overtime_hours_per_week"`
	MaxOvertimeHoursPerDay   float64   `json:"max_overtime_hours_per_day"`
	OvertimeFirst2hSurcharge float64   `json:"overtime_first_2h_surcharge"`
	OvertimeAfter2hSurcharge float64   `json:"overtime_after_2h_surcharge"`
	DefaultLunchStart       string    `json:"default_lunch_start"`
	DefaultLunchEnd         string    `json:"default_lunch_end"`
	GracePeriodMinutes      int       `json:"grace_period_minutes"`
	CreatedAt               time.Time `json:"created_at"`
}

// ShiftAssignment asocia un usuario a un turno
type ShiftAssignment struct {
	AssignmentID string    `json:"assignment_id"`
	UserID       string    `json:"user_id"`
	ShiftID      string    `json:"shift_id"`
	StartDate    string    `json:"start_date"` // YYYY-MM-DD
	EndDate      *string   `json:"end_date,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// DailyAttendanceSummary pre-calculado por día
type DailyAttendanceSummary struct {
	SummaryID       string    `json:"summary_id"`
	UserID          string    `json:"user_id"`
	WorkDate        string    `json:"work_date"` // YYYY-MM-DD
	ExpectedMinutes int       `json:"expected_minutes"`
	WorkedMinutes   int       `json:"worked_minutes"`
	LateMinutes     int       `json:"late_minutes"`
	OvertimeMinutes int       `json:"overtime_minutes"`
	HadLunch        bool      `json:"had_lunch"`
	WasPresent      bool      `json:"was_present"`
	WasLate         bool      `json:"was_late"`
	HadOvertime     bool      `json:"had_overtime"`
	ShiftID         *string   `json:"shift_id,omitempty"`
	CalculatedAt    time.Time `json:"calculated_at"`
}

// WeeklyOvertimeSummary pre-calculado semanal
type WeeklyOvertimeSummary struct {
	SummaryID               string    `json:"summary_id"`
	UserID                  string    `json:"user_id"`
	WeekStart               string    `json:"week_start"` // YYYY-MM-DD (lunes)
	TotalWorkedMinutes      int       `json:"total_worked_minutes"`
	ExpectedMinutes         int       `json:"expected_minutes"`
	OvertimeMinutes         int       `json:"overtime_minutes"`
	OvertimeFirst2hMinutes  int       `json:"overtime_first_2h_minutes"`
	OvertimeAfter2hMinutes  int       `json:"overtime_after_2h_minutes"`
	CalculatedAt           time.Time `json:"calculated_at"`
}