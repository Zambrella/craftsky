---
{
  "id": "2026-11-09-urgent-safety-email-alerts",
  "title": "Private durable urgent-safety email alerts",
  "source": "docs/changes/2026-11-09-urgent-safety-email-alerts",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "appview",
    "docs"
  ],
  "topics": [
    "online-safety",
    "alerts",
    "durability"
  ],
  "keywords": [
    "Brevo",
    "safety_email_deliveries",
    "idempotency window",
    "urgent digest",
    "moderation console",
    "migration 000123"
  ],
  "summary": "The recorded backend change adds durable urgent-safety alert delivery through Brevo, with private queue discovery, leases, retries and source deduplication.",
  "currentDocs": [
    "docs/operations/moderation-operators.md",
    "docs/operations/production-render.md",
    "online-safety/README.md"
  ],
  "code": [
    "appview/internal/safetyalerts"
  ],
  "evidence": [
    "docs/changes/2026-11-09-urgent-safety-email-alerts/validation.md#local-verification"
  ],
  "supersededBy": [],
  "followUps": [
    "Configure/verify Brevo sender and recipient, authorize deployment, and prove inbox/phone receipt plus monitoring/backup coverage before relying on alerts."
  ]
}
---

# Private durable urgent-safety email alerts

## Outcome and decisions

The recorded backend change adds durable urgent-safety alert delivery through Brevo, with private queue discovery, leases, retries and source deduplication. Emails contain only a static prompt and authenticated console URL, excluding report content, identities, categories, case references, attachments and restricted evidence. Committed urgent moderation/intake/scanner/workflow sources converge into the digest; no new moderation decision or public API route is introduced.

## Evidence and limits

Local validation documents queue and privacy behavior and the unshipped migration’s renumbering to 000123. Production sender/domain configuration, exact-tag deployment and authorized inbox/phone verification remain separate. Provider acceptance cannot establish receipt, ambiguous sends may duplicate beyond Brevo’s window, and alerts do not create staffed round-the-clock cover.

Historical evidence: [validation.md — Local Verification](../../../docs/changes/2026-11-09-urgent-safety-email-alerts/validation.md#local-verification).

## Current entry points

[docs/operations/moderation-operators.md](../../../docs/operations/moderation-operators.md); [docs/operations/production-render.md](../../../docs/operations/production-render.md); [online-safety/README.md](../../../online-safety/README.md). Recorded follow-ups are historical evidence boundaries; recheck them against later records and current code before treating them as open defects.
