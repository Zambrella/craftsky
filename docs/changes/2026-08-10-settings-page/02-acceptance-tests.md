# Acceptance Test Specification: Settings Page And Lean Account Deletion

## Development reauthentication correction (2026-08-20)

| Test ID | Requirement | Acceptance criterion | Observable behavior |
|---|---|---|---|
| REG-018 | FR-032 | AC-055 | After intent creation, an ordinary-route 401 cannot remove the exact locally protected recovery lease. |
| REG-019 | FR-032, NFR-001 | AC-055 | The pending job, exact active-account lease, and required handle survive secure-registry reconstruction; another account cannot complete it. |
| REG-020 | FR-016, FR-032 | AC-055 | Cancellation clears pending recovery state without deleting the retained session; successful acceptance clears it while removing only the accepted account. |

AC-055: Reauthentication returns to the exact-handle confirmation after
background ordinary requests and process reconstruction; the account is not
removed until the user submits the exact handle and the server accepts it.

> **AppView audit amendment (2026-08-14):** Section 12 is authoritative for crash-safe PDS/object convergence. The original one-table/no-checkpoint expectations are superseded only for the approved minimized exact-key safety tombstones.

## 1. Test Strategy

Preserve the passing Settings/About/Account and fresh-reauth safety suites, then drive simplification through focused tests that initially expose the old schema and status/convergence contracts. Backend integration tests use disposable Postgres and faked narrow PDS clients. Flutter tests verify immediate account removal and the absence of deletion-status state/routes. Existing Tap/indexer tests remain regression coverage but no longer participate in deletion-job completion.

The product owner explicitly approved the high-risk lean contract and implementation on 2026-08-11.

## 2. Requirement Coverage Matrix

| Requirement IDs | Acceptance Criteria | Test IDs | Level | Automated? |
|---|---|---|---|---|
| BR-001, BR-002, BR-004, FR-001–FR-012, FR-018, NFR-003–NFR-005, RULE-003, RULE-004 | AC-001–AC-014, AC-019–AC-022, AC-024, AC-027–AC-034 | AT-001–AT-003, UT-001–UT-005, UT-020–UT-023, REG-001–REG-007 | Widget / Unit / Regression | Yes |
| BR-003, FR-013, FR-014, FR-019, NFR-001, RULE-001, RULE-006 | AC-012, AC-013, AC-025, AC-035, AC-036, AC-040 | AT-003, UT-024, IT-006 | Widget / Integration | Yes |
| FR-015, FR-020, NFR-002, RULE-002, RULE-005, RULE-007, RULE-009 | AC-015, AC-017, AC-023, AC-030, AC-037, AC-048 | UT-009–UT-011, IT-010, IT-011, IT-027, IT-029, IT-030 | Unit / Integration | Yes |
| FR-015, FR-026, NFR-002 | AC-016, AC-017, AC-037, AC-047 | UT-018, IT-012, IT-013, IT-031 | Integration | Yes |
| FR-016, FR-022, FR-027 | AC-018, AC-036, AC-038, AC-043 | UT-012, UT-014, IT-014, IT-034, REG-014, REG-015 | Unit / Widget / Integration | Yes |
| FR-017, FR-020, NFR-002, NFR-006 | AC-023, AC-039, AC-040, AC-041 | UT-007, IT-009, IT-016, IT-032 | Unit / Integration | Yes |
| FR-021, RULE-010 | AC-016, AC-040–AC-042, AC-046 | UT-011, IT-009, IT-029, IT-032, IT-033 | Integration | Yes |
| FR-023, RULE-007, RULE-008 | AC-044 | IT-030, REG-008, REG-010, REG-013 | Integration / Regression | Yes |
| FR-024, RULE-011 | AC-045 | IT-020, REG-012 | Integration / Regression | Yes |
| FR-025, NFR-006, RULE-010 | AC-046 | IT-029, IT-032 | Migration / Integration | Yes |

## 3. Acceptance Scenarios

### AT-001: Identity-led Settings hierarchy
Requirement IDs: BR-001, BR-002, FR-001–FR-006
Acceptance Criteria: AC-001–AC-006
Priority: Must
Level: Acceptance
Automation Target: `app/test/settings/settings_page_test.dart`, `app/test/router/settings_routes_test.dart`

```gherkin
Scenario: Member opens Settings
  Given a signed-in account with loaded identity
  When Settings opens
  Then identity and Switch account appear first
  And every approved destination is present with the correct affordance
  And Notifications and existing destinations use their canonical routes
```

### AT-002: About page
Requirement IDs: BR-004, FR-007–FR-011
Acceptance Criteria: AC-007–AC-010, AC-019, AC-020
Priority: Must
Level: Acceptance
Automation Target: `app/test/settings/about_page_test.dart`

```gherkin
Scenario: Member opens About
  Then Terms and Privacy use canonical external links
  And Clear image cache preserves existing behavior
  And the shared version/build label is read-only
```

### AT-003: Account page confirmation
Requirement IDs: FR-012–FR-014, FR-019, RULE-001, RULE-006
Acceptance Criteria: AC-011–AC-013, AC-025, AC-035, AC-036
Priority: Must
Level: Acceptance
Automation Target: `app/test/settings/account_page_test.dart`, `app/test/settings/account_deletion_reauth_complete_page_test.dart`

```gherkin
Scenario: Member confirms permanent deletion
  Given fresh OAuth proof bound to the captured account lease
  When the warning names the account and the member types its exact handle
  Then the destructive request becomes enabled
  And the copy explains asynchronous CraftSky-only deletion and eventual AppView cleanup
```

### AT-004: Immediate client handoff after acceptance
Requirement IDs: FR-016, FR-022, FR-027
Acceptance Criteria: AC-018, AC-038, AC-043
Priority: Must
Level: Acceptance
Automation Target: `app/test/settings/account_deletion_controller_test.dart`

```gherkin
Scenario: AppView accepts deletion
  Given the deleting account and an MRU retained account
  When AppView returns 202 Accepted
  Then the client clears the deleting account's local product data
  And removes it from the ordinary account registry
  And activates the MRU account
  And stores no deletion status or progress row
```

### AT-005: Lean durable server lifecycle
Requirement IDs: BR-003, FR-015, FR-017, FR-020, FR-021, FR-023, FR-025, FR-026
Acceptance Criteria: AC-015–AC-017, AC-023, AC-037, AC-041, AC-042, AC-044, AC-046–AC-048
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/accountdeletion/worker_acceptance_test.go`

```gherkin
Scenario: Accepted deletion completes across a restart
  Given Alice has private CraftSky data, Instagram data, scheduled data, CraftSky and non-CraftSky PDS records
  When a freshly bound deletion job runs with an injected interruption
  Then replayed cleanup and repeated CraftSky collection scans converge
  And Alice's private and CraftSky PDS data is gone
  And Bob, other namespaces, the PDS account, and blobs are untouched
  And completion does not wait for Tap/indexer receipts
  And no operation, bound OAuth session, audit, status credential, receipt, expected-record, or checkpoint state remains
```

### AT-006: Pending login remains non-ordinary
Requirement IDs: FR-024, RULE-011
Acceptance Criteria: AC-045
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/auth/account_deletion_reauth_test.go`

```gherkin
Scenario: Same DID signs in while deletion is active
  Given an active deletion operation
  When OAuth login completes
  Then no ordinary CraftSky bearer is minted
  And the response is a coarse deletion-in-progress outcome
  And a later login after operation removal may create a fresh membership
```

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Expected Result | Automation Target |
|---|---|---|---|---|---|
| UT-001–UT-005 | FR-001, FR-003, FR-004, FR-008–FR-014 | AC-001, AC-003, AC-004, AC-008, AC-010–AC-013 | Existing identity, row, version, confirmation, and copy policies remain. | Existing focused tests pass with lean copy. | `app/test/settings/` |
| UT-007 | FR-017, NFR-002 | AC-023, AC-039 | Retry policy automatically schedules every transient failure with a capped maximum delay and exposes no manual/attention decision. | Retry never requires client action. | `appview/internal/accountdeletion/retry_test.go` |
| UT-009 | RULE-005 | AC-030 | Compiled record registry exactly matches primary CraftSky record Lexicons and keeps membership last. | Drift fails. | `appview/internal/accountdeletion/collections_test.go` |
| UT-010 | FR-015, RULE-002, RULE-009 | AC-015, AC-017, AC-048 | PDS deleter repeatedly scans from the beginning, treats not-found as success, and uses only owner/registered collection list/delete calls. | No registrar/marker dependency exists. | `appview/internal/accountdeletion/pds_deleter_test.go` |
| UT-011 | FR-021, NFR-006 | AC-040–AC-042 | Production Store/lifecycle exposes bound OAuth only for the matching worker job and owner; no client projection exists. | Cross-job/cross-owner reads fail and only the bound server session ID is internally available. | `appview/internal/accountdeletion/store_test.go`, `appview/internal/accountdeletion/worker_acceptance_test.go` |
| UT-012 | FR-016 | AC-018, AC-038 | Accepted deletion removes the account and selects MRU/Sign in without creating a status entry. | Registry contains ordinary retained accounts only. | `app/test/settings/account_deletion_controller_test.dart`, `app/test/auth/models/account_switcher_state_test.dart` |
| UT-014 | FR-022 | AC-043 | The production local cleaner attempts drafts/staged media, Instagram verification state, and both caches; the coordinator removes the session even when cleanup reports an error. | Every cleanup step is attempted and no status registry write is required. | `app/test/settings/account_deletion_local_cleanup_test.dart`, `app/test/settings/account_deletion_controller_test.dart` |
| UT-018 | FR-026 | AC-047 | Instagram explicit deletion invokes the terminal owner purge, whose integration test covers private categories and username claims. | All account-owned Instagram categories purge. | `appview/internal/accountdeletion/instagram_cleanup_test.go`, `appview/internal/instagram/account_data_test.go` |
| UT-020–UT-023 | FR-008–FR-010, FR-018, NFR-003, NFR-004 | AC-019, AC-022, AC-024, AC-027, AC-028 | Existing action, localization, semantics, and error styling tests remain. | Existing behavior passes. | `app/test/settings/` |
| UT-024 | FR-013, FR-019, FR-020 | AC-035, AC-040, AC-041 | Fresh account-deletion request metadata is persisted atomically and owner/job scoped. | Ordinary login cannot be upgraded after persistence. | `appview/internal/auth/store_test.go`, `appview/internal/auth/account_deletion_reauth_test.go` |

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Expected Result | Automation Target |
|---|---|---|---|---|---|
| IT-006 | FR-013, FR-019, FR-024 | AC-035, AC-040, AC-045 | OAuth purpose branches before ordinary membership/session creation. | Fresh delete proof or pending outcome; never ordinary access. | `appview/internal/auth/account_deletion_reauth_test.go` |
| IT-009 | FR-020, FR-021 | AC-016, AC-041, AC-042 | Acceptance binds fresh OAuth before removing ordinary sessions. | One operation and one bound OAuth session remain. | `appview/internal/accountdeletion/store_test.go` |
| IT-010, IT-011 | FR-015, NFR-002 | AC-015, AC-017, AC-037 | Paged/restarted PDS deletion converges without expected-record persistence. | Final scan empty; retry safe. | `appview/internal/accountdeletion/pds_deleter_test.go` |
| IT-012, IT-013 | FR-015, FR-026 | AC-016, AC-047 | Current private database/scheduled/Instagram cleanup deletes Alice and preserves Bob/shared rows. | Repeated whole cleanup remains idempotent. | `appview/internal/accountdeletion/private_cleanup_test.go`, `appview/internal/scheduledposts/cleanup_test.go`, `appview/internal/instagram/account_data_test.go` |
| IT-016 | FR-017 | AC-023, AC-039 | Production Worker records automatic retries through its Store boundary with capped backoff. | No needs-attention/manual transition. | `appview/internal/accountdeletion/worker_failure_test.go`, `appview/internal/accountdeletion/retry_test.go` |
| IT-020 | FR-024 | AC-045 | Pending login does not initialize membership or bearer. | Coarse pending result only. | `appview/internal/auth/account_deletion_reauth_test.go` |
| IT-027 | FR-015, FR-020, FR-021, FR-023, FR-025 | AC-015–AC-017, AC-037, AC-044, AC-046 | Complete lifecycle spans restart and reordered/no index events. | Completes from PDS/private gates alone and leaves no deletion state. | `appview/internal/accountdeletion/worker_acceptance_test.go` |
| IT-029 | FR-021, FR-025, RULE-010 | AC-042, AC-046 | Migration creates only the operation table plus OAuth request metadata. | Seven superseded tables are absent. | `appview/internal/db/account_deletion_migration_test.go` |
| IT-030 | FR-023, RULE-007, RULE-008 | AC-044 | Lifecycle succeeds with Tap stopped and no receipt/index queries. | Public projection lag does not block terminal cleanup. | `appview/internal/accountdeletion/worker_acceptance_test.go` |
| IT-031 | FR-015, FR-026, NFR-002 | AC-016, AC-017, AC-037, AC-047 | Whole private cleanup is replayed after injected external failure without checkpoints. | Second run safely reaches empty. | `appview/internal/accountdeletion/private_cleanup_test.go` |
| IT-032 | FR-017, FR-025, NFR-006 | AC-039, AC-046 | Terminal finalization deletes job/OAuth and logs only coarse result; persistent retry remains server-owned. | No audit/sweeper/metrics state. | `appview/internal/accountdeletion/store_test.go`, `appview/internal/accountdeletion/worker_failure_test.go` |
| IT-033 | FR-021, FR-027 | AC-038, AC-039, AC-042 | Route policy exposes start/cancel/accept only; status/retry/recovery endpoints and former status authorization are unavailable. | Superseded route behavior is not registered. | `appview/internal/routes/account_deletion_test.go` |
| IT-034 | FR-016, FR-022, FR-027 | AC-018, AC-038, AC-043 | Flutter acceptance response has no status capability/job projection and immediately performs ordinary account removal. | No secure deletion registry mutation. | `app/test/settings/account_deletion_controller_test.dart` |

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Test |
|---|---|---|---|
| REG-001–REG-007 | Settings destinations, Notifications, cache, switcher, version, legal links | FR-001–FR-012, FR-018 | Existing Settings/router/widget suites remain green. |
| REG-008 | Tap/indexer replay semantics | FR-023, RULE-008 | Existing dispatcher/indexer tests pass without account-deletion observer wiring. |
| REG-009 | Whole-account/namespace/blob prohibition | RULE-002, RULE-009 | Narrow fake never receives account, other-namespace, or blob delete. |
| REG-010 | Public deletion remains indexer-owned | FR-023, RULE-008 | No eager AppView purge is introduced. |
| REG-012 | Fresh rejoin after completion | FR-024, RULE-011 | Operation absence permits ordinary onboarding without restoration. |
| REG-013 | Tap acknowledgement is independent of deletion jobs | FR-023 | Consumer has no must-replay deletion sentinel or receipt failure path. |
| REG-014 | Shared switcher contains ordinary accounts only | FR-016, FR-027 | Deletion status row/model branches are absent. |
| REG-015 | Router/app root have no deletion status polling | FR-016, FR-027 | Status routes and refresh host are absent while reauth callback remains. |
| REG-016 | Ordinary 401 invalidation is the only other-device cleanup path | FR-022 | Interceptor uses established account invalidation without deletion recovery. |

## 7. Test Data

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Multi-account client | Alice deleting, Bob MRU retained | AT-004, UT-012, IT-034 |
| TD-002 | PDS boundary | Multiple CraftSky collections/pages, non-CraftSky record, not-found delete | UT-010, IT-010, IT-011, IT-027 |
| TD-003 | Private cleanup | Alice/Bob/shared database, scheduled-media, Instagram rows | IT-012, IT-013, IT-031 |
| TD-004 | Restart | Accepted operation with bound OAuth and injected transient failure | IT-016, IT-027, IT-030 |

## 8. Manual Checks

| ID | Requirement IDs | Check | Steps | Expected Result |
|---|---|---|---|---|
| MAN-001 | FR-001–FR-018, NFR-003–NFR-005 | Responsive Settings/About/Account UI | Exercise compact/large layouts, themes, text scaling, keyboard/focus. | Requested hierarchy and confirmation remain usable. |
| MAN-002 | FR-013, FR-019 | Real OAuth redirect cancellation/success | Use a disposable development account. | Cancellation mutates nothing; success reaches exact-handle confirmation. |
| MAN-003 | FR-015, RULE-002, RULE-009 | Real PDS deletion boundary | Use only a disposable development PDS account containing CraftSky and non-CraftSky records. | CraftSky records disappear; PDS account, other records, and blob API remain untouched. |

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirements | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | No automated real-PDS/firehose end-to-end | FR-015, FR-023 | Requires disposable external infrastructure. | MAN-003 before release. |
| GAP-002 | No user recovery UI for persistent failures | FR-017 | Deliberately removed by approved contract. | Existing monitoring/operator intervention; reconsider before production users. |

## 10. Out Of Scope

- Expected URI registration, index receipts, convergence queries, Tap acknowledgement coupling, or eager public purge.
- Deletion status credentials/storage/routes/pages/polling, switcher status rows, manual Retry, or needs-attention states.
- Cleanup checkpoint/artifact persistence and executable schema-wide manifest enforcement.
- Deletion audit/sweeper and deletion-specific metrics.
- Direct blob or AT/PDS account deletion.

## 11. Handoff To Document Review

- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Next artifact: `03-document-review.md`
- Recommended first failing test: IT-029, requiring the migration to expose only one deletion table.
- Suggested implementation order: IT-029 → UT-010/IT-010 → IT-031 → IT-030/IT-027 → IT-032/IT-033 → IT-034/REG-014–REG-016 → broad regressions.
- Commands: `go test ./internal/db ./internal/accountdeletion ./internal/auth ./internal/routes ./internal/tap ./internal/index`; focused `flutter test test/settings test/auth test/router`; `dart analyze`.
- Blocking gaps: None. High-risk approval was explicitly granted by the product owner.

## 12. AppView Audit Correction Tests: Exact-Key Safety Tombstones

### 12.1 Superseded expectations

- IT-029 no longer requires exactly one deletion table; it must require the operation table plus only the approved narrow safety-tombstone relation among deletion-specific persistence.
- IT-031 continues to prohibit per-component cleanup checkpoints and replay whole private cleanup, but it no longer prohibits exact-URI/object-key safety tombstones for remote outcome uncertainty.
- IT-032/AC-046 continue to require an artifact-free **post-convergence** terminal state; they do not authorize finalization while a known PDS/object call can still commit.
- The deleted status/recovery/audit/receipt/metrics surfaces remain out of scope and their regression tests remain required.

### 12.2 Amendment coverage matrix

| Requirement IDs | Acceptance Criteria | Test IDs | Level | Automated? |
|---|---|---|---|---|
| FR-028, FR-029, NFR-007, RULE-012, RULE-013 | AC-049, AC-050, AC-052, AC-054 | AT-007, UT-025, IT-035, IT-037, REG-017 | Unit / Integration / Acceptance / Regression | Yes |
| FR-028, FR-030, NFR-007, RULE-012, RULE-013 | AC-049, AC-051, AC-052, AC-054 | AT-008, UT-026, IT-036, IT-037, REG-017 | Unit / PostgreSQL + object-store integration / Acceptance / Regression | Yes |
| FR-031, NFR-007 | AC-053 | AT-009, IT-038, REG-017 | Integration / Acceptance / Regression | Yes |

### 12.3 Acceptance scenarios

#### AT-007: Delayed PDS commit cannot survive accepted deletion

Requirement IDs: FR-028, FR-029, RULE-012, RULE-013

Acceptance Criteria: AC-049, AC-050, AC-052, AC-054

Priority: Must

Level: Acceptance

Automation Target: `appview/internal/accountdeletion/worker_acceptance_test.go`

```gherkin
Scenario: A remotely accepted record commits after the first empty scan
  Given an ordinary write to an exact registered CraftSky AT URI has crossed the remote acceptance boundary
  And AppView crashes before recording its outcome
  And the owner accepts permanent CraftSky deletion
  When the deletion worker first scans the collection and observes no record
  And the delayed PDS commit then makes that exact record visible
  Then the operation remains reconciling rather than terminally successful
  And the retained exact-URI tombstone causes the record to be deleted and absence reverified
  And no non-CraftSky, follow, blob, DID/PDS-account, or other-owner delete is attempted
```

#### AT-008: Delayed object creation cannot escape cleanup

Requirement IDs: FR-028, FR-030, RULE-012, RULE-013

Acceptance Criteria: AC-049, AC-051, AC-052, AC-054

Priority: Must

Level: Acceptance

Automation Target: `appview/internal/scheduledposts/account_deletion_race_test.go`, `appview/internal/scheduledposts/objectstore_minio_test.go`

```gherkin
Scenario: An accepted object Put materializes after an early delete
  Given a generation-specific scheduled-object Put crossed the remote acceptance boundary
  And AppView crashed before marking the media ready
  And account deletion recorded the exact object key and upload generation
  When cleanup deletes or observes the key absent before the delayed Put materializes
  And the delayed Put then creates the object
  Then the safety tombstone remains claimable
  And a later reconciliation deletes and verifies absence of that exact generation-specific key
  And deletion cannot report data-free before that convergence is proven
```

#### AT-009: Proven convergence restores the lean artifact-free end state

Requirement IDs: FR-031, NFR-007

Acceptance Criteria: AC-053

Priority: Must

Level: Acceptance

Automation Target: `appview/internal/accountdeletion/worker_acceptance_test.go`, `appview/internal/accountdeletion/migrations_test.go`

```gherkin
Scenario: Finalization removes temporary safety state
  Given private cleanup and the final registered-collection scan are complete
  And every PDS and object safety tombstone has proven exact-key convergence
  When the worker finalizes the deletion operation
  Then the operation, safety tombstones, and deletion-only OAuth authority are removed atomically
  And no audit, receipt, status, recovery, checkpoint, or detailed deletion-metric artifact remains
```

### 12.4 Unit and integration cases

| ID | Requirement IDs | Acceptance Criteria | Description | Expected Result | Automation Target |
|---|---|---|---|---|---|
| UT-025 | FR-028, FR-029, NFR-007, RULE-012 | AC-049, AC-054 | Validate PDS tombstone construction and capability scope for owner/job/exact typed AT URI. | Registered owner URI is accepted; follow, blob, other namespace/owner/job, malformed URI, secret/payload, and over-broad selectors are unrepresentable or rejected. | `appview/internal/accountdeletion/store_test.go`, `appview/internal/accountdeletion/pds_deleter_test.go` |
| UT-026 | FR-028, FR-030, NFR-007, RULE-012 | AC-049, AC-054 | Validate object tombstone construction, generation identity, lease fencing, and settlement decision table. | Only the matching immutable key/generation is claimable; elapsed client time without a tested backend bound cannot finalize it. | `appview/internal/scheduledposts/tombstone_test.go`, `appview/internal/scheduledposts/media_service_test.go` |

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-035 | FR-028, FR-029, RULE-013 | AC-050 | Reproduce accepted write → AppView crash → first empty deletion scan → delayed PDS commit. | Barrier fake accepts exact registered-collection write while withholding visibility/completion; accepted deletion has adopted its tombstone. | Run first scan, release delayed commit, then resume worker. | Job never finalizes early; exact URI is re-read/deleted and absence is reverified without repeating the write. | `appview/internal/accountdeletion/worker_acceptance_test.go`, `appview/internal/accountdeletion/pds_deleter_test.go` |
| IT-036 | FR-028, FR-030, RULE-013 | AC-051 | Reproduce accepted object `Put` → AppView crash → early absent/delete → delayed creation. | PostgreSQL media reservation and MinIO-compatible barrier fake use an immutable generation-specific key. | Accept deletion, run early cleanup, release delayed `Put`, then reconcile again. | Tombstone survives the early absence; delayed object is removed and exact-key absence is verified before finalization. | `appview/internal/scheduledposts/account_deletion_race_test.go`, `appview/internal/scheduledposts/objectstore_minio_test.go` |
| IT-037 | FR-029, FR-030, RULE-013 | AC-052, AC-054 | Exercise both adapters without a configured/tested finite settlement guarantee. | One unresolved PDS URI and one unresolved object key remain outcome-uncertain. | Advance client clocks and exhaust ordinary retry windows. | Rows remain bounded/lease-claimable and operation remains reconciling; no terminal/data-free outcome, membership, or user status route appears. | `appview/internal/accountdeletion/worker_failure_test.go`, `appview/internal/scheduledposts/cleanup_test.go` |
| IT-038 | FR-031, NFR-007 | AC-053 | Finalize after every exact key has proven convergence. | Private cleanup complete, final collection scan empty, tombstones settled, matching deletion-only OAuth generation bound. | Finalize under owner/job fence; inject retry after commit ambiguity. | Operation, tombstones, and OAuth state are absent idempotently; no other deletion-specific state remains. | `appview/internal/accountdeletion/worker_acceptance_test.go`, `appview/internal/accountdeletion/migrations_test.go`, `appview/internal/accountdeletion/store_test.go` |

### 12.5 Regression, fixtures, gaps, and TDD order

| ID | Existing Behavior Protected | Requirement IDs | Test |
|---|---|---|---|
| REG-017 | Lean user-facing/no-audit boundary and narrow deletion authority | FR-031, NFR-007, RULE-012, RULE-013 | Existing removed-route, no-status Flutter, namespace/blob/follow boundary, no-audit/no-detailed-metrics, and final residue checks stay green while temporary exact-key persistence exists. |

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-005 | Delayed PDS commit | Owner/job/generation, deterministic registered CraftSky AT URI, durable pre-transition effect attempt, barrier-controlled remote visibility | AT-007, UT-025, IT-035, IT-037 |
| TD-006 | Delayed object creation | Owner/job/generation, immutable object key/upload generation, upload attempt/deadline, barrier-controlled MinIO-compatible `Put` | AT-008, UT-026, IT-036, IT-037 |

No additional manual-only gap is introduced. The existing disposable real-PDS gate remains MAN-003; the deterministic barriers are mandatory automated release evidence because the race is impractical to validate manually.

Failure-first implementation order:

1. UT-025 and the migration shape: minimized typed tombstones and scope constraints.
2. IT-035: delayed PDS commit barrier.
3. UT-026 and IT-036: generation-specific object identity and delayed `Put` barrier.
4. IT-037: no invented finite settlement.
5. IT-038 and AT-009: post-convergence artifact removal.
6. REG-017 plus existing full Go/Flutter regressions.

Recommended first failing test: IT-035 in `appview/internal/accountdeletion/worker_acceptance_test.go`. It must fail because the current worker can finalize after the initially empty scan and retains no exact-URI safety tombstone.

Focused commands:

- `cd appview && go test ./internal/accountdeletion -run 'Test.*Delayed.*PDS|Test.*SafetyTombstone' -count=1`
- `cd appview && go test ./internal/scheduledposts -run 'Test.*Delayed.*Put|Test.*Tombstone' -count=1`
- `cd appview && go test ./internal/accountdeletion ./internal/scheduledposts -count=1`

Blocking gaps: None. The product owner approved the exact-key safety-tombstone branch on 2026-08-14.
