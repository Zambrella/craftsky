---
{
  "id": "2026-05-27-profile-social-summary",
  "title": "Profile social summaries and graph lists",
  "source": "docs/changes/2026-05-27-profile-social-summary",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app",
    "appview"
  ],
  "topics": [
    "profiles",
    "social-graph",
    "pagination"
  ],
  "keywords": [
    "mutual followers",
    "account age",
    "recent posts",
    "Following Craftsky-only",
    "Load more",
    "ProfileMutualFollowersSheet"
  ],
  "summary": "The recorded change replaces visible profile follower counts with mutuals, account age and authored-post activity while retaining follower/following access from Settings.",
  "currentDocs": [
    "docs/architecture.md",
    "app/README.md",
    "appview/README.md"
  ],
  "code": [
    "app/lib/profile/widgets/profile_meta_section.dart",
    "app/lib/settings/pages/follow_list_page.dart",
    "appview/internal/api/profile_store.go"
  ],
  "evidence": [
    "docs/changes/2026-05-27-profile-social-summary/05-implementation-plan.md#review-fix-verification",
    "docs/changes/2026-05-27-profile-social-summary/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": [
    "Optional historical large-list/device polish checks were not recorded as executed."
  ]
}
---

# Profile social summaries and graph lists

## Outcome and decisions

The recorded change replaces visible profile follower counts with mutuals, account age and authored-post activity while retaining follower/following access from Settings. Root posts and projects have distinct counts. Later clarification restricts Following to CraftSky members. Review corrections add cursor-driven Load more behavior to graph lists and mutuals, plus approved supporting indexes, preserving count/order semantics.

## Evidence and limits

The initial review identified first-page-only lists; the execution record explicitly closes RF-1/RF-2 and records Flutter, Go and analysis checks. The final following clarification used a pure query test while its seeded DB case skipped locally, so historical verification scopes must not be collapsed into one claim.

Historical evidence: [05-implementation-plan.md — Review Fix Verification](../../../docs/changes/2026-05-27-profile-social-summary/05-implementation-plan.md#review-fix-verification); [06-implementation-review.md](../../../docs/changes/2026-05-27-profile-social-summary/06-implementation-review.md).

## Current entry points

[docs/architecture.md](../../../docs/architecture.md); [app/README.md](../../../app/README.md); [appview/README.md](../../../appview/README.md). Recorded follow-ups are historical evidence boundaries; recheck them against later records and current code before treating them as open defects.
