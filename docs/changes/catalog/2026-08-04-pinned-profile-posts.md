---
{
  "id": "2026-08-04-pinned-profile-posts",
  "title": "Two private profile pin slots",
  "source": "docs/changes/2026-08-04-pinned-profile-posts",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app",
    "appview"
  ],
  "topics": [
    "profiles",
    "pins",
    "pagination"
  ],
  "keywords": [
    "pin-bound cursor",
    "target-specific unpin",
    "profile promotion",
    "two slots"
  ],
  "summary": "Added owner-managed AppView-only profile pins with stale-safe mutations and pin-aware pagination.",
  "currentDocs": [
    "docs/architecture.md",
    "app/README.md"
  ],
  "code": [
    "appview/internal/api/profile_pin.go",
    "appview/internal/api/profile_pin_store.go",
    "app/lib/profile"
  ],
  "evidence": [
    "docs/changes/2026-08-04-pinned-profile-posts/05-implementation-plan.md#final-verification",
    "docs/changes/2026-08-04-pinned-profile-posts/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": [
    "Manual responsive and assistive-technology pin checks remain separate from the recorded automation."
  ]
}
---

# Two private profile pin slots

Two private pin slots control promotion on the owner's profile list only. Mutations are owner-scoped and transactional; unpin identifies its target so a stale action cannot remove a newer occupant. Structural cleanup and visibility checks prevent invalid pinned references from leaking or distorting pagination.

The implementation binds profile continuation cursors to pin state and lets authoritative Flutter state reconcile account-fenced mutation results. Pin attribution belongs to profile presentation rather than global timeline ranking. The approved scope adds no subscription/payment eligibility, public PDS pin records, Lexicon or Tap behavior.

The final review accepts automated correctness, migration, state and privacy behavior while retaining test-traceability, a Should-level diagnostic edge and manual UI notes. Read current profile-pin source for the active contract; the package is historical evidence for private promotion and cursor invalidation, not permission to extend ranking or monetize pins.

Evidence: [05-implementation-plan.md](../2026-08-04-pinned-profile-posts/05-implementation-plan.md#final-verification); [06-implementation-review.md](../2026-08-04-pinned-profile-posts/06-implementation-review.md). Current entry points are listed in the metadata; consult maintained guidance before applying historical rules.
