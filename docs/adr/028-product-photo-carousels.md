# ADR 028: Ordered product photo carousels

- Status: Accepted
- Date: 2026-11-16
- Aspect: Business lexicon and client/API product media
- Lexicon review: Project atproto-lexicon checklist applied

## Context

Products currently expose one optional image in portable business declarations.
CraftSky requires a photo when authoring products. Product discovery needs larger
profile cards and up to four photos, using the existing post photo composer and
carousel interactions. Products have not launched.

## Decision

Add an optional `images` array of `social.craftsky.business.defs#image` objects,
with a maximum of four entries, to the existing product definition. Array order
is display order; the first image is the cover used by compact management rows.
Each entry keeps its own blob, alt text and aspect ratio. First-party writes
require one to four valid photos. Images retain the existing per-blob limit.

Replace the single optional `image` property with optional `images` directly.
The maintainer confirmed products are not live and requested no migration or
legacy compatibility. This is a deliberate pre-launch exception to ADR 010's
post-publication evolution policy. Both clients and AppView change together;
there is no dual-write/read path, PDS rewrite, SQL migration or versioned NSID.
Independent records may omit photos; first-party writes require one to four.

AppView validates every authored image, hydrates safe image metadata in order,
and includes every referenced blob in business-profile image-safety processing.
Public records and blobs remain on the owner's PDS; indexed raw JSON remains the
projection source. No database ownership or authentication boundary changes.

Both clients reuse their shared photo-details composer, enforce the four-photo
limit, preserve descriptions and order, and cache successful uploads across save
retries. Profile product cards show large image carousels above the seller's
title/price and external product link. Swiping/viewing images does not launch an
external shop. The ordinary external-destination confirmation remains in place.

## Alternatives

A new product or business-profile NSID would require parallel record ownership
and migration for an optional additive field. Reusing four separate products
would confuse photo counts with the separate four-product catalogue limit.
Keeping both image fields would add transitional behavior for an unlaunched feature.

## Consequences

Generated Go lexicon/CBOR code changes with the schema. Both API/client mappings,
record reconciliation and image-safety extraction must retain all images. All first-party consumers use the ordered list. No production deployment or PDS rewrite
is part of this local implementation.
