# lexicon

Craftsky atproto lexicon schemas, under the `social.craftsky.*` namespace.

Lexicons are JSON schemas that define the shape of records stored on user PDSes. They are load-bearing: once users create records against a lexicon, changing it is painful because existing data on PDSes can't be easily migrated.

**Design carefully upfront.** Every change to a lexicon in production needs an ADR — use the `writing-architecture-decision-records` skill.

**Before editing or adding schemas, invoke the [`atproto-lexicon`](../.claude/skills/atproto-lexicon/SKILL.md) project skill.** It distils the canonical atproto lexicon docs (guide, style guide, spec) into the NSID/type/style/evolution rules you need to get this right the first time.

## Planned Namespaces

| Lexicon | Purpose |
|---|---|
| `social.craftsky.feed.post` | A post — general or a craft project (via optional `project` sub-object with `common` + open-union `details`). Supports replies, up to 4 images, rich-text facets (reusing `app.bsky.richtext.facet`), and quote embeds. |
| `social.craftsky.feed.defs` | Shared tokens for cross-craft values in `#projectCommon` (`craftType`, `status`) and `#pattern` (`difficulty`). |
| `social.craftsky.feed.repost` | A repost of a Craftsky post |
| `social.craftsky.feed.like` | A like on a Craftsky post (distinct NSID for firehose-filter efficiency) |
| `social.craftsky.actor.profile` | Craftsky-specific profile extension (single record, key `self`); signals active Craftsky users and stores craft preferences |
| `social.craftsky.project.sewing` | Sewing-specific `#details` referenced from `feed.post#project.details`. Defines a referenced type only — no `main` record. |
| `social.craftsky.project.sewing.defs` | Sewing sub-domain tokens (`garment`, `homeGoods`, `accessory`, `softToy`, `costume`, `alteration`). |

**Reused from `app.bsky.*` (not redefined here):**

| Lexicon | Why reuse |
|---|---|
| `app.bsky.actor.profile` | Base profile (display name, avatar, bio) — unchanged across apps |
| `app.bsky.graph.follow` | Standard follow semantics |
| `app.bsky.graph.block` | Standard block semantics |
| `app.bsky.richtext.facet` | Byte-range rich-text annotations (mentions/links/tags); referenced from `social.craftsky.feed.post.facets` |

Note: comments are not a separate record type — a "comment" is a `social.craftsky.feed.post` with its `reply` field set. Quote posts are regular posts whose `embed` carries a `#quoteEmbed` wrapping a strongRef to the quoted record.

Per-craft `#details` lexicons live under the `social.craftsky.project.*` branch (one file per craft). They define a `#details` object type only — they are referenced types, not standalone records. See [ADR 001](../adr/001-post-lexicon-project-extensibility.md) for the extensibility rationale.

## References

- [AT Protocol Lexicon Spec](https://atproto.com/specs/lexicon)
- [Bluesky's `app.bsky` lexicons](https://github.com/bluesky-social/atproto/tree/main/lexicons) — good reference for conventions
