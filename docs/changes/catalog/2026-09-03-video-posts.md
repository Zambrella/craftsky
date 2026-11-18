---
{
  "id": "2026-09-03-video-posts",
  "title": "Video post upload and playback foundation",
  "source": "docs/changes/2026-09-03-video-posts",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app",
    "appview",
    "lexicon"
  ],
  "topics": [
    "video",
    "media",
    "publication"
  ],
  "keywords": [
    "processing poll",
    "service JWT",
    "app interruption",
    "video verification",
    "MAN-001"
  ],
  "summary": "Added video publication with constrained upload authorization, processing recovery and explicit platform release gates.",
  "currentDocs": [
    "docs/adr/012-ephemeral-video-service-token-handoff.md",
    "docs/adr/024-bunny-video-development-integration.md",
    "docs/development/bunny-video.md"
  ],
  "code": [
    "appview/internal/video",
    "app/lib/feed"
  ],
  "evidence": [
    "docs/changes/2026-09-03-video-posts/05-implementation-plan.md",
    "docs/changes/2026-09-03-video-posts/06-implementation-review.md#test-evidence"
  ],
  "supersededBy": [],
  "followUps": [
    "Complete or consult current evidence for MAN-001–MAN-005 live platform/video release checks."
  ]
}
---

# Video post upload and playback foundation

The video foundation adds video-bearing post validation, upload authorization, processing/polling, final verification and playback. The original provider path uses the narrowly scoped service-JWT exception: ephemeral upload authority stays in memory, while record publication and verification remain AppView-mediated.

The final correction stops authorization-stage work on app interruption without broadening the approved user-cancellation controls, which are available during upload and processing. The review accepts the automated implementation after correction of lifecycle, verification and presentation findings.

MAN-001 through MAN-005 remain explicit checks requiring supported platforms, assistive technology, device conditions and an eligible live account. Their absence does not mean local implementation is missing, but it prevents a full release-readiness claim. Bunny video was introduced later; today's provider selection and upload policy are defined by the current video guide and ADRs, not by assuming this original provider remains selected.

Evidence: [05-implementation-plan.md](../2026-09-03-video-posts/05-implementation-plan.md); [06-implementation-review.md](../2026-09-03-video-posts/06-implementation-review.md#test-evidence). Current entry points are listed in the metadata; consult maintained guidance before applying historical rules.
