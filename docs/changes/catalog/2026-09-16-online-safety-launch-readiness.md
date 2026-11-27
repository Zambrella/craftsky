---
{
  "id": "2026-09-16-online-safety-launch-readiness",
  "title": "Online-safety implementation and public-launch gates",
  "source": "docs/changes/2026-09-16-online-safety-launch-readiness",
  "implementation": "partial",
  "archive": "active",
  "components": [
    "app",
    "appview",
    "web",
    "infrastructure"
  ],
  "topics": [
    "online-safety",
    "launch-readiness",
    "policy",
    "evidence"
  ],
  "keywords": [
    "restricted evidence",
    "legal holds",
    "retention",
    "age eligibility",
    "IWF",
    "public-release-check"
  ],
  "summary": "Implemented bounded safety workflows while retaining explicit policy approval, provider and public-launch blockers.",
  "currentDocs": [
    "online-safety/README.md",
    "online-safety/operations/illegal-content-response.md",
    "docs/development/image-moderation.md",
    "docs/operations/restricted-evidence-storage.md",
    "docs/operations/releases.md"
  ],
  "code": [
    "appview/internal/imagesafety",
    "appview/internal/safetyincident",
    "appview/internal/retention"
  ],
  "evidence": [
    "online-safety/operations/illegal-content-response.md#portal-onboarding-and-reporting-exercise--26-november-2026",
    "docs/changes/2026-09-16-online-safety-launch-readiness/05-implementation-plan.md#post-review-remediation",
    "docs/changes/2026-09-16-online-safety-launch-readiness/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": [
    "Record policy approvals and operational evidence in the maintained launch record and gap register.",
    "Verify hosted policy publication, restricted storage, provider readiness and operator procedures separately from local tests."
  ]
}
---

# Online-safety implementation and public-launch gates

The repository implementation covers image-safety indexing, restricted incidents/evidence, legal holds, retention, report handling, age eligibility, video shutdown, policy scaffolding and release gates. The approved scope intentionally does not add a production IWF adapter or an Ozone dependency.

Later remediation verifies current business-record/image gates, private evidence storage, bounded dead-letter behavior, append-only mutation rules and concurrent hold/deletion serialization. Local automated correctness is distinct from policy approval, deployed storage/provider configuration and readiness to launch publicly.

The final review leaves external/manual gates open and warns about policy-build publication safety. Treat its environment observations as dated evidence, not a claim about the current hosting configuration. This package remains active because the broader readiness decision requires current policy/operator review and external verification. Current operations and safety documentation must be checked before any deployment, real report transmission or approval assertion.

On 26 November 2026 the owner confirmed NCA reporting portal access. The [operator record](../../../online-safety/operations/illegal-content-response.md#portal-onboarding-and-reporting-exercise--26-november-2026) records onboarding and the remaining CSEA-01 exercise and outage-procedure checks.

Evidence: [05-implementation-plan.md](../2026-09-16-online-safety-launch-readiness/05-implementation-plan.md#post-review-remediation); [06-implementation-review.md](../2026-09-16-online-safety-launch-readiness/06-implementation-review.md). Current entry points are listed in the metadata; consult maintained guidance before applying historical rules.

The [incident guide](../../../online-safety/operations/illegal-content-response.md) consolidates day-to-day response for a solo operator. The launch approval and operator records are kept outside the repository.

On 26 November 2026 the standalone online-safety readiness checker was retired. The [release runbook](../../operations/releases.md#private-beta-and-full-public-launch) gives the current engineering checks and release process. Earlier implementation and validation references to the checker describe historical work.

The operator console supports preserving an authorised image directly from a moderation case. A linked manual evidence incident reuses restricted storage, access audit, legal holds and retention. See the [operator workflow](../../operations/moderation-operators.md#restricted-evidence-workspace).
