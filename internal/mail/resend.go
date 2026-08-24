package mail

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const defaultResendEndpoint = "https://api.resend.com/emails"
const defaultFrom = "Scaloo <noreply@chibuzo.com.ng>"

type resendPayload struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
}

// ResendMailer sends mail through Resend's HTTP API.
type ResendMailer struct {
	APIKey   string
	From     string
	Endpoint string
	Client   *http.Client
	Log      *slog.Logger
}

func (m *ResendMailer) endpoint() string {
	if strings.TrimSpace(m.Endpoint) != "" {
		return m.Endpoint
	}
	return defaultResendEndpoint
}

func (m *ResendMailer) client() *http.Client {
	if m.Client != nil {
		return m.Client
	}
	return &http.Client{Timeout: 15 * time.Second}
}

func (m *ResendMailer) Send(ctx context.Context, msg Message) error {
	if msg.To == "" {
		return fmt.Errorf("mail: empty recipient")
	}
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
	}

	from := m.From
	if from == "" {
		from = defaultFrom
	}
	body, err := json.Marshal(resendPayload{
		From:    from,
		To:      []string{msg.To},
		Subject: msg.Subject,
		HTML:    msg.HTML,
	})
	if err != nil {
		return fmt.Errorf("mail: resend encode: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.endpoint(), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("mail: resend request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+m.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client().Do(req)
	if err != nil {
		return fmt.Errorf("mail: resend: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("mail: resend %s: %s", resp.Status, strings.TrimSpace(string(respBody)))
	}
	return nil
}

func (m *ResendMailer) SendAsync(msg Message) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := m.Send(ctx, msg); err != nil {
			m.Log.Error("mail: send failed", "to", msg.To, "subject", msg.Subject, "err", err)
		}
	}()
}
