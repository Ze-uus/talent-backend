package store_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Ze-uus/talent-backend/internal/store"
)

func TestUserJSONOmitsSecrets(t *testing.T) {
	u := store.User{
		ID:            "id-1",
		Email:         "a@b.com",
		Password_hash: "$2a$12$secret-hash",
		Role:          store.Role_talent,
		Google_id:     "google-xyz",
		Full_name:     "Ada",
		Totp_secret:   "BASE32SECRET",
		Totp_enabled:  true,
		Totp_verified: true,
		Invite_token:  "invite-tok",
		Invite_expires_at: time.Now().UTC(),
		Active:        true,
	}
	b, err := json.Marshal(u)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, leak := range []string{
		"Password_hash", "password_hash", "$2a$12$",
		"Totp_secret", "BASE32SECRET",
		"Invite_token", "invite-tok",
		"Google_id", "google-xyz",
		"Invite_expires_at",
	} {
		if strings.Contains(s, leak) {
			t.Fatalf("JSON leaked %q: %s", leak, s)
		}
	}
	if !strings.Contains(s, `"Email":"a@b.com"`) {
		t.Fatalf("expected public fields present: %s", s)
	}
}

func TestSessionJSONOmitsToken(t *testing.T) {
	sess := store.Session{
		ID:      "s1",
		User_id: "u1",
		Token:   "raw-session-token",
	}
	b, err := json.Marshal(sess)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "raw-session-token") || strings.Contains(string(b), `"Token"`) {
		t.Fatalf("session token leaked: %s", b)
	}
}

func TestBrandContactJSONOmitsPasswordHash(t *testing.T) {
	c := store.BrandContact{
		ID:                   "c1",
		Viewer_token:         "viewer-tok",
		Access_password_hash: "$2a$12$contact-hash",
	}
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if strings.Contains(s, "Access_password_hash") || strings.Contains(s, "$2a$12$contact-hash") {
		t.Fatalf("contact password hash leaked: %s", s)
	}
	if !strings.Contains(s, "viewer-tok") {
		t.Fatalf("viewer_token should remain for admin share links: %s", s)
	}
}

func TestViewerPasswordJSONOmitsHash(t *testing.T) {
	p := store.ViewerPassword{
		ID:            "p1",
		Password_hash: "$2a$12$viewer-hash",
		Label:         "main",
	}
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "$2a$12$viewer-hash") || strings.Contains(string(b), "Password_hash") {
		t.Fatalf("viewer password hash leaked: %s", b)
	}
}
