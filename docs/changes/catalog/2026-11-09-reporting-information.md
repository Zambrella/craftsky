---
{
  "id": "2026-11-09-reporting-information",
  "title": "Post-report explanation without status tracking UI",
  "source": "docs/changes/2026-11-09-reporting-information",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app",
    "web-client",
    "docs"
  ],
  "topics": [
    "reports",
    "online-safety",
    "user-messages"
  ],
  "keywords": [
    "reportSubmitSuccess",
    "routine reports",
    "UK business days",
    "post-submit confirmation",
    "no reporter status UI"
  ],
  "summary": "The owner narrowed this change to brief shared mobile/web confirmation copy explaining report handling and indicative routine timelines.",
  "currentDocs": [
    "online-safety/README.md",
    "docs/operations/moderation-operators.md"
  ],
  "code": [
    "app/lib/l10n",
    "web-client/src/lib/features/moderation/report-confirmation.ts"
  ],
  "evidence": [
    "docs/changes/2026-11-09-reporting-information/proposal.md",
    "docs/changes/2026-11-09-reporting-information/validation.md#result"
  ],
  "supersededBy": [],
  "followUps": [
    "Pre-submit disclosures, relevant communication/outcome content and operational monitoring retain separate assessment items."
  ]
}
---

# Post-report explanation without status tracking UI

## Outcome and decisions

The owner narrowed this change to brief shared mobile/web confirmation copy explaining report handling and indicative routine timelines. It deliberately adds no reporter status screen, report-history API, contact-preference storage or correspondence automation. CraftSky report success uses the shared notification surfaces with longer informational reading time; Bluesky DM report confirmations remain separate and do not inherit CraftSky handling promises.

## Evidence and limits

The validation records the implemented native localization and shared web confirmation across relevant report flows, with local tests and explicit hosted/device limits. The later child-safety-support package is a separate authorization. This copy-only step does not close all reporting-control assessment or operational-monitoring gaps.

Historical evidence: [proposal.md](../../../docs/changes/2026-11-09-reporting-information/proposal.md); [validation.md — Result](../../../docs/changes/2026-11-09-reporting-information/validation.md#result).

## Current entry points

[online-safety/README.md](../../../online-safety/README.md); [docs/operations/moderation-operators.md](../../../docs/operations/moderation-operators.md). Recorded follow-ups are historical evidence boundaries; recheck them against later records and current code before treating them as open defects.
