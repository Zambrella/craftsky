## Architecture Decision Record
- Status: Approved
- Aspect: Lexicon (atproto schemas), AppView indexing, notifications
- Date: 2026-06-12
- Decision: Pattern metadata text fields carry field-scoped rich-text facets

### Why I needed to decide this

Project posts can credit a pattern, designer, and publisher. The Flutter composer
already offers hashtag autocomplete for the pattern name/tag field and mention
autocomplete for designer and publisher, but before this decision those tokens
were saved as plain strings only. That meant the visible text could not drive
mention notifications and pattern hashtags were not reliably materialized into
search tags.

The records are not in production yet, so breaking lexicon changes are allowed.
This is the cheapest point to make the PDS record shape carry the semantics we
expect AppViews and future clients to use.

### Options I considered

**Option 1: Keep plain strings only**

- Pro: No schema change.
- Con: Mentions and tags in pattern metadata remain UI-only. The AppView would
  need to infer semantics from raw text, which is fragile and inconsistent with
  AT Protocol's existing rich-text facet model.

Not chosen.

**Option 2: Replace designer/publisher strings with account DIDs**

- Pro: Account references are easy to index and notify.
- Con: Pattern credits often refer to non-Craftsky people, companies, physical
  pattern publishers, historical entities, or multiple designers. Replacing the
  visible credit with DIDs would lose necessary free-text expressiveness.

Not chosen.

**Option 3: Add one shared `patternFacets` array**

- Pro: Small schema addition.
- Con: `app.bsky.richtext.facet` byte ranges are scoped to one specific string.
  A shared facet array would need another layer of field identifiers or offset
  concatenation rules, neither of which matches the reused facet type.

Not chosen.

**Option 4: Add field-scoped facet arrays beside the strings**

Add optional `nameFacets`, `designerFacets`, and `publisherFacets` arrays to
`social.craftsky.project.defs#pattern`, each using `app.bsky.richtext.facet` and
scoped to its sibling string.

- Pro: Preserves plain-text display and offline/historical credit use cases.
- Pro: Reuses the standard rich-text facet shape for mentions and tags.
- Pro: Lets the AppView materialize hashtag search and mention notifications
  without guessing from raw text.
- Con: Adds parallel fields that clients must keep in sync with their sibling
  strings.

Chosen.

### What I decided

Pattern metadata remains free text, with optional field-scoped facet arrays:

- `pattern.name` plus `pattern.nameFacets`
- `pattern.designer` plus `pattern.designerFacets`
- `pattern.publisher` plus `pattern.publisherFacets`

The AppView indexes hashtag features from the top-level post text facets, the
structured `project.common.tags` field, and all pattern facet arrays into the
searchable tag columns. It also materializes mention features from top-level text
facets and pattern facet arrays into a post-mentions table used by notifications.

Mention notifications are emitted for metadata mentions. They are deduplicated
per `(post, mentioned account)`, self-mentions do not notify, and direct reply
notifications remain the more specific notification when applicable.

### Consequences

- Composer clients must generate facets over each individual pattern field, not
  over a combined pattern string.
- `project.common.tags` remains useful for structured user/client tags, but the
  AppView search index is the complete searchable union.
- Future clients that ignore the facet arrays still produce valid records; they
  just do not create metadata-driven tag search or mention notifications.
- Adding these optional fields is lexicon-compatible in production, but this ADR
  treats the change as pre-production and updates clients and indexers in one
  pass.
