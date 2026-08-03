# Requirements: Local Post Drafts And Submit-Time Media Uploads

## 1. Initial Request

Add the ability for people to save post drafts locally on their device, including durable app-owned copies of attached media, and manage those drafts from a page similar to Scheduled posts. Determine whether SQLite is necessary or whether a simpler local persistence approach is sufficient.

As part of the same change, stop uploading selected media optimistically. Media must be uploaded only when the member submits the composer, whether they publish immediately or schedule the post. Use this boundary to simplify the standard and project composer submission paths.

While any post submission is running, show a full-screen blocking loading overlay with a spinner and simple status text, and keep the foreground device screen awake until the operation finishes.

## 2. Current Codebase Findings

- Relevant files:
  - `app/lib/feed/widgets/post_composer_sheet.dart` owns standard, reply, quote, and scheduled-standard composer behavior, submission, discard protection, and scheduled-media hydration.
  - `app/lib/projects/widgets/project_composer_sheet.dart` owns the multi-page project composer, project validation, immediate submission, scheduled submission, and scheduled-media hydration.
  - `app/lib/feed/providers/composer_images_provider.dart` currently runs selection, read, prepare, validation, and `POST /v1/blobs/images` upload as one eager pipeline immediately after image selection.
  - `app/lib/feed/providers/composer_image_state.dart` models transient local bytes plus eager PDS-upload state; both composers require images to be uploaded before enabling submission.
  - `app/lib/feed/providers/create_post_provider.dart`, `app/lib/feed/data/post_api_client.dart`, and `app/lib/shared/media/blob_api_client.dart` currently split immediate publication into blob upload followed by `POST /v1/posts`.
  - `app/lib/scheduled_posts/` contains an implemented owner-scoped management page, repository/provider structure, private-media staging, edit hydration, and Settings route that local drafts can mirror without sharing server persistence.
  - `app/lib/auth/providers/unsaved_work_guard_provider.dart` already protects dirty composers during account changes and navigation.
  - `app/lib/settings/pages/settings_page.dart` and `app/lib/router/` expose Scheduled posts from Settings and provide the natural location for Drafts.
  - `app/pubspec.yaml` has `shared_preferences` as a runtime dependency, but no runtime SQLite or `path_provider` dependency. `sqflite` and `path_provider_platform_interface` are test-only dependencies used indirectly by image-cache tests.
- Existing patterns:
  - Composer instances have stable UUIDs and account-operation leases.
  - Standard and project scheduled posts already round-trip complete payloads and app-owned media bytes through dedicated hydrators.
  - Account-bound providers fence stale work when the active account changes.
  - Settings management pages use root-navigator routes, async providers, list rows, edit-on-tap, explicit deletion, empty states, and safe errors.
- Current behavior:
  - Choosing an image immediately reads, prepares, validates, and uploads it through the AppView to the member's public PDS, before Post or Schedule is tapped.
  - Immediate submission references the already-uploaded PDS blob in `POST /v1/posts`.
  - Scheduled submission reprocesses retained local bytes, uploads a second private copy to AppView staging, and creates or updates the schedule only after staging succeeds.
  - Closing a dirty composer asks only whether to discard; no durable local draft is created.
  - Submission is represented by disabled controls and small in-place progress UI. There is no common full-screen overlay or screen-awake lifecycle.
- Constraints discovered:
  - Published media belongs on the member's PDS; unpublished draft data must not be written there.
  - Scheduled media must continue to use authenticated private AppView staging, because the server worker needs a durable copy after the app closes.
  - A draft store must preserve app-owned bytes rather than gallery/photo-picker references, which may become unavailable.
  - SQLite cannot make external media files transactional by itself; even a database design still needs file lifecycle and recovery logic.
  - Current Flutter authentication is mobile-first for iOS and Android; Web and desktop are not current product targets.
  - The current media limit is four images and current preparation/validation rules remain authoritative.
- Current package guidance checked:
  - `path_provider` documents persistent application directories for user-generated or non-recreatable app data and warns that temporary/cache directories may be cleared at any time.
  - `sqflite` provides transactions and schema-version migrations for structured relational data, but those facilities do not remove the need for separate media files.
  - `wakelock_plus` supports enabling screen-awake behavior only while the relevant UI is active and disabling it when that UI finishes.
- Test/build commands discovered:
  - `just app-test [path-or-args]`
  - `just app-analyze`
  - `dart format` from `app/` for changed Dart sources during implementation
  - `dart run build_runner build --delete-conflicting-outputs` from `app/` when generated providers, routes, or mappers change
  - `git diff --check`

## 3. Clarifying Questions And Decisions

### Q1: Does local draft metadata need SQLite?

Answer: Confirmed during grilling: use the simpler file-backed approach for the first release.

Decision / implication: Do not introduce SQLite for the first release. Use a versioned file-backed repository: one private app-managed bundle per draft containing an atomic JSON manifest and app-owned media files. The feature needs only owner-scoped list, get, save, and delete operations over a small local collection; it does not need relational joins, full-text search, or complex queries. Preserve the repository abstraction so SQLite can replace the file implementation later without changing composer or management UI contracts. This is a deliberate product exception to the repository's broad architectural guidance that currently lists drafts as AppView/Postgres private data: these explicitly saved post drafts are device-local by design, with no sync or portability guarantee.

### Q2: Which content types can be saved as drafts?

Answer: Confirmed during grilling: original top-level standard posts and project posts only.

Decision / implication: For the first release, support original top-level standard posts and project posts, matching the existing Scheduled posts management/edit seams. Quote and reply/comment drafts are excluded, but their existing submission flows still receive the submit-time upload and loading-overlay behavior.

### Q3: Is saving explicit or automatic?

Answer: Confirmed during grilling: saving is explicit and there is no background autosave.

Decision / implication: Use explicit saving. A dirty eligible composer provides a Save draft action, and attempting to close it provides Save draft, Discard, and Keep editing choices. Editing an existing draft updates that draft rather than creating a duplicate. Background keystroke autosave and crash recovery for never-saved changes are out of scope.

### Q4: What media representation is durable enough?

Answer: Confirmed during grilling: preserve CraftSky's prepared publishable bytes, not the untouched original.

Decision / implication: Image selection immediately performs local read, validation, metadata stripping, resize/re-encode, and preparation. A saved draft owns the resulting publishable bytes inside persistent application storage. Media order, MIME type, dimensions/aspect ratio, filename display metadata, and alt text round-trip with the manifest. The untouched original, temporary/cache paths, and gallery references are not retained as authoritative sources.

### Q5: When may media use the network?

Answer: Confirmed during grilling: only when the member explicitly submits the composer.

Decision / implication: Image selection may read, inspect, prepare, validate, and render local bytes, but it shall not call either the public PDS blob endpoint or private scheduled-media staging. Immediate publication uploads to `POST /v1/blobs/images` during submission; scheduling uploads to the existing private scheduled-media endpoint during submission. Existing AppView route shapes can remain unchanged.

### Q6: What does the blocking submission state show?

Answer: Confirmed during grilling: a simple full-screen spinner and exact operation text are sufficient for now.

Decision / implication: Fast local form validation and missing-alt-text confirmation run first. Once the member confirms a valid submission, the overlay appears before image transfer and final API work. It is non-dismissible, has no cancel action, and blocks all composer interaction and navigation. Immediate publication shows `Publishing your post…`; scheduling shows `Scheduling your post…`. The foreground screen-awake state begins with the overlay and is always released when the operation succeeds, fails, times out, is abandoned because ownership changed, or the overlay is disposed.

### Q7: What qualifies as a saveable draft?

Answer: Confirmed during grilling: the composer may be incomplete, but it must contain a deliberate change from its initial state.

Decision / implication: Text, an attachment or alt-text edit, a changed language selection, any user-changed project field, or a changed scheduling choice/time makes the composer saveable. Untouched defaults such as the primary language or default project status do not create an empty draft. Save draft remains disabled until every attached image has completed local preparation successfully; failed media must be retried or removed. A successful save closes the composer and shows `Draft saved`; failure keeps it open and intact.

### Q8: How are drafts listed and retained?

Answer: Confirmed during grilling: use a simple Settings management page, with no expiry or count limit.

Decision / implication: Drafts appear newest-saved first beside Scheduled posts. Rows show the first image or a draft icon, project title or trimmed post text, kind, last-saved date/time, tap-to-edit, and confirmed delete. There is no count badge, search, filter, folder, bulk action, manual sort, automatic expiry, or artificial count limit. Ordinary sign-out retains drafts for the same account on that installation. The first release does not add a device-visible signal for terminal account deletion elsewhere, so that remote event does not purge local files; drafts remain until explicit local deletion, confirmed successful publication/scheduling, or app-data removal. Normal private OS app-data backup behavior is allowed, but CraftSky provides no draft sync or recovery guarantee.

### Q9: How do retries and timeouts work?

Answer: Confirmed during grilling: uploads use a one-minute timeout per image and reuse only transient successful results.

Decision / implication: Successful immediate blob references remain in memory only while the same composer is open and the corresponding prepared bytes are unchanged. A retry uploads only missing/changed images. Closing or restarting discards those remote references; drafts never persist them. Scheduled staging continues using its existing idempotent media identifiers. A one-minute per-image upload timeout dismisses the overlay, releases screen-awake state, and preserves the composer/draft for retry.

### Q10: How are existing draft files updated safely?

Answer: Confirmed during grilling: avoid copying the entire bundle, but do not overwrite referenced media in place.

Decision / implication: Reuse unchanged immutable media files, write new or changed media under new immutable filenames, atomically replace the small manifest, and only then delete files no longer referenced. This keeps the implementation smaller than full-bundle copy-on-write while ensuring an interrupted save leaves the previous manifest and media usable.

### Q11: How does damaged local data recover?

Answer: Confirmed during grilling: keep recoverable drafts visible rather than hiding or deleting them.

Decision / implication: Missing/corrupt media renders as `Image unavailable`; text, project fields, order, and alt text remain editable; publication is blocked until the item is removed or replaced; deletion remains available. Unsupported/corrupt manifests expose safe unavailable/delete behavior without leaking content or paths. The platform app sandbox and normal device data protection are sufficient; no custom encryption or biometric gate is required.

### Q12: What is persisted when a member submits?

Answer: Confirmed during grilling: existing drafts protect the exact attempted version, but never-saved composers do not silently become drafts.

Decision / implication: After validation, submitting an edited existing draft shows the overlay, atomically saves the exact validated composer state under that blocking lifecycle, and only then begins network work. Success deletes the draft; failure or termination leaves the attempted version recoverable. A never-saved composer remains in memory only and does not create a recovery draft merely because Post or Schedule was tapped.

## 4. Candidate Approaches

### Option A: Versioned draft bundles plus a shared submit-time media coordinator

Summary: Store each account-owned draft as a private application-support directory containing a versioned JSON manifest and its media files. Split the current image pipeline into local selection/preparation and explicit submission materialization. Route both standard and project composers through a common coordinator that owns the overlay, account fence, screen-awake lifecycle, media transfer, and final create/schedule call.

Pros:

- Smallest persistence surface for list/get/save/delete.
- App-owned media files satisfy the durability requirement without database BLOBs.
- Immutable changed-media writes, atomic manifest replacement, and startup reconciliation prevent partially saved drafts from replacing the last good reference set without copying the full bundle.
- Avoids database schema/migration overhead while retaining a repository seam for future replacement.
- Removes eager upload from the image provider and centralizes the currently duplicated submit-time behavior.
- Reuses the existing immediate and scheduled AppView APIs.

Cons:

- Listing requires reading bounded manifests rather than issuing a database query.
- File lifecycle, atomic replacement, and orphan cleanup must be implemented carefully.
- A future large, searchable, filterable draft library could outgrow the simple manifest index.

Risks:

- A careless save order could orphan files or lose the previous draft on interruption.
- Corrupt or externally removed app files need explicit recovery behavior.

### Option B: SQLite metadata plus app-managed media files

Summary: Add a runtime SQLite package, store draft metadata and ordered media rows in tables, and keep image bytes as private files referenced by database rows. Use the same shared submit-time media coordinator as Option A.

Pros:

- Transactional metadata updates, indexed ordering, and efficient filtering.
- Clear schema-version migration path if draft volume and features grow substantially.
- Easier to add search, folders, tags, or retention queries later.

Cons:

- Adds a runtime database dependency, schema, migration, test-platform setup, and lifecycle code now.
- Media files remain outside the database, so database transactions cannot prevent every file/database partial-failure case.
- More infrastructure than the current small list/get/save/delete use case needs.

Risks:

- The extra abstraction can slow delivery without improving current user-visible behavior.
- Database rows and media files can still drift unless the same reconciliation logic is built.

### Option C: SharedPreferences metadata plus separate media files

Summary: Serialize all draft metadata into preferences and copy images to private files.

Pros:

- Uses an existing dependency.
- Very little initial setup.

Cons:

- Shared preferences are intended for small preferences, not an evolving collection of private content and project payloads.
- Whole-collection rewrites, size growth, corruption isolation, and per-draft atomic updates are poor fits.
- Still requires the complete media-file lifecycle.

Risks:

- One malformed or failed preferences write can affect the entire collection.
- The approach becomes difficult to migrate and test as draft payloads evolve.

## 5. Recommended Direction

Recommended approach: Option A — versioned draft bundles plus a shared submit-time media coordinator.

Why:

- SQLite is not justified by the current access pattern, and it would not solve media-file durability on its own.
- A file-backed repository makes each draft independently readable, writable, recoverable, and deletable while keeping unpublished media in persistent private application storage.
- Reusing unchanged immutable media and atomically replacing only the manifest keeps the agreed safety boundary narrow and the implementation simpler than full-bundle copy-on-write.
- A repository interface preserves the option to migrate to SQLite later if draft search, folders, large volume, or complex queries become real requirements.
- Separating local media preparation from network materialization removes eager PDS uploads and gives immediate and scheduled submissions one explicit, testable transfer boundary.
- A common coordinator provides one place to implement the full-screen overlay, screen-awake lifecycle, account fencing, failure recovery, and success cleanup across standard and project composers.

## 6. Problem / Opportunity

Members currently lose unfinished composer work when they deliberately leave it, and selected media may disappear from the original picker location before they return. At the same time, CraftSky uploads media to the public PDS before the member has committed to publishing, then performs a second private upload if the member schedules the post. Local drafts and an explicit submit-time media boundary protect unpublished intent, remove unnecessary early network/public side effects, and create a simpler shared composer lifecycle.

## 7. Goals

- G-001: Let members deliberately save and later resume unfinished top-level standard and project posts on the same device.
- G-002: Preserve every saved draft's complete editable content and media without depending on the original gallery/file-picker asset.
- G-003: Ensure selected media is never uploaded before the member taps Post, Reply, or Schedule.
- G-004: Reuse one clear submission lifecycle across standard and project composers.
- G-005: Make submission visibly blocking and keep the foreground screen awake until the operation finishes.
- G-006: Keep local drafts private, account-scoped, resilient to interrupted writes, and safe to delete.

## 8. Non-Goals

- NG-001: Server-side or PDS-backed draft storage, CraftSky cross-device sync, sharing, or collaboration.
- NG-002: Automatic per-keystroke saving or recovery of changes that have never been explicitly saved.
- NG-003: Quote-post, reply, or comment drafts in the first release.
- NG-004: Draft folders, search, tags, sort controls, bulk actions, storage quotas, or an artificial draft-count limit.
- NG-005: Video, audio, or new media types and changes to existing image count/type/size/alt-text rules.
- NG-006: A new combined multipart post API, new AppView route, scheduled-worker redesign, database migration, or lexicon change.
- NG-007: Deleting unreferenced blobs from a member's PDS after a partial immediate submission; normal PDS cleanup remains authoritative.
- NG-008: Background execution after the app is suspended or terminated. Screen-awake behavior applies while the blocking submission UI is active in the foreground.
- NG-009: Custom application-level encryption, passcodes, or biometric locks for drafts.
- NG-010: A richer submission animation, progress breakdown, cancel button, or entertainment content in the overlay.
- NG-011: Web, macOS, Windows, or Linux draft persistence in this first mobile release.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Member | Signed-in CraftSky user composing a standard or project post | Save incomplete work, return later, and publish or schedule it safely |
| Multi-account member | Person with more than one account stored on one installation | See and mutate only the active account's drafts and never submit under the wrong account |
| CraftSky app | Flutter client that owns local draft files and orchestrates submission | Atomic persistence, media ownership, recovery, upload timing, and lifecycle cleanup |
| AppView | Existing authenticated API and scheduled-publication service | Receive media only after explicit submission and keep existing immediate/scheduled contracts |
| Device OS | iOS or Android storage, lifecycle, and screen-awake host | Provide private persistent app storage and foreground screen-awake behavior |

## 10. Current Behavior

The composers hold all unsaved state in memory. Their discard guards can prevent accidental loss during navigation or account changes, but choosing Discard or terminating the process removes the work. Selected images are read, transformed, and immediately uploaded to the PDS before final submission. A scheduled submission then reprocesses retained bytes and uploads an additional private copy. Immediate and scheduled operations expose different in-composer progress states, and neither owns a shared full-screen screen-awake lifecycle.

## 11. Desired Behavior

An eligible composer can explicitly save a complete local snapshot without requiring post-valid content or a network connection once every attached image is locally prepared. Saving places an atomic manifest and durable prepared-media copies in the active account's private local draft repository, closes the composer, and confirms `Draft saved`. Settings contains Drafts, where rows show a useful preview, kind, last-saved time, and first image thumbnail/draft icon; tapping opens the correct composer with every saved value restored. Drafts can be updated or deleted, remain after ordinary sign-out, and disappear only after explicit local deletion, confirmed successful publication/scheduling, or removal of app data. Terminal account deletion outside the installation has no first-release propagation path to local files.

Selecting media immediately produces prepared publishable bytes but performs local work only. Post/Reply/Schedule first validates current content and completes any missing-alt confirmation. A valid submission then starts one blocking foreground lifecycle: save the exact attempted state when editing an existing draft, show the full-screen spinner and exact status text, keep the screen awake, upload through the appropriate existing endpoint with a one-minute timeout per image, and complete the final API call. Success closes the composer and removes an originating local draft; failure removes the overlay, releases screen-awake state, and leaves content available for correction or retry. A never-saved composer is not silently persisted by submission.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | Members shall be able to deliberately save and later resume unfinished eligible posts on the same device. | Prevents loss of meaningful composition work. | Prompt | AC-001, AC-004, AC-006 |
| BR-002 | Business | Must | A saved draft shall remain unpublished and private to the owning account until the member explicitly submits it. | Draft content is private-by-intent. | Prompt / Architecture | AC-002, AC-019 |
| BR-003 | Business | Must | CraftSky shall not upload selected media before the member explicitly submits the composer for immediate publication or scheduling. | Removes premature public uploads and duplicate scheduled transfer. | Prompt | AC-009, AC-010, AC-011 |
| BR-004 | Business | Must | Submission shall present one clear, blocking, screen-awake state from start until a terminal result. | Prevents duplicate actions and device sleep during a potentially longer transfer. | Prompt | AC-013, AC-014 |
| FR-001 | Functional | Must | An eligible new composer shall provide Save draft when it differs meaningfully from its initial state through text, media/alt text, changed languages, user-changed project fields, or changed scheduling intent. Untouched default values alone shall not qualify, and saving shall not require publication validation. | Drafts must support incomplete deliberate work without creating empty default-only items. | Prompt / Grilling | AC-001 |
| FR-002 | Functional | Must | A draft snapshot shall preserve its stable ID, owner, kind, created/updated timestamps, text, language selection, relevant standard/project fields, selected When choice, eligible future schedule time, and schema version. | Reopening must reconstruct the same editable intent. | Discovery | AC-004, AC-006, AC-024 |
| FR-003 | Functional | Must | Every attached draft image shall have an app-owned persistent copy of CraftSky's locally prepared, metadata-stripped publishable bytes plus ordered filename/MIME/dimension/aspect-ratio/alt-text metadata. The untouched original shall not be retained, and no picker URI, gallery asset, cache file, or temporary path may be authoritative. | The original asset may disappear, and retaining it would add avoidable storage and private metadata. | Prompt / Package docs / Grilling | AC-002, AC-004, AC-025 |
| FR-004 | Functional | Must | Settings shall expose a Drafts page for the active account, ordered newest-updated first, with empty/error states and rows containing post kind, useful text or project-title preview, updated date/time, and a first-image thumbnail or draft icon. Tapping edits and deletion requires confirmation; there is no count badge, search, filter, folder, bulk action, or manual sort. | Members need a predictable, deliberately small management surface similar to Scheduled posts. | Prompt / Codebase / Grilling | AC-005 |
| FR-005 | Functional | Must | Tapping a draft row shall open the correct standard or project composer and hydrate every saved field and media item in its saved order. | A draft is useful only if it can be resumed faithfully. | Prompt | AC-006 |
| FR-006 | Functional | Must | Saving changes to an existing draft shall preserve its ID and original creation time, reuse unchanged immutable media, write changed media under new immutable filenames, atomically replace the manifest, and only then delete media no longer referenced. It shall not create a duplicate row or overwrite referenced media in place. | Editing must remain simple, predictable, and interruption-safe without copying the complete bundle. | Discovery / Grilling | AC-007, AC-016, AC-017 |
| FR-007 | Functional | Must | Deleting a draft shall require confirmation and idempotently remove its manifest and all unshared media; a failed delete shall remain visible and retryable. | Private content must be controllable without accidental loss claims. | Codebase pattern / Privacy | AC-008 |
| FR-008 | Functional | Must | When a composer originated from a local draft, confirmed successful immediate publication or scheduling shall remove that local draft and its media only after the authoritative API success; unsuccessful or ambiguous submission shall not delete it. | Prevents both duplicates in management and loss on failure. | Discovery | AC-012, AC-014 |
| FR-009 | Functional | Must | Any validation, preparation, one-minute per-image upload timeout, create, scheduling, ownership, or network failure shall dismiss the submission overlay, preserve the editable composer, preserve the applicable saved draft version, and provide safe retryable feedback. | Submission latency and transfer failures must not destroy work or trap a member indefinitely. | Prompt / Codebase / Grilling | AC-014, AC-015, AC-026 |
| FR-010 | Functional | Must | Image selection shall immediately read, inspect, validate, strip metadata, resize/re-encode, and prepare publishable bytes locally. Selection, local preparation, reorder, alt-text edit, draft save, draft open, and draft discard shall make no public-blob or private-staging network request. | Gives early local errors and durable safe bytes while defining the no-optimistic-upload boundary. | Prompt / Grilling | AC-009, AC-025 |
| FR-011 | Functional | Must | Immediate submission with images shall upload the current prepared media through the existing authenticated `POST /v1/blobs/images` path with a one-minute timeout per image, then call `POST /v1/posts` with the returned ordered blob references. Successful references may be reused only in memory for unchanged images while the same composer remains open. | Reuses the existing server contract while moving transfer to explicit submission and avoiding needless same-session retries. | Codebase / Recommended direction / Grilling | AC-010, AC-026 |
| FR-012 | Functional | Must | New or edited scheduled submission with images shall upload the required current prepared bytes to the existing owner-private scheduled-media staging path only after Schedule is tapped, using its existing idempotent media identifiers and a one-minute timeout per image, then create/update the schedule; it shall not upload those images to the PDS from the composer. | Scheduled publication needs durable private bytes, not early public blobs. | Prompt / Scheduled-post architecture / Grilling | AC-011, AC-026 |
| FR-013 | Functional | Must | After fast local form validation and any missing-alt-text confirmation succeed, every immediate or scheduled submission, with or without media and including standard, quote, reply/comment, project, and existing-schedule publication flows, shall show a non-dismissible full-screen modal overlay before upload/final API work begins. It shall have no cancel action, center a spinner with `Publishing your post…` for immediate publication or `Scheduling your post…` for scheduling, and block taps, back navigation, and duplicate submission. | The requested loading state must be consistent without flashing for ordinary validation errors. | Prompt / Grilling | AC-013 |
| FR-014 | Functional | Must | Draft save/delete/open and submission operations shall capture and verify the active account lease; an account switch, sign-out, or stale completion shall never expose, mutate, publish, schedule, or report success for another account. | The installation supports multiple accounts. | Codebase | AC-018 |
| FR-015 | Functional | Must | On repository startup, the app shall ignore and safely reconcile incomplete temporary saves, orphan media, missing manifests, and unsupported/corrupt manifests without hiding healthy drafts or logging private content. | File persistence needs deterministic crash recovery. | Recommended direction | AC-015, AC-016 |
| FR-016 | Functional | Must | If private storage is unavailable or full, saving shall fail without replacing the last good manifest/media set, keep the composer open, and show an actionable safe error. A draft with missing/corrupt media shall remain visible and deletable, open with an `Image unavailable` placeholder while preserving other content and alt text, and block submission until the item is removed or replaced. | Local storage failures must not become silent data loss or hide recoverable work. | Discovery / Grilling | AC-015, AC-017 |
| FR-017 | Functional | Must | The Drafts route and providers shall follow the current Settings/root-navigator and account-bound repository patterns and shall not depend on AppView connectivity to list, open, update, or delete drafts. | Draft management is local and should feel consistent with Scheduled posts. | Prompt / Codebase | AC-002, AC-005 |
| FR-018 | Functional | Must | Closing a dirty eligible new composer shall offer Save draft, Discard, and Keep editing. Closing a dirty existing draft shall offer Save changes, Discard changes, and Keep editing; Discard changes shall leave the last saved version intact. A successful save closes the composer and shows `Draft saved`; a failed save keeps it open and intact. | Makes explicit saving discoverable without introducing autosave. | Discovery / Existing guard / Grilling | AC-003, AC-025 |
| FR-019 | Functional | Must | A saved Later choice shall reopen as Later only while its stored time remains valid under current scheduling rules; an expired or now-invalid time shall reset to Now with an explanation and shall never auto-submit or consume capacity. | Old local intent must not create an invalid schedule. | Scheduled-post behavior | AC-024 |
| FR-020 | Functional | Must | Ordinary sign-out shall retain account-owned drafts for later sign-in on the same installation. The first release shall not add a device-visible propagation path for terminal account deletion occurring elsewhere; without a local signal, those files persist until explicit local deletion, confirmed successful publication/scheduling, or app-data removal. | Preserves the local-only/no-new-API boundary without conflating ordinary sign-out with an unobservable remote lifecycle event. | Architecture / Discovery / Document review | AC-023 |
| FR-021 | Functional | Must | Save draft/changes shall remain disabled while any attached image is locally preparing or failed. Every attachment must become locally ready, be retried successfully, or be removed before the draft can save. | A saved draft cannot promise durable media if it accepts incomplete preparation. | Grilling | AC-025 |
| FR-022 | Functional | Must | After validation, submitting an edited existing draft shall start the blocking overlay lifecycle, atomically save the exact validated attempted state, and only then begin network work. Success shall delete it and failure/termination shall leave that attempted version recoverable. Submitting a never-saved composer shall not create a local recovery draft. | Protects explicit existing drafts without silently changing the no-autosave behavior for new composers. | Grilling | AC-027 |
| FR-023 | Functional | Must | Immediate upload retry shall reuse completed blob references only for unchanged prepared images while the same composer is open, upload only missing/changed images, and discard all remote references when the composer closes. Scheduled staging shall retain its existing idempotent retry behavior. | Reduces duplicate transfer without coupling the durable draft format to remote blob lifecycle. | Grilling | AC-026 |
| NFR-001 | Non-functional | Must | Draft persistence shall use versioned manifests, immutable media filenames, and atomic manifest replacement so interruption yields the prior complete reference set or the new complete reference set, never a manifest that points at a partially overwritten file. | Protects user-authored private content with a narrow atomic boundary. | Recommended direction / Grilling | AC-007, AC-016, AC-017 |
| NFR-002 | Non-functional | Must | Draft payloads, text, alt text, project fields, media bytes, file paths, account identifiers, and thumbnails shall not appear in analytics, logs, traces, metrics, crash reports, or error messages. Local files shall remain inside private application storage. | Unpublished content is sensitive. | Architecture / Privacy | AC-019 |
| NFR-003 | Non-functional | Must | The submission overlay shall expose accessible busy/status semantics, remain readable with text scaling and both themes, and not rely on spinner animation alone to convey state. | Blocking UI must remain understandable and accessible. | Prompt / UI conventions | AC-013 |
| NFR-004 | Non-functional | Should | File I/O and image preparation shall run asynchronously without synchronously blocking the Flutter UI thread; the UI shall show bounded saving/loading state when local work is not immediate. | Draft media can be large enough to cause visible jank. | Discovery | AC-020 |
| NFR-005 | Non-functional | Must | Foreground screen-awake state shall be enabled only while the submission overlay is active and shall be released on success, failure, one-minute image timeout, stale ownership, route disposal, and unexpected exceptions. | Prevents both mid-submit sleep and a leaked permanent wakelock. | Prompt / Package docs / Grilling | AC-014, AC-026 |
| RULE-001 | Business rule | Must | Local drafts shall never be written to the CraftSky AppView, PDS, scheduled-media staging, or any CraftSky cross-device service merely because they were saved or reopened. | “Locally on device” is an explicit product boundary. | Prompt | AC-002, AC-009 |
| RULE-002 | Business rule | Must | First-release draft eligibility is limited to original top-level standard posts and project posts; quote and reply/comment composers do not offer Save draft. | Matches the existing scheduled management/edit seam and bounds target-reference complexity. | Recommended direction / `ASM-001` | AC-001, AC-021 |
| RULE-003 | Business rule | Must | Draft persistence is explicit: Save draft, Save changes, or submitting an already-saved draft may write it. CraftSky shall not create or update a durable draft merely because text/media changed, and submitting a never-saved composer shall not create one. | Autosave was not requested and changes retention expectations. | Recommended direction / Grilling | AC-003, AC-027 |
| RULE-004 | Business rule | Must | Drafts may be incomplete and are validated against current composer/media/post rules only when the member attempts to publish or schedule them. | Saving work must not require completion, while publication rules remain authoritative. | Discovery | AC-001, AC-006, AC-012 |
| RULE-005 | Business rule | Must | Local drafts do not count toward the server-enforced three-scheduled-post capacity; capacity is checked only when Schedule is submitted. | A local draft is not retained server work. | Scheduled-post architecture | AC-022 |
| RULE-006 | Business rule | Must | Drafts have no automatic expiry or count limit in the first release and persist until explicit local deletion, confirmed successful publication/scheduling, or removal of app data. Remote terminal account deletion is not a local cleanup trigger because no device-visible deletion signal is in scope. | The feature should not silently delete unfinished work or invent a remote lifecycle contract. | Recommended direction / `ASM-004` / Document review | AC-008, AC-023 |
| RULE-007 | Business rule | Must | Draft ordering uses `updatedAt` descending, with stable ID as a deterministic tie-breaker; opening a draft does not change `updatedAt` unless the member saves. | Keeps management order predictable. | Discovery | AC-005, AC-007 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-001, RULE-002, RULE-004 | Given a new top-level standard or project composer with deliberate text, media/alt-text, language, project-field, or scheduling changes, when the member chooses Save draft, then one local draft can save without publication validation or network access. Given only untouched defaults, Save draft is unavailable. |
| AC-002 | BR-002, FR-003, FR-017, RULE-001 | Given a valid signed-in account and selected images whose original picker assets are subsequently removed or denied, when the member saves, restarts the app offline, and reopens the draft, then the draft and prepared previews load from private app-owned storage, no untouched original is required, and no AppView/PDS request occurs. |
| AC-003 | FR-018, RULE-003 | Given a dirty eligible new composer, closing offers Save draft, Discard, and Keep editing; given a dirty existing draft, closing offers Save changes, Discard changes, and Keep editing, and Discard changes preserves the prior saved version. No durable write occurs merely from editing. |
| AC-004 | BR-001, FR-002, FR-003 | Given a saved standard or project draft with text, languages, project fields, When state, ordered prepared images, and alt text, when it is reopened, then every saved value and publishable image is restored in the same editable order without reading or retaining the untouched source asset. |
| AC-005 | FR-004, FR-017, RULE-007 | Given multiple active-account drafts, when Settings > Drafts opens offline, then rows are newest-updated first with deterministic ties and show first thumbnail/draft icon, kind, preview, and last-saved date/time; tapping edits and confirmed delete removes; another account's drafts are absent; empty/error/retry states exist; and there is no badge, search, filter, folder, bulk action, or manual sort. |
| AC-006 | BR-001, FR-002, FR-005, RULE-004 | Given a draft row, when it is tapped, then the matching standard or project composer opens with its saved ID and may still contain incomplete fields without being rejected until submission. |
| AC-007 | FR-006, NFR-001, RULE-007 | Given an existing draft, when the member changes and saves it, then the same ID/creation time and unchanged immutable media remain, changed media uses new immutable files, the manifest switches atomically, `updatedAt` advances, the row moves accordingly, and only then are unreferenced old files cleaned. |
| AC-008 | FR-007, RULE-006 | Given a saved draft, when deletion is confirmed, then its row, manifest, thumbnail, and all unshared media are removed; repeating delete is harmless, while a failed delete does not claim success and remains retryable. |
| AC-009 | BR-003, FR-010, RULE-001 | Given selected supported media, when local inspection, metadata stripping, resize/re-encode, preparation, reorder, alt-text editing, save, close, reopen, or discard runs, then prepared publishable bytes and early local errors are available while recorded HTTP traffic contains neither `POST /v1/blobs/images` nor scheduled-media staging requests. |
| AC-010 | BR-003, FR-011 | Given a valid immediate standard, quote, or project composer with locally prepared images, when Post is tapped, then only after the tap the current ordered prepared media is revalidated and uploaded through `POST /v1/blobs/images`, and `POST /v1/posts` receives the returned ordered references. |
| AC-011 | BR-003, FR-012 | Given a valid scheduled composer with images, when Schedule is tapped, then only after the tap the current bytes are uploaded to private scheduled staging and the schedule is created/updated; the composer performs no PDS blob upload and no schedule is accepted until staging succeeds. |
| AC-012 | FR-008, RULE-004 | Given a composer opened from a local draft, when immediate publication or scheduling returns confirmed success, then the post/schedule succeeds once and the local draft disappears; when validation or the API fails, the draft remains. |
| AC-013 | BR-004, FR-013, NFR-003 | Given any immediate or scheduled submission, fast local validation and missing-alt confirmation occur before the overlay. Once confirmed valid, a full-screen non-dismissible overlay with no cancel action appears before upload/final API work, blocks interaction/back navigation, exposes accessible busy text, and shows a centered spinner with exactly `Publishing your post…` or `Scheduling your post…`. |
| AC-014 | BR-004, FR-008, FR-009, NFR-005 | Given submission success, transfer/API failure, one-minute image timeout, account-stale completion, thrown exception, or overlay disposal, duplicate interaction remains blocked while active, the overlay exits at the terminal result, and screen-awake state is disabled; failure leaves editable state and the applicable saved draft intact. |
| AC-015 | FR-009, FR-015, FR-016 | Given a missing/corrupt manifest or media file, when Drafts loads or the item opens, then healthy drafts remain available; recoverable text/project/order/alt-text data remains; missing media shows `Image unavailable` and can be removed/replaced; the item remains deletable; submission cannot reference missing bytes; and no private content or path is logged. |
| AC-016 | FR-006, FR-015, NFR-001 | Given the process terminates before or after each new-media write, manifest replacement, or old-media cleanup boundary, when the app restarts, then the manifest points only to complete immutable media, the prior or new draft remains usable, and orphan artifacts are cleaned without harming referenced files. |
| AC-017 | FR-016, NFR-001 | Given storage becomes full or unavailable while a new or existing draft saves, then the operation reports failure, the composer stays open, the prior saved version remains complete, and no partial replacement is listed. |
| AC-018 | FR-014 | Given save, open, delete, or submit is in flight when the active account changes, then stale completion neither mutates the new account's state nor publishes/schedules as the new account, and no success message is shown for an obsolete lease. |
| AC-019 | BR-002, NFR-002 | Given canary draft text, project fields, alt text, account data, file paths, and image bytes, when save/list/open/delete/recovery/submission paths run and diagnostics are inspected, then none of those canaries appear outside private local storage or in telemetry/error output. |
| AC-020 | NFR-004 | Given the maximum supported draft media set, when it is saved or opened, then file/image work is asynchronous, visible saving/loading state is presented where needed, and the UI remains able to render progress rather than synchronously freezing for the whole operation. |
| AC-021 | RULE-002 | Given a quote or reply/comment composer, when it becomes dirty, then it does not offer Save draft or create a local draft, while its existing Post/Reply submission still uses the common blocking overlay and submit-time media boundary where applicable. |
| AC-022 | RULE-005 | Given any number of local drafts and an account below scheduled capacity, when a draft is saved, then the scheduled count is unchanged; when the member later taps Schedule, the existing server capacity rule is evaluated at that time. |
| AC-023 | FR-020, RULE-006 | Given saved account-owned drafts, ordinary sign-out and later sign-in on the same installation preserve them; explicit local draft deletion, confirmed successful publication/scheduling, or app-data removal removes them; time passage alone does not. The first release neither polls nor receives a server signal to purge local files after terminal account deletion elsewhere. |
| AC-024 | FR-002, FR-019 | Given a saved Later time, reopening preserves it when still valid; if it is in the past or invalid under current rules, reopening selects Now, explains that the saved time can no longer be used, does not consume capacity, and performs no submission. |
| AC-025 | FR-003, FR-010, FR-018, FR-021 | Given attached images still preparing, Save draft/changes is unavailable and local progress is visible; given an image failure, retry/remove is required; given all images locally ready, save can proceed; success closes the composer and shows `Draft saved`; failure leaves it open and intact. |
| AC-026 | FR-009, FR-011, FR-012, FR-023 | Given an immediate image upload succeeds for some unchanged images and another fails or exceeds one minute, the overlay exits and retry reuses same-session successful references while uploading only missing/changed images. Closing loses those references. Scheduled staging retries through its existing idempotent media IDs. |
| AC-027 | FR-022, RULE-003 | Given an edited existing draft, tapping Post or Schedule validates, starts the blocking overlay lifecycle, atomically saves the exact attempted state, and only then begins network work; success deletes it and failure/termination retains it. Given a never-saved composer, tapping Post or Schedule does not create a recovery draft. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | The original gallery/file-picker asset is deleted, moved, permission-revoked, or cloud-evicted after save. | Reopen and submission use the app-owned copy without requesting the original asset. | FR-003 |
| EC-002 | The member saves an otherwise empty project with only one partial field or an image. | The meaningful incomplete state can be saved; publication validation waits until submission. | FR-001, RULE-004 |
| EC-003 | Save is tapped twice or the close prompt races the app-bar action. | One operation owns the stable draft ID; duplicate completion does not create a second draft. | FR-006, NFR-001 |
| EC-004 | Process termination occurs while writing changed immutable media, replacing the manifest, or cleaning old media. | Startup follows the last complete manifest, never a partially overwritten file, and reconciles temporary/orphan files. | FR-006, FR-015, NFR-001 |
| EC-005 | Device storage fills during a media copy. | Save fails safely, current composer remains available, prior version is untouched, and partial bytes are cleaned later. | FR-016, NFR-001 |
| EC-006 | A draft manifest is newer than the app-supported schema. | The draft is not destructively rewritten; it exposes a safe unsupported-version error and remains deletable. | FR-015 |
| EC-007 | One media file is corrupt or missing. | The draft stays visible, opens with `Image unavailable`, preserves other fields/order/alt text, cannot submit with that item, and lets the member remove/replace it or delete the draft. | FR-016 |
| EC-008 | Active account changes during local save or network submission. | The operation is fenced to its captured lease and cannot affect the new account. | FR-014 |
| EC-009 | Immediate upload succeeds for some images but a later upload or post create fails. | Overlay exits, composer/draft remains, same-session retry reuses successful unchanged references and uploads only missing/changed images; closing discards remote references; unreferenced PDS blobs are left to normal PDS cleanup. | FR-009, FR-011, FR-023, NG-007 |
| EC-010 | Private staging succeeds but schedule create/update fails. | No successful schedule is claimed, the local draft/composer remains, and existing scheduled orphan cleanup handles unclaimed staged objects. | FR-009, FR-012 |
| EC-011 | The selected scheduled time passes while a local draft is closed. | Reopen resets to Now with an explanation and never schedules implicitly. | FR-019 |
| EC-012 | A saved draft is opened while offline. | List/open/edit/delete/local save work; immediate/scheduled submit fails safely at the network boundary. | FR-017, FR-009 |
| EC-013 | The app backgrounds during the overlay. | No background-completion guarantee is made; on return, the live operation resolves normally if retained, and every terminal/dispose path releases screen-awake state. | NFR-005, NG-008 |
| EC-014 | External OS behavior releases the wakelock while the overlay is active. | The visible submission owner may reassert its desired state when foreground-active; it never leaves the wakelock enabled after the overlay ends. | NFR-005 |
| EC-015 | The member signs out and another local account opens Drafts. | The signed-out account's files remain private and absent; only signing back into that owner exposes them. | FR-014, FR-020 |
| EC-016 | The member opens a draft without editing it and leaves. | `updatedAt` and ordering remain unchanged. | RULE-007 |
| EC-017 | One image upload runs longer than one minute. | The request times out, the overlay exits, screen-awake state is released, and the current composer/draft remains retryable. Other images do not share one overall timeout budget. | FR-009, FR-011, NFR-005 |
| EC-018 | An edited existing draft is submitted and the app terminates during upload. | The exact validated attempted version was saved before network work and is recoverable on restart. | FR-022 |
| EC-019 | A never-saved composer is submitted and the app terminates during upload. | No durable recovery draft is created; this loss boundary is consistent with explicit saving. | FR-022, RULE-003 |

## 15. Data / Persistence Impact

- New local representation:
  - A private, versioned draft repository under a persistent application-support/documents-equivalent directory, never a temporary/cache directory.
  - One account namespace and one opaque stable draft ID per bundle.
  - A versioned manifest containing draft kind, timestamps, text/languages, standard/project payload, selected When state, optional future UTC schedule instant plus display context as needed, and ordered media metadata.
  - App-owned immutable media files containing CraftSky's prepared, metadata-stripped, publishable image bytes. The untouched original is not retained. File paths are internal implementation details and must not enter logs or analytics.
  - New/changed media is written under new immutable filenames; unchanged files are reused; the manifest is atomically replaced; only then are files no longer referenced cleaned. A full duplicate bundle is not required.
- New fields:
  - `schemaVersion`, `id`, `owner`, `kind`, `createdAt`, `updatedAt`, composer payload, optional scheduling intent, and ordered media descriptors.
  - Media descriptors include opaque media ID, relative file reference, filename, MIME type, byte size/checksum where useful for validation, width/height or aspect ratio, order, and alt text.
  - Immediate PDS blob references are transient submission state only and never appear in the manifest.
- Changed fields:
  - Composer image state changes from public-upload phases to local-ready and submit-time-transfer phases. Exact class names belong in coding planning.
- Migration required:
  - No server or existing-user data migration.
  - The first local format starts at version 1 and must reject or migrate future versions explicitly rather than silently decoding incompatible data.
- Backwards compatibility:
  - Existing `/v1/posts`, `/v1/blobs/images`, scheduled-post, and private scheduled-media contracts remain valid.
  - Existing server-scheduled posts continue to edit/publish independently of local drafts.
  - There are no production users, but scheduled-post regression behavior must still be preserved.
- Storage recommendation:
  - Add a runtime persistent-directory abstraction (likely `path_provider`) during implementation.
  - Do not promote the existing test-only `sqflite` dependency to runtime for this release.
  - Do not use SharedPreferences as the draft collection or media store.

## 16. UI / API / CLI Impact

- UI:
  - Add a localized Drafts Settings row and `/profile/settings/drafts` management route on the root navigator.
  - Add account-scoped newest-updated draft rows, first-image thumbnails, empty/error/retry states, edit-on-tap, and confirmed deletion.
  - Add explicit Save draft / Save changes actions and three-choice close handling for eligible composers.
  - Disable saving until all attached media is locally ready; show local preparation progress and retry/remove failures.
  - Successful save closes the composer and shows `Draft saved`; failed save leaves the composer intact.
  - Add one common, full-screen, non-dismissible submission overlay with a spinner and exact operation text.
  - Preserve the existing composer contents and return focus after failures.
- API:
  - No new route or request/response shape is required.
  - Defer `POST /v1/blobs/images` until immediate submission.
  - Apply a one-minute timeout to each image upload and retain successful unchanged blob references only for same-composer in-memory retry.
  - Defer private scheduled-media staging until scheduled submission.
  - Preserve server-side validation, ownership, idempotency, error envelopes, and scheduled capacity.
- CLI:
  - None.
- Background jobs:
  - No new job. Existing scheduled publication and staged-media cleanup continue unchanged.
- Native/platform:
  - Add a scoped foreground screen-awake integration (likely `wakelock_plus`) controlled only by the overlay lifecycle.
  - Draft persistence targets iOS and Android for this release.

## 17. Security / Privacy / Permissions

- Authentication:
  - Local list/open/save/delete requires an active signed-in account and captured account lease, but no network call.
  - Immediate and scheduled submission continue through existing authenticated AppView clients.
- Authorization:
  - Account namespaces and provider arguments must prevent one local account from listing or mutating another account's drafts.
  - Account identity is captured when a composer opens; stale work cannot be rebound to a newly active account.
- Sensitive data:
  - Draft text, project details, languages, alt text, future schedule intent, and media are unpublished private content.
  - Store them only in the app sandbox; never in SharedPreferences, cache directories, public media storage, diagnostics, screenshots generated by tests, or remote telemetry.
  - Platform OS backup/restore behavior follows the device's app-data policy and is not CraftSky cross-device sync; this release does not promise recovery after uninstall, app-data clearing, or device loss.
  - Custom encryption and biometric/passcode gating are not required; normal device/app sandbox and platform data protection are the boundary for this release.
- Abuse cases:
  - Filenames and manifests are untrusted local input after restoration/corruption; reject traversal, absolute paths, duplicate media IDs, oversized counts, and files outside the draft root.
  - Enforce current media type/count/size validation again at submission, even for a previously valid saved draft.
  - A guessed draft ID must not cross account namespaces or disclose another account's content.

## 18. Observability

- Events:
  - Optional coarse events: `draft_saved`, `draft_opened`, `draft_deleted`, `submission_started`, `submission_succeeded`, `submission_failed`.
  - Allowed properties are coarse kind (`standard`/`project`/other existing submit kind), new-versus-update, media count bucket, target (`immediate`/`scheduled`), safe failure phase/class, and duration bucket.
  - Do not record draft IDs, account identifiers, text, fields, exact schedule times, media identifiers, paths, filenames, MIME-derived source details, or alt text.
- Logs:
  - Safe lifecycle/error classes only, with all private payloads and paths redacted.
- Metrics:
  - Local save/open/delete success/failure count and duration; submission preparation/upload/create/schedule duration and safe failure class; orphan-reconciliation counts without identifiers.
- Alerts:
  - None required for the local feature initially. Existing AppView upload/scheduled alerts remain unchanged.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | A file save is interrupted between changed-media writes, manifest replacement, and old-media cleanup. | Draft corruption, stale files, or loss of the prior version. | Reuse unchanged immutable media, write changed media under new names, atomically replace only the manifest, clean unreferenced files afterward, and reconcile orphans on startup. |
| RISK-002 | Draft media uses cache paths or source picker references. | Media disappears before reopen/submission. | Require app-owned persistent copies and test source removal plus restart/offline reopen. |
| RISK-003 | File-backed listing grows beyond its intended small collection. | Slower management-page startup. | Read small bounded manifests asynchronously; retain a repository seam and migrate to SQLite only when measured volume/query needs justify it. |
| RISK-004 | Eager upload remains in one composer or retry path. | Private media reaches the PDS before intent, and scheduled media is transferred twice. | Centralize the upload boundary and add network-spy regression coverage across standard, quote, reply, project, draft, and scheduled edit paths. |
| RISK-005 | Submit-time transfer makes perceived publishing slower or stalls. | Members may think the app is stuck or become trapped behind a non-dismissible overlay. | Use the immediate full-screen blocking overlay, clear text, screen-awake state, a one-minute timeout per image, safe retryable errors, and submission metrics; richer progress is future work. |
| RISK-006 | Wakelock is not released on an exception or route teardown. | The device remains awake and drains battery. | Tie ownership to overlay lifecycle and test all terminal/dispose/stale branches with an injected screen-awake service. |
| RISK-007 | A draft is deleted before the server confirms publication/scheduling. | User-authored private content is lost on ambiguous failure. | Delete only after authoritative success; preserve the last saved version on all other outcomes. |
| RISK-008 | Account switching races a local or remote operation. | Draft disclosure or publication under the wrong identity. | Capture the account lease, key repositories by owner, and discard stale completions without success UI. |
| RISK-009 | Local draft content leaks through mapper `toString`, errors, file paths, thumbnails, or Sentry breadcrumbs. | Exposure of unpublished text/media/account data. | Redacted models, safe error types, secret-scan/telemetry tests, and no raw payload/path logging. |
| RISK-010 | Removing eager upload regresses current scheduled-post hydration/materialization. | Existing schedules cannot edit, reschedule, or publish now. | Treat current scheduled tests as regression gates and design the shared materializer around local-ready and server-private-ready media sources. |
| RISK-011 | No automatic expiry allows local storage growth. | Device storage pressure. | Show accurate save failures, delete media with drafts, avoid an artificial limit now, and use measured usage to justify future storage management. |
| RISK-012 | Stored schedule time or media rules become invalid before reopen/submit. | Invalid scheduling or confusing rejection. | Reset invalid past schedule intent to Now with explanation and revalidate all content/media at submission without mutating authored content silently. |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | Confirmed during grilling: first-release drafts cover top-level standard and project posts only; quotes and replies/comments remain in-memory only. | Supporting target-bound drafts would require target snapshot/reference persistence, recovery behavior, additional rows, and acceptance coverage. |
| ASM-002 | Confirmed during grilling: draft saving is explicit, with a close prompt, rather than automatic. | Autosave would require debounce, lifecycle flush, conflict/write-amplification rules, and different user expectations about retention. |
| ASM-003 | The first release targets the current iOS and Android product clients; Web and desktop persistence are deferred. | A cross-platform storage backend and screen-awake adapter would be required before those clients can claim feature parity. |
| ASM-004 | Confirmed during grilling: there is no artificial draft count or automatic expiry in the first release. | Product limits, capacity copy, and cleanup rules would need requirements before implementation. |
| ASM-005 | Confirmed during grilling: app sandbox plus platform data protection and normal private OS backup behavior are sufficient; custom encryption, biometric gating, and backup exclusion are not required. | Native encrypted storage/key management, access gating, or platform backup-exclusion work would materially expand the design. |
| ASM-006 | A typical account holds a small enough collection that asynchronously reading compact manifests is acceptable. | Measured scale could justify SQLite metadata while retaining media files and the repository interface. |
| ASM-007 | Existing immediate blob/post and private scheduled-media/schedule endpoints remain the submission contracts. | A server API change would require the AppView API spec, route-policy, error-contract, and integration work to enter scope. |
| ASM-008 | Confirmed during grilling: the existing prepared publishable output is the app-owned durable representation; the original unmodified asset is not preserved. | If editing/cropping or original-quality reprocessing is required, the store may need original bytes and a larger privacy/storage budget. |

## 21. Open Questions

None. The grilling review resolved draft eligibility, explicit-save behavior, storage format, prepared-media ownership, upload orchestration/timing/retry/timeout, overlay lifecycle/copy, meaningful-state detection, management UX, retention, sign-out behavior, OS backups, scheduling intent, success cleanup, atomic local updates, damaged-data recovery, encryption, media preparation readiness, save completion, and submission recovery boundaries. Document review additionally confirmed that remote terminal account deletion has no device-visible propagation or local purge guarantee in the first release, and that device-local draft storage is a deliberate exception to the repository's broader AppView/Postgres draft guidance.

## 22. Review Status

Status: Reviewed

Risk level: Medium

Review recommended: No (completed through grilling)

Reviewer: Product owner

Date: 2026-08-03

Notes: The 2026-08-03 grilling review resolved every product/design branch listed in Open Questions and the additional persistence, retry, timeout, recovery, and submission boundaries. The subsequent document-review revision confirmed remote terminal-deletion propagation is out of scope and device-local storage is a deliberate architecture exception. No new AppView storage, API route, database migration, or lexicon change is proposed. The requirements are ready for acceptance-test design and document re-review.

## 23. Handoff To Test Design

- Requirements file: `01-requirements.md`
- Next test specification: `02-acceptance-tests.md`
- Must-cover requirement IDs:
  - `BR-001`–`BR-004`
  - `FR-001`–`FR-023`
  - `NFR-001`–`NFR-003`, `NFR-005`
  - `RULE-001`–`RULE-007`
- Suggested test levels:
  - Unit tests for manifest encoding/versioning, owner/path validation, deterministic ordering, meaningful-draft detection including untouched defaults, schedule-time restoration, storage error mapping, one-minute per-image timeout, transient reference reuse/invalidation, screen-awake lifecycle, and submission target selection.
  - File-repository integration tests with temporary directories for new save, immutable changed-media writes, unchanged-media reuse, atomic manifest replacement, post-switch cleanup, idempotent delete, interruption at every boundary, corrupt/newer manifests, `Image unavailable` recovery, orphan cleanup, source removal, account isolation, and storage failure.
  - Provider/coordinator tests for account leases, stale completion, local-ready media, immediate upload ordering, one-minute timeout, same-composer partial retry, private scheduled staging/idempotency, edited-draft pre-submit save, never-saved no-recovery behavior, and originating-draft deletion only after authoritative success.
  - Widget tests for Settings entry, account-scoped list/empty/error states, thumbnails, edit/delete, standard/project hydration, incomplete saves, close choices, exact overlay copy, full-screen modal blocking, accessibility, and failure recovery.
  - Network-spy regression tests proving no blob/staging request before submit across selection, draft save/open/discard, and all current composer kinds.
  - Existing standard/project/scheduled composer regression suites, full `just app-test`, `just app-analyze`, generated-code checks, and `git diff --check`.
  - Manual iOS and Android checks for app restart/source-photo removal, offline draft management, maximum-media save/open, text scaling/theme rendering, actual screen sleep prevention during a throttled submission, and release of screen-awake behavior afterward.
- Blocking open questions: None. Requirements review is complete.
