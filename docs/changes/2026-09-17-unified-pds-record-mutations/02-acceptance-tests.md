# Acceptance Test Specification: Tap-Authoritative PDS Data Lifecycle

## 1. Test Strategy
This change is high risk and crosses durable ingestion, PostgreSQL projections, PDS transactions, mutation APIs, scheduled publication, and Flutter state. Verification therefore combines pure unit tests, real-PostgreSQL race-enabled integration tests, scripted PDS protocol tests, API contract tests, Flutter provider/controller tests with injected clocks, and regression coverage for boundaries that remain unchanged.

The acceptance scenarios group related criteria into coherent system behaviors. They are not a substitute for lower-level evidence: each scenario is backed by the unit and integration cases listed below. Real-PostgreSQL tests use `testdb.WithSchema` or `testdb.WithMigratedSchema`; PDS tests use scripted HTTP transports and signed-CAR fixtures; Flutter timing tests use injected clocks and schedulers rather than wall-clock sleeps.

The release gate is `just test` and `just appview-check` for AppView plus the existing `just app-test` recipe for Flutter. Flutter static analysis through `just app-analyze` remains recommended development evidence but is not part of AC-043.

No local PDS process is required. Store interoperability beyond the scripted protocol contract remains a documented residual risk.

## 2. Requirement Coverage Matrix
| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-002, AC-034 | AT-001, IT-001 | Acceptance / Integration | Yes |
| BR-002 | AC-003, AC-016, AC-021 | AT-004, IT-006, IT-007 | Acceptance / Integration | Yes |
| BR-003 | AC-004, AC-035, AC-036 | AT-009, UT-010, UT-011, IT-012 | Acceptance / Unit / Integration | Yes |
| BR-004 | AC-005, AC-006, AC-010, AC-045 | AT-002, IT-003, IT-005 | Acceptance / Integration | Yes |
| FR-001 | AC-001, AC-006 | AT-001, AT-002, UT-001, IT-001 | Acceptance / Unit / Integration | Yes |
| FR-002 | AC-005, AC-006 | AT-002, UT-001, IT-001, IT-005 | Acceptance / Unit / Integration | Yes |
| FR-003 | AC-002, AC-007, AC-008, AC-015 | AT-001, AT-003, UT-002, IT-001 | Acceptance / Unit / Integration | Yes |
| FR-004 | AC-009 | AT-001, UT-002, UT-003, IT-001 | Acceptance / Unit / Integration | Yes |
| FR-005 | AC-001, AC-023, AC-034 | AT-001, AT-011, IT-001, IT-002 | Acceptance / Integration | Yes |
| FR-006 | AC-010, AC-045 | AT-002, IT-005 | Acceptance / Integration | Yes |
| FR-007 | AC-011 | AT-003, IT-002 | Acceptance / Integration | Yes |
| FR-008 | AC-012, AC-041 | AT-003, UT-003, IT-003 | Acceptance / Unit / Integration | Yes |
| FR-009 | AC-013, AC-015 | AT-003, UT-003, IT-002, IT-004 | Acceptance / Unit / Integration | Yes |
| FR-010 | AC-014, AC-050 | AT-003, UT-004, IT-002 | Acceptance / Unit / Integration | Yes |
| FR-011 | AC-015 | AT-003, UT-005, IT-004 | Acceptance / Unit / Integration | Yes |
| FR-012 | AC-003, AC-016 | AT-004, IT-006 | Acceptance / Integration | Yes |
| FR-013 | AC-003, AC-017 | AT-004, UT-006, IT-014 | Acceptance / Unit / Integration | Yes |
| FR-014 | AC-018 | AT-004, UT-006, IT-006 | Acceptance / Unit / Integration | Yes |
| FR-015 | AC-019, AC-046 | AT-004, UT-007, IT-006 | Acceptance / Unit / Integration | Yes |
| FR-016 | AC-020, AC-024 | AT-004, UT-007, UT-009, IT-007 | Acceptance / Unit / Integration | Yes |
| FR-017 | AC-021 | AT-004, IT-007 | Acceptance / Integration | Yes |
| FR-018 | AC-022, AC-035 | AT-004, AT-009, UT-009, IT-007, IT-012 | Acceptance / Unit / Integration | Yes |
| FR-019 | AC-023, AC-024 | AT-004, AT-011, IT-002, IT-007 | Acceptance / Integration | Yes |
| FR-020 | AC-021, AC-025 | AT-005, IT-007, IT-009 | Acceptance / Integration | Yes |
| FR-021 | AC-026 | AT-005, UT-008, IT-009 | Acceptance / Unit / Integration | Yes |
| FR-022 | AC-027 | AT-006, UT-008, IT-008 | Acceptance / Unit / Integration | Yes |
| FR-023 | AC-028 | AT-006, UT-008, IT-008 | Acceptance / Unit / Integration | Yes |
| FR-024 | AC-029 | AT-003, AT-006, IT-008 | Acceptance / Integration | Yes |
| FR-025 | AC-030 | AT-007, REG-008 | Acceptance / Regression | Yes |
| FR-026 | AC-031 | AT-008, IT-010, REG-006 | Acceptance / Integration / Regression | Yes |
| FR-027 | AC-032 | AT-005, IT-009 | Acceptance / Integration | Yes |
| FR-028 | AC-033, AC-042 | AT-007, IT-011 | Acceptance / Integration | Yes |
| FR-029 | AC-002, AC-034 | AT-001, AT-007, IT-011, REG-007 | Acceptance / Integration / Regression | Yes |
| FR-030 | AC-004, AC-035, AC-037, AC-048 | AT-009, AT-010, UT-010, IT-012 | Acceptance / Unit / Integration | Yes |
| FR-031 | AC-035 | AT-009, UT-011, IT-012 | Acceptance / Unit / Integration | Yes |
| FR-032 | AC-036, AC-048 | AT-009, AT-010, UT-010, IT-012 | Acceptance / Unit / Integration | Yes |
| FR-033 | AC-037, AC-048 | AT-010, UT-010, IT-012 | Acceptance / Unit / Integration | Yes |
| FR-034 | AC-038 | AT-011, IT-002, REG-002 | Acceptance / Integration / Regression | Yes |
| FR-035 | AC-039 | AT-007, IT-011, REG-005 | Acceptance / Integration / Regression | Yes |
| FR-036 | AC-040 | AT-012, IT-013 | Acceptance / Integration | Yes |
| FR-037 | AC-010, AC-045 | AT-002, IT-005 | Acceptance / Integration | Yes |
| FR-038 | AC-019, AC-046 | AT-004, UT-007, IT-006, IT-011 | Acceptance / Unit / Integration | Yes |
| FR-039 | AC-047 | AT-010, UT-010, IT-012 | Acceptance / Unit / Integration | Yes |
| FR-040 | AC-048 | AT-010, UT-010, IT-012 | Acceptance / Unit / Integration | Yes |
| FR-041 | AC-049 | AT-006, UT-008, IT-008 | Acceptance / Unit / Integration | Yes |
| FR-042 | AC-050 | AT-003, UT-004, IT-002 | Acceptance / Unit / Integration | Yes |
| FR-043 | AC-051 | AT-004, UT-009, UT-014, IT-007, IT-016 | Acceptance / Unit / Integration | Yes |
| FR-044 | AC-052 | AT-004, UT-007, UT-015, IT-006 | Acceptance / Unit / Integration | Yes |
| FR-045 | AC-053 | AT-004, UT-006, UT-016 | Acceptance / Unit | Yes |
| FR-046 | AC-054 | AT-003, AT-012, UT-003, UT-017, IT-002, IT-013 | Acceptance / Unit / Integration | Yes |
| FR-047 | AC-055 | AT-006, UT-008, IT-008, IT-015 | Acceptance / Unit / Integration | Yes |
| FR-048 | AC-056 | AT-009, UT-011, UT-018, IT-012, IT-016 | Acceptance / Unit / Integration | Yes |
| FR-049 | AC-057 | AT-014, UT-019, IT-017, REG-009, MAN-003 | Acceptance / Unit / Integration / Regression / Manual phase gate | Mixed |
| NFR-001 | AC-041 | AT-003, IT-003 | Acceptance / Integration | Yes |
| NFR-002 | AC-005, AC-015 | AT-002, AT-003, IT-004, IT-005 | Acceptance / Integration | Yes |
| NFR-003 | AC-042, AC-044 | AT-007, UT-012, IT-011 | Acceptance / Unit / Integration | Yes |
| NFR-004 | AC-007, AC-043 | AT-013, IT-001 through IT-017 | Acceptance / Integration | Yes |
| NFR-005 | AC-044 | UT-012, MAN-001 | Unit / Manual operational review | Mixed |
| NFR-006 | AC-005, AC-045 | AT-002, IT-005 | Acceptance / Integration | Yes |
| RULE-001 | AC-001, AC-034, AC-038 | AT-001, AT-011 | Acceptance | Yes |
| RULE-002 | AC-001 | AT-001 | Acceptance | Yes |
| RULE-003 | AC-013, AC-015, AC-029 | AT-003 | Acceptance | Yes |
| RULE-004 | AC-038 | AT-011 | Acceptance | Yes |
| RULE-005 | AC-023, AC-038 | AT-004, AT-011 | Acceptance | Yes |
| RULE-006 | AC-006, AC-015 | AT-002, AT-003 | Acceptance | Yes |
| RULE-007 | AC-017, AC-018, AC-019 | AT-004 | Acceptance | Yes |
| RULE-008 | AC-030, AC-039 | AT-007, REG-005, REG-008 | Acceptance / Regression | Yes |

## 3. Acceptance Scenarios
### AT-001: Federated Source Authority And Validation
Requirement IDs: BR-001, FR-001, FR-003, FR-004, FR-005, FR-019, FR-029, NFR-004, RULE-001, RULE-002, RULE-005
Acceptance Criteria: AC-001, AC-002, AC-007, AC-008, AC-009, AC-023, AC-034
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/ingestion/store_integration_test.go`, proposed `appview/internal/index/source_validation_test.go`

```gherkin
Feature: Federated source authority
  Scenario: Valid records have equal authority regardless of origin
    Given equivalent valid records written through Craftsky and another authorized client
    When Tap installs and projects their latest repository versions
    Then both remain source evidence and contribute equally when locally eligible
    And command correlation does not change projection eligibility

  Scenario: The current invalid version cannot contribute
    Given a URI whose current record changes from valid to invalid and later to valid
    When each version is projected
    Then the latest source evidence remains retained even while invalid
    And the fact is removed while invalid and restored only by the later valid version
    And invalid profile records cannot change membership or terminal owner policy
```

### AT-002: Replay, Revision Ordering, Rebuild, And Verified Repair
Requirement IDs: BR-004, FR-001, FR-002, FR-006, FR-037, NFR-002, NFR-006, RULE-006
Acceptance Criteria: AC-005, AC-006, AC-010, AC-045
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/ingestion/store_integration_test.go`, `appview/internal/ingestion/repository_repair_integration_test.go`

```gherkin
Feature: Source recovery
  Scenario: Delivery order and retries do not alter final state
    Given duplicate, stale, reversed historical, and independently scheduled projection work
    When all work completes
    Then public state equals a clean rebuild from retained current sources
    And equal-revision conflicting content remains uncertain pending verified repair

  Scenario: Verified repair establishes absence
    Given retained sources and empty command tables
    When repair verifies a complete repository snapshot and rebuilds projections
    Then omitted records are removed only from verified absence
    And facts, aggregates, notifications, and serving rows equal repository truth
```

### AT-003: Duplicate-Safe Set Aggregates And Notification Edges
Requirement IDs: FR-003, FR-007, FR-008, FR-009, FR-010, FR-011, FR-024, FR-042, FR-046, NFR-001, NFR-002, RULE-003, RULE-006
Acceptance Criteria: AC-011, AC-012, AC-013, AC-014, AC-015, AC-029, AC-041, AC-050, AC-054
Priority: Must
Level: Acceptance
Automation Target: proposed `appview/internal/index/set_aggregate_integration_test.go`, `appview/internal/index/notification_transaction_test.go`

```gherkin
Feature: Logical set state
  Scenario: Duplicate sources form one logical action
    Given two eligible source URIs for the same follow, block, like, or repost scope
    When one source is retargeted or deleted and then the final source is deleted
    Then every current source URI remains represented
    And every affected old and new aggregate is recomputed atomically
    And activity remains true until the last eligible source leaves
    And raw records, normalized facts, and serving aggregates remain in their defined table boundaries

  Scenario: Aggregate edges control notifications
    Given an aggregate moves through zero, one, two, one, and zero eligible sources
    When projection is retried or interrupted at any transaction boundary
    Then exactly one activation and one final retraction occur
    And representative churn does not reset activity time, newness, or delivery state

  Scenario: A dependency arrives later
    Given a valid interaction whose referenced post is unavailable
    When the dependency later becomes eligible
    Then the source remains durably blocked without counting as active
    And dependency wake-up projects it without replaying the original Tap frame
```

### AT-004: Durable Command Identity, Ambiguity, Replay, And Cleanup
Requirement IDs: BR-002, FR-005, FR-012 through FR-020, FR-038, FR-043, FR-044, FR-045, RULE-005, RULE-007
Acceptance Criteria: AC-003, AC-016 through AC-024, AC-046, AC-051, AC-052, AC-053
Priority: Must
Level: Acceptance
Automation Target: proposed `appview/internal/pdscommands/store_integration_test.go`, `appview/internal/api/public_mutation_command_test.go`

```gherkin
Feature: Durable public mutation commands
  Scenario: A lost create response does not create another record
    Given a command is durably prepared with a scoped UUID, frozen input, selected identity, and lifecycle fences
    And the PDS accepts the write but its response is lost
    When the owner retries the original endpoint with the same key and request
    Then AppView reconciles the frozen effect without selecting another record identity
    And accepted remains terminal independently of later Tap state

  Scenario: Scoped key reuse is immutable
    Given an owner and operation kind already used an operation key
    When the same scoped key is submitted with a different fingerprint or lifecycle generation
    Then AppView returns conflict before another PDS write
    But the same UUID under another owner or operation kind does not collide

  Scenario: Terminal responses compact without reopening identity
    Given accepted and rejected commands, unresolved commands, and expired replay windows
    When cleanup and owner purge run
    Then terminal responses replay for 24 hours before commands, steps, and dispatches atomically become minimal tombstones
    And unresolved exact dispatch plans survive without compaction
    And owner purge removes commands, steps, dispatches, and tombstones

  Scenario: Exact retry and fingerprint contracts remain stable
    Given canonical request and dispatch vectors and a genuinely ambiguous response
    When AppView fingerprints the command and Flutter retries it
    Then the versioned domain-separated hashes match their golden vectors
    And AppView returns exactly 202 with body {"status":"ambiguous"}, integer Retry-After from one through five, and no Location header
    And Flutter waits the greater of Retry-After and local 1, 2, 4, 5, 5, 5 second backoff plus zero through 250 milliseconds jitter
    And automatic retry stops after six retries or 30 seconds without changing the command
```

### AT-005: Addressed, Fixed-Key, And Compound Mutation Correctness
Requirement IDs: FR-020, FR-021, FR-027
Acceptance Criteria: AC-025, AC-026, AC-032
Priority: Must
Level: Acceptance
Automation Target: proposed `appview/internal/pdscommands/record_command_test.go`, `appview/internal/api/profile_command_test.go`

```gherkin
Feature: Guarded record mutations
  Scenario: Addressed retries reconcile exact state
    Given an update or delete names an exact URI and expected CID
    When recovery finds the desired content already present or the deletion already absent
    Then the command completes idempotently
    But a genuine CID or repository-head conflict is rejected

  Scenario: Both profile records change atomically
    Given a personal profile command changes the Bluesky and Craftsky self records
    When one guarded applyWrites transaction succeeds or fails
    Then both records share one commit or neither requested change is applied
    And retries use fixed self keys without allocating a TID
```

### AT-006: Authoritative Set Create And Remove Under Races
Requirement IDs: FR-022, FR-023, FR-024, FR-041, FR-047, RULE-003
Acceptance Criteria: AC-027, AC-028, AC-029, AC-049, AC-055
Priority: Must
Level: Acceptance
Automation Target: proposed `appview/internal/pdscommands/authoritative_set_command_test.go`

```gherkin
Feature: Authoritative set commands
  Scenario: Remove deletes one complete guarded snapshot
    Given several valid records match one logical set scope
    When AppView reads every collection page between identical repository heads
    Then it submits all matching deletions in one transaction guarded by that head
    And it does not infer absence from Tap state or mixed pagination

  Scenario: Repository races trigger bounded rereads
    Given guarded dispatch receives InvalidSwap
    When AppView performs at most three complete read-and-dispatch attempts
    Then each attempt rebuilds steps under the original operation identity
    And full jitter is zero through 25 milliseconds before attempt two and zero through 50 milliseconds before attempt three
    And exhaustion returns 409 pds_repository_conflict without a partial write

  Scenario: Create avoids a known logical duplicate
    Given an unprojected external matching record exists
    When a head-stable complete read precedes set creation
    Then Craftsky returns an idempotent no-op with deterministic representative metadata
    And a later external create after successful removal becomes active normally

  Scenario: Snapshot fallback preserves authoritative read integrity
    Given head-bound pagination is unavailable
    When AppView falls back to a generated signed-CAR snapshot
    Then the snapshot signature, root, MST, repository identity, and source stability are verified

  Scenario: Write fallback preserves the same atomic contract
    Given applyWrites is unsupported for a stable zero-, one-, or multi-write plan
    When AppView evaluates the contract-equivalent write fallback
    Then a zero-write plan completes without mutation
    And a single write carries the same repository head and applicable record CAS
    And multiple writes are never split into sequential operations

  Scenario: Invalid matches are not cleanup targets
    Given valid and invalid records match the same apparent logical scope
    When explicit removal builds its guarded command steps
    Then only semantically valid matching records are selected
    And invalid matching records remain undeleted

  Scenario: Atomic write bounds fail without partial mutation
    Given a command requires 201 writes and the configured cap is 200
    When AppView evaluates the complete guarded plan
    Then it returns 422 pds_atomic_mutation_too_large
    And no PDS write is issued
```

### AT-007: Security, Lifecycle, Telemetry, And Destructive Boundaries
Requirement IDs: BR-001, FR-005, FR-025, FR-028, FR-029, FR-035, NFR-003, RULE-001, RULE-008
Acceptance Criteria: AC-030, AC-033, AC-034, AC-039, AC-042
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/ownerlifecycle/*_integration_test.go`, `appview/internal/accountdeletion/worker_acceptance_test.go`, `appview/internal/observability/*_test.go`

```gherkin
Feature: Mutation security boundaries
  Scenario: Authorization and lifecycle fences survive recovery
    Given an unauthorized, stale-generation, terminal-owner, or unexpected-target command
    When initial dispatch or recovery is attempted
    Then no PDS write occurs
    And no reusable PDS credential reaches Flutter or telemetry
    And valid or invalid external profile records cannot override terminal owner policy

  Scenario: Only explicit authority deletes PDS records
    Given duplicate, invalid, old, or non-representative public records
    When indexers and repair workers run
    Then they issue no PDS deletion
    And permanent deletion continues only through its freshly reauthenticated restricted job
```

### AT-008: Scheduled Final Publication Uses The Common Command Contract
Requirement IDs: FR-026
Acceptance Criteria: AC-031
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/scheduledposts/recovery_acceptance_test.go`, `appview/internal/scheduledposts/worker_acceptance_test.go`

```gherkin
Feature: Scheduled final publication
  Scenario: Recovery preserves one frozen publication
    Given a scheduled publication restarts or loses its PDS response
    When the worker resumes final publication
    Then the frozen publication identity and private schedule state are preserved
    And the common command journal exposes the outcome without duplicate scheduling state
    And no Flutter operation or overlay is created for worker-owned publication
```

### AT-009: Flutter Accepted And Ambiguous Mutation Experience
Requirement IDs: BR-003, FR-018, FR-030, FR-031, FR-032, FR-048
Acceptance Criteria: AC-004, AC-035, AC-036, AC-056
Priority: Must
Level: Acceptance
Automation Target: proposed `app/test/shared/mutations/pds_record_operation_controller_test.dart`, `app/test/shared/mutations/pds_record_operation_overlay_test.dart`

```gherkin
Feature: Flutter mutation experience
  Scenario: Accepted state bridges delayed projection
    Given the PDS definitely accepted a command and ordinary reads remain stale
    When Flutter renders the affected scope
    Then the accepted overlay lifetime starts when Flutter receives acceptance and remains visible without status polling
    And one immediate and one two-second refresh are scheduled
    And agreement retires the overlay
    And 30-second expiry retires it and forces one final ordinary refresh

  Scenario: Each mutation class uses its logical agreement predicate
    Given accepted append, addressed update, addressed delete, set-like, fixed-key, and compound-profile operations
    When ordinary reads contain matching identities/content, logical set state, fixed fields, both profile projections, or disagreement
    Then only the class-specific agreement defined by FR-048 retires the overlay
    And disagreement without a comparable repository revision remains masked until expiry

  Scenario: Ambiguous state freezes one immutable command
    Given the mutation endpoint returns 202 with Retry-After
    When Flutter automatically or explicitly retries
    Then it uses the same endpoint, operation key, and immutable input
    And editing remains frozen until definitive failure permits a new command and key
```

### AT-010: Flutter Account And Scope Fencing
Requirement IDs: FR-030, FR-032, FR-033, FR-039, FR-040
Acceptance Criteria: AC-037, AC-047, AC-048
Priority: Must
Level: Acceptance
Automation Target: proposed `app/test/shared/mutations/pds_record_operation_account_boundary_test.dart`, `app/test/shared/mutations/pds_record_operation_scope_test.dart`

```gherkin
Feature: Flutter operation fencing
  Scenario: Only the newest overlapping operation controls presentation
    Given like then unlike, follow then unfollow, or two profile edits overlap
    When older responses, reads, retries, or timers complete last
    Then only the newest account-local scope sequence controls the UI

  Scenario: Account and process reset restore no mutation state
    Given operations or overlays exist in memory
    When the controller restarts or the user signs out, removes, switches, or permanently deletes the account
    Then no operation or overlay is restored
    And late callbacks cannot affect a later account generation
```

### AT-011: Ordinary Reads Remain Tap-Only
Requirement IDs: FR-005, FR-016, FR-019, FR-034, RULE-001, RULE-004, RULE-005
Acceptance Criteria: AC-023, AC-024, AC-038
Priority: Must
Level: Acceptance
Automation Target: proposed `appview/internal/api/public_read_authority_test.go`

```gherkin
Feature: Public read authority
  Scenario: Commands never become a read overlay
    Given a command is accepted but Tap has not projected it
    When any client calls an ordinary public read endpoint
    Then the response contains only Tap-derived serving state
    And later source observation, projection, or external supersession does not change the terminal command outcome
```

### AT-012: Final Migration Shape
Requirement IDs: FR-036, FR-044, FR-046
Acceptance Criteria: AC-040, AC-052, AC-054
Priority: Must
Level: Acceptance
Automation Target: proposed `appview/internal/db/unified_pds_record_mutations_migration_test.go`

```gherkin
Feature: Final development schema
  Scenario: Fresh migration contains only the final architecture
    Given an empty PostgreSQL 16 schema
    When every production migration is applied
    Then tap_source_records owns raw repository evidence
    And pds_set_sources and pds_set_aggregates contain only their normalized and logical responsibilities
    And pds_commands, pds_command_steps, pds_command_dispatches, and pds_command_tombstones exist with required constraints
    And no relationship-intent compatibility table exists
    And owner_effect_attempts cannot receive new public record commands
```

### AT-013: Release-Gate Mutation-Matrix Coverage
Requirement IDs: FR-003, NFR-004
Acceptance Criteria: AC-007, AC-043
Priority: Must
Level: Acceptance
Automation Target: release scripts plus table-driven conformance tests

```gherkin
Feature: Release evidence
  Scenario: Every public mutation class has contract coverage
    Given the complete mutation matrix
    When just test, just appview-check, and just app-test run
    Then real-PostgreSQL races, source validators, PDS commands, API responses, Flutter reset, account switching, overlap, and overlay timing pass
    And no included mutation class remains marked pending
```

### AT-014: Delivery Order And Legacy Removal
Requirement IDs: FR-049
Acceptance Criteria: AC-057
Priority: Must
Level: Acceptance
Automation Target: MAN-003 implementation phase gate plus proposed automated conformance tests

```gherkin
Feature: Architecture delivery order
  Scenario: Shared foundations precede feature migration
    Given the approved vertical-slice sequence
    When each implementation phase completes
    Then migration, validation, command, authoritative-reader, and rebuild foundations precede like/unlike
    And each later mutation class adopts the shared contracts in the approved order

  Scenario: Final conformance removes superseded machinery
    Given every mutation-matrix row has migrated
    When final static and behavioral conformance runs
    Then no migrated caller uses effect-origin projection gating, relationship intents, winner semantics, superseded feature overlays, or obsolete cache-mutation helpers
```

## 4. Unit Test Cases
| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | BR-004, FR-001, FR-002, NFR-002, NFR-006, RULE-006 | AC-005, AC-006 | Select the current source version by repository revision. | Ordered, reversed, duplicate, stale, delete, and equal-revision-conflict streams. | Latest revision wins; duplicates are inert; conflicting equal revisions become uncertain and request repair. | `appview/internal/ingestion/store_test.go` |
| UT-002 | BR-001, FR-003, FR-004, FR-005, FR-009, FR-011, FR-029, NFR-002, NFR-004, RULE-001, RULE-003, RULE-006 | AC-002, AC-007, AC-008, AC-009, AC-015, AC-034 | Validate every indexed collection and supported action independently of PDS acceptance and local eligibility. | Lexicon shape, rkey, identifier, semantic, dependency, terminal-owner, lifecycle, and unknown-Lexicon fixtures. | Structurally and semantically valid current versions produce facts even when locally ineligible; only eligible facts contribute to aggregates/serving, and only valid profile events may transition non-terminal membership. | Proposed `appview/internal/index/source_validation_test.go` |
| UT-003 | FR-008, FR-009, NFR-001, RULE-003 | AC-012, AC-013, AC-041 | Reduce source replacement into affected logical scopes. | Create, invalidate, retarget, delete, duplicate, and eligibility changes. | Old/new scopes are sorted and recomputed; representative and stable activation time are deterministic. | Proposed `appview/internal/index/set_aggregate_test.go` |
| UT-004 | FR-010, FR-042 | AC-014, AC-050 | Reduce aggregate width changes into notification edges. | `0->1->2->1->0` and representative changes. | Only `0->1` activates and `1->0` retracts; churn preserves newness and delivery. | Proposed `appview/internal/notifications/set_transition_test.go` |
| UT-005 | FR-003, FR-009, FR-011, NFR-002, RULE-003, RULE-006 | AC-015 | Model valid-but-blocked dependency state and wake-up. | Missing, ineligible, later-created, and later-eligible dependencies. | Blocked facts do not count; durable dependency change schedules projection once without Tap replay. | Proposed `appview/internal/index/dependency_test.go` |
| UT-006 | FR-013, FR-014, FR-045, RULE-007 | AC-017, AC-018, AC-053 | Validate scoped UUID identity and canonical fingerprints. | Owners, operation kinds, equivalent JSON, changed fields, blobs, generated timestamps/TIDs, and ordered steps. | Golden request/dispatch hashes use the exact versioned domain separators; changed immutable intent conflicts; owner/kind scopes remain independent. | Proposed `appview/internal/pdscommands/fingerprint_test.go` |
| UT-007 | FR-015, FR-016, FR-019, FR-038, FR-044, RULE-007 | AC-019, AC-020, AC-024, AC-046, AC-052 | Enforce command transitions, terminal immutability, replay retention, and cleanup. | Every state, completion age, reconciliation age, step/dispatch set, and purge state. | Only legal transitions occur; terminal outcomes remain fixed; terminal commands/steps/dispatches compact atomically to a minimal tombstone after 24 hours; unresolved exact plans survive. | Proposed `appview/internal/pdscommands/state_test.go` |
| UT-008 | FR-021, FR-022, FR-023, FR-041, FR-047 | AC-026, AC-027, AC-028, AC-049, AC-055 | Enforce authoritative-reader, retry, atomic-cap, and fallback policy. | Head changes, cursor loops, malformed pages, generated signed-CAR snapshots, invalid duplicates, recorded jitter boundaries, InvalidSwap sequences, and 0/1/2/200/201 writes. | Stable/verified reads pass; unsafe reads fail closed; attempt-two jitter is within 0-25 ms and attempt-three jitter within 0-50 ms; no fourth dispatch occurs; exhaustion is `409 pds_repository_conflict`; 201 writes is `422 pds_atomic_mutation_too_large` with no write; one-write fallback carries both guards; invalid records remain; multi-write fallback is forbidden. | Proposed `appview/internal/pdscommands/authoritative_reader_test.go` |
| UT-009 | FR-016, FR-018, FR-043 | AC-020, AC-022, AC-051 | Map command outcomes to mutation API responses. | Accepted create/update/delete, ambiguous with reconciliation times, rejected, conflict, malformed input, unavailable dispatch. | Normal success, empty `204`, exact `202 {"status":"ambiguous"}`, integer `Retry-After` clamped 1-5, no `Location` header, and standard error envelopes. | Proposed `appview/internal/api/command_response_test.go` |
| UT-010 | BR-003, FR-030, FR-032, FR-033, FR-039, FR-040 | AC-004, AC-036, AC-037, AC-047, AC-048 | Reduce Flutter operation and scope state. | Accepted, ambiguous, failed, changed input, overlapping scopes, session generations, reset events. | Input freezes while ambiguous; same-key retry is retained; newest sequence wins; resets discard state and late results. | Proposed `app/test/shared/mutations/pds_record_operation_controller_test.dart` |
| UT-011 | BR-003, FR-018, FR-030, FR-031, FR-039, FR-048 | AC-035, AC-047, AC-056 | Drive accepted overlay timing and sequence fencing with a fake clock. | Local receipt of definite acceptance; class-specific agreeing/disagreeing/failed reads; two-second and 30-second deadlines; older timers and sequences. | The local receipt instant starts both deadlines; immediate/one grace refresh occurs; only logical agreement retires; disagreement retains; newest sequence wins; expiry retires and forces one refresh without resurrection. | Proposed `app/test/shared/mutations/pds_record_operation_overlay_test.dart` |
| UT-012 | FR-028, NFR-003, NFR-005 | AC-042, AC-044 | Verify telemetry labels and redaction on success and failure paths. | Secret canaries in keys, bodies, OAuth tokens, DPoP material, credentials, and errors. | No secret appears; only bounded approved dimensions are emitted. | `appview/internal/observability/pds_test.go`, `appview/internal/observability/tap_test.go` |
| UT-013 | BR-003, FR-013, FR-020, FR-021, FR-026, FR-027, FR-030, NFR-004, RULE-007 | AC-004, AC-017, AC-025, AC-026, AC-031, AC-032, AC-043 | Check declarative mutation-matrix conformance. | Every included and excluded mutation class. | Included Flutter classes declare key, identity, CAS, steps, and shared-controller strategy; scheduled publication declares AppView journal only; excluded workflows remain outside both contracts. | Proposed `appview/internal/pdscommands/matrix_test.go`, `app/test/shared/api/pds_mutation_contract_test.dart` |
| UT-014 | FR-043 | AC-051 | Drive Flutter ambiguous retry timing deterministically. | Server values clamped to 1-5 seconds, local backoff `1/2/4/5/5/5`, fixed jitter samples at 0 and 250 ms, elapsed-time boundaries, and explicit retry. | Each delay is `max(clamped Retry-After, local backoff)+jitter` with jitter in 0-250 ms; automatic retry makes no more than six retries and stops before exceeding 30 seconds; explicit same-key retry remains available. | Proposed `app/test/shared/mutations/pds_record_operation_retry_test.dart` |
| UT-015 | FR-044 | AC-052 | Verify persistence-boundary ownership and compaction payloads. | Prepared/dispatching/ambiguous/accepted/rejected commands before/at/after 24 hours. | Commands, ordered steps, exact dispatch attempts, and minimal tombstones contain only their defined fields and compact atomically. | Proposed `appview/internal/pdscommands/retention_test.go` |
| UT-016 | FR-045 | AC-053 | Lock exact fingerprint encoding with golden vectors. | Canonical blobs, integers, equivalent object order, server defaults, fixed steps, URI-sorted dynamic steps, and algorithm versions. | Exact request and dispatch digests match committed vectors; excluded inputs do not alter request hashes; version/domain changes do. | Proposed `appview/internal/pdscommands/fingerprint_golden_test.go` |
| UT-017 | FR-046 | AC-054 | Check source/fact/aggregate invariants, sorted scope locks, and representative ordering. | Raw source with versioned validation status, eligible/ineligible facts, duplicates, opposite-order multi-scope retargeting, representative churn. | Raw data and validation status exist only in `tap_source_records`; aggregates equal recomputation; lock acquisition follows sorted scope keys; representative orders by ascending activity then ascending URI; serving selection reads aggregates. | Proposed `appview/internal/index/set_schema_invariants_test.go` |
| UT-018 | FR-048 | AC-056 | Evaluate reconciliation predicates independently for every mutation class. | Flutter receipt timestamps for append create, addressed update/delete, set active/inactive, fixed-key controlled fields, two-part profiles, and CID/content disagreement. | Receipt starts the 30-second lifetime; only the defined logical agreement retires; no-CAS disagreement remains until expiry; both profile projections must agree. | Proposed `app/test/shared/mutations/pds_record_operation_reconciliation_test.dart` |
| UT-019 | FR-049 | AC-057 | Check final legacy-mechanism conformance. | Registered mutation handlers/providers, SQL schema, projection dependencies, and migrated cache helpers. | No migrated registration or caller references relationship intents, effect-origin gating, winner replacement, superseded overlays, or obsolete cache mutation. | Proposed AppView/Flutter architecture conformance tests |

## 5. Integration Test Cases
| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | BR-001, BR-004, FR-001 through FR-005, FR-019, FR-029, NFR-002, NFR-004, NFR-006, RULE-001, RULE-002, RULE-005, RULE-006 | AC-001, AC-002, AC-005 through AC-009, AC-023, AC-034 | Install, validate, invalidate, and restore current source versions with real PostgreSQL. | Isolated schema, complete validator fixtures, Craftsky and external origins. | Ingest versions in ordered and reversed schedules. | Raw source, validation state, fact presence, lifecycle, and eligibility match the latest revision without origin dependence. | `appview/internal/ingestion/store_integration_test.go` |
| IT-002 | FR-005, FR-007 through FR-010, FR-016, FR-019, FR-024, FR-034, FR-042, FR-046, RULE-001, RULE-003 through RULE-005 | AC-011 through AC-014, AC-023, AC-024, AC-029, AC-038, AC-050, AC-054 | Prove the like/unlike reference slice with duplicate external sources and exact table boundaries. | Real PostgreSQL, one post, two like URIs, no command rows, notification recorder, concurrent workers. | Project in reverse order, replay, retarget two scopes in opposite lock order, delete representative, delete final source, and recreate later. | Two normalized facts, one aggregate/read state, one activation, no deadlock or raw duplication, stable activity through representative churn, one final retraction, and later reactivation. | Proposed `appview/internal/index/set_aggregate_integration_test.go` |
| IT-003 | FR-008, NFR-001 | AC-012, AC-041 | Inject failures across source/fact/aggregate/notification/job transitions. | Real PostgreSQL with failure hooks at each boundary. | Interrupt and replay every stage, including retargeting two scopes. | Each transaction fully rolls back or commits; replay converges exactly once. | `appview/internal/index/transactional_pipeline_integration_test.go` |
| IT-004 | FR-003, FR-008, FR-009, FR-011, NFR-001, NFR-002, RULE-003, RULE-006 | AC-015, AC-041 | Reverse cross-repository dependency and worker order. | Interaction before post, durable blocked dependency, concurrent workers. | Make the dependency available while projection retry races wake-up. | Fact becomes eligible once, aggregate counts once, and no Tap replay is needed. | Proposed `appview/internal/ingestion/dependency_integration_test.go` |
| IT-005 | BR-004, FR-001, FR-002, FR-006, FR-037, NFR-002, NFR-006, RULE-006 | AC-005, AC-006, AC-010, AC-045 | Rebuild from retained sources after verified repository repair. | Signed snapshots containing changed, invalid, added, and omitted records; empty commands. | Overlap live ingestion, repair, job reset, and projection rebuild. | Newer live state wins; verified omission deletes; rebuilt output equals clean current-source projection without commands. | `appview/internal/ingestion/repository_repair_integration_test.go` |
| IT-006 | BR-002, FR-012 through FR-016, FR-038, FR-044, FR-045, RULE-007 | AC-003, AC-016 through AC-020, AC-046, AC-052, AC-053 | Exercise concurrent preparation, exact dispatch retention, and command crash boundaries. | Real PostgreSQL, concurrent same-key requests, dispatch failpoints, controllable clock. | Crash before dispatch, while dispatching, after remote acceptance, and before terminal persistence; compact and purge. | One immutable command wins; each exact plan survives until resolution; recovery preserves identity/fences; 24-hour compaction and purge are atomic. | Proposed `appview/internal/pdscommands/store_integration_test.go` |
| IT-007 | BR-002, FR-005, FR-012, FR-013, FR-016 through FR-020, FR-043, RULE-005 | AC-003, AC-020 through AC-025, AC-051 | Reconcile scripted PDS transport ambiguity through the original endpoint. | Scripted accepted/rejected/lost-response/CID-conflict PDS adapter and route harness. | Retry identical requests in every command state. | No blind replay; accepted/rejected replay; genuine ambiguity returns exact `202` and bounded header; addressed matches/absence reconcile; genuine conflicts remain conflicts. | Proposed `appview/internal/api/public_mutation_command_test.go` |
| IT-008 | BR-001, FR-003, FR-022 through FR-025, FR-029, FR-041, FR-047, RULE-003, RULE-008 | AC-002, AC-027 through AC-030, AC-049, AC-055 | Exercise head-bound set create/remove, generated signed-CAR fallback, invalid duplicates, pagination, and repository races. | Scripted heads/raw pages/applyWrites, recorded jitter, and generated valid and invalid signed-CAR snapshots. | Change heads between pages, trigger CAR fallback, return InvalidSwap through exhaustion, include invalid matches, add records between reads/after success, and evaluate 0/1/2/200/201-write plans. | Only verified complete state defines steps; retry delays stay within 0-25 ms then 0-50 ms; no fourth dispatch occurs; exhaustion is `409 pds_repository_conflict`; invalid records remain; later create survives; guarded one-write fallback works; 201 writes is `422 pds_atomic_mutation_too_large`; over-cap and unsupported multi-write issue zero writes. | Proposed `appview/internal/pdscommands/authoritative_set_command_test.go` |
| IT-009 | FR-020, FR-021, FR-027 | AC-025, AC-026, AC-032 | Exercise addressed, fixed-key, and atomic profile commands. | Scripted single-record and `applyWrites` PDS adapter. | Retry already-matching/absent records, stale CIDs, lost compound response, unsupported batch. | Idempotent matches succeed; conflicts fail; both self records share one commit; unsupported multi-write causes no partial mutation. | Proposed `appview/internal/api/profile_command_test.go` |
| IT-010 | FR-026 | AC-031 | Adapt scheduled final publication to common commands. | Existing schedule, frozen publication, leases, media, crash and ambiguity fixtures. | Restart or lose the final PDS response. | One publication identity resumes through the journal without duplicate schedule state or weakened leases/media recovery. | `appview/internal/scheduledposts/recovery_acceptance_test.go` |
| IT-011 | BR-001, FR-005, FR-015, FR-025, FR-028, FR-029, FR-035, FR-038, NFR-003, NFR-005, RULE-001, RULE-008 | AC-030, AC-033, AC-034, AC-039, AC-042, AC-044, AC-046 | Race authorization, owner/target generations, terminal state, recovery, cleanup, and deletion. | Real PostgreSQL lifecycle locks, recording PDS adapter, secret canaries. | Change lifecycle before dispatch/recovery, run indexers/repair, race purge and recovery, run permanent deletion. | Unauthorized effects never dispatch; ordinary workers never delete PDS data; terminal policy wins; purge cannot recreate state; secrets remain bounded. | `appview/internal/ownerlifecycle/*_integration_test.go`, `appview/internal/accountdeletion/worker_acceptance_test.go` |
| IT-012 | BR-003, FR-018, FR-030 through FR-033, FR-039, FR-040, FR-048 | AC-004, AC-035 through AC-037, AC-047, AC-048, AC-056 | Exercise shared Flutter controller with class-specific reads, fake clock, and production account invalidation. | Provider container, fake repositories, injected scheduler, controllable late futures, real `accountStateInvalidatorProvider` wiring. | Accept, return `202`, reconcile each mutation class, overlap operations, expire overlays, restart controller, sign out/switch/remove account. | Logical predicates control retirement; newest scope wins; no ordinary state persists; account invalidator clears state; late work cannot cross generations. | Proposed `app/test/shared/mutations/` and existing account-boundary suites |
| IT-013 | FR-036, FR-044, FR-046 | AC-040, AC-052, AC-054 | Validate migration 73 against exact production migrations. | PostgreSQL 16 isolated schema via `testdb.WithMigratedSchema`. | Apply all up migrations and inspect exact table ownership, versioned validation-status placement, constraints, and forbidden legacy command paths; rely on `appview-check` for down-to-zero/reapply. | Final exact source/fact/aggregate/command schema exists, raw data and validation status remain in `tap_source_records`, relationship intents do not exist, and `owner_effect_attempts` cannot accept public record commands. | Proposed `appview/internal/db/unified_pds_record_mutations_migration_test.go` |
| IT-014 | BR-003, FR-013, FR-016, FR-018, FR-020, FR-021, FR-026, FR-027, FR-030, NFR-004, RULE-007 | AC-004, AC-017, AC-020, AC-022, AC-025, AC-026, AC-031, AC-032, AC-043 | Run table-driven command/API contracts for every mutation-matrix row. | Declarative matrix and per-class fixtures for posts, events, sets, profiles, business profiles, suggestions, and scheduled publication. | Execute accepted, rejected, ambiguous, replay, and relevant CAS cases. | Every included class conforms; scheduled publication is AppView-only; excluded classes are not routed through the command contract; no pending row remains at release. | Proposed AppView API matrix suite and `app/test/shared/api/pds_mutation_contract_test.dart` |
| IT-015 | FR-021, FR-022, FR-023, FR-047 | AC-026, AC-027, AC-028, AC-055 | Verify real Indigo XRPC wire encoding, coordinated-client delegation, and error translation. | Existing `newTestIndigoPDSClient`/`pdsRoundTripFunc` HTTP capture harness plus the production coordinated PDS client wrapper. | Call `getLatestCommit`, paginated raw `listRecords`, ordered `applyWrites`, and single-record fallbacks through both layers; return malformed responses, unsupported errors, and `InvalidSwap`. | Requests carry exact repo, collection, ordered actions, `swapCommit`, and `swapRecord`; the coordinated wrapper preserves lifecycle/owner fencing while delegating every new operation; responses/errors map to the narrow internal contract and fail closed. | Extend `appview/internal/auth/pds_client_indigo_test.go` and `appview/internal/auth/coordinated_pds_client_test.go` |
| IT-016 | BR-003, FR-013, FR-016, FR-018, FR-030 through FR-032, FR-043, FR-048, NFR-004, RULE-007 | AC-017, AC-020, AC-022, AC-035, AC-036, AC-043, AC-051, AC-056 | Verify every real Flutter mutation client and feature provider adopts shared retry/overlay wiring at the HTTP boundary. | Dio capture adapters, fake clock/jitter, production repositories, shared controller, and each migrated feature provider for every keyed Flutter mutation row. | Execute original, ambiguous, automatic retry, explicit retry, accepted update/create, empty delete responses, and provider-visible reconciliation. | Canonical UUID header and immutable body repeat exactly; exact `202` body/header/backoff/jitter/retry bounds are honored; no `Location` or status URL is followed; `204` remains empty; each feature provider delegates operation state and overlay reconciliation to the shared controller. | Proposed `app/test/shared/api/pds_mutation_contract_test.dart` plus feature API-client and provider suites |
| IT-017 | FR-049 | AC-057 | Verify final architecture conformance after all slices migrate. | Registered handlers/providers, migrated SQL schema, code-level architecture inventory, and complete matrix. | Run static registration checks and behavior tests after the last migration slice. | No migrated caller can reach effect-origin gating, relationship intents, winner replacement, superseded overlays, or obsolete cache mutation. | Proposed AppView and Flutter conformance suites |

## 6. Regression Tests
| ID | Existing Behavior Protected | Requirement IDs | Acceptance Criteria | Test |
|---|---|---|---|---|
| REG-001 | Tap frames are durably ingested or quarantined before acknowledgement. | BR-004, FR-001, FR-002, NFR-002, NFR-006, RULE-006 | AC-005, AC-006 | Preserve ingestion receipt, retry, quarantine, and ACK-boundary tests while replacing projection semantics. |
| REG-002 | Ordinary reads retain existing response shapes and remain projection-only. | FR-034, RULE-001, RULE-004, RULE-005 | AC-038 | Existing endpoint contract tests plus command-populated/Tap-stale read tests return no command overlay. |
| REG-003 | Successful DELETE remains empty `204`; standard auth and error envelopes remain intact. | FR-018, FR-020 | AC-022, AC-025 | Route tests assert no DELETE body and unchanged 4xx/5xx envelope casing. |
| REG-004 | Private workflows remain outside the public command contract. | FR-026, FR-035, RULE-008 | AC-031, AC-039 | Mutes, drafts, saves/pins, schedule CRUD/media staging, uploads, customisation, moderation, and push tests remain command-free. |
| REG-005 | Permanent deletion remains freshly reauthenticated and restricted. | FR-035, RULE-008 | AC-039 | Existing account-deletion acceptance suite continues to verify namespace and authority boundaries. |
| REG-006 | Scheduled-post leases, frozen payload/media, private state, and recovery remain intact. | FR-026 | AC-031 | Existing scheduled-post recovery suite passes after only final publication changes journal implementation. |
| REG-007 | Terminal owner and membership policy remain fail-closed. | BR-001, FR-005, FR-028, FR-029, RULE-001 | AC-033, AC-034 | Existing owner lifecycle tests plus valid/invalid external profile cases cannot restore terminal access. |
| REG-008 | Indexers and repair workers never delete PDS records as normalization. | FR-025, RULE-008 | AC-030 | Recording PDS adapter observes zero delete RPCs for duplicate, malformed, old, invalid, or non-representative records. |
| REG-009 | Removed intent, winner, and feature-overlay paths cannot silently regain callers. | FR-049 | AC-057 | Architecture conformance fails when a migrated handler/provider registers or calls any superseded mechanism. |

## 7. Test Data
| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Repository ordering | Ordered, reversed, duplicate, stale, delete, and equal-revision-conflict streams for one and multiple repositories. | UT-001, IT-001, IT-005 |
| TD-002 | Validation coverage | Valid and invalid records for every indexed collection, action, rkey policy, identifier, and semantic dependency. | AT-001, UT-002, IT-001, IT-014 |
| TD-003 | Set-source graph | Follow/block/like/repost duplicates with eligible, blocked, invalid, retargeted, deleted, and later recreated sources. | AT-003, UT-003 through UT-005, IT-002 through IT-004 |
| TD-004 | Fingerprint vectors | Equivalent JSON, changed fields, blobs, arrays, integers, generated timestamps/TIDs, fixed and URI-sorted compound steps. | UT-006, IT-006 |
| TD-005 | PDS outcomes | Accepted, rejected, response lost, ambiguous, InvalidSwap, stale CID, unsupported operation, oversized atomic batch, and malformed XRPC responses. | AT-004 through AT-006, IT-007 through IT-009, IT-014 through IT-016 |
| TD-006 | Collection pages | Stable and changing heads, cursor loops, duplicate/out-of-scope URIs, missing CIDs, endpoint changes, and page/byte/time limit breaches. | UT-008, IT-008 |
| TD-007 | Command lifecycle | Rows in every command and dispatch state, response ages around 24 hours, unresolved ages, tombstones, and owner purge. | UT-007, IT-006, IT-011 |
| TD-008 | Verified snapshots | A shared test helper, extracted from the existing repository-repair snapshot fixtures, programmatically generates signed CARs with added, changed, invalid, omitted, bad-signature, bad-root, bad-MST, and source-change cases. | AT-002, IT-005, IT-008 |
| TD-009 | Flutter event streams | Fake clock/jitter and per-class reads that agree, disagree, fail refresh, complete late, overlap scopes, and cross account generations. | UT-010, UT-011, UT-014, UT-018, IT-012, IT-016 |
| TD-010 | Mutation and migration matrix | Posts, events, follows, blocks, likes, reposts, suggestion acceptance, personal/business profiles, AppView-only scheduled publication, excluded private workflows, and superseded legacy mechanisms. | UT-013, UT-019, IT-014, IT-017, AT-013, AT-014 |
| TD-011 | Secret canaries | Raw operation key, record body markers, OAuth access/refresh tokens, DPoP material, service JWT, and PDS credentials. | AT-007, UT-012, IT-011 |
| TD-012 | Transaction failpoints | Interruptions after source install, validation, fact replacement, each aggregate recompute, notification mutation, and job completion. | IT-003 |

## 8. Manual Checks
| ID | Requirement IDs | Acceptance Criteria | Check | Steps | Expected Result |
|---|---|---|---|---|---|
| MAN-001 | NFR-003, NFR-005 | AC-044 | Operational telemetry remains understandable and bounded. | Run the scripted ambiguity, dependency-blocked, swap-exhaustion, and repair scenarios; inspect local logs and exported metrics. | Operators can distinguish command, source, dependency, projection, aggregate, and convergence stages without secret or unbounded labels. |
| MAN-002 | NFR-004 | AC-043 | Release commands match documented evidence. | Start the isolated dependencies and run `just test`, `just appview-check`, and `just app-test`. Run `just app-analyze` separately as recommended static-analysis evidence. | All acceptance gates pass and produce no unexpected skipped required PostgreSQL or Flutter tests; analysis also passes without being misreported as AC-043. |
| MAN-003 | FR-049 | AC-057 | Delivery phases follow the approved dependency order. | At each phase review, verify migration, validation, command-journal, authoritative-reader, and rebuild foundations completed before like/unlike; verify each later slice follows the sequence in FR-049. | No feature slice begins before its prerequisite phase; final automated IT-017/REG-009 evidence proves all superseded paths have no callers. |

## 9. Test Gaps And Risks
| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | No real local PDS smoke suite is required. | FR-017, FR-021 through FR-023, NFR-004 | Scripted adapters cannot prove quirks of every PDS implementation. | Keep protocol tests exhaustive; add an optional non-gating interoperability suite if production behavior diverges. |
| GAP-002 | Validator behavior may drift from common or newly deployed PDS Lexicons. | FR-003, FR-029, NFR-004 | Craftsky intentionally validates authoritatively in process. | Maintain golden fixtures and monitored rejection reasons; treat drift as the accepted residual risk. |
| GAP-003 | Flutter process death is represented by controller/container disposal. | FR-033, FR-040 | Normal unit/provider tests do not kill the host OS process. | Require disposal/recreation evidence; optionally add a device integration test to confirm no persisted mutation state. |
| GAP-004 | Alert routing and dashboard usefulness are not fully automatable. | NFR-005 | Tests can verify schemas, redaction, and cardinality but not operator workflow quality. | Complete MAN-001 during deployment-readiness review. |
| GAP-005 | Large repository scan latency is workload-dependent. | FR-022, FR-041 | Functional limits prove safety, not production latency. | Add non-gating benchmarks around page, byte, time, and 200-write limits. |
| GAP-006 | Mutation-matrix and legacy-removal conformance cannot be green until every vertical slice migrates. | FR-049, NFR-004 | The architecture is intentionally delivered incrementally. | Keep IT-014 and IT-017 table-driven; temporary pending rows are allowed only during implementation and none may remain at release. |

Blocking test-design gaps: None.

## 10. Out Of Scope
- A required local PDS process or external-network interoperability test.
- Lexicon schema changes or generated Lexicon type changes.
- Device persistence and cross-restart recovery of ordinary Flutter mutation state.
- Direct Flutter-to-PDS ordinary writes or broader credential handoff.
- Tests that move drafts, mutes, saves, schedule CRUD, media staging, moderation, push tokens, or account-deletion state into the public command model.
- Exactly-once Tap transport delivery; the design and tests assume at-least-once delivery.
- Global ordering across repositories or strict callback ordering for historical projection jobs.
- Destructive cleanup of malformed, duplicate, old, or non-representative PDS records.

## 11. Handoff To Document Review
- Requirements file: `01-requirements.md`
- Test specification: `02-acceptance-tests.md`
- Next review artifact: `03-document-review.md`
- External Plannotator review, if initiated by the user: `docs/changes/2026-09-17-unified-pds-record-mutations/`
- Recommended first failing test: the first IT-013 assertion that migration 73 exists and creates the command/source/aggregate ownership baseline; grow IT-013 incrementally to the full schema and legacy-path oracle.
- Suggested test order: IT-013; UT-001, UT-002, UT-006 through UT-009, and UT-014 through UT-017; IT-001, IT-005 through IT-008, and IT-015; IT-002 with UT-003 and UT-004; IT-003 and IT-004; UT-010, UT-011, UT-018, IT-012, and IT-016; remaining set classes; append/addressed records; IT-009; IT-010; IT-011 and UT-012; IT-014, IT-017, UT-019, AT-013, AT-014, regressions, and full gates.
- Commands discovered: `just test`, `just appview-test-shuffle`, `just appview-check`, `just app-test`, and recommended `just app-analyze`.
- Blocking gaps: None.
- Risk level: High. Combined document review is required before coding planning or implementation.
