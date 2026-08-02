# Coding Plan: Scheduled Posts

**Date:** 2026-07-31  
**Status:** Ready for implementation approval  
**Risk:** High  
**Implementation approval:** Required before any source, test, dependency, migration, generated-code, or infrastructure change is made

## 1. Inputs

This plan implements the approved workflow contract in:

- `01-requirements.md`
- `02-acceptance-tests.md`
- `03-document-review.md`

It also follows the current repository architecture in:

- `atproto-craft-social-app-reference.md`
- `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md`
- `docs/superpowers/specs/2026-04-22-api-wire-alignment-design.md`
- `AGENTS.md`

The current highest migration is `000033_post_languages`; the implementation should use `000034_scheduled_posts` if that is still the next number when work begins. No lexicon change is planned.

The implementation must preserve these approved boundaries:

- Only original top-level standard posts and project posts can be scheduled. Quotes, replies, and comments cannot.
- New composers default to `Now` and continue to use the existing immediate `POST /v1/posts` path unless the member deliberately selects `Schedule for later`.
- All unpublished payload and durable media remain private AppView data.
- At most three active unpublished resources exist per owner DID across Scheduled, Publishing, Retrying, and Needs attention.
- Publication is a server-side PDS write through an active owner OAuth session, never a client timer or anonymous/foreign write.
- The first implementation runs in-process but the queue, publisher, cleanup, and object-store interfaces do not depend on HTTP.
- There are no scheduled-post notifications, no history screen, no custom application-level encryption, and no separate worker service in this release.

## 2. Strategy

Implement the feature in vertical TDD slices, beginning with the required first failing test `UT-002` in `appview/internal/scheduledposts/validation_test.go`. Each slice should add one small behavior, make its named test fail for the intended reason, implement the minimum production code, and rerun the focused and surrounding suites before proceeding.

The backend will use:

1. A private Postgres resource and durable queue in `appview/internal/scheduledposts`.
2. Private S3-compatible object storage behind a narrow `ObjectStore` interface.
3. An in-process worker that claims work with Postgres leases and `FOR UPDATE SKIP LOCKED`.
4. Stable publication identity using a persisted atproto TID and first-attempt `createdAt`.
5. A frozen canonical record body assembled before the first PDS upload. The private-upload handler computes the raw blob CID from the prepared bytes, so the worker already knows the final atproto blob reference before it uploads those bytes to the PDS.
6. PDS reconciliation using the stable rkey: read the intended record, write only when absent, then read and compare again. A matching record completes the schedule; a different record at that rkey is a permanent conflict and is never overwritten.
7. A content-free cleanup queue so schedule rows and member-authored payload can be removed on time even when an object deletion must be retried.

The S3 adapter will use the official AWS SDK for Go v2 packages (`config`, `credentials`, and `service/s3`) behind the local interface. Current SDK documentation supports a custom `BaseEndpoint`, `UsePathStyle`, and static credentials, which covers MinIO without coupling the domain package to MinIO. No presigned URL will be returned to Flutter: every preview/download remains an authenticated AppView request.

Use raw parameterized pgx queries for this release, following the repository's existing store patterns. Keep transaction orchestration and public store methods in `internal/scheduledposts/store.go`; centralize SQL constants, explicit column lists, and row-scanning helpers in `internal/scheduledposts/store_queries.go` so the query surface remains reviewable. Clocks, hashing, capacity serialization, version fences, and external-effect locks remain handwritten. Do not introduce sqlc configuration, generated database models, or sqlc tooling as part of scheduled posts.

## 3. Affected Areas

| Area | Planned change | Requirements | Primary tests |
|---|---|---|---|
| Domain and validation | Time, eligibility, state machine, retry schedule, failure classification, retention, canonical payload/body | FR-002, FR-003, FR-011–FR-015, FR-020; RULE-002, RULE-003, RULE-005–RULE-008 | UT-001, UT-002, UT-004–UT-006, UT-009–UT-011, UT-013, UT-020 |
| Postgres queue | Capacity, idempotency, owner scope, versions, leases, frozen identity/body, tombstones, cleanup jobs | BR-002, FR-005–FR-007, FR-009–FR-015, NFR-001 | IT-003, IT-005–IT-007, IT-016, IT-017, IT-025, IT-026 |
| Private media | Idempotent upload, predicted blob CID, authenticated preview, S3 storage, replacement/delete cleanup | BR-004, FR-005, FR-017, FR-020, NFR-003, NFR-004 | UT-015, IT-014–IT-016, IT-018, IT-024, IT-025 |
| Publication worker | Session selection, current-policy validation, PDS upload/write/reconciliation, retries, recovery | BR-001, FR-011–FR-019, NFR-001, NFR-002 | UT-010, UT-011, UT-016, UT-018, IT-008–IT-013, IT-020, IT-024 |
| HTTP API | Owner-authenticated CRUD, media, manual publication from Edit, route policies and envelopes | FR-005–FR-010, FR-017; RULE-004 | IT-001, IT-002, IT-004, IT-014, IT-019 |
| Flutter data/state | Account-keyed repository, submission staging, capacity, refresh, edit hydration | FR-005, FR-006, FR-008, FR-009, FR-016 | UT-007, UT-008, UT-012, UT-014, UT-017, UT-019, IT-021, IT-022 |
| Flutter UI | `When`, picker, progress, Settings tile/page, row actions, full composer edit, accessibility | FR-001–FR-003, FR-006, FR-008–FR-010, NFR-005 | AT-001–AT-008, AT-014, MAN-001, MAN-002, MAN-004 |
| Lifecycle and privacy | Account deletion, bounded cleanup, no notifications, redacted signals and alerts | FR-014, FR-018, FR-020, FR-021, NFR-003, NFR-006 | IT-013, IT-016–IT-018, IT-023, IT-027, MAN-005, MAN-006 |

## 4. Files And Modules

### 4.1 AppView domain and persistence

Create `appview/internal/scheduledposts/` with these production modules:

- `types.go`: typed schedule/media/tombstone/claim models; the four public UI statuses; opaque safe error codes.
- `validation.go`: eligible payload and inclusive whole-minute five-minute/28-day validation using an injected server clock.
- `state.go`: allowed lifecycle transitions, capacity classification, mutation locks, and worker-version rules.
- `retry.go`: due/+1/+3/+7/+15/+30 attempt plan with injected bounded jitter. Use no jitter for due and +30; cap every calculated instant at `scheduledAt + 30m`.
- `payload.go`: canonical private payload encoding/decoding. Store canonical JSON bytes, not JSONB, so the accepted payload and ordered media/facet arrays round-trip without representation drift.
- `blob.go`: SHA-256 and atproto raw CID calculation for prepared private media.
- `publication.go`: record freezing, private-object reads/checksums, PDS blob upload verification, stable record lookup/write/reconciliation, and failure classification.
- `store.go`: transaction boundaries, public store methods, per-owner capacity serialization, per-schedule external-effect locking, and version-fence orchestration.
- `store_queries.go`: raw parameterized pgx queries, explicit result-column lists, and narrowly scoped row-scanning helpers.
- `worker.go`: bounded publication and cleanup batch processing, independent of routes and request context.
- `objectstore.go`: `Put`, `Get`, and idempotent `Delete` interface using opaque object keys only.
- `objectstore_s3.go`: AWS SDK v2 adapter configured with bucket, `BaseEndpoint`, region, credentials, and `UsePathStyle`.
- `retention.go`: 24-hour unclaimed-media, 30-day Needs attention, immediate-success cleanup, and 30-day tombstone deadlines.
- `observer.go`: content-free scheduled-domain observability interface.
- `account_deletion.go`: transaction-aware account cleanup plus terminal DID-deletion handling.

Add the exact test files already named in `02-acceptance-tests.md`, including `validation_test.go`, `state_test.go`, `retry_test.go`, `payload_test.go`, `publication_test.go`, `failure_test.go`, `retention_test.go`, `privacy_test.go`, `worker_test.go`, `store_test.go`, `concurrency_test.go`, `recovery_test.go`, `session_test.go`, `session_integration_test.go`, `cleanup_test.go`, `objectstore_minio_test.go`, `observability_test.go`, and the acceptance test files.

Add:

- `appview/migrations/000034_scheduled_posts.up.sql`
- `appview/migrations/000034_scheduled_posts.down.sql`
- `appview/internal/db/scheduled_posts_migration_test.go`

The migration test must be written before the migration and must use `internal/testdb.WithSchema`, inspect named constraints/indexes, exercise invalid lifecycle/reference combinations, and verify the down migration leaves unrelated sentinel data intact.

### 4.2 AppView API and reusable post assembly

Create:

- `appview/internal/api/scheduled_post.go`
- `appview/internal/api/scheduled_post_request.go`
- `appview/internal/api/scheduled_post_response.go`
- `appview/internal/api/scheduled_media.go`
- `appview/internal/api/scheduled_post_policy.go`
- corresponding tests named by IT-001, IT-002, IT-004, IT-014, and IT-019

`scheduled_post_policy.go` is the adapter between the durable package and current post rules. It must:

- Convert a canonical scheduled payload plus predicted media blob references into the existing `PostCreateRequest` shape.
- Reuse `ValidatePostCreateWithLimits`, project validation, language validation, mention extraction, and `PostStore.AuthorizeDirectedInteraction`.
- Explicitly reject `reply` and quote embed fields at the strict HTTP boundary.
- Build the same lexicon-shaped record as immediate posting, but with an explicit frozen `createdAt`.

Refactor `appview/internal/api/post.go` only enough to expose a pure `lexiconRecordBodyAt(req, createdAt)` seam. The existing immediate handler should continue to call it with `time.Now()` and continue to use `CreateRecord`; this protects FR-004 and REG-001 through REG-004/REG-009. Do not change the lexicon or the immediate blob endpoint.

Extend:

- `appview/internal/routes/routes.go` with scheduled-post/media handlers.
- `appview/internal/routes/policy.go` with matching authenticated/current-member route policies and read/write/upload body/rate classes.
- `appview/internal/app/deps.go` with the store, object store, policy, publisher, cleanup worker, and observer wiring.
- `appview/internal/app/config.go` and tests with validated storage/worker configuration.
- `appview/cmd/appview/main.go` with one cancellable in-process scheduled worker and graceful-shutdown wait.
- `appview/internal/app/instagram_lifecycle.go` (rename later only if useful) so Craftsky profile deletion invokes scheduled cleanup in the same actor-deletion transaction.
- the terminal identity-deletion handler list so a terminal DID deletion also removes retained private state.
- `appview/internal/observability/metric_recorder.go`, `observer.go`, and their Sentry/in-memory/no-op implementations with scheduled-domain signals.

### 4.3 Local object-storage and configuration

Extend:

- `docker-compose.yml` with pinned MinIO, a health check, a one-shot bucket bootstrap service, a persistent `miniodata` volume, and an optional host port for integration tests.
- `scripts/compose-dev` with a stable per-worktree MinIO port and a concise startup address.
- `just test` so it discovers the running MinIO endpoint and supplies `TEST_S3_ENDPOINT`, bucket, and test credentials to IT-015.
- `appview/environments/dev.env` with non-secret local MinIO settings.
- `appview/environments/prod.env.example` with documented secret-backed managed-provider settings.
- `appview/go.mod`/`go.sum` with the AWS SDK v2 S3/config/credentials modules.

Production config must reject a non-HTTPS object endpoint. Development/tests may explicitly opt into the internal Compose HTTP endpoint; that exception must be impossible in `EnvProd`. Bucket public-access blocking, provider encryption at rest, and least-privilege credentials remain MAN-005/GAP-001 deployment evidence, not custom application crypto.

### 4.4 Flutter scheduled-post feature

Create `app/lib/scheduled_posts/` with:

- `models/`: summary, detail, status, media, capacity, schedule-time choice, and submission models using `dart_mappable` and redacted `toString` output.
- `data/scheduled_post_api_client.dart`: fixed-account camelCase API client for schedule/media methods.
- `data/scheduled_post_repository.dart` and `api_scheduled_post_repository.dart`.
- `providers/scheduled_post_repository_provider.dart`: account-keyed repository and a generation boundary matching saved-post isolation.
- `providers/scheduled_posts_provider.dart`: max-three list/summary state, explicit `refresh`, confirmed upsert/remove, and no timer/polling.
- `providers/scheduled_post_submission_provider.dart`: per-composer idempotency key, private staging progress, create/update/manual-publication state, and retry-safe media IDs.
- `composer/schedule_composer_state.dart`: `Now`/future state, server-safe time validation hints, future-edit preservation, and missed-time display.
- `composer/scheduled_post_editor_loader.dart`: owner-fixed detail/media download and full composer seed.
- `pages/scheduled_posts_page.dart` and row/delete/status/thumbnail widgets.

Modify:

- `app/lib/feed/providers/composer_image_state.dart`: add an existing-private-media descriptor/state without fabricating a PDS CID; keep original/preview bytes redacted in diagnostics.
- `app/lib/feed/providers/composer_images_provider.dart`: expose a deterministic hydrate method for owner-authenticated scheduled media and a reusable reprepare operation. Preserve the current eager PDS pipeline for newly picked images.
- `app/lib/feed/widgets/post_composer_sheet.dart`: accept an optional scheduled editor seed, show `When` only for an original non-quote/non-reply composer, route Now/new to the current provider, and route Schedule/existing manual publication to scheduled submission.
- `app/lib/projects/widgets/project_composer_sheet.dart`: accept the same timing/edit seed on the final page and preserve the full three-page flow.
- `app/lib/projects/composer/project_composer_submit_adapter.dart`: preserve stored facets when the corresponding text is unchanged; regenerate only fields the member changed.
- add `app/lib/projects/composer/project_composer_hydrator.dart` to invert stored project data into all existing form fields for full editing.
- `app/lib/settings/pages/settings_page.dart`: add the localized Scheduled posts tile and Needs attention count.
- `app/lib/router/route_locations.dart`, `router.dart`, and generated `router.g.dart`: add `/profile/settings/scheduled` on the root navigator.
- `app/lib/auth/providers/account_boundary_provider.dart`: advance/invalidate all scheduled-post account state on account switch/sign-out.
- `app/lib/l10n/app_en.arb` and generated localization files with all timing, progress, status, capacity, lock, deletion, expiry, and accessibility copy.

Add the Flutter tests named in the acceptance specification under `app/test/scheduled_posts/`, plus project hydrator/adapter tests where necessary. Generated Riverpod, mapper, router, and localization files must be regenerated in their normal locations and committed with their sources during implementation.

## 5. Services And Data Flow

### 5.1 Database contract

Use four private tables:

#### `scheduled_posts`

The active unpublished resource and queue row:

- `id UUID` primary key and `owner_did TEXT` with a composite owner/id unique key.
- `operation_id UUID`, `request_hash BYTEA`, and unique `(owner_did, operation_id)` for idempotent create.
- `status TEXT` limited to `scheduled`, `publishing`, `retrying`, `needs_attention`.
- `scheduled_at`, `next_attempt_at`, `attempt_count`, `last_error_code`.
- canonical `payload_bytes BYTEA`, `payload_hash BYTEA`, and monotonic `payload_version`.
- `lease_token`, `lease_expires_at` only while Publishing.
- frozen `publication_rkey`, `publication_created_at`, `publication_record_bytes`, and `publication_record_hash` once the first attempt is prepared.
- `needs_attention_at`/`needs_attention_expires_at` only in Needs attention.
- `created_at`/`updated_at`.

Named checks must enforce the lifecycle shape. Named partial indexes must support due claims, expired lease recovery, owner/capacity listing, and Needs attention expiry. A unique partial `(owner_did, publication_rkey)` prevents a local identity collision.

#### `scheduled_post_media`

Owner-scoped staged-object metadata:

- client-stable `id UUID`, `owner_did`, opaque `object_key`, and `state` (`uploading` or `ready`).
- `schedule_id` and `ordinal` are both null while unclaimed and both populated when attached.
- `mime_type`, `size_bytes`, `sha256`, and predicted raw `blob_cid`.
- `created_at`, `updated_at`, `unclaimed_expires_at`.
- composite ownership/reference constraints ensure a media row can attach only to a schedule with the same owner.
- unique `(schedule_id, ordinal)` preserves ordered media without sharing objects between schedules.

An upload inserts/resumes the `uploading` metadata before the object write and marks it `ready` afterward. A crash after object creation is therefore recoverable by repeating the idempotent PUT to the same opaque key.

#### `scheduled_post_publication_tombstones`

The 30-day content-free success record:

- schedule ID, owner DID, operation ID/request hash, URI, CID, published timestamp, expiry timestamp.
- no payload, preview, alt text, facet, project, object key, or media identifier.
- unique `(owner_did, operation_id)` and suitable URI/idempotency constraints.

#### `scheduled_post_cleanup_jobs`

Content-free object deletion work:

- job ID, opaque object key, state/lease, attempt count, next attempt, safe last error class, created/updated timestamps.
- no owner DID or member-authored content.
- unique object key and pending/expired-lease indexes.

Replacement, delete, success, expiry, and account deletion insert cleanup jobs and delete the media/payload rows in the same database transaction. S3 deletion can then retry without extending user-visible retention or retaining member content in the active tables.

### 5.2 HTTP contract

Register these `/v1/` routes with the standard auth/device/current-member middleware, camelCase bodies, body limits, and error envelope:

| Method and path | Contract |
|---|---|
| `PUT /v1/scheduled-post-media/{mediaId}` | Idempotently upload one prepared image as the authenticated owner. The client UUID is the operation identity; identical retries return the same media resource and changed bytes conflict. |
| `GET /v1/scheduled-post-media/{mediaId}` | Stream owner-authenticated bytes with `private, no-store`, `X-Content-Type-Options: nosniff`, and no signed/public URL. |
| `DELETE /v1/scheduled-post-media/{mediaId}` | Idempotently enqueue cleanup for an unclaimed owner upload. Claimed media changes only through schedule update/delete. |
| `POST /v1/scheduled-posts` | Validate operation ID, UTC time, eligible payload, media ownership/readiness, and capacity; atomically claim media and return the detailed scheduled resource. |
| `GET /v1/scheduled-posts` | Return `{items, count, needsAttentionCount}` for at most three unpublished owner summaries, ordered by `scheduledAt`; no cursor. |
| `GET /v1/scheduled-posts/{id}` | Return the owner's complete editable payload and media metadata. |
| `PUT /v1/scheduled-posts/{id}` | Last-write-wins full replacement/reschedule. Reject Publishing, claim new media, enqueue removed media, increment `payloadVersion`, and clear old attempt/TID/body state. |
| `DELETE /v1/scheduled-posts/{id}` | Idempotent safe deletion. If Publishing committed first, return `409 scheduled_post_publishing`; otherwise remove/release/enqueue cleanup atomically. |
| `POST /v1/scheduled-posts/{id}/publication` | Composer-only “Post now” for an existing schedule. Apply the full edited payload and transition to a manual Publishing attempt atomically, then use the same stable publisher/recovery protocol. This is not a management-row quick action. |

New composers with `Now` do not call the scheduled endpoints. The publication subresource exists only to make “Post now” from an already-retained schedule safe; a `POST /v1/posts` followed by delete (or the reverse) would have an unavoidable duplicate-or-loss race.

List responses contain only bounded row fields: ID, status, scheduled/missed/expiry timestamps, post type, optional project title, bounded text preview, and first media ID. Full payload is available only from the owner-scoped detail endpoint.

### 5.3 New schedule submission

1. Composer state starts at `Now` with one stable operation UUID.
2. Existing image selection/preparation/eager PDS upload continues unchanged.
3. Only after the member taps Schedule, `scheduled_post_submission_provider` reparses/reprepares retained `previewBytes` through `ComposerImageMediaService`.
4. Each image uses its stable draft UUID in an idempotent private-media PUT; per-image progress is shown.
5. After every media PUT succeeds, Flutter sends the complete canonical content, ordered media IDs/alt/aspect ratios, operation ID, and UTC `scheduledAt`.
6. The AppView serializes create by owner, checks an existing active/tombstone operation first, enforces the count of active rows below three, claims only same-owner ready media, and commits one schedule.
7. Flutter closes only on success, refreshes scheduled state, shows confirmation, and performs no public feed/profile/project cache insertion.
8. Failure leaves text, fields, facets source, timing choice, operation ID, media IDs, and local bytes available for safe retry. Truly unclaimed private uploads expire at 24 hours.

### 5.4 Due publication and recovery

1. The worker polls every 10 seconds by default and claims a bounded batch whose `scheduledAt` and `nextAttemptAt` are not later than the injected/database time. Future rows are never selected.
2. Claim uses `FOR UPDATE SKIP LOCKED`, changes the row to Publishing, sets a lease token/expiry, captures `payloadVersion`, and—on the first attempt—allocates and persists a unique TID plus frozen UTC `createdAt`.
3. Before any PDS call, the worker converts staged-media metadata into final blob references, builds canonical record JSON through the same post assembler, and persists the frozen bytes/hash with the lease/version fence.
4. The worker acquires a per-schedule Postgres advisory effect lock on a dedicated connection, then rechecks row existence, Publishing status, lease, and payload version. Edit/delete/account deletion use the matching transaction lock. Whichever boundary wins is authoritative before the external side effect.
5. Select an active owner session through `auth.BackgroundSessionSelector`. No usable session is a transient `auth_unavailable`; no foreign/anonymous fallback exists.
6. Revalidate current post, media checksum, membership, mention, and block policy without changing the stored payload.
7. For every ordered image, read the private object, verify size/SHA-256/raw CID, upload it to the PDS, and verify the PDS response matches the frozen blob reference. Re-upload after a crash is safe because blobs are content-addressed. Craftsky never requests PDS blob deletion.
8. Call `GetRecord` at the frozen owner/collection/rkey. If the record exists and canonical value matches, use its CID and finalize. If it differs, enter Needs attention with `record_conflict` and never overwrite it.
9. If absent, call `PutRecord` with the frozen body and then `GetRecord` again. A matching read supplies the CID. An ambiguous write remains recoverable because the next attempt repeats the same read/reconcile sequence.
10. Finalization verifies the current fence, inserts the content-free tombstone, enqueues media cleanup, and deletes the active schedule/media in one transaction. Capacity and management therefore clear exactly once.
11. A stale worker may repeat content-addressed uploads or the same stable record write after lease loss, but it cannot finalize or publish different content. The recovered worker observes the same record identity/body.

Transient automatic failures set Retrying and the next approved offset. The sixth failed attempt at +30 minutes enters Needs attention. Permanent payload/media/policy failures enter Needs attention immediately. A manual Post now attempt does not silently start a new 30-minute automatic schedule: a definite failure returns the retained item to Needs attention; only an ambiguous external-write boundary remains Publishing for reconciliation.

### 5.5 Editing, deletion, and account lifecycle

- Scheduled items in the future reopen with Schedule and their absolute instant selected.
- Retrying or Needs attention items whose original instant is past show the missed instant and default to Now; choosing a new time must satisfy a fresh server-relative five-minute/28-day check.
- A successful full PUT increments `payloadVersion`, resets attempts/error/frozen publication fields, and returns the item to Scheduled. Existing private media may be reused; new media is staged first; removed/replaced media becomes cleanup work.
- Publishing rejects Edit/Delete before the UI claims success. Pull-to-refresh is the only status refresh after screen entry.
- Account deletion obtains the same schedule effect locks, inserts opaque cleanup jobs, deletes schedules/media/tombstones in the existing actor-deletion transaction, and then permits profile deletion. If publication already owns the effect lock, deletion waits for that boundary; if deletion wins, the worker's recheck prevents any PDS call.
- Terminal DID deletion invokes the same idempotent cleanup service for accounts that do not pass through a normal Craftsky profile deletion event.
- Signing out a device does not delete data. The background selector may use another active owner session. Needs attention is never selected by the automatic worker after reauthentication.

## 6. State, Providers, And Dependency Injection

### 6.1 AppView

Keep the core worker constructible from interfaces:

- `Store`: create/list/get/update/delete, claim/freeze/fence/retry/finalize, lifecycle expiry, cleanup claims, and account deletion.
- `Clock` and `Jitter`: deterministic time tests.
- `ObjectStore`: private bytes only.
- `SessionSelector`: owner DID to active OAuth session ID.
- `PDSClientFactory`: current existing auth abstraction.
- `PublicationPolicy`: current validation/authorization and record assembly.
- `Observer`: safe counters/gauges/distributions only.

`app.Deps` owns concrete construction. HTTP handlers receive only the schedule service/media service they use; `cmd/appview` receives only the batch processor. No handler or Flutter state is needed to run a batch.

Use two advisory-lock namespaces:

- Per-owner transaction locks serialize capacity-changing create/delete/finalize/expiry operations before counting active rows.
- Per-schedule effect locks serialize the final pre-PDS fence against Edit/Delete/account deletion. Hash collisions may serialize unrelated resources but cannot weaken correctness.

Every mutation/completion query includes owner where applicable plus status, lease token, and `payloadVersion`. No user-visible optimistic-concurrency token is introduced; member saves remain last-write-wins.

### 6.2 Flutter

- All private providers are keyed by `AccountKey` and built from `accountDioProvider(account)`.
- A `scheduledPostAccountBoundaryProvider` generation is captured by list, detail, staging, update, delete, and editor-load operations. Results from an old account generation are discarded.
- The active composer keeps one operation UUID across retry. Per-image private upload IDs are the existing stable draft UUIDs.
- Submission progress lives in `scheduled_post_submission_provider`; it must not overwrite the existing eager-upload phase or destroy local bytes on error.
- Hydrated scheduled media carries a private media ID and authenticated preview bytes, not a fabricated PDS blob. Immediate publication of an existing schedule goes through the publication subresource.
- The list provider fetches on first use/entry and exposes `refresh()` for `RefreshIndicator`; it has no timer, stream, or lifecycle polling.
- Settings derives the Needs attention badge from the same account-keyed summary so it cannot leak or retain another account's count.
- Both composer types bind unsaved-work/account-operation guards to the captured owner lease and close/abort safely on account change.

## 7. UI And Routes

### Composer

- Show a localized `When` row only for a new original top-level standard composer, a project composer, or an eligible scheduled editor.
- Default label/value is `Now`; the primary action remains `Post`.
- `Schedule for later` opens date then time selection, normalizes to a whole local minute, displays local timezone name/offset, converts once to UTC, and changes the primary action to `Schedule`.
- The client picker constrains obvious invalid choices, but the server is authoritative. Inline/server errors announce the valid range.
- At capacity, the Schedule for later choice remains visible but disabled, the composer warns that another post cannot be scheduled while three are retained, and a Manage scheduled posts link is shown. Now keeps Post enabled.
- During private staging, disable destructive submission controls, show per-image/aggregate progress, and announce progress/failure. A failed operation keeps the composer open.

### Settings and management

- Add `Scheduled posts` at `/profile/settings/scheduled` using the root navigator pattern already used by Settings children.
- The Settings tile displays the Needs attention count when non-zero.
- The page uses `RefreshIndicator`, orders by `scheduledAt`, and shows loading/empty/safe error states.
- Each row shows authenticated first-image thumbnail when present, Standard/Project, optional project title, bounded text preview, localized absolute time plus timezone/offset, and exactly Scheduled, Publishing, Retrying, or Needs attention.
- Tapping an editable row loads the owner detail/media and opens the matching full composer.
- The row overflow and editor expose Delete with confirmation. Publishing replaces Edit/Delete with clear locked copy.
- Needs attention shows its missed publication time and deletion date. Published items never appear here.

All new strings must be in ARB localization, every icon-only action needs a tooltip/semantic label, status/progress/errors need live-region announcements, and widget tests must cover large text without clipped required actions.

## 8. Error States And Safe Contracts

Use the standard `{error, message, requestId, fields?}` envelope and keep messages content-free. Planned stable codes:

| Code | HTTP | Behavior |
|---|---:|---|
| `invalid_scheduled_at` | 422 | Field error for non-minute/out-of-range UTC time. |
| `scheduled_post_ineligible` | 422 | Quote/reply/comment/invalid project cannot be retained. |
| `scheduled_post_capacity` | 409 | Three countable items already exist; no media ownership change. |
| `scheduled_post_not_found` | 404 | Same response for missing or foreign schedule. |
| `scheduled_post_publishing` | 409 | Owner resource crossed the Publishing boundary; Edit/Delete did not succeed. |
| `scheduled_media_not_found` | 404 | Same response for missing, unready, or foreign media. |
| `scheduled_media_conflict` | 409 | Same media operation ID was reused with different bytes/metadata. |
| `scheduled_media_invalid` | 422 | Type/size/content/checksum validation failure. |
| `scheduled_operation_conflict` | 409 | Create operation ID was reused with a different canonical request. |
| `scheduled_publication_failed` | 502 | Manual Post now failed definitely; retained item is Needs attention. |

Worker errors remain internal safe classes such as `auth_unavailable`, `object_unavailable`, `pds_unavailable`, `policy_invalid`, `media_invalid`, and `record_conflict`. Never log/emit DIDs, handles, schedule/media IDs, object keys, filenames, alt text, payload fields, exact scheduled timestamps, signed URLs, tokens, provider bodies, or raw errors that may contain them.

Object upload/write partial failures are handled explicitly:

- DB metadata created but object PUT failed: retain/retry the `uploading` row until 24-hour cleanup.
- Object PUT succeeded but ready-mark failed: identical media PUT repeats safely to the same opaque key.
- Schedule create failed after media upload: media remains unclaimed and expires unless the same operation retry claims it.
- PDS media upload partially succeeded: retry/reuse content-addressed blobs; never delete them.
- PDS record write succeeded but local completion failed: stable Get/compare finalizes later.
- Cleanup delete failed: content-free cleanup job retries and contributes to backlog alerts.

## 9. Test Plan

### Slice 1: Pure rules and first red test

1. Write and fail `UT-002` first for inclusive 5-minute/28-day whole-minute server boundaries.
2. Implement only injected-time validation and make UT-002 green.
3. Add UT-001, UT-004–UT-006, UT-009–UT-011, UT-013, UT-015, UT-016, UT-018, and UT-020 for domain rules, payload, publication identity, privacy, and worker construction.
4. Add the Flutter pure-state UT-003, UT-007, UT-008, UT-012, UT-014, UT-017, and UT-019 without wiring routes yet.

Requirements covered: FR-001–FR-003, FR-005, FR-006, FR-008, FR-009, FR-011–FR-020; RULE-001–RULE-008; NFR-003, NFR-004.  
Gate: focused Go/Dart unit suites and no wall-clock waits.

### Slice 2: Migration and durable raw-pgx store

1. Write failing IT-026 catalog/invariant/down-migration test.
2. Add the migration, then implement centralized raw parameterized pgx queries and explicit row scanners in the scheduled-post store.
3. Implement IT-003 final-slot capacity with real concurrent Postgres connections.
4. Implement IT-005 through IT-007 for LWW member updates, version fences, Publishing serialization, due claims, lease expiry, and stale completion.
5. Implement IT-016, IT-017, and IT-025 for lifecycle transactions, tombstones, cleanup jobs, and unclaimed staging.

Requirements covered: BR-002; FR-005–FR-007, FR-009–FR-015, FR-020; NFR-001; RULE-001, RULE-004, RULE-005.  
Gate: `internal/testdb.WithSchema` tests green under race-enabled `just test`; raw queries exercise the real migration and all scan paths through Postgres integration tests.

### Slice 3: Private object storage and media API

1. Add the `ObjectStore` fake and AWS SDK adapter.
2. Add Compose MinIO/bucket bootstrap and IT-015.
3. Implement idempotent staged-media PUT/GET/DELETE and IT-014.
4. Add checksum/raw-CID tests, IT-024 private-copy marker, privacy canaries, and cleanup retry behavior.

Requirements covered: BR-004; FR-005, FR-014, FR-017, FR-020; NFR-003, NFR-004; RULE-004.  
Gate: fast fake tests plus explicit MinIO contract test; anonymous bucket access fails.

### Slice 4: Scheduled CRUD API

1. Implement strict request/response types and operation hashing.
2. Add create/list/get/update/delete handlers and route policies.
3. Make IT-001, IT-002, IT-004, and IT-019 green, including exact owner isolation and safe errors.
4. Add the manual publication subresource contract but keep its publisher fake until Slice 5.

Requirements covered: FR-002, FR-003, FR-005–FR-010, RULE-002–RULE-005.  
Gate: all scheduled API methods pass full-server route tests; existing `/v1/posts` and `/v1/blobs/images` tests remain unchanged/green.

### Slice 5: Publisher, recovery, sessions, and cleanup worker

1. Freeze predicted blob refs, TID, `createdAt`, and canonical body before external calls.
2. Implement session selection, PDS blob verification, Get/Put/Get reconciliation, and atomic finalization.
3. Make IT-008 through IT-013 green with fake clocks and deterministic barriers at every TD-007 boundary.
4. Wire in-process lifecycle and make IT-020 green.
5. Add account deletion/session integration, IT-023 no-notification boundary, IT-018 privacy scan, and IT-027 operational signals.
6. Run AT-009 through AT-013 backend acceptance suites.

Requirements covered: BR-001, BR-004; FR-011–FR-021; NFR-001–NFR-004, NFR-006; RULE-005–RULE-008.  
Gate: no early side effect, stable body/identity under crash, no notification writes, content-free telemetry.

### Slice 6: Flutter submission and management

1. Add account-keyed API/repository/list/submission providers and IT-021/IT-022.
2. Add composer timing state and `When` UI to eligible standard/project composers only.
3. Reprepare/stage retained bytes only after Schedule; preserve eager upload and failure recovery.
4. Add Settings tile, typed route, management page, authenticated thumbnails, pull-to-refresh, Delete, and Publishing locks.
5. Add full editor hydration for standard/project payloads, unchanged facet preservation, private media reuse, Needs attention/missed-time behavior, reschedule, and manual Post now.
6. Make AT-001 through AT-008 and AT-014 green, then REG-001 through REG-010.

Requirements covered: BR-001–BR-003; FR-001–FR-010, FR-016, FR-017, FR-021; NFR-005.  
Gate: scheduled success updates only scheduled state; immediate success retains its existing public-cache behavior; account switch leaks no rows/counts/media/results.

### Slice 7: Aggregate and release verification

Run, in order:

```text
cd appview && go test ./internal/scheduledposts ./internal/api ./internal/db ./internal/app ./cmd/appview
just app-test test/scheduled_posts
just app-analyze
just test
git diff --check
```

Also run the existing complete Flutter suite through `just app-test` before implementation review. Report each command actually run; do not represent MAN-001 through MAN-006 or GAP-001 through GAP-004 as automated passes.

Manual/release gates remain:

- MAN-001 real-device timezone/DST/accessibility/large-text behavior.
- MAN-002 real-device media reprocessing and interrupted staging recovery.
- MAN-003 closed-app/restarted-AppView publication through real dev PDS/Tap.
- MAN-004 multi-device LWW/manual-refresh/session behavior.
- MAN-005 selected production provider TLS/private access/encryption/IAM evidence.
- MAN-006 deployed metric attributes and alert delivery/recovery.
- GAP-001 through GAP-004 exactly as documented in the acceptance specification.

## 10. Sequencing And Guardrails

1. Preserve UT-002 as the first failing implementation test even though later slices may group related tests.
2. Recheck migration numbering and dirty-worktree scope before changing anything. The current workflow documents are untracked user work and must be preserved.
3. Do not edit `lexicon/`; `scheduledAt` remains private and no atproto lexicon ADR is needed.
4. Do not change the default/current immediate post or eager PDS blob behavior except for the pure explicit-time record-builder seam.
5. Do not use device timers, lifecycle callbacks, or Flutter background execution as publication authority.
6. Do not expose object keys or presigned URLs. All media bytes traverse owner-authenticated AppView handlers.
7. Do not add custom encryption, notification producers/categories/preferences, scheduled history, draft autosave, recurrence, or a separate worker binary/service.
8. Use injected clocks, jitter, and barriers. No test may sleep through product deadlines.
9. Use stable typed atproto identifiers at boundaries. Generate TIDs through indigo syntax, persist before PDS calls, and protect `(owner_did, publication_rkey)` in Postgres.
10. Treat `PutRecord` as recoverable, not transactionally exactly-once. Always reconcile the stable record before and after it.
11. Never request PDS blob deletion for partial/superseded scheduled attempts.
12. Do not introduce sqlc in this release. Keep raw SQL centralized, parameterized, and covered by real-Postgres tests. Run the remaining code generators deliberately and inspect their diff; do not mix unrelated formatter/generator churn.
13. Keep all API JSON camelCase and every error in the existing standard envelope.
14. Keep logs/metrics/traces free of account/resource/content identifiers. Test with TD-008/TD-011 canaries before full-suite claims.
15. After implementation, use `review-implementation` before any UI polish; use `polish-ui` only after the TDD implementation and review are approved.

## 11. Risks And Open Questions

There are no blocking product questions. The following implementation/deployment risks have explicit handling:

| Risk | Planned control |
|---|---|
| Duplicate public post after crash | Persist TID/createdAt/body, predicted blob CIDs, Get/Put/Get comparison, stable URI, tombstone dedupe. |
| Edit/delete/account deletion crosses external write | Publishing transition plus version fence and per-schedule advisory effect lock; commit ordering decides the boundary. |
| Capacity exceeds three under concurrent devices | Per-owner transaction advisory lock plus count/insert in one transaction; all slot releases use the same lock. |
| Private upload object exists without DB readiness | Insert resumable `uploading` metadata before PUT; retry same key; 24-hour lifecycle cleanup. |
| Stored media is corrupt or provider returns different metadata | Verify size, SHA-256/raw CID on read and verify PDS upload response before record write. |
| Worker is down while HTTP is healthy | 10-second polling when healthy, durable leases/recovery, queue/overdue/claim metrics, deployed alerts. |
| AppView scales horizontally | SKIP LOCKED claims, leases, advisory locks, stable PDS identity; package can move to a separate process without API/schema change. |
| Raw-query drift or scan mismatch | Centralize SQL in `store_queries.go`, use explicit column lists and typed scanners, and cover every query/scan path with real-Postgres integration tests plus the migration catalog assertions. |
| Managed S3-compatible differences | Narrow standard S3 adapter, MinIO contract test, production HTTPS validation, MAN-005 provider evidence. |
| Local MinIO is not TLS | Explicit dev/test-only insecure endpoint allowance; production validation rejects it. This does not weaken the production NFR or add custom crypto. |
| Alert thresholds differ by deployment | Document initial content-free thresholds (oldest overdue >60 seconds sustained 5 minutes; repeated claim/batch errors; any exhausted/auth-unavailable transition; cleanup oldest >15 minutes or repeated deletion failure), then verify deployed rules in MAN-006. |

The managed production provider remains deliberately unselected in code. Its endpoint, bucket, region, credentials, access policy, default encryption, and lifecycle evidence are deployment configuration covered by GAP-001/MAN-005.

## 12. Handoff

The requirements, acceptance tests, and document review are approved, and this coding plan is ready for user approval. Because the feature is high risk, implementation must not start until the user explicitly chooses the `implement-tdd` next step.

On approval, invoke `implement-tdd` with this directory and begin with `UT-002` only. After the final automated implementation gate, hand the change to `review-implementation`; do not silently absorb remaining manual/deployment gates into a “complete” claim.
