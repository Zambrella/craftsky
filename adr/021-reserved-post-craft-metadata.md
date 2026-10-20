# Reserved post craft metadata

- Status: Accepted
- Date: 2026-10-19
- Aspect: Public record contracts

## Decision

Add optional `crafts` to `social.craftsky.feed.post`, containing at most three
`#craft` objects, each with a required, nonempty, bounded `craftType` string.
Reuse the existing craft tokens as open `knownValues`; unfamiliar future
values remain valid. Objects allow optional per-craft context to be added
without changing the array item type. Order conveys no primary craft.

This field is independent of project metadata. Do not infer it from, copy it
to, or require agreement with `project.common.craftType` or `additionalCrafts`.
Its presence does not make an ordinary post a project. Omission and an empty
list both mean no craft metadata supplied. Existing records remain valid.

Accept, validate, publish, and return the field through the existing post API,
including delayed publication. The PDS record and the AppView's existing raw
record JSON retain it; no column, index, collection, or OAuth scope is needed.
Leave the client UI, keyword search, filters, ranking, timelines, tag discovery,
and notifications unchanged. No automatic population or backfill is introduced.
Any future product use requires a separate decision. Ordinary posts omit it.

Regenerate Go JSON/CBOR types and the embedded schema catalog, verify limits
and persistence, then publish the schema update last. This does not deploy the
application or mutate existing owner records.

The immediate post request accepts the following optional field; scheduled
requests accept the same field inside `payload`, and post responses return it
when present:

```json
{
  "crafts": [
    {"craftType": "social.craftsky.feed.defs#sewing"},
    {"craftType": "social.craftsky.feed.defs#embroidery"}
  ]
}
```

## Alternatives and consequences

An array of strings is smaller but cannot gain per-craft context without
changing its type. Reusing the project additional-craft object would couple
independent post metadata to project semantics. Using project metadata directly
would turn this into an active discovery change, which is outside this decision.
The cap of three belongs to the public contract; reconsidering it later requires
the normal schema evolution review.
