package handlers

import (
	"net/http"
	"regexp"

	"github.com/labstack/echo/v4"

	"turno-papa/internal/models"
	"turno-papa/internal/services"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

type AuthHandler struct {
	authService *services.AuthService
}

func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Register creates a new company owner account
// POST /api/v1/auth/register
func (h *AuthHandler) Register(c echo.Context) error {
	var req models.RegisterRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	// Validate required fields
	if req.Email == "" || req.Password == "" || req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "email, password and name are required")
	}

	// Validate email format
	if !emailRegex.MatchString(req.Email) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid email format")
	}

	// Validate password strength (minimum 8 characters)
	if len(req.Password) < 8 {
		return echo.NewHTTPError(http.StatusBadRequest, "password must be at least 8 characters")
	}

	// Register
	result, err := h.authService.Register(req.Email, req.Password, req.Name, req.Email+"'s Company")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusCreated, result)
}

// Login authenticates a user and returns tokens
// POST /api/v1/auth/login
func (h *AuthHandler) Login(c echo.Context) error {
	var req models.LoginRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Email == "" || req.Password == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "email and password are required")
	}

	// Validate email format
	if !emailRegex.MatchString(req.Email) {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
	}

	result, err := h.authService.Login(req.Email, req.Password)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
	}

	return c.JSON(http.StatusOK, result)
}

// Refresh exchanges a refresh token for new tokens
// POST /api/v1/auth/refresh
func (h *AuthHandler) Refresh(c echo.Context) error {
	var req models.RefreshRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.RefreshToken == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "refresh_token is required")
	}

	result, err := h.authService.Refresh(req.RefreshToken)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid or expired refresh token")
	}

	return c.JSON(http.StatusOK, result)
}

// Logout invalidates refresh tokens for the user
// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c echo.Context) error {
	userID := c.Get("user_id").(string)

	if err := h.authService.Logout(userID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "logout failed")
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "logged out"})
}