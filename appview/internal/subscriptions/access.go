package subscriptions

import "github.com/bluesky-social/indigo/atproto/syntax"

func ProjectSelfAccess(did syntax.DID, assignment *LocalAssignment) SelfAccess {
	access := SelfAccess{DID: did, EffectiveTier: TierFree}
	if assignment == nil {
		return access
	}

	access.AssignedTier = &assignment.Tier
	access.GivesAccess = assignment.GivesAccess
	access.AccessEndsAt = assignment.AccessEndsAt
	if assignment.GivesAccess {
		access.EffectiveTier = assignment.Tier
	}
	return access
}
