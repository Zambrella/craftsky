# Test Performance Improvement Plan

**Date:** 2026-09-17
**Scope:** Go/AppView and Flutter test execution
**Status:** Proposed

## Purpose

The Go and Flutter redundancy audits improved reliability, ownership, and coverage. The next opportunity is to reduce feedback time without weakening the boundaries those audits deliberately retained.

The largest likely gains are in test execution topology, repeated fixture setup, dependency caching, and CI parallelism. Further broad deletion of tests is not the primary strategy.

## Principles

1. Measure before changing concurrency or isolation.
2. Keep release evidence stronger than local and pull-request feedback paths.
3. Do not trade deterministic tests for faster but timing-dependent tests.
4. Keep real PostgreSQL coverage for transactions, locks, constraints, query plans, and migrations.
5. Keep full Go race coverage in a trusted release or main-branch gate.
6. Keep Flutter device tests separate from the fast unit/widget path.
7. Optimize total developer and CI wall time, not only individual test duration.
8. Track compute cost and memory as well as elapsed time when adding parallelism.

## Current State

### Go/AppView

- `scripts/appview-test` provides full, unit-only, shuffle, fuzz, and coverage modes.
- The full mode runs every package with the race detector, `-p=1`, and `-count=1`.
- The unit-only mode still compiles every package and test file; PostgreSQL and MinIO tests skip at runtime.
- `scripts/appview-check` runs the complete suite once normally and once under the race detector.
- Database packages are serialized to avoid exhausting PostgreSQL's shared lock table.
- The suite contains 574 `testdb.WithSchema(...)` calls. Each call currently creates a bootstrap pool, isolated schema, scoped pool, fixture DDL, lifecycle predicates, and cleanup resources.
- Exact migrated-schema fixtures and the operational migration reversal gate protect distinct risks and should remain separate.
- Shuffle, fuzz, and coverage evidence already run outside the pull-request critical path.

Relevant entry points:

- [`scripts/appview-test`](../scripts/appview-test)
- [`scripts/appview-check`](../scripts/appview-check)
- [`appview/internal/testdb/testdb.go`](../appview/internal/testdb/testdb.go)
- [`.github/workflows/backend-ci.yml`](../.github/workflows/backend-ci.yml)
- [`docs/2026-09-17-go-test-redundancy-audit.md`](2026-09-17-go-test-redundancy-audit.md)

### Flutter

- The unit/widget suite contains 483 files and 2,176 direct `test(...)` or `testWidgets(...)` declarations.
- The normal command is `flutter test` with no explicit concurrency, sharding, or `--no-pub` behavior.
- The suite still contains approximately 1,389 `pumpAndSettle()` calls.
- Widget tests frequently construct `ProviderScope`, `MaterialApp`, localization, and scaffold infrastructure.
- The three device/process journeys already share one integration-test entry point and remain outside the normal widget-test path.
- The repository currently has no Flutter pull-request CI workflow.

Relevant entry points:

- [`justfile`](../justfile)
- [`app/integration_test/critical_journeys_test.dart`](../app/integration_test/critical_journeys_test.dart)
- [`app/test/test_support`](../app/test/test_support)
- [`docs/2026-09-16-flutter-test-redundancy-audit.md`](2026-09-16-flutter-test-redundancy-audit.md)

## Baseline Measurement

Performance work should begin by recording a reproducible baseline. The existing Go release gate retains JSON test output, but neither suite currently publishes a concise timing summary.

### Go Measurements

Record:

- Cold and warm local unit time.
- Full non-race and full race durations.
- Package and top-level test durations from `go test -json`.
- PostgreSQL schema setup and cleanup count and cumulative time.
- Peak PostgreSQL connections and lock pressure.
- Service startup time.
- Static analysis, vulnerability scan, Docker build, migration, memory-probe, and Tap acceptance durations.
- CI retries or known flaky failures.

### Flutter Measurements

Record:

- Cold and warm full-suite wall time.
- Per-file or per-suite duration from the JSON reporter.
- Compilation versus test execution time where available.
- Peak memory.
- Default concurrency compared with `-j 2`, `-j 4`, and the available CPU count.
- Two-way and four-way shard durations.
- Device build/install time compared with integration-journey execution time.

### Evidence Format

Each authoritative CI run should retain:

- Raw machine-readable test output.
- A summary of the slowest packages, files, and tests.
- Phase-level wall-clock durations.
- Runner CPU and memory characteristics.
- Test mode, shard, concurrency, toolchain, and commit identity.

## Go Improvements

### GTP-001 - Limit Package Serialization to Database Test Commands

**Priority:** High
**Expected impact:** Moderate
**Risk:** Low

`GOFLAGS=-p=1` currently surrounds the complete release gate through the `justfile` and CI environment. The individual full test commands already specify `-p 1`.

Remove the global setting while retaining explicit serialization on database-backed `go test` commands. This allows module checks, generation, vet, static analysis, tool compilation, and other host-side Go work to use normal parallelism.

Validation:

- Full release gate remains green.
- Database test commands still report `-p 1`.
- PostgreSQL connection and lock pressure do not increase.

### GTP-002 - Add a Genuine Unit/Integration Boundary

**Priority:** High
**Expected impact:** High for local iteration
**Risk:** Medium

The current unit mode removes integration-service configuration but still compiles all packages and test files. Database and MinIO tests skip only after execution begins, and `-count=1` prevents warm test-cache hits.

Introduce distinct modes:

1. `unit`: cached pure tests for normal local iteration.
2. `unit-fresh`: uncached pure tests for suspicious cache behavior and CI.
3. `integration`: uncached PostgreSQL/MinIO tests without the race detector.
4. `race`: targeted concurrency-sensitive packages for pull requests.
5. `release`: full uncached normal and race evidence.

Use build constraints or package/file separation where necessary. File names ending in `_integration_test.go` do not change Go test selection by themselves.

Do not allow the fast unit path to become the only merge or release gate.

### GTP-003 - Reuse Bootstrap Database Infrastructure

**Priority:** High
**Expected impact:** High
**Risk:** Medium

Every `WithSchema` invocation currently creates and closes a one-connection bootstrap pool in addition to its scoped pool. Reuse one bootstrap pool per test package process and database URL.

The package-scoped bootstrap pool should remain responsible for:

- Global extension initialization.
- Schema creation.
- Schema cleanup.

Each test should retain its own schema and scoped pool by default.

Requirements:

- Pool initialization must be concurrency-safe.
- Cleanup failures must remain visible.
- A package must not leak a bootstrap pool into another database target.
- Test cancellation must not prevent schema cleanup.
- Release tests must prove that all temporary schemas are removed.

### GTP-004 - Reduce Focused-Fixture Round Trips

**Priority:** Medium
**Expected impact:** Low to moderate individually, meaningful at suite scale
**Risk:** Low

Focused schema setup currently performs separate lifecycle-predicate discovery queries. Replace them with one query and, when needed, one combined creation operation.

Measure the cumulative change across all schema-backed tests rather than judging a single setup.

### GTP-005 - Selectively Reuse Expensive Fixtures

**Priority:** Medium
**Expected impact:** High in the largest database packages
**Risk:** Medium to high

Start with suites that repeatedly apply the same large fixture DDL, such as post stores, post indexers, search stores, profiles, timelines, and moderation transitions.

Reuse a schema only where state can be reset deterministically through explicit deletion, `TRUNCATE ... RESTART IDENTITY CASCADE`, or transaction rollback.

Do not share schemas for:

- Lock and concurrency tests.
- Migration or DDL tests.
- Tests that intentionally hold sessions or transactions open.
- Query-plan tests that depend on data distribution or statistics.
- Tests whose cleanup correctness is itself under test.

One-schema-per-test remains the default until a fixture group proves safe.

### GTP-006 - Use Targeted Race Coverage on Pull Requests

**Priority:** High
**Expected impact:** Very high for pull-request wall time
**Risk:** Medium; this is a confidence-policy decision

Recommended pull-request coverage:

- Cached or fresh pure unit tests.
- One uncached full PostgreSQL/MinIO integration run.
- Race tests for concurrency-sensitive packages, initially including auth, ingestion, index, owner lifecycle, scheduled posts, moderation, account deletion, and push delivery.

Recommended trusted release coverage:

- Full uncached normal suite.
- Full uncached race suite.
- Migration reversal and reapplication.
- Binary, image, vulnerability, provenance, memory, startup, and Tap checks.

The full race run must continue to execute on main, a scheduled gate, or the exact tagged release commit. It should not be removed from every trusted path.

### GTP-007 - Shard Database Tests with Isolated Databases

**Priority:** Medium after fixture pressure is reduced
**Expected impact:** Potentially high
**Risk:** Medium to high

Package-level sharding can reduce the critical path, but each shard should use its own PostgreSQL instance or database and its own MinIO resources when applicable.

Do not run aggressively parallel shards against one shared database. That would recreate the shared-lock and connection pressure that currently requires package serialization.

Initial shard candidates:

1. API packages.
2. Index and ingestion packages.
3. Auth, application wiring, and command packages.
4. Owner lifecycle, moderation, and account deletion packages.
5. Remaining stores and service packages.

### GTP-008 - Cache Release Tools and Docker Layers

**Priority:** Medium
**Expected impact:** Moderate, especially on cold CI runners
**Risk:** Low

Cache pinned Staticcheck and govulncheck binaries instead of installing them into a disposable directory on every run. Preserve exact version validation.

Use BuildKit cache import/export so production, audit, and media evidence targets share reusable layers. Cache keys must include the Go version, module graph, Dockerfile, architecture, and relevant source inputs.

Release builds should continue to pull pinned base images as required by policy.

### GTP-009 - Overlap Independent Release-Gate Work

**Priority:** Low to medium
**Expected impact:** Small to moderate
**Risk:** Low

The Go suite requires PostgreSQL and one MinIO boundary but does not require the real Tap container. Start Tap asynchronously while tests run, then await and validate it before release-image smoke and handshake checks.

Static checks that do not mutate the checkout can also run in separate CI jobs. Generation drift checks that modify files must use a separate checkout if run concurrently.

### GTP-010 - Keep Reliability Campaigns off the Pull-Request Critical Path

**Priority:** Existing policy to retain
**Expected impact:** Avoids unnecessary pull-request cost
**Risk:** Low

Keep shuffle, fuzz, and coverage-trend runs scheduled or manual. Ensure fuzz failures are either enforced or explicitly documented as advisory; the current periodic outcome check enforces coverage and shuffle but not fuzz.

## Flutter Improvements

### FTP-001 - Resolve Dependencies Once and Use `--no-pub`

**Priority:** High
**Expected impact:** Low locally, moderate across repeated CI commands
**Risk:** Very low

Use this sequence in CI and full pre-push checks:

```bash
cd app
flutter pub get
flutter analyze --no-pub
flutter test --no-pub
```

Keep a clean-checkout command that resolves dependencies automatically, or clearly document the prerequisite for fast local commands.

### FTP-002 - Cache the Flutter SDK and Pub Packages

**Priority:** High
**Expected impact:** Moderate to high on cold CI runners
**Risk:** Low

Pin an exact Flutter SDK version in CI and cache:

- The Flutter SDK installed by the setup action.
- The global pub cache keyed by operating system, Flutter version, and `app/pubspec.lock`.

Do not initially cache the complete `app/build` directory. It is large and sensitive to SDK, platform, compiler, and source changes.

### FTP-003 - Tune Test Concurrency

**Priority:** High
**Expected impact:** Moderate to high
**Risk:** Low when measured

Benchmark the full suite with:

```bash
flutter test --no-pub -j 2
flutter test --no-pub -j 4
flutter test --no-pub -j 8
```

Choose concurrency independently for local machines and CI runner sizes. Record wall time, peak memory, and total CPU-minutes.

Do not apply concurrency tuning to device integration tests; they execute against one device process.

### FTP-004 - Shard Unit and Widget Tests in CI

**Priority:** High
**Expected impact:** High CI wall-clock reduction
**Risk:** Low to medium

Begin with four built-in shards:

```bash
flutter test --no-pub \
  --total-shards 4 \
  --shard-index "$SHARD_INDEX"
```

Measure shard balance. If one shard consistently dominates because of large files, generate a duration-balanced manifest from retained timing evidence instead of maintaining hand-written file lists.

Account for repeated Flutter setup and compilation when choosing shard count. The lowest wall time may not be the lowest compute cost.

### FTP-005 - Add Explicit Local Feedback Lanes

**Priority:** High
**Expected impact:** High for development iteration
**Risk:** Very low

Provide named recipes for:

- One changed test file with `--no-pub --fail-fast`.
- One feature directory with `--no-pub --fail-fast`.
- Full unit/widget pre-push execution.
- Device integration execution.

Examples:

```bash
flutter test --no-pub --fail-fast test/feed
flutter test --no-pub --fail-fast test/feed/widgets/post_card_test.dart
```

The authoritative CI run should usually avoid `--fail-fast` so it returns complete failure evidence.

### FTP-006 - Reduce Test Compilation Units Carefully

**Priority:** Medium
**Expected impact:** Moderate if suite startup dominates
**Risk:** Medium

The redundancy audit reduced assertion duplication but only reduced test files from 490 to 483. Consolidate tiny sibling files only when they share the same dependency graph, fixture, and ownership boundary.

Do not combine unrelated behavior merely to reduce file count. Do not split large files solely for performance; splitting creates additional compilation units and may make the complete suite slower.

### FTP-007 - Introduce Tiered Widget Harnesses

**Priority:** Medium
**Expected impact:** Moderate in widget-heavy suites
**Risk:** Medium

Provide three widget shells:

1. Bare shell with `Directionality` and optional `MediaQuery`.
2. Localized and themed shell.
3. Full CraftSky shell with providers, messenger, router, and scaffold.

Use the lightest shell that satisfies the widget's inherited dependencies. Keep dedicated full-shell tests for integration between those layers.

Do not reuse one global Riverpod container across tests; fresh containers protect state isolation.

### FTP-008 - Replace High-Volume Broad Settles Causally

**Priority:** Medium
**Expected impact:** Low to moderate for speed, high for determinism
**Risk:** Medium if performed mechanically

Continue replacing `pumpAndSettle()` in the highest-volume suites with:

- Direct provider or notifier future awaits.
- A single `pump()` for synchronous state changes.
- Known-duration pumps for debounces or finite transitions.
- Bounded finder or state waits for asynchronous UI.

Retain `pumpAndSettle()` where complete settlement of a finite animation is the behavior under test.

Virtual clock duration is not equivalent to wall-clock time, so prioritize files based on measured runtime rather than summed pump durations.

### FTP-009 - Narrow Broad Bootstrap Dependencies

**Priority:** Medium after timing evidence
**Expected impact:** Moderate compilation improvement in affected suites
**Risk:** Medium

Many tests import application-wide bootstrap code to initialize mappers. Consider splitting mapper registration into feature-level registrars that production bootstrap composes.

Retain a centralized completeness test proving that production initialization registers the full mapper graph. Incomplete nested registration must fail clearly.

### FTP-010 - Optimize CPU, Memory, and Filesystem Hotspots Selectively

**Priority:** Low
**Expected impact:** Low globally, potentially meaningful in individual shards
**Risk:** Low to medium

Candidates include:

- Cache repeated source-tree scans inside one suite.
- Reuse immutable tiny encoded image fixtures where codec behavior is not under test.
- Avoid repeated multi-megabyte parser allocations when metadata can prove the same rejection boundary, while retaining at least one real size-limit test.

Use timing and memory evidence before changing these tests.

### FTP-011 - Keep Device Integration Focused

**Priority:** Existing policy to retain
**Expected impact:** Avoids expensive redundant native builds
**Risk:** Low

Keep the three critical journeys in one integration-test entry point so one native build and installation can exercise all of them. Preserve fresh per-journey application state unless profiling proves setup is significant relative to build/install time.

Run device journeys for startup, account switching, routing, HTTP wiring, publication, notification runtime, plugin, or mobile-platform changes. Run both supported platforms for mobile release candidates if resources permit.

## Proposed Execution Model

### Local Go

| Command class | Intended use |
|---|---|
| Cached pure unit | Normal edit/test loop |
| Focused package integration | Store, query, or indexer work |
| Targeted race | Concurrency-sensitive changes |
| Full release gate | Pre-push when relevant and release preparation |

### Local Flutter

| Command class | Intended use |
|---|---|
| Changed test file | Normal edit/test loop |
| Feature directory | Feature completion |
| Full unit/widget suite | Pre-push |
| Device critical journeys | Platform and end-to-end boundary changes |

### Pull Requests

Run independent jobs where possible:

1. Go pure unit feedback.
2. Go static and generation checks.
3. One full non-race PostgreSQL/MinIO integration run.
4. Targeted Go race packages.
5. Flutter analysis.
6. Three or four Flutter unit/widget shards.
7. Android critical journeys when app or mobile-platform paths require them.

Use path-aware scheduling to avoid backend-heavy jobs for documentation-only or isolated Flutter changes, and vice versa. Required checks must still report an explicit skipped or successful status.

### Main and Scheduled Reliability

Run:

- Full Go normal and race suites.
- Go shuffle, fuzz, and coverage trends.
- Full sharded Flutter unit/widget suite.
- Flutter analysis.
- Android integration and, when affordable, iOS integration.
- Timing, connection, lock, memory, and flaky-retry summaries.

### Tagged Backend Release

Retain exact-commit evidence for:

- Full uncached Go normal and race suites.
- Exact migration up, down-to-zero, and reapply.
- Migration failure fencing.
- Source and binary vulnerability checks.
- Binary and image provenance.
- Memory probes.
- Release startup and Tap handshake.

Flutter does not need to rerun solely because an AppView deployment tag was created if the same commit already passed the trusted main-branch mobile gate. API compatibility policy may still require it for cross-stack changes.

### Mobile Release Candidate

Run:

- Full Flutter analysis and unit/widget shards.
- Generated-code drift checks.
- Android and iOS critical journeys.
- Release-build validation for the target platforms.

## Implementation Sequence

### Phase 1 - Measurement and Safe Runner Changes

1. Add phase and test-duration summaries for Go and Flutter.
2. Record cold and warm baselines on representative local and CI hardware.
3. Remove global Go `GOFLAGS=-p=1` while preserving explicit test serialization.
4. Resolve Flutter dependencies once and use `--no-pub` afterward.
5. Add Flutter SDK, pub, Go tool, and BuildKit caching.
6. Benchmark Flutter concurrency.

### Phase 2 - Fast Feedback Lanes

1. Add genuine cached Go unit and uncached integration commands.
2. Add changed-file and feature-level Flutter recipes.
3. Add Flutter pull-request CI.
4. Add three or four Flutter test shards.
5. Use targeted Go race packages on pull requests while retaining full trusted race evidence.

### Phase 3 - Fixture and Compilation Optimization

1. Reuse the Go bootstrap database pool per package process.
2. Combine focused lifecycle setup round trips.
3. Optimize the highest-density Go fixture suites one at a time.
4. Add tiered Flutter widget harnesses.
5. Consolidate measured tiny Flutter compilation units.
6. Narrow broad Flutter bootstrap imports where profiling justifies it.

### Phase 4 - Controlled Parallelism

1. Re-measure PostgreSQL connections, locks, and package durations.
2. Evaluate `go test -p=2` only if database pressure permits it.
3. Introduce isolated-database Go shards if needed.
4. Replace built-in Flutter shards with duration-balanced manifests only if imbalance is material.
5. Reassess CI job dependencies and overlap independent release work.

## Success Criteria

The work is successful when:

- Local changed-file and feature feedback is materially faster.
- Pull-request wall time decreases without reducing required behavioral coverage.
- Full tagged-release evidence remains at least as strong as today.
- Go database tests retain deterministic isolation and visible cleanup failures.
- Full race coverage still runs in a trusted gate.
- Flutter device journeys remain separate from the fast unit/widget path.
- No new fixed sleeps or unbounded waits are introduced.
- Test failures remain attributable to the owning behavior rather than shared fixture state.
- Timing reports show sustained improvements over multiple runs, not a single favorable result.
- Increased sharding does not cause unacceptable compute cost or memory instability.

## Deferred or Discouraged Approaches

- Do not remove real PostgreSQL tests in favor of mocks for transactions, locks, constraints, or query plans.
- Do not use one shared database schema for all Go tests.
- Do not run multiple aggressive Go shards against one shared database.
- Do not remove full Go race coverage from every trusted gate.
- Do not reuse global Riverpod containers across Flutter tests.
- Do not replace every `pumpAndSettle()` mechanically.
- Do not split large Flutter files solely to improve readability while claiming a speed gain.
- Do not require Flutter's experimental faster-testing mode without sustained repository-specific evidence.
- Do not add full coverage, shuffle, fuzz, and device campaigns to every pull request.
