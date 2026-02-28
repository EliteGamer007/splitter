package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"splitter/internal/db"
	"testing"
)

func registerUserForDeviceTest(t *testing.T, username, email string) (userID, did, token string) {
	t.Helper()

	registerBody, _ := json.Marshal(map[string]string{
		"username": username,
		"email":    email,
		"password": "password123!",
	})

	registerReq, _ := http.NewRequest(http.MethodPost, TestServer.URL+"/api/v1/auth/register", bytes.NewBuffer(registerBody))
	registerReq.Header.Set("Content-Type", "application/json")
	registerResp, err := (&http.Client{}).Do(registerReq)
	if err != nil {
		t.Fatalf("register request failed: %v", err)
	}
	defer registerResp.Body.Close()

	if registerResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected register status 201, got %d", registerResp.StatusCode)
	}

	var registerResult map[string]interface{}
	if err := json.NewDecoder(registerResp.Body).Decode(&registerResult); err != nil {
		t.Fatalf("decode register response failed: %v", err)
	}

	token, _ = registerResult["token"].(string)
	userMap, ok := registerResult["user"].(map[string]interface{})
	if !ok {
		t.Fatalf("register response missing user object")
	}

	userID, _ = userMap["id"].(string)
	did, _ = userMap["did"].(string)
	if userID == "" || did == "" || token == "" {
		t.Fatalf("register response missing id/did/token")
	}

	return userID, did, token
}

func TestMultiDeviceAuthorizationAndEncryptedEnvelopeStorage(t *testing.T) {
	cleanup := SetupTestEnv(t)
	defer cleanup()

	aliceID, aliceDID, aliceToken := registerUserForDeviceTest(t, "alice_devices", "alice_devices@example.com")
	bobID, _, _ := registerUserForDeviceTest(t, "bob_devices", "bob_devices@example.com")

	if aliceID == bobID {
		t.Fatalf("alice and bob should be different users")
	}

	requestDevice := func(deviceID, label, key string, expectedStatus int) {
		t.Helper()
		body, _ := json.Marshal(map[string]string{
			"device_id":             deviceID,
			"device_label":          label,
			"encryption_public_key": key,
		})
		req, _ := http.NewRequest(http.MethodPost, TestServer.URL+"/api/v1/auth/devices/request", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+aliceToken)
		resp, err := (&http.Client{}).Do(req)
		if err != nil {
			t.Fatalf("request device key failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != expectedStatus {
			t.Fatalf("expected request device status %d, got %d", expectedStatus, resp.StatusCode)
		}
	}

	requestDevice("alice-primary", "Alice Laptop", "alice-pk-1", http.StatusCreated)
	requestDevice("alice-phone", "Alice Phone", "alice-pk-2", http.StatusAccepted)

	approveBody, _ := json.Marshal(map[string]string{"approver_device_id": "alice-primary"})
	approveReq, _ := http.NewRequest(http.MethodPost, TestServer.URL+"/api/v1/auth/devices/alice-phone/approve", bytes.NewBuffer(approveBody))
	approveReq.Header.Set("Content-Type", "application/json")
	approveReq.Header.Set("Authorization", "Bearer "+aliceToken)
	approveResp, err := (&http.Client{}).Do(approveReq)
	if err != nil {
		t.Fatalf("approve device key failed: %v", err)
	}
	defer approveResp.Body.Close()
	if approveResp.StatusCode != http.StatusOK {
		t.Fatalf("expected approve status 200, got %d", approveResp.StatusCode)
	}

	listReq, _ := http.NewRequest(http.MethodGet, TestServer.URL+"/api/v1/auth/devices", nil)
	listReq.Header.Set("Authorization", "Bearer "+aliceToken)
	listResp, err := (&http.Client{}).Do(listReq)
	if err != nil {
		t.Fatalf("list device keys failed: %v", err)
	}
	defer listResp.Body.Close()
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected list status 200, got %d", listResp.StatusCode)
	}

	var listResult map[string]interface{}
	if err := json.NewDecoder(listResp.Body).Decode(&listResult); err != nil {
		t.Fatalf("decode list response failed: %v", err)
	}
	if int(listResult["count"].(float64)) != 2 {
		t.Fatalf("expected 2 device keys, got %v", listResult["count"])
	}

	publicReq, _ := http.NewRequest(http.MethodGet, TestServer.URL+"/api/v1/dids/"+url.PathEscape(aliceDID)+"/device-keys", nil)
	publicResp, err := (&http.Client{}).Do(publicReq)
	if err != nil {
		t.Fatalf("public device keys request failed: %v", err)
	}
	defer publicResp.Body.Close()
	if publicResp.StatusCode != http.StatusOK {
		t.Fatalf("expected public list status 200, got %d", publicResp.StatusCode)
	}

	var publicResult map[string]interface{}
	if err := json.NewDecoder(publicResp.Body).Decode(&publicResult); err != nil {
		t.Fatalf("decode public list response failed: %v", err)
	}
	if int(publicResult["count"].(float64)) != 2 {
		t.Fatalf("expected 2 approved device keys, got %v", publicResult["count"])
	}

	sendBody, _ := json.Marshal(map[string]interface{}{
		"recipient_id": bobID,
		"content":      "",
		"ciphertext":   "federated-ciphertext-payload",
		"encrypted_keys": map[string]string{
			"alice-primary": "enc-for-primary",
			"alice-phone":   "enc-for-phone",
		},
	})
	sendReq, _ := http.NewRequest(http.MethodPost, TestServer.URL+"/api/v1/messages/send", bytes.NewBuffer(sendBody))
	sendReq.Header.Set("Content-Type", "application/json")
	sendReq.Header.Set("Authorization", "Bearer "+aliceToken)
	sendResp, err := (&http.Client{}).Do(sendReq)
	if err != nil {
		t.Fatalf("send message request failed: %v", err)
	}
	defer sendResp.Body.Close()
	if sendResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected send status 201, got %d", sendResp.StatusCode)
	}

	var sendResult map[string]interface{}
	if err := json.NewDecoder(sendResp.Body).Decode(&sendResult); err != nil {
		t.Fatalf("decode send response failed: %v", err)
	}

	messageObj, ok := sendResult["message"].(map[string]interface{})
	if !ok {
		t.Fatalf("send response missing message object")
	}
	messageID, _ := messageObj["id"].(string)
	if messageID == "" {
		t.Fatalf("send response missing message id")
	}

	var encryptedKeysText string
	if err := db.DB.QueryRow(
		context.Background(),
		"SELECT COALESCE(encrypted_keys::text, '{}') FROM messages WHERE id = $1",
		messageID,
	).Scan(&encryptedKeysText); err != nil {
		t.Fatalf("failed to query encrypted_keys from DB: %v", err)
	}

	if encryptedKeysText == "{}" {
		t.Fatalf("expected encrypted_keys JSON to be stored, got empty object")
	}
}
