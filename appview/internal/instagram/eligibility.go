package instagram

type EligibilityStage string

const (
	EligibilityAtMatch                EligibilityStage = "match"
	EligibilityAtPersist              EligibilityStage = "persist"
	EligibilityAtList                 EligibilityStage = "list"
	EligibilityAtNotificationCreate   EligibilityStage = "notificationCreate"
	EligibilityAtNotificationDelivery EligibilityStage = "notificationDelivery"
	EligibilityAtFeed                 EligibilityStage = "feed"
	EligibilityAtOpen                 EligibilityStage = "open"
	EligibilityAtAccept               EligibilityStage = "accept"
)

var eligibilityStages = []EligibilityStage{
	EligibilityAtMatch,
	EligibilityAtPersist,
	EligibilityAtList,
	EligibilityAtNotificationCreate,
	EligibilityAtNotificationDelivery,
	EligibilityAtFeed,
	EligibilityAtOpen,
	EligibilityAtAccept,
}

func (s EligibilityStage) Valid() bool {
	for _, stage := range eligibilityStages {
		if s == stage {
			return true
		}
	}
	return false
}

func AllEligibilityStages() []EligibilityStage {
	return append([]EligibilityStage(nil), eligibilityStages...)
}

type EligibilityReason string

const (
	EligibilityAllowed            EligibilityReason = "allowed"
	EligibilityInvalidInput       EligibilityReason = "invalidInput"
	EligibilityMembership         EligibilityReason = "membership"
	EligibilityLink               EligibilityReason = "link"
	EligibilityDiscovery          EligibilityReason = "discovery"
	EligibilityConflict           EligibilityReason = "conflict"
	EligibilityDirection          EligibilityReason = "direction"
	EligibilityUsername           EligibilityReason = "username"
	EligibilitySelf               EligibilityReason = "self"
	EligibilityAlreadyFollowing   EligibilityReason = "alreadyFollowing"
	EligibilityModeration         EligibilityReason = "moderation"
	EligibilityRelationshipSafety EligibilityReason = "relationshipSafety"
	EligibilitySafetyUnavailable  EligibilityReason = "safetyUnavailable"
)

type EligibilitySnapshot struct {
	ImporterCurrentMember bool
	TargetCurrentMember   bool
	LinkState             InstagramLinkState
	DMVerified            bool
	Discoverable          bool
	ConflictFree          bool
	ImportDirection       ImportDirection
	ImportedUsername      string
	CurrentUsername       string
	Self                  bool
	AlreadyFollowing      bool
	TargetHidden          bool
	TargetTakenDown       bool
	ImporterBlocksTarget  bool
	TargetBlocksImporter  bool
	ImporterMutesTarget   bool
	SafetyDataAvailable   bool
}

func (EligibilitySnapshot) String() string {
	return "Instagram suggestion eligibility snapshot [REDACTED]"
}

type EligibilityDecision struct {
	Eligible bool
	Reason   EligibilityReason
}

func EvaluateInstagramSuggestionEligibility(stage EligibilityStage, snapshot EligibilitySnapshot) EligibilityDecision {
	deny := func(reason EligibilityReason) EligibilityDecision {
		return EligibilityDecision{Reason: reason}
	}
	if !stage.Valid() {
		return deny(EligibilityInvalidInput)
	}
	if !snapshot.ImporterCurrentMember || !snapshot.TargetCurrentMember {
		return deny(EligibilityMembership)
	}
	if snapshot.LinkState == LinkDisputed || !snapshot.ConflictFree {
		return deny(EligibilityConflict)
	}
	if snapshot.LinkState != LinkActive || !snapshot.DMVerified {
		return deny(EligibilityLink)
	}
	if !snapshot.Discoverable {
		return deny(EligibilityDiscovery)
	}
	if snapshot.ImportDirection != DirectionFollowing {
		return deny(EligibilityDirection)
	}
	imported, importedErr := NormalizeInstagramUsername(snapshot.ImportedUsername)
	current, currentErr := NormalizeInstagramUsername(snapshot.CurrentUsername)
	if importedErr != nil || currentErr != nil || imported != current {
		return deny(EligibilityUsername)
	}
	if snapshot.Self {
		return deny(EligibilitySelf)
	}
	if snapshot.AlreadyFollowing {
		return deny(EligibilityAlreadyFollowing)
	}
	if snapshot.TargetHidden || snapshot.TargetTakenDown {
		return deny(EligibilityModeration)
	}
	if !snapshot.SafetyDataAvailable {
		return deny(EligibilitySafetyUnavailable)
	}
	if snapshot.ImporterBlocksTarget || snapshot.TargetBlocksImporter || snapshot.ImporterMutesTarget {
		return deny(EligibilityRelationshipSafety)
	}
	return EligibilityDecision{Eligible: true, Reason: EligibilityAllowed}
}
