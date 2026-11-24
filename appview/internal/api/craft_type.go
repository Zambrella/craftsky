package api

import (
	"strings"

	"social.craftsky/appview/internal/projectcatalog"
)

const craftTypePrefix = "social.craftsky.feed.defs#"

var defaultSupportedCraftTypes, supportedCraftTypeAliases = func() ([]string, map[string]string) {
	crafts := projectcatalog.PrimaryCrafts()
	defaults := make([]string, 0, len(crafts))
	aliases := make(map[string]string, 3*len(crafts))
	for _, craft := range crafts {
		defaults = append(defaults, craft.ID)
		aliases[strings.ToLower(craft.ID)] = craft.ID
		aliases[strings.ToLower(strings.TrimPrefix(craft.ID, craftTypePrefix))] = craft.ID
		aliases[craft.ProfileID] = craft.ID
	}
	return defaults, aliases
}()

func CanonicalCraftType(raw string) (string, error) {
	key := strings.ToLower(strings.TrimSpace(raw))
	if key == "" {
		return "", ErrSearchValidation
	}
	canonical, ok := supportedCraftTypeAliases[key]
	if !ok {
		return "", ErrSearchValidation
	}
	return canonical, nil
}

func CanonicalCraftTypes(raw []string, useDefaults bool) ([]string, error) {
	if len(raw) == 0 && useDefaults {
		return append([]string(nil), defaultSupportedCraftTypes...), nil
	}
	out := make([]string, 0, len(raw))
	seen := map[string]bool{}
	for _, value := range raw {
		canonical, err := CanonicalCraftType(value)
		if err != nil {
			return nil, err
		}
		if !seen[canonical] {
			out = append(out, canonical)
			seen[canonical] = true
		}
	}
	return out, nil
}
