# Requirements: AppView Account Subscriptions

## 1. Initial Request

Implement the AppView foundation for account-assigned Plus and Business
subscriptions described by `00-direction.md` and ADR 014. After reviewing the
original comprehensive design, the product owner selected a reconciliation-first
MVP that trusts RevenueCat to interpret store subscription lifecycle while
AppView owns billing identity, licenses, DID assignment, and server-side access.

## 2. Current Codebase Findings

- `adr/014-account-assigned-subscription-licenses.md` requires payer identity to
  remain separate from the DID receiving benefits.
- AppView has no subscription persistence, RevenueCat integration, paid-access
  endpoint, or paid feature gate today.
- Private AppView state uses PostgreSQL, authenticated `/v1/*` JSON routes, direct
  pgx transactions, and lifecycle guards.
- Existing public integrations demonstrate bounded authenticated webhooks,
  durable work before acknowledgement, retryable workers, and redacted
  observability.
- RevenueCat validates store purchases and API v2 subscription resources provide
  stable subscription IDs, `gives_access`, status, product, store, environment,
  period timestamps, and pending changes.
- RevenueCat recommends fetching current customer state after a webhook instead
  of implementing lifecycle logic for each event type.
- RevenueCat webhooks support a configured Authorization value, timestamped HMAC
  signatures, stable event IDs across retries, and at-least-once delivery.
- The Flutter client does not yet include the RevenueCat SDK. Client purchase,
  restore, management, and account-switch behavior remain a later stage.
- `just test` runs the host Go suite with PostgreSQL, and `just appview-check` is
  the release-equivalent AppView gate.

## 3. Clarifying Questions And Decisions

### Q1: Which system interprets subscription lifecycle?

Answer: RevenueCat.

Decision / implication: Webhooks are authenticated invalidation signals. AppView
does not derive access from event types. It reconciles a complete RevenueCat
customer subscription snapshot and persists RevenueCat's `gives_access` result.

### Q2: How does a purchase receive its initial assignment?

Answer: Assignment happens after purchase confirmation.

Decision / implication: A newly discovered license is unassigned. After the
owner-triggered refresh shows it, the owner explicitly assigns it. There are no
purchase-assignment intents, expiry windows, or intent-consumption rules.

### Q3: How are uncommon provider anomalies handled?

Answer: Fail closed and require manual resolution.

Decision / implication: Unexpected same-tier duplicates and cross-tier product
changes are persisted for owner visibility but do not trigger automatic
canonical promotion, assignment inheritance, or cross-tier arbitration.

### Q4: Does an assignment survive temporary loss of provider access?

Answer: Yes.

Decision / implication: The assignment remains attached to the same license
while the effective tier becomes `free`. If RevenueCat restores access on that
same subscription ID, the assignment supplies access again. RevenueCat treats a
subscription after lapse or paid product change as a new subscription; that new
license remains unassigned and requires an explicit owner action. A dormant
assignment still prevents another paid assignment to that DID until removed.

### Q5: Is billing ownership transferable at launch?

Answer: No.

Decision / implication: Billing ownership is immutable in this MVP. Remote or
same-device ownership transfer, account merging, and automated recovery are
deferred.

### Q6: What webhook evidence is retained?

Answer: Sanitized event metadata only.

Decision / implication: AppView stores the event ID, type, mapped billing
account, timestamps, and work outcome. It does not retain exact raw payloads,
invalid-envelope quarantine, collision payloads, or custom retention workers.

### Q7: Where is launch merchandising validated?

Answer: RevenueCat and store activation, not AppView runtime.

Decision / implication: AppView validates only the configured production
project/environment and product-to-tier allowlist needed for authorization.
Prices, territories, offerings, packages, trials, Family Sharing, and current
offering selection are external activation concerns.

### Q8: How does RevenueCat identity interact with CraftSky account switching?

Answer: RevenueCat represents the selected billing account, not the active DID.

Decision / implication: Ordinary switching among beneficiary DIDs does not drive
RevenueCat identity. Billing operations are enabled only for the authenticated
billing owner and use that owner's AppView-issued UUID. Anonymous purchase and
restore remain prohibited.

### Q9: What happens when a billing owner requests account deletion?

Answer: Every provider subscription must be explicitly canceled and have no
pending payment before final deletion.

Decision / implication: AppView warns and blocks final deletion while the latest
RevenueCat snapshot contains any subscription with `pending_payment=true` or
without exact `auto_renewal_status='will_not_renew'`. A canceled subscription may
still be active and give access through its paid period, or may already be
expired and inaccessible; either permits deletion under the same affirmative
non-renewal evidence. AppView does not infer cancellation from access, status,
timestamps, or webhook type and never asks RevenueCat to cancel. Ownership
transfer remains deferred.

## 4. Candidate Approaches

### Option A: Reconciliation-First AppView Licenses

Summary: Keep local billing accounts, provider subscriptions, licenses, and DID
assignments. Treat every verified webhook as a request to fetch and apply current
RevenueCat subscription resources.

Pros:

- Preserves payer/beneficiary separation and local server authorization.
- Uses RevenueCat's normalized lifecycle and access decisions.
- Eliminates event-order, lifecycle-reducer, purchase-intent, automatic-failover,
  and raw-payload-retention state machines.
- Keeps provider outages out of synchronous feature authorization.

Cons:

- Access changes wait for reconciliation rather than being derived directly from
  webhook fields.
- Unexpected duplicates and product changes may need manual support.

Risks:

- Missed webhooks can delay updates until scheduled or owner-triggered refresh.
- An incorrect product allowlist can map a valid subscription to the wrong tier.

### Option B: Comprehensive Local Lifecycle Model

Summary: Interpret every webhook and maintain local freshness, canonicalization,
automatic failover, purchase intents, ownership transfer, and forensic payload
retention.

Pros:

- Fine-grained local automation for rare lifecycle and anomaly cases.

Cons:

- Reimplements RevenueCat behavior and substantially expands schema, workers,
  concurrency rules, privacy obligations, and tests.

Risks:

- Local state may disagree with RevenueCat despite the added machinery.

### Option C: Synchronous RevenueCat Authorization

Summary: Store assignments locally but query RevenueCat during every paid
authorization decision.

Pros:

- Minimal local subscription snapshot state.

Cons:

- Adds provider latency, availability, and rate limits to paid AppView routes.

Risks:

- A RevenueCat outage disables otherwise valid server-backed paid features.

## 5. Recommended Direction

Recommended approach: Option A, reconciliation-first AppView licenses.

Why: AppView must own the relationship between payer subscriptions and assigned
DIDs, but RevenueCat already owns receipt validation and store lifecycle
normalization. This boundary preserves the desired account behavior with a much
smaller and safer implementation.

## 6. Problem / Opportunity

One native-store payer must be able to fund an independently assigned Plus
account and Business account without paid access following the installation,
active account, or complete RevenueCat customer. AppView needs a durable local
authorization projection, but it does not need to become a second subscription
lifecycle provider.

## 7. Goals

- G-001: Give each payer a stable private billing identity.
- G-002: Represent each RevenueCat subscription as one assignable license.
- G-003: Explicitly assign licenses to controlled DIDs.
- G-004: Authorize paid features from indexed AppView state.
- G-005: Converge local provider state from complete RevenueCat snapshots.
- G-006: Keep billing data private and preserve all free social behavior.
- G-007: Fail closed on unexpected catalog or provider states without building
  launch-day automatic repair machinery.

## 8. Non-Goals

- NG-001: Flutter RevenueCat SDK setup, purchase/paywall UI, Customer Center, or
  account-switcher badges.
- NG-002: Purchase-assignment intents or automatic post-purchase assignment.
- NG-003: Billing ownership transfer, billing-account merging, or automated
  account recovery.
- NG-004: Automatic duplicate-license promotion, cross-tier product-change
  arbitration, or assignment inheritance between licenses.
- NG-005: Exact raw webhook retention, malformed-payload quarantine, collision
  payload storage, or custom raw-payload purge workers.
- NG-006: Runtime validation of prices, territories, offerings, packages, trials,
  introductory offers, Family Sharing, or current offering selection.
- NG-007: Annual plans, web checkout, bundles, arbitrary repeated native
  licenses, taxes, invoicing, refunds, or provider cancellation mutations.
- NG-008: Final paid feature definitions or historical-data behavior after
  access loss.
- NG-009: Remote assignment invitations, delegated billing managers, or support
  override APIs.
- NG-010: PDS, Lexicon, Tap, feed, search, ranking, moderation, reach, or public
  data changes.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Billing owner | DID that owns one private billing account. | Stable billing identity, subscription visibility, refresh, and assignment control. |
| Assigned account | DID receiving benefits from a license. | Correct effective tier without payer or payment details. |
| Free account | DID without accessible assigned access. | Existing free behavior and an explicit free tier. |
| Flutter client | Later consumer of AppView and RevenueCat SDKs. | Separate billing-owner and active-account concepts. |
| RevenueCat | Purchase validator and lifecycle authority. | Stable App User ID and authenticated webhook destination. |
| AppView worker | Reconciles RevenueCat snapshots into local authorization state. | Durable, retryable, idempotent work. |

## 10. Current Behavior

Every authenticated account is effectively free. No private billing identity,
subscription/license state, assignment API, RevenueCat webhook, reconciliation,
or shared effective-tier query exists.

## 11. Desired Behavior

An authenticated member explicitly establishes one billing account and receives
its stable RevenueCat UUID. Purchases made under that identity are validated by
RevenueCat. A verified webhook, owner refresh, or periodic schedule causes
AppView to fetch the customer's complete RevenueCat subscription snapshot and
upsert provider subscriptions and licenses. New licenses remain unassigned until
the owner explicitly assigns them to an eligible same-device DID. AppView
authorizes each DID from its local assignment and reconciled `gives_access`
state. Simultaneously accessible duplicate or cross-tier subscriptions grant no
additional assignable access and require manual resolution. A new subscription
ID after lapse or paid product change creates a new unassigned license.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | One payer may own neither, either, or both of one native Plus license and one native Business license, independently assignable to different DIDs. | Supports the accepted personal/creator/shop use case. | Direction; ADR 014 | AC-001, AC-002 |
| BR-002 | Business | Must | Paid access shall follow the assigned DID across sessions and devices, not the installation, active account, handle, or complete RevenueCat customer. | Prevents cross-account access. | Direction; ADR 014 | AC-003 |
| BR-003 | Business | Must | AppView shall be authoritative for DID assignment and effective tier, while RevenueCat shall be authoritative for provider subscription lifecycle and `gives_access`. | Establishes a narrow trust boundary. | User-approved Option A; RevenueCat docs | AC-003, AC-006 |
| BR-004 | Business | Must | Accounts without an accessible assigned license shall remain free and retain all existing social behavior and data. | Paid access must not reduce the free experience. | Direction | AC-004 |
| BR-005 | Business | Should | Provider subscriptions and licenses should remain separate rows so later providers or quantities do not require replacing DID authorization semantics. | Preserves ADR 014's evolution path without implementing future products now. | ADR 014 | AC-005 |
| FR-001 | Functional | Must | Explicit owner selection shall idempotently create or return one private billing account with a stable opaque UUID used as the RevenueCat App User ID; ordinary login and reads shall not create it. | Billing identity must be deliberate and stable. | Direction | AC-007 |
| FR-002 | Functional | Must | Only the billing owner shall receive its billing account and RevenueCat UUID or invoke billing management operations. | Protects payer identity and authority. | Direction | AC-008 |
| FR-003 | Functional | Must | AppView shall persist each RevenueCat API v2 subscription under its configured RevenueCat project and stable subscription ID, with environment, product ID, store, status, `givesAccess`, and relevant period timestamps as mutable attributes. App identity and tier come from the configured product mapping, not from the subscription resource. | Uses RevenueCat's stable identity without turning mutable scope into identity. | RevenueCat API v2 | AC-001, AC-005, AC-009 |
| FR-004 | Functional | Must | Each supported provider subscription shall produce exactly one license linked one-to-one with that subscription; repeated snapshots update the same rows. | Preserves idempotency and license identity. | ADR 014 | AC-001, AC-009 |
| FR-005 | Functional | Must | AppView shall copy RevenueCat's current `gives_access` decision rather than derive access from webhook type or locally reproduce cancellation, grace, retry, pause, renewal, or refund rules. | RevenueCat already normalizes store lifecycle. | User-approved Option A; RevenueCat docs | AC-006, AC-010 |
| FR-006 | Functional | Must | Local access shall copy the latest successfully reconciled `gives_access` value. Billing-period timestamps are informational and shall not override it; while RevenueCat is unavailable, AppView preserves the last accepted value and records reconciliation staleness. | Avoids locally contradicting RevenueCat grace and retry semantics. | User-approved Option A; RevenueCat API v2 | AC-003, AC-010 |
| FR-007 | Functional | Must | An authenticated current-member endpoint shall return only the request DID's canonical DID, effective tier, `givesAccess`, optional access-end time, and optional assigned tier. | Supplies one account-scoped access contract. | API architecture | AC-003, AC-011 |
| FR-008 | Functional | Must | Authenticated owner endpoints shall ensure/read the billing account and list subscriptions, licenses, assignments, access and period-timing state, reconciliation staleness, anomaly state, and limited provider management context. | Separates payer management from beneficiary access. | Direction | AC-007, AC-008, AC-012 |
| FR-009 | Functional | Must | Owner assignment operations shall atomically bind one assignable license to at most one DID or no DID. | Assignment is explicit private state. | Direction | AC-002, AC-013 |
| FR-010 | Functional | Must | Assignment shall require owner authority, license ownership, an assignable license, and a current target session whose most recently seen device ID matches the request device. | Provides launch proof of target control. | Prior approved decision | AC-013 |
| FR-011 | Functional | Must | Assignment shall not depend on the target being the active Flutter account and shall revalidate target eligibility in the mutation transaction. | Avoids implicit or stale relationships. | Direction | AC-013 |
| FR-012 | Functional | Must | A DID shall have at most one current license assignment, including an assignment whose subscription is not currently accessible. | Makes later access restoration conflict-free. | User-approved Option A | AC-002, AC-014 |
| FR-013 | Functional | Must | Effective tier shall be Business for an accessible assigned Business license, Plus for an accessible assigned Plus license, and free otherwise. | Defines deterministic authorization. | Direction | AC-003, AC-004 |
| FR-014 | Functional | Must | The public RevenueCat webhook shall be outside `/v1/*`, bound request size, verify the exact configured Authorization value and RevenueCat timestamped HMAC over the raw body before parsing, and expose only generic failures. | Secures the public integration boundary. | RevenueCat docs; existing integration pattern | AC-015 |
| FR-015 | Functional | Must | A verified, structurally valid webhook shall be durably deduplicated by event ID and enqueue reconciliation for its mapped billing account before success; only sanitized metadata and work state shall be stored. | Supports at-least-once delivery without retaining provider payloads. | User-approved Option A | AC-016, AC-017 |
| FR-016 | Functional | Must | Webhook type shall not directly mutate subscription, license, assignment, tier, or access state; unknown event types shall be accepted as reconciliation signals when the customer identity maps safely. | Prevents a second lifecycle state machine. | User-approved Option A; RevenueCat docs | AC-006, AC-017 |
| FR-017 | Functional | Must | Reconciliation shall fetch a complete RevenueCat customer subscription snapshot before opening its apply transaction and idempotently upsert supported rows. Every trigger advances a requested generation; a claim captures that generation; apply rejects superseded leases and shall not clear a newer requested generation that arrived during fetch. | Makes current provider truth the only reduction input without losing concurrent triggers. | User-approved Option A | AC-006, AC-009, AC-018 |
| FR-018 | Functional | Must | Reconciliation shall run after verified webhooks, after an authenticated owner requests post-purchase refresh, and periodically, with bounded retry and stale-work recovery. | Covers delayed or missed notifications and purchase confirmation. | Direction; RevenueCat docs | AC-018 |
| FR-019 | Functional | Must | Only configured RevenueCat project, production environment, app identities, and product-to-tier mappings may grant production access; unknown or sandbox subscriptions shall not grant production benefits. | Prevents catalog or environment confusion. | Security constraint | AC-019 |
| FR-020 | Functional | Must | Reconciled loss of access shall immediately affect effective-tier reads and authorization without deleting public or retained private member data. | Access changes permission, not ownership. | Direction | AC-004, AC-010 |
| FR-022 | Functional | Must | Account deletion shall warn a billing owner, request reconciliation, and block final deletion until a complete snapshot requested after deletion started shows every persisted subscription has `pending_payment=false` and exact `auto_renewal_status='will_not_renew'`. Active/currently accessible and expired/inaccessible subscriptions are both eligible under that condition. Missing, empty, unknown, `will_renew`, `will_change_product`, `will_pause`, `requires_price_increase_consent`, `has_already_renewed`, or pending-payment evidence shall block. Confirmed deletion may then close local billing authority and live license state without a provider cancellation request. Cancellation shall not be inferred from timestamps, status, access, or webhook type. Deleting any non-owner DID shall atomically remove assignments targeting that DID. | Requires affirmative provider cancellation evidence while allowing deletion during a canceled subscription's remaining access period, and prevents assignments to deleted members. | ADR 014; user-approved deletion-policy correction | AC-020 |
| FR-026 | Functional | Must | Multiple simultaneously accessible same-tier subscriptions or a paid cross-tier product change shall be visible to the owner as anomalies but shall not automatically promote, inherit, move, or create additional assignable access. A new subscription created after a prior subscription lapses is not itself an anomaly, but its license remains unassigned. | Rare provider anomalies do not justify automatic launch machinery. | User-approved Option A; RevenueCat API v2 | AC-021 |
| FR-027 | Functional | Must | RevenueCat unavailability shall preserve the last accepted `givesAccess` value while reconciliation retries and staleness is observable; AppView shall not invent expiry or grace behavior from billing-period timestamps. | Keeps RevenueCat authoritative during outages. | User-approved Option A | AC-010 |
| FR-028 | Functional | Must | An assigned non-owner's self-access response shall expose no payer, provider, product, cancellation, billing-issue, anomaly, or management data. | Protects billing privacy. | Direction | AC-011 |
| FR-029 | Functional | Must | A license assignment shall persist while that provider subscription is inaccessible and supply access again only if a later snapshot restores `givesAccess` on the same RevenueCat subscription ID. A new post-lapse or paid-product-change subscription creates a new unassigned license. | Preserves ordinary recovery without inventing lineage across RevenueCat subscriptions. | User-approved Option A; RevenueCat API v2 | AC-010, AC-011, AC-021 |
| FR-030 | Functional | Must | Initial assignment shall not start a cooldown; changing an assigned license from one DID to another shall be allowed at most once every seven days. No prior-target or automatic-failover exception shall exist. | Retains simple abuse protection. | User-approved Option A | AC-022 |
| FR-031 | Functional | Must | A closed billing account shall retain only its internal ID, RevenueCat App User UUID, and closure time so delayed events can be acknowledged and ignored; it shall not be reconciled or recreate subscriptions, licenses, assignments, ownership, or access. | Prevents resurrection without terminal lifecycle machinery. | User-approved Option A | AC-020 |
| FR-032 | Functional | Must | Restore and scheduled reconciliation shall preserve known assignments and leave newly discovered licenses unassigned. | Provider ownership does not express beneficiary intent. | Direction | AC-018 |
| NFR-001 | Non-functional | Must | Billing state shall remain private PostgreSQL data and shall cause no PDS, Lexicon, Tap, feed, search, ranking, moderation, or reach mutation. | Preserves architecture and product principles. | Direction | AC-004 |
| NFR-002 | Non-functional | Must | `/v1/*` subscription APIs shall follow existing authentication, current-member, device-ID, body, rate, camelCase JSON, and error-envelope conventions. | Maintains the API contract. | API architecture | AC-012, AC-013 |
| NFR-003 | Non-functional | Must | Provider credentials, RevenueCat secret keys, raw webhook bodies, receipts, payment details, and private billing identifiers shall not appear in logs, metrics, errors, operator listings, or unauthorized responses. Raw webhook bodies shall not be persisted. | Minimizes sensitive-data exposure. | User-approved Option A | AC-008, AC-015, AC-023 |
| NFR-004 | Non-functional | Must | Webhook acceptance and reconciliation work shall be transactionally recoverable and idempotent. | Prevents lost or duplicate state changes. | Existing worker pattern | AC-016, AC-018 |
| NFR-005 | Non-functional | Must | Webhook acknowledgement shall use a configured ingress deadline no greater than ten seconds and occur after durable queueing; provider fetch and snapshot application shall never run in ingress. | Stays well within RevenueCat's 60-second timeout without brittle wall-clock assumptions. | RevenueCat docs | AC-016 |
| NFR-006 | Non-functional | Should | Effective-tier reads should use a bounded indexed local lookup and no synchronous RevenueCat call. | Keeps authorization available during provider outages. | AppView authority | AC-003 |
| NFR-007 | Non-functional | Must | Migrations shall support up, down, and up again without weakening existing session or account-deletion constraints. | Billing touches account lifecycle. | Codebase | AC-024 |
| NFR-008 | Non-functional | Must | The release suite shall cover billing identity, assignment, snapshot reconciliation, webhook security/deduplication, anomaly handling, access changes and staleness, deletion, privacy, migrations, and existing auth/social regressions. | Provides proportionate billing assurance. | Risk assessment | AC-024 |
| RULE-001 | Business rule | Must | One active billing account has exactly one owner DID; an owner has at most one billing account; ownership is immutable in this MVP. | Defines launch management authority. | User-approved Option A | AC-007, AC-008 |
| RULE-002 | Business rule | Must | One RevenueCat subscription produces one license; RevenueCat entitlements or combined CustomerInfo never determine DID assignment. | Separates provider access from beneficiary intent. | ADR 014 | AC-001, AC-003 |
| RULE-003 | Business rule | Must | A license has at most one assigned DID and a DID has at most one assigned paid license, whether accessible or dormant. | Prevents overlap and reactivation conflict. | Direction | AC-002, AC-014 |
| RULE-004 | Business rule | Must | The billing owner need not receive either owned license. | Payer and beneficiary are separate. | ADR 014 | AC-002 |
| RULE-005 | Business rule | Must | Handle changes shall not affect ownership, assignment, or tier because relationships bind to DIDs. | Handles are mutable. | ADR 014 | AC-003 |
| RULE-006 | Business rule | Must | Sandbox and production provider state shall remain isolated. | Test purchases must not grant production access. | Security constraint | AC-019 |
| RULE-007 | Business rule | Must | AppView paid tier keys are `plus` and `business`; `free` is the no-access projection. Capability inheritance is defined later with concrete paid features. | Stabilizes authorization terminology without testing unspecified benefits. | Prior approved decision | AC-025 |
| RULE-008 | Business rule | Must | AppView shall return a RevenueCat billing UUID and accept owner refresh only for the authenticated owner of that UUID; it shall never accept an active DID, anonymous identity, or client-supplied substitute as billing identity. Flutter purchase/restore identity behavior remains a later activation requirement. | Enforces the server half of payer identity without claiming Flutter scope. | User-approved Option A; RevenueCat identity docs | AC-026 |

Retired from the MVP: `FR-021`, `FR-023` through `FR-025`, `FR-033`,
`RULE-009`, and `RULE-010`. Their former assignment history, ownership transfer,
purchase-intent, raw-payload retention, runtime merchandising validation, and
automatic inheritance behavior is now explicitly covered by the non-goals.

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-003, FR-004, RULE-002 | Given one billing customer owns supported Plus and Business subscriptions, repeated complete snapshots produce one distinct subscription and license row for each. |
| AC-002 | BR-001, FR-009, FR-012, RULE-003, RULE-004 | The owner can assign Plus to DID A and Business to DID B, neither must be the owner, and assigning both to one DID fails atomically. |
| AC-003 | BR-002, BR-003, FR-006, FR-007, FR-013, NFR-006, RULE-002, RULE-005 | An assigned DID receives its locally derived tier across sessions, devices, handle changes, and active-account switches without a synchronous RevenueCat request. |
| AC-004 | BR-004, FR-013, FR-020, NFR-001 | A DID without accessible assignment remains free and subscription transitions alter no social data, behavior, classification, or reach. |
| AC-005 | BR-005, FR-003 | Provider subscriptions and licenses are separate rows keyed by stable RevenueCat subscription identity, without implementing future providers or quantities. |
| AC-006 | BR-003, FR-005, FR-016, FR-017 | Cancellation, renewal, billing, pause, refund, and unknown webhook types never directly determine local access; the subsequently fetched RevenueCat snapshot does, and a trigger arriving during fetch remains pending afterward. |
| AC-007 | FR-001, FR-008, RULE-001 | Explicit repeated or concurrent owner selection returns one billing account and stable UUID, while ordinary reads create nothing. |
| AC-008 | FR-002, NFR-003, RULE-001 | Only the owner can read or manage its billing account, and no response or diagnostic leaks protected billing data. |
| AC-009 | FR-003, FR-004, FR-017 | Repeated and concurrent application of the same or older complete snapshot leaves one subscription/license at the newest accepted snapshot state. |
| AC-010 | FR-005, FR-006, FR-020, FR-027, FR-029 | The latest accepted `givesAccess` controls authorization even when billing-period timestamps have passed; provider failure preserves that value with observable staleness, and a later snapshot updates it. Same-subscription recovery preserves assignment. |
| AC-011 | FR-007, FR-028, FR-029 | Assigned-account self access exposes only DID, effective tier, access boolean, optional end, and optional assigned tier; inaccessible assignment returns effective free without payer facts. |
| AC-012 | FR-008, NFR-002 | Owner billing state follows the standard private camelCase API contract and distinguishes subscriptions, licenses, assignments, access, informational period timing, reconciliation staleness, and anomalies. |
| AC-013 | FR-009, FR-010, FR-011, NFR-002 | Assignment succeeds only under owner, license, current-member, same-device, and transactional revalidation rules, regardless of which Flutter account is active. |
| AC-014 | FR-012, RULE-003 | Accessible and dormant assignments participate in the same one-license-per-DID uniqueness constraint under concurrency. |
| AC-015 | FR-014, NFR-003 | Missing/invalid Authorization or HMAC, stale signature, malformed body, or oversized input grants nothing, stores no raw body, and returns a generic result. |
| AC-016 | FR-015, NFR-004, NFR-005 | A verified event is durably stored once and queues reconciliation before `200`; persistence failure remains retryable and provider fetch never runs in ingress. |
| AC-017 | FR-015, FR-016 | Duplicate and unknown verified events are idempotent reconciliation signals and never directly mutate access or assignment. |
| AC-018 | FR-017, FR-018, FR-032, NFR-004 | Webhook, owner-refresh, restore, and scheduled reconciliation converge through one snapshot path; known assignments remain, new licenses are unassigned, stale lease completion is rejected, and a trigger received during fetch remains due. |
| AC-019 | FR-019, RULE-006 | Sandbox or unknown-product snapshots, and product mappings that name an unconfigured app, never grant production access. |
| AC-020 | FR-022, FR-031 | Owner deletion requests reconciliation and remains blocked until a post-intent complete snapshot shows every persisted subscription is non-pending and exactly `will_not_renew`. Under that evidence, both active/currently accessible and expired/inaccessible sets permit atomic removal of live authority and assignments; mixed sets block if any row can renew/resume, has missing/empty/unknown auto-renewal status, or has pending payment. Deletion retains only a DID-free closed marker, ignores later events, never infers cancellation from timestamps/status/access/webhook type, and never calls provider cancellation. Deleting an assigned non-owner atomically unassigns its licenses. |
| AC-021 | FR-026, FR-029 | Simultaneously accessible same-tier or paid cross-tier anomaly snapshots cannot create automatic promotion, inheritance, movement, or additional assignable access; a normal post-lapse new subscription is unassigned rather than anomalous. |
| AC-022 | FR-030 | Initial assignment is unrestricted; a second target change inside seven days fails without changing the current assignment, and succeeds at or after the boundary. |
| AC-023 | NFR-003 | Secret and raw-provider canaries are absent from persistence, logs, metrics, errors, listings, and client responses. |
| AC-024 | NFR-007, NFR-008 | `just appview-check` passes migration up/down/up and the revised subscription, auth, deletion, and social regression suites. |
| AC-025 | RULE-007 | API and storage contracts use only `free`, `plus`, and `business` tier keys. |
| AC-026 | RULE-008 | AppView exposes billing readiness and accepts refresh only for the authenticated owner under the server-stored UUID, without accepting any client-supplied DID, anonymous ID, or replacement UUID. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | No billing account exists. | Self access returns free without creating one. | FR-001, FR-007 |
| EC-002 | A new purchase is reconciled. | Its license is visible and unassigned until explicit assignment. | FR-004, FR-032 |
| EC-003 | Webhook events arrive out of order. | Each queues a current-state fetch; event order cannot regress access. | FR-016, FR-017 |
| EC-004 | RevenueCat is unavailable. | The latest accepted `givesAccess` remains authoritative, period timestamps do not change access, staleness is observable, and retries continue. | FR-006, FR-018, FR-027 |
| EC-005 | Access returns on the same subscription ID. | The existing assignment supplies access again without moving licenses. | FR-029 |
| EC-006 | Simultaneously accessible same-tier subscription appears. | Preserve safe existing assignment, mark anomaly, and grant no extra assignable access. | FR-026 |
| EC-007 | New subscription ID appears after lapse or paid product change. | Create a new unassigned license without moving or inheriting the old assignment; classify an anomaly only if simultaneous accessible state conflicts. | FR-026, FR-029 |
| EC-008 | Target session changes during assignment. | Transaction revalidation rejects and preserves prior state. | FR-010, FR-011 |
| EC-009 | Authenticated webhook is malformed. | Reject generically, persist no raw bytes, and grant nothing. | FR-014, NFR-003 |
| EC-010 | Duplicate verified event ID arrives. | Return success after confirming prior durable acceptance; queue at most one equivalent pending reconciliation. | FR-015 |
| EC-011 | Closed billing UUID receives a delayed event. | Acknowledge and ignore it without reconciliation or state recreation. | FR-031 |
| EC-012 | Owner deletes with active/current access, expired subscriptions, uncertain renewal state, or mixed provider rows. | Warn and request reconciliation. Permit closure only when every persisted row has `pending_payment=false` and exact `auto_renewal_status='will_not_renew'`, regardless of access or expiry; otherwise block without a provider cancellation call. | FR-022 |
| EC-013 | An assigned non-owner DID is deleted. | Atomically remove every assignment targeting that DID without changing billing ownership or provider state. | FR-022 |

## 15. Data / Persistence Impact

- `billing_accounts`: active owner, stable RevenueCat UUID, closure state, and
  reconciliation generation/timestamps.
- `provider_subscriptions`: stable RevenueCat subscription identity and latest
  sanitized API v2 snapshot fields.
- `billing_licenses`: one-to-one subscription link, tier, assignment, anomaly
  state, and reassignment cooldown timestamp.
- `revenuecat_events`: sanitized event identity and durable reconciliation work;
  no raw payload.
- Migrations are required and must be reversible.
- Existing free accounts require no billing rows.

## 16. UI / API / CLI Impact

- UI: None in this AppView stage.
- API: Add self access, billing ensure/read, assignment/unassignment, and
  owner-triggered refresh routes under `/v1/*`.
- Integration: Add `POST /integrations/revenuecat/webhook` outside `/v1/*`.
- CLI: None.
- Background jobs: One reconciliation processor handles webhook, scheduled,
  restore-like, and owner-refresh work.

## 17. Security / Privacy / Permissions

- All app-facing routes require current-member authentication and device ID.
- Billing state and mutations are owner-only; self access is DID-scoped.
- Assignment revalidates same-device target control transactionally.
- Webhooks require Authorization and HMAC verification before parsing.
- Raw webhook payloads, credentials, receipts, and payment details are never
  persisted or logged.
- Product and environment allowlists constrain production authorization.

## 18. Observability

- Record sanitized webhook accepted/duplicate/rejected outcomes, reconciliation
  success/failure/staleness, anomaly counts, assignment outcomes, and closure.
- Do not use RevenueCat App User IDs, DIDs, provider subscription identifiers,
  secrets, or raw bodies as logs or metric labels.
- Alert on sustained webhook authentication failure, old reconciliation work,
  provider failures, and subscription anomalies.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Webhooks are missed or delayed. | Local access changes late. | Scheduled reconciliation and owner refresh use the same idempotent path. |
| RISK-002 | A stale snapshot finishes after a newer one, or a trigger arrives during fetch. | Provider state regresses or reconciliation work is lost. | Fence apply with lease/claimed generation and keep `requested_generation > reconciled_generation` due. |
| RISK-003 | Billing owner and beneficiary are conflated. | Wrong DID receives access. | Separate identity, routes, rows, and authorization checks. |
| RISK-004 | RevenueCat is unavailable near renewal. | Local access may remain stale until reconciliation succeeds. | Preserve latest accepted `gives_access`, expose staleness, retry, and never infer access from period timestamps. |
| RISK-005 | Unexpected duplicate/product transition occurs. | Customer may need support. | Fail closed, preserve safe assignment, expose owner anomaly, and alert. |
| RISK-006 | `Keep with original App User ID` blocks recovery. | A legitimate payer cannot restore under another billing UUID. | Require exact UUID and define support/recovery before production activation. |
| RISK-007 | External catalog differs from AppView's product map. | Purchase remains unrecognized. | Activation audit and minimal explicit product-to-tier configuration. |
| RISK-008 | RevenueCat omits or adds an auto-renewal status, or reports pending payment after cancellation. | Account deletion remains blocked despite user intent. | Fail closed until a complete snapshot gives exact `will_not_renew` and non-pending evidence; do not infer cancellation locally or call provider cancellation. |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | A RevenueCat API v2 subscription ID is stable for that subscription resource, while lapse or paid product change may create a new ID; `gives_access` remains authoritative. | Subscription/license identity or reconciliation must be revised. |
| ASM-002 | One native Plus and one native Business product prevent ordinary same-tier duplicate purchases. | More automated anomaly handling may be required. |
| ASM-003 | Same-device current sessions sufficiently prove target control at launch. | Invitation or consent flows are required. |
| ASM-004 | Seven days is an acceptable reassignment interval. | Only the cooldown policy changes. |
| ASM-005 | A billing owner need not receive an owned license. | Assignment rules and UX must tighten. |
| ASM-006 | RevenueCat `auto_renewal_status='will_not_renew'` together with `pending_payment=false` is authoritative affirmative evidence that a persisted subscription cannot renew or resume; access, status, timestamps, and webhook types are not cancellation evidence. | Provider-specific deletion resolution or broader ownership transfer may be required if RevenueCat cannot provide this exact evidence. |

## 21. External Activation Checklist

The following merchandising decisions remain recorded for store and RevenueCat
activation but are not AppView runtime requirements or automated AppView catalog
tests:

- Plus and Business are monthly subscriptions with no trial or introductory
  offer at launch.
- US base prices are USD 2.49 for Plus and USD 7.99 for Business.
- Apple Family Sharing is disabled.
- Apple product IDs are `social.craftsky.app.plus.monthly` and
  `social.craftsky.app.business.monthly` in separate subscription groups.
- Google products are `social.craftsky.app.plus` and
  `social.craftsky.app.business`, each with base plan `monthly`.
- RevenueCat entitlements are `plus` and `business`; separate offerings use the
  `$rc_monthly` package and no global current offering.
- Territory availability and price equalization are verified at activation.
- Restore behavior is `Keep with original App User ID` and requires an account
  recovery/support procedure before production launch.

## 22. Open Questions

- [ ] Production activation: supply native app/product credentials, RevenueCat
  resource IDs, public webhook URL, and API keys.
- [ ] Production activation: define recovery for a payer who cannot access the
  original billing owner/UUID.
- [ ] Later product scope: select concrete Plus/Business capabilities and their
  expired-access behavior.
- [ ] Later Flutter scope: finalize RevenueCat SDK identity selection and billing
  management UX without tying identity to ordinary active-DID switching.

## 23. Review Status

Status: Revised after simplification approval and validated deletion-policy feedback

Risk level: High

Review recommended: Required before implementation

Reviewer: Product owner

Date: 2026-09-10

Notes: The product owner approved Option A, reconciliation-first licenses, and
explicitly moved catalog merchandising out of AppView runtime on 2026-09-09.
Validated manual feedback on 2026-09-10 approved the exact non-renewal and
non-pending account-deletion rule in FR-022 and AC-020.

## 24. Handoff To Test Design

- Requirements file: `docs/changes/2026-09-07-account-subscriptions/01-requirements.md`
- Next artifact: `02-acceptance-tests.md`
- Must-cover IDs: all Must requirements in section 12.
- Focus unit coverage on access projection, snapshot mapping/fencing, allowlists,
  HMAC verification, anomaly classification, and cooldown.
- Focus integration coverage on constraints, reconciliation transactions,
  webhook durability, assignments, closure, routes, and migrations.
- Blocking AppView test-design questions: None.
