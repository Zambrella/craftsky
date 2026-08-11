package accountdeletion

import (
	"context"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

type CleanupPolicy string

const HardDelete CleanupPolicy = "hardDelete"

type InstagramDataCategory struct {
	Name   string
	Policy CleanupPolicy
}

var (
	InstagramAccountLinks               = InstagramDataCategory{Name: "accountLinks", Policy: HardDelete}
	InstagramGraphImports               = InstagramDataCategory{Name: "graphImports", Policy: HardDelete}
	InstagramAutomaticFollowSuggestions = InstagramDataCategory{Name: "automaticFollowSuggestions", Policy: HardDelete}
	InstagramVerification               = InstagramDataCategory{Name: "verification", Policy: HardDelete}
	InstagramPrivateImportedData        = InstagramDataCategory{Name: "privateImportedData", Policy: HardDelete}
	InstagramUsernameClaims             = InstagramDataCategory{Name: "usernameClaims", Policy: HardDelete}
)

func InstagramExplicitDeletionPlan() []InstagramDataCategory {
	return []InstagramDataCategory{
		InstagramAccountLinks,
		InstagramGraphImports,
		InstagramAutomaticFollowSuggestions,
		InstagramVerification,
		InstagramPrivateImportedData,
		InstagramUsernameClaims,
	}
}

// InstagramExplicitPurger deliberately excludes ordinary membership
// inactivation. The existing instagram.PrivateDataService satisfies it with
// its terminal, AppView-only PurgeOwner operation.
type InstagramExplicitPurger interface {
	PurgeOwner(ctx context.Context, owner syntax.DID) error
}

func PurgeInstagramForAccountDeletion(ctx context.Context, purger InstagramExplicitPurger, owner syntax.DID) error {
	return purger.PurgeOwner(ctx, owner)
}
