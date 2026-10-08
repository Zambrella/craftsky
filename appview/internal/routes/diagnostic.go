package routes

import (
	_ "embed"
	"encoding/json"
	"github.com/ipfs/go-cid"
	"net/url"
	"strings"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/observability"
)

// Explicit diagnostic metadata is independent of authorization/access class.
// A new or private route has no permission to expose wildcard targets.
//
//go:embed diagnostic_manifest.json
var diagnosticManifestJSON []byte

type DiagnosticRouteMetadata struct {
	Method   string `json:"method"`
	Path     string `json:"path"`
	Category string `json:"category"`
	Public   bool   `json:"public"`
}

var diagnosticRoutes = loadDiagnosticRoutes()

func loadDiagnosticRoutes() map[string]DiagnosticRouteMetadata {
	var entries []DiagnosticRouteMetadata
	if err := json.Unmarshal(diagnosticManifestJSON, &entries); err != nil {
		panic("invalid diagnostic route manifest")
	}
	result := make(map[string]DiagnosticRouteMetadata, len(entries))
	for _, item := range entries {
		result[policyKey(item.Method, item.Path)] = item
	}
	return result
}

func (c *V1Catalogue) Resolve(method string, value *url.URL) observability.RequestDiagnosticContext {
	result := observability.RequestDiagnosticContext{Path: "/[REDACTED]", RoutePattern: "unmatched"}
	if value == nil {
		return result
	}
	switch value.Path {
	case "/health", "/healthz", "/oauth/client-metadata.json", "/oauth/jwks.json", "/oauth/callback":
		// Non-v1 metadata/probes have no dynamic segments. Queries are never copied.
		result.Path = value.Path
		result.RoutePattern = value.Path
		return result
	}
	result.Path = c.conservativeDiagnosticPath(value.Path)
	canonical, err := canonicalV1Path(value)
	if err != nil {
		return result
	}
	match, ok := c.Match(canonical)
	if !ok {
		return result
	}
	result.RoutePattern = match.PathPattern
	public := diagnosticRoutes[policyKey(method, match.PathPattern)].Public
	if _, knownMethod := match.Policies[method]; !knownMethod {
		public = false
	}
	parts := strings.Split(strings.TrimPrefix(canonical, "/"), "/")
	pattern := strings.Split(strings.TrimPrefix(match.PathPattern, "/"), "/")
	allPublicSegmentsValid := public
	for i, segment := range pattern {
		if !strings.HasPrefix(segment, "{") {
			continue
		}
		if !public || !safePublicDiagnosticSegment(segment, parts[i]) {
			parts[i] = "[REDACTED]"
			allPublicSegmentsValid = false
		}
	}
	result.Path = observability.BoundDiagnosticPath("/" + strings.Join(parts, "/"))
	result.PublicTargets = allPublicSegmentsValid
	if allPublicSegmentsValid {
		for i, segment := range pattern {
			switch segment {
			case "{did}":
				result.PublicRecord.TargetDID = syntax.DID(parts[i])
			case "{rkey}":
				result.PublicRecord.RecordKey = syntax.RecordKey(parts[i])
			case "{captionCid}":
				result.PublicRecord.CID = syntax.CID(parts[i])
			case "{handleOrDid}":
				identifier := strings.TrimPrefix(parts[i], "@")
				if strings.HasPrefix(identifier, "did:") {
					result.PublicRecord.TargetDID = syntax.DID(identifier)
				} else if identifier != "me" {
					result.PublicRecord.Handle = syntax.Handle(identifier)
				}
			}
		}
		if strings.HasPrefix(match.PathPattern, "/v1/posts/") {
			result.PublicRecord.NSID = "social.craftsky.feed.post"
		}
		if strings.HasPrefix(match.PathPattern, "/v1/events/") {
			result.PublicRecord.NSID = "social.craftsky.business.event"
		}
		if result.PublicRecord.TargetDID != "" && result.PublicRecord.RecordKey != "" && result.PublicRecord.NSID != "" {
			result.PublicRecord.URI = syntax.ATURI("at://" + result.PublicRecord.TargetDID.String() + "/" + result.PublicRecord.NSID.String() + "/" + result.PublicRecord.RecordKey.String())
		}
	}
	return result
}

func safePublicDiagnosticSegment(pattern, value string) bool {
	if len(value) > observability.MaxDiagnosticTextBytes {
		return false
	}
	switch pattern {
	case "{did}":
		_, err := syntax.ParseDID(value)
		return err == nil
	case "{rkey}":
		_, err := syntax.ParseRecordKey(value)
		return err == nil
	case "{captionCid}":
		_, err := cid.Decode(value)
		return err == nil
	case "{handleOrDid}":
		value = strings.TrimPrefix(value, "@")
		if value == "me" {
			return true
		}
		_, err := syntax.ParseAtIdentifier(value)
		return err == nil
	default:
		return false
	}
}

func (c *V1Catalogue) conservativeDiagnosticPath(path string) string {
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	prefix := 0
	// Retain only a registered literal prefix; after divergence or a wildcard,
	// every unclassified segment is marked, even if it resembles a public DID.
	for _, route := range c.routes {
		matched := 0
		for i, segment := range route.segments {
			if i >= len(parts) || segment.wildcard || parts[i] != segment.literal {
				break
			}
			matched++
		}
		prefix = max(prefix, matched)
	}
	for i := range parts {
		if i >= prefix {
			parts[i] = "[REDACTED]"
		}
	}
	return observability.BoundDiagnosticPath("/" + strings.Join(parts, "/"))
}
