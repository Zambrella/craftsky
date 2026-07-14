# Coding Plan: AppView Push Notifications

## 1. Inputs

- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` — Approved, High risk (re-reviewed 2026-07-14)
- Repository guidance: `AGENTS.md`
- Architecture references:
  - `atproto-craft-social-app-reference.md`
  - `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md`
  - `docs/superpowers/specs/2026-04-22-api-wire-alignment-design.md`
- Existing implementation inspected:
  - Notification handlers/store and route policies under `appview/internal/api/` and `appview/internal/routes/`
  - Follow, interaction, post, profile, dispatcher, and Tap consumer paths under `appview/internal/index/` and `appview/internal/tap/`
  - App dependency/configuration and process lifecycle under `appview/internal/app/` and `appview/cmd/appview/`
  - Session/logout, observability, migrations, and real-Postgres test helpers
- Approval gate: the user explicitly approved the account-wide newness REST design, document updates, and AppView implementation on 2026-07-14.

## 2. Implementation Strategy

Implement one durable, private AppView notification subsystem with four boundaries:

1. `internal/notifications` owns the closed category model, preferences, eligibility/classification, durable event lifecycle, installation/account-subscription lifecycle, and transaction-aware ingestion API.
2. Existing Tap indexers continue to own their source-row transactions, but call the notification service with the same `pgx.Tx` before commit. This makes source mutation, notification activation/retraction, and delivery fan-out one atomic unit and keeps FCM completely off the Tap acknowledgement path.
3. `internal/api` exposes the durable list/resolution, preference, and device contracts through existing `/v1/` middleware and route-policy conventions.
4. `internal/push` claims per-subscription outbox rows in bounded batches, resolves the installation's current token at send time, sends through an injected `Sender`, and records a terminal or retry state. The production adapter uses the official Firebase Admin Go SDK; automated tests use fakes only.
5. Notification newness remains a small account-scoped layer over durable events: each genuine activation receives a monotonic revision, `GET /v1/notifications/new-count` counts active listable revisions after the account marker, and `POST /v1/notifications/seen` advances that marker through a captured database snapshot.

The first implementation slice is deliberately narrow: define categories, add the migration and query/index contract, then make one like create/delete path transactional end to end. Only after that slice is green should the same seam expand to repost, follow, reply, mention, and quote producers. This preserves the sequencing requested by `03-document-review.md` DR-004.

No source table is backfilled. Durable notification history begins only when new live or newly processed source record events pass through the updated indexers after deployment. No lexicon changes are required.

## 3. Affected Areas

| Area | Existing Pattern | Planned Change | Requirement IDs | Test IDs |
|---|---|---|---|---|
| Category and policy model | Notification types are constants inside `api/notification_store.go`; no preferences | Move the closed seven-category registry and pure preference/eligibility/classification rules into `internal/notifications` | BR-003, FR-007, FR-008, FR-017, FR-019, FR-020, RULE-001, RULE-002, RULE-005, RULE-006 | UT-001–UT-004, AT-002–AT-004 |
| Persistence | Notification feed is a read-time SQL union over mutable source tables | Add durable events/tombstones, preferences, installations, subscriptions, and deliveries in migration `000021`; generate typed queries for the new subsystem | FR-001–FR-005, FR-009–FR-011, FR-018, FR-021, FR-025, FR-029, FR-033, NFR-005 | IT-001–IT-006, IT-008–IT-012, IT-018, IT-020–IT-021, IT-028, IT-031, REG-005 |
| Tap producer transactions | Each indexer directly opens/commits its own transaction; several delete paths use `pool.Exec` | Pass one `pgx.Tx` to a shared notification lifecycle service before source commit; convert relevant delete paths to transactions | FR-002, FR-003, FR-017–FR-021, NFR-001 | IT-003–IT-005, IT-010, IT-018, IT-020, AT-001–AT-006, REG-002 |
| Permanent actor deletion | Tap identity events are decoded but dropped; Craftsky profile deletion removes membership | Decode typed Tap identity events and route only terminal `status: deleted` to actor hard-deletion; leave deactivated/suspended/taken-down behavior unchanged | FR-023, FR-030 | IT-022, REG-006 |
| Durable reads/resolution | `PostStore.ListNotifications` derives and hydrates five categories | Add a dedicated `NotificationStore` that lists active events and resolves owned active/retracted IDs, with batched current-data hydration and visibility checks | FR-004, FR-005, FR-022, FR-023, FR-026, FR-032 | UT-005, IT-006, IT-017, IT-019, IT-024, REG-001, REG-003 |
| Preference APIs | None | Add authenticated GET and PATCH endpoints with effective defaults and partial merge semantics | FR-006–FR-008, RULE-002, RULE-005 | UT-002, UT-003, IT-007, IT-010, IT-025, AT-007 |
| Device/subscription APIs | Device ID exists only as middleware/session instrumentation | Add idempotent registration and owner-scoped removal; separate installation token ownership from account authorization | FR-009, FR-010, FR-016, FR-029, FR-033 | IT-008, IT-011, IT-012, IT-021, IT-023, IT-031, AT-008, AT-009 |
| Logout cleanup | Session revocation does not affect push state | Add fail-closed push-subscription cleanup for ordinary and all-session logout without affecting other local accounts | FR-016 | IT-011, IT-021, REG-008 |
| Push delivery | No worker or provider dependency | Add claim/lease dispatcher, retry/deadline policy, minimal payload builder, Firebase adapter, and graceful lifecycle | FR-012–FR-015, FR-024, FR-027, FR-028, FR-031, RULE-004, NFR-001, NFR-004 | UT-006, UT-007, UT-009, IT-013–IT-016, IT-023, IT-025, IT-026, IT-030, REG-004, REG-007 |
| Observability/privacy | Existing observer has safe logs, Sentry, and metrics abstractions | Extend the recorder/observer with bounded push metrics and sanitized outcome logging; never attach tokens, credentials, IDs, or payload content | NFR-002, NFR-006 | UT-008, IT-027, IT-029 |
| Route contract | Central route-policy table and standard v1 middleware | Register notification routes with read/write policies, camelCase JSON, standard errors, auth, device ID, rate/body limits | NFR-003, FR-006, FR-009, FR-026 | AT-007, IT-007, IT-008, IT-024 |
| Notification newness | Durable rows have activity/index timestamps but no acknowledgement state | Add monotonic activation revisions, one account-wide high-water marker, a read-only count route, and an explicit snapshot-safe mark-seen route | BR-004, FR-034–FR-038, RULE-007, NFR-007 | AT-010, IT-032–IT-035, REG-009 |

## 4. Files And Modules

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `appview/migrations/000021_appview_notifications.up.sql` | Create | Create five private notification tables, checks, uniqueness constraints, claim/pagination indexes, and no backfill | FR-001, FR-005, FR-007, FR-009–FR-015, FR-018, FR-021, FR-025, FR-029, FR-033, NFR-005 | IT-028, REG-005, REG-006 |
| `appview/migrations/000021_appview_notifications.down.sql` | Create | Drop the new tables in dependency order | Same as migration | IT-028 |
| `appview/migrations/000022_notification_newness.up.sql` / `.down.sql` | Create | Add the monotonic revision sequence/column, account acknowledgement table, active count index, and all-state high-water index without rewriting applied migration 000021 | FR-034–FR-038, NFR-007 | IT-032, IT-033 |
| `appview/sqlc.yaml` | Create | Activate the repository's intended `sqlc` model/query generation for the new subsystem, using migration schemas and `pgx/v5` | FR-001–FR-016, FR-021, NFR-005 | Build/compile guard for all store tests |
| `appview/queries/notifications.sql` | Create | Typed event activation/retraction, list/resolution, eligibility snapshot, and hydration-key queries | FR-001–FR-005, FR-018, FR-021–FR-023, FR-026, FR-030 | IT-001–IT-006, IT-017–IT-020, IT-022, IT-024 |
| `appview/queries/notification_preferences.sql` | Create | Effective preference reads and partial per-category upserts | FR-006–FR-008 | IT-007, IT-010, IT-025 |
| `appview/queries/push_installations.sql` | Create | Installation registration/rotation/rebind, subscription activation/deactivation, logout cleanup | FR-009, FR-010, FR-014, FR-016, FR-029, FR-033 | IT-008, IT-011, IT-012, IT-014, IT-021, IT-031 |
| `appview/queries/push_deliveries.sql` | Create | Initial fan-out, bounded claim, lease recovery, cancellation, success/retry/failure/expiry transitions, queue statistics | FR-011–FR-015, FR-021, FR-027, FR-028, FR-031 | IT-002, IT-013–IT-016, IT-018, IT-023, IT-025, IT-026, IT-029, IT-030 |
| `appview/internal/models/*` | Generate | `sqlc` row/query types; no hand-written production logic | Same as query files | Compile guard |
| `justfile` | Change | Add a deterministic `sqlc` generation recipe and optional drift check; do not alter lexgen | NFR-005 | Build/CI guard |
| `appview/internal/notifications/category.go` | Create | Closed category/scope types, validation, ordering, and wire values | BR-003, RULE-005, RULE-006 | UT-001 |
| `appview/internal/notifications/preferences.go` | Create | Default resolution and partial patch merge/validation | FR-006–FR-008, RULE-002 | UT-002, IT-007 |
| `appview/internal/notifications/eligibility.go` | Create | Pure event-time eligibility decision with self/follow/scope/push inputs | FR-008, FR-020, FR-027, RULE-001, RULE-002 | UT-003, IT-010, IT-025 |
| `appview/internal/notifications/classify.go` | Create | Per-recipient reply > quote > mention selection and direct like/repost recipient rules | FR-017, FR-019, FR-020, RULE-006 | UT-004, AT-002–AT-004 |
| `appview/internal/notifications/service.go` | Create | Transaction-aware activation/retraction/fan-out and actor-deletion orchestration | FR-001–FR-003, FR-011, FR-018, FR-021, FR-023, FR-030 | IT-001–IT-005, IT-018, IT-020, IT-022 |
| `appview/internal/notifications/service.go` | Change | Allocate a new revision only on inserted/genuinely updated activation; leave replay and retraction revisions unchanged | FR-034 | IT-032 |
| `appview/internal/notifications/store.go` | Create | Wrap generated queries, UUID creation, clock, and transaction boundaries for APIs/devices | FR-001, FR-005–FR-016, FR-026, FR-033 | IT-006–IT-017, IT-021, IT-024, IT-031 |
| `appview/internal/notifications/*_test.go` | Create | Unit tests UT-001–UT-004 and focused store tests | Linked requirements above | UT-001–UT-004 |
| `appview/internal/index/craftsky_interaction.go` | Change | Accept transaction-aware notification service; activate/retract likes and reposts inside source transaction | FR-002, FR-003, FR-017, FR-018, FR-021 | AT-002, AT-003, IT-003–IT-005, IT-020, REG-002 |
| `appview/internal/index/craftsky_like.go` | Change | Inject notification service into shared interaction handler | Same as interaction | Same as interaction |
| `appview/internal/index/craftsky_repost.go` | Change | Inject notification service into shared interaction handler | Same as interaction | Same as interaction |
| `appview/internal/index/bluesky_follow.go` | Change | Preserve source delete/reactivation semantics in one transaction and apply mutual-follow eligibility | FR-003, FR-008, FR-018, FR-021 | IT-004, IT-005, IT-010, IT-020 |
| `appview/internal/index/craftsky_post.go` | Change | Classify reply/quote/mention recipients after post/mention materialization and retract affected events before post deletion commit | FR-003, FR-019–FR-023 | AT-004, IT-004, IT-005, IT-017–IT-019 |
| `appview/internal/index/craftsky_profile.go` | Change | Hard-delete caused notifications and unsent deliveries in the same membership deletion transaction | FR-023, FR-030 | IT-022 |
| `appview/internal/index/notification_*_test.go` | Create | Real-Postgres acceptance/integration suites named in `02-acceptance-tests.md` | FR-001–FR-003, FR-008, FR-017–FR-023, FR-025, FR-030–FR-032 | AT-001–AT-006, IT-001–IT-005, IT-010, IT-017–IT-020, IT-022, IT-026 |
| `appview/internal/tap/consumer.go` | Change | Decode typed identity payloads and dispatch only terminal account deletion to an injected lifecycle handler; retain record flow | FR-030 | IT-022, consumer regression |
| `appview/internal/tap/consumer_test.go` | Change | Prove `deleted` is surfaced and active/deactivated/suspended/taken-down events do not trigger hard deletion | FR-023, FR-030 | IT-022, REG-006 |
| `appview/internal/api/notifications.go` | Change | Use durable store; add stable `id`, quote/type metadata, bounded hydration response mapping | FR-004, FR-005, FR-022, FR-023 | UT-005, IT-006, IT-019, REG-001, REG-003 |
| `appview/internal/api/notification_store.go` | Change | Remove the derived union implementation; retain/move only reusable response hydration helpers | FR-004, FR-022, FR-023 | IT-006, IT-019, REG-001 |
| `appview/internal/api/notification_resolution.go` | Create | Owner-only active/tombstone resolution with precise/fallback target and non-enumerating 404 | FR-023, FR-026, FR-030 | IT-017, IT-024 |
| `appview/internal/api/notification_preferences.go` | Create | GET effective defaults and PATCH partial category values | FR-006–FR-008 | IT-007, AT-007 |
| `appview/internal/api/notification_devices.go` | Create | POST registration/rebind and DELETE owned account subscription without token echo | FR-009, FR-010, FR-016, FR-029, FR-033 | IT-008, IT-011, IT-012, IT-021, IT-031, AT-008, AT-009 |
| `appview/internal/api/notification_newness.go` | Create | Expose camelCase count and bodyless 204 mark-seen handlers through a narrow store interface | BR-004, FR-035–FR-038, RULE-007 | AT-010, IT-033–IT-035, REG-009 |
| `appview/internal/api/notification_store.go` | Change | Add indexed count and transaction-scoped greatest-revision acknowledgement methods sharing the list actor-visibility predicate | FR-035, FR-036, NFR-007 | IT-033, IT-034, REG-009 |
| `appview/internal/api/notification_newness_test.go` | Create | Real-Postgres count, first-use, visibility, replay/reactivation, snapshot, and account-isolation coverage | FR-034–FR-038, RULE-007, NFR-007 | IT-032–IT-035, REG-009 |
| `appview/internal/api/notification_*_test.go` | Create/Change | HTTP/store/response tests listed in the acceptance specification | FR-004–FR-010, FR-022, FR-023, FR-026, FR-033 | UT-005, IT-006–IT-008, IT-017, IT-019, IT-021, IT-024, IT-031 |
| `appview/internal/auth/handlers.go` or current constructor file | Change | Accept a narrow push-subscription cleanup dependency | FR-016 | IT-011, IT-021, REG-008 |
| `appview/internal/auth/handlers_session.go` | Change | Clean current-device subscription before ordinary session revoke; clean all account subscriptions before all-session revoke | FR-016 | IT-011, IT-021, REG-008 |
| `appview/internal/routes/policy.go` | Change | Add policy entries for list resolution, preferences, registration, and removal | NFR-003 | AT-007, route policy regression |
| `appview/internal/routes/routes.go` | Change | Register new handlers using existing auth/device/body/rate middleware and inject dedicated stores | FR-006, FR-009, FR-026, NFR-003 | AT-007, IT-007, IT-008, IT-024 |
| `appview/internal/routes/policy.go`, `routes.go` | Change | Register literal `/new-count` read and `/seen` write routes ahead of the notification-ID surface with standard v1 middleware | FR-035, FR-036, NFR-003 | AT-010, IT-035, REG-009 |
| `appview/internal/routes/notification_routes_test.go` | Create | Exercise auth, device ID, invalid JSON, camelCase envelope, and cross-owner 404 through the mux | NFR-003, FR-026 | AT-007 |
| `appview/internal/push/sender.go` | Create | Narrow provider-neutral sender request/result/error contract | FR-012–FR-015, NFR-004 | UT-009, IT-013–IT-016 |
| `appview/internal/push/firebase_sender.go` | Create | Firebase Admin Go adapter using one token per delivery and typed error classification | FR-012–FR-014, FR-024, FR-028 | UT-007, UT-009, manual MAN-001/MAN-002 |
| `appview/internal/push/payload.go` | Create | Minimal combined notification-and-data payload with platform TTL/expiry | FR-024, FR-028, FR-029 | UT-007, IT-016, IT-023 |
| `appview/internal/push/retry.go` | Create | Injected-clock exponential backoff, deterministic jitter seam, six-hour deadline | FR-013, FR-028 | UT-006, IT-013, IT-016 |
| `appview/internal/push/dispatcher.go` | Create | Poll/claim/send/finalize loop with leases, batch bounds, provider timeout, and cancellation | FR-012–FR-015, RULE-004, NFR-001, NFR-006 | IT-013–IT-016, IT-029, IT-030, REG-007 |
| `appview/internal/push/*_test.go` | Create | Payload, retry, dispatcher, routing, and concurrency tests named by `02-acceptance-tests.md` | FR-012–FR-015, FR-024, FR-028, FR-029, RULE-004 | UT-006, UT-007, IT-013–IT-016, IT-023, IT-030 |
| `appview/internal/observability/metric_recorder.go` | Change | Add push queue/outcome and notification decision recorder methods with allowlisted labels | NFR-002, NFR-006 | UT-008, IT-029 |
| `appview/internal/observability/observer.go` plus `push.go` | Change/Create | Safe facade methods for push logs/metrics; no token/payload parameters | NFR-002, NFR-006 | UT-008, IT-027, IT-029 |
| `appview/internal/observability/push_test.go` | Create | Validate redaction, low-cardinality values, queue/outcome metrics | NFR-002, NFR-006 | UT-008, IT-029 |
| `appview/internal/app/config.go` | Change | Add validated push enablement, Firebase project ID, batch/poll/lease/send timeout defaults; credentials remain ADC-managed | FR-012, NFR-004 | UT-009 |
| `appview/internal/app/deps.go` | Change | Wire generated queries, notification service/store, fake/disabled/production sender, and dispatcher | FR-003, FR-012, NFR-004 | UT-009, REG-004 |
| `appview/internal/app/push_config_test.go` | Create | Validate production enabled/disabled/fake modes without provider calls | NFR-004 | UT-009 |
| `appview/cmd/appview/main.go` | Change | Run dispatcher beside Tap and HTTP, cancel all workers on listener failure/signal, wait with bounded shutdown | FR-012, NFR-001 | REG-004, IT-030 |
| `appview/cmd/appview/server_test.go` | Change | Preserve server behavior when push is disabled and injected fake lifecycle is used | FR-012, NFR-004 | REG-004 |
| `appview/go.mod`, `appview/go.sum` | Change | Add official `firebase.google.com/go/v4` Admin SDK and direct generation tooling only if required by the selected `sqlc` invocation | FR-012–FR-014, NFR-004 | Compile/config tests |
| `appview/environments/dev.env.example` and production environment documentation if present | Change | Document non-secret `PUSH_ENABLED` and Firebase project ID; point credential loading to ADC/secret mount without committing credentials | NFR-002, NFR-004 | UT-009, MAN-001, MAN-002 |

No files under `lexicon/` or `app/` are changed in this stage. The existing Flutter notification decoder does not yet accept `quote`; production enablement must wait for the separate client pass noted under risks.

## 5. Services, Interfaces, And Data Flow

### 5.1 Durable schema contract

Use application-generated UUID strings for opaque notification, installation, and routing identifiers. Keep atproto identifiers as validated typed values at Go boundaries and store their canonical text forms in Postgres. Do not add foreign keys from durable notification references to mutable source rows: explicit lifecycle processing must preserve tombstones after source deletion.

Planned tables:

```text
notification_events
  id UUID primary key
  recipient_did TEXT not null
  actor_did TEXT not null
  category TEXT not null check closed category set
  subject_key TEXT not null
  source_uri/source_cid/source_rkey TEXT not null
  subject_uri/subject_cid TEXT null
  parent_uri/parent_cid TEXT null
  root_uri/root_cid TEXT null
  quoted_uri/quoted_cid TEXT null
  eligibility_scope TEXT not null
  recipient_followed_actor BOOLEAN not null
  push_enabled_snapshot BOOLEAN not null
  state TEXT not null check active|retracted
  first_activity_at/activity_at/indexed_at TIMESTAMPTZ not null
  initial_push_evaluated_at TIMESTAMPTZ not null
  retracted_at/retraction_reason TIMESTAMPTZ/TEXT null
  newness_revision BIGINT not null default nextval(notification_newness_revision_seq)
  unique(recipient_did, actor_did, category, subject_key)

notification_seen_state
  account_did TEXT primary key
  last_seen_revision BIGINT not null default 0
  updated_at TIMESTAMPTZ not null

notification_preferences
  account_did/category/scope/push_enabled/created_at/updated_at
  primary key(account_did, category)

push_installations
  id UUID primary key, device_id, platform, fcm_token, active, created_at, updated_at, deactivated_at
  unique(device_id)
  partial unique(fcm_token) where active

push_account_subscriptions
  id UUID primary key, installation_id, account_did, routing_id UUID, active, created_at, updated_at, deactivated_at
  unique(installation_id, account_did)
  unique(routing_id)

push_deliveries
  id UUID primary key, notification_id, account_subscription_id
  status pending|leased|retry|succeeded|permanent_failure|expired|cancelled
  attempts, next_attempt_at, deadline_at, lease_owner, lease_expires_at
  provider_result_class, created_at, updated_at, sent_at
  unique(notification_id, account_subscription_id)
```

`subject_key` is a semantic coalescing identity, not necessarily the current source URI:

- `follow`: the followed recipient DID, so a new follow record URI reactivates the same relationship.
- `like` / `repost`: the underlying authored post URI.
- `reply` / `mention` / `quote`: the authored source post URI; the same post can still produce separate rows for different recipients, with per-recipient canonical classification.

`initial_push_evaluated_at` is set on first insertion even when push is disabled or the recipient has no active subscriptions. Reactivation updates `activity_at`, source identity, and lifecycle state but never performs fan-out again. An exact replay of the active current `(source_uri, source_cid)` is a no-op and does not move the row to the top.

`newness_revision` is allocated from a Postgres sequence on first insertion and on the same genuine activation updates that already move the row to the feed top. Exact replay and retraction retain it. `notification_seen_state` is keyed only by account DID. Mark-seen runs in one transaction: capture the greatest currently committed revision for the account, then upsert `GREATEST(existing, captured)`; a later transaction's revision remains above the marker.

Required indexes include:

- Active feed: `(recipient_did, activity_at DESC, id DESC) WHERE state = 'active'`.
- Source lifecycle lookup: `(source_uri)` and destination reference lookups needed for post deletion.
- Actor hard deletion: `(actor_did)`.
- Preferences: primary key `(account_did, category)`.
- Installation/token ownership: unique `device_id`, unique active/current `fcm_token` invariant.
- Active subscriptions/fan-out: `(account_did, active)` and `(installation_id, active)`.
- Claiming: `(next_attempt_at, id)` for `pending/retry`, plus `lease_expires_at` for recovery.
- Queue age: partial index on `created_at` for claimable statuses.

Migration tests assert columns, checks, unique constraints, supporting indexes, absence of source-row cascading, and that applying `000021` over pre-existing source activity creates zero notification/delivery rows. For IT-028, prefer schema/index inspection plus `EXPLAIN (FORMAT JSON)` assertions for feed and claim shapes. This avoids introducing a pgx round-trip tracer solely for this feature; query APIs are also batch-shaped so a page/batch does not perform item-count-dependent queries.

### 5.2 Core transaction-aware service

```text
type Lifecycle interface {
  Activate(ctx, tx, Candidate) (ActivationResult, error)
  RetractBySource(ctx, tx, sourceURI, reason) error
  RetractByDestination(ctx, tx, destinationURI, reason) error
  HardDeleteByActor(ctx, tx, actorDID) error
}

type Candidate struct {
  RecipientDID, ActorDID syntax.DID
  Category Category
  SubjectKey string
  Source SourceRef
  Subject, Parent, Root, Quoted optional SourceRef
  ActivityAt time.Time
}
```

`Activate` performs, inside the caller's transaction:

1. Reject self activity and missing required subject/recipient relationships.
2. Load the recipient's effective category preference and the one recipient->actor follow snapshot.
3. Apply scope. If rejected, record only a safe suppression outcome; create no notification or delivery.
4. Lock the semantic notification key.
5. If absent, insert one active notification with eligibility snapshots and mark initial push evaluation complete. If `pushEnabled`, insert one delivery per currently active subscription using `INSERT ... SELECT ... ON CONFLICT DO NOTHING`.
6. If the exact active source/CID is replayed, return a duplicate/no-op result.
7. If retracted, reactivate the same ID, refresh the current references and `activity_at`, but do not insert deliveries.

`RetractBySource`/`RetractByDestination` mark active rows retracted and cancel only unsent states (`pending`, `retry`, and safely reclaimable `leased`). They never claim an FCM-accepted push was recalled. `HardDeleteByActor` cancels/deletes unsent delivery rows and deletes caused notifications; successful/terminal rows disappear through the notification foreign-key policy so stale IDs become indistinguishable from unknown IDs.

### 5.3 Producer data flow

```text
Tap record event
  -> existing indexer validates membership and record
  -> begin pgx transaction
  -> mutate source/materialized rows
  -> derive candidate recipients/references in the same transaction
  -> notification Lifecycle.Activate/Retract(..., tx, ...)
  -> commit
  -> return to Tap consumer
  -> Tap acknowledgement

Separate dispatcher loop
  -> claim bounded delivery batch with FOR UPDATE SKIP LOCKED
  -> commit lease
  -> resolve current active subscription + installation token + actor display name
  -> build minimal message with remaining TTL
  -> Sender.Send with per-attempt timeout
  -> transactionally finalize success/retry/permanent/expired
```

The first vertical slice uses `CraftskyLike` because its recipient is an indexed post author and its undo path already soft-deletes the source. The service must be proven with forced rollback immediately before and after activation/retraction writes before adapting other producers.

Post classification runs after the post and mention materialization are visible inside `CraftskyPost`'s transaction. Build recipient candidates from direct parent authorship, quoted-post authorship, and materialized mentions; group by recipient; exclude the actor; then select `reply > quote > mention` independently for each recipient. A subject missing from the local index produces no candidate or later reconciliation.

For profile/account deletion:

- Craftsky profile record deletion invokes `HardDeleteByActor` in the existing membership deletion transaction before profile rows are removed.
- The Tap consumer stops dropping every identity envelope. It validates the identity DID at the WebSocket boundary and forwards only `status == "deleted"` to the same hard-delete service in its own transaction.
- `active`, `deactivated`, `suspended`, and `takendown` identity statuses do not hard-delete notifications in this pass; temporary lifecycle policy is explicitly deferred.

### 5.4 API stores and contracts

```text
GET /v1/notifications?limit=&cursor=
  -> {items: [...], cursor?: "..."}

GET /v1/notifications/new-count
  -> {newCount: 0}

POST /v1/notifications/seen
  -> 204 No Content

GET /v1/notifications/{notificationId}
  -> owner-only active/retracted resolution

GET /v1/notifications/preferences
  -> {preferences: {like: {scope, pushEnabled}, ... seven categories}}

PATCH /v1/notifications/preferences
  <- {preferences: {like?: {scope?, pushEnabled?}, ...}}
  -> full effective seven-category response

POST /v1/notifications/devices
  <- {platform: "ios"|"android", token: "..."}
  -> {accountSubscriptionId: "opaque-installation-local-id"}

DELETE /v1/notifications/devices/{accountSubscriptionId}
  -> 204 for an owned current-device subscription; non-enumerating 404 otherwise
```

All handlers obtain the account DID and device ID from authenticated context, never from request JSON. The POST response never echoes the token, installation ID, DID, or another account's subscription state. PATCH rejects unknown categories/scopes and malformed partial objects through the standard `{error, message, requestId, fields?}` envelope.

Registration is one database transaction:

1. Lock any installation by current device ID and any active/current installation owning the presented FCM token.
2. If the token belongs to another device ID, deactivate the old installation and all subscriptions and cancel their unsent deliveries. Copy nothing to the new installation.
3. Upsert/reactivate the current device installation with the current platform/token.
4. Upsert/reactivate only `(installation, authenticated DID)`, generating a new routing UUID when the pairing is newly created after a cross-device rebind.
5. Return only that subscription's opaque routing ID.

Normal logout invokes `DeactivateForInstallation(accountDID, deviceID)` before bearer-session revocation; all-session logout invokes `DeactivateForAccount(accountDID)` before OAuth/session teardown. Push cleanup fails closed: if it cannot deactivate/cancel safely, return a server error rather than reporting a successful logout that may continue push delivery. Cleanup never deactivates another account on a shared installation.

### 5.5 Read hydration and safe resolution

Replace the derived union with a page query over active durable rows ordered by `(activity_at DESC, id DESC)`. The opaque cursor contains only those keys and is encoded through `api/envelope`.

Hydration uses bounded set queries:

- One page query for durable rows.
- One batched profile/display-name query for actor DIDs.
- One batched post/project/engagement query for all referenced post URIs, reusing existing response builders and moderation visibility policy where possible.
- No per-item handle resolver calls should be added. Prefer the indexed identity handle cache for notification pages; if a required handle is unavailable, return an explicitly unavailable actor/content representation rather than making N network lookups.

Active response metadata follows Section 15 of `01-requirements.md`. Retracted resolution uses stored safe references and current authorization/visibility:

- Reply: visible reply, else parent/root, else notifications.
- Quote: visible quote, else quoted post, else notifications.
- Like/repost: visible affected post, else notifications.
- Follow/mention: visible actor profile, else notifications.

An ID owned by another account, a deleted actor's old ID, and a random ID all return the same standard 404 envelope. Resolution never changes tombstone state or returns it to the feed.

### 5.6 Push provider boundary

Use `firebase.google.com/go/v4` and `firebase.google.com/go/v4/messaging`, initialized with `firebase.Config{ProjectID: ...}` and Application Default Credentials. This follows Firebase's documented Go server setup and avoids storing JSON credentials in Postgres or custom auth code. The adapter calls `messaging.Client.Send` for one delivery because each payload contains a subscription-specific routing ID; batching can be considered later without changing the dispatcher interface.

```text
type Sender interface {
  Send(ctx context.Context, request SendRequest) (ProviderResult, error)
}

type SendRequest struct {
  Token string                 // provider adapter only; never observable
  NotificationID string
  Category notifications.Category
  AccountSubscriptionID string
  ActorDisplayName string
  Platform Platform
  TTL time.Duration
}
```

The Firebase adapter maps SDK predicates into a small safe classification:

- Permanent installation failure: unregistered/invalid token -> deactivate installation and all subscriptions.
- Retryable: unavailable, quota/rate, internal/server transport, deadline/temporary network failure -> bounded retry.
- Permanent delivery/config/message failure: invalid argument, sender mismatch, third-party auth, or other reviewed non-retryable result -> terminal delivery; configuration-class failures also emit an operator-visible safe error.

Do not persist provider message IDs or raw provider errors unless a later diagnostic requirement proves they are needed. Persist only an allowlisted result class.

The payload contains:

- Notification title/body: actor display name or `Someone`, plus generic category action text.
- Data: `notificationId`, `type`, `accountSubscriptionId`.
- Android TTL: remaining duration.
- APNs header: `apns-expiration` at the absolute deadline.

It contains no handle, DID, AT-URI, post/project/mention text, title, image URL, token, credential, or full serialized payload in telemetry.

Firebase references used for this design decision:

- [Send a message using Firebase Admin SDK](https://firebase.google.com/docs/cloud-messaging/send/admin-sdk)
- [Set the lifespan of a message](https://firebase.google.com/docs/cloud-messaging/customize-messages/setting-message-lifespan)
- [Firebase Admin Go messaging package](https://pkg.go.dev/firebase.google.com/go/v4/messaging)

## 6. State, Providers, Controllers, Or DI

There is no Flutter/Riverpod work in this pass. AppView dependency injection changes are:

```text
app.Config
  -> PushEnabled
  -> FirebaseProjectID
  -> PushBatchSize (bounded default)
  -> PushPollInterval
  -> PushLeaseDuration
  -> PushSendTimeout

app.Deps
  -> NotificationStore
  -> NotificationLifecycle
  -> PushSender
  -> PushDispatcher

newIndexerDispatcher(..., NotificationLifecycle)
  -> CraftskyLike/Repost/Follow/Post/Profile

routes.AddRoutes(...)
  -> NotificationStore passed only to notification handlers
  -> subscription cleanup passed narrowly to auth handlers

cmd/appview.run
  -> HTTP server
  -> Tap consumer
  -> Push dispatcher (enabled or disabled no-op)
```

Constructor choices:

- Pure policy functions accept explicit inputs and need no DI container.
- `NotificationStore` accepts the pool/generated query factory, `Clock`, and UUID generator for deterministic tests.
- `Dispatcher` accepts store, sender, clock, jitter source, observer, logger, and validated options.
- Dev/test can use `DisabledSender` or a fake. Production with `PUSH_ENABLED=true` must create a valid Firebase client during dependency construction or fail startup with a non-secret error.
- A disabled dispatcher is a no-op `Run(ctx)` implementation or is omitted explicitly; startup tests must show no goroutine/network requirement in that mode.

## 7. UI, Widgets, Routes, Or User-Facing Surfaces

No UI/widgets are changed.

Routes to add to both `baseV1RoutePolicies()` and `AddRoutes()`:

| Method / Route | Rate Class | Body Kind | Auth / Device | Surface |
|---|---|---|---|---|
| `GET /v1/notifications` | Read | No body | Required | Existing route, durable source |
| `GET /v1/notifications/new-count` | Read | No body | Required | Account-wide active/listable revision count; no mutation |
| `POST /v1/notifications/seen` | Write | No body | Required | Snapshot-safe account acknowledgement; 204 |
| `GET /v1/notifications/{notificationId}` | Read | No body | Required | Resolve active/tombstone target |
| `GET /v1/notifications/preferences` | Read | No body | Required | Seven effective preferences |
| `PATCH /v1/notifications/preferences` | Write | Default JSON | Required | Partial preference update |
| `POST /v1/notifications/devices` | Write | Default JSON | Required | Register/rotate/rebind current installation subscription |
| `DELETE /v1/notifications/devices/{accountSubscriptionId}` | Write | No body | Required | Remove owned current-device account subscription |

Route registration order must put fixed `/preferences` and `/devices` patterns alongside the parameterized notification-ID route without ambiguity under Go 1.22 `ServeMux`. Tests call the real mux and verify each policy exists.

The existing `NotificationItem` changes additively to include stable `id` and `quote` metadata. Existing follow/like/repost/reply/mention fields remain available unless the later Flutter compatibility pass explicitly approves a breaking shape. The AppView quote contract is implemented and tested in this pass, but production deployment must be coordinated with a Flutter decoder that accepts the category.

## 8. Error, Loading, Empty, And Edge States

| State / Case | Planned Handling | Requirement IDs | Test IDs |
|---|---|---|---|
| No saved preferences | Materialize seven effective `everyone/true` values in response without requiring seven stored rows | FR-007, RULE-005 | UT-002, IT-007 |
| Invalid category/scope/PATCH body | 400 standard envelope with safe camelCase field errors; no partial write | FR-006, FR-007, NFR-003 | UT-002, IT-007, AT-007 |
| Self-generated activity | Suppress before durable insert/fan-out | FR-020 | UT-003, AT-002, AT-004 |
| `peopleIFollow` actor not followed | Create neither notification nor delivery; follow category therefore requires mutual follow | FR-008, RULE-001, RULE-002 | UT-003, IT-010 |
| Push disabled | Create accepted in-app notification but no delivery; later re-enable does not backfill | FR-027, RULE-002 | IT-025 |
| Missing indexed subject | Suppress with safe reason/metric; no reconciliation or retrospective push | FR-017, FR-022, FR-025 | AT-003, REG-005 |
| Exact Tap replay | No activity timestamp movement, duplicate row, or duplicate delivery | FR-002 | IT-003, REG-002 |
| Undo/recreate | Retract/cancel, then reactivate same ID/top timestamp without new fan-out | FR-018, FR-021 | IT-005, IT-018, IT-020 |
| Source/destination deletion | Retract atomically and cancel unsent rows; keep safe references for owner resolution | FR-021, FR-023, FR-026 | IT-004, IT-005, IT-017, IT-018 |
| Permanent actor deletion | Hard-delete caused notifications and unsent work; stale/unknown resolution is identical 404 | FR-023, FR-030 | IT-022 |
| Unavailable/taken-down content | Return explicit unavailable state or safe fallback; never leak stale presentation data | FR-023, FR-032 | IT-017, IT-019, REG-003, REG-006 |
| Empty feed | Return `items: []` and omit cursor | FR-004, FR-005 | IT-006 |
| No acknowledgement row | Count every active/listable durable revision as new without creating state | FR-035, FR-038 | IT-033 |
| List/count prefetch | GET routes leave the account marker unchanged | FR-036, RULE-007 | REG-009 |
| Concurrent notification during mark-seen | Capture R, upsert through R, leave R+1 new | FR-036 | IT-034 |
| Concurrent devices acknowledge same account | Store the greatest captured revision and never regress | FR-036, FR-037 | IT-034, IT-035 |
| Shared device with multiple accounts | Each authenticated DID reads/writes only its own marker | FR-037 | AT-010, IT-035 |
| Retracted or actor-hidden notification | Exclude from count using the same active/list-level actor visibility predicate | FR-035 | IT-033, REG-009 |
| Tied activity timestamps | Order by activity time then stable ID; opaque cursor round-trips both | FR-005 | IT-006 |
| Cross-owner notification/subscription ID | Non-enumerating standard 404 | FR-026, NFR-003 | IT-024, AT-007 |
| Repeated registration, same device/token/account | Return existing active routing ID; do not duplicate installation/subscription | FR-009, FR-010 | IT-008 |
| Token rotation on same device | Update installation token; keep existing subscriptions/deliveries | FR-010 | IT-012 |
| Same token on new device ID | Atomic old-owner deactivation/cancellation, new installation, current account only, new routing ID | FR-033 | IT-031, AT-009 |
| No active subscriptions | Commit durable notification with zero delivery rows | FR-011 | IT-009 |
| Token becomes unregistered | Terminal delivery plus installation/all-subscription deactivation | FR-014 | IT-014 |
| Provider transient failure | Persist bounded jittered retry before deadline | FR-013 | UT-006, IT-013 |
| Delivery reaches deadline | Mark expired without provider call; TTL never exceeds remaining time | FR-013, FR-028 | IT-016 |
| Two worker processes | `FOR UPDATE SKIP LOCKED` lease gives normal single ownership; expired lease is recoverable | FR-012, RULE-004 | IT-030 |
| Crash after FCM acceptance | Lease recovery may resend; record/test the documented at-least-once window | RULE-004 | IT-015, REG-007 |
| Push disabled in dev/test | No Firebase initialization, network, or dispatcher work | NFR-004 | UT-009, REG-004 |
| Push enabled in prod with missing/invalid configuration | Dependency construction fails with safe configuration error | NFR-004 | UT-009 |
| Logout cleanup failure | Do not report successful logout; log safe operational failure, preserve other accounts | FR-016, NFR-002 | IT-011, IT-021, REG-008 |

## 9. Test Implementation Plan

The order below is the TDD order, not merely a coverage list. Each row should start red, receive the minimum production change, then refactor while its focused package remains green.

| Order | Test ID | Target | Setup / Fixture | Initial Expected Failure |
|---|---|---|---|---|
| 1 | UT-001 | `internal/notifications/category_test.go` | Table of seven approved wire values | Package/registry absent; quote/everythingElse unsupported |
| 2 | UT-002 | `internal/notifications/preferences_test.go` | Missing rows and partial patches | Defaults/merge/validation absent |
| 3 | UT-003 | `internal/notifications/eligibility_test.go` | Scope/push/self/follow matrix | No event-time policy seam |
| 4 | IT-028 | `internal/db/notifications_migration_test.go` | Apply `000021` over representative pre-state | Tables, constraints, indexes, and plan shapes absent |
| 5 | IT-004 like-create case | `internal/index/notification_transaction_test.go` | Like fixture plus forced failure after source and lifecycle writes | Source/notification cannot yet roll back together |
| 6 | IT-001 like slice / IT-002 / IT-003 | `internal/index/notification_ingestion_test.go` | Viewer post, actor like, two subscriptions, replay | No durable row/fan-out/idempotency |
| 7 | IT-004 like-delete case / IT-005 like | transaction/lifecycle suites | Active like then forced deletion failure and successful delete | Delete is not transaction-aware; no tombstone cancellation |
| 8 | IT-020 like | lifecycle suite | Create/delete/recreate with changed source URI | Stable semantic identity/reactivation absent |
| 9 | AT-002, AT-003 | `notification_interaction_test.go` | Like/repost direct author/reposter/self matrix | Repost and attribution rules not generalized |
| 10 | IT-010 follow slice / IT-004 follow | eligibility/transaction suites | Mutual/non-mutual follow and rollback | Follow indexer lacks lifecycle transaction |
| 11 | UT-004, AT-004, IT-004 post | classify/post/transaction suites | Reply/quote/mention overlap per recipient | Canonical classification and post transaction seam absent |
| 12 | IT-005, IT-018 | lifecycle suite | Delete every producer with pending/retry/lease states | Retraction/cancellation incomplete across categories |
| 13 | IT-006, UT-005, REG-001 | API store/response suites | Durable events with tied times/all types | List still derives mutable union; quote/stable ID absent |
| 14 | IT-019, REG-003 | API store suite | Available/missing/taken-down references | Bounded safe hydration absent |
| 15 | IT-017, IT-024 | resolution suite | Active/retracted/cross-owner IDs | Resolution route/store absent |
| 16 | IT-007, AT-007 preference cases | API/routes suites | Authenticated mux and partial request JSON | Preference handlers/routes absent |
| 17 | IT-008, IT-012 | device API suite | Repeated registration and token rotation | Installation/subscription store absent |
| 18 | IT-021, IT-031, AT-008, AT-009 | device API suite | Shared accounts and cross-device token collision | Isolation/rebind transaction absent |
| 19 | IT-011, REG-008 | auth/device suites | Ordinary/all-session logout across installations | Logout does not clean push state |
| 20 | UT-006 | `internal/push/retry_test.go` | Fake clock and deterministic jitter | Retry/deadline policy absent |
| 21 | UT-007 | `internal/push/payload_test.go` | All categories with privacy sentinels | Minimal cross-platform payload absent |
| 22 | IT-013–IT-016, REG-007 | dispatcher suite | Scripted fake sender and fake clock | Claim/send/finalize loop absent |
| 23 | IT-023, IT-025, IT-026 | dispatcher/eligibility/ingestion | Multi-install routing, toggle timeline, burst | Routing/prospective/no-aggregation edges absent |
| 24 | IT-030 | dispatcher concurrency suite | Two workers, lease expiry, cancellation | Claim ownership/recovery absent |
| 25 | IT-022 | actor deletion suite + Tap consumer | Identity `deleted`, active/deactivated controls | Identity deletion is currently dropped |
| 26 | UT-008, IT-027, IT-029 | observability suites | Sentinel secrets and in-memory recorder | Push signals/redaction validation absent |
| 27 | UT-009, REG-004 | app config/deps/server suites | Prod/dev/test configurations and fake sender | Push startup/lifecycle wiring absent |
| 28 | REG-005, REG-006 | migration/structure suites | Old source rows; schema inspection | No-backfill and no-block/mute guards absent |
| 29 | MAN-001, MAN-002 | Non-production Firebase Android/iOS | Real test project/APNs sandbox after automated suite | Provider/OS path not yet proven |

Approved follow-up TDD order after the existing implementation:

| Order | Test ID | Target | Setup / Fixture | Initial Expected Failure |
|---:|---|---|---|---|
| 30 | IT-032 | `internal/notifications/newness_test.go`, migration tests | Apply `000022`; activation/replay/retract/reactivate lifecycle | Revision schema and lifecycle allocation absent |
| 31 | IT-033, REG-009 | `internal/api/notification_newness_test.go` | Active/retracted/hidden rows above/below marker | Count store/handler/route absent; GET mutation regression unproved |
| 32 | IT-034 | `internal/api/notification_newness_test.go` | Deterministic transaction snapshot with concurrent insert | Snapshot-safe greatest-value acknowledgement absent |
| 33 | AT-010, IT-035 | `internal/routes/notification_newness_test.go` | Alice on two devices; Bob sharing one | Account-wide and cross-account behavior absent |

Focused commands by phase, from `appview/`:

```text
go test ./internal/notifications
TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/db ./internal/index
TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes ./internal/auth
TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/push ./internal/observability ./internal/app ./cmd/appview
TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test -race ./...
```

After query changes, run the new `just sqlc`/`just sqlc-check` generation guard before package tests. The final repository command remains `just test` with compose Postgres available.

## 10. Sequencing And Guardrails

- First TDD step: write `UT-001` for exactly `like`, `follow`, `reply`, `mention`, `quote`, `repost`, and `everythingElse`, proving no via-repost category and no `everythingElse` producer.
- First database step: make IT-028 red against `000021`, including the active-feed and delivery-claim indexes and zero-backfill assertion.
- First vertical source step: one like create/delete transaction across source row, durable event, and delivery row; force rollback at both lifecycle boundaries before adding other producers.
- Dependencies between work items:
  - Category/scope types precede schema checks, generated queries, handlers, and payloads.
  - Migration/query generation precedes notification service integration.
  - One producer's atomic slice precedes producer expansion.
  - Durable list/resolution follows event lifecycle because it must read real tombstone semantics.
  - Installation/subscription APIs precede fan-out and dispatcher integration.
  - Retry/payload pure tests precede Firebase adapter and worker lifecycle.
  - Observability hooks follow stable service/worker result enums so labels stay bounded.
- Guardrails:
  - Never call FCM while a source/indexer database transaction is open.
  - Every producer mutation and its notification lifecycle mutation use the same `pgx.Tx`.
  - Parse atproto identifiers at Tap/HTTP boundaries; use `syntax` typed values internally.
  - Never infer recipient identity from repost ownership.
  - Never enqueue from pre-existing rows, registration-time history, preference re-enable, or reactivation.
  - Never copy handles, display names, text, images, or titles into durable notification rows.
  - Never include token/credential/payload/DID/URI values in errors, logs, spans, Sentry attributes, or metric labels.
  - Never authorize resolution, registration, or removal using a request-supplied DID.
  - Keep token ownership and account authorization in separate tables and transactions.
  - Use one active token owner; cross-device collision transfers no subscriptions or routing IDs.
  - Use bounded batch sizes, provider timeouts, leases, polling, and shutdown waits.
  - Keep current moderation/takedown checks authoritative; introduce no notification-only block/mute claims.
  - Preserve all unrelated dirty-worktree changes and stage only workflow artifacts if a later stage commit is approved.
- Out of scope:
  - Flutter Firebase SDK, permission/token-refresh/open handling, settings UI, or quote decoder work.
  - Lexicon/PDS records or ADRs.
  - Historical backfill/reconciliation, retention purge, per-notification read state, per-device acknowledgement, badge rendering, grouping, aggregation, digests.
  - Block/mute integration, temporary account-deactivation semantics, raw APNs provider, web push, CLI operations.

## 11. Risks And Open Questions

| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Resolved | High-risk implementation approval | TDD implementation may proceed | User explicitly approved document updates and AppView implementation on 2026-07-14 |
| CPQ-002 | Resolved | Concrete FCM client/auth mechanism | Affects dependency/config/provider errors | Use official Firebase Admin Go SDK v4, explicit Firebase project ID, and ADC; keep `Sender` fakeable |
| CPQ-003 | Resolved | Query-count verification mechanism from GAP-003 | Exact pgx instrumentation could add test complexity | Use batch-shaped query APIs, migration/index assertions, and `EXPLAIN (FORMAT JSON)` for feed/claim queries; add a tracer only if implementation reveals an N+1 path |
| CPQ-004 | Resolved | Permanent repository deletion signal | Current consumer drops identity events | Decode Tap identity status and hard-delete only on terminal `deleted`; all temporary statuses remain unchanged/out of scope |
| CPQ-005 | Non-blocking operational | Existing Flutter decoder rejects unknown `quote` category | Enabling quote responses before client update can break the current app page | Implement/test AppView contract, but gate production category rollout until separate Flutter work lands |
| CPQ-006 | Non-blocking operational | No retention policy | Durable/terminal tables grow indefinitely | Keep correct indexes and queue metrics; design retention in a later approved requirement set |
| CPQ-007 | Non-blocking known limit | FCM acceptance and success commit are not atomic | Rare duplicate after worker crash | Preserve at-least-once rule, leases, success terminal state, and explicit crash-window regression |
| CPQ-008 | Non-blocking known limit | Out-of-order/missing subject events are not reconciled | Some valid federated activity may never notify | Suppress observably with no retrospective push; defer reconciliation |
| CPQ-009 | Non-blocking rollout | Real APNs/FCM delivery needs external credentials/devices | Cannot be deterministic in normal suite | Keep fake coverage automated; run MAN-001 and MAN-002 in a non-production Firebase project before enablement |

## 12. Handoff To TDD Builder

- Coding plan: `04-coding-plan.md`
- TDD execution plan: `05-implementation-plan.md`
- Start the approved follow-up with test: `IT-032` covering migration/revision lifecycle.
- First focused command: `cd appview && go test ./internal/notifications -run TestNotificationNewness -count=1`
- Follow with `IT-033` count/visibility, `IT-034` snapshot acknowledgement, and `IT-035` account/device isolation.
- Full validation command: from repository root, `just test` with compose Postgres running
- Notes:
  - Update `05-implementation-plan.md` with steps 44 onward before code changes; implementation approval is already recorded.
  - Follow strict red-green-refactor ordering; do not scaffold every package before the first vertical slice is green.
  - Automated tests must never contact Firebase.
  - Read back and reconcile this plan with `01-requirements.md`, `02-acceptance-tests.md`, and `03-document-review.md` if implementation discoveries change an interface or route; product behavior changes require returning to the earlier workflow stage.
