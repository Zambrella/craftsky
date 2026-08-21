# AV-012, AV-013, AV-033, and AV-036 — Verification and release gates

- **Included findings:** AV-012, High — Missing or unreadable migrations are treated as successful migration; AV-013, High — The build contains 19 reachable known vulnerabilities; AV-033, Medium — The default test path skips the database suite, while the DB-enabled suite fails; AV-036, Low — Formatting and static-analysis hygiene is not enforced
- **Priority/order:** Enabling update; land before behavioral remediation is considered verified
- **Status:** Planned
- **Audit sources:** [AV-012](../2026-08-12-appview-code-audit.md#av-012--missing-or-unreadable-migrations-are-treated-as-successful-migration), [AV-013](../2026-08-12-appview-code-audit.md#av-013--the-build-contains-19-reachable-known-vulnerabilities), [AV-033](../2026-08-12-appview-code-audit.md#av-033--the-default-test-path-skips-the-database-suite-while-the-db-enabled-suite-fails), [AV-036](../2026-08-12-appview-code-audit.md#av-036--formatting-and-static-analysis-hygiene-is-not-enforced)

## Shared implementation strategy

Create one reproducible, fail-closed path from migration bundle to release artifact. The same pinned Go toolchain and dependency graph must format/analyze the code, apply and reverse migrations, run unit/PostgreSQL/MinIO/race tests, scan reachable vulnerabilities, and build the binaries/image. A missing prerequisite, skipped database suite, unreadable migration bundle, dirty migration, analyzer finding, or reachable known vulnerability fails the gate rather than producing a misleading green result.

The repository is pre-production, so remove old tests and shortcuts that intentionally treat empty migrations or skipped persistence tests as success. Keep an explicitly named local unit-only command for speed; do not let it be confused with the release gate. Split baseline cleanup into reviewable mechanical/correctness commits, then require the clean baseline for every later update.

## Finding closure

### AV-012 — Missing or unreadable migrations are treated as successful migration

`isMigrationsDirEmpty` currently converts every `os.ReadDir` failure into `true`, and migration commands report success. Compose can therefore start AppView with a stale or absent schema.

AV-012 closes when migration-directory inspection returns `(empty bool, err error)`, missing/unreadable paths are non-zero failures, an empty operational bundle is rejected by default, and Compose/CI prove AppView cannot start behind a falsely successful migration service.

### AV-013 — The build contains 19 reachable known vulnerabilities

The audit found reachable standard-library, pgx, and gRPC advisories. The floating `golang:1.26-alpine` build image also makes the release patch level irreproducible.

AV-013 closes when the release uses at least the audited fixed baselines (Go 1.26.5, pgx 5.9.2, gRPC 1.82.1) or newer verified versions, pins the actual build/runtime image digests, and `govulncheck` reports no reachable vulnerability for source and built release binary. Fixed versions must be rechecked at implementation time rather than assuming this audit remains current.

### AV-033 — The default test path skips the database suite, while the DB-enabled suite fails

Ninety-seven test files use `testdb.WithSchema`; the audit's apparently green commands emitted 484 skips. When PostgreSQL was supplied, the suite exposed a global `pg_trgm` race, parameterized multi-statement SQL rejected by pgx extended protocol, and a timezone-dependent text assertion.

AV-033 closes when CI supplies PostgreSQL 16 and MinIO, zero real-Postgres tests silently skip, the three known harness failures are corrected, migrations pass up/down/up, and the database-backed race suite is required. A local `test-unit` path may still skip integrations, but its name/output must state that limitation.

### AV-036 — Formatting and static-analysis hygiene is not enforced

One hand-written file is not `gofmt`-clean, and staticcheck reports AV-026 plus unused/simplification/deprecated/style diagnostics. No presubmit prevents recurrence.

AV-036 closes when the baseline is clean under documented pinned versions of `gofmt`, `go vet`, and staticcheck, and CI runs check-only commands. Correctness diagnostics are fixed first; package-wide error-string/style normalization follows in separate mechanical commits so it does not obscure behavior changes.

## Scope and design decisions

### In scope

- Migration CLI directory validation and tests.
- Go/tool/dependency/container pinning and vulnerability scanning.
- Required GitHub Actions (or repository CI) with PostgreSQL/MinIO, migration round trips, race tests, and artifact builds.
- The three known DB-test failures, explicit unit-only/full-test commands, format/vet/staticcheck baseline cleanup, and documentation.

### Out of scope

- Fixing unrelated audit behavior under cover of the quality cleanup.
- Treating image/SBOM scanning as a substitute for Go call-graph scanning.
- Requiring developers to keep Compose running for the explicitly unit-only target.
- Suppressing reachable advisories without a reviewed, expiring risk decision.

### Decisions

1. Introduce a single `just appview-check` (name may follow repository convention) that is the local equivalent of CI; it requires PostgreSQL/MinIO and runs all release checks.
2. Keep `just appview-test-unit` explicitly incomplete. `testdb.WithSchema` fails when `TEST_DATABASE_REQUIRED=true` and no URL is present; CI always sets that flag.
3. Parse `go test -json` or otherwise assert zero `skipping real-pg test` events in the full job so a future helper cannot silently bypass the fail-closed flag.
4. Pin Go at the patch level in `go.mod`/toolchain policy and by immutable Docker image digest. Build and scan with that same toolchain.
5. Pin staticcheck and govulncheck versions in repository commands/workflow. Upgrade those tools deliberately and regularly.
6. Reject an empty migration directory in operational CLI commands. If a low-level test needs an empty source, it uses an explicit test-only option/helper rather than the production command shortcut.
7. Make `staticcheck ./...` the target after baseline cleanup. If any check is intentionally excluded, document the exact code, reason, owner, and removal condition rather than using a broad wildcard.

## Unified implementation plan

1. Refactor `appview/cmd/cli/migrate.go`: replace `isMigrationsDirEmpty` with a directory inspector that distinguishes missing, unreadable, empty, and populated states and returns wrapped errors with the path/operation.
2. Update `up`, `down`, `status`, `redo`, and their `runMigrate*` helpers to fail before database startup on missing/unreadable/empty operational bundles. Remove the old AC/comment that defines empty-directory success.
3. Expand `appview/cmd/cli/migrate_test.go` with populated, empty, missing, non-directory, and unreadable cases. Run unreadable permission tests only where the platform can enforce permissions; keep a deterministic non-directory case everywhere.
4. Add a Compose/startup integration test or script that removes/mispoints the migration bundle and asserts the migrate service fails and AppView remains blocked.
5. Pin the release toolchain to at least Go 1.26.5 (or the then-current fixed patch), update `appview/go.mod`, and pin the build/runtime images in `appview/Dockerfile` by full tag and digest. Record a deliberate image-refresh process.
6. Upgrade `github.com/jackc/pgx/v5` to at least 5.9.2 and resolve `google.golang.org/grpc` to at least 1.82.1, then run `go mod tidy` and review the full module diff. Re-run `govulncheck` because transitive reachability may change.
7. Add pinned repository commands for staticcheck and govulncheck. Run source-mode scanning on `./...` and binary-mode scanning on the exact built AppView/CLI artifacts; optionally add container/SBOM scanning as an additional gate.
8. Fix the database-suite blockers:
   - Provision `pg_trgm` once in test-database bootstrap under serialization/advisory locking rather than racing from parallel schema tests.
   - Split the parameterized multi-statement fixture in `accountdeletion/worker_acceptance_test.go` into separate `Exec` calls.
   - Scan/compare `time.Time` in `db/profile_pins_migration_test.go` (UTC-normalized) rather than asserting timezone-sensitive text.
9. Change `internal/testdb.WithSchema` to fail when the explicit full-suite flag is set without a database URL, while retaining the clearly named unit-only skip behavior.
10. Add CI workflow jobs using PostgreSQL 16 and the pinned MinIO release with health checks and isolated credentials. Bootstrap extensions, export all `TEST_*` settings, and wait for services deterministically.
11. Add a migration job that builds the CLI, applies all migrations, verifies current/non-dirty version, rolls down to zero in an isolated database, reapplies, and verifies the same final version. Missing bundle or dirty state fails.
12. Add test jobs for unit/full PostgreSQL+MinIO tests and `go test -race ./...`; save JSON/JUnit/log artifacts on failure without exporting secrets.
13. Establish the hygiene baseline: run `gofmt` on `internal/api/scheduled_post_response.go`, fix AV-026/other correctness diagnostics, remove genuine unused code, update deprecated test APIs, then normalize style/error strings package by package in separate reviewable commits.
14. Add check-only CI steps for `gofmt -l`, `go vet ./...`, pinned `staticcheck ./...`, `go mod verify`, and clean generated/module diffs. Formatting in CI never rewrites the checkout.
15. Build the production binaries/image only after all earlier jobs pass. Report toolchain/module/image identities in build metadata so the scanned artifact is the released artifact.
16. Document the two local workflows in `appview/README.md`/`justfile`: fast unit-only versus release-equivalent full check, including required services and the fact that only the latter is valid PR/release evidence.

## Migration, reconciliation, and operations plan

No application schema change is intrinsically required, but the gate exercises the entire migration history destructively in an isolated CI database. Never run the down-to-zero job against a developer/shared database.

Before merging:

1. Rebuild a clean local PostgreSQL/MinIO environment.
2. Run up/down/up with the patched CLI and confirm missing/unreadable/empty bundles fail non-zero.
3. Run the full database race suite with zero real-DB skips.
4. Build and scan artifacts using the pinned release toolchain/image.

Release operations pin image digests, publish vulnerability/toolchain evidence, and schedule regular dependency/tool refreshes. A later advisory failing the gate blocks release until upgraded or covered by a time-bounded reviewed exception; silent `|| true` and blanket ignores are prohibited.

## API, client, configuration, and operational impact

- No AppView HTTP success/error envelope or Flutter contract changes.
- Migration CLI behavior intentionally breaks: missing, unreadable, and empty operational bundles now exit non-zero instead of printing success.
- CI, `justfile`, Docker, Go module/toolchain, and environment-test settings change together; the full test target explicitly requires disposable PostgreSQL and MinIO.
- Release artifacts become reproducible by toolchain/module/image identity, and releases stop on analyzer, migration, test, or vulnerability failure.
- Developer documentation distinguishes the incomplete unit-only command from acceptable pull-request/release evidence.

## Security, failure, and race considerations

- Never print database URLs, S3 credentials, module proxy credentials, or vulnerability scan environment secrets in CI artifacts.
- CI services bind only within the job network and use disposable databases/buckets.
- Cache keys include toolchain and dependency hashes; cached binaries never override pinned tool versions.
- Migration failure must propagate through Compose service health/dependency semantics.
- Parallel tests must not mutate cluster-global extensions without serialization.
- Source scanning and binary scanning are complementary: source is conservative; binary verifies the built call graph. Scan failure is not waived merely because an exploit is not reproduced.
- Formatting/style cleanup is behavior-neutral and separately reviewable; run full tests after each correctness-bearing analyzer fix.

## Unified test and gate plan

The required gate, in order, is:

1. Verify pinned Go/tool versions, `go mod verify`, and clean module metadata.
2. Fail if `gofmt -l` reports any hand-written Go file.
3. Run `go vet ./...` and pinned `staticcheck ./...`.
4. Run focused migration-directory unit tests, including all error states.
5. Start PostgreSQL 16/MinIO, bootstrap extensions/bucket, and assert service health.
6. Run CLI migrations up/down/up in an isolated database and assert final version/clean state.
7. Run full `go test -json ./...` with database-required mode and assert zero real-DB skips.
8. Run `go test -race ./...` with the same services/settings.
9. Build AppView and CLI with the pinned release flags/toolchain.
10. Run pinned `govulncheck` against source and built binaries; fail on reachable findings.
11. Build the digest-pinned container and run a startup smoke test against the fully migrated database.

## Traceability and acceptance criteria

### AV-012

- **Implementation seams:** `appview/cmd/cli/migrate.go`, migration CLI tests, Compose migration dependency.

- [ ] Missing, unreadable, non-directory, and empty migration sources fail non-zero.
- [ ] A genuinely populated source proceeds normally.
- [ ] Every migration subcommand uses the same fail-closed inspector.
- [ ] Compose does not start AppView after migration-bundle failure.
- [ ] Error output names the operation/path without leaking credentials.

### AV-013

- **Implementation seams:** `appview/go.mod`, `appview/go.sum`, `appview/Dockerfile`, pinned scan/build workflow.

- [ ] Go, pgx, and gRPC meet or exceed currently verified fixed versions.
- [ ] Build/runtime image tags and digests are reproducible.
- [ ] Source and exact built binaries have no reachable govulncheck findings.
- [ ] Dependency upgrades pass unit, integration, race, migration, and startup smoke tests.
- [ ] Toolchain/dependency/image identities are recorded with the artifact.

### AV-033

- **Implementation seams:** `internal/testdb`, the three named failing tests, `justfile`, and CI services/workflow.

- [ ] Full CI supplies PostgreSQL 16 and MinIO and fails if either is unavailable.
- [ ] Full CI has zero real-Postgres skip events.
- [ ] `pg_trgm` setup is serialized, parameterized statements are split, and timestamp assertions are timezone-independent.
- [ ] Migration up/down/up and the full database-backed race suite pass deterministically.
- [ ] The unit-only target is clearly named and cannot be presented as release evidence.

### AV-036

- **Implementation seams:** hand-written Go sources, pinned analyzer commands, `justfile`, and CI workflow.

- [ ] `gofmt -l` is empty.
- [ ] `go vet ./...` passes.
- [ ] The documented pinned staticcheck set passes with no AV-026 `SA4006` or unreviewed suppression.
- [ ] Correctness and mechanical style changes are separated for review.
- [ ] Every pull request is blocked on the check-only hygiene steps.

## Dependencies and coordination

- This grouped update is the verification foundation for every other AV plan.
- AV-026 supplies the first high-signal staticcheck correctness fix.
- AV-028/AV-034 and AV-029 require migration-number coordination and the up/down/up gate.
- AV-001/AV-017 network changes and AV-013 dependency upgrades should not be merged without rerunning the exact release artifact scan.
- AV-037 moves must retain the same gate and should land only after this baseline is clean.

## References

- [AppView development workflow](../../appview/README.md)
- [Pull request template](../../.github/pull_request_template.md)
- [Google Go Style Guide](https://google.github.io/styleguide/go/guide)
- [Go vulnerability management](https://go.dev/doc/security/vuln/)
- [Go vulnerability database](https://pkg.go.dev/vuln/)
- [pgx advisory GO-2026-5004](https://pkg.go.dev/vuln/GO-2026-5004)
- [gRPC advisory GO-2026-6061](https://pkg.go.dev/vuln/GO-2026-6061)
