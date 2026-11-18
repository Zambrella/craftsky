---
{
  "id": "2026-11-09-integrated-instagram-imports",
  "title": "Integrated Instagram post and project imports",
  "source": "docs/changes/2026-11-09-integrated-instagram-imports.md",
  "implementation": "implemented",
  "archive": "archived",
  "components": [
    "web-client",
    "appview"
  ],
  "topics": [
    "instagram",
    "imports",
    "provenance"
  ],
  "keywords": [
    "Instagram ZIP archive",
    "reels",
    "sourceKey",
    "instagramImport",
    "externalImport",
    "ADR 029",
    "duplicate posting",
    "original dates"
  ],
  "summary": "Replaced the standalone importer with verified web-client archive review, standard/project composer handoff, and private durable duplicate tracking.",
  "currentDocs": [
    "web-client/README.md",
    "docs/adr/029-integrated-instagram-post-imports.md",
    "docs/adr/008-instagram-import-provenance.md",
    "docs/architecture.md"
  ],
  "code": [
    "web-client/src/lib/features/instagram-import",
    "web-client/src/lib/features/settings/InstagramPosts.svelte",
    "appview/internal/instagram/post_import_store.go",
    "appview/internal/api/instagram_post_imports.go"
  ],
  "evidence": [
    "docs/changes/2026-11-09-integrated-instagram-imports.md#validation",
    "docs/changes/2026-11-09-integrated-instagram-imports.md#boundaries"
  ],
  "supersededBy": [],
  "followUps": [
    "Local browser and backend doubles do not establish live PDS publication or production indexing; no deployment was performed."
  ]
}
---

# Integrated Instagram post and project imports

## Outcome and decisions

Verified Instagram settings now review JSON exports locally and open the existing
standard or project composer with full captions and selected media. Reels and
supported videos reuse the normal video path. Users choose up to eight photos or
one video; mixed carousels remain unsupported. Archives stay in browser memory;
selected drafts use the account-scoped local recovery store.

AppView checks active uncontested verification, reserves a private source identity
with command preparation, and records only confirmed publication. Source keys stay
private; public provenance and original dates preserve ADR 008 distribution.
[ADR 029](../../adr/029-integrated-instagram-post-imports.md) records the replacement
of the standalone OAuth/site publisher.

## Evidence and limits

The [validation note](../2026-11-09-integrated-instagram-imports.md#validation) records
the release gates before the main merge and the focused checks run afterward.
The [boundaries](../2026-11-09-integrated-instagram-imports.md#boundaries) distinguish
synthetic browser/backend coverage from live PDS publication and production
indexing. No hosted site, DNS, OAuth grants or production resources were changed.

## Current entry points

Use the [web client guide](../../../web-client/README.md) and
[architecture overview](../../architecture.md). The source-key check covers the
same export item, including caption edits; it does not claim similarity detection
for manually copied posts or substantially changed exports.
