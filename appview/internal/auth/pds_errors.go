package auth

import (
	"errors"
	"strings"
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
	return err
}

func isInvalidGrantRefreshError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "invalid_grant") &&
		(strings.Contains(msg, "token refresh failed") ||
			strings.Contains(msg, "failed to refresh OAuth tokens"))
}
