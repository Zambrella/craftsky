---
{
  "id": "2026-05-19-appview-image-blobs",
  "title": "AppView image blobs and post image metadata",
  "source": "docs/changes/2026-05-19-appview-image-blobs",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "appview",
    "lexicon"
  ],
  "topics": [
    "media",
    "images",
    "validation"
  ],
  "keywords": [
    "blob",
    "UploadBlob",
    "aspectRatio",
    "missing blob metadata",
    "images"
  ],
  "summary": "Added AppView-mediated image upload and validated public post-image metadata, with a recorded post-review validation fix.",
  "currentDocs": [
    "docs/adr/011-image-upload-size-and-dimension-limits.md",
    "docs/adr/017-direct-image-upload-service-token-handoff.md"
  ],
  "code": [
    "appview/internal/imageupload",
    "appview/internal/api/blob.go"
  ],
  "evidence": [
    "docs/changes/2026-05-19-appview-image-blobs/05-implementation-plan.md#remediation-verification",
    "docs/changes/2026-05-19-appview-image-blobs/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": [
    "Live PDS/public-media smoke evidence was outside this package."
  ]
}
---

# AppView image blobs and post image metadata

This slice introduced the authenticated image-blob endpoint, PDS upload adapter, upload MIME/size checks, top-level image records and image aspect-ratio metadata through indexing and API responses. Text-only posts remained supported; video was outside this scope.

The first review found that nonempty but malformed blob objects could pass create-post validation. The later implementation-plan remediation records a failing malformed-metadata regression, validation before PDS writes, passing focused packages and regenerated-Lexicon checks. The original Changes required verdict is therefore historical correction input, not a fresh report that this defect remains present.

The original size limits and fully mediated upload path have evolved. Follow ADRs 011 and 017 and current source for authoring limits and the narrow direct-image-upload exception. This package's automated checks do not prove live public-media retrieval.

Evidence: [05-implementation-plan.md](../2026-05-19-appview-image-blobs/05-implementation-plan.md#remediation-verification); [06-implementation-review.md](../2026-05-19-appview-image-blobs/06-implementation-review.md). Current entry points are listed in the metadata; consult maintained guidance before applying historical rules.
