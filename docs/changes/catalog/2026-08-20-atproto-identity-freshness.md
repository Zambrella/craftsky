---
{
  "id": "2026-08-20-atproto-identity-freshness",
  "title": "Fresh authoritative identity resolution",
  "source": "docs/changes/2026-08-20-atproto-identity-freshness",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "appview"
  ],
  "topics": [
    "identity",
    "federation",
    "concurrency"
  ],
  "keywords": [
    "Tap identity event",
    "newest trigger",
    "DID cache invalidation",
    "fresh resolution"
  ],
  "summary": "Separated presentation caches from fresh authority checks and fenced identity refresh completion against newer Tap triggers.",
  "currentDocs": [
    "docs/architecture.md",
    "docs/operations/production-render.md"
  ],
  "code": [
    "appview/internal/ingestion",
    "appview/internal/auth",
    "appview/internal/tap"
  ],
  "evidence": [
    "docs/changes/2026-08-20-atproto-identity-freshness/05-implementation-plan.md#post-review-correction-verification",
    "docs/changes/2026-08-20-atproto-identity-freshness/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": [
    "The correction record explicitly requests a fresh implementation review to close historical IR-001."
  ]
}
---

# Fresh authoritative identity resolution

Cached handle/DID presentation and fresh authoritative resolution serve different purposes. Tap identity events schedule durable refresh and invalidate local directory entries; a Tap-provided handle is not inserted as verified identity. Remote work stays outside the acknowledgement boundary, and bounded periodic scanning remains a convergence fallback.

The initial review found that an older in-flight refresh could delay or delete a newer trigger. The later correction verification records affected PostgreSQL packages with and without race detection, compile/static checks and retention of the newest event identity. It explicitly keeps the old Changes required review as correction input and requires fresh review to close that finding formally.

This is implementation and local verification evidence, not a production migration receipt. The package's migration note and current operational architecture should be read before altering trigger reconciliation or relying on historical rollout assumptions.

Evidence: [05-implementation-plan.md](../2026-08-20-atproto-identity-freshness/05-implementation-plan.md#post-review-correction-verification); [06-implementation-review.md](../2026-08-20-atproto-identity-freshness/06-implementation-review.md). Current entry points are listed in the metadata; consult maintained guidance before applying historical rules.
