# Validation evidence

## Passed locally

- Repository-pinned Flutter SDK 3.47.7: full Flutter suite, **3,104 passed / 38 existing skips**. Subsequent diagnostics-only refinements were checked by affected suites rather than repeating the full suite.
- Final affected observability, API mutation-contract and mutation-controller suites: **133 passed / one existing skip**. Checks include keeping mutation UI mapping and terminal reporting unchanged while retaining original causes and HTTP/request correlation in actual serialized Sentry envelopes.
- Final serialized-SDK regressions after the analyzer type refinement: **five passed**; static explanations survive, protected-value canaries do not, and native SDK exception chains retain both format and exhausted-mutation causes.
- Onboarding/framework follow-up regression run: replaced leases remain cancellation; closed framework-stage evidence survives protected local output. The affected-suite run including onboarding passed before the final diagnostics-only cause-extractor refinement.
- Android path-picker tests: **three passed** (cancellation, successful parsing and cache cleanup, parsing failure and cache cleanup).
- Flutter analysis: **no issues**.
- Android `:app:compileDebugKotlin`: **BUILD SUCCESSFUL**, including the new streaming picker. Used the installed Gradle 9.3.1 distribution because this checkout had no generated `gradlew` script. Existing Kotlin Gradle Plugin deprecation warnings remain.
- Go full suite against isolated local PostgreSQL and MinIO: **6,300 passed, zero failed/skipped test cases**. Six packages have no test files; these are not skipped test cases.
- Go race tests: `internal/api`, `internal/auth`, `internal/observability` all passed against the same real dependencies. Temporary test containers and volumes were cleaned up. The combined runner initially returned nonzero because its cleanup hook resolved a relative Compose path after changing directories; both test commands passed. Cleanup was rerun successfully from the repository root for the exact temporary project.
- `git diff --check`: passed.
- Final Sentry inventory: **25 unresolved**, no pagination remaining. Nine original issues are resolved using the next-release setting; their saved activities contain an empty future target and the existing semantic-version anchor. No other issues were resolved, archived, merged or deleted.

## Initial release gate limitation

`GOFLAGS='-p=1' ./scripts/appview-check` passed module verification, code-generation comparison, formatting/vet/static analysis stages, then stopped at the source vulnerability scan:

- **GO-2026-6809**, reachable standard-library `crypto/x509` issue.
- Found in the repository-pinned **Go 1.27.2**, fixed in **Go 1.27.3**.
- This toolchain finding is independent of the Sentry changes. The toolchain, gate and vulnerability policy were left unchanged. The full Go suite and affected race suites were run separately; this does **not** constitute a passing release gate.
- Docker-image builds, image-level vulnerability scans, migration/media acceptance and production smoke stages after the failed scan were not reached in this gate invocation.

The first Flutter widget run lacked checkout-local Rive libraries and failed in native symbol lookup. Running `fvm dart run rive_native:setup --clean --platform macos` supplied the host test library; the full rerun passed. No native macOS app target or platform folder was created.

## Not performed

- No commits, push, release tags, app distribution, backend deployment or production infrastructure/configuration/data mutation.
- No physical-device import/OOM reproduction, native process-death test, live Sentry delivery/symbolication proof or production recurrence validation.
- No synthetic failures were injected into production.
- Raw Sentry event exports, authentication state and private export files were not added to the repository.

The six newly fixed issues remain open until their uncommitted local fixes are integrated into a release. The other 19 open issues retain the evidence/disposition recorded in [triage.md](triage.md).


## Riverpod ownership follow-up

The subsequent audit added operation-scoped retention for like/repost/follow, retained the resource-owning mute/unmute repository, guarded onboarding invalidation, and removed deferred provider initialization from widget builds. Five new regressions demonstrated the original defects before the fixes. The affected **400-test** suite passed, including read-provider disposal, mutation completion after listener removal, invalidation, avatar lifetime and relationship rollback/rendering coverage. The earlier full Flutter suite predates these follow-up changes; it was not rerun for this follow-up.

Final follow-up checks: `fvm flutter analyze --no-pub` reported **no issues**; `git diff --check` passed.


## PR #208 release gate repair

Reproduced the unchanged `AppView release gate` locally before editing:
`GOFLAGS=-p=1 APPVIEW_CHECK_ARTIFACT_DIR=/tmp/craftsky-pr208-reproduce ./scripts/appview-check`
exited 3 at the source vulnerability scan with the same reachable
`GO-2026-6809` / `crypto/x509@go1.27.2` finding as GitHub run `39621479159`.
The aggregate `PR checks` job failed solely because `APPVIEW_RESULT=failure`;
its other applicable job results were successful.

Updated Go 1.27.2 to 1.27.3 consistently in the CI workflow, both local test/gate
scripts, AppView and tools modules, and the Docker build stage. The official
`golang:1.27.3-alpine` image is pinned to index digest
`sha256:f2c872bcee949b7be9c490efd02b48a8a2fbf947cdf533f1c58525ec9d8170fd`.
No application behavior, dependency versions, or vulnerability policy changed.

Validation after the fix:

- Full `GOFLAGS=-p=1 APPVIEW_CHECK_ARTIFACT_DIR=/tmp/craftsky-pr208-fixed ./scripts/appview-check`: **passed**, exit 0.
- Normal Go suite: **6,300 passed**, no failed or skipped test cases.
- Full Go race suite: **6,300 passed**, no failed or skipped test cases.
- Module/code-generation/format/vet/Staticcheck checks passed. Source scan reports zero reachable vulnerabilities; the existing unused-module advisory remains non-reachable.
- Release and evidence images, binary vulnerability/provenance checks, migration rollback/reapply and failure handling, media probes, startup/console boundaries, and pinned Tap handshake passed.
- Temporary gate containers, volumes, and images were cleaned up by the gate.
- CI-pinned actionlint 1.7.7, shell syntax checks, and `git diff --check`: passed.

This supersedes the initial toolchain-related gate limitation above. Validation
is local; no commit, push, release, or deployment was performed for this repair.
