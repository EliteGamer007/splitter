package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"splitter/internal/db"
	"splitter/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
)

// AccountLookup resolves the current account; token role/status are never authoritative.
type AccountLookup func(context.Context, string) (*models.User, error)

func currentAccount(ctx context.Context, id string) (*models.User, error) {
	pool := db.GetDB()
	if pool == nil {
		return nil, errors.New("authentication database unavailable")
	}
	user := &models.User{ID: id}
	err := pool.QueryRow(ctx, `SELECT did, username, COALESCE(role, 'user'), COALESCE(is_suspended, false) FROM users WHERE id = $1`, id).Scan(&user.DID, &user.Username, &user.Role, &user.IsSuspended)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return user, err
}

// AuthMiddleware requires a valid token and an existing, active account.
// The optional lookup supports alternate account stores while retaining existing calls.
func AuthMiddleware(jwtSecret string, lookups ...AccountLookup) echo.MiddlewareFunc {
	return accountAuthentication(jwtSecret, false, lookups...)
}

// OptionalAuthMiddleware permits anonymous access only when no credentials were sent.
// Present but invalid credentials fail explicitly, rather than exposing stale identity.
func OptionalAuthMiddleware(jwtSecret string, lookups ...AccountLookup) echo.MiddlewareFunc {
	return accountAuthentication(jwtSecret, true, lookups...)
}

func accountAuthentication(secret string, optional bool, lookups ...AccountLookup) echo.MiddlewareFunc {
	lookup := AccountLookup(currentAccount)
	if len(lookups) > 0 && lookups[0] != nil {
		lookup = lookups[0]
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			header := c.Request().Header.Get("Authorization")
			if header == "" && optional {
				return next(c)
			}
			parts := strings.Fields(header)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || secret == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "A valid Bearer token is required"})
			}
			token, err := jwt.Parse(parts[1], func(*jwt.Token) (interface{}, error) { return []byte(secret), nil }, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
			if err != nil || !token.Valid {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid or expired token"})
			}
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid token claims"})
			}
			id, ok := claims["sub"].(string)
			if !ok || strings.TrimSpace(id) == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid token subject"})
			}
			ctx, cancel := context.WithTimeout(c.Request().Context(), 3*time.Second)
			defer cancel()
			user, err := lookup(ctx, id)
			if err != nil {
				return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "Authentication is temporarily unavailable"})
			}
			if user == nil || user.ID != id {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Account no longer available"})
			}
			if user.IsSuspended {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "Account is suspended"})
			}
			role := user.Role
			if role == "" {
				role = "user"
			}
			c.Set("user_id", user.ID)
			c.Set("did", user.DID)
			c.Set("username", user.Username)
			c.Set("role", role)
			return next(c)
		}
	}
}
