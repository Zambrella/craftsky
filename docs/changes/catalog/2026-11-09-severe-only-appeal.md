---
{
  "id": "2026-11-09-severe-only-appeal",
  "title": "Resolve severe-suspension-only appeals",
  "source": "docs/changes/2026-11-09-severe-only-appeal",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "appview",
    "tooling"
  ],
  "topics": [
    "moderation",
    "appeals",
    "restoration"
  ],
  "keywords": [
    "severeSuspension",
    "reversedEffects",
    "Change the decision",
    "atomic appeal",
    "57 admin tests"
  ],
  "summary": "Exposed the existing severe-suspension reversal choice so a severe-only appeal can be resolved through one reviewed command.",
  "currentDocs": [
    "docs/operations/moderation-reference.md",
    "docs/operations/moderation-operators.md"
  ],
  "code": [
    "appview/admin",
    "appview/internal/moderation"
  ],
  "evidence": [
    "docs/changes/2026-11-09-severe-only-appeal/validation.md#verification"
  ],
  "supersededBy": [],
  "followUps": [
    "Hosted console release and a benign operator verification remain necessary for live closure."
  ]
}
---

# Resolve severe-suspension-only appeals

The operator console previously removed severe suspension from the changed-appeal selector while still requiring a selected effect. A severe-only appeal therefore could not be resolved as changed, even though standalone restoration could remove its suspension. The fix exposes the backend's already-supported `severeSuspension` reversal.

The existing revision-checked, idempotent appeal command atomically resolves the appeal, negates selected effects and recomputes standing. Other case grounds and unselected effects can remain active. The ordinary restoration selector is unchanged, and no empty-effect changed-appeal command or new API/migration/PDS authority is introduced.

Validation includes two console interaction regressions, the admin suite/build/type checks and four disposable-PostgreSQL scenarios covering retained unrelated suspension/strike/warning grounds and exact replay. Hosted operator verification awaits frontend release. No real decision, notification or production restoration was performed.

Evidence: [validation.md](../2026-11-09-severe-only-appeal/validation.md#verification). Current entry points are listed in the metadata; consult maintained guidance before applying historical rules.
