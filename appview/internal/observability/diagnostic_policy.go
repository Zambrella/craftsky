package observability

import (
	"net/url"
	"regexp"
	"unicode/utf8"
)

const MaxDiagnosticTextBytes = 2048

var diagnosticCredential = regexp.MustCompile(`(?i)\b(token|access[_-]?token|refresh[_-]?token|session[_-]?token|service[_-]?token|password|client[_-]?secret|pkce[_-]?verifier|dpop|authorization|cookie|set-cookie|code|state|private[_-]?key|webhook[_-]?signature|dsn|x-amz-signature|x-goog-signature)\s*[:=]\s*(?:"[^"]*"|'[^']*'|[^\s,;]+)`)
var diagnosticBearer = regexp.MustCompile(`(?i)\bbearer\s+[^\s,;]+`)

var diagnosticURL = regexp.MustCompile(`(?i)https?://[^\s<>"']+`)
var diagnosticEmail = regexp.MustCompile(`(?i)[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}`)
var diagnosticLocalPath = regexp.MustCompile(`(?:/(?:Users|home|private|tmp|var)/[^\s"']+|[A-Za-z]:\\[^\s"']+)`)
var diagnosticPrivateKey = regexp.MustCompile(`(?s)-----BEGIN [A-Z ]*PRIVATE KEY-----.*?-----END [A-Z ]*PRIVATE KEY-----`)

// This is only for text admitted by a reviewed source adapter. It is not a
// classifier for arbitrary private prose; unknown messages are never admitted.
func sanitizeKnownDiagnosticText(value string) string {
	// Decode supported URL encodings before replacement so escaped secrets cannot
	// survive or be partially exposed when the selected field is truncated.
	for i := 0; i < 2; i++ {
		decoded, err := url.PathUnescape(value)
		if err != nil || decoded == value {
			break
		}
		value = decoded
	}
	value = diagnosticPrivateKey.ReplaceAllString(value, redactedValue)
	value = diagnosticURL.ReplaceAllStringFunc(value, func(raw string) string {
		parsed, err := url.Parse(raw)
		if err != nil || parsed.Hostname() == "" {
			return redactedValue
		}
		// A generic URL is not a public-record adapter. Remove all capabilities,
		// userinfo, query and arbitrary path rather than guessing their provenance.
		return parsed.Scheme + "://[REDACTED]"
	})
	value = diagnosticEmail.ReplaceAllString(value, redactedValue)
	value = diagnosticLocalPath.ReplaceAllString(value, redactedValue)
	value = diagnosticCredential.ReplaceAllString(value, "$1="+redactedValue)
	value = diagnosticBearer.ReplaceAllString(value, "Bearer "+redactedValue)
	if len(value) <= MaxDiagnosticTextBytes {
		return value
	}
	const marker = "[TRUNCATED]"
	end := MaxDiagnosticTextBytes - len(marker)
	for end > 0 && !utf8.ValidString(value[:end]) {
		end--
	}
	return value[:end] + marker
}

func boundDiagnosticText(value string, limit int) string {
	value = sanitizeKnownDiagnosticText(value)
	if len(value) <= limit {
		return value
	}
	const marker = "[TRUNCATED]"
	end := limit - len(marker)
	for end > 0 && !utf8.ValidString(value[:end]) {
		end--
	}
	return value[:end] + marker
}
