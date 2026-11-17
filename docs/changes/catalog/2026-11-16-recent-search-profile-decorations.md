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
  "summary": "Keeps current server-hydrated customisation and account type when displaying recent profile searches on both clients, and applies the selected accent to web recent/autocomplete avatars.",
  "currentDocs": [
    "docs/architecture.md"
  ],
  "code": [
    "app/lib/search/models/recent_search.dart",
    "app/lib/search/pages/blank_search_view.dart",
    "web-client/src/lib/features/social/search-model.ts",
    "web-client/src/lib/features/social/Search.svelte",
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

Both clients retain current server-provided identity decorations rather than
freezing them into private history. Existing entries render correctly on reload.
