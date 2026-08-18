# Coding Plan: Instagram DM Ownership Verification And Automatic Following

> **2026-08-14 AppView audit amendment:** Section 13 supersedes the
> automatic-follow worker, background-session, notification, and
> suggestion-removal portions of this plan.

## 1. Inputs

- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` — **Approved with notes**
- API architecture:
  `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md`
- Existing implementation baseline:
  - Instagram verification, webhook, import, reconciliation, retention, and
    operator services are implemented.
  - Mobile standalone JSON/ZIP parsing, fixed-account providers, resumable
    verification, and the Settings page are implemented.
  - The current behavior still exposes suggestion review APIs/UI, performs the
    PDS follow only after acceptance, and creates coalesced actorless
    `instagramMatch` notifications.
- Review finding resolved by this plan: DR-015. This document replaces the
  stale suggestion/digest coding plan in full.
- Risk level: High. Background public PDS writes and multi-account credential
  selection require test-first owner isolation.

## 2. Implementation Strategy

Implement the approved revision as a delta over the completed verification and
ZIP-import baseline.

Keep the existing private match, source-support, reconciliation-job, and PDS
operation tables where their constraints remain useful, but treat them only as
an internal automatic-follow ledger. Remove all member-facing suggestion
handlers, repository methods, Dart models/providers, and People You May Know
widgets. Import and targeted reconciliation will create a deterministic
operation immediately for every eligible importer/target pair.

Add a separately leased `AutomaticFollowWorker`. It will:

1. claim one owner-scoped pending operation;
2. re-evaluate the shared `InstagramSuggestionEligibilityPolicy`;
3. finish as `alreadyFollowing` without notifying when a follow already exists;
4. select only the most recently active usable OAuth session for the operation
   owner;
5. call the existing deterministic `followwrite.Service`;
6. atomically mark the operation followed and create exactly one actorful
   `instagramMatch` notification; and
7. retry temporary session/PDS failures without exposing or switching owners.

The terminal followed/already-following ledger remains for the current
verification lifetime, so a later manual unfollow cannot cause reconciliation
to enqueue another operation. Revocation deletes the owner's private
import/support/operation ledger after invalidating unwritten work. A later
verification and fresh import therefore creates a new consent lifetime.

Replace the current five-minute actorless notification digest with one event
per successful automatic follow. Its actor is the followed CraftSky profile,
but it has no synthetic AT Protocol source record, system payload, count,
group, or server-selected destination. In-app rows open the actor profile and
reuse the existing Follow/Following control. Identity-free pushes continue to
open the correct account's Notifications feed by client-side category
inference.

Flutter changes remain account-lease scoped. The Instagram page will remove
suggestion state, use verified/verification copy, default to Instagram Export,
use the semantic moss/success color for enabled discovery, disclose automatic
public following, and place confirmed destructive revocation after all import
content.

## 3. Affected Areas

| Area | Existing Pattern | Planned Change | Requirement IDs | Test IDs |
|---|---|---|---|---|
| Background OAuth selection | Request middleware supplies one exact OAuth session ID | Add an exact-owner selector over stored OAuth and active CraftSky sessions, deterministic activity ordering, and narrow invalidation | FR-017, FR-032, NFR-005 | UT-019, IT-024, REG-009 |
| Match/operation ledger | Suggestions plus source support and request-time acceptance | Make matching enqueue deterministic private operations; terminal facts suppress re-creation; no public review semantics | FR-015–FR-019, RULE-008, RULE-012 | UT-006, UT-020, IT-008–IT-011, IT-025, REG-014 |
| PDS follow execution | `followwrite.Service` is shared with suggestion acceptance | Run it from a leased worker using an owner session and stable rkey; preserve ordinary follow behavior | FR-017, FR-032, NFR-005 | IT-009, IT-024, REG-004 |
| Manual-unfollow suppression | Accepted suggestion remains but reconciliation was designed as a review queue | Preserve followed/already-following terminal facts until verification revocation; repeated triggers do not enqueue | FR-017, FR-019, RULE-012 | UT-020, IT-025, REG-014 |
| Notification persistence | `instagramMatch` is actorless, counted, grouped, and coalesced | Store one actorful source-less event per successful operation; remove system/count/group support | FR-020–FR-022 | UT-012–UT-014, IT-001, IT-011, IT-012, IT-021 |
| Push | Delayed coalesced system delivery | Schedule one identity-free delivery after actorful event persistence; infer Notifications destination in Flutter | FR-020–FR-022 | UT-014, IT-012, IT-017, MAN-004 |
| Public AppView API | Three suggestion list/accept/dismiss routes | Unregister routes and policies; keep private ledger unreachable | FR-016, NFR-004 | IT-008, IT-013, IT-021, REG-001, REG-011 |
| Flutter data/state | Suggestion DTO, API methods, repository methods, provider | Delete suggestion surface and its generated files; remove invalidations from imports provider | FR-016, FR-023, NFR-008 | IT-014, IT-015, REG-009, REG-014 |
| Flutter notification model | Social notifications require source fields; match is `SystemNotification` | Introduce actor-bearing/source-less match variant; row opens actor and shows Follow/Following | FR-020, FR-022, FR-026 | UT-011, UT-012, IT-015, IT-017 |
| Instagram Settings UI | Manual input default, People You May Know, revocation within account card, default switch selection color | Default export, remove suggestions, use verification copy/moss success color, move confirmed revoke action to page bottom | FR-024, FR-025, NFR-007 | IT-016, IT-023, MAN-003 |
| Wire corpus/docs | Actorless system fixtures and public suggestion fixtures | Replace with actorful match fixtures and negative removed-surface fixtures | FR-016, FR-020, FR-022 | IT-021, REG-011 |

## 4. Files And Modules

### 4.1 AppView persistence and authentication

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `appview/migrations/000030_instagram_automatic_follows.up.sql` / `.down.sql` | Create | Convert operation states/claim fields to worker semantics; remove obsolete Instagram notification grouping/support; enforce actorful source-less match constraints and per-operation uniqueness | FR-017, FR-020, NFR-005 | IT-001, IT-009, IT-012, REG-010 |
| `appview/internal/auth/background_session_selector.go` | Create | Select candidates by exact owner DID, active unrevoked CraftSky backing session, most-recent activity, stable tie-break; invalidate only an explicitly rejected `(DID, sessionID)` | FR-032 | UT-019, IT-024 |
| `appview/internal/auth/background_session_selector_test.go` | Create | Prove no cross-DID fallback, stable ordering, retryable absence, and narrow invalidation | FR-032, NFR-005 | UT-019, IT-024, REG-009 |
| `appview/internal/auth/store.go` | Change only if needed | Expose existing expiry/inactivity policy to the selector without decoding or logging session JSON | FR-032, NFR-003 | UT-019, UT-015 |
| `appview/internal/db/instagram_migration_test.go` | Change | Test clean up/down bootstrap, operation leases/states, actorful notification constraints, and removal of grouping support | FR-017, FR-020 | IT-001, REG-010 |

The app is pre-production with no active users. The up migration should remove
obsolete pending/dismissed review rows and actorless Instagram digest
events/deliveries rather than fabricate an actor or reinterpret an unaccepted
suggestion as consent. It must not delete any ordinary PDS follow record.
Migration down restores the prior schema shape for testability but does not
attempt to reconstruct removed review/digest history.

The selector query will use `oauth_sessions.account_did = $owner` and require at
least one unrevoked `craftsky_sessions` row for the same composite
`(account_did, oauth_session_id)`. Ordering is:

1. maximum active `craftsky_sessions.last_seen_at` descending;
2. `oauth_sessions.updated_at` descending; and
3. `session_id` ascending as a stable tie-break.

The query never scans another DID as fallback. Provider authentication failure
uses the existing exact-session expiry path before the selector may choose the
next candidate belonging to the same owner.

### 4.2 AppView automatic-follow pipeline

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `appview/internal/instagram/automatic_follow_state.go` | Create | Define internal pending/writing/followed/already-following/invalidated transitions and verification-lifetime suppression | FR-017, FR-019, RULE-012 | UT-020 |
| `appview/internal/instagram/automatic_follow_worker.go` | Create | Claim with `FOR UPDATE SKIP LOCKED`, final-policy check, owner-session rotation, deterministic follow write, retry/backoff, and transactional notification completion | FR-015, FR-017, FR-032, NFR-005 | IT-009, IT-024, IT-025 |
| `appview/internal/instagram/automatic_follow_worker_test.go` | Create | Cover concurrency, last-moment policy changes, session isolation, PDS failure, crash recovery, already-following, and manual suppression | FR-017, FR-032, RULE-012 | IT-009, IT-024, IT-025 |
| `appview/internal/instagram/automatic_follow_store.go` | Create | Expose SQL-backed pair/support storage through operation-oriented methods without list/dismiss/accept suggestion APIs | FR-016–FR-018 | UT-020, IT-008–IT-010 |
| `appview/internal/instagram/suggestions.go` | Delete | Remove member-facing list/accept/dismiss service | FR-016 | IT-008, REG-014 |
| `appview/internal/instagram/automatic_follow_matcher.go` | Create | Upsert source support and queue the stable operation during initial import; terminal same-lifetime rows remain suppression facts | FR-015, FR-017, RULE-008, RULE-012 | UT-006, IT-008, IT-009, IT-025 |
| `appview/internal/instagram/reconciliation.go` | Change | Produce/dedupe private operations only; remove notification activation; preserve targeted future triggers and bounded job leasing | FR-016, FR-019 | IT-008, IT-011, IT-025 |
| `appview/internal/instagram/import_store.go` | Change | Preserve multi-source support; delete cancels only unsupported unwritten work and never terminal public history | FR-018 | IT-007, IT-010 |
| `appview/internal/instagram/account_store.go` | Change | Revocation invalidates unwritten work, deletes imports/support and the full private suppression ledger, and preserves successful PDS follows/notification rows | FR-010, FR-018, RULE-010, RULE-012 | IT-006, IT-010, IT-025 |
| `appview/internal/instagram/account_data.go` | Change | Align scoped export/purge/membership behavior with the operation ledger and remove actorless notification retraction | FR-028 | IT-010, IT-020 |
| `appview/internal/instagram/retention.go` | Change | Retain terminal operation facts for the verification lifetime, not the old suggestion-review retention window | FR-018, FR-028, RULE-012 | IT-010, IT-025 |
| `appview/internal/instagram/eligibility_policy.go` | Change only if needed | Reuse the exact policy at operation creation and immediately before the external write; preserve fail-closed safety | FR-015, FR-017 | UT-006, IT-008, IT-009 |
| `appview/internal/followwrite/service.go` | Change only if needed | Keep deterministic `PutRecord`; expose typed retry/session-expiry classification without embedding Instagram behavior | FR-017, FR-032 | IT-009, IT-024, REG-004 |

The final private storage names are `instagram_automatic_follow_ledger` and
`instagram_automatic_follow_sources`. References from the source table and
`pds_follow_operations` use `automatic_follow_id`. Migration `000031` renames
the originally shipped development schema, including related constraints and
indexes, so suggestion terminology does not remain in the current database
contract.

Partial interface sketch:

```text
type BackgroundSessionSelector interface {
    Select(ctx, ownerDID) -> sessionID | retryable absence
    Invalidate(ctx, ownerDID, sessionID) -> error
}

type AutomaticFollowStore interface {
    ClaimBatch(ctx, limit, leaseOwner, leaseDuration) -> operations
    Retry(ctx, operationID, leaseToken, safeCode, nextAttemptAt)
    CompleteAlreadyFollowing(ctx, operationID, leaseToken)
    CompleteFollowedWithNotification(ctx, operationID, leaseToken, targetDID)
    Invalidate(ctx, operationID, leaseToken, safeReason)
}

AutomaticFollowWorker.ProcessBatch(ctx, limit):
    claim bounded operations
    for each operation:
        re-evaluate shared eligibility
        if already followed: terminal alreadyFollowing, no notification
        if ineligible: invalidate or retry according to policy result
        select exact-owner OAuth session
        deterministic followwrite.Service.Write(owner, target, session, rkey)
        in one DB transaction:
            lock operation
            create actorful event keyed by operation ID if absent
            mark operation followed
            enqueue identity-free push when enabled
```

The notification transaction happens after PDS success. If the process crashes
between those boundaries, retry uses the same rkey, then the unique operation
event key makes completion idempotent.

### 4.3 AppView notifications, push, routes, and lifecycle

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `appview/internal/notifications/instagram_match.go` | Replace | Create one actorful event for one successful operation; no coalescing, count, support attachment, retraction, or delayed release | FR-020, FR-021 | UT-014, IT-011, IT-012 |
| `appview/internal/notifications/service.go` | Change | Add the actorful source-less activation path and preserve preference/newness/idempotency patterns | FR-020, FR-021 | IT-012 |
| `appview/internal/api/notification_store.go` | Change | Hydrate the match actor/profile/current viewer-follow state and emit no system/source fields | FR-020, FR-026 | UT-014, IT-012 |
| `appview/internal/api/notifications.go` | Change | Remove `system` match response and serialize actorful `type: instagramMatch` | FR-020 | IT-012, IT-021 |
| `appview/internal/push/dispatcher.go` | Change | Remove coalescing-close logic; dispatch persisted match events through the normal immediate durable path while keeping payload identity-free | FR-020–FR-022 | UT-014, IT-012 |
| `appview/internal/push/payload.go` | Change only if needed | Ensure match data contains category, stable notification ID, and opaque account binding only | FR-022 | UT-014, IT-012 |
| `appview/internal/api/instagram_suggestions.go` | Delete | Remove list/accept/dismiss handlers | FR-016 | IT-008, IT-021 |
| `appview/internal/routes/routes.go` | Change | Unregister all three suggestion routes | FR-016, NFR-004 | IT-008, IT-013 |
| `appview/internal/routes/policy.go` | Change | Remove suggestion policy entries; retain every verification/account/import policy unchanged | FR-016, NFR-004 | IT-013, REG-001 |
| `appview/internal/app/deps.go` | Change | Remove public suggestion service, wire session selector/follow service/automatic worker, and keep reconciliation as private operation producer | FR-016, FR-017, FR-032 | IT-013, IT-024 |
| `appview/cmd/appview/main.go` | Change | Start/drain `runInstagramAutomaticFollowWorker` with bounded batch/poll/shutdown beside webhook, reconciliation, and retention workers | FR-017, NFR-005 | IT-013 |
| `appview/cmd/appview/server_test.go` | Change | Prove bounded batch, backlog draining, retry cadence, and cancellation | FR-017 | IT-013 |
| `appview/internal/index/bluesky_follow.go` | Change only if required | Preserve normal incoming follow notifications; optionally observe deletion of this worker's deterministic rkey without creating a new operation | RULE-012 | IT-025, REG-004 |
| `appview/internal/notifications/*_test.go`, `appview/internal/api/*notification*_test.go`, `appview/internal/push/*instagram*_test.go` | Change | Replace actorless/count/coalescing expectations with actorful per-operation/history/payload expectations | FR-020–FR-022 | UT-013, UT-014, IT-011, IT-012 |

Successful `instagramMatch` rows are notification history. Revocation, import
deletion, and manual unfollow do not retract them or cancel a validly created
push. Existing actor block/mute/unavailable rules remain the read/delivery
visibility boundary.

### 4.4 Flutter data, providers, notifications, and UI

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `app/lib/instagram_migration/models/instagram_suggestion.dart` and generated mapper | Delete | Remove public suggestion wire/domain types | FR-016 | IT-014, REG-014 |
| `app/lib/instagram_migration/data/instagram_migration_repository.dart` | Change | Remove list/accept/dismiss methods and suggestion import | FR-016 | IT-014 |
| `app/lib/instagram_migration/data/instagram_migration_api_client.dart` | Change | Remove suggestion requests; retain exact fixed-account verification/import contract | FR-016, NFR-008 | IT-014, REG-009 |
| `app/lib/instagram_migration/data/api_instagram_migration_repository.dart` | Change | Remove suggestion forwarding | FR-016 | IT-014 |
| `app/lib/instagram_migration/providers/instagram_suggestions_provider.dart` and generated files | Delete | Remove review selection/actions/pagination state | FR-016 | IT-015, REG-014 |
| `app/lib/instagram_migration/providers/instagram_imports_provider.dart` | Change | Stop invalidating deleted suggestion provider; retain lease-fenced import mutations | FR-016, NFR-008 | IT-015 |
| `app/lib/notifications/models/craftsky_notification.dart` | Change | Add an actor-bearing, source-less `InstagramMatchNotification`; keep social source requirements and inert unknown fallback | FR-020, FR-026 | UT-012, IT-017 |
| `app/lib/notifications/widgets/notification_row.dart` | Change | Render automatic-follow copy/avatar/current relationship and Follow/Following control; row opens target profile | FR-020, FR-026 | UT-012, IT-017 |
| `app/lib/notifications/services/notification_destination_inference.dart` | Change | Map identity-free `instagramMatch` push to Notifications, not Instagram Settings | FR-022 | UT-012, IT-017 |
| `app/lib/notifications/models/notification_open_event.dart` | Change | Validate match push without actor identity/source facts and preserve correct account binding | FR-022, NFR-008 | UT-012, IT-017 |
| `app/lib/instagram_migration/pages/instagram_migration_page.dart` | Change | Remove suggestion card; default export selector; use verified copy and moss switch; move revoke action/dialog to page bottom | FR-024, FR-025 | IT-016, IT-023 |
| `app/lib/l10n/app_en.arb` | Change | Add automatic-follow row/disclosure and verified/revoke copy; remove unused suggestion/digest copy | FR-020, FR-024–FR-026 | IT-016, IT-017 |
| `app/lib/l10n/generated/*` | Generate | Regenerate from ARB; do not hand-edit | NFR-007 | IT-016, IT-017 |
| `docs/changes/2026-07-11-instagram-dm-verification/fixtures/instagram_wire/corpus.json` | Change | Add actorful match success and removed-system/suggestion negative fixtures | FR-016, FR-020, FR-022 | IT-021, REG-011 |

Flutter notification model shape:

```text
CraftskyNotification
├── ActorNotification(actor)
│   ├── SocialNotification(uri, cid, rkey)
│   └── InstagramMatchNotification(no source fields)
└── GenericSystemNotification(unknown/inert fallback only)
```

The row action reuses `_NotificationFollowButton` and the account-scoped
profile relationship providers. It is enabled for `FollowNotification` and
`InstagramMatchNotification`. The existing owner lease check remains ahead of
navigation and mutation.

Provider graph after removal:

```text
activeAccountLeaseProvider
  -> fixedAccountDioProvider(lease)
  -> instagramMigrationRepositoryProvider(lease)
      -> instagramAccountProvider(lease)
      -> instagramVerificationProvider(lease)
      -> instagramImportsProvider(lease)

notificationProvider(owner lease)
  -> actorful InstagramMatchNotification
  -> profile relationship/follow providers(owner account)
  -> target profile route
```

There is no suggestions provider or client-visible automatic-follow progress
provider. Import success reports only the retained import/count contract.

## 5. Services, Interfaces, And Data Flow

### 5.1 Initial match

```text
Flutter local JSON/ZIP/manual parser
  -> normalized following-only import request
  -> ImportService transaction
       -> immutable import + handles
       -> exact eligible match/support upsert
       -> stable PDS operation queued
  -> 201 import + followingCount

AutomaticFollowWorker
  -> final eligibility
  -> exact-owner background session
  -> deterministic PDS putRecord
  -> operation completion + actorful event + push outbox
```

The import response does not wait for external PDS work and exposes no match or
operation count.

### 5.2 Future match

```text
verification/discovery/username/membership/safety restoration trigger
  -> targeted reconciliation job
  -> current retained-handle candidates
  -> shared eligibility policy
  -> source support + stable operation upsert
  -> same AutomaticFollowWorker path
```

Repeated triggers and additive imports converge on one importer/target ledger
row and one deterministic write. An existing followed/already-following row
does not become pending again.

### 5.3 Session rotation and failure

```text
operation owner DID
  -> selector candidates for exactly that DID
  -> ResumeSession(owner, selected session)
       provider-invalid -> expire only owner + selected session, reselect owner
       no owner candidate -> retry operation, no PDS call
       transient PDS error -> retry operation with backoff
       success -> transactional notification completion
```

No error path queries or uses another DID. Session IDs/tokens do not enter the
Instagram tables or diagnostics.

### 5.4 Manual unfollow and revocation

```text
successful operation -> terminal followed ledger
manual PDS unfollow -> ordinary follow disappears
future match triggers -> terminal ledger suppresses requeue

verification revoke
  -> invalidate unwritten work
  -> delete imports, supports, and private operation/suppression ledger
  -> preserve successful public follows and historical notification rows
fresh verification + import -> new authorization may queue a new operation
```

## 6. State, Providers, Controllers, Or DI

### AppView

- `ReconciliationWorker` remains the targeted candidate producer.
- `AutomaticFollowWorker` becomes the sole Instagram PDS writer.
- `BackgroundSessionSelector` belongs in `internal/auth`; it has no knowledge
  of Instagram handles/imports.
- `followwrite.Service` remains the single follow record writer shared with the
  ordinary follow route.
- `notifications.Service` owns event/outbox completion and fixed preference
  semantics.
- `Deps` exposes the worker for process lifecycle only; no HTTP handler receives
  the private operation store.
- Worker claims use opaque lease tokens and `FOR UPDATE SKIP LOCKED`; only the
  current lease may retry, invalidate, or complete an operation.

### Flutter

- Keep the existing family providers keyed by `ActiveAccountLease`.
- Delete suggestion provider state instead of replacing it with worker polling.
- Import creation refreshes only imports/account state that the page actually
  displays.
- Notification Follow/Following mutations use the notification owner's
  account-scoped profile providers.
- Late parser, repository, follow-toggle, and navigation results continue to
  call `ensureInstagramOperationCurrent` or the equivalent owner-lease guard.

## 7. UI, Widgets, Routes, Or User-Facing Surfaces

### Instagram Settings page

The page order becomes:

1. verification/account card;
2. verified-only import composer;
3. retained imports;
4. notification-settings inline text;
5. destructive **Revoke Instagram verification** action at the page bottom.

Detailed changes:

- Use `Verified as @<handle>` and verification terminology in all visible
  states. Internal API/SQL names may retain `link`.
- Keep imports hidden until `account.state == active`.
- Initialize the selector to Instagram Export; manual entry remains available.
- Explain immediately below the selector that current and future exact eligible
  matches are automatically followed publicly and retained until revocation.
- Retain ZIP locality/accounts-followed guidance and the Notification Settings
  inline link.
- Remove People You May Know, selection, accept, dismiss, pagination, and
  related error states.
- Set the enabled discovery switch's active track/thumb treatment from
  `Theme.of(context).extension<SemanticColors>()!.success` or the established
  moss token; do not hard-code green.
- Keep the revoke icon/text in semantic error color. Use the existing CraftSky
  destructive dialog. Cancellation performs no mutation; confirmation submits
  once and explains that imports/authority are deleted while existing follows
  remain.

### Notifications

- Copy must say CraftSky automatically followed the named actor; it must not
  imply that the actor initiated the notification.
- Render the target avatar and bold display identity using the existing row
  composition.
- Show the existing full account-scoped Follow/Following button.
- Row tap opens the target profile.
- Push tap opens the correct account's Notifications page after account
  activation.
- Notification Settings retains fixed `everyone` scope internally, hides the
  scope selector, and allows only push enable/disable.

### Routes

- Retain the typed Flutter `/profile/settings/instagram` page route.
- Remove only the three AppView suggestion endpoints and route-policy entries.
- Keep the retained routes under `/v1/`. The approved removal occurs before
  production and before any active client compatibility obligation, so it does
  not introduce a parallel `/v2/` surface.
- `instagramMatch` no longer navigates to Instagram Settings from either feed
  row or push inference.

## 8. Error, Loading, Empty, And Edge States

| State / Case | Planned Handling | Requirement IDs | Test IDs |
|---|---|---|---|
| No usable owner OAuth session | Keep operation retryable with bounded backoff; no PDS call or notification | FR-017, FR-032 | UT-019, IT-024 |
| Selected session rejected by provider | Invalidate exact owner/session only, then select another owner session | FR-032 | UT-019, IT-024 |
| Other DID has a newer session | Ignore it entirely | FR-032, NFR-005 | UT-019, IT-024, REG-009 |
| Final policy becomes ineligible | Invalidate/leave retryable according to policy; no write/notification | FR-015, FR-017 | UT-006, IT-009 |
| Target already followed | Complete `alreadyFollowing`; no write and no match notification | FR-015, FR-017 | UT-020, IT-009 |
| PDS transient failure | Release lease back to pending with safe code/backoff | FR-017 | IT-009 |
| Crash after PDS success | Retry same rkey; unique transactional completion yields one event | FR-017, FR-020, NFR-005 | IT-009 |
| Duplicate imports/triggers | Add support but preserve one operation/event | FR-018, FR-019 | IT-010, IT-011 |
| Manual unfollow | Current public follow disappears; terminal ledger prevents requeue | RULE-012 | UT-020, IT-025 |
| Verification revoke | Confirm first; invalidate unwritten work and delete private evidence/ledger; preserve successful history | FR-010, FR-018, RULE-010 | IT-006, IT-010, IT-025 |
| Match actor blocked/muted/unavailable | Existing notification visibility policy hides row; history remains stored | FR-020 | IT-012, IT-017 |
| Push disabled | Create in-app row but no provider delivery | FR-021 | UT-013, IT-012 |
| Unknown notification type | Decode/render inert generic fallback; no identity-bearing destination | FR-020, FR-022 | UT-012, IT-021 |
| Account switch during import/follow/open | Discard late result and prevent mutation/navigation under new account | NFR-008 | UT-011, IT-015, IT-017, IT-023 |
| Unverified account page | Show verification controls and basic sync guidance only; no import/suggestion controls | FR-024, FR-025 | IT-016 |

## 9. Test Implementation Plan

| Order | Test IDs | Target | Setup / Fixture | Initial Expected Failure |
|---|---|---|---|---|
| 1 | UT-019, IT-024 | `internal/auth/background_session_selector_test.go`, worker integration | TD-014 exact/cross-owner session matrix | No background selector exists; current writes require a request session |
| 2 | UT-020, IT-025 | `automatic_follow_state_test.go`, reconciliation/worker tests | TD-007/TD-010 terminal ledger, manual deletion, fresh verification | Current suggestion transitions allow review semantics and do not encode the approved suppression contract |
| 3 | IT-001, REG-010 | Migration tests | Fresh/upgraded schema with actorless rows and operation states | Schema still requires actorless system/count/group fields and lacks worker claim constraints |
| 4 | UT-006, IT-008, REG-014 | Eligibility/matcher/routes | TD-006 plus old suggestion route requests | Public routes/services remain and reconciliation activates review notifications |
| 5 | IT-009, REG-004 | Automatic worker/follow service | Final-policy changes, deterministic rkey, PDS crash/failure/already-following | No leased worker completes a follow and notification independently of HTTP |
| 6 | IT-010, IT-011, IT-020 | Import/reconciliation/lifecycle | Multi-source imports, every future trigger, depart/rejoin/revoke | Existing lifecycle is suggestion/digest oriented and does not delete the suppression ledger on revoke |
| 7 | UT-013, UT-014, IT-012 | Notification preference/store/push | TD-008 actorful fixture and identity canaries | Server still persists/serializes/counts/coalesces a system notification |
| 8 | IT-021, REG-011 | Shared Go/Dart corpus | Actorful match plus negative old fields/routes | Corpus still accepts actorless `system` and public suggestion shapes |
| 9 | UT-011, UT-012, IT-014, IT-015, IT-017 | Flutter models/repository/providers/notification row | A/B leases, actor relationship states, removed API symbols | Client still exposes suggestions and treats match as a system destination |
| 10 | IT-016, IT-023 | Instagram page/import privacy widgets | Every page state, JSON/ZIP picker, semantics | Manual is default; suggestions/revoke placement/old copy/color remain |
| 11 | AT-001–AT-009 | AppView/Flutter vertical acceptance targets named in `02-acceptance-tests.md` | Unconfigured, verification, local import, current/future follow, conflict/revoke, notification, membership, and ZIP journeys | Old AT-004/AT-005/AT-007 behavior still expects review or actorless navigation |
| 12 | UT-001–UT-018, IT-002–IT-007, IT-013, IT-018, IT-019, IT-022, IT-023 | Existing completed baseline suites | Existing synthetic Meta/parser/privacy/config/lifecycle fixtures | Must remain green throughout; any failure is a regression, not a reason to weaken the new tests |
| 13 | REG-001–REG-014 | Focused then broad gates | Existing repository regression matrix | Detect API/auth/privacy/cancellation/parser/notification/account-boundary drift |

Red-green-refactor discipline:

- Add only the first failing test group before its production change.
- Keep the owner-session selector independently testable before wiring a PDS
  client or worker.
- Establish database constraints before relying on application-level
  idempotency.
- Do not remove old public APIs/UI until negative route and compile-surface
  tests fail for their presence.
- Regenerate DartMappable/localization output only after source/ARB changes.

Focused commands:

```text
# From appview/
go test ./internal/auth ./internal/instagram ./internal/followwrite
go test ./internal/notifications ./internal/api ./internal/push ./internal/routes ./internal/app

# From app/
dart run build_runner build --delete-conflicting-outputs
flutter gen-l10n
flutter test test/instagram_migration test/notifications test/router test/settings
flutter analyze

# From the repository root
just fmt
just test
```

## 10. Sequencing And Guardrails

- First TDD step: `UT-019` followed by `IT-024`. Prove exact owner-session
  selection before introducing any background PDS call.
- Dependencies:
  1. session selection and operation transitions;
  2. migration constraints;
  3. worker execution and completion;
  4. future triggers/lifecycle;
  5. actorful notification persistence/push;
  6. shared corpus and Flutter model/state;
  7. page polish and broad regressions.
- Guardrails:
  - Owner DID always comes from the stored operation; never from a global
    active session or another row discovered during fallback.
  - Background selector predicates and invalidation use the composite
    `(account_did, session_id)`.
  - No raw handle, username, IGSID, token, session ID, import evidence, or
    provider body enters logs/Sentry/metrics/push/PDS records.
  - The shared eligibility policy runs at creation and immediately before the
    external write.
  - Only a successful deterministic write may create `instagramMatch`.
  - Already-following creates neither a match event nor a false push.
  - Successful follows/history are never deleted by Instagram lifecycle.
  - No member-facing endpoint serializes internal operation or support facts.
  - No new lexicon or PDS record type is introduced.
  - Preserve current verification, webhook, ZIP streaming/isolate, strict
    following-only import, and fixed-account behavior.
- Out of scope:
  - Meta app/dashboard setup and production enablement;
  - additional real export fixtures;
  - repository-wide member export/deletion UI;
  - Flutter web ZIP support;
  - automatic conflict adjudication;
  - progress UI for private background operations.

## 11. Risks And Open Questions

| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Non-blocking | A stored OAuth row can exist after its last CraftSky bearer session is revoked | Worker might use a session the member no longer actively retains | Selector requires an unrevoked CraftSky backing session and orders by its last activity; exact provider invalidation removes only that OAuth session |
| CPQ-002 | Non-blocking | PDS succeeds before DB completion | Retry could otherwise duplicate follow/notification | Stable rkey plus unique operation-keyed event and transactional completion |
| CPQ-003 | Non-blocking | Existing schema/table names contain “suggestion” | Internal naming may be confusing | Keep table names only where migration churn has no value; remove public/domain suggestion types and document the legacy internal name |
| CPQ-004 | Non-blocking | Follow deletion arrives through the firehose after manual unfollow | Reconciliation could see the target as unfollowed | Terminal same-lifetime ledger, not current follow existence, is the suppression authority |
| CPQ-005 | Non-blocking release gate | Live Meta payload/access/token/reply behavior remains unverified | Production verification may differ from fixtures | MAN-001 remains mandatory; integration stays disabled by default |
| CPQ-006 | Non-blocking release gate | Physical push and mobile ZIP memory/path behavior are not fully hermetic | Device lifecycle regressions could escape tests | MAN-003–MAN-005 remain release checks |
| CPQ-007 | Blocking | None identified for implementation | — | Approved requirements/tests and this plan provide the required decisions |

## 12. Handoff To TDD Builder

- Coding plan: `04-coding-plan.md`
- TDD execution evidence: update `05-implementation-plan.md` from this plan;
  do not retain its superseded suggestion/digest steps.
- Start with test: `UT-019` in
  `appview/internal/auth/background_session_selector_test.go`, followed by
  `IT-024`.
- First focused command:
  `cd appview && go test ./internal/auth ./internal/instagram`.
- Then implement `UT-020`/`IT-025` before altering reconciliation behavior.
- Treat `01-requirements.md`, `02-acceptance-tests.md`, and this document as
  authoritative over `design-plan.md` and older implementation artifacts.
- Production enablement remains blocked on MAN-001 and the applicable device
  checks. Run MAN-001–MAN-005 at their documented release stages. No commit,
  push, Meta configuration, or production mutation is authorized by this plan.

## 13. AppView Audit Correction Plan

The correction lands with the coordinated owner-lifecycle/session work. The
central migration owner assigns its final sequence number; the Instagram pair
is named `<next>_instagram_private_suggestions.up.sql` and
`<next>_instagram_private_suggestions.down.sql` until that reservation is made.

### 13.1 Failure-first tests

| Test | File | Required failure before implementation |
|---|---|---|
| UT-021, UT-022 | `appview/internal/instagram/suggestion_state_test.go` | Current storage cannot express private pending/dismissed/accepted state with participant generations. |
| UT-023 | `appview/internal/app/instagram_capability_test.go` | Current dependency graph can construct a background session selector and PDS writer. |
| IT-026 | `appview/internal/instagram/private_suggestion_store_test.go` | Matching currently queues a public-write operation rather than stopping at a private suggestion. |
| IT-027 | `appview/internal/instagram/suggestion_acceptance_test.go` | There is no current-member, owner-effect-fenced suggestion acceptance boundary. |
| IT-028 | `appview/internal/notifications/instagram_match_test.go`, `appview/internal/routes/instagram_routes_test.go` | Automatic-follow notification/schema behavior and removed suggestion routes still reflect the superseded contract. |
| IT-029 | `app/test/instagram_migration/instagram_suggestions_test.dart` | Flutter no longer has a private suggestion review surface. |
| REG-015 | `appview/internal/app/instagram_capability_test.go` | Worker startup/wiring still exposes session/PDS/follow capabilities. |

### 13.2 Schema and stores

- Rewrite the current development-only automatic-follow schema into a private
  suggestion contract. Keep source support and verification-lifetime
  suppression, add importer and target owner generations, and constrain the
  states from Requirements §24.3.
- Remove/reset obsolete `pds_follow_operations` public-write lease/retry state
  and delete pre-production `instagramMatch` rows/preferences. There are no
  production users and no compatibility backfill.
- Replace public-write methods in
  `appview/internal/instagram/automatic_follow_store.go` with narrow suggestion
  create/list/accept/dismiss/invalidate transaction methods, or rename the file
  to `suggestion_store.go` in the same breaking change.
- Refactor `automatic_follow_matcher.go` and `reconciliation.go` to persist only
  private suggestions with both generations. Neither type accepts a session or
  writer dependency.

### 13.3 Server capability boundary

- Delete `appview/internal/instagram/automatic_follow_worker.go` and its retry,
  recovery, and background-write tests.
- Remove its construction from `appview/internal/app/deps.go` and its runner
  from `appview/cmd/appview/main.go` and related server tests.
- Remove Instagram's use of
  `appview/internal/auth/background_session_selector.go`. Delete that selector
  if no non-Instagram caller remains.
- Remove Instagram's `followwrite.Service` dependency. Delete
  `appview/internal/followwrite/` if no ordinary route uses it after the shared
  owner-effect executor lands.
- Add/restore `appview/internal/instagram/suggestions.go` and
  `appview/internal/api/instagram_suggestions.go`. Accept invokes the shared
  ordinary follow effect through a narrow interface and receives no raw PDS
  factory or client.
- Register current-member-only GET/list, POST/accept, and DELETE/dismiss
  policies in `appview/internal/routes/policy.go` and handlers in
  `appview/internal/routes/routes.go`.
- Remove the dedicated automatic `instagramMatch` creator, preference, push,
  and rendering paths from `appview/internal/notifications/`, the notification
  API/store, and `appview/internal/push/`. Preserve ordinary incoming-follow
  behavior.

### 13.4 Flutter boundary

- Restore a private suggestion model, repository/API methods, Riverpod
  provider, and a suggestion section in the existing Instagram page under
  `app/lib/instagram_migration/`.
- Follow and Dismiss use a captured `AccountSessionLease`; late results cannot
  navigate or mutate under another active account.
- Replace automatic-follow import/revocation/notification copy in
  `app/lib/l10n/app_en.arb`, regenerate localization output, and remove the
  `instagramMatch` notification rendering/preference branch.
- Keep the completed file-backed ZIP parser, exact normalization, verification
  UX, discovery controls, and bottom revocation placement unchanged.

### 13.5 TDD order and gate

1. UT-023/REG-015: remove capability from composition.
2. UT-021/UT-022 and the migration: establish suggestion state/generation.
3. IT-026: stop matching at private persistence.
4. IT-027: implement explicit owner-fenced accept/dismiss.
5. IT-028: retire automatic notification and restore route inventory.
6. IT-029: restore the fixed-account Flutter review surface.
7. Run `go test -race` for `internal/instagram`, `internal/app`,
   `internal/routes`, `internal/notifications`, and the shared owner/session
   packages, then the full Go/Flutter gates.
