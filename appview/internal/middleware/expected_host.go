package middleware

import (
	"fmt"
	"net"
	"net/http"
	pathpkg "path"
	"strconv"
	"strings"

	"social.craftsky/appview/internal/api/envelope"
)

// ExpectedHostPolicy is a validated application-level Host allow-list.
// AllowAnyPort is used only for loopback development, where Compose selects a
// per-checkout published port. Production compares the complete authority
// after normalising the default HTTPS port.
type ExpectedHostPolicy struct {
	Authorities  []string
	AllowAnyPort bool
}

func ExpectedHost(policy ExpectedHostPolicy) func(http.Handler) http.Handler {
	expected := make(map[string]struct{}, len(policy.Authorities))
	for _, authority := range policy.Authorities {
		normalized, err := normalizeAuthority(authority)
		if err == nil {
			expected[normalized] = struct{}{}
		}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authority, err := normalizeAuthority(r.Host)
			allowed := err == nil
			if allowed && policy.AllowAnyPort {
				host, _, splitErr := splitAuthority(authority)
				allowed = splitErr == nil && expectedHostOnly(expected, host)
			} else if allowed {
				_, allowed = expected[authority]
			}
			if !allowed {
				RejectBodyWithoutDrain(w, r)
				w.Header().Set("Cache-Control", "no-store")
				w.Header().Set("X-Content-Type-Options", "nosniff")
				if isV1NamespacePath(r.URL.Path) {
					envelope.WriteError(w, http.StatusMisdirectedRequest, "unexpected_host", "request Host is not accepted", GetRunID(r.Context()), nil)
				} else {
					http.Error(w, "misdirected request", http.StatusMisdirectedRequest)
				}
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func isV1NamespacePath(raw string) bool {
	trimmed := strings.TrimLeft(raw, "/")
	if trimmed == "v1" || strings.HasPrefix(trimmed, "v1/") {
		return true
	}
	cleaned := pathpkg.Clean("/" + strings.TrimLeft(raw, "/"))
	return cleaned == "/v1" || strings.HasPrefix(cleaned, "/v1/")
}

func expectedHostOnly(expected map[string]struct{}, host string) bool {
	for authority := range expected {
		expectedHost, _, err := splitAuthority(authority)
		if err == nil && expectedHost == host {
			return true
		}
	}
	return false
}

func normalizeAuthority(raw string) (string, error) {
	if raw == "" || raw != strings.TrimSpace(raw) || strings.ContainsAny(raw, "\\/@?#\r\n\t ") {
		return "", fmt.Errorf("malformed authority")
	}
	host, port, err := splitAuthority(raw)
	if err != nil {
		return "", err
	}
	host = strings.ToLower(host)
	if host == "" || strings.HasSuffix(host, ".") {
		return "", fmt.Errorf("malformed authority")
	}
	if port == "443" {
		port = ""
	}
	if port == "" {
		return host, nil
	}
	return net.JoinHostPort(host, port), nil
}

func splitAuthority(raw string) (host string, port string, err error) {
	if strings.HasSuffix(raw, ":") {
		return "", "", fmt.Errorf("malformed authority")
	}
	if strings.HasPrefix(raw, "[") {
		host, port, err = net.SplitHostPort(raw)
		if err != nil {
			return "", "", fmt.Errorf("malformed authority")
		}
	} else {
		switch strings.Count(raw, ":") {
		case 0:
			host = raw
		case 1:
			host, port, err = net.SplitHostPort(raw)
			if err != nil {
				return "", "", fmt.Errorf("malformed authority")
			}
		default:
			return "", "", fmt.Errorf("malformed authority")
		}
	}
	if port != "" {
		portNumber, parseErr := strconv.Atoi(port)
		if parseErr != nil || portNumber < 1 || portNumber > 65535 {
			return "", "", fmt.Errorf("malformed authority")
		}
	}
	return strings.Trim(host, "[]"), port, nil
}
