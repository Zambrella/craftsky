// Package safetyalerts delivers generic operator alerts without exporting reports.
package safetyalerts

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

const mailbox = "moderation@craftsky.social"

// Brevo holds server-only configuration. It must never be serialized or logged.
type Brevo struct {
	key        string
	consoleURL string
	client     *http.Client
	endpoint   string
}

func NewBrevo(key, origin string) (*Brevo, error) {
	u, err := url.Parse(origin)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") || strings.TrimSpace(key) == "" {
		return nil, errors.New("invalid safety email configuration")
	}
	return &Brevo{key: key, consoleURL: strings.TrimRight(origin, "/") + "/admin/", endpoint: "https://api.brevo.com/v3/smtp/email", client: &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}

// Send exports only static text, console origin and an opaque delivery UUID.
// The same persisted UUID is reused on retries for Brevo's idempotency window.
func (b *Brevo) Send(ctx context.Context, deliveryID uuid.UUID) error {
	payload := struct {
		Sender  map[string]string   `json:"sender"`
		To      []map[string]string `json:"to"`
		Subject string              `json:"subject"`
		Text    string              `json:"textContent"`
		Headers map[string]string   `json:"headers"`
	}{
		Sender:  map[string]string{"email": mailbox, "name": "CraftSky Moderation"},
		To:      []map[string]string{{"email": mailbox}},
		Subject: "CraftSky: urgent safety review needed",
		Text:    "New urgent safety work needs your review. Sign in to the moderation console: " + b.consoleURL + "\n\nAn alert is not a finding of wrongdoing. Review the case or safety queue in the console.",
		Headers: map[string]string{"idempotencyKey": deliveryID.String()},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("api-key", b.key)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := b.client.Do(req)
	if err != nil {
		return fmt.Errorf("send safety email: %w", err)
	}
	defer resp.Body.Close()
	// Never retain or log provider prose, headers or recipient routing.
	// Only the closed duplicate code is selected; arbitrary provider prose is discarded.
	if resp.StatusCode == http.StatusBadRequest {
		var result struct {
			Code string `json:"code"`
		}
		if json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&result) == nil && result.Code == "duplicate_parameter" {
			return nil
		}
	} else {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	}
	if resp.StatusCode != http.StatusCreated {
		return &DeliveryError{Status: resp.StatusCode}
	}
	return nil
}

type DeliveryError struct{ Status int }

func (e *DeliveryError) Error() string {
	return fmt.Sprintf("safety email provider returned status %d", e.Status)
}

func (*Brevo) String() string   { return "safetyalerts.Brevo([REDACTED])" }
func (*Brevo) GoString() string { return "safetyalerts.Brevo([REDACTED])" }
