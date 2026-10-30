package auth

import (
	"fmt"
	"net/url"
	"slices"
	"strings"
)

// OAuthCallbackDiagnostic adds fixed validation facts without retaining provider
// scope strings, account identifiers or callback credentials. The original cause
// remains available to existing errors.Is/errors.As callers.
type OAuthCallbackDiagnostic struct {
	reason       string
	cause        error
	scopeSummary string
}

func (e *OAuthCallbackDiagnostic) Error() string  { return e.cause.Error() }
func (e *OAuthCallbackDiagnostic) Unwrap() error  { return e.cause }
func (e *OAuthCallbackDiagnostic) Reason() string { return e.reason }
func (e *OAuthCallbackDiagnostic) DiagnosticMessage() string {
	return "OAuth callback check failed: " + e.reason + e.scopeSummary
}

func callbackDiagnostic(reason string, cause error) error {
	return &OAuthCallbackDiagnostic{reason: reason, cause: cause}
}

func scopeGrantDiagnostic(raw string, required []string) error {
	granted := strings.Split(raw, " ")
	seen := make(map[string]struct{}, len(granted))
	reason := "scope_required_missing"
	for _, scope := range granted {
		if scope == "" {
			reason = "scope_empty_token"
			break
		}
		invalid := false
		for _, character := range []byte(scope) {
			if character != 0x21 && (character < 0x23 || character > 0x5b) && (character < 0x5d || character > 0x7e) {
				invalid = true
				break
			}
		}
		if invalid {
			reason = "scope_invalid_character"
			break
		}
		if _, exists := seen[scope]; exists {
			reason = "scope_duplicate_token"
			break
		}
		if usesBroadOAuthAuthority(scope) {
			reason = "scope_broad_authority"
			break
		}
		seen[scope] = struct{}{}
	}
	// Only fixed labels for reviewed default scopes may enter diagnostics.
	labels := map[string]string{
		"atproto":                          "atproto",
		"include:social.craftsky.authFull": "craftsky",
		"repo:app.bsky.actor.profile?action=create&action=update": "bluesky_profile",
		"repo:app.bsky.actor.link":                                "profile_links",
		"repo:app.bsky.graph.follow":                              "follow",
		"repo:app.bsky.graph.block":                               "block",
		"blob?accept=image/jpeg&accept=image/png&accept=image/webp&accept=video/mp4&accept=text/vtt": "media",
		"rpc:com.atproto.repo.uploadBlob?aud=*":                                                      "video_upload",
		"rpc:app.bsky.video.getUploadLimits?aud=*":                                                   "video_limits",
	}
	missing := make([]string, 0, len(required))
	alternate := 0
	for _, scope := range required {
		if requiredScopeGranted(scope, granted, seen) {
			continue
		}
		label := "other"
		if known, ok := labels[scope]; ok {
			label = known
		}
		if !slices.Contains(missing, label) {
			missing = append(missing, label)
		}
		// Diagnostic comparison only; authorization continues to use the existing
		// effective-authority validator. Never serialize normalized or raw scope values.
		if len(granted) <= 64 {
			canonical, ok := canonicalDiagnosticScope(scope)
			if ok {
				for _, returned := range granted {
					if value, valid := canonicalDiagnosticScope(returned); valid && value == canonical {
						alternate++
						break
					}
				}
			}
		}
	}
	return &OAuthCallbackDiagnostic{reason: reason, cause: ErrOAuthFlowInvalid,
		scopeSummary: fmt.Sprintf("; required=%d granted=%d missing=%s alternate_forms=%d", len(required), len(granted), strings.Join(missing, ","), alternate)}
}

func canonicalDiagnosticScope(scope string) (string, bool) {
	resource, query, _ := strings.Cut(scope, "?")
	name, positional, hasPositional := strings.Cut(resource, ":")
	parameters, err := url.ParseQuery(query)
	if err != nil {
		return "", false
	}
	if hasPositional {
		key := map[string]string{"repo": "collection", "blob": "accept", "rpc": "lxm", "include": "nsid"}[name]
		value, err := url.QueryUnescape(positional)
		if key == "" || err != nil || len(parameters[key]) > 0 {
			return "", false
		}
		parameters[key] = []string{value}
	}
	for _, values := range parameters {
		slices.Sort(values)
	}
	return name + "?" + parameters.Encode(), true
}

// Select a closed OAuth error code; descriptions, URLs and unknown prose stay private.
func providerCallbackDiagnostic(code string) error {
	reason := "provider_callback_error_unknown"
	switch code {
	case "access_denied", "invalid_request", "invalid_client", "invalid_scope",
		"unauthorized_client", "unsupported_response_type", "server_error", "temporarily_unavailable":
		reason = "provider_callback_error_" + code
	}
	return callbackDiagnostic(reason, ErrOAuthFlowInvalid)
}
