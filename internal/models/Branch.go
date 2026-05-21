package models

import "time"

type Branch struct {
	BranchID   string    `json:"branch_id"`
	CompanyID  string    `json:"company_id"`
	Name       string    `json:"name"`
	Address    string    `json:"address"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}