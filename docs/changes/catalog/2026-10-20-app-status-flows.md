---
{
  "id": "2026-10-20-app-status-flows",
  "title": "Versioned app-status flows and message audience",
  "source": "docs/changes/2026-10-20-app-status-flows",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app",
    "service-status",
    "tooling"
  ],
  "topics": [
    "service-status",
    "updates",
    "onboarding"
  ],
  "keywords": [
    "D-014",
    "D-015",
    "D-016",
    "D-017",
    "pub_semver",
    "message audience"
  ],
  "summary": "Implemented status/version presentation with later amendments for trusted images, bounded SemVer and onboarded signed-in audiences.",
  "currentDocs": [
    "docs/operations/service-status.md",
    "service-status/README.md"
  ],
  "code": [
    "app/lib/service_status",
    "service-status"
  ],
  "evidence": [
    "docs/changes/2026-10-20-app-status-flows/05-implementation-plan.md#onboarded-message-audience--d-017-2026-10-22",
    "docs/changes/2026-10-20-app-status-flows/05-implementation-plan.md#semver-package-simplification--d-015-2026-10-21",
    "docs/changes/2026-10-20-app-status-flows/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": [
    "Review the package’s remaining release checks for hosted status, physical-device and assistive-technology proof."
  ]
}
---

# Versioned app-status flows and message audience

This package separates universal maintenance/update blockers from optional informational messages. Status fetching and polling remain authentication-independent. The message audience, however, was subsequently restricted to a resolved signed-in current account with completed onboarding; uncertain auth, account transitions and onboarding errors fail closed for messages.

Read the later amendments before the original reviewed design. D-014 chooses plain `Image.network` for the trusted public image host, replacing earlier custom-transport guarantees. D-015 replaces custom Dart comparison with a strict `pub_semver` wrapper and shared numeric-component bound. D-016/D-017 narrow the audience without another fetch or dismissal write when onboarding completes.

Execution entries record regression and targeted verification for each amendment. Earlier image-transport tests and approval text describe the superseded design, not current requirements. Hosted/device/manual status behavior remains a release-evidence boundary; consult current service-status source and operations guidance for active wire and publication policy.

Evidence: [05-implementation-plan.md](../2026-10-20-app-status-flows/05-implementation-plan.md#onboarded-message-audience--d-017-2026-10-22); [06-implementation-review.md](../2026-10-20-app-status-flows/06-implementation-review.md). Current entry points are listed in the metadata; consult maintained guidance before applying historical rules.

Section pointers: [semver-package-simplification--d-015-2026-10-21](../2026-10-20-app-status-flows/05-implementation-plan.md#semver-package-simplification--d-015-2026-10-21).
