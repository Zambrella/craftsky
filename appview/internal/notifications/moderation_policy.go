package notifications

import (
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
)

type ModerationEventKind string

const (
	ModerationReportAccepted  ModerationEventKind = "reportAccepted"
	ModerationDecision        ModerationEventKind = "decision"
	ModerationEffectsChanged  ModerationEventKind = "effectsChanged"
	ModerationAppealConfirmed ModerationEventKind = "appealConfirmed"
	ModerationAppealResolved  ModerationEventKind = "appealResolved"
	ModerationStrikeExpired   ModerationEventKind = "strikeExpired"
	ModerationSevereRestored  ModerationEventKind = "severeRestored"
)

type ModerationEvent struct {
	Kind               ModerationEventKind
	ConsequenceChanged bool
	EnforcementChanged bool
}

type ModerationIntent struct {
	RecipientDID  syntax.DID
	CaseID        uuid.UUID
	CaseReference string
	EventID       uuid.UUID
	CreatedAt     time.Time
}

func ShouldNotifyModeration(event ModerationEvent) bool {
	switch event.Kind {
	case ModerationDecision, ModerationEffectsChanged, ModerationAppealResolved, ModerationSevereRestored:
		return event.ConsequenceChanged
	case ModerationStrikeExpired:
		return event.EnforcementChanged
	default:
		return false
	}
}
