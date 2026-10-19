package observability

import (
	"regexp"
	"strconv"
)

// Indigo exposes these failures as formatted errors rather than typed HTTP
// errors. Match only its complete, fixed formats at the OAuth call boundary;
// never admit arbitrary provider prose, URLs or error descriptions.
var oauthPARFailure = regexp.MustCompile(`^PAR request failed \(HTTP ([1-5][0-9]{2})\): ([a-z_]+)$`)
var oauthDiscoveryFailure = regexp.MustCompile(`^HTTP error fetching (?:protected resource document|auth server metadata): ([1-5][0-9]{2})$`)

type oauthRequestDiagnostic struct {
	cause        error
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
		code := "oauth." + match[2]
		if !knownProviderDiagnosticCode(code) {
			code = ""
		}
		return &oauthRequestDiagnostic{cause: err, status: match[1], providerCode: code}
	}
	if match := oauthDiscoveryFailure.FindStringSubmatch(err.Error()); match != nil {
		return &oauthRequestDiagnostic{cause: err, status: match[1]}
	}
	return err
}

func (e *oauthRequestDiagnostic) diagnosticCause() DiagnosticCause {
	cause := DiagnosticCause{Type: "*observability.oauthRequestDiagnostic", Message: "OAuth provider request failed", Code: e.status, ProviderCode: e.providerCode}
	if status, err := strconv.Atoi(e.status); err == nil && status >= 100 && status <= 599 {
		cause.Message += " (HTTP " + e.status + ")"
	}
	return cause
}
