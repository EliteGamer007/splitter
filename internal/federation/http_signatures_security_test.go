package federation

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestSignedRequestRejectsTamperingAndStaleness(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	public := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
	cases := []struct {
		name   string
		before func(*http.Request)
		after  func(*http.Request)
	}{
		{name: "body replacement", after: func(r *http.Request) { r.Body = io.NopCloser(strings.NewReader(`{"type":"Delete"}`)) }},
		{name: "stale date", before: func(r *http.Request) {
			r.Header.Set("Date", time.Now().Add(-10*time.Minute).UTC().Format(http.TimeFormat))
		}},
		{name: "future date", before: func(r *http.Request) {
			r.Header.Set("Date", time.Now().Add(10*time.Minute).UTC().Format(http.TimeFormat))
		}},
		{name: "query replacement", after: func(r *http.Request) { r.URL.RawQuery = "recipient=someoneelse" }},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodPost, "https://peer.example/inbox?recipient=alice", strings.NewReader(`{"type":"Create"}`))
			if tt.before != nil {
				tt.before(req)
			}
			if err := SignRequest(req, key, "https://peer.example/actor#main-key"); err != nil {
				t.Fatal(err)
			}
			if tt.after != nil {
				tt.after(req)
			}
			if err := VerifyRequest(req, public); err == nil {
				t.Fatal("forged/stale activity accepted")
			}
		})
	}
	req, _ := http.NewRequest(http.MethodPost, "https://peer.example/inbox?recipient=alice", strings.NewReader(`{"type":"Create"}`))
	if err := SignRequest(req, key, "https://peer.example/actor#main-key"); err != nil {
		t.Fatal(err)
	}
	if err := VerifyRequest(req, public); err != nil {
		t.Fatalf("valid signed activity: %v", err)
	}
	body, err := io.ReadAll(req.Body)
	if err != nil || string(body) != `{"type":"Create"}` {
		t.Fatal("verification must preserve body")
	}
}
