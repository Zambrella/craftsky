---
{
  "id": "2026-11-09-child-safety-support",
  "title": "Safety information and support signposting",
  "source": "docs/changes/2026-11-09-child-safety-support",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app",
    "web-client",
    "web"
  ],
  "topics": [
    "safety",
    "support",
    "reporting"
  ],
  "keywords": [
    "/safety",
    "young people",
    "parents carers",
    "Find support",
    "Report to CraftSky"
  ],
  "summary": "Implemented public safety guidance and optional support signposting without adding required report steps or certifying wider policy readiness.",
  "currentDocs": [
    "online-safety/README.md",
    "docs/operations/moderation-reference.md"
  ],
  "code": [
    "app/lib/safety",
    "web-client/src",
    "web/safety.html"
  ],
  "evidence": [
    "docs/changes/2026-11-09-child-safety-support/proposal.md",
    "docs/changes/2026-11-09-child-safety-support/validation.md#local-verification"
  ],
  "supersededBy": [],
  "followUps": [
    "Hosted safety-page publication and whole-policy/public-launch approval require separate evidence."
  ]
}
---

# Safety information and support signposting

The owner-approved feature provides public guidance for young people and parents/carers, illustrated report/block/mute steps and equivalent readable text. Native and web clients link the guide from relevant onboarding, settings and messaging surfaces; report completion can offer optional support without creating a mandatory extra step or a report-history product.

The implementation record separates this specific approval from children's risk ratings, full policy approval and final launch readiness. Native signed-out/settings navigation and local UI/browser checks were exercised; live onboarding/provider behavior, hosted publication and real-world support handling are not inferred from those results.

Keep report ownership and external support distinct: CraftSky reports still use its existing reporting/moderation path. Current safety documentation and maintained operator guidance govern policy and operational obligations. Historical screenshots/local validation do not approve production collection, outgoing messages or the broader assessment.

Evidence: [proposal.md](../2026-11-09-child-safety-support/proposal.md); [validation.md](../2026-11-09-child-safety-support/validation.md#local-verification). Current entry points are listed in the metadata; consult maintained guidance before applying historical rules.
