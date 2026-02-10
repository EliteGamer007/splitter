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
