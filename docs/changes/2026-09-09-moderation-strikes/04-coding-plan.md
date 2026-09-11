# Coding Plan: Moderation Cases, Strikes, Appeals, And Suspension

## 1. Inputs

- Requirements: `docs/changes/2026-09-09-moderation-strikes/01-requirements.md`
- Tests: `docs/changes/2026-09-09-moderation-strikes/02-acceptance-tests.md`
- Document review: `docs/changes/2026-09-09-moderation-strikes/03-document-review.md` (`Approved`)
- API conventions: `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md`
- Architecture reference: `atproto-craft-social-app-reference.md`
- Risk: High; implementation requires explicit user approval and all merge-blocking tests from `02-acceptance-tests.md`.
- SQL decision: Use a scoped inline-`pgx` exception for this feature because the current repository has no sqlc configuration or generated query package. Keep SQL private to moderation stores and transaction-aware bridge methods; do not establish a second general data-access convention.

## 2. Implementation Strategy

Implement the approved behavior as one AppView-owned moderation domain with append-only audit events and explicit current projections. Retool and the future source-neutral adapter submit commands to the same application service; neither receives database access. Existing reports remain intake evidence, existing `moderation_outputs` remain the only visibility-enforcement stream, existing notification/push infrastructure remains the delivery mechanism, and owner/PDS lifecycle state remains separate from reversible moderation suspension.

Build in vertical TDD slices. Start with the legacy-safe migration and database invariants, add concurrent report-to-case grouping, then add policy validation and one atomic adjudication transaction. Add standing/expiry, appeals/effect changes, HTTP surfaces, route-level suspension enforcement, notification dispatch, and Flutter UI in that order. This keeps each projection derivable from an already-tested event boundary.

The authoritative lock order for commands that can change account standing is:

1. Ensure and lock `moderation_account_standings` for the affected owner DID.
2. Lock the moderation case and verify its current revision.
3. Claim or replay the `(sourceSystem, replayId)` identity and verify its request fingerprint.
4. Append the case command/event and any decision/effect events.
5. Emit selected visibility effects into existing `moderation_outputs` using the same transaction.
6. Update logical strike/effect projections and derive threshold/severe bases.
7. Update account standing and case revision.
8. Insert an actorless moderation notification event only when the event changes an owner-visible consequence or effective enforcement.
9. Commit once.

Expiry uses the same standing-before-case/effect lock discipline. The stored projection remains authoritative until the expiry transaction commits; request handlers do not perform wall-clock restoration.

## 3. Affected Areas

| Area | Existing Pattern | Planned Change | Requirement IDs | Test IDs |
|---|---|---|---|---|
| Private persistence | Numbered `golang-migrate` SQL; current moderation stores use inline `pgx`. | Add migration `000072` with cases, associations, append-only events, projections, appeals, replay constraints, indexes, and notification-category constraints. Preserve all legacy rows without backfill. | FR-001, FR-006, FR-013, FR-036, NFR-001, RULE-003 | IT-001, IT-002, IT-020, REG-008 |
| Report intake | `api.ReportStore.CreateReport` owns one transaction and permits duplicate reports. | Attach every newly accepted report to the one open canonical-subject case inside the existing report transaction. | FR-001, FR-002, FR-029, RULE-007 | AT-009, IT-001, IT-002, REG-001 |
| Domain policy | Existing dev moderation path couples validation and persistence. | Add pure reason/consequence/reference/standing/expiry/notification policy functions in `internal/moderation`. | FR-012 through FR-017, FR-027 through FR-034, RULE-001 through RULE-017 | UT-001 through UT-005, UT-008, UT-009, UT-012 |
| Adjudication | No production case service exists. | Add source-neutral command service with replay fingerprints, optimistic case revisions, append-only events, partial reversal/reapplication, appeal, and restoration commands. | FR-005, FR-006, FR-011, FR-024, FR-025, FR-030, NFR-001, NFR-002 | AT-005, AT-006, AT-008, AT-010, IT-003 through IT-009, IT-019, IT-022 through IT-024 |
| Visibility outputs | `api.ModerationStore.InsertOutput` writes `moderation_outputs`. | Extract/reuse a transaction-aware output writer; adjudication emits existing apply/negate outputs rather than parallel visibility state. | FR-026, FR-028, RULE-008 | UT-009, IT-015, REG-002 |
| Admin API/auth | Dev-only token route; route catalogue has member access classes only. | Add production moderator access class, constant-time bearer middleware, admin context, curated queue/detail handlers, and command handlers. | BR-004, FR-003 through FR-005, FR-023, NFR-002, NFR-003 | AT-005, IT-010, IT-011, IT-021, MAN-001 |
| Owner API | Normal member auth and envelope/cursor helpers. | Add standing, paginated history, and single-entry reads with strict owner-safe DTOs. | BR-001, FR-007, FR-022, FR-032, NFR-003, NFR-004 | AT-001, UT-006, UT-011, IT-012, IT-023 through IT-025 |
| Suspension authorization | `AccessClass` protects lifecycle membership; no moderation capability class. | Add an independent, exhaustive `SuspensionClass` to route policies and middleware that fails closed before handler/body/PDS work. | FR-018, NFR-005 | AT-003, UT-004, IT-013, IT-014, REG-003, REG-007 |
| Expiry | Existing processors use injected clocks, batches, retries, and process wiring. | Add restart-safe `ExpiryProcessor` and overdue-work instrumentation. | FR-013, FR-016, FR-017, FR-034, RULE-002, RULE-004, RULE-006 | AT-007, UT-001, UT-003, IT-007, IT-022 |
| Notifications | Actorless system events, preferences, fan-out, leases, retry, and opaque account bindings already exist. | Add `moderation` category, transaction-aware event insert, event-specific routing facts, fixed scope, preference support, and complete eligibility policy. | BR-006, FR-019 through FR-021, FR-035, NFR-006, RULE-009, RULE-010 | AT-004, AT-011, UT-007, UT-012, IT-016 through IT-018, IT-026, IT-027, REG-004 |
| Observability | `observability.Observer`, structured safe logging, worker operation metrics. | Add safe moderation operation/auth counters, work-age gauges/evaluation, and concrete alert predicates. | NFR-003, NFR-008 | UT-014, IT-021 |
| Flutter data/state | Dio repositories, `dart_mappable`, account-scoped generated Riverpod providers. | Add owner moderation models/repository/providers with account-switch fencing and paginated history. | FR-007 through FR-010, FR-032 | AT-001, AT-002, UT-005, UT-010, IT-012, IT-025 |
| Flutter UI/routing | Typed `go_router`, settings rows, localized design-system pages, secure notification activation. | Add standing/history routes/page/widgets, appeal launcher/copy fallbacks, moderation settings category, and moderation destination inference after exact-account activation. | FR-008 through FR-010, FR-021, FR-035, NFR-007 | AT-001, AT-002, AT-004, AT-011, IT-025 through IT-027, MAN-002 through MAN-004 |
| Owner deletion | Terminal inventory and private cleanup enumerate DID roles. | Add moderation DID roles and cascade case-owned data; remove affected-owner private history/standing without changing the moderation prohibition on PDS deletion. | BR-007, FR-018, RULE-008 | IT-014, REG-003, REG-006, REG-008 |

## 4. Files And Modules

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `appview/migrations/000072_moderation_cases.up.sql` | Create | Add moderation cases/events/effects/strikes/standing/appeals, report association, indexes/constraints, and `moderation` notification category support without legacy backfill. | FR-001, FR-006, FR-011, FR-013, FR-019, FR-024, FR-036, NFR-001 | IT-020, REG-008 |
| `appview/migrations/000072_moderation_cases.down.sql` | Create | Remove only new moderation structures/category values in dependency-safe order; never mutate retained legacy report/output rows. | FR-036 | IT-020 |
| `appview/internal/moderation/reference.go` | Create | UUIDv4 `MOD-` parse/format/case-insensitive canonicalization. | FR-032 | UT-005, IT-025 |
| `appview/internal/moderation/policy.go` | Create | Closed disposition, reason, consequence, reversal/reapplication, appeal, and notification validation. | FR-012, FR-027 through FR-031, FR-033, RULE-001, RULE-003, RULE-005, RULE-009, RULE-012 through RULE-017 | UT-002, UT-008, UT-009, UT-012 |
| `appview/internal/moderation/standing.go` | Create | Strike due-date calculation and pure standing derivation from projections. | FR-013 through FR-017, FR-034, RULE-002, RULE-004, RULE-006 | UT-001, UT-003 |
| `appview/internal/moderation/models.go` | Create | Typed internal commands/events/projections; use `syntax.DID` and `syntax.ATURI` for atproto identifiers. | FR-004 through FR-006, FR-022, FR-024 | IT-003, IT-023 |
| `appview/internal/moderation/store.go` | Create | Encapsulate scoped inline-`pgx` SQL, scans, replay fingerprints, keyset reads, and transaction helpers. | FR-003 through FR-007, FR-011, FR-013, FR-024, NFR-001, NFR-002 | IT-001 through IT-012, IT-019 through IT-024 |
| `appview/internal/moderation/intake.go` | Create | Attach an accepted report to an existing/new open case within caller transaction. | FR-001, FR-002, FR-029 | IT-001, IT-002 |
| `appview/internal/moderation/service.go` | Create | Sole adjudication command boundary and lock/order orchestration. | FR-005, FR-006, FR-012 through FR-017, FR-024 through FR-030 | IT-003 through IT-008, IT-015, IT-016, IT-019, IT-022 through IT-024 |
| `appview/internal/moderation/appeals.go` | Create | Store untrusted correspondence separately; confirm/resolve the single lifecycle through trusted commands. | BR-003, FR-011, FR-031, RULE-005, RULE-016, RULE-017 | AT-006, UT-008, IT-009 |
| `appview/internal/moderation/expiry.go` | Create | Batch due strikes using injected UTC clock and idempotent projection-authoritative transactions. | FR-016, FR-017, FR-019, FR-034 | UT-001, IT-007, IT-022 |
| `appview/internal/moderation/adapter.go` | Create | Source-neutral trusted command adapter; no database methods. | BR-005, FR-024, FR-025 | UT-013, IT-019 |
| `appview/internal/api/report_store.go` | Change | Accept a transaction-aware case attacher and commit report plus case association once. | FR-001, NFR-001 | IT-001, IT-002, REG-001 |
| `appview/internal/api/moderation_store.go` | Change | Expose a narrow `InsertOutputTx` used by adjudication while preserving dev handler behavior and current policy reads. | FR-026, NFR-001 | IT-003, IT-015, REG-002 |
| `appview/internal/api/moderation_admin.go` | Create | Queue/detail and decision/appeal/effect/restoration handlers with standard envelopes. | BR-004, FR-003 through FR-005, FR-011, FR-023 | AT-005, IT-010, IT-011 |
| `appview/internal/api/moderation_admin_request.go` | Create | CamelCase command DTOs with `expectedRevision`; use `Idempotency-Key` header for replay identity. | FR-005, FR-024, NFR-004 | IT-003, IT-010 |
| `appview/internal/api/moderation_admin_response.go` | Create | Curated admin queue/detail DTOs and opaque cursor response. | FR-003, FR-004, NFR-003, NFR-004 | IT-010, IT-011 |
| `appview/internal/api/moderation_owner.go` | Create | Authenticated owner standing/history/detail handlers. | BR-001, FR-007, FR-022, FR-032 | AT-001, IT-012, IT-025 |
| `appview/internal/api/moderation_owner_response.go` | Create | Explicit allowlisted owner wire DTOs; no internal model serialization. | FR-007, FR-022, NFR-003 | UT-006, IT-012, IT-023 |
| `appview/internal/middleware/moderator.go` | Create | Constant-time production moderator bearer validation and server-derived actor/source context. | FR-023, NFR-002, NFR-003, RULE-001 | IT-011, IT-021 |
| `appview/internal/middleware/moderation_enforcement.go` | Create | Query projected suspension and enforce route `SuspensionClass` before side effects. | FR-018, NFR-005 | UT-004, IT-013, IT-014 |
| `appview/internal/ctxkeys/moderator.go` | Create | Private typed moderator identity/source context accessors. | FR-023, NFR-002 | IT-011 |
| `appview/internal/routes/policy.go` | Change | Add zero-invalid `SuspensionClass` to every route and `AccessModerator` for admin routes. | FR-018, FR-023, NFR-005 | UT-004, IT-011, IT-013 |
| `appview/internal/routes/routes.go` and capability registrars | Change | Register owner/admin routes and compose member/moderator/suspension middleware correctly. | FR-003 through FR-007, FR-018, FR-023 | IT-010 through IT-014, REG-005 |
| `appview/internal/routes/dependencies.go` | Change | Add narrow moderation reader/commander/standing dependencies and moderator config. | FR-003 through FR-007, FR-023 | IT-010 through IT-013 |
| `appview/internal/app/config.go` | Change | Validate moderator secret/actor/source and expiry worker budgets; secrets use `app.Secret`. | FR-023, FR-034, NFR-002 | IT-007, IT-011, IT-021 |
| `appview/internal/app/deps.go`, `routes_adapter.go` | Change | Wire moderation stores/services/processor without passing full `Deps` to handlers. | FR-005, FR-025, FR-034 | IT-019, IT-022 |
| `appview/cmd/appview/main.go` | Change | Start/stop expiry processor with existing worker lifecycle and observer. | FR-034, NFR-008 | IT-007, IT-021 |
| `appview/internal/notifications/category.go`, `preferences.go` | Change | Add fixed-scope `moderation` category with independent `pushEnabled`. | FR-020, FR-035 | UT-007, IT-017, IT-027, REG-004 |
| `appview/internal/notifications/store.go` or existing event writer | Change | Add transaction-aware actorless moderation event insert keyed by case event. | FR-019, NFR-001, NFR-006 | IT-003, IT-016 |
| `appview/internal/push/payload.go` and routing facts | Change | Build safe moderation payload with existing opaque account binding, notification ID, and public case reference. | FR-021, NFR-003, NFR-006 | UT-007, IT-018 |
| `appview/internal/push/dispatcher.go` | Change | Treat moderation as preference-controlled actorless work; retain current post-lease preference recheck/retry behavior. | FR-020, NFR-006 | IT-017, REG-004 |
| `appview/internal/observability/` moderation files | Create / Change | Safe operation/auth/work-age observations and exact alert predicates. | NFR-003, NFR-008 | UT-014, IT-021 |
| `appview/internal/ownerlifecycle/terminal_inventory.go` | Change | Register moderation owner/standing DID roles; case children cascade. | BR-007, NFR-003 | REG-003, REG-008 |
| `appview/internal/accountdeletion/private_cleanup.go` | Change | Remove affected-owner private moderation data and notifications without moderation-initiated PDS deletion. | BR-007, FR-018, RULE-008 | IT-014, REG-003, REG-006 |
| `appview/environments/dev.env.example`, `prod.env.example` | Change | Document moderator/expiry settings; production moderator secret is required only when admin workflow is enabled. | FR-023, FR-034 | IT-007, IT-011 |
| `app/lib/moderation/models/` | Create | Map standing, suspension bases, history entries/effects, appeals, safe snapshots, and UUIDv4 case references. | FR-007 through FR-010, FR-032 | UT-005, UT-010, IT-025 |
| `app/lib/moderation/data/moderation_repository.dart` | Create | Owner-facing repository interface for standing/history pages. | FR-007 | IT-012 |
| `app/lib/moderation/data/api_moderation_repository.dart` | Create | Dio implementation for owner moderation routes with opaque cursor round-trip. | FR-007, NFR-004 | UT-010, IT-012 |
| `app/lib/moderation/providers/moderation_providers.dart` | Create | Account-scoped repository, standing, paginated history, and email-launch dependencies. | FR-007 through FR-010 | AT-001, AT-002, UT-010 |
| `app/lib/moderation/pages/account_standing_page.dart` | Create | Responsive standing summary, history, loading/error/empty states, and optional focused case. | FR-008, NFR-007 | AT-001, MAN-004 |
| `app/lib/moderation/widgets/moderation_history_entry.dart` | Create | Render consequences, dates, expiry/overturn/appeal states, safe snapshots, appeal/copy actions. | FR-008, FR-009 | AT-001, AT-002, MAN-004 |
| `app/lib/settings/models/settings_row.dart`, `settings/pages/settings_page.dart` | Change | Add Account standing row. | FR-008 | AT-001 |
| `app/lib/router/route_locations.dart`, `router.dart` | Change | Add typed standing and focused-case routes. | FR-008, FR-021, FR-032 | AT-001, IT-025, IT-026 |
| `app/lib/notifications/models/notification_category.dart` | Change | Add `moderation` preference category. | FR-035 | AT-011, IT-027 |
| `app/lib/notifications/pages/notification_settings_page.dart` | Change | Render moderation as fixed-scope push toggle with localized explanation. | FR-020, FR-035 | AT-011, IT-027 |
| `app/lib/notifications/services/notification_destination_inference.dart` and destination models | Change | Infer moderation destination from validated public reference after existing exact-account activation. | FR-021 | AT-004, IT-026 |
| `app/lib/l10n/app_en.arb` and generated localization output | Change / Generate | Add all standing/history/appeal/preference/system-notice/error copy. | FR-008, FR-009, NFR-007 | AT-001, AT-002, AT-011, MAN-004 |
| Test files named in `02-acceptance-tests.md` | Create / Change | Implement red-green coverage using existing Go `testdb`, HTTP recorder, Dio adapter, ProviderScope, routing, clipboard, and notification-flow helpers. | All Must requirements | AT-001 through MAN-004 |

## 5. Services, Interfaces, And Data Flow

### Persistence Shape

`000072_moderation_cases` should create these private structures:

| Structure | Key invariants |
|---|---|
| `moderation_cases` | UUIDv4 `id`; canonical `subject_key`; typed subject DID/collection/rkey/URI; owner DID; `open`/`resolved`; revision; safe snapshot; partial unique index on `subject_key` where open. |
| `moderation_case_reports` | One immutable association per accepted post-enable report; unique `report_id`; cascade from case only, without changing `moderation_reports`. |
| `moderation_case_events` | Append-only event identity/type, actor, source system, replay ID, fingerprint, expected/result revision, timestamp; unique `(source_system,replay_id)` and `(case_id,result_revision)`. |
| `moderation_decisions` | One decision payload per resolution event; disposition/reason/internal evidence/user-safe detail/severity rationale with database checks matching policy. |
| `moderation_effect_events` | Append-only apply/negate/expire/restore operations for formal warning, visibility, strike, and severe suspension; references the originating logical effect and existing moderation-output ID when applicable. |
| `moderation_active_case_effects` | Current per-case/effect projection used for fast validation/history; unique logical effect per case. |
| `moderation_case_strikes` | Exactly one logical strike per case; issuance/due/effective-expiry/overturn projection; reapplication updates issuance/due while event history remains append-only. |
| `moderation_account_standings` | One row per owner DID; active count, threshold basis, severe basis, effective suspension, revision/update timestamps. |
| `moderation_appeal_correspondence` | Untrusted correspondence metadata tied to public case lookup; no owner-visible state transition. Do not persist raw email body unless a later retention decision approves it. |
| `moderation_appeals` | At most one row per case; moderator-confirmed `pending`, then `upheld` or `changed`; attributable timestamps/event links. |

Use foreign keys and checks for structural invariants, application validation for cross-row policy, and triggers only in tests for rollback injection. Do not backfill any new table from existing `moderation_reports` or `moderation_outputs`.

### Internal Interfaces

```text
type AcceptedReportAttacher interface {
  AttachAcceptedReportTx(ctx, tx, reportRow) (Case, error)
}

type VisibilityOutputWriter interface {
  InsertOutputTx(ctx, tx, idempotencyIdentity, input) (Output, error)
}

type NotificationIntentWriter interface {
  InsertModerationEventTx(ctx, tx, recipientDID, caseReference, sourceEventID, occurredAt) error
}

type AdjudicationService interface {
  Apply(ctx, TrustedCommand) (CommandResult, error)
}

type TrustedInputAdapter interface {
  Normalize(authenticatedSource, replayID, request) (TrustedCommand, error)
}

type StandingReader interface {
  Standing(ctx, ownerDID) (Standing, error)
  IsSuspended(ctx, ownerDID) (bool, error)
}

type OwnerHistoryReader interface {
  History(ctx, ownerDID, cursor, limit) (HistoryPage, error)
  Entry(ctx, ownerDID, caseReference) (HistoryEntry, error)
}
```

The admin adapter derives `sourceSystem`, moderator actor, and moderation source DID from authenticated context. Request bodies cannot override them. `Idempotency-Key` is 16-128 printable ASCII, consistent with the existing moderation route; `expectedRevision` is required for every command against an existing case. Store replay receipts durably rather than using the current 24-hour dev receipt TTL.

### Command Model

Use separate commands rather than a generic mutable patch:

```text
ResolveCase { disposition, reason?, evidenceNotes?, userSafeDetail?, severityRationale?, consequences, expectedRevision }
ConfirmAppeal { correspondenceReference?, expectedRevision }
ResolveAppeal { outcome: upheld|changed, reversedEffects?, rationale, expectedRevision }
ChangeEffects { negateEffects?, applyEffects?, rationale, expectedRevision }
RestoreSevereSuspension { effectId, rationale, expectedRevision }
```

`ChangeEffects` addresses only known logical effects. Reapplying a strike updates the case's one logical strike projection with the server issuance time and a new due date. A command that changes no active consequence is rejected as semantically invalid and creates no notification.

### HTTP Surface

Use these additive `/v1/` routes:

```text
GET  /v1/admin/moderation/cases
GET  /v1/admin/moderation/cases/{caseReference}
POST /v1/admin/moderation/cases/{caseReference}/decisions
POST /v1/admin/moderation/cases/{caseReference}/appeal-confirmations
POST /v1/admin/moderation/cases/{caseReference}/appeal-resolutions
POST /v1/admin/moderation/cases/{caseReference}/effect-changes
POST /v1/admin/moderation/cases/{caseReference}/restorations

GET  /v1/moderation/standing
GET  /v1/moderation/history
GET  /v1/moderation/history/{caseReference}
```

Admin queue filters: `state`, `subjectType`, `appealStatus`, `limit`, `cursor`. Order by `(created_at DESC,id DESC)` with an opaque keyset cursor and max limit 100. Owner history uses `(occurred_at DESC,event_id DESC)` internally but never exposes event IDs. Owner single-entry lookup accepts case-insensitive `MOD-<UUIDv4>` and authorizes by the session DID.

Standard error mapping:

| Condition | Status | Error code |
|---|---:|---|
| Missing/invalid moderator bearer | 401 | `moderator_authentication_failed` |
| Member/dev credential on admin route | 401 | `moderator_authentication_failed` |
| Invalid public reference/cursor/body | 400 | `invalid_case_reference`, `invalid_cursor`, or `invalid_request` |
| Owner cannot see entry | 404 | `moderation_case_not_found` |
| Stored suspension denies mutation | 403 | `account_suspended` |
| Stale expected revision | 409 | `case_revision_conflict` |
| Replay identity reused for different fingerprint | 409 | `idempotency_conflict` |
| Invalid policy combination/state transition | 422 | `validation_failed` with camelCase field keys |

Every error uses `{error,message,requestId}`. No response/log message includes report text, evidence, correspondence content, credential values, or internal event IDs.

### Report Intake Flow

```text
Authenticated report request
-> existing boundary validation and canonical snapshot
-> ReportStore transaction + owner lifecycle guard
-> insert moderation_reports row
-> moderation.IntakeStore.AttachAcceptedReportTx
-> find/insert one open case by canonical subject_key
-> insert moderation_case_reports association
-> commit
```

Concurrent first reports rely on the partial unique index and conflict retry/select in the same transaction. A resolved case never matches the open-case query, so later intake creates a new UUIDv4 case.

### Adjudication And Notification Flow

```text
Moderator middleware
-> authenticated source/actor context
-> admin adapter + Idempotency-Key + expectedRevision
-> pure policy validation
-> AdjudicationService transaction and fixed lock order
-> append case/effect events
-> existing visibility output bridge when selected
-> strike/standing projections
-> notification eligibility policy
-> actorless notification_events insert in same transaction
-> existing push fan-out/dispatcher
```

Moderation payload data contains `payloadVersion`, `type=moderation`, `accountSubscriptionId`, `notificationId`, and `caseReference`. It contains no actor DID, recipient DID, moderator identity, report data, decision ID, or event ID.

### Expiry Flow

The worker polls at a default five-minute interval with a maximum configuration of 15 minutes and a bounded batch. Candidate selection finds owner DIDs with due active strikes without retaining row locks. For each owner transaction, lock standing first, then due strikes/cases; append deterministic expiry events; update strike/standing projections; lift only threshold suspension when no other basis remains; enqueue only if effective enforcement changed; commit. Record overdue work once the due timestamp is more than one hour old.

### Scoped Inline-Pgx Exception

All new query text lives in `internal/moderation/store.go` or a small file split within that package by read/write concern. The service accepts interfaces, not `*pgxpool.Pool`. API handlers never execute SQL. Transaction bridges for reports, visibility outputs, and notification events accept `pgx.Tx`. This exception is limited to the moderation feature and should be removed or migrated if the repository later bootstraps sqlc globally.

## 6. State, Providers, Controllers, Or DI

### AppView Dependency Graph

```text
pgxpool
-> moderation.Store
-> moderation.IntakeStore
-> moderation.AdjudicationService
   -> VisibilityOutputWriter
   -> NotificationIntentWriter
   -> observability.Observer
-> moderation.ExpiryProcessor
-> admin handler interfaces
-> owner reader interfaces
-> moderation enforcement middleware StandingReader
```

Add only narrow interfaces to `routes.Dependencies`. Keep concrete construction in `internal/app/deps.go`; adapt validated process configuration in `routes_adapter.go`. Existing dev synthetic moderation routes continue to use their current token/store and cannot acquire production moderator context.

Configuration fields:

```text
MODERATION_ADMIN_ENABLED=false
MODERATION_ADMIN_BEARER_TOKEN=<secret>
MODERATION_ADMIN_ACTOR_ID=<stable operator id>
MODERATION_SOURCE_DID=<validated DID used for moderation outputs>
MODERATION_EXPIRY_POLL_INTERVAL=5m   (maximum 15m)
MODERATION_EXPIRY_BATCH_SIZE=100
```

Production startup fails closed when admin is enabled and any credential/identity is missing or invalid. Token comparison is constant-time. Configuration diagnostics use redacting secret wrappers.

### Flutter Provider Graph

```text
active AccountKey
-> accountModerationRepositoryProvider(account)
-> accountStandingProvider(account)          // FutureProvider or generated async provider
-> moderationHistoryProvider(account)        // generated AsyncNotifier with items/cursor/loadMore
-> AccountStandingPage

moderationMailLauncherProvider               // injectable Future<bool> Function(Uri)
moderationClipboardProvider                  // injectable copy boundary or existing Clipboard channel
```

Use account-keyed providers so state cannot cross account switches. Follow existing generation conventions (`@riverpod`, `.g.dart`, `dart_mappable`) and invalidate moderation providers in the account-boundary invalidation path. History pagination keeps existing items during `loadMore`, deduplicates by public case/event presentation identity, and rejects late completions after account/session generation changes.

Do not add a second notification-open coordinator. Extend destination inference only; the existing flow already resolves `accountSubscriptionId`, activates the exact account lease, rechecks it, then emits navigation.

## 7. UI, Widgets, Routes, Or User-Facing Surfaces

### Flutter Routes

```text
/profile/settings/moderation
/profile/settings/moderation/:caseReference
```

Add generated typed routes under the existing settings branch. The focused route validates/canonicalizes the case reference before loading. Invalid or unauthorized references show the standard route/page failure state without leaking whether another account owns the case.

### Account Standing Page

Widget composition:

```text
AccountStandingPage
-> Scaffold/AppBar
-> constrained responsive scroll body
-> StandingSummaryCard
   -> current state
   -> active strike count and threshold
   -> threshold/severe explanation without allowance wording
-> ModerationHistoryList
   -> ModerationHistoryEntry cards
   -> load-more control/progress
-> empty/error/retry states
```

Each entry renders the localized controlled reason, separately authored safe detail, safe subject snapshot/reference, event date, consequences, active/expired/overturned state, expiry where relevant, suspension impact, and appeal state. Formal warning and viewer-facing warning use distinct labels. No color-only status distinction. Focused notification navigation scrolls or directly displays the matching entry.

### Appeal Interaction

Build the `mailto:` URI from constants and the canonical public reference. The subject contains the reference; body copy may include a localized prompt but no private case data. Launch through an injectable provider. Regardless of launch success, never mutate appeal state. Always expose copy-address and copy-reference actions, using existing messenger feedback patterns.

### Notification Settings

Add `NotificationCategory.moderation` to known/preference values and the backend category registry. Like `instagramMatch`, moderation has fixed `everyone` scope, so its card shows explanation plus only the push toggle. Preserve optimistic update/rollback and independent field merging in existing notification preference notifiers.

### Suspended Client State

Server middleware is authoritative. Flutter may disable or annotate denied actions after standing loads, but no acceptance test relies on hidden controls. Reads, safety/private-maintenance surfaces, logout, and account deletion remain reachable. A 403 `account_suspended` from stale UI routes to or offers Account standing and displays localized guidance rather than the raw server message.

### Retool

Retool configuration remains external and manual. It uses only the documented admin endpoints and scoped bearer secret. Queue forms present disposition and consequence dimensions separately, send `expectedRevision` and a unique `Idempotency-Key`, display 409 stale conflicts by reloading detail, and never receive database credentials.

## 8. Error, Loading, Empty, And Edge States

| State / Case | Planned Handling | Requirement IDs | Test IDs |
|---|---|---|---|
| Concurrent first reports | Partial unique index plus select-after-conflict attaches both accepted reports to one open case. | FR-001, RULE-007 | AT-009, IT-001 |
| Report after resolution | Open-only lookup creates a new UUIDv4 case; append commands never reopen. | FR-002, FR-029 | AT-009, IT-002 |
| Lost response/replay | Same source/replay key and fingerprint returns stored result; different fingerprint returns 409. | FR-005, FR-024 | AT-005, IT-003 |
| Stale dashboard revision | Reject 409 before any event/effect/outbox write; Retool reloads. | FR-005, NFR-002 | AT-005, IT-003, MAN-001 |
| Transaction failure | Roll back decision, projections, outputs, standing, and notification event together. | NFR-001 | IT-003, IT-016 |
| Invalid disposition/effects | Return 422 field errors with no side effects. | FR-012, FR-027, FR-033 | AT-008, UT-002, IT-024 |
| Due but unprocessed strike | Continue using stored active/suspended projection until worker commit; expose no request-time restoration. | FR-034, RULE-002 | AT-007, UT-001, IT-007 |
| Threshold plus severe basis | Removing one basis leaves effective suspension while the other remains. | FR-016, FR-017, RULE-004 | AT-007, IT-006, IT-007 |
| Partial reversal/reapplication | Validate exact active logical effects, keep others, require rationale/new replay identity, notify only on consequence change. | FR-030, RULE-003, RULE-009 | AT-010, IT-008 |
| Untrusted appeal correspondence | Store intake metadata only; no pending state until moderator confirmation. | FR-011, RULE-016, RULE-017 | AT-006, UT-008, IT-009 |
| Missing/deleted live subject | Use retained report safe snapshot/reference; clearly mark live content unavailable. | FR-022 | UT-006, IT-012 |
| Suspended retained action | Explicit capability class permits it without changing suspension. | FR-018 | AT-003, IT-013 |
| Suspended denied/unknown action | 403 `account_suspended` before body, store, upload, external, or PDS work. | FR-018, NFR-005 | AT-003, IT-013, IT-014 |
| No owner moderation history | Show good-standing summary and localized empty history; do not imply reports exist. | BR-001, RULE-011 | AT-001, IT-012 |
| History initial load failure | Keep page shell and show retry; do not show stale data from another account. | FR-007, FR-008 | AT-001, UT-010 |
| History next-page failure | Keep loaded entries, stop progress, show retry action; round-trip the same opaque cursor only on retry. | FR-007, NFR-004 | IT-012 |
| Email handler unavailable | Keep copyable address/reference actions; launch failure creates no appeal state. | FR-009, FR-010 | AT-002, MAN-002 |
| Push preference changes while queued | Dispatcher rechecks current preference after lease claim and cancels provider attempt only. | FR-020, RULE-010 | IT-017 |
| Different retained account opens push | Existing opaque binding activates exact account and rechecks lease before moderation navigation. | FR-021 | AT-004, IT-026 |
| Invalid/removed/stale push binding | Show generic unavailable/removed outcome; reveal no case details and do not navigate. | FR-021, NFR-003 | IT-026, MAN-003 |
| Legacy rows at migration | Preserve reports/outputs and visibility behavior; create no synthetic moderation state. | FR-036 | IT-020, REG-008 |
| Alert boundary | Activate only at five failures/five minutes, expiry over one hour, or eligible notification age 15 minutes. | NFR-008 | UT-014, IT-021 |

## 9. Test Implementation Plan

| Order | Test ID | Target | Setup / Fixture | Initial Expected Failure |
|---:|---|---|---|---|
| 1 | UT-005 | `internal/moderation/reference_test.go` | UUIDv4 and malformed/mixed-case references. | No parser/formatter package exists. |
| 2 | IT-020, REG-008 | `internal/db/moderation_cases_migration_test.go` | Pre-000072 schema with legacy reports/outputs. | Migration/tables/constraints/category do not exist. |
| 3 | IT-001, IT-002, REG-001 | `internal/moderation/adjudication_service_integration_test.go`, existing report tests | `testdb.WithSchema`, concurrent barriers, resolved case. | Reports do not create or join cases. |
| 4 | UT-002, UT-009 | `internal/moderation/policy_test.go`, `visibility_effects_test.go` | Full reason/disposition/consequence table. | Decision policy/types do not exist. |
| 5 | IT-003, IT-004, IT-015, IT-022 | `internal/moderation/adjudication_service_integration_test.go` | Failure triggers, replay fingerprints, stale revisions, concurrent locks. | No atomic adjudication/replay/output bridge exists. |
| 6 | UT-001, UT-003 | `internal/moderation/expiry_test.go`, `standing_test.go` | Injected UTC dates, leap day, independent bases. | Due-date and standing derivation do not exist. |
| 7 | AT-007, IT-005 through IT-008 | Moderation integration/expiry suites | Three cases, overlapping bases, due/unprocessed state, reversals. | Strike/standing/expiry projections and worker do not exist. |
| 8 | UT-008, AT-006, IT-009 | Appeal unit/integration suites | Untrusted and moderator-confirmed correspondence. | Appeal lifecycle boundary does not exist. |
| 9 | UT-013, IT-019 | Adapter unit/integration suites | Equivalent trusted normalized commands. | Source-neutral adapter/service interface does not exist. |
| 10 | UT-011, AT-005, IT-010, IT-011 | Admin handler/route tests | Admin/member/dev/invalid credentials, pages/cursors. | Production moderator auth/routes do not exist. |
| 11 | UT-006, IT-012, IT-023 through IT-025 | Owner handler/presentation tests | Sensitive sentinels, mixed history, deleted subjects. | Owner-safe API/DTOs do not exist. |
| 12 | UT-004, AT-003, IT-013, IT-014, REG-003, REG-006, REG-007 | Route capability/middleware suites | Full route catalogue, handler/PDS/store spies, unknown route. | Route policies have no suspension class or middleware. |
| 13 | UT-012, IT-016 | Notification policy/adjudication tests | Every event eligibility class and rollback/replay injection. | Moderation events do not create actorless notification work. |
| 14 | UT-007, IT-017, IT-018, REG-004 | Notification/push suites | Moderation category, preference changes, safe sentinels, retries. | Category/payload/preferences do not support moderation. |
| 15 | AT-011, IT-027 | Backend preference and Flutter settings tests | Existing categories plus ProviderScope/Dio adapter. | Moderation preference is absent from API and UI. |
| 16 | UT-010, AT-001, AT-002, IT-025 | Flutter moderation model/repository/widget/router tests | Owner wire fixtures, launch/clipboard fakes, responsive matrix. | Models, providers, routes, and page do not exist. |
| 17 | AT-004, IT-026 | Existing notification open flow plus moderation route tests | Two retained accounts and invalid/removed/stale bindings. | Destination inference cannot route moderation references. |
| 18 | UT-014, IT-021 | Observability moderation suites | Clocked threshold boundaries and sensitive sentinels. | Moderation metrics/alert predicates are absent. |
| 19 | REG-002, REG-005 | Existing moderation-output and route architecture suites | Existing fixtures unchanged. | Any integration drift becomes visible. |
| 20 | MAN-001 through MAN-004 | External/manual checklist | Retool, devices, email handlers, accessibility matrix. | Operational/UI evidence has not been collected. |

Focused commands by slice:

```text
cd appview && TEST_DATABASE_REQUIRED=false go test ./internal/moderation ./internal/api ./internal/routes ./internal/notifications ./internal/push ./internal/observability
just dev-d
just test
just appview-check
just app-test test/moderation test/router/moderation_routes_test.dart test/notifications
just app-analyze
```

Run repository-standard generation after annotated Dart model/provider/route changes:

```text
cd app && dart run build_runner build --delete-conflicting-outputs
cd app && flutter gen-l10n
```

## 10. Sequencing And Guardrails

- First TDD step: Add `UT-005` for public case reference parsing/formatting, then add `IT-020` as the first PostgreSQL failure before writing migration `000072`.
- Dependency order: reference types -> migration/invariants -> intake grouping -> decision policy/service -> standing/expiry -> appeals/effect changes -> admin/owner APIs -> suspension middleware -> notifications/preferences -> Flutter UI/routing -> observability/deletion -> release regression.
- Transaction guardrail: No handler or adapter writes moderation tables directly. All coupled state changes use one service-owned transaction and the fixed lock order.
- Audit guardrail: Never update/delete append-only case/effect events in ordinary operation. Projections may change only alongside a new event.
- Replay guardrail: Durable source/replay identity and fingerprint are checked before effects. Same fingerprint replays; different fingerprint conflicts.
- Privacy guardrail: Owner DTOs are explicit allowlists. Tests serialize responses, logs, metrics labels, audit records, and push payloads against sensitive sentinels.
- Suspension guardrail: `SuspensionClass` zero is invalid. Catalogue tests fail for every route without an explicit class; unknown mutations fail closed.
- Expiry guardrail: Stored standing is authoritative until worker commit. No request-time expiry fallback or independent count calculation is added.
- Notification guardrail: Create notification events only inside the authoritative transaction. Preference affects provider dispatch only, never history or enforcement.
- PDS guardrail: Moderation transitions never call PDS mutation clients. Only retained owner-requested removal routes may perform their existing PDS deletion behavior.
- Legacy guardrail: Migration contains no `INSERT ... SELECT` from legacy reports/outputs into new case/history/enforcement structures.
- Identity guardrail: Use typed `syntax.DID`/`syntax.ATURI` after request/database boundaries; case reference is UUIDv4, not an atproto identifier.
- Flutter guardrail: Account-key all private moderation state and preserve existing notification exact-account activation before navigation.
- Generated-file guardrail: Edit ARB/annotated Dart sources, then regenerate; do not hand-edit generated mappers/routes/localizations.
- Verification guardrail: `just appview-check` is required release evidence. Unit-only tests cannot approve migration, concurrency, or transaction behavior.
- Out of scope: Live Ozone ingestion/mapping, native appeal submission, verified email, automated email processing, incident grouping, moderator CLI, lexicon changes, identity/PDS deactivation, and guaranteed OS push presentation.

## 11. Risks And Open Questions

| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Non-blocking | Exact Retool screen layout and secret-rotation runbook live outside this repository. | Manual operational setup cannot be automated here. | Use MAN-001 against the fixed admin API; document rotation with deployment configuration before enabling admin access. |
| CPQ-002 | Non-blocking | The repository documents sqlc but has no sqlc setup and current stores use inline `pgx`. | Bootstrapping sqlc would expand scope and complicate shared transactions. | User approved a scoped inline-`pgx` exception. Encapsulate it in `internal/moderation` and transaction bridges only. |
| CPQ-003 | Non-blocking | Exact localized wording is not yet approved. | Copy may change without changing behavior. | Implement keys/placeholders and complete MAN-004 copy review before release; preserve semantic distinctions in tests. |
| CPQ-004 | Non-blocking | Only English localization resources currently exist. | Multi-locale linguistic validation is unavailable. | Require generated localization usage and accessibility/layout tests now; add locale tests when translations exist. |
| CPQ-005 | Non-blocking | The initial admin credential is one shared operator identity. | Attribution is operator-level rather than person-level. | Store stable actor/source fields on every event; later identity auth can replace middleware without changing service commands. |
| CPQ-006 | Non-blocking | Real provider/OS display and email app behavior are external. | Automated tests cannot prove presentation. | Verify payload/URI/routing deterministically and complete MAN-002/MAN-003 on representative devices. |
| CPQ-007 | Resolved | Notification opens for another retained account need safe behavior. | Wrong-account history could leak private moderation data. | Reuse existing opaque binding resolution and automatic exact-account lease activation/recheck; no raw DID in payload. |
| CPQ-008 | Resolved | Route names and admin credential configuration were deferred from requirements. | TDD needs concrete contracts. | Use the HTTP routes, error codes, headers, and environment names fixed in Sections 5 and 6 of this plan. |

No blocking coding-plan questions remain.

## 12. Handoff To TDD Builder

- Coding plan: `docs/changes/2026-09-09-moderation-strikes/04-coding-plan.md`
- TDD execution plan: `docs/changes/2026-09-09-moderation-strikes/05-implementation-plan.md`
- Start with test: `UT-005` for `ParseCaseReference`/`FormatCaseReference`, followed immediately by PostgreSQL `IT-020` for migration and legacy preservation.
- Focused command: `cd appview && TEST_DATABASE_REQUIRED=false go test ./internal/moderation`, then run `just dev-d && just test` once `IT-020` is introduced.
- Notes: Preserve all requirement, AC, and test IDs in implementation commits and `05-implementation-plan.md`. Explicit user approval is required before implementation starts. Do not create lexicon files or expose OAuth/PDS credentials to Flutter.
