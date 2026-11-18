## Architecture Decision Record

- Status: Approved
- Aspect: Lexicon (atproto schemas), post composition, disclosure metadata
- Date: 2026-09-09
- Decision: Record self-drafted pattern authorship and require explicit sponsored status on posts

### Why I needed to decide this

CraftSky's original pattern difficulty vocabulary did not distinguish a beginner
from a confident beginner, while its expert tier was less useful to the intended
four-step progression. Project posts also had no structured way to say that the
author drafted the pattern.

Posts need a durable sponsorship disclosure as part of the public record. Keeping
that fact only in the client or AppView would make it disappear when records move
between PDSes or are rendered by another AppView.

The records are not in production, so this is the appropriate time to replace the
difficulty vocabulary and make sponsorship required without a versioned record
NSID.

### Options I considered

**Option 1: Represent self-drafted as a pattern name - not chosen**

Free text cannot be rendered, filtered, or translated reliably. It also prevents
a self-drafted pattern from retaining a useful name and other metadata.

**Option 2: Put self-drafted on project common - not chosen**

Authorship describes the pattern, not the project or post. Keeping it inside the
pattern object preserves that relationship and allows it to coexist with pattern
name, difficulty, URL, designer, and publisher.

**Option 3: Store sponsorship privately in AppView - not chosen**

Sponsorship is public-by-intent disclosure metadata. Private storage would not
travel with the author's record and would make federation inconsistent.

**Option 4: Add optional sponsored metadata - not chosen**

Omission would leave clients unable to distinguish an explicit negative from a
client that never asked the author. Requiring a boolean makes every newly authored
post's disclosure state unambiguous.

**Option 5: Add required sponsored metadata and optional pattern authorship - chosen**

Every post record carries `sponsored: true|false`. Pattern metadata may carry
`selfDrafted: true`; clients omit it when false.

### What I decided

The pattern difficulty tokens are, in ascending order:

- `social.craftsky.feed.defs#beginner`
- `social.craftsky.feed.defs#confidentBeginner`
- `social.craftsky.feed.defs#intermediate`
- `social.craftsky.feed.defs#advanced`

The pre-production `expert` token is removed.

`social.craftsky.project.defs#pattern` gains an optional `selfDrafted` boolean.
When true, it identifies the pattern as self-drafted by the post author. It may
coexist with every other pattern property, including an author-supplied difficulty
rating.

`social.craftsky.feed.post` gains a required `sponsored` boolean. It is a
self-declared disclosure of sponsorship or other commercial consideration. It is
presentation metadata only and must not affect CraftSky's chronological ordering,
distribution, reach, or moderation.

CraftSky composers expose sponsorship for top-level standard, quote, and project
posts. Replies and comments do not expose the control and are authored with
`sponsored: false`. This client rule cannot be expressed by the Lexicon; AppView
also rejects mediated attempts to create a sponsored reply. Federated records
written elsewhere remain indexable and their disclosure remains visible.

### Compatibility and evolution

This deliberately changes the pre-production contract. Records using `expert` or
missing required `sponsored` are not supported as authored records after the
change. Existing development database rows receive `sponsored = false` during the
migration.

Future difficulty values can be added through the open `knownValues` list.
Additional disclosure context would require optional fields or a dedicated open
object rather than changing the boolean's meaning.

### Consequences

- Every record-writing path, including seeds and scheduled publication, must emit
  an explicit sponsored boolean.
- AppView materializes sponsorship for efficient rendering across post surfaces.
- Flutter preserves sponsorship through drafts, scheduling, uploads, and quote
  previews.
- Self-drafted remains inside raw project metadata; no dedicated database column
  is needed until filtering or analytics requires one.
- This decision does not introduce advertising, paid ranking, sponsorship
  verification, payment processing, or targeting.
