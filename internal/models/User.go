package models

import "time"

type User struct {
	UserID      string    `json:"user_id"`
	CompanyID   string    `json:"company_id"`
	BranchID    *string   `json:"branch_id"`
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	Password    string    `json:"-"`
	RUT         string    `json:"rut"`
	Rol         string    `json:"rol"`
	WorkerType  string    `json:"worker_type"` // "fixed", "flexible", "external"
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Shift info (from JOIN with user_shifts + work_shifts)
	ShiftID    *string `json:"shift_id,omitempty"`
	ShiftName  *string `json:"shift_name,omitempty"`
	ShiftStart *string `json:"shift_start,omitempty"`
	ShiftEnd   *string `json:"shift_end,omitempty"`
}