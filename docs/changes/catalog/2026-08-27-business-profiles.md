---
{
  "id": "2026-08-27-business-profiles",
  "title": "Business profiles and products",
  "source": "docs/changes/2026-08-27-business-profiles",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "appview",
    "lexicon"
  ],
  "topics": [
    "business",
    "profiles",
    "taxonomy"
  ],
  "keywords": [
    "business.profile/self",
    "product images",
    "country currency",
    "golang.org/x/text",
    "ADR 010"
  ],
  "summary": "Implemented business-profile records, indexed product metadata and package-backed country/currency catalogues.",
  "currentDocs": [
    "docs/adr/010-business-profile-event-lexicons-and-pinned-data.md",
    "docs/architecture.md"
  ],
  "code": [
    "appview/internal/business",
    "appview/internal/api/business_profile.go"
  ],
  "evidence": [
    "docs/changes/2026-08-27-business-profiles/05-implementation-plan.md#package-backed-catalog-simplification",
    "docs/changes/2026-08-27-business-profiles/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": []
}
---

# Business profiles and products

The AppView business slice defines public business-profile/product record handling, authoritative writes, indexing, deletion and lifecycle behavior. Business metadata remains attached to the published record and visibility/index policy rather than a private client-only model. Current ADR 010 describes the record boundary and field intent.

Multiple review rounds close contract, validation, migration and lifecycle findings. The final review explicitly treats an earlier deletion-related finding as covered by the existing release migration/deletion-order/registry/retry suites, rather than requiring duplicate tests. A later catalogue simplification delegates countries and currencies to the pinned `golang.org/x/text` package and removes custom generator snapshots.

The record reports a clean release-equivalent gate after that simplification. This package does not establish that every production resource was deployed or every future catalogue remains unchanged. Later business-save recovery and subscription access changes extend the behavior; consult those current boundaries before replaying the original full-replacement editor assumptions.

Evidence: [05-implementation-plan.md](../2026-08-27-business-profiles/05-implementation-plan.md#package-backed-catalog-simplification); [06-implementation-review.md](../2026-08-27-business-profiles/06-implementation-review.md). Current entry points are listed in the metadata; consult maintained guidance before applying historical rules.
