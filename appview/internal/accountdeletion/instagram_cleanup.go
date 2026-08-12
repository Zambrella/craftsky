package accountdeletion

import (
	"context"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// InstagramExplicitPurger deliberately excludes ordinary membership
// inactivation. The existing instagram.PrivateDataService satisfies it with
// its terminal, AppView-only PurgeOwner operation.
type InstagramExplicitPurger interface {
	PurgeOwner(ctx context.Context, owner syntax.DID) error
}

func PurgeInstagramForAccountDeletion(ctx context.Context, purger InstagramExplicitPurger, owner syntax.DID) error {
	return purger.PurgeOwner(ctx, owner)
}
