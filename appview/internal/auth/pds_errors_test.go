package auth_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/bluesky-social/indigo/atproto/atclient"

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

func TestTranslatePDSError_APIAuthFailuresAreExpiredSession(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
	}{
		{
			name: "unauthorized status",
			err:  &atclient.APIError{StatusCode: 401, Name: "AuthenticationRequired"},
		},
		{
			name: "wrapped expired token name",
			err: fmt.Errorf(
				"pds write: %w",
				&atclient.APIError{StatusCode: 400, Name: "ExpiredToken"},
			),
		},
		{
			name: "invalid access token message",
			err: &atclient.APIError{
				StatusCode: 400,
				Name:       "BadToken",
				Message:    "invalid access token",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := auth.TranslatePDSError(tc.err)
			if !errors.Is(got, auth.ErrPDSSessionExpired) {
				t.Fatalf("TranslatePDSError = %v, want ErrPDSSessionExpired", got)
			}
			if !errors.Is(got, tc.err) {
				t.Fatalf("TranslatePDSError should preserve original error, got %v", got)
			}
		})
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
