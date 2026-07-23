# Coding Plan: Instagram DM Ownership Verification And Follow Discovery

## 1. Inputs

- Approved requirements: `01-requirements.md`
- Approved acceptance tests: `02-acceptance-tests.md`
- Approved document review: `03-document-review.md`
- Earlier design context: `design-plan.md`; when it differs, `01` and `02`
  are authoritative.
- Repository guidance: `AGENTS.md`
- Architecture references:
  - `atproto-craft-social-app-reference.md`
  - `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md`
  - `docs/superpowers/specs/2026-04-22-api-wire-alignment-design.md`
- Existing implementation inspected:
  - AppView route policies, authentication middleware, request body and rate
    limiting, API errors, cancellation handling, and dependency wiring
  - pgx stores, migrations, notification lifecycle/newness/push dispatcher,
    profile membership lifecycle, Tap identity deletion, and PDS follow writes
  - Flutter fixed-account Dio providers, `ActiveAccountLease` operation fencing,
    account-boundary invalidation, notification models/providers, generated
    routing, settings UI, localization, and provider logging
- Approval gate: on 2026-07-19 the user confirmed that the request authorizes
  creating the missing workflow artifacts and implementing all feasible AppView
  and Flutter work without a configured Instagram app.
- Out-of-scope authority: no commit, push, Meta dashboard mutation, production
  enablement, or collection of real/user-derived Instagram fixtures.

## 2. Implementation Strategy

Build the feature as private AppView data with a narrow Meta adapter and a
fixed-account Flutter client. There is no lexicon change: only an explicitly
accepted suggestion writes an ordinary `app.bsky.graph.follow` record to the
user's PDS.

Implementation proceeds in independently green vertical slices:

1. Establish the cross-language constants and golden wire fixtures, challenge
   codec, disabled/partial/full configuration semantics, and private schema.
2. Add one shared current-member guard and the attempt/link state machines.
3. Accept signed webhooks into a minimal durable queue, then process them with
   an injected Meta client, bounded leases, retries, and privacy-safe terminal
   clearing.
4. Add local JSON imports, exact matching, the shared fail-closed eligibility
   policy, reconciliation jobs, and deterministic suggestion lifecycle.
5. Extract a stable-rkey follow service and use it for idempotent acceptance
   after a final eligibility check.
6. Extend notification storage and Flutter decoding to a checked social/system
   union, then add five-minute Instagram-match coalescing and retraction.
7. Add lifecycle, retention, export, and operator CLI primitives.
8. Build the Flutter parser/API/repository/controllers/page using
   `accountDioProvider(account)` and `ActiveAccountLease` fencing after every
   await, then add the actorless notification open flow.

Every production path remains safe when Meta is unavailable. Local integration
data, imports, link controls, and suggestions continue to work if their own
dependencies are healthy; only challenge redemption, profile lookup, and reply
work is disabled. A release build must fail closed when configuration is
partial. Production suggestion creation and acceptance also fail closed until
the repository has a current block/mute safety source; automated tests inject a
complete fake policy source.

Use direct pgx stores for this feature. Although repository guidance prefers
sqlc, this checkout has no active sqlc configuration and all neighboring
private subsystems currently use direct pgx. Bootstrapping repository-wide
generation is unrelated risk; the exception is recorded here and all SQL is
covered by real-Postgres migration/store tests.

## 3. Affected Areas

| Area | Existing seam | Planned change | Primary tests |
|---|---|---|---|
| Configuration | One validated `app.Config`; push already supports disabled/enabled modes | Add separate Instagram data and Meta config. Data availability requires the private HMAC material; Meta verification additionally requires explicit enablement plus a complete credential/account/token/signature bundle. Partial production config fails startup. | UT-008, UT-016, IT-013, REG-007 |
| Private persistence | Migrations end at `000022`; direct pgx stores | Add `000023` core Instagram tables and `000024` system-notification union. Avoid owner cascades from `craftsky_profiles`; lifecycle is explicit. | IT-001, IT-010, IT-020, IT-021 |
| Membership boundary | Authentication proves a session DID, but no route-level current-member policy | Add `CurrentMemberRequired` route policy/middleware and a shared membership service used by every Instagram API and worker transition. Missing members receive `404 profile_not_found`. | IT-020, REG-006 |
| Verification attempts | None | Add challenge generation/digesting, public states, expiry/supersession/cancel transitions, owner-scoped current-attempt lookup, same-DID confirmation, and privacy-preserving deletion. | UT-001–UT-004, IT-002, IT-003, IT-022 |
| Meta webhook | No Instagram integration route | Add `/integrations/instagram/webhook` verification and signed POST ingestion outside `/v1`, exact size/event limits, generic ingress throttling, durable deduplication, and no raw payload persistence. | UT-003, UT-004, IT-003, IT-013 |
| Durable worker | Push dispatcher provides the lease/retry lifecycle pattern | Add bounded Instagram work claiming, lease recovery, retry/backoff/deadline, terminal sensitive-field clearing, and injected Meta lookup/reply client. | UT-007, IT-004, IT-019, REG-007 |
| Links and conflicts | No cross-network identity model | Add claims, current username identity, same-DID confirmation, conflict rows, discovery consent, revoke/reactivate semantics, and operator resolution. | UT-006, IT-005, IT-006, IT-018 |
| Imports | No Instagram archive ingestion | Add verified-link-only normalized accounts-followed handles, additive import sources retained until unlink, per-import reactivation/deletion, and bounded pagination. Do not accept or store retention, relationship direction, or follower counts. | UT-005, IT-007, IT-010 |
| Eligibility and matching | Existing follow visibility checks do not include this feature's complete safety policy | Add `InstagramSuggestionEligibilityPolicy`, complete data-source interface, fail-closed missing data, exact imported-username matching, reconciliation jobs, multi-source support, and revalidation at every required boundary. | UT-006, IT-006–IT-009, IT-020 |
| Follow acceptance | Follow handler calls PDS `CreateRecord` | Extract a shared follow service with a stable stored rkey and `PutRecord`; make accepting/retry/crash recovery idempotent despite firehose delay. | IT-009, REG-003 |
| Notifications | Durable rows require an actor and social source fields | Convert storage and Flutter models to a checked `kind: social | system` union; add Instagram-match grouping, five-minute close/push, capped count, newness, retraction, and actorless navigation. | UT-013, UT-014, IT-011, IT-012, IT-021, TD-011 |
| Membership and deletion lifecycle | Profile-record and Tap terminal deletion currently share broad actor cleanup concepts | Treat membership loss as reversible inactivation; terminal Tap identity/future account deletion as permanent Instagram purge; never delete accepted PDS follows. | IT-020, REG-006 |
| Retention/export/operations | No Instagram-specific jobs or CLI | Add deterministic purge batches, owner export with private Instagram data, safe audits/metrics, and CLI list/resolve/revoke/inspect/retry/purge commands. | IT-010, IT-018, IT-019, UT-015 |
| Flutter data/parser | No Instagram feature | Add redacted models, accounts-followed-only JSON parser, 20 MiB/10,000-entry limits, API/repository layer, and cross-language golden contract. Follower data is discarded locally; ZIP parsing is intentionally unsupported. | UT-009, UT-010, IT-014, TD-011 |
| Flutter state/UI | Settings and account boundary have no migration surface | Add fixed-account controllers and one settings route/page for verification, verified-only direct import, link controls, and suggestions; place the discovery selector directly below the verified candidate, default it to discovery allowed, and update the explanation for the selected value; hide all import surfaces until verification, remove normalized-preview/retention controls, link to Notification Settings, reconcile resumable attempts against AppView with a DID-scoped secure display snapshot, invalidate on account changes, and fence every asynchronous effect. | UT-011, IT-015, IT-016, IT-022, REG-009, REG-012 |
| Flutter notifications | Model assumes actor-bearing social notifications | Decode sealed social/system variants, render/open Instagram-match safely, preserve social behavior, and avoid push/diagnostic private content. | UT-012, IT-017, REG-005 |

## 4. Files And Modules

### 4.1 AppView persistence and domain

| Path | Change | Purpose |
|---|---|---|
| `appview/migrations/000023_instagram_migration.up.sql` / `.down.sql` | Create | Core attempts, links, identity claims/conflicts, webhook work, imports/handles, suggestions/sources, reconciliation jobs, stable follow operations, rate buckets, and audit events with checks/indexes/retention fields. |
| `appview/migrations/000024_system_notifications.up.sql` / `.down.sql` | Create | Add checked notification `kind`, nullable social-only columns, system payload/grouping fields, Instagram suggestion support rows, partial uniqueness, and Instagram preference while preserving existing social rows. |
| `appview/internal/instagram/challenge.go` | Create | Canonical 13-symbol challenge generation, parsing, normalization, HMAC digest, and constant-time comparison. |
| `appview/internal/instagram/types.go` | Create | Closed public state types, validated transitions, API/domain DTOs, pagination bounds, clocks, and safe identifiers. |
| `appview/internal/instagram/username.go` | Create | Normalize exact Instagram usernames/handles without fuzzy matching. |
| `appview/internal/instagram/store.go` | Create | Transactional pgx persistence for attempts, links, conflicts, imports, handles, suggestions, support rows, jobs, and audit events. |
| `appview/internal/instagram/verification.go` | Create | Challenge creation/status/cancel/confirm and same-DID ownership workflow. |
| `appview/internal/instagram/webhook.go` | Create | Minimal durable webhook item and dedup/terminal-clear operations. |
| `appview/internal/instagram/worker.go` | Create | Claim, lease, Meta lookup/reply, attempt transitions, reconciliation scheduling, retry, and cancellation-aware processing. |
| `appview/internal/instagram/links.go` | Create | Identity claims, conflict handling, discovery settings, revoke/reactivate, and username transition logic. |
| `appview/internal/instagram/imports.go` | Create | Additive following-only imports, consent, expiry, membership inactivation/reactivation, deletion, and bounded summaries. |
| `appview/internal/instagram/eligibility.go` | Create | One fail-closed policy applied at match, persist, list, notification, open, and final acceptance. |
| `appview/internal/instagram/matcher.go` | Create | Exact imported-username matches, multi-source support, reconciliation, and invalidation. |
| `appview/internal/instagram/suggestions.go` | Create | List/dismiss/accept state machine and crash-safe transitions. |
| `appview/internal/instagram/rate_limiter.go` | Create | Shared Postgres fixed-window limits for DID/device/IP/IGSID/provider boundaries. |
| `appview/internal/instagram/account_data.go` | Create | Owner export and permanent purge primitives. |
| `appview/internal/instagram/retention.go` | Create | Bounded, idempotent retention pass implementing every documented lifetime. |
| `appview/internal/instagram/*_test.go` | Create | Unit and real-Postgres suites from UT-001–UT-008 and IT-001–IT-010/IT-018–IT-020. |

The concrete schema groups related rows but preserves these invariants:

- No raw webhook body, JSON object, message text, plaintext challenge,
  signature, or Meta profile response is stored.
- Durable webhook dedupe uses the Meta message-ID digest; matching uses a keyed
  canonical-challenge digest. Sender IGSID and challenge digest are cleared in
  terminal states.
- Owner references do not cascade from `craftsky_profiles`, because membership
  loss is reversible. Terminal purge is explicit and ordered.
- A link is unique on the active identity claim, with a retained keyed IGSID
  tombstone for the bounded dispute window but no indefinitely retained raw
  identifier.
- Import handles store normalized values only in the private tables and are
  reduced/deleted according to consent and terminal suggestion state.
- Suggestions preserve a row per importer/target and a separate row
  per supporting import so deletion of one import does not destroy remaining
  support.
- The stable follow operation stores the deterministic rkey before the first
  PDS call, allowing a retry to use `PutRecord` safely.

### 4.2 AppView Meta integration, HTTP, and wiring

| Path | Change | Purpose |
|---|---|---|
| `appview/internal/integrations/instagrammeta/signature.go` | Create | Parse and verify `X-Hub-Signature-256` over exact bytes using HMAC-SHA256 and constant-time equality. |
| `appview/internal/integrations/instagrammeta/payload.go` | Create | Strict, bounded decoder for `object=instagram`, entry account ID, IGSID sender, message ID, timestamp, and text only. |
| `appview/internal/integrations/instagrammeta/client.go` | Create | Narrow `LookupUsername` and `SendReply` interface plus typed transient/permanent errors. |
| `appview/internal/integrations/instagrammeta/http_client.go` | Create | Versioned `graph.instagram.com` adapter with 5-second timeout, 64 KiB response cap, redacted errors, and process concurrency cap. |
| `appview/internal/integrations/instagrammeta/handler.go` | Create | GET verification and POST signed ingestion, pre/post-auth rate enforcement, generic 429/Retry-After, and 200 durable acknowledgement. |
| `appview/internal/integrations/instagrammeta/*_test.go` | Create | Signature, payload, route, size/count, duplicate, rate, and httptest provider behavior. |
| `appview/internal/middleware/current_member.go` | Create | Require current `craftsky_profiles` membership using authenticated typed DID and standard `profile_not_found`. |
| `appview/internal/middleware/client_ip.go` | Create/Change | Resolve a trusted forwarded IP only from configured proxy boundaries; otherwise use the peer address. |
| `appview/internal/api/instagram_verifications.go` | Create | Attempt create/get/delete/confirm handlers and exact envelopes. |
| `appview/internal/api/instagram_account.go` | Create | Availability/account GET, settings PATCH, and privacy-preserving DELETE. |
| `appview/internal/api/instagram_imports.go` | Create | Create/list/get/settings/delete with strict bodies, cursors, and 1 MiB/10,000 limits. |
| `appview/internal/api/instagram_suggestions.go` | Create | List/accept/dismiss with final eligibility and stable follow service. |
| `appview/internal/api/instagram_*_test.go` | Create | Route/store/error/wire acceptance suites, including permanent DELETE 204 semantics. |
| `appview/internal/follows/service.go` | Create | Shared deterministic follow ensure operation around `PutRecord` and indexed-state checks. |
| `appview/internal/api/follow.go` | Change | Delegate ordinary follows to the same service without changing existing API behavior. |
| `appview/internal/routes/policy.go` | Change | Add `CurrentMemberRequired` and all authenticated Instagram route policies. |
| `appview/internal/routes/routes.go` | Change | Register authenticated API routes plus the separately composed integration handler. |
| `appview/internal/routes/instagram_routes_test.go` | Create | Prove policy completeness, auth/device/current-member/body/rate/error behavior. |
| `appview/internal/app/config.go` | Change | Parse/validate the separate private-data and Meta configuration bundles and all fixed bounds. |
| `appview/internal/app/deps.go` | Change | Wire stores/services/workers, disabled/fake/HTTP Meta client, safety adapter, notification hooks, and lifecycle composite. |
| `appview/cmd/appview/main.go` | Change | Start bounded Instagram workers/retention alongside existing services and participate in graceful shutdown. |
| `appview/environments/dev.env.example` | Change | Document disabled defaults and synthetic local key generation; never include Meta credentials. |
| `appview/cmd/cli/instagram.go` | Create | Operator conflict/link/job/retention commands with bounded batches and redacted output. |
| `appview/cmd/cli/main.go`, `deps.go` | Change | Register and wire Instagram operator commands. |
| `appview/internal/observability/*` | Change | Add allowlisted low-cardinality outcomes/queue metrics without challenges, usernames, IGSIDs, private graph, tokens, bodies, or payloads. |

### 4.3 Membership, notification, and lifecycle integration

| Path | Change | Purpose |
|---|---|---|
| `appview/internal/instagram/membership.go` | Create | Shared current-member lookup and reversible inactivate/reactivate operations. |
| `appview/internal/index/craftsky_profile.go` | Change | On profile-record deletion, mark Instagram links/imports inactive and invalidate pending discoveries; do not purge private ownership rows. |
| `appview/internal/tap/consumer.go` and dependency composite | Change | Route only terminal identity deletion to permanent Instagram purge while preserving current actor-deletion behavior. |
| `appview/internal/notifications/category.go`, `service.go`, `lifecycle.go` | Change | Add `instagramMatch`, social/system candidate variants, coalescing/retraction, and actorless eligibility. |
| `appview/internal/api/notification_store.go`, `notifications.go` | Change | Read checked union rows and emit exact social or system JSON without nullable social placeholders. |
| `appview/internal/push/payload.go`, dispatcher store/service | Change | Schedule one minimal actorless push when a coalescing window closes; cancel unsent work when the group retracts to zero. |
| `appview/internal/db/notifications_migration_test.go` | Change | Assert safe migration of existing social rows and checked union constraints/indexes. |
| `appview/internal/notifications/*_test.go`, `appview/internal/api/notifications_test.go`, `appview/internal/push/*_test.go` | Change/Create | Cover five-minute grouping, cap 99, newness, retraction, one-push behavior, and unchanged social contracts. |

### 4.4 Flutter feature and shared notification model

| Path | Change | Purpose |
|---|---|---|
| `app/lib/instagram_migration/models/*.dart` | Create | Redacted immutable attempt/link/import/suggestion/page/state models and closed wire states. |
| `app/lib/instagram_migration/data/instagram_migration_api_client.dart` | Create | Exact authenticated `/v1/migrations/instagram/*` requests through an injected fixed-account Dio. |
| `app/lib/instagram_migration/data/instagram_migration_repository.dart` | Create | Narrow repository boundary for verification, account, imports, suggestions, and acceptance. |
| `app/lib/instagram_migration/services/instagram_archive_parser.dart` | Create | Parse supported accounts-followed JSON shapes; extract only following `string_list_data[].value`, discard follower data, normalize/dedupe, and enforce local bounds. |
| `app/lib/instagram_migration/services/instagram_file_picker.dart` | Create | `file_selector` adapter for JSON only, with cancellation and 20 MiB validation. |
| `app/lib/instagram_migration/providers/*_provider.dart` | Create | Fixed-account repository and separate verification/import/suggestion controllers keyed by `ActiveAccountLease`. |
| `app/lib/instagram_migration/pages/instagram_migration_page.dart` | Create | One accessible settings surface with independent verification/import/suggestion failure and retry states. |
| `app/lib/instagram_migration/widgets/*.dart` | Create | Verification instructions/status, account/discovery controls, import picker/summary, and suggestion list/action rows. |
| `app/lib/settings/pages/settings_page.dart` | Change | Add a localized Instagram migration tile. |
| `app/lib/router/route_locations.dart`, `router.dart` | Change | Add `/profile/settings/instagram` on the root navigator beneath settings. |
| `app/lib/auth/providers/account_boundary_provider.dart` | Change | Invalidate all Instagram account-scoped providers when account ownership changes. |
| `app/lib/notifications/models/craftsky_notification.dart` | Change | Make the base hold common fields only and introduce sealed social/system variants; system has no actor/URI/CID/rkey. |
| `app/lib/notifications/models/notification_category.dart` and repository/providers/widgets/navigation | Change | Add `instagramMatch`, render actorless rows, and open the fixed-account migration route while preserving existing social handling. |
| `app/lib/l10n/app_en.arb` and generated localization files | Change/Generate | Add all visible labels, statuses, errors, privacy/consent copy, and accessibility semantics. Generated files are regenerated, never hand-edited. |
| `app/pubspec.yaml`, platform generated plugin registrants if produced | Change/Generate | Add maintained `file_selector` dependency if not already transitive. |
| `app/test/instagram_migration/**` | Create | Parser, wire, repository, provider, routing, fixed-account, redaction, and widget tests from UT-009–UT-012 and IT-014–IT-017. |
| `app/test/notifications/**` and `app/test/settings/settings_page_test.dart` | Change | Prove social regression safety, actorless system decode/render/open, and settings entry. |

All controllers capture `ActiveAccountLease` before work and call
`isActiveAccountOperationCurrent` after every await before mutating state,
cache, snackbar, or navigation. Provider/debug `toString` implementations
redact challenges, usernames, file contents, graph handles, tokens, and IDs.
Parser exceptions exposed to UI are categorized; raw `FormatException` input
snippets are never logged or retained in provider state.

### 4.5 Shared test corpus

Create `testdata/instagram_migration/wire/` at the repository root with only
wholly synthetic JSON:

- request and response fixtures for each route;
- every public state and transition response;
- standard error envelopes and unavailable/conflict variants;
- pagination with omitted and present cursors;
- owned/foreign/absent DELETE expectations;
- social and actorless system notification fixtures;
- supported synthetic export shapes and malformed/oversized examples.

Go and Flutter tests read the same fixtures. A dedicated privacy test scans the
corpus and captured diagnostics/push/PDS requests to prove that synthetic input
appears only in the intentionally private fields under test.

## 5. Services, Interfaces, And Data Flow

### 5.1 Core interfaces

Concrete names may be refined during refactoring, but dependencies remain
narrow and injectable:

```text
type ChallengeCodec interface {
  New() (display string, digest []byte, err error)
  Canonicalize(input string) (string, error)
  Digest(canonical string) []byte
}

type Membership interface {
  RequireCurrent(ctx, txOrPool, did) error
  IsCurrent(ctx, txOrPool, did) (bool, error)
  Inactivate(ctx, tx, did, at) error
  PermanentlyPurge(ctx, tx, did, at) error
}

type MetaClient interface {
  LookupUsername(ctx, igsid) (username string, err error)
  SendReply(ctx, igsid, text) error
}

type SafetyData interface {
  Snapshot(ctx, importerDID, targetDID) (SafetySnapshot, error)
}

type InstagramSuggestionEligibilityPolicy interface {
  Evaluate(ctx, txOrPool, importerDID, targetDID, suggestionID) (Decision, error)
}

type FollowService interface {
  EnsureFollowing(ctx, ownerDID, targetDID, operationID) (Result, error)
}
```

The production `SafetyData` adapter returns unavailable until current block and
mute storage is implemented. `Evaluate` treats unavailable data as ineligible,
records only a bounded reason metric, and never reveals safety state to either
user. Tests use a complete fake to prove positive paths.

### 5.2 Configuration modes

```text
Instagram data unavailable (default)
  INSTAGRAM_DATA_HMAC_KEY absent
  -> account endpoint reports integrationAvailable false
  -> private routes return instagram_unavailable where specified
  -> webhook route is 404

Local/private data available, Meta disabled
  INSTAGRAM_DATA_HMAC_KEY present
  INSTAGRAM_META_ENABLED false
  -> imports, account/link controls, suggestions, retention/export available
  -> new verification and Meta worker actions unavailable
  -> webhook route is 404

Full Meta integration
  INSTAGRAM_META_ENABLED true
  + app secret, verify token, access token, Instagram account ID,
    Graph API version, and reply capability configuration all valid
  -> webhook and worker enabled
```

The HMAC key is secret material and must be stable across processes/deploys.
Development tests inject deterministic bytes; examples explain generation but
contain no real key. Production rejects partial bundles and unsafe placeholder
values. The app never receives Meta tokens or IGSIDs.

### 5.3 Verification flow

```text
authenticated current member
  -> POST verification
  -> transaction: enforce DID/device/IP limit, expire/supersede old pending
     attempt, generate display challenge, persist digest + expiry
  -> return challenge and Instagram DM deep link

Instagram webhook
  -> trusted-peer IP pre-limit
  -> read at most 256 KiB exact bytes
  -> verify signature
  -> post-signature global limit
  -> decode at most 100 supported message events
  -> one transaction: digest message ID/challenge, insert queued rows or
     recognize duplicates; never persist the source payload/text/signature
  -> commit, signal worker, return 200

worker
  -> claim up to configured batch with SKIP LOCKED and 60-second lease
  -> require owning DID is still a current member
  -> re-check attempt expiry/state and per-IGSID invalid-redemption limit
  -> compare keyed canonical challenge digest
  -> MetaClient.LookupUsername with concurrency/rate/timeout caps
  -> transaction: create/update identity claim or conflict, set
     pendingConfirmation, clear work-item sensitive fields
  -> best-effort bounded reply through MetaClient; reply failure never undoes
     durable proof processing

same authenticated DID
  -> POST confirm
  -> transaction: enforce confirmation limit, require pendingConfirmation,
     create/activate link, schedule reconciliation, set confirmed
```

Malformed, unsupported, and unrelated messages become terminal deduplicated
`ignored` work only when a replay shield is required; sensitive fields are
cleared immediately. Duplicate delivery always acknowledges 200 and creates no
second lookup or transition. Worker pressure never changes webhook success once
the durable transaction commits.

### 5.4 Import and matching flow

The Flutter parser sends normalized accounts-followed usernames with no
relationship direction or retention field. The server requires an active
verified Instagram link, strictly rejects retention, direction, and
follower-specific fields, repeats username validation, applies request/entry
limits, deduplicates, and creates an immutable import source. Imports are
additive and retained until per-import deletion or link revocation; each source
keeps its own reversible membership state.

In the initial transaction:

1. Match following handles exactly against current, active, conflict-free,
   discoverable DM-verified links.
2. Run the complete eligibility policy.
3. Upsert one pending suggestion and one source-support row per eligible match.
4. Retain matched and unmatched handles for exact future matching until the
   import is deleted or the verified Instagram link is revoked.
5. Do not create an Instagram-match notification for this initial import.

Targeted reconciliation is queued when a link confirms/enables/reactivates,
its username changes, membership/import is explicitly reactivated, or an
eligibility condition is restored. Every transition again checks membership and
eligibility. List/open/delivery filter ineligible rows and enqueue or perform
invalidation; they do not expose why a target was suppressed.

### 5.5 Suggestion acceptance and PDS flow

Acceptance locks the owned suggestion, changes `pending` to `accepting`, and
persists a deterministic follow operation/rkey before leaving the transaction.
The follow service then:

1. Re-runs the full eligibility policy immediately before the PDS call.
2. Checks the indexed follow relation; if already present, records
   `alreadyFollowing` without another write.
3. Uses the stored rkey and `PutRecord` for `app.bsky.graph.follow` so a timeout
   or retry cannot create two records.
4. Finalizes `accepted` only after a successful PDS response, while treating a
   matching replay as success.
5. Leaves recoverable `accepting` operations for bounded reconciliation after
   a transient failure/crash.

No automatic follow occurs. Dismiss/delete is always owner-scoped and returns
204 for owned, foreign, absent, or already-purged IDs.

### 5.6 System notification flow

`notification_events` becomes a checked union:

```text
kind=social
  actor_did, category/type, source uri/cid/rkey and social references required
  system fields null

kind=system, type=instagramMatch
  actor/source/reference fields null
  system_count, system_count_capped, destination,
  system_group_key, coalesce_until required
```

When targeted reconciliation creates newly eligible suggestions outside the
initial import, attach their IDs to the current five-minute account group. The
window is fixed from first creation and never extended. Additions update count,
cap display at 99, move `indexedAt`/newness, and do not add another push. At
window close the dispatcher rechecks membership and all attached suggestion
eligibility, retracts invalid support, and sends at most one minimal push if the
count remains nonzero. Zero cancels pending/retry/safely reclaimable leased
delivery and retains only the bounded retracted record.

Exact AppView JSON:

```json
{
  "id": "synthetic-notification-id",
  "kind": "system",
  "type": "instagramMatch",
  "createdAt": "2026-07-19T12:00:00Z",
  "indexedAt": "2026-07-19T12:01:00Z",
  "system": {
    "count": 3,
    "countCapped": false,
    "destination": "instagramMigration"
  }
}
```

The Flutter sealed system subtype navigates to the fixed-account Instagram
migration page. If the operation lease is stale after an account switch, the
open is discarded without state or navigation effects.

### 5.7 Membership, verified-link lifetime, export, and purge

- Profile membership loss sets links/imports `membershipInactive`, disables
  discovery, invalidates suggestions and unsent system notifications, and
  pauses jobs. It does not broadly delete private rows.
- Rejoin does not silently restore discovery or imports. The member explicitly
  reactivates the link and each paused import; import lifetime remains tied to
  the verified link.
- Terminal Tap identity deletion invokes one idempotent permanent purge for
  attempts, links/claims/conflicts, webhook work, imports/handles, suggestions,
  jobs, rate/audit data as applicable, and Instagram system notifications.
  Accepted PDS follow records remain untouched.
- Retention uses fixed batches of at most 500, injected time, stable ordering,
  and repeatable transactions. Tests cover every lifetime and boundary instant
  in `01-requirements.md` section 15.
- Owner export contains current private Instagram account/link/import and
  suggestion history permitted by retention. It never exports Meta tokens,
  signatures, HMAC key/digests, rate buckets, other users' private graph, or
  already-discarded unmatched entries.

## 6. State, Providers, And Dependency Injection

### 6.1 AppView graph

`app.Deps` owns process-lifetime shared stores and services:

```text
Config
  -> Instagram key/digest codec
  -> pgx Instagram store + membership service + shared rate limiter
  -> SafetyData -> EligibilityPolicy
  -> MetaClient (disabled, fake, or HTTP)
  -> VerificationService + LinkService + ImportService + SuggestionService
  -> WebhookHandler + WorkerPool + ReconciliationWorker + RetentionRunner
  -> API handlers, notification lifecycle, CLI dependencies
```

HTTP and workers use narrow interfaces so unit tests do not require Meta. Each
worker receives an injected clock and wake channel; database state, not the
channel, is authoritative. Graceful shutdown stops claims, cancels provider
calls, returns or expires leases, and waits within the process's existing
bounded shutdown budget.

### 6.2 Flutter graph

```text
ActiveAccountLease
  -> accountDioProvider(lease.account)
  -> instagramMigrationApiClientProvider(lease)
  -> instagramMigrationRepositoryProvider(lease)
  -> instagramVerificationControllerProvider(lease)
  -> instagramImportsControllerProvider(lease)
  -> instagramSuggestionsControllerProvider(lease)
  -> InstagramMigrationPage(lease)
```

Verification availability is separate from local migration availability so a
Meta outage does not blank imported data. Independent controller state lets one
panel retry without destroying the others. Controllers own only display-safe
summaries; selected file bytes and raw parser input are scoped to the immediate
operation and released afterward.

`accountStateInvalidatorProvider` invalidates every Instagram provider for the
departed account. Activation generation in `ActiveAccountLease` prevents a
switch-away/switch-back race even when the DID/account key is the same.

## 7. UI, Routes, And Surfaces

Add a localized “Find people from Instagram” item to profile settings. The
typed route is `/profile/settings/instagram` and uses the root navigator so it
is not tied to a tab stack. The page receives/captures the active account lease;
it never falls back to a mutable global active token during an operation.

The page contains:

1. Availability/account card: explains local import versus DM verification,
   shows verified username only when allowed, controls discovery, and supports
   explicit reactivation/revoke.
2. Verification card: creates a challenge, provides copy/open-DM actions,
   displays expiry/processing/pending confirmation/conflict/error states, and
   requires explicit confirmation from the same authenticated DID.
3. Import card: rendered only after verification; explains that it imports
   accounts the member follows and retains them until unlink, directly accepts
   manual handles or a matching Accounts Center JSON file without a normalized
   preview step, links to Notification Settings, lists sources, and supports
   source-specific reactivation/deletion.
4. Suggestions card: lists eligible exact matches, supports explicit accept and
   dismiss, prevents double taps while accepting, and refreshes/invalidate rows
   whose eligibility changed.

Meta verification unavailability disables only verification controls and shows
actionable copy; imports/link privacy controls/suggestions remain independently
usable. ZIP files get a clear unsupported-format error. Conflict, expiry,
membership reactivation, provider timeout, invalid JSON/shape, limit, and stale
account operation states each have localized, non-sensitive UI treatment.

Accessibility tests cover semantic labels, focus/tap targets, text scaling, and
screen-reader descriptions for challenge copy, consent, discovery, follow, and
dismiss controls. Final platform/device inspection remains a manual gate.

## 8. Error And Privacy Handling

- AppView uses the standard camelCase `{error,message,requestId}` envelope.
- Authenticated non-members receive `404 profile_not_found`; ownership-sensitive
  IDs never become an existence oracle.
- DELETE attempt/account/import/suggestion always returns 204 and mutates only
  rows owned by the authenticated current member.
- `instagram_unavailable` distinguishes disabled Meta-dependent work from
  ordinary retryable provider failure. Existing local data is not hidden.
- Webhook authentication and limit failures are generic; rate excess returns
  `429` with `Retry-After: 60` and no partial persistence. Valid durable work is
  acknowledged 200 even when workers are busy.
- Provider errors are classified into transient/permanent/auth/rate/invalid
  without retaining response bodies. The retry schedule is 1 second through 5
  minutes, at most five attempts, and no work processes beyond 15 minutes.
- `context.Canceled` follows the existing internal 499/no-Sentry boundary.
- Logs, metrics, traces, Sentry, CLI output, push payloads, and API errors never
  include challenge text/digest, webhook bytes/signature/message, IGSID,
  Instagram username, private graph handles, import contents, Meta tokens, or
  HMAC material.
- Flutter error mapping discards raw decoder/HTTP body excerpts and exposes
  typed display-safe failures. All new models and states have redacted string
  representations because provider logging stringifies values.
- Only wholly synthetic/redacted test inputs may enter committed fixtures.

## 9. Ordered Test-First Plan

Follow the exact order approved in `02-acceptance-tests.md`. Add one failing
test, observe the expected failure, implement the minimum code, rerun green,
then refactor before advancing.

1. **Challenge contract** — `UT-001` in
   `appview/internal/instagram/challenge_test.go`; alphabet, length, entropy
   shape, canonicalization, exact whole-message grammar, injected randomness,
   digest comparison, and redaction.
2. **Configuration and shared limits** — `UT-008`, `UT-016`, `IT-013`;
   disabled/local/full modes, partial production failure, distributed limits,
   trusted IP, and dependency wiring with no provider call.
3. **Migration and state machines** — `IT-001`, `UT-002`; apply/down schema,
   checks/indexes/no cascades, attempt/link/import/suggestion/conflict/work
   transitions, idempotence, and retention timestamps.
4. **Current membership boundary** — `IT-020`; every authenticated route and
   worker transition, reversible inactivation/rejoin, and permanent deletion.
5. **Golden wire contract** — `IT-021`; Go reads all synthetic fixtures before
   handlers and Flutter models depend on them.
6. **Verification routes and webhook** — `IT-002`, `UT-003`, `UT-004`,
   `IT-003`; exact API, DELETE 204, signature/exact bytes, bounds, dedup, invalid
   redemption, privacy, and generic rate handling.
7. **Worker and Meta adapter** — `UT-007`, `IT-004`; claim/lease/retry/deadline,
   cancellation, lookup/reply fakes, HTTP caps, duplicate behavior, and terminal
   clearing.
8. **Links, conflicts, and eligibility** — `UT-006`, `IT-005`, `IT-006`;
   same-DID confirmation, claims/conflicts/settings, exact complete fail-closed
   policy at every boundary.
9. **Imports and matching** — `UT-005`, `IT-007`, `IT-008`; additive sources,
   following-only usernames, limits, consent/retention, multi-source exact matches,
   reactivation/deletion, and reconciliation triggers.
10. **Follow acceptance** — `IT-009`; stable rkey, `PutRecord`, indexed follow,
    timeout/crash/retry races, final eligibility, and ordinary-follow regression.
11. **Retention and export** — `IT-010`; all documented clocks, immediate
    sensitive clearing, batch 500, repeatability, and export exclusions.
12. **System notifications** — `UT-013`, `UT-014`, `IT-011`, `IT-012`;
    migration union, fixed coalescing, cap, newness, retraction, one push, and
    unchanged social notifications.
13. **Operations** — `IT-018`, `IT-019`; conflict/link/job/retention CLI,
    replay/retry, redaction, bounded output, and worker cancellation.
14. **Flutter parser/API** — `UT-009`, `UT-010`, `IT-014`; supported
    accounts-followed JSON shapes, local follower-data discard,
    malformed/private excerpt redaction, bounds,
    every route/error/state/golden fixture, and fixed-account Dio.
15. **Flutter providers/UI** — `UT-011`, `IT-015`, `IT-016`; lease fencing after
    every await, invalidation, independent state, settings route, all panels,
    accessibility, and failure/retry states.
16. **Flutter notification union** — `UT-012`, `IT-017`; actorless decode,
    row/push/open, stale-account no-op, and social regression.
17. **Privacy/regression sweep** — `UT-015`, `REG-001`–`REG-012`, `TD-001`–
    `TD-012`; scan fixtures/diagnostics, run full Go/Flutter gates, race tests,
    migration round-trip, generated code/analyze, and architecture checks.
18. **Resumable verification** — `IT-002`, `IT-022`, `REG-012`; add the
    owner-scoped current-attempt read, DID-scoped secure display snapshot,
    AppView reconciliation, polling/confirmation restoration, and narrow
    terminal/session cleanup without keeping page providers alive.

Focused commands evolve with the slice, then finish with:

```text
cd appview && go test ./...
cd appview && go test -race ./internal/instagram ./internal/integrations/instagrammeta ./internal/notifications ./internal/push
cd app && dart run build_runner build --delete-conflicting-outputs
cd app && dart format --output=none --set-exit-if-changed lib test
cd app && flutter analyze
cd app && flutter test
```

The pre-implementation Flutter baseline has one existing analyzer info at
`lib/auth/providers/active_account_identity_provider.dart:27` for line length.
New work must introduce no additional diagnostics; final reporting must not
misrepresent that baseline as clean unless separately corrected.

## 10. Sequencing And Guardrails

- Do not edit `lexicon/`; this feature has no new public record type.
- Do not call Meta in automated tests. Use fakes and `httptest.Server` only.
- Do not use real or user-derived Instagram exports, DMs, usernames, IDs, or
  tokens in source, tests, screenshots, logs, or demos.
- Land `000023` before code that queries core tables and `000024` before system
  notification code. Down migrations delete dependent system/support rows first.
- Keep migration numbering append-only; never rewrite applied `000021`/`000022`.
- Establish the checked Go wire and shared fixtures before Flutter model work.
- Convert notification consumers exhaustively; do not add nullable fake actor
  fields to make the old model compile.
- Reuse existing auth/device/body/error/cancellation/observer middleware. The
  integration webhook is separately composed because it is signed by Meta, not
  a `/v1` Craftsky session route.
- Require the shared current-member service in APIs and workers; avoid scattered
  ad hoc profile checks.
- Require the shared eligibility policy at all enumerated boundaries; avoid
  route-only filtering.
- Preserve independent imports and support rows so deletion/reactivation is
  source-specific.
- Preserve explicit user intent: no automatic discovery re-enable, import
  reactivation, notification for initial import, or PDS follow.
- Keep DB transactions short around provider calls: claim/finalize in separate
  transactions and revalidate before committing state.
- Do not manually edit Riverpod/go_router/localization generated Dart files;
  regenerate after source/ARB changes.
- Preserve unrelated worktree changes and do not commit without a separate user
  request.

## 11. Risks, Release Gates, And Open Questions

There are no implementation-blocking product questions. The following are
deliberately deferred release gates because they require an Instagram app,
approved real environment, or missing repository safety data:

1. Confirm Meta permits an unrelated Instagram user to DM the configured
   professional account and delivers the expected webhook fields.
2. Validate current Instagram Login permissions, token lifetime/refresh,
   account ID, Graph version, webhook subscription, profile lookup, and reply
   behavior using the real configured app.
3. Validate trusted proxy IP handling and distributed rate buckets against the
   deployed edge and Postgres topology.
4. Inspect currently exported Accounts Center JSON shapes with explicitly
   approved/redacted data and add support only if the parser contract differs.
5. Provide a production current block/mute safety adapter. Until then,
   suggestion creation/listing/acceptance is intentionally fail closed.
6. Complete real-device push coalescing/open, accessibility, lifecycle, and
   platform file-picker checks.
7. Run a security/privacy review of Meta secrets, key rotation, retention,
   deletion/export, and operator access before enabling production.

Schema/API choices are implementable now and remain useful with fakes even if a
live Meta test changes the provider adapter later. Any discovered Meta payload
or permission difference must be confined to `instagrammeta` unless it changes
the approved product contract, in which case requirements return to review.

## 12. Handoff

Workflow status: approved for implementation.

Start with `UT-001` and record red/green evidence in
`05-implementation-plan.md`. Continue through the ordered slices without a new
approval pause because the user already authorized implementation. After the
automated pass, perform the repository's implementation review workflow,
resolve actionable findings, rerun affected/full gates, and report:

- AppView and Flutter behavior completed;
- tests and baseline diagnostics;
- privacy/security constraints preserved;
- production paths intentionally fail closed;
- live Meta, export-shape, safety-adapter, and device gates still outstanding.
