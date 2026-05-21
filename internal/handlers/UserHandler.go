package handlers

import (
	"net/http"
	"regexp"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"

	"turno-papa/internal/models"
	"turno-papa/internal/services"
)

var userEmailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

type UserHandler struct {
	userService *services.UserService
	jwtSecret   string
}

func NewUserHandler(userService *services.UserService, jwtSecret string) *UserHandler {
	return &UserHandler{userService: userService, jwtSecret: jwtSecret}
}

// CreateUser creates a new worker for the company (admin only)
// POST /api/v1/users
func (h *UserHandler) CreateUser(c echo.Context) error {
	companyID := c.Get("company_id").(string)
	role := c.Get("role").(string)

	// Only admins can create users
	if role != "admin" {
		return echo.NewHTTPError(http.StatusForbidden, "only admins can create users")
	}

	var req models.CreateUserRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Name == "" || req.Email == "" || req.Rol == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name, email and rol are required")
	}

	// Validate email format
	if !userEmailRegex.MatchString(req.Email) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid email format")
	}

	// Validate rol
	if req.Rol != "worker" && req.Rol != "manager" {
		return echo.NewHTTPError(http.StatusBadRequest, "rol must be 'worker' or 'manager'")
	}

	// Generate random password for the new user (they should change it)
	tempPassword := uuid.New().String()[:8]
	hash, err := bcrypt.GenerateFromPassword([]byte(tempPassword), bcrypt.DefaultCost)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate password")
	}

	user, err := h.userService.CreateUser(
		companyID,
		req.Name,
		req.Email,
		string(hash),
		req.RUT,
		req.Rol,
		req.BranchID,
	)
	if err != nil {
		// Log the actual error for debugging
		c.Logger().Errorf("CreateUser error: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "failed to create user")
	}

	// Return user with temp password (should be sent securely to the worker)
	response := struct {
		*models.User
		TempPassword string `json:"temp_password"`
	}{
		User:         user,
		TempPassword: tempPassword,
	}

	return c.JSON(http.StatusCreated, response)
}

// GetWorkers returns all workers for the company (admin/manager only)
// GET /api/v1/users
func (h *UserHandler) GetWorkers(c echo.Context) error {
	role := c.Get("role").(string)

	// Only admin and manager and super_admin can list workers
	if role != "admin" && role != "manager" && role != "super_admin" {
		return echo.NewHTTPError(http.StatusForbidden, "only admins and managers can list workers")
	}

	companyID := c.Get("company_id").(string)

	workers, err := h.userService.GetWorkers(companyID)
	if err != nil {
		c.Logger().Errorf("GetWorkers error: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get workers")
	}

	return c.JSON(http.StatusOK, workers)
}

// GetUser returns a specific user (admin/manager only)
// GET /api/v1/users/:id
func (h *UserHandler) GetUser(c echo.Context) error {
	role := c.Get("role").(string)

	// Only admin and manager can view other users
	if role != "admin" && role != "manager" && role != "super_admin" {
		return echo.NewHTTPError(http.StatusForbidden, "only admins and managers can view user details")
	}

	companyID := c.Get("company_id").(string)
	userID := c.Param("id")

	// Verify user belongs to same company (prevents IDOR)
	user, err := h.userService.GetUserByIDAndCompany(userID, companyID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	}

	return c.JSON(http.StatusOK, user)
}

// UpdateUser updates a user
// PUT /api/v1/users/:id
func (h *UserHandler) UpdateUser(c echo.Context) error {
	currentUserID := c.Get("user_id").(string)
	requestedID := c.Param("id")
	companyID := c.Get("company_id").(string)
	role := c.Get("role").(string)

	// Users can only update themselves unless they're admin
	if currentUserID != requestedID && role != "admin" {
		return echo.NewHTTPError(http.StatusForbidden, "cannot update other users")
	}

	var req models.UpdateUserRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	// Verify user belongs to same company (prevents IDOR)
	user, err := h.userService.GetUserByIDAndCompany(requestedID, companyID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "failed to get user")
	}

	// Update fields
	if req.Name != nil {
		user.Name = *req.Name
	}
	if req.Email != nil {
		if !userEmailRegex.MatchString(*req.Email) {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid email format")
		}
		user.Email = *req.Email
	}
	if req.RUT != nil {
		user.RUT = *req.RUT
	}
	if req.BranchID != nil {
		user.BranchID = req.BranchID
	}
	if req.Rol != nil && role == "admin" {
		user.Rol = *req.Rol
	}

	if err := h.userService.UpdateUser(user); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to update user")
	}

	return c.JSON(http.StatusOK, user)
}

// DeleteUser removes a user (admin only)
// DELETE /api/v1/users/:id
func (h *UserHandler) DeleteUser(c echo.Context) error {
	role := c.Get("role").(string)

	if role != "admin" {
		return echo.NewHTTPError(http.StatusForbidden, "only admins can delete users")
	}

	companyID := c.Get("company_id").(string)
	userID := c.Param("id")

	// Verify user belongs to same company (prevents IDOR)
	if _, err := h.userService.GetUserByIDAndCompany(userID, companyID); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	}

	if err := h.userService.DeleteUser(userID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete user")
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "user deleted"})
}

// AssignShift assigns a work shift to a user
// POST /api/v1/users/:id/shifts
func (h *UserHandler) AssignShift(c echo.Context) error {
	userID := c.Param("id")
	role := c.Get("role").(string)
	companyID := c.Get("company_id").(string)

	if role != "admin" {
		return echo.NewHTTPError(http.StatusForbidden, "only admins can assign shifts")
	}

	// Verify user belongs to same company (prevents IDOR)
	if _, err := h.userService.GetUserByIDAndCompany(userID, companyID); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	}

	var req struct {
		ShiftID   string  `json:"shift_id"`
		StartDate *string `json:"start_date,omitempty"` // YYYY-MM-DD - when rotation cycle started
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.ShiftID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "shift_id is required")
	}

	if err := h.userService.AssignShift(userID, req.ShiftID, req.StartDate); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to assign shift")
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "shift assigned"})
}

// GenerateTokenForUser generates a JWT for a specific user (admin creating temp access)
// POST /api/v1/users/:id/token
func (h *UserHandler) GenerateTokenForUser(c echo.Context) error {
	role := c.Get("role").(string)

	if role != "admin" {
		return echo.NewHTTPError(http.StatusForbidden, "only admins can generate tokens")
	}

	companyID := c.Get("company_id").(string)
	userID := c.Param("id")

	// Verify user belongs to same company (prevents IDOR)
	user, err := h.userService.GetUserByIDAndCompany(userID, companyID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	}

	// Generate access token for this user
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":    user.UserID,
		"company_id": user.CompanyID,
		"role":       user.Rol,
		"exp":        jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
	})

	tokenString, err := token.SignedString([]byte(h.jwtSecret))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate token")
	}

	return c.JSON(http.StatusOK, map[string]string{"access_token": tokenString})
}