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
	ScheduleID  uuid.UUID
	Owner       syntax.DID
	OperationID uuid.UUID
	RequestHash [32]byte
	URI         syntax.ATURI
	CID         syntax.CID
	PublishedAt time.Time

	PrivatePayload         []byte   `json:"-"`
	PrivateMediaObjectKeys []string `json:"-"`
}

type PublicationTombstone struct {
	ScheduleID  uuid.UUID    `json:"scheduleId"`
	Owner       syntax.DID   `json:"owner"`
	OperationID uuid.UUID    `json:"operationId"`
	RequestHash [32]byte     `json:"requestHash"`
	URI         syntax.ATURI `json:"uri"`
	CID         syntax.CID   `json:"cid"`
	PublishedAt time.Time    `json:"publishedAt"`
	ExpiresAt   time.Time    `json:"expiresAt"`
}

func NewPublicationTombstone(completed CompletedPublication) (PublicationTombstone, error) {
	if completed.ScheduleID == uuid.Nil || completed.Owner == "" ||
		completed.OperationID == uuid.Nil || completed.URI == "" || completed.CID == "" ||
		completed.PublishedAt.IsZero() {
		return PublicationTombstone{}, ErrInvalidCompletedPublication
	}
	publishedAt := completed.PublishedAt.UTC()
	return PublicationTombstone{
		ScheduleID:  completed.ScheduleID,
		Owner:       completed.Owner,
		OperationID: completed.OperationID,
		RequestHash: completed.RequestHash,
		URI:         completed.URI,
		CID:         completed.CID,
		PublishedAt: publishedAt,
		ExpiresAt:   PublicationTombstoneExpiresAt(publishedAt),
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
