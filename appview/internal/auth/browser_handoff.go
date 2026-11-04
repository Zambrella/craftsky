package auth

import (
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"strconv"
)

const BrowserLoginCompletionPath = "/auth/browser/complete"
const BrowserDeletionCompletionPath = "/account-deletion/browser/reauth-complete"
const DefaultWebAppOrigin = "https://app.craftsky.social"
const PreviewWebAppOrigin = "https://preview-craftsky-public-app.hugetoothstudios.workers.dev"

// ValidateWebAppOrigin permits the controlled production and preview origins, or an
// explicitly configured local development origin. It never accepts a caller's
// Host/Origin header as redirect authority.
func ValidateWebAppOrigin(raw string, allowLoopback bool) (string, error) {
	if raw == "" {
		raw = DefaultWebAppOrigin
	}
	u, err := url.Parse(raw)
	if err != nil || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.RawPath != "" || (u.Path != "" && u.Path != "/") {
		return "", errors.New("invalid web app origin")
	}
	if u.Scheme == "https" && u.Host == "app.craftsky.social" {
		return DefaultWebAppOrigin, nil
	}
	if u.Scheme == "https" && "https://"+u.Host == PreviewWebAppOrigin {
		return PreviewWebAppOrigin, nil
	}
	if allowLoopback && u.Scheme == "http" && u.Hostname() == "127.0.0.1" && u.Port() != "" {
		port, err := strconv.Atoi(u.Port())
		if err == nil && port > 0 && port <= 65535 {
			return "http://" + u.Host, nil
		}
	}
	return "", errors.New("invalid web app origin")
}

// ValidateBrowserCompletionURI requires a fixed purpose-specific path and one
// random browser-intent state. The BFF compares that state with its HttpOnly
// intent cookie before exchanging the device-bound code.
func ValidateBrowserCompletionURI(raw, origin, path string) error {
	if path != BrowserLoginCompletionPath && path != BrowserDeletionCompletionPath {
		return ErrOAuthFlowInvalid
	}
	canonical, err := ValidateWebAppOrigin(origin, true)
	if err != nil {
		return ErrOAuthFlowInvalid
	}
	u, err := url.Parse(raw)
	if err != nil || len(raw) > 1024 || u.User != nil || u.Fragment != "" || u.RawPath != "" || u.Path != path {
		return ErrOAuthFlowInvalid
	}
	targetOrigin := u.Scheme + "://" + u.Host
	// Production also permits this one account-owned stable version alias. Never
	// allow arbitrary workers.dev hosts, version IDs, aliases or request origins.
	if targetOrigin != canonical && !(canonical == DefaultWebAppOrigin && targetOrigin == PreviewWebAppOrigin) {
		return ErrOAuthFlowInvalid
	}
	query, err := url.ParseQuery(u.RawQuery)
	if err != nil || len(query) != 1 || len(query["state"]) != 1 {
		return ErrOAuthFlowInvalid
	}
	state := query.Get("state")
	decoded, err := base64.RawURLEncoding.DecodeString(state)
	if err != nil || len(decoded) != 32 || base64.RawURLEncoding.EncodeToString(decoded) != state {
		return ErrOAuthFlowInvalid
	}
	return nil
}

func validStoredBrowserCompletion(raw, path string) bool {
	u, err := url.Parse(raw)
	return err == nil && ValidateBrowserCompletionURI(raw, u.Scheme+"://"+u.Host, path) == nil
}

func browserCompletionURL(raw, origin, path string, values url.Values) (string, error) {
	if err := ValidateBrowserCompletionURI(raw, origin, path); err != nil {
		return "", err
	}
	u, _ := url.Parse(raw)
	query := u.Query()
	for key, items := range values {
		query[key] = items
	}
	u.RawQuery = query.Encode()
	return u.String(), nil
}

func renderBrowserCompletion(w http.ResponseWriter, location string) {
	setCallbackSecurityHeaders(w, "", "")
	w.Header().Set("Location", location)
	w.WriteHeader(http.StatusSeeOther)
}

func validHandoffDestination(mode HandoffMode, destination string) bool {
	switch mode {
	case HandoffVerifiedLink, HandoffDevScheme:
		return destination == ""
	case HandoffLoopback:
		return loopbackRedirectPattern.MatchString(destination)
	case HandoffBrowser:
		return validStoredBrowserCompletion(destination, BrowserLoginCompletionPath)
	default:
		return false
	}
}
