# Coding Plan: Account Mutes And Blocks

## 1. Inputs

- Requirements: `01-requirements.md` — Approved with notes, High risk, 2026-07-19
- Tests: `02-acceptance-tests.md` — 15 acceptance scenarios, 16 unit tests, 36 integration tests, 9 regression tests, and 2 manual checks
- Document review: `03-document-review.md` — Approved with notes, High risk, 2026-07-19
- Repository guidance: `AGENTS.md`
- Architecture references:
  - `atproto-craft-social-app-reference.md`
  - `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md`
  - `docs/superpowers/specs/2026-04-22-api-wire-alignment-design.md`
  - `docs/superpowers/specs/2026-04-24-bluesky-backfill-ordering-race-design.md`
- Existing implementation inspected:
  - AppView migrations, OAuth/profile initialization, PDS adapter, Tap consumer/admin CLI, indexer dispatcher, Craftsky/Bluesky profile indexers, follow graph write/index pattern, route registry/policy, profile/post/timeline/search/facet stores, moderation policy, notification lifecycle/list/newness, push outbox/dispatcher, and observability wrappers
  - Flutter fixed-account Dio support, account activation guards, profile/post/search/project/notification repositories and providers, profile actions, shared post card menus, thread state, settings lists, GoRouter routes, localization, and account-bound notification providers
  - Existing real-Postgres Go tests and Flutter repository/provider/widget/router test conventions
- Review findings incorporated:
  - DR-001: confirmed account-scoped Flutter overlays survive stale pre-Tap refreshes and retire only when matching indexed state is observed.
  - DR-002: block creation returns URI/CID/rkey and rapid unblock falls back to a narrow authenticated PDS lookup when `atproto_blocks` has not converged.
  - DR-003: `craftsky_profiles EXISTS` remains the sole membership predicate; join/rejoin requests ordinary Tap backfill without persisted activation/readiness state.
- Approval gate: coding planning is approved. No source implementation, migration execution, or test creation is authorized until the user explicitly approves or invokes `implement-tdd`.

## 2. Implementation Strategy

Implement one AppView-owned indexed relationship subsystem and one DID/account-generation-owned Flutter state boundary. PDS records are canonical for public blocks, Tap owns AppView projection, and Flutter bridges the accepted convergence window with confirmed optimistic state.

1. Add a reversible migration for private mutes and public block projection. Mute ownership has a foreign key/cascade only on the owner, never the subject. Block rows have no membership foreign key so public records survive subject or owner absence. Do not add write-intent or membership-activation tables.
2. Create `appview/internal/relationships` as the auditable policy and persistence boundary. A pure decision table owns moderation > block > mute precedence and surface-specific outcomes. The package also owns current-member checks, target canonicalization, interaction authorization, batched viewer state, opaque relationship-list cursors, reference checks, and the small set-based SQL predicate builder used by query stores.
3. Keep block canonicality on the caller's PDS. Use the existing result-returning `createRecord` path and report success after the PDS returns URI/CID/rkey, without synchronously writing `atproto_blocks`. On create retry or unblock, use indexed rkeys when present and fall back to listing the caller's `app.bsky.graph.block` records through the authenticated PDS client when Tap may lag. Never create while any matching subject record exists; unblock deletes every matching caller-owned rkey idempotently. Record-not-found is converged success.
4. Add one `app.bsky.graph.block` indexer using Indigo's generated `bsky.GraphBlock` type. Retain valid rows even when either account is absent, collapse duplicate active pairs deterministically for policy/list shaping, process repository-ordered create/update/delete events idempotently, register the NSID once, and add it to Tap collection filters. The indexer is the only writer of `atproto_blocks`.
5. Keep membership and backfill on existing boundaries. A `craftsky_profiles` row alone makes a DID current. OAuth/profile initialization requests Tap repository tracking through a reusable bounded admin client but does not wait for block convergence or create readiness state. Current-member-owned retained blocks targeting the joining DID apply immediately when membership appears; joining-owned historical blocks become effective as Tap backfills them. Restart/retry uses the existing OAuth/repository-tracking path and Tap redelivery, with lag/failure observability rather than a persisted activation worker.
6. Apply `craftsky_profiles EXISTS` before identity hydration or PDS mutation and inside collection/count queries. Remove the non-Craftsky profile hydration fallback from user-facing profile/report/follow paths. Membership loss hides retained public/private subject relationships; profile-owner deletion cascades only owner-private mutes.
7. Apply policy in SQL selection, not after pagination. Timeline, search, discovery, profile tabs, graph lists/counts, mentions, notifications, and relationship lists select eligible rows before `LIMIT`. Batch response hydration adds viewer state and shapes quotes/replies without N+1 relationship reads. Direct muted reads remain allowed; blocked direct content uses the ordinary generic unavailable/not-found envelope and never serializes protected content.
8. Keep notification events as retained history when ordinary preference policy permits, but dynamically exclude relationship-ineligible events from list/newness/badge. Mute mutation and Tap block indexing transactionally cancel pending/retry/leased-unsent deliveries when the relationship becomes effective in AppView state. Both claim and final lease ownership checks re-evaluate indexed relationship policy; unmute/unblock never creates a new delivery for an old event.
9. Add account-family Flutter relationship repositories using `accountDioProvider(account)`, plus a notifier keyed by `AccountKey` and subject DID. Seed it from additive profile, profile-summary, post-author, and notification-actor viewer fields. Optimistic mutation is fenced by the captured session generation, updates loaded state immediately, rolls back only the initiating account, and upgrades to a confirmed overlay after PDS success. The overlay wins over stale pre-Tap refreshes, invalidates affected pages for eligible-row refill, and retires when matching indexed server state is observed.
10. Centralize profile and `PostCard` safety menus so every non-self ordinary post, project post, comment, and reply receives the same current Mute/Unmute, Block/Unblock, and Report behavior. Extend thread response/state with a content-free muted-branch placeholder and an explicit direct branch-load path for temporary reveal. Existing quote-view states become `muted` (revealable) or `blocked`/`unavailable` (never revealable).

No Craftsky lexicon or generated Craftsky lexicon file changes are planned. No private mute is written to any PDS. No public follow/content/interaction record is deleted as a side effect of relationship or subject-membership changes.

## 3. Affected Areas

| Area | Existing Pattern | Planned Change | Requirement IDs | Acceptance Criteria IDs | Test IDs |
|---|---|---|---|---|---|
| Persistence | Direct pgx migrations and store SQL; active follow rows only | Add owner-private mutes and Tap-owned block projection with bidirectional/owned-list indexes; no intent or activation tables | FR-002–FR-006, FR-020, FR-029, NFR-004 | AC-008–AC-014, AC-031, AC-042, AC-052–AC-053, AC-059 | IT-002, IT-005–IT-009, IT-025–IT-026, IT-030, IT-035 |
| Relationship policy | Moderation predicates are embedded per store; no personal relationship layer | Add pure policy/authorization/reference decisions plus tested set-based SQL predicates and batch state reads | BR-001, BR-003, NFR-002, RULE-001–RULE-003, RULE-006–RULE-008 | AC-002, AC-005, AC-040, AC-045–AC-050, AC-058 | UT-001, UT-005–UT-008, UT-015, IT-027–IT-028, REG-001, REG-003, REG-006, REG-008 |
| Membership boundary | `craftsky_profiles` joins coexist with non-member profile hydration and follow/report targeting | Use profile-row existence everywhere, remove user-facing external hydration, retain source records on absence, and request ordinary Tap backfill on join/rejoin | BR-004, FR-001, FR-005, FR-028–FR-029, RULE-004–RULE-005 | AC-007, AC-013, AC-051–AC-053, AC-059 | UT-002, UT-014, UT-016, IT-001, IT-008, IT-024–IT-026, REG-005 |
| PDS block writes | Follow uses `createRecord`; durable reads wait for Tap | PDS-confirmed create/delete with URI/CID/rkey response; no local projection; authenticated PDS lookup fallback for rapid pre-index unblock | BR-002, FR-003, FR-006 | AC-003, AC-010–AC-011, AC-014 | IT-005–IT-006, IT-009, IT-032, MAN-002 |
| Tap/indexing | Dispatcher has follow/profile/post interaction indexers; profile backfill is best effort | Register block once; make Tap the sole block-projection writer; reconcile ordered events and duplicate replay; observe backfill lag/failure | BR-002, FR-004–FR-006 | AC-004, AC-012–AC-014, AC-059 | UT-010, IT-006–IT-009, IT-025, IT-032, IT-036, MAN-002 |
| API contracts | `/v1/` stdlib routes, bearer/device middleware, route-policy registry, camelCase envelopes | Add six relationship routes, limits/cursors, viewer fields, stable errors, and existing rate/body classes | FR-001–FR-008, FR-026 | AC-007–AC-016, AC-038 | UT-003, UT-009, IT-001–IT-011, IT-023, REG-007 |
| Read surfaces | Store-specific moderation SQL; some post methods lack viewer DID | Carry viewer/surface context and filter/shape profile, graph, post, timeline, search, project, facet, thread, quote, repost, and aggregate paths | FR-009–FR-016, FR-020–FR-021, FR-028, FR-030, RULE-006, RULE-008 | AC-017–AC-026, AC-031–AC-032, AC-049, AC-051–AC-052, AC-056, AC-058, AC-060 | UT-003–UT-006, UT-014–UT-015, IT-010–IT-015, IT-020–IT-022, IT-024–IT-025, IT-027, IT-029, IT-033–IT-034 |
| Directed writes | Follow/like/repost/reply/quote/mention handlers validate target but have no block authorization | Resolve all direct and embedded actors before PDS work; deny new writes symmetrically; keep owned cleanup/report/reciprocal block | FR-017–FR-018, FR-028, RULE-007 | AC-027–AC-028, AC-050–AC-052 | UT-002, UT-008, IT-001, IT-016–IT-017, IT-024, REG-004 |
| Notifications and push | Preference-time eligibility, durable list/newness, cancellable leased delivery | Retain eligible history, dynamically filter relationships, cancel unsent work transactionally, and recheck before provider send | FR-013, FR-019, FR-031 | AC-021–AC-022, AC-029–AC-030, AC-057 | UT-007, IT-018–IT-020, AT-006 |
| Flutter state/data | Active-account repositories plus fixed account Dio for account families; optimistic follow state | Add account-keyed relationship cache/controller and repository; generation-fence late completion, retain confirmed overlays across stale pre-Tap refreshes, and invalidate affected surfaces | FR-007–FR-008, FR-025, NFR-001 | AC-015–AC-016, AC-036–AC-037, AC-039 | UT-003, UT-011–UT-012, IT-009, AT-012, REG-009 |
| Flutter UI | Profile follow/share/report actions, shared `PostCard` report/delete menu, follower/following settings lists | Add relationship states/actions/confirmations, content menus, placeholders/reveal, settings lists, and not-found behavior | BR-001, FR-010–FR-016, FR-022–FR-024, FR-027–FR-028, NFR-005 | AC-001, AC-018–AC-026, AC-033–AC-035, AC-043, AC-051, AC-054–AC-055 | AT-001, AT-003–AT-005, AT-007–AT-011, AT-013–AT-015, UT-013, MAN-001 |
| Observability/performance | Bounded existing HTTP/PDS/Tap/push metrics | Add outcome/stage/error-class signals only; query-plan/no-N+1 gates and private sentinel tests | NFR-001, NFR-003–NFR-004, NFR-006 | AC-006, AC-039, AC-041–AC-042, AC-044 | UT-009, UT-012, IT-003, IT-029–IT-031, REG-009 |

## 4. Files And Modules

Generated `.g.dart`, mapper, and localization files are regenerated from source; they are not hand-edited. File groupings below are implementation targets, not permission to edit them during this planning stage.

### AppView

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `appview/migrations/000023_mutes_blocks.up.sql`, `.down.sql` | Create | Create only `actor_mutes` and `atproto_blocks`; add pair/URI/rkey constraints, owner cleanup, and bidirectional/owned-list indexes; do not create intent or activation tables | FR-002–FR-006, FR-020, FR-029, NFR-004 | IT-002, IT-007–IT-008, IT-025–IT-026, IT-030, IT-035 |
| `appview/internal/db/mutes_blocks_migration_test.go` | Create | First red/reversible migration contract, constraint/index inspection, and owner-versus-subject lifecycle | FR-002, FR-004, FR-029, NFR-004 | IT-035 |
| `appview/internal/relationships/state.go`, `policy.go` | Create | Typed `State`, surface/operation enums, precedence, visibility/placeholder/write decisions, and privacy-safe bounded result labels | BR-001, NFR-002, RULE-001–RULE-003 | UT-001, IT-028, REG-001, REG-003, REG-008 |
| `appview/internal/relationships/store.go`, `sql_predicates.go` | Create | Mute CRUD, read-only block projection queries, batch states, effective-pair checks, eligible relationship lists, and set-based query fragments | FR-002, FR-006–FR-008, NFR-003–NFR-004 | IT-002–IT-004, IT-009–IT-011, IT-029–IT-030 |
| `appview/internal/relationships/target.go`, `membership.go` | Create | Parse/canonicalize handle or DID once, reject self, and apply active current-member predicate without external hydration | FR-001, FR-028, RULE-004 | UT-002, UT-014, IT-001, IT-024 |
| `appview/internal/relationships/authorization.go`, `reference.go`, `lifecycle.go` | Create | Directed-write matrix, third-party/indirect reference checks, and distinct owner/subject/session lifecycle decisions | FR-017–FR-018, FR-029–FR-030, RULE-006–RULE-007 | UT-008, UT-015–UT-016, IT-016–IT-017, IT-026–IT-027, IT-033 |
| `appview/internal/relationships/*_test.go` | Create | Pure table tests, real-Postgres store tests, privacy isolation, pagination, query plans, and inventory/import-boundary guard | All server relationship requirements | UT-001–UT-002, UT-007–UT-010, UT-014–UT-016, IT-002–IT-004, IT-011, IT-024, IT-028–IT-031 |
| `appview/internal/api/relationship.go`, `relationship_request.go` | Create | Six authenticated handlers, standard envelopes, stable limits/cursors, updated relationship/profile responses including block URI/CID/rkey, and PDS/store failure mapping | FR-001–FR-003, FR-006, FR-008, FR-026 | UT-009, IT-001–IT-006, IT-009, IT-011, IT-023, REG-007 |
| `appview/internal/api/relationship_test.go`, `block_test.go`, relationship list/store tests | Create | Handler/PDS ordering, no synchronous block projection, idempotency, rapid pre-index unblock, ownership, lists, and route contract behavior | FR-001–FR-008, FR-026 | IT-001–IT-006, IT-009, IT-011, IT-023, IT-032 |
| `appview/internal/auth/pds_client.go`, `pds_client_indigo.go` | Change | Preserve result-returning `createRecord`; add a narrow authenticated paginated `app.bsky.graph.block` listing operation used only when unblock cannot find an indexed row; wrap it in existing PDS error/session semantics | FR-003, FR-006 | IT-005–IT-006, IT-009, MAN-002 |
| `appview/internal/observability/pds.go` and PDS fakes/tests | Change | Observe block create/delete/list-fallback operations with bounded operation/result/stage fields and update interface implementations | NFR-006 | UT-012, IT-031 |
| `appview/internal/index/bluesky_block.go` | Create | Decode Indigo `bsky.GraphBlock`, validate subject/time, idempotently apply ordered URI/CID/rkey create/update/delete events, collapse duplicate pairs for effective policy, retain absent subjects, and invoke delivery suppression when a block becomes indexed | FR-004, FR-006, FR-019–FR-020 | UT-010, IT-006–IT-007, IT-018–IT-020, IT-032 |
| `appview/internal/index/bluesky_block_test.go` | Create | External/replay/malformed/duplicate/mutual/absent-member/delayed-Tap convergence fixtures | BR-002, FR-004–FR-006, RULE-002 | UT-010, IT-006–IT-009, IT-032 |
| `appview/internal/tap/admin_client.go`, `admin_client_test.go` | Create | Extract URL conversion and `/repos/add` from CLI into a reusable bounded client for ordinary OAuth/join/rejoin repository tracking | FR-005 | IT-008, IT-025, IT-036 |
| `appview/cmd/cli/tap_repo_check.go`, `tap_test.go` | Change | Reuse the internal Tap admin client; retain repo-check behavior and test collection configuration | FR-004–FR-005 | IT-036, MAN-002 |
| `appview/internal/index/craftsky_profile.go`, `bluesky_backfiller.go` | Change | Keep profile-row membership canonical, preserve existing Bluesky profile hydration, ensure join/rejoin repository tracking is requested, and owner-delete private mutes via transaction semantics | FR-005, FR-029 | IT-008, IT-025–IT-026 |
| `appview/internal/index/craftsky_profile_test.go`, `bluesky_backfiller_test.go` | Change | Hold/restart Tap backfill; assert retained inbound applies with profile membership and joining-owned outbound converges later without readiness state | FR-005 | IT-008, IT-025, GAP-005 |
| `appview/internal/auth/handlers_oauth.go`, `initialize_profile.go`, tests | Change | Request Tap tracking/resync during initialization, preserve current profile-invalid/PDS error behavior, and allow session/profile completion without waiting for block convergence | FR-005, FR-028 | IT-008, IT-025, REG-005 |
| `appview/internal/app/deps.go`, `appview/cmd/appview/server.go` | Change | Wire relationship/membership queries, block indexer, notification lifecycle, and Tap admin client; do not add activation timeout or startup activation worker | FR-004–FR-005, NFR-006 | IT-008, IT-025, IT-031, IT-036 |
| `docker-compose.yml` | Change | Add `app.bsky.graph.block` exactly once to `TAP_COLLECTION_FILTERS`; retain signal collection and no-replay configuration | BR-002, FR-004 | IT-007, IT-036, MAN-002 |
| `appview/internal/routes/routes.go`, `policy.go`, route tests | Change | Register six `/v1/profiles/...` relationship routes with existing auth/device middleware; lists are Read/no-body and mutations Write/no-body | FR-008, FR-026 | IT-023, REG-007 |
| `appview/internal/api/profile_store.go`, `profile.go`, `profile_response.go` | Change | Remove non-member hydration from user-facing reads/reports, require active membership, batch viewer relationship state, shape blocked shell, filter graph counts/lists, and add booleans to full/summary shapes | FR-007, FR-016, FR-021, FR-028 | UT-003–UT-004, UT-014, IT-003, IT-010, IT-021, IT-024, REG-005–REG-006 |
| `appview/internal/api/follow.go`, `follow_store.go` | Change | Require active target; deny create across either block before PDS; keep owned unfollow cleanup; hide follow state/count/list pair without deleting rows | FR-017–FR-018, FR-021, FR-028 | IT-016–IT-017, IT-021, IT-024, REG-002, REG-004–REG-006 |
| `appview/internal/api/post_store.go`, `post.go`, `post_response.go` | Change | Add viewer/surface context, query-time policy, minimal direct block error, muted branch views, quote states, blocked-reference stripping, and relationship-aware engagement hydration | FR-010–FR-018, FR-020, FR-027, FR-030 | UT-005–UT-006, UT-008, UT-015, IT-013–IT-017, IT-020, IT-027, IT-033–IT-034, REG-001–REG-004 |
| `appview/internal/api/timeline_store.go`, `timeline.go` | Change | Exclude muted/blocked authors, repost actors, and reposted authors before limit/cursor; batch quote/reference policy | FR-009, FR-014–FR-015 | IT-012–IT-013, IT-015, IT-029, REG-008 |
| `appview/internal/api/search_store.go`, `search.go`, `search_response.go` | Change | Carry viewer DID through every post/project/hashtag/profile query, omit protected rows, allow only exact-handle minimal blocked management result, and preserve eligible pagination | FR-009, FR-014, FR-021–FR-022, FR-028 | IT-012–IT-013, IT-021–IT-022, IT-024, IT-029, REG-008 |
| `appview/internal/api/facet_store.go` and facet handlers/tests | Change | Restrict mention suggestions/resolution to active members and omit blocked accounts; return identical not-found for absent direct resolution | FR-017, FR-028 | IT-016, IT-024, REG-005 |
| `appview/internal/notifications/service.go`, `eligibility.go`, `newness.go` | Change | Evaluate relationship state at ingestion, retain normally eligible history, skip outbox when suppressed, and expose relationship decision classes | FR-013, FR-019, FR-031 | UT-007, IT-018–IT-020 |
| `appview/internal/api/notification_store.go`, `notification_newness.go`, `notifications.go` | Change | Filter list/new-count at query time, shape every hydrated reference under current policy, and prevent stale actors/subjects from leaking | FR-013, FR-019, FR-028, FR-031, RULE-006 | IT-018–IT-020, IT-024, IT-033, REG-008 |
| `appview/internal/push/dispatcher.go`, tests | Change | Exclude/cancel relationship-ineligible rows at claim, recheck under exact lease immediately before provider send, and never recreate sent/cancelled deliveries | FR-013, FR-019, FR-031 | UT-007, IT-018–IT-020 |
| `appview/internal/observability/relationship.go` and tests | Create | Bounded counters/timers and safe structured fields for mutation/index/backfill-lag/denial/suppression/cancellation; no target/pair/rkey/post URI labels | NFR-001, NFR-006 | UT-012, IT-031 |

### Flutter

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `app/lib/profile/models/profile.dart`, `profile_account_summary.dart`, generated mappers | Change / Generate | Decode additive `muted`, `blocking`, and `blockedBy`; model blocked shell nullable detail/count fields | FR-007, FR-016 | UT-003–UT-004, IT-010, AT-009 |
| `app/lib/feed/models/post.dart`, `post_comment_section.dart`, generated mappers | Change / Generate | Add viewer relationship fields to authors, quote `muted`/`blocked` states, and content-free muted branch references/placeholders | FR-010–FR-015, FR-027 | UT-005–UT-006, AT-003–AT-005, AT-007–AT-010, AT-014 |
| `app/lib/profile/data/profile_api_client.dart`, `profile_repository.dart`, `api_profile_repository.dart` | Change | Add mute/unmute/block/unblock and paginated owned-list operations with the existing `unwrapApi` error boundary | FR-008, FR-025–FR-026 | IT-023, AT-011–AT-012 |
| `app/lib/profile/providers/profile_repository_provider.dart` and account-family providers | Change / Create | Add `accountRelationshipRepositoryProvider(AccountKey)` backed by `accountDioProvider(account)`; keep ordinary active profile repository behavior | NFR-001, FR-025 | UT-011–UT-012, REG-009 |
| `app/lib/profile/models/profile_relationship.dart` | Create | Immutable relationship state, action enum, pending/error metadata, and strict block-over-mute computed state | FR-007, FR-023, FR-025 | UT-011, AT-009, AT-012 |
| `app/lib/profile/providers/profile_relationship_provider.dart` | Create | Account-keyed notifier/cache seeded from server DTOs; one in-flight subject/action; optimistic apply/rollback; confirmed-overlay reconciliation; account-session generation fencing; targeted invalidation effects | FR-025, NFR-001 | UT-011–UT-012, IT-009, AT-012, REG-009 |
| `app/lib/auth/providers/account_boundary_provider.dart` | Change | Invalidate/dispose relationship state and reveal state with the initiating account; include new repositories/providers in the boundary inventory | FR-025, NFR-001 | UT-011, AT-004, AT-012, REG-009 |
| `app/lib/profile/widgets/profile_actions.dart`, `profile_relationship_menu.dart` | Change / Create | Keep Follow and current Mute/Unmute visible; use the shared adaptive More menu for Share, Block/Unblock, and Report | FR-022–FR-023 | AT-001, AT-009, AT-013 |
| `app/lib/profile/pages/profile_page.dart`, profile meta/tab widgets | Change | Render distinct muted/blocking/blockedBy annotation; use minimal blocked shell; hide invalid tabs/metrics/follow; wire confirmations and feedback | FR-016, FR-022–FR-023, FR-025 | AT-001, AT-003, AT-009, AT-012–AT-013, AT-015 |
| `app/lib/feed/widgets/post_card.dart` | Change | Make the shared More menu relationship-aware for every non-self post-shaped use; retain owned Delete and non-owned Report; render quote placeholders | FR-011, FR-015, FR-027 | AT-005, AT-008, AT-010, AT-013, UT-006 |
| Feed/search/project/profile list builders and providers | Change | Apply optimistic author removal/collapse from account state, then invalidate/refetch timeline, search, discovery, profile tabs, and project pages to fill from server | FR-009, FR-014, FR-025, FR-027 | AT-007, AT-010, AT-012, REG-009 |
| `app/lib/feed/pages/post_thread_page.dart`, `post_comment_section_provider.dart` | Change | Render one muted branch placeholder, load its direct branch explicitly on reveal, keep reveal only in that account/root provider instance, collapse on refresh/navigation/switch, and never reveal blocked branches | FR-010, FR-014–FR-015, FR-025, FR-027 | AT-003–AT-004, AT-007–AT-008, AT-010, AT-012, AT-014, UT-005 |
| `app/lib/settings/pages/settings_page.dart`, `relationship_list_page.dart` | Change / Create | Add localized Muted accounts and Blocked accounts entries/pages with loading, empty, error, retry, pagination, and row-level reversal | FR-008, FR-024–FR-025 | AT-011–AT-013 |
| Notification models/providers/pages/effect routing | Change | Seed actor state, optimistically remove muted/blocked actor rows, refresh per-account new count/badge, and rely on destination server recheck for stale opens | FR-013, FR-019, FR-025, FR-031, RULE-006 | AT-006, AT-008, AT-012, UT-007, UT-015, REG-009 |
| `app/lib/l10n/app_en.arb`, generated localization files | Change / Generate | Add all controls, confirmations, public-block warning, relationship annotations, placeholders, list states, feedback, and accessibility labels | NFR-005 | UT-013, AT-013, MAN-001 |
| Flutter relationship/profile/feed/settings/router tests and fakes | Change / Create | Extend fixed-account fake repositories and localized widget/router harnesses for every relationship and lifecycle state | All Flutter-facing requirements | AT-001–AT-015, UT-003–UT-006, UT-011–UT-013, UT-015, REG-009 |

## 5. Services, Interfaces, And Data Flow

### Persistence contract

`actor_mutes` is private and owner-scoped:

- Primary key: `(owner_did, subject_did)`.
- `owner_did` references `craftsky_profiles(did) ON DELETE CASCADE` so permanent owner membership removal deletes owned mutes.
- `subject_did` deliberately has no membership foreign key; subject departure hides rather than deletes the preference.
- `created_at` and `updated_at` plus `(owner_did, created_at DESC, subject_did)` support stable owned-list pagination.
- No SQL query exposes another owner's rows.

`atproto_blocks` is the Tap-owned public projection:

- URI is the stable primary identity; blocker DID/rkey, CID, subject DID, generated record JSON, created time, and indexed time are retained.
- Owner/rkey and blocker/subject indexes support exact deletes, both policy directions, duplicate-subject collapse in reads, and owned lists.
- A block row has no foreign key to membership. User-facing joins apply `craftsky_profiles EXISTS` separately.
- Create/update upserts by URI; delete removes that URI. Repository-ordered Tap delivery and idempotent replay provide convergence without local tombstones or generations.
- API block/unblock handlers never insert, update, or delete this table directly.

No `block_write_intents` or `craftsky_membership_activations` table is planned. Canonical PDS state, Tap-owned projection, and Flutter's confirmed account-scoped overlay remain separate consistency layers.

### Core interfaces

```text
// Partial Go signatures only.
package relationships

type State struct {
    Muted    bool // viewer -> subject only
    Blocking bool // viewer owns an active block
    BlockedBy bool // subject owns an active block
}

type Surface int // profile, directPost, topLevel, thread, quote, graph, notification, push, searchExact
type Decision int // allow, omit, mutedPlaceholder, blockedPlaceholder, minimalProfile, denyInteraction

func Decide(platform ModerationState, relationship State, surface Surface) Decision
func Authorize(state State, operation Operation, ownsResource bool) error

type Reader interface {
    State(ctx context.Context, viewer, subject syntax.DID) (State, error)
    BatchStates(ctx context.Context, viewer syntax.DID, subjects []syntax.DID) (map[syntax.DID]State, error)
    AnyBlock(ctx context.Context, left, right syntax.DID) (bool, error)
    IsCurrentMember(ctx context.Context, did syntax.DID) (bool, error)
}

type MutationService interface {
    Mute(ctx context.Context, owner, subject syntax.DID) (State, error)
    Unmute(ctx context.Context, owner, subject syntax.DID) (State, error)
    Block(ctx context.Context, owner, subject syntax.DID, pds auth.PDSClient) (BlockMutationResult, error)
    Unblock(ctx context.Context, owner, subject syntax.DID, pds auth.PDSClient) (BlockMutationResult, error)
}

type BlockMutationResult struct {
    State State
    URI   syntax.ATURI
    CID   syntax.CID
    Rkey  syntax.RecordKey
}
```

The pure Go decision is normative for response and write behavior. `SQLPredicates` mirrors the same table for set-based selection and is covered by the same matrix plus the inventory test; stores do not hand-author slightly different mute/block clauses.

### Mute mutation flow

```text
authenticated owner
  -> canonical target resolver
  -> active membership + non-self check
  -> transaction
       -> INSERT actor_mutes ... ON CONFLICT idempotent
       -> cancel pending/retry/leased-unsent deliveries for owner <- subject
  -> commit
  -> batch-read updated viewer state/profile
  -> 200 camelCase response
```

Unmute deletes only the authenticated owner's pair. Notification events remain. No old push delivery is created during unmute, and retained events become list/newness eligible only through the ordinary dynamic query.

### Block mutation and reconciliation flow

```text
POST block
  -> target/member/non-self checks
  -> find indexed caller-owned block for subject
  -> if missing, authenticated PDS listRecords fallback finds an existing
     caller-owned subject record (covers retry before Tap)
  -> if still missing, PDS createRecord(app.bsky.graph.block) -> URI + CID + rkey
  -> return response with blocking=true and record identity
  -> perform no atproto_blocks write; Tap may not have delivered yet

DELETE block
  -> select caller-owned indexed URI/rkey(s) for subject
  -> if missing, authenticated PDS listRecords fallback finds exact record(s)
  -> PDS deleteRecord(each exact rkey); RecordNotFound is converged success
  -> return response with blocking=false and no local projection mutation
  -> a reciprocal inbound block still appears as blockedBy after Tap/current reads

Tap block create/update/delete
  -> validate generated external record + typed identifiers/time
  -> upsert or delete the URI idempotently in repository order
  -> use pair EXISTS/deduped list shaping when duplicate records exist
  -> cancel unsent deliveries in the transaction that indexes a newly effective pair
```

The response uses the same override pattern as follow: read the current profile shape, apply the PDS-confirmed relationship override, and return it without claiming the AppView projection has converged. A retry during Tap lag first checks the PDS collection, preventing a second Craftsky-created active record without durable intents. Rapid unblock similarly deletes the existing PDS rkey even when the local table is still empty. Ordered Tap create/delete events may briefly move indexed policy through the intermediate state, while Flutter retains the confirmed final overlay.

### Membership and join/rejoin backfill flow

```text
OAuth callback / profile initialization
  -> InitializeProfileAndIdentityCache
  -> tapAdmin.EnsureRepo(did) [/repos/add requests ordinary tracking/backfill]
  -> Craftsky profile event inserts/updates craftsky_profiles
  -> CraftskySessions.Create + token handoff follows existing behavior

Tap repository backfill (serial, key ordered)
  -> app.bsky.graph.block events for joining repo
       -> block indexer retains joining-owned outbound blocks
  -> social.craftsky.actor.profile event
       -> upsert craftsky_profiles; membership is now current
       -> run required Bluesky profile backfill
  -> retained current-member-owned blocks targeting the joining DID become
     effective as soon as craftsky_profiles EXISTS
  -> any joining-owned blocks not yet indexed become effective as Tap reaches them
```

There is no activation wait, public-record-set verifier, startup activation scan, or second membership state. IT-008 and IT-025 hold/restart Tap after the profile row is visible: retained inbound rows must apply immediately, joining-owned outbound rows may lag, and resumed Tap processing must converge them. Repeated OAuth/join initialization may call `EnsureRepo` idempotently. Tap lag and poison-event failures emit bounded operational signals; they do not change membership eligibility.

### Read and response policy

- Full profiles and account summaries batch-load `State`. `muted`, `blocking`, and `blockedBy` are additive camelCase fields scoped to the requesting viewer.
- A block in either direction produces the minimal profile shell: DID, current handle, safe display identity needed for orientation, relationship booleans, and report/owned-unblock/reciprocal-block capability. Bio, crafts, metrics, mutuals, follow state, tabs, and activity are omitted.
- Direct muted profile/post/profile-tab reads are allowed and annotated. Top-level lists omit muted authors and muted repost actor/subject combinations before pagination.
- Direct blocked post/thread reads return the same generic unavailable/not-found contract regardless of direction and serialize no post text, media, facets, quote payload, author detail, counts, or viewer metrics.
- Thread list items become a small view union. A muted branch item contains only placeholder state plus a stable direct branch reference; the initial response omits that branch's payload and descendants. Explicit reveal loads the already-allowed direct comment/reply endpoints and keeps the resulting branch only in the current Flutter provider instance. A blocked branch is omitted or represented by an unrevealable generic placeholder with no reference payload that can navigate to protected content.
- Quote hydration uses `visible`, `muted`, `blocked`, `hidden`, and `unavailable`. Only `visible` has a post payload. Flutter may explicitly fetch a muted direct reference to reveal; blocked/hidden/unavailable never offer reveal.
- Third-party reference checks evaluate both viewer-to-participant policy and block edges between referenced participants. This prevents Carol's thread/quote/mention from connecting Alice and Bob across their block.
- Relationship and content pages filter in the SQL/recursive selection. Cursors are produced from the last eligible row, never a hidden row.

### Directed-write authorization

- Follow, like, repost, reply, quote/embed, and mention creation resolve every target/current author before constructing a PDS body. Any block direction returns `interaction_blocked` and performs zero PDS/local write calls.
- Follow/profile/report/mention targets must be active current members. Unknown and resolvable non-members both return `404 profile_not_found` at account-target boundaries.
- Unfollow, unlike, unrepost, owned post/comment delete, profile/content report, and reciprocal block remain allowed. Ownership is checked independently; a block never authorizes deleting another account's record.
- `CreatePostHandler` gathers the reply parent author, quoted author, and valid mention-facet DIDs as a batch and authorizes all before the PDS write. This avoids partial writes and per-mention queries.

### Notification and push boundaries

`notifications.Service.Activate` first applies existing preference/self/follow rules. If normally retainable, it upserts the durable event even when a current relationship suppresses delivery; it records that initial push was evaluated and creates no outbox row. List/newness queries dynamically join current relationship and active membership, allowing retained history to reappear after relationship removal without reinsertion.

Mute creation and Tap block indexing cancel `pending`, `retry`, and `leased` deliveries in the same database transaction that makes the relationship effective in AppView state. The push dispatcher repeats eligibility in its claim query and in `ownsCurrentDelivery` immediately before `Sender.Send`. Compare-and-set lease checks remain intact. `succeeded`, `cancelled`, and expired deliveries are never changed back to sendable on unmute/unblock.

## 6. State, Providers, Controllers, Or DI

### AppView dependency graph

```text
app.newDeps
  -> relationships.Store + Policy + MutationService
       -> authenticated PDS block record lister fallback
       -> notifications relationship lifecycle callback
       -> observer
  -> tap.AdminClient [ordinary join/rejoin repository tracking]
  -> notifications.Service(relationship reader)
  -> index.Dispatcher
       -> BlueskyBlock(relationship store/lifecycle)
       -> CraftskyProfile(existing membership row + bsky backfiller)
  -> push.Dispatcher(relationship reader)
  -> api Profile/Post/Search/Facet/Follow stores(relationship reader/predicates)
  -> routes.AddRoutes(specific handler dependencies only)
```

`Deps` carries the concrete shared stores/services, but handler constructors continue receiving only the narrow interfaces they use. No activation service, startup resumer, or membership worker is added.

### Flutter provider graph

```text
sessionRegistryProvider
  -> accountDioProvider(AccountKey) [fixed token + generation]
  -> accountRelationshipRepositoryProvider(AccountKey)
  -> accountRelationshipsProvider(AccountKey)
       key: subject DID
       value: indexed seed + optimistic/confirmed overlay + pending action
       -> relationshipMutationEffectsProvider
            -> update existing loaded profile/post/thread/notification state
            -> invalidate timeline/search/project/profile/notification pages

active UI
  -> active AccountKey from session registry
  -> seed relationship notifier from server DTO
  -> watch only target DID entry
  -> invoke mutation with captured ActiveAccountLease
```

Provider choices:

- `accountRelationshipRepositoryProvider(AccountKey)` is a `FutureProvider` family because `accountDioProvider` is asynchronous and fixed to one account generation.
- `accountRelationshipsProvider(AccountKey)` is a generated `Notifier` family holding a DID-keyed immutable map. Network methods set per-subject pending state rather than putting the whole account map into `AsyncLoading`.
- One in-flight operation is allowed per account/subject/action. Same-action duplicate taps share/no-op; conflicting action waits for or rejects the current operation rather than racing state generations.
- Completion is applied only when the captured `ActiveAccountLease` remains current. A late completion never edits the new active account. The old family may be invalidated/disposed normally; no private pair is copied into active global state.
- A successful block/unblock promotes the optimistic value to a confirmed overlay carrying the returned URI/CID/rkey and expected state. A stale server seed cannot replace it. Matching indexed state retires the overlay; an account-scoped bounded reconciliation timeout triggers another refresh and diagnostics but does not silently flip the UI.
- Muted-branch reveal lives in the existing `postCommentSectionProvider` family state keyed by account-owned dependencies, root, sort, and focus. It is not persisted in `ProfileRelationship`, disk, or shared cache.
- Mutation start updates currently mounted state immediately. PDS-confirmed success retains the overlay and invalidates server-backed providers; failure restores the exact previous account-local state and shows localized feedback only in the initiating context.

Account-bound invalidation must include relationship repository/notifier, profile caches, timeline and post families, all search/project families, notification list/new-count/badge providers, settings relationship pages, and thread reveal/loaders. Sign-out/device removal does not call any server mute delete API.

## 7. UI, Widgets, Routes, Or User-Facing Surfaces

### API routes

| Method and path | Handler result | Rate/body policy |
|---|---|---|
| `POST /v1/profiles/{handleOrDid}/mutes` | Idempotent mute; updated viewer-relative profile relationship | Write / no body |
| `DELETE /v1/profiles/{handleOrDid}/mutes` | Idempotent unmute; updated viewer-relative profile relationship | Write / no body |
| `POST /v1/profiles/{handleOrDid}/blocks` | PDS-confirmed block; overridden relationship plus URI/CID/rkey; no local projection | Write / no body |
| `DELETE /v1/profiles/{handleOrDid}/blocks` | Delete indexed or PDS-discovered caller-owned record(s); overridden relationship; no local projection | Write / no body |
| `GET /v1/profiles/me/mutes?limit=&cursor=` | Owner-private current-member summaries | Read / no body |
| `GET /v1/profiles/me/blocks?limit=&cursor=` | Caller-owned current-member block summaries | Read / no body |

All six use the existing bearer/device middleware, request ID, camelCase JSON, standard `{error,message,requestId}` envelope, and policy registry. Defaults are 50, maximum 100. Invalid/tampered cursors use the existing cursor error mapping. No owner DID appears in path/query/body.

### Profile UI

- Self profile remains Edit + Settings.
- Eligible ordinary visitor profile keeps Follow and current Mute/Unmute visible. The shared adaptive More menu opens Share, current Block/Unblock, and Report actions as a compact bottom sheet or larger-screen anchored popup.
- Mute/Unmute applies immediately with localized success/failure feedback and no confirmation.
- Block confirmation states that the public block record is visible on the AT Protocol and explains mutual visibility/interaction consequences. Unblock also confirms restoration consequences. Both use destructive semantics where appropriate.
- Muted profile stays full and adds a distinct muted annotation plus Unmute.
- `blocking` and `blockedBy` render distinct annotations. A blocked shell has no bio, metrics, mutuals, tabs, or follow controls. It exposes Report plus only the block action valid for that direction: owned Unblock, or reciprocal Block when only inbound.
- A non-member uses the existing permanent `profile_not_found` experience; the client never renders a special former-member tombstone.

### Content menus and optimistic effects

`PostCard` is the single menu boundary used by feed, project discovery, search, profile posts/comments/projects, root posts, comments, and replies. It receives/derives active viewer and author state and suppresses relationship actions for self-authored content.

- Non-self menu: current Mute/Unmute author, Block/Unblock author, and Report post. Owned content keeps Delete and no self-relationship controls.
- A successful mute removes top-level list/discovery items by that author and their straight repost activity immediately; a direct root remains with muted annotation; other reply branches collapse.
- A successful block removes protected loaded content immediately through the confirmed account overlay. Until Tap converges, stale reload payloads are discarded rather than allowed to repopulate protected content. After convergence, ordinary server policy produces the same result.
- Project posts use the containing `PostCard` menu; `ProjectCard` does not add a duplicate menu.

### Threads and quotes

- Muted branch placeholder copy and reveal control are localized and accessible. One placeholder represents the muted parent and complete descendant branch.
- Reveal directly loads that branch through existing authenticated post/reply repository calls, replaces only the current placeholder, and resets on provider refresh, route disposal/navigation, or account switch.
- Blocked branches are omitted or generic/unrevealable; no button or semantic action offers reveal.
- Muted quote preview has a reveal control that loads the allowed direct quoted post for that card/view only. Blocked quote preview stays generic and cannot navigate. Straight reposts of a blocked author never reach Flutter.

### Settings lists

Settings adds Muted accounts and Blocked accounts beside Followers/Following. A shared `RelationshipListPage(kind)` follows the current `FollowListPage` pagination pattern but uses localized loading, empty, error, Retry, Load more, title/count, relationship annotation, row navigation, and row-level Unmute/Unblock confirmation/action. The API cursor, not list length, controls `hasMore`.

### Routing and deep links

No new public GoRouter path is required for settings list pages; they may use the existing settings navigator push pattern. Profile/post/notification deep links keep their typed routes and always re-fetch server policy. A stale blocked content link lands in the existing generic unavailable destination state, never cached protected content. If route generation is changed during implementation, regenerate `router.g.dart` and cover it in the router suite.

## 8. Error, Loading, Empty, And Edge States

| State / Case | Planned Handling | Requirement IDs | Test IDs |
|---|---|---|---|
| Invalid identifier or self target | Reject before store/PDS; invalid identifier uses current bad-request mapping, self uses stable self-relationship error; no state | FR-001, RULE-004 | UT-002, IT-001 |
| Resolvable non-member or unknown account | Identical `404 profile_not_found`; no hydration, list row, PDS write, report, follow, mention, or relationship action | BR-004, FR-028 | UT-014, IT-001, IT-024, AT-015, REG-005 |
| Duplicate mute/unmute | Upsert/delete converge; return current state; no duplicate row or cross-owner change | FR-002 | IT-002 |
| Block PDS failure/session expiry | No success or confirmed overlay; Flutter rolls back; existing session-expiry behavior runs; no local block projection exists to clean up | FR-003, FR-006, FR-026 | IT-005, IT-009 |
| PDS succeeds while Tap is delayed | Return PDS identity and expected state; keep confirmed Flutter overlay; server policy remains on prior indexed state until Tap | FR-006, FR-025 | IT-005, IT-009, UT-011 |
| Unblock before Tap create | Local miss triggers authenticated PDS list-by-subject fallback; delete exact owned rkey(s); ordered create/delete events eventually converge | FR-003, FR-006 | UT-010, IT-006 |
| Mutual blocks | Removing one owner record leaves inbound block and every symmetric restriction active until final direction is removed | RULE-002 | IT-032 |
| Mute plus block | Block decision wins; mute row remains private and becomes effective again only after all blocks disappear | RULE-003 | UT-001 |
| Join/rejoin backfill / process exits | Profile row remains the sole membership state; ordinary Tap retry or repeated repository tracking resumes indexing; lag/failure is observed | FR-005 | IT-008, IT-025 |
| Joining-owned outbound block | Becomes effective when Tap indexes it; temporary post-membership lag is accepted and tested | FR-005 | IT-008 |
| Current-member-owned retained inbound block | Existing block row targeting the joining DID is not deleted and becomes effective immediately when the profile row appears | FR-005 | IT-007–IT-008, IT-025 |
| Subject membership loss/rejoin | Hide lists/counts/actions without deleting mute/follow/block/content rows; retained indexed state restores with the profile row and other public state converges through Tap | FR-020, FR-028–FR-029 | IT-025, REG-005 |
| Mute owner permanent removal | Profile-delete transaction cascades only owned private mutes; sign-out/device/switch and subject removal do not | FR-029 | UT-016, IT-026 |
| Dense protected pagination | Relationship predicate is inside selection/recursive query; eligible rows fill; cursor derives from eligible last row | NFR-003 | UT-009, IT-011–IT-013, IT-021–IT-022, IT-029 |
| Direct muted content | Return full annotated content/profile; no unmute requirement | FR-012 | AT-003, REG-001 |
| Direct blocked content | Generic unavailable/not-found response, no protected payload or stale cache | FR-014, RULE-006 | IT-014, AT-014, UT-015 |
| Muted reply/quote | Content-free revealable placeholder; explicit direct load only for current view | FR-010–FR-011 | UT-005–UT-006, AT-004–AT-005 |
| Blocked reply/quote/repost/reference | Omit or unrevealable generic placeholder; revalidate every hydration/open | FR-014–FR-015, FR-030 | UT-006, UT-015, IT-015, IT-027, IT-033, AT-008 |
| Directed interaction across block | `interaction_blocked` before PDS or local optimistic write; cleanup/report/reciprocal block still allowed | FR-017–FR-018 | UT-008, IT-016–IT-017, REG-004 |
| Notification existed before relation | Hide list/new-count/badge and cancel unsent work; retain event for possible later view | FR-013, FR-019 | IT-018–IT-020, AT-006 |
| Leased delivery races relation | Transactional cancellation plus exact final eligibility/lease check prevents provider send unless already sent | FR-013, FR-019 | IT-019, GAP-001 |
| Unmute/unblock retained history | Event may reappear once within retention; sent/cancelled push never replays | FR-031 | UT-007, IT-020, AT-006 |
| Flutter duplicate tap/failure | One account/subject/action in flight; optimistic state; exact rollback and localized error in initiating account | FR-025 | UT-011, AT-012 |
| Account switch during mutation/reveal | Captured lease/generation rejects late completion; new account has independent cache and collapsed branch | FR-025, NFR-001 | UT-011, AT-004, AT-012, REG-009 |
| Relationship list loading/empty/error/more | Localized progress, empty copy, Retry, guarded load-more, stable dedupe; failed mutation keeps row and retry | FR-008, FR-024–FR-025 | AT-011–AT-013 |
| Platform hide/takedown overlaps relation | Existing platform decision remains stricter; mute reveal never bypasses it | RULE-003 | UT-001, REG-003 |
| Telemetry captures operation | Emit bounded operation/result/stage/error class only; redact mute target/pair and avoid block target as metric dimension | NFR-001, NFR-006 | UT-012, IT-031 |

## 9. Test Implementation Plan

Tests are added and made green in dependency order. A phase is not complete until its listed regression tests also pass.

| Order | Test IDs | Target | Setup / Fixture | Initial Expected Failure |
|---:|---|---|---|---|
| 1 | IT-035 | `appview/internal/db/mutes_blocks_migration_test.go` | Pre-feature real Postgres; up/down/up; constraint/index and owner/subject fixtures | Migration 000023 and both relationship tables do not exist |
| 2 | IT-002 | `appview/internal/relationships/store_test.go`, API mute store suite | Alice/Bob/Carol active members; repeated owner-scoped mutations | No private mute schema/store or immediate state exists |
| 3 | UT-001, UT-014, UT-016 | Relationship policy, membership, lifecycle unit suites | Full state/precedence and membership/lifecycle matrices | No canonical decision, active-member predicate, or owner/subject distinction exists |
| 4 | UT-002, IT-001, IT-003–IT-004 | Target and relationship handler/list privacy suites | Handles/DIDs/self/nonmember/unknown plus multiple owners/devices | Existing resolvers hydrate nonmembers and no owner-private routes exist |
| 5 | UT-009, IT-011, IT-023 | Relationship request/cursor, list, route-policy suites | 110+ equal-time rows, changed handles, invalid limits/cursors, auth/device matrix | No routes, limits, cursors, or eligible owned lists exist |
| 6 | UT-010, IT-007, IT-036 | Block indexer/dispatcher/compose wiring suites | Generated block records, duplicate URI/pair, ordered update/delete/replay, and absent subjects | Block NSID is neither configured nor indexed |
| 7 | IT-005–IT-006, IT-009, IT-032 | Block handler/reconciliation suites | Recording/listing PDS, no-local-write spy, held ordered Tap, stale refresh, rapid unblock, mutual blocks | No block API/indexer contract or confirmed overlay exists |
| 8 | IT-008 | Membership/backfill convergence suite | Joining-owned outbound and current-member-owned retained inbound fixtures; held Tap and recreated service | Block collection is not tracked/indexed and profile membership does not exercise either direction |
| 9 | IT-025, REG-005 | Rejoin/restart restoration suites | Remove/re-add same DID; hold/restart Tap; retained public/private state | Non-member filtering and rejoin convergence are not implemented consistently |
| 10 | UT-003–UT-004, IT-010 | Profile response/handler suites | Viewer matrix with full metadata and every relationship direction | Profile DTO has follow only and cannot shape a minimal blocked shell |
| 11 | IT-024, AT-015 | Complete membership inventory suites in Go and Flutter | Never-member/former/resolvable/unknown identities across every account target | User-facing hydration/search/follow/report surfaces still expose nonmembers |
| 12 | IT-021–IT-022, REG-002, REG-006 | Profile/follow/search graph suites | Existing follows/mutuals/counts, block directions, Carol as unrelated viewer | Graph rows/counts/search ignore block and active membership |
| 13 | IT-012–IT-013, IT-029–IT-030 | Timeline/search/project/profile-list pagination and query-plan suites | Dense three-page authored/repost fixtures and query instrumentation | Queries apply only moderation and some lack viewer DID |
| 14 | UT-005–UT-006, IT-014–IT-015, IT-027, IT-033 | Post/thread/quote/reference response/store suites | Muted branches, cached quotes, straight reposts, third-party blocked pair | Existing response types cannot represent relationship placeholders and hydration leaks payloads |
| 15 | UT-008, IT-016–IT-017, REG-001, REG-004 | Authorization and affected write-handler suites | Recording PDS/local counters across both directions and cleanup ownership | New writes are allowed regardless of blocks; cleanup exception matrix is absent |
| 16 | IT-020, IT-034, REG-003 | Restoration/moderation/aggregate suites | Pre-existing public records, mute/block/remove/rejoin, unrelated viewer | Policy restoration and non-destructive aggregate guarantees are unproved |
| 17 | UT-007, IT-018 | Notification ingestion/eligibility suites | Every category under mute and both block directions | Relationship state does not affect event/outbox creation |
| 18 | IT-019–IT-020, AT-006 | Durable list/newness/badge/push race suites plus Flutter notification page | Pending/retry/leased/sending/sent/cancelled events and retained history | List/newness and final push checks ignore relationships |
| 19 | UT-012, IT-031 | Go/Flutter observability and redaction sentinel suites | Unique target/pair/rkey/URI sentinels across failures and routine operations | New operations have no bounded metrics/redaction coverage |
| 20 | UT-011–UT-013, REG-009 | Flutter relationship provider, account boundary, localization/semantics suites | Fixed account repositories, captured generations, failure/duplicate/switch fixtures, every locale key | No account-keyed relationship state or new localized semantics exist |
| 21 | AT-001–AT-003, AT-009 | Profile action/state/direct-mute widget suites | Ordinary, muted, blocking, blockedBy, self and nonmember profiles | Profile actions lack More relationship controls and blocked shell |
| 22 | AT-004–AT-005, AT-007–AT-008, AT-014 | Thread, quote, feed/search, and stale-route widget/provider suites | Muted branch reveal/reset; muted/blocked quotes; blocked repost/direct link | Client models/widgets cannot collapse/reveal or discard blocked payloads |
| 23 | AT-010 | Shared post/project/comment/reply menu suites | Every non-self `PostCard` call-site variant plus owned content | Menus expose only Report/Delete and project coverage is not centralized |
| 24 | AT-011–AT-013 | Settings relationship list and accessibility suites | Multi-page lists, empty/error/retry, row reversal, account switch | Settings has only follower/following lists and hard-coded English in nearby page |
| 25 | REG-007–REG-008 | Raw API/middleware and older-client regression suites | Direct route calls without Flutter filtering and full route policy inventory | Server paths are not yet proven independently of client behavior |
| 26 | MAN-001 | Real phone/tablet accessibility/localization smoke | VoiceOver/TalkBack/keyboard, phone/large layout, non-default locale | Host widget semantics cannot prove platform navigation quality |
| 27 | MAN-002 | Local PDS/Tap compatible-client smoke | Craftsky create/delete and external create/delete with repository inspection | Automated fakes cannot prove live interoperability |

Focused commands by phase:

```text
# First red/green schema step, from appview/ (compose Postgres required)
TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable \
  go test ./internal/db -run TestMutesBlocksMigration -count=1

# Relationship, API, index, membership/backfill, notification, and push work, from appview/
TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable \
  go test ./internal/relationships ./internal/api ./internal/index ./internal/notifications ./internal/push ./internal/app ./internal/routes ./internal/db -count=1

# Flutter relationship/data/provider/widget work, from app/
flutter test test/profile test/feed test/projects test/settings test/notifications test/search test/router test/auth

# Regenerate source-derived files, from app/
dart run build_runner build --delete-conflicting-outputs
flutter gen-l10n

# Full repository gates, from repository root
just test
just app-test
just app-analyze
git diff --check
```

`just test` remains the canonical Go gate and requires the compose Postgres (`just dev-d`) on host port 5433. Manual checks run only after all automated gates pass.

## 10. Sequencing And Guardrails

- First TDD step: write IT-035 and IT-002 together, run IT-035 first, and green only the reversible schema/index/ownership contract. Then green owner-scoped idempotent mute storage as the first feature behavior. Do not begin handlers or Flutter in that first green step.
- Dependency order:
  1. Schema, store primitives, pure policy, membership, and lifecycle.
  2. Block indexer/configuration, PDS-confirmed handler, authenticated PDS list fallback, and ordered Tap reconciliation.
  3. Profile-row membership, ordinary Tap admin repository tracking, and both join/rejoin convergence fixtures.
  4. API routes, profile state/lists, and uniform membership boundary.
  5. Query-time read filtering, response/reference shaping, dense pagination, and query plans.
  6. Directed-write authorization and cleanup/report exceptions.
  7. Notification creation/list/newness and push cancellation/final eligibility.
  8. Flutter models/data/account-keyed state, profile/content UI, settings, notifications, localization, and accessibility.
  9. Full regression, privacy, performance, manual accessibility, and live interoperability gates.
- Guardrails:
  - Never add `social.craftsky.*` mute/block records or edit `lexicon/`; use Indigo's maintained generated `app.bsky.graph.block` type.
  - Never add a second membership predicate: `craftsky_profiles EXISTS` is complete membership state.
  - Never write `atproto_blocks` from an API handler; Tap is its sole writer.
  - Never let a stale pre-Tap response replace a PDS-confirmed Flutter overlay.
  - Never use membership loss to delete public block/follow/content/interaction rows or another owner's private mute.
  - Never make a private mute pair observable through another viewer's response, shared client cache, log text, trace attribute, or metric label.
  - Never perform per-item relationship queries. Use set-based SQL or one batch state load.
  - Never filter after a page limit/cursor. The database/recursive selection must choose eligible rows.
  - Never serialize protected block payload and hope Flutter hides it. Blocked direct/indirect responses are data-minimal.
  - Never claim Flutter enforces raw API or other-client behavior during Tap lag. Raw API and older-client regressions must pass against indexed state after convergence.
  - Never mutate canonical records/aggregates to implement visibility. Relationship removal restores otherwise-eligible views.
  - Preserve exact lease fencing, PDS session expiry, request envelope, device authentication, and account-generation semantics.
  - Preserve unrelated worktree changes and do not commit/push without explicit user authority.
- Out of scope:
  - Mute words/tags/threads, temporary controls, moderation lists, DMs, anonymous reads, hosted Bluesky private-mute synchronization, lexicon changes, destructive public-record cleanup, numerical latency SLA, and retraction of provider-accepted push.

## 11. Risks And Open Questions

| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Non-blocking | A confirmed Flutter overlay needs a retirement policy if Tap is delayed or an event is dropped. | Retiring too early causes visible reversal; retaining forever can mask a later external change. | Retire on matching indexed state. Until then, retain it for the account/session and perform bounded reconciliation refreshes with diagnostics; define the operational timeout in provider tests without changing the UI silently. |
| CPQ-002 | Non-blocking | Tap's `/repos/add` response confirms request acceptance, not that AppView consumed every event. | Join/rejoin can expose the profile before joining-owned historical blocks are indexed. | This is the approved contract. Use the call only to request ordinary tracking, test retained-inbound versus joining-owned convergence separately, and observe lag. |
| CPQ-003 | Non-blocking | The current Tap consumer poison-pill policy can eventually ack a repeatedly failing block event. | Indexed policy can remain behind the canonical PDS state. | Emit bounded failure/lag signals and make repeated OAuth/join repository tracking idempotent. Do not hide the condition behind activation state; a stronger recovery worker would require a separate reviewed change. |
| CPQ-004 | Non-blocking | External clients can create duplicate block records for one subject. | Deleting only one could leave the pair blocked; creating during local lag could add another duplicate. | On local miss or ambiguous pair, list authenticated PDS records, sort matching records by URI/rkey, delete every caller-owned matching rkey idempotently, and never create when any match exists. Assert in UT-010/IT-006–IT-007. |
| CPQ-005 | Non-blocking | Requirements specify bounded/indexed checks but no numerical performance SLA. | Query plans can be gated, but latency has no pass/fail threshold. | Capture baseline `EXPLAIN` and store-call counts in IT-030. Propose a separate reviewed SLA only if representative fixtures show material regression. |
| CPQ-006 | Non-blocking | Existing nearby Settings strings are hard-coded English. | New relationship pages could accidentally repeat the pattern and fail NFR-005. | Localize every new string and include the touched Settings labels if the shared page is edited; UT-013 remains the completeness gate. |

Blocking open questions: None. Any change to mute privacy, non-member eligibility, direct muted access, minimal blocked profiles, public PDS blocks, notification restoration, or owner/subject lifecycle is a requirements change and must return to document review.

## 12. Handoff To TDD Builder

- Coding plan: `04-coding-plan.md`
- TDD execution artifact to create during implementation: `05-implementation-plan.md`
- Start with test: IT-035 in `appview/internal/db/mutes_blocks_migration_test.go`; write IT-002 beside it but do not mix handler behavior into the first green step.
- First focused command, from `appview/`:

```text
TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable \
  go test ./internal/db -run TestMutesBlocksMigration -count=1
```

- First implementation target: `appview/migrations/000023_mutes_blocks.up.sql` and `.down.sql`, followed by the minimum owner-scoped mute store.
- Mandatory early consistency gate: IT-005–IT-009 and IT-025 must prove no synchronous block projection, stale-refresh-resistant Flutter state, rapid pre-index unblock, and both join/rejoin block-owner directions before relationship consistency work is considered complete.
- Mandatory pre-UI gate: IT-028 inventory, IT-029 dense pagination, IT-030 query plans, IT-031 telemetry privacy, and raw-API REG-008 must pass before Flutter is treated as more than presentation.
- Final automated gates: `just test`, `just app-test`, `just app-analyze`, and `git diff --check`.
- Final manual gates: MAN-001 accessibility/localization and MAN-002 compatible-client/local-PDS interoperability.
- Source implementation remains unapproved until the user explicitly chooses `implement-tdd`.
