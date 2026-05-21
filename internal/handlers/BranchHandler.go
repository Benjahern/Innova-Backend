package handlers

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"turno-papa/internal/models"
	"turno-papa/internal/repository"
)

type BranchHandler struct {
	branchRepo repository.BranchRepository
}

func NewBranchHandler(branchRepo repository.BranchRepository) *BranchHandler {
	return &BranchHandler{branchRepo: branchRepo}
}

// CreateBranch creates a new branch
// POST /api/v1/branches
func (h *BranchHandler) CreateBranch(c echo.Context) error {
	companyID := c.Get("company_id").(string)
	role := c.Get("role").(string)

	if role != "admin" {
		return echo.NewHTTPError(http.StatusForbidden, "only admins can create branches")
	}

	var req struct {
		Name    string `json:"name"`
		Address string `json:"address"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is required")
	}

	branch := &models.Branch{
		BranchID:  uuid.New().String(),
		CompanyID: companyID,
		Name:      req.Name,
		Address:   req.Address,
	}

	if err := h.branchRepo.Create(branch); err != nil {
		c.Logger().Errorf("CreateBranch error: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create branch")
	}

	return c.JSON(http.StatusCreated, branch)
}

// GetBranches returns all branches for the company
// GET /api/v1/branches
func (h *BranchHandler) GetBranches(c echo.Context) error {
	companyID := c.Get("company_id").(string)

	branches, err := h.branchRepo.GetByCompany(companyID)
	if err != nil {
		c.Logger().Errorf("GetBranches error: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get branches")
	}

	return c.JSON(http.StatusOK, branches)
}

// DeleteBranch removes a branch
// DELETE /api/v1/branches/:id
func (h *BranchHandler) DeleteBranch(c echo.Context) error {
	role := c.Get("role").(string)

	if role != "admin" {
		return echo.NewHTTPError(http.StatusForbidden, "only admins can delete branches")
	}

	branchID := c.Param("id")
	if branchID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "branch id is required")
	}

	if err := h.branchRepo.Delete(branchID); err != nil {
		c.Logger().Errorf("DeleteBranch error: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete branch")
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "branch deleted"})
}