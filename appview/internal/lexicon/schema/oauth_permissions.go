package schema

import (
	"fmt"
	"slices"

	indigolexicon "github.com/bluesky-social/indigo/atproto/lexicon"
)

// CraftskyOAuthRepoPermissions reads the exact embedded publication source, so
// token validation tracks permission-set evolution without a second allowlist.
func CraftskyOAuthRepoPermissions() ([]indigolexicon.SchemaPermission, error) {
	definition, err := craftskyCatalog.Resolve("social.craftsky.authFull")
	if err != nil {
		return nil, err
	}
	set, ok := definition.Def.(indigolexicon.SchemaPermissionSet)
	if !ok {
		return nil, fmt.Errorf("CraftSky OAuth permission set has an invalid type")
	}
	permissions := make([]indigolexicon.SchemaPermission, len(set.Permissions))
	for i, permission := range set.Permissions {
		// Current CraftSky set intentionally grants only its own public collections.
		if permission.Resource != "repo" || len(permission.Collection) == 0 {
			return nil, fmt.Errorf("CraftSky OAuth permission set has an unsupported resource")
		}
		permissions[i] = permission
		permissions[i].Collection = slices.Clone(permission.Collection)
		permissions[i].Action = slices.Clone(permission.Action)
	}
	return permissions, nil
}
