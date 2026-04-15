# lexicon

Craftsky atproto lexicon schemas, under the `social.craftsky.*` namespace.

Lexicons are JSON schemas that define the shape of records stored on user PDSes. They are load-bearing: once users create records against a lexicon, changing it is painful because existing data on PDSes can't be easily migrated.

**Design carefully upfront.** Every change to a lexicon in production needs an ADR — use the `writing-architecture-decision-records` skill.

**Before editing or adding schemas, invoke the [`atproto-lexicon`](../.claude/skills/atproto-lexicon/SKILL.md) project skill.** It distils the canonical atproto lexicon docs (guide, style guide, spec) into the NSID/type/style/evolution rules you need to get this right the first time.

## Planned Namespaces

| Lexicon | Purpose |
|---|---|
| `social.craftsky.feed.post` | A post — general or a craft project (via optional `project` sub-object with craftType/status/patternUrl) |
| `social.craftsky.feed.repost` | A repost of a Craftsky post |
| `social.craftsky.actor.profile` | Craftsky-specific profile extension (single record, key `self`); signals active Craftsky users and stores craft preferences |

**Reused from `app.bsky.*` (not redefined here):**

| Lexicon | Why reuse |
|---|---|
| `app.bsky.actor.profile` | Base profile (display name, avatar, bio) — unchanged across apps |
| `app.bsky.graph.follow` | Standard follow semantics |
| `app.bsky.graph.block` | Standard block semantics |
| `app.bsky.feed.like` | Generic strongRef + createdAt; fine for liking Craftsky posts |

## References

- [AT Protocol Lexicon Spec](https://atproto.com/specs/lexicon)
- [Bluesky's `app.bsky` lexicons](https://github.com/bluesky-social/atproto/tree/main/lexicons) — good reference for conventions
