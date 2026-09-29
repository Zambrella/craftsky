# Go test redundancy and reliability audit

- Date: 17 September 2026
- Repository snapshot: `980aee5f7f2ebf3a1e38ba3cb314f51ba1b510fd`
- Scope: all `*_test.go` files under `appview/`, shared test infrastructure, production code needed to establish test ownership, and AppView test entry points in `justfile`, `scripts/appview-check`, and backend CI
- Audit type: static test-ownership, redundancy, reliability, and integration-boundary review; the original audit did not execute the full suite
- Implementation status: implemented and verified on 17 September 2026

## Executive summary

The AppView suite is broad, security-conscious, and unusually strong at real PostgreSQL behavior, lifecycle fencing, race detection, moderation, OAuth, account deletion, and PDS-boundary testing. Its main problem is not a lack of tests. It is that a large amount of test code and runtime is spent repeatedly reconstructing infrastructure or reasserting behavior already owned by another layer.

The audit found five recurring causes:

1. Focused store tests hand-write partial production schemas, creating both duplication and schema-drift risk.
2. Route suites repeatedly retest generic authentication, device, and body policies already owned by the route catalogue and middleware.
3. Large handler suites duplicate exhaustive validation matrices from request decoders and validators.
4. Concurrency tests sometimes use elapsed time as evidence that a lock was attempted, which can produce false confidence under a slow scheduler.
5. Shared test infrastructure uses unbounded contexts and silent cleanup, so infrastructure failures can hang until Go's package timeout or leave state behind.

The suite contains 560 test files, 135,705 lines, and 2,258 top-level `Test*` functions. It also contains 575 `testdb.WithSchema` calls and 424 inline `CREATE TABLE` statements. The scale itself is justified by the backend's responsibilities, but these duplication signals are material:

| Signal | Count |
|---|---:|
| Test files | 560 |
| Test lines | 135,705 |
| Top-level `Test*` functions | 2,258 |
| Top-level fuzz targets | 2 |
| Top-level benchmarks | 1 |
| `testdb.WithSchema(...)` calls | 575 |
| Inline `CREATE TABLE` statements | 424 |
| `httptest.NewRequest(...)` calls | 444 |
| `httptest.NewRecorder()` calls | 602 |
| `t.Parallel()` calls | 900 |
| `context.Background()` calls in tests | 2,682 |
| `t.Context()` calls in tests | 101 |
| `time.Sleep(...)` calls | 21 |
| `time.After(...)` calls | 95 |
| Inline discard logger constructions | 74 |

A conservative cleanup should remove or consolidate approximately 4,000 to 8,000 lines and 15 to 30 standalone or duplicated test executions without reducing meaningful behavioral coverage. The more important outcome is to make production-schema compatibility explicit, reduce the cost of each database test, and turn concurrency tests into deterministic proofs rather than timing observations.

## Overall assessment

| Area | Assessment | Direction |
|---|---|---|
| Pure unit tests | Broad and generally well targeted | Keep local tables and focused assertions |
| Handler tests | Strong coverage, oversized suites and validation overlap | Keep transport/error translation; move exhaustive rules to validators |
| Route tests | Strong policy enforcement, repeated full-mux checks | Centralize catalogue contracts and sample each access class |
| Store tests | Excellent real-Postgres usage, extensive handwritten DDL | Separate focused fixtures from migrated-schema compatibility tests |
| Concurrency tests | Important race coverage, some timing-based false-positive risk | Use barriers and observable lock contention |
| Integration tests | Strong internal vertical coverage, a few missing deployed boundaries | Add real Tap and selected routed HTTP slices |
| Test infrastructure | Fail-closed release path, unbounded setup and cleanup | Add test-scoped deadlines and visible cleanup failures |
| Generated/source tests | Useful semantic guards mixed with brittle implementation scans | Keep semantic contracts; replace text checks with AST or behavior |
| CI and release gate | Strong race and migration gates, expensive serial repetition | Preserve release evidence; add faster feedback and targeted periodic jobs |

## Existing strengths to retain

The cleanup must preserve these properties:

- `appview-check` requires PostgreSQL and MinIO coverage and rejects accidental required-suite skips.
- The full suite runs under the race detector for release evidence.
- The release gate disables Go test caching with `-count=1`.
- Database-backed tests use isolated schemas rather than shared tables.
- Release-gate services use random Compose projects, ephemeral host ports, and disposable volumes.
- Full migration up, down-to-zero, and reapply behavior is checked.
- Container images, the Go toolchain, and major release dependencies are pinned.
- OAuth, owner lifecycle, account deletion, moderation, scheduled publication, indexer idempotency, and PDS write boundaries have substantial focused coverage.
- The image decoder is exercised under the release runtime and a 512 MiB cgroup memory limit.
- A repository-wide guard prevents tests from contacting live Bluesky services.

## Best-practice baseline

### Test behavior at one primary layer

Exhaustive input validation belongs at the decoder or validator. Handler tests should prove representative decode failure, validation failure, dependency failure, success, response encoding, and absence of forbidden side effects. Route tests should prove registration and policy metadata. Middleware tests should prove the policy behavior itself.

Repeating the complete matrix at every layer increases maintenance cost without increasing confidence proportionally.

### Use production schema deliberately

Focused SQL fixtures are useful when a test owns a narrow query and needs a minimal schema. They must not be mistaken for production compatibility. Critical stores and indexers need a separate suite against the exact fully migrated PostgreSQL 16 schema.

Historical migration tests are a third category: they should construct the exact intended pre-migration state and remain independent of ordinary store fixtures.

### Synchronize on observable events, not elapsed time

A concurrency test should prove that the contender reached the operation and is waiting on the expected fence or lock. A fixed 50 or 100 millisecond window proves only that a goroutine did not finish during that interval; scheduler delay can satisfy the assertion even when locking is broken.

### Bound infrastructure operations

Database setup, MinIO calls, goroutine completion, HTTP calls, and cleanup should use test-scoped contexts or explicit deadlines. Failures should report quickly and cleanup errors should be visible. Waiting for Go's default package timeout produces poor diagnostics and wastes CI capacity.

### Share infrastructure, keep scenarios local

Stable helpers such as migrated-schema setup, HTTP request construction, JSON decoding, error-envelope assertions, and discard logging should be shared. Scenario-specific fakes, state machines, and recording callbacks should remain near the tests that use them.

## Findings

### GT-001 - Handwritten schemas are pervasive and can drift from production

**Priority:** High

The suite has 575 `testdb.WithSchema(...)` calls and 424 inline `CREATE TABLE` statements. Core tables are repeatedly redefined with different reduced schemas. Representative examples include:

- [federated_real_flow_integration_test.go](../appview/internal/app/federated_real_flow_integration_test.go#L58-L145)
- [owner_lifecycle_migration_test.go](../appview/internal/db/owner_lifecycle_migration_test.go#L15-L75)
- [scheduledposts/store_test.go](../appview/internal/scheduledposts/store_test.go#L21-L53)
- [scheduled_post_test.go](../appview/internal/api/scheduled_post_test.go#L279-L309)
- [business_routes_test.go](../appview/internal/routes/business_routes_test.go#L107-L119)
- [onboarding_route_test.go](../appview/internal/routes/onboarding_route_test.go#L40-L63)

The shared helper also installs all-active lifecycle predicates when a partial fixture omits them, as shown in [testdb.go](../appview/internal/testdb/testdb.go#L74-L116). That is appropriate for some focused pre-lifecycle tests, but it can conceal production lifecycle semantics.

The release gate proves that migrations execute and reverse. It does not prove that every store query and indexer still works against the final schema, including constraints, defaults, triggers, and lifecycle predicates.

**Recommendation:** define three explicit fixture classes:

- Focused schema fixtures for narrow SQL contracts
- Exact fully migrated PostgreSQL 16 schemas for compatibility tests
- Historical pre-state fixtures owned by migration tests

Add `testdb.WithMigratedSchema(...)`, centralized named fixture builders, and a small compatibility suite that exercises critical stores and indexers against the complete migrated schema.

### GT-002 - Database bootstrap work is repeated hundreds of times

**Priority:** High

Each `WithSchema` call opens a bootstrap pool, serializes `pg_trgm` setup through a database-global advisory lock, creates a schema, opens a scoped pool, inspects two lifecycle functions, and later drops the schema. See [testdb.go](../appview/internal/testdb/testdb.go#L30-L117).

That process runs at 575 call sites. Some files create the same substantial fixture separately for many sibling tests. For example, the moderation transition integration suite repeatedly calls `WithSchema` with the same DDL.

Package serialization with `-p 1` protects PostgreSQL's lock table, but it does not prevent parallel tests inside a package from opening many pools and schemas. Thirty-six files combine `WithSchema` and `t.Parallel()`.

**Recommendation:**

- Initialize database-global extensions once per test database or process.
- Set deliberate low pool limits for test pools.
- Reuse one schema per parent test with transaction or fixture reset between subtests when lock/concurrency behavior is not being tested.
- Keep one-schema-per-test isolation where transaction boundaries, DDL, or concurrent sessions are the behavior.
- Measure suite duration and peak PostgreSQL connections before and after consolidation.

### GT-003 - Route suites repeatedly retest generic middleware behavior

**Priority:** High

[routes_test.go](../appview/internal/routes/routes_test.go#L529-L727) repeatedly performs two checks for each route family:

- Inspect `V1RoutePolicies` for access, rate, and body metadata.
- Build the full mux and reassert generic authentication, device-header, body, or envelope behavior for every route.

Similar checks recur in feature route suites such as:

- [onboarding_route_test.go](../appview/internal/routes/onboarding_route_test.go#L70-L104)
- [business_routes_test.go](../appview/internal/routes/business_routes_test.go#L52-L168)
- [language_preferences_test.go](../appview/internal/routes/language_preferences_test.go#L86-L102)

The middleware itself already has focused suites for authentication, device IDs, CORS, admission, and request bodies.

**Recommendation:** keep one catalogue-driven test proving every registered route's policy metadata, focused middleware suites proving each policy, and one or two full-mux smoke tests per access/body class. Feature route tests should own handler dispatch, path extraction, and feature-specific behavior.

### GT-004 - Request helpers silently repair invalid fixtures

**Priority:** High

The `authedReq` helper in [post_test.go](../appview/internal/api/post_test.go#L424-L436) inserts `"sponsored":false` into POST bodies that omit it. The scheduled-post suite contains a similar helper, and the direct request-validation suite performs comparable body rewriting.

This hides the actual wire payload from the test reader and can turn an intentionally incomplete request into a valid one. It weakens future required-field tests because helper behavior, not the fixture text, determines what reaches production code.

**Recommendation:** stop modifying arbitrary JSON strings. Require complete payloads explicitly or construct typed request fixtures and marshal them. Tests for missing fields must bypass all defaulting builders.

### GT-005 - Concurrency assertions sometimes prove only scheduler delay

**Priority:** High

Several lock/fence tests launch a contender and treat failure to finish within 100 milliseconds as proof that it blocked. One representative case appears in [ownerlifecycle/store_integration_test.go](../appview/internal/ownerlifecycle/store_integration_test.go#L232-L249).

Similar patterns occur in owner lifecycle, scheduled publication, account deletion, and ingestion tests. A slow scheduler can satisfy these assertions before the contender reaches the database, allowing a broken lock to pass.

Other suites already demonstrate the stronger approach by observing PostgreSQL lock waiters before releasing the holder.

**Recommendation:** introduce deterministic barriers:

- Signal immediately before the contender enters the operation.
- Where possible, query `pg_locks` or a fixture-side signal to prove the contender is waiting.
- Release the holder only after contention is observed.
- Bound all completion receives and report goroutine state on timeout.

Do not replace a fixed sleep with a longer fixed sleep.

### GT-006 - One delayed-write assertion is ineffective

**Priority:** High

In [federated_real_flow_integration_test.go](../appview/internal/app/federated_real_flow_integration_test.go#L1984-L1994), the test reads owner and session counts, sleeps for 50 milliseconds, and then asserts the values read before sleeping. It never re-queries after the delay.

The sleep therefore cannot detect a late write during the interval, despite appearing intended to do so.

**Recommendation:** replace the test with a bounded eventually-never contract tied to the worker or request lifecycle. At minimum, re-query after synchronization. Prefer waiting for the relevant asynchronous operation to report completion before asserting final durable state.

### GT-007 - Unbounded waits and infrastructure contexts delay failures

**Priority:** High

The suite has 2,682 `context.Background()` calls but only 101 `t.Context()` calls. It also launches many goroutines and has channel receives that are not guarded by a timeout or test cancellation.

Shared database setup and cleanup use `context.Background()` throughout [testdb.go](../appview/internal/testdb/testdb.go#L40-L70). The schema cleanup discards `DROP SCHEMA` errors at lines 53-55. MinIO integration tests similarly use unbounded contexts and package-level HTTP calls.

Consequences include:

- Deadlocks can consume Go's default package timeout.
- A stalled database or MinIO call can block cleanup.
- Failed schema or object cleanup can leak state into the persistent developer stack.
- Failure diagnostics point to package timeout rather than the causal operation.

**Recommendation:** derive setup and operation contexts from `t.Context()`, add bounded cleanup contexts, make cleanup failures visible without masking the original test failure, and guard channel receives with named timeout helpers. Add an explicit `go test -timeout` value to local and release commands.

### GT-008 - Very large files combine unrelated test layers

**Priority:** Medium

The largest suites are:

| File | Lines |
|---|---:|
| `internal/api/post_test.go` | 4,217 |
| `internal/app/federated_real_flow_integration_test.go` | 3,963 |
| `internal/routes/routes_test.go` | 2,313 |
| `internal/api/post_store_test.go` | 1,977 |
| `internal/index/craftsky_post_test.go` | 1,921 |
| `internal/push/dispatcher_test.go` | 1,799 |
| `internal/app/config_test.go` | 1,611 |
| `internal/auth/session_coordinator_test.go` | 1,575 |

For example, `post_test.go` contains a broad fake store/effect framework, request builders, interaction handlers, post creation, comments, response shaping, validation mapping, and PDS error behavior.

**Recommendation:** split by owned concern, not arbitrary line count. Package-local `_test.go` fixture files can hold reusable fakes and builders without exporting test infrastructure. Suggested `post` divisions are reads, writes, interactions, comments, validation translation, PDS errors, and test support.

### GT-009 - Handler suites duplicate exhaustive validator matrices

**Priority:** Medium

Post creation has overlapping malformed JSON, project validation, image-count, and required-field cases in both [post_test.go](../appview/internal/api/post_test.go) and [post_request_test.go](../appview/internal/api/post_request_test.go).

Handler coverage remains valuable when it proves HTTP status/error translation and that no PDS write occurs. It does not need to repeat every branch owned by the decoder and validator.

**Recommendation:** keep exhaustive tables in request/validator tests. Keep a compact handler matrix covering one representative decode error, validation error, dependency error, PDS error, and success path, including important side-effect assertions.

### GT-010 - Business-neutrality contracts execute twice

**Priority:** Medium

[business_non_entitlement_acceptance_test.go](../appview/internal/api/business_non_entitlement_acceptance_test.go#L5-L9) and [business_neutrality_regression_test.go](../appview/internal/api/business_neutrality_regression_test.go#L5-L12) call the same feed, search, and authorization assertion functions under different wrapper names.

This is exact duplicate execution rather than distinct boundary coverage.

**Recommendation:** retain one clearly named owner suite. If acceptance identifiers must remain discoverable, record them in test names or documentation without running the same function twice.

### GT-011 - HTTP and assertion setup is repeatedly reimplemented

**Priority:** Medium

There are 444 `httptest.NewRequest(...)` and 602 `httptest.NewRecorder()` calls. Multiple packages locally reconstruct the same authenticated context, device headers, path values, content type, JSON decoding, and error-envelope checks.

The suite also contains many response-body substring checks and ignored JSON decode errors, which can turn malformed responses into misleading downstream failures.

**Recommendation:** add small package-appropriate helpers for:

- Direct-handler requests with DID, owner generation, and path values
- Full-route requests with authorization and device headers
- Generic `decodeJSON[T]`
- Exact `{error, message, requestId}` envelope assertions
- Semantic JSON equality where field order is irrelevant

Keep direct-handler and full-route authentication helpers separate so tests do not accidentally bypass the boundary they intend to exercise.

### GT-012 - Migration loading and compatibility rewriting are open-coded

**Priority:** Medium

Many tests resolve and read migration files directly. Three full-migration behavioral helpers also rewrite production SQL to support older local PostgreSQL versions, replacing PostgreSQL 15/16 features and appending substitute indexes.

The release gate separately verifies exact migrations on PostgreSQL 16, but behavioral tests using rewritten SQL do not exercise exact production semantics.

**Recommendation:** centralize migration discovery and application in `testdb`. Production-schema compatibility tests should require PostgreSQL 16 and apply exact files. Historical or compatibility transformations must be explicitly named and must not serve as production-schema evidence.

### GT-013 - Source-text architecture tests are brittle

**Priority:** Medium

Several tests inspect raw implementation text, literal counts, exact call strings, or hard-coded file inventories. Examples include:

- [routes/architecture_test.go](../appview/internal/routes/architecture_test.go#L96-L161)
- [follower_growth_architecture_test.go](../appview/internal/routes/follower_growth_architecture_test.go#L15-L68)
- [followergrowth/query_plan_test.go](../appview/internal/followergrowth/query_plan_test.go#L13-L53)
- [release_gate_inventory_test.go](../appview/internal/app/release_gate_inventory_test.go#L19-L98)

These tests can fail after safe refactors or pass when matching text remains but behavior is wrong.

**Recommendation:** use AST/type analysis for structural constraints, behavioral tests for wiring, and dedicated scripts for repository inventories. Preserve source scans where no typed alternative exists, but make discovery recursive and diagnostics semantic.

### GT-014 - Generated-type tests mix semantic contracts with generator retesting

**Priority:** Medium

Lexicon tests currently combine valuable source-schema invariants with JSON and CBOR behavior of generated types. A business contract also pins a source file using an opaque SHA-256 digest.

Schema semantics are load-bearing and should remain covered. Broadly retesting generated serialization or opaque file hashes adds maintenance without clearly stating the protected compatibility rule.

**Recommendation:** retain semantic schema assertions, one representative JSON/CBOR round trip per generator path, and the existing generated-output drift check. Replace raw hashes with named semantic assertions or explain the exact invariant the digest protects.

### GT-015 - The release gate does not exercise the real Tap protocol boundary

**Priority:** High

`appview-check` starts PostgreSQL and MinIO for tests and later smoke-tests the AppView image, but it does not start the pinned Tap image and pass an event through the real WebSocket/ACK protocol. Tap consumer tests use a hand-written fake WebSocket server.

The fake provides excellent deterministic protocol coverage, but it cannot catch protocol or image-configuration drift between AppView and the exact Tap artifact deployed in production.

**Recommendation:** add one release-gate acceptance journey with the pinned Tap image. Verify AppView connects and processes at least one record event and one identity event through the real channel and ACK contract. Do not duplicate every fake-server case at this layer.

### GT-016 - Federated caption fetching lacks direct boundary tests

**Priority:** High

[caption_fetcher.go](../appview/internal/video/caption_fetcher.go#L29-L78) performs identity resolution, PDS-origin validation, redirect rejection, query construction, status validation, MIME parsing, bounded reads, and cancellation propagation. Production wires it into video dependencies, but handler tests substitute a fake fetcher.

This is a security-sensitive federated network boundary with no direct owner suite.

**Recommendation:** add direct tests for validated URL construction, encoded DID/CID query parameters, redirect rejection, non-200 responses, MIME parameters, missing or wrong content type, declared and chunked oversized bodies, read errors, cancellation propagation, and non-mutation of the supplied HTTP client.

### GT-017 - No real routed HTTP vertical slice covers production middleware and storage together

**Priority:** Medium

`NewServer` has strong middleware and route-contract tests with stub dependencies. Handler and PostgreSQL stores are heavily tested separately. The image release smoke reaches health endpoints only.

**Recommendation:** add a small set of routed vertical slices through `NewServer` with production middleware and real PostgreSQL. Good candidates are one authenticated timeline read and one AppView-mediated write against a fake PDS HTTP server. Keep the set small and do not duplicate the handler matrix.

### GT-018 - Dispatcher registration does not enforce one indexer per NSID

**Priority:** Medium

[transactional_dispatcher.go](../appview/internal/index/transactional_dispatcher.go#L50-L68) rejects nil indexers but silently overwrites an existing handler for the same collection. Production registration is centralized, but no test protects the architectural rule that one indexer owns each NSID.

**Recommendation:** make duplicate registration fail immediately and test nil registration, duplicate registration, deterministic collection ordering, and parity between registered collections and Tap collection filters.

### GT-019 - Local and CI execution paths leave useful diagnostics unused

**Priority:** Medium

The release gate runs the entire suite twice, serially: normal and race, both uncached. This is strong release evidence but provides slow feedback. There is no integration build-tag or `testing.Short()` split, no coverage profile, no shuffled-order run, and fuzz targets are only executed as seed tests.

The local `just test` path also omits `-count=1`, so Go may reuse cached integration-package results even when PostgreSQL or MinIO state has changed.

**Recommendation:**

- Add `-count=1` and an explicit timeout to `just test`.
- Preserve the full release-equivalent `appview-check` gate.
- Add a fast unit-only job for early feedback.
- Add periodic `-shuffle=on` and scheduled fuzz jobs.
- Generate coverage as trend evidence before considering a threshold.
- Evaluate package sharding only after database fixture pressure is reduced.

### GT-020 - Test database selection can mutate a non-test database

**Priority:** Medium

[testdb.go](../appview/internal/testdb/testdb.go#L144-L163) falls back from `TEST_DATABASE_URL` to `DATABASE_URL`. Tests isolate ordinary tables in schemas, but `WithSchema` creates `pg_trgm` in the database-global `public` schema.

An accidental direct `go test` can therefore mutate whichever database `DATABASE_URL` names and may require extension-creation privileges.

**Recommendation:** require `TEST_DATABASE_URL` for database tests or add an explicit opt-in allowing the fallback only in known development workflows. Validate the target database name or a test-only marker before creating extensions or schemas.

## Intentional repetition to retain

The following similar-looking coverage protects distinct risks and should not be consolidated away:

- Real PostgreSQL tests for transaction, lock, query-plan, and constraint behavior
- Normal and race-detector release runs
- Migration up, down-to-zero, and reapply coverage
- Owner-generation, terminal-lifecycle, and account-isolation variants
- Idempotency checks for firehose create, update, delete, replay, and duplicate delivery
- PDS boundary tests proving forbidden deletion or credential behavior
- Moderation state transitions, appeals, expiry, and presentation as separate policy owners
- OAuth discovery, DPoP, quarantine, refresh, and callback atomicity cases
- PostgreSQL query-plan tests where the index is part of an explicit performance contract
- One full-mux example for every materially different access or body-policy class
- One generated JSON/CBOR round trip for each generator path
- Scenario-specific recording fakes and barriers whose behavior is local to one test

## Improvement outline

### Phase 1 - Fix false-confidence and hang risks

1. Fix the ineffective delayed-write assertion.
2. Replace fixed lock windows with deterministic contention barriers.
3. Bound goroutine completion, database operations, MinIO calls, and cleanup.
4. Report schema and object cleanup failures.
5. Add explicit test timeouts and `-count=1` to local integration execution.

Expected result: failures are causal and fast, and concurrency tests prove the intended lock rather than scheduler timing.

### Phase 2 - Establish explicit database fixture layers

1. Add exact migrated-schema support requiring PostgreSQL 16.
2. Add named focused fixture builders for common tables and lifecycle predicates.
3. Centralize migration loading.
4. Initialize global extensions once and cap test pools.
5. Add critical store/indexer compatibility tests against the final schema.

Expected result: focused tests stay fast while production-schema compatibility has a clear owner.

### Phase 3 - Consolidate repeated contracts and setup

1. Collapse route policy checks into a catalogue-driven contract.
2. Remove duplicate business-neutrality executions.
3. Add request, response, JSON, envelope, config, and discard-logger helpers.
4. Stop request helpers from mutating JSON.
5. Convert straightforward middleware and config variants to readable tables.

Expected result: feature tests describe feature behavior rather than rebuilding generic transport and policy machinery.

### Phase 4 - Restore test ownership boundaries

1. Keep exhaustive validation at decoder/validator layers.
2. Reduce handlers to representative transport, translation, side-effect, and success cases.
3. Split oversized files by concern and move stable package-local test support into fixture files.
4. Replace brittle source text assertions with AST, type, behavior, or dedicated repository checks.
5. Reduce generated-code testing to semantic schema contracts and representative serialization smoke tests.

Expected result: every behavior has one primary owner and failures identify the relevant layer.

### Phase 5 - Add missing boundary confidence

1. Add direct `CaptionFetcher` tests.
2. Add a real Tap image acceptance journey.
3. Add selected routed HTTP plus PostgreSQL vertical slices.
4. Enforce duplicate indexer registration failure and collection parity.
5. Add timeline and notification query-plan guards if realistic-cardinality profiling shows they are warranted.

Expected result: confidence increases at currently uncovered production boundaries rather than through more mocked duplication.

### Phase 6 - Improve feedback and evidence

1. Add a fast unit-only feedback job while retaining `appview-check` as release evidence.
2. Add periodic shuffled-order and fuzz runs.
3. Produce coverage profiles as a trend, initially without a blocking threshold.
4. Record package durations, PostgreSQL connections, schema count, and flaky retries.
5. Reassess full/race sharding after fixture consolidation.

Expected result: regressions surface earlier without weakening the release-equivalent gate.

## Implementation results

All six phases were implemented in this branch. The work deliberately retained focused real-PostgreSQL coverage while reducing duplicate executions and making the release boundary more explicit.

Key outcomes:

- Removed every `time.Sleep(...)` from Go tests and replaced lock-delay assertions with channels, PostgreSQL lock observation, or explicit resource-contention barriers.
- Hardened `testdb` with bounded setup and cleanup, visible cleanup failures, collision-resistant schemas, deliberate pool limits, PostgreSQL 16 enforcement for migrated fixtures, and a fail-closed test database URL policy.
- Added centralized exact migration loading and migrated 110 tests away from open-coded migration path handling.
- Added `WithMigratedSchema` coverage for routed timeline reads, AppView-mediated durable PDS writes, critical indexers, and realistic-cardinality timeline and notification query plans.
- Removed duplicate business-neutrality execution and consolidated repeated route middleware checks into catalogue, architecture, and representative full-mux owners.
- Stopped HTTP helpers from silently modifying request JSON and reduced post-handler validation duplication while retaining representative no-side-effect and transport assertions.
- Split the largest post and federated-flow suites into concern-owned files with package-local support fixtures.
- Replaced brittle source-string checks with AST or semantic checks where practical and reduced generated-type tests to semantic contracts plus representative JSON/CBOR round trips.
- Added direct caption-fetcher status, content-type, size, redirect, and cancellation coverage.
- Made duplicate indexer registration fail closed and added ordering and duplicate-registration tests.
- Added pinned Tap `0.1.10` image startup, identity, health, filter, offline configuration, and AppView WebSocket handshake evidence. The gate records `events_injected=0` because that Tap version has no safe deterministic event-injection API; fake-server event and ACK behavior remains covered separately.
- Added explicit unit, full, shuffle, fuzz, and coverage commands, a fast CI feedback job, and periodic shuffled/fuzz/coverage evidence while preserving `appview-check` as the release-equivalent gate.

Measured static signals changed as follows:

| Signal | Before | After |
|---|---:|---:|
| Test files | 560 | 573 |
| Test lines | 135,705 | 135,841 |
| Top-level `Test*` functions | 2,258 | 2,227 |
| `testdb.WithSchema(...)` calls | 575 | 574 |
| Inline `CREATE TABLE` statements | 424 | 426 |
| `context.Background()` calls in tests | 2,682 | 2,647 |
| `t.Context()` calls in tests | 101 | 120 |
| `time.Sleep(...)` calls | 21 | 0 |
| Inline discard logger constructions | 74 | 0 |

The file and line totals remained approximately flat because the implementation added missing boundary, migrated-schema, query-plan, and test-infrastructure coverage while consolidating 31 duplicate top-level executions. The primary result is stronger ownership and determinism rather than deletion for its own sake.

Final verification:

- Focused affected packages passed under `go test -race`.
- `just appview-check` passed in full with uncached normal and race tests, required PostgreSQL and MinIO coverage, migration up/down/reapply checks, formatting, vet, Staticcheck, source and binary vulnerability checks, release builds, provenance comparison, image-memory evidence, startup health, and the pinned Tap handshake.
- Release-gate evidence was written to `/var/folders/zl/ymtyvzvn6510ld99pymykhy80000gn/T/tmp.yzit3AumzN` for this run.

## Suggested ownership rules

| Behavior | Primary owner |
|---|---|
| Request decoding and field validation | Decoder/validator test |
| HTTP status, headers, envelope, and error translation | Handler test |
| Authentication, device, rate, and body policy behavior | Middleware test |
| Route inventory and policy assignment | Route catalogue test |
| Handler registration and path-value wiring | Focused route/full-mux test |
| SQL result, transaction, lock, and constraint behavior | Focused real-Postgres store test |
| Compatibility with the exact final schema | Migrated-schema integration test |
| Historical schema transition | Migration test |
| Firehose projection and idempotency | Indexer/ingestion test |
| PDS request shape and effect semantics | PDS client/effect test |
| Cross-component process journey | Routed or release-gate integration test |
| Cross-file structural invariant | AST/type analysis or architecture test |
| Generated schema semantics | Lexicon contract test |
| Generated output freshness | Generation drift command |

If a second test asserts the same behavior, its name should state the distinct boundary, race, or regression it protects. Otherwise, remove it or move the assertion to the primary owner.

## Completion criteria

The recommendations are complete when:

- Every deleted test has a named surviving owner for meaningful behavior.
- Database fixtures clearly identify focused, migrated, or historical schema intent.
- Critical stores and indexers pass against the exact PostgreSQL 16 migrated schema.
- Route suites no longer repeat middleware matrices per feature.
- Request helpers do not mutate fixtures silently.
- Concurrency tests observe contention before asserting blocking.
- All goroutine waits and external infrastructure operations are bounded.
- Cleanup failures are visible and local test runs cannot target an arbitrary `DATABASE_URL` silently.
- `CaptionFetcher`, real Tap protocol compatibility, and selected routed vertical slices have direct owners.
- `just test`, `appview-test-unit`, and `appview-check` have clearly documented confidence levels.
- Normal, race, migration, image-memory, and required PostgreSQL/MinIO release evidence remains intact.
- Suite size, runtime, peak database connections, flaky retries, and coverage trend are recorded before and after cleanup.

## Risks

- Over-centralized DDL can make focused store tests harder to understand. Prefer named composable fixtures over one universal schema string.
- Reusing a schema across subtests can hide transaction leaks or couple cases. Do it only where reset semantics are explicit.
- Reducing handler validation cases can lose side-effect coverage. Keep representative no-write assertions for each error class.
- Replacing timing windows requires understanding the exact database or goroutine synchronization point; superficial channel barriers can recreate the same false confidence.
- Full migrated-schema tests can become slow. Keep the compatibility matrix intentionally small and risk-based.
- AST checks can be as brittle as source scans if they encode implementation shape rather than an architectural rule.
- Adding coverage thresholds prematurely can reward low-value tests. Establish a trend before setting policy.
- Sharding database tests before controlling pool and schema pressure can increase contention rather than reduce duration.

## Recommended first pull requests

1. Reliability fixes: delayed-write assertion, deterministic lock barriers, bounded waits, and visible cleanup.
2. `testdb` hardening: test-only URL policy, deadlines, one-time extension setup, and pool limits.
3. Exact migrated-schema fixture plus critical store/indexer compatibility tests.
4. Route catalogue consolidation and duplicate neutrality-test removal.
5. Shared HTTP/JSON/envelope helpers and removal of fixture body rewriting.
6. Handler/validator ownership cleanup beginning with post creation.
7. Direct caption-fetcher tests and duplicate dispatcher-registration protection.
8. Real Tap acceptance journey and selected routed vertical slices.
9. Fast feedback, shuffle, fuzz, and coverage-trend jobs.

Each pull request should report tests removed or consolidated, lines removed, package and full-suite duration, database-schema count, peak connections where measurable, and the surviving owner for every deleted behavior.
