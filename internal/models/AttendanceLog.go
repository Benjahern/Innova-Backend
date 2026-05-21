package models

import "time"

type AttendanceLog struct {
	LogID     string    `json:"log_id"`
	UserID    string    `json:"user_id"`
	BranchID  string    `json:"branch_id"`
	Type      string    `json:"type"`      // "checkin" or "checkout"
	Timestamp time.Time `json:"timestamp"`
	IsLate    int       `json:"is_late"`   // minutes late (0 if early or on checkout)
	Source    string    `json:"source"`     // "mobile", "rfid", etc.
}