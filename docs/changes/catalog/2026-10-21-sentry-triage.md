---
{
  "id": "2026-10-21-sentry-triage",
  "title": "Sentry-led fixes and Riverpod ownership follow-up",
  "source": "docs/changes/2026-10-21-sentry-triage",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app",
    "appview"
  ],
  "topics": [
    "observability",
    "riverpod",
    "diagnosis"
  ],
  "keywords": [
    "Sentry issue inventory",
    "file picker",
    "post keyboard",
    "ActiveAccountOperation",
    "provider retention",
    "inNextRelease"
  ],
  "summary": "This package records a Sentry issue investigation and confirmed local fixes for file picking, post keyboard behavior, provider ownership and diagnostic cause preservation.",
  "currentDocs": [
    "docs/development/logging-error-reporting.md",
    "app/README.md"
  ],
  "code": [
    "app/lib/shared/observability",
    "app/lib/auth",
    "appview/internal/observability"
  ],
  "evidence": [
    "docs/changes/2026-10-21-sentry-triage/triage.md",
    "docs/changes/2026-10-21-sentry-triage/riverpod-audit.md",
    "docs/changes/2026-10-21-sentry-triage/file-picker.md",
    "docs/changes/2026-10-21-sentry-triage/post-keyboard.md",
    "docs/changes/2026-10-21-sentry-triage/validation.md#validation-evidence"
  ],
  "supersededBy": [],
  "followUps": [
    "Historical issue inventory/statuses are time-sensitive; requery Sentry before acting on them.",
    "Physical-device and post-release recurrence/symbolication checks remain separate."
  ]
}
---

# Sentry-led fixes and Riverpod ownership follow-up

## Outcome and decisions

This package records a Sentry issue investigation and confirmed local fixes for file picking, post keyboard behavior, provider ownership and diagnostic cause preservation. The triage distinguishes demonstrated causes, evidence gaps and deliberate retry escalation. Riverpod follow-up addresses operation retention and dependency ownership, rather than masking late completion failures. Selected issue statuses were set for the next release; that is not proof a new build was deployed or recurrence ceased.

## Evidence and limits

Validation records complete Flutter/Go release checks and final serialized native SDK regressions. Its final gate repair supersedes the earlier toolchain limitation. Raw events remain outside the repository, and physical-device/native picker behavior plus hosted collection/symbolication retain separate evidence boundaries.

Historical evidence: [triage.md](../../../docs/changes/2026-10-21-sentry-triage/triage.md); [riverpod-audit.md](../../../docs/changes/2026-10-21-sentry-triage/riverpod-audit.md); [file-picker.md](../../../docs/changes/2026-10-21-sentry-triage/file-picker.md); [post-keyboard.md](../../../docs/changes/2026-10-21-sentry-triage/post-keyboard.md); [validation.md — Validation evidence](../../../docs/changes/2026-10-21-sentry-triage/validation.md#validation-evidence).

## Current entry points

[docs/development/logging-error-reporting.md](../../../docs/development/logging-error-reporting.md); [app/README.md](../../../app/README.md). Recorded follow-ups are historical evidence boundaries; recheck them against later records and current code before treating them as open defects.
