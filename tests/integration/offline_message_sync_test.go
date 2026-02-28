package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

func registerAndLogin(t *testing.T, username, email, password string) (string, string) {
	t.Helper()

	registerBody, _ := json.Marshal(map[string]string{
		"username": username,
		"email":    email,
		"password": password,
	})

	registerReq, err := http.NewRequest(http.MethodPost, TestServer.URL+"/api/v1/auth/register", bytes.NewBuffer(registerBody))
	if err != nil {
		t.Fatalf("failed to create register request: %v", err)
	}
	registerReq.Header.Set("Content-Type", "application/json")

	registerResp, err := (&http.Client{}).Do(registerReq)
	if err != nil {
		t.Fatalf("failed to execute register request: %v", err)
	}
	defer registerResp.Body.Close()

	if registerResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected register status 201, got %d", registerResp.StatusCode)
	}

	var registerResult map[string]interface{}
	if err := json.NewDecoder(registerResp.Body).Decode(&registerResult); err != nil {
		t.Fatalf("failed to decode register response: %v", err)
	}

	userMap, ok := registerResult["user"].(map[string]interface{})
	if !ok {
		t.Fatalf("register response missing user object")
	}

	userID, _ := userMap["id"].(string)
	if userID == "" {
		t.Fatalf("register response missing user id")
	}

	loginBody, _ := json.Marshal(map[string]string{
		"username": username,
		"password": password,
	})

	loginReq, err := http.NewRequest(http.MethodPost, TestServer.URL+"/api/v1/auth/login", bytes.NewBuffer(loginBody))
	if err != nil {
		t.Fatalf("failed to create login request: %v", err)
	}
	loginReq.Header.Set("Content-Type", "application/json")

	loginResp, err := (&http.Client{}).Do(loginReq)
	if err != nil {
		t.Fatalf("failed to execute login request: %v", err)
	}
	defer loginResp.Body.Close()

	if loginResp.StatusCode != http.StatusOK {
		t.Fatalf("expected login status 200, got %d", loginResp.StatusCode)
	}

	var loginResult map[string]interface{}
	if err := json.NewDecoder(loginResp.Body).Decode(&loginResult); err != nil {
		t.Fatalf("failed to decode login response: %v", err)
	}

	token, _ := loginResult["token"].(string)
	if token == "" {
		t.Fatalf("login response missing token")
	}

	return userID, token
}

func TestOfflineMessageSync_IdempotentByClientMessageID(t *testing.T) {
	cleanup := SetupTestEnv(t)
	defer cleanup()

	aliceID, aliceToken := registerAndLogin(t, "alice_sync", "alice_sync@example.com", "password123!")
	bobID, _ := registerAndLogin(t, "bob_sync", "bob_sync@example.com", "password123!")

	if aliceID == bobID {
		t.Fatalf("alice and bob must be different users")
	}

	syncPayload := map[string]interface{}{
		"queued_messages": []map[string]interface{}{
			{
				"client_message_id": "offline-msg-001",
				"recipient_id":      bobID,
				"content":           "",
				"ciphertext":        "encrypted-payload-1",
				"client_created_at": "2026-03-01T10:00:00Z",
			},
		},
	}

	payloadBytes, _ := json.Marshal(syncPayload)

	firstReq, err := http.NewRequest(http.MethodPost, TestServer.URL+"/api/v1/messages/sync", bytes.NewBuffer(payloadBytes))
	if err != nil {
		t.Fatalf("failed to create first sync request: %v", err)
	}
	firstReq.Header.Set("Content-Type", "application/json")
	firstReq.Header.Set("Authorization", "Bearer "+aliceToken)

	firstResp, err := (&http.Client{}).Do(firstReq)
	if err != nil {
		t.Fatalf("failed to execute first sync request: %v", err)
	}
	defer firstResp.Body.Close()

	if firstResp.StatusCode != http.StatusOK {
		t.Fatalf("expected first sync status 200, got %d", firstResp.StatusCode)
	}

	var firstResult map[string]interface{}
	if err := json.NewDecoder(firstResp.Body).Decode(&firstResult); err != nil {
		t.Fatalf("failed to decode first sync response: %v", err)
	}

	createdOrDedup := int(firstResult["created_count"].(float64)) + int(firstResult["deduplicated_count"].(float64))
	if createdOrDedup != 1 {
		t.Fatalf("expected exactly one processed message on first sync, got created=%v deduplicated=%v", firstResult["created_count"], firstResult["deduplicated_count"])
	}
	if int(firstResult["failed_count"].(float64)) != 0 {
		t.Fatalf("expected failed_count=0 on first sync, got %v", firstResult["failed_count"])
	}

	secondReq, err := http.NewRequest(http.MethodPost, TestServer.URL+"/api/v1/messages/sync", bytes.NewBuffer(payloadBytes))
	if err != nil {
		t.Fatalf("failed to create second sync request: %v", err)
	}
	secondReq.Header.Set("Content-Type", "application/json")
	secondReq.Header.Set("Authorization", "Bearer "+aliceToken)

	secondResp, err := (&http.Client{}).Do(secondReq)
	if err != nil {
		t.Fatalf("failed to execute second sync request: %v", err)
	}
	defer secondResp.Body.Close()

	if secondResp.StatusCode != http.StatusOK {
		t.Fatalf("expected second sync status 200, got %d", secondResp.StatusCode)
	}

	var secondResult map[string]interface{}
	if err := json.NewDecoder(secondResp.Body).Decode(&secondResult); err != nil {
		t.Fatalf("failed to decode second sync response: %v", err)
	}

	if int(secondResult["created_count"].(float64)) != 0 {
		t.Fatalf("expected created_count=0 on duplicate sync, got %v", secondResult["created_count"])
	}
	if int(secondResult["deduplicated_count"].(float64)) != 1 {
		t.Fatalf("expected deduplicated_count=1 on duplicate sync, got %v", secondResult["deduplicated_count"])
	}
	if int(secondResult["failed_count"].(float64)) != 0 {
		t.Fatalf("expected failed_count=0 on duplicate sync, got %v", secondResult["failed_count"])
	}
}
