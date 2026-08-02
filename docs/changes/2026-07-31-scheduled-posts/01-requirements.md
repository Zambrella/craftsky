# Requirements: Scheduled Posts

## 1. Initial Request

Let a signed-in Craftsky member schedule up to three posts at a time. Scheduling is available for original top-level standard posts and project posts, but not quote posts, comments, or replies. The composer defaults to posting now and offers scheduling as an explicit alternative. A management screen, initially reachable from Settings, lets the member view, edit, and delete scheduled posts.

The backend approach was not predetermined. This document compares practical options and recommends an AppView-owned durable schedule queue because unpublished content is private, publication requires a later authenticated PDS write, and the app cannot be assumed to be open at the scheduled time.

## 2. Current Codebase Findings

- Relevant files:
  - `app/lib/feed/widgets/post_composer_sheet.dart` owns the standard, reply, and quote composer. A non-reply composer can attach images; replies cannot.
  - `app/lib/projects/widgets/project_composer_sheet.dart` owns the three-page project composer and requires at least one image.
  - `app/lib/feed/providers/create_post_provider.dart`, `app/lib/feed/data/post_repository.dart`, and `app/lib/feed/data/post_api_client.dart` form the current Flutter create-post path.
  - `app/lib/feed/providers/composer_images_provider.dart` currently prepares and eagerly uploads each selected image through `POST /v1/blobs/images` before the post is submitted.
  - `appview/internal/api/blob.go` immediately forwards uploaded image bytes to the member's PDS.
  - `appview/internal/api/post_request.go` validates the post payload. Project posts must be top-level and cannot quote; standard quote posts are top-level.
  - `appview/internal/api/post.go` serves `POST /v1/posts`, stamps `createdAt` on the server, writes the record directly to the member's PDS, and returns a synthetic post while Tap catches up.
  - `app/lib/settings/pages/settings_page.dart` and `app/lib/router/router.dart` provide the existing Settings list and root-navigator child-page pattern.
  - `appview/internal/auth/background_session_selector.go` already selects an owner-scoped OAuth session for server-initiated PDS writes.
  - `appview/internal/instagram/automatic_follow_worker.go` and `appview/internal/push/dispatcher.go` provide existing durable worker patterns: background-session selection, bounded retries, claim leases, and `FOR UPDATE SKIP LOCKED`.
  - `appview/cmd/appview/main.go` currently starts long-running workers as goroutines alongside the HTTP server.
- Existing patterns:
  - Writes go through the PDS; reads come from the AppView.
  - Private-by-intent data belongs in AppView storage, not in a public PDS record.
  - The Flutter app holds only a Craftsky session token. OAuth sessions and PDS write capability remain server-side.
  - Every `/v1/*` request uses camelCase JSON, authenticated ownership, the standard error envelope, and the route-policy registry.
  - Server-initiated PDS writes can use a currently usable owner session without binding the operation to one device.
- Current behavior:
  - Composer submission always means publish now.
  - `POST /v1/posts` accepts standard posts, replies/comments, quote posts, and project posts, then writes immediately to the PDS.
  - A newly created top-level post is optimistically inserted into Flutter timeline/profile caches.
  - Selected images are uploaded to the public PDS before composer submission.
  - There is no scheduled-post persistence, management API/UI, due-job worker, private media staging service, or scheduled-publication status.
- Constraints discovered:
  - Client-side timers are not reliable because the app may be backgrounded, terminated, offline, or signed into a different active account.
  - Scheduled text, project details, facets, alt text, and the durable media copy are unpublished private content and must be held by the AppView until the due publication attempt.
  - The existing eager image-upload path may remain: the official `com.atproto.repo.uploadBlob` contract says an uploaded blob is deleted when it remains unreferenced for a short time window. When the member chooses Schedule, Flutter must still reprocess its retained local bytes and upload a durable private copy to the AppView; the initial unreferenced PDS blob is left to normal PDS cleanup.
  - Publication is an external side effect. A worker can crash after a successful PDS write but before marking the schedule complete, so stable record identity and recovery are required to prevent duplicate public posts.
  - A member's available OAuth session may expire or be removed between scheduling and publication.
  - The existing media limit is up to four images and the configured upload ceiling may be as high as 15 MB per prepared image; storing all staged image bytes directly in frequently queried Postgres rows would create avoidable database and backup pressure.
- Test/build commands discovered:
  - Repository aggregate gate: `just test`.
  - AppView focused/full suites from `appview/`: `go test ./...` when the test database is available.
  - Flutter tests from `app/`: `flutter test`.
  - Flutter static analysis from `app/`: `dart analyze`.

## 3. Clarifying Questions And Decisions

### Q1: Which post types can be scheduled?

Answer: Original top-level standard posts and project posts. Quote posts, comments, and replies are excluded from the first release.

Decision / implication: The Flutter control and AppView API reject any scheduled payload containing `reply` or a quote embed. Project posts retain their existing standalone validation.

### Q2: How many posts can a member schedule?

Answer: Up to three retained unpublished schedules per account.

Decision / implication: Scheduled, Publishing, Retrying, and Needs attention items count transactionally toward the per-DID limit. Published and deleted items do not. At capacity, Schedule remains visible but disabled with `3 of 3 scheduled` and a management link; Post now remains available.

### Q3: What times may be selected?

Answer: Any whole-minute instant from five minutes through 28 days after server acceptance.

Decision / implication: The composer exposes a `When` row defaulting to `Now`. Schedule for later opens a whole-minute picker. The API stores an absolute UTC instant; travel, daylight-saving changes, or device-timezone changes do not move it. The chosen timezone/offset is displayed clearly.

### Q4: How are scheduled posts managed and edited?

Answer: Settings gains Scheduled posts. The screen shows only unpublished items and exposes Edit and Delete; tapping a row opens the full composer.

Decision / implication: Rows show an authenticated first-image thumbnail when present, post/project type, project title when present, bounded text preview, localized scheduled time/timezone, and status. Edit includes every field in the original eligible composer. A future item preserves its scheduled choice/time; Needs attention opens with Post now selected and shows the missed time. Member-to-member concurrent saves are last-write-wins, while internal worker version fencing remains mandatory.

### Q5: How does image staging interact with the current eager PDS upload?

Answer: Preserve eager PDS upload. If the member later chooses Schedule, reprocess retained local bytes and upload them to AppView-private staging during submission.

Decision / implication: The initial PDS blob remains unreferenced and is left to the PDS's documented short-window cleanup. The composer stays open with progress until private staging and schedule creation both succeed; failure preserves the complete composer for retry. At publication, the worker uploads the private copy to the PDS and references that blob in the record.

### Q6: What storage protects staged media?

Answer: An S3-compatible private object-storage interface, with MinIO for local development and a managed S3-compatible bucket in production.

Decision / implication: Use TLS, private access controls, least-privilege credentials, and the platform's standard encryption-at-rest protection. Custom application-level encryption is not required. Unclaimed staging expires after 24 hours; replaced/deleted media is cleaned promptly; successful publication triggers immediate asynchronous cleanup; Needs attention content expires after 30 days.

### Q7: What backend execution model should be used?

Answer: A durable Postgres-backed queue processed by an in-process Go worker for the first release.

Decision / implication: Queue/store/worker interfaces remain independent of HTTP so the worker can move to a separate Compose service later. Claims use leases and concurrency-safe database semantics.

### Q8: What are the publication and retry semantics?

Answer: Never publish early. Under healthy conditions, begin within 60 seconds after `scheduledAt`. Retry transient failures only through 30 minutes late, with attempts at approximately due time and +1, +3, +7, +15, and +30 minutes; bounded jitter must not extend that window.

Decision / implication: At the first attempt, atomically enter Publishing, allocate/persist the stable PDS TID, and freeze `createdAt` at that attempt time. Reuse the identical record identity/body for retry and crash recovery. After the final failed attempt, or immediately for a permanent validation/authorization failure, enter Needs attention without altering content.

### Q9: How does a member recover or lose retained content?

Answer: Needs attention exposes only Edit and Delete on the management screen. Edit lets the member Post now or choose a new schedule; no automatic resume occurs after reauthentication.

Decision / implication: Needs attention continues to occupy a slot and is deleted with its private media after 30 days; the expiry date is visible. Successfully published items disappear from management, their private content/media is removed, and a content-free idempotency tombstone remains for 30 days.

### Q10: What happens around sign-out and account deletion?

Answer: Signing out one device does not affect publication while another active Craftsky session remains. Signing out the last active session prevents automatic publication but does not delete the schedule. Account deletion removes schedules and staged media.

Decision / implication: A due item without an active usable session follows the 30-minute retry window, then enters Needs attention. It never writes anonymously or through another account.

### Q11: What happens during publication and concurrent actions?

Answer: Edit/Delete remain available until the worker transitions the item to Publishing at or after the scheduled time and before any external PDS write. Publishing locks both actions.

Decision / implication: The worker and mutation paths require internal version/cancellation fencing. The management screen does not poll; it refreshes on entry and through user-initiated pull-to-refresh.

### Q12: Does this release notify members proactively?

Answer: No push or in-app notification is added for success or Needs attention.

Decision / implication: Settings shows a Needs attention count on the Scheduled posts tile. The management screen shows Scheduled, Publishing, Retrying, and Needs attention. Published items appear only through normal feed/profile surfaces.

## 4. Candidate Approaches

### Option A: Durable AppView queue with an in-process Go worker and private media staging

Summary: Persist each schedule and its state in Postgres, re-upload scheduled media to S3-compatible private storage at submit time, and poll/claim due rows from a Go worker running alongside the AppView HTTP server. At the due time, select a usable owner OAuth session, upload staged media to the PDS, allocate stable record identity on the first attempt, and write the post idempotently.

Pros:

- Fits the existing private-data and server-side PDS-token boundaries.
- Reuses proven Craftsky worker concepts: leases, `SKIP LOCKED`, background-session selection, retries, and observability.
- Works when the Flutter app is closed or offline.
- Keeps scheduling state authoritative in one place for create, edit, delete, capacity enforcement, and recovery.
- Has low initial deployment complexity.
- Can be extracted to a separate worker later if the AppView scales horizontally or HTTP and worker failure domains need separation.

Cons:

- AppView instances must be configured so worker concurrency is safe; the database lease model is load-bearing.
- Worker availability is initially tied to AppView process availability.
- S3-compatible storage, local MinIO, authenticated preview, and cleanup behavior are new infrastructure.
- Scheduling an image duplicates transfer once: the current eager PDS upload remains, then Flutter sends a private staging copy if Schedule is selected.

Risks:

- A process crash around the PDS write can duplicate posts unless stable record keys and write reconciliation are designed correctly.
- Incorrect lease/version checks can let an edit or cancellation race with publication.

### Option B: Durable AppView queue with a separate Go worker service

Summary: Use the same Postgres schedule state, private media staging, and Go publication logic as Option A, but deploy the worker as a separate command/container from the outset.

Pros:

- Separates HTTP availability and scheduled-publication failure domains.
- Allows worker scaling, resource limits, and deployment cadence to differ from the API.
- Avoids every AppView HTTP replica running a polling loop, although leases would still protect correctness.

Cons:

- Adds a deployable service, health checks, configuration, and operational ownership before current scale requires them.
- Local Compose and production deployment become more complex.
- Does not remove the need for durable AppView state, media staging, or idempotency.

Risks:

- Configuration drift between API and worker can break validation, media limits, or credentials.
- A worker deployment omission can leave schedules overdue even while the API appears healthy.

### Option C: Provider-managed per-post tasks

Summary: Persist authoritative schedule state in Postgres, then create one delayed task per schedule in infrastructure such as Cloud Tasks. The task invokes or runs Go publication logic at the due time.

Pros:

- Avoids frequent database polling.
- Can provide managed delivery, retry, and horizontal execution.
- Naturally separates delayed execution from the HTTP server lifecycle.

Cons:

- Edit/delete must transactionally reconcile database state with an external task identifier, and stale tasks must still be harmless.
- The three-post limit, private payload, media lifecycle, ownership, and idempotency still require AppView storage.
- Couples scheduling behavior and local development to a deployment provider.
- Very long delays, task-retention limits, pricing, and emulator support vary by provider.

Risks:

- Database/task partial failure can create missing or duplicate wakeups unless an additional outbox/reconciler is introduced.
- A task can arrive after edit/delete, so version and cancellation checks remain mandatory.

### Option D: Client timers or immediate PDS staging

Summary: Store schedules on-device and publish from Flutter when a local timer fires, or immediately upload draft payload/media to the PDS and defer only creation of the final record.

Pros:

- Appears smaller on the backend.
- Reuses parts of the current composer path.

Cons:

- Fails when the app is closed, offline, background-restricted, signed out, or switched to another account.
- Cannot enforce a cross-device limit of three.
- Cannot offer authoritative cross-device management.
- PDS-only staging cannot durably hold private text/project data or provide authoritative cross-device management. Although unreferenced PDS blobs are documented to expire, the AppView still needs a private durable media copy for scheduled publication.

Risks:

- Missed or duplicate publication.
- Missing durable private content or reliance on an unreferenced blob that the PDS is expected to delete.
- Device clock and timezone inconsistencies.

## 5. Recommended Direction

Recommended approach: Option A — a durable AppView schedule queue with private media staging and an in-process Go worker, deliberately structured so the worker can later become Option B without changing product/API behavior.

Why:

- Scheduling a public PDS write is domain logic, not database-only housekeeping; Go should own validation, OAuth-session selection, media upload, PDS calls, retries, and reconciliation.
- Postgres is the correct authority for the three-item limit, ownership, editable state, cancellation, due-time ordering, leases, and recovery.
- A database-only scheduler such as `pg_cron` is not appropriate for authenticated PDS and object-storage calls.
- An infrastructure scheduler may eventually be useful as a wakeup mechanism, but it would not eliminate the durable queue and introduces a second consistency boundary today.
- The existing AppView already runs durable polling workers. Starting in-process minimizes operational change while lease-based claims preserve a clean extraction path.
- S3-compatible private object storage gives the worker a durable media source without putting potentially large image bytes in normal Postgres rows/backups or relying on the initial unreferenced PDS upload. MinIO keeps local Compose development provider-compatible.
- Platform-managed encryption at rest is sufficient; custom application-level encryption is intentionally not required.

## 6. Problem / Opportunity

Craftsky members cannot currently prepare a post for later publication. They must be available at the intended time and manually submit from the app. A reliable server-side scheduler lets a member prepare up to three future standard or project posts, manage them from any signed-in device, and have Craftsky publish them through the normal PDS ownership boundary even when the app is not running.

## 7. Goals

- G-001: Let members schedule eligible posts from the existing composer while keeping `Post now` as the safe default.
- G-002: Publish each due post no earlier than its selected time and at most once as a visible PDS record.
- G-003: Let members view, edit, and delete up to three retained unpublished schedules from Settings.
- G-004: Keep all unpublished content private to the member and the AppView until publication.
- G-005: Make due publication durable across app closure, device changes, AppView restarts, transient outages, and worker concurrency.
- G-006: Preserve Craftsky's architectural rule that public writes go through the member's PDS using server-held OAuth sessions.

## 8. Non-Goals

- NG-001: Scheduling comments or replies.
- NG-002: Scheduling reposts, likes, follows, profile changes, Instagram imports, or any record type other than `social.craftsky.feed.post`.
- NG-003: Recurring posts, content calendars, bulk scheduling, queues larger than three, suggested posting times, or automatic timezone optimization.
- NG-004: Draft autosave, general-purpose cross-device drafts, templates, approval workflows, or team/business-account collaboration.
- NG-005: Editing or deleting a post through scheduled-post management after its PDS record has been published; normal post controls apply after publication.
- NG-006: A public scheduled-time field or a lexicon change. `scheduledAt` remains private AppView metadata.
- NG-007: A dedicated one-tap Post now action on the management row. A member may open Edit and submit the full composer with Post now selected.
- NG-008: Email, push, or in-app notifications for upcoming, successful, retrying, or Needs attention schedules.
- NG-009: Selecting a final cloud/object-storage provider in this requirements stage.
- NG-010: Extracting all existing AppView workers into a separate service as part of this feature.
- NG-011: Deleting unreferenced blobs from a member's PDS after a partial publication attempt; Craftsky does not attempt to delete PDS data outside normal owned-record deletion behavior.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Scheduling member | Signed-in Craftsky member preparing an original standard or project post for later. | Clear default-to-now behavior, a future-time picker, capacity feedback, and confidence that the durable unpublished copy remains private. |
| Managing member | The same account viewing schedules from Settings, potentially on another device. | An authoritative list plus safe edit/delete controls and visible failure status. |
| Scheduled-post worker | AppView-owned background process claiming due work. | Durable state, exclusive leases, a usable owner OAuth session, private media, stable record identity, and retry/recovery rules. |
| Member's PDS | External authority that stores media blobs and the final public record. | Valid authenticated writes using the existing lexicon and media constraints. |
| AppView operator | Person operating the API, database, worker, and private media store. | Queue health metrics, safe diagnostics, cleanup, and no sensitive-content leakage. |

## 10. Current Behavior

The standard and project composers submit through `createPostProvider`, which calls `POST /v1/posts`. Images have already been prepared and uploaded to the member's PDS as they are selected. The AppView validates the payload, stamps the current `createdAt`, creates the PDS record immediately, and returns a synthetic post that Flutter inserts into live caches. Replies/comments use the same standard composer with a `reply` reference. Settings contains no scheduled-post destination.

## 11. Desired Behavior

An eligible composer has a visible `When` row that initially shows `Now`. The member can explicitly choose Schedule for later and select a whole-minute instant from five minutes through 28 days ahead. On Schedule, Flutter reprocesses retained image bytes, uploads them to private staging, and creates the schedule. It does not optimistically add a public post to timeline/profile caches. Initial eager PDS uploads remain unreferenced and expire through normal PDS cleanup.

Settings contains Scheduled posts with a Needs attention count. The screen shows only the current account's retained unpublished schedules ordered by intended publication time. Rows include an authenticated thumbnail, kind/title/text preview, localized absolute time/timezone, and one of Scheduled, Publishing, Retrying, or Needs attention. Tapping opens the full composer; Delete is also available with confirmation. Publishing locks both actions, and pull-to-refresh is the only live refresh control.

At or after `scheduledAt`, a durable Go worker exclusively transitions the schedule to Publishing, allocates its stable TID, freezes `createdAt`, revalidates current policy, selects a usable owner OAuth session, uploads staged media to the PDS, and writes the identical record body on every attempt. Transient failures retry only at the agreed cadence through 30 minutes late. Permanent or exhausted failure becomes Needs attention without changing content. Success removes the item from management, cleans staged media, releases capacity, and leaves a content-free 30-day reconciliation tombstone.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | A signed-in member shall be able to schedule an eligible post for future publication without keeping the Craftsky app open or online. | Scheduling is useful only if execution is server-reliable. | Prompt / Discovery | AC-001, AC-016 |
| BR-002 | Business | Must | A member shall be able to retain and manage no more than three unpublished schedules for their account at one time. | Implements the requested product limit across devices. | Prompt | AC-006, AC-007 |
| BR-003 | Business | Must | A member shall be able to view, edit, and delete their retained unpublished schedules from a destination under Settings. | Implements the requested management experience. | Prompt | AC-008, AC-009, AC-010 |
| BR-004 | Business | Must | The authoritative unpublished payload and durable media copy shall remain private to the member and AppView until a due publication attempt writes the final record/media to the member's PDS. | Schedules are private-by-intent even though the existing eager pipeline may leave a short-lived unreferenced PDS blob. | Architecture / Discovery / Grilling | AC-012, AC-013 |
| FR-001 | Functional | Must | Eligible standard and project composers shall expose a visible `When` row defaulting to `Now`; scheduling shall require selecting `Schedule for later`, and the primary action shall change from Post to Schedule. | Prevents accidental delayed publication and makes timing explicit. | Prompt / Grilling | AC-001, AC-025 |
| FR-002 | Functional | Must | Scheduling shall be available only for original top-level standard posts and project posts; the UI and API shall reject quote posts and any payload containing a reply/comment reference. | Locks the agreed first-release content boundary. | Prompt / Grilling | AC-002, AC-003 |
| FR-003 | Functional | Must | Scheduling shall accept only whole-minute absolute instants from five minutes through 28 days after server acceptance, display the relevant local timezone/offset, encode UTC on the API, and never move the instant after device timezone or daylight-saving changes. | Aligns UX with media staging and the 30-day OAuth inactivity boundary. | Grilling | AC-004, AC-025 |
| FR-004 | Functional | Must | Submitting with `Post now` shall continue through the immediate `POST /v1/posts` behavior and shall not create scheduled-post state. | Preserves the existing happy path. | Prompt / Codebase | AC-001 |
| FR-005 | Functional | Must | On Schedule, Flutter shall reprocess retained local image bytes, upload them to private staging with visible progress, then submit the fully validated standard/project/language/facet/media payload; the AppView shall idempotently persist the owner-scoped snapshot using a client operation key and return a scheduled resource without creating a PDS post record. The composer shall close only after all steps succeed and remain intact with retry on failure. | Gives the worker a durable private source and makes lost responses safe. | Discovery / Grilling | AC-005, AC-012, AC-025 |
| FR-006 | Functional | Must | The AppView shall enforce the three-schedule limit transactionally per authenticated DID across concurrent devices/requests; at capacity Flutter shall keep Schedule visible but disabled with `3 of 3 scheduled`, provide a management link, and leave Post now enabled. | Client checks alone race, while clear capacity UX avoids a dead end. | Prompt / Grilling | AC-006, AC-007, AC-026 |
| FR-007 | Functional | Must | The AppView shall provide owner-authenticated `/v1/scheduled-posts` create/list/get/update/delete operations using camelCase JSON, the standard error envelope, and route-policy protections. | Gives Flutter an explicit private resource contract consistent with the API architecture. | Codebase / Recommended direction | AC-005, AC-008, AC-009, AC-010, AC-011 |
| FR-008 | Functional | Must | Settings shall show Scheduled posts with a Needs attention count. Its screen shall list only the active account's unpublished items ordered by `scheduledAt`, with authenticated first-image thumbnail when present, type, project title when present, bounded text preview, localized time/timezone, and exactly one of Scheduled, Publishing, Retrying, or Needs attention. Published history shall not appear. | Makes up to three private items identifiable while keeping status plain. | Prompt / Grilling | AC-008, AC-011, AC-027, AC-031, AC-032 |
| FR-009 | Functional | Must | Tapping a row shall open the complete standard/project composer with every existing editable field and private media previews. Future items preserve Schedule/time; Needs attention shows the missed time and defaults to Post now. Member saves are last-write-wins, while the worker still fences stale payload versions internally. | Delivers full editing without user-visible merge conflicts. | Prompt / Grilling | AC-009, AC-028 |
| FR-010 | Functional | Must | Delete shall be available from the row overflow and editor with confirmation until Publishing begins; an accepted delete shall prevent publication, remove the item, release its slot, and promptly clean private staged media. | Gives cancellation concrete race and cleanup semantics. | Prompt / Grilling | AC-010, AC-014, AC-029 |
| FR-011 | Functional | Must | A durable Go worker shall claim due schedules in bounded batches with exclusive expiring leases and `FOR UPDATE SKIP LOCKED`-equivalent semantics, and shall atomically enter Publishing at or after `scheduledAt` before any external PDS upload/write. Publishing shall lock Edit/Delete; the screen refreshes only on entry or pull-to-refresh. | Multiple processes, user mutations, and external writes need a precise serialization boundary. | Codebase / Grilling | AC-014, AC-015, AC-029 |
| FR-012 | Functional | Must | At the first publication attempt, the worker shall select a currently usable owner session, revalidate current post/media/authorization rules, allocate and persist a stable PDS TID, freeze `createdAt` at that attempt time, upload current staged media, and write `social.craftsky.feed.post`; every retry shall reuse the same record identity and body. | Preserves ownership and makes the external side effect recoverable and CID-stable. | Architecture / Grilling | AC-013, AC-016, AC-017, AC-018, AC-020 |
| FR-013 | Functional | Must | Recovery after lease loss, restart, or ambiguous PDS response shall reconcile the stable record identity so the schedule produces at most one visible post even when the PDS write succeeded before local completion was recorded. | PDS writes and AppView state cannot be committed atomically. | Discovery / Grilling | AC-017 |
| FR-014 | Functional | Must | On success, the system shall persist URI/CID, mark the item published, remove it from management/capacity, immediately enqueue staged-media/payload cleanup, and retain only a content-free owner/schedule/idempotency/URI/CID/timestamp tombstone for 30 days. | Completes lifecycle and protects delayed replay without retaining content. | Grilling | AC-016, AC-017, AC-030, AC-032 |
| FR-015 | Functional | Must | Transient failures shall retry at approximately due time and +1, +3, +7, +15, and +30 minutes, with bounded jitter that never extends the 30-minute window. The final failure becomes Needs attention. Missing active usable authentication follows the same window; permanent validation/authorization/media failure enters Needs attention immediately. Craftsky shall never alter invalid content automatically. | Bounds lateness and preserves member intent. | Grilling | AC-018, AC-019, AC-020 |
| FR-016 | Functional | Must | Flutter shall not optimistically insert a scheduled item into public timeline/profile/project caches; after successful scheduling it shall close the composer with clear scheduled confirmation and refresh scheduled-post state. | The PDS post does not exist yet. | Codebase / Discovery | AC-005 |
| FR-017 | Functional | Must | The current eager `POST /v1/blobs/images` path may upload selected images to the PDS. If Schedule is submitted, Flutter shall independently reprocess retained local bytes and upload them to authenticated owner-scoped private staging. The unreferenced initial PDS blob is left for documented PDS cleanup. Private previews require the owner, and replaced/deleted objects are cleaned safely. | Preserves current composer latency while establishing a durable private source. | AT Protocol docs / Grilling | AC-012, AC-013, AC-014 |
| FR-018 | Functional | Must | A single-device sign-out shall not delete schedules. Publication may continue only while at least one active usable Craftsky session exists for that DID; last-session sign-out prevents the write and follows the retry/Needs attention flow. Reauthentication never auto-resumes Needs attention. Account deletion cancels work and removes unpublished payload/media. | Scheduled intent is account-owned, but public writes require active authorization. | Auth architecture / Grilling | AC-018, AC-021 |
| FR-019 | Functional | Should | The first implementation should run the worker inside AppView but keep queue/store/worker interfaces independent of HTTP routing so it can move to a separate Compose service without API/schema changes. | Minimizes initial operations while preserving the scale path. | Recommended direction / Grilling | AC-022 |
| FR-020 | Functional | Must | Unclaimed private uploads shall expire after 24 hours; Needs attention items shall show their deletion date and expire with payload/media after 30 days; successful publication cleanup shall begin immediately; published content-free tombstones shall expire after 30 days. Cleanup failures shall retry without extending user-visible retention intentionally. | Gives every private/stable artifact a bounded lifecycle. | Grilling | AC-030 |
| FR-021 | Functional | Must | This release shall create no push or in-app notification for scheduled publication success, retry, or Needs attention. Visibility is limited to the Settings count, management status, and normal published-post surfaces. | Avoids expanding into notification producers/preferences. | Grilling | AC-031, AC-032 |
| NFR-001 | Non-functional | Must | Schedule state, payloads, leases, attempts, and recovery identity shall survive Flutter termination and AppView/worker restart. | Durability is the core value over client timers. | Prompt / Discovery | AC-015, AC-016, AC-017 |
| NFR-002 | Non-functional | Must | Under healthy dependencies, a schedule shall never publish before `scheduledAt` and its first publication attempt shall begin within 60 seconds after `scheduledAt`. | Sets a testable timeliness expectation without pretending external PDS completion is instantaneous. | Recommended direction / Assumption | AC-015, AC-016 |
| NFR-003 | Non-functional | Must | Scheduled payloads, image bytes, alt text, project details, facets, OAuth tokens, and full content previews shall not appear in logs, metrics, traces, error messages, object keys, or analytics events. | Unpublished content and credentials are sensitive. | Privacy boundary | AC-023 |
| NFR-004 | Non-functional | Must | Private staged media shall use TLS in transit, private S3-compatible storage with platform-standard encryption at rest, least-privilege service credentials, and authenticated owner-scoped access. Custom application-level encryption is not required. | States the necessary storage controls without inventing a key-management subsystem. | User clarification / Grilling | AC-012, AC-023, AC-030 |
| NFR-005 | Non-functional | Should | Scheduling and management UI shall use localized, accessible labels; announce validation/status changes; and remain usable with large text and screen readers. | Matches the rest of the Flutter product's accessibility and localization expectations. | Codebase convention | AC-024 |
| NFR-006 | Non-functional | Must | The system shall emit content-free, low-cardinality operational signals sufficient to measure unpublished queue depth by safe status, due/overdue depth and oldest age, publication start latency and duration, attempts/retries/Needs attention transitions, stale-worker fences, lease recovery, and cleanup outcomes. Production configuration shall alert on sustained overdue work, worker/claim failures, exhausted retries/auth unavailability, or cleanup backlog without content or account identifiers. | The initial in-process worker can fail while HTTP remains healthy; operators need safe evidence that schedules and cleanup are progressing. | Document review DR-002 | AC-033 |
| RULE-001 | Business rule | Must | Scheduled, Publishing, Retrying, and Needs attention resources count toward the limit of three; published or deleted resources do not. | Makes the cap deterministic through failure states. | Prompt / Grilling | AC-006, AC-007, AC-019 |
| RULE-002 | Business rule | Must | A schedulable payload must contain neither `reply` nor a quote embed; only original top-level standard posts and existing-valid project posts are eligible. | Defines the agreed post set. | Grilling | AC-002, AC-003 |
| RULE-003 | Business rule | Must | On create/update, `scheduledAt` shall be a whole-minute absolute UTC instant at least five minutes and no more than 28 days after server acceptance. Later timezone/DST changes never recompute it. | Makes time validation deterministic. | Grilling | AC-004, AC-025 |
| RULE-004 | Business rule | Must | Only the authenticated owner DID may list, read, update, delete, preview media for, or otherwise act on a schedule. | Scheduled content is private account data. | Architecture | AC-011, AC-012 |
| RULE-005 | Business rule | Must | An Edit/Delete committed before the due worker enters Publishing wins; the Publishing transition occurs before external PDS writes and then locks both actions. Member saves are last-write-wins, but stale worker versions may never publish. | Defines the mutation/publication race without an artificial pre-time lockout. | Grilling | AC-014, AC-028, AC-029 |
| RULE-006 | Business rule | Must | The automatic attempt window ends 30 minutes after `scheduledAt`; bounded jitter may move attempts within but never beyond it. Permanent invalidity moves directly to Needs attention and content is never automatically changed. | Prevents unexpectedly stale or altered publication. | Grilling | AC-019, AC-020 |
| RULE-007 | Business rule | Must | A schedule shall never be published early to compensate for worker shutdown, deployment, daylight-saving changes, or anticipated dependency outage. | The chosen instant is the earliest allowed public write time. | Product expectation | AC-015 |
| RULE-008 | Business rule | Must | Needs attention never auto-resumes after reauthentication; recovery requires Edit followed by Post now or a new valid schedule, or Delete. | Prevents stale content appearing unexpectedly. | Grilling | AC-018, AC-019 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-001, FR-004 | Given a new eligible composer, when it opens and the member submits without changing the When row, then Now is selected and the existing immediate-create path runs without scheduled state. |
| AC-002 | FR-002, RULE-002 | Given an original standard top-level post or project post, when its composer is shown, then Schedule for later is available. |
| AC-003 | FR-002, RULE-002 | Given a quote, comment, or reply composer or direct API payload, when scheduling is attempted, then the UI offers no enabled schedule path and the server rejects it without storing payload/media ownership. |
| AC-004 | FR-003, RULE-003 | Given Schedule for later, when create/update supplies a non-minute, less-than-five-minute, more-than-28-day, or otherwise invalid instant, then the server rejects it; a valid choice round-trips as the same UTC instant and remains fixed across timezone/DST changes. |
| AC-005 | FR-005, FR-016 | Given valid content and a slot, when Schedule is tapped, then private staging progress is shown, the schedule is created idempotently only after staging succeeds, no PDS post record/public cache item is created, and success closes the composer; failure preserves it for retry. |
| AC-006 | BR-002, FR-006, RULE-001 | Given an account already has three retained unpublished schedules, when any device attempts to create a fourth, then the API rejects it with a stable capacity error and stores no payload/media ownership change. |
| AC-007 | BR-002, FR-006, RULE-001 | Given two concurrent requests compete for the final available slot, when both transactions complete, then exactly one schedule is accepted and the retained unpublished count never exceeds three; deleting or publishing one later frees one slot. |
| AC-008 | BR-003, FR-007, FR-008 | Given retained unpublished schedules, when Settings/management loads, then the tile shows Needs attention count and only the active account's items appear in time order with authenticated thumbnail, type/title/text preview, localized absolute time/timezone, and one of the four agreed statuses. |
| AC-009 | BR-003, FR-007, FR-009 | Given a future item, Edit opens the full matching composer with Schedule/time preserved; given Needs attention, it shows the missed time and defaults to Post now; any full valid last-write-wins save replaces the content used by a later worker. |
| AC-010 | BR-003, FR-007, FR-010 | Given a non-Publishing schedule, when its owner confirms Delete from row/editor, then it disappears, cannot later publish, releases its slot, and staged cleanup begins. |
| AC-011 | FR-007, FR-008, RULE-004 | Given another authenticated DID guesses or obtains a schedule or media identifier, when it attempts list/get/update/delete/preview access, then the resource is not disclosed or mutated and the standard safe error contract is used. |
| AC-012 | BR-004, FR-005, FR-017, NFR-004, RULE-004 | Given an eagerly PDS-uploaded composer image, when the member schedules, then retained local bytes are reprocessed into owner-scoped private staging before acceptance, the original PDS blob remains unreferenced for PDS cleanup, no post record exists, and only the owner can preview the staged copy. |
| AC-013 | BR-004, FR-012 | Given a due image schedule with healthy dependencies, when publication runs, then each current staged image is uploaded to the owner's PDS before the final record references it, and no unpublished text/project payload was written to the PDS beforehand. |
| AC-014 | FR-010, FR-011, FR-017, RULE-005 | Given Edit/Delete races the worker, when the mutation commits before Publishing it wins and stale media/content cannot publish; when Publishing commits first both actions are locked and no false cancellation is shown. |
| AC-015 | FR-011, NFR-001, NFR-002, RULE-007 | Given a not-yet-due schedule, concurrency, restart, timezone changes, and downtime never publish it early; under healthy dependencies exactly one Publishing transition begins within 60 seconds after due time. |
| AC-016 | BR-001, FR-012, FR-014, NFR-001, NFR-002 | Given a healthy due schedule while Flutter is closed/offline, then it publishes once using the first-attempt frozen `createdAt`, records URI/CID, disappears/releases capacity, and enqueues immediate private cleanup plus the 30-day tombstone. |
| AC-017 | FR-011, FR-013, FR-014, NFR-001, RULE-005 | Given the worker loses its lease or crashes immediately before or after a successful PDS write, when work is recovered, then stable record reconciliation completes the same schedule without a second visible post or a false cancellation result. |
| AC-018 | FR-012, FR-015, FR-018, RULE-008 | Given the last active session is gone or expires, then no other-account/anonymous write occurs; attempts stop at 30 minutes and Needs attention never auto-resumes after sign-in, while Edit can Post now/reschedule. |
| AC-019 | FR-015, RULE-001, RULE-006, RULE-008 | Given a transient failure, attempts occur at due/+1/+3/+7/+15/+30 minutes within bounded jitter; final failure becomes Needs attention, remains Edit/Delete-capable, counts toward three, and is never retried automatically afterward. |
| AC-020 | FR-012, FR-015, RULE-006 | Given current validation, media, mention authorization, or block state makes the content invalid, the first such attempt enters Needs attention immediately without removing/changing any member-authored content. |
| AC-021 | FR-018 | Given one of multiple sessions signs out, publication may use another active owner session; given last-session sign-out it cannot publish; given account deletion, due work is fenced and all unpublished payload/media is removed. |
| AC-022 | FR-019 | Given the worker is wired in-process initially, when its package dependencies and construction are inspected/tested, then queue processing is callable independently of HTTP routing and does not require request context or a Flutter device. |
| AC-023 | NFR-003 | Given canary text, alt text, facet, project, media, token, and object-key values, when create/edit/preview/publish/retry/delete paths run, then automated log/trace/metric/error capture contains none of the canaries or raw unpublished content. |
| AC-024 | NFR-005 | Given screen-reader use, large text, and localized strings, when a member schedules, manages, edits, encounters validation/failure, or confirms deletion, then controls have meaningful labels, status changes are announced, and required actions remain reachable without clipping. |
| AC-025 | FR-001, FR-003, FR-005, RULE-003 | Given an eligible composer, the When row defaults to Now; choosing Schedule for later opens a whole-minute picker limited to five minutes through 28 days, changes the action to Schedule, and begins private staging only after that action is tapped. |
| AC-026 | FR-006 | Given all three slots are occupied, Schedule remains visible but disabled with `3 of 3 scheduled` and a Manage scheduled posts link, while Post now remains enabled. |
| AC-027 | FR-008 | Given the management screen is open, status changes are not polled automatically; pull-to-refresh or re-entry fetches the latest Scheduled/Publishing/Retrying/Needs attention state. |
| AC-028 | FR-009, RULE-005 | Given two devices save full edits concurrently, the last accepted member save is authoritative; given a worker holds an older internal payload version, its fencing prevents that stale version from publishing. |
| AC-029 | FR-010, FR-011, RULE-005 | Given status is Publishing, Edit/Delete are locked with clear copy until manual refresh shows success/disappearance or a recoverable state. |
| AC-030 | FR-014, FR-020, NFR-004 | Given unclaimed staging, it is deleted after 24 hours; Needs attention shows and enforces a 30-day deletion date; successful private cleanup begins immediately; content-free published tombstones are deleted after 30 days; referenced/live objects are never removed by orphan cleanup. |
| AC-031 | FR-008, FR-021 | Given a schedule retries or enters Needs attention, no push/in-app notification is created; the Settings count and management status are the only proactive product surfaces. |
| AC-032 | FR-008, FR-014, FR-021 | Given publication succeeds, the item disappears from management, no success notification/history row is created, and the post is discoverable only through normal post surfaces while the internal tombstone remains hidden. |
| AC-033 | NFR-006 | Given deterministic queue, publication, recovery, authentication-failure, and cleanup fixtures, when lifecycle work runs, then the operational recorder exposes the required queue/latency/outcome signals using only approved low-cardinality, content-free attributes; and the production alert configuration has verifiable thresholds for sustained overdue/worker/auth/cleanup failure conditions. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | Two devices create the third and fourth schedules concurrently. | The database accepts only one final-slot claim and returns a stable capacity error for the other. | FR-006, RULE-001 |
| EC-002 | Client-selected future time becomes past during network transit. | Server-time validation rejects the request; the client keeps the composer content and asks for a new future time. | FR-003, RULE-003 |
| EC-003 | Device timezone or daylight-saving rules change after scheduling. | The stored UTC instant does not move; UI re-renders that instant in the device's current local timezone and shows the relevant offset. | FR-003, RULE-003 |
| EC-004 | App is terminated immediately after the schedule API succeeds but before local confirmation. | A refresh/list shows the authoritative schedule; retrying create with the same client idempotency key does not consume a second slot. | FR-005, FR-006, NFR-001 |
| EC-005 | AppView is down at the scheduled instant. | The durable row remains due and is claimed after recovery; it is never published early. Queue lateness is observable. | FR-011, NFR-001, RULE-007 |
| EC-006 | Two worker instances see the same due row. | Only the valid lease/version owner may cross the PDS-write boundary and finalize it. | FR-011, FR-013 |
| EC-007 | Worker crashes after one or more PDS blob uploads but before the record write. | The schedule remains recoverable and reuses/re-uploads content-addressed media as needed; no record is published until all current media is available. Craftsky does not attempt PDS blob deletion. | FR-012, FR-013, NG-011 |
| EC-008 | Worker crashes after the PDS record write but before local success. | Stable record identity allows recovery to find/replace/reconcile the same record rather than create another. | FR-013, FR-014 |
| EC-009 | Member edits or deletes while the worker is approaching due work. | A committed mutation before Publishing wins; once Publishing commits, both actions are locked and manual refresh reveals the eventual state. | FR-009, FR-010, FR-011, RULE-005 |
| EC-010 | The last active session expires or signs out. | No anonymous/other-account write occurs. Attempts end after 30 minutes in Needs attention; later sign-in never auto-resumes it. | FR-015, FR-018, RULE-008 |
| EC-011 | Member attempts to schedule a quote post through Flutter or the API. | Scheduling is unavailable/rejected with no private schedule retained. | FR-002, RULE-002 |
| EC-012 | Validation or configured media limits change before due time. | Current rules are checked; invalid content enters Needs attention immediately and is not altered automatically. | FR-012, FR-015, RULE-006 |
| EC-013 | Scheduled image is removed or replaced during edit. | The new version alone is publishable; old private objects are reference-checked and cleaned without deleting media still used by the new version. | FR-009, FR-017, NFR-004 |
| EC-014 | Private object upload succeeds but schedule creation fails. | No slot/schedule is created; an idempotent ownership/expiry mechanism deletes the unclaimed object after 24 hours. | FR-017, FR-020, NFR-004 |
| EC-015 | Publication succeeds but Tap indexing is delayed. | Worker completion uses the direct PDS URI/CID and does not republish; the normal firehose path eventually populates read models. | FR-013, FR-014 |
| EC-016 | A Needs attention schedule occupies the third slot. | It remains counted until edited and posted/rescheduled, deleted, or its visible 30-day expiry removes it; Schedule is disabled with management guidance meanwhile. | FR-006, FR-015, FR-020, RULE-001 |
| EC-017 | Member tries to schedule a reply/comment by calling the API directly. | Server rejects it regardless of Flutter UI state. | FR-002, RULE-002 |
| EC-018 | Account deletion races a due worker. | Terminal deletion/cancellation fencing prevents a later PDS write and removes private data. | FR-018, NFR-004 |
| EC-019 | Member changes device timezone after scheduling. | The UTC instant does not move; management localizes that instant with an explicit timezone/offset. | FR-003, RULE-003 |
| EC-020 | Image was eagerly uploaded to the PDS before Schedule was chosen. | Scheduling reprocesses retained local bytes into private staging; the original PDS blob remains unreferenced and is left for protocol-defined cleanup. | FR-005, FR-017 |
| EC-021 | Two devices save different full edits nearly simultaneously. | The last accepted member save wins, while internal version fencing prevents a previously claimed payload from publishing. | FR-009, RULE-005 |
| EC-022 | Needs attention reaches its 30-day expiry. | The item, payload, and private media are deleted, its slot is released, and it cannot later auto-publish. | FR-020, RULE-008 |

## 15. Data / Persistence Impact

- New fields/tables:
  - A private `scheduled_posts` resource keyed by opaque UUID, with owner DID, post kind, validated payload snapshot, absolute `scheduled_at`, lifecycle status, internal version, client idempotency key, first-attempt/frozen-`createdAt`, PDS TID/URI/CID allocated at first attempt, attempt/next-attempt fields, lease token/expiry, safe last-error code, Needs attention expiry, and created/updated timestamps.
  - Private staged-media metadata keyed by opaque IDs, with owner/schedule/claim ownership, MIME/size/checksum, S3-compatible object reference, lifecycle state, claim/cleanup expiry, and timestamps. Raw bytes do not belong in schedule JSON or logs.
  - A published tombstone representation containing no post payload/media/preview data and expiring 30 days after publication.
- Changed fields:
  - None in public post lexicons.
  - Existing immediate post and PDS blob request shapes remain valid.
- Migration required:
  - Yes. Add private AppView tables, constraints, indexes for owner/capacity and due-work claims, lifecycle checks, and cleanup/recovery.
  - The exact schema normalization and retention partitioning belong in coding planning, but the transactional limit and lease/idempotency invariants are requirements.
- Backwards compatibility:
  - The app is not in production, so coordinated Flutter/AppView API addition does not require compatibility shims.
  - No existing PDS record or lexicon migration is required.
- Retention:
  - Unclaimed staging expires after 24 hours.
  - Replaced/deleted/account-deleted private media is cleaned promptly with retry.
  - Successful publication immediately enqueues private payload/media cleanup and retains only a content-free tombstone for 30 days.
  - Needs attention displays its deletion date and expires with payload/media after 30 days.

## 16. UI / API / CLI Impact

- UI:
  - Add a `When` row defaulting to Now to original-standard and project composers only; Schedule for later uses whole-minute selection from five minutes through 28 days and changes the primary action to Schedule.
  - Preserve eager PDS image upload. On Schedule submission, reprocess retained local bytes and show private-staging progress; keep the composer intact on failure.
  - Add Settings → Scheduled posts with Needs attention count and an unpublished-only, pull-to-refresh list of authenticated thumbnail/type/title/text/timezone/status rows.
  - Row tap opens the full standard/project composer. Future edits preserve schedule/time; Needs attention shows the missed time and defaults to Post now. Row/editor Delete requires confirmation.
  - Expose only Scheduled, Publishing, Retrying, and Needs attention; Publishing locks Edit/Delete.
  - Use account-keyed providers so switching active accounts cannot show or mutate another account's schedules.
- API:
  - `POST /v1/scheduled-posts` — create a schedule with a client idempotency key and validated payload/private staged-media references.
  - `GET /v1/scheduled-posts` — list the owner's retained unpublished schedules; pagination is unnecessary while the hard cap is three.
  - `GET /v1/scheduled-posts/{id}` — fetch the owner's full editable representation.
  - `PUT /v1/scheduled-posts/{id}` — atomically replace the full editable content/timing with member-visible last-write-wins semantics; internal version fencing remains for worker races.
  - `DELETE /v1/scheduled-posts/{id}` — cancel/delete before publication, idempotently where safe.
  - An authenticated private staged-image upload/preview contract separate from `POST /v1/blobs/images`; exact URL shape is deferred to API design/coding planning but remains under `/v1/`.
  - Stable machine errors include capacity reached, invalid scheduled time, invalid post kind, publication already started/completed, private media invalid/missing, and standard validation/auth errors.
- CLI:
  - None required for the member experience.
  - A future separate worker command may use the same worker package but is not required in the first deployment.
- Background jobs:
  - Add an in-process due-publication worker with database leases, Publishing fencing, first-attempt TID/`createdAt`, current validation/authorization, active owner-session selection, private-media upload, idempotent PDS write/reconciliation, and the fixed six-attempt/30-minute policy.
  - Add S3-compatible orphan/lifecycle cleanup, including 24-hour unclaimed, immediate success/delete/replacement, 30-day Needs attention, and 30-day tombstone expiry.

## 17. Security / Privacy / Permissions

- Authentication:
  - Every schedule and private-media API requires a valid Craftsky bearer session and device ID under existing middleware.
  - The worker uses only a currently usable OAuth session selected for the schedule owner DID that is backed by at least one active Craftsky session.
- Authorization:
  - Owner DID is derived from authentication, never trusted from request JSON.
  - All reads/mutations/previews filter by owner DID and opaque ID.
  - Publication-time mention/directed-interaction authorization is rechecked because relationships may change after scheduling.
- Sensitive data:
  - Unpublished text, project fields, facets, alt text, images, local timezone preference, and failure context are private AppView data.
  - OAuth tokens remain solely in the existing OAuth session store and are never copied into schedule rows, tasks, media metadata, or logs.
  - Private object identifiers must be opaque and must not embed DID, handle, text, filename, alt text, schedule time, or other member content.
  - Private staging uses TLS, S3-compatible private access, least-privilege credentials, and provider-standard encryption at rest. Custom application-level encryption is not required.
  - Initial eager PDS uploads are not treated as the durable scheduled copy; they remain unreferenced and are left for PDS cleanup.
- Abuse cases:
  - The server-enforced cap bounds per-account queued payloads but does not replace upload-body limits, rate limits, authenticated staging quotas, or orphan cleanup.
  - Concurrent create/edit/delete and stale external wakeups must be harmless through idempotency/version checks.
  - A guessed schedule/media ID must disclose neither existence nor content to another account.
  - Retry policy must avoid an unbounded PDS write loop.
- Account lifecycle:
  - Single-device sign-out does not erase account-owned schedules.
  - Last-active-session sign-out prevents publication; the item follows the retry window then Needs attention and never auto-resumes after sign-in.
  - Terminal account deletion cancels due work and deletes private content/media.

## 18. Observability

- Events:
  - Safe lifecycle transitions: created, edited, deleted, Publishing entered, publication attempted, Retrying scheduled, Needs attention entered/expired, PDS write reconciled, published, tombstone expired, media cleanup completed/failed.
  - Analytics, if added, record only coarse action/post-kind/status and never scheduled content, exact schedule time, media identifiers, DID/handle, or failure payload.
- Logs:
  - Structured component/operation/result/error-category attributes plus opaque schedule ID/run ID where operationally necessary.
  - Never log payload JSON, text, facets, project details, alt text, image bytes, raw object keys, bearer/OAuth tokens, full PDS error bodies, or signed preview URLs.
- Metrics:
  - Unpublished queue depth by safe status.
  - Due/overdue depth and oldest due age.
  - Claim, publish, retry, Needs attention, stale-worker-fence, and cleanup counts.
  - Publication start latency from `scheduledAt` and completion duration.
  - Lease recoveries and stable-record reconciliations.
- Alerts:
  - Oldest due age or overdue depth above the healthy 60-second start target.
  - Sustained worker/claim failures, growing Needs attention rates, exhausted retries, repeated auth unavailability, or cleanup backlog.
  - Alert dimensions must remain low-cardinality and content-free.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Worker crashes after the PDS accepts the post but before AppView records success. | Duplicate public posts on retry. | Allocate/persist the TID and freeze the body when the first attempt enters Publishing; every retry reconciles that identity/body and persists URI/CID. |
| RISK-002 | Edit/delete races a due worker. | Cancelled or stale content is published. | Versioned payloads, exclusive leases, cancellation fencing, and a defined public-write boundary. |
| RISK-003 | OAuth session expires or the last active Craftsky session signs out before due time. | Post cannot publish automatically. | Owner-scoped active-session selection, fixed 30-minute retries, Needs attention, and explicit Edit/Post now/reschedule recovery after sign-in. |
| RISK-004 | Eager PDS upload succeeds but private re-upload fails or the PDS does not promptly clean the unreferenced blob. | Scheduling fails or a short-lived unreferenced blob persists longer than expected. | Preserve composer bytes/state for retry, accept a schedule only after private staging succeeds, never reference the initial blob, and rely on the documented PDS cleanup contract rather than it as durable storage. |
| RISK-005 | Private staged media is stored in Postgres/local container disk or retained indefinitely. | Database/backup bloat, host-coupled loss, privacy exposure, and high storage cost. | Use S3-compatible private storage/MinIO interface plus exact 24-hour/30-day/immediate cleanup policies and metrics. |
| RISK-006 | In-process worker is omitted, disabled, or unavailable while HTTP remains healthy. | Schedules become overdue without obvious API failure. | Startup/config tests, queue-age metrics/alerts, health visibility, and an extraction-ready worker interface. |
| RISK-007 | Publication-time mention/block/validation state differs from scheduling time. | Unauthorized or semantically changed content is published. | Revalidate at publication and move the unchanged draft directly to Needs attention. |
| RISK-008 | Scheduled time is interpreted using changing device timezone rules. | Publication occurs at an unexpected instant. | Store absolute UTC, show local timezone/offset when chosen and managed, and never recompute from a floating local time. |
| RISK-009 | Infrastructure scheduler is introduced without a database outbox/reconciler. | Missing or stale wakeups after partial failure. | Keep Postgres authoritative; make all wakeups hints and all handlers version/idempotency checked. |
| RISK-010 | Needs attention does not count or expire. | Failed private drafts/media accumulate. | Count it toward three, display the deletion date, and expire content/media after 30 days. |
| RISK-011 | Logs/traces capture request bodies, signed URLs, or provider errors. | Unpublished content or credentials leak operationally. | Redaction-by-construction, canary tests, bounded safe error codes, and no raw body instrumentation. |
| RISK-012 | Object storage is unavailable after schedule creation. | Due posts cannot access media. | Accept only durably staged objects, retry within the fixed window, and enter Needs attention without altering the post on persistent loss. |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | Conforming PDS implementations honor `com.atproto.repo.uploadBlob`'s documented deletion of blobs left unreferenced beyond a short window. | Eager uploads may remain retrievable longer than expected; the feature still works because it never relies on them, but privacy copy/risk language and possibly the eager pipeline would need revision. |
| ASM-002 | A managed S3-compatible private bucket is available for production and MinIO is acceptable in local Compose. | Storage adapter/provider and deployment scope must be revisited; local AppView disk and Postgres bytes remain rejected defaults. |
| ASM-003 | Under healthy dependencies, beginning Publishing within 60 seconds is an acceptable target. | A tighter service target may require a different poll/wakeup or deployment model. |
| ASM-004 | The existing app can retain/reprocess selected local image bytes until Schedule submission completes. | Composer image state must be adjusted to retain a suitable source; falling back to the unreferenced PDS blob is not allowed. |

## 21. Open Questions

None identified. The specific managed S3-compatible production vendor is a deployment choice, not a product or requirements blocker.

## 22. Review Status

Status: Revised; pending re-review

Risk level: High

Review recommended: Required

Reviewer: Codex

Date: 2026-07-31

Notes: The 2026-07-31 grilling review resolved post scope, time range/precision, media flow/storage/encryption, worker placement, retry cadence, failure recovery/retention, session behavior, management UX, concurrency, notifications, and PDS identity/timestamps. The user then authorized test design and document review. This revision formalizes the observability requirement identified by DR-002; the paired acceptance-test revision addresses DR-001 and DR-003. Re-review and explicit approval are still required before implementation because this feature stores private unpublished content and performs delayed authenticated public writes.

## 23. Handoff To Test Design

- Requirements file: `01-requirements.md`
- Next test specification: `02-acceptance-tests.md`
- Must-cover requirement IDs:
  - `BR-001` through `BR-004`
  - `FR-001` through `FR-018`, `FR-020`, and `FR-021`
  - `NFR-001` through `NFR-004`, plus `NFR-006`
  - `RULE-001` through `RULE-008`
- Suggested test levels:
  - Flutter widget/provider tests for When/Now, eligible/excluded composer types, five-minute/28-day whole-minute UTC behavior, full-cap UX, four statuses, Settings count/list rows, full edit defaults, last-write-wins, Publishing locks, pull-to-refresh, submit-time private staging, deletion, and no optimistic public-cache insertion.
  - AppView request/contract tests for strict bodies, quote/reply rejection, time bounds, ownership, capacity, idempotent create, last-write-wins member updates, internal publishing conflicts, error envelope, route policy, private preview, and lifecycle transitions.
  - Database integration tests for the scheduled-post migration schema/constraints/indexes, concurrent final-slot creation, Publishing claims, lease expiry/recovery, edit/delete fencing, stale-worker version rejection, exact status/count constraints, and tombstone/expiry cleanup ownership.
  - Worker unit/integration tests for no-early/60-second timing, exact six-attempt/30-minute policy, active-session selection/sign-out, current validation without mutation, media upload, first-attempt TID/`createdAt`, stable body/identity, crash windows, retry exhaustion, and Needs attention recovery rules.
  - S3-compatible integration tests against MinIO for owner-scoped access, 24-hour orphan cleanup, replacement/delete/account cleanup, immediate success cleanup, 30-day Needs attention expiry, and 30-day tombstone expiry.
  - Privacy canary tests across logs, traces, metrics, errors, object keys, and unauthorized preview access.
  - Operational signal tests for queue depth/age, publication latency/outcomes, retries/fences/recovery, and cleanup, plus a manual production alert-configuration check.
  - Manual device tests for date/time picker/When row, DST/travel display, accessibility/large text, eager-PDS-to-private staging progress/failure, app termination, multi-device last-write-wins/manual refresh, and end-to-end publication against a dev PDS.
- Blocking open questions:
  - No product questions remain. Re-review and explicit approval are required because the review risk is High.
