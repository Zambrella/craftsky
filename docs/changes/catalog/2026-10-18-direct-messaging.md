---
{
  "id": "2026-10-18-direct-messaging",
  "title": "Private Bluesky messaging and foreground synchronization",
  "source": "docs/changes/2026-10-18-direct-messaging",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app",
    "appview",
    "web-client"
  ],
  "topics": [
    "chat",
    "privacy",
    "synchronization"
  ],
  "keywords": [
    "Bluesky Chat",
    "Retry-After",
    "accepted unread badge",
    "sync coordinator",
    "cached inbox",
    "ambiguous send",
    "message drafts"
  ],
  "summary": "This evolving package records private one-to-one Bluesky messaging, authorization boundaries and later foreground/inbox lifecycle refinements. Both composers keep unsent text in memory while open. Only submitted attempts use device-local delivery recovery.",
  "currentDocs": [
    "docs/architecture.md",
    "app/README.md",
    "appview/README.md"
  ],
  "code": [
    "app/lib/chat",
    "appview/internal/chat",
    "web-client/src/lib/features/messages"
  ],
  "evidence": [
    "docs/changes/2026-10-18-direct-messaging/06-validation-evidence.md#cached-inbox-on-conversation-return-2026-10-31",
    "docs/changes/2026-10-18-direct-messaging/05-implementation-plan.md",
    "docs/changes/2026-10-18-direct-messaging/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": [
    "Owner-run physical/provider messaging/report gates remain distinct from local synchronization and SDK fixtures."
  ]
}
---

# Private Bluesky messaging and foreground synchronization

## Outcome and decisions

This evolving package records private one-to-one Bluesky messaging, authorization boundaries and later foreground/inbox lifecycle refinements. Chat stays on its separate provider boundary with server-owned OAuth credentials. Account/access/generation fences protect private state, accepted unread counts exclude requests, and returning from a conversation refreshes retained inbox controllers instead of destroying visible rows. Ambiguous non-idempotent sends require an explicit recovery action. The original local draft requirement (FR-019) is superseded. Both clients keep unsent text in memory while the composer is open, including shared links. Device storage supports recovery of submitted attempts.

## Evidence and limits

The review retains IR-006 as correction input, while later execution/validation records rate-limit fixes and October 31 cached-inbox/app-wide refresh regressions. Local client/backend builds and suites are extensive; live Bluesky report acceptance and MAN-001–MAN-004/AT-024 remain explicitly outside automated proof.

Historical evidence: [06-validation-evidence.md — Cached inbox on conversation return: 2026-10-31](../../../docs/changes/2026-10-18-direct-messaging/06-validation-evidence.md#cached-inbox-on-conversation-return-2026-10-31); [05-implementation-plan.md](../../../docs/changes/2026-10-18-direct-messaging/05-implementation-plan.md); [06-implementation-review.md](../../../docs/changes/2026-10-18-direct-messaging/06-implementation-review.md).

## Current entry points

[docs/architecture.md](../../../docs/architecture.md); [app/README.md](../../../app/README.md); [appview/README.md](../../../appview/README.md). Recorded follow-ups are historical evidence boundaries; recheck them against later records and current code before treating them as open defects.
