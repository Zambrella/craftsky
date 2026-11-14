---
{
  "id": "2026-10-29-basic-image-editing",
  "title": "Crop, rotation and avatar image editing",
  "source": "docs/changes/2026-10-29-basic-image-editing",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app"
  ],
  "topics": [
    "media",
    "images",
    "editing",
    "lifecycle"
  ],
  "keywords": [
    "crop rotate",
    "avatar temporary draft",
    "busy state",
    "image editor",
    "IR-001"
  ],
  "summary": "Added crop/rotate controls and avatar editing while fencing stale temporary image operations.",
  "currentDocs": [
    "docs/adr/011-image-upload-size-and-dimension-limits.md",
    "app/README.md"
  ],
  "code": [
    "app/lib/shared/image",
    "app/lib/profile",
    "app/lib/onboarding"
  ],
  "evidence": [
    "docs/changes/2026-10-29-basic-image-editing/05-implementation-plan.md#ir-001-correction-completion",
    "docs/changes/2026-10-29-basic-image-editing/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": [
    "Manual crop/export/platform and representative device checks remain separate from the recorded widget regressions."
  ]
}
---

# Crop, rotation and avatar image editing

Image editing adds the approved crop/rotation and avatar workflow while trusting the selected package for geometry/export/platform behavior. It reuses existing image preparation/publication boundaries rather than introducing another image editor framework or new integration-test harness.

A re-review accepts correction of stale avatar operations: their matching temporary busy state is released, an already accepted avatar and unrelated form changes survive, and disposed forms cannot receive completion updates. The execution record names the observable widget regressions and focused correction gates; an earlier full-suite result is explicitly historical rather than claimed as rerun after that fix.

The maintainer's decision not to add new integration tests remains part of the scope. Manual package/platform checks and optional presentation polish were not proven by unit/widget success. Current image-limit ADRs and current shared image utilities govern authoring limits and data handling beyond this UI slice.

Evidence: [05-implementation-plan.md](../2026-10-29-basic-image-editing/05-implementation-plan.md#ir-001-correction-completion); [06-implementation-review.md](../2026-10-29-basic-image-editing/06-implementation-review.md). Current entry points are listed in the metadata; consult maintained guidance before applying historical rules.
