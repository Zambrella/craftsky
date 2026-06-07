## Architecture Decision Record
- Status: Approved — implemented by 2026-04-23 post lexicon fields plan
- Aspect: Lexicon (atproto schemas)
- Date: 2026-04-23
- Decision: One post lexicon with an optional, extensible `project` sub-object — not separate lexicons per craft, not a free-form extensions map

### Why I needed to decide this

Craftsky has two kinds of posts: general social posts (rich text + images, like any other microblog) and craft project posts (tagged with a specific craft, carrying craft-specific fields like needle size, fabric type, block dimensions, etc.). The current `social.craftsky.feed.post` lexicon already hedges toward a single post type with an optional `project` sub-object, but the `projectDetails` shape only carries shared fields (`craftType`, `status`, `patternUrl`) and has no story for per-craft specialisation.

This decision needs to be locked in before we start writing real records to user PDSes. Lexicons are load-bearing — once users create records against a schema, changing it is painful because existing data on PDSes can't be easily migrated (see `lexicon/README.md`). The roadmap also explicitly flags "project-post field set validation with real crafters" and "pattern/material/technique taxonomy" as open questions that won't be answered until we have real users. We therefore need an extensible shape that absorbs future per-craft fields without forcing a migration, even though today we don't know what those fields will be.

The forcing question is: when we add knitting-specific, sewing-specific, or (later) woodworking-specific fields, where do they live? Is a knitting project the same record type as a sewing project, or a different one?

### Options I considered

**Option 1: One post lexicon, nested `project: {common, details?}` with open union for details — CHOSEN**

`social.craftsky.feed.post` stays as the only post record type. Its optional `project` field holds a required `common` object (shared fields, including `craftType`) and an optional `details` field that is an open union of craft-specific `#details` objects defined in separate lexicons (e.g. `social.craftsky.project.knitting#details`).

A post with no `project` is a general post. A post with `project.common` but no `project.details` is a tagged project post with no craft-specific specialisation yet. A post with both is a fully specialised project post.

- Pro: Adding a new craft = one new lexicon file + one new enum entry on `craftType` + one new ref in the union. No change to existing craft lexicons. No PDS migration for existing posts.
- Pro: Shared fields are defined once in `#projectCommon`. Renaming or adding a shared field (e.g. adding `difficulty`) is one edit, not N per-craft edits.
- Pro: Atproto unions are open — old clients rendering a post with an unknown `$type` under `details` gracefully fall back to showing the `common` part. No breakage when new crafts are added.
- Pro: Preserves the "one post record type" precedent already established in `lexicon/README.md` (comments are posts with `reply` set; quote posts are posts whose `embed` wraps a strongRef).
- Pro: Stays typed. Each craft's `#details` is a real lexicon with real validation and code-generation support — not a free-form map.
- Con: Slightly nested access at read time (`project.common.craftType` vs `project.craftType`).
- Con: When a new craft gets its own `#details` lexicon, we do need to touch `feed.post` to register the new union member — but this is additive, not a breaking change.

**Option 2: Separate top-level record types per craft**

`social.craftsky.feed.post` for general posts, plus `social.craftsky.feed.knittingProject`, `social.craftsky.feed.sewingProject`, etc. — each its own NSID, its own collection, its own firehose filter.

- Pro: Maximum per-craft freedom. Strict validation.
- Pro: Clean separation between "social content" and "project content."
- Con: Every new craft is a new collection, which means changes to the indexer (new table/columns), feed queries (union over N collections for a cross-craft timeline), and client dispatch (N record types to render).
- Con: Breaks the "one post type" precedent from the README without a forcing reason. Comments and quote posts stayed as variants of `post`; project posts should too.
- Con: Locks each craft into its own collection *now*, when the roadmap explicitly says the craft taxonomy is an open question to be validated with real crafters. Premature commitment.
- Con: Interactions (like, repost, reply) would need to work uniformly across all project types. Uniformity is free with a single record type; with N types it becomes something to maintain.

Not chosen.

**Option 3: Free-form `extensions: {}` map on `projectDetails`**

Keep `projectDetails` roughly as-is, add a schemaless `extensions` object that can hold arbitrary per-craft key/value pairs.

- Pro: Simplest possible change. No new lexicons ever needed.
- Con: Discards the point of lexicons. Lexicons exist to give records a machine-readable schema so clients, indexers, and validators can trust the shape. An untyped bag moves validation into application code.
- Con: No code generation for per-craft fields. Every consumer has to know by convention that a knitting project has `extensions.needleSize` as a string.
- Con: Two apps could disagree on the shape of the same craft's extension bag, and there's no schema to arbitrate.

Not chosen.

### What I decided

**Option 1.** One post lexicon with an optional `project` sub-object structured as `{ common: project.defs#projectCommon, details?: union<per-craft #details> }`. The reusable project wrapper/common/pattern objects live at `social.craftsky.project.defs`; per-craft `#details` lexicons live at `social.craftsky.project.<craft>`, parallel to `feed`, `actor`, `graph`.

**Why:**

- Extensibility without migration is the hard constraint (roadmap's open questions about taxonomy + atproto's "changing a lexicon is painful" reality). Option 1 adds new crafts without touching existing ones and without forcing PDS data migration.
- Shared fields need to be defined once. Option 1 puts them in `social.craftsky.project.defs#projectCommon`; Option 2 would duplicate them across N craft lexicons.
- Atproto's open-union semantics give us forward compatibility for free — old clients see new-craft posts as "generic project post, unknown details" rather than breaking.
- Preserves the architectural precedent already in the codebase (posts are one type; variants are expressed by optional sub-objects).

### Trade-offs

**Good:**
- New crafts are strictly additive: new file under `lexicon/social/craftsky/project/`, new enum value on `craftType`, new ref in the `details` union. No change to any existing craft lexicon, no change to any record already on a PDS.
- Shared fields live in one place. Future additions like `difficulty`, `estimatedHours`, `visibility` are one-file edits.
- General posts and project posts share a single collection, so feeds, indexers, likes, reposts, replies, and notifications all operate uniformly. "Show me this user's posts" is one query. "Show me this user's project posts" is the same query plus `is_project = true`.
- `craftType` can grow its `knownValues` enum independently of whether that craft has a `#details` lexicon yet. A user posting about basketweaving before we've written `social.craftsky.project.basketweaving` gets a valid tagged project post with no `details` — not an error.
- Typed per-craft schemas mean code generation still works. The Flutter client and Go AppView both get proper types for knitting-specific fields once we define them.

**Bad:**
- Two-level nesting at read time: `post.project.common.craftType` instead of `post.project.craftType`. Cosmetic, but real — every consumer of the project shape pays this cost.
- The `social.craftsky.project.defs` lexicon becomes the registry of known craft detail variants (via the `details` union). Adding a craft requires editing `project.defs`, not just publishing the new craft lexicon in isolation. This is additive and non-breaking, but it means the shared project defs evolve over time.
- `#projectCommon` will accumulate fields as we learn what "common" really means. Some field we put there today may later turn out to be craft-specific, and moving it out is the kind of change the atproto spec warns about. We mitigate by keeping `#projectCommon` conservative — only fields that are genuinely craft-independent.
- If a future craft truly cannot be expressed as "common + specialised details" (e.g. it needs a fundamentally different top-level shape), this pattern will strain. We accept the bet that most crafts fit the mould.

### Notes

- The specific field set of `#projectCommon` and of each craft's `#details` is explicitly out of scope for this ADR. Roadmap items "project-post field set validation with real crafters" and the pattern/material/technique taxonomy question remain open and will be answered iteratively, likely against real user feedback.
- **AppView commitment:** the indexer will materialise `project.common.craftType` and a boolean `is_project` into dedicated columns at index time. `craftType` is a first-class queryable dimension for feeds and search, not a free-form tag. This is an implementation detail of the AppView (no lexicon impact) but is recorded here so the commitment is explicit: future feed and search specs should assume these columns exist and should not need to revisit whether `craftType` should be indexed.
- Per-craft lexicons (`social.craftsky.project.<craft>`) define a `#details` object type only — they do not declare a `main` record. They are referenced types, not standalone records. Records still live under `social.craftsky.feed.post`.
- The existing `#projectDetails` local def inside `social/craftsky/feed/post.json` is replaced by the new `{common, details?}` structure. Since no records exist on production PDSes yet (per the user: "The existing lexicons are not in production"), this is a free edit.
- Atproto lexicon evolution rules: adding an optional field is safe; adding a new member to an open union is safe; removing or renaming fields is not. The design favours additive changes for exactly this reason.
- Related references: [lexicon/README.md](../lexicon/README.md), [atproto-craft-social-app-reference.md](../atproto-craft-social-app-reference.md) §"Lexicon Design" (flags this as the most important early decision), `docs/roadmap.md` under "Lexicons."
