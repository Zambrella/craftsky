package api

import "strings"

const craftTypePrefix = "social.craftsky.feed.defs#"

var defaultSupportedCraftTypes = []string{
	craftTypePrefix + "knitting",
	craftTypePrefix + "crochet",
	craftTypePrefix + "sewing",
	craftTypePrefix + "embroidery",
	craftTypePrefix + "quilting",
}

var supportedCraftTypeAliases = map[string]string{
	"knitting":                     craftTypePrefix + "knitting",
	craftTypePrefix + "knitting":   craftTypePrefix + "knitting",
	"crochet":                      craftTypePrefix + "crochet",
	craftTypePrefix + "crochet":    craftTypePrefix + "crochet",
	"sewing":                       craftTypePrefix + "sewing",
	craftTypePrefix + "sewing":     craftTypePrefix + "sewing",
	"embroidery":                   craftTypePrefix + "embroidery",
	craftTypePrefix + "embroidery": craftTypePrefix + "embroidery",
	"quilting":                     craftTypePrefix + "quilting",
	craftTypePrefix + "quilting":   craftTypePrefix + "quilting",
}

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
