package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"splitter/internal/db"
	"testing"
)

/*
INTEGRATION TEST SUMMARY:
- What was tested: Post creation and retrieval flow.
- What passed:
    - Creating a post (authenticated) works.
    - Retrieving the created post via ID works.
    - Public feed includes the new post.
    - Post has correct author and initial zero counters.
- Any limitations: File upload not tested here (requires multipart setup).
- Why this test matters: Verifies core social functionality.
*/

func TestPostFlow(t *testing.T) {
	// 1. Setup
	cleanup := SetupTestEnv(t)
	defer cleanup()

	// 2. Create User & Get Token
	username := "post_author"
	token := registerUser(t, username, "author@example.com", "password123")

	var postID string

	// Test 1: Create Post
	t.Run("Create Post", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		_ = writer.WriteField("content", "Hello, Integration World!")
		_ = writer.WriteField("visibility", "public")
		_ = writer.Close()

		req, err := http.NewRequest(http.MethodPost, TestServer.URL+"/api/v1/posts", body)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", resp.StatusCode)
		}

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		id, ok := result["id"].(string)
		if !ok {
			t.Fatal("Response missing 'id'")
		}
		postID = id

		// Verify DB
		var content string
		var directCount, totalCount int
		err = db.DB.QueryRow(context.Background(),
			"SELECT content, direct_reply_count, total_reply_count FROM posts WHERE id=$1", postID).
			Scan(&content, &directCount, &totalCount)

		if err != nil {
			t.Fatalf("Failed to query post from DB: %v", err)
		}
		if content != "Hello, Integration World!" {
			t.Errorf("Content mismatch: %s", content)
		}
		if directCount != 0 || totalCount != 0 {
			t.Error("New post counters should be zero")
		}
	})

	// Test 2: Get Post by ID
	t.Run("Get Post", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/v1/posts/%s", TestServer.URL, postID), nil)
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var post map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&post); err != nil {
			t.Fatal(err)
		}

		if post["content"] != "Hello, Integration World!" {
			t.Error("Content mismatch in GET")
		}
		// Verify author fields (flattened in API)
		if post["username"] != username {
			t.Errorf("Expected username %s, got %v", username, post["username"])
		}
		if post["author_did"] == "" {
			t.Error("Author DID missing")
		}
	})

	// Test 3: Public Feed
	t.Run("Public Feed", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, TestServer.URL+"/api/v1/posts/public", nil)
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var feed []interface{}
		if err := json.NewDecoder(resp.Body).Decode(&feed); err != nil {
			t.Fatal(err)
		}

		found := false
		for _, item := range feed {
			p := item.(map[string]interface{})
			if p["id"] == postID {
				found = true
				break
			}
		}
		if !found {
			t.Error("Created post not found in public feed")
		}
	})
}

// Helper to register user and get token
func registerUser(t *testing.T, username, email, password string) string {
	reqBody, _ := json.Marshal(map[string]string{
		"username": username,
		"email":    email,
		"password": password,
	})

	resp, err := http.Post(TestServer.URL+"/api/v1/auth/register", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Register returned status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	return result["token"].(string)
}

func TestEphemeralPostExpiryEnforcement(t *testing.T) {
	cleanup := SetupTestEnv(t)
	defer cleanup()

	token := registerUser(t, "ephemeral_author", "ephemeral@example.com", "password123")

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("content", "This post should expire")
	_ = writer.WriteField("visibility", "public")
	_ = writer.WriteField("expires_in_minutes", "60")
	_ = writer.Close()

	createReq, err := http.NewRequest(http.MethodPost, TestServer.URL+"/api/v1/posts", body)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	createReq.Header.Set("Authorization", "Bearer "+token)
	createReq.Header.Set("Content-Type", writer.FormDataContentType())

	createResp, err := (&http.Client{}).Do(createReq)
	if err != nil {
		t.Fatalf("Failed to create post: %v", err)
	}
	defer createResp.Body.Close()

	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d", createResp.StatusCode)
	}

	var created map[string]interface{}
	if err := json.NewDecoder(createResp.Body).Decode(&created); err != nil {
		t.Fatalf("Failed to decode create response: %v", err)
	}

	postID, _ := created["id"].(string)
	if postID == "" {
		t.Fatal("create response missing post id")
	}
	if _, ok := created["expires_at"].(string); !ok {
		t.Fatal("create response missing expires_at")
	}

	_, err = db.DB.Exec(context.Background(), `UPDATE posts SET expires_at = NOW() - interval '1 minute' WHERE id = $1`, postID)
	if err != nil {
		t.Fatalf("Failed to force post expiration: %v", err)
	}

	getReq, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/v1/posts/%s", TestServer.URL, postID), nil)
	getResp, err := (&http.Client{}).Do(getReq)
	if err != nil {
		t.Fatalf("Failed to get post: %v", err)
	}
	defer getResp.Body.Close()

	if getResp.StatusCode != http.StatusNotFound {
		t.Fatalf("Expected status 404 for expired post, got %d", getResp.StatusCode)
	}

	feedReq, _ := http.NewRequest(http.MethodGet, TestServer.URL+"/api/v1/posts/public", nil)
	feedResp, err := (&http.Client{}).Do(feedReq)
	if err != nil {
		t.Fatalf("Failed to get public feed: %v", err)
	}
	defer feedResp.Body.Close()

	if feedResp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200 for feed, got %d", feedResp.StatusCode)
	}

	var feed []map[string]interface{}
	if err := json.NewDecoder(feedResp.Body).Decode(&feed); err != nil {
		t.Fatalf("Failed to decode feed response: %v", err)
	}

	for _, item := range feed {
		if item["id"] == postID {
			t.Fatalf("expired post %s should not appear in feed", postID)
		}
	}
}
