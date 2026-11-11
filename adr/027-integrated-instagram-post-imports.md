# Integrated Instagram post imports

- Status: Accepted
- Date: 2026-11-09
- Scope: Web client, AppView publication and private import history

## Decision

Replace the standalone Instagram post importer with a verified-account flow in
the Svelte web client at `/settings/instagram`. Reuse its local archive parser
safeguards and the existing standard/project composers. Do not support mixed
image/video carousels: users select up to eight photos or one supported video.
Keep source captions intact and require users to resolve character/media limits.

Archives are inspected locally in a dedicated browser worker. The ZIP and its
unselected contents are never uploaded. Retain the archive in browser memory
across composer navigation; account changes, clearing the export and page reload
release that session. Selected media and a private import reference use the
existing account-scoped browser draft/recovery store.

Publish through AppView using the existing server-held OAuth session, normal
media upload paths and durable post-command recovery. The standalone public
OAuth client, direct record writes and bulk rollback UI are removed. This does
not authorize infrastructure deletion or PDS cleanup.

AppView requires an active, verified Instagram account without pending ownership
conflicts for import publication. Verification is a gate for this CraftSky flow,
not proof that a user-selected ZIP belongs to that Instagram account.

Use the existing public `externalImport: {source: "instagram"}` field and the
original source `createdAt`. Preserve ADR 008's historical profile ordering,
search visibility, home-timeline exclusion and source-notification suppression.
No public Lexicon changes are required. Normal post creation still cannot set
its creation date.

Keep source fingerprints and account-scoped publication history private in
AppView. The fingerprint is independent of caption edits and composer kind, so
one source cannot accidentally create both a standard and a project post.
Server publication must guard concurrent attempts as well as ordinary retries;
browser history alone is insufficient. Only confirmed writes are marked posted.
Similarity to manually copied posts is not asserted by this source-key check.

## Consequences

The import UI shares composer validation, project fields, video processing,
accessibility controls and publication recovery instead of maintaining another
publisher. Historical-import metadata must travel through the private draft and
AppView request without leaking source paths or fingerprints into public records.

The earlier standalone implementation documents remain historical records.
This decision supersedes their separate-site publication architecture; ADR 008's
public provenance and distribution rules remain in force.

Removing source files does not retire an existing hosted site, DNS records or
OAuth grants. Any such production changes require a separate requested operation.
