---
{
  "id": "2026-07-17-multi-account-sessions",
  "title": "Multi-account sessions and recipient-safe routing",
  "source": "docs/changes/2026-07-17-multi-account-sessions",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app",
    "appview"
  ],
  "topics": [
    "auth",
    "account-isolation",
    "notifications"
  ],
  "keywords": [
    "AccountSessionLease",
    "SessionRegistry",
    "fixed-session client",
    "stale 401",
    "fallback sign-out",
    "secure snapshot"
  ],
  "summary": "The recorded change enables retained independent account sessions, account switching and notification routing without cross-account response/cache leakage.",
  "currentDocs": [
    "docs/architecture.md",
    "app/README.md",
    "appview/README.md"
  ],
  "code": [
    "app/lib/auth/models",
    "app/lib/auth/providers",
    "appview/internal/auth"
  ],
  "evidence": [
    "docs/changes/2026-07-17-multi-account-sessions/05-implementation-plan.md#sign-out-confirmation-final-verification",
    "docs/changes/2026-07-17-multi-account-sessions/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": [
    "Historical physical-device/platform checks MAN-001 through MAN-003 were not executed."
  ]
}
---

# Multi-account sessions and recipient-safe routing

## Outcome and decisions

The recorded change enables retained independent account sessions, account switching and notification routing without cross-account response/cache leakage. Fixed-session clients, session-scoped invalidation and stale-completion fences are intentional correctness boundaries. Later sign-out feedback distinguishes last-account sign-out from activation of a retained fallback account. Review simplification proposals are choices, not authority to discard isolation invariants.

## Evidence and limits

The review is Approved with notes. Final sign-out verification records 919 Flutter tests, clean analysis and backend race checks passing. Physical mobile/platform gates MAN-001 through MAN-003 remained unexecuted. Later account-operation packages may simplify persistence and logout policy.

Historical evidence: [05-implementation-plan.md — Sign-Out Confirmation Final Verification](../../../docs/changes/2026-07-17-multi-account-sessions/05-implementation-plan.md#sign-out-confirmation-final-verification); [06-implementation-review.md](../../../docs/changes/2026-07-17-multi-account-sessions/06-implementation-review.md).

## Current entry points

[docs/architecture.md](../../../docs/architecture.md); [app/README.md](../../../app/README.md); [appview/README.md](../../../appview/README.md). Recorded follow-ups are historical evidence boundaries; recheck them against later records and current code before treating them as open defects.
