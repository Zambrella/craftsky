---
{
  "id": "2026-10-14-moderation-console",
  "title": "Custom operator console and protected record previews",
  "source": "docs/changes/2026-10-14-moderation-console",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "appview",
    "tooling"
  ],
  "topics": [
    "moderation",
    "operator-console",
    "evidence"
  ],
  "keywords": [
    "operator session",
    "review-before-write",
    "CSP",
    "record CID",
    "business versions",
    "rich preview"
  ],
  "summary": "Built the authenticated operator console and later added bounded context previews, business records and version notices.",
  "currentDocs": [
    "docs/operations/moderation-operators.md",
    "docs/operations/moderation-reference.md",
    "docs/adr/018-atmosphere-operator-console-sessions.md"
  ],
  "code": [
    "appview/admin",
    "appview/internal/operatorconsole",
    "appview/internal/api"
  ],
  "evidence": [
    "docs/changes/2026-10-14-moderation-console/09-final-validation-rollout.md#current-browser-and-consumer-http-verification",
    "docs/changes/2026-10-14-moderation-console/12-preview-context-business-versions.md#verification"
  ],
  "supersededBy": [],
  "followUps": [
    "Production operator rollout and narrow-screen/manual validation need their own current evidence."
  ]
}
---

# Custom operator console and protected record previews

The staged console adds attributable operator access, case review, specialist workflows, restricted evidence controls and review-before-write commands. It uses existing authenticated admin APIs with revisions and command identity, not direct client database mutation. Publicly packaged assets and no-store/CSP/session boundaries are verified separately from operator authorization.

Later stages add one-level quote/reply/root preview context, indexed business profiles/products and explicit comparisons between captured and current record CIDs. Each referenced version keeps its own lifecycle and image-safety gate; missing or unsafe content becomes a placeholder rather than bypassing moderation controls.

The rollout-preparation record includes local real-cookie walkthroughs, API readback, admin tests/build and release-image asset checks. Production enablement/provisioning, narrow-screen acceptance and live operator/provider checks remain distinct. Read the current operator runbook before acting; historical prototype or capability-map details are not permission to bypass present safeguards.

Evidence: [09-final-validation-rollout.md](../2026-10-14-moderation-console/09-final-validation-rollout.md#current-browser-and-consumer-http-verification); [12-preview-context-business-versions.md](../2026-10-14-moderation-console/12-preview-context-business-versions.md#verification). Current entry points are listed in the metadata; consult maintained guidance before applying historical rules.
