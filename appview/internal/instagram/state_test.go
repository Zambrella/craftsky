package instagram

import (
	"errors"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

func TestVerificationAttemptStateTransitions(t *testing.T) {
	t.Parallel()

	states := []VerificationAttemptState{
		AttemptPendingDM,
		AttemptProcessing,
		AttemptPendingConfirmation,
		AttemptConfirmed,
		AttemptExpired,
		AttemptCancelled,
		AttemptSuperseded,
		AttemptRejected,
		AttemptConflicted,
	}
	allowed := transitionSet[VerificationAttemptState]{
		{AttemptPendingDM, AttemptProcessing}:           {},
		{AttemptPendingDM, AttemptExpired}:              {},
		{AttemptPendingDM, AttemptCancelled}:            {},
		{AttemptPendingDM, AttemptSuperseded}:           {},
		{AttemptProcessing, AttemptPendingConfirmation}: {},
		{AttemptProcessing, AttemptRejected}:            {},
		{AttemptProcessing, AttemptExpired}:             {},
		{AttemptProcessing, AttemptCancelled}:           {},
		{AttemptProcessing, AttemptSuperseded}:          {},
		{AttemptPendingConfirmation, AttemptConfirmed}:  {},
		{AttemptPendingConfirmation, AttemptConflicted}: {},
		{AttemptPendingConfirmation, AttemptExpired}:    {},
		{AttemptPendingConfirmation, AttemptCancelled}:  {},
		{AttemptPendingConfirmation, AttemptSuperseded}: {},
	}
	assertTransitionMatrix(t, states, allowed, ValidateVerificationAttemptTransition)

	for _, state := range states {
		wantTerminal := state != AttemptPendingDM && state != AttemptProcessing && state != AttemptPendingConfirmation
		if got := state.Terminal(); got != wantTerminal {
			t.Errorf("%s.Terminal() = %t, want %t", state, got, wantTerminal)
		}
	}
	for _, retryCode := range []AttemptRetryCode{
		RetryProfileLookupUnavailable,
		RetryInvalidProfileResponse,
		RetryMembershipInactive,
	} {
		if !retryCode.Valid() {
			t.Errorf("documented retry code %q is invalid", retryCode)
		}
	}
	if AttemptRetryCode("provider-secret").Valid() {
		t.Fatal("unknown/internal retry code was accepted")
	}
}

func TestInstagramLinkImportSuggestionAndConflictTransitions(t *testing.T) {
	t.Parallel()

	assertTransitionMatrix(t,
		[]InstagramLinkState{LinkActive, LinkMembershipInactive, LinkRevoked, LinkSuperseded, LinkDisputed},
		transitionSet[InstagramLinkState]{
			{LinkActive, LinkMembershipInactive}:     {},
			{LinkActive, LinkRevoked}:                {},
			{LinkActive, LinkSuperseded}:             {},
			{LinkMembershipInactive, LinkActive}:     {},
			{LinkMembershipInactive, LinkRevoked}:    {},
			{LinkMembershipInactive, LinkSuperseded}: {},
			{LinkDisputed, LinkActive}:               {},
			{LinkDisputed, LinkRevoked}:              {},
			{LinkDisputed, LinkSuperseded}:           {},
		},
		ValidateInstagramLinkTransition,
	)
	if !LinkRevoked.Terminal() || !LinkSuperseded.Terminal() {
		t.Fatal("revoked and superseded links must be terminal")
	}
	if LinkActive.Terminal() || LinkMembershipInactive.Terminal() || LinkDisputed.Terminal() {
		t.Fatal("active, membership-inactive, and disputed links must remain resolvable")
	}

	assertTransitionMatrix(t,
		[]InstagramImportState{ImportActive, ImportMembershipInactive, ImportExpired},
		transitionSet[InstagramImportState]{
			{ImportActive, ImportMembershipInactive}:  {},
			{ImportActive, ImportExpired}:             {},
			{ImportMembershipInactive, ImportActive}:  {},
			{ImportMembershipInactive, ImportExpired}: {},
		},
		ValidateInstagramImportTransition,
	)
	if !ImportExpired.Terminal() || ImportActive.Terminal() || ImportMembershipInactive.Terminal() {
		t.Fatal("only expired imports are terminal")
	}

	assertTransitionMatrix(t,
		[]InstagramSuggestionState{
			SuggestionPending,
			SuggestionAccepting,
			SuggestionAccepted,
			SuggestionAlreadyFollowing,
			SuggestionDismissed,
			SuggestionInvalidated,
		},
		transitionSet[InstagramSuggestionState]{
			{SuggestionPending, SuggestionAccepting}:          {},
			{SuggestionPending, SuggestionAlreadyFollowing}:   {},
			{SuggestionPending, SuggestionDismissed}:          {},
			{SuggestionPending, SuggestionInvalidated}:        {},
			{SuggestionAccepting, SuggestionPending}:          {},
			{SuggestionAccepting, SuggestionAccepted}:         {},
			{SuggestionAccepting, SuggestionAlreadyFollowing}: {},
			{SuggestionAccepting, SuggestionInvalidated}:      {},
		},
		ValidateInstagramSuggestionTransition,
	)
	for _, state := range []InstagramSuggestionState{
		SuggestionAccepted,
		SuggestionAlreadyFollowing,
		SuggestionDismissed,
		SuggestionInvalidated,
	} {
		if !state.Terminal() {
			t.Errorf("suggestion state %q must be terminal", state)
		}
	}
	if SuggestionPending.Terminal() || SuggestionAccepting.Terminal() {
		t.Fatal("pending and accepting suggestions cannot be terminal")
	}

	assertTransitionMatrix(t,
		[]InstagramConflictState{
			ConflictOpen,
			ConflictResolvedKeepExisting,
			ConflictResolvedRevokeExisting,
			ConflictExpired,
		},
		transitionSet[InstagramConflictState]{
			{ConflictOpen, ConflictResolvedKeepExisting}:   {},
			{ConflictOpen, ConflictResolvedRevokeExisting}: {},
			{ConflictOpen, ConflictExpired}:                {},
		},
		ValidateInstagramConflictTransition,
	)
}

func TestInstagramStateValidationAndOwnerBoundary(t *testing.T) {
	t.Parallel()

	if VerificationAttemptState("futureState").Valid() ||
		InstagramLinkState("futureState").Valid() ||
		InstagramImportState("futureState").Valid() ||
		InstagramSuggestionState("futureState").Valid() ||
		InstagramConflictState("futureState").Valid() {
		t.Fatal("server state validation accepted an unknown public value")
	}

	alice := syntax.DID("did:plc:synthetic-alice")
	bob := syntax.DID("did:plc:synthetic-bob")
	if err := RequireAggregateOwner(alice, alice); err != nil {
		t.Fatalf("matching owner rejected: %v", err)
	}
	if err := RequireAggregateOwner(alice, bob); !errors.Is(err, ErrInstagramResourceNotFound) {
		t.Fatalf("wrong-owner error = %v, want privacy-safe not found", err)
	}
}

type transitionSet[S comparable] map[[2]S]struct{}

func assertTransitionMatrix[S interface {
	comparable
	Valid() bool
}](t *testing.T, states []S, allowed transitionSet[S], validate func(S, S) error) {
	t.Helper()
	for _, from := range states {
		if !from.Valid() {
			t.Errorf("documented state %v is invalid", from)
		}
		for _, to := range states {
			err := validate(from, to)
			_, explicitlyAllowed := allowed[[2]S{from, to}]
			wantAllowed := from == to || explicitlyAllowed
			if wantAllowed && err != nil {
				t.Errorf("transition %v -> %v rejected: %v", from, to, err)
			}
			if !wantAllowed && err == nil {
				t.Errorf("transition %v -> %v unexpectedly allowed", from, to)
			}
		}
	}
}
