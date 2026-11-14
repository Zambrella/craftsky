---
{
  "id": "2026-08-29-flutter-business-profiles",
  "title": "Flutter business profiles and event/product editors",
  "source": "docs/changes/2026-08-29-flutter-business-profiles",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app",
    "appview"
  ],
  "topics": [
    "business",
    "profiles",
    "concurrency"
  ],
  "keywords": [
    "BusinessEvent",
    "business account type",
    "record generation",
    "TabController",
    "Products",
    "expected CID"
  ],
  "summary": "The recorded Flutter slice adds regular/business account switching, business profile information, event/product surfaces and integrated editing against AppView contracts.",
  "currentDocs": [
    "app/README.md",
    "docs/architecture.md",
    "docs/adr/010-business-profile-event-lexicons-and-pinned-data.md"
  ],
  "code": [
    "app/lib/business",
    "app/lib/profile",
    "appview/internal/business"
  ],
  "evidence": [
    "docs/changes/2026-08-29-flutter-business-profiles/05-implementation-plan.md#execution-notes",
    "docs/changes/2026-08-29-flutter-business-profiles/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": [
    "Historical MAN-001 and MAN-002 business/device checks remain release boundaries."
  ]
}
---

# Flutter business profiles and event/product editors

## Outcome and decisions

The recorded Flutter slice adds regular/business account switching, business profile information, event/product surfaces and integrated editing against AppView contracts. Accepted overlays reconcile against authoritative records with expected CIDs and account/record-generation fences. Corrections preserve an entire list and cursor when an in-flight row becomes stale, rather than publishing partial old data. Fixed-length keyed TabController ownership prevents semantics failures when profile tab counts change.

## Evidence and limits

The review retains IR-020; later R14/R15 execution closes it and the runtime tab semantics regression. The final record reports 1,802 Flutter tests, clean analysis and a successful hot reload. Manual business/mobile checks MAN-001 and MAN-002 remain separate release evidence.

Historical evidence: [05-implementation-plan.md — Execution Notes](../../../docs/changes/2026-08-29-flutter-business-profiles/05-implementation-plan.md#execution-notes); [06-implementation-review.md](../../../docs/changes/2026-08-29-flutter-business-profiles/06-implementation-review.md).

## Current entry points

[app/README.md](../../../app/README.md); [docs/architecture.md](../../../docs/architecture.md); [docs/adr/010-business-profile-event-lexicons-and-pinned-data.md](../../../docs/adr/010-business-profile-event-lexicons-and-pinned-data.md). Recorded follow-ups are historical evidence boundaries; recheck them against later records and current code before treating them as open defects.
