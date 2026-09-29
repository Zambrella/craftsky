package subscriptions

import (
	"crypto/sha256"
	"encoding/binary"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

func accessFenceKey(did syntax.DID) int64 {
	digest := sha256.Sum256([]byte("social.craftsky.subscription-access-fence.v1\x00" + did.String()))
	return int64(binary.BigEndian.Uint64(digest[:8]))
}

// AccessFenceKey identifies the per-DID subscription effect lock domain.
func AccessFenceKey(did syntax.DID) int64 { return accessFenceKey(did) }

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

// AllowsPlus reports whether this DID currently holds a Plus capability.
func (access SelfAccess) AllowsPlus() bool {
	return access.GivesAccess && (access.EffectiveTier == TierPlus || access.EffectiveTier == TierBusiness)
}

// AllowsBusiness reports whether this DID currently holds a Business capability.
func (access SelfAccess) AllowsBusiness() bool {
	return access.GivesAccess && access.EffectiveTier == TierBusiness
}
