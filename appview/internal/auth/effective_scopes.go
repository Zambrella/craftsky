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

// HasEffectiveScopes uses the same semantic evaluator as login, refresh and
// the fenced optional-operation boundary. It does not authorize a session.
func HasEffectiveScopes(granted, required []string) bool {
	return validRegistrationTokenScopes(strings.Join(granted, " "), required)
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
	if permission, ok := parseRPCScope(required); ok {
		for _, method := range permission.methods {
			covered := false
			for _, raw := range granted {
				candidate, valid := parseRPCScope(raw)
				if valid && candidate.audience == permission.audience {
					for _, allowed := range candidate.methods {
						covered = covered || allowed == method
					}
				}
			}
			if !covered {
				return false
			}
		}
		return true
	}
	return false
}

type rpcScopePermission struct {
	methods  []syntax.NSID
	audience string
}

// Parse only explicit method grants; wildcard RPC scopes cannot establish a
// narrow chat capability. Audience equality includes the service fragment.
// https://atproto.com/specs/permission defines repeated lxm parameters and
// forbids specifying the positional parameter twice.
func parseRPCScope(raw string) (rpcScopePermission, bool) {
	resource, query, _ := strings.Cut(raw, "?")
	name, positional, hasPositional := strings.Cut(resource, ":")
	if name != "rpc" {
		return rpcScopePermission{}, false
	}
	parameters, err := url.ParseQuery(query)
	if err != nil {
		return rpcScopePermission{}, false
	}
	for key := range parameters {
		if key != "lxm" && key != "aud" {
			return rpcScopePermission{}, false
		}
	}
	audiences := parameters["aud"]
	if len(audiences) != 1 || audiences[0] == "" {
		return rpcScopePermission{}, false
	}
	if audiences[0] != "*" {
		did, fragment, ok := strings.Cut(audiences[0], "#")
		if _, err := syntax.ParseDID(did); err != nil || !ok || fragment == "" || strings.Contains(fragment, "#") {
			return rpcScopePermission{}, false
		}
	}
	methods := parameters["lxm"]
	if hasPositional {
		if _, present := parameters["lxm"]; present {
			return rpcScopePermission{}, false
		}
		decoded, err := url.PathUnescape(positional)
		if err != nil {
			return rpcScopePermission{}, false
		}
		methods = []string{decoded}
	}
	if len(methods) == 0 {
		return rpcScopePermission{}, false
	}
	permission := rpcScopePermission{audience: audiences[0]}
	for _, rawMethod := range methods {
		method, err := syntax.ParseNSID(rawMethod)
		if err != nil {
			return rpcScopePermission{}, false
		}
		permission.methods = append(permission.methods, method)
	}
	return permission, true
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
