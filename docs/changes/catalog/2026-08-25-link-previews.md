---
{
  "id": "2026-08-25-link-previews",
  "title": "Private link-preview fetching and scheduled media lifecycle",
  "source": "docs/changes/2026-08-25-link-previews",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app",
    "appview"
  ],
  "topics": [
    "link-previews",
    "media",
    "scheduling"
  ],
  "keywords": [
    "LinkPreviewService",
    "SSRF",
    "external card",
    "scheduled thumbnail",
    "object attempt identity",
    "timestamp precision"
  ],
  "summary": "The recorded slice fetches bounded external preview metadata and thumbnails through AppView and integrates preview cards into post composition and scheduling.",
  "currentDocs": [
    "docs/architecture.md",
    "appview/README.md",
    "app/README.md"
  ],
  "code": [
    "appview/internal/linkpreview",
    "appview/internal/scheduledposts",
    "app/lib/feed/composer"
  ],
  "evidence": [
    "docs/changes/2026-08-25-link-previews/05-implementation-plan.md#third-correction-final-verification",
    "docs/changes/2026-08-25-link-previews/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": [
    "Consult later execution corrections before treating the initial review findings as current defects; external/mobile runtime evidence remains separate."
  ]
}
---

# Private link-preview fetching and scheduled media lifecycle

## Outcome and decisions

The recorded slice fetches bounded external preview metadata and thumbnails through AppView and integrates preview cards into post composition and scheduling. Fetching must retain destination safety, redirect/response limits and private input protection. Later corrections preserve deliberate card removal/selection during delayed hydration, reject malformed blobs and allocate fresh immutable media identities. A runtime fix makes PostgreSQL’s canonical timestamp precision authoritative for stored object-attempt deadlines.

## Evidence and limits

The review is an earlier correction snapshot. Execution extends through Step 85 with additional lifecycle/privacy corrections, full backend race verification and an AppView rebuild confirming the thumbnail deadline fix. That local runtime evidence is not hosted/provider or physical-device proof.

Historical evidence: [05-implementation-plan.md — Third Correction Final Verification](../../../docs/changes/2026-08-25-link-previews/05-implementation-plan.md#third-correction-final-verification); [06-implementation-review.md](../../../docs/changes/2026-08-25-link-previews/06-implementation-review.md).

## Current entry points

[docs/architecture.md](../../../docs/architecture.md); [appview/README.md](../../../appview/README.md); [app/README.md](../../../app/README.md). Recorded follow-ups are historical evidence boundaries; recheck them against later records and current code before treating them as open defects.
