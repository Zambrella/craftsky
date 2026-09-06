## Architecture Decision Record

- Status: Accepted
- Aspect: Lexicon, post embeds, video metadata
- Date: 2026-09-03
- Decision: Reuse `app.bsky.embed.video` for native video posts
- Lexicon review: Completed using the project `atproto-lexicon` checklist

### Why I needed to decide this

CraftSky posts need a portable native-video representation whose processed blob,
alt text, aspect ratio, and optional captions remain in the author's public PDS
record. The existing `social.craftsky.feed.post#embed` field is an optional open
union containing the local quote variant and `app.bsky.embed.external`.

Adding a variant is additive, but it commits CraftSky to a public record shape
and determines how other atproto clients can read the post.

### Options I considered

**Option 1: Reuse `app.bsky.embed.video` - chosen**

Pros:

- Uses the standard interoperable video, caption, and aspect-ratio shapes.
- Keeps the processed blob and authored accessibility metadata portable.
- Adds one variant to the existing open union without changing old records.
- Reuses Indigo's generated `appbsky.EmbedVideo` types and CBOR support.

Cons:

- CraftSky must follow evolution of the standard schema and playback service.
- Product restrictions such as MP4-only composition and no mixed media remain
  application policy rather than constraints on federated records.

**Option 2: Define a CraftSky-specific video object - not chosen**

Pros:

- Could encode every current CraftSky product restriction directly.

Cons:

- Duplicates a standard record shape and reduces interoperability.
- Freezes service-specific product choices into CraftSky's public schema.

**Option 3: Keep video only in AppView storage - not chosen**

Pros:

- Requires no public record change.

Cons:

- Breaks portability and the public-data-on-PDS rule.
- Prevents another AppView from reconstructing the authored video post.

### What I decided

Add `app.bsky.embed.video` to the existing open `embed` union in
`social.craftsky.feed.post`. Do not add local video, caption, blob, or aspect-ratio
definitions.

The standard variant carries one video blob, optional alt text, optional
`app.bsky.embed.defs#aspectRatio`, and optional WebVTT caption blobs. CraftSky's
authenticated authoring path accepts MP4 up to 300,000,000 bytes, limits alt text
to 1,000 graphemes and 10,000 bytes, and applies the approved caption checks.

Lexicon generation explicitly resolves Indigo's `app/bsky/embed/video.json` and
`app/bsky/embed/defs.json`. Generated CraftSky code references Indigo's existing
types; generated files are not edited by hand.

The currently pinned Indigo schema copy predates the standard's 300,000,000-byte
limit and still describes 100,000,000 bytes. Its generated Go shape does not
encode that maximum, so CraftSky retains the reviewed Indigo pin and enforces the
current 300,000,000-byte policy at its boundaries. A broad Indigo upgrade is not
bundled into this feature unless generation or API support proves incompatible;
any upgrade must be reviewed separately because Indigo also implements OAuth and
PDS access.

### Compatibility and evolution

- The union remains open.
- Existing quote, external-card, and no-embed records remain valid.
- New standard video records are additive.
- Video, quote, and external card cannot coexist because they are union variants.
- CraftSky's authoring path also rejects top-level images and replies with video.
- Project posts may use the same top-level video variant; scheduled posts may not.
- Federated records are validated and indexed by their public shape. CraftSky's
  stricter composition rules must not cause otherwise safe standard metadata to
  be silently rewritten.

### Consequences and notes

- Update `lexicon/social/craftsky/feed/post.json` and `lexicon/README.md`.
- Add only the two required external schemas to `just lexgen`, run `just lexgen`,
  and commit generated output with the schema change.
- Store the complete record as today; do not add redundant video columns.
- ADR 012 separately governs the ephemeral service-JWT handoff and completion
  verification. This ADR grants no additional credential authority.
