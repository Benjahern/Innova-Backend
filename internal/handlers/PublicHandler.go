package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"
)

// GetPublicLogo serves company logos publicly without authentication
// GET /api/v1/public/logos/:filename
func GetPublicLogo(c echo.Context) error {
	filename := c.Param("filename")
	if filename == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "filename required")
	}

	// Security: prevent path traversal
	filename = filepath.Base(filename)
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid filename")
	}

	// Only allow image files
	ext := strings.ToLower(filepath.Ext(filename))
	if ext != ".png" && ext != ".jpg" && ext != ".jpeg" && ext != ".gif" && ext != ".webp" && ext != ".svg" {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid file type")
	}

	logosDir := "/app/uploads/logos"
	filePath := filepath.Join(logosDir, filename)

	// Verify file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return echo.NewHTTPError(http.StatusNotFound, "logo not found")
	}

	// Determine content type
	contentType := "application/octet-stream"
	switch ext {
	case ".png":
		contentType = "image/png"
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".gif":
		contentType = "image/gif"
	case ".webp":
		contentType = "image/webp"
	case ".svg":
		contentType = "image/svg+xml"
	}

	c.Response().Header().Set("Content-Type", contentType)
	c.Response().Header().Set("Cache-Control", "public, max-age=86400")
	return c.File(filePath)
}