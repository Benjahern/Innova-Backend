package main

import (
	"time"

	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"

	"turno-papa/internal/handlers"
	"turno-papa/internal/middleware"
)

func setupRoutes(e *echo.Echo, h *Handlers, jwtMw *middleware.JWTMiddleware) {
	// Middleware
	e.Use(echoMiddleware.Logger())
	e.Use(echoMiddleware.Recover())
	e.Use(echoMiddleware.CORSWithConfig(echoMiddleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{echo.GET, echo.POST, echo.PUT, echo.DELETE},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAuthorization},
	}))

	// Health check
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	// Public auth routes with rate limiting
	auth := e.Group("/api/v1/auth")
	auth.Use(echoMiddleware.RateLimiterWithConfig(echoMiddleware.RateLimiterConfig{
		Store: echoMiddleware.NewRateLimiterMemoryStoreWithConfig(
			echoMiddleware.RateLimiterMemoryStoreConfig{
				Rate:      10,
				Burst:     20,
				ExpiresIn: time.Minute,
			},
		),
		IdentifierExtractor: func(c echo.Context) (string, error) {
			return c.RealIP(), nil
		},
	}))
	auth.POST("/register", h.Auth.Register)
	auth.POST("/login", h.Auth.Login)
	auth.POST("/refresh", h.Auth.Refresh)

	// Public routes for branded login (must be before protected routes)
	public := e.Group("/api/v1/public")
	public.GET("/companies/:name/config", h.Admin.GetCompanyConfig)
	public.GET("/logos/:filename", handlers.GetPublicLogo)

	// Public admin login route (no JWT required) - must be before JWT middleware
	e.POST("/api/v1/admin/login", h.Admin.AdminLogin)

	// Protected routes
	api := e.Group("/api/v1")
	api.Use(jwtMw.JWTAuth())

	// Auth (protected)
	api.POST("/auth/logout", h.Auth.Logout)

	// Attendance
	api.POST("/attendance", h.Attendance.RecordAttendance)
	api.GET("/attendance/me", h.Attendance.GetMyAttendance)
	api.GET("/attendance", h.Attendance.GetAttendance)
	api.GET("/attendance/audit", h.Attendance.GetWeeklyAudit)
	api.GET("/attendance/export", h.Attendance.ExportAttendance)
	api.GET("/attendance/export/:userId", h.Attendance.ExportUserAttendance)

	// Users
	api.GET("/users", h.User.GetWorkers)
	api.POST("/users", h.User.CreateUser)
	api.GET("/users/:id", h.User.GetUser)
	api.PUT("/users/:id", h.User.UpdateUser)
	api.DELETE("/users/:id", h.User.DeleteUser)
	api.POST("/users/:id/shifts", h.User.AssignShift)
	api.POST("/users/:id/token", h.User.GenerateTokenForUser)

	// Dashboard
	api.GET("/dashboard", h.Dashboard.GetDashboard)

	// Shifts
	api.GET("/shifts", h.Shift.GetShifts)
	api.POST("/shifts", h.Shift.CreateShift)
	api.PUT("/shifts/:id", h.Shift.UpdateShift)
	api.DELETE("/shifts/:id", h.Shift.DeleteShift)
	api.GET("/shifts/:id", h.Shift.GetShift)

	// Patterns
	api.GET("/patterns", h.Shift.GetPatterns)
	api.POST("/patterns", h.Shift.CreatePattern)
	api.PUT("/patterns/:id", h.Shift.UpdatePattern)
	api.DELETE("/patterns/:id", h.Shift.DeletePattern)

	// Branches
	api.GET("/branches", h.Branch.GetBranches)
	api.POST("/branches", h.Branch.CreateBranch)
	api.DELETE("/branches/:id", h.Branch.DeleteBranch)

	// Company settings (admin/manager only)
	api.GET("/company", h.Company.GetCompany)
	api.PUT("/company", h.Company.UpdateCompany)
	api.POST("/company/logo", h.Company.UploadLogo)

	// Super-admin routes (bypass company scoping)
	admin := e.Group("/api/v1/admin")
	admin.Use(jwtMw.JWTAuth())
	admin.Use(middleware.SuperAdminOnly())
	admin.GET("/companies", h.Admin.ListCompanies)
	admin.POST("/companies", h.Admin.CreateCompany)
	admin.PUT("/companies/:id", h.Admin.UpdateCompany)
	admin.DELETE("/companies/:id", h.Admin.DeleteCompany)
	admin.POST("/companies/:id/logo", h.Admin.UploadCompanyLogo)
}