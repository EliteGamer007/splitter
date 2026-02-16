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
	"os"
	"time"
)

const baseURL = "http://localhost:8000/api/v1"

// ─── Counters ──────────────────────────────────────────────────────────────────
var passed, failed, total int

func check(name string, ok bool, detail string) {
	total++
	if ok {
		passed++
		log.Printf("  ✅ PASS: %s", name)
	} else {
		failed++
		log.Printf("  ❌ FAIL: %s — %s", name, detail)
	}
}

// ─── HTTP helpers ──────────────────────────────────────────────────────────────

func doJSON(method, url string, body interface{}, token string) (int, map[string]interface{}, error) {
	var reqBody io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(b)
	}
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(raw, &result)
	return resp.StatusCode, result, nil
}

// ─── Key helpers ───────────────────────────────────────────────────────────────

type TestUser struct {
	ID, Username, Token   string
	SignKey               *ecdsa.PrivateKey
	EncKey                *ecdh.PrivateKey
	SignPubB64, EncPubB64 string
}

func genKeys() (*ecdsa.PrivateKey, string, *ecdh.PrivateKey, string, error) {
	sk, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, "", nil, "", err
	}
	spki, _ := x509.MarshalPKIXPublicKey(&sk.PublicKey)
	signB64 := base64.StdEncoding.EncodeToString(spki)

	ek, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return nil, "", nil, "", err
	}
	epki, _ := x509.MarshalPKIXPublicKey(ek.PublicKey())
	encB64 := base64.StdEncoding.EncodeToString(epki)

	return sk, signB64, ek, encB64, nil
}

func registerAndLogin(username string) (*TestUser, error) {
	sk, signB64, ek, encB64, err := genKeys()
	if err != nil {
		return nil, err
	}
	pw := "password123"

	// Register
	code, res, err := doJSON("POST", baseURL+"/auth/register", map[string]interface{}{
		"username":              username,
		"email":                 username + "@test.local",
		"password":              pw,
		"instance_domain":       "localhost",
		"public_key":            signB64,
		"encryption_public_key": encB64,
	}, "")
	if err != nil {
		return nil, err
	}
	if code != 200 && code != 201 {
		return nil, fmt.Errorf("register %d: %v", code, res)
	}
	userMap, ok := res["user"].(map[string]interface{})
	if !ok || userMap == nil {
		return nil, fmt.Errorf("register response missing user: %v", res)
	}
	id, ok := userMap["id"].(string)
	if !ok {
		return nil, fmt.Errorf("register response user.id not string: %v", userMap["id"])
	}

	// Login
	code, res, err = doJSON("POST", baseURL+"/auth/login", map[string]string{
		"username": username,
		"password": pw,
	}, "")
	if err != nil {
		return nil, err
	}
	if code != 200 {
		return nil, fmt.Errorf("login %d: %v", code, res)
	}
	token, ok := res["token"].(string)
	if !ok {
		return nil, fmt.Errorf("login response missing token: %v", res)
	}

	return &TestUser{
		ID: id, Username: username, Token: token,
		SignKey: sk, EncKey: ek,
		SignPubB64: signB64, EncPubB64: encB64,
	}, nil
}

// ─── Crypto helpers ────────────────────────────────────────────────────────────

func parseECDHPub(b64 string) (*ecdh.PublicKey, error) {
	der, _ := base64.StdEncoding.DecodeString(b64)
	pub, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return nil, err
	}
	return pub.(*ecdsa.PublicKey).ECDH()
}

func aesEncrypt(plaintext string, secret []byte) (ivB64, ctB64 string, err error) {
	block, _ := aes.NewCipher(secret)
	gcm, _ := cipher.NewGCM(block)
	iv := make([]byte, 12)
	io.ReadFull(rand.Reader, iv)
	ct := gcm.Seal(nil, iv, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(iv),
		base64.StdEncoding.EncodeToString(ct), nil
}

func aesDecrypt(ctB64, ivB64 string, secret []byte) (string, error) {
	iv, _ := base64.StdEncoding.DecodeString(ivB64)
	ct, _ := base64.StdEncoding.DecodeString(ctB64)
	block, _ := aes.NewCipher(secret)
	gcm, _ := cipher.NewGCM(block)
	pt, err := gcm.Open(nil, iv, ct, nil)
	return string(pt), err
}

// ═══════════════════════════════════════════════════════════════════════════════
func main() {
	log.Println("═══════════════════════════════════════════════")
	log.Println("  Splitter Comprehensive Test Suite")
	log.Println("═══════════════════════════════════════════════")

	ts := fmt.Sprintf("%d", time.Now().UnixNano())

	// ── 1. Health Check ────────────────────────────────────────────────────
	log.Println("\n📋 [1] HEALTH CHECK")
	code, _, err := doJSON("GET", baseURL+"/health", nil, "")
	check("Health endpoint reachable", err == nil && code == 200, fmt.Sprintf("code=%d err=%v", code, err))

	// ── 2. Auth: Register two users ────────────────────────────────────────
	log.Println("\n📋 [2] AUTH — REGISTER + LOGIN")
	alice, err := registerAndLogin("alice_" + ts)
	check("Register + Login Alice", err == nil && alice != nil, fmt.Sprintf("%v", err))

	bob, err := registerAndLogin("bob_" + ts)
	check("Register + Login Bob", err == nil && bob != nil, fmt.Sprintf("%v", err))

	if alice == nil || bob == nil {
		log.Fatalf("🛑 Cannot continue without both users")
	}

	// ── 3. Auth: Duplicate register should fail ────────────────────────────
	code, _, _ = doJSON("POST", baseURL+"/auth/register", map[string]interface{}{
		"username": alice.Username, "email": alice.Username + "@test.local",
		"password": "password123", "instance_domain": "localhost",
	}, "")
	check("Duplicate register rejected", code == 400 || code == 409 || code == 500, fmt.Sprintf("code=%d", code))

	// ── 4. Auth: Bad login ─────────────────────────────────────────────────
	code, _, _ = doJSON("POST", baseURL+"/auth/login", map[string]string{
		"username": alice.Username, "password": "wrong",
	}, "")
	check("Wrong password rejected", code == 401 || code == 400, fmt.Sprintf("code=%d", code))

	// ── 5. User profile ────────────────────────────────────────────────────
	log.Println("\n📋 [3] USER PROFILE")
	code, res, _ := doJSON("GET", baseURL+"/users/me", nil, alice.Token)
	check("GET /users/me returns 200", code == 200, fmt.Sprintf("code=%d", code))
	if code == 200 {
		if user, ok := res["user"].(map[string]interface{}); ok && user != nil {
			check("Profile has encryption_public_key", user["encryption_public_key"] != nil && user["encryption_public_key"] != "", "missing key")
		} else {
			check("Profile has encryption_public_key", false, fmt.Sprintf("response format: %v", res))
		}
	}

	code, res, _ = doJSON("GET", baseURL+"/users/"+bob.ID, nil, "")
	check("GET /users/:id returns Bob", code == 200, fmt.Sprintf("code=%d", code))

	// Update profile
	code, _, _ = doJSON("PUT", baseURL+"/users/me", map[string]string{
		"display_name": "Alice Test", "bio": "Testing E2EE",
	}, alice.Token)
	check("PUT /users/me updates profile", code == 200, fmt.Sprintf("code=%d", code))

	// ── 6. Posts ────────────────────────────────────────────────────────────
	log.Println("\n📋 [4] POSTS — CREATE, READ, UPDATE, DELETE")
	// Create post (use JSON since no file)
	code, res, _ = doJSON("POST", baseURL+"/posts", map[string]string{
		"content": "Hello from Alice! #testing", "visibility": "public",
	}, alice.Token)
	// Posts may require multipart — let's try JSON first
	var postID string
	if code == 200 || code == 201 {
		check("Create post (Alice)", true, "")
		if p, ok := res["post"].(map[string]interface{}); ok {
			postID = p["id"].(string)
		}
	} else {
		// Try multipart
		check("Create post (Alice) — JSON failed, might need multipart", false, fmt.Sprintf("code=%d res=%v", code, res))
	}

	// Public feed
	code, res, _ = doJSON("GET", baseURL+"/posts/public?limit=5&offset=0", nil, "")
	check("GET /posts/public returns 200", code == 200, fmt.Sprintf("code=%d", code))

	if postID != "" {
		// Get single post
		code, _, _ = doJSON("GET", baseURL+"/posts/"+postID, nil, "")
		check("GET /posts/:id returns 200", code == 200, fmt.Sprintf("code=%d", code))

		// Update post
		code, _, _ = doJSON("PUT", baseURL+"/posts/"+postID, map[string]string{
			"content": "Updated post content",
		}, alice.Token)
		check("PUT /posts/:id updates post", code == 200, fmt.Sprintf("code=%d", code))
	}

	// ── 7. Follows ─────────────────────────────────────────────────────────
	log.Println("\n📋 [5] FOLLOWS")
	code, _, _ = doJSON("POST", baseURL+"/users/"+bob.ID+"/follow", nil, alice.Token)
	check("Alice follows Bob", code == 200 || code == 201, fmt.Sprintf("code=%d", code))

	code, res, _ = doJSON("GET", baseURL+"/users/"+bob.ID+"/stats", nil, "")
	check("Bob stats endpoint", code == 200, fmt.Sprintf("code=%d", code))

	code, res, _ = doJSON("GET", baseURL+"/users/"+bob.ID+"/followers?limit=10&offset=0", nil, "")
	check("Bob followers list", code == 200, fmt.Sprintf("code=%d", code))

	// Feed (Alice should see Bob's posts if Bob posted)
	code, _, _ = doJSON("GET", baseURL+"/posts/feed?limit=5&offset=0", nil, alice.Token)
	check("GET /posts/feed returns 200", code == 200, fmt.Sprintf("code=%d", code))

	// Unfollow
	code, _, _ = doJSON("DELETE", baseURL+"/users/"+bob.ID+"/follow", nil, alice.Token)
	check("Alice unfollows Bob", code == 200, fmt.Sprintf("code=%d", code))

	// ── 8. Interactions ────────────────────────────────────────────────────
	log.Println("\n📋 [6] INTERACTIONS (Like, Repost, Bookmark)")
	if postID != "" {
		code, _, _ = doJSON("POST", baseURL+"/posts/"+postID+"/like", nil, bob.Token)
		check("Bob likes Alice's post", code == 200 || code == 201, fmt.Sprintf("code=%d", code))

		code, _, _ = doJSON("DELETE", baseURL+"/posts/"+postID+"/like", nil, bob.Token)
		check("Bob unlikes Alice's post", code == 200, fmt.Sprintf("code=%d", code))

		code, _, _ = doJSON("POST", baseURL+"/posts/"+postID+"/repost", nil, bob.Token)
		check("Bob reposts Alice's post", code == 200 || code == 201, fmt.Sprintf("code=%d", code))

		code, _, _ = doJSON("DELETE", baseURL+"/posts/"+postID+"/repost", nil, bob.Token)
		check("Bob un-reposts", code == 200, fmt.Sprintf("code=%d", code))

		code, _, _ = doJSON("POST", baseURL+"/posts/"+postID+"/bookmark", nil, alice.Token)
		check("Alice bookmarks her own post", code == 200 || code == 201, fmt.Sprintf("code=%d", code))

		code, _, _ = doJSON("GET", baseURL+"/users/me/bookmarks", nil, alice.Token)
		check("GET bookmarks", code == 200, fmt.Sprintf("code=%d", code))

		code, _, _ = doJSON("DELETE", baseURL+"/posts/"+postID+"/bookmark", nil, alice.Token)
		check("Alice removes bookmark", code == 200, fmt.Sprintf("code=%d", code))
	} else {
		log.Println("  ⏭️  Skipping interaction tests (no post created)")
	}

	// ── 9. Search ──────────────────────────────────────────────────────────
	log.Println("\n📋 [7] USER SEARCH")
	code, res, _ = doJSON("GET", baseURL+"/users/search?q="+alice.Username[:5]+"&limit=5&offset=0", nil, alice.Token)
	check("Search users returns 200", code == 200, fmt.Sprintf("code=%d", code))

	// ── 10. Messages — Plain text (backward compat) ────────────────────────
	log.Println("\n📋 [8] MESSAGES — PLAIN TEXT (backward compatibility)")
	code, res, _ = doJSON("POST", baseURL+"/messages/send", map[string]string{
		"recipient_id": bob.ID,
		"content":      "Hello Bob, plain text!",
	}, alice.Token)
	check("Send plain-text message", code == 200 || code == 201, fmt.Sprintf("code=%d res=%v", code, res))
	var threadID string
	if t, ok := res["thread"].(map[string]interface{}); ok {
		threadID = t["id"].(string)
	}

	// Get threads
	code, res, _ = doJSON("GET", baseURL+"/messages/threads", nil, bob.Token)
	check("Bob GET /messages/threads", code == 200, fmt.Sprintf("code=%d", code))

	// Get messages
	if threadID != "" {
		code, res, _ = doJSON("GET", baseURL+"/messages/threads/"+threadID+"?limit=50&offset=0", nil, bob.Token)
		check("Bob GET thread messages", code == 200, fmt.Sprintf("code=%d", code))
		if msgs, ok := res["messages"].([]interface{}); ok && len(msgs) > 0 {
			lastMsg := msgs[len(msgs)-1].(map[string]interface{})
			check("Plain-text msg content preserved", lastMsg["content"] == "Hello Bob, plain text!", fmt.Sprintf("got: %v", lastMsg["content"]))
		}
	}

	// ── 11. Messages — E2EE ────────────────────────────────────────────────
	log.Println("\n📋 [9] MESSAGES — E2E ENCRYPTED")
	originalMsg := "This is a super secret message! 🔐"

	// Alice derives shared secret with Bob's public key
	bobPub, err := parseECDHPub(bob.EncPubB64)
	check("Parse Bob's encryption public key", err == nil, fmt.Sprintf("%v", err))

	aliceSecret, err := alice.EncKey.ECDH(bobPub)
	check("Alice derives shared secret", err == nil, fmt.Sprintf("%v", err))

	// Hash to 32 bytes for AES-256
	// ECDH P-256 already gives 32 bytes
	check("Shared secret is 32 bytes", len(aliceSecret) == 32, fmt.Sprintf("len=%d", len(aliceSecret)))

	// Encrypt
	ivB64, ctB64, err := aesEncrypt(originalMsg, aliceSecret)
	check("AES-GCM encrypt succeeds", err == nil, fmt.Sprintf("%v", err))

	// Build ciphertext payload
	cipherPayload, _ := json.Marshal(map[string]string{"c": ctB64, "iv": ivB64})

	// Send encrypted message
	code, res, _ = doJSON("POST", baseURL+"/messages/send", map[string]string{
		"recipient_id": bob.ID,
		"content":      "🔒 Encrypted Message",
		"ciphertext":   string(cipherPayload),
	}, alice.Token)
	check("Send encrypted message", code == 200 || code == 201, fmt.Sprintf("code=%d res=%v", code, res))

	// Bob retrieves thread
	code, res, _ = doJSON("GET", baseURL+"/messages/threads", nil, bob.Token)
	check("Bob gets threads after E2EE msg", code == 200, fmt.Sprintf("code=%d", code))

	// Find thread with Alice
	var e2eThreadID string
	if threads, ok := res["threads"].([]interface{}); ok {
		for _, t := range threads {
			tm := t.(map[string]interface{})
			pa, _ := tm["participant_a_id"].(string)
			pb, _ := tm["participant_b_id"].(string)
			if pa == alice.ID || pb == alice.ID {
				e2eThreadID = tm["id"].(string)
				break
			}
		}
	}
	check("Found thread between Alice & Bob", e2eThreadID != "", "thread not found")

	// Get messages and decrypt
	if e2eThreadID != "" {
		code, res, _ = doJSON("GET", baseURL+"/messages/threads/"+e2eThreadID+"?limit=50&offset=0", nil, bob.Token)
		check("Bob gets E2EE thread messages", code == 200, fmt.Sprintf("code=%d", code))

		if msgs, ok := res["messages"].([]interface{}); ok {
			// Find the encrypted message
			var foundEncrypted bool
			for _, m := range msgs {
				mm := m.(map[string]interface{})
				ct, hasCT := mm["ciphertext"].(string)
				if !hasCT || ct == "" {
					continue
				}
				foundEncrypted = true

				// Bob derives shared secret using Alice's public key
				alicePub, err := parseECDHPub(alice.EncPubB64)
				check("Parse Alice's encryption public key", err == nil, fmt.Sprintf("%v", err))

				bobSecret, err := bob.EncKey.ECDH(alicePub)
				check("Bob derives shared secret", err == nil, fmt.Sprintf("%v", err))
				check("Shared secrets match", string(aliceSecret) == string(bobSecret), "secrets differ!")

				// Parse ciphertext JSON
				var payload map[string]string
				// Handle potential double-encoding
				raw := ct
				for i := 0; i < 3; i++ {
					if err := json.Unmarshal([]byte(raw), &payload); err == nil {
						break
					}
					var s string
					if err := json.Unmarshal([]byte(raw), &s); err == nil {
						raw = s
					} else {
						break
					}
				}
				check("Parse ciphertext JSON", payload != nil && payload["c"] != "" && payload["iv"] != "",
					fmt.Sprintf("payload=%v raw=%s", payload, ct[:min(len(ct), 60)]))

				if payload != nil {
					decrypted, err := aesDecrypt(payload["c"], payload["iv"], bobSecret)
					check("Decrypt message", err == nil, fmt.Sprintf("%v", err))
					check("Decrypted matches original", decrypted == originalMsg,
						fmt.Sprintf("got=%q want=%q", decrypted, originalMsg))
				}
				break // Only test the first encrypted message
			}
			check("Found encrypted message in thread", foundEncrypted, "no message with ciphertext field")
		}
	}

	// ── 12. Messages — Mark as Read ────────────────────────────────────────
	log.Println("\n📋 [10] MESSAGES — MARK AS READ")
	if e2eThreadID != "" {
		code, _, _ = doJSON("POST", baseURL+"/messages/threads/"+e2eThreadID+"/read", nil, bob.Token)
		check("Mark thread as read", code == 200, fmt.Sprintf("code=%d", code))
	}

	// ── 13. Messages — Start Conversation ──────────────────────────────────
	log.Println("\n📋 [11] MESSAGES — START CONVERSATION")
	code, res, _ = doJSON("POST", baseURL+"/messages/conversation/"+bob.ID, nil, alice.Token)
	check("Start/get conversation with Bob", code == 200, fmt.Sprintf("code=%d", code))

	// ── 14. Post deletion ──────────────────────────────────────────────────
	log.Println("\n📋 [12] POST DELETION")
	if postID != "" {
		code, _, _ = doJSON("DELETE", baseURL+"/posts/"+postID, nil, alice.Token)
		check("Delete post", code == 200, fmt.Sprintf("code=%d", code))
	}

	// ── 15. Edge cases ─────────────────────────────────────────────────────
	log.Println("\n📋 [13] EDGE CASES")
	// Send to self
	code, _, _ = doJSON("POST", baseURL+"/messages/send", map[string]string{
		"recipient_id": alice.ID, "content": "self",
	}, alice.Token)
	check("Cannot send message to self", code == 400, fmt.Sprintf("code=%d", code))

	// Empty content + no ciphertext
	code, _, _ = doJSON("POST", baseURL+"/messages/send", map[string]string{
		"recipient_id": bob.ID, "content": "",
	}, alice.Token)
	check("Empty content rejected", code == 400, fmt.Sprintf("code=%d", code))

	// Unauthenticated
	code, _, _ = doJSON("GET", baseURL+"/users/me", nil, "")
	check("Unauthenticated /users/me rejected", code == 401 || code == 400, fmt.Sprintf("code=%d", code))

	code, _, _ = doJSON("GET", baseURL+"/messages/threads", nil, "")
	check("Unauthenticated /messages/threads rejected", code == 401 || code == 400, fmt.Sprintf("code=%d", code))

	// ═══════════════════════════════════════════════════════════════════════
	log.Println("\n═══════════════════════════════════════════════")
	log.Printf("  RESULTS: %d/%d passed, %d failed", passed, total, failed)
	log.Println("═══════════════════════════════════════════════")

	if failed > 0 {
		log.Printf("❌ SOME TESTS FAILED (%d failures)", failed)
		os.Exit(1)
	} else {
		log.Println("✅ ALL TESTS PASSED  🎉")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
