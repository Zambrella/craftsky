# TDD Implementation Plan: Flutter Error Handling And Sentry Reporting

## Inputs
- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md`
- Coding plan: `04-coding-plan.md`

## Implementation Rules
- Do not implement behavior without a linked requirement ID.
- Write or update a failing test before implementation.
- Run the smallest relevant test first.
- Refactor only after tests pass.
- Keep traceability updated.
- Keep Sentry SDK imports behind `app/lib/shared/observability/`.
- Do not enable Sentry tracing, profiling, metrics, session replay, broad user correlation, or `sentry_dio` in this slice.

## Test Order
| Step | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| 1 | UT-001, UT-012 | FR-001, FR-016, NFR-001 | AC-001, AC-010, AC-014, AC-019 | Fails because Sentry config/options do not exist |
| 2 | UT-011 | FR-004, FR-005, FR-006, FR-019, NFR-002 | AC-010, AC-016, AC-022 | Fails because reporter interface/result types do not exist |
| 3 | UT-002 | BR-003, FR-011 | AC-004, AC-007, AC-021 | Fails because finite app error taxonomy does not exist |
| 4 | UT-003, UT-007 | FR-007, FR-012, RULE-003 | AC-004, AC-012, AC-021 | Fails because mapper/reportability classifier do not exist |
| 5 | UT-004, IT-008 | BR-002, FR-014, FR-015, RULE-001 | AC-005, AC-006 | Fails because shared safe copy/l10n keys do not exist |
| 6 | UT-005, UT-008 | FR-005, FR-006, FR-020, RULE-002, RULE-005 | AC-011, AC-020 | Fails because Sentry sanitizer and breadcrumbs do not exist |
| 7 | UT-006, IT-003 | FR-004, RULE-002, RULE-003 | AC-003 | Fails because log bridge does not exist |
| 8 | IT-004 | FR-006, FR-007, FR-012, RULE-002, RULE-003 | AC-011, AC-012 | Fails because ApiException lacks safe diagnostics |
| 9 | UT-010, IT-005 | FR-008, FR-009, FR-021 | AC-008, AC-013, AC-016 | Fails because retry helper and reporter-aware ProviderLogger do not exist |
| 10 | AT-001, IT-001 | FR-001, NFR-001, NFR-002 | AC-001, AC-010, AC-019 | Fails because startup bootstrap is not injectable/reporting-aware |
| 11 | AT-002, IT-002, REG-009 | FR-002, FR-003 | AC-002, AC-010 | Fails because handlers only log locally |
| 12 | REG-001, REG-002, REG-003, REG-004, REG-005 | FR-013, FR-014, FR-015, RULE-001 | AC-005, AC-006 | Fails where raw error strings still render |
| 13 | AT-004, UT-009 | FR-018 | AC-018 | Fails because support reference model/UI does not exist |
| 14 | IT-006, REG-010 | FR-021, NFR-002 | AC-008, AC-016 | Fails because app-owned test harness is missing |
| 15 | REG-006, REG-007, REG-008, REG-011, IT-007 | FR-017, FR-019, RULE-004, RULE-005 | AC-014, AC-015, AC-017, AC-022 | Fails because static guards and release docs are missing |

## Implementation Steps

### Step 1: UT-001, UT-012
- Write failing test: `app/test/observability/sentry_options_test.dart`
- Run command: `cd app && flutter test test/observability/sentry_options_test.dart`
- Confirmed failure: Missing `shared/observability/sentry_config.dart` and `SentryConfig`.
- Implement: Added pure-Dart `SentryConfig` and `SentryFeatureOptions` with DSN/environment/local-opt-in rules, release/dist preservation, logs enabled, PII disabled, and tracing/profiling/metrics/session replay disabled.
- Run command: `cd app && flutter test test/observability/sentry_options_test.dart` passed.
- Refactor: None.
- Notes: This step does not initialize the Sentry SDK yet; it creates the app-owned option policy used by later startup wiring.

### Step 2: UT-011
- Write failing test: `app/test/observability/error_reporter_test.dart`
- Run command: `cd app && flutter test test/observability/error_reporter_test.dart`
- Confirmed failure: Missing `shared/observability/error_reporter.dart` and reporter contract types.
- Implement: Added `ErrorReporter`, `NoopErrorReporter`, `GuardedErrorReporter`, `ReportContext`, `SafeBreadcrumb`, `ReportStatus`, and `ReportResult`.
- Run command: `cd app && flutter test test/observability/error_reporter_test.dart` passed.
- Refactor: None.
- Notes: Reporter failures are contained so observability cannot interrupt app startup, logging, or UI work.

### Step 3: UT-002
- Write failing test: `app/test/shared/errors/app_error_taxonomy_test.dart`
- Run command: `cd app && flutter test test/shared/errors/app_error_taxonomy_test.dart`
- Confirmed failure: Missing `shared/errors/app_error.dart` and taxonomy types.
- Implement: Added finite `AppErrorKind`, severity, surface, action policy, metadata, and `AppError` value type.
- Run command: `cd app && flutter test test/shared/errors/app_error_taxonomy_test.dart` passed.
- Refactor: None.
- Notes: Default reportability is conservative for routine network/auth/content states and enabled for defect/degraded-state fallbacks.

### Step 4: UT-003, UT-007
- Write failing test: `app/test/shared/errors/app_error_mapper_test.dart`
- Run command: `cd app && flutter test test/shared/errors/app_error_mapper_test.dart`
- Confirmed failure: Missing mapper and reportability classifier modules.
- Implement: Added `AppErrorMapper`, `AppErrorSource`, `ReportabilityClassifier`, and per-instance reportability/classification overrides on `AppError`.
- Run command: `cd app && flutter test test/shared/errors/app_error_mapper_test.dart test/shared/errors/reportability_classifier_test.dart` passed.
- Refactor: None.
- Notes: Mapper diagnostics are allowlisted and do not carry raw exception text; richer API diagnostics will be added in the API interceptor loop.

### Step 5: UT-004, IT-008
- Write failing test: `app/test/shared/errors/app_error_copy_test.dart` and `app/test/l10n/error_l10n_test.dart`
- Run command: `cd app && flutter test test/shared/errors/app_error_copy_test.dart test/l10n/error_l10n_test.dart`
- Confirmed failure: Missing `AppErrorPresenter` and generated l10n getters for new error messages/action label.
- Implement: Added ARB keys for every finite error case and sign-in action, regenerated l10n, and added `AppErrorPresenter`.
- Run command: `cd app && flutter test test/shared/errors/app_error_copy_test.dart test/l10n/error_l10n_test.dart` passed.
- Refactor: None.
- Notes: Presenter ignores diagnostics entirely, so raw exception/backend/request/token strings cannot enter user copy through this path.

### Step 6: UT-005, UT-008
- Write failing test: `app/test/shared/errors/sentry_redaction_test.dart` and `app/test/shared/errors/sentry_breadcrumb_test.dart`
- Run command: `cd app && flutter test test/shared/errors/sentry_redaction_test.dart test/shared/errors/sentry_breadcrumb_test.dart`
- Confirmed failure: Missing `SentrySanitizer` module.
- Implement: Added context and breadcrumb allowlist sanitizer and breadcrumb value equality for tests.
- Run command: `cd app && flutter test test/shared/errors/sentry_redaction_test.dart test/shared/errors/sentry_breadcrumb_test.dart` passed.
- Refactor: None.
- Notes: Sanitizer drops unknown payloads and sensitive-looking values even when a key is otherwise allowlisted.

### Step 7: UT-006, IT-003
- Write failing test: `app/test/observability/log_bridge_test.dart`
- Run command: `cd app && flutter test test/observability/log_bridge_test.dart`
- Confirmed failure: Missing `LogForwarder`.
- Implement: Added `LogForwarder` with severe/shout forwarding and explicit promoted-warning forwarding.
- Run command: `cd app && flutter test test/observability/log_bridge_test.dart` passed.
- Refactor: None.
- Notes: The bridge sends logger/category metadata only, not raw log messages as Sentry context.

### Step 8: IT-004
- Write failing test: `app/test/shared/api/providers/error_mapping_interceptor_test.dart`
- Run command: `cd app && flutter test test/shared/api/providers/error_mapping_interceptor_test.dart`
- Confirmed failure: `ApiException` lacked safe diagnostics and the interceptor discarded AppView request/error/status/endpoint context.
- Implement: Added `ApiFailureDetails`, populated it in `ErrorMappingInterceptor`, normalized endpoint categories, and included safe API details in `AppErrorMapper` diagnostics.
- Run command: `cd app && flutter test test/shared/api/providers/error_mapping_interceptor_test.dart test/shared/errors/app_error_mapper_test.dart` passed.
- Refactor: None.
- Notes: AppView `message`, response bodies, query strings, and raw URLs are still not stored in the exception details.

### Step 9: UT-010, IT-005
- Write failing test: `app/test/shared/riverpod/retry_policy_test.dart` and `app/test/bootstrap/provider_logger_test.dart`
- Run command: `cd app && flutter test test/shared/riverpod/retry_policy_test.dart test/bootstrap/provider_logger_test.dart`
- Confirmed failure: Missing `appProviderRetry` and reporter injection on `ProviderLogger`.
- Implement: Added `appProviderRetry`, applied it to production/startup provider scopes, and routed reportable provider failures through `ErrorReporter`.
- Run command: `cd app && flutter test test/shared/riverpod/retry_policy_test.dart test/bootstrap/provider_logger_test.dart` passed.
- Refactor: None.
- Notes: Expected provider failures such as transient network/API states remain local-only.

### Step 10: AT-001, IT-001
- Write failing test: `app/test/observability/sentry_bootstrap_test.dart`
- Run command: `cd app && flutter test test/observability/sentry_bootstrap_test.dart`
- Confirmed failure: Missing app-owned observability bootstrap boundary.
- Implement: Added `ObservabilityBootstrap`, `SentryBootstrapAdapter`, `SentryFlutterBootstrapAdapter`, and `SentryErrorReporter`; installed `sentry_flutter` 9.23.0; wired `main.dart` to initialize reporting before `bootstrap()`.
- Run command: `cd app && flutter test test/observability/sentry_bootstrap_test.dart test/observability/sentry_options_test.dart` passed.
- Refactor: None.
- Notes: Sentry SDK imports are currently confined to `shared/observability/sentry_error_reporter.dart`.

### Step 11: AT-002, IT-002, REG-009
- Write failing test: `app/test/observability/error_handlers_test.dart`
- Run command: `cd app && flutter test test/observability/error_handlers_test.dart`
- Confirmed failure: Error handlers logged locally but did not capture through the reporter.
- Implement: Added reporter capture for Flutter framework, platform, and root-zone errors while preserving existing local logging and debug presentation.
- Run command: `cd app && flutter test test/observability/error_handlers_test.dart` passed.
- Refactor: None.
- Notes: `FlutterError.presentError` still runs, so debug diagnostics remain visible.

### Step 12: REG-001, REG-002, REG-003, REG-004, REG-005
- Write failing tests: update existing widget tests for initialization, router, settings cache-clear, profile projects, and l10n-preserved surfaces.
- Run command: focused `flutter test` targets for each touched surface.
- Confirmed failure: Initialization, router, settings cache-clear, and profile projects still rendered raw exception text.
- Implement: Switched those surfaces to `AppErrorMapper` and `AppErrorPresenter`; added router error regression coverage.
- Run command: `cd app && flutter test test/app_test.dart test/router/router_error_screen_test.dart test/settings/clear_image_cache_tile_test.dart test/profile/widgets/profile_projects_tab_test.dart` passed.
- Refactor: None.
- Notes: Local logs still include diagnostic errors for developers; user-facing widgets show localized safe copy only.

### Step 13: AT-004, UT-009
- Write failing test: `app/test/shared/errors/support_reference_test.dart`
- Run command: `cd app && flutter test test/shared/errors/support_reference_test.dart`
- Confirmed failure: Missing support-reference model and localized formatter.
- Implement: Added `SupportReference`, `supportReferenceLabel` l10n key, and generated localization updates.
- Run command: `cd app && flutter test test/shared/errors/support_reference_test.dart` passed.
- Refactor: None.
- Notes: Empty/disabled/failed captures and Sentry empty IDs do not produce fake references.

### Step 14: IT-006, REG-010
- Write failing test: `app/test/test_support/app_harness_test.dart`
- Run command: `cd app && flutter test test/test_support/app_harness_test.dart`
- Confirmed failure: Missing app test harness and reporter provider.
- Implement: Added `errorReporterProvider`, `appHarness()`, and harness tests for no-op reporter override and retry disabled.
- Run command: `cd app && flutter test test/test_support/app_harness_test.dart` passed.
- Refactor: None.
- Notes: Existing tests can migrate to the harness incrementally; production scopes already use the same retry callback.

### Step 15: REG-006, REG-007, REG-008, REG-011, IT-007
- Write failing tests: static import-boundary, secret-scan, disabled-feature, and release-symbolication checks.
- Run command: `cd app && flutter test test/observability/import_boundary_test.dart test/observability/secret_scan_test.dart test/observability/sentry_release_config_test.dart`
- Confirmed failure: Release-symbolication plugin config/docs were missing.
- Implement: Added static import-boundary, secret-scan, and release config tests; installed `sentry_dart_plugin`; added secret-free Sentry plugin settings and `release-symbolication.md`.
- Run command: `cd app && flutter test test/observability/import_boundary_test.dart test/observability/secret_scan_test.dart test/observability/sentry_release_config_test.dart test/observability/sentry_options_test.dart` passed.
- Refactor: None.
- Notes: Upload auth tokens remain environment-only and are not represented in `pubspec.yaml`.

## Completion Checklist
- [x] All Must requirements covered by tests or documented gaps
- [x] All planned Must tests passing
- [x] Relevant regression tests passing
- [x] No unlinked behavior implemented
- [x] Docs updated
- [x] Review completed or explicitly skipped

## Final Verification
- `cd app && dart analyze` passed with no issues.
- `cd app && flutter test test/settings/settings_page_test.dart test/settings/clear_image_cache_tile_test.dart --reporter=compact` passed after updating the stale localization wrapper in `settings_page_test.dart`.
- `cd app && flutter test --reporter=compact` passed: 759 tests.
- Post-review focused regression run passed: `cd app && flutter test test/observability/log_bridge_test.dart test/observability/sentry_error_reporter_test.dart test/bootstrap/provider_logger_test.dart test/shared/api/providers/error_mapping_interceptor_test.dart test/profile/profile_page_test.dart --reporter=compact`.
- Post-review `cd app && dart analyze` passed with no issues.
- Post-review `cd app && flutter test --reporter=compact` passed: 766 tests.
- `git status --short` reviewed. The pre-existing `docs/roadmap.md` change remains unrelated and was not modified as part of this implementation.

## Handoff Notes
- Sentry runtime imports are confined to `app/lib/shared/observability/sentry_error_reporter.dart`.
- `sentry_flutter` generated desktop registrant updates and an iOS SwiftPM `Package.resolved` pin for `sentry-cocoa`.
- Implementation review findings IR-001 through IR-006 were addressed in follow-up changes and recorded in `06-implementation-review.md`.
