# Coding Plan: PDS Migration And Handle Change Resilience

## 1. Inputs

- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` (`Approved`, high risk)
- Architecture references: `AGENTS.md`, `atproto-craft-social-app-reference.md`, `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md`
- Approved approach: lean hybrid with Tap-first ingestion, uncached DID-to-PDS authority checks, bounded effect-only OAuth metadata caching, fully fresh OAuth callbacks, same-DID reauthorization, and a narrow verified repository repair sweep
- Lexicon impact: none expected. Stop and run the lexicon/ADR workflow if implementation discovers a record-shape change.

## 2. Implementation Strategy

Keep DID as the existing account, ownership, and persistence boundary. Add one authority-verification abstraction to the shared OAuth session coordinator and ordinary OAuth callback path; do not add checks to individual handlers. DID resolution remains uncached. Ordinary effect verification may reuse successfully validated OAuth metadata from a bounded five-minute origin-keyed cache, while OAuth start, registration, and callback verification remain fully fresh. A conclusively obsolete parent is fenced by the existing row-version, lifecycle-generation, and auth-epoch terminalization path. Resolution, metadata, destination-policy, timeout, and generic upstream failures remain retryable and send no credential-bearing request.

Keep Tap as the normal create/update and backfill source. Reuse `tap_repository_jobs` for durable `tap_add_repo` and `pds_reconcile` work, but replace the current source-by-source reconcile behavior with complete `com.atproto.sync.getRepo` acquisition. Trust a snapshot only after bounded download, clean EOF, commit/root/DID/revision/signature verification against a freshly resolved signing key, MST verification, and a post-fetch authority check. Compare only collections obtained from the same `index.TransactionalDispatcher` registry used by normal ingestion, then submit deterministic create/update/delete source events through the existing durable ingestion and projector path. No inferred delete is constructed before verification succeeds.

Make Flutter identity contracts DID-first at the session, Riverpod family, route, cache, navigation, ownership, recent-search, mention, search-merge, and deletion boundaries. Handles remain mutable presentation metadata or explicit external alias input. Reuse the current lease-scoped `401` invalidation and sign-in flow for `pds_session_expired`; do not add migration state or global write gating.

Make the smallest direct contract replacements because there are no production compatibility obligations. The only database migration is for data that cannot be represented safely in current tables: authorization-time resource authority, generalized unverified-credential cleanup if needed by ordinary callbacks, and DID-bound deletion confirmation. No persisted authority cache, migration state, signing-key fingerprint, snapshot-progress table, or lexicon migration is planned. The effect-only metadata cache is process-local and lazy-expiring.

## 3. Affected Areas

| Area | Existing Pattern | Planned Change | Requirement IDs | Test IDs |
|---|---|---|---|---|
| OAuth effect authority | `OAuthSessionCoordinator.withFencedSession` validates stored endpoint safety before creating an Indigo session | Inject one uncached verifier; verify current PDS and issuer before any token, proof, refresh, revocation, or effect use; classify current, proven stale, and retryable/unverified outcomes | FR-001, FR-002, FR-003, FR-004, FR-005, NFR-001, NFR-002, RULE-002, RULE-003 | AT-004, AT-005, UT-001, UT-003, UT-004, IT-001, IT-002, IT-004, IT-005, IT-015 |
| OAuth callback race | Login request persists owner authority but not authorization-time PDS origin; callback can combine old credentials with newly resolved host | Persist original resource-server origin and issuer; verify both uncached after exchange and before session persistence; queue bounded cleanup only against original issuer on mismatch | FR-003, FR-031, NFR-002, NFR-005 | AT-009, UT-002, IT-016, IT-017 |
| Parent selection and cleanup | Background selection excludes terminal parents; terminalization and revocation cleanup already exist | Route proven-stale foreground/background parents through exact-parent CAS terminalization; preserve independent parents; keep cleanup asynchronous and original-issuer-only | FR-002, FR-006, FR-028, NFR-001 | AT-001, AT-004, UT-003, IT-002, IT-003, IT-004 |
| Durable repository work | `tap_repository_jobs` coalesces `tap_add_repo` and `pds_reconcile`; profile initialization also calls Tap directly best-effort | Add transaction-aware enqueue; persist tracking for onboarding, sign-in, and reauthorization; enqueue repair for same-DID reauthorization and existing source-order uncertainty | FR-008, FR-010, NFR-001 | AT-001, IT-007 |
| Repository repair | `reconcileTapRepository` fetches only known ambiguous records individually | Fully acquire and verify one authoritative CAR, compare every registered collection, and feed idempotent repairs through existing ingestion/indexers | FR-009, FR-011, FR-027, NFR-001, NFR-003 | AT-001, AT-010, UT-006, IT-006, IT-008, IT-009, IT-010, IT-011 |
| Indexer registry | `TransactionalDispatcher.handlers` is the effective collection registry | Expose a sorted immutable `Collections()` snapshot and use it for both registration coverage and repository comparison | FR-009 | UT-006, IT-006 |
| Tap identity and replay | Non-`deleted` identity events enqueue refresh; `deleted` terminalizes owner; Compose starts Tap at live head | Treat every accepted identity/account status as a refresh/sync hint only; remove terminal participant wiring; remove `TAP_NO_REPLAY=true` | FR-007, FR-029, FR-030, RULE-006 | AT-011, UT-005, IT-014, REG-006, REG-007 |
| Identity cache | Version-fenced refresh exists, but unconditional authoritative upserts and invalid-handle retry preserve stale aliases | Route authoritative writes through the version-fenced path; commit `handle.invalid`; atomically release old aliases; invalidate DID and old/new local keys after commit | FR-019, FR-020, FR-021, NFR-001 | AT-006, UT-012, IT-012 |
| Deletion identity | Intent and Flutter pending state bind to a handle | Return, persist, display, hash, and require exact full owner DID; remove handle resolution from intent creation | FR-017, RULE-005 | AT-007, UT-013, IT-013 |
| Flutter session refresh | `/v1/whoami` DID is checked but returned handle is discarded | Add a lease-fenced handle-only registry mutation and persist before publishing successful validation | FR-012, FR-023 | AT-002, AT-012, UT-007, IT-015 |
| Flutter profile state | `userProfileProvider` is keyed by handle-or-DID; active identity loads by stored handle; caches are dual-published | Make profile/provider/post cache families DID-keyed; use `fetchMe()` for active DID; remove dual handle cache publication | FR-013, FR-014, RULE-001, RULE-007 | AT-002, AT-003, UT-008, UT-018, REG-004 |
| Flutter routes and navigation | Canonical internal route is `/profile/:handle`; known-DID call sites pass handles | Replace canonical profile path and route field with DID; resolve explicit handle aliases once and replace route with DID; update all inventoried call sites | FR-014, FR-016, FR-021, RULE-005, RULE-007 | AT-003, UT-008, UT-009, UT-018, REG-004 |
| Flutter ownership and search | Profile ownership and some actions use handle; profile pagination deduplicates by DID and handle | Compare loaded/viewer DIDs; pass DID to actions/tabs; deduplicate profiles only by DID | FR-015, FR-018, RULE-005 | AT-003, UT-010, UT-011 |
| Unavailable-handle presentation | Non-null handle strings render directly across identity surfaces | Add one sentinel-aware presentation/alias policy and localized `Handle unavailable` copy; preserve immutable authored mention text | FR-020 | AT-006, UT-014, MAN-001 |
| Scheduled posts | Existing retries and 30-minute late cutoff | Reuse unchanged lifecycle; prove exactly 30 minutes is the final eligible attempt and stale parents cannot be selected | FR-025 | AT-008, UT-015, IT-003, REG-005 |
| Observability | Existing `Observer`, structured `slog`, and worker logs expose bounded labels | Add authority/repair/tracking outcome metrics and safe reason codes; expose backlog age/attempt signals and no secret-bearing fields | NFR-003, NFR-005 | UT-016, IT-017, MAN-002, REG-008 |
| Effect-time OAuth metadata cache | Indigo's resolver fetches protected-resource and authorization-server metadata on every call and provides no cache | Add a bounded fixed-expiry origin-keyed cache with same-key miss coalescing; inject it only into ordinary session operations and retain a fresh verifier for OAuth flows | FR-032, NFR-001, NFR-002, NFR-005 | UT-019, UT-020, IT-018, REG-009 |

## 4. Files And Modules

### AppView: Authority And OAuth

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `appview/internal/auth/oauth_authority.go` | Create | Define current-authority value, verifier interface, proven-stale error/result, equality rules, and secret-free reason categories shared by callback and session coordinator | FR-001, FR-003, FR-004, FR-031, NFR-002 | UT-001, UT-002, AT-005 |
| `appview/internal/app/federated_authority.go` | Create | Implement uncached DID document, PDS endpoint, authorization-server URL, and metadata resolution through the hardened federated boundary | FR-001, FR-031, NFR-002 | IT-001, IT-005, IT-016 |
| `appview/internal/auth/session_coordinator.go` | Change | Verify authority after loading/fencing the exact parent and before constructing/using `oauth.ClientSession`; terminalize only conclusive stale parent | FR-001, FR-002, FR-003, FR-004, FR-005 | UT-001, UT-003, IT-001, IT-002, IT-005 |
| `appview/internal/auth/session_coordinator_test.go` | Change | First red tests and table-driven current/stale/transient/policy/exact-parent concurrency coverage | FR-001, FR-002, FR-004, NFR-001, NFR-002 | UT-001, UT-003, IT-002, IT-005 |
| `appview/internal/auth/oauth_flow.go` | Change | Save authorization-time resource origin; verify exchanged callback authority before constructing/persisting mixed session; retain original cleanup target | FR-003, FR-031 | UT-002, IT-016 |
| `appview/internal/auth/account_deletion_reauth.go` | Change | Extend `AuthRequestMetadata` and context constructors with typed resource-server origin for non-registration flows | FR-031, NFR-002 | UT-002, IT-016 |
| `appview/internal/auth/store.go` | Change | Persist/load original resource origin and support durable cleanup of ordinary callback credentials without weakening request-state/expiry fences | FR-003, FR-031, NFR-001, NFR-005 | UT-002, IT-004, IT-016 |
| `appview/internal/auth/session_cleanup_processor.go` | Change | Generalize existing bounded unverified-credential cleanup to ordinary callback mismatch while targeting stored original issuer only | FR-003, FR-028, FR-031 | IT-004, IT-016 |
| `appview/internal/auth/oauth_test.go` | Change | Add callback race matrix and same-DID reauthorization repair/tracking enqueue assertions | FR-006, FR-008, FR-031 | UT-002, IT-007, IT-016 |
| `appview/internal/auth/session_cleanup_processor_test.go` | Change | Capture cleanup destination and prove no old credential reaches new issuer/PDS | FR-003, FR-028 | IT-004, IT-016 |
| `appview/internal/api/pds_migration_auth_acceptance_test.go` | Create | API-level stale-parent fencing and exact `401` camelCase envelope | FR-002, FR-005 | AT-004, IT-002 |
| `appview/internal/app/federated_real_flow_integration_test.go` | Create | Exercise a protected effect through HTTP fakes with current, stale, unsafe, and unavailable authority | FR-001, FR-003, NFR-002, RULE-003 | IT-001 |

### AppView: Durable Tracking And Verified Repair

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `appview/internal/ingestion/repository_jobs.go` | Change | Add `EnqueueRepositoryJobTx`; retain `(did, kind)` coalescing, lease CAS, and exponential backoff; use bounded reason codes from handlers | FR-008, FR-010, NFR-001, NFR-003 | IT-006, IT-007, IT-008 |
| `appview/internal/ingestion/repository_snapshot.go` | Create | Define verified snapshot/record contracts and ensure absence comparison is unavailable until verification marker exists | FR-009, NFR-003 | UT-006, IT-008 |
| `appview/internal/ingestion/repository_repair.go` | Create | Compare verified snapshot against all DID-owned indexed source rows for registered collections and enqueue deterministic idempotent source changes in batches | FR-009, FR-027, NFR-001, NFR-003 | UT-006, IT-006, IT-008, IT-010 |
| `appview/internal/ingestion/repository_repair_test.go` | Create | Unit-test shared registry filtering and the no-delete-before-verification invariant | FR-009, NFR-003 | UT-006, AT-010 |
| `appview/internal/ingestion/repository_repair_integration_test.go` | Create | PostgreSQL create/update/delete convergence, interruption, retry, duplicate jobs, and every trust-boundary failure | BR-001, FR-009, NFR-001, NFR-003 | IT-006, IT-008 |
| `appview/internal/ingestion/repository_repair_profile_integration_test.go` | Create | Apply existing departure policy only for verified profile absence and never terminalize DID | FR-027, RULE-006 | IT-010 |
| `appview/internal/app/repository_snapshot_fetcher.go` | Create | Bounded `getRepo` download plus Indigo CAR/commit/signature/root/MST verification; repeat authority resolution after acquisition | FR-009, FR-011, NFR-002, NFR-003 | AT-010, IT-008, IT-011 |
| `appview/internal/app/tap_repository.go` | Change | Dispatch `tap_add_repo` as today; replace per-record reconcile with snapshot fetch plus repair service; retain 5-second lease finalization margin | FR-008, FR-009, FR-010 | AT-001, IT-006, IT-007 |
| `appview/internal/index/transactional_dispatcher.go` | Change | Add deterministic sorted `Collections() []syntax.NSID`; keep `Register` as the single registry mutation point | FR-009 | UT-006 |
| `appview/internal/app/indexer_wiring_test.go` | Change | Assert all registered collections are exported once and match Tap/indexer wiring | FR-009 | UT-006 |
| `appview/internal/app/deps.go` | Change | Construct one shared ingestion store before auth/Tap wiring and inject authority verifier, transaction enqueue hooks, snapshot fetcher, repair service, and observer | FR-001, FR-008, FR-010, FR-031 | IT-001, IT-007, IT-016 |
| `appview/internal/app/deps_tap.go` | Change | Accept shared ingestion store and dispatcher; remove terminal-identity participant; wire verified repair worker | FR-007, FR-008, FR-009, FR-029 | IT-006, IT-014 |
| `appview/internal/auth/initialize_profile.go` | Change | Replace warning-only direct `AddRepo` with transactionally durable enqueue participation | FR-010, NFR-001 | IT-007 |
| `appview/internal/ingestion/repository_jobs_integration_test.go` | Create | Table-drive onboarding, ordinary sign-in, and reauthorization; fail remote Tap, restart worker, recover, and coalesce duplicates | FR-008, FR-010, NFR-001 | IT-007 |
| `appview/internal/app/pds_migration_acceptance_test.go` | Create | End-to-end same-DID migration with HTTP fakes, signed CAR, ownership fixtures, reauthorization, repair, and write/read direction | BR-001, FR-006, FR-008, FR-009, FR-011, RULE-001, RULE-003 | AT-001 |
| `appview/internal/app/pds_migration_ownership_integration_test.go` | Create | Seed representative DID-owned private/public state and prove no reassignment or migration-only deletion | BR-001, FR-004, FR-006, RULE-006 | IT-009 |
| `appview/internal/tap/pds_migration_contract_test.go` | Create | Pin HTTP-fake same/changed-handle, PDS/signing-key rotation, missed-event, reset-chain, and complete-snapshot cases | FR-011 | IT-011 |

### AppView: Identity, Lifecycle, API, Scheduling, And Operations

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `appview/internal/ingestion/service.go` | Change | Send ordinary, deleted, deactivated, suspended, and takendown identity events through one idempotent refresh-hint path; remove Tap terminalization | FR-007, FR-029, RULE-006 | UT-005, IT-014 |
| `appview/internal/ingestion/identity_event_policy.go` | Create | Pure accepted-status classification used before persistence; use pinned Tap wire value `deactivated` rather than inventing `inactive` | FR-007, FR-029 | UT-005 |
| `appview/internal/ingestion/identity_event_policy_test.go` | Create | Table-test equal/changed handle and all real status variants without lifecycle/OAuth decisions | FR-007, FR-029 | UT-005 |
| `appview/internal/ingestion/identity_refresh_trigger_integration_test.go` | Change | Add restart, duplicate, and out-of-order identity status persistence with no terminalization | FR-007, FR-029, NFR-001 | IT-014 |
| `appview/internal/ingestion/lifecycle_integration_test.go` | Change | Replace Tap-deleted terminal expectations with hint-only lifecycle preservation | FR-029, RULE-006 | IT-014, REG-007 |
| `appview/internal/api/identity_cache_store.go` | Change | Make version-fenced refresh the sole authoritative path; atomically release conflicting/old aliases; do not expose sentinel as alias input | FR-019, FR-020, FR-021 | UT-012, IT-012 |
| `appview/internal/api/identity_cache_refresh.go` | Change | Treat `syntax.HandleInvalid` as a successful current result; invalidate DID and prior/new handles only after commit | FR-019, FR-020, NFR-001 | UT-012, IT-012 |
| `appview/internal/api/identity_cache_refresh_test.go` | Change | Race old/new/sentinel completions and assert newest version wins plus post-commit invalidation set | FR-019, FR-020, FR-021 | UT-012, IT-012 |
| `appview/internal/api/identity_cache_store_test.go` | Change | Remove unconditional authoritative-upsert expectations and test alias reassignment/removal | FR-019, FR-020, FR-021 | UT-012, IT-012 |
| `appview/internal/api/identity.go` and authoritative resolver adapter | Change | Return `handle.invalid` successfully for DID-to-handle resolution but reject sentinel as handle input; retain bidirectional alias proof | FR-020, FR-021 | AT-006, IT-012 |
| `appview/internal/accountdeletion/app_service.go` | Change | Bind intent to owner DID, remove authoritative handle lookup/cache write, and return full confirmation DID | FR-017 | IT-013 |
| `appview/internal/accountdeletion/store.go` | Change | Hash/verify exact DID confirmation under existing lifecycle and reauthentication fences | FR-017, NFR-005 | IT-013, IT-017 |
| `appview/internal/api/account_deletion.go` | Change | Replace camelCase `confirmationHandle` with `confirmationDid`; return server-bound full DID and DID mismatch code | FR-017 | IT-013 |
| `appview/internal/api/account_deletion_identity_test.go` | Create | Test valid/stale/invalid handle states all produce the same DID confirmation contract | FR-017 | IT-013 |
| `appview/internal/scheduledposts/retry_test.go` and `failure_acceptance_test.go` | Change | Lock 29:59, exactly 30:00 success/failure, and after-30:00 outcomes with controllable clock and authority-aware selection | FR-025 | UT-015, IT-003, AT-008 |
| `appview/internal/observability/observer.go`, metric/log adapters | Change | Record bounded authority check, stale parent, reauth response, Tap tracking, repair result/backlog age/attempt count labels | NFR-003, NFR-005 | IT-017, MAN-002 |
| `appview/internal/observability/pds_migration_test.go` | Create | Inject canary secrets across stale/transient/repair/cleanup outcomes and scan all emitted telemetry | NFR-005 | UT-016, IT-017 |
| `appview/internal/app/config.go` and config tests | Change | Add/validate repository snapshot size, acquisition timeout, projection batch, and alert thresholds; align repository lease default | FR-009, NFR-002, NFR-003 | IT-008, IT-017 |
| `appview/environments/*.env` and `.env.example` files | Change | Document bounded defaults without secrets | FR-009, NFR-003 | REG-008 |
| `docker-compose.yml` | Change | Remove `TAP_NO_REPLAY=true`; keep exact Tap `0.1.10` image pin | FR-011, FR-030 | AT-011, REG-006 |
| `appview/migrations/000066_pds_migration_identity.up.sql` / `.down.sql` | Create | Persist OAuth request resource origin, generalize safe unverified callback cleanup constraints if required, and rename deletion confirmation hash to DID | FR-017, FR-031, NFR-001 | IT-013, IT-016 |

### Flutter

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `app/lib/auth/models/session_registry.dart` | Change | Add lease-fenced handle-only update; bump strict secure-storage schema for DID deletion shape; preserve token/generation/routing/cached fields | FR-012, FR-017 | UT-007, UT-013 |
| `app/lib/auth/providers/session_registry_provider.dart` | Change | Serialize and persist handle reconciliation before publishing state | FR-012, NFR-001 | UT-007 |
| `app/lib/auth/services/session_validation_coordinator.dart` | Change | On matching DID, commit whoami handle through captured lease; leave cold-start validation non-gating | FR-012, FR-023 | AT-002, AT-012, UT-007 |
| `app/test/auth/services/session_validation_coordinator_test.dart` | Create | Prove only handle changes and stale lease/DID mismatch cannot overwrite another account generation | FR-012 | UT-007 |
| `app/lib/auth/providers/active_account_identity_provider.dart` | Change | Target active DID and consume the DID-keyed own-profile provider rather than stored handle | FR-013 | AT-002, UT-008 |
| `app/test/auth/providers/active_account_identity_provider_test.dart` | Create | Use fake repository `onFetchMe`/`onFetch` to prove stale handle is never own-profile identity | FR-013 | UT-008 |
| `app/lib/profile/providers/user_profile_provider.dart` and generated `.g.dart` | Change / Regenerate | Change family key to typed `Did`; use `fetchMe()` when key equals active account DID, otherwise fetch canonical DID | FR-013, FR-014, FR-021 | UT-008, REG-004 |
| `app/lib/profile/providers/profile_cache_publication.dart` and dependent business/feed providers | Change | Publish/invalidate profile and authored-content caches by DID only | FR-014, RULE-007 | UT-008, UT-018 |
| `app/lib/router/router.dart`, `route_locations.dart`, generated router files | Change / Regenerate | Canonical known-profile route uses `did`; explicit `@handle` alias route resolves and replaces with DID; own `/profile` remains self route | FR-014, FR-021, RULE-007 | AT-003, UT-008, UT-018, REG-004 |
| `app/lib/profile/widgets/profile_card_modal.dart`, `profile_route_presentation.dart`, `profile_presentation_page.dart` | Change | Accept typed DID for known identity and key provider/cache state by DID | FR-014, FR-015 | AT-003, UT-008, UT-010 |
| `app/lib/profile/pages/profile_page.dart`, `edit_profile_dialog.dart` | Change | Load by DID/`me`; compare loaded profile DID to viewer DID; pass DID to follow/report/mute/block/tabs | FR-013, FR-014, FR-015 | UT-008, UT-010 |
| `app/lib/feed/widgets/post_card.dart`, notification/search/follow/relationship/Instagram-suggestion navigation call sites | Change | Pass already-available model DID into canonical route and profile cache; retain handles only for display | FR-014, RULE-005, RULE-007 | AT-003, UT-008, UT-018 |
| `app/lib/shared/rich_text/facet_action_handler.dart` | Change | Navigate from `MentionFacetFeature.did`; never parse visible historical text as destination | FR-016 | UT-009, AT-003 |
| `app/test/shared/rich_text/faceted_text_actions_test.dart` | Change | Reverse old handle-based expectation: visible text stays old, destination is facet DID | FR-016 | UT-009 |
| `app/lib/search/pages/blank_search_view.dart`, `search_page.dart`, `search_results_tabs.dart` | Change | Reopen recent/known profiles by stored/model DID | FR-014 | UT-008, UT-009 |
| `app/lib/search/providers/search_pagination.dart` | Change | Remove handle deduplication and merge profile pages by DID only | FR-018 | UT-011 |
| `app/test/search/providers/search_pagination_merge_test.dart` | Change | Cover same DID/new handle, same handle/different DID, and adjacent-page duplicate DID | FR-018 | UT-011 |
| `app/lib/profile/models/profile_handle.dart` | Create | Centralize sentinel recognition, current-handle presentation, and alias eligibility | FR-020 | UT-014 |
| Current identity widgets/models across profile, auth, settings, feed, notifications, search, and account lists | Change | Render localized unavailable copy instead of `handle.invalid` or stale handle; leave authored text untouched | FR-020 | AT-006, UT-014, MAN-001 |
| `app/lib/auth/models/account_deletion.dart`, `pending_account_deletion.dart` | Change | Capture lease without local handle; persist exact server-returned DID after equality check | FR-017 | UT-013, AT-007 |
| `app/lib/auth/data/account_deletion_repository.dart` | Change | Decode intent `confirmationDid`; submit camelCase `confirmationDid` | FR-017 | UT-013 |
| `app/lib/settings/providers/account_deletion_controller.dart`, acceptance coordinator, confirmation model/pages | Change | Display and exact-match full DID through reauthorization and acceptance | FR-017 | UT-013, AT-007 |
| `app/lib/l10n/app_en.arb` and generated localization files | Change / Regenerate | Add handle-unavailable and DID-specific deletion copy | FR-017, FR-020 | UT-013, UT-014, MAN-001 |
| `app/test/auth/handle_change_acceptance_test.dart` | Create | Cold-start stale-handle acceptance scenario with preserved session fields and `/me` loading | FR-012, FR-013 | AT-002 |
| `app/test/profile/did_first_identity_acceptance_test.dart` | Create | Cross-surface handle reassignment, DID route/cache, mention, and own-profile scenario | FR-014, FR-015, FR-016, FR-021 | AT-003 |
| `app/test/profile/invalid_handle_acceptance_test.dart` | Create | Verify sentinel wire decoding and unavailable presentation without stale alias | FR-020 | AT-006 |
| `app/test/settings/account_deletion_did_acceptance_test.dart` | Create | Verify exact full-DID flow for all handle states | FR-017 | AT-007 |
| `app/test/auth/cold_start_write_behavior_test.dart` | Create | Prove pending validation adds no global write gate and standard `401` recovery remains actionable | FR-005, FR-023 | AT-012, IT-015 |
| `app/test/router/router_usage_test.dart` and focused feature tests | Change | Maintain source inventory for every known identity surface and require DID route/provider use | RULE-007, NFR-006 | UT-018, REG-004 |
| `app/test/observability/secret_scan_test.dart`, secure storage/interceptor tests | Create / Change | Prove Flutter represents and transmits only Craftsky credentials and no secret leaks | NFR-005, RULE-004 | UT-016, UT-017, REG-008 |

## 5. Services, Interfaces, And Data Flow

### OAuth Authority Contract

The auth package owns the policy and coordinator contract; the app package owns the concrete network adapter so all lookups use the existing hardened federated clients. Normalize URL origins before equality checks. Compare the current PDS origin and authorization-server issuer to the values stored in the OAuth parent/request; do not compare handles. A verifier error means authority is unverified/retryable. A successful result with unequal origins is the only migration-specific proof of staleness.

```text
type OAuthAuthority struct {
    DID          syntax.DID
    PDSOrigin    url.URL
    IssuerOrigin url.URL
}

type OAuthAuthorityVerifier interface {
    ResolveCurrent(ctx context.Context, did syntax.DID) (OAuthAuthority, error)
}

verifyExpected(current, expected OAuthAuthority) ->
    current | proven-stale(expected/current origins only) | retryable error
```

Protected effect flow:

```text
owner/session fence
  -> load exact parent + row version
  -> validate stored endpoints through outbound policy
  -> uncached ResolveCurrent(parent.AccountDID)
  -> transient/unverified: return retryable error, send no credential
  -> proven stale: CAS-terminalize exact parent and children
       -> enqueue existing original-issuer revocation cleanup
       -> return ErrPDSSessionExpired
  -> current: construct Indigo ClientSession
       -> execute existing bounded effect
       -> persist token rotation under current row version
```

The same verifier is called for foreground and background effects because both reach `OAuthSessionCoordinator`. Deletion OAuth continues to use its stricter deletion authority and should call the same current-authority verifier before its credential-bearing operation, mapping staleness to `ErrDeletionReauthenticationRequired` as well as the ordinary expired-session error.

Callback flow:

```text
StartLogin:
  resolve handle -> DID -> PDS A -> issuer A
  persist owner fences + resource origin A + issuer A with auth request

CompleteCallback:
  exchange code only with issuer A
  construct unverified credential with PDS A/issuer A
  uncached resolve callback subject DID
  current A/A -> persist parent and finalize handoff
  current B/B -> persist no parent/child; queue credential cleanup to issuer A; reject
  lookup/metadata/policy failure -> persist no mixed session; retain bounded retry/cleanup state
```

### Durable Trigger Contract

`EnqueueRepositoryJobTx(ctx, tx, did, kind)` becomes the shared transaction participant. It must preserve the existing unique `(did, job_kind)` coalescing semantics and reactivate completed work when a new trigger requires another run.

Trigger matrix:

| Trigger | `tap_add_repo` | `pds_reconcile` | Transaction boundary |
|---|---|---|---|
| New onboarding completes | Yes | No | Same successful handoff/profile initialization transaction |
| Ordinary sign-in for existing DID | Yes | No, unless it is a same-DID new parent | Same handoff activation transaction |
| Same-DID reauthorization/new parent | Yes | Yes | Same handoff activation transaction |
| Existing source-order uncertainty | No | Yes | Same transaction that records blocked dependency/job state |
| Tap identity hint/status | No | No new classification-triggered repair | Existing identity receipt + refresh enqueue transaction |

Remote Tap administration and CAR fetches remain worker effects after commit. No login/onboarding transaction waits for Tap availability.

### Verified Repository Snapshot Contract

The snapshot fetcher resolves the DID authoritatively and records the expected PDS and signing key only in memory for the attempt. It performs `com.atproto.sync.getRepo` through a new read-only federated HTTP purpose and a 64 MiB limited reader. It must fully consume the response and establish clean EOF before returning a candidate snapshot.

Verification order:

1. Resolve DID uncached and validate the PDS destination through the outbound boundary.
2. Download at most 64 MiB within 2 minutes; reject truncation, overflow, malformed CAR, duplicate/ambiguous root, or cancellation.
3. Load the repository with Indigo primitives and require one expected commit/root.
4. Require commit DID equals the requested DID and commit structure/revision are valid.
5. Verify commit signature against the current DID signing key.
6. Load and verify the entire MST and decode every path needed for registered collections.
7. Resolve the DID uncached again and require PDS origin and signing key to match the attempt start.
8. Only then construct `VerifiedRepositorySnapshot` and permit comparison or inferred absence.

```text
type VerifiedRepositorySnapshot struct { // constructor private to verifier
    DID      syntax.DID
    Revision syntax.TID
    Root     syntax.CID
    Records  map[RepositoryPath]VerifiedRecord
}

Repair(ctx, claim, verifiedSnapshot, dispatcher.Collections())
  -> load all indexed source rows for DID + registered collections
  -> present only in snapshot: deterministic create
  -> same URI, different CID/content: deterministic update
  -> present only in AppView: deterministic delete
  -> unchanged CID: no-op
  -> apply batches of 100 through durable ingestion/projector machinery
```

Repair event IDs/fingerprints must be deterministic from DID, authoritative root/revision, action, URI, and CID so duplicate jobs converge. Existing source revision/CID checks remain the final defense against a repair overwriting a newer Tap event. Repair never writes to or deletes from a PDS. Verified absence of `social.craftsky.actor.profile` follows the existing profile-departure participant, not owner terminalization.

### Identity Cache Contract

All authoritative DID-to-handle results, including `handle.invalid`, enter the existing refresh-request/version flow. The transaction compares candidate version, releases any old mapping for the DID, prevents another DID from retaining a newly verified handle, writes the new current value, and commits. Process-local invalidation happens only after commit for the DID and deduplicated prior/new valid handles. `handle.invalid` can be returned in non-null profile/whoami models but is never accepted in a handle alias lookup.

### Account Deletion Contract

```text
POST intent response: { jobId, confirmationDid, ... }
POST accept request:  { confirmationDid, proof, ... }

CreateIntent(authenticated owner D):
  bind intent to D -> hash exact full D -> return exact full D

AcceptIntent(authenticated owner D, input):
  require constant-time/existing hash verification of exact D
```

No handle resolution, handle cache write, or handle-state branch participates in intent creation or acceptance.

## 6. State, Providers, Controllers, Or DI

### AppView Dependency Graph

```text
federated clients
  -> authoritativeOAuthVerifier
  -> repositorySnapshotFetcher

shared ingestion.Store
  -> auth handoff transaction participant (durable trigger enqueue)
  -> ingestion.Service
  -> projection/repository/quarantine workers

index.TransactionalDispatcher
  -> normal ProjectionWorker
  -> Collections() -> RepositoryRepair

OAuthSessionCoordinator
  -> authoritativeOAuthVerifier
  -> PostgresAuthStore
  -> owner lifecycle fences
  -> existing revocation cleanup queue
```

`newDeps` must build the shared ingestion store before `newAuthDependencies` and `newTapDependencies`, or accept a narrow enqueue interface in auth to avoid an auth-to-ingestion package dependency. Prefer a function/interface adapter over importing `ingestion.Store` into auth. HTTP handlers continue receiving narrow existing dependencies; no handler gets direct authority-verifier access.

### Flutter Provider Graph

```text
SecureSessionRegistryStorage
  -> sessionRegistryProvider
     -> authSessionProvider (SignedIn DID + mutable presentation handle)
     -> sessionValidationLauncherProvider
          -> whoami
          -> updateHandle(captured lease, current handle)

active DID + active lease
  -> activeAccountIdentityProvider
     -> userProfileProvider(Did)
          -> own DID: ProfileRepository.fetchMe()
          -> other DID: ProfileRepository.fetch(did.value)

known model DID / alias resolver result DID
  -> UserProfileRoute(did)
  -> userProfileProvider(Did)
  -> DID-keyed authored-content and mutation caches
```

Provider choices:

- Keep `SessionRegistry` as the existing Riverpod notifier and add one serialized lease-fenced mutation; no new migration provider.
- Keep `UserProfile` as the existing generated `AsyncNotifier` family but change its argument to typed `Did`.
- Keep repository providers and API clients unchanged except where method arguments become DID-explicit.
- Add no global loading gate. Existing async cold-start validation remains fire-and-forget.
- Regenerate Riverpod, go_router, dart_mappable, and localization outputs using existing project generation commands; never hand-edit generated files.

## 7. UI, Widgets, Routes, Or User-Facing Surfaces

Canonical Flutter navigation directly replaces `/profile/:handle` with a DID route. The own profile remains `/profile`. The backend `GET /v1/profiles/@{handleOrDid}` remains the external alias boundary. If Flutter exposes an external handle URL, give it a syntactically distinct `@handle` route, resolve bidirectionally through the AppView, then replace navigation state with the canonical DID route; do not preserve an old-handle redirect.

Widget tree effects:

```text
ProfileRoute (/profile)
  -> activeAccountIdentityProvider
  -> ProfilePage(targetDid: active DID)

UserProfileRoute (/profile/:did)
  -> parse Did at route boundary
  -> ProfileRoutePresentation(did)
  -> userProfileProvider(did)
  -> ProfilePage(targetDid: loaded profile.did)

ProfileAliasRoute (/profile/@:handle), if retained for external input
  -> validate non-sentinel handle
  -> AppView profile alias lookup
  -> replace UserProfileRoute(resolved did)
```

Every self/visitor action is derived after profile load from `profile.did == viewer.did`. Post authors, quoted authors, reposters, notification actors, search results, follow lists, relationship lists, Instagram suggestions, and recent profiles route by their model DID. Mention taps use facet DID while the rich-text widget continues rendering the authored handle bytes.

`ProfileHandle` presentation policy applies to current identity labels in at least account switcher, settings, own profile, profile cards, feed authors, notifications, search results, follow/relationship lists, and mutual followers. With `handle.invalid`, show localized `Handle unavailable` and any available display name/avatar. Never show the sentinel or fall back to a previously valid handle as current identity. Alias-entry validation rejects the sentinel.

The existing `SignOutOn401Interceptor` and account-boundary invalidator handle `pds_session_expired`: remove only the captured child lease, preserve other accounts, and navigate through ordinary sign-in. API failure details retain `error`, `message`, and `requestId`; no migration-reason state or global disabled controls are added.

Account deletion displays the exact server-returned full DID before and after fresh deletion OAuth. The controller enables acceptance only for exact string equality and submits `confirmationDid`. Copy must refer to DID/account identifier, not handle.

No end-user CLI surface is planned. An operator repair command is unnecessary for approved behavior because durable triggers and repository-job inspection already exist; add one only under a separately reviewed operations requirement.

## 8. Error, Loading, Empty, And Edge States

| State / Case | Planned Handling | Requirement IDs | Test IDs |
|---|---|---|---|
| Current PDS and issuer match parent | Continue through existing bounded coordinator and effect; reads remain AppView-backed | FR-001, RULE-003 | UT-001, IT-001, REG-002 |
| PDS or issuer conclusively differs | Fence exact parent/children, enqueue original-issuer cleanup, return `401 pds_session_expired`, send no stale credential/effect | FR-002, FR-003, FR-005 | AT-004, IT-002, IT-004 |
| DID timeout, DNS error, PDS outage, metadata failure, unsafe destination, or generic upstream error | Return retryable federation/policy error, send no credential-bearing effect, preserve parent/owner/data | FR-004, NFR-002, RULE-006 | AT-005, IT-005 |
| Identity hint or equal handle with changed authority | Enqueue/refresh presentation identity only; defer authority decision to next uncached protected boundary | FR-001, FR-007 | UT-005, IT-014 |
| Multiple parents for one DID | CAS-fence only proven parent version and children; retry once on version change; independently valid parent remains selectable | FR-002, FR-006, NFR-001 | UT-003, IT-002, IT-003 |
| Callback authority changes after login start | Persist no parent/child, send no credential to new authority, queue bounded original-issuer cleanup | FR-003, FR-031 | AT-009, IT-016 |
| Callback authority cannot be verified | Persist no mixed session; return failure and retain only bounded cleanup-safe state | FR-004, FR-031, NFR-002 | UT-002, IT-016 |
| Tap admin unavailable after handoff | Handoff remains committed; durable coalesced tracking job retries after restart | FR-010, NFR-001 | IT-007 |
| CAR exceeds 64 MiB, times out, truncates, is malformed, has wrong root/DID/signature, invalid MST, or changes source during fetch | Return bounded reason code, infer no deletes, mark no success, reschedule with existing exponential backoff | FR-009, NFR-003 | AT-010, IT-008 |
| Verified snapshot adds/updates/omits records | Apply deterministic batches through existing source/indexer path; idempotent retries converge | FR-009, NFR-001 | AT-001, IT-006 |
| Verified snapshot omits Craftsky profile | Apply existing departure policy and stop member-only/scheduled work; do not terminalize DID | FR-027, RULE-006 | AT-010, IT-010 |
| Repair fails after verification during projection | Leave job retryable; already applied idempotent batches may be visible; rerun complete snapshot and converge | FR-027, NFR-003 | IT-006, IT-008 |
| Tap reports deleted/deactivated/suspended/takendown | Persist receipt and refresh hint as applicable; never terminalize or purge owner | FR-029, RULE-006 | AT-011, IT-014 |
| Authoritative identity has no valid handle | Commit `handle.invalid`, remove old alias/search mapping, preserve DID content, render unavailable copy | FR-020 | AT-006, UT-014, IT-012 |
| Older identity refresh finishes last | Version CAS fails/no-ops; newer handle remains; invalidate only committed result keys | FR-019, NFR-001 | UT-012, IT-012 |
| Explicit alias fails bidirectional verification | Reject/not-found at alias boundary; never create canonical route/cache entry from one-way mapping | FR-021 | UT-008, IT-012 |
| Flutter cold-start validation pending | Render normal reads and controls; backend remains mandatory effect gate | FR-023 | AT-012, IT-015 |
| Whoami returns same DID/new handle | Persist handle under captured lease and publish updated presentation without changing account identity | FR-012 | AT-002, UT-007 |
| Whoami result belongs to stale lease or different DID | Existing invalidation/fence behavior wins; do not update another generation | FR-012, NFR-001 | UT-007 |
| Mention has historical handle text | Keep authored text and route using facet DID | FR-016 | AT-003, UT-009 |
| Same handle appears for distinct search DIDs | Keep both; deduplicate only exact DID | FR-018 | UT-011 |
| Deletion intent under valid/stale/invalid handle | Ignore handle state; return and require exact authenticated DID | FR-017 | AT-007, IT-013 |
| Scheduled authorization at exactly 30 minutes | Permit the final attempt; success may publish, failure becomes `needs_attention`; later auth cannot auto-publish | FR-025 | AT-008, UT-015, IT-003 |
| Empty profile/search states after DID refactor | Preserve existing empty-state widgets and copy; no migration-specific empty state | NFR-006 | REG-003 |
| `pds_session_expired` on a device with other accounts | Invalidate only captured account lease and select existing MRU fallback through current flow | FR-005, FR-006 | IT-002, REG-001 |

## 9. Test Implementation Plan

Use deterministic HTTP recorders for DID documents, OAuth metadata, old/new PDS and issuers, Tap admin, and CAR downloads. Use real PostgreSQL for fencing, lifecycle, job, identity-cache, and projection assertions. CAR fixtures must be generated/signed deterministically in test helpers and include all registered collections; do not check in secret-bearing real session data.

| Order | Test ID | Target | Setup / Fixture | Initial Expected Failure |
|---|---|---|---|---|
| 1 | UT-001 | `appview/internal/auth/session_coordinator_test.go` | Fake `OAuthAuthorityVerifier`; current, PDS mismatch, issuer mismatch, timeout, DNS, unsafe destination, equal handle/changed authority | Coordinator currently reaches effect without uncached authority comparison |
| 2 | UT-002, IT-016 | `appview/internal/auth/oauth_test.go` | Login starts at A; callback token from A after DID resolves to B; request recorder | Callback currently can persist credentials with newly resolved host |
| 3 | UT-003, IT-002 | Coordinator/background selector and API integration tests | Several parents/children with distinct versions/epochs and concurrent stale checks | Current flow lacks authority-stale exact-parent transition trigger |
| 4 | UT-004, AT-004 | API profile/error-path tests | Coordinator returns confirmed stale error with request ID middleware | Stale authority is not yet produced/mapped end to end |
| 5 | IT-001, IT-004, IT-005 | Federated real-flow, cleanup, lifecycle tests | HTTP destination recorders and seeded owner/private/indexed rows | Old/new authority isolation and transient preservation are unproven |
| 6 | IT-007 | `repository_jobs_integration_test.go`, OAuth/profile initialization tests | Table of onboarding/sign-in/reauth, failed Tap admin, worker restart, duplicate trigger | Direct best-effort `AddRepo` does not survive restart |
| 7 | UT-006 | Dispatcher and repository-repair unit tests | Registered NSIDs plus unverified/verified snapshot builders | Dispatcher exposes no registry and repair can only inspect known ambiguous sources |
| 8 | AT-010, IT-008 | Snapshot fetcher/repair acceptance and integration tests | Truncated, oversized, bad signature, wrong root/DID/MST, changed source, valid signed CAR | No complete CAR trust boundary exists |
| 9 | IT-006, IT-010 | Repair PostgreSQL integration tests | TD-003/TD-004 across all registered collections and optional missing profile | Full repository creates/updates/deletes and departure behavior are absent |
| 10 | IT-011, AT-001, IT-009 | Tap contract and AppView migration acceptance tests | Same/changed handles, PDS/signing-key rotation, missed events, preserved account data | No migration contract proves end-to-end convergence |
| 11 | UT-012, IT-012 | Identity cache refresh/store tests | Controlled refresh versions, old/new/reassigned/sentinel handles, invalidator spy | Invalid result retries and unconditional writes can retain/overwrite aliases |
| 12 | UT-005, IT-014, AT-011 | Identity policy and ingestion lifecycle tests | Ordinary/deleted/deactivated/suspended/takendown events, restart and duplicates | `deleted` currently terminalizes owner; Compose disables replay |
| 13 | UT-007, AT-002 | Flutter session validation and handle-change tests | Strict registry fixture with stale handle and unrelated fields; whoami current handle | Validation currently discards handle |
| 14 | UT-008, REG-004 | Active identity/provider/router tests | Fake `fetchMe`, known DID models, explicit valid alias | Provider and canonical route accept mutable handles |
| 15 | UT-009, AT-003 | Mention, recent search, and cross-feature DID navigation tests | Historical facet DID, reassigned old handle, known-DID surfaces | Mention/recent/internal routes use visible/stored handles |
| 16 | UT-010, UT-011 | Profile page/actions and search pagination tests | Same DID/different handle; same handle/different DIDs | Ownership and deduplication still compare handles |
| 17 | UT-013, IT-013, AT-007 | Backend and Flutter deletion tests | Full DID plus all handle states and near-match confirmation | Contract is handle-bound |
| 18 | UT-014, AT-006 | `profile_handle_test.dart`, profile/card/account identity tests | Sentinel with/without display name and stale local handle | Sentinel renders directly and old alias can remain usable |
| 19 | UT-017, UT-018, AT-012, IT-015 | Credential/inventory/cold-start tests | Source inventory, secure storage snapshot, pending validation and write harness | DID-only inventory and explicit no-global-gate assertions are absent |
| 20 | UT-015, IT-003, AT-008 | Scheduled retry/acceptance tests | Controllable clock at 29:59, 30:00 success/failure, 30:01 | Exact final-attempt boundary is not locked by migration tests |
| 21 | UT-016, IT-017 | Go/Flutter observability secret scans | Unique canaries in all error/result paths and fake telemetry sinks | New signals/redaction coverage do not exist |
| 22 | REG-001 through REG-008 | Existing Go/Flutter suites | Stable DID/handle and unchanged authority fixtures | Detect unintended normal-flow regressions |
| 23 | MAN-001, MAN-002 | Device/staging-like verification | Fake/local expired session and sentinel; generated repair backlog/outcomes | Automated checks cannot prove full presentation or external alert delivery |
| 24 | UT-019, UT-020, IT-018, REG-009 | OAuth metadata cache unit, wiring, real-flow, and observability tests | Controllable clock, bounded cache, concurrent request counters, PDS/issuer transitions, warm cache, callbacks, and TD-009 canaries | Indigo performs two repeated metadata GETs per authority check and has no caching |

Focused commands by phase:

```text
cd appview && go test ./internal/auth ./internal/api
cd appview && go test ./internal/ingestion ./internal/tap ./internal/app
cd appview && go test ./internal/scheduledposts ./internal/observability
cd app && flutter test test/auth/services/session_validation_coordinator_test.dart
cd app && flutter test test/router test/profile test/search test/shared/rich_text test/settings
just appview-test-unit
just app-test
just test                 # Compose PostgreSQL/MinIO required
just appview-check        # release-equivalent AppView evidence
just app-analyze
```

Generation/format commands are run only after the relevant red/green implementation step:

```text
cd app && flutter gen-l10n
cd app && dart run build_runner build --delete-conflicting-outputs
cd appview && gofmt -w <changed Go paths>
```

Inspect generated diffs for unrelated churn before retaining them.

## 10. Sequencing And Guardrails

- First TDD step: add UT-001 to `appview/internal/auth/session_coordinator_test.go`. Inject a fake uncached verifier and prove that a verified PDS/issuer mismatch prevents the operation callback before any credential-bearing client use, while a timeout remains retryable and does not terminalize the parent.
- Dependencies between work items: define the authority contract before callback/coordinator changes; complete effect and callback safety before reauthorization triggers; expose the dispatcher registry before repair comparison; prove snapshot verification before inferred deletes; make identity-cache backend canonical before Flutter unavailable-handle presentation; update backend deletion wire contract before Flutter deletion flow; convert Flutter provider/route identity before broad call-site inventory.
- TDD cadence: each row in Section 9 starts red, receives the smallest implementation to turn green, then is refactored before moving to the next row. Do not build the whole migration path before running focused tests.
- Repository limits: default CAR limit 64 MiB, acquisition/verification timeout 2 minutes, repository lease 3 minutes, finalization margin 5 seconds, repair projection batch 100, repository worker claim batch 8 processed sequentially, retry backoff 2 seconds to 5 minutes.
- Alert thresholds: emit an alertable signal when oldest pending/processing repair age reaches 15 minutes or a repair reaches 5 consecutive attempts. Alerts use bounded job kind/result/reason labels, not DID, handle, URL, or secrets as metric dimensions.
- Authority guardrail: never construct an OAuth client session or call refresh/effect/revocation against newly resolved authority until expected/current origins are proven equal. Revocation of a stale parent uses only its persisted original issuer.
- Fencing guardrail: stale transition predicates include DID, parent session ID, row version, owner lifecycle generation, and auth epoch. A version change reloads once; it never broad-terminalizes all parents for a DID.
- Snapshot guardrail: no comparison API exposes absent paths until complete verification succeeds. Any limit, timeout, parse, EOF, commit, root, DID, revision, signature, MST, or source-change failure produces zero inferred deletes.
- Projection guardrail: repair source rows/events are deterministic and DID-scoped; all changes flow through existing lifecycle-aware indexers. Never mutate another DID or call a PDS mutation endpoint.
- Membership guardrail: missing Craftsky profile invokes existing departure participants only after verified absence. Tap account statuses and migration do not terminalize or purge owners.
- Identity guardrail: `handle.invalid` is valid DID-presentation output but invalid alias input. Historical authored text remains unchanged.
- Flutter guardrail: all known identities use typed DID in provider and route signatures. Do not add handle fallback branches, route redirects for old aliases, PDS endpoint/token fields, or migration-wide state.
- API guardrail: all `/v1/*` request/response changes remain camelCase and all errors retain `{error, message, requestId}`. Confirmed stale authority is exactly HTTP `401` / `pds_session_expired`.
- Security guardrail: test fakes capture destinations and secret canaries; logs, metrics, errors, and snapshots never include access/refresh tokens, DPoP keys/proofs, Craftsky bearer tokens, deletion hashes, or raw session JSON.
- Dependency guardrail: keep exact Indigo and Tap pins. `IT-011` and `REG-006` are mandatory before any later upgrade.
- Migration guardrail: migration `000066` must have reversible up/down coverage and update fixture schemas used by isolated tests. Do not add authority/snapshot progress columns without a demonstrated failing test.
- Out of scope: orchestrating PDS migration; token/key/blob transfer; Flutter-to-PDS communication; standalone importer-specific recovery UX; repository-wide atomic serving; notification/push/analytics suppression during repair; old route/storage compatibility; generic PDS deletion or repair mutation; lexicon changes; proactive identity-event OAuth invalidation; global client write gating.

## 11. Risks And Open Questions

| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Non-blocking, resolved | CAR size, batch, lease, retry, and alert thresholds were unspecified | Without bounds, repair could exceed memory/lease or fail silently | Use 64 MiB, 2-minute operation, 3-minute lease, 100-record projection batch, existing 8-job sequential claim, 2s-5m backoff, 15-minute backlog and 5-attempt alert defaults; validate in config and boundary tests |
| CPQ-002 | Non-blocking | `02-acceptance-tests.md` says Tap status `inactive`, while pinned Tap `0.1.10` emits `deactivated` | A fake-only `inactive` test would not prove real decoder behavior | Implement/test actual wire status `deactivated`; document it as the protocol fixture representing the inactive case without expanding product behavior |
| CPQ-003 | Non-blocking | Current repository worker lease is 45 seconds and tied to JSON request timeout | Complete CAR verification may expire the lease | Raise default to 3 minutes, add explicit 2-minute snapshot operation timeout, retain 5-second completion margin, and validate timeout < lease-margin |
| CPQ-004 | Non-blocking | Repair may apply some verified batches before a later projection failure | Readers can see mixed projections and retry repeats work | Accepted by FR-027; retain deterministic idempotency, source ordering, durable retry, and never report completion until all batches commit |
| CPQ-005 | Non-blocking | Generalizing `oauth_unverified_credentials` from registration to login callback cleanup touches security-sensitive state constraints | Incorrect constraints could leak credentials or block cleanup | Keep purpose/state/expiry predicates explicit, add migration constraint tests and destination-capture IT-016; do not create a second cleanup subsystem |
| CPQ-006 | Non-blocking | DID-first Flutter conversion spans generated families and many feature call sites | Hidden handle-keyed navigation/cache can remain | Use typed DID signatures so missed call sites fail compilation, maintain UT-018 source inventory, and pair each inventoried surface with focused tests |
| CPQ-007 | Non-blocking | Uncached authority verification adds network latency and availability dependence to every authenticated effect | Writes may be slower or transiently unavailable | Preserve 30-second coordinator bound and hardened clients; emit latency/outcome metrics; do not add cache until measured evidence and a separate design justify it |
| CPQ-008 | Non-blocking | No two-real-PDS integration environment exists | HTTP fixtures may miss provider-specific behavior | Approved GAP-001: keep protocol-realistic signed fixtures and make IT-011/REG-006 upgrade gates |
| CPQ-009 | Non-blocking | External alert delivery is deployment-specific | Local automation proves signals, not pager delivery | Retain MAN-002 and capture staging evidence before release |
| CPQ-010 | Non-blocking | Migration number and proposed test paths in earlier documents lag current repository layout | Builders could create duplicate or misplaced files | Use next migration `000066`; use `active_account_identity_provider_test.dart`, existing router/search harnesses, and distributed handler error tests as listed in Section 4 |

No blocking implementation question remains.

## 12. Handoff To TDD Builder

- Coding plan: `04-coding-plan.md`
- TDD execution plan: `05-implementation-plan.md`
- Start with test: UT-001 in `appview/internal/auth/session_coordinator_test.go`
- Focused command: `cd appview && go test ./internal/auth -run 'TestOAuthSessionCoordinator.*Authority'`
- Initial expected failure: no verifier dependency or pre-effect authority comparison exists, so a PDS/issuer mismatch cannot prevent the protected operation.
- First implementation target: `appview/internal/auth/oauth_authority.go` plus the minimal coordinator injection/check in `session_coordinator.go`; use a fake verifier before wiring real federated resolution.
- Next test: UT-002 in `appview/internal/auth/oauth_test.go`, proving ordinary callback authority mismatch persists no session and cleans up only through the original issuer.
- Required evidence before handoff/review: focused red/green output per phase, generated-file diff inspection, `just appview-test-unit`, `just app-test`, full `just test` with Compose PostgreSQL/MinIO, `just appview-check`, `just app-analyze`, migration up/down evidence, and MAN-001/MAN-002 release notes.
- Notes: keep Tap primary; perform repair only after same-DID reauthorization or existing synchronization uncertainty; do not add migration UX/state beyond the standard invalid-session path; stop for lexicon/ADR review if any PDS record shape is implicated.
