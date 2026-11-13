---
{
  "id": "2026-05-20-flutter-image-posts",
  "title": "Initial Flutter image posting and gallery",
  "source": "docs/changes/2026-05-20-flutter-image-posts",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app"
  ],
  "topics": [
    "media",
    "composer",
    "privacy"
  ],
  "keywords": [
    "image metadata",
    "WebP",
    "alt text",
    "image upload progress",
    "gallery",
    "selection cap"
  ],
  "summary": "The execution record implements mobile image selection/preparation, upload progress, post-card images and a fullscreen gallery for the initial image-posting slice.",
  "currentDocs": [
    "app/README.md",
    "docs/architecture.md",
    "docs/adr/005-optional-post-image-alt-text.md",
    "docs/adr/017-direct-image-upload-service-token-handoff.md"
  ],
  "code": [
    "app/lib/feed/media",
    "app/lib/feed/widgets/post_image_gallery.dart"
  ],
  "evidence": [
    "docs/changes/2026-05-20-flutter-image-posts/05-implementation-plan.md#remediation-pass-3-verification",
    "docs/changes/2026-05-20-flutter-image-posts/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": [
    "Historical MAN-001 through MAN-006 include real-photo metadata and physical-device checks."
  ]
}
---

# Initial Flutter image posting and gallery

## Outcome and decisions

The execution record implements mobile image selection/preparation, upload progress, post-card images and a fullscreen gallery for the initial image-posting slice. Its final correction rejects WebP when metadata stripping cannot be demonstrated and rejects partial selections that would exceed the draft cap. Early required-alt-text and upload architecture are historical; consult current media ADRs and guidance before reusing those constraints.

## Evidence and limits

The review file retains a Changes required snapshot, but later execution entries R10–R12 and Remediation Pass 3 record corrective tests and 120 passing related tests. All six physical/manual checks remained unexecuted, with no approved deferral recorded in that package.

Historical evidence: [05-implementation-plan.md — Remediation Pass 3 Verification](../../../docs/changes/2026-05-20-flutter-image-posts/05-implementation-plan.md#remediation-pass-3-verification); [06-implementation-review.md](../../../docs/changes/2026-05-20-flutter-image-posts/06-implementation-review.md).

## Current entry points

[app/README.md](../../../app/README.md); [docs/architecture.md](../../../docs/architecture.md); [docs/adr/005-optional-post-image-alt-text.md](../../../docs/adr/005-optional-post-image-alt-text.md); [docs/adr/017-direct-image-upload-service-token-handoff.md](../../../docs/adr/017-direct-image-upload-service-token-handoff.md). Recorded follow-ups are historical evidence boundaries; recheck them against later records and current code before treating them as open defects.
