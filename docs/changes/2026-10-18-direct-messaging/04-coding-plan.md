# Coding Plan: Direct Messaging

## Combined initial conversation load: 2026-10-21

Add GET /v1/chat/conversations/{id}/details with no query/body, returning
{conversation: <existing Conversation>, messages: <existing MessagePage>} and
existing no-store, read/no-activity/current-member policy. Details uses one
withRPC union for getStatus/getConvo/getMessages, captures the connection
generation, validates membership/CraftSky profiles once, computes canSend with
existing block checks, then reuses the extracted history-page implementation.
Keep standalone History's complete boundary; do not nest public service calls.

Add a redacted ChatConversationDetails value and ChatRepository.details. Initial
controller load starts details and local draft futures together, attaches both
to Future.wait(eagerError: true), preserves original typed failures, then
publishes only after both and the existing epoch/active-operation checks. Refresh
and pagination retain their current endpoints; no fallback to duplicate calls.
Update affected repository fakes/contracts. Validate exact upstream call order,
negative authorization/private-boundary cases, HTTP route policy/wire shape,
client response identity and concurrent draft restoration. Rebuild only the local
AppView after passing tests; apply Flutter changes, preserving local/live limits.

## Silent background refresh presentation: 2026-10-21

Remove the inherited background-loading presentation signal and render plain
inbox/conversation titles. Retain the title progress helper only for explicit
settings mutations and recipient selection/start actions. Keep the activity
scope's coordinator ownership, lifecycle/view registration, warnings and retry
messages unchanged. Do not alter controller loading state or scheduling.

## Conversation header profile action: 2026-10-21

Wrap the loaded peer avatar/identity in an accessible InkWell using the existing
showUserProfileCard helper and current active-account operation guard. Leave the
background loading indicator outside the hit target. Preserve the route stack
and draft; reuse the existing profile route's compact presentation and expansion.
Verify the typed peer DID/presentation extra and Back using production chat routes.

## Request-action implementation: 2026-10-21

Reuse the existing conversation controller leave path for an explicit Ignore
button beside Accept request. Add a local in-flight flag solely for the Ignore
button's progress; keep controller busy/capability/owner guards and check route
and widget lifetime before closing on success. Use existing themed secondary
button styling and responsive button spacing. Remove the request explainer from
rendering; localize Ignore. No API, backend, permissions, persistence or automatic
remote action change. The More-menu Leave path keeps its confirmation dialog.

## Loading-style polish: 2026-10-21

Reuse `CraftskySkeletonList` and `AccountRowSkeleton` for initial chat/request
lists and recipient lists. Add a local message-bubble skeleton using the shared
wrapper/effect and existing spacing/radius tokens. Pass the appropriate body
through the status gate and content-loading Scaffold, so both acquisition
stages share the same visual treatment. Keep background refresh, pagination,
actions, permission/setup state and all controller/provider behavior unchanged.

## Approved runtime/cache design: 2026-10-21

Use an authenticated-shell consumer to own a manual auto-disposed runtime
provider. Watch the active lease, status/repository and, when eligible, unread
counts, sync coordinator and an account cache. Keep existing view registration
and polling/backoff. Refresh lightweight state on app resume. Retain successful
inbox providers through cache-owned KeepAliveLinks; retain idle conversation
providers with a three-entry/two-minute policy. Active listeners suspend idle
eviction. Providers only read the cache owner, avoiding retention cycles. Cache
disposal closes links; existing lease/status/generation dependencies invalidate
controllers at security boundaries. Search/settings/report lifetimes stay local.
No persistence, backend/API/permission change, automatic mutation or deployment.

## Feedback correction design: 2026-10-21

UT-017 / NFR-006, FR-015: remove the bespoke CraftskyMessageBanner and above-Scaffold banner layout. A pass-through ChatMessageEffect schedules existing context.showInfo/showWarning/showError after layout; it adds no renderer, deduplicates unchanged state, checks route/mount lifetime and guards retry callbacks. Use the existing AppMessenger unchanged. Authorization's guarded async handler calls it directly. Share routine background state with ChatAppBarTitle through a layout-free inherited value, suppressing the title loader on centered initial-load screens. Keep necessary persistent state explanations as Text beside existing controls. No server, API, scope, migration or remote-operation change.

## Approved revision design: combined OAuth/setup, 2026-10-21

Inputs: latest approved requirements and acceptance revision. Explicit local high-risk approval is the maintainer's “agreed on all accounts (apart from the alternative). Implement this.” No new migration, dependency, route, live action or production mutation.

1. AUTH-001: keep Config.Scopes/DefaultScopes as the required base capability; use advertisedConsumerScopes(base) for login and registration PAR requests only. Validate ordinary callback/refresh against base. Keep operator and account-deletion auth purpose-separated.
2. AUTH-002: keep authenticated, same-owner capability-recovery OAuth and its existing session/device/generation fences, but confirmation must not set enabled, advance messaging generation, overwrite push or resume a disabled connection. Resume retained synchronization only when the account was already enabled. A capable session is independent of explicit account enablement.
3. SETUP-001: extend the existing settings controller with confirmed connection status and reversible setEnabled. Off mode loads no shared-policy/push preferences; on mode loads existing choices. Use existing SetConnection capability/eligibility checks; publish confirmed status to evict private providers on disable. Retain draft behavior and generation fencing. Failures preserve confirmed state and expose themed retry/recovery.
4. ROUTE-001: reuse one Direct messages settings surface from the disabled content gate and settings route. The gate retains setup after enable until Open messages; later enabled visits use their existing inbox. Keep account lease keys, cold routes, loading patterns and guarded navigation. Replace ordinary consent/button with concise disclosures + Material enable switch. Push and shared-audience controls appear only while enabled. Recovery remains an explicit action when authority is missing.
5. Run focused Go tests against the local disposable PostgreSQL schema and nearby auth/chat regressions; run Flutter chat/auth/routing tests and analysis. Update implementation evidence and read it back. Production/live/device proof remains owner-run.

## Approved revision: profile actions, 2026-10-20

UT-015 moves the existing mute/unmute callback and state-dependent label/icon from the visitor profile action row and collapsed-header control into a shared ProfileMoreActionsButton using CraftskyContextMenuButton. Preserve canToggleMute, busy, block/report/share and compact/wide menu behavior. UT-016 keeps ProfileMessageButton actionable except explicit caller busy disabling; use the existing account operation guard and context.showInfo AppMessenger entry point to explain recipient denial or generic availability failure. No backend API, permission, relationship mutation, create/send behavior or dependency changes.


## Approved follow-up design: 2026-10-20

The explicit maintainer revision at the top of `01-requirements.md` and its IT-021 / UT-012 / IT-022 / UT-013 cases supersede conflicting earlier policy/design below.

1. Add a sqlc query against existing `craftsky_profiles` and `owner_lifecycles` to select indexed, non-inactive participants. Batch per list page; reuse a single server eligibility boundary for direct conversation access, operations and background/delivery paths. Do not require peers to have local sessions; do not add a migration or modify upstream records.
2. Derive existing count response fields from authoritative accepted/request lists filtered by the same eligibility. Accepted counts require unread, unmuted, unblocked and upstream-active peers; requests count all pending eligible requests. Deduplicate and cap at 100. Bound each count bucket to 20 pages of 100 and the request to 20 seconds; fail unavailable on incomplete enumeration rather than return partial totals. Preserve provider rate-limit errors and existing API error handling. Drop the unused global `getUnreadCounts` permission from new consent and read readiness requirements. No transcript or new durable count cache.
3. Use existing `/messages/requests` typed route as a dedicated request screen, replacing the tab presentation. Add its badged AppBar action on accepted chats and the existing branded New message FAB. Use `YarnRefreshScrollView`, listen to existing controller loading state for small title indicators, and preserve loaded content/scroll identity during refresh. Hide the composer's visible label, add the localized `Message` hint, and align the input and send control.
4. Execute IT-021, UT-012, IT-022, then UT-013 red-green loops. Preserve all existing ownership, capability, operation-ledger, request lifetime and private diagnostics boundaries. Run affected Go suites under race plus focused Flutter navigation/widget/controller regressions and analysis. No push/deploy/live messaging or live reporting.

Baseline exception for notification suppression: retain existing private opaque conversation/message dedupe IDs and request-lifetime flags for pre-enable direct state, including hidden peers. Store no peer identity or message body. These operational tombstones never populate consumer views/counts and cannot authorize delivery. Without them, a peer who later creates a CraftSky profile could make old history or an existing baseline request appear newly notifiable. Recovery/incremental eligibility and both dispatch checks remain profile-filtered.

## 1. Inputs

- Requirements: `01-requirements.md`, including the owner-authorized AC-002/AC-010 clarifications after review. Sections 8, 12 and 13 remain authoritative; SIM-01–SIM-04 supersede older discussion.
- Tests: `02-acceptance-tests.md`, revised to address DR-001/DR-002: 29 acceptance scenarios, 11 unit cases, 20 integration cases, 6 regressions and 4 manual checks. Every scenario now assigns verification to specific suites or controlled live checks.
- Document review: `03-document-review.md`, verdict Approved with notes. It is the pre-revision record; DR-003/DR-004 are addressed by the persistence/recovery and live-evidence designs below.
- Risk: High. This is an implementation design contract, not implementation authorization or evidence of a working integration. No application/test/dependency/migration changes or live actions occur in this stage.
- Read-only repository inspection: architecture reference, API architecture and mandatory logging guide; auth scope/config/flow/coordinator/background/session-lifecycle code; route catalogue/policy; federated HTTP composition; push sender/payload/dispatcher; private account cleanup; Flutter account leases/guards/Dio, notification destinations/providers, profile search, draft/secure-storage seams and typed router.
- Existing paths supersede stale discovery labels: OAuth handlers are in `appview/internal/auth/handlers_oauth.go` and flow logic in `oauth_flow.go`; hardened client composition is `appview/internal/app/federated.go`. There is no `auth/federated.go` extension point. The migration directory currently ends at 000101; choose the next free prefix at implementation time rather than reserving it now.

## 2. Implementation Strategy

Add a narrow `appview/internal/chat` feature with authenticated PDS-proxied adapters, private operational storage and one account-scoped synchronization worker. Flutter adds an account-keyed `chat/` feature using existing Riverpod code generation, account-specific Dio, typed routes, active-account leases and the selected `flutter_chat_ui` presentation adapter. Bluesky remains authoritative for messages, membership, request/read/mute state and incoming policy. No message archive, new transport, public record schema or Tap indexer is introduced.

Separate base OAuth requirements from optional advertised chat permissions. Bind additional-consent attempts to the current owner/session/device and server-selected permission bundle. After successful same-owner consent, use the existing session/handoff machinery to install a capable session; account-wide enablement permits each device to use its own sufficient session, while a base-only device sees authorization-needed. Workers may select another usable capable session belonging to that same owner. Do not hand a different device's credentials to Flutter or invalidate base-only sessions for missing optional permissions.

Reuse push provider transport, subscription ownership, bounded retries and lease conventions. Give DM notifications their own logical source and eligibility path instead of inserting private messages into social `notification_events` or inheriting `actor_mutes`. Extract the common provider-ready payload/delivery mechanics behind narrow internal interfaces, preserving the existing social path and its tests. This is a focused delivery extension, not a replacement notification framework.

For active views, use a cheap owner-scoped revision/activity endpoint and one shared backend account poller. Flutter refreshes affected inbox/history/count snapshots on revision changes, foreground, reconnect or explicit action. Repeated device requests do not each invoke getLog. Store no message content merely to fan out updates; accept authenticated on-demand upstream reads for actual snapshot hydration. Retain loaded UI data only inside the current owner lease and clear it on revocation/switch/disconnect as appropriate.

## 3. Affected Areas

| Area | Existing Pattern | Planned Change | Requirement IDs | Test IDs |
|---|---|---|---|---|
| Optional authority | Config scopes, semantic repo grants, fenced OAuth coordinator | Base/advertised/requested/granted separation; semantic RPC grants; same-owner capability consent | BR-002, FR-001, FR-016, RULE-004, NFR-001 | AT-002, AT-014, AT-019, UT-001, IT-005, IT-015, REG-001 |
| Chat operations/API | Hardened federated HTTP, stdlib handlers and route catalogue | Fixed chat/report/declaration adapters and consumer routes, owner/membership checks, bounded DTOs | BR-001, BR-003, FR-002, FR-003, FR-004, FR-005, FR-007, FR-008, FR-009, FR-010, FR-011, FR-018, RULE-001, RULE-002, NFR-003 | AT-001, AT-003–AT-011, AT-018, AT-021, AT-027, UT-002–UT-005, IT-001–IT-004, IT-007, IT-008, IT-014, IT-019 |
| Durable operations/sync | PostgreSQL, lifecycle fences and background session selection | Single-attempt sends/reports; atomic progress/outbox; baseline, request lifetime and recovery state | FR-006, FR-012, FR-013, FR-015, FR-016, NFR-002, NFR-004, NFR-005 | AT-007, AT-012–AT-014, AT-017, AT-022, AT-029, UT-006, IT-006, IT-009, IT-010, IT-012, IT-020 |
| DM delivery | Existing subscriptions, sender, leases and current eligibility checks | Private DM destination, source-specific queue checks and sender-name-only payloads | FR-013, FR-014, RULE-001, RULE-003, RULE-005, NFR-001, NFR-002 | AT-013, AT-015, AT-016, UT-007, IT-010, IT-011, IT-013, REG-002, REG-003 |
| Account-scoped Flutter/UI | Account Dio/leases, generated providers and GoRouter | Inbox/requests, explicit consent/actions, chat package adapters, local drafts, safe routing/previews | BR-001, BR-002, BR-003, FR-001–FR-011, FR-014–FR-019, RULE-002, RULE-005, NFR-006, NFR-008 | AT-001–AT-011, AT-014–AT-017, AT-023, AT-025–AT-028, UT-003, UT-004, UT-008–UT-011, IT-013, IT-016, IT-017 |
| Privacy/cleanup/evidence | Source-selected diagnostics, private deletion cleanup, mobile journey harness | DM sink canaries, minimal retention, logout/deletion cleanup, real service/device validation | FR-016, FR-019, NFR-001, NFR-002, NFR-006, NFR-007, NFR-008 | AT-020, AT-023, AT-024, AT-026, AT-028, IT-018, REG-004–REG-006, MAN-001–MAN-004 |

## 4. Files And Modules

Paths marked Create are proposed. Keep test names/IDs from the revised test specification; adjust physical grouping when a coherent existing seam fits better.

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `appview/internal/auth/scopes.go`, `effective_scopes.go`, `config.go` | Change | Explicit optional bundle and semantic method/audience evaluation without widening base validation | BR-002, RULE-004, NFR-001 | UT-001, IT-015, REG-001 |
| `appview/internal/auth/oauth_flow.go`, `handlers_oauth.go`, auth-request persistence and handoff seams | Change | Chat-consent purpose with owner/device/generation binding, requested-scope capture and same-owner callback finalization | FR-001, FR-016, RULE-004 | AT-002, IT-005, IT-015 |
| `appview/internal/auth/session_coordinator.go`, `background_session_selector.go`, `craftsky_session.go`; middleware auth seam | Change | Required-operation capabilities, capable candidate selection and no-activity session validation for automatic polling | FR-016, RULE-004, NFR-001, NFR-005 | IT-005, IT-015, IT-020, REG-001 |
| `appview/internal/app/config.go`, configuration fixtures and local environment documentation | Change during implementation only | Redacted dedicated chat request-digest secret/key ID, validated resource defaults and injected worker options; no credentials committed | FR-006, NFR-002, NFR-004, NFR-005 | IT-006, IT-018, IT-020 |
| `appview/internal/chat/{types,capability,transport,service,validation,report,declaration}.go` | Create | Narrow chat domain and fixed authenticated adapters; tolerant upstream views, bounded input and CAS declaration merge | FR-001–FR-005, FR-007–FR-011, FR-018, RULE-001, RULE-002, NFR-001, NFR-003 | UT-001–UT-005, IT-001–IT-004, IT-007, IT-008, IT-014, IT-019 |
| `appview/internal/chat/{store,operations,sync,scheduler,notification_policy}.go` | Create | Operational metadata, single-attempt operation receipts, event reconciliation, leases and request/catch-up state | FR-006, FR-012, FR-013, FR-015, NFR-002, NFR-004, NFR-005 | IT-006, IT-009, IT-010, IT-012, IT-020 |
| `appview/migrations/<next>_direct_messaging.{up,down}.sql`; `appview/queries/chat.sql` | Create | Private tables/indexes and named SQL for state, constraints, claims and cleanup | FR-006, FR-012–FR-014, FR-016, NFR-002 | IT-005, IT-006, IT-010–IT-012, IT-018 |
| `appview/sqlc.chat.yaml`, `appview/internal/chatdb/`, `scripts/chat-db-generate`, `justfile` | Create / Change during implementation | Feature-scoped PostgreSQL/pgx query generation, reproducible generator pin and drift check; use actual migration schema inputs | FR-006, FR-012, FR-013, NFR-002 | IT-006, IT-010, IT-012, IT-018 |
| `appview/internal/api/chat.go`, `appview/internal/routes/routes_chat.go` | Create | Consumer DTO/handler bundle with narrow service dependency | FR-001–FR-016, FR-018, NFR-001, NFR-003 | IT-003–IT-009, IT-013, IT-019 |
| `appview/internal/routes/{policy,dependencies,routes}.go`; route manifest/diagnostic classification | Change | Catalogue entries, auth/activity policy and private-segment classification | NFR-001, NFR-002, NFR-003 | IT-013, IT-018, IT-019 |
| `appview/internal/app/{deps,federated}.go`, composition helpers; `appview/cmd/appview` workers | Change | Build narrow adapters, inject clock/store/session runner, start/stop sync and DM delivery under application context | FR-012, FR-013, NFR-004, NFR-005 | IT-009–IT-012, IT-020 |
| `appview/internal/push/{sender,payload,firebase_sender,dispatcher}.go`; `chat_dispatcher.go` and shared delivery helpers | Change / Create | Share provider payload validation/send/retry and lease finalization; retain separate social/DM source eligibility | FR-013, FR-014, RULE-003, RULE-005 | UT-007, IT-010, IT-011, REG-002 |
| `appview/internal/accountdeletion/private_cleanup.go`; auth logout/subscription cleanup | Change | Cancel/purge owner DM state; ordinary logout removes only device subscription/activity | FR-016, NFR-002 | IT-005, IT-018, REG-006 |
| `app/lib/chat/models/`, `data/chat_repository.dart`, `data/api_chat_repository.dart` | Create | Typed DTOs and narrow account-authenticated repositories; no raw upstream payload/domain models in UI logging | FR-001–FR-011, FR-015, FR-018, NFR-001, NFR-003 | UT-002–UT-005, IT-019 |
| `app/lib/chat/providers/`, `services/chat_sync_coordinator.dart` | Create | Account families, guarded completions, revision refresh and explicit message operations | FR-001, FR-002, FR-003, FR-005–FR-009, FR-014–FR-018 | UT-003, UT-004, UT-006, IT-007, IT-013, IT-016 |
| `app/lib/chat/data/chat_draft_store.dart`, `providers/chat_draft_provider.dart` | Create | Private local draft/index keys, serialized writes and removal-generation fences | FR-005, FR-016, FR-019, NFR-002 | UT-009, IT-017, IT-018 |
| `app/lib/chat/pages/`, `widgets/`, `adapters/flutter_chat_adapter.dart` | Create | Enablement, inbox/requests/new message/settings/conversation and report/leave/uncertain dialogs; compact cards | BR-001, BR-002, BR-003, FR-001–FR-011, FR-017–FR-019, NFR-006, NFR-008 | AT-001–AT-011, AT-023, AT-025–AT-028, UT-008–UT-011 |
| `app/lib/router/{router,app_shell,app_shell_drawer}.dart`, profile action caller(s), settings page | Change | Inbox entry, typed routes, profile availability and messaging settings | FR-004, FR-014, FR-018, NFR-006 | IT-003, IT-013, UT-010, REG-003, REG-004 |
| `app/lib/notifications` destination/runtime/inference/navigation; auth account-boundary cleanup | Change | Opaque DM notification resolution after exact owner activation and lease check | FR-014, FR-016, NFR-001 | IT-013, IT-017, REG-003 |
| `app/pubspec.yaml`, `app/pubspec.lock`; localization sources and generated files | Change during implementation only | Selected chat UI/core dependency and generated localized copy/providers/mappers/router | BR-003, NFR-006, NFR-008 | UT-010, UT-011, MAN-003, MAN-004 |
| Proposed Go and Flutter test paths in `02-acceptance-tests.md`; existing mobile critical journeys | Create / Change during TDD only | Preserve scenario-specific test allocation and existing regression coverage | All 35 requirements | AT-001–AT-029, UT-001–UT-011, IT-001–IT-020, REG-001–REG-006 |
| `docs/changes/2026-10-18-direct-messaging/06-validation-evidence.md` | Create during implementation | Safe functional/live/package/platform evidence and unresolved cases | NFR-007, NFR-008 | MAN-001–MAN-004 |

The checked-in `appview/queries/` directory is presently empty and no sqlc config/recipe was found. Follow the repository's SQL/sqlc convention: introduce a minimal feature-scoped sqlc configuration/generation recipe and generated query package during the storage slice, with chat queries against the actual migration schema inputs. Name the config `appview/sqlc.chat.yaml`, generated package `internal/chatdb`, and checked-in root command `just chat-db-generate` backed by `scripts/chat-db-generate`; pin the generator version and make regeneration/check reproducible. Do not claim a nonexistent `just sqlc` command or retrofit unrelated stores. Tests must exercise migrated tables, not a second hand-maintained schema that bypasses production constraints.

## 5. Services, Interfaces, And Data Flow

### 5.1 Authority and transport

`ChatService` depends on a chat `Store`, a capability-aware `SessionRunner`, `UpstreamChat`, `ReportAdapter`, `DeclarationAdapter`, current block/lifecycle readers and an injected clock. `UpstreamChat` exposes only the 15 required methods listed in requirements §24. The report adapter exposes only createReport to the fixed moderation audience; the declaration adapter only authoritative read and conditional create/update of `chat.bsky.actor.declaration/self`. No generic RPC/URL/audience input is accepted.

Use existing `OAuthSessionCoordinator.WithActiveSession` for reads and `WithActiveEffectSession` for remote effects that require participant lifecycle fencing. Respect its documented lock order; never nest ordinary session acquisition inside an existing effect fence. Validate capability again after refresh, before the exact operation. A missing optional capability is a chat-level recovery result, not a terminal base-session error. Base scope loss still follows existing session policy. Preserve original typed causes privately and never persist provider prose.

Extend the background selector with `SelectForCapability(owner, required)` over existing lifecycle/expiry-eligible candidates. Evaluate effective grants semantically; do not select the newest base-only candidate and give up while another capable same-owner session exists. Do not call foreground bearer lookup or touch idle expiry while selecting/polling. Client metadata advertises the explicit union. Ordinary login and registration request this union but validate ordinary session/callback/refresh access against the independent base scopes. Explicit capability recovery requests/records the same narrow bundle. Operator/deletion OAuth clients remain purpose-separated.

A chat-consent auth request stores purpose, owner DID, initiating child/session/device, lifecycle/auth epoch, messaging generation, requested permission profile and existing callback/handoff constraints. Callback DID must match. Cancellation/partial consent preserves base access and does not enable a partial ready feature. Finalization atomically installs the new capable parent/child through existing lifecycle/handoff patterns only if the initiating generation remains current. It leaves account enablement and push preference untouched, retaining the sync cursor and resuming only already-enabled accounts. Disconnect racing recovery must win; a late callback cannot re-enable it. No arbitrary return URL or forced switch to a different identity.

Use Indigo's authenticated API client through `internal/app/federated.go` and the hardened PDS JSON transport. Apply fixed service proxy routing separately for chat and moderation. Keep the existing dependency unless a reviewed adapter cannot represent needed responses; implement small typed response views for missing getUnreadCounts support and forward-compatible getLog envelopes. Do not modify generated CraftSky record structs or `lexicon/`.

The [AT Protocol permission specification](https://atproto.com/specs/permission) was checked for the permission boundary during planning. Exact deployed support still belongs to MAN-001; the published spec is not a successful granular grant.

Illustrative signatures only:

```text
SessionRunner.WithCapability(ctx, owner syntax.DID, sessionID, required, operation)
UpstreamChat.ListAccepted(ctx, session, cursor, limit) -> DirectPage
UpstreamChat.SendTextOnce(ctx, session, convoID, text) -> MessageResult
ChatService.Send(ctx, authority, operationID, convoID, text) -> OperationOutcome
ChatService.Report(ctx, authority, operationID, messageID, reason) -> OperationOutcome
Store.CommitPoll(ctx, expectedGeneration, expectedCursorVersion, events, nextCursor)
ChatDraftStore.load/save/discard/removeAccount(account, conversation, localGeneration)
```

DIDs use `syntax.DID`; handles/NSIDs use existing typed wrappers. Chat conversation/message IDs and cursors are bounded opaque domain types, not AT-URIs/CIDs. Parse once at the HTTP/upstream boundary. Neither client-provided sender DID nor client-provided messageRef establishes report authority; derive the sender/reference from a current accessible direct-message view. Reject cross-owner IDs and unsupported group variants before any effect.

### 5.2 Consumer API contract

Register these feature routes in the existing catalogue and diagnostic manifest, using normal consumer bearer/device authentication, existing current-member/eligibility policy and appropriate read/write/auth rate classes. Preserve existing suspension policy; do not use chat as a new recovery/suspension bypass. All keys camelCase; errors use `{error,message,requestId}` with safe static copy. Missing optional chat authority yields a classified chat 403 rather than a session-invalidating 401. Actual invalid CraftSky credentials retain existing 401 handling.

| Method and path | Contract and effect | Requirement / AC / Tests |
|---|---|---|
| `GET /v1/chat/status` | Account enablement, messaging `generation`, capability booleans, eligibility/sync state, last successful sync and safe recovery code; no tokens, cursor or provider prose | FR-001, AC-002/AC-017, UT-001/UT-006 |
| `POST /v1/chat/authorization` | Start bound additional-consent attempt through existing handoff flow; fixed scope profile only; optional `{pushEnabled}` initial choice; callback validates profile and same owner | BR-002, FR-016, RULE-004, AC-002/AC-014/AC-019, IT-005/IT-015 |
| `PUT /v1/chat/connection` | Full replacement `{enabled}`; false disconnects atomically, true only with sufficient current authority and initializes/resumes account state; never changes declaration | FR-016, AC-014/AC-028, IT-005/IT-017 |
| `GET /v1/chat/conversations?status=accepted\|request&cursor=&limit=` | One explicit bucket per request; accepted uses direct/accepted listing, request uses dedicated upstream union filtering | FR-002/FR-008, AC-001/AC-003, IT-001 |
| `GET /v1/chat/availability?recipientDid=` | Resolve availability and optional canonical existing direct conversation; zero creation | FR-004/FR-018, AC-005/AC-027, IT-003 |
| `POST /v1/chat/conversations` | `{recipientDid}` exactly one other DID; explicit canonical get/create with fresh availability/block checks | FR-004, AC-005, IT-003/IT-014 |
| `GET /v1/chat/conversations/{id}` | Current membership/status/mute/unread/participants; sanitized direct summary | FR-003/FR-009, AC-004/AC-009, IT-002/IT-007 |
| `GET /v1/chat/conversations/{id}/messages?cursor=&limit=` | `{items,cursor?}` chronological model, safe deleted/unsupported placeholders; valid cursor survives filtered-short pages | FR-003, AC-004, UT-003/IT-002 |
| `POST /v1/chat/conversations/{id}/messages` | `{operationId,text}`; success 200 with confirmed authoritative message, 202 for pending/uncertain receipt; definitive failures use standard error envelope; receipt remains queryable | FR-005/FR-006, AC-006/AC-007, IT-006 |
| `GET /v1/chat/operations/{operationId}` | Owner-scoped metadata `{operationId,kind,state,messageId?,failureCode?}`; never stored message/report text; zero replay | FR-006, AC-007, IT-006/IT-018 |
| `PUT /v1/chat/conversations/{id}/read` | `{messageId}` required, representing last actually presented message; never omit it upstream | FR-007, AC-008, IT-007 |
| `POST /v1/chat/conversations/{id}/accept` | Explicit accept; optional absent upstream rev is supported; composer waits for confirmed accepted state | FR-008, AC-003/AC-009, IT-007 |
| `PUT` / `DELETE /v1/chat/conversations/{id}/mute` | Empty-body shared mute/unmute; refresh authoritative state/counts; no read mutation | FR-009/RULE-005, AC-008/AC-009/AC-015, IT-007/IT-011 |
| `POST /v1/chat/conversations/{id}/leave` | Explicit confirmed owner leave; successful result cancels pending DM delivery; peer history unaffected | FR-009, AC-009, IT-007/MAN-001 |
| `POST /v1/chat/reports` | `{operationId,conversationId,messageId,reasonType,reason?}`; authorized messageRef built server-side, fixed moderation destination; same once-attempt receipt discipline as send | FR-010, AC-010, UT-005/IT-004 |
| `GET` / `PUT /v1/chat/incoming-policy` | Read shared enum `{allowIncoming:everyone\|following\|nobody}`; PUT replaces this setting only through internal conditional declaration merge; result includes `{persisted:true}` without claiming instantaneous propagation | FR-011/RULE-002, AC-011, IT-008 |
| `GET` / `PUT /v1/chat/push-preferences` | `{pushEnabled}` account-wide only; no social audience selector and no Bluesky notification preference write | FR-014/RULE-005, AC-015, IT-011 |
| `GET /v1/chat/unread-counts` | Native `{accepted,requests}` values 0–100; pass includeGroupChats=false; 100 is a cap sentinel, no custom totals | FR-007, AC-008, UT-004/IT-007 |
| `POST /v1/chat/sync` | `{view:inbox\|conversation\|none,conversationId?,knownRevision?}` updates short-lived authenticated view activity and returns current revision/state only; schedules/coalesces account work, not an inline getLog per caller | FR-012/FR-015, NFR-005, AC-012/AC-022, IT-009/IT-020 |
| `GET /v1/chat/notification-opens/{reference}` | Resolve random private push reference only for its authenticated owner and current access; returns typed conversation destination or safe unavailable result | FR-014, NFR-001, AC-016, IT-013 |

Do not add a preview-scraping endpoint: accepted message cards use existing authenticated CraftSky post/profile/event reads. Reuse existing CraftSky profile search for New message. Endpoint count is not a requirement.

Use provider limits via a server-issued opaque pagination token bound to owner, bucket/conversation and messaging generation. Prefer random persisted continuation handles with upstream cursor in private storage: this avoids leaking private cursor contents in client logs and needs no new encryption secret. Cap continuation state at 256 handles/account, expire after 30 minutes, and delete on disconnect/deletion. Invalid/expired/mismatched handles return a safe invalid-cursor result; first-page retry never means silently skipping history. Carry the continuation through filtered-empty pages and bound upstream calls/page. Messages/conversation IDs must not enter diagnostics even when used in routes.

### 5.3 Operational persistence and retention (DR-003)

Store operational metadata only. Proposed entities and invariants:

| Entity | Fields / constraints / transaction boundary | Retention / cleanup |
|---|---|---|
| `chat_accounts` | Owner PK, enabled, messaging generation, pushEnabled, baseline initialized, sync revision, safe state/error code. Owner lifecycle/auth epoch checked alongside generation | Keep disabled row while owner exists to preserve decisions; delete on permanent CraftSky cleanup. First successful enable defaults push on unless the owner explicitly changes the visible initial toggle; subsequent reconnect preserves explicit preference |
| `chat_sync_state` | Owner PK, opaque upstream cursor, cursor version, poll claim/version/expiry, next due/backoff, recovery identity/state; no event payload | Preserve resume cursor across transient failure/disconnect; baseline flag distinguishes first enable from recovery. Cursor becomes unusable on expiry and triggers reconciliation, never fresh-head replacement on ordinary failure |
| `chat_view_activity` | Owner + child/session/device reference, view type, optional convo ID, expiry | TTL 12 seconds; automatic heartbeat does not extend child inactivity. Logout removes that device row; disconnect removes all; expiry simply returns scheduler to background cadence |
| `chat_operations` | Owner + UUID PK, kind, messaging generation at creation, conversation/message reference as needed, keyed request digest, prepared/attempting/confirmed/failed/uncertain state, authoritative result message ID and vetted failure code; never body, report prose or raw response | Preserve operation tombstones until owner deletion, across reconnect/disable. No time-based deletion that would let an old key send again. Retain minimal keyed digest/outcome/reference; no confirmed response body. Redact diagnostic formatting |
| `chat_message_dedupe` | Owner + conversation + message identity PK, logical disposition, observed revision/recovery, timestamp | Keep minimal identifiers until owner deletion to protect replay/catch-up; do not expire by a guessed upstream replay window. No sender text/name/archive |
| `chat_conversation_state` | Owner + convo PK, request/accepted/left eligibility state, request-generation token, last notification/read/recovery watermarks sufficient for dedupe | Preserve request lifetime across leave, reinstall, disable/re-enable and OAuth churn; never reset just because a conversation disappears from a list. Store only state/evidence references, no history |
| `chat_notifications` | Owner + logical key unique, kind normal/request/catchUp, convo/message or generation reference, route UUID, eligibility/delivery state; no visible text/name | Durable eligibility/lifetime ledgers outlive delivery rows. Remove full routing references/delivery detail 30 days after terminal delivery; retain minimal notified-state tombstone until owner deletion |
| `chat_push_deliveries` | Notification + subscription unique, shared lease/retry/expiry fields and vetted provider outcome | Reuse subscription FK cleanup; expire undelivered alerts after 24 hours, then terminally suppress without re-alerting the same logical event. Delete terminal detail after 30 days; provider may still redeliver an issued request |
| Continuation handles and active receipt locks | Owner/scope/generation-bound random handles; operation/poll claims with CAS tokens | Handles TTL/cap as above; expired locks are recovered without replaying uncertain writes; never leave a long-lived DB transaction open while waiting for a client |

These are private Postgres tables, not PDS records. Keep ownership FKs/indexes explicit; purge order must respect notification/delivery/subscription references. Extend the existing deletion worker to delete all chat rows before owner finalization without touching the upstream chat account, messages, declaration or other namespaces. Ordinary device logout leaves owner dedupe/operations intact; disconnect increments messaging generation, cancels pending deliveries/work and clears activity/continuation handles while preserving minimal resume/dedupe/operation state and device-local hidden drafts. Existing lifecycle/auth epoch additionally fences sign-out-all/deletion.

Permanent tombstones are a deliberate minimal-metadata tradeoff: required lifetime request dedupe and indefinite old operation retry cannot safely use an arbitrary short TTL. Document it in code/operations; revisit only with a product/contract revision or a provably safe signed operation-expiry design. No unkeyed digest of private text: introduce a dedicated redacted `CHAT_OPERATION_HMAC_KEY` with a key ID for domain-separated request digests during authorized implementation/configuration. Do not reuse the existing Instagram HMAC key or OAuth signing material. Key provision is a deployment prerequisite; never commit a key or substitute a plain text hash. Bind digest to owner, operation kind, target and exact payload; same ID with changed content returns conflict and never invokes upstream.

### 5.4 Single-attempt sends/reports and crash safety

Validate owner/capability/membership, target and input before reserving an operation. Creation/send additionally require current block and availability checks; a block must not reject reporting of upstream-accessible history. Insert prepared receipt with request digest in a short transaction, unique by owner/operation. A claim commits `attempting` before any external write. Never persist the text/report or enqueue it for automatic delivery; the current authenticated request holds it transiently. While under the supported effect/session boundary, recheck authority and generation before the one external call, plus block/availability for sending. Reporting rechecks message accessibility without applying a send prohibition. Persist only result identifiers/state.

On duplicate operation submission: mismatch is conflict; existing prepared/attempting/uncertain/confirmed/failed yields its receipt without a new upstream effect. A confirmed receipt may hydrate its message through an authorized upstream read. A crashed prepared operation with a durable guarantee no send began may be terminal failed; an abandoned attempting receipt becomes uncertain even if a crash happened just before the actual socket write. This conservatism is preferable to an avoidable duplicate. OAuth credential-persistence failure after a possible successful send is also uncertain, not safely retryable.

A duplicate caller never takes over a prepared operation to send it; only the original request may advance its own prepared claim. Crash recovery terminates an orphaned prepared receipt as definitely not submitted, leaving explicit fresh-operation retry to the owner.

Only a definitive upstream/auth rejection known to precede an effect permits the narrowly supported auth nonce/expiry challenge exchange. Verify the complete Indigo/HTTP/coordinator/Dio stack with listener call-count fixtures; no generic 429/5xx/network retry around send/report. A local mutex or operation row alone does not protect against hidden SDK replay. Safe read retries have separate bounded policy.

Flutter retains its text, fetches the operation receipt and refreshes history after uncertainty. Confirmation requires a returned authoritative message ID/result linked to that operation; identical text/timestamp/sender resemblance is insufficient. With no correlation upstream, unresolved stays uncertain. Explicit Send again explains duplicate risk and creates a new UUID; reconnect/restart never creates or replays a send. Reporting uses equivalent receipt discipline with a recoverable failed/uncertain UI and no blind resend, without promising send-again report UX not required by the contract.

### 5.5 Synchronization, baseline, recovery and dispatch

Acquire one account poll claim with expiring CAS token. Select a live capable session without touching child inactivity; load saved cursor/version and captured lifecycle/messaging generations. First-ever enable performs a cursorless baseline and authoritative enumeration without push; records existing request/message state as baseline/notifiable=false. Do not rebaseline on reconnect or partial transient failure. A crash during initial seeding resumes baseline initialization before enabling notifications.

Decode each getLog event into bounded type/revision plus reviewed supported identifiers; preserve unknown envelopes/revision in transient decoder memory long enough to advance safely. Do not persist their raw bodies or interpolate them into errors. Reconcile authoritative direct state for relevant events; groups never create DM work. The database transaction atomically applies dedupe/policy changes, inserts logical notification and subscription jobs, advances cursor/version and increments owner revision under current generation/claim checks. On crash, either all required durable work commits or none does. No cursor can commit ahead of outbox/dedupe.

Transient failures keep a valid cursor. On invalid cursor or interruption/reauthentication, create one durable recovery identity (persisted until reconciliation completes), enumerate authoritative direct state through all pages, reconcile native counts and request/accepted eligibility, then obtain a new valid continuation only after required state is accounted for. Interrupted enumeration persists page/checkpoint metadata without message content; replays remain deduped. Never call a new head a successful resume while skipping recovery.

Request state machine: baseline request alerts are suppressed; genuinely new eligible unread request generation gets one logical alert; additional messages, leave/reappearance, reinstall and reconnect never reset it. Only positive evidence of accepted state followed by a genuinely new request transition establishes another request generation. Without enough service evidence, preserve the prior alerted generation and mark the transition unresolved; do not fabricate a new generation from list disappearance. MAN-001 must establish the relevant deployed transitions.

Accepted recovery state machine: coalesce previously unseen, still-unread eligible accepted message states into at most one logical catch-up per conversation/recovery. Mark covered messages with their logical disposition in the same transaction; prevent both normal-message and catch-up jobs for the same missed state. Repeated recovery/reconnect with unchanged unread/already-notified state creates no alert. Pre-enable history remains suppressed permanently. Request state uses its own lifetime rule rather than this catch-up coalescer.

Before dispatch: under owner/generation/subscription fences, reconcile current upstream conversation/membership, read/mute/request/leave state and either-direction current indexed/newly confirmed blocks; check account push preference, enabled/capable/live session and active subscription. Suppress owner-sent/groups. Ignore profile mute for DMs; retain existing social suppression unchanged. If upstream eligibility cannot be verified, defer with bounded backoff rather than sending stale information. Current sender name/handle is fetched transiently for generic visible copy, never substituted with message text. Provider data carries only reviewed version/type, opaque account subscription and random notification-open reference; no raw conversation/message/cursor/text/preview fields. Resolve the reference after exact owner activation. Shared send infrastructure must validate the new type without relaxing existing payload checks.

Native unread counts use only getUnreadCounts(includeGroupChats=false); accepted/request badges do not derive from the operational ledger or enumeration. Pass through row unread-message counts, display 100 as 99+, and refresh capped totals after actions. Read watermark is last actually presented message in active view, never newest fetched message or omitted ID.

### 5.6 Resource budgets and session activity

Initial implementation constants (functional bounds, not product performance promises): foreground account polling 4 seconds when a conversation heartbeat is active; authorized background/inbox-only polling 30 seconds; one in-flight getLog/account and 2 concurrent account poll tasks globally. Use fair next-due selection and per-operation session-coordinator serialization. A tick processes one bounded provider page then yields; continuation may be scheduled immediately subject to fairness/backoff. Page size defaults to 50 with maximum 100 only where the operation schema allows it; retain tighter per-method limits when upstream differs. Limit each HTTP list response to 5 upstream filtered pages, returning a continuation even for an empty supported-items page.

Use a 10-second chat upstream timeout within the existing coordinator outer deadline; explicit API request body maximum 64 KiB, upstream JSON maximum 2 MiB; a 10,000-byte send and 20,000-byte report still receive their stricter independent validation. Safe-read retries maximum 2 additional attempts with exponential jitter, base 1 second/max 60 seconds; honor provider retry-after even if longer by scheduling the owner later, not truncating it to the normal cap. Persistent recovery enumeration has checkpoints; never loop through an entire remote account inside a single unbounded request/transaction. Graceful shutdown cancels tasks, releases claims safely and leaves attempting writes uncertain.

Add a `LookupWithoutActivity` authentication path that performs the same expiry/owner/session checks but does not update last_seen_at/idle_expires_at. Select it by server-owned route policy for `/chat/sync` and automatic snapshot/count/status/operation refreshes; never accept a client header that can increase authority. Explicit send/action/consent requests can retain normal user-activity semantics. UI polling, upstream refresh and background selectors never extend inactivity. API snapshot views are still authenticated; no-activity means no lifetime extension, not a weaker authorization check. Cover untouched default expiry and expired-child rejection with IT-005/IT-020/REG-001.

## 6. State, Providers, Controllers, Or DI

### 6.1 Flutter provider graph

Follow generated `@riverpod`/`@Riverpod(keepAlive: true)` conventions; generated `.g.dart`/mapper/router outputs accompany implementation changes. Use owner-keyed `AccountKey` families and existing `accountDioProvider(account)` rather than a global active Dio captured before a switch.

```text
sessionRegistryProvider -> AccountSessionLease / ActiveAccountLease
accountDioProvider(account) -> accountChatRepositoryProvider(account)
accountChatRepositoryProvider -> chatStatusProvider(account)
                             -> ChatInboxNotifier(account, bucket)
                             -> ChatConversationNotifier(account, conversationId)
                             -> ChatUnreadCountsNotifier(account)
                             -> ChatPreferencesNotifier(account)
chatStatus + foreground lifecycle + active views -> ChatSyncCoordinator(account)
ChatSyncCoordinator revision -> invalidate loaded snapshots (one coalesced refresh)
chatDraftStoreProvider -> ChatDraftNotifier(account, conversationId)
conversation accepted + message text -> chatPreviewProvider(account, conversationId, messageId)
notification runtime -> existing exact recipient activation -> ConversationDestination
```

| Provider / controller | Kind and lifetime | Responsibilities | Requirement / AC / Tests |
|---|---|---|---|
| `accountChatRepositoryProvider(account)` | Auto-dispose FutureProvider of interface, matching account notification repository pattern | Await account-specific Dio; narrow status/inbox/conversation/actions/preferences/receipt methods; no OAuth service SDK in Flutter | NFR-001/NFR-003, AC-019/AC-021, IT-015/IT-019 |
| `chatStatusProvider(account)` | AsyncNotifier family | Load disabled/authorizationNeeded/ready/syncing/degraded/unavailable state and capabilities; invalidate after consent/lifecycle | FR-001/FR-016, AC-002/AC-014/AC-017, UT-001/IT-005 |
| `ChatInboxNotifier(account,bucket)` | AsyncNotifier family | Pagination/ID merge and current-owner stale data; accepted/request buckets independent; preserve data when loading more | FR-002/FR-008, AC-001/AC-003, IT-001 |
| `ChatConversationNotifier(account,id)` | AsyncNotifier family | History pages/tombstones, actions, operation receipts; presented-message watermark separate from newest fetched ID; publish only under captured lease | FR-003/FR-005–FR-009, AC-004/AC-006–AC-009, UT-003/IT-002/IT-006/IT-007 |
| `ChatUnreadCountsNotifier(account)` | AsyncNotifier family | Native aggregates only, refresh after upstream state change; expose cap-aware display; no optimistic list arithmetic | FR-007/RULE-005, AC-008, UT-004/IT-007 |
| `ChatPreferencesNotifier(account)` | AsyncNotifier family | Separate push boolean from public/shared incoming policy; generation-fenced confirmed updates/rollback; no social audience semantics | FR-011/FR-014, AC-011/AC-015, IT-008/IT-011 |
| `chatSyncCoordinatorProvider(account)` | Provider owning one disposable coordinator/account | Foreground view registration, single 4-second revision timer while conversation active, 30-second inbox-only refresh; resume/reconnect refresh; cancel in-flight private work on owner boundary; no timer when backgrounded | FR-015/NFR-005, AC-012/AC-017/AC-022, UT-006/IT-020 |
| `ChatDraftNotifier(account,id)` | Notifier with serialized async storage effects | Text/revision, debounced local save, flush on navigation, send snapshot, clear only that unchanged snapshot after confirmation; disconnect hides, logout purges | FR-005/FR-019, AC-028, UT-009/IT-017 |
| `chatPreviewProvider(account,conversationId,messageId)` | Auto-dispose FutureProvider, additionally dependent on session/messaging generation | Accepted-only first-supported-link resolution; current-viewer existing content repositories; at most one candidate, discard stale result | FR-017, AC-025, UT-008/IT-016 |
| `chatRecipientAvailabilityProvider(account,did)` | FutureProvider family | Debounced selected profile availability; never create; revalidate before explicit start | FR-004/FR-018, AC-005/AC-027, IT-003 |

`ChatSyncCoordinator` owns Flutter timers; widgets do not create independent loops. The four-second backend interval applies to upstream shared polling; revision reads do not each force an upstream poll. First view activation requests immediate scheduling if due and waits for a new revision without bypassing an in-flight claim or provider backoff. Dispose timers/Dio cancellation tokens when the owning family loses a lease; account-wide disconnect status also evicts all private history/card providers across devices on their next authenticated refresh.

Wrap UI mutations and async navigation with `ActiveAccountOperation.capture(ref)` and lease checks before acquisition, after awaited work and before publish, using existing guards. Invalidate chat families in `account_boundary_provider.dart`. Do not persist conversation messages in Riverpod disk persistence or package caches. Existing generic provider error capture cannot be fed a DTO in a thrown error/toString; build source-selected private diagnostic context.

### 6.2 Local draft storage

Use a dedicated `ChatDraftStore` backed by the existing `flutter_secure_storage` dependency on iOS, Android and web, with a replaceable in-memory test backend. Keep it separate from the session-registry snapshot and server-side post draft repository. Existing file-backed draft code imports `dart:io`; do not reuse it unconditionally on web. Configure platform-local, non-synchronizing storage; do not enable keychain/iCloud synchronization or an app cloud backup/draft upload path. Check the selected plugin's platform options during implementation and document browser-origin storage limits without claiming end-to-end encryption or full offline history.

Use a per-owner manifest of opaque conversation keys plus per-draft text/revision, backed by serialized per-account writes. A small local account-removal generation/tombstone prevents a delayed save from restoring drafts after logout. On startup, remove draft keys for accounts no longer in the retained registry and finish interrupted purges before exposing any draft. Await purge on explicit account removal; do not purge merely on activation of another retained account. Sign-out-all/revocation clears that account when the device processes it. Disconnect only hides draft providers; re-enable restores same-owner data. Confirmed send clears the submitted draft revision, not newer text typed while the request was in flight; the sent text's uncertainty remains available locally until resolved/discarded.

Storage failures preserve in-memory text and show recoverable save feedback, with no automatic network submission. Draft/text bodies never enter structured errors, push, analytics, AppView draft fields or test evidence. Persist normal local input rather than silently truncating an over-limit draft; validation prevents sending until corrected. Add storage-bound checks only if required by platform constraints and return a recoverable failure rather than truncating or discarding work. Verify real process/plugin behavior in the mobile journey extension; host fake storage alone is insufficient.

### 6.3 Composition and lifecycle

Build ChatService/Store/Scheduler/DM dispatcher in `internal/app` and start workers from the existing application lifecycle. Inject upstream/session runner, fakeable clock/random source and existing private observer. No Redis, Render worker service or second backend is needed for the approved singleton deployment. Private DB poll leases additionally prevent duplicate account workers during restarts/concurrency; they do not replace deployment serialization or owner/auth fencing.

## 7. UI, Widgets, Routes, Or User-Facing Surfaces

### 7.1 Navigation and widget boundaries

Keep the existing five main branches. Add an inbox action with the native accepted unread badge to the shared app shell/drawer; add messaging settings under existing Settings. New typed routes under the authenticated shell: `ChatInboxRoute` at `/messages`, `ChatRequestsRoute` at `/messages/requests`, `NewMessageRoute` at `/messages/new`, `ChatConversationRoute(conversationId)` at `/messages/conversations/:conversationId`, and `ChatSettingsRoute` at `/settings/messaging`. Place literal paths ahead of dynamic paths as existing GoRouter conventions require. Route extras may carry transient hints only; cold-start reconstruction never depends on a cached conversation object.

`ChatInboxPage` composes enablement/recovery notice, accepted list, New message action and a separate requests entry/count. `ChatRequestsPage` retains read requests until accepted/left. `NewMessagePage` reuses CraftSky profile search and selected availability; profile Message actions check availability before explicit canonical start. Both start paths explain unavailable state without creating a conversation on navigation/hover/check. Existing external conversations render upstream identity/handle or safe fallback without CraftSky membership.

`ChatConversationPage` composes participant/header actions, shared/request/degraded notices, a `Chat` widget with CraftSky builders, older-history loading, message-state affordances and the local composer. Requests expose Accept/Leave; the composer is unavailable until acceptance is confirmed. Mute does not mark read. Leave requires localized accurate confirmation of the verified owner-history effect and never promises permanent dismissal or peer deletion. Report dialog names Bluesky and reference/evidence scope before submit, supports bounded explanation and recoverable accepted/failed/uncertain outcomes. Uncertain-send action refreshes history first and requires explicit duplicate-risk acknowledgment for Send again.

Consumer login/registration request messaging authority during their ordinary OAuth browser flow. The disabled Messages gate reuses Direct messages settings; its explainer precedes the explicit CraftSky enable switch. Positive copy covers shared history/settings, Bluesky storage and Bluesky-only reporting; negative semantic assertions reject privacy/E2EE/PDS-portability promises while permitting accurate explicit denials. The default-on account push choice appears only while messaging is enabled and applies to the owner, with separate OS permission states. Disable explains hidden retained local drafts and stops CraftSky access without changing shared incoming policy.

### 7.2 Selected package fit (DR-001/DR-002 and AC-026)

Keep `flutter_chat_ui` 2.12.0 and add direct `flutter_chat_core` 2.9.1 only where its types are imported. Published [package metadata](https://pub.dev/packages/flutter_chat_ui/versions/2.12.0) identifies an Apache-2.0 backend-independent package and iOS/Android/web support; these are package claims, not CraftSky platform evidence. Cached package manifests were inspected: both require Dart >=3.8 <4 and Flutter >=3.32; CraftSky declares Dart ^3.13 and Flutter >=3.47.7. Core's Dio/intl/path_provider ranges fit the declared app ranges; the existing isolated resolution dry run is supporting evidence only. Keep lockfile-resolved versions explicit at implementation and include Apache license/NOTICE obligations in the application's normal acknowledgments.

The [Builders API](https://pub.dev/documentation/flutter_chat_core/latest/flutter_chat_core/Builders-class.html) exposes custom text/composer/message/list/unsupported builders. Cached UI source additionally exposes `ChatAnimatedList.onEndReached/onStartReached`; map provider chronological history to its expected order and test scroll anchors rather than assuming reversing a list preserves position. Use its in-memory controller only as a presentation mirror of the account conversation notifier, disposing it on owner/generation changes.

| Fit area | Planned adapter / evidence | Verification |
|---|---|---|
| Themes/localization/accessibility | Existing CraftSky colors/fonts/buttons and localized labels through custom builders; no separate Provider business state despite package-internal Provider | UT-010/UT-011; MAN-003 |
| History/deleted/unsupported messages | Typed transient mapping by upstream message ID; tombstones replace cached text; list pagination hooked to cursor-based loadMore and view-anchor preservation | UT-003/IT-002, AT-004 |
| Pending/failed/uncertain | Operation UUID for local pending identity; only authoritative response ID confirms; custom state/status affordances including uncertain | IT-006, AT-006/AT-007 |
| Local composer/drafts/request acceptance | Custom composer bound to draft notifier; explicit send; disabled until accepted; no package-provided network sender or offline queue | UT-009/IT-017, AT-003/AT-028 |
| First supported compact card | Custom text/message builder parses approved CraftSky routes; resolver only in accepted bubbles; compact permitted content/thumbnail; no default arbitrary URL preview widget | UT-008/IT-016, AT-025 |
| No hidden preview fetch | Bypass/remove default fetching text/preview builders and cache/network resolver; composer and unaccepted messages never instantiate a resolver. Test actual network call count, not just absence of a rendered card | UT-008/UT-010/IT-016 |
| Actual iOS/Android/web and dependency/license | Resolve/install only in implementation, analyze, host tests, platform build/run and assistive-technology checks with selected adapters | MAN-003/MAN-004, GAP-006 |

No concrete package incompatibility was established in planning. Rendering, real plugin behavior and platform execution are unverified; return a concrete failure for review before replacing the package. Prevent whole private message models, preview URLs or private contents from entering package diagnostics/cache serialization. Plain text links remain usable independently of card success. For focused reply targets, fetch/render the reply and navigate via existing thread route with canonical root and focused source.

### 7.3 Private notification opens

Extend `NotificationDestination` with a bounded `ConversationDestination`; provider payload first produces an opaque-reference open attempt tied to the existing account subscription binding. The existing open coordinator activates the exact stored owner lease, then requests `/chat/notification-opens/{reference}` and checks the active lease again before routing. Check membership/messaging state server-side and immediately at the UI boundary. Cold start waits for registry initialization; missing owner/expired grant/disconnect/member loss yields a safe unavailable/authorization destination. Do not resolve the private reference with whichever account happens to be active or fetch A's content under B while planning a switch.

Mobile notification permission/token registration continues through current runtime subscriptions. Web supports authenticated inbox/revision refresh only; no web background push work. Provider payload and OS copy tests remain distinct from runtime-injected journey tests.

## 8. Error, Loading, Empty, And Edge States

| State / Case | Planned Handling | Requirement IDs | Test IDs |
|---|---|---|---|
| Disabled / base-only / partial optional authority | Explicit Enable or reauthorize; ordinary CraftSky remains usable; no silent permission escalation or empty inbox success | BR-002, FR-001, RULE-004 | AT-002, UT-001, IT-015, REG-001 |
| Valid OAuth but chatDisabled / generic restriction | Safe service-restricted/unavailable state; no invented age/email cause or bypass | FR-001 | AT-002, UT-001, MAN-001 |
| Empty accepted/request page with cursor | Keep Load more/continuation; render real empty state only at cursor exhaustion | FR-002, FR-003 | AT-001, AT-004, IT-001, IT-002 |
| Offline / transient / 429 / slow or oversized upstream | Retain same-owner loaded state with stale/retry notice; backoff/bounds; healthy accounts progress; no read side effects | FR-015, NFR-004, NFR-005 | AT-017, AT-022, UT-006, IT-020 |
| Block/incoming restriction race | Recheck before creation/send/push; retain accessible history/reporting; unavailable when upstream denies; unblock refreshes availability | RULE-001, FR-004 | AT-005, AT-018, IT-014 |
| Rejected text/report explanation | No upstream write, retained correction text, localized nonblank/grapheme/UTF-8 feedback | FR-005, FR-010 | UT-002, UT-005, IT-004, IT-006 |
| Possible remote acceptance with lost result | Durable uncertain receipt; no replay/false confirmation; explicit warned Send again for sends | FR-006, FR-010 | AT-007, AT-010, IT-004, IT-006 |
| Request acceptance failure / no rev on successful accept | Disabled composer until confirmed accepted; absent rev is not an invented failure | FR-008 | AT-003, IT-007 |
| Concurrent shared declaration write | Conditional conflict refetch/merge; preserve unknown fields; confirm persistence not instantaneous service propagation | FR-011, RULE-002 | AT-011, IT-008 |
| Unknown/group/delete event / expired cursor | Transient tolerant envelope handling; no group work; deletion removes cached text; authoritative recovery with durable policies | FR-012, FR-013 | AT-012, AT-013, AT-029, IT-009, IT-010, IT-012 |
| Account switch, revoked lease, disconnect, logout, deletion | Cancel/ignore stale completions; stop unauthorized work; clear private visible state; preserve or purge drafts according to event | FR-014, FR-016, FR-019, NFR-001 | AT-014, AT-016, AT-028, IT-005, IT-013, IT-017 |
| First supported preview denied/deleted/fails | Preserve links/message; no card, later-link lookup, composer/request lookup or stale-account publish | FR-017 | AT-025, UT-008, IT-016 |
| Draft storage failure / logout during save | Recoverable private feedback, in-memory text retained; serialized purge generation rejects late save; no network fallback | FR-019, NFR-002 | AT-028, UT-009, IT-017, IT-018 |
| Provider push after eligibility changed/in flight | Final current checks before send; defer uncertain eligibility; cannot retract issued network request; provider redelivery is not exactly once | FR-013, FR-016 | AT-013–AT-016, IT-010, IT-011, MAN-002 |

Use mandatory private diagnostic entry points and safe stable codes with typed causes/stacks. Mark all DM dynamic route segments private in server/client route classification before registering telemetry. No body/report/draft/credential/raw response/conversation/cursor/device routing in logs, exception prose, analytics, metrics labels or public/social notification rows. Worker `ObservePrivateFailure` or the appropriate consuming boundary owns one issue; expected per-account reauth is recovery state, not repeated issue creation. Verify production serializers with positive approved fields and protected canaries, not just test log strings. No new latency telemetry obligation.

## 9. Test Implementation Plan

Treat `02-acceptance-tests.md` as the detailed assertion contract; the order below maps every test ID to a work slice. Each slice runs red → minimal green → focused refactor; do not prewrite the complete feature and add tests later. Proposed target filenames remain explicit there, with the corrections below for real observed seams.

| Order | Test ID | Target | Setup / Fixture | Initial Expected Failure |
|---|---|---|---|---|
| 1 | UT-001, IT-015, AT-002, AT-019, REG-001 | `internal/auth/effective_scopes_test.go`, `chat_oauth_integration_test.go`, coordinator/flow tests; later `internal/chat/transport_test.go` | TD-001/TD-009; base, combined method/audience, partial/full grants; same-owner callback and disconnect race | Advertised union incorrectly invalidates base login/refresh or equivalent RPC grant fails; chat request not capability-gated |
| 2 | IT-005, IT-013, IT-019, AT-014, AT-016, AT-021 | Auth lifecycle/no-activity lookup tests; `internal/api/chat_contract_test.go`, route policy/catalogue, `internal/chat/authorization_test.go` | TD-004/TD-009; two devices, invalid owners/IDs, automatic polling across expiry and in-flight fences | No chat route/policy or no-activity path; automatic poll touches idle expiry; stale authority can reach effects |
| 3 | UT-002, UT-005, UT-011, IT-004, AT-006, AT-010 | Go validation/report adapter tests; Flutter validation, `messaging_consent_test.dart` and `report_page_test.dart` | TD-003/TD-006/TD-011; Unicode/explanation boundaries and localized consent/disclosures | Missing independent bounds, authorized reference routing or explicit pre-consent/pre-report disclosure assertions |
| 4 | IT-006, AT-007 | Real PostgreSQL `internal/chat/send_integration_test.go`; Flutter `send_flow_test.dart` | TD-003; crash/acceptance barriers, duplicate UUID, changed payload, nested auth/transport retries | Duplicate operation sends again, body persists or ambiguous result is retried/confirmed |
| 5 | IT-001, IT-002, UT-003, IT-003, AT-001, AT-004, AT-005, AT-027 | Chat inbox/history/start adapter tests and Flutter inbox/history/new-message suites | TD-001/TD-002; opaque filtered-empty pages, tombstones, external peer, same canonical conversation | No adapter/direct filtering/cursor continuation or availability accidentally creates |
| 6 | UT-004, IT-007, IT-008, IT-014, AT-003, AT-008, AT-009, AT-011, AT-018 | Chat read/actions/declaration/block tests and Flutter conversation/preferences | TD-002/TD-004/TD-005 | Read marks unseen messages, badge is list-derived, request implicit acceptance, mute/profile-mute conflation, missing CAS merge or block fence |
| 7 | IT-009, IT-010, IT-012, AT-012, AT-013, AT-029 | Migrated PostgreSQL sync/crash/notification-policy tests | TD-007/TD-008; first baseline, cursor/version claims, event variants, replay/crashes and recovery lifetimes | Cursor advances without durable outbox or resume/reappearance creates duplicate/baseline alerts |
| 8 | UT-007, IT-011, AT-015, REG-002 | Push payload/DM dispatcher and existing social dispatcher suites; Flutter push settings | TD-004/TD-006; current eligibility races, subscription fanout, missing sender | Social FK/profile-mute path is reused incorrectly, private provider field leaks or current-state suppression absent |
| 9 | UT-006, IT-020, AT-017, AT-022 | Scheduler unit/integration and Flutter sync state/controller | TD-007; fake clock, two owners, held calls, retry-after and expired view/session | Multiple views create parallel account polls; four-second configuration, fairness/bounds/background work or no-touch expiry fails |
| 10 | UT-010, AT-023, AT-026, MAN-004 | `conversation_page_test.dart`, `chat_localizations_test.dart`, `messaging_flows_accessibility_test.dart`; package-fit record/builds | TD-011; actual selected package builders and supported locale/layout/semantics fixtures | Missing package/adapter, builder defaults perform network previews, anchor/focus/message-state presentation breaks |
| 11 | UT-009, IT-017, AT-028 | Draft controller/storage suites and existing mobile critical journey extension | TD-011; production storage seam with temporary test backend then real plugin/process | Draft isolation/retention/logout purge/late-save fencing fails, reconnect auto-sends or old send clears newer text |
| 12 | UT-008, IT-016, AT-025, REG-004 | Chat preview policy/flow and content router tests | TD-010; denied first target plus accessible later URL, focused reply, request acceptance and switched account | Wrong candidate/fallback fetch, unauthorized/composer/request resolution or stale publish |
| 13 | IT-018, AT-020, REG-005, REG-006 | Real Go/Flutter serialized sinks, private DB cleanup, public-write recording and deletion regressions | TD-006/TD-011; success/error/cancel paths across all prior slices | Private canary in a sink/public row, lost approved cause/context, retained archive or overbroad upstream cleanup |
| 14 | REG-003, AT-016, AT-023, AT-028 | Existing notification/account routing plus mobile critical journeys | TD-001/TD-004/TD-011; injected runtime opens, cold start and retained registry/plugin storage | Wrong-account navigation, lost process draft or existing destination behavior regression |
| 15 | AT-024, MAN-001, MAN-002, MAN-003, MAN-004 | Controlled actual service, signed iOS/Android devices and web/assistive-technology runs | Consenting test accounts and authorized test environment; no private evidence artifacts | Integration may reveal unsupported granular/report/settings/eligibility/cursor/leave semantics or actual device/UI incompatibilities; record observed difference, not a manufactured mock pass |

Do not postpone privacy or owner fencing until slice 13: add corresponding assertions to each owning slice; slice 13 consolidates full serialization/cleanup coverage. Similarly pair Flutter tests with service behavior as each view lands; order 10 integrates the full selected UI adapter rather than delaying all user-visible tests to the end.

### Focused and full commands

First failing test (planned new name in the existing auth package): `TestChatOptionalScopesPreserveBaseSession`. Start by modeling advertised-vs-base requirements and asserting base login/refresh/coordinator remains valid while optional chat is denied. Companion `TestChatCapabilityNormalizesRPCScopes` covers semantic equivalent grants and wrong audience/wildcard rejection. No executable test is created in this stage.

```sh
# From appview/, once the named focused tests exist:
go test ./internal/auth -run 'TestChat(OptionalScopesPreserveBaseSession|CapabilityNormalizesRPCScopes)' -count=1

# From repository root; these existing recipes configure their own test services/setup:
just appview-test-unit
just test
just appview-check
just app-test test/chat
just app-test test/notifications test/router test/observability
just app-test
just app-analyze
just app-test-integration <device-id>
```

The first command is focused unit evidence only; it must not imply PostgreSQL integration passed when DB helpers skip. `just test` uses current dev Compose PostgreSQL/MinIO and host Go race tests; `just appview-check` is the isolated backend gate. Do not run appview via a host `go run` server. Add a focused `scripts/appview-test focused` mode only if implementation repeatedly needs DB-configured filtered package/test runs; it must reuse existing dev-service resolution and fail closed instead of hardcoding ports. Do not write a separate service setup script merely for this plan.

Generation in implementation: use `dart run build_runner build --delete-conflicting-outputs` from `app/` for selected Riverpod/mappable/router outputs, the configured Flutter localization workflow for ARB changes, and `just chat-db-generate` (new during implementation) after migrations/queries. The existing `just lexgen` is unnecessary because no local Lexicon changes are planned. Builds use existing iOS/Android/web recipes; native desktop remains unsupported. None of these commands was run during planning.

### Controlled live evidence (DR-004)

Record MAN-001–MAN-004 in `06-validation-evidence.md` with safe account aliases, build/version/platform, action/outcome and explicit unverified items. Prepare consenting compatible accounts, one external peer, a controlled request sender and an authorized report-testing procedure. Use actual CraftSky/AppView OAuth, not pasted bearer tokens or bypass adapters. Validate all implemented granular operations, base/partial/full consent and refresh, fixed PDS chat/report proxying, service status/request/count behavior, actual peer messaging/read/mute/accept/leave/restart, declaration readback/propagation and closed-app getLog/recovery with a live session.

Separate actual iOS/Android APNs/FCM delivery and cold/foreground OS taps from runtime injection. Verify web foreground/resume and assistive-technology flows separately; the current dart:io mobile harness cannot prove web. A restricted account, expired upstream cursor or controlled report acceptance may be unavailable: keep the affected item unverified and resolve the material gap before claiming full acceptance. Do not send fabricated abuse reports to production just to get a green check. There is no separate feasibility spike, pre-design service gate or message-arrival benchmark. Exact service transition evidence may require requirements review if behavior cannot meet the contract; no silent broad permission fallback or safety/control omission.

## 10. Sequencing And Guardrails

- First TDD step: UT-001/IT-015 optional scope separation, starting with `TestChatOptionalScopesPreserveBaseSession` and the focused command above. Read `implement-tdd` before creating `05-implementation-plan.md` or writing executable tests.
- Dependencies: auth/capabilities and fixed adapter seam precede private effects; migrated operational state precedes send/sync; upstream actions and durable policy precede push; account-safe state/drafts/routes underpin UI; every slice carries its own privacy and lifecycle tests. Package setup occurs only when implementation is authorized and its owning failing widget test requires it.
- Preserve the coordinator lock order and existing direct PostgreSQL advisory-lock connection model. Never nest owner/session fences or use transaction-mode PgBouncer. Account messaging generation joins existing lifecycle/auth epochs; it does not replace them.
- Every API/worker/private open validates owner, enabled/capable session and current membership/state; apply block checks to creation/send/dispatch while retaining accessible history/reporting. Explicit acceptance remains mandatory before request composition.
- No text/report body in AppView persistence, receipts, social notifications, logs or provider payloads; only device-local unsent drafts are permitted persisted content. No automatic replay of send/report/offline drafts. No text-similarity delivery confirmation.
- Atomic progress/dedupe/notification commit and unique notification/subscription keys are required. Request lifetime survives disconnect/reinstall/leave; accepted catch-up is coalesced by persisted recovery identity and unseen state; pre-enable history is permanently baseline-suppressed.
- `flutter_chat_ui` remains selected. Do not ship package default preview fetch/cache/sender behavior; use approved accepted-bubble content reads only. First-supported-link failure performs no later-link lookup. Resolve reply focus via existing canonical navigation.
- Before migration implementation, check numbering again for duplicate prefixes; use normal migration/SQL/query generation and production-schema tests. The plan does not authorize applying production schema or altering Render configuration/secrets.
- Do not change user-facing release timing, introduce groups/media/reactions/global Bluesky search/typing/read receipts/offline history or queues, web push, a DM moderator console, a new messaging backend, a public Lexicon or desktop targets. Keep native unread aggregates and supported platforms unchanged.
- Commit only if explicitly requested later; user selection of coding planning authorizes this artifact, not source changes, push or production operations.

## 11. Risks And Open Questions

| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Non-blocking | Deployed granular grants/eligibility/report and request/leave/cursor behavior still unproven | May reveal a real contract incompatibility despite schema/adapter fit | MAN-001 during full integration; record exact material differences, return for requirements review if required behavior is unavailable. No separate spike or broad fallback |
| CPQ-002 | Non-blocking | Persistent minimal operation/message/request ledgers trade bounded content retention for lifetime deduplication | Metadata grows with supported usage; guessed TTL could duplicate sends/alerts | Retain minimal identifiers/digests until owner deletion as §5.3; purge terminal routing/delivery detail after specified TTLs; revisit only with proved safe expiry design and approved scope |
| CPQ-003 | Non-blocking | Dedicated chat operation request-digest key provisioning/rotation | Cannot safely use unkeyed hashes of private low-entropy text; old receipts must stay non-replayable | Introduce `CHAT_OPERATION_HMAC_KEY` as a redacted secret config with key ID; use deterministic test key only in fixtures. Document local development generation and separately authorized production provisioning. Preserve old verification key IDs across rotation; if an old verification key is unavailable, return the existing protected receipt without sending, never treat it as a new operation. Do not reuse the Instagram feature key or OAuth signing key |
| CPQ-004 | Non-blocking | sqlc convention exists in guidance but no active feature recipe/config was found | Blindly invoking a presumed generator fails or creates duplicate handwritten/generated queries | Add narrow feature-scoped config/recipe in storage slice and validate with real migrations; generated outputs follow repository Go conventions |
| CPQ-005 | Non-blocking | Chat UI resolution/source hooks do not establish rendering, accessibility or real secure-storage behavior | Potential platform/plugin builder incompatibility | UT-010/UT-011, IT-002/IT-016/IT-017 and MAN-003/MAN-004; report a concrete blocker before substituting the selected UI package |
| CPQ-006 | Non-blocking | Current bearer lookup touches inactivity; automatic calls currently share normal middleware | Polling could defeat existing expiry and keep revoked/idle authority usable longer than intended | Introduce no-activity route auth with identical validation and negative expiry-extension tests; no client-controlled auth bypass, no worker bearer touch |
| CPQ-007 | Non-blocking | Existing sender/dispatcher types encode social category and event FKs | Adding DM as a simple category could inherit profile mute or expose private routing | Shared provider-ready delivery helper plus DM-specific source/eligibility store; retain social invariants and REG-002/REG-003; final typed payload contract stays closed |
| CPQ-008 | Non-blocking | State changes after final remote-effect/dispatch check are unavoidable | Already-issued network effects or provider redelivery can outlive a local state change | Test barriers before final checks; fence subsequent commits/jobs, and document residual in-flight race without exactly-once/retraction claims |

No open product decision prevents implementation design. Key provisioning, live service prerequisites and concrete package/platform checks are implementation/deployment tasks with explicit evidence boundaries, not authority to mutate production now.

## 12. Handoff To TDD Builder

- Coding plan: `04-coding-plan.md`.
- TDD execution plan: create `05-implementation-plan.md` only after the owner chooses implementation and the `implement-tdd` skill is loaded.
- Start with test: `TestChatOptionalScopesPreserveBaseSession` in the auth test package, mapped to UT-001/IT-015, BR-002/RULE-004/NFR-001 and AC-002/AC-019. Follow with RPC normalization and same-owner consent/fencing cases.
- Focused command: from `appview/`, `go test ./internal/auth -run 'TestChat(OptionalScopesPreserveBaseSession|CapabilityNormalizesRPCScopes)' -count=1`; this is unit-only until DB-backed flow fixtures are selected/configured. Use `just test`/`just appview-check` for fail-closed persistence evidence.
- All planned Must tests have an implementation target in §4/§9 and the detailed revised test specification. Every work slice maps requirements/criteria/test IDs; do not collapse live evidence into mock evidence or manufacture additional acceptance obligations from reference sections.
- DR-001/DR-002 remain addressed in the revised earlier artifacts and are concretized by consent-copy tests and scenario-specific targets. DR-003 is addressed by metadata/retention/claim/crash/local-storage rules; DR-004 by the controlled live evidence checklist and CPQ-001/CPQ-005.
- First implementation tasks must include redacted `CHAT_OPERATION_HMAC_KEY` configuration and the no-activity auth seam before dependent durable writes/automatic polling. Production key provisioning/config changes require their normal later authorization and verification.
- Validation performed in this stage: read-only code/package-manifest/document inspection and document traceability/structure checks only. Prior isolated dependency solver evidence is attributed to requirements; no install, application compile, executable tests, migration, deployment or live messages/reports occurred here.
- Stage output: only this uncommitted plan; earlier workflow artifacts remain unchanged by this stage. Stop for the owner's choice of implementation, manual note or no further work.

### Implementation detail: same-owner consent handoff on Flutter

IT-015/IT-017 and REG-001 reuse the durable handoff protocol without treating chat consent as a new login. A chat exchange additionally returns its server-owned `purpose: chat` and `sourceTokenHash` (base64url SHA-256 of the initiating child bearer, never the bearer). Ordinary exchanges omit these optional fields. Flutter matches this hash to the already retained same-owner session before staging; the durable pending receipt captures that local session generation. Confirmation rechecks the exact source hash/generation before dispatch and before local publication. Missing/replaced/removed source sessions fail closed. Successful chat confirmation replaces only that owner's child token, preserves another active account, and advances activation authority only when replacing the active owner. The pending receipt carries these fields through process restart. No new OAuth credential handoff, public record or production mutation is introduced. Chat handoff diagnostics select private operational context.
