package revenuecat

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var errInvalidWebhookAuthentication = errors.New("invalid webhook authentication")

func VerifyWebhookAuthentication(headers http.Header, body []byte, expectedAuthorization, signingSecret string, now time.Time, tolerance time.Duration) error {
	authorization := headers.Values("Authorization")
	if len(authorization) != 1 || len(authorization[0]) != len(expectedAuthorization) ||
		subtle.ConstantTimeCompare([]byte(authorization[0]), []byte(expectedAuthorization)) != 1 {
		return errInvalidWebhookAuthentication
	}
	signatures := headers.Values("X-RevenueCat-Webhook-Signature")
	if len(signatures) != 1 {
		return errInvalidWebhookAuthentication
	}
	var timestampText, signatureText string
	for _, component := range strings.Split(signatures[0], ",") {
		key, value, ok := strings.Cut(strings.TrimSpace(component), "=")
		if !ok || value == "" {
			return errInvalidWebhookAuthentication
		}
		switch key {
		case "t":
			if timestampText != "" {
				return errInvalidWebhookAuthentication
			}
			timestampText = value
		case "v1":
			if signatureText != "" {
				return errInvalidWebhookAuthentication
			}
			signatureText = value
		}
	}
	unix, err := strconv.ParseInt(timestampText, 10, 64)
	if err != nil || signatureText == "" {
		return errInvalidWebhookAuthentication
	}
	signedAt := time.Unix(unix, 0)
	if signedAt.Before(now.Add(-tolerance)) || signedAt.After(now.Add(tolerance)) {
		return errInvalidWebhookAuthentication
	}
	provided, err := hex.DecodeString(signatureText)
	if err != nil || len(provided) != sha256.Size {
		return errInvalidWebhookAuthentication
	}
	mac := hmac.New(sha256.New, []byte(signingSecret))
	_, _ = mac.Write([]byte(timestampText + "."))
	_, _ = mac.Write(body)
	if !hmac.Equal(mac.Sum(nil), provided) {
		return errInvalidWebhookAuthentication
	}
	return nil
}
