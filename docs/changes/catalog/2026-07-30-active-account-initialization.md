---
{
  "id": "2026-07-30-active-account-initialization",
  "title": "Unapproved account initialization gate discussion",
  "source": "docs/changes/2026-07-30-active-account-initialization",
  "implementation": "proposed",
  "archive": "archived",
  "components": [
    "app"
  ],
  "topics": [
    "auth",
    "lifecycle",
    "languages"
  ],
  "keywords": [
    "eager initialization",
    "activeContentLanguagePolicyProvider",
    "account critical readiness",
    "requireValue",
    "session lease"
  ],
  "summary": "This discussion note explores a shared account-scoped readiness gate to replace repeated language-policy barriers in content providers.",
  "currentDocs": [
    "app/README.md",
    "docs/architecture.md"
  ],
  "code": [
    "app/lib/app.dart",
    "app/lib/auth/providers",
    "app/lib/languages/providers"
  ],
  "evidence": [
    "docs/changes/2026-07-30-active-account-initialization/00-future-work-notes.md#status"
  ],
  "supersededBy": [],
  "followUps": [
    "This discussion is unapproved; resolve the gate scope and session-lease questions before using it as implementation direction."
  ]
}
---

# Unapproved account initialization gate discussion

## Outcome and decisions

This discussion note explores a shared account-scoped readiness gate to replace repeated language-policy barriers in content providers. The language barrier has a privacy purpose: filtered content must not appear before preferences load. Proposed gating would cover only genuinely critical asynchronous dependencies, retain reactive preference changes and stale-session fencing, and preserve account switching/sign-out after failures. Push and analytics are explicitly non-blocking.

## Evidence and limits

The source explicitly says Not yet approved for implementation. There is no implementation plan, executed test record or decision resolving the listed shell/onboarding/error-screen questions. Current account lifecycle code must be inspected independently before reviving this proposal.

Historical evidence: [00-future-work-notes.md — Status](../../../docs/changes/2026-07-30-active-account-initialization/00-future-work-notes.md#status).

## Current entry points

[app/README.md](../../../app/README.md); [docs/architecture.md](../../../docs/architecture.md). Recorded follow-ups are historical evidence boundaries; recheck them against later records and current code before treating them as open defects.
