# Coding Plan: Unified Tap-Authoritative PDS Record Mutations

## 1. Inputs

- Requirements: `docs/changes/2026-09-17-unified-pds-record-mutations/01-requirements.md`
- Acceptance tests: `docs/changes/2026-09-17-unified-pds-record-mutations/02-acceptance-tests.md`
- Document review: `docs/changes/2026-09-17-unified-pds-record-mutations/03-document-review.md` (`Approved`)
- API conventions: `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md`
- Architecture reference: `atproto-craft-social-app-reference.md`
- Risk: High. Implementation requires the ordered phase gates in FR-049 and all release evidence in AC-043.
- Schema decision: Create migration 73 directly as the final development architecture. There is no production migration compatibility obligation and no relationship-intent compatibility table.
- SQL decision: Use the repository's current scoped inline-`pgx` pattern. `appview/queries/` contains no active sqlc definitions or generated package, so bootstrapping sqlc is outside this change.
- Lexicon decision: No Lexicon shape changes are planned. Existing generated record types remain validation inputs.

## 2. Implementation Strategy

Implement two independent but coordinated systems:

1. A Tap-authoritative source pipeline in which `tap_source_records` retains current repository evidence, `pds_set_sources` retains normalized set facts, and `pds_set_aggregates` alone supplies logical follow/block/like/repost serving state.
2. A durable AppView command journal in which authenticated mutation endpoints prepare immutable commands before PDS dispatch, retain exact attempts for recovery, and return terminal or ambiguous outcomes without becoming a public read overlay.

Flutter adds one account-fenced, in-memory controller for keyed user mutations. It owns operation identity, bounded ambiguous retry, accepted overlays, and class-specific reconciliation against ordinary Tap-derived reads. The server never exposes a status resource, and ordinary AppView reads never merge command state.

Use the exact FR-049 vertical order:

1. Establish migration, source validation, source/fact/aggregate projection, command journal, authoritative PDS reader, protocol transport, rebuild, and Flutter-controller foundations.
2. Prove the complete architecture through like/unlike.
3. Generalize set commands to repost, follow, block, and Instagram suggestion follow acceptance.
4. Migrate immediate post create/delete.
5. Migrate business event and fixed-key business-profile writes.
6. Replace personal-profile partial writes with one atomic two-record command.
7. Adapt scheduled final publication to the AppView journal only.
8. Remove superseded effect-origin, relationship-intent, winner, and feature-overlay callers and run full conformance.

Do not remove a legacy implementation merely because its replacement exists. Remove or narrow it only after the corresponding vertical slice passes its backend, protocol, Flutter, ordinary-read, lifecycle, and regression tests.

## 3. Architectural Invariants

### Public State Authority

- PDS repository state is authoritative for record existence.
- `tap_source_records` is the sole durable owner of raw record JSON, repository revision, action, CID, ordering evidence, and versioned validation status.
- Validity and local projection eligibility are separate. Valid-but-blocked or policy-ineligible records remain source facts where required but do not count in active aggregates.
- Commands describe requested effects and delivery outcomes only. They are never queried by ordinary public reads or projection rebuilds.
- Tap projectors remain the only ordinary writers of public serving projections.
- Missing events and Tap omission never prove deletion. Only a later source version or a complete verified repository snapshot may establish absence.

### Set Semantics

- Every valid current follow, block, like, and repost URI has one normalized fact.
- Logical activity is `eligible_source_count > 0`; representative URI is metadata only.
- Recompute every old and new scope touched by source replacement in one transaction.
- Acquire aggregate locks by sorted canonical scope key to avoid opposite-order deadlocks.
- Choose representatives by ascending activity time and then ascending URI.
- Preserve activation time and notification delivery/newness through representative churn.
- Emit activation only on `0 -> 1` and retraction only on `1 -> 0`.
- Indexers, rebuilds, and repair workers never delete PDS records as normalization.

### Command Semantics

- Command identity is `(authenticated owner DID, operation kind, canonical UUID)`.
- Persist command identity, immutable request fingerprint, selected identity, lifecycle fences, CAS/head inputs, and ordered steps before dispatch.
- Persist each exact dispatch attempt separately; rebuilding steps after `InvalidSwap` creates another immutable attempt under the same command.
- States are `prepared`, `dispatching`, `accepted`, `ambiguous`, and `rejected`; accepted and rejected are terminal.
- Request and dispatch fingerprints use centralized canonical JSON, versioned domain separators, and SHA-256.
- Full terminal responses replay for 24 hours. Cleanup atomically replaces command, response, steps, and attempts with a minimal scoped key-hash tombstone. Unresolved commands never compact.
- Every dispatch and recovery rechecks authentication-derived owner, owner generation, expected-owner/target fences, deletion policy, and terminal-owner policy inside the existing lifecycle boundary.
- Genuine ambiguity returns exactly `202`, `Content-Type: application/json`, `{"status":"ambiguous"}`, integer `Retry-After` clamped to 1-5 seconds, and no `Location`.
- Successful deletes retain empty `204` responses.

### Flutter Semantics

- Operation keys, immutable requests, timers, sequences, and overlays are in memory only.
- An ambiguous operation retries the original endpoint with the same key and byte-equivalent logical request; edits remain frozen.
- Automatic delay is `max(clamped Retry-After, local backoff) + jitter`, where backoff is `1, 2, 4, 5, 5, 5` seconds and jitter is inclusive `0-250 ms`.
- Automatic retry makes at most six retries and stops before exceeding 30 seconds. Explicit same-key retry remains available.
- Definite acceptance stops mutation retry, starts the overlay lifetime from local receipt, invalidates ordinary reads immediately, and schedules exactly one grace refresh after two seconds.
- Overlay hard expiry is 30 seconds. Expiry retires the overlay and forces one refresh; a failed or late refresh cannot resurrect it.
- Account generation and monotonic logical-scope sequence fence every response, retry, read, and timer.
- Controller restart, process restart, sign-out, account removal, account switch, and permanent-deletion completion restore no mutation state.

## 4. Affected Areas

| Area | Existing Pattern | Planned Change | Requirement IDs | Acceptance / Test IDs |
|---|---|---|---|---|
| Source persistence | `ingestion.Store` retains source records but carries effect-origin projection fields. | Make current repository version and versioned validation state authoritative without command matching. | FR-001 through FR-006, FR-019, FR-029, FR-046 | AC-001, AC-002, AC-005 through AC-010, AC-023, AC-034, AC-054; UT-001, UT-002, UT-017; IT-001, IT-013 |
| Set projection | Feature tables collapse or replace duplicate physical records. | Add per-URI normalized facts and logical aggregates with sorted locks and edge transitions. | FR-007 through FR-011, FR-042, FR-046, NFR-001, NFR-002 | AC-011 through AC-015, AC-041, AC-050, AC-054; UT-003 through UT-005, UT-017; IT-002 through IT-004 |
| Repair/rebuild | Repository repair and effect reconciliation are partly coupled to prior effects. | Verify complete snapshots, reconcile source absence, and rebuild without commands or PDS cleanup. | FR-006, FR-025, FR-037, NFR-006 | AC-010, AC-030, AC-045; IT-005, REG-008 |
| Command durability | `owner_effect_attempts` combines attempt identity, dispatch, and projection-origin correlation. | Add separate commands, steps, dispatch attempts, and tombstones with immutable scoped identity. | FR-012 through FR-019, FR-038, FR-044, FR-045 | AC-003, AC-016 through AC-024, AC-046, AC-052, AC-053; UT-006, UT-007, UT-015, UT-016; IT-006, IT-007 |
| PDS protocol | Narrow single-record client plus optional list/conditional capabilities. | Add repository-head reads, raw pagination, ordered `applyWrites`, explicit swap/unsupported outcomes, and verified snapshot fallback. | FR-017, FR-020 through FR-024, FR-041, FR-047 | AC-021, AC-025 through AC-029, AC-049, AC-055; UT-008; IT-008, IT-009, IT-015 |
| Mutation HTTP | Feature handlers call `pdseffects.Executor` and return feature-specific outcomes. | Require canonical `Idempotency-Key`, call one command service, and map shared outcomes while retaining route/response compatibility. | FR-013, FR-016, FR-018, FR-020, FR-027, FR-043 | AC-017, AC-020, AC-022, AC-025, AC-032, AC-051; UT-009; IT-007, IT-014 |
| Lifecycle/security | Coordinated PDS boundary and owner generation fence current effects. | Preserve those guards for prepare, dispatch, reconciliation, recovery, cleanup, and purge. | FR-028, FR-029, FR-035, NFR-003 | AC-033, AC-034, AC-039, AC-042; IT-011, REG-005, REG-007 |
| Flutter mutation state | Providers own optimistic transforms, rollback, and ad hoc reconciliation. | Add one shared account-scoped controller and make feature providers adapters over it. | BR-003, FR-030 through FR-033, FR-039, FR-040, FR-043, FR-048 | AC-004, AC-035 through AC-037, AC-047, AC-048, AC-051, AC-056; UT-010, UT-011, UT-014, UT-018; IT-012, IT-016 |
| Scheduled publication | Private schedule recovery ultimately uses the old PDS effect path. | Keep private scheduling state and move only final public publication to the journal. | FR-026 | AC-031; IT-010, REG-006 |
| Observability | Existing PDS/Tap observers expose bounded operational stages. | Add command/source/dependency/aggregate/convergence dimensions without values or secrets. | FR-028, NFR-003, NFR-005 | AC-042, AC-044; UT-012, IT-011, MAN-001 |
| Legacy conformance | Effect gating, relationship intents, winner replacement, and feature overlays remain reachable. | Remove all migrated callers after final slice; retain private/excluded workflows outside the new contract. | FR-025, FR-035, FR-049, RULE-008 | AC-030, AC-039, AC-057; UT-019, IT-017, REG-004, REG-009, MAN-003 |

## 5. Files And Modules

### Database And Source Projection

| Path / Module | Create / Change | Purpose |
|---|---|---|
| `appview/migrations/000073_unified_pds_record_mutations.up.sql` | Create | Add validation-state fields, `pds_set_sources`, `pds_set_aggregates`, four command-journal tables, constraints, indexes, and owner purge relationships as the final schema. |
| `appview/migrations/000073_unified_pds_record_mutations.down.sql` | Create | Remove migration-73 structures in dependency-safe order without changing prior source or serving records. |
| `appview/internal/db/unified_pds_record_mutations_migration_test.go` | Create | First red PostgreSQL test for exact table ownership and forbidden legacy paths. |
| `appview/internal/ingestion/store.go` | Change | Install latest source versions and validation state without effect-origin eligibility gating. |
| `appview/internal/ingestion/repository_repair.go` | Change | Reconcile verified snapshot presence/absence and initiate source-only rebuild. |
| `appview/internal/ingestion/repository_snapshot.go` | Change | Share verified signed-CAR construction/verification primitives with authoritative command reads where safe. |
| `appview/internal/ingestion/reconciliation.go` | Change / Remove legacy branches | Stop public projection eligibility from depending on effect attempts. |
| `appview/internal/ingestion/effect_attempt_reconciliation.go` | Remove after migration | Delete public effect-origin reconciliation once no caller remains. |
| `appview/internal/index/source_validation.go` | Create | Central validation registry that returns validity, normalized fact, eligibility, and durable dependency separately. |
| `appview/internal/index/set_aggregate.go` | Create | Replace facts, lock sorted scopes, recompute aggregates, select representative metadata, and emit transition data in caller transaction. |
| `appview/internal/index/dependency.go` | Create | Register durable dependency wake-ups without Tap replay. |
| `appview/internal/notifications/set_transition.go` | Create | Convert aggregate width edges into stable logical notification activation/retraction. |
| `appview/internal/index/craftsky_interaction.go` | Change | Project like/repost source facts instead of replacing physical duplicates. |
| `appview/internal/index/bluesky_follow.go` | Change | Project each follow URI and recompute the logical scope. |
| `appview/internal/index/bluesky_block.go` | Change | Project each block URI and recompute the logical scope. |
| Set-state consumers under `appview/internal/api/`, `relationships/`, `notifications/`, `push/`, `business/`, `instagram/`, and `followergrowth/` | Change by owning slice | Read `pds_set_aggregates` for logical activity/counts and stop counting or authorizing from physical follow/block/like/repost rows. Include timeline/profile/search/facet/interaction-list stores, notification eligibility/newness, push eligibility, relationship authorization, business/Instagram eligibility, and follower-growth queries. |

### Command Journal And PDS Transport

| Path / Module | Create / Change | Purpose |
|---|---|---|
| `appview/internal/pdscommands/command.go` | Create | Typed operation kinds, immutable command requests, lifecycle fences, results, and states. |
| `appview/internal/pdscommands/store.go` | Create | Transactional prepare/replay, step installation, dispatch-attempt append, state transitions, and recovery claims. |
| `appview/internal/pdscommands/fingerprint.go` | Create | Versioned request and dispatch fingerprint entry points. |
| `appview/internal/pdscommands/canonical_json.go` | Create | One deterministic JSON encoder for fingerprint inputs and golden vectors. |
| `appview/internal/pdscommands/dispatcher.go` | Create | Recheck fences, mark dispatching, execute one retained plan, and classify definite/ambiguous outcomes. |
| `appview/internal/pdscommands/reconciler.go` | Create | Resolve ambiguous addressed, append, set, fixed-key, and compound plans before any redispatch. |
| `appview/internal/pdscommands/authoritative_reader.go` | Create | Head-before/head-after paginated reads with signed-CAR fallback and complete-snapshot limits. |
| `appview/internal/pdscommands/set_command.go` | Create | Build create/no-op or complete delete plans from authoritative valid matches; apply three-attempt swap policy and atomic cap. |
| `appview/internal/pdscommands/record_command.go` | Create | Build append, addressed, fixed-key, and compound plans with frozen generated values and CAS. |
| `appview/internal/pdscommands/retention.go` | Create | Compact terminal commands after 24 hours and preserve unresolved plans. |
| `appview/internal/pdscommands/errors.go` | Create | Stable internal outcomes including repository conflict, atomic-too-large, unsupported atomic write, and ambiguity. |
| `appview/internal/pdscommands/matrix.go` | Create | Declarative inventory of included/excluded mutation classes and per-class identity/CAS/reconciliation strategy. |
| `appview/internal/auth/pds_client.go` | Change | Add narrow typed repository-head, raw list, ordered apply-writes, and explicit swap/unsupported contracts. |
| `appview/internal/auth/pds_client_indigo.go` | Change | Encode `com.atproto.sync.getLatestCommit`, raw `listRecords`, `applyWrites`, `swapCommit`, and `swapRecord` exactly. |
| `appview/internal/auth/coordinated_pds_client.go` | Change | Delegate the new capabilities while retaining active-effect lifecycle fencing. |
| `appview/internal/api/command_response.go` | Create | Map shared outcomes to existing success payloads, empty delete, exact ambiguity response, and standard camelCase errors. |
| `appview/internal/observability/pds.go` and Tap observer files | Change | Record only bounded stage/outcome/kind/retry/latency dimensions. |
| `appview/internal/ownerlifecycle/terminal_inventory.go` | Change | Register command, step, dispatch, tombstone, source-fact, and aggregate owner roles where applicable. |
| `appview/internal/accountdeletion/store.go` and cleanup tests | Change | Purge all owner command/tombstone state while preserving the restricted external PDS deletion workflow. |

### Flutter Shared Mutation Layer

| Path / Module | Create / Change | Purpose |
|---|---|---|
| `app/lib/shared/mutations/pds_record_operation.dart` | Create | Immutable operation identity, account generation, scope sequence, endpoint/request, and UI state. |
| `app/lib/shared/mutations/pds_record_operation_controller.dart` | Create | Account-scoped Riverpod controller for dispatch, ambiguity, explicit retry, acceptance, failure, reset, and late-result fencing. |
| `app/lib/shared/mutations/pds_record_operation_overlay.dart` | Create | Accepted overlay state, fakeable deadlines, refresh scheduling, and newest-scope presentation selection. |
| `app/lib/shared/mutations/pds_record_reconciliation.dart` | Create | Strongly typed reconciliation predicates for append, addressed, set, fixed-key, and compound-profile classes. |
| `app/lib/shared/api/pds_mutation_contract.dart` | Create | Canonical UUID header, exact `202` parsing, empty `204` handling, retry metadata, and shared response classification. |
| `app/lib/auth/providers/account_boundary_provider.dart` | Change | Advance the shared controller generation and clear all operations/overlays before invalidating feature reads. |
| Existing feature API clients/repositories | Change | Accept or generate no retry identity themselves; send the shared operation key and return shared mutation outcomes. |
| Existing feature providers | Change | Delegate operation state/overlay timing to the shared controller and retain only feature-specific read invalidation and pure overlay transforms. |

### Vertical Slice Files

| Slice | Backend Files | Flutter Files | Primary Tests |
|---|---|---|---|
| Like/unlike | `internal/api/post_interactions.go`, `internal/index/craftsky_interaction.go`, post engagement/read/list/timeline/profile/search stores, `internal/notifications/`, `internal/push/`, `internal/app/deps.go` | `feed/data/post_api_client.dart`, `post_repository.dart`, `feed/providers/toggle_like_post_provider.dart` | `internal/index/set_aggregate_integration_test.go`, `internal/api/post_interactions_test.go`, affected read/notification/push tests, `app/test/feed/providers/toggle_post_interactions_provider_test.dart`, `app/test/feed/data/post_api_client_test.dart` |
| Repost/unrepost | Same interaction handler/indexer plus every repost count/viewer/list/notification consumer | `feed/providers/toggle_repost_post_provider.dart` plus feed API/repository | Interaction index/API/read/notification and provider suites |
| Follow/unfollow | `internal/api/follow.go`, `follow_store.go`, `internal/index/bluesky_follow.go`, `internal/notifications/service.go`, `internal/push/dispatcher.go`, `internal/instagram/eligibility_policy.go`, `internal/followergrowth/`, and profile/search/facet consumers | `profile/data/profile_api_client.dart`, `profile_repository.dart`, `profile/providers/toggle_follow_profile_provider.dart` | Follow/index/notification/push/Instagram/follower-growth/read and provider/API-client tests |
| Block/unblock | `internal/api/relationship.go`, `internal/relationships/mutation_service.go`, `internal/relationships/store.go`, `internal/index/bluesky_block.go`, `internal/notifications/service.go`, `internal/push/dispatcher.go`, and all visibility/authorization consumers | `profile/providers/profile_relationship_provider.dart` | Relationship/block/index/notification/push/visibility and provider tests |
| Suggestion acceptance | `internal/instagram/suggestions.go`, `internal/app/instagram_suggestion_follow.go`, `internal/api/instagram_suggestions.go` | Instagram migration repository/provider files | Suggestion acceptance, app coordinator, repository/provider tests |
| Immediate posts | `internal/api/post_create.go`, post delete handler in `post_interactions.go`, route composition | `feed/providers/create_post_provider.dart`, `delete_post_provider.dart`, feed API/repository | Post request/write/API-client/provider tests |
| Business records | `internal/api/business_event.go`, `business_profile.go`, business route composition | Business API/repository, `business_event_mutation_controller.dart`, `products_controller.dart`, projection overlay | Business event/profile acceptance, API-client, reconciliation/provider tests |
| Personal profile | `internal/api/profile.go`, `profile_request.go`, profile route composition | Profile API/repository, `profile/providers/save_profile_provider.dart` | `profile_effects_test.go`, `profile_command_test.go`, API-client/provider tests |
| Scheduled publication | `internal/scheduledposts/publication_processor.go` | None | `internal/scheduledposts/recovery_acceptance_test.go`, worker acceptance tests |
| Final wiring/removal | `internal/app/deps.go`, route dependency files, old `pdseffects`/relationship reconciliation callers | Account invalidator and migrated providers; remove superseded feature overlays/helpers | UT-019, IT-014, IT-017, REG-009 |

Generated Riverpod files remain generated artifacts. Edit annotated Dart sources, then run `dart run build_runner build --delete-conflicting-outputs`; do not hand-edit `.g.dart` files.

## 6. Persistence Design

### Source And Aggregate Tables

Migration 73 should make the boundaries mechanically testable:

| Structure | Owned Data And Invariants |
|---|---|
| `tap_source_records` | Existing URI/DID/collection/rkey, latest source event/revision/CID/action/raw record, source fingerprint, observed time, ordering state, plus versioned structural/semantic validation status and bounded reason. Remove public dependence on `effect_operation_id`; no raw source field is copied to set tables. |
| `pds_set_sources` | One row per current semantically valid set source URI. Store set kind, actor DID, canonical logical scope, normalized subject identifiers, activity timestamp, current eligibility, bounded ineligibility/dependency reference, and update time. Invalid/deleted current sources have no fact row. |
| `pds_set_aggregates` | One row per active `(kind, actor, canonical scope)` with eligible source count, representative source URI, stable activation time, and bounded representative metadata. Delete or mark inactive only on final `1 -> 0` according to existing serving-query needs. |

Use foreign keys from normalized facts to `tap_source_records` by source URI with delete behavior that permits transactional fact removal. Add checks for positive active counts, known set kinds, canonical non-empty scope keys, and representative membership. Add indexes for source replacement, dependency wake-up, actor/subject reads, and aggregate serving queries.

`replaceSetSourceTx` should perform this sequence within the projection transaction:

1. Read the prior fact and derive its old scope if present.
2. Validate the installed current source and derive zero or one normalized fact plus eligibility/dependency.
3. Replace or remove the normalized fact.
4. Union old and new scopes, sort by canonical lock key, and acquire per-scope transaction advisory locks in that order.
5. Recompute each aggregate from eligible facts, selecting representative by `(activity_at ASC, source_uri ASC)`.
6. Preserve prior `activated_at` while count remains non-zero; allocate a new activation only on `0 -> 1`.
7. Apply notification transition and serving-row changes in the same transaction.
8. Complete the projection job in that transaction.

### Command Tables

Use the names fixed by AT-012:

| Structure | Owned Data And Invariants |
|---|---|
| `pds_commands` | Owner DID, operation kind, canonical UUID operation key, request-fingerprint version/digest, immutable validated intent needed for reconciliation, selected URI/rkey, owner/target fences, active plan version, state, terminal response metadata/body where needed, retry guidance, timestamps. Unique `(owner_did, operation_kind, operation_key)`; legal states constrained. |
| `pds_command_steps` | Append-only ordered frozen steps by command plan version: action, repo, collection, rkey, canonical body, expected CID, selected identity/generated values. Unique `(command_id, plan_version, ordinal)`; a guarded reread appends a new plan version rather than replacing steps retained for an earlier dispatch. |
| `pds_command_dispatches` | Append-only exact attempts with attempt ordinal, referenced plan version, dispatch-fingerprint version/digest, repository head, started/completed times, bounded outcome/error class, returned commit/CIDs. Never overwrite an ambiguous attempt; the referenced immutable step version is its exact plan. |
| `pds_command_tombstones` | Minimal owner DID, operation kind, scoped key hash, compacted timestamp, and expiry policy needed to reject reuse until owner purge. No record bodies, response payloads, steps, dispatch plans, raw operation key, or credentials. |

Store the canonical UUID on the live command for scoped replay, but never log it. Tombstones retain only the scoped hash. Child rows cascade from commands. Owner purge explicitly covers all four structures. Migration and tests must prove `owner_effect_attempts` cannot receive any operation kind in the public mutation matrix.

### Transactions And Locking

- Command prepare serializes on scoped command identity and checks an existing tombstone before insertion.
- A command row is locked before transitions, step replacement after guarded reread, terminal completion, or compaction.
- A dispatch row and the exact step plan it represents commit before the remote call is issued; post-call completion occurs in a second guarded transaction.
- A crash between durable dispatch intent and remote outcome produces recovery/reconciliation, never blind replay.
- Source/fact/aggregate/notification transitions use a separate projection transaction and never lock command rows.
- Owner lifecycle locking remains outermost around PDS dispatch/recovery. Do not introduce command/source lock cycles.

## 7. Services, Interfaces, And Data Flow

### Internal Interfaces

```text
type CommandService interface {
  Execute(ctx, AuthenticatedCommand) (CommandResult, error)
}

type GuardedCommandCoordinator interface {
  WithGuardedSession(ctx, expectedOwners, func(context.Context, GuardedCapabilities) error) error
}

type GuardedCapabilities interface {
  BlobEffects() BlobEffectExecutor
  Commands() AlreadyFencedCommandExecutor
}

type CommandStore interface {
  PrepareOrReplay(ctx, PrepareRequest) (PreparedCommand, error)
  BeginDispatch(ctx, commandID, ExactPlan) (DispatchAttempt, error)
  CompleteDispatch(ctx, attemptID, DispatchOutcome) error
  CompleteCommand(ctx, commandID, TerminalResult) error
  MarkAmbiguous(ctx, commandID, retryAfter) error
}

type AuthoritativeRepositoryReader interface {
  ExactRecord(ctx, repo, collection, rkey) (RecordState, error)
  CompleteCollection(ctx, repo, collection) (RepositoryHead, []RawRecord, error)
}

type PDSRepositoryTransport interface {
  LatestCommit(ctx, repo) (RepositoryHead, error)
  ListRecords(ctx, repo, collection, cursor, limit) (RawRecordPage, error)
  ApplyWrites(ctx, repo, swapCommit, orderedWrites) (ApplyResult, error)
  PutRecordWithSwap(ctx, repo, collection, rkey, body, swapCommit, swapRecord) (RecordResult, error)
  DeleteRecordWithSwap(ctx, repo, collection, rkey, swapCommit, swapRecord) error
}

type MutationSpec interface {
  Prepare(validatedInput) (ImmutableRequest, error)
  Plan(ctx, command, authoritativeReader) (ExactPlan, error)
  Reconcile(ctx, command, attempts, authoritativeReader) (Resolution, error)
}
```

Keep handlers dependent on `CommandService` or narrower feature adapters, never `*pgxpool.Pool` or a raw OAuth client. Keep PDS protocol types private to `auth`/`pdscommands`; API DTOs preserve existing feature responses.

`CommandService.Execute` acquires the lifecycle/session boundary for ordinary HTTP handlers. `AlreadyFencedCommandExecutor` is an internal composition capability for code that already holds the same coordinated session; it must not reacquire lifecycle or session locks. Scheduled publication receives `GuardedCommandCoordinator`, which supplies blob effects and command execution from one active PDS client under one lifecycle fence. This replaces, rather than nests inside, its current `pdseffects.GuardedExecutorFactory` boundary.

### Mutation Request Flow

```text
authenticated existing /v1 mutation endpoint
-> parse canonical Idempotency-Key where matrix requires it
-> validate body/path and derive owner, operation kind, lifecycle/target fences
-> canonical request fingerprint
-> CommandStore.PrepareOrReplay
   -> replay accepted/rejected response
   -> reject changed fingerprint/generation or tombstoned reuse
   -> continue prepared/dispatching/ambiguous recovery
-> lifecycle-coordinated PDS boundary
-> reconcile prior uncertain attempt before redispatch
-> freeze exact step plan and dispatch fingerprint
-> persist dispatch intent
-> issue one guarded PDS operation
-> persist definite accepted/rejected or ambiguous outcome
-> map shared result to existing endpoint response
```

No command path waits for Tap. No ordinary GET reads command tables.

### Authoritative Set Flow

```text
capture repository CID/revision
-> paginate raw collection with strict cursor/page/byte/time bounds
-> capture repository CID/revision again
-> accept only if both heads are identical
-> otherwise retry read or use fully verified signed-CAR snapshot
-> validate each candidate with the same collection validator as ingestion
-> build deterministic URI-sorted matching set
-> create: no-op on valid match, otherwise one head-guarded create
-> remove: delete every valid match in one head-guarded applyWrites
-> enforce cap before dispatch
-> on InvalidSwap: jitter, reread, rebuild, append another exact attempt
-> after third failed attempt: 409 pds_repository_conflict
```

Attempt two uses full jitter `0-25 ms`; attempt three uses `0-50 ms`; there is no fourth attempt. Zero steps complete as no-op. One step may fall back to the corresponding single-record XRPC only when both repository-head and applicable record guards remain equivalent. Two or more steps never use sequential fallback. A plan above the configurable default cap of 200 returns `422 pds_atomic_mutation_too_large` before any write.

### Source Projection Flow

```text
Tap frame
-> transaction A: install current source version and enqueue projection work
-> durable Tap acknowledgement remains governed by the existing ingestion contract
-> asynchronous worker claims projection work
-> authoritative collection validator
-> transaction B: persist current validation state
-> normalized fact replacement / durable blocked dependency
-> sorted affected-scope locks
-> aggregate recomputation
-> 0/1 edge notification and serving projection
-> projection job completion
```

Transaction A remains durable even when validation/projection is blocked or retryable. Transaction B atomically commits validation, fact, every affected aggregate, notification transition, serving projection, and job completion. An equal-revision content conflict marks the source uncertain, removes it from eligible projection, and schedules verified repository repair. A dependency block completes durable ingestion and is awakened by dependency change without replaying the Tap frame.

### Verified Repair Flow

```text
administrative repair request/job
-> obtain signed repository snapshot at a known head
-> verify signature, root, MST, repository identity, and source stability
-> install present current records using repository ordering rules
-> mark absent retained records deleted only after complete verification
-> reset/requeue projection work
-> rebuild facts, aggregates, notifications, and serving rows from sources
-> never consult commands and never delete PDS records
```

Extract reusable signed-CAR fixture builders from current repair tests instead of creating a second ad hoc snapshot format.

## 8. Mutation Matrix And Feature Semantics

`pdscommands/matrix.go` and Flutter's contract test should encode the same reviewed matrix. Keep the production implementation typed; use the matrix for conformance, not as a generic runtime mutation engine.

| Mutation | Kind | Identity / CAS | Command Steps | Server Reconciliation | Flutter Overlay Agreement |
|---|---|---|---|---|---|
| Like / unlike | Set | Logical actor + post scope; authoritative repository head | No-op/create one like, or delete all valid matching likes | Complete verified scope active/inactive | Logical active state, never representative URI |
| Repost / unrepost | Set | Logical actor + post scope; authoritative repository head | No-op/create one repost, or delete all valid matches | Complete verified scope active/inactive | Logical active state |
| Follow / unfollow | Set | Logical actor + target DID; authoritative repository head | No-op/create one follow, or delete all valid matches | Complete verified scope active/inactive | Logical active state |
| Block / unblock | Set | Logical actor + target DID; authoritative repository head | No-op/create one block, or delete all valid matches | Complete verified scope active/inactive | Logical active state |
| Instagram suggestion acceptance | Set follow | Same follow scope and head contract; private suggestion reservation remains separate | Same authoritative follow create plan | Follow active/no-op plus existing private completion | Logical follow active state |
| Immediate post create | Append | Frozen generated TID, created time, body, and selected URI | One create | Exact selected URI/CID or canonical controlled content | Selected URI with accepted CID or controlled content |
| Immediate post delete | Addressed delete | Exact URI/rkey and expected CID or stronger head CAS | One delete | Exact URI absent or true conflict | Exact URI absent |
| Business event create | Append | Frozen TID/time/body | One create | Selected URI/CID or controlled content | Selected URI with accepted CID or controlled content |
| Business event update/delete | Addressed | Exact URI/rkey and expected CID/head | One put/delete | Desired controlled content or absence | Exact URI content/absence |
| Business profile put/delete | Fixed key | Lexicon `self`; exact record CID/head | One put/delete; no TID | Canonical controlled fields or absence | Fixed URI controlled fields/absence |
| Personal profile save | Compound fixed key | Bluesky `self` and Craftsky `self`; one captured head and per-record guards | Ordered two-write `applyWrites`; no sequential fallback | Both fixed records agree under one commit | Both controlled logical projections agree |
| Scheduled final publication | Worker-owned append | Existing frozen publication identity and schedule operation identity | One journaled final create | Exact frozen publication result | None; no Flutter controller |

Explicitly excluded: mutes, drafts, saves/pins, schedule CRUD, schedule media staging, blob/video upload, profile customisation/account type, moderation, push tokens, and permanent account deletion.

## 9. Flutter State And Provider Design

### Provider Graph

```text
active AccountSessionLease / account generation
-> pdsRecordOperationControllerProvider
   -> operation registry by scoped UUID
   -> monotonic sequence by logical scope
   -> accepted overlay registry
   -> injectable clock, scheduler, jitter, and UUID source
-> feature mutation provider
   -> immutable endpoint request adapter
   -> feature read invalidator
   -> pure presentation overlay transform
-> ordinary account-scoped read providers
```

Use one non-persistent provider/controller instance for the active account. `accountStateInvalidatorProvider` first advances/clears its generation, then invalidates feature providers. Late futures capture and compare both account generation and logical-scope sequence before any state change.

### Controller Operations

```text
start(spec, immutableInput, scopes)
-> allocate canonical UUID and next sequence for every scope
-> call original feature endpoint
-> accepted: install overlay, invalidate reads, schedule one grace refresh
-> ambiguous: freeze operation and run bounded same-key retry
-> rejected/definite failure: retire operation and expose failure
-> explicitRetry: only for retained ambiguous operation, same input/key
-> reconcile(readSnapshot): apply mutation-class predicate
-> expire: retire overlay, invalidate once, ignore late refresh result
-> resetAccountBoundary: increment generation, cancel timers, clear all state
```

Feature providers stop owning retry loops and overlay deadlines. Existing cache mutation helpers may survive temporarily as pure transforms invoked by the active shared overlay, but they must not own independent rollback, timers, sequence, or reconciliation state and are removed when no longer needed.

### UI States

- Accepted: show intended result through the active overlay; mutation is complete even if the read model is stale.
- Ambiguous: show pending/retry state, freeze editable inputs for that operation, and retain explicit retry after automatic retry stops.
- Failed: reveal the definitive error and allow an intentional changed command to allocate a new key.
- Superseded locally: an older operation may finish, but cannot alter presentation once a newer sequence overlaps its scope.
- Expired overlay: remove optimistic state and refresh ordinary reads once; do not display a fabricated command failure.

No new global mutation screen or operation-status route is added.

## 10. Error And Edge-State Handling

| Condition | Planned Handling | Requirement / Test Evidence |
|---|---|---|
| Missing or malformed `Idempotency-Key` | Reject through standard error envelope before PDS access. | FR-013; AC-017; IT-014 |
| Same scoped key, changed request or generation | `409` idempotency conflict before another write. | FR-014; AC-018; UT-006, IT-006 |
| Terminal replay within 24 hours | Return the stored endpoint-specific response exactly. | FR-015; AC-019; UT-007 |
| Tombstoned key reuse | Reject without reconstructing or dispatching the command. | FR-015, FR-044; AC-019, AC-052 |
| Lost response after dispatch | Reconcile authoritative exact/compound/repository state before redispatch. | FR-017; AC-021; IT-007 through IT-009 |
| Genuine ambiguity | Exact bare `202` contract and bounded retry guidance; no status URL. | FR-016, FR-043; AC-020, AC-051; UT-009, UT-014 |
| Repository changes during paginated set read | Reject mixed snapshot; reread or use verified signed CAR. | FR-022, FR-047; AC-027, AC-055; UT-008, IT-008 |
| Repeated `InvalidSwap` | Three total attempts with exact jitter windows, then `409 pds_repository_conflict`. | FR-023, FR-047; AC-028, AC-055 |
| More than 200 atomic writes | `422 pds_atomic_mutation_too_large`; issue zero writes. | FR-047; AC-055 |
| Unsupported multi-write | Fail safely with no sequential fallback or partial mutation. | FR-021, FR-047; AC-026, AC-055 |
| Existing valid external set record | Stable-head no-op with deterministic representative metadata. | FR-041; AC-049 |
| Invalid apparent set match | Retain on PDS and exclude from removal plan. | FR-025, FR-047; AC-030, AC-055 |
| Equal source revision with different content | Mark uncertain/ineligible and request verified repair. | FR-001, FR-002; AC-006; UT-001 |
| Valid source missing dependency | Persist blocked fact/dependency; wake once dependency changes. | FR-011; AC-015; UT-005, IT-004 |
| Representative source deleted while duplicates remain | Select next representative; preserve logical activity, activation time, and notification state. | FR-009, FR-010, FR-042; AC-013, AC-014, AC-050 |
| Valid record created after set removal commit | Later Tap source reactivates the aggregate normally. | FR-024; AC-029 |
| Command accepted while Tap is stale | Ordinary API remains stale/Tap-only; Flutter overlay bridges locally. | FR-019, FR-031, FR-034; AC-024, AC-035, AC-038 |
| Overlay read disagrees without comparable revision | Keep newest overlay until agreement or 30-second expiry. | FR-048; AC-056; UT-018 |
| Grace/final refresh fails or completes late | Keep overlay until deadline or retire at expiry; never resurrect after expiry/generation change. | FR-031, FR-033, FR-048; AC-035, AC-037, AC-056 |
| Account switch/sign-out/removal/deletion completion | Clear operations, overlays, timers, and sequences; ignore late work. | FR-033, FR-040; AC-037, AC-048; IT-012 |
| Lifecycle changes during command recovery | Recheck fences and deny dispatch; purge cannot recreate owner state. | FR-028; AC-033, AC-042; IT-011 |
| Verified snapshot omits old source | Install deletion evidence and rebuild; Tap omission alone does nothing. | FR-037; AC-045; IT-005 |

Error envelopes continue to use `{error,message,requestId}` with camelCase JSON. Add stable API codes `pds_repository_conflict` and `pds_atomic_mutation_too_large`; map malformed input, auth failures, and unsupported PDS behavior through existing conventions without exposing upstream bodies or credentials.

## 11. TDD Phase Plan

### Phase 1: Final Schema Baseline

1. Add failing IT-013 in `internal/db/unified_pds_record_mutations_migration_test.go` against `testdb.WithMigratedSchema`.
2. Assert exact source/fact/aggregate ownership, command table boundaries, constraints, indexes, and forbidden relationship/effect command paths.
3. Add migration 73 up/down files and owner-purge inventory.
4. Add UT-015 retention-payload assertions as schema/store design pressure.

Gate: IT-013 passes against fresh PostgreSQL migrations; `appview-check` can migrate down-to-zero and reapply.

Requirements: FR-036, FR-044, FR-046. Acceptance: AC-040, AC-052, AC-054. Tests: AT-012, UT-015, IT-013.

### Phase 2: Source Validation, Facts, Aggregates, And Rebuild

1. Add UT-001/UT-002 for repository ordering and every indexed collection/action validator.
2. Add UT-003/UT-004/UT-005/UT-017 for the feature-neutral fact replacement, sorted scope locks, representative selection, edge transitions, and dependency wake-up primitives.
3. Refactor ingestion to install validation state without effect-origin gating.
4. Add real-PostgreSQL IT-001, IT-003 through IT-005, including failpoints and generated signed-CAR rebuild fixtures. Exercise the generic fact/aggregate primitive with test-only fixtures, but do not register non-like set projectors or convert their production reads yet.
5. Prepare an explicit inventory test for every physical follow/block/like/repost consumer. Production registration and serving-query conversion occur only in the owning vertical slice, beginning with like/unlike.

Gate: Clean rebuild with empty command tables equals incremental projection for the foundational fixture; source/fact/aggregate primitives satisfy their invariants; ordinary reads remain projection-only; no remaining set class has been prematurely migrated.

Requirements: BR-001, BR-004, FR-001 through FR-011, FR-019, FR-025, FR-029, FR-034, FR-037, FR-042, FR-046, NFR-001, NFR-002, NFR-006, RULE-001 through RULE-006, RULE-008. Acceptance: AC-001, AC-002, AC-005 through AC-015, AC-023, AC-030, AC-034, AC-038, AC-041, AC-045, AC-050, AC-054. Tests: AT-001 through AT-003, AT-007, AT-011, UT-001 through UT-005, UT-017, IT-001, IT-003 through IT-005, REG-001, REG-002, REG-007, REG-008. This phase proves generic set primitives only; feature production registration remains deferred.

### Phase 3: Command Journal And Protocol Transport

1. Add UT-006, UT-007, UT-015, and UT-016 for canonical fingerprints, state transitions, retention, and golden vectors.
2. Implement command store and lifecycle-coordinated dispatch/recovery with failpoints; prove IT-006.
3. Extend the Indigo and coordinated PDS clients test-first for IT-015.
4. Add authoritative reader/set planner UT-008 and IT-008 with bounded pagination, generated signed CARs, jitter capture, cap, and fallback cases.
5. Add feature-neutral addressed, append, fixed-key, and compound plan primitives plus shared response mapper UT-009; do not register personal-profile endpoints here.
6. Exercise dispatcher recovery through `pdscommands` integration harnesses under IT-006. Defer original-endpoint IT-007 to the like/unlike and later owning slices; defer profile transaction IT-009 and AT-005 to Phase 9.
7. Add bounded telemetry/redaction UT-012 and lifecycle/purge IT-011.

Gate: The command system survives every crash boundary, never blindly redispatches ambiguity, produces exact wire requests/responses, and remains absent from ordinary reads.

Foundation traceability: BR-002, FR-012 through FR-025, FR-027, FR-028, FR-035, FR-038, FR-041, FR-043 through FR-047, NFR-003, NFR-005, RULE-005, RULE-007, RULE-008. The package-level evidence exercises AC-016, AC-018, AC-019, AC-027 through AC-030, AC-033, AC-039, AC-042, AC-044, AC-046, AC-049, and AC-052 through AC-055. Original-endpoint AC-003 and AC-020 through AC-025, plus feature AC-026, AC-031, and AC-032, remain vertical-slice gates. Tests: AT-006, AT-007, UT-006 through UT-009, UT-012, UT-015, UT-016, IT-006, IT-008, IT-011, IT-015, REG-005. UT-009 locks response mapping here; IT-007 proves it through real endpoints beginning in Phase 5.

### Phase 4: Flutter Shared Controller Foundation

1. Add UT-010 for immutable operation state, account generation, and logical-scope sequence reduction.
2. Add UT-014 for exact retry scheduling using injected clock, scheduler, jitter, and elapsed budget.
3. Add UT-011/UT-018 for receipt-based overlay timing and every mutation-class reconciliation predicate.
4. Implement shared API contract and controller providers.
5. Wire reset into `accountStateInvalidatorProvider` and prove IT-012 before migrating a feature.
6. Add the declarative client matrix half of UT-013 with included rows initially marked pending; release conformance rejects pending rows only in the final phase.

Gate: Controller reset/account-switch/overlap/timing tests pass without a feature provider and no operation state is durable.

Requirements: BR-003, FR-013, FR-018, FR-030 through FR-033, FR-039, FR-040, FR-043, FR-048. Acceptance: AC-004, AC-017, AC-022, AC-035 through AC-037, AC-047, AC-048, AC-051, AC-056. Tests: AT-009, AT-010, UT-010, UT-011, UT-013, UT-014, UT-018, IT-012.

### Phase 5: Like/Unlike Reference Slice

1. Route like/unlike through authoritative set commands; stop using projected `FindActiveLike` as mutation authority.
2. Complete interaction source-fact and aggregate projection, read state/counts, and logical notification edges.
3. Convert every like consumer, including engagement counts, viewer state, lists, timeline/profile/search surfaces, notification eligibility/newness, and push delivery, to the logical aggregate.
4. Adapt the feed API/repository and `ToggleLikePost` to the shared Flutter controller.
5. Exercise stable-head create/no-op, accepted, rejected, ambiguous, duplicate external source, remove-all, race, overlay, overlap, account switch, and delayed Tap paths.

Gate: IT-002 is the complete reference-slice proof; the like row in backend and Flutter matrices is no longer pending.

Requirements: FR-005, FR-007 through FR-010, FR-013, FR-016, FR-018, FR-019, FR-022 through FR-024, FR-030 through FR-034, FR-039, FR-041 through FR-043, FR-046 through FR-048. Acceptance: AC-004, AC-011 through AC-014, AC-017, AC-020, AC-022 through AC-024, AC-027 through AC-029, AC-035 through AC-038, AC-047, AC-049 through AC-051, AC-054 through AC-056. Tests: UT-003, UT-004, UT-008, UT-010, UT-011, UT-014, UT-017, UT-018, IT-002, IT-007, IT-008, IT-012, IT-016 plus existing interaction/read/notification/push/provider/API-client suites.

### Phase 6: Remaining Set Actions

Migrate in this internal order: repost, follow, block, Instagram suggestion follow acceptance.

- Reuse exactly the like set planner, aggregate semantics, and Flutter logical-state predicate.
- Remove deterministic-key or projected-row assumptions from normal follow/unfollow.
- Remove relationship mutation-intent lifecycle after block/unblock uses the common journal; retain relationship authorization policy.
- Keep private Instagram suggestion reservation/completion, but delegate its public follow to the common follow command.
- In each sub-slice, convert every physical-table consumer identified in the inventory: repost reads/notifications first, then follow notification/push/Instagram/follower-growth/profile/search consumers, then block relationship/visibility/notification/push consumers.

Gate after each row: backend/client matrix contract, external duplicate create/no-op, remove-all race, aggregation, overlay overlap, and account reset pass; MAN-003 confirms sequence.

Requirements: FR-003, FR-005, FR-007 through FR-010, FR-013, FR-016, FR-022 through FR-025, FR-028 through FR-034, FR-039, FR-041 through FR-048. Acceptance: AC-002, AC-004, AC-007, AC-011 through AC-015, AC-017, AC-020, AC-027 through AC-030, AC-033 through AC-038, AC-047 through AC-056. Tests: UT-002 through UT-005, UT-008, UT-010, UT-011, UT-013, UT-014, UT-017, UT-018, IT-002, IT-008, IT-011, IT-012, IT-015, IT-016, REG-007, REG-008. Add row-local IT-014 fixtures, but the complete matrix test remains a Phase 11 gate.

### Phase 7: Immediate Posts

1. Move immediate create to an append command with frozen TID, timestamp, body, URI, and dispatch fingerprint.
2. Move delete to exact addressed URI/rkey plus expected CID or stronger head guard; retain empty `204`.
3. Adapt create/delete providers to shared overlays and ordinary feed/profile/project invalidation.
4. Keep blob/image upload and direct Bluesky video upload outside the command; only final post record mutation is journaled.

Gate: append lost-response reconciliation, exact delete absence, stale CID conflict, empty `204`, accepted overlay, and Tap-only reads pass.

Requirements: FR-012 through FR-020, FR-028, FR-030 through FR-034, FR-039, FR-043 through FR-045, FR-048. Acceptance: AC-003, AC-004, AC-016 through AC-025, AC-033, AC-035 through AC-038, AC-047, AC-048, AC-051 through AC-053, AC-056. Tests: IT-006, IT-007, IT-014, IT-016, REG-002 through REG-004 plus post/API-client/provider suites.

### Phase 8: Business Records

1. Migrate business-event create to append and update/delete to addressed CID/head-guarded commands.
2. Migrate business-profile put/delete to the fixed `self` key with exact-record reconciliation and no TID.
3. Replace business mutation/overlay ownership with shared controller adapters while preserving business read-provider shapes.
4. Keep account-type/customisation/private workflows outside the contract.

Gate: controlled-content/absence reconciliation, ambiguous retries, overlay expiry, and existing business authorization/neutrality tests pass.

Requirements: FR-013, FR-016, FR-018, FR-020, FR-027, FR-028, FR-030 through FR-034, FR-039, FR-043 through FR-045, FR-048. Acceptance: AC-004, AC-017, AC-020, AC-022, AC-025, AC-032, AC-033, AC-035 through AC-038, AC-047, AC-048, AC-051 through AC-053, AC-056. Tests: row-local IT-014 and IT-016 fixtures plus existing business event/profile API/provider tests; the full IT-014 matrix remains a Phase 11 gate.

### Phase 9: Personal Profile Compound Command

1. Construct Bluesky and Craftsky `self` writes as one ordered two-step plan under one captured `swapCommit`.
2. Fail safely when atomic `applyWrites` is unsupported; never sequence two single-record writes.
3. Reconcile both fixed records and require both controlled projections to agree before retiring Flutter overlay.
4. Preserve semantic profile validation and terminal membership policy in the Tap path.

Gate: one shared commit or no requested partial change, fixed keys/no TID, lost-response reconciliation, and two-projection overlay tests pass.

Requirements: FR-020, FR-021, FR-027 through FR-034, FR-039, FR-043 through FR-045, FR-048. Acceptance: AC-002, AC-004, AC-025, AC-026, AC-032 through AC-038, AC-047, AC-048, AC-051 through AC-053, AC-056. Tests: AT-005, IT-009, row-local IT-014 and IT-016 fixtures, REG-007, and profile effect/API-client/provider suites. Full IT-014 remains a Phase 11 gate; IT-015 protocol transport already passed in Phase 3.

### Phase 10: Scheduled Final Publication

1. Replace `PublicationProcessorOptions.NewEffects` with the combined `GuardedCommandCoordinator` seam. It acquires the lifecycle/session boundary once and exposes the existing durable blob-effect capability plus `AlreadyFencedCommandExecutor` over the same active PDS client.
2. Keep `AcquirePublishingEffect` inside that one guarded callback so its lease/session advisory lock still spans private-media verification/upload, command execution, and `FinalizePublication`; the command executor must not recursively enter `WithGuardedEffects` or reacquire owner/session locks.
3. Adapt only the final public record create to the command store, using the existing frozen publication identity as immutable command identity and preserving response recovery.
4. Preserve the existing schedule record, frozen publication version, leases, media readiness, blob effect attempts, and crash recovery.
5. Use no Flutter controller or overlay and do not migrate schedule CRUD/media staging.

Gate: IT-010 and REG-006 pass under restart and lost-response fixtures.

Requirements: FR-026. Acceptance: AC-031. Tests: AT-008, IT-010, REG-004, REG-006.

### Phase 11: Final Wiring, Legacy Removal, And Release Evidence

1. Complete `pdscommands` wiring in `internal/app/deps.go` and narrow route dependencies.
2. Mark every included mutation-matrix row complete and run table-driven IT-014/IT-016.
3. Remove public callers of `pdseffects.ExecutorFactory`, effect-origin eligibility, relationship intents, winner/duplicate replacement, feature-owned overlays, and obsolete cache mutation.
4. Remove legacy files or narrow them to genuinely excluded private behavior only after no-caller proof.
5. Run architecture conformance UT-019/IT-017/REG-009 and full release gates.

Gate: AC-057 and MAN-003 pass; no included row is pending and no superseded mechanism has a migrated caller.

Requirements: FR-049, NFR-004. Acceptance: AC-007, AC-043, AC-057. Tests: AT-013, AT-014, UT-013, UT-019, IT-014, IT-016, IT-017, REG-009, MAN-002, MAN-003.

## 12. Test Implementation Commands

Focused backend commands evolve with the slices:

```text
cd appview && TEST_DATABASE_REQUIRED=false go test ./internal/pdscommands ./internal/auth ./internal/api ./internal/index ./internal/ingestion ./internal/ownerlifecycle ./internal/accountdeletion ./internal/scheduledposts
```

Database integration tests use the repository-standard compose PostgreSQL and `testdb.WithSchema` or `testdb.WithMigratedSchema`; they must not silently skip in release evidence.

Focused Flutter commands:

```text
cd app && flutter test test/shared/mutations test/shared/api
cd app && flutter test test/feed test/profile test/business test/instagram_migration
```

Generation after annotated provider changes:

```text
cd app && dart run build_runner build --delete-conflicting-outputs
```

Required final release evidence:

```text
just test
just appview-check
just app-test
```

Recommended separate evidence:

```text
just app-analyze
```

Do not report `just app-analyze` as part of AC-043. No local PDS is required; protocol tests use scripted HTTP transport and programmatically generated signed-CAR fixtures.

## 13. Sequencing And Guardrails

- First red test: IT-013 in `appview/internal/db/unified_pds_record_mutations_migration_test.go`.
- Phase guardrail: Foundations must pass before like/unlike; each later slice follows FR-049 and records MAN-003 evidence.
- Authority guardrail: Never use Tap lag, a command row, or projected state as authoritative PDS state for mutation planning.
- Read guardrail: Never merge command results into AppView ordinary GET responses.
- Validation guardrail: Ingestion and authoritative command reads use the same authoritative per-collection semantic validators.
- Storage guardrail: Raw source content and versioned validation status live only in `tap_source_records`; command records retain only reconciliation inputs.
- Transaction guardrail: Transaction A atomically installs source evidence and enqueues projection work before Tap acknowledgement. A later worker transaction B atomically commits validation state, fact replacement, all affected aggregates, logical notifications, serving changes, and projection-job completion.
- Lock guardrail: Aggregate scope locks are sorted; owner lifecycle locks remain outside remote command dispatch; command/source transactions do not cross-lock.
- Retry guardrail: An ambiguous remote call is reconciled before redispatch. Only explicit `InvalidSwap` enters the three-attempt set loop.
- Atomicity guardrail: Never split a multi-record plan into sequential writes. Enforce the 200-step cap before dispatch.
- Destruction guardrail: Only explicit owner commands and the separately authorized permanent-deletion job remove PDS records.
- Lifecycle guardrail: Recheck owner/target fences during recovery, not only initial request handling.
- Privacy guardrail: Never log raw operation keys, request/record bodies, OAuth tokens, DPoP material, service JWTs, PDS credentials, or unbounded identifiers as metric labels.
- Flutter guardrail: All keyed feature providers delegate operation/retry/overlay ownership to the shared controller; account invalidation clears it before reads reset.
- Overlay guardrail: Receipt of acceptance, not server timestamp or Tap observation, starts the two-second and 30-second clocks.
- Scheduled guardrail: Final publication uses the journal, but private scheduling and media state stay in `scheduledposts` and no Flutter state is created.
- Compatibility guardrail: Preserve existing endpoint paths, normal success payloads, empty DELETE `204`, auth behavior, error envelope casing, and ordinary read response shapes.
- Generated-file guardrail: Edit annotated Dart sources and regenerate. No Lexicon-generated Go files should change because no Lexicon change is planned.
- Removal guardrail: Do not delete old mechanisms until the matching slice and final no-caller conformance pass.

## 14. Risks And Resolutions

| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Resolved | Current source persistence mixes effect-origin reconciliation with repository evidence. | External valid records can be gated by local command history. | Migration 73 and ingestion refactor make validation/eligibility source-state properties; final conformance removes effect-origin callers. |
| CPQ-002 | Resolved | Existing set tables and handlers assume one winner or deterministic Craftsky record. | Duplicate external records can be lost or incompletely removed. | Generic per-URI facts plus logical aggregates become serving authority; explicit removal uses a verified head-bound snapshot. |
| CPQ-003 | Resolved | Current PDS client lacks repository-head and atomic-write operations. | Safe complete set removal and profile atomicity are impossible. | Extend the narrow client and coordinated wrapper test-first; fail closed on unsupported multi-write. |
| CPQ-004 | Resolved | A remote write can succeed between durable dispatch and local completion. | Blind retry can duplicate append records or overwrite newer state. | Retain exact dispatch plans and reconcile authoritative state before any redispatch. |
| CPQ-005 | Resolved | Existing Flutter providers have independent optimistic state and rollback. | Late reads/responses can overwrite newer actions or cross accounts. | One account-generation and logical-scope-sequenced controller owns operation and overlay lifetime. |
| CPQ-006 | Residual | Scripted protocol tests cannot prove every third-party PDS implementation supports standard `applyWrites` identically. | Some PDSes may reject compound or large set mutations. | Verify exact Indigo wire format, classify unsupported behavior explicitly, permit only contract-equivalent one-write fallback, and document interoperability as residual risk. |
| CPQ-007 | Residual | Migration and serving-query conversion touch multiple mature read paths. | A physical-source count or representative lookup could survive unnoticed. | Add schema/query architecture tests, ordinary-read regression tests, mutation matrix coverage, and final static no-caller conformance. |
| CPQ-008 | Resolved | The repository mentions sqlc but has no active query generation. | Introducing sqlc would add unrelated infrastructure and complicate transactions. | Keep new SQL scoped to `ingestion`, `index`, and `pdscommands` stores using current inline-`pgx` conventions. |
| CPQ-009 | Residual | Exact user-visible pending/failed copy may vary by feature. | UI wording can change without changing state semantics. | Keep controller states and tests semantic; use existing feature error presentation and review copy during each UI slice. |

No blocking coding-plan questions remain. Local choices such as small file splits, internal transaction interfaces, and pure Flutter adapter types may be refined during TDD if they preserve the fixed contracts above.

## 15. Handoff To TDD Builder

- Coding plan: `docs/changes/2026-09-17-unified-pds-record-mutations/04-coding-plan.md`
- TDD execution plan: `docs/changes/2026-09-17-unified-pds-record-mutations/05-implementation-plan.md`
- Start with IT-013 against the real migrated PostgreSQL schema before writing migration 73.
- Then implement source/fact/aggregate and command foundations before touching like/unlike production registration.
- Preserve requirement, acceptance, and test IDs in `05-implementation-plan.md` and phase evidence.
- Explicit user approval is required before implementation starts.
- Do not change Lexicons, expose PDS credentials to Flutter, add a mutation-status API, persist ordinary Flutter mutation state, or make command history a public projection input.
