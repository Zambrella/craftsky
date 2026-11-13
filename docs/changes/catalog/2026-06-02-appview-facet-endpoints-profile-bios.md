---
{
  "id": "2026-06-02-appview-facet-endpoints-profile-bios",
  "title": "Facet autocomplete and plain profile bios",
  "source": "docs/changes/2026-06-02-appview-facet-endpoints-profile-bios",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app",
    "appview"
  ],
  "topics": [
    "facets",
    "identity",
    "profiles"
  ],
  "keywords": [
    "IdentityCacheStore",
    "handle_lower",
    "ResolveMention",
    "descriptionFacets",
    "hashtag suggestions",
    "identity-cache backfill"
  ],
  "summary": "The package records AppView-backed mention/hashtag autocomplete, resolver-backed identity caching and render-time parsing of plain profile bios.",
  "currentDocs": [
    "docs/architecture.md",
    "app/README.md",
    "appview/README.md",
    "docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md"
  ],
  "code": [
    "appview/internal/api/facet_store.go",
    "appview/internal/api/identity_cache_store.go",
    "app/lib/shared/rich_text"
  ],
  "evidence": [
    "docs/changes/2026-06-02-appview-facet-endpoints-profile-bios/05-implementation-plan.md#verification-log",
    "docs/changes/2026-06-02-appview-facet-endpoints-profile-bios/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": [
    "Historical MAN-001 through MAN-004 and skipped DB portions need separate evidence if relevant to a new release."
  ]
}
---

# Facet autocomplete and plain profile bios

## Outcome and decisions

The package records AppView-backed mention/hashtag autocomplete, resolver-backed identity caching and render-time parsing of plain profile bios. Profile descriptionFacets are removed from the wire/client model. Cache ownership follows resolver-confirmed DID/handle identity. Later correction steps atomically displace a reassigned handle and escape wildcard input in mention/hashtag lookup; these execution notes supersede the stale review findings.

## Evidence and limits

The final execution log records focused Flutter tests, broader Go/Flutter checks and analysis. Some identity-cache integration commands explicitly skipped real-database portions when no test database was configured. Four manual checks remained unexecuted.

Historical evidence: [05-implementation-plan.md — Verification Log](../../../docs/changes/2026-06-02-appview-facet-endpoints-profile-bios/05-implementation-plan.md#verification-log); [06-implementation-review.md](../../../docs/changes/2026-06-02-appview-facet-endpoints-profile-bios/06-implementation-review.md).

## Current entry points

[docs/architecture.md](../../../docs/architecture.md); [app/README.md](../../../app/README.md); [appview/README.md](../../../appview/README.md); [docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md](../../../docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md). Recorded follow-ups are historical evidence boundaries; recheck them against later records and current code before treating them as open defects.
