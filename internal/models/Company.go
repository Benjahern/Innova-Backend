package models

import "time"

type Company struct {
	CompanyID            string    `json:"company_id"`
	Name                string    `json:"name"`
	LogoURL             *string   `json:"logo_url"`
	Config              *string   `json:"config"` // JSON stored as string
	DefaultStartTime    string    `json:"default_start_time"` // "10:00"
	DefaultEndTime      *string   `json:"default_end_time"`   // "18:00" (optional, defaults to start + 8hrs)
	WorkHoursPerWeek    float64   `json:"work_hours_per_week"` // 42.0 or 40.0
	LunchStart          *string   `json:"lunch_start"` // "13:00" (hour:minute)
	LunchEnd            *string   `json:"lunch_end"`   // "14:00" (hour:minute)
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// CompanyConfig represents the JSON config stored in companies.config
type CompanyConfig struct {
	Theme    ThemeConfig    `json:"theme"`
	Branding BrandingConfig `json:"branding"`
}

type ThemeConfig struct {
	PrimaryColor    string `json:"primary_color"`
	SecondaryColor string `json:"secondary_color"`
	AccentColor     string `json:"accent_color"`
	BackgroundColor string `json:"background_color"`
	TextColor       string `json:"text_color"`
}

type BrandingConfig struct {
	CompanyName string `json:"company_name"`
	Tagline     string `json:"tagline"`
}

// CreateCompanyRequest is the request body for creating a new company
type CreateCompanyRequest struct {
	Name               string           `json:"name"`
	LogoURL            *string          `json:"logo_url"`
	Config             CompanyConfig    `json:"config"`
	DefaultStartTime   string           `json:"default_start_time"`
	DefaultEndTime     *string          `json:"default_end_time"`
	WorkHoursPerWeek   float64          `json:"work_hours_per_week"`
	AdminUser          AdminUserRequest `json:"admin_user"`
	Branch             BranchRequest    `json:"branch"`
}

type AdminUserRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
	RUT      string `json:"rut"`
}

type BranchRequest struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}