package relationships

// ModerationState represents the stricter platform-wide decision already made
// for content or an actor before personal relationship policy is applied.
type ModerationState uint8

const (
	ModerationVisible ModerationState = iota
	ModerationHidden
	ModerationTakenDown
)

// Surface identifies the delivery or rendering context for a relationship
// decision. It is intentionally closed so policy additions are auditable.
type Surface uint8

const (
	SurfaceProfile Surface = iota
	SurfaceDirectPost
	SurfaceTopLevel
	SurfaceThread
	SurfaceQuote
	SurfaceGraph
	SurfaceNotification
	SurfacePush
	SurfaceSearchExact
)

// Decision is the data-minimizing outcome a caller must apply.
type Decision uint8

const (
	DecisionAllow Decision = iota
	DecisionOmit
	DecisionMutedPlaceholder
	DecisionBlockedPlaceholder
	DecisionMinimalProfile
)

// Decide applies platform moderation first, then symmetric blocks, then the
// viewer's private one-way mute.
func Decide(moderation ModerationState, relationship State, surface Surface) Decision {
	if moderation != ModerationVisible {
		return DecisionOmit
	}

	if relationship.HasBlock() {
		switch surface {
		case SurfaceProfile, SurfaceSearchExact:
			return DecisionMinimalProfile
		case SurfaceDirectPost, SurfaceThread, SurfaceQuote:
			return DecisionBlockedPlaceholder
		default:
			return DecisionOmit
		}
	}

	if relationship.Muted {
		switch surface {
		case SurfaceTopLevel, SurfaceNotification, SurfacePush:
			return DecisionOmit
		case SurfaceThread, SurfaceQuote:
			return DecisionMutedPlaceholder
		}
	}

	return DecisionAllow
}
