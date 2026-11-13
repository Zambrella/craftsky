---
{
  "id": "2026-10-23-edit-posts",
  "title": "Published post editing and version reconciliation",
  "source": "docs/changes/2026-10-23-edit-posts",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "app",
    "appview",
    "lexicon"
  ],
  "topics": [
    "posts",
    "editing",
    "concurrency"
  ],
  "keywords": [
    "edit seed",
    "same-CID observation",
    "quote revision",
    "uncertain save",
    "materials and style"
  ],
  "summary": "Added owner editing of published posts with frozen uncertain writes, record-version ordering and safe typed prefills.",
  "currentDocs": [
    "docs/adr/022-published-post-editing.md",
    "docs/adr/023-quote-preview-image-selection.md",
    "docs/development/web-client-publishing.md"
  ],
  "code": [
    "appview/internal/postpublication",
    "appview/internal/pdscommands",
    "app/lib/feed",
    "app/lib/projects/composer"
  ],
  "evidence": [
    "docs/changes/2026-10-23-edit-posts/05-implementation-plan.md#ir-011-final-verification",
    "docs/changes/2026-10-23-edit-posts/05-implementation-plan.md#materials-and-style-typed-prefill-correction-2026-10-28",
    "docs/changes/2026-10-23-edit-posts/06-implementation-review.md"
  ],
  "supersededBy": [],
  "followUps": [
    "IR-003 / MAN-001 / MAN-002 platform validation remains explicitly pending in the final implementation review."
  ]
}
---

# Published post editing and version reconciliation

Published edits use an owner-authorized seed and version-aware PDS command rather than creating a new post or overwriting from an incomplete public preview. The editor preserves input through uncertain saves and retries the same frozen command; current observation ordering distinguishes cached rendering from fresh same-CID evidence.

Multiple approved simplifications and correction passes close implementation findings, including partial/full quote handoff and materials/style typed-prefill failures. Later refinements place edit labels and combine post-time disclosure without changing record identity. The last review accepts the code corrections and keeps Changes required solely for required manual platform validation.

Automated fixtures and full/focused gates do not replace those manual checks or prove live PDS reconciliation on all devices. Current post-editing and quote-preview ADRs define active semantics. Use this package for command identity, immutable editor state and why a cached or partial quote cannot fabricate a complete authoritative edit seed.

Evidence: [05-implementation-plan.md](../2026-10-23-edit-posts/05-implementation-plan.md#ir-011-final-verification); [06-implementation-review.md](../2026-10-23-edit-posts/06-implementation-review.md). Current entry points are listed in the metadata; consult maintained guidance before applying historical rules.

Section pointers: [materials-and-style-typed-prefill-correction-2026-10-28](../2026-10-23-edit-posts/05-implementation-plan.md#materials-and-style-typed-prefill-correction-2026-10-28).
