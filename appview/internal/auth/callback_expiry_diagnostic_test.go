package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/url"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"social.craftsky/appview/internal/ctxkeys"
)

func TestCallbackStorageFailureIsNotInvalidFlow(t *testing.T) {
	// A closed pool fails immediately, without needing an external database.
	pool, err := pgxpool.New(context.Background(), "postgres://unused@localhost/craftsky_test")
	if err != nil {
		t.Fatal(err)
	}
	pool.Close()
	service := &OAuthFlowService{
		store:                    &PostgresAuthStore{pool: pool},
		callbackOperationTimeout: time.Second,
	}
	err = service.CompleteCallback(context.Background(), url.Values{"state": {"private-state-canary"}},
		func(context.Context, OAuthCallbackResult) error {
			t.Fatal("storage failure reached finalization")
			return nil
		})
	var diagnostic *OAuthCallbackDiagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Reason() != "callback_request_load_failed" {
		t.Fatalf("missing storage failure diagnostic: %T", err)
	}
	if errors.Is(err, ErrOAuthFlowInvalid) {
		t.Fatal("storage failure was mislabeled as invalid or expired user flow")
	}
}

func TestCallbackExpiryIsValidationButStorageFailureIsActionable(t *testing.T) {
	for _, test := range []struct {
		name     string
		cause    error
		category string
	}{
		{"owner generation changed", callbackDiagnostic("callback_owner_generation_changed", ErrOAuthFlowInvalid), "validation"},
		{"expired", callbackDiagnostic("callback_request_unavailable", ErrOAuthFlowInvalid), "validation"},
		{"missing", callbackDiagnostic("callback_request_unavailable", errors.Join(ErrOAuthFlowInvalid, ErrOAuthSessionNotFound)), "validation"},
		{"storage", callbackDiagnostic("callback_request_load_failed", errors.New("private-storage-canary")), "finalization"},
		{"exchange", callbackDiagnostic("token_exchange_failed", errors.New("private-provider-canary")), "finalization"},
	} {
		t.Run(test.name, func(t *testing.T) {
			called := false
			ctx := ctxkeys.WithDiagnosticBoundary(context.Background(), func(_ context.Context, cause error, fields map[string]any) []slog.Attr {
				called = true
				if cause != test.cause || fields["error_category"] != test.category {
					t.Fatalf("cause/category not preserved: %T %v", cause, fields["error_category"])
				}
				return nil
			})
			authLogErrorAttrs(ctx, "", "oauth.callback", "finalization", test.cause)
			if !called {
				t.Fatal("failure did not reach diagnostic boundary")
			}
		})
	}
}

func TestQuarantineInvalidInputKeepsReasonAndSentinel(t *testing.T) {
	store := &PostgresAuthStore{}
	err := store.quarantineCallbackCredential(t.Context(), "", uuid.Nil, oauth.ClientSessionData{}, time.Time{}, "")
	var diagnostic *OAuthCallbackDiagnostic
	if !errors.Is(err, ErrCallbackAttemptInvalid) || !errors.As(err, &diagnostic) || diagnostic.Reason() != "quarantine_invalid_input" {
		t.Fatal("quarantine reason or original sentinel lost")
	}
}
