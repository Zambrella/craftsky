---
{
  "id": "2026-10-27-profile-links",
  "title": "Shared Bluesky profile links",
  "source": "docs/changes/2026-10-27-profile-links.md",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app",
    "appview"
  ],
  "topics": [
    "profiles",
    "links",
    "federation"
  ],
  "keywords": [
    "app.bsky.actor.link",
    "strong references",
    "compound profile write",
    "favicon",
    "ten links",
    "ADR 022"
  ],
  "summary": "The recorded implementation adopts shared Bluesky profile-link records and ordered profile strong references instead of defining a competing CraftSky link schema.",
  "currentDocs": [
    "docs/adr/022-shared-bluesky-profile-links.md",
    "docs/architecture.md"
  ],
  "code": [
    "app/lib/profile/widgets/profile_links.dart",
    "app/lib/profile/widgets/profile_links_editor.dart",
    "appview/internal/pdscommands"
  ],
  "evidence": [
    "docs/changes/2026-10-27-profile-links.md#local-validation"
  ],
  "supersededBy": [],
  "followUps": [
    "Disposable-account Bluesky interoperability and physical link-opening/reordering checks remain unverified by host suites."
  ]
}
---

# Shared Bluesky profile links

## Outcome and decisions

The recorded implementation adopts shared Bluesky profile-link records and ordered profile strong references instead of defining a competing CraftSky link schema. The editor supports bounded links/titles, URL policy, duplicate detection, local reorder/edit/removal and advisory favicons. Owner-authorized deletion is narrow: successfully remove the current profile reference before or atomically with deleting that owner’s link record; no reusable generic PDS deletion capability is introduced.

## Evidence and limits

The standalone record reports complete backend release gates, focused race/static checks, 371 Flutter tests and operator-console tests. Live Bluesky interoperability, real favicon scanning and physical browser/drag/screen-reader interaction remain distinct checks. No production account mutation or infrastructure deployment is claimed.

Historical evidence: [2026-10-27-profile-links.md — Local validation](../../../docs/changes/2026-10-27-profile-links.md#local-validation).

## Current entry points

[docs/adr/022-shared-bluesky-profile-links.md](../../../docs/adr/022-shared-bluesky-profile-links.md); [docs/architecture.md](../../../docs/architecture.md). Recorded follow-ups are historical evidence boundaries; recheck them against later records and current code before treating them as open defects.
