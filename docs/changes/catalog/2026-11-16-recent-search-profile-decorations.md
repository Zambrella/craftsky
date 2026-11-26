---
{
  "id": "2026-11-16-recent-search-profile-decorations",
  "title": "Preserve current profile decorations in recent searches",
  "source": "docs/changes/2026-11-16-recent-search-profile-decorations",
  "implementation": "implemented",
  "archive": "active",
  "components": [
    "app",
    "web-client",
    "appview"
  ],
  "topics": [
    "search",
    "profiles",
    "business",
    "customisation"
  ],
  "keywords": [
    "recentAuthor",
    "ProfileRecentSearchPayload",
    "snapshot",
    "business icon",
    "avatar colour",
    "DID hydration"
  ],
  "summary": "Shows the 10 newest searches on both clients and uses the current indexed avatar, customisation and account type for recent profile rows.",
  "currentDocs": [
    "docs/architecture.md"
  ],
  "code": [
    "app/lib/search/models/recent_search.dart",
    "app/lib/search/pages/blank_search_view.dart",
    "web-client/src/lib/features/social/search-model.ts",
    "web-client/src/lib/features/social/Search.svelte",
    "appview/internal/api/search_recent_store.go",
    "appview/internal/api/search_recent_identity_test.go"
  ],
  "evidence": [
    "docs/changes/2026-11-16-recent-search-profile-decorations/README.md"
  ],
  "supersededBy": [],
  "followUps": []
}
---

# Recent search profile decorations

Both clients show the 10 newest searches. Recent profile entries use the current
avatar, customisation and account type when history is loaded.
