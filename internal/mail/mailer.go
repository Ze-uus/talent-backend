package mail

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/smtp"
	"strings"
	"time"
)

// Message is a ready-to-send email.
type Message struct {
	To      string
	Subject string
	HTML    string
}

// Mailer sends email messages.
type Mailer interface {
	Send(ctx context.Context, msg Message) error
	SendAsync(msg Message)
}

// Config holds SMTP connection settings.
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
	TLS      bool
	AppURL   string
	Log      *slog.Logger
}

// New returns an SMTPMailer when Host is set, otherwise a NopMailer.
func New(cfg Config) Mailer {
	log := cfg.Log
	if log == nil {
		log = slog.Default()
	}
	if strings.TrimSpace(cfg.Host) == "" {
		log.Info("mail: SMTP_HOST empty — using NopMailer")
		return &NopMailer{Log: log}
	}
	return &SMTPMailer{
		Host:     cfg.Host,
		Port:     cfg.Port,
		User:     cfg.User,
		Password: cfg.Password,
		From:     cfg.From,
		TLS:      cfg.TLS,
		Log:      log,
	}
}

// SMTPMailer sends mail over SMTP.
type SMTPMailer struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
	TLS      bool
	Log      *slog.Logger
}

func (m *SMTPMailer) Send(ctx context.Context, msg Message) error {
	if msg.To == "" {
		return fmt.Errorf("mail: empty recipient")
	}
	addr := fmt.Sprintf("%s:%d", m.Host, m.Port)
	from := m.From
	if from == "" {
		from = "reward@usecaloo.com"
	}

	headers := []string{
		fmt.Sprintf("From: %s", from),
		fmt.Sprintf("To: %s", msg.To),
		fmt.Sprintf("Subject: %s", msg.Subject),
		"MIME-Version: 1.0",
		"Content-Type: text/html; charset=UTF-8",
	}
	body := strings.Join(headers, "\r\n") + "\r\n\r\n" + msg.HTML

	var auth smtp.Auth
	if m.User != "" {
		auth = smtp.PlainAuth("", m.User, m.Password, m.Host)
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		deadline, _ = ctx.Deadline()
	}

	d := net.Dialer{}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("mail: dial: %w", err)
	}
	defer func() {_ = conn.Close() } ()
	_ = conn.SetDeadline(deadline)

	var client *smtp.Client
	client, err = smtp.NewClient(conn, m.Host)
	if err != nil {
		return fmt.Errorf("mail: client: %w", err)
	}
	defer func() { _ = client.Close() }()

	if m.TLS {
		if err := client.StartTLS(&tls.Config{ServerName: m.Host, MinVersion: tls.VersionTLS12}); err != nil {
			return fmt.Errorf("mail: STARTTLS: %w", err)
		}
	}

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("mail: auth: %w", err)
		}
	}

	fromAddr := extractEmail(from)
	if err := client.Mail(fromAddr); err != nil {
		return fmt.Errorf("mail: MAIL FROM: %w", err)
	}
	if err := client.Rcpt(msg.To); err != nil {
		return fmt.Errorf("mail: RCPT TO: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("mail: DATA: %w", err)
	}
	if _, err := w.Write([]byte(body)); err != nil {
		_ = w.Close()
		return fmt.Errorf("mail: write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("mail: close data: %w", err)
	}
	return client.Quit()
}

func (m *SMTPMailer) SendAsync(msg Message) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := m.Send(ctx, msg); err != nil {
			m.Log.Error("mail: send failed", "to", msg.To, "subject", msg.Subject, "err", err)
		}
	}()
}

// NopMailer logs messages instead of sending them.
type NopMailer struct {
	Log *slog.Logger
}

func (m *NopMailer) Send(_ context.Context, msg Message) error {
	m.Log.Info("mail: nop send", "to", msg.To, "subject", msg.Subject)
	return nil
}

func (m *NopMailer) SendAsync(msg Message) {
	_ = m.Send(context.Background(), msg)
}

// RecordingMailer stores sent messages for tests.
type RecordingMailer struct {
	Messages []Message
}

func (m *RecordingMailer) Send(_ context.Context, msg Message) error {
	m.Messages = append(m.Messages, msg)
	return nil
}

func (m *RecordingMailer) SendAsync(msg Message) {
	_ = m.Send(context.Background(), msg)
}

func extractEmail(from string) string {
	if i := strings.LastIndex(from, "<"); i >= 0 {
		if j := strings.LastIndex(from, ">"); j > i {
			return strings.TrimSpace(from[i+1 : j])
		}
	}
	return strings.TrimSpace(from)
}
