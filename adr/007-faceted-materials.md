## Architecture Decision Record
- Status: Approved
- Aspect: Lexicon (atproto schemas), AppView indexing, notifications, Flutter composer
- Date: 2026-06-16
- Decision: Project materials are faceted rich-text entries

### Why I needed to decide this

Project materials were stored as plain strings. That worked for simple searchable
labels like "linen", but it could not preserve the semantics users expect from a
material note such as "3m of @sewmesunshine.bsky.social #viscose fabric". The
visible text should remain flexible, while mentions and hashtags should be stored
as standard AT Protocol rich-text facets rather than inferred later from raw text.

The records are not in production, so a breaking lexicon rewrite is acceptable.
This is the cheapest point to make the record shape match the intended composer
experience.

### Options I considered

**Option 1: Keep `materials` as `array<string>`**

- Pro: No schema or model rewrite.
- Con: Mentions and hashtags in materials stay UI-only or require fragile
  server-side parsing without byte-range annotations.

Not chosen.

**Option 2: Add one parallel `materialFacets` array**

- Pro: Keeps the existing string array.
- Con: Facet byte ranges are scoped to one string. A shared facet array would
  need positional coupling to `materials[]` and would be easy for clients to get
  out of sync.

Not chosen.

**Option 3: Replace materials with structured quantity/vendor/tag fields**

- Pro: Queryable dimensions would be explicit.
- Con: Material structure varies too much across crafts. Fabric, yarn, thread,
  batting, floss, and other supplies do not share one useful cross-craft object
  shape beyond visible text.

Not chosen.

**Option 4: Make each material a visible text object with scoped facets**

Use `materials: array<#material>`, where each material has required `text` and
optional `facets` over that exact text.

- Pro: Preserves free-form material descriptions.
- Pro: Reuses `app.bsky.richtext.facet` for mentions and hashtags.
- Pro: Lets the AppView materialize searchable material text, hashtag tags, and
  mention notifications without guessing from raw text.
- Con: Clients must generate facets per material entry and keep them scoped to
  the sibling `text` field.

Chosen.

### What I decided

`project.common.materials` is now a list of material objects:

- `text` is the visible material entry.
- `facets` is an optional list of `app.bsky.richtext.facet` annotations over
  `text`.
- The list is capped at 10 entries.
- Each entry is capped at 100 graphemes and 1000 bytes.

The AppView continues to materialize the `materials TEXT[]` column from the
material `text` values. It also indexes hashtag features from material facets
into the searchable tag columns and material mention facets into the post
mentions table used by notifications.

### Consequences

- Composer clients generate facets independently for each material entry.
- The Flutter composer adds one material at a time, clears the input after Add,
  and keeps the material field focused.
- Future clients that omit facets still produce valid material entries; those
  entries remain searchable by material text but do not create material-driven
  hashtag search or mention notifications.
- Because this is a pre-production breaking change, old string material entries
  are not supported as a compatibility path.
