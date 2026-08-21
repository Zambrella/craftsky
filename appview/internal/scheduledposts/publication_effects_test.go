package scheduledposts

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"social.craftsky/appview/internal/pdseffects"
)

func TestScheduledPublicationEffectIdentitiesIncludeGenerationAndPayloadVersion(t *testing.T) {
	t.Parallel()

	claim := PublishingClaim{
		ID:              uuid.MustParse("f8ad39ab-6cda-44c5-a260-a4e48dcb42fb"),
		OwnerGeneration: 7,
		PayloadVersion:  3,
	}
	if got, want := scheduledRecordEffectIdentity(claim),
		"scheduled-post/f8ad39ab-6cda-44c5-a260-a4e48dcb42fb/g7/v3/record"; got != want {
		t.Fatalf("record effect identity = %q, want %q", got, want)
	}
	if got, want := scheduledBlobEffectIdentity(claim, 2),
		"scheduled-post/f8ad39ab-6cda-44c5-a260-a4e48dcb42fb/g7/v3/blob/2"; got != want {
		t.Fatalf("blob effect identity = %q, want %q", got, want)
	}

	otherGeneration := claim
	otherGeneration.OwnerGeneration++
	if scheduledRecordEffectIdentity(otherGeneration) == scheduledRecordEffectIdentity(claim) {
		t.Fatal("record effect identity was reused across owner generations")
	}
	otherVersion := claim
	otherVersion.PayloadVersion++
	if scheduledRecordEffectIdentity(otherVersion) == scheduledRecordEffectIdentity(claim) {
		t.Fatal("record effect identity was reused across payload versions")
	}
}

func TestScheduledEffectErrorsMapToSafePublicationDecisions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		kind scheduledEffectKind
		err  error
		want error
	}{
		{
			name: "record conflict",
			kind: scheduledRecordEffect,
			err:  &pdseffects.ConflictError{OperationID: "record"},
			want: ErrRecordConflict,
		},
		{
			name: "record rejected",
			kind: scheduledRecordEffect,
			err:  pdseffects.ErrEffectRejected,
			want: ErrRecordConflict,
		},
		{
			name: "blob conflict",
			kind: scheduledBlobEffect,
			err:  &pdseffects.ConflictError{OperationID: "blob"},
			want: ErrMediaInvalid,
		},
		{
			name: "ambiguous",
			kind: scheduledRecordEffect,
			err:  &pdseffects.OutcomeAmbiguousError{OperationID: "record"},
			want: ErrPDSUnavailable,
		},
		{
			name: "dependency",
			kind: scheduledBlobEffect,
			err:  errors.New("database unavailable"),
			want: ErrPDSUnavailable,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mapped := classifyScheduledEffectError(test.kind, test.err)
			if !errors.Is(mapped, test.want) || !errors.Is(mapped, test.err) {
				t.Fatalf("mapped error = %v, want both %v and original %v", mapped, test.want, test.err)
			}
		})
	}
}
