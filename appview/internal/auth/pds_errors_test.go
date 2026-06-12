package auth_test

import (
	"errors"
	"testing"

	"social.craftsky/appview/internal/auth"
)

func TestTranslatePDSError_InvalidGrantRefreshIsExpiredSession(t *testing.T) {
	err := errors.New("failed to refresh OAuth tokens: token refresh failed: auth server request failed (HTTP 400): invalid_grant")
	got := auth.TranslatePDSError(err)
	if !errors.Is(got, auth.ErrPDSSessionExpired) {
		t.Fatalf("TranslatePDSError = %v, want ErrPDSSessionExpired", got)
	}
	if !errors.Is(got, err) {
		t.Fatalf("TranslatePDSError should preserve original error, got %v", got)
	}
}

func TestTranslatePDSError_OAuthSessionNotFoundIsExpiredSession(t *testing.T) {
	got := auth.TranslatePDSError(auth.ErrOAuthSessionNotFound)
	if !errors.Is(got, auth.ErrPDSSessionExpired) {
		t.Fatalf("TranslatePDSError = %v, want ErrPDSSessionExpired", got)
	}
}

func TestTranslatePDSError_DPopNonceAloneIsNotExpiredSession(t *testing.T) {
	err := errors.New("auth server request failed (HTTP 400): use_dpop_nonce")
	got := auth.TranslatePDSError(err)
	if errors.Is(got, auth.ErrPDSSessionExpired) {
		t.Fatalf("TranslatePDSError = %v, did not want ErrPDSSessionExpired", got)
	}
	if got != err {
		t.Fatalf("TranslatePDSError = %v, want original error", got)
	}
}

func TestTranslatePDSError_GenericErrorPassesThrough(t *testing.T) {
	err := errors.New("pds down")
	got := auth.TranslatePDSError(err)
	if got != err {
		t.Fatalf("TranslatePDSError = %v, want original error", got)
	}
}
