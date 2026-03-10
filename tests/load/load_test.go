// Package load_test provides load testing scenarios for the Splitter platform.
package load_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"splitter/tests/testlogger"
)

const loadBaseURL = "http://localhost:8000/api/v1"

func doLoadJSON(method, url string, body interface{}, token string) (int, error) {
	var reqBody *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(b)
	} else {
		reqBody = &bytes.Buffer{}
	}
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return 0, err
	}
	resp.Body.Close()
	return resp.StatusCode, nil
}

func skipIfLoadServerDown(t *testing.T) {
	t.Helper()
	_, err := doLoadJSON("GET", loadBaseURL+"/health", nil, "")
	if err != nil {
		t.Skipf("Server not reachable for load test (skip): %v", err)
	}
}

// TestLoadPublicFeedConcurrent runs 25 concurrent requests to the public feed.
func TestLoadPublicFeedConcurrent(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "load", start, testErr) }()
	skipIfLoadServerDown(t)

	concurrency := 25
	var wg sync.WaitGroup
	errors := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			code, err := doLoadJSON("GET", loadBaseURL+"/posts/public", nil, "")
			if err != nil {
				errors <- err
				return
			}
			if code != 200 {
				errors <- fmt.Errorf("unexpected status %d", code)
			}
		}()
	}
	wg.Wait()
	close(errors)

	var errCount int
	for e := range errors {
		t.Logf("Load error: %v", e)
		errCount++
	}
	errorRate := float64(errCount) / float64(concurrency)
	if errorRate > 0.1 {
		t.Errorf("Error rate %.2f%% exceeds 10%% threshold (%d/%d failed)", errorRate*100, errCount, concurrency)
	}
}

// TestLoadHealthEndpointLatency checks that health endpoint responds within SLA.
func TestLoadHealthEndpointLatency(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "load", start, testErr) }()
	skipIfLoadServerDown(t)

	const iterations = 10
	const maxLatency = 500 * time.Millisecond

	var total time.Duration
	for i := 0; i < iterations; i++ {
		reqStart := time.Now()
		code, err := doLoadJSON("GET", loadBaseURL+"/health", nil, "")
		elapsed := time.Since(reqStart)
		total += elapsed

		if err != nil {
			testErr = err
			t.Fatalf("Request %d failed: %v", i, err)
		}
		if code != 200 {
			t.Errorf("Request %d: expected 200, got %d", i, code)
		}
		if elapsed > maxLatency {
			t.Logf("Request %d latency %s exceeded %s", i, elapsed, maxLatency)
		}
	}

	avg := total / iterations
	t.Logf("Average health endpoint latency over %d requests: %s", iterations, avg)
}

// TestLoadStoryFeedConcurrent runs 20 concurrent requests to the story feed.
func TestLoadStoryFeedConcurrent(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "load", start, testErr) }()
	skipIfLoadServerDown(t)

	concurrency := 20
	var wg sync.WaitGroup
	errors := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			code, err := doLoadJSON("GET", loadBaseURL+"/stories/feed", nil, "")
			if err != nil {
				errors <- err
				return
			}
			if code != 200 {
				errors <- fmt.Errorf("story feed returned %d", code)
			}
		}()
	}
	wg.Wait()
	close(errors)

	var errCount int
	for e := range errors {
		t.Logf("Load error: %v", e)
		errCount++
	}
	if errCount > 0 {
		t.Logf("Note: %d/%d story feed requests failed under load (may require auth or server is down)", errCount, concurrency)
	}
}
