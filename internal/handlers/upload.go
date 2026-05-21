package handlers

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

func uploadFile(c echo.Context, fieldName, subDir string) (string, error) {
	uploadsDir := filepath.Join("/app/uploads", subDir)
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := c.FormFile(fieldName)
	if err != nil {
		return "", fmt.Errorf("no file provided: %w", err)
	}

	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	// Generate unique filename preserving original extension
	ext := filepath.Ext(file.Filename)
	safeName := strings.ReplaceAll(file.Filename, " ", "_")
	safeName = strings.ReplaceAll(safeName, "'", "")
	safeName = strings.ReplaceAll(safeName, "\"", "")
	filename := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), safeName, ext)

	dst, err := os.Create(filepath.Join(uploadsDir, filename))
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer dst.Close()

	if _, err = io.Copy(dst, src); err != nil {
		return "", fmt.Errorf("failed to copy file: %w", err)
	}

	return fmt.Sprintf("/uploads/%s/%s", subDir, filename), nil
}