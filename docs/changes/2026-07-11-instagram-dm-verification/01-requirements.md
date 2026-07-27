# Requirements: Instagram DM Ownership Verification And Automatic Following

## 1. Initial Request

Implement the Instagram DM verification and follow-discovery work described in `design-plan.md` across AppView and Flutter. Complete everything that can be built and tested before a Meta app and official CraftSky Instagram professional account are configured. Live Meta dashboard setup, credential-backed calls, app review, and the real end-to-end capability spike remain release blockers rather than coding blockers.

Follow-on request (2026-07-23): extend the mobile import flow to accept
Instagram `.zip` exports as well as the standalone `following.json`. Large ZIPs
must be accessed from disk with the `archive` package rather than loaded
wholesale, and archive inspection/parsing must run in a background isolate.
The UI should recommend exporting only accounts the member follows while still
handling a full "all information" export safely.

Follow-on request (2026-07-27): replace reviewable Instagram match suggestions
with automatic CraftSky follows. Both initial and future eligible matches shall
be followed by AppView, with a per-account notification that identifies the
followed profile and offers the ordinary Follow/Following control. Remove the
People You May Know surface, default the import selector to Instagram export,
use user-facing "verified" rather than "link" terminology, use the themed green
success colour for enabled discovery, and move verification revocation to the
bottom of the page. A manual unfollow suppresses any later automatic re-follow
while the current verification/import evidence remains.

## 2. Current Codebase Findings

- Relevant AppView files:
  - `appview/internal/routes/routes.go` and `appview/internal/routes/policy.go` register authenticated `/v1/*` routes and apply bearer authentication, device-ID validation, body limits, rate limits, recovery, and observability.
  - `appview/internal/app/config.go`, `appview/internal/app/deps.go`, and `appview/environments/prod.env.example` own configuration, dependency injection, and production secret documentation.
  - `appview/internal/push/dispatcher.go` provides the durable Postgres worker pattern: leased jobs, `FOR UPDATE SKIP LOCKED`, bounded provider calls, retry/backoff, and bounded shutdown.
  - `appview/internal/api/follow.go`, `appview/internal/followwrite/service.go`,
    and `appview/internal/api/follow_store.go` provide the existing PDS
    follow-write, deterministic `putRecord`, and indexed-follow read paths.
  - `appview/internal/instagram/suggestions.go`,
    `suggestion_store.go`, `matcher.go`, and `reconciliation.go` currently
    create/list/accept suggestions and already contain a durable deterministic
    follow-operation ledger that can become an internal automatic-follow
    queue.
  - `appview/internal/notifications/`, `appview/internal/api/notification_store.go`, `appview/internal/push/payload.go`, and migrations `000021`/`000022` implement durable actor-driven notifications, preferences, push fan-out, and newness.
  - Background PDS writes need an owner-scoped OAuth session. The current
    request path receives an exact OAuth session ID from authentication
    middleware; a background worker will need a bounded selector for the most
    recently active, unrevoked stored session belonging to that same DID.
  - `appview/internal/index/craftsky_profile.go` owns current CraftSky membership removal; `appview/internal/notifications/actor_deletion.go` is a narrower deletion lifecycle precedent.
  - The implemented Instagram private schema uses migration
    `000025_instagram_migration`; the ZIP extension requires no migration.
- Relevant Flutter files:
  - `app/lib/settings/pages/settings_page.dart` is the natural entry point for **Find people from Instagram**.
  - `app/lib/router/router.dart` and `app/lib/router/route_locations.dart` own typed, deep-linkable routes.
  - Feature data layers use an API client, repository interface, API repository, Riverpod providers, and `dart_mappable` models.
  - Account-sensitive work must use a fixed-account Dio client or an active-account operation lease so polling, imports, and follow actions cannot cross account switches.
  - `app/lib/profile/data/profile_repository.dart` and
    `app/lib/notifications/widgets/notification_row.dart` expose the existing
    follow/unfollow operation and notification-row Follow/Following control.
  - Notification decoding, rendering, settings, push-open inference, and navigation are spread across `app/lib/notifications/`; all currently assume actor-driven social notifications.
  - The original baseline had no general file picker or Instagram import
    parser; the implemented flow now has a direct `file_selector` dependency.
  - The implemented mobile import path now accepts standalone JSON and
    file-backed ZIP exports, isolates parsing, and sends only normalized
    following handles to AppView.
  - `archive` 4.0.9 is already present transitively, but ZIP support requires
    it as a direct dependency. Its `InputFileStream` plus
    `ZipDecoder.decodeStream` keeps archive payloads on disk and lazily reads
    entry contents.
- Existing patterns:
  - Private-by-intent data belongs in AppView Postgres. A public
    `app.bsky.graph.follow` belongs on the PDS; the approved import action will
    now be the informed consent that authorizes automatic matching writes.
  - `/v1/*` JSON uses camelCase and the standard `{error, message, requestId}` error envelope.
  - Meta callbacks are external integration routes and must not be placed under `/v1/*`.
  - The app never holds Meta credentials or PDS credentials.
- Current behavior:
  - The completed feature has Instagram verification, private links/imports,
    reviewable suggestions, actorless digest `instagramMatch` notifications,
    and migration UI.
  - No general private-data export or member-initiated account-deletion endpoint exists yet.
  - The completed Flutter flow accepts manual handles and bounded standalone
    JSON/ZIP exports. It defaults to manual entry, shows a People You May Know
    review card, renders the discoverability switch with the theme default,
    and keeps revocation inside the verification card.
  - The approved real sample ZIP is 3.6 MiB with 74 entries. Its target
    `connections/followers_and_following/following.json` is 17,295 bytes and
    contains 88 records using `title` plus
    `https://www.instagram.com/_u/<username>` `href` values; it contains no
    `string_list_data[].value` fields.
- Constraints discovered:
  - The Meta capability spike remains mandatory because dashboard access, Live-mode behavior, webhook delivery from unrelated personal accounts, token lifecycle, and profile lookup cannot be verified without an app and owned professional account.
  - Meta's current official API collection documents `instagram_business_basic` and `instagram_business_manage_messages`, an IGSID in `messaging.sender.id`, profile lookup by that IGSID, and messaging through `graph.instagram.com`; all upstream shapes remain isolated behind tested adapters.
  - Local rate limiting is process-local and lacks direct IP/IGSID/global keys. Instagram abuse controls require an integration-specific shared/persistent limiter or an explicitly single-instance pre-production limitation.
  - The raw Instagram archive, webhook message history, handles, IGSIDs, challenges, and Meta secrets must not enter logs, Sentry, metric labels, push payloads, or PDS records.
- Test/build commands discovered:
  - AppView focused tests: `go test ./internal/...` from `appview/` when no database is needed; database-backed tests use the compose Postgres and `testdb.WithSchema`.
  - AppView broad gate: `just test` and `just fmt` from the repository root.
  - Flutter generation: `dart run build_runner build --delete-conflicting-outputs` and `flutter gen-l10n` from `app/`.
  - Flutter focused gate: `flutter test test/instagram_migration test/notifications test/router test/settings`.
  - Flutter broad gate: `flutter analyze` and `flutter test`.

## 3. Clarifying Questions And Decisions

### Q1: Is the proposed high-risk design approved for formalization and implementation?

Answer: Yes. The user explicitly approved treating `design-plan.md` and its settled product decisions as the implementation basis, creating the missing workflow artifacts, and implementing all feasible phases.

Decision / implication: This approval covers authentication-adjacent, private social-graph, webhook, migration, notification, and identity-linking changes within this requirement set. It does not authorize a commit, push, production enablement, or creation/configuration of a Meta app.

### Q2: How should a suggestion be accepted when the existing follow route and firehose index are not atomic?

Answer: Use a dedicated authenticated suggestion-accept operation that internally reuses a single extracted follow service. The operation is idempotent by suggestion ID and a stable follow operation key, writes the PDS follow only after explicit member action, records acceptance only after a successful or already-satisfied follow, and remains safely retryable across a firehose delay.

Decision / implication: Flutter does not perform an uncoordinated two-request “follow then mark accepted” sequence. No PDS follow is created while importing or merely viewing suggestions.

Superseded by Q9: the same durable follow-operation mechanism is retained, but
import creation is now the informed authorization and there is no
member-facing accept step.

### Q3: How should actorless `instagramMatch` preferences remain compatible with the existing preference wire shape?

Answer: `instagramMatch` is a first-class actorless system category. Its server-owned scope is fixed to `everyone` only for wire/storage compatibility, the Flutter settings UI hides the actor-scope control and explains migration eligibility, and PATCH attempts to change its scope are rejected. `pushEnabled` remains user-configurable.

Decision / implication: The category is never represented as `everythingElse`, never uses a synthetic actor, and never treats `everyone` as an eligibility decision.

Superseded by Q9 for notification payload/rendering only: the fixed preference
contract remains, while each successful automatic follow now produces an
actorful row identifying the followed profile.

### Q4: How should conflicting link claims warn both affected members before a full support workflow exists?

Answer: Persist a private conflict/audit record and expose a generic warning on the claimant's attempt and the existing owner's account-link status. Do not send the existing owner the claimant's DID, handle, username, IGSID, or challenge, and do not add another push category in this slice. Operator CLI tooling may inspect opaque link/conflict IDs and resolve or revoke links after manual support review.

Decision / implication: The current link remains authoritative; no automatic reassignment occurs. A future general security-notification design may add proactive delivery without changing link ownership semantics.

### Q5: What import formats are in the first implementation?

Answer: The initial implementation supported manual text and selected Accounts
Center JSON files containing accounts the member follows, parsed on-device.
ZIP selection and decompression were deferred until real export fixtures
justified a stable, bounded implementation. Follower data is not needed for
the approved discovery model, is ignored locally when present beside following
data, and never crosses the repository/API boundary. Follower-only and unknown
shapes fail locally with guidance. Q7 supersedes only the deferred ZIP portion
after an approved real export became available.

Decision / implication: The client adds a direct `file_selector` dependency but not an archive dependency. The raw selected bytes and decoded JSON never cross the repository/API boundary.

### Q6: How should verification, discoverability, import retention, and future-match notifications be simplified?

Answer: Import controls are hidden until the member has a verified Instagram
account. Confirmation defaults to discoverable, while still allowing the member
to choose private. Creating an import always retains every normalized following
handle until Instagram verification is revoked; there is no separate retention
consent, expiry, renewal, or withdrawal control. Every newly eligible future
match creates the normal private in-app `instagramMatch` notification. Push
delivery remains configurable from Notification Settings, which is linked from
the import UI.

Decision / implication: Import creation itself is the retention action. The
AppView rejects imports without a verified account, removes retention fields from
the wire and storage model, and deletes all owner imports and their dependent
pending discovery state when Instagram verification is revoked. Previously
accepted PDS follows remain unchanged.

### Q7: How should mobile ZIP exports and the observed current following shape be supported?

Answer: Accept standalone JSON and Instagram ZIP exports on mobile. ZIP access
shall use the `archive` package's file-backed streaming API in a background
isolate, locate only the canonical following JSON entry, and never extract the
archive to disk or read unrelated entries. For the observed shape, derive the
username only from the exact
`https://www.instagram.com/_u/<username>` URL grammar and require a present
record `title` to normalize to the same username; never accept an arbitrary
title as username evidence.

Decision / implication: ZIP support remains local to Flutter. Both standalone
JSON and ZIP imports use the existing `instagramJson` AppView source type and
are labelled "Instagram export" in the UI. AppView API/storage do not change.
ZIP support targets iOS and Android (and may work on file-backed desktop
builds); Flutter web ZIP support is out of scope because it cannot provide the
same native file-streaming/isolate contract.

### Q8: How should notification payload families be discriminated?

Answer: Use the existing notification `type` as the sole wire and storage
discriminator. `instagramMatch` is an actorless type with a required `system`
payload; every existing social type retains its required actor and AT Protocol
source fields. Do not add or retain a separate `kind` field that is derivable
from `type`.

Decision / implication: Flutter selects the concrete notification model from
`type`, AppView omits `kind` from notification JSON, and Postgres constraints
enforce the two payload shapes directly from `category`. Unknown types remain
inert and must not gain an identity-bearing destination merely because optional
fields are present.

Superseded by Q9 for `instagramMatch`: it is actorful but still uses `type` as
the sole discriminator and still omits a separate `kind` or destination field.

### Q9: How should automatic follows, notifications, and manual unfollows behave?

Answer: Use a durable AppView pipeline for both initial and future eligible
matches. Matching creates an internal idempotent operation rather than a
member-facing suggestion. A background worker revalidates the complete safety
policy, chooses only a current stored OAuth session for the importing DID,
writes one deterministic PDS follow, and creates an actorful `instagramMatch`
notification only after that write succeeds. The notification opens the
matched profile and reuses the ordinary Follow/Following control. A member who
unfollows that account shall not be automatically re-followed while the same
Instagram verification/import evidence remains. Revoking verification deletes
the private evidence and suppression ledger; a later verification plus fresh
import is new consent and may follow the account again.

Decision / implication: Import copy must disclose that matching accounts are
followed publicly. Existing member-facing suggestion list/accept/dismiss routes
and Flutter state are removed. Existing internal suggestion/follow-operation
storage may be retained as an implementation ledger, but it is not a public
review workflow. Temporary PDS/session failure is retryable; already-followed
targets are recorded without emitting a false auto-follow notification.

## 4. Candidate Approaches

### Option A: Direct Meta Integration With Durable Private AppView State

Summary: AppView issues hashed challenges, accepts signed webhook deliveries into a durable inbox, resolves candidate usernames through a narrow Meta adapter, requires same-DID in-app confirmation, and owns private verification mappings/imports/automatic-follow operations. Flutter parses exports locally and sends only normalized entries.

Pros:

- Preserves the selected privacy and identity boundaries.
- Avoids third-party automation contact retention and pricing.
- Makes webhook replay, conflict, retention, deletion, and audit semantics testable.
- Allows nearly all logic to be completed using fake Meta adapters before credentials exist.

Cons:

- Requires migrations, workers, new API routes, extensive notification changes, and Flutter UI/parser work.
- Carries an operational burden for Meta API/token changes.

Risks:

- The final external contract can only be proven with a configured Meta app and unrelated personal sender.
- Cross-network identity linking can expose a member if discoverability or conflict behavior is wrong.

### Option B: ManyChat Or Another Automation Adapter

Summary: A third party receives Instagram DMs and calls AppView with contact data.

Pros:

- Faster dashboard prototype.
- Less webhook infrastructure initially.

Cons:

- Adds a data processor, duplicated contact retention, vendor pricing, and a critical dependency.
- May not expose the raw stable IGSID needed for a durable identity anchor.

Risks:

- Vendor contact semantics or privacy behavior can undermine verification assurance.

### Option C: Export Possession Or Instagram Bio As Proof

Summary: Treat an export or temporary public bio value as account ownership evidence.

Pros:

- Avoids the Messaging API.

Cons:

- Export archives are copyable and stale; bio checks rely on unsupported or fragile profile reading.

Risks:

- Produces weaker ownership claims and possible false identity links.

### ZIP Extension Option A: Stream Only The Target Entry In An Isolate

Summary: Pass the selected native file path to a background isolate, inspect
ZIP metadata through bounded file reads, use `archive` file-backed decoding,
and decompress only the canonical following JSON entry.

Pros:

- Large media/message content stays on disk and is never inspected.
- UI responsiveness and the existing local-only privacy boundary are
  preserved.
- Standalone JSON and ZIP can share one normalized parser result.

Cons:

- Requires native conditional wiring and explicit malformed/archive-bomb
  limits.

Risks:

- ZIP metadata and export shapes can drift, so exact bounds and fixture tests
  are required.

### ZIP Extension Option B: Read The Whole ZIP Into Memory

Summary: Keep the current byte-based picker and pass all ZIP bytes to
`ZipDecoder.decodeBytes`.

Pros:

- Smallest code change.

Cons:

- Full Instagram exports can be dominated by media and exceed device memory.
- Copies unrelated private content between the UI isolate and parser.

Risks:

- UI stalls or process termination on otherwise valid exports.

### ZIP Extension Option C: Extract The Archive To A Temporary Directory

Summary: Expand every entry to temporary storage and then open `following.json`.

Pros:

- Simple target-file lookup after extraction.

Cons:

- Writes unrelated messages/media to app-controlled storage and needs cleanup.
- Increases I/O, disk usage, path-traversal surface, and privacy exposure.

Risks:

- Partial extraction or cleanup failure leaves sensitive data behind.

### Auto-Follow Option A: Durable AppView Match And Follow Workers

Summary: Matching records a private deterministic follow operation. A bounded
worker selects an owner-scoped stored OAuth session, revalidates eligibility,
writes the PDS follow idempotently, then creates one actorful notification.

Pros:

- Handles initial and future matches while the app is closed.
- Keeps PDS writes retryable and idempotent across crashes and firehose delay.
- Reuses the existing private operation ledger and ordinary follow service.
- Supports durable manual-unfollow suppression without retaining raw exports.

Cons:

- Requires background OAuth-session selection and a checked actorful
  notification variant.
- The follow may be delayed while no usable OAuth session exists.

Risks:

- A policy/session-selection bug could write under the wrong account; owner
  predicates, exact DID matching, final revalidation, and deterministic tests
  are mandatory.

### Auto-Follow Option B: Synchronous Import-Time Writes

Summary: Write each initial match during the import request and retain the
existing reconciliation worker only for future matches.

Pros:

- Initial matches appear immediately.

Cons:

- A large import can require thousands of external writes and exceed request
  timeouts.
- Initial and future matches use different correctness/retry paths.

Risks:

- Partial request failure can leave unclear progress and encourage duplicate
  writes.

### Auto-Follow Option C: Flutter-Driven Follow Loop

Summary: Return matches to Flutter and let the app call the ordinary follow
route for each one.

Pros:

- Reuses the current authenticated request session directly.

Cons:

- Works only while the app is open and cannot implement future matching
  reliably.
- Keeps the obsolete suggestion payload and client orchestration.

Risks:

- Account switching, interruption, and retry can produce partial or
  cross-account behavior.

## 5. Recommended Direction

Recommended approach: Option A for the integration, ZIP Extension Option A for
archive import, and Auto-Follow Option A for matches: the direct Meta
integration keeps its disabled-by-default adapter and private on-device/AppView
boundaries; selected mobile ZIPs remain file-streamed in an isolate; every
eligible initial or future match enters one durable owner-scoped automatic
follow pipeline.

Why: This preserves live control evidence, stable IGSID anchoring,
discoverability consent, minimal imported data, deterministic PDS writes, and
auditable retries without requiring the app to remain open. The import action
becomes the explicit consent for automatic public follows, while manual
unfollow remains authoritative for the lifetime of that private evidence.

## 6. Problem / Opportunity

People moving from Instagram cannot currently recreate the CraftSky portion of
their chosen following graph without repeatedly reviewing matches, unreliable
name matching, scraping, or publicizing their imported social graph. DM-based
proof establishes high-confidence handle ownership, while on-device export
parsing and private AppView matching can safely create the corresponding
CraftSky follows without uploading raw archives.

## 7. Goals

- G-001: Let a signed-in CraftSky member prove current control of one Instagram account through a short-lived DM challenge and same-DID in-app confirmation.
- G-002: Keep the IGSID as the stable identity anchor and the normalized username as a mutable verified attribute.
- G-003: Let members choose handle-based CraftSky discoverability while confirming the displayed account, change that setting later, and revoke their verification.
- G-004: Parse manual lists and supported standalone JSON or ZIP Instagram
  exports on-device and send only minimal normalized relationship entries.
- G-005: Automatically create exact, privacy-filtered CraftSky follows for
  current discoverable matches from both new imports and retained handles.
- G-006: Make import an informed authorization for automatic follows, while
  respecting manual unfollow as suppression for the current evidence lifetime.
- G-007: Retain imported following handles until Instagram verification is
  revoked, and notify importers through a first-class private actorful
  `instagramMatch` item after each successful automatic follow.
- G-008: Complete fixture-driven AppView and Flutter implementation before Meta credentials exist, while failing closed in unconfigured runtime environments.

## 8. Non-Goals

- NG-001: Read follower or following lists through the Instagram API.
- NG-002: Scrape Instagram pages or treat export possession as ownership proof.
- NG-003: Upload or server-parse raw Instagram JSON, ZIPs, media, messages, biographies, photos, counts, or unrelated export data.
- NG-004: Write Instagram identity, verification, import, or raw graph data to
  a PDS; only the resulting ordinary `app.bsky.graph.follow` record is public.
- NG-005: Collect, upload, persist, or create suggestions from accounts that follow the importing member.
- NG-006: Send marketing, future-match, or follow-suggestion DMs through Instagram.
- NG-007: Use ManyChat or another automation SaaS.
- NG-008: Add a new AT Protocol lexicon.
- NG-009: Undo successful PDS follows when verification, username, import, or
  discoverability changes.
- NG-013: Automatically re-follow a member who was manually unfollowed while
  the same Instagram verification/import evidence remains.
- NG-010: Enable the production integration before the Meta capability spike, secret provisioning, dashboard configuration, privacy-policy/data-deletion requirements, and required access review are complete.
- NG-011: Support ZIP import on Flutter web, load an entire ZIP into memory, or
  extract unrelated archive entries to device storage.
- NG-012: Build the repository-wide member data-export/account-deletion API that does not currently exist; this change supplies scoped purge/export primitives and schema cascades for future composition.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Verifying member | Signed-in current CraftSky member verifying an Instagram account. | Secure challenge, clear consent, accurate candidate confirmation, revocation, and conflict recovery. |
| Instagram sender | Personal or professional Instagram account sending the challenge. | Generic responses that do not leak whether another member's challenge exists. |
| Importing member | Signed-in member supplying following handles. | Local parsing, privacy guarantees, informed automatic follows, notifications, manual-unfollow control, and deletion/retention choices. |
| Matched member | Member with an active discoverable verified account mapping. | Control over discoverability and no disclosure of who imported/searched for them. |
| Meta | External webhook and profile/messaging provider. | Signed callback delivery and server-held access credentials. |
| AppView | Private system of record and PDS write mediator. | Idempotent state transitions, strict authorization, durable work, deletion, and bounded observability. |
| Flutter app | Account-scoped client and local parser. | Typed APIs, fixed-account operations, local-only raw data, explicit user actions, and safe navigation. |
| CraftSky operator | Human handling exceptional disputes. | Opaque audit/conflict inspection and explicit non-automatic resolution/revocation tools. |

## 10. Current Behavior

The implemented flow supports DM verification, private matching, manual and
standalone JSON/ZIP imports, reviewable suggestions, and actorless future-match
digests. An import creates pending suggestions; only explicit member acceptance
writes a PDS follow. Flutter defaults to manual entry, renders People You May
Know, uses the theme's current light-red selected switch treatment, and keeps
revocation inside the account card.

## 11. Desired Behavior

When configured, AppView creates a ten-minute, single-use, DID-bound challenge
and returns only its display value, verification ID, expiry, and official
Instagram DM URL. A valid signed Meta message event is acknowledged quickly and
durably queued. Background processing deduplicates the message ID, recognizes
only verification text, finds the attempt by a keyed challenge digest, fetches
only the sender's current username when needed, and transitions the attempt to
pending confirmation. The creating DID confirms the actual username in-app
before an active verified mapping exists. Discovery remains a separate
explicit choice.

Flutter offers **Find people from Instagram** in Settings. It supports
verification status/confirmation, discoverability/revocation, manual handles,
and selected Instagram exports in standalone JSON or ZIP form. Import controls
remain hidden until ownership is verified and Instagram export is the default
input. The UI explicitly explains that imported matches are automatically
followed on CraftSky, recommends requesting only accounts followed, and handles
an all-information ZIP without reading unrelated entries. Native file
inspection, extraction of the one canonical following JSON entry, JSON
parsing, size/shape validation, and normalization remain on-device in a
background isolate.

AppView receives only normalized entries, retains them for the verification
lifetime, queues exact eligible current/future matches, and uses a durable
background operation to revalidate and write deterministic PDS follows through
the importing DID's stored OAuth session. A successful automatic follow creates
one actorful notification with the ordinary Follow/Following action. There is
no member-facing suggestion review surface. A manual unfollow suppresses later
automatic re-follow for the same retained evidence. Unconfigured environments
remain gracefully unavailable and never accept unsigned or partially
configured live traffic.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | A current signed-in CraftSky member shall be able to prove control of one Instagram account through a live DM challenge plus same-DID in-app confirmation. | Provides high-confidence cross-network ownership evidence. | Design / User approval | AC-001–AC-008, AC-014, AC-015, AC-048 |
| BR-002 | Business | Must | Instagram graph migration shall preserve privacy by parsing raw exports on-device and retaining only the minimum normalized private data needed for matching. | Raw exports contain unrelated sensitive data. | Design / User approval | AC-016–AC-019, AC-039, AC-054 |
| BR-003 | Business | Must | Importing following handles shall authorize AppView to automatically create ordinary CraftSky follows for exact currently eligible matches and future matches, without a separate per-match review step. | Recreates the member's chosen following graph with substantially less friction while keeping imported evidence private. | User request / User approval | AC-020–AC-025, AC-029, AC-055 |
| BR-004 | Business | Must | Members shall control verification discoverability, verification revocation, import deletion, push delivery for automatic-follow notifications, and later follow/unfollow state. Imported handles and automatic re-follow authority shall exist only while the verified Instagram account remains verified. | Cross-network discovery and automatic public writes require informed, reversible controls. | User request / User approval | AC-009, AC-024–AC-031, AC-034, AC-056 |
| FR-001 | Functional | Must | AppView shall expose a disabled-by-default Instagram integration configuration whose enabled state requires a complete validated secret/account/API bundle. Missing Meta configuration or an upstream outage shall disable only new verification/profile/reply work; local import, existing verification status/disable/revoke, import retention/deletion, queued automatic-follow processing, and privacy controls shall remain available when their own dependencies are healthy. | Prevents accidental unsigned exposure without turning an external outage into a private-data lockout. | Design / Codebase / Document review | AC-001, AC-040 |
| FR-002 | Functional | Must | `POST /v1/migrations/instagram/verifications` shall create one opaque DID-bound attempt, supersede any earlier active attempt for that DID, and return a display challenge, expiry, verification ID, and server-configured HTTPS DM URL without returning server secrets or stored digests. | Starts the verification flow safely. | Design | AC-002, AC-003 |
| FR-003 | Functional | Must | Authenticated current members shall inspect only their own verification attempts. `GET /v1/migrations/instagram/verifications/current` shall return the caller's one non-terminal attempt or `{verification: null}` after expiring stale work, without requiring a client-retained ID. Cancellation shall mutate only an owned attempt but return the same idempotent `204` for owned, absent, or foreign IDs. Status reads shall use the exact public attempt-state contract in §12.1 plus only the candidate username and retry code needed for explicit confirmation/recovery. | Lets an interrupted client resume safely while preventing cross-member attempt access and wire-state drift without making DELETE an existence oracle. | Design / Document review / User approval | AC-004, AC-005, AC-048, AC-049 |
| FR-004 | Functional | Must | `GET /integrations/instagram/webhook` shall implement Meta callback verification using exact expected mode/token semantics and shall reveal the supplied challenge only on a valid request. | Required external callback setup. | Design / Meta contract | AC-006 |
| FR-005 | Functional | Must | `POST /integrations/instagram/webhook` shall enforce the §12.4 body/event limits, verify `X-Hub-Signature-256` over exact raw bytes before decoding, accept only the configured official account and supported message events, and use the exact ingress/valid-sender throttling responses in §12.4. Invalid signatures or unsupported bodies receive only a generic failure. | Establishes the external trust boundary and predictable Meta retry behavior. | Design / Security review / Re-review | AC-007, AC-008, AC-041 |
| FR-006 | Functional | Must | Valid webhook messages shall be deduplicated by a keyed digest of Meta message ID and persisted as the exact minimal work item in §12.2: message-ID digest, sender IGSID, official-account ID, keyed normalized-challenge digest, event timestamp, and job/lease/retry fields. Raw bodies, message text, plaintext challenges, and unrelated payload fields shall not be persisted. The handler shall acknowledge only after the durable transaction and without waiting for profile lookup/reply; a leased bounded-retry worker shall process the item. | Meta delivery is at-least-once and slow work must not block callbacks. | Design / Codebase / Document review | AC-008, AC-010, AC-011 |
| FR-007 | Functional | Must | Background processing shall ignore non-text, self/echo, non-verification, malformed, expired, cancelled, superseded, or already-redeemed messages without retaining message text; a valid challenge shall bind exactly one candidate IGSID and obtain only its current username through an injectable Meta profile client. | Minimizes retained DM data and prevents replay/rebinding. | Design / Meta contract | AC-010–AC-013 |
| FR-008 | Functional | Must | `POST /v1/migrations/instagram/verifications/{verificationId}/confirm` shall require the creating DID, a pending candidate, and an explicit confirmation carrying the displayed discoverability value; it shall be idempotent for the same result and create/update a verified link only after all uniqueness and conflict checks pass. | The DM alone must not finalize exposure. | Design / User approval | AC-014, AC-015 |
| FR-009 | Functional | Must | AppView shall enforce one active Instagram link per CraftSky DID, one active owner per IGSID, and one current owner claim per normalized username regardless of discoverability; collisions shall create a private conflict and shall never transfer ownership automatically. | Prevents identity takeover and hidden-link username reuse. | Design | AC-012, AC-015, AC-032 |
| FR-010 | Functional | Must | Authenticated account/link endpoints shall return the caller's own link state, allow discoverability changes, expose a generic pending-conflict warning, and revoke the link idempotently. | Makes consent and recovery visible and reversible. | Design / Route review | AC-009, AC-031, AC-032 |
| FR-011 | Functional | Should | The Meta adapter and operator refresh path should support observing a changed username for an existing IGSID, updating it only after validation, invalidating old-handle queued automatic-follow operations, and routing collisions to conflict handling. | Usernames are mutable while IGSIDs are the anchor. | Design | AC-033 |
| FR-012 | Functional | Must | `POST /v1/migrations/instagram/imports` shall require an active verified Instagram account and accept a bounded strict schema containing only source type and normalized usernames for accounts the member follows. It shall create a private import retained until verification revocation and return only the import and following count; it shall not accept retention, relationship direction, follower data, raw archive fields, or expose pending match/suggestion counts. | Defines the minimal server boundary and ties private graph lifetime to verification without exposing internal automatic-follow work. | Design / User approval | AC-016–AC-019, AC-026 |
| FR-013 | Functional | Must | Flutter shall parse manual text and supported Instagram following export shapes locally from bounded standalone JSON or native ZIP input, normalize/deduplicate only accounts the member follows, ignore follower sections/unrelated archive entries, and reject follower-only, missing-target, duplicate-target, unsupported, malformed, encrypted, or oversized input before any network request. | Enforces data minimization for the chosen discovery model while supporting the normal download container. | Design / Codebase / User approval | AC-016–AC-018, AC-050–AC-052 |
| FR-014 | Functional | Must | Handle normalization shall trim outer whitespace and at most one leading `@`, ASCII-case-fold, accept only `^[a-z0-9._]{1,30}$`, and deduplicate by normalized username. A legacy `string_list_data[].value` is direct username evidence. For the observed current shape, the username shall come only from the exact HTTPS URL grammar `https://www.instagram.com/_u/<username>` and a present record `title` must normalize to the same value. Arbitrary titles, display names, other hosts/paths/query strings/fragments, and mismatched title/URL pairs are invalid rather than fuzzy evidence. | Matching must be deterministic and exact while supporting the current export shape without treating display text as identity. | Design / User approval | AC-019, AC-020, AC-051 |
| FR-015 | Functional | Must | Match discovery, operation creation, final PDS write, and notification creation/delivery shall use one `InstagramSuggestionEligibilityPolicy`: both importer and target are current CraftSky members; the verified mapping is active, DM-verified, discoverable, exact-current-username, and conflict-free; target is not self; no effective account hide/takedown applies; no active block exists in either direction; and the importer has not muted the target. A target already followed at initial operation creation is excluded; if it becomes followed before the write, the operation completes as `alreadyFollowing` without an automatic-follow notification. An unavailable required relationship-safety source fails closed outside explicit tests. | Prevents low-confidence or unsafe automatic follows and policy drift between worker boundaries. | Design / Codebase review / User approval | AC-020–AC-022, AC-025, AC-031, AC-048 |
| FR-016 | Functional | Must | Member-facing Instagram suggestion list, accept, and dismiss routes and Flutter models/providers/UI shall be removed. Internal match/support/operation rows shall remain owner-scoped implementation facts and shall never be exposed as a pending review surface. | There is no member decision left for a People You May Know workflow once imports authorize automatic following. | User request / API architecture | AC-021, AC-023, AC-038 |
| FR-017 | Functional | Must | Each newly eligible importer/target pair shall claim one durable owner-scoped automatic-follow operation with a stable PDS rkey before any external call. A bounded worker shall re-evaluate `InstagramSuggestionEligibilityPolicy` immediately before writing, use the shared CraftSky follow service and exact owner OAuth session selected by FR-032, and mark success only after deterministic PDS `putRecord` success. Temporary session/PDS failure remains retryable; a pre-existing follow ends as `alreadyFollowing` without a duplicate write or automatic-follow notification. | Makes background public writes safe across retries, crashes, and firehose delay. | User request / Codebase review / User approval | AC-024, AC-025, AC-029, AC-048, AC-055, AC-056 |
| FR-018 | Functional | Must | Imports are additive immutable source snapshots retained while the owner keeps Instagram verified. AppView shall list and inspect caller-owned imports, delete one import immediately, and PATCH only explicit `reactivate` after membership restoration. Internal matches are deduplicated per importer/target but retain support references to every active import. Deleting one source cancels an unwritten operation only when no other eligible source supports it; successful PDS follows and their historical notifications remain. Revoking verification deletes every owner import and cancels dependent unwritten work. | Makes the lifetime obvious and keeps source-specific deletion/restoration durable without rewriting public graph history. | Design / User approval | AC-026–AC-028, AC-031, AC-048 |
| FR-019 | Functional | Must | Initial import, verification confirmation/discovery enablement/reactivation, verified username change, membership restoration, and visibility/safety-policy restoration shall re-evaluate eligible retained following handles and enqueue deduplicated automatic-follow operations. Successful operations shall create the per-account `instagramMatch` notification in §12.3. | Enables immediate and future graph restoration without a review queue. | User request / User approval | AC-029, AC-030, AC-056 |
| FR-020 | Functional | Must | `instagramMatch` shall be an explicit actorful automatic-follow notification `type` in §12.3 across schema, registry, preference API, feed API, newness, push outbox, Flutter decoding/rendering, localization, and navigation. `type` shall remain the sole wire/storage discriminator; no derivable `kind` or client-route destination field is exposed or stored. It shall identify the followed target as its actor but shall not pretend that target performed the notification action or require synthetic AT Protocol source fields. Unknown types remain inert without an identity-bearing destination. | The notification must show the followed account and follow state while honestly representing a system-triggered action. | User request / User approval | AC-029, AC-034–AC-037 |
| FR-021 | Functional | Must | The `instagramMatch` preference shall expose fixed `scope: everyone` for compatibility, reject scope mutation, allow `pushEnabled` changes, and hide the scope control in Flutter with an automatic-follow explanation. | Ordinary incoming-actor scope is semantically irrelevant because eligibility comes from the member's private import and safety policy. | Design / Contract decision | AC-034 |
| FR-022 | Functional | Must | Push data for `instagramMatch` shall contain only the opaque account-subscription binding, category, and stable notification ID; it shall contain no client-route destination, handle, IGSID, DID, challenge, import, or match data. Flutter shall infer from the category that a push opens the correct account's Notifications feed; selecting the hydrated row opens the matched profile. | Protects private graph data at the provider boundary while still allowing actorful in-app rendering. | Design / Push architecture | AC-035–AC-037 |
| FR-023 | Functional | Must | Flutter shall retain a typed, authenticated **Find people from Instagram** route reachable from Settings with account-switch-safe navigation and state. `instagramMatch` no longer targets this route; its hydrated row targets the matched profile. | Keeps verification/import settings separate from per-account automatic-follow activity. | User request / Codebase | AC-038, AC-042 |
| FR-024 | Functional | Must | Flutter verification UI shall use user-facing "verified"/"verification" terminology instead of "linked"/"link", explain discoverability, create/copy/open a challenge, poll through a fixed-account client, show expiry/cancellation/unavailable states, display the actual candidate username, and support confirmation, cancellation, settings, and revocation. The active account summary shall read `Verified as @…`. The enabled discoverability switch shall use the theme's semantic success/moss green rather than the light-red selected treatment. Revocation shall appear after all import content at the bottom of the page and require a destructive confirmation explaining that imported handles and automatic-follow authority are deleted while existing CraftSky follows remain; cancellation shall preserve verification. Pending confirmation and secure resumption retain their existing account-first/default-discoverable contract. | Uses clearer terminology and hierarchy while preserving safe resumable verification and destructive-action consent. | User request / User approval | AC-003, AC-005, AC-009, AC-014, AC-038, AC-049 |
| FR-025 | Functional | Must | Flutter shall hide all import controls until a verified Instagram account exists. The input selector shall default to `Instagram export` while retaining manual entry. The import UI shall support local `.json`/`.zip`, recommend exporting only accounts followed, explain that unrelated all-information ZIP content stays local, explicitly disclose that current and future eligible matches are automatically followed publicly on CraftSky, disclose retention until verification revocation, upload only normalized entries, list/delete imports, explicitly reactivate each eligible import after rejoin, contain no People You May Know section, and link to Notification Settings for push control. | Makes the automatic behavior understandable while preserving the simplest normal export path. | User request / User approval | AC-018, AC-023, AC-026, AC-034, AC-038, AC-048, AC-050 |
| FR-026 | Functional | Must | Flutter shall render `instagramMatch` as an actorful row with localized automatic-follow copy, avatar/profile identity, current Follow/Following state, and the existing account-scoped follow/unfollow control. Tapping the row opens that profile. Optimistic toggles, refreshes, navigation, and late results shall remain fenced to the notification owner account. | Gives members immediate visibility and ordinary control over each automatic follow. | User request / Multi-account architecture | AC-024, AC-025, AC-035, AC-037, AC-042 |
| FR-027 | Functional | Should | The worker should send bounded immediate accepted/expired/invalid/completed DM replies only when allowed and configured, with idempotent reply state and no later marketing or match messages. | Improves flow feedback without making replies correctness-critical. | Design | AC-043 |
| FR-028 | Functional | Must | Instagram private data shall have reusable export, membership-inactivation, and terminal-purge services. Loss of `craftsky_profiles` membership shall set verified mappings `membershipInactive`, disable discovery, pause owner imports, cancel dependent unwritten automatic-follow operations, and block member-facing operations without deleting owner data. Rejoining requires explicit owner reactivation and never silently restores discoverability. A terminal atproto identity-deletion event, future explicit whole-account deletion, or scoped user delete shall permanently purge the applicable private data, cancel unsent work, and leave successful PDS follows untouched. | Separates a reversible membership boundary from permanent deletion. | Design / Codebase review / Document review | AC-028, AC-031, AC-044, AC-048 |
| FR-029 | Functional | Should | Operator CLI tooling should list opaque unresolved conflicts, revoke links, retry/inspect bounded job state, run retention for other terminal private records, and resolve a conflict only through an explicit audited action that never silently transfers ownership. | Supports exceptional recovery before a full admin UI. | Design | AC-032, AC-045 |
| NFR-001 | Non-functional | Must | Challenges shall use a cryptographically secure source and at least 60 bits of entropy after formatting, omit ambiguous characters, be stored only as a keyed one-way digest, and never encode member data. | Resists guessing and disclosure. | Design / Security | AC-002, AC-012 |
| NFR-002 | Non-functional | Must | Challenge creation/redemption, invalid messages, profile lookup, confirmation, imports, and global webhook volume shall use the persistent/shared defaults and hard maxima in §12.4. Source IP shall come from a configured trusted-proxy/edge policy and shall never trust arbitrary forwarding headers. Production multi-instance enablement requires shared enforcement. | Limits abuse and downstream exhaustion. | Design / Codebase review / Document review | AC-041, AC-046 |
| NFR-003 | Non-functional | Must | Logs, errors, spans, Sentry, and metric labels shall never contain challenge plaintext/digests, webhook bodies/message text, usernames, IGSIDs, imported handles/lists, Meta tokens/secrets, signature headers, or upstream response bodies. | These values are sensitive or identifying. | Design / Observability review | AC-039 |
| NFR-004 | Non-functional | Must | All authenticated routes shall use the existing auth/device-ID middleware, the shared current-member guard, camelCase JSON, strict request decoding, route policy inventory, bounded bodies, and standard error envelopes; caller ownership shall come only from the authenticated DID and current membership shall return `404 profile_not_found` when absent. | Preserves the v1 API and membership contracts. | API architecture / Document review | AC-004, AC-005, AC-040, AC-048 |
| NFR-005 | Non-functional | Must | Webhook, matching, automatic-follow, and notification state transitions shall be idempotent under duplicate, replayed, out-of-order, and concurrent work, with deterministic PDS rkeys, transactions, leases, and database constraints enforcing invariants. | External delivery and worker execution are at-least-once and concurrency is adversarial. | Design / User approval | AC-008, AC-010, AC-015, AC-025, AC-029, AC-055 |
| NFR-006 | Non-functional | Must | Meta HTTP calls shall use a narrow injectable client, explicit API version/base URL, bounded timeouts, response-size limits, retry classification, and server-held bearer tokens; Flutter shall have no Meta credential configuration. | Contains upstream drift and secrets. | Design / Meta contract | AC-011, AC-040 |
| NFR-007 | Non-functional | Must | All Instagram UI, models, parsing, and controls shall be localized, accessible, and usable in loading, empty, disabled, error, and retry states. | The feature handles sensitive consent and failure states. | Flutter conventions | AC-038 |
| NFR-008 | Non-functional | Must | Fixed-account clients/operation leases shall fence every polling response, mutation, notification follow toggle, navigation, and cache update so an account switch cannot expose or mutate another account's Instagram state. Any resumable verification snapshot shall remain secure-storage-backed and keyed by CraftSky DID under the existing reconciliation/cleanup contract. | Preserves multi-account isolation while allowing bounded page-level resumption. | Codebase / Approved multi-account contract / User approval | AC-042, AC-049, AC-053 |
| NFR-009 | Non-functional | Must | Client cancellation shall remain classified as internal 499/canceled and shall not be captured as a server failure or Sentry event. | Account switching and polling cancellation are expected client behavior. | Existing observability contract | AC-047 |
| FR-030 | Functional | Must | A shared current-member guard shall protect every authenticated Instagram route and every worker transition that verifies, matches, auto-follows, or notifies. A still-valid CraftSky session whose DID is absent from `craftsky_profiles` receives `404 profile_not_found`; workers pause/reject the transition and invoke membership-inactivation behavior instead of surfacing FK/internal errors. | Current membership is a hard user-facing boundary independent of session validity. | Document review / Existing membership contract | AC-048 |
| FR-031 | Functional | Must | On iOS and Android, Flutter shall pass the selected native file path—not complete ZIP bytes—to a background isolate. That isolate shall distinguish JSON from ZIP, use the direct `archive` dependency with `InputFileStream` and streaming ZIP decoding, locate exactly one canonical `connections/followers_and_following/following.json`, decode only that entry, close all file/archive resources, and return only normalized entries plus bounded ignored/duplicate counts. It shall never extract archive contents to disk. | Keeps large exports off the UI isolate and prevents unrelated private data from entering memory or app-controlled storage. | User request / Codebase / Archive 4.0.9 API | AC-050, AC-052, AC-053 |
| FR-032 | Functional | Must | The automatic-follow worker shall select only an unrevoked stored OAuth session owned by the importing DID, preferring the most recently active usable session deterministically. It shall never borrow another DID's session. If no owner session is usable, the operation remains retryable and no PDS write occurs; an expired or otherwise invalid session is invalidated only for that exact account/session under the existing narrow session lifecycle. | Background PDS writes must preserve the same account and credential isolation as authenticated request-path writes. | Codebase review / Multi-account architecture | AC-055 |
| NFR-010 | Non-functional | Must | ZIP processing shall remain bounded independently of total media payload size: inspect no more than 100,000 central-directory entries and 64 MiB of central-directory metadata, require the one target entry's declared and actual uncompressed content to be at most 20 MiB, preserve the 10,000 normalized-entry cap, and fail locally without partial upload when any limit or archive-integrity check fails. No total ZIP byte cap is imposed solely because unrelated media makes an otherwise valid export large. | Prevents archive bombs/OOM while allowing legitimate all-information exports whose size is dominated by ignored media. | User request / Security analysis | AC-052, AC-053 |
| RULE-001 | Business rule | Must | A challenge is case-insensitive, single-use, valid for ten minutes, bound to one DID/attempt, and invalid after redemption, expiry, cancellation, or supersession. | Defines proof validity. | Design | AC-002, AC-003, AC-010, AC-012 |
| RULE-002 | Business rule | Must | The Instagram sender proves control at DM time, but only explicit confirmation by the same authenticated CraftSky DID creates the link and applies the discoverability value displayed at confirmation. | Separates proof from exposure consent. | Design / User approval | AC-014, AC-015 |
| RULE-003 | Business rule | Must | The IGSID is the identity anchor; a username is a mutable normalized attribute and shall never cause automatic ownership transfer between IGSIDs or DIDs. | Handles username changes safely. | Design | AC-015, AC-032, AC-033 |
| RULE-004 | Business rule | Must | Discovery is selected by default during account confirmation, is enabled only by the member's affirmative confirmation of that displayed choice, remains independently disableable, and is disabled for revoked, disputed, superseded, departed, or otherwise inactive links. | Keeps the confirmation choice visible and reversible while preventing a DM alone from exposing the identity. | Design / User approval | AC-009, AC-014, AC-020, AC-031 |
| RULE-005 | Business rule | Must | A high-confidence match is exact and current; display-name similarity, case variants after normalization, old usernames, and unverified mappings are not alternative evidence. | Avoids false identity matches. | Design | AC-019–AC-022 |
| RULE-006 | Business rule | Must | Imports contain only normalized usernames for accounts the member follows. Flutter shall not offer a follower import option and shall discard follower data locally; AppView shall reject any entry containing a relationship direction or any follower-specific field. | Follower relationships do not express the importing member's follow intent and are unnecessary private data. | User approval | AC-016, AC-018, AC-022, AC-029 |
| RULE-007 | Business rule | Must | Raw JSON/ZIP bytes, decoded raw export objects, and unrelated fields shall never be supplied to AppView or persisted; strict server requests containing raw/archive-like fields are rejected. | Enforces local-only parsing, not merely UI convention. | Design | AC-016, AC-017, AC-039, AC-054 |
| RULE-008 | Business rule | Must | Creating an import is informed authorization for AppView to automatically write ordinary `app.bsky.graph.follow` records for every current or future exact eligible match from that retained evidence; no separate per-match acceptance is required. | Defines the approved consent boundary for public graph restoration. | User request / User approval | AC-024, AC-025, AC-029 |
| RULE-009 | Business rule | Must | Normalized accounts-followed handles are private, accepted only after Instagram ownership verification, retained without expiry while the account remains verified, deleted on verification revocation, and never reveal the importer or import evidence to a matched member. | Implements the approved verification lifetime and privacy decision. | User approval | AC-026–AC-030 |
| RULE-010 | Business rule | Must | Successful automatic follows remain ordinary DID-based PDS records and are not undone by later import deletion, verification revocation, conflict, username change, or membership cleanup. | Cross-network metadata must not silently rewrite the public social graph. | Design / User approval | AC-031, AC-044 |
| RULE-011 | Business rule | Must | Standalone JSON and ZIP are local containers for the same Instagram following export and shall both cross the repository/AppView boundary only as existing `sourceType: instagramJson` plus normalized username entries. UI/history copy shall label both as an "Instagram export"; no ZIP-specific API value, database column, raw filename, or archive metadata is added. | Keeps the server contract based on normalized evidence rather than the member's download container. | User approval | AC-050, AC-054 |
| RULE-012 | Business rule | Must | Once an automatically followed target is later manually unfollowed, the terminal match ledger shall suppress any automatic re-follow from the same verification/import evidence. Verification revocation deletes that private ledger; a later verification plus fresh import is new authorization. | Prevents background reconciliation from fighting an explicit later user action. | User approval | AC-025, AC-056 |

### 12.1 State And Wire Contracts

Public enums are closed on the server and forward-compatible on the client. The
server never exposes internal lease, counter, digest, IGSID, conflict-party, or
PDS-operation fields.

| Aggregate | Public states | Required transition rules |
|---|---|---|
| Verification attempt | `pendingDm`, `processing`, `pendingConfirmation`, `confirmed`, `expired`, `cancelled`, `superseded`, `rejected`, `conflicted` | Creation enters `pendingDm` and supersedes the owner's earlier non-terminal attempt. A valid unique DM atomically consumes and clears the challenge digest and moves `pendingDm` to `processing`. Successful profile lookup moves to `pendingConfirmation`; bounded terminal provider/shape failure moves to `rejected` with a safe `retryCode`; explicit confirmation moves to `confirmed` or `conflicted`. Expiry, owner cancellation, and supersession are terminal. Only `pendingDm`, `processing`, and `pendingConfirmation` are non-terminal. |
| Verified account mapping | `active`, `membershipInactive`, `revoked`, `superseded`, `disputed` | Only a DM-verified, explicitly confirmed mapping becomes `active`. Discovery is possible only while `active`, conflict-free, and explicitly enabled. Membership loss moves an extant mapping to `membershipInactive`; rejoin preserves that state until the owner explicitly reactivates it and chooses discovery again. Revocation and supersession are terminal for that mapping version. A collision moves the new claim to `disputed` and leaves the current authoritative mapping unchanged. Internal schema/error names may retain `link`, but user-facing copy does not. |
| Import | `active`, `membershipInactive` | Creation is additive and enters `active`. Membership loss pauses it as `membershipInactive`; reactivation is explicit. DELETE removes it rather than exposing a `deleted` resource. Imports do not expire while their verified account mapping remains active. |
| Automatic-follow operation | `pending`, `writing`, `followed`, `alreadyFollowing`, `invalidated` | A newly eligible pair enters `pending` once per verification lifetime. A worker lease may move it to `writing`; retryable session/PDS failure returns it to `pending`. Deterministic PDS success ends as `followed`; a follow already present before this operation ends as `alreadyFollowing` without an automatic-follow notification. Loss of eligibility/support before a successful write ends as `invalidated`. `followed` and `alreadyFollowing` are terminal suppression facts for the verification lifetime. `invalidated` is terminal only for that operation; a later eligibility-restoration trigger may create/requeue work if current retained support again permits it. No state is publicly listed for review. |
| Link conflict | `open`, `resolvedKeepExisting`, `resolvedRevokeExisting`, `expired` | Conflicts are private operator-controlled records addressed only by opaque ID. Creation never transfers ownership. Resolution is an explicit audited operation; expiry anonymizes evidence and never grants the claimant ownership. |

Every `/v1/*` request and response uses camelCase JSON, rejects unknown request
fields, and uses the standard `{error, message, requestId}` error envelope. A
valid session without current membership receives `404 profile_not_found` on
every route below. Owner-scoped resources belonging to another DID are
indistinguishable from missing resources. Timestamps are UTC RFC 3339 strings;
IDs and cursors are opaque.

| Route | Success contract | Route-specific errors |
|---|---|---|
| `POST /v1/migrations/instagram/verifications` | `201`; request `{}`; response `{verificationId, state: pendingDm, challenge, expiresAt, dmUrl}`. Only this response exposes challenge plaintext. | `503 instagram_verification_unavailable`; `429 rate_limited`. |
| `GET /v1/migrations/instagram/verifications/current` | `200`; `{verification: null | {verificationId, state, expiresAt, candidateUsername?, retryCode?}}`. Returns only the caller's current non-terminal attempt after atomically expiring stale work; never returns challenge plaintext or DM URL. | No absent-resource error; `{verification: null}` is the normal empty result. |
| `GET /v1/migrations/instagram/verifications/{verificationId}` | `200`; `{verificationId, state, expiresAt, candidateUsername?, retryCode?}`. `candidateUsername` appears only in `pendingConfirmation`; safe retry codes are `profileLookupUnavailable`, `invalidProfileResponse`, and `membershipInactive`. | `404 instagram_verification_not_found`. |
| `DELETE /v1/migrations/instagram/verifications/{verificationId}` | Always `204`; cancel only when the ID is caller-owned and cancellable. Absent, expired-tombstone, and foreign IDs are indistinguishable successful no-ops. | No resource-specific error. |
| `POST /v1/migrations/instagram/verifications/{verificationId}/confirm` | `200`; request `{discoverable: boolean}`; response `{state, account}`. An identical replay returns the same result. | `404 instagram_verification_not_found`; `409 instagram_verification_state_conflict` or `instagram_link_conflict`; `429 rate_limited`. |
| `GET /v1/migrations/instagram/account` | `200`; `{integrationAvailable, account: null | {state, username, discoverable, conflictPending, reactivationRequired, verifiedAt}}`. Meta outage changes only `integrationAvailable`. | No Meta-outage error; `404 profile_not_found` still applies. |
| `PATCH /v1/migrations/instagram/settings` | `200`; strict request `{discoverable?: boolean, reactivate?: boolean}` with at least one field; response uses the account shape. Reactivation requires `reactivate: true` and an explicit `discoverable` value. | `404 instagram_link_not_found`; `409 instagram_reactivation_required` or `instagram_link_conflict`. |
| `DELETE /v1/migrations/instagram/account` | Always `204`; revoke the caller's current link if one exists. Repeated calls remain successful after tombstone purge. | No ownership disclosure. |
| `POST /v1/migrations/instagram/imports` | `201`; request `{sourceType: manual | instagramJson, entries: [{username}]}`; response `{import, counts: {followingCount}}`. Retention, direction, follower-specific fields, and public suggestion/match counts are unknown fields and are rejected. Requires an active verified Instagram account mapping. | `400 invalid_request`; `409 instagram_verification_required`; `413 request_too_large`; `422 invalid_instagram_import`; `429 rate_limited`. |
| `GET /v1/migrations/instagram/imports` | `200`; opaque cursor page `{items: [import], cursor?}` using the §12.4 page limits. | `400 invalid_cursor`. |
| `GET /v1/migrations/instagram/imports/{importId}` | `200`; `{importId, state, sourceType, followingCount, createdAt}`. No handle list is returned. | `404 instagram_import_not_found`. |
| `PATCH /v1/migrations/instagram/imports/{importId}` | `200`; strict request `{reactivate: true}`. Reactivation changes an owner import from `membershipInactive` to `active` after the verified link is reactivated. | `404 instagram_import_not_found`; `409 instagram_import_inactive` or `instagram_verification_required`. |
| `DELETE /v1/migrations/instagram/imports/{importId}` | Always `204`; remove support only when the ID is caller-owned. Absent, purged, and foreign IDs are indistinguishable successful no-ops. | No resource-specific error. |
| `GET /integrations/instagram/webhook` | `200 text/plain` containing `hub.challenge` only for exact `hub.mode=subscribe` plus constant-time verify-token match. | Generic `403` with no reflected challenge/token; `404` while integration is disabled. |
| `POST /integrations/instagram/webhook` | `200` after the durable transaction for every valid signed supported/duplicate delivery. A valid signed event over the per-IGSID invalid-redemption limit is recorded as a terminal deduplicated ignored fact with sensitive fields cleared, performs no lookup, and still returns `200`. Profile concurrency/backoff defers durable work and still returns `200`. | Generic `400`, `401`, or `413`; `404` while disabled. A pre-auth trusted-source-IP limit or post-signature global ingress limit returns generic `429` plus bounded `Retry-After` and persists no partial body. Never expose challenge/link existence. |

`integrationAvailable` means new Meta-dependent verification/profile/reply work
can proceed. It does not gate local imports, current account status,
disable/revoke, per-import retention/delete, queued automatic-follow work, or
notification preference changes.

### 12.2 Durable Webhook Work Contract

One supported message event creates at most one private work row containing
only:

- a versioned keyed digest of Meta message ID, used as the unique deduplication
  key;
- sender IGSID and configured official-account ID;
- a versioned keyed digest of the canonical normalized challenge;
- Meta event timestamp; and
- internal `queued`, `processing`, `retryable`, `completed`, `ignored`, or
  `failed` status plus attempt count, next-attempt, lease-owner, lease-expiry,
  created, and updated timestamps.

The row never stores raw webhook bytes, JSON, message text, plaintext challenge,
signature header, username/profile response, or unrelated event fields. The
handler validates the exact whole-token grammar before hashing and persists all
supported events from one body in a single transaction. A duplicate digest is
a successful no-op. Worker claim uses a lease plus `FOR UPDATE SKIP LOCKED`;
terminal processing clears sender IGSID and challenge digest immediately.

### 12.3 Actorful Automatic-Follow Notification Contract

The notification feed uses `type` as its sole discriminator. Existing social
types keep their actor and AT Protocol source fields. `instagramMatch`
identifies the profile that AppView automatically followed, but does not claim
that profile initiated the event:

```json
{
  "id": "opaque-notification-id",
  "type": "instagramMatch",
  "createdAt": "2026-07-19T12:00:00Z",
  "indexedAt": "2026-07-19T12:00:01Z",
  "actor": {
    "did": "did:plc:followed-account",
    "handle": "example.craftsky.social",
    "displayName": "Example",
    "avatar": "https://cdn.example/avatar.jpg",
    "viewer": {
      "isFollowing": true
    }
  }
}
```

`instagramMatch` items require `actor` and omit `uri`, `cid`, `rkey`, `system`,
and social source references. Unknown types decode to safe generic client copy
and no identity-bearing destination. Feed ordering/newness uses `indexedAt`.
There is one notification per successfully created automatic PDS follow; items
are not counted, grouped, or coalesced. A target that was already followed
before the operation produces no `instagramMatch` notification.

The notification is persisted only after the deterministic PDS write succeeds,
in an idempotent completion path that cannot create duplicates after a crash.
Its row remains ordinary notification history after import deletion,
verification revocation, or a later manual unfollow; current profile and
relationship state are hydrated when read, and existing block/mute/unavailable
actor rules continue to control feed visibility. Push scheduling begins only
after notification persistence. The provider payload contains no identity; it
opens the correct account's Notifications feed, while selecting the hydrated
row opens the actor's profile.

Initial import and every future eligibility trigger may enqueue automatic
follows. Evaluation is triggered by verification
confirmation/enable/reactivation, validated username change, membership
restoration after explicit reactivation, or relationship/moderation safety
restoration. Trigger handling is targeted and idempotent. The same
`InstagramSuggestionEligibilityPolicy` is checked at operation creation and
again immediately before the PDS write and notification completion.

### 12.4 Fixed Limits, Defaults, And Trust Policy

These values are fixed privacy/security maxima in production. Configuration may
tighten them but may not raise them; explicit test-only wiring may use smaller
windows. Persistent/shared rate buckets use Postgres so all replicas observe the
same limits.

The canonical challenge display grammar is
`CSKY-XXXX-XXXX-XXXX-X`, where the thirteen `X` values are the random
symbols. Matching trims outer whitespace, folds ASCII case, and accepts only
the complete token; inner whitespace, missing/extra hyphens, Unicode lookalikes,
or surrounding prose are invalid.

| Boundary | Production maximum/default |
|---|---|
| Challenge | 13 uniformly random characters from `23456789ABCDEFGHJKMNPQRSTVWXYZ` (about 63.8 bits); ten-minute lifetime. |
| Webhook request | 256 KiB exact raw-body cap; at most 100 supported message events; 1,000 requests/minute global and 300/minute per trusted source IP. |
| Challenge creation | 5/15 minutes per DID, 10/15 minutes per device, and 30/15 minutes per trusted source IP. |
| Invalid redemption | 10/15 minutes per sender IGSID and 30/15 minutes per trusted source IP; excess valid signed deliveries acknowledge generically and defer/drop without lookup. |
| Confirmation | 20/hour per DID and 30/hour per device. |
| Import | Existing `/v1` one-MiB body cap, at most 10,000 deduplicated entries/import, 10 imports/hour per DID and 20/hour per device. A standalone JSON file or the ZIP target entry is capped at 20 MiB before/through decode. ZIP metadata is capped at 100,000 central-directory entries and 64 MiB; total ZIP size is not capped when ignored media accounts for the size. |
| Pagination | Default 20 and maximum 50 items/page. Invalid/foreign cursors are `400 invalid_cursor`. |
| Meta HTTP | Five-second total timeout, 64-KiB response cap, at most 20 concurrent profile calls/process and 5 lookups/hour per IGSID. |
| Webhook worker | Four concurrent jobs/process, 60-second lease, five provider attempts, exponential backoff from one second capped at five minutes, and 15-minute maximum `processing` age before safe terminal rejection. |
| DM reply | At most one idempotent reply/event and only within the provider's configured interaction window, never assumed longer than 24 hours. |
| Automatic-follow worker | Batch default 20 and maximum 100; 60-second lease; five provider attempts per claim cycle; exponential backoff from one second capped at five minutes. |
| Operator/list purge | Batches of at most 500 rows with an explicit opaque cursor or ID; no unbounded command. |

Trusted source IP defaults to the socket peer. Forwarded headers are considered
only when the peer belongs to configured trusted proxy CIDRs, using the first
untrusted hop selected by the shared edge policy. Production startup fails if
multiple replicas are configured without persistent Instagram abuse storage.

Webhook throttling order is fixed. The trusted-source-IP ingress bucket runs
before body read and may return generic `429` with `Retry-After: 60`. Body-size
and signature checks run next. Only a valid signature consumes the shared
global-webhook bucket; excess returns the same generic `429` without decoding or
persisting events. Per-IGSID invalid-redemption limits run after bounded decode;
excess events create only a deduplicated terminal ignored fact, clear sensitive
fields, make no Meta call, and return the normal `200` acknowledgement. Worker
profile concurrency and retry pressure defer already durable work and never
change the webhook acknowledgement.

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-001 | Given Instagram configuration is absent or incomplete, when a member opens the feature or creates an attempt, then AppView reports a stable unavailable state/error, the Flutter page explains that verification is unavailable, and all unrelated routes continue normally. |
| AC-002 | BR-001, FR-002, NFR-001, RULE-001 | Given a configured fake integration, when a member creates an attempt, then the display challenge has at least 60 bits of formatted entropy, contains no ambiguous/personal data, expires in ten minutes, and only a keyed digest is persisted. |
| AC-003 | BR-001, FR-002, FR-024, RULE-001 | Given a member has a pending attempt, when they create another, then the first is superseded, the new response includes an opaque ID/expiry/HTTPS DM URL, and Flutter copies/opens only the new display challenge. |
| AC-004 | BR-001, FR-003, NFR-004 | Given one DID reads or confirms another DID's attempt, then the standard not-found boundary reveals nothing; cancellation with an owned, foreign, absent, or already-purged ID always returns `204` and mutates only the caller-owned attempt. |
| AC-005 | BR-001, FR-003, FR-024, NFR-004 | Given an owned attempt is `pendingDm`, `processing`, `pendingConfirmation`, `confirmed`, `expired`, `cancelled`, `superseded`, `rejected`, or `conflicted`, then polling returns the exact bounded state and no private IGSID/digest metadata. |
| AC-006 | BR-001, FR-004 | Given Meta verification query parameters, then only the exact configured verify token and expected subscribe mode return the supplied challenge; invalid requests return no challenge or secret. |
| AC-007 | BR-001, FR-005 | Given identical raw webhook bytes, then a valid HMAC-SHA256 signature is accepted, changed bytes/malformed headers/wrong secrets are rejected before JSON processing, and oversized bodies are rejected. |
| AC-008 | BR-001, FR-005, FR-006, NFR-005 | Given a valid supported message event, then the handler durably records its minimal event/message ID and acknowledges promptly; duplicate delivery returns success without duplicate work or reply. |
| AC-009 | FR-010, FR-024, BR-004, RULE-004 | Given an active verified account, when its owner disables discovery or re-enables it without conflict, then queued unwritten operations update accordingly and the owner's status reflects the action. When the owner chooses revocation at the bottom of the page, Flutter first explains that imported handles and automatic-follow authority are deleted while existing CraftSky follows remain; cancellation preserves verification, and destructive confirmation revokes it exactly once. |
| AC-010 | FR-006, FR-007, NFR-005, RULE-001 | Given replayed, out-of-order, concurrent, expired, cancelled, or superseded challenge messages, then at most one attempt reaches pending confirmation and a redeemed challenge cannot bind another sender. |
| AC-011 | FR-006, FR-007, NFR-006 | Given profile lookup is transiently unavailable, then the webhook remains acknowledged, the job retries with bounded backoff/timeout, status remains processing, and terminal exhaustion becomes a safe retryable user state without logging the response. |
| AC-012 | FR-007, FR-009, NFR-001, RULE-001 | Given invalid/non-verification/self/echo messages or an IGSID replay against another attempt, then no link is created, no message body is retained, no challenge existence is disclosed, and the appropriate bounded invalid/conflict outcome is recorded. |
| AC-013 | FR-007 | Given a valid challenge message, then only the sender IGSID and current normalized/display username are retained as the pending candidate; profile pictures, names, counts, and message history are not stored. |
| AC-014 | BR-001, FR-008, FR-024, RULE-002, RULE-004 | Given a pending candidate, when Flutter shows the actual username followed by a selector defaulted to `Allow discovery`, then the selected option's explanation appears immediately below it. When the creating member explicitly confirms without changing the default, the account is discoverable; changing to private is also honored. Verification is finalized exactly once, and a different DID or unconfirmed UI cannot finalize it. |
| AC-015 | BR-001, FR-008, FR-009, NFR-005, RULE-002, RULE-003 | Given concurrent confirmation or an existing DID/IGSID/username constraint, then one valid ownership result wins, conflicts remain unresolved without transfer, and retries return the same safe result. |
| AC-016 | BR-002, FR-012, FR-013, RULE-006, RULE-007 | Given a supported local standalone JSON or ZIP export, then Flutter extracts only normalized usernames from accounts the member follows and the outgoing request contains no direction, follower data, raw bytes, filename, archive metadata, arbitrary JSON subtree, URL, message, or unrelated field. |
| AC-017 | BR-002, FR-012, FR-013, RULE-007 | Given malformed, unsupported, partially changed, encrypted, missing/duplicate-target, integrity-failing, or oversized JSON/ZIP input, then parsing fails locally with guidance and no network call; given a server request with unknown/raw/archive-like fields, then strict decoding rejects it. |
| AC-018 | BR-002, FR-012, FR-013, FR-025, RULE-006 | Given following data plus follower data or unrelated all-information archive entries, then Flutter processes only accounts-followed data, discards/ignores everything else locally, and sends usernames without directions. A follower-only selection produces local guidance and no network request; an AppView request containing `direction` or follower-specific fields is rejected. |
| AC-019 | BR-002, FR-012, FR-014, RULE-005 | Given whitespace, a leading `@`, supported case variants, duplicates, invalid characters, overlong values, display names, and supported/unsupported Instagram URLs, then normalization/deduplication is deterministic and invalid/non-username evidence is rejected without fuzzy matching. |
| AC-020 | BR-003, FR-014, FR-015, RULE-004, RULE-005 | Given active discoverable verified, disabled, revoked, disputed, superseded, old-username, unverified, self, and departed mappings, then only the exact current eligible mapping can match. |
| AC-021 | BR-003, FR-015, FR-016, RULE-005 | Given eligible and ineligible targets, then internal operations are created only for current CraftSky members allowed by the complete visibility/moderation policy and never for self or a target already followed at creation time. No operation or private match reason is exposed through a member-facing route. |
| AC-022 | BR-003, FR-015, RULE-005, RULE-006 | Given follower-only, fuzzy, stale, or case-normalized-but-otherwise-non-exact evidence, then no automatic-follow operation is created; follower-only evidence cannot enter AppView persistence. |
| AC-023 | BR-003, FR-016, FR-025 | Given the updated AppView and Flutter surfaces, then the old Instagram suggestion list/accept/dismiss routes are unregistered, private operation rows are not serialized through another route, and the page contains no People You May Know section, suggestion models/providers, or review actions. |
| AC-024 | BR-003, BR-004, FR-017, FR-026, RULE-008 | Given a verified member creates an import containing a current eligible match, then creation itself authorizes one queued automatic-follow operation; no separate accept request is required. The operation creates no notification before PDS success, and a target already followed before the operation produces neither a duplicate write nor an automatic-follow notification. |
| AC-025 | BR-003, BR-004, FR-015, FR-017, FR-026, NFR-005, RULE-008, RULE-012 | Given duplicate eligibility triggers, retries, concurrent workers, temporary PDS failure, crash recovery, and firehose delay, then at most one logical follow and one corresponding `instagramMatch` notification are created; failure stays retryable, final eligibility is rechecked, and the notification's Follow/Following action cannot mutate under another account. |
| AC-026 | BR-004, FR-012, FR-018, FR-025, RULE-009 | Given an active verified Instagram account, direct manual or Instagram-export import retains every normalized following handle without a retention field or normalized-preview step. Creating an import without verification is rejected. Deleting an owned import removes its handles/support and cancels an unwritten operation only when no other active source supports it; deleting an absent/foreign ID is the same `204` no-op. |
| AC-027 | BR-004, FR-018, RULE-009 | Given a retained import, then its handles do not expire or require renewal while the Instagram account remains verified. A `membershipInactive` import may be explicitly reactivated after the verified account mapping is reactivated. |
| AC-028 | BR-004, FR-018, FR-028, RULE-009 | Given an import is deleted or Instagram verification is revoked, then applicable imports, handles, support rows, and unwritten operations are removed. Given the owner merely departs current membership, imports are paused rather than deleted. Successful PDS follows and existing delivered notification rows remain, unsent notification/push work for operations that never succeeded is absent or cancelled, and no dangling support row remains. |
| AC-029 | BR-003, BR-004, FR-017, FR-019, FR-020, NFR-005, RULE-006, RULE-008, RULE-009 | Given several retained following handles become eligible in one transaction/batch, then one deduplicated operation exists per importer/target and each successfully created automatic follow produces exactly one separate actorful `instagramMatch` notification; no digest, count, grouping, or coalescing is used. |
| AC-030 | BR-004, FR-019, RULE-009 | Given a current or future match, then the target may observe the resulting ordinary public follow and its normal incoming-follow effects but cannot learn that Instagram evidence caused it or which import/handle set was retained. Only the importer receives the private automatic-follow notification. |
| AC-031 | BR-004, FR-010, FR-015, FR-018, FR-028, RULE-004, RULE-010 | Given verification revocation, discovery disablement, owner departure, username/conflict invalidation, or import deletion before the PDS write, then the unsupported operation cannot write or notify. If the write already succeeded, the ordinary follow and historical notification remain unchanged. |
| AC-032 | FR-009, FR-010, FR-029, RULE-003 | Given a conflicting IGSID or username claim, then both owners see generic private conflict state, the existing link remains in place, operator audit uses opaque IDs, and no reassignment occurs without an explicit resolution action. |
| AC-033 | FR-011, RULE-003 | Given a validated username change for the same IGSID, then current username updates, old-handle unwritten operations invalidate, the old username is not transferred, and collisions enter conflict. |
| AC-034 | BR-004, FR-020, FR-021, FR-025 | Given notification preferences are read or patched, then `instagramMatch` appears with fixed `scope: everyone` and configurable `pushEnabled`; scope mutation is rejected, Flutter shows no actor-scope control, and the Instagram import UI links to Notification Settings. In-app automatic-follow notifications are still created when push is disabled. |
| AC-035 | FR-020, FR-022, FR-026 | Given one or several successful automatic follows, then each notification-feed row has localized automatic-follow copy, the target's hydrated profile/avatar, current Follow/Following action, and opens that profile; a push opens the correct account's Notifications feed without identifying the target. |
| AC-036 | FR-020, FR-022 | Given provider payload inspection, then it contains only category, stable notification ID, and opaque account-subscription binding—never a client route, handle, actor DID, IGSID, challenge, import, match, count, or suggestion data. |
| AC-037 | FR-020, FR-022, FR-026 | Given import deletion, verification revocation, or manual unfollow after a successful automatic follow, then the historical notification remains, hydrates current relationship state, and its row still opens the profile. Existing block/mute/unavailable-actor policy may hide it. Given invalidation before PDS success, no notification or push is created. |
| AC-038 | FR-016, FR-023, FR-024, FR-025, NFR-007 | Given each loading/empty/disabled/error/success state, then the typed Settings route renders localized accessible verification, import, consent, retention, and retry controls without exposing raw server errors; user-facing copy says verified/verification, the active summary says `Verified as @…`, enabled discovery uses semantic success/moss green, Instagram export is the default selector option, People You May Know is absent, and revocation appears after imports at the page bottom. |
| AC-039 | BR-002, NFR-003, RULE-007 | Given wholly synthetic canaries or explicitly approved redacted fixtures as controlled inputs across success and failure paths, then each value appears only in its specifically intended private database/API/UI field and never in logs, errors, spans, Sentry, metric labels, push payloads, PDS writes, raw-request reserialization, or unrelated snapshots. No real or user-derived secret/private value is committed as a fixture. |
| AC-040 | FR-001, NFR-004, NFR-006 | Given route/config tests, then every authenticated Instagram route has the required policy and standard wire contract, integration secrets exist only in AppView config, and partial production config is rejected. |
| AC-041 | FR-005, NFR-002 | Given excessive activity, then client DID/device operations receive generic `429`; pre-auth webhook IP or post-signature global ingress receives generic `429` plus bounded `Retry-After`; per-IGSID invalid redemptions are deduplicated/ignored with `200` and no lookup; profile concurrency defers durable work. No response reveals challenge/link existence. |
| AC-042 | FR-023, FR-026, NFR-008 | Given an account switch during polling, parsing, import, confirmation, notification follow/unfollow, notification open, or response completion, then stale work cannot update UI/cache, navigate, disclose state, or mutate for the new account. |
| AC-043 | FR-027 | Given configured reply support, then accepted/expired/invalid/completed replies are idempotent, bounded, and only sent inside the allowed interaction window; disabled or failed replies never change verification correctness. |
| AC-044 | FR-028, RULE-010 | Given private export, membership-inactivation, scoped-delete, and terminal-purge service tests, then export contains only that member's facts; membership loss retains but disables them within §15 limits; scoped delete removes only the selected verification/import facts; terminal purge removes all owned private facts and invalidates dependent unwritten work; successful public follows are untouched. |
| AC-045 | FR-029 | Given operator CLI tests, then conflict listing/retry/revoke/purge/resolve actions require explicit identifiers, emit bounded audit records, redact identity/secrets, and are safe to repeat. |
| AC-046 | NFR-002 | Given production configuration requests multiple AppView replicas without shared Instagram abuse enforcement, then startup/readiness fails closed or clearly disables Instagram verification. |
| AC-047 | NFR-009 | Given polling/import/confirmation is cancelled by account switching or client teardown, then AppView classifies it as canceled/499 and does not report it as a 5xx/Sentry failure. |
| AC-048 | BR-001, FR-003, FR-015, FR-017, FR-018, FR-025, FR-028, FR-030, NFR-004 | Given a still-valid session whose DID is absent from `craftsky_profiles`, then every authenticated Instagram route returns `404 profile_not_found`; queued workers inactivate/pause owned Instagram state and create no verified mapping, automatic-follow notification, or PDS follow. Rejoining requires explicit verification-mapping reactivation and per-import reactivation, neither of which silently restores discovery. |
| AC-049 | FR-003, FR-024, NFR-008 | Given a member leaves and reopens the Instagram page before an attempt expires, when Flutter fetches the caller's current attempt, then it resumes polling or pending confirmation without creating or superseding an attempt. A matching account-scoped secure snapshot restores the display challenge and DM URL; an absent/mismatched snapshot exposes neither but still shows and permits cancellation of the server attempt. Expiry, cancellation, confirmation, supersession, and session invalidation clear only that account's snapshot, and late work cannot cross an account switch. |
| AC-050 | FR-013, FR-025, FR-031, RULE-011 | Given a verified mobile member selects either `following.json` or the original Instagram ZIP, then one local parse creates the same normalized import request using `sourceType: instagramJson`; the picker/UI calls both an "Instagram export", recommends exporting only accounts followed, and explains that an all-information ZIP stays local. |
| AC-051 | FR-013, FR-014 | Given the approved current shape with `title` and `https://www.instagram.com/_u/<username>` but no `value`, then every agreeing URL/title pair yields that normalized username. A mismatched title, arbitrary title-only record, non-HTTPS URL, different host/path, query, fragment, encoded separator, invalid username, or ambiguous multiple evidence value is ignored/rejected according to the bounded parser contract and never used for fuzzy inference. |
| AC-052 | FR-013, FR-031, NFR-010 | Given a ZIP containing messages, media, follower files, and exactly one canonical following entry, then processing occurs off the UI isolate, archive payload stays file-backed, only the target entry is decompressed, no entry is written to disk, and the isolate returns only normalized entries/counts. Missing, duplicate, encrypted, malformed, unsupported-compression, CRC/integrity-failing, over-20-MiB target, over-100,000-entry, or over-64-MiB-directory archives fail locally with no request. |
| AC-053 | FR-031, NFR-010, NFR-008 | Given a large valid archive, selection/cancellation/account switching/page disposal, then the UI remains responsive; late isolate results are fenced by the captured account lease; native file/archive handles close on success and every failure; and no complete archive byte buffer crosses into or out of the worker isolate. |
| AC-054 | BR-002, RULE-007, RULE-011 | Given JSON and ZIP imports with controlled private canaries in filename, archive paths, unrelated entries, raw JSON, URLs, follower records, messages, and media, then the serialized repository request, AppView diagnostics/storage, Sentry, push, and PDS contain only the existing `instagramJson` source value and normalized following usernames where explicitly intended. |
| AC-055 | BR-003, FR-017, FR-032, NFR-005 | Given the importer owns multiple valid sessions and another DID also has stored sessions, then the worker deterministically selects the most recently active unrevoked usable session for the importer only. With no usable owner session it performs no PDS write and leaves the operation retryable; a provider-invalid session is invalidated only for that exact account/session before another owner session may be selected. |
| AC-056 | BR-004, FR-017, FR-019, RULE-012 | Given a target was automatically followed and the member later unfollows it, then subsequent matching, reconciliation, relationship restoration, and duplicate-import triggers do not create another automatic-follow operation from the same verification lifetime. After verification revocation deletes that private ledger, a later verification plus fresh import is new authorization and may follow again. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | Challenge text includes surrounding whitespace or different case. | Normalize only the documented display grammar; accept the same token without weakening entropy or accepting extra prose. | FR-007, RULE-001 |
| EC-002 | One webhook body contains several entries/messages. | Independently enqueue supported unique message IDs within the bounded batch; ignore unsupported entries and never acknowledge before the durable write completes. | FR-005, FR-006 |
| EC-003 | Meta sends a message from the official account or an echo. | Ignore it; never treat the official account as a candidate. | FR-007 |
| EC-004 | Candidate deletes the DM after delivery. | A previously validated/redeemed candidate remains pending confirmation; message deletion does not reassign or expose the challenge. | FR-007, RULE-001 |
| EC-005 | User cancels after webhook enqueue but before worker redemption. | Worker observes cancellation and does not attach a candidate. | FR-003, NFR-005 |
| EC-006 | User revokes while automatic-follow operations or push jobs exist. | Invalidate unwritten operations and cancel delivery for any notification not yet validly created; successful follows and their historical notification rows remain. | FR-010, FR-018, FR-028 |
| EC-007 | Same normalized handle appears in both follower and following sections of a selected export. | Keep one normalized accounts-followed entry and discard the follower occurrence locally. | FR-013, FR-014, RULE-006 |
| EC-008 | Import contains more than the server/client cap. | Reject locally when possible and server-side otherwise, without partial persistence. | FR-012, FR-013 |
| EC-009 | A queued target loses eligibility between matching and the worker write. | Final policy validation invalidates or defers the operation without a PDS write, notification, or private-data disclosure. | FR-015, FR-017 |
| EC-010 | The target becomes followed outside the automatic-follow worker before its write. | Mark the operation `alreadyFollowing` without a duplicate PDS write or false automatic-follow notification. | FR-015, FR-017 |
| EC-011 | An automatic-follow notification opens after verification/import deletion or manual unfollow. | Open the target profile using current hydrated relationship state; never reconstruct deleted private import evidence from notification or push facts. | FR-020, FR-022, FR-026 |
| EC-012 | Unknown future notification category reaches an older client. | Existing generic forward-compatible behavior remains usable; `instagramMatch` is fully known to the updated client. | FR-020 |
| EC-013 | Meta returns a valid response with missing/invalid username. | Keep the job bounded/retryable or reject the candidate; never link only by a missing username. | FR-007 |
| EC-014 | Username is released and claimed by another IGSID. | The old link does not confer ownership; active collision is disputed and no automatic transfer occurs. | FR-009, FR-011 |
| EC-015 | Integration is disabled after challenges exist. | New attempts stop; existing state remains private and cancellable; workers stop external calls without losing durable jobs. | FR-001, FR-006 |
| EC-016 | A ZIP contains no canonical following entry or contains it more than once. | Reject as unsupported/ambiguous without inspecting fallback filenames or uploading anything. | FR-013, FR-031 |
| EC-017 | A ZIP is very large because it contains media. | Do not reject solely on total ZIP bytes; keep media file-backed and decode only the bounded target entry. | FR-031, NFR-010 |
| EC-018 | ZIP central-directory counts/size or target size exceed limits. | Reject before target decompression and without a network call. | FR-031, NFR-010 |
| EC-019 | ZIP metadata lies about target size or decompression/integrity exceeds the declared bound. | Abort bounded output, close resources, and report a local invalid-export error. | FR-031, NFR-010 |
| EC-020 | Current export URL and title disagree. | Treat the record as invalid; neither field independently becomes a username. | FR-014 |
| EC-021 | Current export contains title only, a lookalike host/path, query/fragment, or encoded separator. | Reject that evidence without fuzzy/title inference. | FR-014 |
| EC-022 | Account switch or page disposal occurs while the isolate is parsing. | Allow bounded worker cleanup to finish but discard its result through the captured account lease; never upload under the new account. | FR-031, NFR-008 |
| EC-023 | A member selects ZIP on Flutter web. | ZIP is unavailable/unsupported; direct JSON behavior remains a separate platform capability. | FR-031 |
| EC-024 | A queued operation has no valid OAuth session for its owner. | Make no PDS call, keep the operation retryable with bounded backoff, and never select a session owned by another DID. | FR-017, FR-032 |
| EC-025 | A member manually unfollows an automatically followed target and later imports the same handle again. | The terminal ledger suppresses re-follow for the current verification lifetime; revocation followed by new verification and a fresh import is new consent. | RULE-012 |
| EC-026 | The PDS follow succeeds and the worker crashes before recording completion. | Deterministic retry reconciles the same PDS record and produces exactly one completed operation and one notification. | FR-017, FR-020, NFR-005 |

## 15. Data / Persistence Impact

- New private tables, subject to the final coding plan and the next free migration number:
  - `instagram_verification_attempts`
  - `instagram_account_links`
  - `instagram_link_conflicts` and/or a bounded audit table
  - `instagram_webhook_events` / Meta work queue
  - `instagram_graph_imports`
  - `instagram_graph_handles`
  - `instagram_follow_suggestions`
- Existing notification persistence changes:
  - Add `instagramMatch` to category constraints and registries.
  - Replace the actorless digest/count payload with one actorful
    `instagramMatch` event per successful automatic follow, discriminated only
    by category/type and containing no synthetic AT Protocol source record.
  - Enforce idempotent one-notification-per-operation completion and preserve
    historical rows after successful follows.
- Identity and uniqueness:
  - Active one-to-one constraints for owner DID and IGSID.
  - Partial uniqueness for discoverable normalized usernames.
  - Stable operation/import/event IDs are opaque UUIDs; automatic follows use a
    deterministic PDS rkey.
- Ownership/deletion:
  - Owner-scoped rows use the DID as owner identity but do not broadly cascade
    from `craftsky_profiles`; membership loss invokes the explicit reversible
    inactivation service.
  - Cross-user operation rows have explicit invalidation paths before write
    when either member departs. Successful notification rows use existing
    actor-visibility policy.
  - The worker references existing owner-scoped OAuth sessions and never copies
    session tokens into Instagram tables.
Retention is intentionally bounded. Raw webhook message bodies and raw exports
are never retained at all. The following maxima are fixed privacy limits;
configuration may shorten, never extend, them:

| Private record | Sensitive-field handling | Maximum retention |
|---|---|---|
| Non-terminal verification attempt | Challenge digest exists only through redemption/terminal transition; candidate IGSID exists only while profile confirmation is needed. | Ten-minute challenge validity; processing is terminally rejected within 15 minutes; any remaining non-terminal row expires within 24 hours. |
| Terminal verification attempt | Clear challenge digest, candidate IGSID, and candidate username immediately after link/conflict result is durably represented. Retain only opaque ID/owner/state/timestamps/retry code for idempotency/support. | 30 days after terminal state. |
| Webhook work | Never store raw body/text/signature. Clear sender IGSID and challenge digest on terminal processing; retain keyed message digest/status/timestamps for replay suppression. | Seven days after terminal processing. |
| Active link | Retain IGSID/current username only while required for an active member-owned link. | Until owner revokes, terminal account deletion, or 12 months continuously `membershipInactive`. |
| Revoked/superseded link tombstone | Remove plaintext IGSID/username immediately. A versioned keyed IGSID digest may block rebinding for 90 days and a keyed username digest may enforce a 30-day cooldown. | 90 days. |
| Link conflict and operator audit | Keep minimum encrypted/private evidence while open; resolution/expiry removes identity evidence and retains opaque action/result facts. | Open conflict 365 days, then `expired`; resolved audit 365 days. |
| Import and graph handles | Storage contains accounts-followed usernames only and has no retention-consent, expiry, direction, or follower-count fields. All normalized handles remain available for exact future matching while the Instagram account remains verified. Membership inactivity pauses matching but does not delete the import. | Until the owner deletes the import, revokes Instagram verification, or terminally deletes the CraftSky identity. |
| Automatic-follow operation/support | Pending support cannot outlive all supporting imports or the verified account mapping. Invalidated rows retain no username/IGSID. `followed`/`alreadyFollowing` retain only opaque terminal operation facts needed for replay and manual-unfollow suppression. | Through the current verification lifetime; verification revocation deletes the operation ledger after cancelling unwritten work. |
| `instagramMatch` event/delivery | Event identifies recipient and followed target actor plus category/status; it contains no import, Instagram, grouping, or count facts. | Existing notification retention policy, currently 90 days after last activity; invalid unsent delivery is purged within seven days. |
| Abuse counters | Keyed identifiers only; never raw challenge/message/username. | Window end plus 24 hours. |
| Generated private export | Stream from an owner-scoped snapshot; do not persist an export blob. | Request lifetime only. |

Explicit import deletion removes that import and its support immediately;
an unwritten operation survives only when another active import supports it.
Explicit verification revocation deletes every owner import and private
operation/suppression ledger, applies the pseudonymous cooldown tombstone
above, and invalidates dependent unwritten state but never a successful public
follow or its historical notification. Terminal
identity/account deletion purges all member-identifying Instagram rows
immediately, anonymizes any required bounded operator audit, and cancels
dependent work. Membership loss alone follows the reversible rules above.
- Backwards compatibility:
  - The app has no production users. This intentionally removes the public
    suggestion routes and `initialSuggestionCount`, replaces the actorless
    `instagramMatch` payload with an actorful payload, and continues to omit the
    redundant `kind` and destination fields while preserving safe
    unknown-notification behavior.
  - No lexicon change is required.
  - ZIP and standalone JSON both retain the existing `instagramJson` wire and
    database value; no AppView migration or route change is required.

## 16. UI / API / CLI Impact

- UI:
  - Add **Find people from Instagram** under Settings.
  - Add typed verification/discovery, verified-only direct
    manual/Instagram-export import, empty/error/unavailable, and
    conflict-warning states. Use verified/verification terminology throughout
    user-facing copy.
  - Default to Instagram export, remove People You May Know, use semantic
    success/moss green for enabled discovery, and place destructive revocation
    at the bottom after import content.
  - The file action accepts `.json` and `.zip`, is labelled as an Instagram
    export rather than its container, and recommends requesting only accounts
    followed while explaining that full ZIP exports remain local.
  - Extend notification settings/feed/icon/copy/open behavior for actorful
    `instagramMatch` rows with the existing Follow/Following action.
- API:
  - No API change for ZIP support; both containers submit `instagramJson`.
  - `POST /v1/migrations/instagram/verifications`
  - `GET /v1/migrations/instagram/verifications/{verificationId}`
  - `POST /v1/migrations/instagram/verifications/{verificationId}/confirm`
  - `DELETE /v1/migrations/instagram/verifications/{verificationId}`
  - `GET /v1/migrations/instagram/account`
  - `DELETE /v1/migrations/instagram/account`
  - `PATCH /v1/migrations/instagram/settings`
  - `POST /v1/migrations/instagram/imports`
  - `GET /v1/migrations/instagram/imports`
  - `GET /v1/migrations/instagram/imports/{importId}`
  - `PATCH /v1/migrations/instagram/imports/{importId}`
  - `DELETE /v1/migrations/instagram/imports/{importId}`
  - `GET /integrations/instagram/webhook`
  - `POST /integrations/instagram/webhook`
  - Remove the three `/v1/migrations/instagram/suggestions*` routes and remove
    `initialSuggestionCount` from import creation.
- CLI:
  - Add bounded operator commands for conflict inspection/resolution, link revocation, job retry/inspection, and supported terminal-record retention where covered by FR-029.
- Background jobs:
  - Durable webhook/profile/reply worker.
  - Bounded durable automatic-follow worker with owner-session selection,
    final policy validation, deterministic PDS writes, retry/backoff, and
    idempotent notification completion.
  - Bounded expiry/retention purge.
  - Targeted initial/future match production; no broad periodic scan when a
    verification/import state transition can target candidates.

## 17. Security / Privacy / Permissions

- Authentication:
  - Every client route requires a valid CraftSky bearer session and device ID.
  - Meta callback verification/signature is independent of CraftSky session auth.
- Authorization:
  - Caller DID always comes from middleware.
  - Attempt/verified-mapping/import queries use owner DID in the storage
    predicate and return not-found across ownership boundaries. Internal
    automatic-follow operations are never exposed by member-facing routes.
  - Confirmation must use the same DID that created the attempt.
  - Background writes select an unrevoked OAuth session for the operation owner
    DID only; absence or invalidity is retryable and cannot fall through to
    another account.
- Sensitive data:
  - Meta app secret, verify token, account access token/ID, challenge digest key, IGSIDs, usernames, imported handles, conflicts, and graph state remain server-side/private as applicable.
  - Flutter receives an Instagram username only for the member's own
    verification candidate/account. The importer may receive the target's
    ordinary CraftSky profile only after a successful public follow, and never
    receives IGSID or import/match evidence.
  - The ZIP filename, archive directory, entry names, compressed bytes,
    unrelated media/messages/follower data, and decoded raw JSON remain inside
    the native parser boundary. Only normalized entries/counts leave the
    isolate.
- Abuse cases:
  - Challenge guessing/replay, IGSID rebinding, duplicate webhook delivery, forged signatures, oversized bodies, username collision/takeover, import enumeration, follow duplication, and notification leakage have explicit constraints and tests.
  - ZIP bombs, misleading size metadata, duplicate canonical entries,
    encrypted/unsupported entries, path lookalikes, CRC/integrity failures, and
    title/URL disagreement fail locally within fixed bounds.
- External enablement:
  - Production readiness requires the Phase 0 capability spike with an unrelated personal account, confirmed profile lookup, token lifecycle, webhook subscription, app review/access level, privacy policy, deletion callback, and business requirements.

## 18. Observability

- Events/metrics use bounded dimensions only:
  - challenge issued/redeemed/expired/cancelled/confirmed/conflicted
  - webhook accepted/signature-failed/duplicate/unsupported and processing latency/queue depth
  - profile/reply success/retry/terminal failure
  - link activate/revoke/discovery-change/username-change/conflict
  - import size bucket/source type/match-rate bucket/deletion/inactivation/reactivation (the
    source type remains `instagramJson`; do not add filename/archive metadata)
  - match operation queued/invalidated/already-following/manual-unfollow-suppressed
  - automatic follow succeeded/retried/terminal-failed and owner-session-unavailable
  - `instagramMatch` created/duplicate-suppressed/push outcome
- Logs:
  - Use bounded component/operation/result/state/error-category/run-ID attributes.
  - Never log request bodies, query verification tokens, signature headers, upstream URLs containing tokens, identity values, handles, challenges, or imported counts precise enough to expose a list.
- Alerts:
  - Signature-failure spikes, queue age/depth, terminal Meta errors, conflict rate, purge failure, and integration disabled/misconfigured state.
- Cancellation:
  - Preserve canceled/499 classification and no-Sentry behavior.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Meta dashboard/API behavior differs from fixtures or current documentation. | Live verification fails or profile data is unavailable. | Keep a narrow versioned adapter; require the capability spike before enablement; fixture-test all known payloads. |
| RISK-002 | Link/discovery bug exposes a member's cross-network identity. | Privacy and safety harm. | Separate proof from discoverability, fail closed, enforce owner predicates/constraints, and test revocation/conflicts/concurrency. |
| RISK-003 | Raw exports or webhook content leak into requests/telemetry. | Broad private-data disclosure. | On-device parser boundary, strict request schema, redaction/secret scans, and no raw persistence. |
| RISK-004 | Duplicate webhook/follow operations create inconsistent state, duplicate public records, or duplicate notifications. | Wrong verified mappings or duplicate follows/alerts. | Database idempotency, durable deduplication, deterministic PDS rkey, one completion record, and crash/concurrency tests. |
| RISK-005 | Process-local rate limiting is insufficient after horizontal scaling. | Brute force or Meta API exhaustion. | Persistent/shared integration limiter and fail-closed multi-replica readiness requirement. |
| RISK-006 | An actorful automatic-follow notification could imply that the target initiated the action. | Misleading feed copy or accidental source-field fabrication. | Explicit automatic-follow copy, target only as the followed profile, no synthetic URI/CID/rkey, and contract tests. |
| RISK-007 | Instagram export shapes drift. | Imports fail until a client update. | Versioned tolerant local parser, clear unsupported guidance, real redacted fixtures, and accepted client-release trade-off. |
| RISK-008 | Username reuse creates an apparent identity transfer. | Wrong-person automatic follow. | Anchor on IGSID, invalidate old-handle unwritten operations, partial uniqueness, final policy validation, conflict/re-verification, and no automatic transfer. |
| RISK-009 | Cross-account async work updates the wrong Flutter account. | Private-data disclosure or wrong follow. | Fixed-account clients, operation leases, account-keyed state, and switch-during-operation tests. |
| RISK-010 | No repository-wide account export/deletion endpoint exists. | Lifecycle integration is incomplete. | Implement reusable scoped export/purge services and cascades now; document future composition as a release checklist item. |
| RISK-011 | A large or adversarial ZIP exhausts memory, storage, or the UI isolate. | App termination, stalled UI, or partial private-data handling. | Native path handoff, background isolate, file-backed archive decoding, bounded central-directory/target/output limits, no full extraction, and failure-path resource tests. |
| RISK-012 | The observed current URL/title export shape changes again. | Valid imports fail or ambiguous strings are mistaken for usernames. | Version exact supported shapes, require strict URL grammar/title agreement, keep manual entry fallback, and add only synthetic fixtures derived from approved structural observations. |
| RISK-013 | A member does not understand that import creates public follows. | Surprising social-graph changes. | Explicit import copy describing current/future automatic follows, verification-gated imports, ordinary unfollow control, and full verification revocation. |
| RISK-014 | A background worker selects the wrong stored OAuth session. | Cross-account public write. | Query by exact owner DID, deterministic most-recent valid selection, narrow invalidation, no fallback across owners, and multi-session integration tests. |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | Meta continues to provide an Instagram-scoped sender ID and username lookup after a user messages the owned professional account. | The ownership anchor or automatic username candidate cannot be established; revisit the verification approach. |
| ASM-002 | Standard Access is sufficient when CraftSky manages only its own professional account. | Advanced Access/app review may become a pre-launch requirement; adapter code remains unchanged. |
| ASM-003 | An HTTPS `ig.me` or equivalent DM link can be configured server-side and opened through the existing external-link helper. | Flutter needs a separately reviewed safe custom-scheme fallback. |
| ASM-004 | Selected real exports keep either the supported direct `value` shape or the observed exact `_u/<username>` URL plus agreeing title shape under the canonical following path. | Manual text remains usable while fixture-backed parser support is revised. |
| ASM-005 | PostgreSQL is available for durable webhook inbox, shared abuse counters, and workers. | A different shared store would be needed before multi-instance enablement. |
| ASM-006 | A member has at most one active Instagram account link in this product version. | Data model/UI must be generalized before supporting several accounts. |
| ASM-007 | On supported iOS/Android file selection, `file_selector` supplies a native readable path for the lifetime of the isolate operation. | The picker adapter must first copy the selected stream into an app-owned temporary file with explicit cleanup, without changing the parser/privacy contract. |
| ASM-008 | At least one stored OAuth session for the importing DID will normally remain usable for background PDS writes. | Automatic follows remain queued until the member establishes a usable session; no cross-account fallback is permitted. |
| ASM-009 | The verification-gated import action and its explicit copy are sufficient informed authorization for ordinary public follows to exact eligible matches. | Product/legal review must add a stronger confirmation step before automatic writing is enabled. |

## 21. Open Questions

- [ ] Non-blocking for implementation, blocking for production enablement: complete the Meta capability spike with a real app, official professional account, and unrelated personal sender; record actual webhook/profile/reply fixtures and access requirements.
- [ ] Non-blocking for implementation, blocking for broader export-shape
  confidence: obtain additional consented/redacted current Instagram exports.
  The approved 2026-07-21 sample establishes the observed URL/title structure,
  but no user-derived archive or private values may be committed.
- [ ] Non-blocking until the repository-wide lifecycle feature exists: compose the new Instagram private export/purge services into the eventual member data-export and account-deletion endpoints/UI.
- [ ] Non-blocking until abuse/operations deployment design: confirm the production AppView replica count and selected shared rate-limit deployment.

## 22. Review Status

Status: Draft
Risk level: High
Review recommended: Required
Reviewer: Awaiting user approval of the 2026-07-27 revision
Date: 2026-07-27
Notes: The user approved the recommended direction: durable automatic following,
actorful per-account notifications, and manual-unfollow suppression for the
current verification lifetime. Detailed requirements require explicit approval
before acceptance tests or implementation are revised. Production enablement
remains blocked on the capability spike and external configuration. No commit
or push was authorized.

## 23. Handoff To Test Design

- Requirements file: `01-requirements.md`
- Next test specification: `02-acceptance-tests.md`
- Must-cover requirement IDs:
  - `BR-001`–`BR-004`
  - `FR-001`–`FR-010`, `FR-012`–`FR-026`, `FR-028`, `FR-030`–`FR-032`
  - `NFR-001`–`NFR-010`
  - `RULE-001`–`RULE-012`
- Suggested test levels:
  - Pure unit tests for challenge grammar/digest, signatures, webhook decoding,
    username normalization, strict current URL/title parsing, standalone
    JSON/streamed ZIP parsing and bounds, matching/eligibility, state
    transitions, retry/backoff, automatic-follow eligibility, manual-unfollow
    suppression, notification inference, and fixed-account fencing.
  - Database integration tests for migrations, constraints, concurrency,
    ownership, durable inbox leasing/deduplication,
    verification/import/automatic-follow operation lifecycle, owner OAuth
    session selection, deterministic PDS completion, actorful notification
    persistence, and deletion/purge.
  - HTTP contract tests for every retained authenticated/integration route,
    removed suggestion routes, import response shape, and standard
    errors/body/rate limits.
  - Flutter API/provider/widget/router tests for privacy boundary, all page
    states, informed automatic-follow consent, default export selection,
    JSON/ZIP selection and isolate parsing, absence of People You May Know,
    bottom revocation, actorful notification follow/unfollow/rendering/open,
    and account switching.
  - Manual Meta dashboard/capability tests after credentials exist.
- Blocking open questions: explicit approval of this revised requirements
  document is required before acceptance-test design and implementation. The
  four open items in §21 block production enablement or later repository-wide
  lifecycle composition only.
