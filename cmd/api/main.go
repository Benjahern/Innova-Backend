package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/labstack/echo/v4"

	"turno-papa/internal/config"
	"turno-papa/internal/db"
	"turno-papa/internal/handlers"
	"turno-papa/internal/middleware"
	"turno-papa/internal/repository"
	"turno-papa/internal/services"
)

type Handlers struct {
	Auth       *handlers.AuthHandler
	Attendance *handlers.AttendanceHandler
	User       *handlers.UserHandler
	Dashboard  *handlers.DashboardHandler
	Branch     *handlers.BranchHandler
	Shift      *handlers.ShiftHandler
	Admin      *handlers.AdminHandler
	Company    *handlers.CompanyHandler
}

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to database
	database, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Run migrations
	m, err := migrate.New(
		"file:///app/migrations",
		cfg.DatabaseURL,
	)
	if err != nil {
		log.Fatalf("Failed to create migration instance: %v", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Migrations applied successfully")

	// Initialize repositories
	userRepo := repository.NewUserRepository(database)
	companyRepo := repository.NewCompanyRepository(database)
	attendanceRepo := repository.NewAttendanceLogRepository(database)
	shiftRepo := repository.NewWorkShiftRepository(database)
	userShiftRepo := repository.NewUserShiftRepository(database)
	patternRepo := repository.NewPatternRepository(database)
	refreshTokenRepo := repository.NewRefreshTokenRepository(database)
	weeklyHoursRepo := repository.NewWeeklyHoursRepository(database)
	monthlyArrearsRepo := repository.NewMonthlyArrearsRepository(database)
	branchRepo := repository.NewBranchRepository(database)
	adminUserRepo := repository.NewAdminUserRepository(database)

	// Initialize services
	authService := services.NewAuthService(userRepo, refreshTokenRepo, cfg)
	patternService := services.NewPatternService(patternRepo, shiftRepo, userShiftRepo)
	attendanceService := services.NewAttendanceService(attendanceRepo, companyRepo, userRepo, patternService)
	userService := services.NewUserService(userRepo, shiftRepo, userShiftRepo)
	dashboardService := services.NewDashboardService(attendanceRepo, weeklyHoursRepo, monthlyArrearsRepo, userRepo, companyRepo)
	adminService := services.NewAdminService(companyRepo, userRepo, branchRepo, adminUserRepo)

	// Initialize handlers
	h := &Handlers{
		Auth:       handlers.NewAuthHandler(authService),
		Attendance: handlers.NewAttendanceHandler(attendanceService, patternService, userRepo, shiftRepo, userShiftRepo),
		User:       handlers.NewUserHandler(userService, cfg.JWTSecret),
		Dashboard:  handlers.NewDashboardHandler(dashboardService),
		Branch:     handlers.NewBranchHandler(branchRepo),
		Shift:      handlers.NewShiftHandler(shiftRepo, patternRepo),
		Admin:      handlers.NewAdminHandler(adminService, companyRepo, cfg.JWTSecret),
		Company:    handlers.NewCompanyHandler(companyRepo),
	}

	// JWT middleware
	jwtMw := middleware.NewJWTMiddleware(cfg.JWTSecret)

	// Echo instance
	e := echo.New()

	// Serve static files from /app/uploads
	uploadsDir := "/app/uploads"
	os.MkdirAll(uploadsDir, 0755)
	e.Static("/uploads", uploadsDir)

	// Setup routes
	setupRoutes(e, h, jwtMw)

	// Health check (overridable by setupRoutes, but keeping as fallback)
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"status":    "ok",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	// Start server
	addr := fmt.Sprintf("%s:%d", cfg.ServerHost, cfg.ServerPort)
	log.Printf("Starting server on %s", addr)
	if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to start server: %v", err)
	}
}