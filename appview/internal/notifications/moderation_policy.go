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

// VisibleActorModerationSQL filters active account hide and takedown outputs
// for a notification alias e. An actorless event still passes the predicate.
const VisibleActorModerationSQL = `
	  AND NOT EXISTS (
		SELECT 1
		FROM moderation_outputs mo
		WHERE mo.action = 'apply'
		  AND NOT appview_owner_is_terminal(mo.source_did)
		  AND mo.subject_type = 'account'
		  AND mo.subject_did = e.actor_did
		  AND mo.value IN ('hide', 'takedown')
		  AND (mo.expires_at IS NULL OR mo.expires_at > now())
		  AND NOT EXISTS (
			SELECT 1
			FROM moderation_outputs neg
			WHERE neg.action = 'negate'
			  AND neg.source_did = mo.source_did
			  AND neg.subject_type = mo.subject_type
			  AND neg.subject_did = mo.subject_did
			  AND neg.value = mo.value
			  AND (neg.expires_at IS NULL OR neg.expires_at > now())
			  AND neg.indexed_at > mo.indexed_at
		  )
	  )
`
