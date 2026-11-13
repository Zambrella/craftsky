---
{
  "id": "2026-10-11-account-operation-simplification",
  "title": "Account-bound operation and retry simplification",
  "source": "docs/changes/2026-10-11-account-operation-simplification",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app"
  ],
  "topics": [
    "account-isolation",
    "concurrency",
    "retries"
  ],
  "keywords": [
    "ActiveAccountOperation",
    "retained session",
    "captured repository",
    "retry component",
    "80-file inventory"
  ],
  "summary": "Consolidated selected account-bound operations and retries while retaining distinct authority, revision and domain fences.",
  "currentDocs": [
    "app/README.md",
    "docs/development/logging-error-reporting.md"
  ],
  "code": [
    "app/lib/auth",
    "app/lib/shared",
    "app/lib/feed"
  ],
  "evidence": [
    "docs/changes/2026-10-11-account-operation-simplification/05-implementation-plan.md#test-execution-and-evidence",
    "docs/changes/2026-10-11-account-operation-simplification/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": [
    "Signed-in physical-device validation was not established by this package’s automated gate."
  ]
}
---

# Account-bound operation and retry simplification

The simplification removes repeated retry orchestration where feature outcomes genuinely match. Operations capture their repositories/dependencies before awaiting work, so retry cannot silently select a replacement account's dependency. Active-account authority remains distinct from retained-session authority, request revisions and domain fences.

The implementation deliberately does not migrate every provider into a new family. Its inventory records dispositions for the original 80 files and specialized areas, including retained guards where a shared abstraction would add complexity. The review finds no blocking account-isolation, retry-contract or diagnostic regression.

The execution record reports focused and complete Flutter verification and static analysis, while retaining signed-in device validation as a separate gap. Later Riverpod lifecycle and client error-reporting passes refine current source. Use this package for the rationale behind selective consolidation and captured dependencies, not as a mandate for a global session-family or generic retry framework.

Evidence: [05-implementation-plan.md](../2026-10-11-account-operation-simplification/05-implementation-plan.md#test-execution-and-evidence); [06-implementation-review.md](../2026-10-11-account-operation-simplification/06-implementation-review.md). Current entry points are listed in the metadata; consult maintained guidance before applying historical rules.
