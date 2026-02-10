package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"splitter/internal/db"
	"testing"
)

/*
INTEGRATION TEST SUMMARY:
- What was tested: User Registration and Login flows.
- What passed:
    - Successful registration returns 201 and JWT.
    - Database reflects the new user.
    - Login with valid credentials returns 200 and JWT.
    - Login with invalid credentials returns 401.
- Any limitations: DID generation is automatic, not testing custom DIDs here.
- Why this test matters: Verifies the core entry point for all users.
*/

func TestAuthFlow(t *testing.T) {
	// 1. Setup
	cleanup := SetupTestEnv(t)
	defer cleanup()

	// Shared test data
	username := "integration_test_user"
	email := "test@example.com"
	password := "securePass123!"

	t.Run("Register User", func(t *testing.T) {
		// Prepare Request
		reqBody, _ := json.Marshal(map[string]string{
			"username": username,
			"email":    email,
			"password": password,
		})

		req, err := http.NewRequest(http.MethodPost, TestServer.URL+"/api/v1/auth/register", bytes.NewBuffer(reqBody))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		// Execute Request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}
		defer resp.Body.Close()

		// Validate Response Status
		if resp.StatusCode != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", resp.StatusCode)
		}

		// Validate Response Body
		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if _, ok := result["token"]; !ok {
			t.Error("Response missing 'token'")
		}

		userMap, ok := result["user"].(map[string]interface{})
		if !ok {
			t.Error("Response missing 'user' object")
		} else {
			if userMap["username"] != username {
				t.Errorf("Expected username %s, got %v", username, userMap["username"])
			}
		}

		// Validate Database State
		var dbUsername string
		err = db.DB.QueryRow(context.Background(), "SELECT username FROM users WHERE email=$1", email).Scan(&dbUsername)
		if err != nil {
			t.Errorf("Failed to find user in DB: %v", err)
		}
		if dbUsername != username {
			t.Errorf("DB username mismatch: expected %s, got %s", username, dbUsername)
		}
	})

	t.Run("Login User Success", func(t *testing.T) {
		// Prepare Request
		reqBody, _ := json.Marshal(map[string]string{
			"username": username, // Login allows username or email
			"password": password,
		})

		req, err := http.NewRequest(http.MethodPost, TestServer.URL+"/api/v1/auth/login", bytes.NewBuffer(reqBody))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		// Execute Request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}
		defer resp.Body.Close()

		// Validate Response Status
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		// Validate Token
		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}
		if _, ok := result["token"]; !ok {
			t.Error("Login response missing 'token'")
		}
	})

	t.Run("Login User Invalid Password", func(t *testing.T) {
		reqBody, _ := json.Marshal(map[string]string{
			"username": username,
			"password": "wrongpassword",
		})

		req, err := http.NewRequest(http.MethodPost, TestServer.URL+"/api/v1/auth/login", bytes.NewBuffer(reqBody))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", resp.StatusCode)
		}
	})
}
