package handlers

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"splitter/internal/auth"
	"splitter/internal/config"
	"splitter/internal/models"
)

type proofUsers struct {
	authUserRepository
	users map[string]*models.User
}

func (r *proofUsers) GetByDID(_ context.Context, did string) (*models.User, error) {
	if u := r.users[did]; u != nil {
		return u, nil
	}
	return nil, errors.New("missing")
}
func proofFixture(t *testing.T) (*AuthHandler, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	u := &models.User{ID: "alice", DID: "did:alice", Username: "alice", Role: "user", PublicKey: base64.StdEncoding.EncodeToString(pub)}
	return &AuthHandler{userRepo: &proofUsers{users: map[string]*models.User{u.DID: u}}, cfg: &config.Config{JWT: config.JWTConfig{Expiration: 2}}, jwtSecret: "test-secret-only", challenges: map[string]*models.AuthChallenge{u.DID: {DID: u.DID, Challenge: "nonce", ExpiresAt: time.Now().Add(time.Minute)}}}, priv
}
func invokeJSON(handler echo.HandlerFunc, body any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(string(b)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e := echo.New()
	c := e.NewContext(req, rec)
	if err := handler(c); err != nil {
		e.HTTPErrorHandler(err, c)
	}
	return rec
}
func signProof(priv ed25519.PrivateKey, challenge string) string {
	return base64.StdEncoding.EncodeToString(ed25519.Sign(priv, []byte(challenge)))
}
func TestDIDProofRejectsInvalidSignatures(t *testing.T) {
	for _, kind := range []string{"missing", "malformed", "wrong-key", "wrong-message", "missing-public-key", "suspended"} {
		t.Run(kind, func(t *testing.T) {
			h, priv := proofFixture(t)
			sig := signProof(priv, "nonce")
			switch kind {
			case "missing":
				sig = ""
			case "malformed":
				sig = "not-base64"
			case "wrong-key":
				_, wrong, _ := ed25519.GenerateKey(rand.Reader)
				sig = signProof(wrong, "nonce")
			case "wrong-message":
				sig = signProof(priv, "other nonce")
			case "missing-public-key":
				h.userRepo.(*proofUsers).users["did:alice"].PublicKey = ""
			case "suspended":
				h.userRepo.(*proofUsers).users["did:alice"].IsSuspended = true
			}
			rec := invokeJSON(h.VerifyChallenge, models.VerifyChallengeRequest{DID: "did:alice", Challenge: "nonce", Signature: sig})
			if rec.Code < 400 {
				t.Fatalf("invalid proof issued credentials: %d %s", rec.Code, rec.Body.String())
			}
			if strings.Contains(rec.Body.String(), `"token"`) {
				t.Fatal("rejected response contains token")
			}
		})
	}
}
func TestDIDValidProofSingleUse(t *testing.T) {
	h, priv := proofFixture(t)
	req := models.VerifyChallengeRequest{DID: "did:alice", Challenge: "nonce", Signature: signProof(priv, "nonce")}
	if rec := invokeJSON(h.VerifyChallenge, req); rec.Code != 200 {
		t.Fatalf("valid proof: %d %s", rec.Code, rec.Body.String())
	}
	if rec := invokeJSON(h.VerifyChallenge, req); rec.Code != 401 {
		t.Fatalf("replayed proof: %d", rec.Code)
	}
}
func TestDIDProofExpiryWrongUserAndChallenge(t *testing.T) {
	for _, kind := range []string{"expired", "wrong-user", "wrong-challenge"} {
		t.Run(kind, func(t *testing.T) {
			h, priv := proofFixture(t)
			req := models.VerifyChallengeRequest{DID: "did:alice", Challenge: "nonce", Signature: signProof(priv, "nonce")}
			switch kind {
			case "expired":
				h.challenges["did:alice"].ExpiresAt = time.Now().Add(-time.Second)
			case "wrong-user":
				req.DID = "did:bob"
			case "wrong-challenge":
				req.Challenge = "other"
			}
			if rec := invokeJSON(h.VerifyChallenge, req); rec.Code != 401 {
				t.Fatalf("invalid proof: %d", rec.Code)
			}
		})
	}
}
func TestDIDConcurrentProofOnlyOneToken(t *testing.T) {
	h, priv := proofFixture(t)
	req := models.VerifyChallengeRequest{DID: "did:alice", Challenge: "nonce", Signature: signProof(priv, "nonce")}
	var wg sync.WaitGroup
	codes := make(chan int, 24)
	start := make(chan struct{})
	for i := 0; i < 24; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); <-start; codes <- invokeJSON(h.VerifyChallenge, req).Code }()
	}
	close(start)
	wg.Wait()
	close(codes)
	success := 0
	for code := range codes {
		if code == 200 {
			success++
		} else if code != 401 {
			t.Errorf("unexpected status %d", code)
		}
	}
	if success != 1 {
		t.Fatalf("same proof issued %d tokens", success)
	}
}
func TestInvalidDIDProofDoesNotConsumeValidChallenge(t *testing.T) {
	h, priv := proofFixture(t)
	invokeJSON(h.VerifyChallenge, models.VerifyChallengeRequest{DID: "did:alice", Challenge: "nonce", Signature: "invalid"})
	if rec := invokeJSON(h.VerifyChallenge, models.VerifyChallengeRequest{DID: "did:alice", Challenge: "nonce", Signature: signProof(priv, "nonce")}); rec.Code != 200 {
		t.Fatalf("invalid signature consumed challenge: %d", rec.Code)
	}
}
func TestAuthHandlerUsesConfiguredTokenLifetime(t *testing.T) {
	h, priv := proofFixture(t)
	rec := invokeJSON(h.VerifyChallenge, models.VerifyChallengeRequest{DID: "did:alice", Challenge: "nonce", Signature: signProof(priv, "nonce")})
	var response struct {
		Token string `json:"token"`
	}
	json.Unmarshal(rec.Body.Bytes(), &response)
	claims, err := authClaims(response.Token, h.jwtSecret)
	if err != nil {
		t.Fatal(err)
	}
	lifetime := claims["exp"].(float64) - claims["iat"].(float64)
	if lifetime != 7200 {
		t.Fatalf("configured 2h token has lifetime %.0fs", lifetime)
	}
}

func authClaims(raw, secret string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(raw, func(*jwt.Token) (interface{}, error) { return []byte(secret), nil })
	if err != nil {
		return nil, err
	}
	return token.Claims.(jwt.MapClaims), nil
}

type avatarSignupUsers struct {
	authUserRepository
	saved     []byte
	avatarURL string
}

func (*avatarSignupUsers) UsernameExists(context.Context, string) (bool, error) { return false, nil }
func (*avatarSignupUsers) EmailExists(context.Context, string) (bool, error)    { return false, nil }
func (*avatarSignupUsers) Create(context.Context, *models.UserCreate, string) (*models.User, error) {
	return &models.User{ID: "new-user", DID: "did:new", Username: "new-user", Role: "user"}, nil
}
func (r *avatarSignupUsers) UpdateAvatar(_ context.Context, _ string, b []byte, _ string, url string) (*models.User, error) {
	r.saved = append([]byte(nil), b...)
	r.avatarURL = url
	return &models.User{ID: "new-user", DID: "did:new", Username: "new-user", AvatarURL: url}, nil
}
func TestSignupAvatarIsStoredDurably(t *testing.T) {
	t.Chdir(t.TempDir())
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for k, v := range map[string]string{"username": "new-user", "email": "new@example.com", "password": "a-test-password"} {
		if err := writer.WriteField(k, v); err != nil {
			t.Fatal(err)
		}
	}
	part, err := writer.CreateFormFile("avatar", "avatar.png")
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(part, image.NewRGBA(image.Rect(0, 0, 8, 8))); err != nil {
		t.Fatal(err)
	}
	writer.Close()
	repo := &avatarSignupUsers{}
	h := &AuthHandler{userRepo: repo, jwtSecret: "test-secret"}
	e := echo.New()
	req := httptest.NewRequest("POST", "/", &body)
	req.Header.Set(echo.HeaderContentType, writer.FormDataContentType())
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	if err := h.Register(c); err != nil {
		e.HTTPErrorHandler(err, c)
	}
	if rec.Code != 201 {
		t.Fatalf("signup failed: %d %s", rec.Code, rec.Body.String())
	}
	if len(repo.saved) == 0 || repo.avatarURL != "/api/v1/users/new-user/avatar" {
		t.Fatalf("avatar is not durable: bytes=%d url=%q", len(repo.saved), repo.avatarURL)
	}
}

type deviceApprovalUsers struct {
	authUserRepository
	user   *models.User
	writes int
}

func (r *deviceApprovalUsers) GetByID(context.Context, string) (*models.User, error) {
	return r.user, nil
}
func (r *deviceApprovalUsers) ListDeviceKeys(context.Context, string) ([]*models.DeviceKey, error) {
	return []*models.DeviceKey{{DeviceID: "primary", EncryptionPublicKey: "old-key", Status: "approved"}, {DeviceID: "phone", EncryptionPublicKey: "new-key", Status: "pending"}}, nil
}
func (r *deviceApprovalUsers) ApproveDeviceKey(context.Context, string, string, string) (*models.DeviceKey, error) {
	r.writes++
	return &models.DeviceKey{DeviceID: "phone", Status: "approved"}, nil
}
func TestDeviceApprovalRequiresAccountProof(t *testing.T) {
	for _, kind := range []string{"missing", "wrong-target", "valid"} {
		t.Run(kind, func(t *testing.T) {
			pub, priv, _ := ed25519.GenerateKey(rand.Reader)
			repo := &deviceApprovalUsers{user: &models.User{ID: "alice", Username: "alice", PublicKey: base64.StdEncoding.EncodeToString(pub)}}
			h := &AuthHandler{userRepo: repo}
			timestamp := time.Now().Unix()
			message := fmt.Sprintf("approve-device|alice|phone|primary|new-key|%d", timestamp)
			if kind == "wrong-target" {
				message = fmt.Sprintf("approve-device|alice|other|primary|new-key|%d", timestamp)
			}
			body := map[string]any{"approver_device_id": "primary", "timestamp": timestamp}
			if kind != "missing" {
				body["signature"] = signProof(priv, message)
			}
			rec := invokeJSON(func(c echo.Context) error {
				c.Set("user_id", "alice")
				c.SetParamNames("deviceId")
				c.SetParamValues("phone")
				return h.ApproveDeviceKey(c)
			}, body)
			if kind == "valid" {
				if rec.Code != 200 || repo.writes != 1 {
					t.Fatalf("valid approval: %d writes %d", rec.Code, repo.writes)
				}
			} else if rec.Code < 400 || repo.writes != 0 {
				t.Fatalf("unproven device approved: %d writes %d", rec.Code, repo.writes)
			}
		})
	}
}

type passwordKeyRecoveryUsers struct {
	authUserRepository
	user      *models.User
	hash      string
	rotations int
}

func (r *passwordKeyRecoveryUsers) GetByID(context.Context, string) (*models.User, error) {
	return r.user, nil
}
func (r *passwordKeyRecoveryUsers) GetByUsername(context.Context, string) (*models.User, string, error) {
	return r.user, r.hash, nil
}
func (r *passwordKeyRecoveryUsers) RotateKey(_ context.Context, id, old, newKey, nonce, reason string) error {
	r.rotations++
	return nil
}
func TestSigningKeyRecoveryRequiresCurrentPassword(t *testing.T) {
	hash, err := auth.HashPassword("correct-test-password")
	if err != nil {
		t.Fatal(err)
	}
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	for _, password := range []string{"", "wrong-test-password", "correct-test-password"} {
		t.Run(password, func(t *testing.T) {
			repo := &passwordKeyRecoveryUsers{user: &models.User{ID: "alice", Username: "alice", DID: "did:alice", PublicKey: "legacy-invalid-key"}, hash: hash}
			h := &AuthHandler{userRepo: repo}
			body := map[string]any{"new_public_key": base64.StdEncoding.EncodeToString(pub), "signature": "legacy-invalid-signature", "nonce": "recovery-nonce", "timestamp": time.Now().Unix(), "current_password": password}
			rec := invokeJSON(func(c echo.Context) error { c.Set("user_id", "alice"); return h.RotateKey(c) }, body)
			if password == "correct-test-password" {
				if rec.Code != 200 || repo.rotations != 1 {
					t.Fatalf("explicit password recovery rejected: %d writes%d %s", rec.Code, repo.rotations, rec.Body.String())
				}
			} else if rec.Code < 400 || repo.rotations != 0 {
				t.Fatalf("invalid recovery changed key: %d writes%d", rec.Code, repo.rotations)
			}
		})
	}
}
