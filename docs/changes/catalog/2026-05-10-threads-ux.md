---
{
  "id": "2026-05-10-threads-ux",
  "title": "Thread anchoring and reply context",
  "source": "docs/changes/2026-05-10-threads-ux",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app"
  ],
  "topics": [
    "threads",
    "navigation",
    "composer"
  ],
  "keywords": [
    "PostThread",
    "selected post",
    "ancestor context",
    "reply preview",
    "scroll position"
  ],
  "summary": "The recorded Flutter change anchors a thread on its selected post, retains ancestors above it, prevents self-navigation and shows replied-to context in the composer.",
  "currentDocs": [
    "docs/architecture.md",
    "app/README.md"
  ],
  "code": [
    "app/lib/feed/pages/post_thread_page.dart",
    "app/lib/feed/widgets/post_composer_sheet.dart"
  ],
  "evidence": [
    "docs/changes/2026-05-10-threads-ux/05-implementation-plan.md#final-verification"
  ],
  "supersededBy": [],
  "followUps": [
    "Historical MAN-001 scroll-jank and MAN-002 theme-fit checks were not executed."
  ]
}
---

# Thread anchoring and reply context

## Outcome and decisions

The recorded Flutter change anchors a thread on its selected post, retains ancestors above it, prevents self-navigation and shows replied-to context in the composer. The slice deliberately preserves the AppView thread response and nested reply references. Layout and navigation tests cover narrow widths and enlarged text; the selected post is the reading anchor rather than the first ancestor.

## Evidence and limits

The execution record reports focused thread/composer and broader feed tests passing, with its completion checklist closed. Visual scroll-jank and theme-fit checks were not executed in that pass.

Historical evidence: [05-implementation-plan.md — Final Verification](../../../docs/changes/2026-05-10-threads-ux/05-implementation-plan.md#final-verification).

## Current entry points

[docs/architecture.md](../../../docs/architecture.md); [app/README.md](../../../app/README.md). Recorded follow-ups are historical evidence boundaries; recheck them against later records and current code before treating them as open defects.
