package retention

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrInvalidPolicy = errors.New("invalid retention policy")

type DataClass string

const DataClassRestrictedEvidence DataClass = "restrictedEvidence"

type HoldCoverage struct {
	EvidenceID uuid.UUID
	ExpiresAt  time.Time
}

type Decision struct {
	DeleteAt time.Time
	Due      bool
	Held     bool
}

type Policy struct{ Periods map[DataClass]time.Duration }

func (policy Policy) Decide(class DataClass, itemID uuid.UUID, createdAt, now time.Time, holds []HoldCoverage) (Decision, error) {
	period, ok := policy.Periods[class]
	if !ok || period <= 0 || itemID == uuid.Nil || createdAt.IsZero() || now.IsZero() {
		return Decision{}, ErrInvalidPolicy
	}
	deleteAt := createdAt.Add(period)
	for _, hold := range holds {
		if hold.EvidenceID == itemID && hold.ExpiresAt.After(now) && hold.ExpiresAt.After(deleteAt) {
			deleteAt = hold.ExpiresAt
		}
	}
	return Decision{DeleteAt: deleteAt, Due: !now.Before(deleteAt), Held: deleteAt.After(createdAt.Add(period)) && now.Before(deleteAt)}, nil
}
