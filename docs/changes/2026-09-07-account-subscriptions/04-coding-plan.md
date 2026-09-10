# Coding Plan: AppView Account Subscriptions

## 1. Inputs

- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` (`Approved`, high risk)
- Direction: `00-direction.md`
- Architecture decision: `adr/014-account-assigned-subscription-licenses.md`
- API conventions:
  `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md`
- Provider contract: RevenueCat API v2 subscription resources and current
  webhook authentication/retry documentation.

## 2. Implementation Strategy

Build a private AppView capability with a deliberately narrow provider boundary:

1. `internal/subscriptions` owns billing accounts, provider subscription rows,
   one-to-one licenses, assignments, anomaly containment, effective access, and
   reconciliation scheduling.
2. `internal/integrations/revenuecat` verifies webhook requests and fetches
   complete API v2 customer subscription snapshots.
3. Webhooks, owner refresh, and the periodic schedule advance a billing account's
   requested reconciliation generation. One processor claims the current
   generation, fetches RevenueCat outside a database transaction, and applies the
   complete snapshot without clearing a newer request.

RevenueCat's `gives_access` is copied as provider truth. AppView does not reduce
webhook lifecycle events or infer cancellation, renewal, grace, pause, billing
retry, refund, or product-change access. Server authorization remains an indexed
local read over license assignment and the latest accepted snapshot.

Use direct pgx SQL and existing route, lifecycle, worker, and observability
patterns. Use two migrations rather than the prior three. Do not add purchase
intents, ownership transfers, assignment history, raw payload tables, quarantine,
retention processors, canonical failover, or runtime merchandising validation.

## 3. Affected Areas

| Area | Existing Pattern | Planned Change | Requirement IDs | Test IDs |
|---|---|---|---|---|
| Subscription domain | Capability packages own focused policy | Add tiers, access projection, snapshot mapping/application, anomaly classification, and cooldown | BR-001 through BR-005; FR-003 through FR-006; FR-012, FR-013, FR-026 through FR-030; RULE-002 through RULE-007 | UT-001 through UT-003, UT-005 through UT-007; AT-002, AT-003, AT-006, AT-008, AT-011 |
| Persistence | Paired migrations and pgx transactions | Add four primary tables plus constraints, indexes, closure, and reconciliation lease fields | FR-001, FR-003, FR-004, FR-009, FR-012, FR-015, FR-017, FR-018, FR-031; NFR-004, NFR-006, NFR-007 | IT-001 through IT-009 |
| Authenticated API | Narrow handlers and route bundles | Add self access, billing ensure/read, assignment, and refresh routes | FR-001, FR-002, FR-007 through FR-011, FR-018, FR-028, FR-030, FR-032; NFR-002 | AT-001, AT-003 through AT-006; IT-002 through IT-004, IT-010; REG-005 |
| RevenueCat ingress | Public integrations authenticate exact bytes | Add bounded dual-authenticated webhook that stores only sanitized event metadata and requests reconciliation | FR-014 through FR-016; NFR-003 through NFR-005 | UT-004; IT-006; AT-007, AT-012 |
| Reconciliation | Existing durable workers use leases and token fencing | Add one billing-account reconciliation processor shared by every trigger | FR-005, FR-017 through FR-019, FR-026, FR-032 | UT-002, UT-003, UT-005; IT-005, IT-007; AT-006, AT-011 |
| Account deletion | Lifecycle transitions accept participants | Warn and block until every post-intent persisted subscription is non-pending and exactly `will_not_renew`; permit closure whether canceled access is active or expired | FR-022, FR-031 | AT-009; IT-008; REG-002 |
| Configuration | Validated and redacted app config | Add only provider credentials, webhook security, project/environment, product mappings carrying app/tier, and worker budgets | FR-014, FR-018, FR-019; NFR-003, NFR-005 | UT-003, UT-004; AT-006, AT-007 |
| Release verification | `testdb.WithSchema`, route inventory, `appview-check` | Add proportionate subscription and migration coverage | NFR-007, NFR-008 | AT-013; IT-009; REG-001 through REG-006 |

## 4. Files And Modules

Migration numbers must be rechecked before implementation; `000069` and
`000070` are placeholders based on the current plan context.

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `appview/migrations/000069_subscription_accounts.{up,down}.sql` | Create | Billing accounts, provider subscriptions, licenses/assignments, closure state, constraints, and access indexes | FR-001, FR-003, FR-004, FR-009, FR-012, FR-022, FR-026, FR-030, FR-031 | IT-001 through IT-005, IT-008, IT-009 |
| `appview/migrations/000070_revenuecat_events.{up,down}.sql` | Create | Sanitized event deduplication and reconciliation request/lease indexes; no raw columns | FR-015, FR-017, FR-018; NFR-003 through NFR-005 | IT-006, IT-007, IT-009 |
| `appview/internal/subscriptions/types.go` | Create | Tier, provider snapshot, billing state, access projection, anomaly, and domain errors | BR-001 through BR-005; FR-003 through FR-013 | UT-001, UT-007; AT-002 through AT-005 |
| `appview/internal/subscriptions/access.go` | Create | Pure effective-access projection from local assignment and provider access | FR-005 through FR-007, FR-013, FR-027 through FR-029 | UT-001; IT-004; AT-003, AT-008 |
| `appview/internal/subscriptions/catalog.go` | Create | Minimal project/app/environment and product-to-tier allowlist | FR-019; RULE-006, RULE-007 | UT-003; AT-006 |
| `appview/internal/subscriptions/anomaly.go` | Create | Fail-closed duplicate and cross-tier anomaly classification | FR-026 | UT-005; IT-005; AT-011 |
| `appview/internal/subscriptions/store.go` | Create | Billing ensure/read, access read, snapshot application, and reconciliation scheduling/claiming | FR-001 through FR-008, FR-015, FR-017 through FR-019, FR-026 through FR-032 | IT-001, IT-002, IT-004 through IT-008 |
| `appview/internal/subscriptions/assignment.go` | Create | Assignment/unassignment transaction, same-device checks, and simple cooldown | FR-009 through FR-012, FR-030 | UT-006; IT-003; AT-005 |
| `appview/internal/subscriptions/reconciliation.go` | Create | Shared trigger service and one leased snapshot processor | FR-017, FR-018, FR-032; NFR-004, NFR-005 | IT-007; AT-006 |
| `appview/internal/subscriptions/deletion.go` | Create | Exact non-renewal/non-pending deletion gate, transactional live-state cleanup after post-intent reconciliation, and closed marker | FR-022, FR-031 | IT-008; AT-009; REG-002 |
| `appview/internal/integrations/revenuecat/signature.go` | Create | Authorization and timestamped HMAC verification | FR-014 | UT-004; AT-007 |
| `appview/internal/integrations/revenuecat/webhook.go` | Create | Bounded parse, safe identity mapping, sanitized deduplication, and reconcile request | FR-014 through FR-016; NFR-003, NFR-005 | IT-006; AT-007, AT-012 |
| `appview/internal/integrations/revenuecat/client.go` | Create | Hardened API v2 customer-subscription pagination and snapshot mapping | FR-003, FR-005, FR-017 through FR-019 | UT-002; IT-005, IT-007 |
| `appview/internal/api/subscriptions.go` | Create | Strict camelCase subscription handlers | FR-001, FR-002, FR-007 through FR-011, FR-018, FR-028, FR-030, FR-032 | AT-001, AT-003 through AT-006; IT-010 |
| `appview/internal/routes/routes_subscription.go` | Create | Authenticated route bundle and policies | NFR-002 | REG-005 |
| `appview/internal/routes/routes_public_auth.go` | Change | Register webhook only under complete secure configuration | FR-014 | AT-007; REG-005 |
| `appview/internal/app/revenuecat_config.go` | Create | Disabled-by-default redacted provider and worker configuration | FR-014, FR-018, FR-019; NFR-003, NFR-005 | UT-003, UT-004; REG-006 |
| `appview/internal/app/deps_subscriptions.go` | Create | Construct store, client, handlers, processor, and deletion participant | FR-001 through FR-032 excluding retired IDs | Acceptance and integration suites |
| `appview/internal/app/{config.go,deps.go,routes_adapter.go}` | Change | Compose narrow capabilities | NFR-002 through NFR-005 | AT-007, AT-013; REG-005, REG-006 |
| `appview/cmd/appview/main.go` | Change | Start and stop the single reconciliation processor | FR-018; NFR-005 | IT-007; AT-013 |
| `appview/internal/accountdeletion/{service.go,app_service.go}` | Change | Add warning, post-intent reconciliation prerequisite, owner blocker/closure, and target-assignment cleanup participant | FR-022, FR-031 | IT-008; AT-009; REG-002 |
| `appview/internal/api/account_deletion.go` | Change | Return provider-billing warning on intent and a standard conflict while final deletion is blocked | FR-022 | AT-009; REG-002 |
| `appview/internal/ownerlifecycle/terminal_inventory.go` | Change | Register live billing DID columns; closed marker contains none | FR-022, FR-031; NFR-007 | IT-008, IT-009 |
| `appview/internal/observability/{observer.go,metric_recorder.go,redaction.go}` | Change | Add safe billing outcomes and sensitive header classification | NFR-003 | AT-012; REG-006 |

## 5. Services, Interfaces, And Data Flow

### Core Types

Use `syntax.DID` for DIDs, `uuid.UUID` for internal IDs and RevenueCat App User
IDs, and closed string types for tier, anomaly, work, and account state.

```text
type AccessReader interface {
    SelfAccess(context.Context, syntax.DID, time.Time) (SelfAccess, error)
}

type BillingService interface {
    EnsureAccount(context.Context, syntax.DID) (BillingState, bool, error)
    OwnerState(context.Context, syntax.DID, time.Time) (BillingState, error)
    Assign(context.Context, AssignParams) (Assignment, error)
    Unassign(context.Context, UnassignParams) error
    RequestRefresh(context.Context, syntax.DID) error
}

type ProviderClient interface {
    ListCustomerSubscriptions(context.Context, uuid.UUID) (CompleteSnapshot, error)
}

type SnapshotApplier interface {
    ApplySnapshot(context.Context, SnapshotClaim, CompleteSnapshot) error
}
```

`SelfAccess` contains only `did`, `effectiveTier`, `givesAccess`, optional
informational `accessEndsAt`, and optional `assignedTier`. `BillingState` contains
the owner's billing UUID, subscriptions, licenses, assignments, access and period
timestamps, reconciliation staleness, assignability, and safe anomaly/management
context. Neither contains raw provider data or credentials. Period timestamps do
not participate in effective-tier calculation.

### Persistence Shape

Use four primary tables:

- `billing_accounts`: internal ID, nullable active owner DID, unique RevenueCat
  UUID, active/closed state, closure time, requested/reconciled times, requested,
  reconciled, and claimed generations, attempts, next attempt, and lease
  token/expiry.
- `provider_subscriptions`: billing account, unique `(project_id,
  revenuecat_subscription_id)`, product/store/environment provider attributes,
  mapped app/tier authorization attributes, status, `gives_access`,
  `pending_payment`, `auto_renewal_status`, period times, accepted generation,
  and anomaly state. App identity is taken from the configured product mapping
  because it is not supplied by the subscription resource.
- `billing_licenses`: unique provider-subscription FK, tier, assigned DID,
  assignment time, last target-change time, and assignable/anomaly state. A
  partial unique index enforces one assigned license per DID, including dormant
  licenses.
- `revenuecat_events`: unique event ID, mapped billing account when known,
  bounded event type, accepted time, and outcome. It has no body, checksum,
  customer UUID, receipt, transaction, or payment columns.

Closing an account nulls/removes the owner DID, records `closed_at`, deletes live
provider subscriptions/licenses through cascades, and clears reconciliation
fields. The remaining row is the closed marker. Deleting a non-owner target DID
sets matching license assignments to unassigned in the same lifecycle
transaction without changing billing ownership or provider subscription state.

### Snapshot Fencing

```text
trigger reconcile
  -> increment requested_generation and mark billing account due

processor claim transaction
  -> claim due active account with SKIP LOCKED
  -> copy requested_generation to claimed_generation
  -> store claimed generation and lease token/expiry

outside transaction
  -> fetch every API v2 subscription page
  -> reject incomplete pagination/oversized response

apply transaction
  -> lock active billing account
  -> require matching claimed generation + lease token
  -> map allowlisted products
  -> classify anomalies against existing assignments and complete snapshot
  -> upsert subscription/license rows
  -> preserve existing assignments
  -> leave new licenses unassigned
  -> mark absent previously known rows inaccessible only after a complete fetch
  -> advance reconciled_generation to claimed_generation and clear lease
  -> remain due when requested_generation > reconciled_generation
```

A stale worker cannot apply after lease recovery because its generation/token no
longer matches. A trigger arriving during fetch increments `requested_generation`,
so applying the older claim cannot clear that newer request. No provider HTTP
request occurs while a database transaction is open.

### Webhook Flow

```text
bounded raw request bytes
  -> exact Authorization comparison
  -> parse unique t/v1 signature components
  -> five-minute timestamp tolerance
  -> HMAC-SHA256(timestamp + "." + raw body)
  -> parse only required envelope identity
  -> map RevenueCat App User UUID to active/closed/unknown billing account
  -> insert sanitized event ID/type/outcome
  -> if active, increment requested_generation and mark billing account due
  -> commit
  -> return 200
```

Invalid authentication, malformed bodies, and oversized input are rejected with
generic responses and no persistence. A duplicate event ID is a successful
idempotent no-op. An event for a closed account is stored as ignored and never
requests reconciliation. Unknown event types follow the same mapped-account
reconciliation path as known types.

### Anomaly Policy

- One supported accessible subscription for a tier is ordinary and its license is
  assignable.
- If a tier already has a safely assigned license, preserve it and mark
  additional same-tier licenses unassignable.
- If multiple same-tier subscriptions appear without a safe existing choice,
  mark all affected licenses unassignable.
- If simultaneously accessible subscriptions create a paid cross-tier conflict,
  preserve only demonstrably non-conflicting existing access and otherwise mark
  the new license unassignable. Never move an assignment automatically.
- If RevenueCat introduces a new subscription ID after lapse or paid product
  change, create a new unassigned license. Do not inherit the prior subscription's
  assignment or classify the new ID as anomalous solely because it is new.
- Anomaly state is owner-visible and observable but not exposed to assigned
  accounts.

## 6. State, Providers, Controllers, Or DI

There is no Flutter/Riverpod work in this stage.

```text
RevenueCatConfig + pgx pool + HTTP boundary + clock + observer
  -> revenuecat.Client
  -> subscriptions.Store
  -> subscriptions.ReconciliationProcessor
  -> revenuecat.WebhookHandler
  -> subscriptions.Service + AccessReader + DeletionParticipant
  -> routes.Dependencies
```

`RevenueCatConfig` contains only API origin/key, project ID, production
environment, product mappings carrying app ID and tier, webhook
Authorization/HMAC, body/response/page limits, an ingress deadline no greater
than ten seconds, reconciliation interval, retry budget, and lease duration. It
contains no prices, territory list, offering/package expectations, trial policy,
Family Sharing policy, or current-offering rule.

## 7. UI, Widgets, Routes, Or User-Facing Surfaces

Every `/v1/*` route uses current-member and device middleware, strict camelCase
JSON, standard errors, and existing body/rate policies.

| Method / Path | Request | Success | Notes |
|---|---|---|---|
| `GET /v1/subscriptions/access` | No body | `200 {did,effectiveTier,givesAccess,accessEndsAt?,assignedTier?}` | Local read; creates nothing; no provider call; `accessEndsAt` is informational. |
| `PUT /v1/billing/account` | No body | `201` first creation or `200 BillingState` retry | Explicit idempotent owner selection. |
| `GET /v1/billing/account` | No body | `200 BillingState` | Owner-only. |
| `PUT /v1/billing/licenses/{licenseId}/assignment` | `{targetDid}` | `200 {licenseId,targetDid,assignedAt}` | Same-device transaction and cooldown. |
| `DELETE /v1/billing/licenses/{licenseId}/assignment` | No body | `204` | Idempotent; does not reset last target-change time. |
| `POST /v1/billing/reconciliation` | No body | `202 {status:"pending"}` | Owner-only; advances requested generation; no client-supplied billing UUID. |

The billing response includes subscriptions and licenses with safe IDs, tier,
product display identifier where needed, store, status, `givesAccess`, optional
end, assignment, assignability, and bounded anomaly code. It excludes raw
provider subscription identifiers, receipts, transactions, aliases, payment
details, credentials, webhook data, and merchandising configuration.

`POST /integrations/revenuecat/webhook` remains outside `/v1/*`. It returns `200`
after durable acceptance, `401` for generic authentication failure, `400` for
generic malformed input, `413` for oversized input, and `503` for persistence
failure. It never uses CraftSky session middleware.

Account deletion keeps its existing routes, adds the warning, and returns a
standard conflict when any persisted subscription is pending or its
`auto_renewal_status` is not exactly `will_not_renew`. Starting deletion advances
reconciliation generation; confirmation requires a complete result for that or
a later generation before evaluating every persisted row. `gives_access`,
provider status, period/end timestamps, and webhook types never substitute for
the exact cancellation evidence, and confirmation never calls the provider:

```text
code: provider_billing_not_canceled
message: Deleting your CraftSky account does not cancel provider billing and may end CraftSky access.

code: provider_billing_must_be_resolved
message: Resolve provider billing and refresh subscription status before deleting this account.
```

## 8. Error, Loading, Empty, And Edge States

| State / Case | Planned Handling | Requirement IDs | Test IDs |
|---|---|---|---|
| No billing account | Self access is free; owner GET is non-leaking not-found; explicit PUT creates | FR-001, FR-007 | AT-001, AT-003 |
| Empty owner account | Return stable UUID and empty subscription/license arrays | FR-001, FR-008 | AT-001, AT-004 |
| New purchase | Reconciliation creates an unassigned license | FR-004, FR-032 | AT-002, AT-006 |
| RevenueCat unavailable | Local reads preserve latest accepted `givesAccess`; staleness is observable; worker retries; period timestamps do not override access | FR-006, FR-018, FR-027 | AT-003, AT-008 |
| Out-of-order webhook | Queue current snapshot; do not apply event content | FR-016, FR-017 | AT-006, AT-007 |
| Stale snapshot worker | Generation/token fence rejects apply | FR-017; NFR-004 | IT-005, IT-007 |
| Trigger during provider fetch | Advance requested generation; older successful apply leaves newer generation due | FR-017; NFR-004 | AT-006, IT-007 |
| Incomplete API pagination | Apply nothing and retry | FR-017 | UT-002, IT-007 |
| Unknown/sandbox product | Persist safe provider visibility as needed but grant no production license access | FR-019 | UT-003, IT-005 |
| Dormant assignment | Effective free; assignment and assigned tier remain | FR-012, FR-029 | AT-003, AT-008 |
| Same-tier duplicate | Preserve safe existing assignment; extras or ambiguous set are unassignable | FR-026 | AT-011 |
| Simultaneously accessible cross-tier change | Mark anomaly; never auto-move assignment or tier benefit | FR-026 | AT-011 |
| New subscription ID after lapse/product change | Create a new unassigned license without inheriting the dormant assignment | FR-026, FR-029 | AT-011, IT-005 |
| Target session races mutation | Revalidation fails and prior assignment remains | FR-010, FR-011 | IT-003 |
| Reassignment inside cooldown | Reject without changing assignment | FR-030 | UT-006, AT-005 |
| Invalid webhook | Generic failure; no raw persistence or access | FR-014; NFR-003 | UT-004, AT-007, AT-012 |
| Duplicate event ID | Successful no-op; no additional equivalent work | FR-015 | IT-006 |
| Owner deletion before post-intent reconciliation or with any pending/non-`will_not_renew` row | Warn, request reconciliation, and reject final deletion without removing billing state | FR-022 | AT-009, IT-008 |
| Owner deletion after exact non-pending `will_not_renew` reconciliation | Close local billing authority whether subscriptions remain active/accessible or are expired/inaccessible; never request provider cancellation | FR-022, FR-031 | AT-009, IT-008 |
| Assigned non-owner deletion | Atomically clear target assignments without changing billing ownership/provider state | FR-022 | AT-009, IT-008 |
| Closed account event | Record ignored outcome and do not reconcile | FR-031 | IT-008 |

## 9. Test Implementation Plan

| Order | Test ID | Target | Setup / Fixture | Initial Expected Failure |
|---|---|---|---|---|
| 1 | UT-001 | `internal/subscriptions/access_test.go` | Free/Plus/Business/dormant table | Subscription package absent |
| 2 | IT-001 | `internal/subscriptions/store_integration_test.go` | Core schema constraints | Tables absent |
| 3 | IT-002 | Same store suite | Concurrent account ensure | Ensure operation absent |
| 4 | AT-001 | `internal/routes/subscription_routes_test.go` | Owner/non-owner routes | Routes absent |
| 5 | UT-002 | `internal/integrations/revenuecat/client_test.go` | API v2 fixtures | Snapshot mapper absent |
| 6 | UT-003 | `internal/subscriptions/catalog_test.go` | Minimal allowlist matrix | Catalog absent |
| 7 | UT-005 | `internal/subscriptions/anomaly_test.go` | Duplicate/change fixtures | Anomaly policy absent |
| 8 | IT-005 | `internal/subscriptions/snapshot_integration_test.go` | Complete concurrent snapshots | Snapshot transaction absent |
| 9 | AT-006 | `internal/subscriptions/reconciliation_test.go` | All trigger modes | Reconciliation absent |
| 10 | UT-004 | `internal/integrations/revenuecat/signature_test.go` | HMAC boundaries | Verifier absent |
| 11 | IT-006 | `internal/subscriptions/webhook_store_integration_test.go` | Duplicate/failed commits | Event store absent |
| 12 | AT-007 | `internal/integrations/revenuecat/webhook_handler_test.go` | Real ingress with fake store | Handler absent |
| 13 | IT-007 | `internal/subscriptions/reconciliation_integration_test.go` | Retry and stale lease | Processor absent |
| 14 | IT-004 | `internal/subscriptions/access_integration_test.go` | Local indexed fixture | Access query absent |
| 15 | AT-003 | Route suite | Self-access privacy | Access route absent |
| 16 | IT-003 | `internal/subscriptions/assignment_integration_test.go` | Session and concurrency matrix | Assignment absent |
| 17 | UT-006 | `internal/subscriptions/cooldown_test.go` | Seven-day boundaries | Cooldown absent |
| 18 | AT-005 | `internal/routes/subscription_assignment_test.go` | Assignment route matrix | Routes absent |
| 19 | AT-002 | Subscription acceptance suite | Plus and Business snapshot | Full flow incomplete |
| 20 | AT-008 | Access acceptance suite | Stale reconciliation, passed period timestamp, and same-ID recovery | Access-authority flow incomplete |
| 21 | AT-011 | Anomaly acceptance suite | Duplicate/change snapshots | Fail-closed flow incomplete |
| 22 | IT-008 | Account deletion integration | Active and expired canceled rows plus mixed renewable/resumable/unknown/missing/pending rows | Closure participant absent |
| 23 | AT-009 | Account deletion acceptance | Warning, exact-evidence blocker, active canceled confirmation, expired canceled confirmation, and later event | Deletion contract incomplete |
| 24 | AT-004 | Route suite | Owner aggregate and identity | Owner contract incomplete |
| 25 | IT-010 | Identity route suite | Exact/mismatched identity | Readiness check absent |
| 26 | AT-010, AT-012 | Isolation/privacy suites | Social and sensitive canaries | Guardrails unproven |
| 27 | REG-001 through REG-006 | Existing suites | Regression fixtures | Coverage absent |
| 28 | IT-009, AT-013 | Migration and release gate | Up/down/up environment | Full gate incomplete |

Focused commands:

```text
cd appview && go test ./internal/subscriptions -run '^TestSelfAccessProjection$'
cd appview && go test ./internal/subscriptions ./internal/integrations/revenuecat ./internal/routes ./internal/accountdeletion ./internal/db
```

Finish with `just fmt`, `just test`, and `just appview-check`.

## 10. Sequencing And Guardrails

- First TDD step: Add `UT-001` for local effective access.
- Build the core schema before snapshot and assignment transactions.
- Fetch every RevenueCat page before applying any snapshot.
- Never make provider HTTP calls inside database transactions.
- Increment requested generation for every trigger, and never clear a request
  newer than the generation being applied.
- Never mutate provider state from webhook event type or payload lifecycle fields.
- For deletion, require every persisted subscription to have
  `pending_payment=false` and exact `auto_renewal_status='will_not_renew'`;
  neither require inaccessibility nor infer cancellation from status, access,
  timestamps, or webhook types, and never call provider cancellation.
- Never persist a raw webhook body after verification/parsing completes.
- Preserve existing assignments only for the same provider subscription ID.
- Leave every newly discovered license unassigned, including new IDs created
  after lapse or paid product change.
- Treat simultaneously accessible duplicate and cross-tier states as
  non-assignable anomalies; do not add automatic repair while implementing.
- Keep one fixed transaction lock order: billing account, involved lifecycle
  rows, target sessions, provider subscriptions, licenses.
- Keep only authorization catalog inputs in AppView configuration.
- Do not implement retired IDs or the external activation checklist as runtime
  validation.
- Do not add Flutter, PDS, Lexicon, Tap, paid-feature, store-dashboard, or live
  RevenueCat configuration changes.

## 11. Risks And Open Questions

| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Non-blocking | Exact API v2 customer-subscription pagination and optional fields may evolve. | Mapping could reject valid snapshots. | Recheck current docs, capture sanitized sandbox fixtures, tolerate additive fields, and require complete pagination. |
| CPQ-002 | Resolved | Billing-period timestamps may not include provider grace semantics. | Local expiry could contradict `gives_access`. | Keep period timestamps informational; latest accepted `gives_access` alone controls access, with reconciliation staleness observable. |
| CPQ-003 | Production blocking | `Keep with original App User ID` lacks a defined recovery process. | A payer may be unable to restore under another UUID. | Define support/recovery before production activation; ownership transfer remains out of this MVP. |
| CPQ-004 | Release blocking | Flutter cannot yet prove owner-only billing identity behavior. | AC-026 is incomplete end to end. | Complete later Flutter integration and MAN-001. |
| CPQ-005 | Production blocking | Real app/product IDs, credentials, webhook URL, and merchandising configuration are absent. | Live integration cannot activate. | Complete requirements section 21 as a separate activation stage. |
| CPQ-006 | Non-blocking | No concrete paid capability is selected. | Authority exists without an end-feature test. | Add capability-specific tests when benefits are approved. |
| CPQ-007 | Non-blocking | Unexpected provider anomalies require support. | A paid customer may have an unassignable license. | Alert and expose bounded owner anomaly state; add automation only after observing real cases. |

## 12. Handoff To TDD Builder

- Coding plan: `docs/changes/2026-09-07-account-subscriptions/04-coding-plan.md`
- TDD execution plan: `05-implementation-plan.md`
- Start with: `UT-001` in
  `appview/internal/subscriptions/access_test.go`.
- Focused command:
  `cd appview && go test ./internal/subscriptions -run '^TestSelfAccessProjection$'`.
- Follow section 9 in order, maintaining a strict red-green-refactor loop.
- AppView implementation blockers: None.
- Production activation remains blocked by CPQ-003 through CPQ-005.
