# Coding Plan: Settings Page And Account Management

## 1. Inputs

- Requirements: `01-requirements.md` — approved, High risk, 2026-08-10.
- Acceptance tests: `02-acceptance-tests.md` — 15 acceptance scenarios, 24 unit tests, 28 integration tests, 12 regression tests, and 4 manual checks.
- Document review: `03-document-review.md` — Approved with notes, High risk, 2026-08-10.
- Repository guidance: `AGENTS.md`.
- Architecture references:
  - `atproto-craft-social-app-reference.md` for the PDS/AppView ownership boundary.
  - `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md` for `/v1/*`, route naming, middleware, error envelopes, and camelCase JSON.
  - `docs/superpowers/specs/2026-04-18-appview-oauth-bff-design.md` for server-held OAuth sessions and callback handoff.
- Existing Flutter seams inspected:
  - `settings_page.dart`, `clear_image_cache_tile.dart`, `sign_out_tile.dart`, Settings route tests, and notification settings.
  - Active identity, retained-session registry, account switcher content/state, activation coordinator, local account cleanup, fixed-account Dio clients, OAuth deep-link completion, and router redirects.
  - Shared external-link launcher, package metadata/build-label formatting, CraftSky dialogs, theme error colours, account avatar, draft file storage, and localization resources.
- Existing AppView seams inspected:
  - OAuth handler/store/callback, `craftsky_sessions`, background OAuth selector, PDS client list/delete methods, route policies, and error envelope.
  - Tap `Event` identity (`URI`, `DID`, `Collection`, `Action`, `ID`, `Rev`), dispatcher, indexer-before-ack flow, and poison-event retry behavior.
  - Profile/post/like/repost delete handlers and their notification/membership side effects.
  - Scheduled-post durable worker/lease/retry patterns, scheduled private-media cleanup, notification actor cleanup, language cleanup, and Instagram `PurgeOwner`.
  - Migrations through `000035_profile_pins` and every DID-bearing or indirectly owner-derived private table listed in TD-008/TD-009.
- Approval gates carried forward:
  - The owner-approved PDS-write exception is limited to freshly reauthenticated deletion of that owner's `social.craftsky.*` record collections. It does not permit whole-account deletion, other namespaces, direct blob deletion, or a generic PDS-delete API.
  - `DR-001` requires the repository guidance/reference amendment before a destructive route is enabled.
  - `DR-004` requires the adjacent Customisation route to be merged before the final Settings surface and REG-001 can pass.
  - Implementation remains a later `implement-tdd` stage; this document changes no source, schema, generated output, or dependency.

## 2. Implementation Strategy

Implement the feature in ten ordered increments, with the destructive namespace boundary fixed before any deletion side effect exists.

1. **Establish the deletion safety boundary first.** Add a compiled `social.craftsky.*` record-collection registry containing the current profile, post, like, and repost collections, with the membership profile ordered last. The first red test is `UT-009`; it walks `lexicon/social/craftsky/`, includes only schemas whose primary definition is `record`, and requires an exact set match. Add the guidance/reference amendment immediately after this test and before route registration. Do not edit a Lexicon file for this feature.
2. **Build the non-destructive Settings hierarchy from existing seams.** Refactor the main page into an identity-led, sectioned list; extract the responsive account-switcher launcher currently private to `app_shell.dart`; add Account and About child routes; reuse notification settings, legal-link launching, cache clearing, account avatar, and the localized package build label. Keep Sign out's existing behavior and change only its presentation to the theme error colour.
3. **Introduce an explicit pre-acceptance OAuth intent.** `POST /v1/account-deletion/intents` creates a short-lived, cancelable intent, status capability, and fresh OAuth flow for the authenticated DID. OAuth request purpose/owner/intent metadata is written atomically inside `PostgresAuthStore.SaveAuthRequestInfo`; the callback validates the returned DID, stores only a one-way proof hash plus the exact OAuth session ID, and deep-links to deletion reauthentication completion without creating an ordinary CraftSky bearer session. The intent is not an accepted deletion job and expires after 10 minutes.
4. **Accept deletion as one database boundary.** Typed-handle submission calls `POST /v1/account-deletions/{jobId}` with the ordinary bearer, intent status capability, and one-time reauthentication proof. In one owner-locked transaction AppView validates the exact captured handle hash and fresh proof, makes the operation non-cancelable, binds the exact OAuth session, converts old bearer hashes into status-only recovery records for offline devices, removes account subscriptions, hard-deletes every ordinary CraftSky bearer, and deletes every unbound OAuth session. Binding is established before any bearer/OAuth deletion. The response is `202`; duplicate submission resolves the same operation. The device already persisted its status capability before opening OAuth, so a lost acceptance response can be reconciled without another ordinary session.
5. **Run one durable, leased deletion worker.** The worker progresses idempotently through private cleanup, CraftSky PDS enumeration/deletion, indexer convergence, final verification, and finalization. Defaults are batch 20, lease 2 minutes, poll interval 2 seconds, and automatic attempts at `0`, `1m`, `5m`, `15m`, `1h`, and `6h`, with deterministic bounded jitter after the first attempt. Exhaustion becomes `needsAttention`; manual Retry resets the schedule on the same job. Unusable OAuth authority goes directly to attention with a fresh-reauth action rather than restoring ordinary access.
6. **Keep PDS operations narrow and observable.** The worker may resume only the OAuth session referenced by its job and may call only `ListRecords` and `DeleteRecord`. For every listed record it persists the expected URI before delete. Missing records are idempotent. Each collection is paged until empty, then all collections are rescanned; any newly found URI re-enters the same register-before-delete loop. Profile is deleted last. No account-delete, blob-delete, upload, create, put, or non-CraftSky collection method is present on the deletion worker interface.
7. **Leave public AppView deletion with existing indexers.** Wrap the current dispatcher with a post-handler observer. It calls the existing indexer first; for a relevant owner-scoped CraftSky delete it then records an idempotent `(job, URI, collection, Tap event ID, repo revision)` receipt before the consumer sends its ack. Expected records and owner/job-scoped delete observations use the same receipt table, covering the narrow race in which an external delete occurs between PDS listing and expected-URI registration. A relevant handler or receipt failure is marked `mustReplay` so the current Tap poison-pill limit cannot acknowledge it.
8. **Make private cleanup explicit and drift-resistant.** A compiled private-data manifest names every direct table, indirect FK-derived table, service, orphan rule, and external object-store effect from TD-008/TD-009, with an explicit `delete`, `indexerOwned`, or `retainShared` policy. The completeness test introspects every `did`/`*_did` table and recursively reachable FK table in the migrated test schema; every discovered surface must be classified. Cleanup components are idempotent and checkpointed. Scheduled media reuses the existing cleanup queue and records only cleanup job IDs until object deletion completes. Instagram uses an explicit-deletion purge, not ordinary inactivation/retention.
9. **Finalize only after every authoritative gate.** Required gates are: private manifest complete; ordinary sessions/subscriptions absent; every expected URI receipted; indexed CraftSky rows and derived effects absent/retracted; scheduled object cleanup complete; and an immediately preceding full PDS rescan empty. The worker then removes the bound OAuth session and, in the terminal database transaction, inserts the minimized audit and deletes the operational row, status/recovery credentials, expected URIs, receipts, and checkpoints. The audit contains only job ID, DID, accepted/terminal timestamps, coarse outcome, and `expiresAt = terminalSuccessAt + 30 days`.
10. **Reconcile local state across accounts/devices.** Flutter stores deletion status in a separate secure registry, never in `SessionRegistry` as an ordinary session. On acceptance it persists status first, purges that account's drafts/media, Instagram snapshot, caches, providers, and ordinary session, then activates the MRU remaining account or shows status as the primary signed-out route. The switcher combines ordinary rows with disabled accepted-deletion rows. An offline device's former bearer can be exchanged exactly once through the recovery route for status-only access; ordinary middleware still rejects it. Normal OAuth for a DID with an active deletion returns status-only handoff and never initializes membership; after terminal success normal OAuth may create fresh membership.

The exact user-facing support destination will be centralized as the repository's existing public discussion channel, `https://github.com/Zambrella/craftsky/discussions`. Changing it later is a copy/configuration update and must not alter deletion authorization or state.

## 3. Affected Areas

| Area | Existing Pattern | Planned Change | Requirement IDs | Test IDs |
|---|---|---|---|---|
| Settings hierarchy | Flat `ListView` with mixed destinations/actions | Identity header, Switch account, four exact titled sections, new Account/About rows, separate Sign out | BR-001, BR-002, FR-001–FR-007, FR-012, FR-018, RULE-003 | AT-001–AT-004, AT-014, UT-001, UT-002, UT-021, IT-001–IT-005, REG-001–REG-005 |
| Responsive switcher | Private compact sheet/large popover launchers in `app_shell.dart` | Extract one reusable presenter; Settings supplies a row anchor and uses the same coordinator/content/guard | FR-002, FR-005, NFR-001 | AT-002, UT-013, IT-003, REG-005 |
| About | Legal links/build label exist only in shell; cache action top-level | Shared canonical links/launcher, moved cache action, read-only shared version label | BR-004, FR-007–FR-011 | AT-004, UT-003, UT-020, IT-004, REG-003, REG-006, REG-007 |
| Account confirmation | No Account page or deletion flow | Destructive Account row, boundary dialog, fresh OAuth, exact typed handle, cancel only before acceptance | FR-012–FR-014, FR-019, RULE-009 | AT-005, UT-004, UT-005, UT-015, UT-021, UT-024, IT-005, IT-006 |
| AppView API/auth | Ordinary bearer and generic OAuth callback only | Intent/accept/status/retry/reauth/recovery APIs; purpose-aware callback; ordinary/status/recovery authorization kept separate | BR-003, FR-013, FR-019–FR-021, NFR-001, NFR-002, NFR-006, RULE-001 | AT-005–AT-008, AT-011, UT-011, UT-024, IT-006, IT-007, IT-009, IT-017, IT-028 |
| Durable deletion | No account-deletion job | Owner-unique operation, phase/checkpoint state, leases, bounded retry, attention/manual Retry, minimized audit | FR-015, FR-017, FR-020, FR-021, FR-025, FR-027, RULE-006, RULE-010 | AT-008–AT-011, UT-006–UT-008, UT-017, UT-019, IT-008, IT-011, IT-016, IT-021, IT-023, IT-027 |
| PDS boundary | PDS client can list/delete individual records for normal features | Worker-only list/delete capability over the exact record registry; expected URI before delete; final empty rescan; no blobs/account/other namespaces | FR-015, RULE-002, RULE-005, RULE-009 | AT-009, UT-009, UT-010, IT-010, IT-022, IT-024, REG-009 |
| Tap/indexers | Dispatcher handles then consumer acks; poison events may eventually be dropped | Dispatcher wrapper writes deletion receipts post-handler/pre-ack; relevant failures remain replayable; existing public deletion remains sole path | FR-023, FR-027, RULE-007, RULE-008 | AT-012, UT-008, IT-018, IT-019, IT-027, REG-008, REG-010 |
| Private AppView data | Several lifecycle services and FK cascades, no complete explicit deletion contract | Checkpointed manifest covering direct/indirect/private/object/Instagram state; schema/FK completeness gate | FR-015, FR-026, RULE-010 | AT-010, UT-018, IT-012, IT-013, IT-027 |
| Local account state | `SessionRegistry` holds only usable sessions; private cleaner currently removes Instagram snapshot | Separate secure deletion registry, full account-local purge, disabled status rows, MRU fallback/status primary | FR-016, FR-021, FR-022, FR-027 | AT-006, AT-007, UT-012–UT-016, IT-014, IT-015, IT-020, IT-025, IT-026, IT-028 |
| Rejoin | OAuth callback always initializes profile and creates bearer | Pending DID gets status-only handoff; successful prior deletion allows normal fresh membership | FR-024, RULE-011 | AT-013, UT-016, IT-020, REG-012 |
| Guidance/architecture | Blanket prohibition on CraftSky deleting PDS data | Document the narrow owner-reauthenticated `social.craftsky.*` exception before route enablement | BR-003, RULE-002, RULE-005, RULE-009 | REG-009, MAN-003 |

## 4. Files And Modules

Generated Riverpod, GoRouter, mapper, and localization files are regenerated from source and never hand-edited. Paths marked “likely” are module boundaries; the TDD builder may split a large file without changing the interfaces below.

### Flutter

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `app/lib/settings/pages/settings_page.dart` | Change | Render active identity, reusable switcher row, exact section order, disclosure icons, and separate error-coloured Sign out | FR-001–FR-006, FR-018, NFR-003–NFR-005, RULE-003 | UT-001, UT-002, UT-021–UT-023, IT-001–IT-003, REG-001, REG-004 |
| `app/lib/settings/models/settings_row.dart` | Create | Define disclosure, external-link, action, destructive-action, and read-only row kinds plus direction-aware trailing semantics | FR-003, FR-004, NFR-003 | UT-002, UT-023 |
| `app/lib/settings/widgets/settings_identity_header.dart` | Create | Project `activeAccountIdentityProvider`, reuse `AccountAvatar`, omit duplicated fallback handle, and fence stale leases | FR-001, NFR-001 | UT-001, IT-001 |
| `app/lib/auth/widgets/account_switcher_presenter.dart` | Create | Reuse compact sheet or anchored `showCraftskyContextPopover`, `AccountSwitcherContent`, activation coordinator, Add account, and MRU behavior | FR-002, FR-005, NFR-001, NFR-003 | AT-002, IT-003, REG-005 |
| `app/lib/router/app_shell.dart` | Change | Consume the extracted switcher presenter and shared Settings links/build-label helper without changing shell behavior | FR-002, FR-008, FR-009, FR-011 | IT-003, IT-004, REG-005–REG-007 |
| `app/lib/settings/pages/about_page.dart`, `about_version.dart` | Create | Terms, Privacy, cache action, and read-only shared build label with safe incomplete metadata | FR-007–FR-011 | UT-003, UT-020, IT-004 |
| `app/lib/settings/settings_links.dart` | Create | Canonical Terms, Privacy, and support URIs used by shell/About/status; one safe launcher/error path | FR-008, FR-009, FR-027 | UT-020, IT-004, IT-016, REG-007 |
| `app/lib/settings/widgets/clear_image_cache_tile.dart` | Move/reuse | Keep the same busy guard and feedback inside About; no chevron or confirmation | FR-010, RULE-003 | IT-004, REG-003 |
| `app/lib/settings/widgets/sign_out_tile.dart` | Change | Apply `colorScheme.error` to icon/label only; preserve immediate existing controller path | FR-018, RULE-003, RULE-004 | UT-021, IT-001, REG-004 |
| `app/lib/settings/pages/account_page.dart` | Create | Show Delete account as the sole destructive, non-disclosure action | FR-012 | IT-005 |
| `app/lib/settings/widgets/delete_account_dialogs.dart` | Create | First full-boundary dialog and second exact-handle field; no mutation/cancel restriction before submit | FR-013, FR-014, NFR-003, NFR-004, RULE-009 | UT-004, UT-005, IT-005, MAN-002 |
| `app/lib/settings/models/account_deletion_status.dart`, `account_deletion_intent.dart` | Create | Redacted wire/domain states, phases, retry/reauth capability, and terminal projection | FR-017, FR-020, FR-021, FR-027, NFR-006 | UT-006, UT-013, UT-019 |
| `app/lib/settings/data/account_deletion_api_client.dart`, `account_deletion_repository.dart` | Create | Call intent, accept, status, Retry, reauth/complete, cancel-intent, and recovery routes with the correct credential type | FR-013, FR-019–FR-021, NFR-002, NFR-006 | IT-005, IT-006, IT-017, IT-028, REG-011 |
| `app/lib/settings/providers/account_deletion_controller.dart` | Create | Capture active lease/handle, persist intent capability before browser launch, accept once, reconcile uncertainty, purge locally, poll status, retry/reauth, and fence late completion | FR-013, FR-016, FR-017, FR-019–FR-022, FR-027, NFR-001 | UT-012, UT-014, UT-015, IT-014–IT-016, IT-025, IT-026, IT-028 |
| `app/lib/settings/pages/deletion_reauth_complete_page.dart`, `deletion_status_page.dart` | Create | Consume proof deep link; show phase-only status, attention/Retry/support/fresh reauth, no cancel after acceptance | FR-017, FR-021, FR-027 | AT-007, AT-008, IT-016, MAN-001, MAN-002 |
| `app/lib/settings/data/deletion_status_storage.dart`, `providers/deletion_status_registry_provider.dart` | Create | Versioned secure snapshot containing only DID, job ID, status capability, minimal identity, and coarse state; no ordinary token/product data | FR-016, FR-021, FR-022, NFR-006 | UT-012–UT-014, IT-014, IT-015, IT-025 |
| `app/lib/auth/models/account_switcher_state.dart`, `account_switcher_content.dart` | Change | Merge accepted-deletion rows with ordinary rows; disable activation; open status; label `Deleting…`/`Deletion needs attention`; remove at success | FR-016, FR-027, NFR-003 | UT-013, IT-014, IT-016, REG-005 |
| `app/lib/auth/providers/account_boundary_provider.dart` | Change | Add explicit account-product purge orchestration while preserving the existing sign-out cleaner contract | FR-022, NFR-001 | UT-014, IT-014, IT-025, REG-004 |
| `app/lib/drafts/data/local_post_draft_repository.dart`, `file_local_post_draft_repository.dart` | Change | Add owner-root purge using the existing validated account path/file-store boundary | FR-022 | UT-014, IT-014, IT-025 |
| `app/lib/shared/api/providers/dio_provider.dart`, `sign_out_on_401_interceptor.dart`, `session_validation_coordinator.dart` | Change | Add status-only/recovery clients and detect `account_deletion_pending` without allowing a former bearer to call ordinary APIs | FR-021, FR-022, FR-024, NFR-006 | UT-011, UT-016, IT-017, IT-020, IT-025, REG-011 |
| `app/lib/router/route_locations.dart`, `router.dart` | Change | Add Account, About, deletion reauth-complete, and deletion-status routes; preserve lifted-shell behavior; allow status as signed-out primary | FR-005–FR-007, FR-016, FR-021, FR-024 | IT-002, IT-015, IT-020, REG-001, REG-002 |
| `app/lib/l10n/app_en.arb` and generated localization | Change/Generate | Localize all section, identity, disclosure, About, Account, boundary, status, retry, reauth, support, and semantic copy | NFR-004 | UT-005, UT-022, IT-001, IT-005, IT-016 |
| `app/test/settings/*`, `app/test/router/*`, `app/test/auth/models/*`, relevant existing notification/auth tests | Change/Create | Implement UT/IT/REG Flutter coverage exactly as mapped in `02-acceptance-tests.md` | All UI/client requirements | UT-001–UT-005, UT-012–UT-016, UT-020–UT-023, IT-001–IT-005, IT-014–IT-016, IT-020, IT-025, IT-026, IT-028, REG-001–REG-007, REG-011, REG-012 |

### AppView, data, and documentation

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `AGENTS.md`, `atproto-craft-social-app-reference.md` | Change | Replace the blanket PDS-delete ban with the exact fresh-owner/own-DID/`social.craftsky.*` exception and retain every account/namespace/blob prohibition | BR-003, RULE-002, RULE-005, RULE-009 | REG-009, MAN-003 |
| `appview/migrations/000036_account_deletion.up.sql`, `.down.sql` | Create | Add operation, status/recovery credential, expected record, receipt, cleanup checkpoint/artifact, and minimized audit tables; extend OAuth request metadata | FR-017, FR-020, FR-021, FR-023, FR-025, NFR-002, NFR-006, RULE-010 | IT-008, IT-009, IT-012, IT-017–IT-021, IT-027 |
| `appview/internal/db/account_deletion_migration_test.go` | Create | Verify constraints, cascades, partial uniqueness, minimal audit columns, indexes, and down migration | NFR-002, NFR-006, RULE-010 | IT-008, IT-021 |
| `appview/internal/accountdeletion/collections.go` | Create | Compiled `syntax.NSID` registry, profile-last order, and worker-only collection iteration | FR-015, RULE-002, RULE-005 | UT-009, UT-010, IT-024 |
| `appview/internal/accountdeletion/state.go`, `retry.go`, `terminal.go` | Create | State machine, legal transitions, fixed retry schedule, attention classification, and all terminal gates | FR-017, FR-020, FR-027, RULE-006, RULE-007 | UT-006–UT-008, IT-016, IT-019 |
| `appview/internal/accountdeletion/store.go` | Create | Owner locking, intent/job persistence, acceptance transaction, leases, phase/checkpoint writes, status/recovery hashes, receipts, audit, and exact expiry | FR-020, FR-021, FR-025, NFR-002, NFR-006, RULE-010 | IT-008, IT-009, IT-021 |
| `appview/internal/accountdeletion/status_capability.go`, `authorization.go` | Create | Generate signed opaque status capabilities, persist only hashes while active, authorize same job/owner, project terminal audit only, and deny ordinary/PDS use | FR-021, NFR-006 | UT-011, IT-017 |
| `appview/internal/accountdeletion/service.go` | Create | Create/cancel intent, validate proof/handle/status, accept atomically, status, Retry, replacement reauth, offline recovery, and pending-login policy | FR-013, FR-019–FR-021, FR-024, NFR-001, NFR-002 | UT-024, IT-006, IT-007, IT-009, IT-017, IT-020, IT-028 |
| `appview/internal/accountdeletion/pds_deleter.go` | Create | Page registered collections, register URI before delete, tolerate missing, prevent any other PDS operation, and perform final full rescan | FR-015, FR-023, RULE-002, RULE-005, RULE-009 | UT-010, IT-010, IT-011, IT-022, REG-009 |
| `appview/internal/accountdeletion/private_manifest.go`, `private_cleanup.go` | Create | Explicit delete/indexer/retain classifications, ordered idempotent cleanup components, checkpointing, orphan rules, and completeness inspection helpers | FR-015, FR-026, RULE-010 | IT-012, IT-013, IT-027 |
| `appview/internal/accountdeletion/convergence.go`, `convergence_indexer.go` | Create | Post-handler receipt write, must-replay error, expected/receipt/absence verification, and no eager public mutation | FR-023, FR-027, RULE-007, RULE-008 | IT-018, IT-019, REG-008, REG-010 |
| `appview/internal/accountdeletion/worker.go`, `audit_sweeper.go`, `observability.go` | Create | Lease/claim/process phases, retry/attention, exact finalization, audit expiry, and coarse redacted telemetry | FR-017, FR-020, FR-025, FR-027, NFR-006 | UT-006–UT-008, UT-017, UT-019, IT-011, IT-016, IT-021, IT-023, IT-027 |
| `appview/internal/auth/store.go`, `handlers_session.go`, `handlers_oauth.go`, `background_session_selector.go` | Change | Atomically persist auth-request purpose metadata, route deletion callbacks, return pending-login status handoff, exclude bound OAuth from cleanup/selection, and never create an ordinary bearer for deletion OAuth | FR-019–FR-021, FR-024, NFR-006 | UT-011, UT-024, IT-006, IT-009, IT-017, IT-020, REG-011 |
| `appview/internal/auth/pds_client.go` | Change | Define a deletion-specific interface exposing only `ListRecords` and `DeleteRecord`; production adapter reuses current implementations | FR-015, RULE-002, RULE-009 | UT-010, IT-010, IT-022, REG-009 |
| `appview/internal/tap/consumer.go` | Change | Add a classified `mustReplay` handler error that bypasses poison-event acknowledgement for tracked deletion events | FR-023, RULE-007 | IT-018, IT-019 |
| `appview/internal/api/account_deletion.go` | Create | Decode/validate camelCase DTOs and map domain errors to standard envelopes/statuses without accepting a target DID | FR-013, FR-017, FR-021, NFR-002 | IT-007, IT-017, IT-028 |
| `appview/internal/middleware/account_deletion_status.go`, `account_deletion_recovery.go` | Create | Separate status capability and former-bearer recovery auth from ordinary `Authenticated` middleware | FR-021, FR-022, NFR-006 | UT-011, IT-017, IT-025 |
| `appview/internal/routes/policy.go`, `routes.go`, route tests | Change | Declare/register intent, accept, status, Retry, reauth, completion, cancel, and recovery policies with correct auth/body/rate classes | FR-013, FR-017, FR-019–FR-021, NFR-002 | IT-007, IT-017, IT-028 |
| `appview/internal/app/config.go`, `deps.go`, `appview/cmd/appview/main.go`, environment examples | Change | Validate status HMAC secret, wire services/observer/worker, and run deletion/audit loops beside existing workers | FR-017, FR-020, FR-021, NFR-006 | IT-008, IT-011, IT-023, IT-027 |
| `appview/internal/notifications/actor_deletion.go`, notification store cleanup | Change | Add recipient/preferences/seen/subscription/delivery hard-delete transaction and orphan-only installation handling | FR-015, FR-026 | IT-009, IT-012 |
| `appview/internal/scheduledposts/account_deletion.go` | Change | Return cleanup job IDs for explicit deletion and expose completion verification while retaining idempotent profile/identity lifecycle behavior | FR-015, FR-026 | IT-012, IT-027, REG-008 |
| `appview/internal/instagram/account_data.go` | Change/test | Complete any TD-009 gaps in explicit `PurgeOwner`, including orphan handles, rate buckets, audits, notification joins/effects, and claim release | FR-026, RULE-010 | UT-018, IT-013, IT-027 |
| `appview/internal/accountdeletion/*_test.go`, auth/route/index existing suites | Create/Change | Implement server tests from UT-006–UT-011, UT-017–UT-019, UT-024, IT-006–IT-013, and IT-016–IT-024/IT-027 | All server requirements | Mapped IDs above |

No file under `lexicon/` or `appview/internal/lexicon/craftsky/` changes in this feature. `UT-009` and `IT-024` read the Lexicon sources only.

## 5. Services, Interfaces, And Data Flow

### 5.1 HTTP contract

All bodies use camelCase and every error uses `{error, message, requestId}`.

| Method and path | Authorization | Request | Success | Important errors |
|---|---|---|---|---|
| `POST /v1/account-deletion/intents` | Ordinary bearer + device ID + current member | No body | `201 {jobId,statusToken,authUrl,expiresAt}` | `409 deletion_already_pending`; `502 authorization_server_unavailable` |
| `DELETE /v1/account-deletion/intents/{jobId}` | Ordinary bearer + `X-Craftsky-Deletion-Status` | No body | `204` | `409 deletion_already_accepted`; idempotent `204` if already canceled/expired |
| `POST /v1/account-deletions/{jobId}` | Ordinary bearer + `X-Craftsky-Deletion-Status` | `{reauthProof, confirmationHandle}` | `202 {jobId,status,phase}` | `400 confirmation_handle_mismatch`; `401 reauthentication_required`; `409 deletion_already_pending` |
| `GET /v1/account-deletions/{jobId}` | `Authorization: DeletionStatus <token>` + device ID | No body | `200 {jobId,status,phase,canRetry,needsReauthentication,supportUrl}` | `401 invalid_deletion_status`; `410 deletion_status_expired` |
| `POST /v1/account-deletions/{jobId}/retry` | DeletionStatus | No body | `202` current status | `409 deletion_retry_unavailable` |
| `POST /v1/account-deletions/{jobId}/reauth` | DeletionStatus | No body | `200 {authUrl,expiresAt}` | `409 deletion_reauth_unavailable`; `502 authorization_server_unavailable` |
| `POST /v1/account-deletions/{jobId}/reauth/complete` | DeletionStatus | `{reauthProof}` | `202` current status | `401 reauthentication_required`; `409 reauthentication_replayed` |
| `POST /v1/account-deletions/recover` | Former bearer + device ID, recovery middleware only | No body | `200 {jobId,statusToken,status,phase}` | `401 invalid_deletion_recovery`; `410 deletion_status_expired` |

The client never supplies a DID. `{jobId}` is a UUID resource identifier, not authority. The ordinary bearer establishes the owner during intent/accept; status/recovery middleware independently verifies the same stored owner/job. Rate class is `write` for intent/accept/cancel/retry/reauth/recovery and `read` for status. Intent, accept, reauth, and recovery get tighter per-DID/device limits within the existing rate-limiter configuration.

### 5.2 OAuth intent and callback

```text
captured active lease + exact handle
  -> create owner-unique intent and status capability (10 minute TTL)
  -> StartAuthFlow(context carrying purpose=accountDeletion, owner DID, job ID)
  -> PostgresAuthStore atomically inserts purpose metadata with indigo auth request
  -> browser authenticates at the PDS/authorization server
  -> callback loads purpose metadata before ProcessCallback consumes auth request
  -> validate callback AccountDID == intent owner
  -> store OAuth session ID + SHA-256(one-time proof), never a CraftSky bearer
  -> craftsky:///account-deletion/reauth-complete?jobId=...&proof=...
```

The proof expires with the intent, is single-use, is never logged, and is cleared at acceptance. Cancel/expiry deletes the intent, status hash, proof, and any OAuth session created for it. A normal login callback checks active deletion before profile initialization: pending/attention mints status-only access and deletes the newly created unbound OAuth session; a successful prior audit does not block fresh membership initialization.

### 5.3 Acceptance transaction

```text
lock operation by job ID + owner
  -> verify intent, ordinary bearer owner, status hash, proof hash/TTL, exact handle hash
  -> lock fresh oauth_sessions row for the same DID/session ID
  -> mark accepted and bind deletion_oauth_session_id
  -> copy ordinary bearer hashes to account_deletion_recovery_credentials
  -> hard-delete account subscriptions/deliveries and orphan installations
  -> hard-delete all craftsky_sessions for owner
  -> delete all oauth_sessions for owner except the bound session
  -> clear proof/handle hashes and schedule worker now
  -> commit
```

The operation's FK/reference prevents accidental deletion of the bound OAuth row. Generic OAuth lazy cleanup excludes bound sessions. `BackgroundSessionSelector` still requires an unrevoked ordinary bearer and additionally excludes a bound session, so no normal background writer can select it. The deletion worker obtains the session only through `BoundOAuthAuthority.Resume(jobID, ownerDID)` and receives only the narrow list/delete PDS interface.

### 5.4 Durable schema

`000036_account_deletion` adds:

- `account_deletion_operations`: UUID ID; owner DID; state/phase; accepted timestamp; bound OAuth session ID; attempt/next-attempt/error code; worker lease; intent/proof/handle hashes and expiry only before acceptance; final-rescan timestamp; created/updated timestamps. A unique owner constraint permits one intent or accepted operation per DID.
- `account_deletion_status_credentials`: token hash, job ID, owner DID, device ID, creation/revocation metadata. Plaintext is returned once and never persisted.
- `account_deletion_recovery_credentials`: hash of each former bearer, job/owner/device binding, creation time. These are status-exchange credentials, not ordinary sessions, and cascade at success.
- `account_deletion_expected_records`: `(job_id, uri)` primary key, collection, registration/delete-request timestamps. No CID or record body.
- `account_deletion_index_receipts`: `(job_id, uri)` primary key, collection, Tap event ID, repo revision, handled timestamp. Extra owner/job-scoped delete observations are allowed and ignored unless joined to an expected URI.
- `account_deletion_cleanup_steps`: `(job_id, component)` and completed timestamp.
- `account_deletion_cleanup_artifacts`: job ID plus scheduled cleanup-job UUID only; no object key/content.
- `account_deletion_audits`: exactly job ID, owner DID, accepted timestamp, terminal-success timestamp, coarse outcome, and expiry timestamp. No FK to membership and no uniqueness that blocks rejoin.
- `oauth_auth_requests` sibling columns for purpose, expected account DID, and deletion job ID. Normal login defaults remain backward compatible.

Deleting an operation cascades all non-audit operational rows. Audit expiry uses `DELETE ... WHERE expires_at <= now`, proving the exact boundary. Migration tests reject additional token, handle, URI, content, relationship, preference, or settings columns in the audit table.

### 5.5 Worker phases

```text
queued
  -> removingPrivateData
  -> removingCraftskyRecords
  -> waitingForIndexerConvergence
  -> finalizing
  -> deleted audit only

any retryable phase -> retrying -> same phase
unusable OAuth or retry exhaustion -> needsAttention
manual Retry -> same durable job/phase
replacement fresh OAuth -> swap bound session -> same durable job/phase
```

- Private cleanup runs the manifest components in deterministic order and checkpoints each successful component. A failed later component does not repeat completed external effects unnecessarily, but every component remains idempotent if replayed.
- PDS deletion loops each registry collection with page size 100. It parses every AT-URI and derives the record key server-side. Expected-URI persistence commits before `DeleteRecord`. `ErrRecordNotFound` is success for the PDS side effect, not terminal success.
- Convergence compares the expected set to receipts and explicit absence queries for profile/post/like/repost/project/mention/derived notification state. It never deletes or hides those rows.
- Finalization performs one more full PDS rescan immediately before authority removal. If it finds a record, the worker returns to record deletion. Once empty and all local gates pass, OAuth logout/removal is attempted and local authority is deleted. A retry after authority removal may only finish the already-qualified terminal transaction; it may not restore OAuth or ordinary access.
- Audit cleanup is independent of fresh rejoin and uses the injected clock.

### 5.6 Tap receipt flow

```text
Tap record delete
  -> ConvergenceIndexer determines whether owner has an active deletion operation
  -> existing Dispatcher.Handle(event)
       -> existing profile/post/like/repost indexer and notification lifecycle
  -> ReceiptStore.Record(job, URI, collection, event ID, rev)
  -> return nil
  -> WSConsumer ack
```

Duplicate/reordered events upsert only by `(job, URI)` and retain a deterministic receipt (highest event ID/revision observation). If relevant index handling, active-job lookup, or receipt persistence fails, the wrapper returns `tap.ErrMustReplay`; `WSConsumer` does not increment toward poison-drop or ack that event. Non-deletion and unrelated events preserve current poison behavior.

### 5.7 Private cleanup manifest

The manifest is code, not prose-only documentation. Each entry includes ownership path, policy, cleanup component, and verification query. It covers:

- sessions/auth requests/status recovery and notification subscriptions;
- recent searches, mutes, saved folders/posts, language preferences, pins, and adjacent customisation storage;
- received notification events, preferences, seen state, subscription/delivery rows, and orphan-only installations;
- scheduled posts/media/tombstones, private objects, and cleanup jobs;
- moderation reporter/source/subject rows;
- every Instagram table and indirect graph named in TD-009, including claim release, orphan-only imported handles, owner-derived rate buckets/audits, PDS follow ledgers, automatic-follow sources, and notification effects;
- public/indexer-owned CraftSky tables, classified but never directly purged by this service;
- retained shared/public caches and non-CraftSky indexes such as identity cache, Bluesky profiles, follows, and blocks.

The completeness test queries the migrated test database's columns and foreign keys. Every table with `did` or `*_did`, and every table reachable by FK from a classified owner-private table, must appear exactly once with an explicit policy. External object storage and adjacent customisation are explicit non-schema manifest entries. This conservative gate may require classifying a new public table, which is intentional.

## 6. State, Providers, Controllers, And Dependency Injection

### Flutter state

```text
sessionRegistryProvider                    deletionStatusRegistryProvider
  -> ordinary retained sessions              -> intent/accepted status capabilities
  -> active lease/MRU                         -> minimal DID/handle/display/avatar/job state
       \                                     /
        -> accountSwitcherStateProvider (ordinary + disabled deleting rows)

activeAccountIdentityProvider
  -> SettingsIdentityHeader

accountDeletionControllerProvider
  -> captured ActiveAccountLease + exact handle
  -> accountDeletionRepositoryProvider(account/status target)
  -> authUrlLauncherProvider
  -> deletionStatusRegistryProvider
  -> accountProductDataCleanerProvider
  -> sessionRegistryProvider.removeConfirmed
  -> accountHomeReset/status routing
```

Rules:

- The controller captures `AccountSessionLease` before intent creation and rechecks it before showing dialogs, launching OAuth, accepting, mutating local state, navigating, or showing feedback.
- Status updates are keyed by redacted job/account keys, not raw DID/token strings in provider diagnostics.
- The status registry is persisted before ordinary local session removal. A failed secure write does not publish partial deletion state; recovery remains possible through normal pending-login OAuth or a former-bearer exchange.
- `AccountProductDataCleaner` purges the account's draft root and Instagram verification snapshot, clears image caches through the existing cache provider, invalidates account-scoped Riverpod state, and removes the ordinary session. It preserves only the status registry record.
- The status poller runs while a status page/row is observed and on app resume; it uses bounded backoff and stops at terminal/attention. Server worker retry remains authoritative.
- The existing auth/session provider treats accepted deletion rows as non-authenticated. If no ordinary session remains, router state is signed out with deletion status as an allowed primary route.

### AppView dependencies

`app.Deps` gains explicit account-deletion service, status/recovery authenticators, convergence wrapper, worker, and audit sweeper fields. Interfaces are arranged to avoid cycles:

- `auth` defines the narrow callback/pending-login interface; `accountdeletion.Service` implements it while importing existing auth PDS/OAuth abstractions.
- `middleware` defines status/recovery authenticator interfaces; the account-deletion store implements them.
- `accountdeletion.ConvergenceIndexer` accepts a `tap.HandlerIndexer`; Tap and index packages do not import account deletion.
- API handlers depend on service interfaces and the envelope package remains transport-only.
- `BoundOAuthAuthority` verifies the job/owner/session reference before constructing a PDS client; the generic background selector is never injected into the deletion worker.

Configuration:

- Add `ACCOUNT_DELETION_STATUS_HMAC_KEY`, at least 32 random bytes, required in production and set to an explicit local-only value in `appview/environments/dev.env`.
- Add validated optional defaults for poll interval `2s`, batch `20`, lease `2m`, and intent/proof TTL `10m`. Retry offsets stay code constants so tests and operations share one policy.
- Start the worker and audit sweeper in `cmd/appview/main.go` using the same cancellation/wait pattern as existing workers.

## 7. UI, Widgets, Routes, And User-Facing Surfaces

### Main Settings

The scrollable content uses the existing theme/spacing and a centered maximum width consistent with other settings pages:

1. Active avatar + display name + `@handle`; blank display name shows `@handle` once.
2. Switch account disclosure row immediately below identity.
3. **Preferences:** Customisation, Languages, Notifications.
4. **Connections:** Followers, Following, Muted accounts, Blocked accounts.
5. **Discovery:** Find people from Instagram.
6. **General:** Account, About.
7. Separate Sign out action.

Every in-app destination and Switch account has a direction-aware forward chevron. Terms/Privacy use external-link icons. Clear image cache, Delete account, Sign out, and build version have no chevron. Sign out's icon and label use `colorScheme.error`; it remains an immediate non-destructive sign-out.

The Customisation route is a merge dependency, not a placeholder. Before implementing this list, rebase/merge the adjacent customisation work, add its typed route case, and run REG-001. Do not ship a dead row or silently omit it.

### About

- Terms and Privacy open `https://craftsky.social/terms` and `https://craftsky.social/privacy` through the same external launcher/error path as the shell.
- Clear image cache embeds the existing immediate tile and existing success/error behavior.
- App version reads `packageInfoProvider` and calls the same `navigationBuildVersion(version, buildNumber)` localization. Empty build metadata renders only available text without malformed punctuation.

### Account and confirmation

- Account initially contains only Delete account, styled destructive with no disclosure icon.
- First dialog names the captured `@handle` and states: CraftSky membership/private data and all `social.craftsky.*` records are permanently removed; every device is signed out; the AT/PDS account and other namespaces remain; CraftSky does not directly delete blobs or wait for PDS garbage collection; offline devices erase local data on next contact; and deletion cannot be undone after submission.
- After fresh OAuth succeeds, the second dialog requires the exact captured handle (without accepting display name, DID, aliases, whitespace/case variants, or another account). The destructive submit remains disabled until exact.
- Cancel/dismiss before submit deletes the intent best-effort and performs no membership/product deletion. After accepted submit, no cancel/pause action exists.

### Deletion status and switcher

- Member-visible phase labels are coarse: Preparing deletion, Removing private CraftSky data, Removing CraftSky records, Waiting for CraftSky to update, Finalizing, Retrying, Deletion needs attention, and Deleted. Do not show counts, URIs, record types, private-store names, OAuth state, or raw errors.
- Attention shows Retry when permitted, fresh reauthentication when authority is unusable, and the support link. Retry uses the same job.
- With another account active, the deleting account remains a disabled switcher row. Tapping it opens status but cannot activate it.
- With no ordinary account, `/account-deletion/:jobId` is the primary signed-out route.
- Terminal success removes the row/status record after the device observes the terminal audit response. After audit expiry, a stale local status entry clears and normal Welcome is shown.

### Routes

- Settings child routes: `/profile/settings/account` and `/profile/settings/about`, lifted to the authenticated shell navigator like Languages.
- Notification row keeps the existing `/notifications/settings` typed destination.
- Root routes: `/account-deletion/reauth-complete` and `/account-deletion/:jobId`, allowed without an ordinary session but requiring local status context.
- Back from Account/About returns to Settings. Back never returns an accepted deleting account to ordinary Settings.

## 8. Error, Loading, Empty, And Edge States

| State / Case | Planned Handling | Requirement IDs | Test IDs |
|---|---|---|---|
| Identity loading or stale completion | Render cached/fallback active identity; accept async profile only for same lease; never show DID/other account | FR-001, NFR-001 | UT-001, IT-001 |
| One retained account | Switcher still opens with current selected and Add account available | FR-002 | AT-002, REG-005 |
| Five retained accounts | Reused disabled Add account/helper remains | FR-002 | REG-005 |
| LTR/RTL | `Icons.arrow_forward_ios`/directional semantics point forward; row order unchanged | FR-004, NFR-003 | UT-002, UT-023, IT-001 |
| Link false/throw | Stay on About/status and show existing localized safe error | FR-008, FR-009, FR-027 | UT-020, IT-004, REG-007 |
| Cache repeated tap/failure | Existing AsyncLoading guard runs once and uses safe feedback | FR-010 | IT-004, REG-003 |
| Missing build number/metadata | Render available version or omit label; no punctuation crash/interactive row | FR-011 | UT-003, IT-004 |
| Reauth canceled/expired/wrong DID | Delete/expire intent and OAuth session; typed submit unavailable; no accepted job | FR-013, FR-019, NFR-001 | UT-024, IT-005, IT-006 |
| Handle mismatch | Reject locally and server-side with no mutation | FR-013 | UT-004, IT-005, IT-007 |
| Acceptance response lost | Poll pre-persisted status capability/job ID; server returns same accepted operation | FR-020, NFR-002 | IT-028 |
| Another account activates mid-flow | All completion/local cleanup/navigation remains bound to original lease/job | NFR-001, RULE-001 | UT-015, IT-026 |
| Other account remains at acceptance | Persist status, purge deleting account, activate MRU remaining, expose disabled status row | FR-016, FR-022 | UT-012–UT-014, IT-014 |
| No account remains | Persist status, purge ordinary state, route directly to status | FR-016, FR-021, FR-022 | IT-015 |
| Offline secondary device | Ordinary call gets `account_deletion_pending`; exchange former bearer once; purge locally; never regain ordinary access | FR-022 | IT-025, MAN-004 |
| Duplicate/reordered PDS/Tap work | Expected/receipt/upsert/delete/checkpoint operations are idempotent | FR-020, FR-023, NFR-002 | IT-011, IT-018, IT-019, IT-027 |
| Record already missing | PDS side effect converges; absence/receipt/final gates still decide terminal state | FR-015, NFR-002 | UT-010, IT-010, IT-011 |
| New record during deletion | Final rescan registers/deletes it and returns to convergence | FR-015, RULE-005 | IT-019, IT-024 |
| Tap/indexer or receipt unavailable | No ack for tracked event; job stays convergence/retry, then attention if exhausted | FR-017, FR-023, RULE-007 | IT-016, IT-018, IT-019 |
| Private component partial failure | Keep completed checkpoints; retry same owner/job; no false success/ordinary restore | FR-015, FR-017, FR-020 | IT-011, IT-012 |
| Bound OAuth unusable | Attention with fresh reauth; replacement swaps exact binding and never creates bearer | FR-017, FR-019, FR-021 | UT-007, UT-024, IT-006, IT-016 |
| Automatic retry exhausted | `needsAttention`, support and authorized Retry; no cancel | FR-017, FR-027 | UT-006, UT-007, IT-016 |
| Status token cross-owner/job/ordinary use | Deny standardly; never expose OAuth/session details | FR-021, NFR-006 | UT-011, IT-017 |
| PDS final rescan non-empty | Return to deletion; do not remove OAuth or report success | FR-015, FR-023 | UT-008, IT-019 |
| Scheduled object cleanup pending | Keep private cleanup/finalization incomplete until cleanup job IDs disappear | FR-015, FR-026 | IT-012, IT-027 |
| Terminal success | Insert minimal audit, delete all operational/status/OAuth state, then client clears local status | FR-021, FR-025, NFR-006 | UT-008, UT-017, UT-019, IT-021, IT-027 |
| Audit boundary/rejoin | Audit exists strictly before +30d, absent at/after; fresh membership unaffected | FR-024, FR-025, RULE-011 | UT-016, UT-017, IT-020, IT-021, REG-012 |

## 9. Test Implementation Plan

Follow the approved `02-acceptance-tests.md` IDs and fixtures; do not replace them with a smaller ad hoc suite.

| Order | Test IDs | Target | Initial Expected Failure / Purpose |
|---|---|---|---|
| 1 | UT-009, IT-024 | `accountdeletion/collections_test.go` | No compiled exact record registry or Lexicon drift gate exists. This is the first red test. |
| 2 | UT-001–UT-005, UT-020–UT-023 | Settings pure/widget tests | Identity, row kinds/order, version fallback, exact handle, boundary copy, localization, and semantics are absent. |
| 3 | REG-001 | Settings route regression | Customisation must be merged and every preserved row routed once before the new hub is considered complete. |
| 4 | UT-006–UT-008, UT-017, UT-019 | Server state/retry/terminal/audit/observability unit tests | No deletion state machine, bounded retry, terminal gate, or minimized audit projection exists. |
| 5 | UT-010, UT-011, UT-018, UT-024 | PDS/status/Instagram/OAuth unit tests | Namespace restriction, credential separation, explicit Instagram purge, and fresh proof binding are not enforced. |
| 6 | IT-006, IT-007 | OAuth/route contracts | Callback cannot create deletion proofs and no authenticated owner-scoped acceptance endpoint exists. |
| 7 | IT-008, IT-009, IT-017, IT-021, IT-023 | Migration/store/acceptance/auth/audit/telemetry | Durable minimized schema, bind-before-revoke, restricted status, expiry, and redaction do not exist. |
| 8 | IT-010, IT-011, IT-022 | PDS deleter/failure/blob boundary | No expected-before-delete pagination/restart loop or hard blob/account prohibition exists. |
| 9 | IT-012, IT-013 | Private manifest and Instagram | Current lifecycle coverage is incomplete and has no schema/FK drift gate. |
| 10 | IT-016 | Worker retry + Flutter status | Retry exhaustion and manual Retry/reauth/support have no end-to-end state. |
| 11 | IT-018, REG-008 | Existing indexer replay plus receipt observer | Receipts are not post-handler/pre-ack and tracked failures can poison-drop. |
| 12 | IT-019 | Convergence terminal gate | Worker cannot wait on receipt + indexed absence + final rescan + OAuth removal. |
| 13 | IT-027 | Full server acceptance | Prove one restart and duplicate/reordered event across all server phases and preservation controls. |
| 14 | IT-001–IT-005, REG-002–REG-007 | Settings/About/Account/routing | New surfaces and reuse contracts are absent. |
| 15 | UT-012–UT-016, IT-014, IT-015, IT-025, IT-026, IT-028 | Flutter secure status, cleanup, fallback, recovery, lease, uncertainty | Client has no separate deletion state or cross-device recovery. |
| 16 | IT-020, REG-011, REG-012 | Pending login, credential scan, fresh rejoin | OAuth always creates an ordinary session/membership and no forbidden-credential scan covers deletion. |
| 17 | REG-009, REG-010 | PDS/indexer architectural regressions | Prove no whole-account/blob/other namespace call and no eager hide/direct purge. |
| 18 | MAN-001–MAN-004 | Supported layouts, assistive tech, disposable real stack, offline device | Complete the controlled release gates after implementation review. |

Focused first-red command from `appview/`:

```sh
go test ./internal/accountdeletion -run 'TestCraftskyRecordCollections'
```

Focused Flutter commands from `app/` as the client steps land:

```sh
flutter test test/settings
flutter test test/router/settings_routes_test.dart test/router/notification_settings_route_test.dart
flutter test test/auth/models/account_switcher_state_test.dart
dart analyze lib/settings lib/router lib/auth test/settings test/router
```

Generation and full verification:

```sh
# from app/
dart run build_runner build
flutter gen-l10n
flutter analyze
flutter test

# from repository root with compose dependencies running
just test
```

`MAN-003` uses only a disposable development AT account. Seed current CraftSky and non-CraftSky records and at least one shared blob reference; restart the app/worker once; inspect PDS, Tap receipts, AppView, ordinary-session rejection, OAuth removal, and fresh rejoin. Never run it against a personal or production account.

## 10. Sequencing And Guardrails

### Required sequence

1. Write and make red `UT-009`; implement the exact collection registry only.
2. Amend `AGENTS.md` and the architecture reference with the narrow approved exception; keep all broader prohibitions.
3. Merge/rebase adjacent Customisation route work and establish REG-001; do not create a placeholder destination.
4. Implement non-destructive Settings/About/Account structure, localization, routing, switcher extraction, cache/legal/version reuse, and Sign out colour.
5. Implement pure server state/retry/terminal/status authorization and the migration before any route calls PDS deletion.
6. Add purpose-aware OAuth intent/callback/proof and prove wrong-DID/replay/expiry rejection.
7. Implement and test the atomic acceptance boundary, recovery hashes, bound-session exclusion, and status-only routes.
8. Add the narrow PDS deleter and prove account/blob/namespace methods are impossible through its interface.
9. Add private manifest/checkpoint cleanup and its schema/FK completeness test.
10. Add post-handler/pre-ack convergence receipts and must-replay behavior; keep existing indexers unchanged except for required test fixtures/interfaces.
11. Complete worker/finalization/audit expiry and full AppView acceptance test.
12. Add Flutter secure status state, local purge, disabled switcher rows, signed-out status route, offline recovery, pending-login policy, and stale-lease fencing.
13. Run full regressions, implementation review, then MAN-001–MAN-004.

### Guardrails

- Never accept a target DID from Flutter. Owner DID comes only from ordinary auth, then must match intent, proof, OAuth session, job, and status/recovery capability.
- Never create an ordinary CraftSky bearer from a deletion-purpose OAuth callback or replacement reauth.
- Bind the fresh OAuth session inside the acceptance transaction before deleting any bearer/unbound OAuth row.
- A bound OAuth session is server-only, worker-only, excluded from generic background selection/lazy cleanup, and usable only for list/delete against the registered CraftSky collections.
- Persist each expected URI before its delete. Never infer success from a successful HTTP delete call alone.
- A tracked indexer/receipt failure is never acknowledged by poison-pill handling. Receipt persistence happens after existing indexer success and before ack.
- Public/indexed deletion remains entirely indexer-driven. Do not add read filters, eager purge, tombstone hiding, or a second notification retraction path.
- Terminal success requires all gates in UT-008. Blob garbage collection is explicitly not one of them.
- Checkpoint private cleanup, but delete every checkpoint/artifact/URI/receipt/status/recovery row at terminal success.
- The post-success audit schema is closed: only DID, job ID, timestamps, coarse outcome, and exact expiry are allowed.
- Never log/stringify handles, DIDs, ordinary/status/OAuth tokens, proofs, expected URIs, event revisions, record content, relationship/settings/import data, or full URLs. Telemetry uses only coarse phase/outcome/error categories.
- Status/recovery clients never install the ordinary auth interceptor or gain a PDS client. Former bearer recovery is one-time and status-only.
- Local status persistence precedes local ordinary-session removal; active-account lease checks precede all UI mutation/navigation/feedback.
- Sign out continues to call only the existing session lifecycle. Delete account never reuses the Sign out callback.
- No Lexicon/schema generation is required because no Lexicon changes. If the collection test reveals a real adjacent Lexicon record, add it to the compiled registry; do not alter that Lexicon in this feature.
- `MAN-003` and implementation review are release gates. Automated green tests do not authorize production destructive testing.

### Explicitly out of scope

- Deleting/deactivating the DID, PDS account, general AT identity, non-CraftSky records, or blobs.
- Waiting for or controlling PDS garbage collection.
- Canceling an accepted deletion, pausing it, restoring ordinary access, or restoring deleted product data.
- Export, password/handle management, session-management UI, Sign out all, update checking, in-app legal browser, or redesign of existing destination pages.
- Changing ordinary Instagram retention behavior outside explicit account deletion.
- Editing any Lexicon or adding a reusable generic PDS deletion facility.

## 11. Risks And Open Questions

| Risk / Question | Decision / Mitigation | Requirement IDs | Test / Gate |
|---|---|---|---|
| Repository guidance contradicts approved behavior | Mandatory step 2 amends both guidance sources before route enablement and preserves whole-account/namespace/blob bans | RULE-002, RULE-005, RULE-009 | REG-009, MAN-003 |
| Customisation route absent in checkout | Mandatory merge gate before main Settings implementation; no placeholder or omission | FR-003, FR-004 | REG-001 |
| OAuth callback metadata race | Write purpose/owner/job columns atomically in `SaveAuthRequestInfo` via context, not the current follow-up handoff update | FR-019, NFR-001 | UT-024, IT-006 |
| Acceptance response lost after bearer revocation | Device stores job/status capability before opening OAuth; acceptance is idempotent and status resolves the same operation | FR-020 | IT-028 |
| Bound OAuth selected by another worker | Explicit FK/reference, selector exclusion, no backing bearer, and worker-only factory validation | FR-020, FR-021 | UT-011, IT-009, IT-017 |
| OAuth expires mid-job | Mark attention, require status-authorized fresh reauth, swap exact binding, never mint ordinary bearer | FR-017, FR-019, FR-021 | UT-007, UT-024, IT-006, IT-016 |
| External delete races expected registration | Owner/job-scoped post-index delete observations may precede expected registration; terminal joins only expected URIs and still verifies indexed absence/final rescan | FR-023, RULE-007 | IT-011, IT-018, IT-019 |
| Current Tap poison drop could ack a failed receipt | `ErrMustReplay` bypasses retry-count/drop for tracked deletion events | FR-023, RULE-007 | IT-018 |
| New private table escapes cleanup | Conservative DID/FK schema introspection requires every discovered surface to be classified; external stores are explicit manifest entries | FR-026, RULE-010 | IT-012 |
| Profile delete cascade races explicit cleanup | Private cleanup precedes PDS deletion and profile is deleted last; all components/indexers remain idempotent if events reorder | FR-015, FR-023 | IT-011, IT-012, IT-027 |
| Scheduled media row disappears before object deletion | Record cleanup-job UUIDs as operation artifacts and block terminal success until those jobs are gone | FR-015, FR-026 | IT-012, IT-027 |
| User writes a new CraftSky record during deletion | Ordinary CraftSky sessions are gone; final PDS rescan loops any externally created record back through delete/convergence | FR-015, FR-023 | IT-019 |
| Status capability exists after operational row removal | While active, token hash is required; after success, its signed job/owner claims may read only the allowed coarse audit until expiry, then returns `410` | FR-021, FR-025, NFR-006 | UT-011, UT-017, IT-017, IT-021 |
| Offline device still has a former bearer | Former hash is status-recovery-only; ordinary auth returns pending/denied, one exchange mints status and triggers local purge | FR-022 | IT-025, MAN-004 |
| Support destination changes | Use the repository's existing public discussion channel through one centralized URI constant; update only that constant/copy if a dedicated support surface lands | FR-027 | IT-016, MAN-003 |
| Real PDS/OAuth/Tap provider variation | Exhaust deterministic fakes, then require disposable real-stack MAN-003; never use personal/production identity | FR-015, FR-019, FR-023 | MAN-003 |
| Physical offline storage/backups cannot be remotely erased | Copy states next-contact cleanup only; client purges on first learned pending state | FR-014, FR-022 | UT-005, IT-025, MAN-004 |

No open product decision blocks TDD implementation. The support URL and Customisation merge are explicit pre-release/pre-UI gates, not authorization ambiguities.

## 12. Handoff To TDD Builder

- Requirements: `docs/changes/2026-08-10-settings-page/01-requirements.md`.
- Acceptance tests: `docs/changes/2026-08-10-settings-page/02-acceptance-tests.md`.
- Document review: `docs/changes/2026-08-10-settings-page/03-document-review.md`.
- Coding plan: `docs/changes/2026-08-10-settings-page/04-coding-plan.md`.
- Risk level: High.
- First red test: `UT-009` in `appview/internal/accountdeletion/collections_test.go`.
- First implementation increment: only the compiled collection registry required to make UT-009 green. Do not begin PDS deletion, routes, OAuth mutation, Flutter UI, or migration in that increment.
- Required implementation workflow: strict red-green-refactor through the order in section 9, recording actual evidence in `05-implementation-plan.md` when `implement-tdd` is invoked.
- Mandatory pre-destructive gate: guidance/reference amendment committed in scope and namespace/account/blob regression tests green before registering the acceptance route.
- Mandatory release gates: implementation review, full Flutter/AppView suites, REG-001 through REG-012, and MAN-001 through MAN-004, with MAN-003 restricted to a disposable development account.
- Stage exit options:
  1. Continue with `implement-tdd` using this approved plan.
  2. Add a manual note/change to this coding plan before implementation.
  3. Stop after coding planning.
