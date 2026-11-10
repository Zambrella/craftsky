## Architecture Decision Record

- Status: Accepted
- Aspect: Lexicon (atproto schemas), post embeds, AppView presentation
- Date: 2026-08-25
- Decision: Reuse `app.bsky.embed.external` for website-card embeds
- Lexicon review: Completed using the project `atproto-lexicon` checklist

### Why I needed to decide this

CraftSky posts need a portable website-card representation containing a final
URI, title, description, and optional thumbnail. This data is authored into the
member's public PDS record so readers do not refetch the destination page.

The existing `social.craftsky.feed.post#embed` field is an open union containing
the local quote variant. Adding another variant is load-bearing and must remain
compatible with existing quote posts and posts that omit `embed`.

### Options I considered

**Option 1: Reuse `app.bsky.embed.external` — CHOSEN**

Pros:

- Uses an established interoperable website-card record shape.
- Reuses Indigo's pinned generated Go type and CBOR implementation.
- Adds one variant to the existing open union without changing old records.
- Avoids defining a redundant CraftSky-specific schema.

Cons:

- The standard schema does not encode CraftSky's stricter metadata limits or
  thumbnail dimensions, so those remain application-level validation policy.
- The standard object occupies the same union slot as a quote embed.

**Option 2: Define a CraftSky-specific external-card object — not chosen**

Pros:

- Could encode CraftSky-specific metadata and dimension constraints directly.

Cons:

- Duplicates an existing standard record shape.
- Reduces interoperability with other atproto clients.
- Freezes product-specific validation into a public schema unnecessarily.

**Option 3: Keep cards only in AppView storage — not chosen**

Pros:

- Requires no public record change.

Cons:

- Cards would not be portable with the member's PDS data.
- Another AppView could not reproduce the authored card without refetching.
- Conflicts with the public-data-on-PDS architecture.

### What I decided

Add `app.bsky.embed.external` to the existing open `embed` union in
`social.craftsky.feed.post`. Do not add a local external-card definition.

The standard external record requires `uri`, `title`, and `description` and may
carry one `image/*` blob capped at 1,000,000 bytes. CraftSky applies its tighter
metadata, image-format, decoded-size, and dimension rules at authenticated API
boundaries rather than altering the standard schema.

Lexicon generation resolves `app.bsky.embed.external` from the exact Indigo
version pinned by `appview/go.mod`. The generation recipe names that external
schema explicitly and generated CraftSky types reference Indigo's
`appbsky.EmbedExternal` type.

### Compatibility and evolution

- The existing union remains open.
- Existing local quote records remain valid and retain their generated variant.
- Existing records without `embed` remain valid and continue to omit the field.
- New standard external records are additive under the updated schema.
- External cards and quotes cannot coexist because they are union variants.
- Top-level image conflict behavior is CraftSky application policy, not a
  lexicon restriction; federated records remain indexable without data loss.
- New embed variants may be added through a separately reviewed additive change.

### Consequences and notes

- `just lexgen` must run whenever the union or its explicit external input changes.
- Generated files under `appview/internal/lexicon/craftsky/` are never edited by
  hand.
- AppView stores the complete record JSON and projects cards from that record;
  no redundant public database column or migration is introduced.
- Metadata fetching, SSRF controls, scheduling, rendering, and telemetry policy
  are outside this lexicon decision and remain governed by the approved link
  preview requirements and coding plan.
