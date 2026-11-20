package observability

import (
	"strconv"
	"strings"

	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

// Indigo reports non-success identity HTTP statuses as strings wrapping a
// sentinel. Select the fixed source and numeric status from its known forms.
func identityHTTPDiagnostic(err error) (message, status string) {
	wrapped, ok := err.(interface{ Unwrap() error })
	if !ok {
		return "", ""
	}
	var rawStatus string
	switch wrapped.Unwrap() {
	case identity.ErrHandleResolutionFailed:
		suffix, ok := strings.CutPrefix(err.Error(), "handle resolution failed: HTTP well-known status ")
		if !ok {
			return "", ""
		}
		var host string
		rawStatus, host, _ = strings.Cut(suffix, " for ")
		if _, err := syntax.ParseHandle(host); err != nil {
			return "", ""
		}
		message = "Handle well-known request failed"
	case identity.ErrDIDResolutionFailed:
		switch {
		case strings.HasPrefix(err.Error(), "DID resolution failed: PLC directory status "):
			rawStatus = strings.TrimPrefix(err.Error(), "DID resolution failed: PLC directory status ")
			message = "PLC directory request failed"
		case strings.HasPrefix(err.Error(), "DID resolution failed: did:web HTTP status "):
			rawStatus = strings.TrimPrefix(err.Error(), "DID resolution failed: did:web HTTP status ")
			message = "DID well-known request failed"
		}
	}
	number, err := strconv.Atoi(rawStatus)
	if err != nil || len(rawStatus) != 3 || number < 100 || number > 599 {
		return "", ""
	}
	return message + " (HTTP " + rawStatus + ")", rawStatus
}
