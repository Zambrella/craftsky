package observability

import (
	"net/http"
	"net/textproto"
	"strings"
)

const redactedValue = "[REDACTED]"

var sensitiveHeaderNames = map[string]struct{}{
	"authorization":             {},
	"cookie":                    {},
	"dpop":                      {},
	"x-craftsky-device-id":      {},
	"x-craftsky-session-token":  {},
	"x-forwarded-authorization": {},
	"x-forwarded-access-token":  {},
	"x-forwarded-refresh-token": {},
	"x-oauth-access-token":      {},
	"x-oauth-refresh-token":     {},
	"x-pds-token":               {},
	"x-session-token":           {},
}

// RedactHeaders returns a copy of headers with secrets and raw identity-like
// values replaced by a fixed marker suitable for local structured logs.
func RedactHeaders(headers http.Header) http.Header {
	redacted := make(http.Header, len(headers))
	for key, values := range headers {
		canonicalKey := textproto.CanonicalMIMEHeaderKey(key)
		if _, ok := sensitiveHeaderNames[strings.ToLower(key)]; ok {
			redacted[canonicalKey] = []string{redactedValue}
			continue
		}
		redacted[canonicalKey] = append([]string(nil), values...)
	}
	return redacted
}
