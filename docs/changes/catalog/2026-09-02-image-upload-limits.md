---
{
  "id": "2026-09-02-image-upload-limits",
  "title": "Shared image admission and preparation limits",
  "source": "docs/changes/2026-09-02-image-upload-limits",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app",
    "appview",
    "lexicon"
  ],
  "topics": [
    "media",
    "images",
    "resource-limits"
  ],
  "keywords": [
    "header-first",
    "20:1 output ratio",
    "4000 pixels",
    "serialized image preparation",
    "ADR 011"
  ],
  "summary": "Aligned image limits and header-first admission across mobile, AppView and the then-existing importer.",
  "currentDocs": [
    "docs/adr/011-image-upload-size-and-dimension-limits.md",
    "docs/adr/017-direct-image-upload-service-token-handoff.md"
  ],
  "code": [
    "appview/internal/imageupload",
    "app/lib/shared/image",
    "web-client/src/lib/features/composer/media.ts"
  ],
  "evidence": [
    "docs/changes/2026-09-02-image-upload-limits/05-implementation-plan.md#correction-pass",
    "docs/changes/2026-09-02-image-upload-limits/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": []
}
---

# Shared image admission and preparation limits

The image-limit correction makes source inspection header-first and serializes memory-heavy preparation. Shared authoring rules bound encoded output, aspect ratio and dimensions; the historical importer sanitizer resizes before returning, including highly compressible oversized cases. The final review records closure of the source-inspection, concurrency, ratio and output-bound findings.

ADR 011 is updated to describe the actual direct-PDS validation boundary rather than promising server-side inspection of bytes that never pass through AppView. Immediate image upload capability remains narrow and does not hand general OAuth credentials to clients.

Recorded tests cover exact boundaries and resize regressions for JPEG, PNG and WebP. The old importer website has since been removed; its historical execution evidence is retained, not a request to recreate it. Consult current ADRs and each client's preparation code before treating these records as proof that browser decoded-memory behavior is bounded identically.

Evidence: [05-implementation-plan.md](../2026-09-02-image-upload-limits/05-implementation-plan.md#correction-pass); [06-implementation-review.md](../2026-09-02-image-upload-limits/06-implementation-review.md). Current entry points are listed in the metadata; consult maintained guidance before applying historical rules.
