package mail_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Ze-uus/talent-backend/internal/mail"
)

func TestRenderTalentApproved(t *testing.T) {
	html, err := mail.Render("talent_approved.html", mail.TalentStatusData{
		BaseData: mail.BaseData{
			AppURL:    "http://localhost:3000",
			Name:      "Ada",
			Subject:   "Your Scaloo account is approved",
			Preheader: "Approved",
		},
		LoginURL: "http://localhost:3000/login",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(html, "Ada") {
		t.Fatalf("expected name in html")
	}
	if !strings.Contains(html, "approved") {
		t.Fatalf("expected approved copy")
	}
	if !strings.Contains(html, "http://localhost:3000/login") {
		t.Fatalf("expected login url")
	}
}

func TestNotifyTalentApprovedSends(t *testing.T) {
	rec := &mail.RecordingMailer{}
	svc := mail.NewService(rec, "http://localhost:3000", nil)
	svc.NotifyTalentApproved("ada@example.com", "Ada")
	if len(rec.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(rec.Messages))
	}
	msg := rec.Messages[0]
	if msg.To != "ada@example.com" {
		t.Fatalf("to=%q", msg.To)
	}
	if !strings.Contains(msg.Subject, "approved") {
		t.Fatalf("subject=%q", msg.Subject)
	}
	if !strings.Contains(msg.HTML, "Ada") {
		t.Fatalf("html missing name")
	}
}

func TestNewNopWhenHostEmpty(t *testing.T) {
	m := mail.New(mail.Config{Host: ""})
	if _, ok := m.(*mail.NopMailer); !ok {
		t.Fatalf("expected NopMailer, got %T", m)
	}
}

func TestNewResendWhenAPIKeySet(t *testing.T) {
	m := mail.New(mail.Config{APIKey: "re_test", Host: "localhost"})
	if _, ok := m.(*mail.ResendMailer); !ok {
		t.Fatalf("expected ResendMailer, got %T", m)
	}
}

func TestNewSMTPWhenOnlyHostSet(t *testing.T) {
	m := mail.New(mail.Config{Host: "localhost"})
	if _, ok := m.(*mail.SMTPMailer); !ok {
		t.Fatalf("expected SMTPMailer, got %T", m)
	}
}

func TestResendSendPostsExpectedJSON(t *testing.T) {
	var gotAuth, gotCT, gotMethod string
	var payload map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotAuth = r.Header.Get("Authorization")
		gotCT = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &payload)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"email_123"}`))
	}))
	t.Cleanup(srv.Close)

	m := mail.New(mail.Config{
		APIKey:   "re_test",
		From:     "Scaloo <reward@chibuzo.com.ng>",
		Endpoint: srv.URL,
	})
	err := m.Send(context.Background(), mail.Message{
		To: "ada@example.com", Subject: "Hello", HTML: "<p>Hi</p>",
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("method=%s", gotMethod)
	}
	if gotAuth != "Bearer re_test" {
		t.Fatalf("auth=%q", gotAuth)
	}
	if gotCT != "application/json" {
		t.Fatalf("content-type=%q", gotCT)
	}
	if payload["from"] != "Scaloo <reward@chibuzo.com.ng>" {
		t.Fatalf("from=%v", payload["from"])
	}
	if payload["subject"] != "Hello" {
		t.Fatalf("subject=%v", payload["subject"])
	}
	if payload["html"] != "<p>Hi</p>" {
		t.Fatalf("html=%v", payload["html"])
	}
	to, _ := payload["to"].([]any)
	if len(to) != 1 || to[0] != "ada@example.com" {
		t.Fatalf("to=%v", payload["to"])
	}
}

func TestResendSendSurfacesError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"message":"Invalid from address"}`))
	}))
	t.Cleanup(srv.Close)

	m := mail.New(mail.Config{APIKey: "re_test", Endpoint: srv.URL})
	err := m.Send(context.Background(), mail.Message{
		To: "ada@example.com", Subject: "Hello", HTML: "<p>Hi</p>",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "422") && !strings.Contains(err.Error(), "Unprocessable") {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(err.Error(), "Invalid from address") {
		t.Fatalf("err=%v", err)
	}
}
