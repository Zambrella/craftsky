---
{
  "id": "2026-07-29-post-languages",
  "title": "Post languages and private language preferences",
  "source": "docs/changes/2026-07-29-post-languages",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app",
    "appview"
  ],
  "topics": [
    "languages",
    "feeds",
    "search",
    "preferences"
  ],
  "keywords": [
    "BCP47",
    "langs",
    "craftsky_posts_langs_gin",
    "base language",
    "184 tags"
  ],
  "summary": "Added public post-language tags and private account filters applied before eligible browse pagination.",
  "currentDocs": [
    "docs/development/home-feeds.md",
    "docs/development/search-fields.md"
  ],
  "code": [
    "appview/internal/languages",
    "app/lib/languages"
  ],
  "evidence": [
    "docs/changes/2026-07-29-post-languages/05-implementation-plan.md#implementation-review-correction-pass-2026-07-30",
    "docs/changes/2026-07-29-post-languages/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": []
}
---

# Post languages and private language preferences

Post language tags travel with public PDS records; account language preferences remain private AppView state. App-language selection is device-local and, in this slice, English-only. Server-authoritative language filtering occurs before pagination on the eight scoped browsing/discovery surfaces, while deliberate contextual reads retain documented exceptions.

The approved review follows a correction pass that exercises the 20,000-row PostgreSQL plan contract, actual visibility predicate, private preference primary-key lookup and language GIN index. Flutter supplies friendly English names for the pinned 184-tag Bluesky snapshot and preserves valid unknown/future tags with raw fallback.

The implementation includes preference persistence, response propagation, settings/composer behavior and test-harness corrections for newly required dependencies. Historical catalogue counts describe the pinned snapshot rather than a rule to reject future valid tags. Use current language code and home-feed guidance when assessing policy exceptions or changing browse eligibility.

Evidence: [05-implementation-plan.md](../2026-07-29-post-languages/05-implementation-plan.md#implementation-review-correction-pass-2026-07-30); [06-implementation-review.md](../2026-07-29-post-languages/06-implementation-review.md). Current entry points are listed in the metadata; consult maintained guidance before applying historical rules.
