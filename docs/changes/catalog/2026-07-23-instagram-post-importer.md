---
{
  "id": "2026-07-23-instagram-post-importer",
  "title": "Retired standalone Instagram archive importer",
  "source": "docs/changes/2026-07-23-instagram-post-importer",
  "implementation": "superseded",
  "archive": "archived",
  "components": [
    "app",
    "appview",
    "web-client"
  ],
  "topics": [
    "instagram",
    "imports",
    "provenance"
  ],
  "keywords": [
    "retired importer website",
    "Instagram ZIP archive",
    "externalImport",
    "ADR 008",
    "ownership verification",
    "historical dates"
  ],
  "summary": "This is the historical design and verification record for the standalone Instagram archive-import website, which was removed on 13 November 2026 after client implementation.",
  "currentDocs": [
    "web-client/README.md",
    "docs/adr/008-instagram-import-provenance.md",
    "docs/architecture.md"
  ],
  "code": [
    "web-client/src/lib/features",
    "appview/internal/instagram"
  ],
  "evidence": [
    "docs/changes/2026-07-23-instagram-post-importer/05-implementation-plan.md#6-verification-evidence",
    "docs/changes/2026-07-23-instagram-post-importer/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": []
}
---

# Retired standalone Instagram archive importer

## Outcome and decisions

This is the historical design and verification record for the standalone Instagram archive-import website, which was removed on 13 November 2026 after client implementation. The standalone React/Pages commands and deployment assumptions are retired. The enduring public-record provenance decision remains ADR 008; current import UI and verified ownership controls belong to the web client and AppView. Historical export contents and source identities remain private.

## Evidence and limits

The preserved review explicitly marks the website retired. Its original approval had external OAuth/PDS/export/browser/firehose gates and did not establish production publication. Those old runtime paths must not be used as current setup instructions.

Historical evidence: [05-implementation-plan.md — 6. Verification Evidence](../../../docs/changes/2026-07-23-instagram-post-importer/05-implementation-plan.md#6-verification-evidence); [06-implementation-review.md](../../../docs/changes/2026-07-23-instagram-post-importer/06-implementation-review.md).

## Current entry points

[web-client/README.md](../../../web-client/README.md); [docs/adr/008-instagram-import-provenance.md](../../../docs/adr/008-instagram-import-provenance.md); [docs/architecture.md](../../../docs/architecture.md). This card preserves the recorded change outcome; it does not replace current code or maintained guidance.
