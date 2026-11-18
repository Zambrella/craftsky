---
{
  "id": "2026-05-12-reply-comment-section",
  "title": "Reply and comment-section replacement",
  "source": "docs/changes/2026-05-12-reply-comment-section",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app",
    "appview"
  ],
  "topics": [
    "comments",
    "navigation",
    "pagination"
  ],
  "keywords": [
    "comments",
    "thread",
    "focused reply",
    "flattened descendants",
    "PostCommentSection"
  ],
  "summary": "Replaced the active thread surface with focused, paginated comment branches and two visible reply levels.",
  "currentDocs": [
    "docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md",
    "app/README.md"
  ],
  "code": [
    "appview/internal/api",
    "app/lib/feed"
  ],
  "evidence": [
    "docs/changes/2026-05-12-reply-comment-section/05-implementation-plan.md#follow-up-final-verification",
    "docs/changes/2026-05-12-reply-comment-section/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": [
    "OS-level notification/deep-link delivery was a manual gap in this package."
  ]
}
---

# Reply and comment-section replacement

The historical implementation replaces `/thread` use with a root comment section, branch continuation, comment sorting and reply parent/root preservation. A focused reply can promote its branch even when the target is outside the first reply page. Recursive descendants are deliberately flattened into the visual branch; the earlier direct-only interpretation was corrected in the requirements.

The execution record includes the off-page focused-reply fix and the final review accepts that correction. Backend and focused Flutter model/provider/widget/router tests are recorded as passing, without Lexicon changes. This evidence does not establish OS push/deep-link delivery or the visual clarity of the deliberately stubbed follows sort. Use this package for focus and branch ancestry reasoning, then check current comment queries and routing.

Evidence: [05-implementation-plan.md](../2026-05-12-reply-comment-section/05-implementation-plan.md#follow-up-final-verification); [06-implementation-review.md](../2026-05-12-reply-comment-section/06-implementation-review.md). Current entry points are listed in the metadata; consult maintained guidance before applying historical rules.
