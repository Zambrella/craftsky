# Acceptance Test Specification: Flutter Error Handling And Sentry Reporting

## 1. Test Strategy

This feature is medium risk because it touches app startup, global error handlers, Riverpod observers, API error mapping, localization, privacy filtering, and user-visible fallback UI. The test design therefore uses layered verification:

- Unit tests for the finite error taxonomy, technical-to-user mapping, reportability classifier, Sentry option construction, redaction allowlists, breadcrumb filtering, support-reference formatting, and Riverpod retry helpers.
- Widget tests for user-visible safe error copy, retry affordances, copyable support references, and existing `AppMessenger` surfaces.
- Integration-style Flutter tests with fake/no-op Sentry reporters or transports for startup behavior, error-handler wiring, log forwarding, provider failure reporting, and Dio/AppView error reporting without real credentials or network calls.
- Regression/static checks for raw error leakage, direct Sentry imports outside the reporting package, committed secrets, Sentry features that must stay disabled, l10n coverage, and release symbolication documentation.
- Manual checks only for behavior that is impractical to prove in local Flutter tests, mainly real staging Sentry delivery and native crash symbolication.

Primary command discovered: `cd app && flutter test`. Root `just test` is Go/AppView-only and does not run Flutter tests.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-002, AC-003 | AT-001, AT-002, AT-006, UT-001, UT-006, IT-001, IT-002, IT-003, MAN-001 | Acceptance / Unit / Integration / Manual | Mostly |
| BR-002 | AC-004, AC-005, AC-006 | AT-003, UT-002, UT-003, UT-004, IT-008, REG-001, REG-002, REG-003, REG-004 | Acceptance / Unit / Integration / Regression | Yes |
| BR-003 | AC-004, AC-007 | AT-003, UT-002, UT-003 | Acceptance / Unit | Yes |
| BR-004 | AC-008, AC-009 | AT-005, UT-010, IT-005, REG-001, REG-010 | Acceptance / Unit / Integration / Regression | Yes |
| FR-001 | AC-001, AC-010, AC-019 | AT-001, UT-001, UT-011, IT-001 | Acceptance / Unit / Integration | Yes |
| FR-002 | AC-002 | AT-002, IT-002, MAN-002 | Acceptance / Integration / Manual | Mostly |
| FR-003 | AC-002, AC-010 | AT-002, UT-011, IT-002, REG-009 | Acceptance / Unit / Integration / Regression | Yes |
| FR-004 | AC-003 | AT-006, UT-006, UT-007, IT-003 | Acceptance / Unit / Integration | Yes |
| FR-005 | AC-011 | AT-006, UT-005, IT-004, REG-007 | Acceptance / Unit / Integration / Regression | Yes |
| FR-006 | AC-003, AC-011, AC-012, AC-020 | AT-006, AT-007, UT-005, UT-008, IT-004 | Acceptance / Unit / Integration | Yes |
| FR-007 | AC-012 | AT-006, UT-003, UT-007, IT-004 | Acceptance / Unit / Integration | Yes |
| FR-008 | AC-013 | AT-006, UT-007, IT-005 | Acceptance / Unit / Integration | Yes |
| FR-009 | AC-008 | AT-005, UT-010, IT-005, REG-010 | Acceptance / Unit / Integration / Regression | Yes |
| FR-010 | AC-009 | AT-005, REG-001 | Acceptance / Regression | Yes |
| FR-011 | AC-004, AC-007, AC-021 | AT-003, UT-002, UT-004 | Acceptance / Unit | Yes |
| FR-012 | AC-004, AC-005, AC-012, AC-021 | AT-003, AT-006, UT-003, UT-004, IT-004 | Acceptance / Unit / Integration | Yes |
| FR-013 | AC-005, AC-006 | AT-003, REG-001, REG-002, REG-003, REG-004 | Acceptance / Regression | Yes |
| FR-014 | AC-006 | AT-003, IT-008, REG-005 | Acceptance / Integration / Regression | Yes |
| FR-015 | AC-004, AC-005 | AT-003, UT-004, REG-005 | Acceptance / Unit / Regression | Yes |
| FR-016 | AC-014 | UT-012, REG-008 | Unit / Regression | Yes |
| FR-017 | AC-015 | REG-011, MAN-001 | Regression / Manual | Partly |
| FR-018 | AC-018 | AT-004, UT-009 | Acceptance / Unit | Yes |
| FR-019 | AC-016, AC-022 | AT-008, UT-011, REG-006 | Acceptance / Unit / Regression | Yes |
| FR-020 | AC-020 | AT-007, UT-008 | Acceptance / Unit | Yes |
| FR-021 | AC-008, AC-016 | AT-005, UT-010, IT-006, REG-010 | Acceptance / Unit / Integration / Regression | Yes |
| NFR-001 | AC-010 | AT-001, UT-011, IT-001 | Acceptance / Unit / Integration | Yes |
| NFR-002 | AC-010, AC-016 | AT-001, UT-011, IT-006 | Acceptance / Unit / Integration | Yes |
| NFR-003 | AC-006 | AT-003, IT-008, REG-005 | Acceptance / Integration / Regression | Yes |
| NFR-004 | AC-003, AC-012 | AT-006, UT-007, IT-004 | Acceptance / Unit / Integration | Yes |
| RULE-001 | AC-005 | AT-003, UT-004, REG-001, REG-002, REG-003, REG-004 | Acceptance / Unit / Regression | Yes |
| RULE-002 | AC-011, AC-020 | AT-006, AT-007, UT-005, UT-008, IT-004 | Acceptance / Unit / Integration | Yes |
| RULE-003 | AC-003, AC-012, AC-013 | AT-006, UT-007, IT-003, IT-004, IT-005 | Acceptance / Unit / Integration | Yes |
| RULE-004 | AC-017 | REG-007, REG-011 | Regression | Yes |
| RULE-005 | AC-011, AC-017 | UT-005, REG-007 | Unit / Regression | Yes |

## 3. Acceptance Scenarios

### AT-001: Sentry Startup Is Optional And Early
Requirement IDs: BR-001, FR-001, NFR-001, NFR-002
Acceptance Criteria: AC-001, AC-010, AC-019
Priority: Must
Level: Acceptance
Automation Target: `app/test/observability/sentry_bootstrap_test.dart`

```gherkin
Feature: Sentry startup
  Scenario: Staging or production DSN initializes Sentry before the app renders
    Given build-time configuration includes SENTRY_DSN and an environment of staging or production
    And the app uses a fake Sentry adapter
    When the Flutter app starts
    Then Sentry initialization is invoked before runApp
    And the adapter receives the configured environment and release values when available
    And the app renders normally

  Scenario: Missing DSN leaves Sentry disabled without blocking startup
    Given build-time configuration has no SENTRY_DSN
    When the Flutter app starts
    Then no Sentry network transport is created
    And local logging remains available
    And the app renders normally

  Scenario: Local builds require explicit opt-in
    Given the app is running with a local or debug environment
    And no explicit local Sentry opt-in is configured
    When the Flutter app starts
    Then no Sentry events are sent
```

### AT-002: Uncaught Errors Are Captured Without Losing Local Debug Behavior
Requirement IDs: BR-001, FR-002, FR-003
Acceptance Criteria: AC-002, AC-010
Priority: Must
Level: Acceptance
Automation Target: `app/test/observability/error_handlers_test.dart`

```gherkin
Feature: Global Flutter error handling
  Scenario: Framework and root-zone failures reach Sentry and local logs
    Given Sentry is configured through the app-owned reporter
    When a Flutter framework error or root-zone async error is raised
    Then the failure is captured with its stack trace
    And a local severe log record is still emitted
    And Flutter's normal debug error presentation remains available in debug mode
```

### AT-003: User-Facing Error Copy Is Safe, Localized, And Finite
Requirement IDs: BR-002, BR-003, FR-011, FR-012, FR-013, FR-014, FR-015, RULE-001
Acceptance Criteria: AC-004, AC-005, AC-006, AC-007, AC-021
Priority: Must
Level: Acceptance
Automation Target: `app/test/shared/errors/app_error_test.dart`, `app/test/app_test.dart`, `app/test/router/router_error_screen_test.dart`, `app/test/settings/clear_image_cache_tile_test.dart`, `app/test/profile/widgets/profile_projects_tab_test.dart`

```gherkin
Feature: Safe user error UX
  Scenario: Known and unknown failures map to finite user-safe cases
    Given an API, auth, network, storage, image-picker, routing, initialization, or unknown failure
    When the app maps the failure for user display
    Then exactly one finite app error or warning case is selected
    And the case provides severity, localization key, display surface, recovery action policy, reportability, and safe Sentry classification

  Scenario: UI never displays raw diagnostic text
    Given an error contains raw exception text, backend message, request ID, status code, token-like text, identifier, raw URL, stack trace, or payload value
    When an app screen displays the error or warning
    Then the user-visible copy comes from AppLocalizations
    And none of the raw diagnostic text appears on screen or in AppMessenger messages
```

### AT-004: Reportable User Failures Can Show A Copyable Support Reference
Requirement IDs: FR-018
Acceptance Criteria: AC-018
Priority: Should
Level: Acceptance
Automation Target: `app/test/shared/errors/support_reference_test.dart`, plus widget tests for the chosen full-screen/message surface

```gherkin
Feature: Support references
  Scenario: Reportable user-visible failure captured by Sentry
    Given a reportable user-visible failure is captured
    And Sentry returns a non-empty event ID
    When the error UI or message is shown
    Then the user sees localized safe error copy
    And a localized copyable support reference is available
    And no AppView request ID, status code, backend error code, or raw exception text is shown

  Scenario: No support reference is shown without a Sentry event ID
    Given Sentry is disabled, capture fails, or capture returns an empty event ID
    When the error UI or message is shown
    Then the user still sees localized safe error copy
    And no fake support reference is displayed
```

### AT-005: Provider Failures Do Not Auto-Retry But Explicit Retry Still Works
Requirement IDs: BR-004, FR-009, FR-010, FR-021
Acceptance Criteria: AC-008, AC-009, AC-016
Priority: Must
Level: Acceptance
Automation Target: `app/test/app_test.dart`, `app/test/shared/riverpod/retry_policy_test.dart`, provider-specific tests that use app-owned harnesses

```gherkin
Feature: Riverpod retry policy
  Scenario: Production and test scopes do not automatically retry failed providers
    Given a provider fails in the production ProviderScope, startup ProviderContainer, or app-owned test harness
    When the provider enters an error state
    Then Riverpod automatic retry does not rerun it by default
    And the error state remains visible or observable

  Scenario: User retry explicitly recovers from a failure
    Given a failed provider is displayed with a Retry action
    When the user taps Retry
    Then the provider is explicitly invalidated or retried
    And the screen can recover when the provider succeeds
```

### AT-006: Reportability And Log Forwarding Are Privacy-Bounded
Requirement IDs: BR-001, FR-004, FR-005, FR-006, FR-007, FR-008, FR-012, NFR-004, RULE-002, RULE-003
Acceptance Criteria: AC-003, AC-011, AC-012, AC-013
Priority: Must
Level: Acceptance
Automation Target: `app/test/shared/errors/reportability_test.dart`, `app/test/shared/errors/sentry_redaction_test.dart`, `app/test/shared/api/providers/error_mapping_interceptor_test.dart`, `app/test/bootstrap/provider_logger_test.dart`

```gherkin
Feature: Privacy-bounded reporting
  Scenario: Reportable severe failures are sent with allowlisted context only
    Given a severe app log, provider failure, or AppView/Dio failure is classified as reportable
    When the app captures it through the reporter
    Then the event includes stack trace when available
    And it includes only allowlisted bounded context
    And forbidden fields such as headers, bodies, raw URLs, tokens, identifiers, user text, and unknown payloads are absent

  Scenario: Expected failures do not create Sentry error issues
    Given a validation error, user cancellation, expected auth/session state, ordinary not-found state, or single transient connectivity failure occurs
    When the app classifies the failure
    Then it may show safe UX or local logs
    And it does not create a Sentry error issue by default
```

### AT-007: Breadcrumbs Are Minimal And Safe
Requirement IDs: FR-006, FR-020, RULE-002
Acceptance Criteria: AC-020
Priority: Should
Level: Acceptance
Automation Target: `app/test/shared/errors/sentry_breadcrumb_test.dart`

```gherkin
Feature: Safe breadcrumbs
  Scenario: Breadcrumb data is reduced to bounded categories
    Given a breadcrumb candidate contains route, feature, lifecycle, retry action, handle, search query, cursor, raw path, or post/project text
    When the breadcrumb sanitizer processes it
    Then only coarse route or feature categories, lifecycle labels, and explicit recovery-action labels remain
    And identifiers, typed input, handles, raw URLs, query strings, cursors, post text, project text, and payloads are dropped
```

### AT-008: Feature Code Uses The App-Owned Reporter Boundary
Requirement IDs: FR-019
Acceptance Criteria: AC-016, AC-022
Priority: Must
Level: Acceptance
Automation Target: static regression check over `app/lib`

```gherkin
Feature: Reporting boundary
  Scenario: UI, provider, and API feature code cannot bypass reporting policy
    Given feature code needs to report or classify an error
    When imports under app/lib are inspected
    Then UI, provider, and API feature code imports the app-owned reporter interface or provider
    And only the central reporting implementation imports Sentry SDK packages
```

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-001, NFR-001 | AC-001, AC-010, AC-019 | Builds Sentry configuration from build-time values. | DSN absent, DSN empty, staging DSN, production DSN, local DSN with and without opt-in, release/dist values. | Initialization config is enabled only for configured staging/production or explicit local opt-in; disabled config performs no network setup and preserves release/environment values when supplied. | `app/test/observability/sentry_options_test.dart` |
| UT-002 | BR-003, FR-011 | AC-004, AC-007, AC-021 | Enumerates the finite app error/warning taxonomy. | All defined taxonomy cases. | Every case has severity, localization key, display surface/action policy, reportability flag, and safe Sentry category/classification. | `app/test/shared/errors/app_error_taxonomy_test.dart` |
| UT-003 | FR-012, FR-007 | AC-004, AC-012, AC-021 | Maps existing API and known feature exceptions into app error cases. | `ApiUnauthorized`, `ApiBadRequest`, `ApiServerError`, `ApiNetworkError`, `ApiCanceled`, storage failure, JSON/schema failure, image-picker cancellation, routing failure, unknown error. | Expected cases map to safe UX and reportability decisions; unknown cases use surface-specific safe fallbacks. | `app/test/shared/errors/app_error_mapper_test.dart` |
| UT-004 | BR-002, RULE-001, FR-015 | AC-005 | Rejects forbidden user-facing copy. | Error text containing `Exception:`, stack frames, AppView `message`, `requestId`, status codes, `did:`, handles, bearer tokens, raw URLs, query strings, and payload snippets. | Generated user messages and action labels are localized safe strings and contain none of the forbidden diagnostic values. | `app/test/shared/errors/app_error_copy_test.dart` |
| UT-005 | FR-005, FR-006, RULE-002, RULE-005 | AC-011 | Filters Sentry event/log context through an allowlist. | Maps containing headers, cookies, request bodies, response bodies, raw URLs, tokens, DIDs, handles, emails, device IDs, user text, AppView request ID, AppView error, HTTP status, normalized endpoint, feature area. | Forbidden fields are absent; allowlisted bounded fields remain with stable names and bounded values. | `app/test/shared/errors/sentry_redaction_test.dart` |
| UT-006 | FR-004 | AC-003 | Applies log forwarding severity rules. | `Level.INFO`, `Level.WARNING`, promoted warning, `Level.SEVERE`, records with and without stack traces. | Only severe/error records and deliberately promoted reportable warnings are sent to the reporter; warnings remain local by default; stack traces are attached when provided. | `app/test/observability/log_bridge_test.dart` |
| UT-007 | FR-004, FR-007, FR-008, NFR-004, RULE-003 | AC-003, AC-012, AC-013 | Classifies failures as reportable or expected. | Validation error, user cancellation, expected auth/session failure, ordinary not-found, single network timeout, repeated AppView 5xx, unknown API failure, JSON parse failure, secure storage failure, provider failure. | Expected states are not captured as Sentry errors by default; reportable defects/degraded states are captured with safe classifications. | `app/test/shared/errors/reportability_classifier_test.dart` |
| UT-008 | FR-020, RULE-002 | AC-020 | Sanitizes breadcrumb candidates. | Route names, feature areas, lifecycle labels, retry taps, handles, DIDs, text input, search terms, cursors, raw paths, query strings. | Safe breadcrumbs keep only coarse categories and recovery-action labels; unsafe values are dropped or reduced. | `app/test/shared/errors/sentry_breadcrumb_test.dart` |
| UT-009 | FR-018 | AC-018 | Formats support references. | Non-empty Sentry event ID, empty ID, disabled reporter result, failed capture result. | Copyable localized support reference appears only for non-empty event IDs and never includes backend identifiers. | `app/test/shared/errors/support_reference_test.dart` |
| UT-010 | FR-009, FR-021, BR-004 | AC-008, AC-016 | Verifies app-owned Riverpod retry helpers/defaults. | Production root scope config, startup container config, app test harness config, explicit opt-in retry test config. | Default retry callback returns `null`; explicit test opt-in can supply retry behavior only where requested. | `app/test/shared/riverpod/retry_policy_test.dart` |
| UT-011 | FR-001, FR-003, FR-019, NFR-002 | AC-010, AC-016, AC-022 | Exercises app-owned reporter interface and no-op/fake implementations. | Disabled reporter, fake reporter, throwing reporter, Sentry-backed reporter behind interface. | Callers can capture safely without real DSN or network; reporter failures do not block app code; policy remains centralized. | `app/test/observability/error_reporter_test.dart` |
| UT-012 | FR-016 | AC-014 | Guards disabled Sentry features. | Constructed Sentry options for all enabled environments. | Tracing, profiling, metrics, and session replay remain disabled; the test fails if any is enabled in this slice. | `app/test/observability/sentry_options_test.dart` |

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | BR-001, FR-001, NFR-001, NFR-002 | AC-001, AC-010, AC-019 | Verifies app startup with configured and unconfigured Sentry using fakes. | Inject fake reporter/bootstrap adapter and build-time config variants. | Pump the app startup path. | DSN path initializes before render; no-DSN path renders with no Sentry network requirement; local opt-in rules are enforced. | `app/test/observability/sentry_bootstrap_test.dart` |
| IT-002 | FR-002, FR-003 | AC-002, AC-010 | Verifies global error handlers capture and locally log errors. | Install handlers with fake reporter and log capture. | Trigger Flutter framework and root-zone async errors under test control. | Fake reporter receives captured errors with stack traces where available; local log records are emitted; debug presentation is not suppressed. | `app/test/observability/error_handlers_test.dart` |
| IT-003 | FR-004, RULE-003 | AC-003 | Verifies `package:logging` bridge behavior. | Configure root logger with fake Sentry log adapter. | Emit severe/error, warning, promoted warning, and expected-failure log records. | Severe/reportable records reach fake reporter with safe fields; ordinary warnings and expected states do not become Sentry issues. | `app/test/observability/log_bridge_test.dart` |
| IT-004 | FR-005, FR-006, FR-007, FR-012, RULE-002, RULE-003 | AC-011, AC-012 | Verifies Dio/AppView failure reporting through `ApiException` mapping. | Use `http_mock_adapter` or existing API client test pattern with AppView error envelopes and fake reporter. | Simulate 400, 401, 404, 500, timeout, cancellation, malformed JSON, and response bodies containing forbidden data. | Sentry-bound data contains endpoint category/status/AppView error/request ID when allowed; headers, bodies, raw URLs, backend messages, auth/session expected errors, cancellation, and routine transient failures are not captured as issues. | `app/test/shared/api/providers/error_mapping_interceptor_test.dart`, `app/test/shared/errors/app_error_mapper_test.dart` |
| IT-005 | FR-008, FR-009, FR-021 | AC-008, AC-013, AC-016 | Verifies provider observer routing and retry stability. | Create failing providers in a `ProviderScope`/`ProviderContainer` with app observer and fake reporter. | Let provider fail once and wait through the period where default retry would normally schedule work. | Failure is locally logged, routed through classifier, provider identity is bounded, reporter captures only if classified reportable, and auto-retry does not rerun by default. | `app/test/bootstrap/provider_logger_test.dart`, `app/test/shared/riverpod/retry_policy_test.dart` |
| IT-006 | FR-019, FR-021, NFR-002 | AC-016 | Ensures Flutter tests use fakes/no-op reporter and retry defaults. | App-owned widget/provider test harness. | Run representative widget/provider tests without DSN or Sentry auth token. | Tests pass without networked Sentry project; harness disables Riverpod retry by default unless a test opts in. | `app/test/test_support/app_harness_test.dart` or equivalent |
| IT-007 | FR-017, RULE-004 | AC-015, AC-017 | Verifies release symbolication configuration is documented and secret-free. | Repository files after implementation. | Inspect `pubspec.yaml`, Sentry Dart Plugin config/docs, and build documentation. | Debug-symbol/source-map/source-context/release/dist path is documented; upload auth token is referenced as build-environment only and not committed. | `app/test/observability/sentry_release_config_test.dart` or static script invoked by Flutter test |
| IT-008 | FR-014, NFR-003 | AC-006 | Verifies localization generation for new error UX keys. | New ARB keys and generated `AppLocalizations`. | Run localization generation as part of Flutter tooling and compile tests that access every key. | Every user-facing message and action label has an ARB entry, description, and generated getter. | `app/test/l10n/error_l10n_test.dart` |

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Test |
|---|---|---|---|
| REG-001 | Initialization errors render a full-screen error with Retry and recover after invalidation. | FR-010, FR-013, RULE-001 | Update `app/test/app_test.dart` so the error screen shows localized safe copy, never `Exception: boot failed`, and the existing Retry recovery test still passes with retry disabled. |
| REG-002 | GoRouter error screen provides safe navigation recovery. | FR-013, RULE-001 | Add/extend router error screen test so unexpected route errors show localized generic copy and Go Home without raw exception text. |
| REG-003 | Settings cache-clear failure reports through `AppMessenger`. | FR-013, RULE-001 | Update `app/test/settings/clear_image_cache_tile_test.dart` so `StateError('disk full')` does not appear in the user message and the localized safe cache-clear failure copy is shown. |
| REG-004 | Profile project loading fallback does not expose raw provider/repository errors. | FR-013, RULE-001 | Extend `app/test/profile/widgets/profile_projects_tab_test.dart` with a failing repository/provider and assert safe localized copy only. |
| REG-005 | Existing feature-specific localized messages remain available. | FR-014, FR-015, NFR-003 | Keep/extend existing sign-in, feed, notifications, profile, compose, search, and project tests so localized generic failure messages still render after adopting the shared error API. |
| REG-006 | Feature code cannot bypass centralized Sentry policy. | FR-019 | Static check: only the central observability/reporting implementation may import `sentry_flutter`, `sentry_logging`, or `sentry_dio`; UI/provider/API feature code must not import Sentry directly. |
| REG-007 | No committed secrets or token material are introduced. | RULE-004, RULE-005 | Static check over repository files for Sentry auth token patterns, AppView session token fixtures, PDS OAuth token material, committed DSN secrets, authorization headers, cookies, and raw token-like config values. |
| REG-008 | Tracing, profiling, metrics, and session replay remain out of scope. | FR-016 | Unit/static test fails if Sentry option construction enables traces sample rate, profiler sample rate, metrics, session replay, or equivalent features. |
| REG-009 | Flutter debug diagnostics remain useful. | FR-003 | Test `registerErrorHandlers` in debug-style configuration so `FlutterError.presentError` behavior is preserved and local log records still include the error and stack. |
| REG-010 | Riverpod retry remains disabled by default in production and app-owned tests. | FR-009, FR-021 | Existing provider error tests should use the app harness default; add a regression test that a failing provider runs once until explicit invalidation. |
| REG-011 | Release symbolication upload path stays documented and secret-free. | FR-017, RULE-004 | Static check validates the expected Sentry Dart Plugin configuration/docs exist and do not contain upload auth tokens. |

## 7. Test Data

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Startup config variants. | `SENTRY_DSN` absent, empty, staging value, production value, local value with explicit opt-in, release `1.0.0`, dist/build `1`. | AT-001, UT-001, IT-001 |
| TD-002 | Forbidden text corpus for user copy and Sentry filtering. | `Exception: boot failed`, `ApiServerError: boom`, stack frame text, `requestId=req_123`, `HTTP 500`, `did:plc:alice`, `alice.craftsky.social`, bearer token, cookie, raw URL with query string, JSON payload containing user text. | AT-003, AT-006, UT-004, UT-005, REG-001, REG-002, REG-003, REG-004 |
| TD-003 | AppView error envelopes. | `{error: "internal_error", message: "database failed for did:plc:alice", requestId: "req_123"}`, validation envelope, auth envelope, not-found envelope, 5xx envelope. | UT-003, UT-005, IT-004 |
| TD-004 | API exception set. | `ApiUnauthorized`, `ApiBadRequest("handle_required")`, `ApiBadRequest(null)`, `ApiServerError("boom")`, `ApiNetworkError("offline")`, `ApiCanceled()`. | UT-003, UT-007, IT-004 |
| TD-005 | Expected non-reportable failures. | Validation failure, user-cancelled image picker, user-cancelled auth handoff, expected session expiry, ordinary not-found, single transient connectivity failure. | AT-006, UT-007, IT-003, IT-004, IT-005 |
| TD-006 | Reportable handled failures. | Initialization failure, router failure, unknown API failure, JSON/schema parse failure, secure storage failure, cache-clear failure, provider failure outside expected states, repeated AppView 5xx. | AT-002, AT-004, AT-006, UT-007, IT-002, IT-005 |
| TD-007 | Breadcrumb candidates. | Safe route/feature/lifecycle/retry labels plus unsafe handles, DIDs, typed input, project/post text, raw URLs, query strings, cursors, payload snippets. | AT-007, UT-008 |
| TD-008 | Support reference variants. | Non-empty 32-character lowercase hex Sentry event ID, empty event ID, disabled capture, capture failure. | AT-004, UT-009 |
| TD-009 | Provider retry fixture. | Failing provider with attempt counter and explicit invalidation path mirroring `appDependenciesProvider` retry tests. | AT-005, UT-010, IT-005, REG-010 |

## 8. Manual Checks

| ID | Requirement IDs | Check | Steps | Expected Result |
|---|---|---|---|---|
| MAN-001 | BR-001, FR-017 | Staging Sentry delivery and symbolication smoke test. | Build a staging app with DSN and build-environment upload token, trigger a controlled reportable test error, and inspect the Sentry issue. | Event arrives in the staging project with readable stack/source context where supported, correct release/environment, no forbidden sensitive fields, and no tracing/profiling/session replay data. |
| MAN-002 | FR-002 | Native crash capture sanity check on a supported platform. | Use a controlled native crash path or SDK validation mechanism in a non-production build with staging Sentry configured. | Sentry receives the supported native crash event; missing native crash automation is documented as a platform validation item, not a blocker for unit/widget coverage. |

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | Real Sentry ingestion is mostly verified with fakes in automated tests. | BR-001, FR-001, FR-002, FR-017 | CI/local tests must not require real credentials or networked Sentry projects. | Keep automated tests around the app-owned reporter contract and perform MAN-001 before release. |
| GAP-002 | Native crash capture cannot be fully proven in normal Flutter widget/unit tests. | FR-002 | Native crash behavior depends on platform build artifacts and SDK support. | Perform MAN-002 on at least one supported mobile platform before production release. |
| GAP-003 | Static secret scans can miss novel token formats. | RULE-004, RULE-005 | Regex/static checks reduce risk but do not prove absence of all sensitive values. | Pair static checks with code review focused on build config, Sentry options, and reporter context allowlists. |
| GAP-004 | The later privacy-policy/legal documentation question remains outside this implementation slice. | NFR-004, RULE-005 | Requirements marked it as a non-blocking follow-up. | Track a separate pre-production privacy/legal task for crash/error reporting disclosure. |

## 10. Out Of Scope

- End-to-end AppView or PDS tests. This change consumes existing `/v1/*` error envelopes and does not alter backend routes or lexicons.
- Real Sentry credentials in automated tests.
- Sentry tracing, profiling, metrics, session replay, analytics funnels, or alert-rule verification.
- New non-English translations. Tests should verify the existing ARB/generation pipeline and English keys only.
- Offline retry queues, optimistic write reconciliation, or background sync behavior.
- Legal consent/privacy-policy implementation beyond preventing sensitive Sentry-bound data by default.

## 11. Handoff To Document Review

- Requirements file: `docs/changes/2026-07-03-flutter-error-handling-sentry/01-requirements.md`
- Test specification: `docs/changes/2026-07-03-flutter-error-handling-sentry/02-acceptance-tests.md`
- Next review artifact: `03-document-review.md`
- External Plannotator review, if the user initiates it outside this skill: `docs/changes/2026-07-03-flutter-error-handling-sentry/`
- Recommended first failing test for implementation: `UT-001` in `app/test/observability/sentry_options_test.dart`, proving Sentry option construction is disabled without a DSN, enabled for configured staging/production, supports explicit local opt-in, and keeps tracing/profiling/metrics/session replay disabled.
- Suggested test order for implementation:
  1. `UT-001`, `UT-011`, `UT-012` for reporter abstraction and Sentry option boundaries.
  2. `UT-002`, `UT-003`, `UT-004`, `IT-008` for finite taxonomy, mapping, safe localized copy, and l10n coverage.
  3. `UT-005`, `UT-006`, `UT-007`, `UT-008`, `IT-003`, `IT-004`, `IT-005` for redaction, reportability, logs, API, provider, and breadcrumb behavior.
  4. `AT-001`, `AT-002`, `IT-001`, `IT-002`, `REG-009` for startup and global error handlers.
  5. `AT-003`, `AT-004`, `AT-005`, `REG-001` through `REG-005`, `REG-010` for user-facing screens, support references, and retry behavior.
  6. `REG-006`, `REG-007`, `REG-008`, `REG-011`, then `MAN-001` and `MAN-002` before production release.
- Commands discovered:
  - `cd app && flutter test`
  - `cd app && flutter test test/app_test.dart`
  - `cd app && flutter test test/shared/api/api_exception_test.dart`
  - `cd app && flutter test test/settings/clear_image_cache_tile_test.dart`
  - `cd app && flutter test test/router/router_redirect_test.dart`
  - `cd app && flutter gen-l10n`
  - `cd app && dart analyze`
  - `just test` is Go/AppView-only and does not run Flutter tests.
- Blocking gaps: None. Medium risk remains; document review is recommended before coding.
