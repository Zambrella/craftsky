package pdscommands

import (
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

type CommandState string

const (
	CommandPrepared     CommandState = "prepared"
	CommandDispatching  CommandState = "dispatching"
	CommandAccepted     CommandState = "accepted"
	CommandAmbiguous    CommandState = "ambiguous"
	CommandRejected     CommandState = "rejected"
	DispatchInvalidSwap CommandState = "invalid_swap"
	ReplayRetention                  = 24 * time.Hour
)

type Tombstone struct {
	OwnerDID      syntax.DID
	OperationKind string
	ScopedKeyHash [32]byte
	CompactedAt   time.Time
}

func NewTombstone(owner syntax.DID, operationKind string, scopedKeyHash [32]byte, compactedAt time.Time) Tombstone {
	return Tombstone{
		OwnerDID:      owner,
		OperationKind: operationKind,
		ScopedKeyHash: scopedKeyHash,
		CompactedAt:   compactedAt,
	}
}

func ReplayExpiresAt(terminalAt time.Time) time.Time {
	return terminalAt.Add(ReplayRetention)
}

func ShouldCompact(state CommandState, replayExpiresAt, now time.Time) bool {
	if replayExpiresAt.IsZero() || now.Before(replayExpiresAt) {
		return false
	}
	return state == CommandAccepted || state == CommandRejected
}
