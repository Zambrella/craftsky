package ownerlifecycle

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"slices"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

type State string

const (
	StateActive          State = "active"
	StateDeparted        State = "departed"
	StateDeletionPending State = "deletion_pending"
	StateDeleting        State = "deleting"
	StateTerminal        State = "terminal"
)

var (
	ErrInvalidOwner       = errors.New("invalid owner DID")
	ErrInvalidTransition  = errors.New("invalid owner lifecycle transition")
	ErrGenerationChanged  = errors.New("owner lifecycle generation changed")
	ErrOwnerNotActive     = errors.New("owner is not active")
	ErrOwnerNotDeleting   = errors.New("owner is not deleting")
	ErrOwnerNotOnboarding = errors.New("owner is not in departed onboarding state")
	ErrTerminalOwner      = errors.New("owner lifecycle is terminal")
	ErrFenceReentry       = errors.New("owner fence reentry")
	ErrFenceRequired      = errors.New("active owner effect fence required")
	ErrPurgeLeaseLost     = errors.New("owner purge component lease lost")
	ErrPurgeIncomplete    = errors.New("owner terminal purge is incomplete")
	ErrAttemptConflict    = errors.New("owner effect operation conflicts with existing attempt")
	ErrAttemptState       = errors.New("invalid owner effect attempt transition")
)

func (state State) valid() bool {
	switch state {
	case StateActive, StateDeparted, StateDeletionPending, StateDeleting, StateTerminal:
		return true
	default:
		return false
	}
}

// ValidateTransition validates a state-changing lifecycle transition. Replays
// of terminal transitions are handled idempotently by Store.Terminalize and
// are intentionally not a general state transition.
func ValidateTransition(from, to State) error {
	if !from.valid() || !to.valid() || from == to || from == StateTerminal {
		return fmt.Errorf("%w: %q to %q", ErrInvalidTransition, from, to)
	}

	valid := false
	switch from {
	case StateActive:
		valid = to == StateDeparted || to == StateDeletionPending || to == StateTerminal
	case StateDeparted:
		valid = to == StateActive || to == StateTerminal
	case StateDeletionPending:
		valid = to == StateActive || to == StateDeparted || to == StateDeleting || to == StateTerminal
	case StateDeleting:
		valid = to == StateDeparted || to == StateTerminal
	}
	if !valid {
		return fmt.Errorf("%w: %q to %q", ErrInvalidTransition, from, to)
	}
	return nil
}

// transitionAdvancesAuthEpoch encodes the lifecycle changes that invalidate
// ordinary authentication without relying on a later session cleanup pass.
// Departure and terminal state always invalidate the epoch. Creating a
// deletion intent suspends ordinary access by lifecycle state alone so that
// the same device can still cancel or accept through the narrow recovery
// routes. Acceptance is the point of no return: it advances the epoch while
// the coordinated auth participant rebases only the exact deletion credential.
func transitionAdvancesAuthEpoch(from, to State) bool {
	return to == StateDeparted || to == StateTerminal ||
		(from == StateDeletionPending && to == StateDeleting)
}

// CanonicalOwners validates, sorts, and de-duplicates owner DIDs. All
// multi-owner fence acquisition goes through this function so caller-provided
// order cannot create a deadlock.
func CanonicalOwners(owners []syntax.DID) ([]syntax.DID, error) {
	if len(owners) == 0 {
		return nil, ErrInvalidOwner
	}
	canonical := slices.Clone(owners)
	for _, owner := range canonical {
		if owner == "" {
			return nil, ErrInvalidOwner
		}
	}
	slices.SortFunc(canonical, func(a, b syntax.DID) int {
		if a < b {
			return -1
		}
		if a > b {
			return 1
		}
		return 0
	})
	canonical = slices.Compact(canonical)
	return canonical, nil
}

// FenceKey returns the single stable PostgreSQL advisory-lock key for owner.
// SHA-256 collisions conservatively serialize owners; they cannot let one
// owner bypass another lock.
func FenceKey(owner syntax.DID) (int64, error) {
	if owner == "" {
		return 0, ErrInvalidOwner
	}
	digest := sha256.Sum256([]byte("social.craftsky.owner-effect-fence.v1\x00" + owner.String()))
	return int64(binary.BigEndian.Uint64(digest[:8])), nil
}

// ParentSessionFenceKey returns the domain-separated advisory-lock key for one
// OAuth parent. A hash collision only adds conservative serialization; it can
// never merge session authority or bypass the owner fence.
func ParentSessionFenceKey(owner syntax.DID, sessionID string) (int64, error) {
	if owner == "" || sessionID == "" {
		return 0, ErrInvalidOwner
	}
	digest := sha256.Sum256([]byte(
		"social.craftsky.oauth-parent-session-fence.v1\x00" + owner.String() + "\x00" + sessionID,
	))
	return int64(binary.BigEndian.Uint64(digest[:8])), nil
}
