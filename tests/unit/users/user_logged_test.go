// Package users_test — logger bridge for unit tests.
package users_test

import (
	"splitter/internal/models"
	"strings"
	"testing"
	"time"

	"splitter/tests/testlogger"
)

func TestUserCreateValidationLogged(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "unit", start, testErr) }()

	tests := []struct {
		name    string
		user    models.UserCreate
		wantErr bool
		errMsg  string
	}{
		{"Valid User", models.UserCreate{Username: "validuser", Email: "test@example.com", Password: "securepassword"}, false, ""},
		{"Short Username", models.UserCreate{Username: "ab", Email: "test@example.com", Password: "securepassword"}, true, "username must be between 3 and 50 characters"},
		{"Invalid Email", models.UserCreate{Username: "validuser", Email: "notanemail", Password: "securepassword"}, true, "invalid email format"},
		{"Short Password", models.UserCreate{Username: "validuser", Email: "test@example.com", Password: "short"}, true, "password must be at least 8 characters"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := tt.user.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Validate() error = %v, expected to contain %v", err, tt.errMsg)
			}
		})
	}
}
