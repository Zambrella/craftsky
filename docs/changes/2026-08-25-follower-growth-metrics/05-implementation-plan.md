# TDD Implementation Plan: Follower Growth Metrics

## Inputs

- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md`
- Coding plan: `04-coding-plan.md`
- Review note: `03-document-review.md` retains its pre-revision verdict. Per `04-coding-plan.md`, DR-001 through DR-005 were resolved in the revised requirements and tests, another review pass was explicitly skipped, and no blocking questions remain.
- Worktree: `.worktrees/follower-growth-metrics`
- Commits: Disabled unless explicitly requested.

## Implementation Rules

- Do not implement behavior without a linked requirement ID.
- Write or update one focused failing test before implementation.
- Run the smallest relevant test first and confirm a meaningful failure.
- Refactor only after the focused test passes.
- Keep traceability and loop evidence updated after each step.
- Use real Postgres with `TEST_DATABASE_REQUIRED=true` for persistence evidence.
- Preserve owner-private account boundaries and bounded telemetry.
- Do not modify lexicons, PDS write paths, feed ranking, or public profile contracts.

## Test Order

| Step | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| 1 | IT-001 | FR-001, FR-002, NFR-001 | AC-006, AC-007 | Migration objects and constraints are absent. |
| 2 | IT-002, REG-001 | FR-004, RULE-001 | AC-009 | Canonical view and capture store are absent. |
| 3 | IT-003 | FR-004 | AC-010 | Set-based capture implementation is absent. |
| 4 | IT-004 | FR-001, FR-005, FR-006, NFR-001 | AC-007, AC-011, AC-012 | Atomic run ledger and advisory coordination are absent. |
| 5 | UT-001 | FR-011, RULE-002, NFR-003 | AC-018, AC-034 | Period parser and UTC range calculation are absent. |
| 6 | UT-002, UT-003 | BR-003, FR-012, FR-014, FR-015 | AC-004, AC-005, AC-019, AC-021, AC-022 | Sparse series shaping and net change are absent. |
| 7 | UT-005, IT-005 | FR-003, NFR-002 | AC-008, AC-033 | Worker scheduling and process wiring are absent. |
| 8 | IT-006, REG-002 | FR-007, FR-008 | AC-013, AC-014 | Later-capture-only behavior is unprotected. |
| 9 | UT-004 | FR-010 | AC-015, AC-017 | API response DTO and JSON contract are absent. |
| 10 | AT-004, IT-007 | BR-001, FR-009, FR-010 | AC-002, AC-015, AC-016, AC-017 | Owner route, policy, and handler are absent. |
| 11 | IT-008 | BR-003, FR-011, FR-012, FR-015, RULE-002, NFR-003 | AC-004, AC-005, AC-018, AC-019, AC-022, AC-034 | Bounded sparse reads are absent. |
| 12 | IT-009 | FR-013, NFR-002 | AC-020 | Persisted-only endpoint behavior is absent. |
| 13 | IT-010 | RULE-003 | AC-030 | Retention non-participation has no executable evidence. |
| 14 | IT-011, REG-005 | RULE-004 | AC-031 | Terminal inventory does not purge snapshots. |
| 15 | UT-012, IT-013 | NFR-007 | AC-038 | Bounded follower-growth observations are absent. |
| 16 | REG-003, REG-004 | RULE-005 | AC-032 | Architecture and public-response boundaries lack growth evidence. |
| 17 | UT-006 | BR-001, FR-010, RULE-002, NFR-006 | AC-002, AC-017, AC-029, AC-037 | Flutter model and API decode path are absent. |
| 18 | AT-005, UT-008 | FR-020 | AC-028 | Account-keyed provider is absent. |
| 19 | IT-012 | FR-016 | AC-023 | Growth Settings row and typed route are absent. |
| 20 | AT-001, AT-002, UT-007 | BR-001, BR-002, FR-015, FR-017, FR-018, FR-019 | AC-001, AC-002, AC-003, AC-022, AC-024, AC-025, AC-026, AC-027 | Growth page and states are absent. |
| 21 | AT-003 | BR-003, FR-012, FR-015, FR-018 | AC-004, AC-005, AC-019, AC-022, AC-026 | Honest incomplete-history UI evidence is absent. |
| 22 | AT-006, UT-009, UT-010 | FR-016, NFR-004, NFR-005 | AC-023, AC-035, AC-036 | Responsive chart and semantics are absent. |
| 23 | UT-011 | NFR-006 | AC-037 | Locale-aware formatting evidence is absent. |
| 24 | REG-001 through REG-005 | FR-008, RULE-001, RULE-004, RULE-005 | AC-009, AC-014, AC-031, AC-032 | Full regression pass has not run. |
| 25 | MAN-001, MAN-002 | NFR-004, NFR-005, NFR-006 | AC-035, AC-036, AC-037 | Release-device manual evidence remains outstanding. |
| 26 | IT-011, REG-005 correction | RULE-004 | AC-031 | Capture and terminal purge are not coordinated. |
| 27 | UT-005, IT-013 correction | NFR-002, NFR-007 | AC-033, AC-038 | Failure observations report zero latest-success age. |
| 28 | AT-004, IT-007 correction | FR-009, FR-010 | AC-015, AC-016, AC-017 | Unknown query keys and production middleware scenarios lack coverage. |
| 29 | IT-009 correction | FR-013, NFR-002 | AC-020 | Persisted-only behavior lacks route-level database evidence. |
| 30 | IT-006 correction | FR-007, FR-008 | AC-013, AC-014 | Rejoin behavior is not covered. |
| 31 | UT-006 correction | FR-010, FR-012, FR-017 | AC-017, AC-019, AC-024 | Flutter accepts internally inconsistent responses. |
| 32 | AT-005, UT-008 correction | FR-020 | AC-028 | Active switch and logout are not exercised through presentation state. |
| 33 | UT-007 correction | FR-017, FR-018 | AC-024, AC-026 | Delayed stale period completion is not covered. |
| 34 | IT-012 correction | FR-016 | AC-023 | Growth routing is not exercised at compact and large widths. |
| 35 | AT-006, UT-009, UT-010 correction | NFR-004, NFR-005 | AC-035, AC-036 | Dark-theme, large-label, and complete trend semantics evidence is absent. |
| 36 | UT-012, IT-013 correction | NFR-007 | AC-038 | Planned bounded per-attempt logs are absent. |
| 37 | REG-001 through REG-005 | All corrected requirements | All linked criteria | Corrected full gates have not run. |
| 38 | UT-005, IT-013 correction | NFR-002, NFR-007 | AC-033, AC-038 | Unknown latest-success age is emitted as zero. |
| 39 | UT-006 correction | FR-010, FR-012, FR-014, FR-015 | AC-017, AC-019, AC-021, AC-022 | Cross-field response contradictions are accepted. |
| 40 | UT-007 correction | FR-017, FR-018 | AC-024, AC-026 | Stale completion during selected-period loading is not proved. |
| 41 | UT-009, UT-011 correction | NFR-004, NFR-006 | AC-035, AC-037 | Compact localized Y-axis labels can exceed reserved space without detectable evidence. |
| 42 | REG-001 through REG-005 | All corrected requirements | All linked criteria | Final correction gates have not run. |
| 43 | UT-006 correction | FR-010, FR-015 | AC-017, AC-022 | In-range global boundary dates need not match the first and last observations. |
| 44 | REG-001 through REG-005 | All corrected requirements | All linked criteria | Boundary-correction release gates have not run. |

## Implementation Steps

### Step 1: IT-001

- Write failing test: Add a focused reversible-migration integration test for snapshot row shape, non-negative counts, composite uniqueness, next-date insertion, run-ledger constraints, and owner/date index order.
- Run command: Start the worktree Postgres with `just dev-d`, then run `go test -race ./internal/db -run '^TestFollowerGrowthMigration$' -count=1 -v` with the worktree database URL and `TEST_DATABASE_REQUIRED=true`.
- Confirmed failure: `TestFollowerGrowthMigration` failed because `000060_follower_growth_snapshots.up.sql` did not exist.
- Implement: Add only migration `000060` objects needed by IT-001.
- Refactor: The `(profile_did, snapshot_date)` primary key provides the owner/date range index; no redundant secondary index was added.
- Notes: Added reversible `000060` migrations with the canonical count view, immutable daily snapshot key, non-negative count constraints, and successful-run ledger. The focused test and full `internal/db` package pass with required real Postgres.

### Step 2: IT-002 And REG-001

- Write failing test: Capture mixed current/external relationships and compare snapshots to existing profile reads.
- Run command: Focus `internal/followergrowth` and `internal/api` test names.
- Confirmed failure: `TestStoreCaptureCanonicalCounts` failed to compile because the follower-growth store and `NewStore` did not exist. REG-001 was an existing green regression baseline.
- Implement: Add canonical count view usage and minimum store capture.
- Refactor: Switched `ProfileStore.Read` from its embedded follower subquery to `craftsky_profile_follower_counts` while REG-001 remained green.
- Notes: Added the minimum one-statement capture store. The mixed graph test proves three current members are captured, zero-follower members receive zero, the eligible current-member edge is counted, and external actors/targets are excluded.

### Step 3: IT-003

- Write failing test: Assert capture uses one set-based statement and inspect for pathological repeated work.
- Run command: Focus `internal/followergrowth` query-plan test.
- Confirmed failure: This evidence was inherited green from IT-002 because its minimum implementation was necessarily one `INSERT ... SELECT`; no additional production behavior was added.
- Implement: Keep capture as one `INSERT ... SELECT`, with no DID loop.
- Refactor: None.
- Notes: Added source-shape and real `EXPLAIN (FORMAT JSON)` evidence for one insert operation over the canonical all-member view with no repeated subplan.

### Step 4: IT-004

- Write failing test: Prove rollback and race two process-equivalent capture attempts.
- Run command: Focus store concurrency/rollback tests with required Postgres.
- Confirmed failure: The injected run-ledger failure did not fail capture and left snapshots committed; concurrent attempts returned a snapshot primary-key violation.
- Implement: Transaction advisory lock, immutable completed-run row, atomic commit.
- Refactor: Kept one system-scoped advisory lock and the existing one-statement member capture; no generic locking facility was introduced.
- Notes: Capture now acquires a transaction advisory lock, checks immutable completion state, writes snapshots and the run ledger in one transaction, rolls back both on failure, and returns `AlreadyCompleted` to the serialized retry. Focused and package tests pass with real Postgres.

### Step 5: UT-001

- Write failing test: Cover `7d`, `30d`, `1y`, UTC normalization, and 2028-02-29 clamping.
- Run command: `go test ./internal/followergrowth -run '^TestPeriod' -count=1`.
- Confirmed failure: The test failed to compile because `ParsePeriod` and period/range types did not exist.
- Implement: Period parser and inclusive date range.
- Refactor: Kept date identity as UTC midnight values with no timezone dependency.
- Notes: Added strict `7d`, `30d`, and `1y` parsing; inclusive 7/30-day ranges; prior-year anniversary calculation; February 29 to February 28 clamp; and a 367-point maximum. Focused tests pass.

### Step 6: UT-002 And UT-003

- Write failing test: First sparse calendar shaping, then selected-range net change.
- Run command: Focus each series test independently.
- Confirmed failure: UT-002 failed to compile because history/series types and `BuildSeries` did not exist. UT-003 then failed because increasing, decreasing, and flat multi-point series returned a nil change.
- Implement: Fill explicit null gaps and calculate change from in-range observations only.
- Refactor: Net change is derived during selected-range calendar shaping, so global availability/latest metadata cannot influence it.
- Notes: Added complete chronological points with explicit null gaps, global `availableFrom` and latest metadata propagation, and nullable latest-minus-earliest in-range change. Focused series tests pass.

### Step 7: UT-005 And IT-005

- Write failing test: First deterministic worker timing, then process lifecycle wiring.
- Run command: Focus follower-growth worker package and command tests.
- Confirmed failure: UT-005 failed to compile because the worker and injected time boundary did not exist. IT-005 failed to compile because the command-layer starter did not exist.
- Implement: Immediate run, UTC-midnight wait, five-minute retry, cancellation, bounded shutdown.
- Refactor: Extracted a narrow `followerGrowthRunner` command boundary and kept feature construction in `newContentDependencies` to satisfy dependency architecture rules.
- Notes: Added immediate startup capture, next-UTC-midnight scheduling, five-minute error retry, prompt cancellation, dependency construction, command startup, and shutdown joining. Focused worker, command, and app dependency tests pass.

### Step 8: IT-006 And REG-002

- Write failing test: Mutate membership/follows between dates and prove no event-time analytics writes.
- Run command: Focus store and index boundary tests.
- Confirmed failure: Both checks were inherited green from the established snapshot boundary; no production change was needed.
- Implement: No indexer changes expected; capture reads current canonical projection.
- Refactor: None.
- Notes: Real-Postgres coverage proves joins/follows after a completed run do not alter that date, later capture observes the changes, departure/unfollow affects only a subsequent date, and index package sources contain no follower-growth dependency or SQL reference.

### Step 9: UT-004

- Write failing test: Marshal populated and no-history camelCase response bodies.
- Run command: Focus API response test.
- Confirmed failure: The API package failed to compile because the follower-growth response DTO and mapper did not exist.
- Implement: Add response DTOs with always-present nullable fields.
- Refactor: Centralized date-only pointer formatting and kept the wire DTO separate from domain history.
- Notes: Populated and no-history responses now retain all camelCase keys, explicit null metadata, chronological points, date-only buckets, and RFC3339 capture time.

### Step 10: AT-004 And IT-007

- Write failing test: Add owner-only route/policy/query validation tests.
- Run command: Focus route tests.
- Confirmed failure: Route tests failed to compile because the handler did not exist; after registration, the exhaustive route inventory failed until the new route was explicitly added.
- Implement: Add policy, narrow dependency, handler, and route registration.
- Refactor: The handler depends only on a narrow persisted-history reader and derives its typed owner DID from middleware context.
- Notes: Added strict single-value period validation, canonical errors, current-member read policy, owner-only handler, capability registration, process-to-route dependency lowering, and exact route inventory coverage. Route package tests pass.

### Step 11: IT-008

- Write failing test: Read exact sparse ranges, global availability, no-history, and leap-day bounds.
- Run command: Focus store read integration test.
- Confirmed failure: The placeholder production reader returned nil `availableFrom` for populated history.
- Implement: One bounded owner-history read plus deterministic Go series shaping.
- Refactor: One SQL statement uses global owner-history metadata plus an independently bounded range CTE; Go remains responsible for explicit calendar gaps.
- Notes: Real-Postgres coverage proves global earliest/latest metadata, chronological in-range observations, no-history success, sparse gaps, and the 367-point leap-day series.

### Step 12: IT-009

- Write failing test: Mutate active follows after a snapshot and verify endpoint remains persisted-only.
- Run command: Focus route live-overlay test.
- Confirmed failure: Inherited green from the narrow handler dependency established by IT-007.
- Implement: Ensure route never calls live profile count.
- Refactor: None.
- Notes: The HTTP handler can only call `FollowerGrowthReader.Read`; no live profile-count dependency is accepted or wired, and persisted latest metadata drives the response.

### Step 13: IT-010

- Write failing test: Assert ordinary retention inventories do not include growth snapshots.
- Run command: Focus retention test.
- Confirmed failure: Inherited green because no ordinary retention subsystem referenced the new table.
- Implement: No production retention registration was added.
- Refactor: None.
- Notes: Added executable source-inventory coverage ensuring ordinary retention files do not reference follower-growth snapshots.

### Step 14: IT-011 And REG-005

- Write failing test: Purge one owner's snapshots while retaining another owner's rows and public follows.
- Run command: Focus owner-lifecycle tests.
- Confirmed failure: The fail-closed migrated-schema test reported both the undeclared snapshot DID role and the derived view incorrectly treated as persisted storage.
- Implement: Add snapshot DID inventory and base-table-only schema discovery.
- Refactor: Persisted-DID discovery now joins `information_schema.tables` and accepts only base tables.
- Notes: Added the owner snapshot role to terminal inventory and a real purge test proving only the terminal owner's snapshots are removed while another owner's snapshot and the public follow projection remain.

### Step 15: UT-012 And IT-013

- Write failing test: Capture telemetry for success/failure paths and scan for private/high-cardinality fields.
- Run command: Focus observability tests.
- Confirmed failure: The focused worker test failed to compile because follower-growth recorder and worker-observer methods did not exist. API logging privacy was inherited green from the bounded HTTP logging middleware.
- Implement: Add bounded worker observation and recorder support.
- Refactor: Added feature-specific allowlists so arbitrary single-token results and categories collapse to `unknown`; count and age remain dimension-free numeric measurements.
- Notes: Added worker success, failure, and already-complete observations; synchronized noop, in-memory, and Sentry recorders; AppView observer wiring; hostile synthetic-label coverage; and API error logging coverage proving DIDs, handles, rows, per-account counts, and raw errors are absent. Focused follower-growth, observability, API, and app dependency tests pass.

### Step 16: REG-003 And REG-004

- Write failing test: Inventory allowed growth dependencies and assert public profile responses remain unchanged.
- Run command: Focus route architecture and profile response tests.
- Confirmed failure: Both checks were inherited green because production references were already confined to the private owner route/service, runtime composition, telemetry, and terminal-deletion boundaries, and public profile DTOs were unchanged.
- Implement: Keep growth reads confined to owner route/deletion/runtime boundaries.
- Refactor: None.
- Notes: Added an exhaustive production-reference allowlist, targeted feed/timeline/ranking/recommendation/search/discovery/moderation/advertising guards, explicit absence checks for recommendation and advertising subsystems, and public-profile JSON assertions that permit the existing summary count but reject every growth-history field. Focused route and API tests pass.

### Step 17: UT-006

- Write failing test: Decode populated/no-history payloads and preserve UTC date-only identity.
- Run command: `just app-test test/profile/models/follower_growth_test.dart`.
- Confirmed failure: The model test failed to compile because `FollowerGrowth`, `FollowerGrowthPeriod`, and `follower_growth.dart` did not exist. The subsequent API-client test failed because `getFollowerGrowth` did not exist.
- Implement: Add strict Growth models and profile API/repository method.
- Refactor: Kept date-only parsing in the model boundary, normalized capture timestamps to UTC, and exposed one enum wire value for the allowlisted query.
- Notes: Added strict populated/no-history camelCase decoding, UTC date identity, period and chronology validation, bounded points, and the exact owner endpoint/query. Focused model and API-client tests and analysis pass. Repository delegation is sequenced with the account-bound provider in Step 18.

### Step 18: AT-005 And UT-008

- Write failing test: Delay account A, switch to B, complete B then A, and test logout.
- Run command: Focus follower-growth provider test.
- Confirmed failure: The provider test failed to compile because the account repository family, growth provider, repository method, and fake callback did not exist.
- Implement: Add account-and-period keyed auto-disposed provider and invalidation.
- Refactor: Reused the existing account-bound Dio repository pattern and account-state invalidator; no manual request cancellation or unkeyed cache was added.
- Notes: Added repository delegation, a programmable test fake, generated account repository and growth provider families, and invalidation registration. Focused tests prove Bob's result remains unchanged after Alice's delayed completion and active retained state reloads after account invalidation.

### Step 19: IT-012

- Write failing test: Navigate Settings to Growth and back at compact and large widths.
- Run command: Focus Settings route/page tests.
- Confirmed failure: Tests failed because the Growth row identity, localized Settings entry, typed route, and destination page did not exist.
- Implement: Add descriptor, row, location, typed route, and generated route output.
- Refactor: Made the shared route-loop test ensure each scrolled Settings row is actually visible before tapping; the added row exposed the old viewport-edge assumption.
- Notes: Added Growth before Followers in Connections, canonical `/profile/settings/growth`, authenticated-shell navigation, generated route output, and a minimal localized destination. Focused model, Settings page, and production-router tests pass, including back navigation and large-shell preservation.

### Step 20: AT-001, AT-002, And UT-007

- Write failing test: Add 30-day default, period switching, scope/freshness copy, and async/history states one behavior at a time.
- Run command: `just app-test test/settings/follower_growth_page_test.dart`.
- Confirmed failure: The minimal destination had no metric content, period controls, active-account provider watch, loading/error states, summaries, freshness metadata, or retry behavior.
- Implement: Add localized Growth page with exact account/period provider watches.
- Refactor: Kept period selection local and switched provider family keys rather than carrying an `AsyncValue` between requests; safe error copy never renders the underlying exception.
- Notes: Added the 30-day default, all period controls, latest count and signed/flat/insufficient summaries, Craftsky scope and daily freshness copy, latest snapshot date, loading, and exact-key retry. Focused widget tests prove period-specific content replaces stale content and retry reloads the current account/period only.

### Step 21: AT-003

- Write failing test: Cover no, one, partial, leading-gap, and interior-gap histories without interpolation.
- Run command: Focus Growth page test and sparse Go series tests.
- Confirmed failure: The page rendered only the generic latest summary and could not distinguish no global history, an empty selected range with older history, activation-limited history, or missing dates.
- Implement: Add honest state and gap explanations.
- Refactor: Derived observed and missing counts directly from nullable server points and kept global availability separate from selected-range observations.
- Notes: Added distinct no-history and no-observations-in-period states, available-since copy, singular/plural missing-date explanations, and one-point insufficient-change behavior. Focused page tests pass across no, one, partial, leading-gap, and interior-gap fixtures without carrying values forward.

### Step 22: AT-006, UT-009, And UT-010

- Write failing test: Add compact/large/text-scale layout and stable semantic-summary assertions.
- Run command: Focus Growth page and accessibility tests.
- Confirmed failure: Tests failed to compile because `fl_chart` and `FollowerGrowthChart` did not exist.
- Implement: Add `fl_chart`, explicit null spots, responsive constraints, and package-independent semantics.
- Refactor: Kept the renderer non-interactive and animation-free, constrained only the plot to left-to-right chronology, and moved meaning into one `Semantics` node with renderer internals excluded. Wrapped the period selector in horizontal scrolling for 200% compact text.
- Notes: Added pinned `fl_chart` 1.2.0, explicit `FlSpot.nullSpot` gaps with fixed x bounds, visible observation dots, flat-series padding, sampled axes, and a bounded chart height. Focused tests pass at 320x640 and 1200x800 with 200% text and prove period/count/trend/range/freshness/missing-date semantics independent of colour.

### Step 23: UT-011

- Write failing test: Format a fixed UTC date and large count with explicit locales.
- Run command: Focus follower-growth format test.
- Confirmed failure: The formatting test failed to compile because locale-aware count and snapshot-date helpers did not exist.
- Implement: Add locale-aware presentation helpers without timezone rebucketing.
- Refactor: Replaced temporary ISO/raw page, axis, and semantic labels with shared `intl` helpers while leaving model dates untouched and UTC.
- Notes: English and German tests prove count grouping and date order change by locale while UTC identity remains unchanged. Focused formatting, page, chart, and accessibility tests pass.

### Step 24: REG-001 Through REG-005

- Run command: Focus all new regression tests, then broader AppView regression gates.
- Notes: The initial correction-pass `just test` exposed a schema-unqualified migration assertion and the intentionally added terminal-purge dependency missing from the architecture allowlist. Both test contracts were corrected. The final `just test` real-Postgres/race suite and `just appview-check` disposable release gate pass.

### Step 25: MAN-001 And MAN-002

- Manual checks: Real screen-reader traversal, grayscale/non-colour comprehension, representative compact/large/RTL/large-text chart legibility.
- Notes: Not performed in this environment. These remain explicit release-candidate checks on a supported mobile platform and representative devices; automated semantics, RTL-independent chronology, compact/large, light/dark, and 200%-text evidence passes.

### Correction Pass From `06-implementation-review.md`

The implementation review returned `Changes required`. Corrections proceed in privacy/risk order and retain the existing test IDs rather than inventing new contract behavior.

### Step 26: IT-011 And REG-005 Correction

- Write failing test: Deterministically pause capture after snapshot insertion, run terminal purge concurrently, and prove purge cannot complete before the writer is fenced and no private snapshot can reappear afterward.
- Run command: Focus the owner-lifecycle follower-growth purge integration test with required real Postgres.
- Confirmed failure: The purge completed while capture was paused before its run-ledger insert, leaving capture able to commit the owner's private snapshot after purge completion.
- Implement: Added one feature-specific transaction advisory-lock function and acquired it from both capture and the follower-growth terminal-purge component.
- Refactor: Capture now uses the shared feature-specific lock function; no generic purge lock or owner loop was introduced.
- Notes: Addresses IR-001. The deterministic race test and existing owner-only purge test pass with required real Postgres.

### Step 27: UT-005 And IT-013 Correction

- Write failing test: Prove latest-success age comes from the completed-run ledger for success, duplicate, and failure attempts and increases across retries.
- Run command: Focus follower-growth store, worker, and observability tests.
- Confirmed failure: An injected current-run ledger failure returned a zero latest-success age despite a completed run 24 hours and 5 minutes earlier.
- Implement: Read current-date completion and the latest completed-run timestamp together before capture; preserve the derived age on later snapshot/run-write failures.
- Refactor: Extended the deterministic worker observation fixture to assert that failure and retry ages are forwarded and increase, without adding account dimensions.
- Notes: Addresses IR-002. Focused store, worker, and observability tests pass with required real Postgres.

### Step 28: AT-004 And IT-007 Correction

- Write failing test: Reject every non-period query key and exercise valid/invalid owners through the production route middleware.
- Run command: Focus follower-growth route tests.
- Confirmed failure: Requests containing `did`, `handle`, `timezone`, `cursor`, or `range` alongside a valid period returned 200.
- Implement: Require the parsed query to contain exactly one key named `period` with exactly one value.
- Refactor: Added production-mux evidence for auth, device, current/departed membership, arbitrary-profile rejection, all periods, two owners, and no-history success; existing middleware behavior required no production change.
- Notes: Addresses IR-003 and IR-004. Focused route tests pass with real Postgres membership/lifecycle fixtures.

### Step 29: IT-009 Correction

- Write failing test: Persist yesterday's snapshot, mutate the active graph, and request Growth through the route.
- Run command: Focus follower-growth route integration evidence with required real Postgres.
- Confirmed failure: Inherited green because the production route already delegates only to the persisted-history store.
- Implement: No production change.
- Refactor: None.
- Notes: Addresses IR-005. Real-Postgres production-route evidence returns persisted count 8/date while the canonical live count is 1.

### Step 30: IT-006 Correction

- Write failing test: Rejoin a departed member and prove snapshotting resumes only on a later successful date.
- Run command: Focus follower-growth store integration test with required real Postgres.
- Confirmed failure: Inherited green from canonical capture semantics.
- Implement: No production change.
- Refactor: None.
- Notes: Addresses IR-006. The expanded real-Postgres lifecycle test proves rejoin/refollow does not change the prior date and appears on the next capture.

### Step 31: UT-006 Correction

- Write failing test: Reject response-period mismatch, out-of-range/non-contiguous/incomplete points, and contradictory nullable metadata.
- Run command: Focus follower-growth model and API-client tests.
- Confirmed failure: The model accepted missing and out-of-range dates plus no-history metadata with a non-null latest count; the client accepted a valid 7-day body for a 30-day request.
- Implement: Enforce period-specific inclusive ranges, one exact point per date, coherent history/latest fields and net change, and request/response period equality.
- Refactor: Updated API fixtures to represent the complete server contract rather than abbreviated point lists.
- Notes: Addresses IR-007. Focused model and API-client tests pass.

### Step 32: AT-005 And UT-008 Correction

- Write failing test: Switch the active session from delayed account A to B, then log out, and prove A is never rendered and private state clears.
- Run command: Focus follower-growth provider/page tests.
- Confirmed failure: Inherited green from account-keyed providers and direct active-session selection.
- Implement: No production change.
- Refactor: Added a page-level mutable-session test covering delayed Alice, active Bob, late Alice completion, and removal of every retained session.
- Notes: Addresses IR-008. The page never renders Alice for Bob and removes Bob's private value immediately on logout.

### Step 33: UT-007 Correction

- Write failing test: Switch periods while the old request is pending, complete the selected request and then the old request, and prove selected content remains stable.
- Run command: Focus Growth page tests.
- Confirmed failure: Period controls were absent during loading, so a second period could not be selected while the first request was pending.
- Implement: Keep the period selector outside the account/period async result while transitioning only the result body through loading/error/data.
- Refactor: Reused one selector widget and kept exact account/period invalidation for retry.
- Notes: Addresses IR-009. A controlled three-period test proves late 30-day and 7-day completions cannot replace selected 1-year content.

### Step 34: IT-012 Correction

- Write failing test: Open Growth and navigate back through the production router at compact and large widths.
- Run command: Focus Settings route tests.
- Confirmed failure: Inherited green from the typed route and authenticated shell.
- Implement: No production change.
- Refactor: Added parameterized compact/large production-router evidence with canonical location, one appropriate shell, and Settings back-stack restoration.
- Notes: Addresses IR-010. Both focused route cases pass.

### Step 35: AT-006, UT-009, And UT-010 Correction

- Write failing test: Add dark-theme, large localized count/200%-text bounds, and negative/flat/one-point semantic fixtures.
- Run command: Focus Growth page and accessibility tests.
- Confirmed failure: Existing fixed axis reservation and edge titles did not account for measured localized text at the active scale.
- Implement: Measure localized min/max labels inside `LayoutBuilder`, reserve bounded responsive axis width, and fit sampled edge-date titles inside the chart axis.
- Refactor: Expanded tests to compact/large, light/dark, 200% text, a nine-digit count, gaps, and positive/negative/flat/one-point semantics.
- Notes: Addresses IR-011. All focused page and accessibility tests pass; MAN-001/MAN-002 remain release-device checks.

### Step 36: UT-012 And IT-013 Correction

- Write failing test: Capture bounded per-attempt logs and reject identifiers, per-owner values, rows, and raw errors.
- Run command: Focus follower-growth observability tests.
- Confirmed failure: The observer emitted no follower-growth attempt log.
- Implement: Emit one bounded observer log per attempt with only component, operation, sanitized result, and sanitized error category; use error level only for bounded error outcomes.
- Refactor: Reuse the same sanitized labels for metrics and logs.
- Notes: Addresses IR-013 and the existing coding-plan requirement. Focused hostile-label log and metric tests pass without identifiers, counts, ages, or raw errors.

### Step 37: Corrected Full Regression Pass

- Run command: Full Flutter tests and analysis, `just test`, `just appview-check`, and `git diff --check`.
- Notes: Full Flutter suite passed all 1,507 tests; Dart workspace analysis reported no errors; `just test` passed every package with real Postgres and the race detector; `just appview-check` reported all release gates passed; `git diff --check` passed. The prior unrelated auth flake did not recur. The gate's existing vulnerability policy reported allowed `GO-2026-5932` in transitive `x/crypto/openpgp` with no available fix and still passed.

### Final Correction Pass From `06-implementation-review.md`

The correction review retained four Must-level findings. Steps 38 through 42 address them in backend behavior, Flutter contract, presentation-state, and responsive-layout order.

### Step 38: UT-005 And IT-013 Correction

- Write failing test: Prove an unavailable latest-success age does not emit a zero freshness gauge while a genuinely fresh successful run emits zero.
- Run command: Focus follower-growth observability and worker/store tests.
- Confirmed failure: The focused observability test failed to compile because the recorder accepted only a concrete duration and therefore could not represent an unknown age.
- Implement: Changed capture results and follower-growth observations to carry a nullable latest-success age. Successful current runs carry a known zero; attempts that cannot read a completed run carry null; recorders omit only the freshness gauge when null.
- Refactor: Kept duration/count/attempt telemetry unchanged and retained a plain duration behind the optional boundary rather than adding another outcome dimension.
- Notes: Addresses IR-014. Focused follower-growth, observability, app-dependency, and command tests pass.

### Step 39: UT-006 Correction

- Write failing test: Reject incorrect net change, observations before availability, and an in-range latest point/count mismatch.
- Run command: Focus follower-growth model tests.
- Confirmed failure: The malformed-payload test returned a valid `FollowerGrowth` for an incorrect change, an observation before `availableFrom`, and an in-range latest-count mismatch.
- Implement: Validate net change against first/last selected observations, reject observations before global availability, reject a latest date after the current range, and require an in-range latest point/count match.
- Refactor: Reused the observed-point list for count and cross-field checks; no client-side recalculation replaces server values.
- Notes: Addresses IR-015. Focused model, formatting, and API-client tests pass.

### Step 40: UT-007 Correction

- Write failing test: Complete stale period requests while the selected request is loading, then verify selected control, loading, chart, and semantics remain authoritative.
- Run command: Focus Growth page tests.
- Confirmed failure: The first focused run failed only because the test deferred semantics disposal past Flutter's end-of-test verification; after correcting the fixture lifecycle, the strengthened behavior was inherited green.
- Implement: No production change.
- Refactor: Complete both stale requests while the selected one-year request remains pending, then assert loading, selected control, summary, chart, and semantic identity before and after selected completion.
- Notes: Addresses IR-016. Account-and-period provider keys prevent stale completion from disturbing selected presentation state.

### Step 41: UT-009 And UT-011 Correction

- Write failing test: Render a large localized Y-axis value at compact width and 200% text and prove every visible axis label remains inside its reserved chart bounds.
- Run command: Focus Growth page/chart tests.
- Confirmed failure: The render-bounds test first found no inspectable Y-axis labels. After labels were bounded and keyed, the top title still painted above the chart, proving the prior exception-only test missed real clipping.
- Implement: Use locale-aware compact notation when full count labels exceed the bounded axis reservation, scale labels down as a final single-line guard, and fit left-axis edge titles inside chart bounds.
- Refactor: Added one rendered German-locale, 320-pixel, 200%-text, nine-digit-count test that inspects every Y-axis label's global render bounds.
- Notes: Addresses IR-017. The focused page and accessibility suites pass; MAN-002 remains release-device evidence.

### Step 42: Final Corrected Regression Pass

- Run command: Full Flutter tests and analysis, `just test`, `just appview-check`, and `git diff --check`.
- Notes: All 1,509 Flutter tests passed; Flutter analysis reported no issues; `just test` passed every AppView package with real Postgres and the race detector; `just appview-check` reported all release gates passed; `git diff --check` passed. The existing allowlisted `GO-2026-5932` transitive `x/crypto/openpgp` advisory remains unchanged and has no available fix.

### Boundary Correction From `06-implementation-review.md`

The final review retained one cross-field wire invariant. Steps 43 and 44 require global availability/latest metadata inside the selected range to identify the actual first/last observed points.

### Step 43: UT-006 Correction

- Write failing test: Reject an in-range `availableFrom` date with a null point and an observation newer than `latestSnapshotDate`.
- Run command: Focus follower-growth model tests.
- Confirmed failure: Both malformed payloads decoded successfully: one had a null point on its in-range availability date, and one had an observation after its claimed latest snapshot.
- Implement: When `availableFrom` is inside the selected range, require it to identify the first observed point. When selected observations exist, require latest metadata to identify the last observed point before validating its count.
- Refactor: Reused the already-derived ordered observed-point list; valid partial histories whose global availability/latest metadata predate an empty selected range remain supported.
- Notes: Addresses IR-018. Focused model and affected client suites pass.

### Step 44: Boundary-Corrected Regression Pass

- Run command: Focused Flutter suites, Flutter analysis and full tests, `just test`, `just appview-check`, and `git diff --check`.
- Notes: All 1,510 Flutter tests passed; Flutter analysis and focused Dart analysis reported no issues; `just test` passed every AppView package with real Postgres and the race detector; `git diff --check` passed. Two default-concurrency `just appview-check` reruns exited nonzero only because disposable Postgres returned `out of shared memory (SQLSTATE 53200)` while unrelated `pdseffects`, then `ownerlifecycle`/`index`, tests created isolated schemas. The exact release gate passed immediately before this client-only Step 43 change, and no AppView source changed in Step 43. This infrastructure-capacity rerun gap is retained explicitly for final review rather than marked green.

## Completion Checklist

- [x] All Must requirements covered by tests or documented gaps.
- [x] All planned automated Must tests passing.
- [x] Relevant regression tests passing.
- [x] No unlinked behavior implemented.
- [x] Go formatting, checks, and static analysis pass.
- [x] Flutter tests and analysis pass.
- [x] Manual checks completed or explicitly recorded as outstanding.
- [x] Docs updated and read back.
- [ ] Final correction implementation review pending.
