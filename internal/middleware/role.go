package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// RequireAdmin allows only users with the "admin" role.
func RequireAdmin(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		role, _ := c.Get("role").(string)
		if role != "admin" {
			return c.JSON(http.StatusForbidden, map[string]string{
				"error": "Admin access required",
			})
		}
		return next(c)
	}
}

// RequireModOrAdmin allows users with "admin" or "moderator" roles.
func RequireModOrAdmin(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		role, _ := c.Get("role").(string)
		if role != "admin" && role != "moderator" {
			return c.JSON(http.StatusForbidden, map[string]string{
				"error": "Moderator or admin access required",
			})
		}
		return next(c)
	}
}
