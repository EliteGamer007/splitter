package middleware

import (
	"context"
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"net/http/httptest"
	"splitter/internal/models"
	"testing"
	"time"
)

func TestAuthRejectsIncompleteAndWrongAlgorithmTokens(t *testing.T) {
	for _, tc := range []struct {
		name   string
		claims jwt.MapClaims
		method jwt.SigningMethod
	}{
		{"missing-sub", jwt.MapClaims{"exp": time.Now().Add(time.Hour).Unix()}, jwt.SigningMethodHS256},
		{"missing-exp", jwt.MapClaims{"sub": "alice"}, jwt.SigningMethodHS256},
		{"alternate-hmac", jwt.MapClaims{"sub": "alice", "exp": time.Now().Add(time.Hour).Unix()}, jwt.SigningMethodHS384},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := jwt.NewWithClaims(tc.method, tc.claims).SignedString([]byte("test-secret"))
			if err != nil {
				t.Fatal(err)
			}
			e := echo.New()
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set("Authorization", "Bearer "+raw)
			c := e.NewContext(req, rec)
			called := false
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("invalid token reached database instead of rejection: %v", r)
					}
				}()
				if err := AuthMiddleware("test-secret")(func(echo.Context) error { called = true; return nil })(c); err != nil {
					e.HTTPErrorHandler(err, c)
				}
			}()
			if called || rec.Code != 401 {
				t.Errorf("invalid token passed auth: next=%v status=%d", called, rec.Code)
			}
		})
	}
}

func TestAuthUsesCurrentAccountAndOptionalIdentity(t *testing.T) {
	for _, optional := range []bool{false, true} {
		for _, kind := range []string{"demoted", "suspended", "deleted", "database-error"} {
			t.Run(kind+map[bool]string{true: "-optional", false: "-required"}[optional], func(t *testing.T) {
				raw, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "alice", "role": "admin", "did": "stale", "exp": time.Now().Add(time.Hour).Unix()}).SignedString([]byte("test-secret"))
				lookup := func(_ context.Context, id string) (*models.User, error) {
					if id != "alice" {
						t.Errorf("lookup wrong identity %q", id)
					}
					if kind == "database-error" {
						return nil, errors.New("database offline")
					}
					if kind == "deleted" {
						return nil, nil
					}
					return &models.User{ID: "alice", DID: "did:alice", Username: "alice", Role: "user", IsSuspended: kind == "suspended"}, nil
				}
				middleware := AuthMiddleware("test-secret", lookup)
				if optional {
					middleware = OptionalAuthMiddleware("test-secret", lookup)
				}
				e := echo.New()
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("Authorization", "Bearer "+raw)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)
				called := false
				func() {
					defer func() {
						if r := recover(); r != nil {
							t.Errorf("account lookup bypassed: %v", r)
						}
					}()
					if err := middleware(func(c echo.Context) error {
						called = true
						if c.Get("user_id") != "alice" || c.Get("did") != "did:alice" || c.Get("role") != "user" {
							t.Errorf("current account context missing or stale: id=%v did=%v role=%v", c.Get("user_id"), c.Get("did"), c.Get("role"))
						}
						return nil
					})(c); err != nil {
						e.HTTPErrorHandler(err, c)
					}
				}()
				if kind == "demoted" {
					if !called {
						t.Errorf("valid account blocked: %d", rec.Code)
					}
				} else if called || rec.Code < 400 {
					t.Errorf("unavailable account allowed downstream: called=%v status=%d", called, rec.Code)
				}
			})
		}
	}
}
