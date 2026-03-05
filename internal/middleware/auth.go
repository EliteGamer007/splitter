package middleware

import (
	"net/http"
	"strings"

	"splitter/internal/db"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

// AuthMiddleware validates JWT tokens and sets user DID context
func AuthMiddleware(jwtSecret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Get token from Authorization header
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Missing authorization header",
				})
			}

			// Extract token (format: "Bearer <token>")
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Invalid authorization header format",
				})
			}

			tokenString := parts[1]

			// Parse and validate token
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				// Validate signing method
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, echo.NewHTTPError(http.StatusUnauthorized, "Invalid signing method")
				}
				return []byte(jwtSecret), nil
			})

			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Invalid or expired token",
				})
			}

			// Extract claims
			if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
				// Set user ID (subject) in context
				if userID, ok := claims["sub"].(string); ok {
					c.Set("user_id", userID)
				}
				// Set DID in context
				if did, ok := claims["did"].(string); ok {
					c.Set("did", did)
				}
				if username, ok := claims["username"].(string); ok {
					c.Set("username", username)
				}
				// Set role in context
				if role, ok := claims["role"].(string); ok {
					c.Set("role", role)
				} else {
					c.Set("role", "user") // Default to user role
				}
			} else {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Invalid token claims",
				})
			}

			// Check if user is suspended
			if userID, ok := c.Get("user_id").(string); ok && userID != "" {
				var isSuspended bool
				err := db.GetDB().QueryRow(c.Request().Context(),
					"SELECT is_suspended FROM users WHERE id = $1", userID,
				).Scan(&isSuspended)
				if err == nil && isSuspended {
					return c.JSON(http.StatusForbidden, map[string]string{
						"error": "Account suspended",
					})
				}
			}

			return next(c)
		}
	}
}

// OptionalAuthMiddleware validates JWT tokens if present but doesn't require them
func OptionalAuthMiddleware(jwtSecret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return next(c)
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				return next(c)
			}

			tokenString := parts[1]
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, echo.NewHTTPError(http.StatusUnauthorized, "Invalid signing method")
				}
				return []byte(jwtSecret), nil
			})

			if err == nil {
				if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
					// Set DID in context
					if did, ok := claims["did"].(string); ok {
						c.Set("did", did)
					}
					if username, ok := claims["username"].(string); ok {
						c.Set("username", username)
					}
				}
			}

			return next(c)
		}
	}
}
