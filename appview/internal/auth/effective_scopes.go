package auth

import (
	"net/url"
	"strings"

	"github.com/bluesky-social/indigo/atproto/syntax"
	lexiconschema "social.craftsky/appview/internal/lexicon/schema"
)

type repoScopePermission struct {
	collections []syntax.NSID
	actions     []string
}

func requiredScopeGranted(required string, granted []string, exact map[string]struct{}) bool {
	if _, ok := exact[required]; ok {
		return true
	}
	if required == "include:social.craftsky.authFull" {
		permissions, err := lexiconschema.CraftskyOAuthRepoPermissions()
		if err != nil || len(permissions) == 0 {
			return false
		}
		for _, permission := range permissions {
			actions := permission.Action
			if len(actions) == 0 {
				actions = []string{"create", "update", "delete"}
			}
			for _, raw := range permission.Collection {
				collection, err := syntax.ParseNSID(raw)
				if err != nil || !strings.HasPrefix(collection.String(), "social.craftsky.") {
					return false
				}
				if !repoAuthorityGranted(collection, actions, granted) {
					return false
				}
			}
		}
		return true
	}
	// Providers may combine collections or split actions for direct repo grants.
	if permission, ok := parseRepoScope(required); ok {
		for _, collection := range permission.collections {
			if !repoAuthorityGranted(collection, permission.actions, granted) {
				return false
			}
		}
		return true
	}
	return false
}

func repoAuthorityGranted(collection syntax.NSID, actions []string, granted []string) bool {
	covered := make(map[string]bool, 3)
	for _, raw := range granted {
		permission, ok := parseRepoScope(raw)
		if !ok {
			continue
		}
		for _, candidate := range permission.collections {
			if candidate == collection {
				for _, action := range permission.actions {
					covered[action] = true
				}
				break
			}
		}
	}
	for _, action := range actions {
		if !covered[action] {
			return false
		}
	}
	return true
}

// Interpret only the defined repository permission syntax. Unknown parameters,
// wildcards, invalid NSIDs and partial actions never establish full authority.
func parseRepoScope(raw string) (repoScopePermission, bool) {
	resource, query, _ := strings.Cut(raw, "?")
	name, positional, hasPositional := strings.Cut(resource, ":")
	if name != "repo" {
		return repoScopePermission{}, false
	}
	parameters, err := url.ParseQuery(query)
	if err != nil {
		return repoScopePermission{}, false
	}
	for key := range parameters {
		if key != "collection" && key != "action" {
			return repoScopePermission{}, false
		}
	}
	collections := parameters["collection"]
	if hasPositional {
		if len(collections) > 0 {
			return repoScopePermission{}, false
		}
		decoded, err := url.PathUnescape(positional)
		if err != nil {
			return repoScopePermission{}, false
		}
		collections = []string{decoded}
	}
	if len(collections) == 0 {
		return repoScopePermission{}, false
	}
	permission := repoScopePermission{}
	for _, rawCollection := range collections {
		collection, err := syntax.ParseNSID(rawCollection)
		if err != nil {
			return repoScopePermission{}, false
		}
		permission.collections = append(permission.collections, collection)
	}
	actions, present := parameters["action"]
	if !present {
		actions = []string{"create", "update", "delete"}
	}
	seen := make(map[string]bool, 3)
	for _, action := range actions {
		if (action != "create" && action != "update" && action != "delete") || seen[action] {
			return repoScopePermission{}, false
		}
		seen[action] = true
	}
	if len(actions) == 0 {
		return repoScopePermission{}, false
	}
	permission.actions = actions
	return permission, true
}
