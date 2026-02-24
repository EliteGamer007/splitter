package integration

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

/*
KEY ROTATION INTEGRATION TEST SUMMARY (Story 4.3):
- What is tested: POST /api/v1/auth/rotate-key and GET /api/v1/auth/key-history
- Test cases:
    1. Happy path: valid rotation signed by current key succeeds
    2. Replay attack: reusing a nonce is rejected with 409
    3. Expired timestamp: request older than 5 minutes rejected with 401
    4. Invalid signature: wrong signature rejected with 401
    5. Password-only user: no public_key → rejected with 400
- Why this matters: Ensures key rotation is secure, atomic, and replay-proof.
*/

// registerWithKey registers a user and includes an Ed25519 public key.
// Returns the JWT token and the user's ID.
func registerWithKey(t *testing.T, username, email, password string, pubKey ed25519.PublicKey) (token, userID string) {
	t.Helper()
	pubKeyB64 := base64.StdEncoding.EncodeToString(pubKey)

	body, _ := json.Marshal(map[string]string{
		"username":   username,
		"email":      email,
		"password":   password,
		"public_key": pubKeyB64,
		"did":        fmt.Sprintf("did:splitter:%s", username),
	})
	req, _ := http.NewRequest(http.MethodPost, TestServer.URL+"/api/v1/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		t.Fatalf("registerWithKey: request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("registerWithKey: expected 201, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	token, _ = result["token"].(string)
	if userMap, ok := result["user"].(map[string]interface{}); ok {
		userID, _ = userMap["id"].(string)
	}
	return token, userID
}

// buildRotationRequest creates and signs a key rotation request.
func buildRotationRequest(t *testing.T, privKey ed25519.PrivateKey, newPubKey ed25519.PublicKey, nonce string, timestamp int64) map[string]interface{} {
	t.Helper()
	newPubKeyB64 := base64.StdEncoding.EncodeToString(newPubKey)
	message := fmt.Sprintf("%s|%s|%d", newPubKeyB64, nonce, timestamp)
	sig := ed25519.Sign(privKey, []byte(message))
	sigB64 := base64.StdEncoding.EncodeToString(sig)

	return map[string]interface{}{
		"new_public_key": newPubKeyB64,
		"signature":      sigB64,
		"nonce":          nonce,
		"timestamp":      timestamp,
	}
}

// doRotateKey sends a POST /api/v1/auth/rotate-key request and returns the response.
func doRotateKey(t *testing.T, token string, body map[string]interface{}) *http.Response {
	t.Helper()
	bodyBytes, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, TestServer.URL+"/api/v1/auth/rotate-key", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		t.Fatalf("doRotateKey: request failed: %v", err)
	}
	return resp
}

func TestKeyRotation(t *testing.T) {
	cleanup := SetupTestEnv(t)
	defer cleanup()

	t.Run("Happy Path - Valid Rotation", func(t *testing.T) {
		// Generate initial key pair
		oldPub, oldPriv, _ := ed25519.GenerateKey(nil)
		// Generate new key pair
		newPub, _, _ := ed25519.GenerateKey(nil)

		// Register user with initial key
		token, _ := registerWithKey(t,
			"keyrotation_happy",
			"keyrotation_happy@test.com",
			"securePass123!",
			oldPub,
		)

		// Build valid rotation request
		nonce := "test-nonce-happy-001"
		ts := time.Now().Unix()
		reqBody := buildRotationRequest(t, oldPriv, newPub, nonce, ts)

		resp := doRotateKey(t, token, reqBody)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			var errBody map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&errBody)
			t.Fatalf("Expected 200, got %d: %v", resp.StatusCode, errBody)
		}

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		if result["message"] != "Key rotated successfully" {
			t.Errorf("Unexpected message: %v", result["message"])
		}
		newPubKeyB64 := base64.StdEncoding.EncodeToString(newPub)
		if result["new_public_key"] != newPubKeyB64 {
			t.Errorf("new_public_key mismatch in response")
		}

		// Verify key history shows 1 entry
		histReq, _ := http.NewRequest(http.MethodGet, TestServer.URL+"/api/v1/auth/key-history", nil)
		histReq.Header.Set("Authorization", "Bearer "+token)
		histResp, _ := (&http.Client{}).Do(histReq)
		defer histResp.Body.Close()

		if histResp.StatusCode != http.StatusOK {
			t.Errorf("key-history: expected 200, got %d", histResp.StatusCode)
		}
		var histResult map[string]interface{}
		json.NewDecoder(histResp.Body).Decode(&histResult)
		if count, ok := histResult["count"].(float64); !ok || int(count) != 1 {
			t.Errorf("Expected 1 key history entry, got %v", histResult["count"])
		}
	})

	t.Run("Replay Attack - Nonce Reuse Rejected", func(t *testing.T) {
		oldPub, oldPriv, _ := ed25519.GenerateKey(nil)
		newPub, _, _ := ed25519.GenerateKey(nil)
		newPub2, _, _ := ed25519.GenerateKey(nil)

		token, _ := registerWithKey(t, "keyrotation_replay", "keyrotation_replay@test.com", "securePass123!", oldPub)

		nonce := "test-nonce-replay-001"
		ts := time.Now().Unix()

		// First rotation — should succeed
		reqBody := buildRotationRequest(t, oldPriv, newPub, nonce, ts)
		resp1 := doRotateKey(t, token, reqBody)
		resp1.Body.Close()
		if resp1.StatusCode != http.StatusOK {
			t.Fatalf("First rotation should succeed, got %d", resp1.StatusCode)
		}

		// Second rotation with the SAME nonce — must be rejected
		reqBody2 := buildRotationRequest(t, oldPriv, newPub2, nonce, ts)
		resp2 := doRotateKey(t, token, reqBody2)
		defer resp2.Body.Close()

		if resp2.StatusCode != http.StatusConflict {
			t.Errorf("Replay attack: expected 409, got %d", resp2.StatusCode)
		}
	})

	t.Run("Expired Timestamp - Rejected", func(t *testing.T) {
		oldPub, oldPriv, _ := ed25519.GenerateKey(nil)
		newPub, _, _ := ed25519.GenerateKey(nil)

		token, _ := registerWithKey(t, "keyrotation_expired", "keyrotation_expired@test.com", "securePass123!", oldPub)

		// Use a timestamp 10 minutes in the past
		staleTS := time.Now().Add(-10 * time.Minute).Unix()
		nonce := "test-nonce-expired-001"
		reqBody := buildRotationRequest(t, oldPriv, newPub, nonce, staleTS)

		resp := doRotateKey(t, token, reqBody)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("Stale timestamp: expected 401, got %d", resp.StatusCode)
		}
	})

	t.Run("Invalid Signature - Rejected", func(t *testing.T) {
		oldPub, _, _ := ed25519.GenerateKey(nil)
		newPub, _, _ := ed25519.GenerateKey(nil)

		token, _ := registerWithKey(t, "keyrotation_badsig", "keyrotation_badsig@test.com", "securePass123!", oldPub)

		// Sign with a DIFFERENT (wrong) private key
		_, wrongPriv, _ := ed25519.GenerateKey(nil)
		nonce := "test-nonce-badsig-001"
		ts := time.Now().Unix()
		reqBody := buildRotationRequest(t, wrongPriv, newPub, nonce, ts)

		resp := doRotateKey(t, token, reqBody)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("Bad signature: expected 401, got %d", resp.StatusCode)
		}
	})

	t.Run("Password-Only User - No Public Key", func(t *testing.T) {
		// Register WITHOUT a public_key (password-only user)
		body, _ := json.Marshal(map[string]string{
			"username": "keyrotation_pwonly",
			"email":    "keyrotation_pwonly@test.com",
			"password": "securePass123!",
		})
		req, _ := http.NewRequest(http.MethodPost, TestServer.URL+"/api/v1/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := (&http.Client{}).Do(req)

		var regResult map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&regResult)
		resp.Body.Close()
		token, _ := regResult["token"].(string)

		// Try to rotate — should get 400 (no public key)
		somePub, somePriv, _ := ed25519.GenerateKey(nil)
		nonce := "test-nonce-pwonly-001"
		ts := time.Now().Unix()
		reqBody := buildRotationRequest(t, somePriv, somePub, nonce, ts)

		rotResp := doRotateKey(t, token, reqBody)
		defer rotResp.Body.Close()

		if rotResp.StatusCode != http.StatusBadRequest {
			t.Errorf("Password-only user: expected 400, got %d", rotResp.StatusCode)
		}
	})
}
