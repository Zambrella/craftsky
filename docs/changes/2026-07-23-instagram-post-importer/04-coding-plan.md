# Coding Plan: Instagram Historical Post Importer

## 1. Inputs And Approval

- Requirements: `01-requirements.md` — approved and amended by document
  review, High risk, 2026-07-23.
- Acceptance tests: `02-acceptance-tests.md` — 49 Must requirements, one
  Should requirement, 23 acceptance criteria, and 60 unique test definitions.
- Document review: `03-document-review.md` — Approved with notes, High risk,
  2026-07-23.
- Repository guidance: `AGENTS.md`,
  `atproto-craft-social-app-reference.md`, the AppView API architecture
  specification, and the project `atproto-lexicon` skill.
- Current external contracts consulted:
  - AT Protocol browser OAuth and granular permission specifications.
  - `@atproto/oauth-client-browser` and `@atproto/api` browser-agent flow.
  - zip.js Blob-backed `ZipReader`, entry generator, writer, cancellation, and
    ZIP64 behavior.
  - Vite module workers and environment modes.
  - Dexie versioned IndexedDB tables and transactions.
  - Playwright Vite `webServer`, routing, network observation, and persistent
    storage behavior.
- Implementation approval: explicit. The user asked for the existing workflow
  to proceed through implementation without intermediate review waits.
- Git authority: no new staging, commit, push, or pull-request action is
  authorized by this implementation request.

## 2. Implementation Strategy

Implement one vertical contract and four independently testable layers:

1. Add a minimal optional `externalImport` object to
   `social.craftsky.feed.post` through ADR 008. Generate the Go contract and a
   web contract from the Lexicon and make drift a failing test.
2. Extend AppView persistence and reads with exact Instagram provenance and a
   separate `profile_sort_at`. Imported originals remain indexable and
   searchable, are excluded only from the authored home-timeline arm, and do
   not activate source-record notifications. Ordinary records, later
   reposts/quotes, and later engagement retain existing behavior.
3. Propagate provenance through canonical and quoted-post response shapes into
   Flutter. Render one subtle localized label from the shared post-card and
   quote-preview widgets, with no label when the optional field is absent or
   unknown.
4. Add a standalone `instagram-importer/` Vite/React/TypeScript package. A
   dedicated module worker inspects Blob-backed ZIP/ZIP64 archives, returns a
   minimized review manifest, and sanitizes one selected image at a time.
   OAuth starts only after local review. The authenticated browser agent checks
   `social.craftsky.actor.profile/self`, confirms the destination/counts, and
   writes deterministic create-only posts directly to the PDS.
5. Keep importer progress and OAuth persistence separate. Dexie stores only
   DID-bound fingerprints, item states, safe codes, deterministic rkeys, and
   AT URIs. The OAuth library exclusively owns its session material. Resume,
   retry, cancel, clear, and rollback operate on explicit per-session state.
6. Build production as a static Cloudflare Pages artifact for
   `import.craftsky.social`. Localhost has a loopback OAuth client identity.
   Stable staging is separately configurable. Ephemeral previews compile with
   real OAuth and PDS writes hard-disabled.

The first red test is `IT-001`: the current post schema/generated types do not
contain the approved minimal provenance contract or ADR.

## 3. Architectural Boundaries

### Public record

```json
{
  "$type": "social.craftsky.feed.post",
  "text": "final reviewed caption",
  "createdAt": "2024-01-02T03:04:05.000Z",
  "externalImport": {
    "source": "instagram"
  }
}
```

`externalImport` is optional and self-asserted. `source` has an open
`knownValues` declaration whose only v1 recognized token is `instagram`.
Neither the public record nor AppView persistence contains an Instagram
handle, media ID, post ID, filename, archive name/fingerprint, caption hash, or
external URL.

### AppView classification

The authoritative indexed record controls provenance:

- exact `externalImport.source == "instagram"`:
  `external_import_source = 'instagram'` and
  `profile_sort_at = record.createdAt`;
- absent or unknown source: `external_import_source = NULL` and
  `profile_sort_at = indexed_at`;
- a replacement retaining the exact field stays imported;
- a replacement omitting it clears the classification;
- a delete removes the row through the existing lifecycle.

AppView does not infer provenance from timestamps, deterministic keys, or old
database state.

### Local-first browser boundary

```text
File object
  -> dedicated module worker
     -> zip.js BlobReader central-directory iteration
     -> supported post JSON entries only
     -> versioned shape adapters
     -> minimized normalized manifest
  -> virtualized review/edit/select
  -> OAuth + profile/self preflight
  -> exact account/count confirmation
  -> worker sanitizes one selected image
  -> authenticated PDS blob upload(s)
  -> create-only post record
  -> content-free Dexie progress update
```

The raw File remains a browser-owned Blob. It is never converted to one whole
`ArrayBuffer`, persisted, cached, logged, placed in a URL, or sent over the
network.

## 4. Affected Areas

| Area | Existing Pattern | Planned Change | Requirement IDs | Test IDs |
|---|---|---|---|---|
| Lexicon / ADR | Post schema has no import classification | Optional minimal `externalImport` object, ADR 008, regenerated Go and web contract | FR-018, FR-019, NFR-006 | IT-001, UT-009, REG-001 |
| Post persistence | `craftsky_posts` orders profiles by `indexed_at` | Nullable exact source plus non-null `profile_sort_at`, reversible migration and author/profile index | FR-020, NFR-008 | IT-009, IT-010, IT-011, REG-001, REG-002 |
| Indexing / notification activation | Every indexed post activates post-derived notifications | Classify exact Instagram records and skip only source-record notification activation while retaining mention/reply/quote indexing | FR-020, FR-022, RULE-004 | IT-010, IT-013, REG-004, REG-006 |
| Timeline / search | Authored and repost arms are separate; search already uses post dates | Filter imports only from the authored home-timeline arm; retain later repost/quote and ordinary search behavior | FR-020, FR-021, RULE-004 | IT-011, IT-012, IT-014, REG-002, REG-003, REG-005 |
| AppView responses | Canonical and quote-preview DTOs have no provenance | Add optional `externalImport` to both shapes through shared mapping | FR-020, FR-023 | IT-010, IT-015, REG-001 |
| Flutter models / UI | Shared post card and separate quote preview | Optional decoded provenance and one localized subtle label in both surfaces | FR-023 | UT-015, IT-015, REG-007 |
| Importer build / modes | No Node workspace or importer | Standalone checked npm package, Vite/Vitest/Playwright, local/staging/production/preview policy, static headers | FR-001, FR-003, NFR-004, NFR-007 | UT-013, IT-016, AT-010, REG-008 |
| ZIP worker | No web archive parser | Header-first EOCD/ZIP64 envelope validation before zip.js allocation, Blob-backed enumeration, target-only extraction, typed operation protocol, cancellation and safety envelope | FR-005, FR-006, FR-007, NFR-001, NFR-002, NFR-003 | UT-001, UT-002, UT-018, IT-002, IT-003 |
| Export adapters | No historical post normalization | Legacy and detailed adapters, explicit draft/unpublished skip, wrapper handling, ordering, dedupe/conflict policy | FR-007, FR-008, FR-009 | UT-002, UT-003, IT-002 |
| Text / identity | No import caption transformation | Timestamp validation, reversible mojibake repair, 2,000-grapheme bound, URL/hashtag UTF-8 facets, deterministic TID | FR-013, FR-014, FR-015, FR-018, FR-024 | UT-004, UT-005, UT-006, UT-008 |
| Media | No browser media pipeline | Header-first dimension/animation/signature inspection, one-at-a-time review preflight and publish-time decode/re-encode, metadata removal, limits, first-four policy, empty alt text | FR-011, FR-012, FR-016, FR-017, RULE-010 | UT-007, UT-019, IT-004 |
| Review UI | No importer UI | Public-content acknowledgement, virtualized/filterable/selectable review, editable captions, warnings/counts, destination confirmation | BR-001, BR-004, FR-010 | AT-001, AT-004, IT-008 |
| OAuth / PDS | Flutter-mediated AppView session model | Separate approved browser OAuth client, exact permissions, profile/self preflight, create/delete-only publisher | FR-002, FR-003, FR-004, FR-017, FR-018 | AT-003, AT-005, IT-006, IT-007 |
| Progress / recovery | No importer state | Content-free Dexie session/item state, DID/fingerprint binding, retry/cancel/resume/clear/rollback ownership | FR-024, FR-025, FR-026, FR-027, FR-028 | UT-010, UT-011, UT-012, UT-017, IT-005, IT-007, IT-018 |
| Privacy / diagnostics | No importer boundary | Closed safe-error union, no telemetry, observed network/storage canaries, redacted strings | BR-002, FR-009, NFR-001, NFR-005, RULE-008 | AT-002, UT-012, IT-003, IT-017 |

## 5. Files And Modules

Generated Go, web-contract, Dart mapper, and Flutter localization outputs are
generated from source and are not hand-edited.

### Lexicon and AppView

| Path | Create / Change | Purpose |
|---|---|---|
| `adr/008-instagram-import-provenance.md` | Create | Record the optional self-asserted field, alternatives, evolution, privacy, distribution, and compatibility decision. |
| `lexicon/social/craftsky/feed/post.json` | Change | Add optional `externalImport` ref and `#externalImport` object with required bounded `source`. |
| `appview/internal/lexicon/craftsky/feedpost.go` | Generate | Add generated Go post field/object. |
| `appview/internal/lexicon/craftsky/feedpost_import_test.go` | Create | Validate optional/valid/invalid source shapes and generation contract. |
| `appview/migrations/000032_instagram_post_imports.up.sql` | Create | Add nullable source, `profile_sort_at`, backfill, checks, and profile ordering index after the merged main migration sequence. |
| `appview/migrations/000032_instagram_post_imports.down.sql` | Create | Remove the new indexes/columns reversibly. |
| `appview/internal/db/instagram_import_migration_test.go` | Create | Exercise migrations through 000031, then 000032 up/down/up against real Postgres. |
| `appview/internal/index/craftsky_post.go` | Change | Unmarshal the generated `craftskylex.FeedPost`, extract exact source, choose profile sort timestamp, persist both fields, and suppress only original-import notification activation; retain raw JSON separately only for existing exact-record preservation. |
| `appview/internal/index/craftsky_post_import_test.go` | Create | Cover create, retaining replacement, omitting replacement, unknown source, delete, and safe notification behavior. |
| `appview/internal/api/post_store.go` | Change | Select/scan source and profile cursor fields; order author profile reads by `profile_sort_at`. |
| `appview/internal/api/post_response.go` | Change | Add optional external-import DTO and map it into canonical and quote-preview responses. |
| `appview/internal/api/timeline_store.go` | Change | Exclude exact Instagram rows only in the original authored-post arm. |
| `appview/internal/api/imported_post_store_test.go` | Create | Exercise imported/ordinary profile cursors and the `(did, profile_sort_at DESC, uri DESC)` query plan against real Postgres. |
| Existing AppView post/timeline/search/notification tests | Change | Add imported/ordinary controls and positional-scan regression coverage. |

### Standalone importer

| Path | Create / Change | Purpose |
|---|---|---|
| `instagram-importer/package.json`, `package-lock.json` | Create | Independent scripts/dependencies and reproducible npm install. |
| `instagram-importer/tsconfig*.json`, `vite.config.ts`, `vitest.config.ts`, `playwright.config.ts` | Create | Strict browser/worker types, module-worker build, unit/component tests, and Chromium acceptance harness. |
| `instagram-importer/index.html`, `src/main.tsx`, `src/app/App.tsx` | Create | Accessible SPA entry and state-machine composition root. |
| `instagram-importer/src/styles/*.css` | Create | CraftSky warm-paper/cobalt/hard-rule design tokens, responsive layout, focus/contrast/motion states, and self-hosted fonts. |
| `instagram-importer/src/config/safety.ts` | Create | Named `instagram-import-v1` safety limits and inclusive helpers. |
| `instagram-importer/src/config/runtime.ts` | Create | Local, stable-staging, production, and preview origin/OAuth/write policy. |
| `instagram-importer/src/privacy/safeDiagnostics.ts` | Create | Closed content-free error codes and redacted diagnostic/string boundary. |
| `instagram-importer/src/generated/craftsky-feed-post.ts` | Generate | Lexicon-derived collection/type/max/source constants and web wire types. |
| `instagram-importer/scripts/generate-lexicon-contract.mjs` | Create | Deterministically generate/check the web contract from the repository Lexicon. |
| `instagram-importer/src/domain/types.ts` | Create | Minimized worker/review/publication types with no raw source identity. |
| `instagram-importer/src/domain/timestamps.ts` | Create | Strict seconds/milliseconds normalization and approved inclusive date window. |
| `instagram-importer/src/domain/recordKey.ts` | Create | Stable normalized identity hash and deterministic `TID.fromTime` clock ID. |
| `instagram-importer/src/archive/paths.ts` | Create | Optional wrapper plus exact supported post JSON/media path validation and conflict detection. |
| `instagram-importer/src/archive/zipEnvelope.ts` | Create | Parse classic EOCD and ZIP64 locator/EOCD with `BigInt`, reject multi-disk/ambiguous layouts, and enforce entry/directory bounds before constructing zip.js. |
| `instagram-importer/src/archive/adapters/legacy.ts` | Create | Normalize legacy `posts_N.json` media-object shape. |
| `instagram-importer/src/archive/adapters/detailed.ts` | Create | Normalize label/value detail shape, reverse nested carousel order, ignore duplicate cover, and skip explicit draft/unpublished items. |
| `instagram-importer/src/archive/normalize.ts` | Create | Merge/deduplicate equivalent variants and emit skipped ambiguity items. |
| `instagram-importer/src/text/caption.ts`, `facets.ts` | Create | Reversible repair, grapheme truncation, and byte-accurate link/tag facets without mention resolution. |
| `instagram-importer/src/media/inspect.ts`, `policy.ts`, `sanitize.ts` | Create | Parse JPEG/PNG/WebP headers before decode, reject animated PNG/WebP, settle review eligibility, and perform one-at-a-time OffscreenCanvas re-encode with metadata stripping and final limit checks. |
| `instagram-importer/src/worker/protocol.ts`, `archive.worker.ts`, `client.ts` | Create | Typed operation IDs, inspect/sanitize/cancel messages, stale-result fencing, and Vite module-worker construction. |
| `instagram-importer/src/review/reviewState.ts` | Create | Default selection, per-post/media toggles, bulk operations, filters, edits, warnings, confirmations, and exact aggregate counts. |
| `instagram-importer/src/progress/database.ts`, `repository.ts`, `fingerprint.ts`, `state.ts` | Create | Versioned Dexie schema and content-free DID/session/item state transitions. |
| `instagram-importer/src/auth/oauthConfig.ts`, `authService.ts` | Create | Exact scope, BrowserOAuthClient lifecycle, granted-authority check, sign-out, and profile/self preflight. |
| `instagram-importer/src/pds/postRecord.ts`, `publisher.ts`, `retry.ts` | Create | Wire-record validation, lazy image upload, create-only collision handling, bounded retry, cancellation, and scoped rollback. |
| `instagram-importer/src/app/components/*` | Create | Upload dropzone, privacy notice, review list/row, filters/counts, auth/account confirmation, progress, completion/recovery, and safe error surfaces. |
| `instagram-importer/public/oauth-client-metadata.json` | Create/Generate | Canonical production client metadata; stable staging uses separately configured metadata. |
| `instagram-importer/public/_headers`, `_redirects`, `robots.txt` | Create | Cloudflare static security baseline, SPA fallback, and noindex policy where configured. |
| `instagram-importer/README.md` | Create | Local commands, OAuth/PDS setup, origin modes, Pages deployment, limits, privacy boundary, and manual gates. |
| `instagram-importer/src/**/*.test.ts(x)`, `e2e/*.spec.ts` | Create | Synthetic-only unit, integration, component, network/privacy, and recovery tests from `02-acceptance-tests.md`. |

### Flutter and repository entry points

| Path | Create / Change | Purpose |
|---|---|---|
| `app/lib/feed/models/post.dart`, `app/lib/feed/models/post.mapper.dart` | Change / Generate | Decode optional full-post and quote-preview provenance. |
| `app/lib/feed/widgets/post_card.dart` | Change | Render localized label in the full post card and quote preview without changing ordinary layout/actions. |
| `app/lib/l10n/app_en.arb`, generated `app/lib/l10n/app_localizations*.dart` | Change / Generate | Add concise “Imported from Instagram” copy and semantics. |
| `app/test/feed/models/post_test.dart`, `app/test/feed/widgets/post_card_test.dart`, `app/test/l10n/imported_post_l10n_test.dart` | Change / Create | Full/quote imported, absent, unknown, localized-copy, and ordinary regression fixtures. |
| `justfile` | Change | Add importer install, test, typecheck, lint, build, generation-check, and e2e recipes without changing existing app commands. |
| `.gitignore` | Change | Ignore importer `node_modules`, build, coverage, Playwright, and local environment outputs. |

## 6. Core Interfaces And State

### Safety policy

```ts
export interface ImportSafetyPolicy {
  readonly id: 'instagram-import-v1'
  readonly maxEntries: 100_000
  readonly maxCentralDirectoryBytes: 64 * MiB
  readonly maxCandidateJsonBytes: 32 * MiB
  readonly maxCombinedCandidateJsonBytes: 128 * MiB
  readonly maxPosts: 25_000
  readonly maxSourceImageBytes: 64 * MiB
  readonly maxDecodedPixels: 25_000_000
  readonly maxDimension: 12_000
  readonly maxCompressionRatio: 200
  readonly maxConcurrentImageTransforms: 1
  readonly maxFinalBlobBytes: 15 * MiB
}
```

There is no whole-ZIP-size field. “Any ZIP size” means no importer-defined
overall cap within File API/platform limits.

The worker reports enumeration progress at least every 250 entries or 50 ms,
whichever occurs first, and checks cancellation before/after every entry,
candidate extraction, normalized post, media read, decode, canvas encode, and
result transfer. Component tests use a 100 ms controlled-worker cancellation
budget; device/browser performance remains `MAN-003`.

### Minimized manifest

```ts
interface ReviewPost {
  itemKey: string          // local deterministic identity, never public
  rkey: string             // deterministic TID
  createdAt: string
  caption: string
  media: ReviewMedia[]     // bounded path token + type, at most four selected
  warnings: SafeWarningCode[]
  selected: boolean
  needsTextOnlyConfirmation: boolean
}

interface SkippedPost {
  itemKey: string
  createdAt?: string
  code: SafeSkipCode
}

interface ReviewManifest {
  schemaVersion: 1
  fingerprint: string
  posts: ReviewPost[]
  skipped: SkippedPost[]
  counts: ReviewCounts
}
```

No raw adapter object crosses the worker boundary. `itemKey`, media lookup
tokens, and fingerprint are one-way digests derived locally. Worker-held path
lookups live only for the active File and are reconstructed after reload.
Warnings are a closed, deduplicated, bounded code set with aggregate counts;
no raw source value or unbounded per-entry message crosses the boundary.
Thumbnail/object URLs exist only for the active visible item and are revoked on
replacement, deselection, unmount, and worker result release. Worker media
buffers use transferable ownership rather than structured-clone duplication.

### Progress state

```ts
interface ImportSessionRow {
  id: string
  schemaVersion: 1
  manifestFingerprint: string
  did: string
  status: 'reviewed' | 'running' | 'paused' | 'complete' | 'rollingBack'
  createdAt: string
  updatedAt: string
}

interface ImportItemRow {
  sessionId: string
  itemKey: string
  rkey: string
  status:
    | 'remaining'
    | 'running'
    | 'created'
    | 'alreadyExisting'
    | 'collision'
    | 'failed'
    | 'rolledBack'
    | 'rollbackConflict'
  atUri?: string
  createdCid?: string
  safeCode?: SafePublicationCode
  attempts: number
}
```

Only `created` rows with both the returned AT URI and CID are rollback-owned by
that session. `alreadyExisting`, including an exact imported record discovered
after a create response was lost before local persistence, is deliberately
unowned. The tables contain no archive path/name, caption, source JSON, image,
thumbnail, hash of caption, handle, OAuth value, or raw error. Clearing a
session removes resume and bulk-rollback ownership after explicit warning.

### Publication sequence

Before OAuth and final account/count confirmation, the worker preflights every
candidate selected-by-default image one at a time. It parses format headers
before browser decode, rejects excessive dimensions/pixels and animation,
decodes/re-encodes to prove the supported final shape, records warnings and
final image counts, and immediately releases the transformed blob. This phase
settles omissions, empty posts, and explicit text-only confirmations without
retaining public image bytes. User selection changes recompute the confirmed
counts from these preflight results.

For one finally confirmed post:

1. Read the deterministic target record. If absent, continue. If present and
   it is the exact expected imported record, mark `alreadyExisting` without
   claiming it for this session. Any other record is a `collision` and is never
   overwritten.
2. Request each selected source image from the worker again. Repeat header
   inspection and sanitization, require the preflight-confirmed shape, upload,
   release local source/sanitized buffers, then process the next image.
3. If any finally confirmed image fails, mark the post failed and do not create
   a reduced record.
4. Generate facets from the final edited caption and validate the generated
   post wire shape. Re-run the 2,000-grapheme and 20,000-byte text limits after
   every edit before final counts and record construction.
5. Call create-record with the deterministic rkey and no update fallback.
6. Persist `created`, the returned AT URI, and returned CID transactionally.
   If the create result is network-ambiguous or the page closes before that
   durable write, a later exact record is treated as unowned
   `alreadyExisting`; safety takes priority over bulk rollback completeness.

Transient network, server, and rate-limit failures retry at most three times
with jittered delays capped at 30 seconds and honor `Retry-After`. Validation,
authority, collision, unsupported media, and other permanent failures do not
retry automatically. Pause/cancel stops scheduling new posts and aborts the
current request where the library permits; it never deletes successes.

Rollback iterates only `created` rows owned by the selected session, verifies
the exact DID/collection/rkey target, and calls delete-record with
`swapRecord: createdCid`. An edited/replaced or CID-different
delete-then-recreated record fails the CID precondition, becomes
`rollbackConflict`, and is never deleted. Because AT record CIDs identify
content rather than record incarnations, a byte-identical recreation at the
same rkey cannot be distinguished; the rollback confirmation discloses that
residual limitation.
The session clears only when every owned record is deleted or already absent.

## 7. Build And Deployment Security Baseline

Runtime modes are explicit:

- `local`: localhost loopback OAuth client identity; writes allowed.
- `staging`: exact configured stable HTTPS origin and client metadata; writes
  allowed only when `VITE_ENABLE_PDS_WRITES=true`.
- `production`: exact `https://import.craftsky.social`; canonical metadata;
  writes allowed.
- `preview`: any other deployed origin; OAuth initiation, session restoration,
  PDS preflight, upload, create, and delete throw `previewWritesDisabled`.

Minimum emitted header policy:

```text
Content-Security-Policy:
  default-src 'self';
  script-src 'self';
  style-src 'self';
  style-src-attr 'unsafe-inline';
  font-src 'self';
  img-src 'self' blob: data:;
  connect-src 'self' https:;
  worker-src 'self' blob:;
  object-src 'none';
  base-uri 'none';
  form-action 'self';
  frame-ancestors 'none';
  manifest-src 'self'
Referrer-Policy: no-referrer
X-Content-Type-Options: nosniff
Permissions-Policy: camera=(), microphone=(), geolocation=(),
                    payment=(), usb=(), browsing-topics=()
Cross-Origin-Opener-Policy: same-origin-allow-popups
```

`style-src-attr 'unsafe-inline'` is the narrow static-origin exception required
for TanStack Virtual's runtime height and transform attributes. It does not
permit inline scripts or inline `<style>` blocks; React escapes member content,
and no imported HTML is rendered. `same-origin-allow-popups` keeps the
standards-based OAuth popup attached long enough to return its callback while
retaining same-origin isolation for unrelated windows.

Dynamic OAuth/PDS destinations require `https:`, forbid credentials/fragments
and IP-literal/local-network hosts outside explicit localhost development, and
must match an origin returned by authenticated OAuth/session discovery before
the application sends archive-derived public content. Playwright observes all
requests and fails for any unapproved destination or pre-confirmation write.
Because compatible PDSs may use arbitrary HTTPS origins, `connect-src https:`
is intentionally scheme-wide and is not claimed as an exfiltration boundary
after same-origin script compromise; application audience/origin validation,
no third-party script, dependency review, and observed-network tests enforce
the practical destination boundary.

No analytics, session replay, service worker, third-party font/CDN resource, or
source-map publication is added.

Production writes remain disabled until the deployment order is complete:

1. deploy migration 000032;
2. deploy the generated-schema-aware indexer, profile ordering, timeline
   exclusion, notification suppression, and provenance responses;
3. deploy the Flutter provenance label where practical;
4. deploy/enable `import.craftsky.social` OAuth and PDS writes.

Preview builds remain write-disabled at every stage.

## 8. TDD Execution Order

Each step starts with the named failing test, applies the smallest production
change, runs the focused test, refactors while green, and records evidence in
`05-implementation-plan.md`.

1. Contract and AppView:
   `IT-001`, `UT-009`, `IT-009`, `UT-014`, `IT-010`, `IT-011`,
   `IT-012`, `IT-013`, `IT-014`, `REG-001`, `REG-002`, `REG-003`,
   `REG-004`, `REG-005`, `REG-006`.
2. Flutter response and label:
   `UT-015`, `IT-015`, `REG-007`.
3. Importer scaffold/config:
   `UT-001`, `UT-013`, `IT-016`, `REG-008`.
4. Archive and domain:
   `UT-002`, `UT-003`, `UT-004`, `UT-005`, `UT-006`, `UT-007`,
   `UT-008`, `UT-016`, `UT-018`, `UT-019`, `IT-002`, `IT-003`,
   `IT-004`.
5. Progress/auth/publication:
   `UT-010`, `UT-011`, `UT-012`, `UT-017`, `IT-005`, `IT-006`,
   `IT-007`.
6. Browser acceptance and recovery:
   `AT-001`, `AT-002`, `AT-003`, `AT-004`, `AT-005`, `AT-006`,
   `AT-007`, `AT-008`, `AT-009`, `AT-010`, `IT-008`, `IT-017`,
   `IT-018`.
7. Final cross-stack regressions:
   `REG-008` plus the complete Go, Flutter, importer unit/component, build,
   generation-check, and available Playwright suites.

The full explicit test set is:

- Acceptance: `AT-001`, `AT-002`, `AT-003`, `AT-004`, `AT-005`,
  `AT-006`, `AT-007`, `AT-008`, `AT-009`, `AT-010`.
- Unit: `UT-001`, `UT-002`, `UT-003`, `UT-004`, `UT-005`, `UT-006`,
  `UT-007`, `UT-008`, `UT-009`, `UT-010`, `UT-011`, `UT-012`,
  `UT-013`, `UT-014`, `UT-015`, `UT-016`, `UT-017`, `UT-018`,
  `UT-019`.
- Integration: `IT-001`, `IT-002`, `IT-003`, `IT-004`, `IT-005`,
  `IT-006`, `IT-007`, `IT-008`, `IT-009`, `IT-010`, `IT-011`,
  `IT-012`, `IT-013`, `IT-014`, `IT-015`, `IT-016`, `IT-017`,
  `IT-018`.
- Regression: `REG-001`, `REG-002`, `REG-003`, `REG-004`, `REG-005`,
  `REG-006`, `REG-007`, `REG-008`.
- Manual release checks: `MAN-001`, `MAN-002`, `MAN-003`, `MAN-004`,
  `MAN-005`.

## 9. Verification Commands

Focused commands will be replaced with exact package/test names as files land:

```sh
just lexgen
just lexgen-check
cd appview && go test ./internal/lexicon/craftsky ./internal/index ./internal/api
cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test -race ./...
cd app && flutter test test/feed/models/post_test.dart test/feed/widgets/post_card_test.dart
cd app && flutter analyze
cd instagram-importer && npm ci
cd instagram-importer && npm run generate:check
cd instagram-importer && npm run test
cd instagram-importer && npm run typecheck
cd instagram-importer && npm run lint
cd instagram-importer && npm run build
cd instagram-importer && npm run test:e2e
```

Before handoff:

- run `git diff --check`;
- audit that every test ID above has implementation evidence or an explicit
  external/manual status;
- scan importer output/source for telemetry, remote fonts, raw fixture data,
  archive persistence, wildcard/update OAuth authority, and preview writes;
- inspect generated static metadata and headers;
- inspect migration up/down, query plans, and positional row scans;
- run the broad Go, Flutter, and importer gates that the local environment can
  execute.

## 10. External And Manual Release Gates

Repository completion does not imply these external checks have passed:

- `MAN-001`: live OAuth consent, granted-authority inspection, profile/self,
  create, and delete against compatible test PDSs.
- `MAN-002`: additional consented current Instagram export shapes recreated as
  synthetic fixtures, plus Chrome/Edge/Firefox/Safari compatibility.
- `MAN-003`: generated very large ZIP64 files, browser suspension/resume,
  memory observation, and real PDS rate limiting.
- `MAN-004`: real PDS to relay/Tap/AppView convergence and live Flutter
  profile/search/timeline/notification behavior.
- `MAN-005`: deployed Cloudflare Pages origin/project, headers, CSP, metadata,
  network isolation, and rollback.

These remain marked pending until performed. The implementation must not claim
live interoperability or deployment verification without that evidence.

## 11. Out Of Scope

- Instagram account-ownership verification.
- Following/follower import, Instagram messaging, or AppView migration routes.
- A Flutter archive parser/import flow.
- Video publication, story import, reels-specific behavior, comments, likes,
  location, music, collaborators, or Instagram identity mapping.
- AppView-mediated importer OAuth or PDS writes.
- Post update permission, synchronization, or overwrite-on-rerun.
- Alt-text editing or warnings in v1.
- Server-side archive uploads, queues, analytics, telemetry, or importer
  accounts.
- Guaranteeing undocumented future Instagram export compatibility.
