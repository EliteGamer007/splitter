package handlers

import (
	"context"
	"encoding/json"
	"github.com/labstack/echo/v4"
	"net/http/httptest"
	"splitter/internal/models"
	"strings"
	"testing"
)

type profileFixture struct {
	userProfileRepository
	user   *models.User
	writes *int
}

func (f profileFixture) GetByID(context.Context, string) (*models.User, error)  { return f.user, nil }
func (f profileFixture) GetByDID(context.Context, string) (*models.User, error) { return f.user, nil }
func TestPublicProfilesOmitPrivateAttributes(t *testing.T) {
	user := &models.User{ID: "account", DID: "did:key:account", Username: "alice", Email: "secret@example.com", ModerationRequested: true}
	h := &UserHandler{userRepo: profileFixture{user: user}}
	for _, name := range []string{"id", "did", "me"} {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			c := echo.New().NewContext(httptest.NewRequest("GET", "/?did=did:key:account", nil), rec)
			c.SetParamNames("id")
			c.SetParamValues("account")
			c.Set("did", user.DID)
			var err error
			if name == "me" {
				err = h.GetCurrentUser(c)
			} else if name == "id" {
				err = h.GetProfile(c)
			} else {
				err = h.GetProfileByDID(c)
			}
			if err != nil {
				t.Fatal(err)
			}
			var body map[string]any
			if err = json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if name == "me" {
				if body["email"] != user.Email {
					t.Fatal("own account lost private details")
				}
			} else {
				for _, key := range []string{"email", "moderation_requested", "moderation_requested_at"} {
					if _, ok := body[key]; ok {
						t.Errorf("public response includes %s", key)
					}
				}
			}
			if user.Email != "secret@example.com" {
				t.Fatal("public conversion mutated account")
			}
		})
	}
}

type updateProfileFixture struct {
	profileFixture
	writes int
}

func (f *updateProfileFixture) Update(context.Context, string, *models.UserUpdate) (*models.User, error) {
	f.writes++
	return f.user, nil
}
func TestInvalidPrivacyUpdateHasNoMutation(t *testing.T) {
	fixture := &updateProfileFixture{profileFixture: profileFixture{user: &models.User{ID: "account", DID: "did:key:account"}}}
	h := &UserHandler{userRepo: fixture}
	req := httptest.NewRequest("PATCH", "/", strings.NewReader(`{"display_name":"changed","message_privacy":"invented"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(req, rec)
	c.Set("did", "did:key:account")
	if err := h.UpdateProfile(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != 400 {
		t.Fatalf("status=%d", rec.Code)
	}
	if fixture.writes != 0 {
		t.Errorf("rejected privacy request mutated profile %d times", fixture.writes)
	}
}
