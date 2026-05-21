package models

type WorkShift struct {
	ShiftID     string   `json:"shift_id"`
	CompanyID   string   `json:"company_id"`
	Name        string   `json:"name"`
	ShiftType   string   `json:"shift_type"` // "fixed", "rotating", "flexible"
	Days        []string `json:"days"`       // ["monday", "tuesday", ...]
	PatternID   *string  `json:"pattern_id,omitempty"`
	PatternName *string  `json:"pattern_name,omitempty"` // from JOIN with shift_patterns
	StartTime   string   `json:"start_time"` // "10:00"
	EndTime     string   `json:"end_time"`   // "19:00"
	LunchStart  *string  `json:"lunch_start,omitempty"` // "13:00"
	LunchEnd    *string  `json:"lunch_end,omitempty"`   // "14:00"
	IsActive    bool     `json:"is_active"`
}