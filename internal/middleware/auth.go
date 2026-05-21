package middleware

import (
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

type JWTMiddleware struct {
	jwtSecret string
}

func NewJWTMiddleware(jwtSecret string) *JWTMiddleware {
	return &JWTMiddleware{jwtSecret: jwtSecret}
}

// JWTAuth middleware extracts and validates JWT from Authorization header
// On success, sets user_id, company_id, role in context
// Usage: api.Use(middleware.JWTAuth())
func (m *JWTMiddleware) JWTAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Get Authorization header
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing authorization header")
			}

			// Check Bearer prefix
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid authorization format, use: Bearer <token>")
			}

			tokenString := parts[1]

			// Parse and validate token
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				// Validate signing method
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, echo.NewHTTPError(http.StatusUnauthorized, "invalid signing method")
				}
				return []byte(m.jwtSecret), nil
			})

			if err != nil || !token.Valid {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid or expired token")
			}

			// Extract claims
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid token claims")
			}

			// Set user info in context
			if adminID, ok := claims["admin_id"].(string); ok {
				// Super-admin token
				c.Set("user_id", adminID)
				c.Set("company_id", "")
				c.Set("role", claims["role"])
				c.Set("branch_id", "")
				c.Set("is_super_admin", true)
			} else {
				// Regular user token
				userID, ok := claims["user_id"].(string)
				if !ok {
					return echo.NewHTTPError(http.StatusUnauthorized, "invalid token claims")
				}
				companyID, ok := claims["company_id"].(string)
				if !ok {
					return echo.NewHTTPError(http.StatusUnauthorized, "invalid token claims")
				}
				role, ok := claims["role"].(string)
				if !ok {
					return echo.NewHTTPError(http.StatusUnauthorized, "invalid token claims")
				}
				branchID, _ := claims["branch_id"].(string)

				c.Set("user_id", userID)
				c.Set("company_id", companyID)
				c.Set("role", role)
				c.Set("branch_id", branchID)
				c.Set("is_super_admin", false)
			}

			return next(c)
		}
	}
}

// RequireRole middleware checks if user has one of the required roles
// Usage: api.GET("/admin", handler.AdminOnly, middleware.RequireRole("admin"))
func RequireRole(roles ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userRole := c.Get("role").(string)

			for _, role := range roles {
				if userRole == role {
					return next(c)
				}
			}

			return echo.NewHTTPError(http.StatusForbidden, "insufficient permissions")
		}
	}
}

// SuperAdminOnly middleware ensures the user has super_admin role
// and bypasses company_id scoping by setting it to empty string
// Usage: admin := e.Group("/api/v1/admin"); admin.Use(middleware.SuperAdminOnly())
func SuperAdminOnly() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			role := c.Get("role").(string)
			if role != "super_admin" {
				return echo.NewHTTPError(http.StatusForbidden, "super admin access required")
			}
			// Bypass company scoping for super-admin
			c.Set("company_id", "")
			return next(c)
		}
	}
}