package handlers

import (
	"context"
	"errors"
	"github.com/labstack/echo/v4"
	"net/http/httptest"
	"strings"
	"testing"
)

type deniedAdminUsers struct {
	adminUserRepository
	writes int
}

func (r *deniedAdminUsers) UpdateUserRole(context.Context, string, string) error {
	r.writes++
	return errors.New("must not write")
}
func (r *deniedAdminUsers) SuspendUser(context.Context, string) error {
	r.writes++
	return errors.New("must not write")
}
func TestDeniedAdminActionsCannotMutate(t *testing.T) {
	for _, role := range []string{"user", "moderator", ""} {
		t.Run(role, func(t *testing.T) {
			repo := &deniedAdminUsers{}
			h := &AdminHandler{userRepo: repo}
			e := echo.New()
			req := httptest.NewRequest("POST", "/", strings.NewReader(`{"role":"admin"}`))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.Set("role", role)
			c.Set("user_id", "attacker")
			c.SetParamNames("id")
			c.SetParamValues("victim")
			if err := h.UpdateUserRole(c); err != nil {
				e.HTTPErrorHandler(err, c)
			}
			if rec.Code != 403 || repo.writes != 0 {
				t.Fatalf("denied role %q: status %d downstream writes %d", role, rec.Code, repo.writes)
			}
		})
	}
}
func TestDeniedModeratorActionCannotMutate(t *testing.T) {
	repo := &deniedAdminUsers{}
	h := &AdminHandler{userRepo: repo}
	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(httptest.NewRequest("POST", "/", nil), rec)
	c.Set("role", "user")
	c.Set("user_id", "attacker")
	c.SetParamNames("id")
	c.SetParamValues("victim")
	if err := h.SuspendUser(c); err != nil {
		e.HTTPErrorHandler(err, c)
	}
	if rec.Code != 403 || repo.writes != 0 {
		t.Fatalf("status %d downstream writes %d", rec.Code, repo.writes)
	}
}
