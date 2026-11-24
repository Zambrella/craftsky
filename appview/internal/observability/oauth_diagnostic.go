package observability

import (
	"regexp"
	"strconv"
	"strings"
)

// Indigo exposes these failures as formatted errors rather than typed HTTP
// errors. Match only its complete, fixed formats at the OAuth call boundary;
// never admit arbitrary provider prose, URLs or error descriptions.
var oauthPARFailure = regexp.MustCompile(`^(?:PAR request failed|(?:initial OAuth token request: )?initial token request failed) \(HTTP ([1-5][0-9]{2})\): ([a-z_]+)$`)
var oauthDiscoveryFailure = regexp.MustCompile(`^HTTP error fetching (protected resource document|auth server metadata): ([1-5][0-9]{2})$`)
var oauthRefreshResponseFailure = regexp.MustCompile(`^(?:token refresh failed|token refresh failed: auth server request failed) \(HTTP ([1-5][0-9]{2})\): `)

type oauthRequestDiagnostic struct {
	cause        error
	message      string
	status       string
	providerCode string
}

func (e *oauthRequestDiagnostic) Error() string { return e.cause.Error() }
func (e *oauthRequestDiagnostic) Unwrap() error { return e.cause }

// OAuthRequestFailure selects safe facts from the pinned Indigo OAuth API's
// returned error while retaining the original error for classification/stacks.
func OAuthRequestFailure(err error) error {
	if err == nil {
		return nil
	}
	if match := oauthPARFailure.FindStringSubmatch(err.Error()); match != nil {
		return &oauthRequestDiagnostic{cause: err, message: "OAuth provider request failed", status: match[1], providerCode: oauthDiagnosticCode(match[2])}
	}
	if match := oauthDiscoveryFailure.FindStringSubmatch(err.Error()); match != nil {
		// The document names come from the two fixed library formats above.
		return &oauthRequestDiagnostic{cause: err, message: "OAuth " + match[1] + " request failed", status: match[2]}
	}
	return err
}

// OAuthRefreshFailure is applied to errors returned by Indigo's authenticated
// API client. A proxied request can fail in OAuth refresh before it completes
// at the PDS. Preserve that distinction without exporting the refresh response.
func OAuthRefreshFailure(err error) error {
	if err == nil {
		return nil
	}
	detail, ok := strings.CutPrefix(err.Error(), "failed to refresh OAuth tokens: ")
	if !ok {
		return err
	}
	selected := &oauthRequestDiagnostic{cause: err, message: "OAuth token refresh failed"}
	if match := oauthRefreshResponseFailure.FindStringSubmatch(detail); match != nil {
		selected.status = match[1]
		selected.providerCode = oauthDiagnosticCode(detail[len(match[0]):])
	}
	return selected
}

func oauthDiagnosticCode(reason string) string {
	if len(reason) > 64 {
		return ""
	}
	code := "oauth." + reason
	if knownProviderDiagnosticCode(code) {
		return code
	}
	return ""
}

func (e *oauthRequestDiagnostic) diagnosticCause() DiagnosticCause {
	cause := DiagnosticCause{Type: "*observability.oauthRequestDiagnostic", Message: e.message, Code: e.status, ProviderCode: e.providerCode}
	if status, err := strconv.Atoi(e.status); err == nil && status >= 100 && status <= 599 {
		cause.Message += " (HTTP " + e.status + ")"
	}
	return cause
}
