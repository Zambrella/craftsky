---
{
  "id": "2026-09-09-moderation-strikes",
  "title": "Moderation cases, strikes, appeals and suspension",
  "source": "docs/changes/2026-09-09-moderation-strikes",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app",
    "appview"
  ],
  "topics": [
    "moderation",
    "appeals",
    "standing"
  ],
  "keywords": [
    "immutable effect history",
    "strike suspension",
    "appeal decision",
    "stored suspension",
    "IR-010"
  ],
  "summary": "Implemented moderated standing and owner appeals with historical effect records and server-authoritative suspension enforcement.",
  "currentDocs": [
    "docs/operations/moderation-reference.md",
    "docs/operations/moderation-operators.md",
    "docs/architecture.md"
  ],
  "code": [
    "appview/internal/moderation",
    "app/lib/moderation",
    "appview/admin"
  ],
  "evidence": [
    "docs/changes/2026-09-09-moderation-strikes/05-implementation-plan.md",
    "docs/changes/2026-09-09-moderation-strikes/06-implementation-review.md#test-evidence"
  ],
  "supersededBy": [],
  "followUps": [
    "Consult current release records for the package’s manual owner-history and assistive-technology checks."
  ]
}
---

# Moderation cases, strikes, appeals and suspension

Moderation decisions produce attributable case history, effects, strikes and suspension state. Owner history presents immutable effect actions as historical facts and offers an appeal only for an unappealed decision event. Enforcement belongs to server access boundaries, not merely hidden client controls.

The final review accepts IR-008 through IR-010, including actual route/post-handler checks against stored suspension and a recording PDS effect executor. It records complete required-PostgreSQL and Flutter/static/release-equivalent verification. Earlier capacity-blocked release attempts remain in the execution log as historical failures, not the final verdict.

Later sanction-coverage and severe-only appeal work extends reason eligibility and operator controls. The maintained moderation reference and operator runbook govern current review, rationale and restoration policy. No local test outcome in this package authorizes a real sanction, a public policy publication or deletion of another person's PDS records.

Evidence: [05-implementation-plan.md](../2026-09-09-moderation-strikes/05-implementation-plan.md); [06-implementation-review.md](../2026-09-09-moderation-strikes/06-implementation-review.md#test-evidence). Current entry points are listed in the metadata; consult maintained guidance before applying historical rules.
