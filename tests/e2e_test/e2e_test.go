package main_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"splitter/tests/testlogger"
)

const e2eBaseURL = "http://localhost:8000/api/v1"

func doE2EJSON(method, url string, body interface{}, token string) (int, map[string]interface{}, error) {
	var reqBody *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(b)
	} else {
		reqBody = &bytes.Buffer{}
	}
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	return resp.StatusCode, result, nil
}

func skipIfServerDown(t *testing.T) {
	t.Helper()
	_, _, err := doE2EJSON("GET", e2eBaseURL+"/health", nil, "")
	if err != nil {
		t.Skipf("Server not reachable (skip): %v", err)
	}
}

// TestE2EHealthCheck verifies the health endpoint is reachable.
func TestE2EHealthCheck(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "e2e", start, testErr) }()
	skipIfServerDown(t)
	code, _, _ := doE2EJSON("GET", e2eBaseURL+"/health", nil, "")
	if code != 200 {
		t.Errorf("Health check: expected 200, got %d", code)
	}
}

// TestE2ERegisterAndLogin simulates a full register + login flow.
func TestE2ERegisterAndLogin(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "e2e", start, testErr) }()
	skipIfServerDown(t)

	username := fmt.Sprintf("e2e_%d", time.Now().UnixNano())
	code, res, err := doE2EJSON("POST", e2eBaseURL+"/auth/register", map[string]string{
		"username": username, "email": username + "@test.local", "password": "password123!",
	}, "")
	if err != nil {
		testErr = err
		t.Fatalf("Register failed: %v", err)
	}
	if code != 200 && code != 201 {
		t.Fatalf("Register: expected 201, got %d: %v", code, res)
	}
	token, _ := res["token"].(string)
	if token == "" {
		t.Fatal("Register response missing token")
	}
	code, res, _ = doE2EJSON("POST", e2eBaseURL+"/auth/login", map[string]string{
		"username": username, "password": "password123!",
	}, "")
	if code != 200 {
		t.Fatalf("Login: expected 200, got %d: %v", code, res)
	}
}

// TestE2EUnauthenticatedAccessRejected verifies protected endpoints require auth.
func TestE2EUnauthenticatedAccessRejected(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "e2e", start, testErr) }()
	skipIfServerDown(t)

	code, _, _ := doE2EJSON("GET", e2eBaseURL+"/users/me", nil, "")
	if code != 401 && code != 400 {
		t.Errorf("/users/me without auth: expected 400/401, got %d", code)
	}
}

// TestE2EDuplicateRegistrationRejected verifies duplicate usernames are rejected.
func TestE2EDuplicateRegistrationRejected(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "e2e", start, testErr) }()
	skipIfServerDown(t)

	username := fmt.Sprintf("dup_%d", time.Now().UnixNano())
	email := username + "@test.local"
	doE2EJSON("POST", e2eBaseURL+"/auth/register", map[string]string{
		"username": username, "email": email, "password": "pass123!",
	}, "")
	code, _, _ := doE2EJSON("POST", e2eBaseURL+"/auth/register", map[string]string{
		"username": username, "email": email, "password": "pass123!",
	}, "")
	if code != 400 && code != 409 && code != 500 {
		t.Errorf("Duplicate register: expected 400/409/500, got %d", code)
	}
}

// TestE2EStoryFeedEndpoint verifies the story feed endpoint responds correctly.
func TestE2EStoryFeedEndpoint(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "e2e", start, testErr) }()
	skipIfServerDown(t)

	code, res, err := doE2EJSON("GET", e2eBaseURL+"/stories/feed", nil, "")
	if err != nil {
		testErr = err
		t.Fatalf("Request failed: %v", err)
	}
	if code != 200 {
		t.Errorf("Story feed: expected 200, got %d: %v", code, res)
	}
	if _, ok := res["stories"]; !ok {
		t.Error("Story feed response missing 'stories' key")
	}
}

// TestE2EPublicFeed verifies the public post feed is accessible without auth.
func TestE2EPublicFeed(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "e2e", start, testErr) }()
	skipIfServerDown(t)

	code, _, err := doE2EJSON("GET", e2eBaseURL+"/posts/public", nil, "")
	if err != nil {
		testErr = err
		t.Fatalf("Request failed: %v", err)
	}
	if code != 200 {
		t.Errorf("Public feed: expected 200, got %d", code)
	}
}
