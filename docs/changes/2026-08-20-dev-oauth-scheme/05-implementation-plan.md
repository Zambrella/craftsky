# TDD Implementation Plan: Development OAuth Scheme

## Inputs

- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Coding plan: `04-coding-plan.md`

## Implementation Rules

- Do not implement behavior without a linked requirement ID.
- Write or update a failing test before implementation.
- Run the smallest relevant test first.
- Refactor only after tests pass.
- Keep traceability updated.

## Test Order

| Step | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| 1 | BE-001 | DEV-OAUTH-001 | AC-001, AC-003 | Fails: mode and config do not exist. |
| 2 | DB-001 | DEV-OAUTH-001 | AC-002 | Fails: database constraint rejects the mode. |
| 3 | BE-002 | DEV-OAUTH-001, DEV-OAUTH-002 | AC-002, AC-004 | Fails: callback supports only verified links/loopback. |
| 4 | FL-001 | DEV-OAUTH-003 | AC-005 | Fails: client always requests `verified_link`. |
| 5 | PLAT-001 | DEV-OAUTH-004 | AC-006 | Fails: no debug custom scheme is registered. |

## Implementation Steps

### Step 1: BE-001
- Write failing test: `TestLoginDevSchemeRequiresExplicitServerCapability` and `TestLoadConfigDevOAuthSchemeIsExplicitAndDevelopmentOnly`.
- Run command: `go test ./internal/auth -run '^TestLoginDevSchemeRequiresExplicitServerCapability$' -count=1`; then the focused config test.
- Confirmed failure: `HTTPHandlers.AllowDevScheme` and `Config.EnableDevOAuthScheme` were undefined.
- Implement: added the `dev_scheme` mode, an explicit false-by-default AppView capability, production startup rejection, and narrow route composition into `HTTPHandlers`.
- Run command: the two focused tests, followed by the required-PostgreSQL package and race gates recorded below.
- Refactor: kept the server capability at the HTTP boundary and retained the existing OAuth service contract.
- Notes: disabled requests fail before OAuth work; production cannot opt in.

### Step 2: DB-001
- Write failing test: `TestDevOAuthSchemeMigrationUpDownUp` and `TestStoreSavesDevSchemeAuthRequestWithoutClientRedirect`.
- Run command: `TEST_DATABASE_REQUIRED=true TEST_DATABASE_URL=... go test ./internal/db -run '^TestDevOAuthSchemeMigrationUpDownUp$' -count=1`.
- Confirmed failure: migration `000052` did not exist; after the test fixture was added, the old constraint rejected `dev_scheme` metadata.
- Implement: migration `000052` permits `dev_scheme` only with a nonblank device and no client redirect URI; down migration removes ephemeral dev requests before restoring the old constraint.
- Run command: focused migration/store tests and full required-PostgreSQL auth/database packages.
- Refactor: reused the existing auth-request table and lifecycle instead of adding a second callback store.
- Notes: up/down/up and invalid-shape rejection pass.

### Step 3: BE-002
- Write failing test: `TestDevelopmentSchemeCallbacksUseOnlyFixedCodeAndDeletionProofURLs` plus route capability composition coverage.
- Run command: `go test ./internal/auth -run '^TestDevelopmentSchemeCallbacksUseOnlyFixedCodeAndDeletionProofURLs$' -count=1`.
- Confirmed failure: both callback cases rendered HTTP 500 because only verified-link and loopback destinations existed.
- Implement: fixed server-owned `craftsky-dev` URLs for login and account-deletion reauthentication using escaped query values; no client callback URI is accepted.
- Run command: focused callback/route tests and the full backend gates below.
- Refactor: centralized the exact two allowed development paths in `devSchemeCompletionURL`.
- Notes: tests reject `token=` and `Bearer ` in callback markup and verify only code or job/proof parameters.

### Step 4: FL-001
- Write failing test: `oauth_handoff_mode_test.dart` plus the existing auth-client wire test in both compile-time modes.
- Run command: focused `flutter test` without and with `--dart-define=CRAFTSKY_DEV_OAUTH_SCHEME=true`.
- Confirmed failure: the policy file/function did not exist and the client always sent `verified_link`.
- Implement: added a pure policy requiring both `kDebugMode` and the compile-time define; `AuthApiClient.login` sends the selected mode.
- Run command: focused tests in both modes, then `flutter test test/auth test/router/oauth_handoff_route_test.dart`.
- Refactor: isolated compile-time policy from HTTP transport so all four gate combinations are directly unit tested.
- Notes: release/profile selects `verified_link` even if the define is accidentally supplied.

### Step 5: PLAT-001
- Write failing test: expanded `verified_link_configuration_test.dart` to require debug-only Android/iOS registrations and scheme-free production sources.
- Run command: focused platform test.
- Confirmed failure: Android debug had no custom scheme and `Info-Debug.plist` did not exist.
- Implement: added two Android debug intent filters and a Debug-only iOS plist selected only by the Runner Debug configuration.
- Run command: built and inspected Debug and Release APKs and iOS apps as recorded below.
- Refactor: retained the existing Android main manifest, iOS release plist, and associated-domain entitlement unchanged.
- Notes: compiled Debug artifacts contain the scheme; compiled Release artifacts do not.

## Final Verification Evidence

- Backend required PostgreSQL: `TEST_DATABASE_REQUIRED=true TEST_DATABASE_URL=... go test ./internal/auth ./internal/app ./internal/routes ./internal/db -count=1` — PASS (`9.552s`, `3.312s`, `1.097s`, `4.115s`).
- Backend race: the same command with `-race` — PASS (`13.322s`, `5.295s`, `3.953s`, `5.562s`).
- Whole AppView compile: `go test ./... -run '^$' -count=1` — PASS.
- Backend static analysis: `go vet` and pinned Staticcheck 2026.1 over auth/app/routes/db — PASS.
- Flutter focused default mode: handoff policy, auth client, platform configuration, and callback routes — 13 PASS.
- Flutter focused opt-in mode: handoff policy and auth client with `CRAFTSKY_DEV_OAUTH_SCHEME=true` — 7 PASS.
- Flutter auth regression: `flutter test test/auth test/router/oauth_handoff_route_test.dart` — 93 PASS.
- Dart analysis: `dart analyze lib/auth test/auth test/router/oauth_handoff_route_test.dart` — no issues.
- Android artifacts: debug and release APK builds PASS. Compiled manifest inspection found two `craftsky-dev` filters in Debug and zero in Release; both retain the two verified HTTPS filters.
- iOS artifacts: Debug simulator and unsigned Release device builds PASS. Compiled plist inspection found `craftsky-dev` in Debug and no URL-scheme registration in Release.
- Nonblocking existing warning: Flutter reports that project Kotlin `2.2.10` support will be dropped in a future Flutter version and recommends `2.2.20` or newer.

## Completion Checklist

- [x] All Must requirements covered by tests or documented gaps
- [x] All planned Must tests passing
- [x] Relevant regression tests passing
- [x] No unlinked behavior implemented
- [x] Docs updated
- [ ] Review completed or explicitly skipped
