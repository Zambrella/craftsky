---
{
  "id": "2026-07-11-instagram-dm-verification",
  "title": "Instagram ownership verification and private follow suggestions",
  "source": "docs/changes/2026-07-11-instagram-dm-verification",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app",
    "appview"
  ],
  "topics": [
    "instagram",
    "verification",
    "relationships"
  ],
  "keywords": [
    "DM challenge",
    "private suggestions",
    "AV-007",
    "automatic follow",
    "verified ownership"
  ],
  "summary": "Records Instagram DM ownership verification and its later replacement of background follows with private member-accepted suggestions.",
  "currentDocs": [
    "docs/adr/008-instagram-import-provenance.md",
    "docs/architecture.md",
    "docs/development/logging-error-reporting.md"
  ],
  "code": [
    "appview/internal/instagram",
    "appview/internal/integrations/instagrammeta",
    "app/lib/instagram_migration"
  ],
  "evidence": [
    "docs/changes/2026-07-11-instagram-dm-verification/05-implementation-plan.md#verification-evidence",
    "docs/changes/2026-07-11-instagram-dm-verification/06-implementation-review.md#appview-audit-strict-branch-final-re-review-2026-08-20",
    "docs/changes/2026-07-11-instagram-dm-verification/06-implementation-review.md#ir-021-resolution-2026-08-21"
  ],
  "supersededBy": [],
  "followUps": [
    "Consult current deployment records for production Meta/webhook and eligible-account verification."
  ]
}
---

# Instagram ownership verification and private follow suggestions

The original package explored official Instagram DM challenges, durable webhook work, verified mappings and automatic follow matching. Its later strict-branch correction is essential: background public follow writes were removed, and matching now produces private suggestions that a current member must explicitly accept. Historical automatic-follow requirements are retained as history, not current authority.

Subsequent reconciliation fixes evaluate the job owner before loading candidates, including zero-candidate jobs, and independently verify the other party. Membership/dependency failures retain durable retries; departure inactivates mappings/import work without PDS writes. Match-notification restoration is distinct from reintroducing automatic follows.

The package contains multiple reviews and amendments. The strict-branch final re-review and notification-restoration sections supersede earlier blockers, while a terminology finding is explicitly closed as superseded. Production Meta permissions/delivery and device/provider checks remain external evidence boundaries; local tests cannot establish those integrations.

Evidence: [05-implementation-plan.md](../2026-07-11-instagram-dm-verification/05-implementation-plan.md#verification-evidence); [06-implementation-review.md](../2026-07-11-instagram-dm-verification/06-implementation-review.md#appview-audit-strict-branch-final-re-review-2026-08-20). Current entry points are listed in the metadata; consult maintained guidance before applying historical rules.

Section pointers: [ir-021-resolution-2026-08-21](../2026-07-11-instagram-dm-verification/06-implementation-review.md#ir-021-resolution-2026-08-21).
