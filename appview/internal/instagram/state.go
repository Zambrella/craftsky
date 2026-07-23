package instagram

import (
	"errors"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

var (
	ErrInvalidInstagramState     = errors.New("invalid Instagram state")
	ErrInstagramStateTransition  = errors.New("invalid Instagram state transition")
	ErrInstagramResourceNotFound = errors.New("Instagram resource not found")
)

type VerificationAttemptState string

const (
	AttemptPendingDM           VerificationAttemptState = "pendingDm"
	AttemptProcessing          VerificationAttemptState = "processing"
	AttemptPendingConfirmation VerificationAttemptState = "pendingConfirmation"
	AttemptConfirmed           VerificationAttemptState = "confirmed"
	AttemptExpired             VerificationAttemptState = "expired"
	AttemptCancelled           VerificationAttemptState = "cancelled"
	AttemptSuperseded          VerificationAttemptState = "superseded"
	AttemptRejected            VerificationAttemptState = "rejected"
	AttemptConflicted          VerificationAttemptState = "conflicted"
)

func (s VerificationAttemptState) Valid() bool {
	switch s {
	case AttemptPendingDM,
		AttemptProcessing,
		AttemptPendingConfirmation,
		AttemptConfirmed,
		AttemptExpired,
		AttemptCancelled,
		AttemptSuperseded,
		AttemptRejected,
		AttemptConflicted:
		return true
	default:
		return false
	}
}

func (s VerificationAttemptState) Terminal() bool {
	return s.Valid() && s != AttemptPendingDM && s != AttemptProcessing && s != AttemptPendingConfirmation
}

type AttemptRetryCode string

const (
	RetryProfileLookupUnavailable AttemptRetryCode = "profileLookupUnavailable"
	RetryInvalidProfileResponse   AttemptRetryCode = "invalidProfileResponse"
	RetryMembershipInactive       AttemptRetryCode = "membershipInactive"
)

func (c AttemptRetryCode) Valid() bool {
	switch c {
	case RetryProfileLookupUnavailable, RetryInvalidProfileResponse, RetryMembershipInactive:
		return true
	default:
		return false
	}
}

func ValidateVerificationAttemptTransition(from, to VerificationAttemptState) error {
	if !from.Valid() || !to.Valid() {
		return ErrInvalidInstagramState
	}
	if from == to {
		return nil
	}
	allowed := false
	switch from {
	case AttemptPendingDM:
		allowed = oneOf(to, AttemptProcessing, AttemptExpired, AttemptCancelled, AttemptSuperseded)
	case AttemptProcessing:
		allowed = oneOf(to, AttemptPendingConfirmation, AttemptRejected, AttemptExpired, AttemptCancelled, AttemptSuperseded)
	case AttemptPendingConfirmation:
		allowed = oneOf(to, AttemptConfirmed, AttemptConflicted, AttemptExpired, AttemptCancelled, AttemptSuperseded)
	}
	return transitionResult(allowed, from, to)
}

type InstagramLinkState string

const (
	LinkActive             InstagramLinkState = "active"
	LinkMembershipInactive InstagramLinkState = "membershipInactive"
	LinkRevoked            InstagramLinkState = "revoked"
	LinkSuperseded         InstagramLinkState = "superseded"
	LinkDisputed           InstagramLinkState = "disputed"
)

func (s InstagramLinkState) Valid() bool {
	switch s {
	case LinkActive, LinkMembershipInactive, LinkRevoked, LinkSuperseded, LinkDisputed:
		return true
	default:
		return false
	}
}

func (s InstagramLinkState) Terminal() bool {
	return s == LinkRevoked || s == LinkSuperseded
}

func ValidateInstagramLinkTransition(from, to InstagramLinkState) error {
	if !from.Valid() || !to.Valid() {
		return ErrInvalidInstagramState
	}
	if from == to {
		return nil
	}
	allowed := false
	switch from {
	case LinkActive:
		allowed = oneOf(to, LinkMembershipInactive, LinkRevoked, LinkSuperseded)
	case LinkMembershipInactive:
		allowed = oneOf(to, LinkActive, LinkRevoked, LinkSuperseded)
	case LinkDisputed:
		allowed = oneOf(to, LinkActive, LinkRevoked, LinkSuperseded)
	}
	return transitionResult(allowed, from, to)
}

type InstagramImportState string

const (
	ImportActive             InstagramImportState = "active"
	ImportMembershipInactive InstagramImportState = "membershipInactive"
)

func (s InstagramImportState) Valid() bool {
	switch s {
	case ImportActive, ImportMembershipInactive:
		return true
	default:
		return false
	}
}

func (s InstagramImportState) Terminal() bool { return false }

func ValidateInstagramImportTransition(from, to InstagramImportState) error {
	if !from.Valid() || !to.Valid() {
		return ErrInvalidInstagramState
	}
	if from == to {
		return nil
	}
	allowed := false
	switch from {
	case ImportActive:
		allowed = oneOf(to, ImportMembershipInactive)
	case ImportMembershipInactive:
		allowed = oneOf(to, ImportActive)
	}
	return transitionResult(allowed, from, to)
}

type InstagramSuggestionState string

const (
	SuggestionPending          InstagramSuggestionState = "pending"
	SuggestionAccepting        InstagramSuggestionState = "accepting"
	SuggestionAccepted         InstagramSuggestionState = "accepted"
	SuggestionAlreadyFollowing InstagramSuggestionState = "alreadyFollowing"
	SuggestionDismissed        InstagramSuggestionState = "dismissed"
	SuggestionInvalidated      InstagramSuggestionState = "invalidated"
)

func (s InstagramSuggestionState) Valid() bool {
	switch s {
	case SuggestionPending,
		SuggestionAccepting,
		SuggestionAccepted,
		SuggestionAlreadyFollowing,
		SuggestionDismissed,
		SuggestionInvalidated:
		return true
	default:
		return false
	}
}

func (s InstagramSuggestionState) Terminal() bool {
	return s == SuggestionAccepted || s == SuggestionAlreadyFollowing || s == SuggestionDismissed || s == SuggestionInvalidated
}

func ValidateInstagramSuggestionTransition(from, to InstagramSuggestionState) error {
	if !from.Valid() || !to.Valid() {
		return ErrInvalidInstagramState
	}
	if from == to {
		return nil
	}
	allowed := false
	switch from {
	case SuggestionPending:
		allowed = oneOf(to, SuggestionAccepting, SuggestionAlreadyFollowing, SuggestionDismissed, SuggestionInvalidated)
	case SuggestionAccepting:
		allowed = oneOf(to, SuggestionPending, SuggestionAccepted, SuggestionAlreadyFollowing, SuggestionInvalidated)
	}
	return transitionResult(allowed, from, to)
}

type InstagramConflictState string

const (
	ConflictOpen                   InstagramConflictState = "open"
	ConflictResolvedKeepExisting   InstagramConflictState = "resolvedKeepExisting"
	ConflictResolvedRevokeExisting InstagramConflictState = "resolvedRevokeExisting"
	ConflictExpired                InstagramConflictState = "expired"
)

func (s InstagramConflictState) Valid() bool {
	switch s {
	case ConflictOpen, ConflictResolvedKeepExisting, ConflictResolvedRevokeExisting, ConflictExpired:
		return true
	default:
		return false
	}
}

func (s InstagramConflictState) Terminal() bool {
	return s == ConflictResolvedKeepExisting || s == ConflictResolvedRevokeExisting || s == ConflictExpired
}

func ValidateInstagramConflictTransition(from, to InstagramConflictState) error {
	if !from.Valid() || !to.Valid() {
		return ErrInvalidInstagramState
	}
	if from == to {
		return nil
	}
	allowed := from == ConflictOpen && oneOf(to, ConflictResolvedKeepExisting, ConflictResolvedRevokeExisting, ConflictExpired)
	return transitionResult(allowed, from, to)
}

func RequireAggregateOwner(owner, authenticated syntax.DID) error {
	if owner != authenticated {
		return ErrInstagramResourceNotFound
	}
	return nil
}

func transitionResult[S ~string](allowed bool, from, to S) error {
	if allowed {
		return nil
	}
	return fmt.Errorf("%w: %s -> %s", ErrInstagramStateTransition, from, to)
}

func oneOf[S comparable](value S, allowed ...S) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}
