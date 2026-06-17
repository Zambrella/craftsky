# Post Lexicon Fields Design

- **Status:** Draft
- **Date:** 2026-04-23
- **Related:** [ADR 001 — post lexicon project extensibility](../../../adr/001-post-lexicon-project-extensibility.md)

## Summary

Implement the field set for `social.craftsky.feed.post` and its sub-objects, following the shape locked in by ADR 001 (one post lexicon, optional `project` sub-object, `{common, details?}` structure with open union on `details`). Covers the top-level post shape (Level A), the cross-craft `#projectCommon` object (Level B), and a first per-craft `#details` lexicon for sewing (Level C) as a worked example.

## Goals

1. Lock in the full field set of `social.craftsky.feed.post` and `#projectCommon` before the first real PDS writes, so crafters creating records under v1 don't have their data stranded by schema changes.
2. Prove the `{common, details?}` pattern from ADR 001 works end-to-end by defining one craft's `#details` lexicon (sewing).
3. Optimise the field set for **discoverability** — search, tag filtering, craft/material/pattern browsing — since this is the primary reason for adding structured fields over free-text prose.
4. Favour minimalism wherever safe. Adding optional fields later is allowed under atproto evolution rules; removing or tightening fields is not. When in doubt, leave it out.

## Non-goals

- Per-craft `#details` lexicons beyond sewing. Knitting, crochet, quilting, embroidery, etc. will each get their own spec when designed. The sewing lexicon is a worked example proving the pattern; it is not the template for every craft.
- The AppView indexer schema. This spec names which fields the indexer is expected to materialise as queryable columns, but the exact SQL schema is a separate AppView spec.
- Cross-post "project identity" (e.g. grouping WIP posts under a finished-project post). Each project post is a standalone snapshot; relationships between posts about the same project are user-maintained via text and links. Confirmed during brainstorming.
- Rich pattern metadata beyond `{url, name, difficulty}`. Designer name, publisher, pattern version, size range, pattern PDF hosting — all deferred.
- Structured garment sizing, body-measurement references, or finished-dimensions fields on sewing projects. Deferred pending real user feedback.
- Composer UI implementation details beyond naming the help-text hints and merge rules the composer is expected to implement.

## Context

Craftsky's posts come in two flavours:

- **General posts.** Text + images + optional quote/reply. Like any social-media post.
- **Project posts.** Same as above, plus a `project` sub-object that tags the post with a craft and carries structured metadata (pattern, materials, tags, etc.) to make it findable.

ADR 001 settled the *shape* of this — one `social.craftsky.feed.post` record type, optional `project: {common, details?}` with an open union for craft-specific details. This spec settles the *fields*. The current lexicon file (`lexicon/social/craftsky/feed/post.json`) is a skeleton: its `#projectDetails` has only `craftType`, `status`, `patternUrl`, and its text length, image size, and tag/material/duration fields are absent or placeholder. None of this is in production yet — the user explicitly confirmed "the existing lexicons are not in production" — so this is free to rewrite rather than evolve.

### Discoverability framing

A recurring question during brainstorming was "does this field belong in the schema, or should it live in post text?" The governing answer is **discoverability**: structured fields let the AppView index and filter, while prose in `text` is only searchable via full-text search. Anything a crafter might want to filter on (craft, status, pattern, materials, tags, sewing sub-type) gets a structured field. Things that are descriptive but not filterable (exactly *why* you picked this yarn, how you modified the pattern, what went wrong on row 47) stay in prose.

### Snapshot model

Project posts are snapshots — typically of a finished project, occasionally of a WIP — and are immutable in practice (edits only for typos, via atproto `put`). There is no cross-post project record. This shapes several decisions:

- `status` captures state *at the moment of posting* (wip/finished), not a mutable lifecycle flag.
- `duration` is a free-text description ("3 weeks", "a long weekend"), not a structured start/end datetime pair. Crafters describe duration informally; forcing precision would invite inaccuracy and unused fields.
- `createdAt` is sufficient as the only datetime — it already records when the snapshot happened.

## Design

### File layout

```
lexicon/social/craftsky/
├── feed/
│   ├── post.json          (MODIFIED — shape change + new fields)
│   └── defs.json          (NEW — shared tokens: craftType, status, difficulty)
└── project/
    ├── sewing.json        (NEW — sewing #details lexicon)
    └── sewing.defs.json   (NEW — sewing sub-domain tokens)
```

`#image`, `#quoteEmbed`, and `#replyRef` are local defs inside `feed/post.json` because they are post/media concerns. Project metadata is reusable and lives under `social.craftsky.project.defs`: `#project`, `#projectCommon`, and `#pattern`.

Per-craft `#details` (e.g. sewing) live under `social.craftsky.project.<craft>` — a new NSID branch per ADR 001. These files declare a `#details` object def only; they do not declare a `main` record.

### `social.craftsky.feed.post`

Top-level shape:

| Field | Type | Required | Notes |
|---|---|---|---|
| `text` | string | yes | `maxGraphemes: 2000, maxLength: 20000`. Bumped from 300/3000 — crafters write longer write-ups than Bluesky/Twitter posts. |
| `facets` | `array<app.bsky.richtext.facet>` | no | Byte-range annotations over `text` for mentions, links, inline hashtags. Reused as-is. |
| `project` | `social.craftsky.project.defs#project` | no | Present iff this post is a project post. |
| `images` | `array<#image>` | no | `maxLength: 4`. Top-level, not inside `embed` — see [Alternatives considered](#alternatives-considered). |
| `embed` | `union<#quoteEmbed>` | no | Open union; today only quote embeds. |
| `reply` | `#replyRef` | no | If present, this post is a reply. |
| `createdAt` | `string<datetime>` | yes | Client-declared creation timestamp. |

### `social.craftsky.project.defs#project`

```
#project {
  common: social.craftsky.project.defs#projectCommon    (required)
  details?: union<
    social.craftsky.project.sewing#details
    // additional per-craft #details lexicons added here as they are defined
  >
}
```

A post with `project` present but no `project.details` is a valid craft-tagged project post with no specialised fields. This lets `craftType` (on `social.craftsky.project.defs#projectCommon`) grow its known-values enum ahead of defining per-craft `#details` lexicons, so a user can post about basketweaving before we write `social.craftsky.project.basketweaving`.

The union on `details` is open (no `closed: true`). Adding a new craft is purely additive: new file under `project/`, new token in `feed.defs`, new entry in the union. No existing records break.

### `social.craftsky.project.defs#projectCommon`

| Field | Type | Required | Constraints | Notes |
|---|---|---|---|---|
| `craftType` | string | **yes** | `knownValues` → `feed.defs` tokens | The craft this project belongs to. Token-backed so new crafts can be added without breaking old clients. Required because a project post without a craft type is meaningless — the whole reason `project` is present is to tag the post with a craft. |
| `status` | string | no | `knownValues` → `feed.defs` tokens (`wip`, `finished`) | Token-backed, extensible. Future values like `planned`, `frogged`, `abandoned` can be added. Optional — crafters posting finished work don't need to think about it. |
| `title` | string | no | `maxGraphemes: 200, maxLength: 500` | Optional project name. Not required — matches the casual Instagram-style "finished this!" posting pattern. Clients wanting grid/card views fall back to truncated `text` when title is absent. |
| `duration` | string | no | `maxGraphemes: 100, maxLength: 500` | Free-text description of how long the project took ("3 weeks", "a weekend", "6 months of evenings"). Deliberately unstructured — crafters describe duration informally. Not queryable by range as a result. |
| `pattern` | `social.craftsky.project.defs#pattern` | no | | Optional pattern reference. See [`#pattern`](#socialcraftskyprojectdefspattern). |
| `materials` | `array<social.craftsky.project.defs#material>` | no | Array `maxLength: 10`. Each `material.text` has `maxGraphemes: 100, maxLength: 1000`; `material.facets[]` uses `app.bsky.richtext.facet` and is scoped to `text`. | Free-form visible material entries with optional mention/hashtag facets. `text` is indexed as the material multi-value column; hashtag facets are merged into searchable tags. See [Materials design rationale](#materials). |
| `tags` | `array<string>` | no | Each `maxGraphemes: 64, maxLength: 64`. Array `maxLength: 10`. Per-entry pattern `^[a-z0-9]+(-[a-z0-9]+)*$` | Structured search tags. Composer-side normalised. See [Tags design rationale](#tags). |

### `social.craftsky.project.defs#pattern`

| Field | Type | Required | Notes |
|---|---|---|---|
| `url` | `string<uri>` | no | Link to the pattern (Ravelry, indie designer website, pattern PDF, etc.). |
| `name` | string | no | `maxGraphemes: 200, maxLength: 500`. Pattern name — e.g. "Simplicity 8265" or "Hitchhiker Shawl". Used when there is no URL, or alongside one. |
| `difficulty` | string | no | `knownValues` → `feed.defs` tokens (`beginner`, `intermediate`, `advanced`, `expert`). Difficulty is a *property of the pattern*, not of the post. Posts without a pattern can't meaningfully have a difficulty. Tokens are extensible if real patterns use different scales. |

All three fields optional — a pattern with just a name ("self-drafted" isn't a pattern but "Butterick 6092" is) is valid; a pattern with just a URL is valid; a pattern with all three is richest.

### `#image` (local def)

```
#image {
  image: blob     (accept: [image/jpeg, image/png, image/webp], maxSize: 15 MB)
  alt: string     (required, maxLength: 1000, maxGraphemes: 1000)
}
```

`maxSize` bumped from 10 MB to 15 MB — comfortably handles high-res phone photos with room for occasional scanned patterns or DSLR shots, without inviting 50 MB image abuse. Still well under the PDS blob limit (~50 MB). Raising the limit later is a safe additive change.

### `#quoteEmbed`, `#replyRef` (local defs)

Unchanged from the current lexicon.

### `social.craftsky.feed.defs`

Shared tokens referenced by `social.craftsky.project.defs#projectCommon` and `social.craftsky.project.defs#pattern`. One file, clearly-named tokens avoid NSID collisions between categories.

- **`craftType` tokens:** `knitting`, `crochet`, `sewing`, `embroidery`, `quilting`. Initial set; more added as the craft taxonomy settles.
- **`status` tokens:** `wip`, `finished`.
- **`difficulty` tokens:** `beginner`, `intermediate`, `advanced`, `expert`.

Each token is an empty named def (`{ type: "token", description: "..." }`). Descriptions should be short and user-facing — they may end up in client tooltips or documentation.

### `social.craftsky.project.sewing` and `social.craftsky.project.sewing.defs`

Sewing `#details`:

```
social.craftsky.project.sewing {
  defs: {
    details: {
      type: "object",
      description: "Sewing-specific fields for a craft project post.",
      properties: {
        projectType: {
          type: "string",
          knownValues: [
            "social.craftsky.project.sewing.defs#garment",
            "social.craftsky.project.sewing.defs#home-goods",
            "social.craftsky.project.sewing.defs#accessory",
            "social.craftsky.project.sewing.defs#soft-toy",
            "social.craftsky.project.sewing.defs#costume",
            "social.craftsky.project.sewing.defs#alteration"
          ],
          maxLength: 100
        }
      }
    }
  }
}
```

One field — `projectType` — with token-backed `knownValues` covering the sewing sub-domains: garment, home goods, accessory, soft toy, costume, alteration. Optional: a sewing post that doesn't specify a sub-domain is still a valid sewing project, just won't surface in sub-domain filters.

`projectType` is deliberately sewing-scoped. Other crafts (knitting, quilting, etc.) will define their own analogous field with their own tokens and potentially a different field name — we are **not** trying to make `projectType` a cross-craft concept. "Sweater" is knitting-scoped; "garment" is sewing-scoped. Forcing a shared `projectType` taxonomy across all crafts would be premature and inaccurate.

Sewing sub-domain tokens live in a sibling `social.craftsky.project.sewing.defs` file — separate from `feed.defs` because they're sewing-specific.

### Materials design rationale

`materials: array<#material>` — each entry is a free-form visible text object with optional field-scoped facets. Rationale:

- Materials are universal (every craft has them) but their *structure* varies wildly per craft: yarn has weight/fibre/skeins; fabric has type/yardage/width; wood has species/dimensions/board-feet. Forcing a shared object shape would either be too thin (only a `name` field) or too domain-specific.
- Free-form visible text works for every craft and stays composer-simple: type one material, press Add, clear the field for the next material.
- Mentions and hashtags inside material text are semantic, not decoration. They use `app.bsky.richtext.facet` on the individual material string so byte ranges remain local to the text they annotate.
- Structured material data (yarn weight as a token, fabric yardage as a number) can still live in per-craft `#details` if needed — they answer different questions. Cross-craft search on "linen" still works via `materials[]`; "show me all DK-weight knitting projects" would use a knitting-specific `yarnWeight` field in knitting's `#details`.
- Indexer materialises `materials[].text` as a multi-value searchable column and merges hashtag facets from `materials[].facets` into searchable tags. Mention facets contribute to the post-mentions table used by notifications.

### Tags design rationale

`tags: array<string>` is structured, separate from inline `#hashtags` in text.

**Why both:**
- Inline hashtags are valuable for rendering (`#amigurumi` becomes a clickable link in prose). `facets` on `app.bsky.richtext.facet` already support this via tag features. We keep them.
- A structured `tags` field keeps prose clean and gives the indexer a single column to query on, rather than parsing facets at index time for every post.

**Composer responsibility — client-side merge.** The composer:
1. Parses inline `#hashtag` facets from `text`.
2. Merges them (deduplicated) into `tags[]` before writing the record.
3. Normalises user input in the tags field: "Fair Isle" typed in the tags field → written as `"fair-isle"`.

So the record on PDS has:
- `text` with hashtags as-typed (e.g. `"Finally finished! Took forever #amigurumi"`)
- `facets` with tag features for `amigurumi`
- `tags[]` with the merged, deduplicated, normalised set (e.g. `["amigurumi", "sock-knitting", "fair-isle"]`)

**Indexer responsibility — belt-and-suspenders.** The AppView indexer:
1. Reads `tags[]` as the authoritative search field.
2. *Also* scans `facets` for tag features and adds any missing ones to the search index.

This catches records from third-party clients that skip the composer-side merge. The PDS record remains the source of truth; the indexer's job is to be forgiving about what it can discover.

**Kebab-case pattern.** `tags[]` entries validate against `^[a-z0-9]+(-[a-z0-9]+)*$`: lowercase alphanumeric, optional single-hyphen separators, no leading/trailing/doubled hyphens.

- Inline `#hashtags` can't contain spaces (facet limitation), so they merge into `tags[]` as single-word strings (e.g. `"sockknitting"`).
- Tags-field entries get spaces auto-converted by the composer: `"Fair Isle"` → `"fair-isle"`.
- Display side: the client renders `"fair-isle"` as "Fair Isle" (hyphens → spaces, title-cased) for readability.

This accepts that inline-authored tags will look uglier than field-authored tags. It's the crafter's choice which path they use.

### Composer help text (client-side, non-normative)

The spec documents the following help-text hints as a client contract, so every composer (Flutter first, hypothetical third-party clients later) teaches users the same conventions:

- **Tags field:** "Use spaces for multi-word tags — we'll format them. These help people find your project in search."
- **Materials field:** "List materials one at a time. You can include @mentions and #tags."

These are not lexicon concerns — they're UX hints captured here so they don't get lost when building the composer screen.

### AppView indexer commitments

Beyond ADR 001's commitment to materialise `is_project` and `craftType` as queryable columns, this spec commits the indexer to also materialise:

- `project.common.status`
- `project.common.pattern.difficulty`
- `project.common.materials[].text` (multi-value material index) and material facet hashtags/mentions
- `project.common.tags[]` (multi-value index, with inline-facet merge as documented above)
- Per-craft `#details` fields, each as its own column(s) as each craft is added. Sewing's `projectType` is the first.

The exact indexer schema is out of scope for this spec — it belongs to the AppView indexer spec — but the *commitment* that these fields are queryable dimensions is locked here so future feed/search specs can assume it.

## Alternatives considered

### Images inside `embed` (Bluesky pattern) vs top-level

Bluesky's `app.bsky.feed.post` puts images inside `embed` as a union variant (`app.bsky.embed.images`) alongside video, external links, quote records, and quote-with-media wrappers. All non-text content goes through one polymorphic field.

**Rejected** in favour of keeping `images` top-level (current Craftsky shape). Rationale:

- Simpler for the common case: a post with images and no quote is just `{images: [...]}`.
- "Quote post with images" works trivially by having both fields present — no need for a `recordWithMedia` wrapper variant.
- Craftsky's v1 scope has no video planned. If video is added later, it can be a second top-level field or a new `embed` variant — either works.
- Cost of diverging from Bluesky's embed pattern is low: we're a separate app, no one expects bit-for-bit compatibility.

### `difficulty` on `#projectCommon` vs inside `#pattern`

Early brainstorming considered `difficulty` as a top-level field on `#projectCommon`. Rejected in favour of nesting it inside `#pattern`.

**Reason:** difficulty is a property of the *pattern*, not of the post. Patterns have difficulty ratings printed on them by the designer; posts without patterns have no meaningful difficulty. Self-drafted or free-formed projects shouldn't be forced to self-rate.

### `duration` as `workPeriod: {startedAt, completedAt}` datetime pair

Considered and rejected in favour of free-text `duration`.

**Reason:** crafters describe duration informally ("a weekend", "3 weeks of evenings", "6 months but I kept putting it down"). A structured datetime pair invites precision that doesn't match how project time is remembered or described. The snapshot model also argues against it — a snapshot doesn't inherently have a start/end; the project it depicts does, and we explicitly have no cross-post project identity.

Cost: duration is not queryable by range ("all projects that took more than a month"). Accepted as a non-goal — this is a niche filter at best.

### `materials` as structured quantity/vendor fields

Considered `materials: array<{name, quantity?, notes?}>`. Rejected in favour of rich-text material entries.

**Reason:** the `quantity` field would often duplicate what's already in post text, and since it'd be optional, most records wouldn't have it — making search on it unreliable. A visible `text` field covers the discovery use case (search on "merino") while facets preserve mentions and hashtags without forcing a cross-craft material schema.

### Rich sewing `#details` (garment sizing, finished dimensions, fit notes)

Considered several variants: garment-first (size, bodice type, fit adjustments), dimension-first (width × height × length), polymorphic (optional `garment?` and `finishedDimensions?` sub-objects). Rejected in favour of minimal sewing `#details` with just `projectType`.

**Reason:** sewing covers wildly different sub-domains (garments, home goods, accessories, toys, costume, alterations) with different structural needs. Designing a shape that serves all of them well is a large design project, and the roadmap explicitly flags "validate project-post field set with real crafters" as an open question. First sewing lexicon is conservative — `projectType` is enough to categorise for discovery; everything else can be added later as optional fields (safe under atproto evolution rules).

## Consequences

### Breaking changes

None for records in production — there are no records in production yet per confirmed user statement. Existing lexicon files are being rewritten.

### Code changes required

- `lexicon/social/craftsky/feed/post.json` — restructure per this spec.
- `lexicon/social/craftsky/feed/defs.json` — new file.
- `lexicon/social/craftsky/project/sewing.json` — new file.
- `lexicon/social/craftsky/project/sewing.defs.json` — new file.
- `lexicon/README.md` — update the "Planned Namespaces" table to include the new files and the `project.*` branch.
- `social.craftsky.actor.profile` — currently references `social.craftsky.feed.post#projectDetails.craftType` in a field description. Update to reference `feed.defs` tokens.
- Any Go/Dart code generation in `appview/` and `app/` will regenerate against the new schemas. No expected schema conflict because there are no records yet.
- Indexer implementation (future spec) needs to know to materialise the fields listed under [AppView indexer commitments](#appview-indexer-commitments).
- Composer implementation (future spec) needs to implement the tag merge, kebab-case normalisation, and help-text hints documented under [Tags design rationale](#tags-design-rationale) and [Composer help text](#composer-help-text-client-side-non-normative).

### Migration path

None needed. First real PDS writes happen against the new schema.

### Risks

- **Field set may still be wrong.** The roadmap item "validate project-post field set with real crafters" remains open. This spec is our best guess before real user feedback. Mitigated by atproto evolution rules (adding optional fields later is safe) and the decision to stay minimal (Level C sewing `#details`, no difficulty on common, no structured duration).
- **Token taxonomy may shift.** `craftType` tokens in particular — today's list is a guess at what crafts will be active on Craftsky. Adding new tokens is safe; users who posted before a token existed will have used free-form strings (which `knownValues` permits) and search still works.
- **Composer-side tag merge is a client contract, not enforceable at lexicon level.** Third-party clients that skip the merge produce records where `tags[]` misses facet-tags. Indexer-side belt-and-suspenders mitigates for search; direct record readers still see an incomplete `tags[]`. Accepted — the alternative (putting merge logic in the lexicon) isn't possible.

## Open questions

None at time of writing. Resolved during review:

- **Initial `craftType` token list:** knitting, crochet, sewing, embroidery, quilting. More can be added later without breaking changes.
- **Tag character set:** ASCII-only kebab-case as specified. Revisit if international tags become a real user need.
- **Image count limit:** `maxLength: 4` matches Bluesky. Revisit if crafters ask for more (e.g. WIP → finished before-and-after sequences).
