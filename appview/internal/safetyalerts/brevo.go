// Package safetyalerts delivers generic operator alerts without exporting reports.
package safetyalerts

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
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

// ConsoleOrigin validates a console link independently of OAuth configuration.
// Plain HTTP is restricted to loopback hosts and explicitly enabled development.
func ConsoleOrigin(origin string, allowLoopbackHTTP bool) (string, error) {
	u, err := url.Parse(origin)
	if err != nil || origin != strings.TrimSpace(origin) || strings.ContainsAny(origin, "\\\\#\r\n\t") ||
		u.Opaque != "" || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.ForceQuery ||
		u.Fragment != "" || u.RawPath != "" || (u.Path != "" && u.Path != "/") {
		return "", errors.New("invalid safety email console origin")
	}
	if port := u.Port(); port != "" {
		n, err := strconv.Atoi(port)
		if err != nil || n < 1 || n > 65535 {
			return "", errors.New("invalid safety email console origin")
		}
	}
	ip := net.ParseIP(u.Hostname())
	loopback := u.Hostname() == "localhost" || (ip != nil && ip.IsLoopback())
	if u.Scheme != "https" && !(allowLoopbackHTTP && u.Scheme == "http" && loopback) {
		return "", errors.New("safety email console origin requires HTTPS or development loopback HTTP")
	}
	u.Path = ""
	return u.String(), nil
}

func NewBrevo(key, origin string, allowLoopbackHTTP bool) (*Brevo, error) {
	canonical, err := ConsoleOrigin(origin, allowLoopbackHTTP)
	if err != nil || strings.TrimSpace(key) == "" {
		return nil, errors.New("invalid safety email configuration")
	}
	return &Brevo{key: key, consoleURL: canonical + "/admin/", endpoint: "https://api.brevo.com/v3/smtp/email", client: &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
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
