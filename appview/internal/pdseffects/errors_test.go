package pdseffects

import (
	"errors"
	"strings"
	"testing"
)

func TestTypedEffectErrorsDoNotRenderRemoteIdentityOrCause(t *testing.T) {
	cause := errors.New("upstream response exposed secret-token and did:plc:private")
	tests := []struct {
		name     string
		err      error
		sentinel error
	}{
		{
			name: "ambiguous",
			err: &OutcomeAmbiguousError{
				OperationID: "safe-operation-id",
				ExactKey:    "at://did:plc:private/social.craftsky.feed.post/private-rkey",
				Cause:       cause,
			},
			sentinel: ErrOutcomeAmbiguous,
		},
		{
			name: "conflict",
			err: &ConflictError{
				OperationID: "safe-operation-id",
				ExactKey:    "at://did:plc:private/social.craftsky.feed.post/private-rkey",
				Cause:       cause,
			},
			sentinel: ErrEffectConflict,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			message := test.err.Error()
			for _, forbidden := range []string{
				"secret-token",
				"did:plc:private",
				"private-rkey",
			} {
				if strings.Contains(message, forbidden) {
					t.Fatalf("safe error message %q contains %q", message, forbidden)
				}
			}
			if !strings.Contains(message, "safe-operation-id") {
				t.Fatalf("safe error message %q omitted operation correlation", message)
			}
			if !errors.Is(test.err, test.sentinel) || !errors.Is(test.err, cause) {
				t.Fatalf("typed error lost classification: %v", test.err)
			}
		})
	}
}
