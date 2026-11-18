---
{
  "id": "2026-11-09-moderation-sanction-coverage",
  "title": "Moderation sanction reason coverage",
  "source": "docs/changes/2026-11-09-moderation-sanction-coverage",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "appview",
    "tooling"
  ],
  "topics": [
    "moderation",
    "sanctions",
    "policy"
  ],
  "keywords": [
    "illegal_content",
    "otherIllegalContent",
    "severe rationale",
    "000122",
    "human review"
  ],
  "summary": "Added factual-evidence and private-rationale rules for Other illegal content and extended supported sanction eligibility.",
  "currentDocs": [
    "docs/operations/moderation-reference.md",
    "docs/operations/moderation-operators.md"
  ],
  "code": [
    "appview/internal/moderation",
    "appview/admin",
    "appview/migrations"
  ],
  "evidence": [
    "docs/changes/2026-11-09-moderation-sanction-coverage/proposal.md",
    "docs/changes/2026-11-09-moderation-sanction-coverage/validation.md#local-verification"
  ],
  "supersededBy": [],
  "followUps": [
    "Production migration/deployment and benign hosted operator verification were not performed in this implementation record."
  ]
}
---

# Moderation sanction reason coverage

The owner-approved change adds `illegal_content` as Other illegal content. Validation requires factual evidence, exactly the intended classification, a private legal rationale and a safe owner explanation. Legal rationale is rejected on unrelated or no-action decisions rather than becoming an optional free-form field everywhere.

Strike/severe eligibility expands for that reason and severe eligibility for selected existing fraud/scam and self-harm reasons. Generic Other remains ineligible; severe action retains separate severity rationale and supported human review. Private persistence and operator request/history forms are updated with a dedicated migration.

The validation record describes local checks and source behavior, not a real sanction or production migration. Report mappings, ordinary signal behavior, PDS ownership and external reporting destinations are not widened by this scope. Current moderation reference and operator runbook should be consulted before applying a reason or releasing the migration.

Evidence: [proposal.md](../2026-11-09-moderation-sanction-coverage/proposal.md); [validation.md](../2026-11-09-moderation-sanction-coverage/validation.md#local-verification). Current entry points are listed in the metadata; consult maintained guidance before applying historical rules.
