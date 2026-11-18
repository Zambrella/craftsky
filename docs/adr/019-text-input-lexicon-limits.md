## Architecture Decision Record
- Status: Accepted
- Aspect: Record text contracts and authoring
- Date: 2026-10-16
- Decision: Allow 600 graphemes of sewing fit notes and enforce lexicon text limits in authoring inputs

### Why I needed to decide this

A published sewing project with 482 graphemes of fit notes was quarantined by AppView because the lexicon allowed only 300. Several app inputs allowed values longer than their record contracts, so successful authoring did not imply a record could be indexed.

### Options I considered

- Keep 300 graphemes and shorten the record: insufficient space for the requested fit and alteration notes.
- Raise the limit in the existing schema: chosen under the repository's no-active-users compatibility policy.
- Introduce a new project-details NSID: unnecessary before launch and would duplicate the existing variant.

### What I decided

Raise `social.craftsky.project.sewing#details.fitNotes` from 300 to 600 graphemes and from 3000 to 6000 UTF-8 bytes, retaining the existing byte-to-grapheme allowance. Regenerate the embedded schemas and Go artifacts. Older records remain valid; older consumers reject newly permitted values, so consumers must be updated together before release.

Align record-authoring text inputs with both grapheme and byte contracts. Preserve whole Unicode grapheme clusters and active IME composition. Validate restored values and autocomplete selections as well as typed text. Inputs for searches, credentials and other non-record operations retain their own contracts. URI fields retain URI validation rather than acquiring invented character caps.

Production deployment and replay of quarantined records remain separate operations. This decision does not rewrite PDS records or bypass ingestion validation.
