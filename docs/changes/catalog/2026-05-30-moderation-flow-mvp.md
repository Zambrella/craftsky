---
{
  "id": "2026-05-30-moderation-flow-mvp",
  "title": "Moderation visibility and reporting foundation",
  "source": "docs/changes/2026-05-30-moderation-flow-mvp",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app",
    "appview"
  ],
  "topics": [
    "moderation",
    "reports",
    "visibility"
  ],
  "keywords": [
    "report subject",
    "hide",
    "takedown",
    "warn",
    "non-Craftsky profile",
    "Ozone"
  ],
  "summary": "The recorded MVP adds report intake and moderation-aware content/profile visibility without live Ozone or PDS report submission.",
  "currentDocs": [
    "docs/architecture.md",
    "docs/operations/moderation-operators.md",
    "docs/operations/moderation-reference.md"
  ],
  "code": [
    "app/lib/moderation",
    "appview/internal/moderation",
    "appview/internal/api"
  ],
  "evidence": [
    "docs/changes/2026-05-30-moderation-flow-mvp/05-implementation-plan.md#third-review-fix-final-verification",
    "docs/changes/2026-05-30-moderation-flow-mvp/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": [
    "Historical manual moderation checks MAN-001 through MAN-004 were not executed in the final automated pass."
  ]
}
---

# Moderation visibility and reporting foundation

## Outcome and decisions

The recorded MVP adds report intake and moderation-aware content/profile visibility without live Ozone or PDS report submission. AppView controls presentation rather than deleting public PDS content. Review corrections close enforcement bypasses for non-CraftSky/Bluesky profile fallbacks, preserve canonical DID subjects with submitted-handle snapshots, and standardize report-sheet validation and wire models.

## Evidence and limits

The re-review is Approved with notes. A later third correction pass records focused and broad feed/profile/moderation/notification tests and clean analysis. MAN-001 through MAN-004 remain historical human/local checks; the package does not establish external moderation-provider operation.

Historical evidence: [05-implementation-plan.md — Third Review Fix Final Verification](../../../docs/changes/2026-05-30-moderation-flow-mvp/05-implementation-plan.md#third-review-fix-final-verification); [06-implementation-review.md](../../../docs/changes/2026-05-30-moderation-flow-mvp/06-implementation-review.md).

## Current entry points

[docs/architecture.md](../../../docs/architecture.md); [docs/operations/moderation-operators.md](../../../docs/operations/moderation-operators.md); [docs/operations/moderation-reference.md](../../../docs/operations/moderation-reference.md). Recorded follow-ups are historical evidence boundaries; recheck them against later records and current code before treating them as open defects.
