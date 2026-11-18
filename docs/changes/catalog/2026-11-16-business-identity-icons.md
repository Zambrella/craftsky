---
{
  "id": "2026-11-16-business-identity-icons",
  "title": "Business storefront before profile names",
  "source": "docs/changes/2026-11-16-business-identity-icons",
  "implementation": "implemented",
  "archive": "active",
  "components": ["app", "web-client"],
  "topics": ["business", "profiles", "identity"],
  "keywords": ["Business account", "business identity icon", "storefront before name", "storefront optical sizing", "Business details icon parity", "accountType", "CompactProfileIdentity", "CompactIdentity", "ProfileCraftChips", "PostAuthor", "ProfileAccountSummary", "ProfileSearchResult", "craft icon overflow"],
  "summary": "Shows a visually size-matched storefront in the profile owner's theme colour before profile names, removes the separate web account label, and carries existing accountType metadata through native identity models.",
  "currentDocs": ["docs/development/web-client-parity.md"],
  "code": [
    "app/lib/profile/widgets/compact_profile_identity.dart",
    "app/lib/shared/widgets/business_identity_icon.dart",
    "app/lib/profile/widgets/profile_business_hero.dart",
    "web-client/src/lib/features/social/BusinessIdentityIcon.svelte",
    "app/lib/profile/widgets/profile_craft_chips.dart",
    "app/lib/profile/widgets/profile_sliver_app_bar.dart",
    "app/lib/feed/models/post.dart",
    "app/lib/profile/models/profile_account_summary.dart",
    "app/lib/search/models/profile_search_page.dart",
    "web-client/src/lib/features/social/compact-identity.ts",
    "web-client/src/lib/features/social/CompactIdentity.svelte",
    "web-client/src/lib/PostCard.svelte",
    "web-client/src/lib/features/social/Profile.svelte"
  ],
  "evidence": ["docs/changes/2026-11-16-business-identity-icons/README.md"],
  "supersededBy": [],
  "followUps": []
}
---

# Business storefront before profile names

The server's business account type is shown through the shared craft/icon rows
on both clients, with accessible labels and the existing compact overflow rule.
