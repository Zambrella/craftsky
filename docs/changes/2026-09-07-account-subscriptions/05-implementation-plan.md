# TDD Implementation Plan: AppView Account Subscriptions

## Inputs

- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` (`Approved`)
- Coding plan: `04-coding-plan.md`

## Implementation Rules

- Do not implement behavior without a linked requirement ID.
- Write or update one failing test before its implementation.
- Run the smallest relevant test first.
- Refactor only after tests pass.
- Keep traceability and executed commands updated after every loop.
- Keep RevenueCat fetches outside database transactions.
- Never derive access from webhook event types or billing-period timestamps.
- Never persist raw webhook bodies, credentials, receipts, or payment details.
- Do not add Flutter, PDS, Lexicon, Tap, merchandising, or paid-feature changes.

## Test Order

| Step | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| 1 | UT-001 | FR-005 through FR-007, FR-013, FR-027 through FR-029 | AC-003, AC-010, AC-011 | Subscription package absent |
| 2 | IT-001 | BR-001, BR-005, FR-003, FR-004, RULE-001 through RULE-003 | AC-001, AC-002, AC-005 | Tables absent |
| 3 | IT-002 | FR-001, RULE-001 | AC-007 | Ensure operation absent |
| 4 | AT-001 | FR-001, FR-002, FR-008, RULE-001, NFR-003 | AC-007, AC-008 | Routes absent |
| 5 | UT-002 | BR-003, FR-003, FR-005, FR-017 | AC-006, AC-009 | Snapshot mapper absent |
| 6 | UT-003 | FR-019, RULE-006 | AC-019 | Catalog absent |
| 7 | UT-005 | FR-026, FR-029 | AC-021 | Anomaly policy absent |
| 8 | IT-005 | FR-003, FR-004, FR-017, FR-019, FR-026, FR-029 | AC-009, AC-019, AC-021 | Snapshot transaction absent |
| 9 | AT-006 | BR-003, FR-005, FR-016 through FR-019, FR-032, NFR-004, RULE-006 | AC-006, AC-009, AC-018, AC-019 | Reconciliation absent |
| 10 | UT-004 | FR-014 | AC-015 | Verifier absent |
| 11 | IT-006 | FR-015, FR-016, NFR-004, NFR-005 | AC-016, AC-017 | Event store absent |
| 12 | AT-007 | FR-014 through FR-016, NFR-003 through NFR-005 | AC-015, AC-016, AC-017 | Handler absent |
| 13 | IT-007 | FR-017, FR-018, FR-032, NFR-004 | AC-018 | Processor absent |
| 14 | IT-004 | FR-007, FR-020, FR-027, NFR-006 | AC-003, AC-010 | Access query absent |
| 15 | AT-003 | BR-002, BR-003, FR-006, FR-007, FR-013, FR-028, FR-029, NFR-006, RULE-002, RULE-005 | AC-003, AC-011 | Access route absent |
| 16 | IT-003 | FR-009 through FR-012, FR-030, RULE-003 | AC-013, AC-014, AC-022 | Assignment absent |
| 17 | UT-006 | FR-030 | AC-022 | Cooldown absent |
| 18 | AT-005 | FR-009 through FR-012, FR-030, RULE-003 | AC-013, AC-014, AC-022 | Assignment routes absent |
| 19 | AT-002 | BR-001, FR-003, FR-004, FR-009, FR-012, RULE-002 through RULE-004 | AC-001, AC-002 | Full flow incomplete |
| 20 | AT-008 | FR-005, FR-006, FR-020, FR-027, FR-029 | AC-010 | Access-authority flow incomplete |
| 21 | AT-011 | FR-026, FR-029 | AC-021 | Fail-closed flow incomplete |
| 22 | IT-008 | FR-022, FR-031 | AC-020 | Closure participant absent |
| 23 | AT-009 | FR-022, FR-031 | AC-020 | Deletion contract incomplete |
| 24 | AT-004 | FR-002, FR-008, NFR-002, RULE-008 | AC-008, AC-012, AC-026 | Owner contract incomplete |
| 25 | IT-010 | RULE-008 | AC-026 | Readiness check absent |
| 26 | AT-010, AT-012 | BR-004, FR-013, FR-020, NFR-001, NFR-003 | AC-004, AC-023 | Guardrails unproven |
| 27 | REG-001 through REG-006 | Subscription-linked requirements | AC-003, AC-004, AC-008, AC-012, AC-013, AC-020, AC-023, AC-024 | Regression coverage absent |
| 28 | IT-009, AT-013 | NFR-007, NFR-008 | AC-024 | Full release gate incomplete |

## Implementation Steps

### Step 1: UT-001
- Write failing test: `TestSelfAccessProjection` for free, Plus, Business, dormant, stale, and passed-period cases.
- Run command: `cd appview && go test ./internal/subscriptions -run '^TestSelfAccessProjection$'`
- Confirmed failure: Build failed because `LocalAssignment`, `SelfAccess`, tier constants, and `ProjectSelfAccess` were absent.
- Implement: Added the closed tier constants, five-field `SelfAccess`, local assignment projection input, and a pure projector that copies provider `givesAccess` without interpreting timestamps or staleness.
- Run command: `cd appview && go test ./internal/subscriptions -run '^TestSelfAccessProjection$'` passed.
- Refactor: None; implementation is already the minimum pure projection.
- Notes: A dormant assignment retains `assignedTier` and informational `accessEndsAt`; only `givesAccess` selects a paid effective tier.

### Step 2: IT-001
- Write failing test: `TestCoreBillingConstraints` applies migration `000069` and exercises active-owner uniqueness, stable provider-subscription identity, one-to-one licenses, one assignment per DID including dormant licenses, separate row cardinality, and subscription-to-license cascade.
- Run command: `cd appview && TEST_DATABASE_URL=... TEST_DATABASE_REQUIRED=true go test ./internal/subscriptions -run '^TestCoreBillingConstraints$'`.
- Confirmed failure: After starting the stopped worktree Compose stack, the focused test failed reading absent `000069_subscription_accounts.up.sql`. The earlier stopped-service result was discarded as infrastructure setup, not TDD evidence.
- Implement: Added reversible migration `000069_subscription_accounts` with billing account lifecycle/generation/lease constraints, stable provider subscriptions, one-to-one licenses, assignment uniqueness, tier/anomaly checks, cascades, and bounded access/reconciliation indexes.
- Run command: The focused command passed; `cd appview && TEST_DATABASE_URL=... TEST_DATABASE_REQUIRED=true go test ./internal/subscriptions` also passed.
- Refactor: Removed an unused test fixture field; no production refactor.
- Notes: Provider subscriptions and licenses remain separate rows. Closed accounts require a DID-free marker, while all non-null assignments share one uniqueness constraint regardless of provider access.

### Step 3: IT-002
- Write failing test: `TestEnsureBillingAccountIsExplicitStableAndConcurrent` first performs a non-creating owner read, then releases eight concurrent ensure calls and repeats ensure afterward.
- Run command: `cd appview && TEST_DATABASE_URL=... TEST_DATABASE_REQUIRED=true go test ./internal/subscriptions -run '^TestEnsureBillingAccountIsExplicitStableAndConcurrent$'`.
- Confirmed failure: Build failed because `Store`, `BillingAccount`, `ErrBillingAccountNotFound`, `OwnerAccount`, and `EnsureAccount` were absent.
- Implement: Added the minimal pgx `Store`, owner read, and idempotent `INSERT ... ON CONFLICT DO NOTHING` ensure path using server-generated UUIDs and typed DIDs/UUIDs.
- Run command: The focused command passed; the complete `internal/subscriptions` package also passed against PostgreSQL.
- Refactor: Shared one row scanner between ensure and read while green.
- Notes: Reads do not create billing identity. Concurrent explicit selection converges through the database owner uniqueness constraint and returns one stable RevenueCat App User UUID.

### Step 4: AT-001
- Write failing test: `TestBillingIdentityIsExplicitStableAndPrivate` verifies the empty precondition, first/repeated explicit ensure statuses and stable UUID, owner read, and a canonical non-leaking unrelated-member response.
- Run command: `cd appview && TEST_DATABASE_URL=... TEST_DATABASE_REQUIRED=true go test ./internal/routes -run '^TestBillingIdentityIsExplicitStableAndPrivate$'`.
- Confirmed failure: Build failed because the route dependency had no subscription store and the approved billing routes/handlers were absent.
- Implement: Added narrow billing account handlers, current-member route policies, a subscription route bundle, route dependency wiring, and camelCase account/RevenueCat UUID output. Owner-not-found maps to a standard generic 404.
- Run command: The focused command passed. Nearby `go test ./internal/routes ./internal/subscriptions` initially exposed the expected exact-route inventory drift; after adding only the two approved routes to that inventory, both packages passed.
- Refactor: Shared authenticated owner scoping, response projection, and error mapping while green.
- Notes: Both routes use current-member, device, body, rate, and canonical envelope middleware. The endpoint never accepts a client-supplied billing UUID or DID.

### Step 5: UT-002
- Write failing test: `TestClientMapsCompleteCustomerSubscriptionSnapshot` serves two API v2 pages containing additive fields and verifies stable IDs, direct `gives_access`, status/store/environment, pending-payment/renewal fields, and millisecond timestamps. At this original step it did not include or assert `pending_changes.product_id`; IR-004 below completes that requirement.
- Run command: `cd appview && go test ./internal/integrations/revenuecat -run '^TestClientMapsCompleteCustomerSubscriptionSnapshot$'`.
- Confirmed failure: Build failed because `ClientConfig`, `NewClient`, and the complete customer-subscription fetcher were absent.
- Implement: Added a bounded API v2 client that authenticates with the configured secret, follows only same-origin `next_page` URLs, rejects incomplete page-limit traversal, bounds response bytes, tolerates additive JSON, and maps sanitized subscription snapshots.
- Run command: The focused command passed; `go test ./internal/integrations/revenuecat ./internal/subscriptions` passed (database-only tests skip when no database URL is supplied to this nearby unit command).
- Refactor: Extracted response-resource mapping and millisecond conversion while green.
- Notes: Current RevenueCat docs were rechecked through Context7: list pagination uses `next_page`, retries retain event IDs, and `gives_access` is authoritative even as status definitions evolve. App identity and tier are intentionally absent from provider-resource mapping and will come from the configured catalog.

### Step 6: UT-003
- Write failing test: `TestCatalogAuthorizesOnlyConfiguredProductionProducts` covers Plus/Business and unknown project, sandbox, unknown product, unconfigured app, and invalid tier cases.
- Run command: `cd appview && go test ./internal/subscriptions -run '^TestCatalogAuthorizesOnlyConfiguredProductionProducts$'`.
- Confirmed failure: Build failed because `CatalogConfig`, `ProductMapping`, `NewCatalog`, and `Authorize` were absent.
- Implement: Added an immutable copied authorization catalog that requires the exact project, `production`, a known product, a configured mapped app, and a paid tier before returning authorization attributes.
- Run command: The focused command passed; `go test ./internal/subscriptions ./internal/integrations/revenuecat` passed (database tests skipped without a URL).
- Refactor: None; the direct fail-closed checks are the minimum implementation.
- Notes: The catalog contains no prices, territories, offering/package state, trials, or other merchandising concerns.

### Step 7: UT-005
- Write failing test: `TestClassifySnapshotAnomaliesFailsClosedWithoutInventingLineage` covers legitimate independent tiers, safe and ambiguous same-tier duplicates, an explicit paid cross-tier change, and a post-lapse new ID.
- Run command: `cd appview && go test ./internal/subscriptions -run '^TestClassifySnapshotAnomaliesFailsClosedWithoutInventingLineage$'`.
- Confirmed failure: Build failed because anomaly candidates, decisions, codes, and classification were absent.
- Implement: Added pure classification that makes only accessible ordinary licenses assignable, preserves one safely assigned same-tier license, marks duplicate extras/ambiguity unassignable, and contains explicitly provider-reported cross-tier changes without inferring lineage from new IDs.
- Run command: The focused command and complete `internal/subscriptions` package passed.
- Refactor: Kept classification in one function and used the existing tier pointer fixture while green.
- Notes: Simultaneous Plus and Business subscriptions are legitimate by default. Cross-tier anomaly classification requires explicit pending-change evidence rather than treating the approved two-license product shape as anomalous.

### Step 8: IT-005
- Write failing test: `TestApplyCompleteSnapshotsIsFencedIdempotentAndFailClosed` applies claimed generations covering older-state rejection, same-ID dormancy/recovery, a post-lapse new ID, sandbox/unknown/unconfigured-app rows, same-tier duplication, and provider-reported cross-tier change.
- Run command: `cd appview && TEST_DATABASE_URL=... TEST_DATABASE_REQUIRED=true go test ./internal/subscriptions -run '^TestApplyCompleteSnapshotsIsFencedIdempotentAndFailClosed$'`.
- Confirmed failure: Build failed because `SnapshotClaim`, `ErrStaleSnapshot`, and `Store.ApplySnapshot` were absent.
- Implement: Added one fenced pgx transaction that locks the claimed active account, maps the complete snapshot through the catalog, classifies anomalies with existing assignments, upserts stable provider/license rows, preserves same-ID assignments, leaves new licenses unassigned, marks absent known rows inaccessible, and advances only the claimed generation.
- Run command: The first implementation run exposed pgx rejecting two SQL commands in one prepared `Exec`; splitting those two local updates fixed it. The focused command then passed, followed by real-PostgreSQL `go test ./internal/subscriptions ./internal/integrations/revenuecat`.
- Refactor: Added small anomaly and nullable-string persistence adapters while green; provider fetch remains outside this transaction by interface design.
- Notes: Unsupported and sandbox subscriptions remain provider-visible with `unsupported` state but produce no license. Billing timestamps are stored but do not participate in access decisions.

### Step 9: AT-006
- Write failing test: `TestReconciliationTriggersUseOneCompleteSnapshotPath` queues webhook, owner-refresh, restore-like, and scheduled generations, injects a fifth trigger during provider fetch, and verifies generation 5 remains due after generation 4 applies.
- Run command: `cd appview && TEST_DATABASE_URL=... TEST_DATABASE_REQUIRED=true go test ./internal/subscriptions -run '^TestReconciliationTriggersUseOneCompleteSnapshotPath$'`.
- Confirmed failure: Build failed because reconciliation triggers and the shared reconciler did not exist.
- Implement: Added generation-incrementing trigger storage, a short `FOR UPDATE SKIP LOCKED` claim transaction with lease token, and one reconcile operation that commits the claim before provider fetch and opens snapshot apply only after the complete fetch returns.
- Run command: The focused command and complete real-PostgreSQL `internal/subscriptions` package passed.
- Refactor: Kept trigger type informational; every trigger follows the same queue path and cannot encode lifecycle mutation.
- Notes: Applying generation 4 advances only `reconciled_generation` to 4 and leaves requested generation 5 due. Provider fetch is structurally outside both claim and apply transactions.

### Step 10: UT-004
- Write failing test: `TestVerifyWebhookAuthenticationUsesExactRawBody` covers missing/wrong/duplicate Authorization, missing/duplicate/malformed signature, duplicate components, stale/future timestamps, valid input, and body mutation.
- Run command: `cd appview && go test ./internal/integrations/revenuecat -run '^TestVerifyWebhookAuthenticationUsesExactRawBody$'`.
- Confirmed failure: Build failed because `VerifyWebhookAuthentication` was absent.
- Implement: Added exact single-header validation, constant-time Authorization comparison, unique timestamp/signature parsing, symmetric five-minute freshness, hex HMAC-SHA256 over `timestamp + "." + rawBody`, and constant-time HMAC comparison.
- Run command: The focused command and complete `internal/integrations/revenuecat` package passed.
- Refactor: None; verification remains one bounded function before parsing.
- Notes: RevenueCat documentation confirms retry signatures are recomputed while stable event IDs are reused. No parsed or reserialized bytes enter signature verification.

### Step 11: IT-006
- Write failing test: `TestAcceptRevenueCatEventDeduplicatesSanitizedWorkTransactionally` races eight stable-ID deliveries, verifies one metadata row/generation, retries after a forced constraint failure, and inventories prohibited columns.
- Run command: `cd appview && TEST_DATABASE_URL=... TEST_DATABASE_REQUIRED=true go test ./internal/subscriptions -run '^TestAcceptRevenueCatEventDeduplicatesSanitizedWorkTransactionally$'`.
- Confirmed failure: Build failed because sanitized event types/storage were absent; migration `000070` was also intentionally not present.
- Implement: Added reversible `000070_revenuecat_events` with bounded sanitized metadata only, plus one pgx transaction that maps the server-stored UUID, inserts idempotently, and advances reconciliation only for a newly accepted active-account event.
- Run command: The focused command and complete real-PostgreSQL `internal/subscriptions` package passed.
- Refactor: Shared zero-time null conversion while green.
- Notes: Unknown event types are accepted exactly like known types. Unmapped and closed customers are durably acknowledged without queueing; no raw body, customer UUID, receipt, payload, or secret column exists.

### Step 12: AT-007
- Write failing test: `TestWebhookHandlerAuthenticatesBoundsAndQueuesSanitizedEvents` covers valid unknown/duplicate delivery, generic invalid auth, malformed and oversized bodies, retryable persistence failure, and the sanitized event boundary.
- Run command: `cd appview && go test ./internal/integrations/revenuecat -run '^TestWebhookHandlerAuthenticatesBoundsAndQueuesSanitizedEvents$'`.
- Confirmed failure: Build failed because `WebhookConfig` and `NewWebhookHandler` were absent.
- Implement: Added a disabled-unless-complete webhook handler with an ingress deadline capped at ten seconds, bounded raw read, authentication before parsing, narrow additive-tolerant envelope parsing, UUID validation, sanitized store call, and generic status responses.
- Run command: The focused command and complete `internal/integrations/revenuecat` package passed.
- Refactor: None; provider fetching is impossible through the handler dependency graph.
- Notes: The raw body exists only in the bounded request scope for HMAC and parsing. Receipt/payment/additive fields are discarded and never cross the store interface.

### Step 13: IT-007
- Write failing test: The interrupted prior run added `TestReconciliationProcessorRetriesAndRejectsSupersededLeases` for provider failure retry state, successful retry, expired-lease recovery, stale completion rejection, and replacement completion.
- Run command: Prior red output is unavailable after interruption. The current focused verification was `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:<worktree-port>/craftsky_dev?sslmode=disable TEST_DATABASE_REQUIRED=true go test ./internal/subscriptions -run '^TestReconciliationProcessorRetriesAndRejectsSupersededLeases$' -count=1`.
- Confirmed failure: The prior meaningful red reportedly occurred before the interruption, but no surviving command output is available; this plan does not claim independently observed red evidence.
- Implement: Existing dirty-worktree code adds a retrying processor over the shared claim/fetch/apply path, clears failed leases with a bounded retry delay, recovers expired leases with a new token, and fences stale applies. Inspection confirms provider fetch occurs after the claim transaction commits and before snapshot apply opens its transaction.
- Run command: The focused real-PostgreSQL rerun passed (`ok social.craftsky/appview/internal/subscriptions 0.318s`).
- Refactor: None during this resumed run; the existing implementation is focused on the approved processor behavior.
- Notes: This is honest post-interruption green evidence. The test verifies the last accepted snapshot is untouched on provider failure and a superseded token cannot apply; Step 9 already covers trigger-during-fetch generation preservation.

### Steps 14-28
- Follow the approved order above one test ID at a time.
- Record each meaningful red failure, minimum implementation, focused green command, nearby regression command, and any refactor here before advancing.

### Step 14: IT-004
- Write failing test: `TestSelfAccessReadsLatestProviderTruthLocally` seeds a stale reconciliation and a passed current-period end while persisted `gives_access` remains true, then checks paid and no-assignment projections.
- Run command: `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:<worktree-port>/craftsky_dev?sslmode=disable TEST_DATABASE_REQUIRED=true go test ./internal/subscriptions -run '^TestSelfAccessReadsLatestProviderTruthLocally$' -count=1`.
- Confirmed failure: Build failed because `Store.SelfAccess` was absent. A subsequent multi-command fixture error was test setup only and was corrected before accepting green evidence.
- Implement: Added one indexed assignment/provider join by DID and projected its stored tier, `gives_access`, and informational period end through `ProjectSelfAccess`; no provider dependency or timestamp decision exists on this path.
- Run command: The focused real-PostgreSQL command passed (`ok ... 0.426s`). Nearby access projection and snapshot regressions also passed (`ok ... 0.340s`).
- Refactor: None; the single query and existing pure projector are the minimum implementation.
- Notes: Reconciliation staleness and the passed period timestamp do not override RevenueCat's latest accepted provider truth. Missing assignment returns explicit free access without creating billing state.

### Step 15: AT-003
- Write failing test: `TestSelfAccessRouteIsDIDBoundLocalAndMinimal` reads an assigned member's accessible and dormant states and inventories the exact permitted self-access fields against provider/payer/anomaly canaries.
- Run command: `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:<worktree-port>/craftsky_dev?sslmode=disable TEST_DATABASE_REQUIRED=true go test ./internal/routes -run '^TestSelfAccessRouteIsDIDBoundLocalAndMinimal$' -count=1`.
- Confirmed failure: The request returned the fallback `404 page not found` because the approved self-access route was absent.
- Implement: Added `GET /v1/subscriptions/access` with current-member/read/no-body policy, authenticated DID scoping, injected clock, local `Store.SelfAccess`, and direct serialization of the five-field minimal projection.
- Run command: The focused command passed. The access, route-policy inventory, and local projection regression group passed across `internal/routes` and `internal/subscriptions` (`ok ... 0.668s`, `ok ... 0.846s`).
- Refactor: Defaulted the route clock only at route composition while keeping the API handler clock-injected.
- Notes: The response contains only DID, effective tier, `givesAccess`, informational access end, and assigned tier. It exposes no payer UUID, product, store, provider subscription identifier, status, or anomaly.

### Step 16: IT-003
- Write failing test: `TestAssignLicenseRevalidatesOwnerTargetDeviceAndUniquenessAtomically` covers valid owner assignment, dormant one-license-per-DID enforcement, latest-session device mismatch, wrong-owner concealment, and preservation of the prior assignment after failures.
- Run command: `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:<worktree-port>/craftsky_dev?sslmode=disable TEST_DATABASE_REQUIRED=true go test ./internal/subscriptions -run '^TestAssignLicenseRevalidatesOwnerTargetDeviceAndUniquenessAtomically$' -count=1`.
- Confirmed failure: Build failed because `AssignParams`, assignment errors, and `Store.Assign` were absent. A timestamp parameter inference error in the first implementation run was fixture setup and was corrected with explicit casts.
- Implement: Added one assignment transaction with fixed owner, current-member/session/device, provider/license lock order; owner/license scoping; latest active unexpired target-session proof; assignability; dormant DID uniqueness mapping; cooldown enforcement; and commit-only response.
- Run command: The focused real-PostgreSQL command passed (`ok ... 0.405s`). The assignment plus core billing-constraint regression group passed (`ok ... 0.336s`).
- Refactor: Kept assignment-specific persistence in `assignment.go`; no broader store refactor.
- Notes: Failed checks return before mutation or roll back. Wrong-owner lookup is intentionally indistinguishable from a missing license, and the database partial unique index remains the concurrency authority for accessible and dormant assignments.

### Step 17: UT-006
- Write failing test: `TestCanChangeAssignmentTargetUsesSevenDayBoundary` covers initial assignment and changes immediately before, exactly at, and immediately after the seven-day boundary.
- Run command: `cd appview && go test ./internal/subscriptions -run '^TestCanChangeAssignmentTargetUsesSevenDayBoundary$' -count=1`.
- Confirmed failure: Build failed because the pure cooldown decision function was absent.
- Implement: Added `CanChangeAssignmentTarget` with an inclusive seven-day boundary and delegated the assignment transaction's existing cooldown check to it.
- Run command: The focused unit command passed (`ok ... 0.377s`). The unit boundary and real-PostgreSQL assignment integration regression group passed (`ok ... 0.288s`).
- Refactor: Replaced the inline timestamp comparison with the tested pure policy while green.
- Notes: A nil prior target-change timestamp permits initial assignment; equality at seven days permits reassignment.

### Step 18: AT-005
- Write failing test: `TestAssignmentRoutesEnforceDeviceUniquenessCooldownAndUnassignment` exercises initial assignment, dormant target conflict, same-device rejection, first target change, cooldown rejection, and unassignment retaining the target-change timestamp.
- Run command: `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:<worktree-port>/craftsky_dev?sslmode=disable TEST_DATABASE_REQUIRED=true go test ./internal/routes -run '^TestAssignmentRoutesEnforceDeviceUniquenessCooldownAndUnassignment$' -count=1`.
- Confirmed failure: The first assignment returned fallback `404 page not found` because assignment routes and handlers were absent.
- Implement: Added strict camelCase assignment parsing, typed DID/UUID boundary validation, middleware device scoping, standard non-leaking assignment errors, owner-scoped unassignment that preserves cooldown state, and authenticated current-member route policies for PUT/DELETE assignment operations.
- Run command: The focused real-PostgreSQL command passed (`ok ... 0.559s`). Route inventory, integration assignment, and cooldown regressions passed across route/subscription packages (`ok ... 0.389s`, `ok ... 0.680s`).
- Refactor: Shared assignment error mapping across assign and unassign while green.
- Notes: Assignment does not depend on a client active-account concept; the target's latest current same-device session is revalidated in the mutation transaction.

### Step 19: AT-002
- Write failing test: Added `TestPlusAndBusinessLicensesAreIndependentlyAssigned` for repeated complete snapshot application, one neutral license per tier, non-owner beneficiaries, independent assignments, overlapping-DID rejection, and preserved access after the rejected change.
- Run command: `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:<worktree-port>/craftsky_dev?sslmode=disable TEST_DATABASE_REQUIRED=true go test ./internal/subscriptions -run '^TestPlusAndBusinessLicensesAreIndependentlyAssigned$' -count=1`.
- Confirmed failure: No red occurred: the new vertical acceptance test passed immediately because Steps 8, 14, and 16 already supplied the minimum snapshot, access, and assignment behavior. No artificial failure or redundant abstraction was introduced.
- Implement: None required for this acceptance composition.
- Run command: The focused real-PostgreSQL command passed (`ok ... 0.446s`). The snapshot, assignment, access, and AT-002 regression group passed (`ok ... 0.470s`).
- Refactor: None.
- Notes: This is an explicit TDD-loop exception rather than fabricated red evidence. The test confirms payer/beneficiary separation and atomic overlap rejection through public store operations.

### Step 20: AT-008
- Write failing test: Added `TestReconciledAccessUsesProviderTruthWithoutLocalExpiry` for passed period time, provider failure and retry visibility, same-ID access loss with dormant assignment, and same-ID recovery.
- Run command: `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:<worktree-port>/craftsky_dev?sslmode=disable TEST_DATABASE_REQUIRED=true go test ./internal/subscriptions -run '^TestReconciledAccessUsesProviderTruthWithoutLocalExpiry$' -count=1`.
- Confirmed failure: No red occurred: the composed behavior passed on its first run from Steps 8, 13, and 14. No artificial production change was made.
- Implement: None required for this acceptance composition.
- Run command: The focused real-PostgreSQL command passed (`ok ... 0.426s`). Reconciliation retry, local access, and snapshot regressions passed (`ok ... 0.425s`).
- Refactor: None.
- Notes: Provider failure records retry timing while leaving accepted access untouched. The expired informational period end remains visible but never overrides `gives_access`; only later complete snapshots change access.

### Step 21: AT-011
- Write failing test: Added `TestProviderAnomaliesFailClosedWithoutMovingAssignments` for safe assigned same-tier containment, paid cross-tier conflict, and a normal post-lapse replacement ID.
- Run command: `cd appview && go test ./internal/subscriptions -run '^TestProviderAnomaliesFailClosedWithoutMovingAssignments$' -count=1`.
- Confirmed failure: No red occurred because Step 7's pure anomaly policy already implemented the acceptance behavior; no artificial change was introduced.
- Implement: None required for the acceptance composition.
- Run command: The focused command passed (`ok ... 0.352s`). The pure classifier plus real-PostgreSQL snapshot-application anomaly regressions passed (`ok ... 0.302s`).
- Refactor: None.
- Notes: Existing assignments are never moved. Conflicting extra licenses are unassignable and owner-visible through persisted bounded anomaly codes; a new ID is not anomalous solely because it is new.

### Step 22: IT-008
- Write failing test: `TestSubscriptionDeletionParticipantBlocksClosesAndUnassigns` covers post-intent generation, accessible/pre-reconciliation blocking, terminal closure, live-row cascade, non-owner target unassignment, and delayed closed-account event handling.
- Run command: `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:<worktree-port>/craftsky_dev?sslmode=disable TEST_DATABASE_REQUIRED=true go test ./internal/accountdeletion -run '^TestSubscriptionDeletionParticipantBlocksClosesAndUnassigns$' -count=1`.
- Confirmed failure: Build failed because `subscriptions.NewDeletionParticipant` and `ErrProviderBillingMustBeResolved` were absent. Multi-command prepared fixture errors were corrected before green evidence.
- Implement: Added a database-only deletion participant that atomically clears target assignments, advances and records the owner deletion generation, requires post-intent reconciliation, fails closed unless every row is expired/inaccessible/not pending/not renewing, deletes live provider/license state, and closes the DID-free account marker. Delayed events map the retained UUID to `ignored_closed` without queueing.
- Run command: The focused real-PostgreSQL test passed (`ok ... 0.451s`); account-deletion and subscription persistence regressions passed (`ok ... 0.317s`, `ok ... 0.673s`). After lifecycle composition, the focused participant and existing deletion lifecycle regressions passed (`ok ... 1.928s`).
- Refactor: Exposed one narrow `BillingDeletionParticipant` interface and composed it into existing lifecycle transactions; no provider cancellation capability was added.
- Notes: Intent reports whether the DID owns billing so only owners receive the warning. Confirmation closure occurs inside the accepted lifecycle transition transaction; provider I/O is absent.

### Step 23: AT-009
- Write failing test: Added `TestSubscriptionAwareAccountDeletionWarnsAndBlocksGenerically` for the owner warning object and exact standard conflict envelope.
- Run command: `cd appview && go test ./internal/accountdeletion -run '^TestSubscriptionAwareAccountDeletionWarnsAndBlocksGenerically$' -count=1`.
- Confirmed failure: No red occurred because the warning and conflict mapping were added while composing Step 22 into the lifecycle transaction. This ordering dependency is recorded rather than hidden.
- Implement: The Step 22 composition adds an owner-only `warning` with `provider_billing_not_canceled`, and maps terminal-state rejection to `409 provider_billing_must_be_resolved`; existing confirmation semantics remain unchanged.
- Run command: The focused command passed (`ok ... 0.428s`). The acceptance, persistence participant, and existing exact-DID deletion regression group passed (`ok ... 0.341s`, `ok ... 0.444s`).
- Refactor: None after the acceptance test.
- Notes: The API does not claim provider cancellation and exposes no provider identifiers. Non-billing-owner intent results omit the warning.

### Step 24: AT-004
- Write failing test: Added `TestOwnerBillingStateUsesPrivateCamelCaseContract` with one stale reconciliation, provider subscription, assigned license, and bounded anomaly.
- Run command: `cd appview && POSTGRES_ADDRESS=$(../scripts/compose-dev port postgres 5432) && POSTGRES_PORT=${POSTGRES_ADDRESS##*:} && TEST_DATABASE_URL="postgres://craftsky:dev@localhost:${POSTGRES_PORT}/craftsky_dev?sslmode=disable" TEST_DATABASE_REQUIRED=true go test ./internal/routes -run '^TestOwnerBillingStateUsesPrivateCamelCaseContract$' -count=1`.
- Confirmed failure: GET returned only billing identity; `reconciliationStale`, `subscriptions`, and `licenses` were absent.
- Implement: Added a transactionally consistent owner billing aggregate with local IDs, safe product/store/status and renewal context, provider access truth, assignments, reconciliation generations/timestamps/staleness, and bounded anomalies. GET and idempotent PUT return this aggregate.
- Run command: Focused test passed (`ok ... 0.531s`); focused route inventory, billing identity, and snapshot regressions passed (`ok ... 0.723s`, `ok ... 0.425s`).
- Refactor: Initialized aggregate collections to empty arrays and kept raw provider project, app-user subscription ID, and environment fields off the wire.
- Notes: The route remains DID-bound and does not perform provider I/O.

### Step 25: IT-010
- Write failing test: Added `TestOwnerRefreshUsesOnlyServerStoredBillingIdentity` for owner refresh, cross-owner rejection, client UUID override rejection, and exact persisted UUID targeting.
- Run command: `cd appview && POSTGRES_ADDRESS=$(../scripts/compose-dev port postgres 5432) && POSTGRES_PORT=${POSTGRES_ADDRESS##*:} && TEST_DATABASE_URL="postgres://craftsky:dev@localhost:${POSTGRES_PORT}/craftsky_dev?sslmode=disable" TEST_DATABASE_REQUIRED=true go test ./internal/routes -run '^TestOwnerRefreshUsesOnlyServerStoredBillingIdentity$' -count=1`.
- Confirmed failure: `POST /v1/billing/reconciliation` was not registered and returned 404.
- Implement: Added the current-member, write-rate, no-body reconciliation request route. It resolves the billing account from authenticated owner DID, increments that account's generation, and returns `202 {"status":"pending"}`.
- Run command: Focused test and exact route inventory passed together (`ok ... 0.508s`).
- Refactor: Kept provider identity out of the request contract; non-empty bodies are rejected by standard middleware before the handler.
- Notes: This trigger only records local pending work. The worker remains the sole RevenueCat fetch path.

### Step 26: AT-010, AT-012
- Write failing test: Added `TestBillingTransitionsDoNotMutateFreeSocialData` and `TestBillingPersistenceContainsNoRawPayloadOrCredentialColumns`; the former holds profile/post/feed/moderation sentinels across snapshot/access/assignment transitions, and the latter inventories both billing migrations for prohibited secret/raw payload fields.
- Run command: `cd appview && POSTGRES_ADDRESS=$(../scripts/compose-dev port postgres 5432) && POSTGRES_PORT=${POSTGRES_ADDRESS##*:} && TEST_DATABASE_URL="postgres://craftsky:dev@localhost:${POSTGRES_PORT}/craftsky_dev?sslmode=disable" TEST_DATABASE_REQUIRED=true go test ./internal/subscriptions ./internal/integrations/revenuecat ./internal/routes -run '^(TestBillingTransitionsDoNotMutateFreeSocialData|TestBillingPersistenceContainsNoRawPayloadOrCredentialColumns|TestRevenueCatWebhook|TestOwnerBillingStateUsesPrivateCamelCaseContract|TestAcceptRevenueCatEventDeduplicatesSanitizedWorkTransactionally)' -count=1`.
- Confirmed failure: No red occurred because the earlier transaction boundaries, sanitized webhook event model, generic ingress failures, and private owner projection already supplied the guardrails. This verification-only result is recorded explicitly.
- Implement: No additional production behavior was needed.
- Run command: Subscription, RevenueCat ingress, and owner-route privacy groups passed (`ok ... 0.773s`, `ok ... 0.362s`, `ok ... 0.376s`).
- Refactor: None.
- Notes: Provider lifecycle changes affect only billing authorization rows. The persistence schemas have no columns for raw bodies, credentials, receipts, or payment details.

### Step 27: REG-001 through REG-006
- Write failing test: Added `TestRouteDependenciesIncludeSubscriptionStore` after inspection found that the production dependency adapter dropped the implemented route store.
- Run command: `cd appview && go test ./internal/app -run '^TestRouteDependenciesIncludeSubscriptionStore$' -count=1`.
- Confirmed failure: Build failed because process `Deps` had no `Subscriptions` field.
- Implement: Added the narrow subscription dependency constructor, process field, route lowering, and account-deletion billing participant. Added billing owner/beneficiary roles to terminal DID inventory and the required role-leading partial indexes.
- Run command: Composition and constructor architecture tests passed (`ok ... 0.714s`). The terminal inventory initially failed on missing `(owner_did,id)` and `(assigned_did,id)` indexes; after adding them, all terminal inventory tests passed (`ok ... 1.535s`). The broad focused package run passed except one transient PostgreSQL `out of shared memory` failure caused by concurrently creating many isolated schemas; that exact account-deletion test passed alone (`ok ... 0.783s`).
- Refactor: Moved `subscriptions.NewStore` out of `newDeps` into `newSubscriptionDependencies` to preserve the repository's capability-constructor boundary.
- Notes: `just test` later passed every package serially with the race detector, including all account-deletion and owner-lifecycle regressions.

### Step 28: IT-009, AT-013
- Write failing test: Added `TestSubscriptionMigrationsUpDownUpPreserveExistingState` for two complete `000069`/`000070` up/down cycles with auth, session, and deletion sentinels.
- Run command: `cd appview && POSTGRES_ADDRESS=$(../scripts/compose-dev port postgres 5432) && POSTGRES_PORT=${POSTGRES_ADDRESS##*:} && TEST_DATABASE_URL="postgres://craftsky:dev@localhost:${POSTGRES_PORT}/craftsky_dev?sslmode=disable" TEST_DATABASE_REQUIRED=true go test ./internal/db -run '^TestSubscriptionMigrationsUpDownUpPreserveExistingState$' -count=1`.
- Confirmed failure: No red occurred because both reversible migration files had already been implemented in Steps 2 and 11. This verification-only result is recorded explicitly.
- Implement: Removed an obsolete billing response helper during cleanup before running the release gate; no behavioral change.
- Run command: Migration test passed (`ok ... 0.426s`). `just fmt` passed `gofmt` and `go vet ./...`. `just test` passed all packages serially with `-race`. After the helper cleanup and formatting, `just appview-check` completed successfully with `appview-check: all release gates passed`; the gate itself did not report or discover the obsolete helper.
- Refactor: Removed the superseded two-field response projection now that billing endpoints return `BillingState`.
- Notes: The release vulnerability scan reports existing `GO-2026-5932` in transitive `golang.org/x/crypto/openpgp` with no upstream fixed version; the repository gate treats it as acknowledged and passed.

## Execution Notes

## Supplemental Review-Fix Steps

### Supplemental IR-001: Runtime RevenueCat Composition And Processing
- Red: Added `TestRevenueCatConfigIsDisabledByDefaultAndRequiresCompleteConfiguration`, `TestRevenueCatCompositionRequiresCompleteConfiguration`, `TestRevenueCatWebhookRouteIsPublicAndConditional`, and `TestRevenueCatProcessorStartsOnceSchedulesProcessesAndStops`. `go test ./internal/app -run '^(TestRevenueCatConfigIsDisabledByDefaultAndRequiresCompleteConfiguration|TestRevenueCatCompositionRequiresCompleteConfiguration)$' -count=1`, `go test ./internal/routes -run '^TestRevenueCatWebhookRouteIsPublicAndConditional$' -count=1`, and `go test ./cmd/appview -run '^TestRevenueCatProcessorStartsOnceSchedulesProcessesAndStops$' -count=1` failed to build because the config loader/type, complete composition fields, route registrar, and processor startup function were absent.
- Implementation: Added disabled-by-default, complete-bundle-validated, formatting-redacted RevenueCat configuration; complete-config-only catalog, bounded no-redirect client, webhook, and reconciler construction; conditional lowering and registration of `POST /integrations/revenuecat/webhook`; active-account periodic generation scheduling; and one cancellable command processor that drains owner-refresh, webhook, retry, and stale-lease-recovery work through the existing `ProcessOne` path.
- Green: The three exact focused commands passed: `ok social.craftsky/appview/internal/app 1.299s`, `ok social.craftsky/appview/internal/routes 0.464s`, and `ok social.craftsky/appview/cmd/appview 0.603s`.
- Refactor: Kept the always-available local store separate from optional provider runtime dependencies, reused the existing generation/lease reconciler, and centralized periodic scheduling on `Store.ScheduleActiveAccounts`; no second processor or ingress path was added.
- Notes: Incomplete configuration exposes no provider handler or worker. Complete configuration carries only authorization catalog inputs and bounded operational budgets; credentials and private catalog values are redacted from `%v`, `%+v`, and `%#v` formatting. The worker is included in the existing bounded shutdown wait.

### Supplemental IR-002: RevenueCat Privacy And Bounded Observability Canaries
- Red: Added `TestRevenueCatObservationsAreBoundedAndPrivate`, expanded `TestWebhookHandlerAuthenticatesBoundsAndQueuesSanitizedEvents` with accepted/duplicate/malformed/error outcomes, expanded sensitive-header and owner/non-owner checks, and added provider-error and command-log canaries. `go test ./internal/observability ./internal/integrations/revenuecat ./cmd/appview ./internal/routes -run '^(TestRevenueCatObservationsAreBoundedAndPrivate|TestRedactHeadersRemovesSensitiveTelemetryValues|TestWebhookHandlerAuthenticatesBoundsAndQueuesSanitizedEvents|TestClientProviderErrorsDoNotExposeResponseCanaries|TestRevenueCatProcessorLogsOnlyBoundedProviderFailure|TestOwnerBillingStateUsesPrivateCamelCaseContract)$' -count=1` failed to build because RevenueCat observation methods and webhook observer wiring were absent; the existing provider-error, route, and worker-log canaries already passed independently.
- Implementation: Added closed-vocabulary webhook and reconciliation observations for metrics and operator logs; wired them through webhook and reconciler composition; classified `X-RevenueCat-Webhook-Signature` as sensitive; retained generic provider/client, webhook, and command-worker errors without raw error text or private identifiers.
- Green: The exact focused multi-package command passed: `ok social.craftsky/appview/internal/observability 0.887s`, `ok social.craftsky/appview/internal/integrations/revenuecat 0.365s`, `ok social.craftsky/appview/cmd/appview 0.858s`, and `ok social.craftsky/appview/internal/routes 1.231s`.
- Refactor: Used optional narrow observer interfaces so RevenueCat telemetry does not expand unrelated recorder contracts. Shared sanitizers constrain every operation/outcome before either logging or metric emission.
- Notes: Metrics contain only bounded operation/outcome labels. Logs contain only component, operation, result, and bounded failure stage. DIDs, billing UUIDs, event/subscription IDs, credentials, raw bodies, provider response bodies, and raw errors are not recorded. Authorized owner responses retain the approved billing UUID while non-owner and assigned-account responses expose no billing identity.

### Supplemental IR-003: Social And Retained-Private Isolation Coverage
- Red: Replaced the toy `social_sentinels` check with `TestBillingLifecycleIsIsolatedFromSocialAndRetainedPrivateState`, using representative `craftsky_profiles`, `craftsky_posts`, `atproto_follows`, `actor_mutes`, `saved_posts`, `moderation_reports`, and `scheduled_posts` fixtures. The first focused run, `go test ./internal/subscriptions -run '^TestBillingLifecycleIsIsolatedFromSocialAndRetainedPrivateState$' -count=1`, passed (`ok social.craftsky/appview/internal/subscriptions 0.387s`), so no behavioral red existed; IR-003 was a missing-coverage finding and no artificial production defect was introduced.
- Implementation: No production behavior was required. The acceptance flow applies provider snapshots through the reconciler, assigns a license, contains a same-tier duplicate anomaly, proves local access loss and recovery, clears an assigned non-owner during deletion, and closes a terminal billing owner.
- Green: After strengthening the fixtures with fail-on-mutation triggers and exact before/after table snapshots, the same focused command passed (`ok social.craftsky/appview/internal/subscriptions 0.376s`). The full race suite then exposed two fixture issues: combined prepared-statement inserts and an omitted duplicate subscription whose prior `active` status correctly blocked owner closure. Splitting the inserts and supplying explicit terminal states for both provider subscriptions made ten consecutive race-enabled runs pass (`go test -race ./internal/subscriptions -run '^TestBillingLifecycleIsIsolatedFromSocialAndRetainedPrivateState$' -count=10`, `ok ... 1.402s`). The controlled provider fails the test on any unexpected customer or excess fetch and confirms exactly five expected snapshot fetches; all local access, assignment, and deletion operations remain database-only.
- Refactor: Added a reusable canonical table-snapshot helper and retained mutation triggers so update/delete attempts fail at their source while inserts or truncation are caught by exact snapshot comparison.
- Notes: Billing lifecycle operations changed only billing tables. Representative public social state and retained private moderation, mute, saved-post, and scheduled-post state remained byte-for-byte equivalent across every transition. No PDS, Tap, feed, search, ranking, moderation, reach, or classification capability is present on these production billing APIs.

### Supplemental IR-004: Closed Tier Validation
- Red: Added the dedicated UT-007 table test `TestTierValidationIsClosed` for `free`, `plus`, `business`, empty, unknown, and wrong-case values in both complete-tier and paid-provider contexts. Its first focused run, `go test ./internal/subscriptions -run '^TestTierValidationIsClosed$' -count=1`, passed (`ok social.craftsky/appview/internal/subscriptions 0.386s`) because `Tier.Valid` and `Tier.Paid` had already been added while implementing IR-001 catalog configuration; this ordering dependency is recorded rather than manufacturing a false red.
- Implementation: No additional production behavior was required. `Tier.Valid` accepts exactly `free`, `plus`, and `business`; `Tier.Paid` accepts exactly `plus` and `business` for provider product mappings.
- Green: The exact focused UT-007 command passed on the implementation under review (`ok social.craftsky/appview/internal/subscriptions 0.386s`).
- Refactor: Kept the two closed-domain predicates directly on `Tier`; no validator abstraction or duplicate tier list was added.
- Notes: The direct domain test now complements database constraints and catalog tests, including rejection of empty, unknown, and case-variant values.

### Supplemental Final Verification (Superseded By Implementation Review)
- Focused regressions: The combined serial IR-001 through IR-004 command passed for `internal/app`, `internal/routes`, `cmd/appview`, `internal/observability`, `internal/integrations/revenuecat`, and `internal/subscriptions` (`ok` for all six packages).
- Formatting and static analysis: `just fmt` completed `gofmt -w .` and `go vet ./...`; a separate `go vet ./...` also passed with no output.
- Full tests: The final `just test` race-enabled suite passed every AppView package. Two earlier runs exposed and corrected isolation-test fixture defects; neither failure required a production-code change.
- Release gate: `just appview-check` completed with `appview-check: all release gates passed`, including migrations, tests, builds, image checks, and vulnerability auditing.
- Known external finding: The release scan continues to report acknowledged transitive `GO-2026-5932` in `golang.org/x/crypto/openpgp`; no fixed upstream version is available and the repository gate passed.
- Independent final verification: `just test` passed every AppView package with the race detector, and a subsequent isolated `just appview-check` completed with `appview-check: all release gates passed`.

- 2026-09-09: Contract loaded. Document review is approved with no AppView implementation blockers.
- 2026-09-09: User requested implementation, authorizing the approved high-risk billing, webhook, migration, and deletion scope.
- 2026-09-09: Current migration head is `000068`; planned `000069` and `000070` are available.
- 2026-09-09: Existing code patterns selected: `testdb.WithSchema`, owner-lifecycle transaction locks/participants, generation and lease-token fencing, current-session device revalidation, standard route bundles and error envelopes.
- 2026-09-09: `just dev-d` started the isolated worktree stack required for real PostgreSQL red/green evidence.
- Commits are disabled because the user did not request a stage commit.

## Post-Review TDD Corrections

### IR-001: Previously Supported Licenses Fail Closed
- Requirements/tests: FR-019, FR-020, RULE-006; AC-019; IT-005.
- Red: Extended `TestApplyCompleteSnapshotsIsFencedIdempotentAndFailClosed` with an assigned supported production subscription transitioning first to an unknown product and then to sandbox while RevenueCat still reports `gives_access=true`. `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:15830/craftsky_dev?sslmode=disable' TEST_DATABASE_REQUIRED=true go test ./internal/subscriptions -run '^TestApplyCompleteSnapshotsIsFencedIdempotentAndFailClosed$' -count=1` failed because self access remained Plus with access.
- Implementation: Kept the provider row and existing assignment, made the retained license unassignable while unsupported, and required the joined provider row to retain a valid production mapping before its `gives_access` can grant effective access.
- Green: The exact focused command passed (`ok social.craftsky/appview/internal/subscriptions 0.472s`). Nearby IT-005/access regressions passed with `go test ./internal/subscriptions -run '^(TestApplyCompleteSnapshotsIsFencedIdempotentAndFailClosed|TestSelfAccessReadsLatestProviderTruthLocally|TestReconciledAccessUsesProviderTruthWithoutLocalExpiry)$' -count=1` (`ok ... 0.392s`).
- Refactor: None; the authorization predicate remains in the indexed local access query and unsupported reconciliation uses one targeted license update.

### IR-002: Assignment Uses Authoritative Lifecycle Fencing
- Requirements/tests: FR-010, FR-011, NFR-002; AC-013; IT-003, REG-005.
- Red: Extended `TestAssignLicenseRevalidatesOwnerTargetDeviceAndUniquenessAtomically` with real lifecycle rows, deletion-pending/departed targets, and an exclusive lifecycle transition racing assignment on a separate connection. After correcting the session lifetime so lifecycle was the only rejection reason, the focused command failed because a deletion-pending target was assigned (`error = <nil>`). A final forced transition/assignment interleaving then exposed reverse lock acquisition: assignment held the billing account while waiting for the shared owner fence, and PostgreSQL aborted it with `SQLSTATE 40P01` when the fenced transition attempted to lock that billing account.
- Implementation: Assignment now takes the repository's canonical transaction-scoped shared owner fence via `ownerlifecycle.LockOwnerStatesTx` before any database row lock, requires the target lifecycle to be active, then locks the billing account and rechecks `appview_owner_is_active` while locking the profile/session before provider/license rows.
- Green: `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:15830/craftsky_dev?sslmode=disable' TEST_DATABASE_REQUIRED=true go test ./internal/subscriptions -run '^TestAssignLicenseRevalidatesOwnerTargetDeviceAndUniquenessAtomically$' -count=1` passed after the final lock-order correction (`ok ... 0.603s`). The uncached serial race regression across `internal/subscriptions`, `internal/accountdeletion`, and `internal/routes` passed (`2.365s`, `6.328s`, `4.407s`).
- Refactor: Updated the route fixture to model authoritative active lifecycle rows rather than relying on the pre-lifecycle all-active test predicate.

### IR-003: Reversible Deletion Intent Preserves Assignments
- Requirements/tests: FR-022, FR-029, FR-031; AC-020; IT-008, AT-009, REG-002.
- Red: Extended `TestSubscriptionDeletionParticipantBlocksClosesAndUnassigns` with self-assigned owner and non-owner target licenses. The focused test failed because `BeginDeletion` cleared the self-assignment during reversible intent creation (`assignment count = 0, want 1`).
- Implementation: Removed target unassignment from `BeginDeletion` and moved it into `ConfirmDeletion`, which already executes inside the irreversible accepted/deleting lifecycle transaction. Billing-owner closure and non-owner target unassignment therefore commit or roll back with acceptance.
- Green: The focused participant command passed (`ok social.craftsky/appview/internal/accountdeletion 0.484s`). Existing full-service tests now use the production billing participant and prove assignment preservation through blocked confirmation, explicit cancellation, intent expiry, and OAuth-start rollback/failure; the accepted terminal owner path closes billing and removes its self-assignment, and accepted non-owner deletion removes only its target assignment. The combined full-service/participant command passed (`ok ... 1.636s`).
- Refactor: Removed an unnecessary constraint-error fixture and made pending/accepted assignment assertions direct while green.

### IR-004: RevenueCat Pending Product Changes Reach Reconciliation
- Requirements/tests: FR-003, FR-026, FR-031; AC-009, AC-021; UT-002, IT-005.
- Documentation: Rechecked the current RevenueCat API v2 subscription reference through Context7 (`/websites/revenuecat_api-v2`). `pending_changes` is an optional object containing only fields that differ at the next billing-period transition, including scheduled product changes.
- Red: Added `pending_changes.product_id` plus nested/top-level additive fields to `TestClientMapsCompleteCustomerSubscriptionSnapshot` and asserted `PendingProductID`. The focused test failed with an empty pending product ID.
- Implementation: Added additive-tolerant decoding of optional `pending_changes.product_id` and copied it into the sanitized provider snapshot.
- Green: The exact mapper command passed (`ok social.craftsky/appview/internal/integrations/revenuecat 0.329s`). Added a provider-boundary PostgreSQL fixture that fetches the API v2 JSON and applies it through the real store; the combined mapper/persistence command passed (`ok ... 0.394s`) and proves the pending Business product persists and makes both provider/license anomaly state `cross_tier_conflict` with `assignable=false`.
- Refactor: Kept the provider response shape private to the RevenueCat client and exposed only the existing sanitized `PendingProductID` field.

### IR-005: Webhook Deadline Covers Full Ingress
- Requirements/tests: FR-014, NFR-005; AC-015, AC-016; AT-007, IT-006.
- Red: Added `TestWebhookIngressDeadlineCoversBodyRead` with a deterministic blocking `ReadCloser`. The handler failed to return within 250ms for a configured 10ms deadline, proving the prior timeout began before reading but could not interrupt body ingestion.
- Implementation: The handler now derives and installs its deadline context before any body work, schedules body closure when that context expires, checks deadline state after bounded ingestion, and returns a generic retryable `503` without authenticating, parsing, or calling the durable store. Added `deadline_exceeded` to the closed webhook observation vocabulary.
- Green: `go test ./internal/integrations/revenuecat -run '^TestWebhookIngressDeadlineCoversBodyRead$' -count=1` passed (`ok ... 0.369s`). The handler/deadline/observability regression command passed for both packages (`ok ... 0.258s`, `ok ... 0.631s`).
- Refactor: Used `context.AfterFunc` so cancellation actively unblocks the request body without a leaked waiter goroutine.

### IR-006: Public-Boundary, Concurrency, And Privacy Coverage
- Requirements/tests: NFR-003 through NFR-005, NFR-008; IT-003, IT-005, IT-006, IT-008, AT-012, REG-006. Scenarios already completed in IR-001 through IR-005 were not duplicated.
- Test additions: IT-005 now races two applications of one valid claim on separate pool connections and proves exactly one commit and one stale rejection. IT-006 now drives a correctly signed request through the real HTTP handler and PostgreSQL store while another connection blocks the account row, forcing the ingress context to interrupt and roll back the durable event/generation transaction before `503`. The same public-boundary test carries raw receipt/payment, auth, signing, billing UUID, and event-ID canaries through body, headers, generic response, logs, metrics, and persistence assertions.
- Initial state: Both new focused scenarios passed on their first run after the IR-001/IR-005 behavior corrections; this was missing coverage, so no artificial production defect was introduced. The combined privacy regression run then exposed an outdated isolation fixture lacking authoritative lifecycle rows after IR-002, not a production failure.
- Green: Uncached race runs passed for concurrent IT-005 (`ok social.craftsky/appview/internal/subscriptions 1.532s`) and real handler/store IT-006 (`ok social.craftsky/appview/internal/integrations/revenuecat 1.558s`). After updating the isolation fixture to production lifecycle semantics and accepted-deletion timing, its focused race test passed (`ok ... 1.538s`). Existing config formatting, provider errors, owner/non-owner responses, self-access minimization, persistence schema inventory, and bounded observation canaries remain part of the combined regression set.
- Refactor: Reused the real store and observer rather than adding test-only persistence or logging seams.

### IR-008: Identifier-Free Billing Observations
- Requirements/tests: NFR-003, NFR-008; AC-021, AC-023, AC-024; AT-011, AT-012, REG-006.
- Red: Extended `TestRevenueCatObservationsAreBoundedAndPrivate` to require assignment outcome, anomaly count, and billing closure observations while passing a private canary as every string input. The focused test failed to compile because all three observer methods were absent.
- Implementation: Added closed assignment operation/outcome and closure-outcome vocabularies, a numeric anomaly gauge, bounded logs, and matching in-memory/Sentry recorder methods. `Store.Assign`/`Unassign` observe classified outcomes without IDs; successful snapshot commits observe only an integer anomaly count; full-service deletion observes `blocked` or `closed` only after the lifecycle transaction returns. Production composition now supplies the observer to both subscription storage and account deletion.
- Green: The observer contract passed (`ok social.craftsky/appview/internal/observability 0.616s`). The uncached race regression across observer, real assignment/snapshot stores, full-service blocked/successful closure, and production composition passed for all four packages (`observability 2.402s`, `subscriptions 2.106s`, `accountdeletion 2.204s`, `app 2.462s`).
- Privacy evidence: Tests assert the observer API receives only closed vocabulary or an integer, closure metrics emit both blocked/closed outcomes, and logs contain no DID, billing UUID, RevenueCat subscription ID, product ID, secret, or raw provider canary.
- Refactor: Used optional narrow recorder and domain observer interfaces; the existing global metric recorder contract and unrelated capabilities were not expanded.

### IR-007: Corrected Traceability And Fresh Verification
- Traceability corrections: Corrected the owner-identity route step to approved test ID IT-010, corrected UT-002's original claim so it no longer says the pre-review fixture covered `pending_changes.product_id`, and marked the earlier supplemental final-verification claims as superseded by `06-implementation-review.md`. The post-review sections above distinguish meaningful red failures from coverage-only tests that were already green.
- Formatting/static analysis: `gofmt -w internal/subscriptions internal/integrations/revenuecat internal/accountdeletion internal/observability internal/app` completed with no output. `go vet ./...` completed with no output.
- PostgreSQL race suites: The first uncached parallel run passed seven affected packages but `internal/app` hit PostgreSQL `out of shared memory` while isolated schemas were created concurrently. The exact affected suite was rerun uncached and serially with `go test -race -p=1 ./internal/subscriptions ./internal/integrations/revenuecat ./internal/routes ./internal/accountdeletion ./internal/observability ./internal/app ./internal/db ./cmd/appview -count=1`; all eight packages passed (`2.251s`, `1.487s`, `4.315s`, `6.187s`, `2.089s`, `14.091s`, `4.106s`, `3.117s`).
- Full tests: `just test` passed every AppView package with the race detector, including `internal/push` and all subscription-related packages.
- Release gate: A fresh `just appview-check` after the final assignment lock-order correction completed successfully with `appview-check: all release gates passed`. The previously reported intermittent push timeout did not recur, so no push-specific rerun or code change was needed. The gate continues to report acknowledged transitive `GO-2026-5932` in `golang.org/x/crypto/openpgp`, with no available fix.
- Diff hygiene: `git diff --check` passed with no output. `06-implementation-review.md` was not edited, and no commit or push was performed.
- Remaining gaps: MAN-001 (Flutter RevenueCat identity behavior), MAN-002 (external store/RevenueCat activation), account recovery, live credentials/catalog resources, and concrete paid capabilities remain intentionally deferred production-activation/product work. No AppView correction from IR-001 through IR-008 remains incomplete.

### IR-009: Completed Same-ID Tier Changes Fail Closed
- Requirements/tests: G-007, FR-003, FR-026, FR-029; AC-009, AC-021; UT-002, IT-005, AT-011.
- Red: Added `TestCompletedSameIDProductChangesRemainFailClosed` through the real RevenueCat API-v2 client and PostgreSQL store. Its two isolated sequences cover assigned Plus -> pending Business -> completed Business on the same provider subscription ID, and assigned Plus -> unsupported product -> Business on that same ID. After correcting an initial cross-fixture global subscription-ID collision, the exact focused command `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:15830/craftsky_dev?sslmode=disable' TEST_DATABASE_REQUIRED=true go test ./internal/integrations/revenuecat -run '^TestCompletedSameIDProductChangesRemainFailClosed$' -count=1` failed meaningfully in both cases because self access became Business with `givesAccess=true` instead of remaining free with the Plus assignment.
- Implementation: Snapshot application now retains each existing license tier while classifying. A supported same-ID snapshot whose mapped tier differs from the persisted license tier is a `cross_tier_conflict`; the provider row preserves the current product, mapped tier, and `gives_access` for owner visibility, while the license keeps its prior tier and assigned DID, becomes unassignable, and carries the conflict anomaly. The existing access mapping/anomaly predicate therefore prevents provider `gives_access` from promoting inherited access. Unsupported intermediate provider state continues to preserve the same license tier and assignment.
- Green: The exact focused command passed (`ok social.craftsky/appview/internal/integrations/revenuecat 0.474s`). The nearby API mapper, pending-change persistence, snapshot fencing/anomaly, pure anomaly acceptance, and local access regressions passed across `internal/integrations/revenuecat` and `internal/subscriptions` (`ok ... 0.416s`, `ok ... 0.749s`).
- Refactor: Reused the existing license candidate/decision policy and persisted license row; no lineage, paid capability, automatic repair, or new externally visible state was added. Existing privacy canaries remain applicable because the bounded owner anomaly code is unchanged and self access still exposes only its five approved fields.

### IR-010: Accepted Deletion And Snapshot Use Compatible Lock Order
- Requirements/tests: FR-017, FR-022, FR-031, NFR-004, NFR-008; AC-018, AC-020, AC-024; IT-005, IT-007, IT-008.
- Red: Added `TestAcceptedSelfAssignedDeletionSerializesWithSnapshotApply` using separate PostgreSQL connections and a trigger/advisory-lock barrier. The barrier pauses accepted self-assignment cleanup after deletion reaches the license, then starts a valid generation-fenced snapshot and waits until it is row-lock blocked before releasing deletion. `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:15830/craftsky_dev?sslmode=disable' TEST_DATABASE_REQUIRED=true go test ./internal/accountdeletion -run '^TestAcceptedSelfAssignedDeletionSerializesWithSnapshotApply$' -count=1` failed with `upsert billing license: ERROR: deadlock detected (SQLSTATE 40P01)`, proving the license-before-account inversion.
- Implementation: `ConfirmDeletion` now locks an owned active billing account before clearing assignments. This matches snapshot application's billing-account-before-provider/license order and remains inside the account-deletion service's already-held owner lifecycle fence. A non-owner accepted deletion still performs target assignment cleanup without taking an unrelated billing-account lock.
- Green: The exact focused command passed (`ok social.craftsky/appview/internal/accountdeletion 0.512s`). Five uncached race-enabled repetitions of the deterministic race plus participant/full-service deletion regressions passed (`ok ... 5.658s`). The assignment, snapshot, accepted-deletion race, and participant lock-order group passed serially with `-race -p=1` across `internal/subscriptions` and `internal/accountdeletion` (`ok ... 1.705s`, `ok ... 1.562s`).
- Refactor: Kept the correction inside `ConfirmDeletion`; no new transaction, lock abstraction, or provider call was added. The race asserts deletion commits, snapshot returns `ErrStaleSnapshot`, the DID-free closed marker remains, provider/license rows cannot resurrect, and no assigned row survives.

### IR-011: Deletion Requires Affirmative Non-Renewal Evidence
- Requirements/tests: G-007, FR-022, FR-031; AC-020; ASM-006; UT-002, IT-008, AT-009.
- Documentation: Rechecked the current RevenueCat API v2 subscription reference through Context7 (`/websites/revenuecat_api-v2`). The documented `auto_renewal_status` values are `will_renew`, `will_not_renew`, `will_change_product`, `will_pause`, `requires_price_increase_consent`, and `has_already_renewed`; omission/nullability is not documented as terminal evidence.
- Red: Added API-v2 boundary coverage in `TestClientPreservesAutoRenewalStatusEvidence` for omitted, empty, unknown/future, resumable `will_pause`, and explicit `will_not_renew` values. Extended the full `AppService` lifecycle test to attempt irreversible acceptance for every uncertain value after a post-intent expired/inaccessible/non-pending snapshot, asserting the active billing account, provider row, self-assignment, and deletion-pending lifecycle remain intact; only exact `will_not_renew` may close. The combined command passed the API boundary test but failed the full service on the omitted value because acceptance returned nil and closed billing: `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:15830/craftsky_dev?sslmode=disable' TEST_DATABASE_REQUIRED=true go test ./internal/integrations/revenuecat ./internal/accountdeletion -run '^(TestClientPreservesAutoRenewalStatusEvidence|TestAppServiceOwnsDeletionCredentialAcrossLifecycle)$' -count=1` (`missing auto-renewal acceptance = <nil>, want provider billing blocker`). Later cascading fixture failures from that unexpected closure were removed before implementation by making the matrix stop on the first failed invariant.
- Implementation: Replaced the nullable/empty allowlist with PostgreSQL `auto_renewal_status IS DISTINCT FROM 'will_not_renew'`. Closure now requires every row to be explicitly `expired`, inaccessible, non-pending, and exactly `will_not_renew`; missing, empty, future, and resumable values block fail-closed while the client continues to preserve provider status verbatim for owner visibility.
- Green: The exact combined focused command passed (`ok social.craftsky/appview/internal/integrations/revenuecat 0.391s`, `ok social.craftsky/appview/internal/accountdeletion 1.251s`). The API mapper, full deletion service, participant, public deletion acceptance, and deterministic deletion/snapshot regression group passed uncached and serially under `-race -p=1` (`ok ... 1.419s`, `ok ... 2.325s`).
- Refactor: Kept the terminal rule as one exhaustive SQL predicate at the destructive transaction boundary. No provider vocabulary is converted into local access logic, no new response/error state was added, and the existing full-service identifier-free closure log/metric canaries cover every additional blocked outcome.

### IR-009 Through IR-011 Final Verification
- Formatting/static analysis: `gofmt -w` on every touched Go file completed with no output; `go vet ./...` completed with no output.
- Required affected packages: The uncached serial command `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:15830/craftsky_dev?sslmode=disable' TEST_DATABASE_REQUIRED=true go test -race -p=1 ./internal/subscriptions ./internal/integrations/revenuecat ./internal/accountdeletion ./internal/routes ./internal/observability ./internal/app ./internal/db ./cmd/appview -count=1` passed all eight packages (`2.478s`, `1.621s`, `6.886s`, `4.446s`, `2.299s`, `14.414s`, `4.284s`, `3.203s`).
- Full tests: `just test` passed every AppView package under the repository's serial race configuration. No intermittent or unrelated failure occurred, so no exact rerun was needed.
- Release gate: `just appview-check` completed with `appview-check: all release gates passed`, including migration, test, build/image, and vulnerability gates. It continues to report the acknowledged transitive `GO-2026-5932` in `golang.org/x/crypto/openpgp`; no fixed version is available.
- Diff hygiene: `git diff --check` passed with no output. `06-implementation-review.md` was not edited. No commit or push was performed.
- Remaining intentional gaps: MAN-001 (Flutter RevenueCat identity behavior), MAN-002 (external store/RevenueCat activation), account recovery, live credentials/catalog resources, and concrete paid capabilities remain deferred production-activation/product work. IR-009 through IR-011 add no capabilities and leave no known AppView implementation gap from the review.

### IR-012: Documented Pending Product Changes Fail Closed End To End
- Requirements/tests: G-007, FR-003, FR-026; AC-006, AC-009, AC-021; UT-002, UT-005, IT-005, AT-006, AT-011.
- Documentation: Context7 (`/websites/revenuecat_api-v2`) confirmed the API v2 customer-subscription list and optional `pending_changes` contract. RevenueCat's current OpenAPI-derived subscription reference and literal response example confirm the changed product is `pending_changes.product.id`; `pending_changes.product_id` is not documented.
- Red: Reworked `TestPendingProductChangeReachesPersistedFailClosedAnomaly` into a real API-v2-client-to-PostgreSQL-store sequence: establish and assign an accessible Plus subscription, then return documented nested pending Business product data together with a distinct accessible Business destination subscription. `TEST_DATABASE_URL="postgres://craftsky:dev@localhost:${$(../scripts/compose-dev port postgres 5432)##*:}/craftsky_dev?sslmode=disable" TEST_DATABASE_REQUIRED=true go test ./internal/integrations/revenuecat -run '^TestPendingProductChangeReachesPersistedFailClosedAnomaly$' -count=1` failed with `PendingProductID` empty while both subscriptions were present (`client snapshot ... want nested pending product ID and destination subscription`).
- Implementation: Replaced the private fixture-only `pending_changes.product_id` decoder with the documented optional nested `pending_changes.product.id` shape, retaining normal `encoding/json` additive-field tolerance and adding no undocumented fallback. Updated stale client fixtures to the documented shape. The expanded end-to-end assertion then exposed that persisted `cross_tier_conflict` rows remained access-granting; the indexed self-access predicate now grants only provider rows with anomaly `none`, rather than excluding only `unsupported`.
- Green: The exact focused command passed (`ok social.craftsky/appview/internal/integrations/revenuecat 0.473s`). The nearby mapper, same-ID transition, snapshot/anomaly, local access, and independent-license regression command passed across `internal/integrations/revenuecat` and `internal/subscriptions` (`ok ... 0.749s`, `ok ... 1.139s`). Two intermediate focused runs exposed test-only nullable scan/format fixture defects and were corrected; after decoding was fixed, the meaningful production assertion also failed once with paid Plus access before the anomaly access predicate was tightened.
- Refactor: None. The production change is limited to the private response shape, its mapper, and the existing indexed authorization predicate; no lineage, compatibility decoder, automatic assignment, or new persisted/provider field was added.
- Notes: The final test proves `PendingProductID` propagation, owner-visible persisted `cross_tier_conflict` on both subscriptions/licenses, both licenses unassignable, the original Plus assignment preserved, the Business destination unassigned, and beneficiary access fail-closed to free. Additive provider fields remain ignored and no provider-private data enters self access.

### IR-013: Documented Relative Pagination Is Complete And Scoped
- Requirements/tests: FR-003, FR-017, FR-018, FR-027; NFR-004; AC-009, AC-010, AC-018; UT-002, IT-005, IT-007, AT-006.
- Documentation: Context7 (`/websites/revenuecat_api-v2`) confirmed forward-only list pagination through `next_page` and `starting_after`. RevenueCat's current API v2 overview and literal customer-subscriptions response show `next_page` as a root-relative `/v2/projects/{project_id}/customers/{customer_id}/subscriptions?starting_after={last_id}` URL. If no next page exists, `next_page` is absent; the endpoint schema also permits null.
- Red: Changed `TestClientMapsCompleteCustomerSubscriptionSnapshot` to use the documented root-relative URL and exact `/v2` customer endpoint. `go test ./internal/integrations/revenuecat -run '^TestClientMapsCompleteCustomerSubscriptionSnapshot$' -count=1` failed with `invalid RevenueCat pagination URL`. Added `TestPaginationFailureDoesNotApplyPartialSnapshot`; its real client/reconciler/PostgreSQL run failed because only page 1 was requested (`provider request count = 1, want 2`). Added a redirect credential-scope test; `go test ./internal/integrations/revenuecat -run '^TestClientDoesNotForwardAuthorizationOnRedirect$' -count=1` failed because the bearer authorization was forwarded to same-origin `/private` and three requests occurred instead of one.
- Implementation: Added a small `ResolveReference`-based next-page resolver that accepts documented relative and existing same-origin absolute links only after resolving them to the configured scheme/host and exact project/customer subscriptions path. It requires one non-empty `starting_after`, rejects credentials, fragments, opaque/malformed URLs, cross-origin and scheme-relative external hosts, different projects/customers/endpoints, and traversal/endpoint escape. Canonical requested URLs are tracked to stop loops, while `MaxPages` still bounds non-repeating chains. `NewClient` clones the supplied HTTP client and disables redirects so authorization is sent only on explicitly validated customer-subscription requests.
- Green: Focused pagination/security tests passed (`ok social.craftsky/appview/internal/integrations/revenuecat 0.617s`), covering a complete two-page snapshot, cross-origin absolute and scheme-relative URLs, project/customer/endpoint escape, traversal, malformed URLs, fragments, loops, maximum pages, oversized second pages, provider-error second pages, generic errors, and redirect credential isolation. The real-store no-partial-commit test passed (`ok ... 0.448s`), proving page 2 is reached but a page-2 failure leaves zero provider rows and does not advance reconciliation. Full nearby `internal/integrations/revenuecat` and `internal/subscriptions` tests passed against PostgreSQL (`ok ... 0.563s`, `ok ... 1.326s`).
- Refactor: None. URL validation remains private to the client and uses the existing page/status/byte fetch bounds and complete-snapshot return contract; no new configuration or provider data is exposed.
- Notes: Any pagination failure returns an empty incomplete result, so `Reconciler` calls `FailReconciliation` without opening snapshot apply. Same-origin is necessary but not sufficient: authorization is restricted to the configured project's exact customer endpoint, and redirects cannot broaden that scope.

### IR-012 And IR-013 Final Verification
- Formatting/static analysis: `gofmt -w` on the four touched Go files completed with no output. `go vet ./...` completed with no output.
- Required affected packages: The final uncached serial command `TEST_DATABASE_URL="postgres://craftsky:dev@localhost:${$(../scripts/compose-dev port postgres 5432)##*:}/craftsky_dev?sslmode=disable" TEST_DATABASE_REQUIRED=true go test -race -p=1 ./internal/integrations/revenuecat ./internal/subscriptions ./internal/accountdeletion ./internal/routes ./internal/observability ./internal/app ./internal/db ./cmd/appview -count=1` passed all eight packages (`1.885s`, `2.334s`, `7.446s`, `4.407s`, `2.170s`, `16.789s`, `4.404s`, `3.123s`).
- Full tests: The final `just test` passed every AppView package under the repository's serial race configuration, including `internal/integrations/revenuecat` (`1.713s`) and `internal/subscriptions` (`2.356s`).
- Release gate: The first `just appview-check` reached static analysis and failed on `S1021` in the modified mapper test because its old two-step test-server declaration was no longer necessary after replacing the absolute pagination fixture. The declaration was simplified without behavior change, and the complete final verification sequence above was rerun. The final `just appview-check` completed with `appview-check: all release gates passed`, including generated-code, static-analysis, migration, test, image/build, and vulnerability gates.
- Vulnerability evidence: The release scan continues to report acknowledged transitive `GO-2026-5932` in `golang.org/x/crypto/openpgp`; no upstream fixed version is available, no other imported vulnerability was reported, and the repository gate passed.
- Diff hygiene: `git diff --check` passed with no output. `06-implementation-review.md` was not edited. No commit or push was performed.
- Remaining intentional gaps: MAN-001 (Flutter RevenueCat identity behavior), MAN-002 (external store/RevenueCat activation), account recovery, live credentials/catalog resources, and concrete paid capabilities remain deferred production-activation/product work. No IR-012 or IR-013 implementation gap remains known.

## 2026-09-10 Manual-Feedback TDD Correction

### Revised Account-Deletion Evidence
- Manual feedback: Replace the earlier terminal-and-inaccessible requirement. Final deletion is permitted only when every persisted provider subscription has `pending_payment=false` and exact `auto_renewal_status='will_not_renew'`, whether it remains active/currently accessible or is already expired/inaccessible. Every other or missing auto-renewal value and every pending payment blocks. AppView does not infer cancellation from timestamps, provider status, access, or webhook type and never calls provider cancellation.
- Red: Updated the full `AppService` lifecycle and participant integration success paths to retain `status='active'` and `gives_access=true` with exact `will_not_renew`, and added `TestSubscriptionDeletionParticipantBlocksMixedProviderBilling` with a safe canceled row plus each of `will_renew`, `will_change_product`, `will_pause`, `requires_price_increase_consent`, `has_already_renewed`, missing, empty, unknown, and pending-payment blockers. The focused command `TEST_DATABASE_URL="postgres://craftsky:dev@localhost:${$(../scripts/compose-dev port postgres 5432)##*:}/craftsky_dev?sslmode=disable" TEST_DATABASE_REQUIRED=true go test ./internal/accountdeletion -run '^(TestSubscriptionDeletionParticipantBlocksClosesAndUnassigns|TestSubscriptionDeletionParticipantBlocksMixedProviderBilling|TestAppServiceOwnsDeletionCredentialAcrossLifecycle)$' -count=1` failed on both active canceled success paths with `provider billing must be resolved`; the mixed blocker matrix remained blocked.
- Green: Removed `gives_access` and `status <> 'expired'` from the destructive-boundary blocker while retaining `pending_payment OR auto_renewal_status IS DISTINCT FROM 'will_not_renew'`. The exact focused command then passed (`ok social.craftsky/appview/internal/accountdeletion 1.334s`). The tests prove accepted closure atomically clears assignments and live provider/license state, closes the DID-free billing marker, and preserves blocking state atomically for every mixed-set case.
- Refactor: Kept the correction in the existing single SQL predicate and transaction. No new lifecycle inference, helper, provider request, cancellation call, response, or error was added. The warning remains `provider_billing_not_canceled` and continues to state that account deletion does not cancel provider billing and may end CraftSky access.
- Race verification: The focused uncached serial race command across `internal/accountdeletion` and `internal/subscriptions`, including `TestAcceptedSelfAssignedDeletionSerializesWithSnapshotApply`, passed (`2.696s`, with the subscription package reporting no matching tests). It continues to prove a delayed snapshot returns `ErrStaleSnapshot`, closed state does not resurrect, and no assignment survives.
- Full verification: `go vet ./...` and `git diff --check` passed with no output. The uncached serial eight-package race suite passed: `internal/integrations/revenuecat` 1.805s, `internal/subscriptions` 2.358s, `internal/accountdeletion` 8.755s, `internal/routes` 4.655s, `internal/observability` 2.315s, `internal/app` 16.636s, `internal/db` 5.034s, and `cmd/appview` 3.209s. `just test` passed every AppView package. `just appview-check` completed with `appview-check: all release gates passed`; it continued to report acknowledged transitive `GO-2026-5932` with no fixed version and no other imported vulnerability.
- Documentation/status: `01-requirements.md` through `04-coding-plan.md` now state the revised exact-evidence rule. `06-implementation-review.md` preserves its historical findings but is marked stale because its approval predates this correction; a follow-up implementation review is required.

## Completion Checklist

- [x] All AppView Must requirements covered by tests or documented external gaps
- [x] All planned AppView Must tests passing
- [x] Relevant regression tests passing
- [x] No unlinked behavior implemented
- [x] Docs updated
- [x] Required implementation-review corrections IR-009 through IR-013 and the 2026-09-10 validated manual-feedback correction implemented
- [ ] Follow-up implementation review completed for the revised account-deletion requirement
