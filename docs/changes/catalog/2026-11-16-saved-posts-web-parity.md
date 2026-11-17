---
{
  "id": "2026-11-16-saved-posts-web-parity",
  "title": "Saved posts web and mobile parity",
  "source": "docs/changes/2026-11-16-saved-posts-web-parity",
  "implementation": "implemented",
  "archive": "active",
  "components": ["web-client"],
  "topics": ["saved-posts", "parity", "folders"],
  "keywords": ["Saved.svelte", "PostSummary", "MemberPost", "Unfiled", "No folder", "protected saves", "saved reply focus", "Unicode folder names"],
  "summary": "Aligns web saved pages, summaries, folder dialogs and save/move flows with Flutter, including protected-save actions and access-aware loading.",
  "currentDocs": ["docs/development/web-client-parity.md"],
  "code": ["web-client/src/lib/features/social/Saved.svelte", "web-client/src/lib/features/social/MemberPost.svelte", "web-client/src/lib/features/social/PostSummary.svelte", "web-client/src/lib/features/social/saved-model.ts", "web-client/tests/authenticated-saved-parity.spec.ts"],
  "evidence": ["docs/changes/2026-11-16-saved-posts-web-parity/README.md"],
  "supersededBy": [],
  "followUps": []
}
---

# Saved posts web parity

Records the implemented mobile-reference layout, folder and summary behavior,
loading/error fixes and local browser verification. Flutter remains unchanged;
the change uses the existing private saved-post APIs and Plus gate.
