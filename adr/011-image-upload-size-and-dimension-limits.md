## Architecture Decision Record
- Status: Accepted
- Aspect: Lexicon (atproto schemas) and image upload policy
- Date: 2026-09-02
- Decision: Limit uploaded images to 2,000,000 encoded bytes and 4000 pixels per decoded side

### Why I needed to decide this

Craftsky's post and business image Lexicons allowed 15 MiB blobs, while the current Bluesky image contract and client preparation policy use exactly 2,000,000 bytes. Craftsky also needs a decoded geometry ceiling to bound client rendering and AppView image validation work.

Tightening an existing Lexicon constraint is normally incompatible schema evolution. Craftsky is not in production and has no active users or persisted public records, so this deliberate pre-launch breaking change is acceptable. This decision supersedes the 15 MiB image statement in ADR 010.

### Options I considered

**Option 1: Client preparation plus AppView-mediated validation and Lexicon metadata constraints - CHOSEN**

- Flutter resizes and compresses images before transfer.
- AppView validates encoded bytes, MIME, decode success, and dimensions before mediated PDS upload.
- Record Lexicons constrain declared image blob metadata to 2,000,000 bytes.

Pros:
- Avoids unnecessary transfer and PDS writes.
- Prevents custom AppView API clients from bypassing the mediated upload boundary.
- Keeps uploaded bytes and their CID unchanged by AppView.

Cons:
- Both Flutter and AppView must maintain matching policy constants.
- Records written directly to a PDS do not pass through AppView's decoded-byte validation.

**Option 2: Client-only enforcement**

Not chosen because custom AppView clients could bypass size and dimension checks.

**Option 3: AppView transcoding**

Not chosen because server-side image encoding would add CPU and memory load, change blob CIDs, and duplicate client work.

### What I decided

Adopt Option 1 with these exact limits:

- Encoded image blob: at most `2000000` bytes.
- Decoded uploaded image: at most `4000` pixels wide and `4000` pixels high.
- Flutter source admission: provisionally at most `50000000` bytes, 8192 pixels per source side, and 16,000,000 decoded source pixels before resizing.

The pixel limit remains outside the Lexicons. `aspectRatio` is client-authored layout metadata and is not proof of the dimensions in an image blob, so AppView enforces geometry by decoding bytes received through its immediate and scheduled upload paths. Firehose indexers do not fetch and decode blobs referenced by records written directly to a PDS; bounded asynchronous validation or quarantine for those records requires a separate architecture decision.

### Notes

- `social.craftsky.feed.post#image.image` and `social.craftsky.business.defs#image.image` both use `maxSize: 2000000`.
- Link-preview thumbnails retain their stricter 1,000,000-byte policy.
- Lexicon `maxSize` constrains declared blob metadata; it does not verify actual MIME, decode success, or decoded geometry.
- If Craftsky has active records in the future, another tightening change will require a new record version rather than an in-place Lexicon edit.
