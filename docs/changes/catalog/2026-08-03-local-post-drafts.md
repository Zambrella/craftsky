---
{
  "id": "2026-08-03-local-post-drafts",
  "title": "Local explicit drafts and submit-time media upload",
  "source": "docs/changes/2026-08-03-local-post-drafts",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app"
  ],
  "topics": [
    "drafts",
    "media",
    "durability"
  ],
  "keywords": [
    "LocalPostDraftRepository",
    "manifest",
    "submit-time upload",
    "screen awake",
    "atomic save",
    "source loss"
  ],
  "summary": "The recorded mobile slice persists explicit account-owned post drafts locally and delays media publication until submission.",
  "currentDocs": [
    "app/README.md",
    "docs/architecture.md"
  ],
  "code": [
    "app/lib/drafts",
    "app/lib/feed/widgets/post_composer_sheet.dart"
  ],
  "evidence": [
    "docs/changes/2026-08-03-local-post-drafts/05-implementation-plan.md#verification-log",
    "docs/changes/2026-08-03-local-post-drafts/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": [
    "Historical MAN-001 through MAN-004 and the four-media Drafts-page responsiveness check remain manual evidence boundaries."
  ]
}
---

# Local explicit drafts and submit-time media upload

## Outcome and decisions

The recorded mobile slice persists explicit account-owned post drafts locally and delays media publication until submission. Draft manifests and retained media must survive reopen without crossing account boundaries. Screen-awake release is best effort and cannot erase a successful result or replace the original operation failure. Post-review corrections retain the save controller through asynchronous completion and handle realistic upload duration.

## Evidence and limits

The review is Approved with notes. Final execution records 1,287 app tests, clean analysis and production-shaped delayed-save regressions. Physical persistence/interruption, screen-awake, accessibility/maximum-media responsiveness and sign-out/data-removal checks remain manual boundaries.

Historical evidence: [05-implementation-plan.md — Verification Log](../../../docs/changes/2026-08-03-local-post-drafts/05-implementation-plan.md#verification-log); [06-implementation-review.md](../../../docs/changes/2026-08-03-local-post-drafts/06-implementation-review.md).

## Current entry points

[app/README.md](../../../app/README.md); [docs/architecture.md](../../../docs/architecture.md). Recorded follow-ups are historical evidence boundaries; recheck them against later records and current code before treating them as open defects.
