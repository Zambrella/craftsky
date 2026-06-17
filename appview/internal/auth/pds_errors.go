package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/bluesky-social/indigo/atproto/atclient"
)

// ErrPDSSessionExpired means the stored OAuth session for the user's PDS can
// no longer be resumed/refreshed. Callers should require a fresh sign-in.
var ErrPDSSessionExpired = errors.New("pds session expired")

// TranslatePDSError wraps known terminal OAuth/PDS auth failures in
// ErrPDSSessionExpired while preserving unrelated upstream errors.
func TranslatePDSError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrPDSSessionExpired) {
		return err
	}
	if errors.Is(err, ErrOAuthSessionNotFound) {
		return errors.Join(ErrPDSSessionExpired, err)
	}
	if isInvalidGrantRefreshError(err) {
		return errors.Join(ErrPDSSessionExpired, err)
	}
	var apiErr *atclient.APIError
	if errors.As(err, &apiErr) && isPDSAuthAPIError(apiErr) {
		return errors.Join(ErrPDSSessionExpired, err)
	}
	return err
}

func isInvalidGrantRefreshError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "invalid_grant") &&
		(strings.Contains(msg, "token refresh failed") ||
			strings.Contains(msg, "failed to refresh OAuth tokens"))
}

func isPDSAuthAPIError(err *atclient.APIError) bool {
	if err == nil {
		return false
	}
	if err.StatusCode == http.StatusUnauthorized {
		return true
	}
	name := strings.ToLower(strings.TrimSpace(err.Name))
	switch name {
	case "authenticationrequired", "expiredtoken", "invalidtoken", "invalidaccesstoken":
		return true
	}
	message := strings.ToLower(err.Message)
	return strings.Contains(message, "expired token") ||
		strings.Contains(message, "invalid token") ||
		strings.Contains(message, "invalid access token")
}
