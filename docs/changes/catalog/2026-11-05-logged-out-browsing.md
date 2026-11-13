---
{
  "id": "2026-11-05-logged-out-browsing",
  "title": "Anonymous browsing and explicit guest visibility",
  "source": "docs/changes/2026-11-05-logged-out-browsing",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app",
    "appview",
    "web-client"
  ],
  "topics": [
    "guest-browsing",
    "privacy",
    "visibility"
  ],
  "keywords": [
    "!no-unauthenticated",
    "public SSR",
    "Explore CraftSky",
    "guest transport",
    "profile loader lifetime",
    "guest reports"
  ],
  "summary": "The recorded cross-client change enables anonymous craft/content browsing with server-owned visibility and account-only actions behind sign-in.",
  "currentDocs": [
    "docs/architecture.md",
    "app/README.md",
    "web-client/README.md"
  ],
  "code": [
    "app/lib/shared/browsing",
    "appview/internal/guestvisibility",
    "web-client/src/lib/features/browsing"
  ],
  "evidence": [
    "docs/changes/2026-11-05-logged-out-browsing/05-implementation-plan.md#10-november-2026-explicit-opt-out-amendment",
    "docs/changes/2026-11-05-logged-out-browsing/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": [
    "Historical platform/manual guest-browsing checks remain separate from automated loader/visibility coverage."
  ]
}
---

# Anonymous browsing and explicit guest visibility

## Outcome and decisions

The recorded cross-client change enables anonymous craft/content browsing with server-owned visibility and account-only actions behind sign-in. Shared profile loaders own refreshed content/continuations so equivalent route identities cannot retain stale guest content. A later approved amendment treats missing/unknown preference state as guest-visible, with only explicit !no-unauthenticated opting out; lifecycle, moderation and media-safety restrictions still apply. No refresh/backfill is required to infer the default.

## Evidence and limits

The final implementation review is Approved with notes, followed by the visibility amendment’s backend/client regressions. Local fixtures, SSR and widget checks do not prove every hosted provider/platform state. Historical requirements before the amendment must not override the current architecture’s explicit-opt-out rule.

Historical evidence: [05-implementation-plan.md — 10 November 2026: explicit opt-out amendment](../../../docs/changes/2026-11-05-logged-out-browsing/05-implementation-plan.md#10-november-2026-explicit-opt-out-amendment); [06-implementation-review.md](../../../docs/changes/2026-11-05-logged-out-browsing/06-implementation-review.md).

## Current entry points

[docs/architecture.md](../../../docs/architecture.md); [app/README.md](../../../app/README.md); [web-client/README.md](../../../web-client/README.md). Recorded follow-ups are historical evidence boundaries; recheck them against later records and current code before treating them as open defects.
