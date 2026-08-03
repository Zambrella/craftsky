package scheduledposts

import (
	"context"
	"fmt"
	"testing"
)

func TestClassifyPublicationFailure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		err             error
		wantDisposition FailureDisposition
		wantCode        string
	}{
		{name: "timeout", err: context.DeadlineExceeded, wantDisposition: FailureRetry, wantCode: "dependency_unavailable"},
		{name: "authentication unavailable", err: fmt.Errorf("wrapped: %w", ErrAuthUnavailable), wantDisposition: FailureRetry, wantCode: "auth_unavailable"},
		{name: "object unavailable", err: fmt.Errorf("wrapped: %w", ErrObjectUnavailable), wantDisposition: FailureRetry, wantCode: "object_unavailable"},
		{name: "PDS unavailable", err: fmt.Errorf("wrapped: %w", ErrPDSUnavailable), wantDisposition: FailureRetry, wantCode: "pds_unavailable"},
		{name: "unknown dependency error", err: fmt.Errorf("unexpected dependency failure"), wantDisposition: FailureRetry, wantCode: "dependency_unavailable"},
		{name: "policy invalid", err: fmt.Errorf("wrapped: %w", ErrPolicyInvalid), wantDisposition: FailureNeedsAttention, wantCode: "policy_invalid"},
		{name: "media invalid", err: fmt.Errorf("wrapped: %w", ErrMediaInvalid), wantDisposition: FailureNeedsAttention, wantCode: "media_invalid"},
		{name: "record conflict", err: fmt.Errorf("wrapped: %w", ErrRecordConflict), wantDisposition: FailureNeedsAttention, wantCode: "record_conflict"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := ClassifyPublicationFailure(test.err)
			if got.Disposition != test.wantDisposition || got.SafeCode != test.wantCode {
				t.Fatalf("ClassifyPublicationFailure() = %#v, want disposition %q code %q", got, test.wantDisposition, test.wantCode)
			}
		})
	}
}
