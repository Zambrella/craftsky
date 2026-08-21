package scheduledposts

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
)

var ErrInvalidCompletedPublication = errors.New("invalid completed scheduled publication")

type CompletedPublication struct {
	ScheduleID      uuid.UUID
	Owner           syntax.DID
	OwnerGeneration int64
	OperationID     uuid.UUID
	RequestHash     [32]byte
	URI             syntax.ATURI
	CID             syntax.CID
	PublishedAt     time.Time

	PrivatePayload         []byte   `json:"-"`
	PrivateMediaObjectKeys []string `json:"-"`
}

type PublicationTombstone struct {
	ScheduleID      uuid.UUID    `json:"scheduleId"`
	Owner           syntax.DID   `json:"owner"`
	OwnerGeneration int64        `json:"ownerGeneration"`
	OperationID     uuid.UUID    `json:"operationId"`
	RequestHash     [32]byte     `json:"requestHash"`
	URI             syntax.ATURI `json:"uri"`
	CID             syntax.CID   `json:"cid"`
	PublishedAt     time.Time    `json:"publishedAt"`
	ExpiresAt       time.Time    `json:"expiresAt"`
}

// ObjectCleanupSettlement is the minimum non-secret proof needed to remove an
// exact-key object cleanup tombstone. An absence observed before a tested
// server-side settlement boundary is never proof that an accepted Put cannot
// materialize later.
type ObjectCleanupSettlement struct {
	OutcomeUncertain    bool
	RemoteDeadline      time.Time
	SettlementNotBefore *time.Time
	LastAbsenceAt       *time.Time
}

func (settlement ObjectCleanupSettlement) ProvesSettlement() bool {
	if settlement.LastAbsenceAt == nil {
		return false
	}
	if !settlement.OutcomeUncertain {
		return true
	}
	if settlement.RemoteDeadline.IsZero() || settlement.SettlementNotBefore == nil ||
		!settlement.SettlementNotBefore.After(settlement.RemoteDeadline) {
		return false
	}
	return !settlement.LastAbsenceAt.Before(*settlement.SettlementNotBefore)
}

func NewPublicationTombstone(completed CompletedPublication) (PublicationTombstone, error) {
	if completed.ScheduleID == uuid.Nil || completed.Owner == "" || completed.OwnerGeneration <= 0 ||
		completed.OperationID == uuid.Nil || completed.URI == "" || completed.CID == "" ||
		completed.PublishedAt.IsZero() {
		return PublicationTombstone{}, ErrInvalidCompletedPublication
	}
	publishedAt := completed.PublishedAt.UTC()
	return PublicationTombstone{
		ScheduleID:      completed.ScheduleID,
		Owner:           completed.Owner,
		OwnerGeneration: completed.OwnerGeneration,
		OperationID:     completed.OperationID,
		RequestHash:     completed.RequestHash,
		URI:             completed.URI,
		CID:             completed.CID,
		PublishedAt:     publishedAt,
		ExpiresAt:       PublicationTombstoneExpiresAt(publishedAt),
	}, nil
}

func (PublicationTombstone) String() string {
	return "Scheduled publication tombstone [REDACTED]"
}

func (tombstone PublicationTombstone) GoString() string {
	return tombstone.String()
}

func (tombstone PublicationTombstone) Format(state fmt.State, _ rune) {
	_, _ = io.WriteString(state, tombstone.String())
}
