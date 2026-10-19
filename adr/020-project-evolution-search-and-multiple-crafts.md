# Project evolution, discovery, and additional crafts

- Status: Accepted
- Date: 2026-10-19
- Aspect: Public record contracts and application taxonomy

## Decision

Retain one required primary `project.common.craftType` and add optional
`additionalCrafts`, an array of at most ten `#additionalCraft` objects with a
required `craftType`. The primary craft determines the existing single details
variant. Additional crafts participate in discovery without owning additional
detail variants. First-party authoring rejects duplicate/primary additional
crafts and mismatched recognized detail types; unfamiliar open-union variants
remain preserved. Existing records require no rewrite. No new collection or
OAuth permission is introduced.

Preserve public extension fields through API and Flutter model round trips,
including known objects, and preserve unknown generated union variants in JSON
and CBOR. Keep source JSON for projection rather than reconstructing records
from generated structs that do not retain unknown object properties.

Increase image alt text bytes from 1,000 to 10,000 while retaining 1,000
graphemes, consistently for post and business images. This is a deliberate
prelaunch compatibility change under AGENTS.md; older schemas reject newly
permitted values. ADRs 018 and 019 already authorize the pending eight-image
and 600-grapheme fit-note changes. Publish all reviewed schema updates only
after implementation and validation. This does not deploy AppView or rewrite
owner PDS records.

Maintain a versioned application taxonomy of existing stable record values,
English labels, aliases, and parent/equivalent relationships. Broader concepts
are directional; related types are not all synonyms. Taxonomy changes do not
rewrite public values. Generated Flutter data and embedded Go data share the
same catalog. A searchable-field contract covers every project property,
explicit exclusions, weights, visibility, and catalog version. Hashtag counts
and discovery use all-time visible posts, with one count per post and tag.

## Alternatives and consequences

An unordered craft list would make the detail variant ambiguous and require
changing existing authoring and display semantics. A primary plus additional
list preserves that meaning while making mixed-craft posts discoverable.
Separate detail objects per craft can be added later if demonstrated needs
justify them. Material identity, pattern identity, and mutable project lifecycle
remain unchanged.

All AppView reads retain moderation, blocks, mutes, owner lifecycle, language,
and hidden-media gates. No unknown/private record fields are made searchable;
only explicitly contracted public fields are indexed. Search relevance remains
separate from chronological feeds and does not favor business or paid accounts.
