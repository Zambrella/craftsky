# Coding Plan: Local Post Drafts And Submit-Time Media Uploads

## 1. Inputs

- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` — Approved, medium risk, no blocking findings
- Repository context: Flutter/Riverpod client under `app/`; existing AppView routes and scheduled-post worker remain unchanged
- Package guidance: persistent user-authored files belong under `getApplicationDocumentsDirectory()` rather than cache/temporary storage; `wakelock_plus` enable/disable calls are scoped to the visible submission lifecycle

## 2. Implementation Strategy

Implement the feature entirely in the Flutter client, behind three boundaries:

1. A device-local `LocalPostDraftRepository` stores one versioned manifest and immutable prepared-media files per account-owned draft under persistent private application documents storage. The repository exposes list/get/save/delete/read-media operations and hides paths, serialization, atomic replacement, reconciliation, and error redaction from UI code. It is intentionally file-backed; SQLite and SharedPreferences are not used for draft content.
2. `ComposerImages` becomes a local preparation state machine only. It retains the metadata-stripped, resized/re-encoded publishable bytes and dimensions needed for preview, draft persistence, immediate upload, or scheduled staging. It no longer depends on `PostApiClient` and has no eager upload phase.
3. A composer-scoped `ComposerSubmissionController` materializes prepared media only after a validated Post, Reply, Publish now, or Schedule action. It owns exact pre-network snapshotting for an existing local draft, per-image one-minute budgets, transient immediate blob-reference reuse, scheduled idempotent staging, account fencing, the full-screen overlay, and screen-awake cleanup.

Standard and project composers keep their existing validation and payload adapters. Thin draft snapshot adapters convert each composer's editable state to and from the local manifest without requiring publication completeness. Scheduled-post hydration and APIs remain separate from local drafts, but scheduled submission uses the same local-ready image representation and common lifecycle controller.

This design deliberately implements device-local drafts despite the repository's broad AppView/Postgres draft guidance. It adds no AppView storage, API route, database migration, worker change, PDS draft record, or lexicon change.

## 3. Affected Areas

| Area | Existing Pattern | Planned Change | Requirement IDs | Test IDs |
|---|---|---|---|---|
| Draft domain and persistence | No general local draft store | Add account-scoped v1 manifests, immutable media, atomic update planning, reconciliation, and redacted errors | FR-002, FR-003, FR-006, FR-007, FR-015, FR-016, FR-020, NFR-001, NFR-002, RULE-001, RULE-006, RULE-007 | AT-002, AT-004, AT-007, AT-008, AT-015–AT-020, AT-023; UT-002–UT-007, UT-014, UT-015, UT-017; IT-001–IT-005, IT-016, IT-018 |
| Image selection and preparation | `ComposerImages` prepares and immediately uploads | Retain prepared bytes locally; remove upload client dependency and transfer phases; add unavailable/replace state | FR-003, FR-010, FR-016, FR-021 | AT-002, AT-009, AT-015, AT-025; UT-003, UT-008; IT-004, IT-010 |
| Draft composer adaptation | Scheduled hydrators only; no local snapshot contract | Add standard and project snapshot adapters, eligibility policy, schedule restoration, and durable-media hydration | FR-001, FR-002, FR-005, FR-018, FR-019, RULE-002–RULE-004 | AT-001, AT-003, AT-004, AT-006, AT-021, AT-024, AT-025; UT-001, UT-005; IT-008, IT-009 |
| Submission orchestration | Submission and scheduled staging duplicated in composer widgets | Add target-aware controller for snapshot, upload/stage, API call, retry ledger, cleanup, and lease checks | FR-008, FR-009, FR-011, FR-012, FR-014, FR-022, FR-023, RULE-003–RULE-005 | AT-010–AT-012, AT-014, AT-018, AT-022, AT-026, AT-027; UT-009–UT-013, UT-016; IT-011, IT-012, IT-014, IT-015, IT-017 |
| Submission feedback | Inline flags and progress; no common wakelock | Add blocking full-screen overlay with exact copy, semantics, duplicate/back/tap protection, and scoped screen-awake adapter | FR-009, FR-013, NFR-003, NFR-005 | AT-013, AT-014, AT-026; UT-012; IT-013 |
| Draft management | Scheduled-post Settings page is nearest pattern | Add account-local Drafts route/page, rows, thumbnails, edit, confirmed delete, empty/error/retry states | FR-004, FR-007, FR-014, FR-017, RULE-007 | AT-005, AT-008, AT-018; UT-004, UT-014, UT-016; IT-006, IT-007, IT-015 |
| Existing API clients | Immediate and scheduled wire contracts already exist | Preserve routes and payloads; allow cancellation/timeout injection at transfer calls only | FR-011, FR-012 | AT-010, AT-011; IT-011, IT-012, IT-017; REG-006 |
| Dependencies and generated code | `path`/`path_provider` are transitive; no wakelock runtime package | Add direct runtime `path`, `path_provider`, `crypto`, and `wakelock_plus`; regenerate providers/routes/localizations/mappers affected by state changes | FR-003, FR-004, FR-013, NFR-001, NFR-005 | REG-009, REG-010 |

## 4. Files And Modules

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `app/pubspec.yaml`, `app/pubspec.lock` | Change | Add direct runtime dependencies for persistent directories, safe path joining, prepared-byte digests, and screen-awake control; do not add runtime SQLite | FR-003, NFR-001, NFR-005 | REG-009, REG-010 |
| `app/lib/drafts/models/local_post_draft.dart` | Create | Define draft kind, summary/detail, timestamps, owner, schedule intent, standard/project snapshot envelopes, and safe unavailable state | FR-001, FR-002, FR-004, FR-015, FR-019 | AT-001, AT-004–AT-006, AT-015, AT-024; UT-001, UT-002, UT-004, UT-005, UT-014 |
| `app/lib/drafts/models/draft_media_descriptor.dart` | Create | Define ordered media metadata, SHA-256 digest, opaque stored-media identity, display name, MIME, dimensions, alt text, and unavailable status without exposing paths | FR-003, FR-016 | AT-002, AT-004, AT-015, AT-025; UT-003 |
| `app/lib/drafts/models/draft_manifest_codec.dart` | Create | Hand-write strict schema-version-1 JSON encode/decode, bounded fields, unknown-field tolerance, incompatible-version handling, and content-free errors | FR-002, FR-015, NFR-002 | AT-004, AT-015, AT-019; UT-002, UT-015; IT-004, IT-016 |
| `app/lib/drafts/data/local_post_draft_repository.dart` | Create | Declare repository and write/read DTO contracts independent of file storage | FR-003–FR-007, FR-015–FR-017 | AT-002, AT-005–AT-008, AT-015–AT-017 |
| `app/lib/drafts/data/draft_file_store.dart` | Create | Wrap asynchronous directory/file/list/flush/atomic-replace/delete operations for production and deterministic failure injection | FR-006, FR-015, FR-016, NFR-001, NFR-004 | AT-016, AT-017, AT-020; UT-006, UT-007, UT-017; IT-002–IT-004, IT-018 |
| `app/lib/drafts/data/draft_update_plan.dart` | Create | Compute immutable-file reuse/write/manifest-switch/cleanup actions before mutating storage | FR-006, NFR-001 | AT-007, AT-016; UT-006; IT-002 |
| `app/lib/drafts/data/file_local_post_draft_repository.dart` | Create | Implement account namespace, list/get/save/delete/read-media, atomic updates, startup reconciliation, idempotent deletion, and safe damaged rows | FR-003, FR-006, FR-007, FR-015, FR-016, FR-020, NFR-001, NFR-002, RULE-001, RULE-006, RULE-007 | AT-002, AT-007, AT-008, AT-015–AT-020, AT-023; IT-001–IT-005, IT-016, IT-018 |
| `app/lib/drafts/composer/draft_save_eligibility.dart` | Create | Compare current editable state with initial state and restrict saving to top-level standard/project composers | FR-001, FR-021, RULE-002–RULE-004 | AT-001, AT-021, AT-025; UT-001, UT-008 |
| `app/lib/drafts/composer/draft_schedule_restoration.dart` | Create | Restore valid Later intent or reset to Now with an explanation using current schedule rules and injected clock | FR-019 | AT-024; UT-005 |
| `app/lib/drafts/composer/standard_draft_snapshot_adapter.dart` | Create | Convert standard composer text/languages/schedule/media to/from an incomplete local snapshot | FR-002, FR-005 | AT-004, AT-006; IT-008 |
| `app/lib/projects/composer/project_draft_snapshot_adapter.dart` | Create | Explicitly encode/decode every known project form field, including partial values and `ProjectMaterial`, without requiring a valid `Project` | FR-002, FR-005, RULE-004 | AT-004, AT-006; IT-009 |
| `app/lib/drafts/composer/draft_composer_hydrator.dart` | Create | Resolve stored media asynchronously, seed locally-ready/unavailable attachments, preserve ID/timestamps, and apply schedule restoration | FR-002, FR-003, FR-005, FR-016, FR-019 | AT-002, AT-004, AT-006, AT-015, AT-024; IT-001, IT-004, IT-008, IT-009 |
| `app/lib/drafts/providers/local_post_draft_repository_provider.dart` plus generated `.g.dart` | Create | Resolve Documents root, file store, and account-keyed repository | FR-014, FR-017, FR-020 | AT-005, AT-018, AT-023; IT-005–IT-007, IT-015 |
| `app/lib/drafts/providers/local_post_drafts_provider.dart` plus generated `.g.dart` | Create | Account-keyed async list/refresh/delete state with stale-lease fencing | FR-004, FR-007, FR-014, FR-017 | AT-005, AT-008, AT-018; IT-006, IT-015 |
| `app/lib/drafts/providers/draft_save_controller.dart` plus generated `.g.dart` | Create | Own explicit save/save-changes progress, repository calls, list refresh, safe feedback, and stale-owner suppression | FR-014, FR-016, FR-018, FR-021, NFR-004 | AT-017, AT-018, AT-020, AT-025; UT-016, UT-017; IT-003, IT-008, IT-009, IT-018 |
| `app/lib/drafts/pages/drafts_page.dart` | Create | Render active-account draft management and open matching composer | FR-004, FR-005, FR-007, FR-017 | AT-005, AT-006, AT-008; IT-006, IT-007 |
| `app/lib/drafts/widgets/draft_row.dart`, `draft_thumbnail.dart`, `draft_close_dialog.dart` | Create | Provide bounded row presentation, lazy local thumbnails, delete confirmation, and exact three-choice close behavior | FR-004, FR-007, FR-016, FR-018 | AT-003, AT-005, AT-008, AT-015; UT-004, UT-014; IT-006, IT-008, IT-009 |
| `app/lib/feed/providers/composer_image_state.dart` plus mapper | Change | Replace eager-upload phases with local preparing/ready/failed/unavailable states holding prepared bytes and opaque persisted origin | FR-003, FR-010, FR-016, FR-021 | AT-009, AT-015, AT-025; UT-003, UT-008; REG-002 |
| `app/lib/feed/providers/composer_images_provider.dart` plus generated `.g.dart` | Change | Remove `PostApiClient` dependency; locally prepare, retry, reorder, edit, seed, replace, and remove media | FR-003, FR-010, FR-016, FR-021 | AT-002, AT-009, AT-015, AT-025; UT-008; IT-010 |
| `app/lib/feed/widgets/composer_image_attachment_section.dart` | Change | Show local preparation only, `Image unavailable`, retry/remove/replace, and alt/reorder behavior without upload progress | FR-010, FR-016, FR-021 | AT-009, AT-015, AT-025; IT-004, IT-008, IT-009, IT-010 |
| `app/lib/feed/composer/submission_materialization_plan.dart` | Create | Select immediate blob or scheduled private-staging path without performing work until invoked | FR-011, FR-012 | AT-010, AT-011; UT-009 |
| `app/lib/feed/composer/immediate_upload_retry_state.dart` | Create | Hold non-serializable blob references by composer, media ID, and prepared-byte digest; invalidate changed/removed bytes and dispose on close | FR-011, FR-023 | AT-026; UT-010; IT-011 |
| `app/lib/feed/composer/composer_media_uploader.dart` | Create | Sequentially revalidate and upload locally-ready bytes with independent one-minute cancellation budgets and ordered results | FR-009, FR-011, FR-012, FR-023 | AT-010, AT-011, AT-026; UT-009–UT-011; IT-011, IT-012, IT-017 |
| `app/lib/feed/composer/submission_screen_awake.dart` | Create | Abstract `enable`/`disable`; provide `wakelock_plus` adapter and test fake | NFR-005 | AT-014; UT-012; IT-013 |
| `app/lib/feed/composer/composer_submission_controller.dart` plus generated `.g.dart` | Create | Coordinate overlay lifecycle, exact draft snapshot, target materialization, final API, success-only draft deletion, account fence, retry state, and cleanup | FR-008, FR-009, FR-011–FR-014, FR-022, FR-023 | AT-010–AT-014, AT-018, AT-026, AT-027; UT-012, UT-013, UT-016; IT-011–IT-015, IT-017 |
| `app/lib/feed/widgets/submission_blocking_overlay.dart` | Create | Full-screen `PopScope`/modal barrier/semantics/spinner surface with exact operation text and no cancel path | FR-013, NFR-003, NFR-005 | AT-013, AT-014; UT-012; IT-013 |
| `app/lib/feed/providers/create_post_provider.dart` | Change | Expose an awaitable authoritative create result/error while retaining current optimistic-cache and success refresh ownership | FR-008, FR-011, FR-014 | AT-010, AT-012, AT-018; IT-011, IT-014, IT-015; REG-001 |
| `app/lib/feed/widgets/post_composer_sheet.dart` | Change | Accept optional local draft seed/origin, add explicit save/close flows for eligible posts, validate before controller start, and remove inline upload/staging lifecycle | FR-001, FR-005, FR-011–FR-014, FR-018, FR-022 | AT-001, AT-003, AT-006, AT-010, AT-013, AT-021, AT-025, AT-027; IT-008, IT-010, IT-013, IT-014 |
| `app/lib/projects/widgets/project_composer_sheet.dart` | Change | Apply the same local-draft and shared-submission integration while preserving multi-page project state and validation | FR-001, FR-005, FR-011–FR-014, FR-018, FR-022 | AT-001, AT-003, AT-006, AT-010, AT-013, AT-025, AT-027; IT-009, IT-010, IT-013, IT-014 |
| `app/lib/projects/composer/project_composer_submit_adapter.dart` | Change | Consume locally-ready ordered media supplied by the controller instead of eager-upload state | FR-010, FR-011 | AT-009, AT-010; IT-011; REG-001 |
| `app/lib/scheduled_posts/services/scheduled_composer_media.dart` | Change | Reuse locally prepared bytes directly, retain existing staged IDs for scheduled edits, and delegate transfer timing/cancellation to the controller | FR-010, FR-012, FR-023 | AT-009, AT-011, AT-026; IT-010, IT-012; REG-003 |
| `app/lib/scheduled_posts/data/scheduled_post_api_client.dart`, `scheduled_post_repository.dart`, `api_scheduled_post_repository.dart` | Change | Thread optional transfer cancellation through existing staging method without changing route, headers, body, or error envelope | FR-009, FR-012 | AT-011, AT-026; IT-012, IT-017; REG-006 |
| `app/lib/settings/pages/settings_page.dart` | Change | Add localized Drafts entry beside Scheduled posts with no badge | FR-004, FR-017 | AT-005; IT-007; REG-004 |
| `app/lib/router/route_locations.dart`, `router.dart`, generated route files | Change | Add typed `/profile/settings/drafts` root-navigator route | FR-004, FR-017 | AT-005; IT-007; REG-004 |
| `app/lib/l10n/app_en.arb`, generated localizations | Change | Add Drafts management, save/close, unavailable-media, invalid-time, errors, and exact submission status copy | FR-004, FR-013, FR-016, FR-018, FR-019 | AT-003, AT-005, AT-013, AT-015, AT-024, AT-025; REG-010 |
| `app/test/drafts/**`, `app/test/feed/composer/**`, `app/test/feed/widgets/submission_overlay_test.dart`, `app/test/scheduled_posts/scheduled_post_api_client_test.dart` | Create | Add the acceptance/unit/integration targets specified in `02-acceptance-tests.md` | All in-scope requirements | AT-001–AT-027, UT-001–UT-017, IT-001–IT-018 |
| Existing composer, scheduled-post, Settings, route, provider, media, and API-client tests | Change | Update fakes and expectations for local-ready state and submit-time transfer while retaining existing behavior | FR-004, FR-010–FR-014, FR-017, FR-023 | REG-001–REG-008, REG-010 |

Generated file names may follow the generator's actual output rather than being created by hand. The implementation must keep the manifest codec hand-written and versioned; code generation for in-memory Riverpod/state types must not become the persistence schema.

## 5. Services, Interfaces, And Data Flow

### 5.1 Domain and repository contracts

```text
enum LocalPostDraftKind { standard, project }

sealed class LocalDraftContent
  StandardDraftContent(text, languages)
  ProjectDraftContent(body, languages, knownProjectFieldValues)

class DraftScheduleIntent(choice, scheduledAtUtc?, savedOffsetMinutes?)

class DraftMediaDescriptor(
  mediaId, storageRevision, displayFileName, mimeType,
  byteLength, sha256, width, height, altText, order
)

class LocalPostDraft(
  id, owner, kind, createdAt, updatedAt,
  content, schedule, media, availability
)

class DraftMediaWrite
  ExistingStoredMedia(mediaId, storageRevision, expectedSha256)
  PreparedMedia(mediaId, displayFileName, mimeType, bytes, width, height, altText)
  UnavailableMedia(mediaId, preservedMetadata, altText)

class DraftWriteRequest(
  id, owner, kind, createdAt?, expectedRevision?, content, schedule, orderedMedia
)

abstract interface class LocalPostDraftRepository {
  Future<List<LocalPostDraftSummary>> list();
  Future<LocalPostDraft> get(String draftId);
  Future<LocalPostDraft> save(DraftWriteRequest request);
  Future<Uint8List> readMedia(String draftId, String mediaId);
  Future<void> delete(String draftId);
}
```

The repository is constructed for one captured `AccountKey`; owner is still encoded and verified in the manifest as defense in depth. IDs are validated UUIDs. The account directory component is a deterministic base64url encoding of the owner DID with padding removed; neither that value nor the DID is logged. Media references in JSON are repository-generated leaf filenames only. Absolute paths, separators, traversal, links escaping the draft directory, duplicate IDs, excessive image counts, unsupported MIME types, invalid dimensions, and digest/length mismatches are rejected as content-free unavailable errors.

Project draft serialization is an explicit whitelist keyed by `ProjectComposerFields`. It encodes each known scalar/date/enum/list value and converts `ProjectMaterial` through a dedicated map adapter. It does not serialize arbitrary FormBuilder runtime values and does not call `ProjectComposerSubmitAdapter`, because incomplete projects are saveable.

### 5.2 Persistent layout and atomic update

```text
<application-documents>/CraftSky/drafts/v1/<safe-account-key>/<draft-id>/
  manifest.json
  media/<media-id>-<storage-revision>.<extension>
  .pending-<operation-id>-manifest.json
```

`getApplicationDocumentsDirectory()` is resolved once through an injected provider. Drafts never use temporary/cache directories. The app sandbox supplies privacy; normal OS backup behavior is accepted. No custom encryption, SQLite, SharedPreferences payload, or remote index is added.

Save/update ordering:

1. Validate and normalize the complete request off the widget build path. Compute SHA-256 for new prepared bytes asynchronously.
2. Under a per-draft repository mutex, reload the current manifest and check `expectedRevision` so two callers cannot silently overwrite a newer save.
3. Build a `DraftUpdatePlan`. Reuse a media file only when the request carries its opaque stored origin and ID/digest/length still match the current manifest. Reorder and alt-text-only edits reuse bytes. Replaced or changed bytes receive a new immutable storage revision/filename.
4. Write each new media file inside the target draft directory, request a flushed write, and verify its length/digest. A new draft remains unlisted until its manifest exists.
5. Encode the complete v1 manifest to a same-directory pending file, flush it, and atomically replace `manifest.json`. This is the sole visibility switch.
6. Only after the manifest switch succeeds, best-effort delete media no longer referenced and pending/orphan files. Cleanup failure is recorded only as a coarse content-free diagnostic and retried by reconciliation; the successful save remains successful.
7. On failure before the manifest switch, delete only newly created unreferenced files where safe, preserve the last manifest and files, return a safe retryable error, and keep the composer open.

Repository startup/list performs bounded asynchronous reconciliation per account: remove stale pending manifests, remove media not referenced by any valid manifest, never delete a referenced filename, keep healthy drafts visible, and surface unsupported/corrupt manifests as content-free unavailable/deletable summaries. A valid manifest with missing/corrupt media retains its authored content and descriptor but marks that item unavailable. `delete` renames or otherwise isolates the precise validated draft bundle before recursive cleanup where supported, is idempotent when absent, and never accepts a caller-supplied filesystem path.

### 5.3 Local image preparation

```text
ComposerImagePhase:
  queued -> reading -> preparing -> ready(preparedBytes, width, height, sha256)
                                 -> failed(localPreparationError)
  unavailable(preserved descriptor/alt text)

ComposerImages methods:
  addPickedImages(...)
  seedPreparedDraftImages(...)
  seedUnavailableDraftImage(...)
  replaceImage(mediaId, pickedImage)
  retryPreparation(mediaId)
  updateAltText(...), reorder(...), remove(...)
```

`ComposerImages` continues using `ComposerImageMediaService` for inspect/prepare/validate work, but stores the prepared output rather than only the picker preview. It does not watch `PostApiClient`, `BlobApiClient`, or scheduled repository providers. `canSaveDraftMedia` means every retained attachment is locally ready; `canSubmitImages` additionally rejects unavailable content. Prepared bytes are the one authoritative byte sequence for draft save, immediate upload, and new scheduled staging.

Existing scheduled-post edits may seed `ScheduledImageReady` with their private staged media ID and remain submission-ready through the current scheduled hydrator. Scheduled-post edits are not local-draft origins and never enter the local draft save path. Local drafts are opened only through the local hydrator and therefore always supply app-owned prepared bytes or an explicit unavailable item.

### 5.4 Submission request and coordinator

```text
enum ComposerSubmissionTarget { immediate, schedule }

class ComposerSubmissionRequest(
  target,
  capturedAccountLease,
  composerId,
  originDraft?,
  exactDraftWriteRequest?,
  orderedMedia,
  createImmediatePost(),
  stageAndCreateOrUpdateSchedule()
)

sealed class ComposerSubmissionState
  idle
  running(target)
  failed(safeMessage)
  succeeded(result)

abstract interface class SubmissionScreenAwake {
  Future<void> enable();
  Future<void> disable();
}

ComposerSubmissionController.submit(request, waitUntilOverlayPresented)
```

Widget-owned validation, form focus/commit, missing-alt confirmation, and synchronous payload construction run before `submit`. Once the request is confirmed valid, the controller moves to `running`; the widget immediately renders the non-dismissible full-screen overlay. `SubmissionBlockingOverlay` completes a one-shot presentation gate from its first post-frame callback. The controller awaits that gate before enabling screen-awake behavior or performing the existing-draft save, upload, staging, or final API work. Tests replace the gate with a controlled completer so “overlay visible before work” is deterministic rather than relying on scheduler timing. If the route disposes before presentation, submission terminates without a side effect.

After the overlay presentation gate completes, failure to enable screen-awake behavior is a pre-network terminal failure: dismiss the overlay, preserve the composer/draft, and show a safe retryable message. Once enabled, the controller executes:

```text
verify captured account lease
if originDraft exists:
  atomically save exact validated DraftWriteRequest
  verify lease again
materialize ordered media for target
verify lease again
perform authoritative post or schedule create/update/publish-now call
verify lease before user-visible/cache completion
if authoritative success and originDraft exists:
  delete originating local draft and refresh draft list
finish success
finally:
  disable screen awake
  leave running state / remove overlay
```

For immediate submission, `ComposerMediaUploader` processes images sequentially to make order and independent timeouts deterministic. Each missing/changed locally-ready image gets a fresh `CancelToken` and its own 60-second timer. Success is stored only in `ImmediateUploadRetryState` under `(composerId, mediaId, preparedSha256)`. A retry reuses entries with the same digest regardless of reorder, uploads missing/changed entries, and supplies final references in current image order. Removal/change invalidates the entry; provider disposal clears the entire ledger. No retry entry implements JSON serialization or enters a draft manifest.

For scheduled submission, the materializer reuses existing `ScheduledImageReady` identifiers and stages new local-ready prepared bytes through the current private route using the existing stable media ID. Each actual staging transfer receives its own 60-second cancellation budget. Existing server idempotency remains authoritative; the composer never invokes public blob upload for a scheduled target. Schedule capacity is checked through the current server path only when submission runs.

`CreatePost.create` returns or completes with the authoritative created `Post` and rethrows/maps failure so the controller can await it, while preserving current optimistic feed/cache updates after authoritative success. Widget listeners no longer own pop/error timing. The controller reports success only if the captured lease is still current. If account ownership becomes stale, it stops before the next side effect where possible, suppresses success/cache/UI mutation for the new account, preserves any local draft, and always releases the overlay/wakelock.

If deletion of an origin draft fails after the remote operation authoritatively succeeds, do not repeat the post/schedule. Keep the draft row and show a safe cleanup warning with an explicit delete retry; record the remote success in the one-shot controller result so the composer cannot submit it again during that completion path. This is the only non-network cleanup exception to “success deletes”: it preserves honesty and avoids duplicate publication.

### 5.5 Data-flow overview

```text
Picker -> ComposerImageMediaService -> ComposerImages.localReady
                                      |-> explicit Save -> snapshot adapter
                                      |                    -> LocalPostDraftRepository
                                      |-> validated Submit -> ComposerSubmissionController
                                                           |-> existing draft exact save
                                                           |-> immediate BlobApiClient
                                                           |   -> CreatePost / POST /v1/posts
                                                           |-> scheduled private staging
                                                           |   -> schedule create/update
                                                           |-> success-only local draft delete

Settings -> DraftsRoute -> localPostDraftsProvider(AccountKey)
                       -> DraftsPage -> repository list/delete
                       -> row tap -> repository get/readMedia
                                  -> DraftComposerHydrator
                                  -> standard or project composer
```

No arrow from draft save/open/list/delete reaches AppView, PDS, scheduled capacity, public caches, or notifications.

## 6. State, Providers, Controllers, Or DI

Use generated Riverpod providers consistently with the current app:

```text
applicationDocumentsDirectoryProvider (FutureProvider<Directory>, keepAlive)
  -> draftFileStoreProvider (Provider<DraftFileStore>)
  -> localPostDraftRepositoryProvider(AccountKey)
       (FutureProvider<LocalPostDraftRepository>, keepAlive per account)
       -> localPostDraftsProvider(AccountKey)
            (AsyncNotifier<List<LocalPostDraftSummary>>)
       -> draftThumbnailProvider((AccountKey, draftId, mediaId, revision))
            (AutoDisposeFutureProvider<Uint8List?>)

draftSaveControllerProvider((composerId, AccountKey))
  (AutoDisposeAsyncNotifier/DraftSaveState)
  -> activeAccountOperationCoordinator / captured lease
  -> localPostDraftRepositoryProvider
  -> localPostDraftsProvider refresh

composerImagesProvider(composerId)
  (existing AutoDisposeNotifier; local media service only)

submissionScreenAwakeProvider (Provider<SubmissionScreenAwake>)
composerSubmissionControllerProvider((composerId, AccountKey))
  (AutoDisposeNotifier<ComposerSubmissionState>)
  -> active account lease
  -> composerMediaUploaderProvider
  -> immediateUploadRetryStateProvider(composerId)
  -> createPostProvider / scheduledPostRepositoryProvider
  -> local draft repository/list provider when origin exists
  -> submissionScreenAwakeProvider
```

Provider requirements:

- Every repository/controller method captures `AccountKey` and the current activation lease before async work. It verifies both at completion boundaries and before any later side effect.
- Repository providers are keyed by account but do not delete storage on invalidation or sign-out. Re-sign-in reconstructs the same namespace. Remote terminal deletion is neither polled nor subscribed to.
- List state sorts `updatedAt` descending and stable draft ID ascending as the deterministic tie-break. `get`/open does not mutate timestamps.
- Thumbnail providers load only the first media item, are keyed by storage revision to invalidate after replacement, and map missing/corrupt bytes to the draft icon rather than logging raw errors.
- Save state distinguishes idle/preparing-files/saving/succeeded/failed so local work can present bounded progress. It never emits authored content in `toString`.
- Submission state alone controls overlay visibility. `running` rejects duplicate `submit` calls and owns a one-shot overlay-presentation gate. Controller disposal completes or abandons that gate safely and calls the idempotent screen-awake release path even if a future is unresolved.
- Existing `unsavedWorkGuardProvider` remains registered by each open composer. A local save updates its baseline; a successful save closes; Discard changes restores no in-memory snapshot because the route closes and the prior durable snapshot remains unchanged.

## 7. UI, Widgets, Routes, Or User-Facing Surfaces

### 7.1 Draft management

Add `DraftsRoute` at `/profile/settings/drafts` on the same root navigator as Scheduled posts. Settings shows a localized `Drafts` row beside Scheduled posts with no count or attention badge.

`DraftsPage` follows the existing `ScheduledPostsPage` layout and account-init gate:

- App bar title `Drafts`.
- Loading, safe error with Retry, and empty state.
- Newest-saved-first rows with first local thumbnail or kind icon, project title or trimmed standard text, `Post`/`Project`, and localized `updatedAt` date/time.
- Tap loads detail/media, then opens the matching composer with a stable `LocalPostDraftSeed` and captured owner.
- Delete asks for confirmation, waits for repository success, and removes the row only after success. A failure keeps it visible and retryable.
- Damaged manifest rows expose only unavailable/delete behavior. A valid manifest with damaged media opens recoverable authored fields and an `Image unavailable` attachment.
- No badge, search, filter, folder, bulk action, manual sort, artificial limit, or expiry UI.

### 7.2 Composer draft actions

`showPostComposerSheet` and `showProjectComposerSheet` receive an optional local draft seed/origin. Standard composer arguments assert that local draft origin is mutually exclusive with reply, quote, or scheduled-post edit origins. Project composer asserts local draft and scheduled edit are mutually exclusive.

For eligible new or local-draft composers:

- Show `Save draft` for new origins and `Save changes` for existing drafts.
- Disable it for untouched defaults, no meaningful change, any preparing/failed/unavailable image, or active save/submission.
- Saving snapshots current incomplete state without publication validation or network calls. Success closes the composer and shows exactly `Draft saved`; failure leaves every field/image intact and shows a safe actionable storage error.
- Back, drag-to-dismiss, close button, and route pop pass through one close handler. Dirty new work offers `Save draft`, `Discard`, `Keep editing`; dirty existing work offers `Save changes`, `Discard changes`, `Keep editing`.
- Keep editing dismisses the dialog. Discard closes without a write. Discard changes does not overwrite or delete the prior durable snapshot.
- Quote/reply composers retain the existing non-draft discard behavior and expose no save action, but still use submit-time upload and the common overlay.
- Scheduled-post edits remain server schedules, not local drafts, and keep their existing discard/edit behavior.

Opening a draft asynchronously seeds form values and media before normal editing. If a saved Later time is still valid, preserve it. If not, select Now and show a localized explanation without submitting or checking capacity. Project hydration writes each partial known form value directly; it does not force full form validation.

An unavailable attachment preserves its order and alt text, shows exactly `Image unavailable`, and offers Replace and Remove. Replace runs normal local preparation under the same stable media ID and invalidates any old stored origin. Submission remains unavailable until no unavailable/preparing/failed items remain.

### 7.3 Submission overlay

Wrap the composer scaffold/body in a `Stack` driven by `ComposerSubmissionState`:

```text
PopScope(canPop: !submissionRunning)
  Stack
    existing composer scaffold
    if running
      Positioned.fill
        Semantics(container: true, liveRegion: true, label: operationText)
          ModalBarrier(dismissible: false)
          Center(Column(CircularProgressIndicator, operationText))
```

Use theme surfaces/contrast and flexible text layout so the overlay remains readable in light/dark themes and at large text scale. Immediate standard/quote/reply/project and publish-now flows show exactly `Publishing your post…`; new/edit schedule flows show exactly `Scheduling your post…`. The barrier covers the full route, blocks taps and duplicate actions, and has no cancel affordance. Fast validation and missing-alt confirmation remain visible before the overlay begins.

## 8. Error, Loading, Empty, And Edge States

| State / Case | Planned Handling | Requirement IDs | Test IDs |
|---|---|---|---|
| Untouched defaults or ineligible quote/reply | No draft save action/enabled state; retain current close protection for ineligible composers | FR-001, RULE-002–RULE-004 | AT-001, AT-003, AT-021; UT-001; IT-008–IT-010 |
| Image reading/preparing | Local progress only; save/submit disabled; no upload/staging call | FR-010, FR-021, NFR-004 | AT-009, AT-020, AT-025; UT-008, UT-017; IT-010, IT-018 |
| Local image preparation failure | Safe retry/remove UI; keep authored fields; no durable save until resolved | FR-009, FR-021 | AT-025; UT-007, UT-008; IT-008, IT-009 |
| Missing/corrupt saved media | Preserve descriptor/order/alt and other authored fields; show `Image unavailable`; allow replace/remove/delete; block submission | FR-015, FR-016 | AT-015; UT-003, UT-007; IT-004 |
| Unsupported/corrupt manifest | Show content-free unavailable/deletable row; do not rewrite, follow unsafe paths, or hide healthy drafts | FR-015, NFR-002 | AT-015, AT-019; UT-002, UT-007, UT-015; IT-004, IT-016 |
| New-media write failure | Remove only safe unreferenced artifacts, leave composer open, list no partial draft | FR-016, NFR-001 | AT-017; UT-007; IT-003 |
| Manifest replacement failure | Preserve previous manifest/media and composer; safe retryable error | FR-016, NFR-001 | AT-016, AT-017; UT-006, UT-007; IT-002, IT-003 |
| Old-file cleanup failure after switch | Keep new snapshot authoritative; defer safe orphan cleanup to reconciliation; content-free diagnostic only | FR-015, NFR-001, NFR-002 | AT-016, AT-019; IT-002, IT-016 |
| Empty drafts list | Local empty state; no network dependency | FR-004, FR-017 | AT-005; IT-006 |
| Draft list/open/delete failure | Safe error and retry; never claim delete/save/open success; keep row where applicable | FR-007, FR-009, FR-016 | AT-005, AT-008, AT-017; IT-003, IT-006 |
| Equal `updatedAt` or open without save | Stable-ID tie-break; opening does not reorder | FR-004, RULE-007 | AT-005, AT-007; UT-004; IT-006 |
| Ordinary sign-out / later sign-in | Dispose in-memory providers only; keep account directory; rebuild same namespace on return | FR-020, RULE-006 | AT-023; IT-005 |
| Remote terminal deletion elsewhere | No poll, listener, or purge in v1; retain until a defined local removal event | FR-020, RULE-006 | AT-023; IT-005 |
| Stale account lease mid-operation | Stop before next side effect when possible, suppress stale success/new-account mutation, keep draft, release overlay/wakelock | FR-014, NFR-005 | AT-018; UT-016; IT-005, IT-015 |
| Saved Later time now invalid | Reset to Now, explain, do not submit or consume scheduled capacity | FR-019, RULE-005 | AT-024; UT-005 |
| Validation or alt confirmation fails/cancels | Do not start overlay, wakelock, network, or pre-submit draft write | FR-013, FR-022 | AT-013, AT-027; UT-013; IT-013, IT-014 |
| Immediate upload partially succeeds | Exit overlay on failure; retain same-composer unchanged refs in memory; retry missing/changed only | FR-009, FR-011, FR-023 | AT-026; UT-010, UT-011; IT-011 |
| Individual upload reaches 60 seconds | Cancel only that transfer, exit overlay, release wakelock, preserve composer/draft and retry ledger | FR-009, FR-011, FR-012, NFR-005 | AT-014, AT-026; UT-011, UT-012; IT-011–IT-013 |
| Composer closes after partial immediate upload | Dispose controller and all transient refs; drafts persist no remote blob data | FR-023 | AT-026; UT-010; IT-011 |
| Scheduled staging/create failure | Keep stable media IDs and composer/draft; retry through current idempotent path; never call public blob API | FR-009, FR-012, FR-023 | AT-011, AT-026; IT-012 |
| Existing local draft submission | Overlay, exact atomic save, then network; authoritative success removes origin; failure retains attempted snapshot | FR-008, FR-022 | AT-012, AT-027; UT-013; IT-014 |
| Never-saved submission | No recovery-draft write; ordinary in-memory state remains on failure | FR-022, RULE-003 | AT-027; UT-013; IT-014 |
| Remote success followed by local-delete failure | Do not repeat remote submission; retain visible draft and present safe cleanup/delete retry warning | FR-007, FR-008, FR-009 | AT-008, AT-012; IT-006, IT-014 |
| Route/controller disposal or unexpected exception | Idempotently disable wakelock, leave running state, unblock UI if route survives, preserve saved draft | FR-009, NFR-005 | AT-014; UT-012; IT-013 |
| Maximum four-image local I/O | Async repository/media service work and bounded saving/loading feedback; no synchronous whole-operation file work in build | NFR-004 | AT-020; UT-017; IT-018 |

## 9. Test Implementation Plan

Each row is a red-green-refactor slice. Add the listed failing test(s), make the smallest production change that passes them, run the focused suite, and only then advance.

| Order | Test ID | Target | Setup / Fixture | Initial Expected Failure |
|---|---|---|---|---|
| 1 | AT-009, UT-008, IT-010 | `app/test/feed/providers/composer_images_provider_test.dart`, `composer_image_state_test.dart` | Existing recording blob client plus valid prepared image fixture | Selecting a valid image still invokes upload and cannot reach a local-ready state without a remote blob |
| 2 | AT-001, AT-021, UT-001 | `app/test/drafts/composer/draft_save_eligibility_test.dart` | Standard/project/quote/reply states and one-change-at-a-time matrix | No eligibility policy exists and default/ineligible cases cannot be classified |
| 3 | AT-004, AT-015, UT-002, UT-003, UT-015 | `app/test/drafts/models/draft_manifest_test.dart`, `draft_media_descriptor_test.dart`, `draft_privacy_test.dart` | Complete v1 snapshots, partial project fields, malformed/future JSON, unsafe refs, privacy canaries | No versioned domain/codec validates or safely redacts the data |
| 4 | AT-007, AT-016, AT-017, UT-006, UT-007, IT-002, IT-003 | `draft_update_plan_test.dart`, `draft_storage_error_test.dart`, repository recovery/failure suites | In-memory/fake file store with failpoints at every write/switch/cleanup boundary | No immutable update plan or atomic failure behavior exists |
| 5 | AT-002, AT-008, AT-015, AT-023, IT-001, IT-004, IT-005 | `file_draft_repository_test.dart`, recovery and retention suites | Temporary Documents root, original/prepared canaries, damaged bundles, Alice/Bob namespaces | Repository cannot save/restart/offline-open/delete/reconcile/retain account bundles |
| 6 | AT-005, AT-008, AT-018, AT-020, UT-004, UT-014, UT-016, UT-017, IT-006, IT-015, IT-018 | Row model, providers, async-state, and `drafts_page_test.dart` | Fake account repositories, delayed futures, equal times, switch barriers, four-image fixture | No account-keyed list/controller/page state or bounded progress exists |
| 7 | AT-005, IT-007 | `settings_page_test.dart`, `settings_routes_test.dart` | Production router/localization harness | Drafts Settings row and typed root route do not exist |
| 8 | AT-002–AT-004, AT-006, AT-024, UT-005, IT-008, IT-009 | Draft hydrator, schedule restoration, standard/project draft composer suites | Complete/incomplete snapshots, valid/expired Later time, local media repository | Composers cannot seed a stable local draft or round-trip incomplete state |
| 9 | AT-003, AT-025, IT-008, IT-009 | Existing discard tests plus standard/project draft composer tests | New/existing origins, save completers, preparing/failed/ready media | Eligible close flow has only two choices and no explicit successful/failed save lifecycle |
| 10 | AT-010, AT-011, UT-009, IT-017 | `submission_materialization_plan_test.dart`, existing post API test, new scheduled API-client test | Recording clients and `http_mock_adapter` fixtures for current routes | There is no target plan and cancellation threading/wire preservation is unproved |
| 11 | AT-026, UT-010, UT-011 | `immediate_upload_retry_state_test.dart`, `media_upload_timeout_test.dart` | Fake clock/timers, digest-distinct images, partial success/reorder/change/close | Retry refs are not modeled and transfers have no independent one-minute budget |
| 12 | AT-010–AT-012, AT-022, AT-026, IT-011, IT-012 | `composer_submission_coordinator_test.dart`, `scheduled_post_submission_test.dart` | Ordered local-ready images, recording blob/post/staging/schedule clients, capacity and failure barriers | Submission is still widget-owned and cannot guarantee target ordering, retries, or success-only cleanup |
| 13 | AT-013, AT-014, UT-012, IT-013 | `submission_overlay_state_test.dart`, `submission_overlay_test.dart`, composer widget suites | Gated validation/confirmation/API futures, fake navigator/semantics/wakelock | No common blocking overlay or leak-proof screen-awake lifecycle exists |
| 14 | AT-012, AT-027, UT-013, IT-014 | `draft_submission_policy_test.dart`, `draft_submission_lifecycle_test.dart` | Ordered event recorder for validation/overlay/save/network/delete and new/existing origins | Existing draft is not snapshotted before network and new origin policy is absent |
| 15 | AT-018, UT-016, IT-015 | `drafts_account_boundary_test.dart`, account-switch routing test | Lease changes at save/upload/final API/delete completion boundaries | Stale operations can still report or mutate after account change |
| 16 | AT-019, UT-015, IT-016 | `draft_privacy_test.dart` end-to-end pass | Logger/error/event/metric recorders with canaries in all private fields | New errors/model strings may expose private identifiers/content until redaction is complete |
| 17 | REG-001–REG-010 | Existing composer/media/scheduled/settings/router/API suites plus full app test/analyze | Updated fakes and generated output | Timing refactor initially breaks assumptions about eager upload and inline submission state |

Manual release checks carried from `02-acceptance-tests.md`:

- `MAN-001`: physical-device save/process-kill/relaunch at each file boundary; prior or new snapshot remains complete.
- `MAN-002`: physical iOS/Android submit success/failure/timeout/disposal; screen remains awake only while overlay is visible.
- `MAN-003`: both themes, large text, screen reader/busy semantics, full-route blocking, exact overlay copy.
- `MAN-004`: offline sign-out/re-sign-in retention, no time expiry, explicit local delete/success/app-data removal, and no remote-deletion poll.

## 10. Sequencing And Guardrails

- First TDD step: change only the image-provider contract. Add a red test proving selection reaches a locally-ready phase while a recording blob client receives zero calls. Then remove eager upload from `ComposerImages` without yet building draft persistence.
- Dependencies between work items:
  - Local-ready prepared bytes are required before manifest media tests, draft repository save, or submit-time transfer.
  - Manifest/domain contracts and filesystem fault injection precede provider/page/composer integration.
  - Draft snapshot adapters precede save/open UI.
  - Target materialization, retry, and timeout policies precede the shared controller and overlay wiring.
  - The controller precedes removing inline submission handling from both composers.
  - Route/localization/provider generation happens alongside its first consumer, then once more at the final integration gate.
- Guardrails:
  - Never import `BlobApiClient`, `PostApiClient`, or scheduled repositories into local draft repository/model/provider modules.
  - Never upload/stage during selection, preparation, alt edits, reorder, save, open, close, or delete.
  - Preserve existing `/v1/blobs/images`, `/v1/posts`, scheduled-media, and schedule route contracts exactly; add no endpoint.
  - Keep prepared bytes authoritative. Do not persist picker handles, original paths, unstripped originals, remote blob references, access tokens, or scheduled capacity state.
  - Keep the manifest codec explicit and versioned. Do not serialize FormBuilder values generically or use generated mapper output as the durable schema.
  - Validate every ID/reference before joining paths; filesystem callers never supply an arbitrary path. Do not follow out-of-root links.
  - Write changed media immutably, flush/verify it, atomically switch the manifest, then clean old files. Never overwrite referenced media in place.
  - Keep draft errors, diagnostics, state strings, analytics, traces, metrics, crash reports, and test failure messages free of content, file paths, raw account IDs, stable draft IDs, and thumbnails.
  - Capture account lease once per operation and recheck before each later side effect and user-visible success.
  - Overlay begins only after validation/alt confirmation and ends with an idempotent `finally` cleanup. Screen-awake enable/disable is owned by the same lifecycle.
  - A successful remote operation is never retried merely because local draft cleanup failed.
  - Local draft save/open/delete does not mutate feed caches, scheduled count, notification state, or AppView state.
  - Preserve the four-image limit and existing media validation, facets, languages, payload, scheduling, hydration, and optimistic-cache behavior unless the approved requirements explicitly change timing.
- Out of scope:
  - SQLite, SharedPreferences draft content, database migrations, AppView draft APIs/storage, PDS draft records, cross-device sync/recovery, lexicon changes, combined multipart APIs, worker redesign, automatic saving, crash recovery for never-saved work, quote/reply drafts, folders/search/tags/limits/expiry, custom encryption, remote terminal-deletion propagation, blob cleanup, and background submission.

Focused verification during implementation:

```text
just app-test test/feed/providers/composer_images_provider_test.dart
just app-test test/drafts
just app-test test/feed/composer test/feed/widgets/submission_overlay_test.dart
just app-test test/scheduled_posts
just app-test
just app-analyze
cd app && dart format <changed Dart paths>
cd app && dart run build_runner build --delete-conflicting-outputs
git diff --check
```

Run formatting and generation only after the relevant red-green slice is passing, and inspect generated/format diffs to avoid unrelated churn.

## 11. Risks And Open Questions

| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPR-001 | Non-blocking | Atomic replace and process-termination behavior depends on the mobile filesystem implementation | A faulty adapter could expose a partial manifest or orphan files | Keep temp manifest in the same directory, flush writes, use one replace primitive, inject every boundary, reconstruct repository in tests, and complete `MAN-001` on iOS/Android |
| CPR-002 | Non-blocking | The existing project composer stores heterogeneous partial FormBuilder values | Generic serialization could lose incomplete work or become unstable | Use an explicit `ProjectComposerFields` whitelist and per-type adapter; fail tests whenever a known field is not mapped |
| CPR-003 | Non-blocking | Prepared bytes increase in-memory use while a composer with four images is open | Large media could cause UI jank or pressure | Retain only the existing bounded maximum, run preparation/hash/file work asynchronously, lazy-load management thumbnails, expose progress, and cover `UT-017`/`IT-018`/`MAN-003` |
| CPR-004 | Non-blocking | Wakelock calls can fail or the route can dispose during an unresolved submission | Screen could sleep or remain awake | Abort before network if enable fails; make disable idempotent and best-effort in `finally` and provider disposal; test all terminal paths and perform `MAN-002` |
| CPR-005 | Non-blocking | Immediate partial upload leaves unreferenced remote blobs if the composer closes | PDS storage may retain blobs not referenced by a post | Explicitly accepted by NG-007; keep refs transient and rely on normal PDS cleanup rather than adding deletion behavior |
| CPR-006 | Non-blocking | Remote success followed by local draft-delete failure creates a visible stale source draft | Member could otherwise publish it twice | Treat remote result as final, prevent automatic resubmit, retain the row, and show a safe explicit cleanup warning/delete retry |
| CPR-007 | Non-blocking | Account switching can occur between remote dispatch and completion | A side effect already accepted by the captured account cannot be undone | Dispatch only with captured owner credentials, recheck before subsequent effects, suppress stale success/new-account mutation, and retain draft for reconciliation |
| CPQ-001 | Non-blocking | Should OS backup restore local draft files on another device? | Backup behavior differs by platform and configuration | No app-level guarantee or custom exclusion in v1; normal private OS app-data backup is accepted by requirements |
| CPQ-002 | Non-blocking | Should remote terminal account deletion purge local drafts? | Local sensitive files may remain after an unobservable remote event | Deliberate v1 decision: no polling/listener/API; retain until explicit local deletion, confirmed success, or app-data removal |

Blocking questions: None identified.

## 12. Handoff To TDD Builder

- Coding plan: `04-coding-plan.md`
- TDD execution plan: `05-implementation-plan.md`, to be created by `implement-tdd` before production changes
- Start with test: update `app/test/feed/providers/composer_images_provider_test.dart` so selecting a valid image becomes locally ready and a recording blob client observes zero uploads
- Focused command: `just app-test test/feed/providers/composer_images_provider_test.dart`
- First implementation target after the red test: `app/lib/feed/providers/composer_image_state.dart` and `app/lib/feed/providers/composer_images_provider.dart`; remove eager transfer ownership while retaining current local preparation/validation behavior
- Notes:
  - Run strict red-green-refactor slices in the order in section 9.
  - Keep the API, AppView, database, worker, and lexicon surfaces unchanged.
  - Treat filesystem fault injection, no-pre-submit-network spies, privacy canaries, account fences, timeout tests, and wakelock cleanup as early guardrails.
  - The local storage decision is approved and deliberately conflicts with the broad AppView/Postgres draft guidance.
  - Complete all four manual checks before release; they supplement rather than replace automated coverage.
