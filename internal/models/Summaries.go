package models

import "time"

type MonthlyArrearsSummary struct {
	SummaryID        string `json:"summary_id"`
	UserID           string `json:"user_id"`
	Year             int    `json:"year"`
	Month            int    `json:"month"`
	TotalArrearsMin  int    `json:"total_arrears_minutes"`
	DaysWithArrears  int    `json:"days_with_arrears"`
}

type WeeklyHoursSummary struct {
	SummaryID     string    `json:"summary_id"`
	UserID        string    `json:"user_id"`
	WeekStart     time.Time `json:"week_start"`
	TotalHours    float64   `json:"total_hours"`
	ExpectedHours float64   `json:"expected_hours"`
}