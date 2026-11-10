## Architecture Decision Record

- Status: Accepted, amended 2026-08-29
- Aspect: Lexicon (atproto schemas), external schema pinning, package-backed catalogs
- Date: 2026-08-28; catalog policy amended 2026-08-29
- Decision: Separate business declaration and event records with a pinned external schema and maintained Go catalog data
- Lexicon review: Completed using the project `atproto-lexicon` checklist

### Why I needed to decide this

Craftsky business accounts need portable public details, ordered product cards,
and independently addressable event appearances without making a public record
authoritative for account classification. The records must remain useful to
independent atproto authors while Craftsky applies stricter first-party write
and hydration policy.

The declaration also reuses `community.lexicon.location.address`. Depending on
the mutable network copy during generation would make builds non-reproducible,
and hydrating its complete street-address shape would expose more location data
than Craftsky needs.

### Options I considered

**Option 1: Private account type plus separate declaration and events - CHOSEN**

Pros:

- Keeps authoritative `regular` or `business` classification private to the
  AppView and independent from optional public presentation.
- Gives events stable TID-keyed identities for pagination and moderation.
- Lets declaration deletion leave classification and events unchanged.
- Permits open known-value catalogs and safe ingestion of independent records.

Cons:

- Requires two new record collections, projection tables, and deletion stages.
- Requires a pinned external definition and generated-code support.

**Option 2: Add business fields to `social.craftsky.actor.profile` - not chosen**

This would couple membership, classification, and presentation; rewrite the
membership record for product changes; and make event identity and pagination
impractical.

**Option 3: Put declaration and events in one business record - not chosen**

This separates membership but still rewrites every event together, prevents
record-level event moderation, and imposes a fixed event-array ceiling.

### What I decided

Keep `social.craftsky.actor.profile` unchanged. Add these schemas:

- `social.craftsky.business.profile` is a `literal:self` record. Every field is
  optional, including `businessTypes`, `offerings`, `tagline`, `hoursNote`,
  `serviceArea`, `location`, `primaryAction`, and `products`. An empty record is
  valid. Arrays of types, offerings, and products have schema maximum 20.
- `social.craftsky.business.event` is a `tid` record. `name`, `startsAt`,
  `endsAt`, nonempty `roles`, and `createdAt` are required. `summary`,
  `venueName`, `mode`, `status`, `timeZone`, `isAllDay`, `eventUri`,
  `registrationUri`, and `image` are optional. Roles have schema maximum 10.
- `social.craftsky.business.defs` contains reusable typed action, product, price,
  image, and aspect-ratio objects. Product requires only `title` and `uri` at
  schema level; Craftsky additionally requires an image on first-party writes.
- Catalog strings use open `knownValues`, not `enum`. Craftsky's recognized
  catalogs and lower first-party maxima remain application policy. Unknown
  lexicon-valid values can therefore be retained and represented safely.
- Text, array, blob, and record-key bounds are the exact bounds approved in
  `docs/changes/2026-08-27-business-profiles/01-requirements.md`. Product and
  event images accept JPEG, PNG, or WebP up to 15 MiB, with optional aspect
  ratio and alt text up to 1000 graphemes and bytes.
- No inventory, availability, synchronization, shipping, tax, checkout,
  disclaimer, guarantee, `onlineUri`, or structured event-location field is
  defined.

The declaration's `location` references
`community.lexicon.location.address`. Craftsky preserves the complete raw
  independent value but writes and hydrates only a two-letter non-private country
  recognized by `golang.org/x/text/language` and optional bounded `locality`.

### External address pin

The exact dependency is:

- AT-URI: `at://did:plc:mtr7qrqtcyseedx3jyr5o7db/com.atproto.lexicon.schema/community.lexicon.location.address`
- CID: `bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq`
- Local value: `appview/cmd/lexgen/external/community.lexicon.location.address.bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq.json`

On 2026-08-28, PLC resolution returned PDS endpoint
`https://eurosky.social`. The record was retrieved with:

```text
GET https://eurosky.social/xrpc/com.atproto.repo.getRecord?repo=did%3Aplc%3Amtr7qrqtcyseedx3jyr5o7db&collection=com.atproto.lexicon.schema&rkey=community.lexicon.location.address
```

The response URI and CID matched the values above. The vendored file is the
complete response `value`, including `$type`, `lexicon`, `id`, and `defs`; it is
not the surrounding `getRecord` response envelope.

A contract test decodes the JSON with lossless integer handling, encodes the
value as canonical atproto DAG-CBOR, and computes CIDv1 with codec `dag-cbor`
and multihash `sha2-256`. Generation fails unless the recomputed CID is the
pinned CID. `just lexgen` maps this NSID only to the local CID-named file and
does not resolve it over the network.

### Country and currency catalogs

Runtime country and currency decisions use the repository's pinned
`golang.org/x/text` module, never network lookups or ambient operating-system
data.

- `language.ParseRegion` plus `Region.IsCountry` recognizes two-letter country
  codes. Craftsky rejects private-use regions and canonicalizes accepted input.
- `currency.ParseISO` recognizes uppercase currency codes, and
  `currency.Standard.Rounding` supplies their decimal scale. Craftsky rejects
  `XXX` (no currency) and `XTS` (testing), but does not maintain a separate
  current/withdrawn/fund/metal classification.
- Catalog behavior changes only when the pinned module version changes through
  an ordinary reviewed dependency update. Representative domain tests protect
  Craftsky's casing, exclusions, and canonical decimal grammar.

This amendment deliberately trades first-party source provenance and exact
List One recency semantics for less code, fewer checked-in data artifacts, and
routine upstream maintenance.

### Compatibility and evolution

- These are new record NSIDs; no existing record shape changes.
- The existing actor profile remains byte-for-byte unchanged.
- All declaration fields are optional, so an empty declaration remains valid.
- Required event identity/time fields are limited to the minimum needed by any
  consumer; Craftsky-only requirements stay at the API boundary.
- `knownValues` remain open and may gain recognized values additively.
- New optional fields may be added after review. Existing fields, types,
  required sets, record keys, and bounds must not be removed, renamed, or
  tightened. A breaking change requires a versioned NSID.
- Complete source JSON is retained by projection so unknown top-level fields
  and independently authored values survive re-projection and replacement.

### Consequences and notes

- Lexicons, the pinned address value, generated lexicon Go files, and pinned
  module behavior are reviewed as one reproducible unit.
- AppView validation remains stricter than the interoperable schemas where the
  approved requirements call for recognized catalogs, lower array maxima,
  destination safety, canonical money, and event temporal policy.
- AppView never fetches authored action, product, event, registration, or email
  destinations.
