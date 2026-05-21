package models

// Auth DTOs

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	User         *User  `json:"user"`
}

// Attendance DTOs

type RecordAttendanceRequest struct {
	BranchID string `json:"branch_id"`
}

type AttendanceResponse struct {
	LogID     string `json:"log_id"`
	UserID    string `json:"user_id"`
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
	IsLate    bool   `json:"is_late"`
}

// User DTOs

type CreateUserRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	RUT      string `json:"rut"`
	Rol      string `json:"rol"`
	BranchID string `json:"branch_id"`
}

type UpdateUserRequest struct {
	Name     *string `json:"name,omitempty"`
	Email    *string `json:"email,omitempty"`
	RUT      *string `json:"rut,omitempty"`
	Rol      *string `json:"rol,omitempty"`
	BranchID *string `json:"branch_id,omitempty"`
}

type UserResponse struct {
	UserID    string `json:"user_id"`
	CompanyID string `json:"company_id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	RUT       string `json:"rut"`
	Rol       string `json:"rol"`
	BranchID  string `json:"branch_id"`
}

// Error response

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}