---
{
  "id": "2026-10-14-large-screen-discovery-sidebar",
  "title": "Large-screen craft discovery and member suggestions",
  "source": "docs/changes/2026-10-14-large-screen-discovery-sidebar",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app",
    "appview"
  ],
  "topics": [
    "discovery",
    "navigation",
    "responsive-layout"
  ],
  "keywords": [
    "Your crafts",
    "New to CraftSky",
    "accepted block overlay",
    "member selection",
    "sidebar",
    "visibility before limit"
  ],
  "summary": "The recorded change adds preferred-craft links and a bounded member-discovery section to the large-screen Flutter shell.",
  "currentDocs": [
    "app/README.md",
    "docs/architecture.md"
  ],
  "code": [
    "app/lib/router/widgets/desktop_discovery_sidebar.dart",
    "appview/internal/api"
  ],
  "evidence": [
    "docs/changes/2026-10-14-large-screen-discovery-sidebar/05-implementation-plan.md#ir-001-correction-pass--2026-10-15",
    "docs/changes/2026-10-14-large-screen-discovery-sidebar/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": [
    "Historical physical-device/VoiceOver checks and bounded-overlay limitations remain in the execution record."
  ]
}
---

# Large-screen craft discovery and member suggestions

## Outcome and decisions

The recorded change adds preferred-craft links and a bounded member-discovery section to the large-screen Flutter shell. AppView applies membership/relationship eligibility before limiting candidates; client sections have independent loading/error state and account isolation. A correction suppresses accepted blocked members during the existing local overlay window while waiting for a projected refill. Later styling records a bordered member section and aligned plain craft links.

## Evidence and limits

The opening review’s stale-member finding is addressed by the later IR-001 correction tests, followed by local regression/analysis checks and a full backend gate. The overlay is bounded, so the record does not promise indefinite suppression during arbitrary ingestion outages. Device/VoiceOver and legacy-data evidence remain separate.

Historical evidence: [05-implementation-plan.md — IR-001 correction pass — 2026-10-15](../../../docs/changes/2026-10-14-large-screen-discovery-sidebar/05-implementation-plan.md#ir-001-correction-pass--2026-10-15); [06-implementation-review.md](../../../docs/changes/2026-10-14-large-screen-discovery-sidebar/06-implementation-review.md).

## Current entry points

[app/README.md](../../../app/README.md); [docs/architecture.md](../../../docs/architecture.md). Recorded follow-ups are historical evidence boundaries; recheck them against later records and current code before treating them as open defects.
