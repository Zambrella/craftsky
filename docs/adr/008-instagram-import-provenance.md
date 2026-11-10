## Architecture Decision Record

- Status: Accepted
- Aspect: Lexicon (atproto schemas), AppView distribution and presentation
- Date: 2026-07-23
- Decision: Add optional, self-asserted external-import provenance to CraftSky posts

### Why I needed to decide this

The historical Instagram importer publishes old posts directly to a member's
PDS. Without an explicit record-level signal, AppView cannot distinguish a
historical backfill from an ordinary new post. Inferring that distinction from
an old `createdAt`, deterministic record key, or ingestion timing would
misclassify legitimate posts and would not survive re-indexing consistently.

The signal must also remain portable with the public post and visible to other
clients without publishing an Instagram handle, post identifier, media
identifier, filename, archive fingerprint, or external URL.

Because Lexicon changes are load-bearing, this decision is recorded before
editing `lexicon/social/craftsky/feed/post.json`.

### Options I considered

**Option 1: Add an optional minimal object to the post record — CHOSEN**

Shape:

```json
{
  "externalImport": {
    "source": "instagram"
  }
}
```

Pros:

- Gives AppView an explicit and portable classification signal.
- Is additive for existing records and clients.
- Leaves room for other import sources without changing the field shape.
- Publishes no source account or archive identifier.

Cons:

- Any compatible client may self-assert the field.
- A replacement record may omit the optional field and remove the
  classification.

**Option 2: Infer historical imports from `createdAt` or record keys — not chosen**

Pros:

- Requires no public schema change.

Cons:

- Confuses deliberately backdated posts and clock errors with imports.
- Cannot reliably distinguish later ordinary records.
- Couples AppView policy to one importer's key algorithm.

**Option 3: Store provenance only in AppView — not chosen**

Pros:

- Keeps the public record unchanged.

Cons:

- Requires a trusted registration side channel or new importer endpoint.
- Is not portable when another AppView indexes the same record.
- Conflicts with the client-only importer architecture.

**Option 4: Publish a sidecar provenance record — not chosen**

Pros:

- Keeps the post schema unchanged and permits richer future metadata.

Cons:

- Creates two-record ordering and convergence problems for a signal needed
  when the post event itself is indexed.
- Adds unnecessary record lifecycle and OAuth authority for one bounded token.

### What I decided

Adopt **Option 1**. Add optional `main.externalImport`, referencing a local
`#externalImport` object. When present, that object requires a `source` string
bounded to 64 bytes. Its open `knownValues` list initially contains only
`instagram`.

The provenance is self-asserted display and distribution metadata. It does not
verify ownership of an Instagram account, grant trust, or grant elevated
visibility. AppView recognizes only the exact `instagram` token for the
historical-import behavior in this version. Unknown future values remain valid
but receive ordinary behavior until explicitly supported.

The current authoritative PDS record controls classification. A replacement
that retains the field remains classified; a replacement that omits it clears
the classification. AppView must not make provenance sticky.

For exact Instagram imports, AppView:

- preserves the record on profiles and in search;
- orders author-profile lists by the record's original `createdAt`;
- excludes only the original authored item from the home-timeline authored
  arm;
- suppresses notification activation caused by that source record while
  retaining indexing relationships and ordinary later engagement; and
- exposes the optional source through canonical and quoted-post response
  shapes for a subtle client label.

### Compatibility and evolution

- Existing posts omit `externalImport` and remain valid ordinary posts.
- The new field is optional, so old records remain valid under the updated
  schema and older consumers can ignore it.
- `knownValues` is deliberately open rather than an enum. New recognized
  sources may be added later without changing the object shape.
- Richer or source-specific metadata is out of scope. If it becomes necessary,
  prefer a separately reviewed schema rather than adding identity or archive
  linkage to this minimal object.
- After the schema edit, regenerate Lexicon-derived Go types with
  `just lexgen` and keep other generated/validated client contracts
  synchronized.

### Privacy and trust notes

- The public record contains only the source token.
- It contains no Instagram handle, source post/media ID, filename, archive
  name or fingerprint, caption hash, or external URL.
- The UI wording must say "Imported from Instagram", not "verified", and must
  not imply that the current CraftSky account owns a particular Instagram
  identity.
