package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"turno-papa/internal/services"
)

type DashboardHandler struct {
	dashboardService *services.DashboardService
}

func NewDashboardHandler(dashboardService *services.DashboardService) *DashboardHandler {
	return &DashboardHandler{dashboardService: dashboardService}
}

// GetDashboard returns the dashboard summary for the company
// GET /api/v1/dashboard
func (h *DashboardHandler) GetDashboard(c echo.Context) error {
	role := c.Get("role").(string)

	// Only admin, manager and super_admin can view dashboard
	if role != "admin" && role != "manager" && role != "super_admin" {
		return echo.NewHTTPError(http.StatusForbidden, "only admins and managers can view dashboard")
	}

	companyID := c.Get("company_id").(string)

	summary, err := h.dashboardService.GetDashboard(companyID)
	if err != nil {
		c.Logger().Errorf("GetDashboard error: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get dashboard")
	}

	return c.JSON(http.StatusOK, summary)
}