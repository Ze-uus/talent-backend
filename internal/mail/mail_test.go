package mail_test

import (
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
