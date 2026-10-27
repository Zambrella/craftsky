# Published post editing

- Status: Accepted
- Date: 2026-10-26
- Aspect: Public record contracts and current AppView state

## Decision

Allow authenticated authors to update the content of their published
`social.craftsky.feed.post` at any age through AppView-mediated conditional
updates to the existing record. Preserve the DID, collection, record key,
original publication timestamp, general/project classification, and complete
reply/quote references including their presence. Other users' interaction
records are never rewritten. Reposts and quotes resolve the latest visible
original by URI, even when their stored strong reference has an earlier CID.

Add optional `editedAt`, a string with Lexicon `datetime` format, to the post
record. It declares the latest content edit time and is not tamper-proof
evidence. Existing records without it remain valid. Malformed datetime values or wrong
field types make an external record invalid and follow existing rejection
rules; they do not receive an observed-time fallback. The maintainer confirmed
this strict-validation edge case on 2026-10-27. CraftSky chooses a stable
server timestamp for each successful logical edit; no-op, failed and replayed
requests do not manufacture further edit times. All record replacements
preserve unknown fields and retain original timestamp strings.

AppView stores only current-version edit metadata. Its presentation separates
server-confirmed, externally reported and locally observed edit times. A
first-seen current snapshot without valid metadata does not prove a previous
edit. Observation metadata need not survive a rebuild from public records
alone. There is no revision collection, public history API or old-media archive.
Existing command recovery and moderation evidence keep their normal policies.

Record writes stay server-side, under existing session, lifecycle, media and
moderation controls. Require a concrete expected record CID and an immutable
operation key; an edit must not recreate a deleted record or overwrite a newer
version. Existing OAuth scopes already authorize the collection; no broader
credential handoff is introduced.

Edits do not generate social alerts. Preserve bounded operational facts about
original publication recipients before current-source coalescing, intersect
them with current applicable reasons, and never add recipients introduced only
by an edit. Removed pending reasons suppress dispatch; already dispatched
notifications cannot be recalled. Retain existing safety and eligibility gates.

## Alternatives and consequences

Immutable revisions would preserve original quote context but require the
history feature expressly excluded by the product decision. Deleting and
reposting would fragment identity, engagement and conversations. AppView-only
timestamps would not provide portable self-declared edit metadata.

The optional field is additive; required fields and existing types/limits do
not change. Regenerate JSON/CBOR types and the embedded schema catalog and
verify both old and edited records. Publishing schemas or deploying services
is a separate release action and is not authorized by this decision.

The maintainer approved the coding plan and implementation and explicitly
authorized using the repository's ADR format in place of the unavailable
`writing-architecture-decision-records` skill. The project atproto-lexicon
skill still applies.
