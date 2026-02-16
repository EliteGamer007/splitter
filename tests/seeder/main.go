package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const (
	baseURL = "http://localhost:8000/api/v1"
)

type User struct {
	ID                   string
	Username             string
	Password             string
	PublicKey            string // Signing Key (Base64 SPKI)
	EncryptionPublicKey  string // Encryption Key (Base64 SPKI)
	PrivateKey           *ecdsa.PrivateKey
	EncryptionPrivateKey *ecdh.PrivateKey
	Token                string
}

func main() {
	log.Println("🌱 Starting Seeder & E2EE Verifier...")

	users := make([]*User, 0)

	// Create Admin + 5 Users
	roles := []string{"admin", "user", "user", "user", "user", "user"}

	for i, role := range roles {
		username := fmt.Sprintf("user_%d_%d", i, time.Now().UnixNano())
		if role == "admin" {
			username = fmt.Sprintf("admin_%d", time.Now().UnixNano())
		}
		password := "password123"

		user, err := registerUser(username, password)
		if err != nil {
			log.Fatalf("❌ Failed to register %s: %v", username, err)
		}

		// Login to get token
		token, err := loginUser(username, password)
		if err != nil {
			log.Fatalf("❌ Failed to login %s: %v", username, err)
		}
		user.Token = token
		user.Password = password

		log.Printf("✅ Created %s (%s) - Token acquired", username, role)
		users = append(users, user)
		time.Sleep(500 * time.Millisecond) // Slight delay to be safe
	}

	log.Println("✨ All users created successfully!")
	log.Println("🔒 Starting E2E Encryption Test...")

	// Test E2EE between User 0 (Alice) and User 1 (Bob)
	alice := users[0]
	bob := users[1]

	// 1. Alice derives shared secret to talk to Bob
	// Need Bob's public key (stored in ALice's memory for now, usually fetched via API)

	// In real app, Alice fetches Bob's profile to get EncryptionPublicKey string
	// We have it in `bob.EncryptionPublicKey` (Base64 SPKI)

	bobPubKey, err := parsePublicKey(bob.EncryptionPublicKey)
	if err != nil {
		log.Fatalf("❌ Failed to parse Bob's public key: %v", err)
	}

	aliceSharedSecret, err := alice.EncryptionPrivateKey.ECDH(bobPubKey)
	if err != nil {
		log.Fatalf("❌ Failed to derive Alice's shared secret: %v", err)
	}
	log.Printf("🔑 Alice derived shared secret")

	// 2. Alice encrypts message for Bob
	messageContent := "Hello Bob, this is a secret message! 🤫"
	iv, ciphertext, err := encryptMessage(messageContent, aliceSharedSecret)
	if err != nil {
		log.Fatalf("❌ Encryption failed: %v", err)
	}

	// Prepare payload: {"c": ciphertext_base64, "iv": iv_base64}
	// Note: In frontend crypto.ts, we used different format?
	// Frontend crypto.ts `encryptMessage` returns { ciphertext: string, iv: string } (Base64).
	// Backend API expects `ciphertext` string.
	// We decided to store JSON in that string.

	payloadData := map[string]string{
		"c":  base64.StdEncoding.EncodeToString(ciphertext),
		"iv": base64.StdEncoding.EncodeToString(iv),
	}
	payloadBytes, _ := json.Marshal(payloadData)
	finalCiphertext := string(payloadBytes)

	log.Printf("📝 Encrypted message: %s...", finalCiphertext[:20])

	// 3. Alice sends message to Bob via API
	err = sendMessage(alice, bob.ID, "🔒 Encrypted Message", finalCiphertext)
	if err != nil {
		log.Fatalf("❌ Failed to send message: %v", err)
	}
	log.Printf("📨 Message sent from Alice to Bob")

	// 4. Bob fetches message (simulated by sleep then fetch)
	time.Sleep(1 * time.Second)

	// Helper: Get thread with Alice
	threadID, err := getThreadID(bob, alice.ID)
	if err != nil {
		log.Fatalf("❌ Failed to get thread ID: %v", err)
	}

	msgs, err := getThreadMessages(bob, threadID)
	if err != nil {
		log.Fatalf("❌ Failed to get messages: %v", err)
	}

	if len(msgs) == 0 {
		log.Fatalf("❌ No messages found for Bob")
	}

	lastMsg := msgs[len(msgs)-1] // Assuming last message is the one

	// 5. Bob decrypts message
	log.Printf("📥 Received Ciphertext: %s", lastMsg.Ciphertext)

	// Parse JSON from ciphertext
	var receivedPayload map[string]string
	var temp = lastMsg.Ciphertext
	var decoded = false

	for i := 0; i < 3; i++ {
		// Try to unmarshal into map
		if err := json.Unmarshal([]byte(temp), &receivedPayload); err == nil {
			decoded = true
			break
		}

		// If failed, try to unmarshal as string (unquote)
		var next string
		if err := json.Unmarshal([]byte(temp), &next); err == nil {
			temp = next
			log.Printf("⚠️ Unquoted ciphertext layer %d", i+1)
		} else {
			// Try manual unquote if standard unmarshal fails (sometimes quotes are weird)
			if len(temp) > 2 && temp[0] == '"' && temp[len(temp)-1] == '"' {
				temp = temp[1 : len(temp)-1]
				log.Printf("⚠️ Manually unquoted layer %d", i+1)
			} else {
				log.Printf("❌ Failed to unmarshal ciphertext at layer %d: %v. Content: %s", i, err, temp)
			}
		}
	}

	if !decoded {
		// one last hail mary - maybe it IS the map already but stringified weirdly?
		if err := json.Unmarshal([]byte(temp), &receivedPayload); err == nil {
			decoded = true
		} else {
			log.Fatalf("❌ Failed to decode ciphertext after 3 attempts. Final content: %s", temp)
		}
	}

	recIV, _ := base64.StdEncoding.DecodeString(receivedPayload["iv"])
	recCipher, _ := base64.StdEncoding.DecodeString(receivedPayload["c"])

	// Derive Bob's shared secret (should be same)
	alicePubKey, err := parsePublicKey(alice.EncryptionPublicKey)
	if err != nil {
		log.Fatalf("❌ Failed to parse Alice's public key: %v", err)
	}

	bobSharedSecret, err := bob.EncryptionPrivateKey.ECDH(alicePubKey)
	if err != nil {
		log.Fatalf("❌ Failed to derive Bob's shared secret: %v", err)
	}

	decryptedContent, err := decryptMessage(recCipher, recIV, bobSharedSecret)
	if err != nil {
		log.Fatalf("❌ Decryption failed: %v", err)
	}

	log.Printf("🔓 Decrypted message: %s", decryptedContent)

	if decryptedContent != messageContent {
		log.Fatalf("❌ Mismatch! Expected '%s', got '%s'", messageContent, decryptedContent)
	}

	log.Println("✅ E2E Encryption Verified Successfully! 🎉")
}

// --- Helpers ---

func registerUser(username, password string) (*User, error) {
	// Generate Keys

	// 1. Signing Key (ECDSA P-256)
	signKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	signPubKeyBytes, err := x509.MarshalPKIXPublicKey(&signKey.PublicKey)
	if err != nil {
		return nil, err
	}
	signPubKeyBase64 := base64.StdEncoding.EncodeToString(signPubKeyBytes)

	// 2. Encryption Key (ECDH P-256)
	encKey, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	encPubKeyBytes, err := x509.MarshalPKIXPublicKey(encKey.PublicKey())
	if err != nil {
		return nil, err
	}
	encPubKeyBase64 := base64.StdEncoding.EncodeToString(encPubKeyBytes)

	// Register via API
	data := map[string]interface{}{
		"username":              username,
		"password":              password,
		"email":                 username + "@example.com",
		"instance_domain":       "localhost",
		"public_key":            signPubKeyBase64,
		"encryption_public_key": encPubKeyBase64,
	}

	jsonData, _ := json.Marshal(data)
	resp, err := http.Post(baseURL+"/auth/register", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var res map[string]interface{}
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("failed to decode register response: %v. Body: %s", err, string(body))
	}

	// Extract ID (assuming backend returns user object)
	// Response format: { "user": { "id": "...", ... }, "token": "..." }
	userMap, ok := res["user"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("register response missing user field. Body: %s", string(body))
	}
	id, ok := userMap["id"].(string)
	if !ok {
		return nil, fmt.Errorf("user ID not found or not string. Body: %s", string(body))
	}

	return &User{
		ID:                   id,
		Username:             username,
		PublicKey:            signPubKeyBase64,
		EncryptionPublicKey:  encPubKeyBase64,
		PrivateKey:           signKey,
		EncryptionPrivateKey: encKey,
	}, nil
}

func loginUser(username, password string) (string, error) {
	data := map[string]string{
		"username": username,
		"password": password,
	}
	jsonData, _ := json.Marshal(data)
	resp, err := http.Post(baseURL+"/auth/login", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("login failed: %d", resp.StatusCode)
	}

	var res map[string]interface{}
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &res); err != nil {
		return "", fmt.Errorf("failed to decode login response: %v. Body: %s", err, string(body))
	}

	token, ok := res["token"].(string)
	if !ok {
		return "", fmt.Errorf("login response missing token. Body: %s", string(body))
	}
	return token, nil
}

func sendMessage(sender *User, recipientID, content, ciphertext string) error {
	data := map[string]string{
		"recipient_id": recipientID,
		"content":      content,
		"ciphertext":   ciphertext,
	}
	jsonData, _ := json.Marshal(data)

	req, _ := http.NewRequest("POST", baseURL+"/messages/send", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+sender.Token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func getThreadID(user *User, otherUserID string) (string, error) {
	// Need to fetch threads and find one with otherUserID
	req, _ := http.NewRequest("GET", baseURL+"/messages/threads", nil)
	req.Header.Set("Authorization", "Bearer "+user.Token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var res map[string][]map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&res)

	for _, thread := range res["threads"] {
		pa := thread["participant_a_id"].(string)
		pb := thread["participant_b_id"].(string)
		if pa == otherUserID || pb == otherUserID {
			return thread["id"].(string), nil
		}
	}
	return "", fmt.Errorf("thread not found")
}

func getThreadMessages(user *User, threadID string) ([]struct{ Ciphertext string }, error) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/messages/threads/%s?limit=10&offset=0", baseURL, threadID), nil)
	req.Header.Set("Authorization", "Bearer "+user.Token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var res struct {
		Messages []struct{ Ciphertext string } `json:"messages"`
	}
	json.NewDecoder(resp.Body).Decode(&res)
	return res.Messages, nil
}

// --- Crypto Helpers ---

func parsePublicKey(base64SPKI string) (*ecdh.PublicKey, error) {
	der, err := base64.StdEncoding.DecodeString(base64SPKI)
	if err != nil {
		return nil, err
	}
	pub, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return nil, err
	}
	ecdsaPub, ok := pub.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an ECDSA public key")
	}
	return ecdsaPub.ECDH()
}

func encryptMessage(plaintext string, sharedSecret []byte) ([]byte, []byte, error) {
	block, err := aes.NewCipher(sharedSecret)
	if err != nil {
		return nil, nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}

	iv := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, nil, err
	}

	ciphertext := aesgcm.Seal(nil, iv, []byte(plaintext), nil)
	return iv, ciphertext, nil
}

func decryptMessage(ciphertext, iv, sharedSecret []byte) (string, error) {
	block, err := aes.NewCipher(sharedSecret)
	if err != nil {
		return "", err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	plaintext, err := aesgcm.Open(nil, iv, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
