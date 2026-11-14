---
{
  "id": "2026-08-31-onboarding-experience",
  "title": "Account-wide onboarding and initial profile projection",
  "source": "docs/changes/2026-08-31-onboarding-experience",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app",
    "appview"
  ],
  "topics": [
    "onboarding",
    "auth",
    "profiles"
  ],
  "keywords": [
    "Skip",
    "Bluesky prefill",
    "craft preferences",
    "Instagram step",
    "confirmed handoff",
    "craftsky_profiles"
  ],
  "summary": "The recorded change replaces placeholder onboarding with skippable profile/craft/Instagram steps and AppView-owned account-wide completion state.",
  "currentDocs": [
    "app/README.md",
    "docs/architecture.md"
  ],
  "code": [
    "app/lib/onboarding",
    "appview/internal/auth",
    "appview/internal/api/onboarding.go"
  ],
  "evidence": [
    "docs/changes/2026-08-31-onboarding-experience/05-implementation-plan.md#runtime-registration-regression-verification",
    "docs/changes/2026-08-31-onboarding-experience/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": [
    "Historical MAN-001 and MAN-002 physical onboarding/screen-reader checks were deferred."
  ]
}
---

# Account-wide onboarding and initial profile projection

## Outcome and decisions

The recorded change replaces placeholder onboarding with skippable profile/craft/Instagram steps and AppView-owned account-wide completion state. Identity prefill, unsaved edits, account changes and retry remain distinct states. Later correction ensures a newly registered member’s canonical profile projection exists before handoff confirmation, preventing startup profile_not_found while asynchronous Tap projection catches up.

## Evidence and limits

The opening review lists historical loading/lifecycle and verification gaps. Later execution records all listed corrections, full Flutter tests/analysis and the complete AppView release gate. A real-PostgreSQL registration regression asserts membership before handoff creation; physical mobile and screen-reader checks MAN-001/MAN-002 remain deferred.

Historical evidence: [05-implementation-plan.md — Runtime Registration Regression Verification](../../../docs/changes/2026-08-31-onboarding-experience/05-implementation-plan.md#runtime-registration-regression-verification); [06-implementation-review.md](../../../docs/changes/2026-08-31-onboarding-experience/06-implementation-review.md).

## Current entry points

[app/README.md](../../../app/README.md); [docs/architecture.md](../../../docs/architecture.md). Recorded follow-ups are historical evidence boundaries; recheck them against later records and current code before treating them as open defects.
