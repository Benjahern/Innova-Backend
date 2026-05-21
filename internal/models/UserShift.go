package models

import "time"

type UserShift struct {
	UserShiftID string    `json:"user_shift_id"`
	UserID      string    `json:"user_id"`
	ShiftID     string    `json:"shift_id"`
	StartDate   *string   `json:"start_date,omitempty"`  // YYYY-MM-DD - when user's rotation cycle began
	EndDate     *string   `json:"end_date,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}