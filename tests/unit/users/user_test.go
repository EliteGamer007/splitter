package users_test

import (
	"splitter/internal/models"
	"strings"
	"testing"
)

/*
WHY THIS TEST EXISTS:
- Ensures user input is validated before reaching the database.
- Prevents bad data (short passwords, invalid emails) from entering the system.

EXPECTED BEHAVIOR:
- Valid users pass validation.
- Users with short usernames, invalid emails, or short passwords fail validation.

TEST RESULT SUMMARY:
- Passed: Valid user, invalid username, invalid email, short password cases.
- Failed: None
- Limitations: Regex for email is basic.
*/

func TestUserCreateValidation(t *testing.T) {
	tests := []struct {
		name    string
		user    models.UserCreate
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid User",
			user: models.UserCreate{
				Username: "validuser",
				Email:    "test@example.com",
				Password: "securepassword",
			},
			wantErr: false,
		},
		{
			name: "Short Username",
			user: models.UserCreate{
				Username: "ab",
				Email:    "test@example.com",
				Password: "securepassword",
			},
			wantErr: true,
			errMsg:  "username must be between 3 and 50 characters",
		},
		{
			name: "Invalid Email",
			user: models.UserCreate{
				Username: "validuser",
				Email:    "notanemail",
				Password: "securepassword",
			},
			wantErr: true,
			errMsg:  "invalid email format",
		},
		{
			name: "Short Password",
			user: models.UserCreate{
				Username: "validuser",
				Email:    "test@example.com",
				Password: "short",
			},
			wantErr: true,
			errMsg:  "password must be at least 8 characters",
		},
	}

	for _, tt := range tests {
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
