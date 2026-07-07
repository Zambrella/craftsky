# Coding Plan: Flutter Error Handling And Sentry Reporting

## 1. Inputs
- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md`
- Additional references inspected: `atproto-craft-social-app-reference.md`, Sentry Flutter SDK docs via Context7, Sentry Flutter skill references for error monitoring/logging/ecosystem integrations, and Riverpod 3 retry docs via Context7.

## 2. Implementation Strategy
Add one app-owned observability boundary and one finite user-facing error boundary, then wire existing app startup, API mapping, provider logging, l10n, and message/full-screen UI through those boundaries.

This fits the current codebase because the app already centralizes startup in `main.dart`/`bootstrap.dart`, uses `package:logging`, maps Dio failures to sealed `ApiException` types, uses Riverpod observers/listeners for failures, and routes user-visible copy through generated `AppLocalizations`. The implementation should not add AppView routes, PDS behavior, lexicon changes, product analytics, Sentry tracing, profiling, metrics, session replay, or broad user correlation.

Use `sentry_flutter` for app error capture and Sentry structured log APIs, plus `sentry_dart_plugin` for release symbolication. Do not add `sentry_dio` in the first implementation; report network failures through the existing `ApiException` mapping layer so AppView request IDs, error codes, status, and endpoint categories can be allowlisted without exposing headers, bodies, raw URLs, or backend messages.

## 3. Affected Areas
| Area | Existing Pattern | Planned Change | Requirement IDs | Test IDs |
|---|---|---|---|---|
| Startup and Sentry initialization | `main.dart` configures root logging, `runZonedGuarded`, error handlers, then `bootstrap()` | Introduce app-owned Sentry bootstrap/config; initialize before app render when enabled; preserve local logging and debug presentation | BR-001, FR-001, FR-002, FR-003, FR-016, NFR-001 | AT-001, AT-002, UT-001, UT-011, UT-012, IT-001, IT-002, REG-009 |
| Reporting boundary | No Sentry package in app; local `logging` only | Create central reporter interface, no-op/fake/Sentry implementations, import-boundary guard | FR-004, FR-005, FR-006, FR-019, NFR-002, RULE-002, RULE-005 | AT-006, AT-008, UT-005, UT-006, UT-011, REG-006, REG-007 |
| Error taxonomy and UX mapping | Feature-specific l10n messages plus some raw `error.toString()` fallbacks | Add finite app error/warning taxonomy, mapper, presentation helpers, support reference model | BR-002, BR-003, FR-011, FR-012, FR-013, FR-015, FR-018, RULE-001 | AT-003, AT-004, UT-002, UT-003, UT-004, UT-009, REG-001, REG-002, REG-003, REG-004 |
| API failure details | `ErrorMappingInterceptor` maps Dio to `ApiException` but drops request ID/status/endpoint detail | Preserve safe AppView diagnostics inside typed API exception details; feed mapper/reportability classifier | FR-006, FR-007, FR-012, RULE-002, RULE-003 | AT-006, UT-003, UT-005, UT-007, IT-004 |
| Riverpod provider failures and retry | `ProviderLogger` logs provider failures; production scopes omit global retry override | Inject reporter/classifier into provider observer and set global retry disablement in app/test scopes | BR-004, FR-008, FR-009, FR-010, FR-021 | AT-005, UT-007, UT-010, IT-005, IT-006, REG-010 |
| Localization | `app_en.arb` plus generated l10n | Add keys for finite error cases, support references, and action labels; run gen-l10n | FR-014, NFR-003 | AT-003, IT-008, REG-005 |
| Release symbolication | No Flutter Sentry release setup | Add minimal `sentry_dart_plugin` config/docs; keep upload token in build environment only | FR-017, RULE-004 | IT-007, REG-011, MAN-001 |

## 4. Files And Modules
| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `app/pubspec.yaml`, `app/pubspec.lock` | Change | Add `sentry_flutter` dependency and `sentry_dart_plugin` dev dependency; do not add `sentry_dio` in this slice | FR-001, FR-016, FR-017, FR-019 | UT-001, UT-012, IT-007, REG-006, REG-011 |
| `app/lib/main.dart` | Change | Build observability config from `--dart-define`, initialize reporting before `bootstrap`, register handlers with reporter, keep local debug sink | FR-001, FR-002, FR-003, FR-004 | AT-001, AT-002, IT-001, IT-002, REG-009 |
| `app/lib/bootstrap.dart` | Change | Accept reporter/observer dependencies, pass `ProviderScope` overrides, set `retry: appProviderRetry`, update `ProviderLogger` | FR-008, FR-009, FR-019, FR-021 | UT-010, IT-005, REG-010 |
| `app/lib/shared/observability/sentry_config.dart` | Create | Parse build-time Sentry config and disabled-feature options | FR-001, FR-016, RULE-004 | UT-001, UT-012, REG-008 |
| `app/lib/shared/observability/error_reporter.dart` | Create | App-owned reporter interface, capture result, no-op implementation, fake-friendly contract | FR-004, FR-005, FR-006, FR-018, FR-019 | UT-005, UT-006, UT-011 |
| `app/lib/shared/observability/error_reporter_provider.dart` | Create | Riverpod provider for feature code to access app-owned reporter/fakes | FR-019, FR-021 | IT-006, REG-006 |
| `app/lib/shared/observability/sentry_error_reporter.dart` | Create | Only production file allowed to import `sentry_flutter`; capture exceptions/logs/breadcrumbs through sanitizer | FR-001, FR-002, FR-004, FR-005, FR-006, FR-016, FR-020 | UT-005, UT-006, UT-008, UT-011, UT-012, REG-006 |
| `app/lib/shared/observability/log_forwarder.dart` | Create | Route severe/error log records and deliberately promoted warnings to reporter with safe messages/context | FR-004, RULE-002, RULE-003 | UT-006, IT-003 |
| `app/lib/shared/observability/sentry_sanitizer.dart` | Create | Allowlist event/log/breadcrumb fields and normalize endpoint/provider/feature classifications | FR-005, FR-006, FR-020, RULE-002, RULE-005 | UT-005, UT-008, IT-004 |
| `app/lib/shared/errors/app_error.dart` | Create | Finite error/warning cases, severity, surface, action policy, reportability, safe classifications | BR-002, BR-003, FR-011, FR-018 | UT-002, UT-009 |
| `app/lib/shared/errors/app_error_mapper.dart` | Create | Map `ApiException`, storage, routing, initialization, image-picker, provider, and unknown failures to finite cases | FR-011, FR-012, RULE-001, RULE-003 | UT-003, UT-004, UT-007 |
| `app/lib/shared/errors/app_error_presenter.dart` | Create | Convert finite errors to localized messages/actions for full-screen, inline, and `AppMessenger` surfaces | FR-013, FR-014, FR-015, FR-018 | AT-003, AT-004, UT-004, UT-009 |
| `app/lib/shared/api/api_exception.dart` | Change | Add safe diagnostic details such as status, request ID, AppView error code, endpoint category; keep sealed hierarchy | FR-006, FR-007, FR-012 | UT-003, UT-005, IT-004 |
| `app/lib/shared/api/providers/error_mapping_interceptor.dart` | Change | Extract AppView envelope diagnostics and normalized endpoint category without raw URL/body/header propagation | FR-005, FR-006, FR-007, FR-012 | UT-003, UT-005, IT-004 |
| `app/lib/app.dart` | Change | Map/report initialization failures through shared error/reporting boundary while preserving one-log-per-transition behavior | FR-008, FR-013, FR-018 | AT-003, AT-004, REG-001 |
| `app/lib/initialization_error_screen.dart` | Change | Accept display model or finite error; show localized safe copy, retry, optional support reference | FR-013, FR-014, FR-018, RULE-001 | REG-001, UT-009 |
| `app/lib/router/error_screen.dart`, `app/lib/router/router.dart` | Change | Map GoRouter errors to safe navigation failure and breadcrumb/report when reportable | FR-013, FR-014, FR-020, RULE-001 | REG-002, AT-007 |
| `app/lib/profile/widgets/profile_tabs/profile_projects_tab.dart` | Change | Replace `error.toString()` fallback with shared safe app error presentation | FR-013, FR-015, RULE-001 | REG-004 |
| `app/lib/settings/widgets/clear_image_cache_tile.dart` | Change | Keep existing messenger pattern but route failure through safe app error presentation/reportability | FR-013, FR-015, RULE-001 | REG-003 |
| `app/lib/l10n/app_en.arb`, generated l10n files | Change | Add all new user-facing error, support reference, and action strings | FR-014, NFR-003 | IT-008, REG-005 |
| `app/test/test_support/app_harness.dart` | Create | Shared widget/provider harness with fake reporter and `retry: appProviderRetry` default | FR-021, NFR-002 | IT-006, REG-010 |
| `app/test/observability/*_test.dart` | Create | Sentry config, reporter, handlers, log bridge, import boundary, secret/release config static tests | FR-001, FR-003, FR-004, FR-016, FR-017, FR-019 | UT-001, UT-006, UT-011, UT-012, IT-001, IT-002, IT-003, REG-006, REG-007, REG-008, REG-011 |
| `app/test/shared/errors/*_test.dart` | Create | Taxonomy, mapper, copy safety, redaction, reportability, breadcrumbs, support references | FR-005, FR-006, FR-011, FR-012, FR-018, FR-020 | UT-002, UT-003, UT-004, UT-005, UT-007, UT-008, UT-009 |
| `app/README.md` or `docs/changes/2026-07-03-flutter-error-handling-sentry/release-symbolication.md` | Change / Create | Document staging build, obfuscation, split debug info, source-map upload, auth-token env var, manual Sentry smoke checks | FR-017, RULE-004 | IT-007, REG-011, MAN-001, MAN-002 |

## 5. Services, Interfaces, And Data Flow
Use two boundaries:

1. `shared/errors` owns finite user-facing classification.
2. `shared/observability` owns Sentry/no-op/fake capture, sanitization, and import policy.

```text
Object error + StackTrace + optional source context
  -> AppErrorMapper.map(error, surface, source)
  -> AppError
     - kind: finite user-message-oriented case
     - severity/surface/action policy
     - reportability: expected vs reportable
     - sentryClassification: bounded technical category
     - safeDiagnostics: allowlisted key/value data
  -> UI: AppErrorPresenter.message(l10n, error)
  -> Reporting: ErrorReporter.capture(error, stackTrace, safeDiagnostics)
  -> optional SupportReference from non-empty Sentry event ID
```

Partial interface sketch:

```text
abstract interface class ErrorReporter {
  bool get enabled;
  Future<ReportResult> captureException(
    Object error, {
    StackTrace? stackTrace,
    required ReportContext context,
  });
  Future<void> captureLog(LogRecord record, {required ReportContext context});
  void addBreadcrumb(SafeBreadcrumb breadcrumb);
}

enum AppErrorKind {
  networkUnavailable,
  serviceUnavailable,
  sessionExpired,
  permissionDenied,
  contentUnavailable,
  storageUnavailable,
  initializationFailed,
  navigationFailed,
  actionFailed,
  backgroundLoadFailed,
  unexpected,
}

final class ApiFailureDetails {
  final int? statusCode;
  final String? appViewError;
  final String? requestId;
  final String endpointCategory; // e.g. appview.feed.list, not raw URL
}
```

Sentry options must be constructed from build-time values:

```text
SentryConfig.fromEnvironment()
  reads SENTRY_DSN, SENTRY_ENVIRONMENT, SENTRY_RELEASE, SENTRY_DIST,
        SENTRY_LOCAL_OPT_IN
  enabled when:
    - DSN is non-empty and environment is staging/production, or
    - DSN is non-empty and local opt-in is explicit
  always:
    - sendDefaultPii = false
    - tracing/profiling/metrics/session replay disabled
    - beforeSend, beforeSendLog, beforeBreadcrumb route through sanitizer
```

Do not pass raw `RequestOptions.uri`, query strings, headers, bodies, backend `message`, DIDs, handles, emails, device IDs, session tokens, OAuth/DPoP material, or arbitrary object payloads into `ReportContext`.

## 6. State, Providers, Controllers, Or DI
Planned provider graph:

```text
main()
  -> ObservabilityBootstrap.initialize(SentryConfig)
  -> ErrorReporter reporter (Sentry or Noop)
  -> registerErrorHandlers(reporter)
  -> bootstrap(binding, reporter)

bootstrap(binding, reporter)
  -> ProviderContainer(
       observers: [ProviderLogger(reporter)],
       retry: appProviderRetry,
       overrides: [errorReporterProvider.overrideWithValue(reporter)],
     )
  -> runApp(ProviderScope(
       observers: [ProviderLogger(reporter)],
       retry: appProviderRetry,
       overrides: [errorReporterProvider.overrideWithValue(reporter)],
       child: App(),
     ))
```

Provider choices:

- `errorReporterProvider`: `Provider<ErrorReporter>` with no-op default for tests that do not use the app harness.
- `appProviderRetry`: top-level function returning `null`; shared by production `ProviderScope`, startup `ProviderContainer`, and test harnesses.
- `ProviderLogger`: keep local `logging` behavior, but route failures through `AppErrorMapper` and `ErrorReporter` only when classified reportable. Provider identity must be normalized to a bounded provider name/category.
- Existing feature providers remain unchanged unless they currently display raw errors or need explicit retry behavior preserved.

## 7. UI, Widgets, Routes, Or User-Facing Surfaces
Use existing surfaces rather than adding a new global error framework:

- Full-screen initialization failure: `InitializationErrorScreen` shows `initializationFailedTitle`, safe body copy, Retry, and optional support reference.
- Router failure: `ErrorScreen` shows `routingErrorTitle`, safe navigation copy, Go Home, and optional report/support reference if classified reportable.
- Inline/profile tab failure: `ProfileTabErrorSliver` receives localized safe copy from `AppErrorPresenter`; Retry continues invalidating `userProjectsProvider(handle)`.
- Settings cache-clear failure: existing `AppMessenger.error` path remains, but message comes from `AppErrorPresenter` and reportable cache failures route through reporter.
- Existing feature-specific messages for sign-in, feed, notifications, profile, compose, search, and project flows should remain unless a touched path currently exposes raw diagnostics.
- Support reference UI should be a small, localized, copyable value attached only to reportable user-visible failures when capture returns a non-empty event ID. Do not display AppView request IDs, HTTP status codes, or backend error codes.

No route changes are planned. GoRouter may receive a `SentryNavigatorObserver` only if it can emit coarse route-name breadcrumbs without enabling tracing; otherwise use explicit safe breadcrumb calls in router/error/retry surfaces. Do not add analytics-style route/user-action tracking.

## 8. Error, Loading, Empty, And Edge States
| State / Case | Planned Handling | Requirement IDs | Test IDs |
|---|---|---|---|
| No Sentry DSN | Use no-op reporter; app starts; local logging remains | FR-001, NFR-001, NFR-002 | AT-001, UT-001, IT-001 |
| Local/debug without opt-in | No external events; local debug sink keeps printing | FR-001 | AT-001, UT-001 |
| Sentry init/capture failure | Catch and degrade to no-op/local logging; never block UI | NFR-001 | UT-011, IT-001 |
| Flutter framework/root-zone error | Local severe log plus reporter capture with stack where available; preserve debug presentation | FR-002, FR-003 | AT-002, IT-002, REG-009 |
| Native crash | Rely on `sentry_flutter` supported native capture when configured; do not add PII scope; validate manually | FR-002, RULE-005 | MAN-002 |
| Severe log with stack | Forward safe record through custom log bridge; attach stack and bounded logger/category | FR-004, FR-005 | UT-006, IT-003 |
| Warning log | Keep local by default; only promoted classified warnings reach reporter | FR-004, RULE-003 | UT-006, UT-007 |
| API validation/auth/cancel/not-found/transient network | Safe UX or no message as appropriate; no Sentry error issue by default | FR-007, FR-012, RULE-003 | UT-003, UT-007, IT-004 |
| Repeated 5xx/unknown API/parse/storage/provider defects | Report with endpoint category/status/AppView error/request ID when present and allowlisted | FR-006, FR-007, FR-008, FR-012 | AT-006, UT-005, UT-007, IT-004, IT-005 |
| Raw UI fallback errors | Replace `error.toString()` with localized finite fallback | FR-013, RULE-001 | AT-003, REG-001, REG-002, REG-004 |
| Provider failure after retry disabled | State stays failed until explicit invalidation/retry | FR-009, FR-010, FR-021 | AT-005, UT-010, REG-010 |
| Support reference unavailable | Show safe copy without fake reference | FR-018 | AT-004, UT-009 |
| Breadcrumb candidate with identifiers/content | Drop or reduce to coarse category/action | FR-020, RULE-002 | AT-007, UT-008 |
| Release symbol upload token | Read from build environment only; never commit | FR-017, RULE-004 | IT-007, REG-007, REG-011 |

## 9. Test Implementation Plan
| Order | Test ID | Target | Setup / Fixture | Initial Expected Failure |
|---|---|---|---|---|
| 1 | UT-001, UT-012 | `app/test/observability/sentry_options_test.dart` | Build-time config matrix from TD-001; disabled feature assertions | `SentryConfig` and option builder do not exist |
| 2 | UT-011 | `app/test/observability/error_reporter_test.dart` | No-op/fake reporter, throwing reporter, capture result variants | Reporter interface/result types do not exist |
| 3 | UT-002 | `app/test/shared/errors/app_error_taxonomy_test.dart` | Enumerate finite `AppErrorKind` cases | Taxonomy does not exist or lacks metadata |
| 4 | UT-003, UT-007 | `app/test/shared/errors/app_error_mapper_test.dart`, `reportability_classifier_test.dart` | TD-003 through TD-006 API/known/expected/reportable failures | Mapper/classifier missing or maps expected failures as reportable |
| 5 | UT-004, IT-008 | `app/test/shared/errors/app_error_copy_test.dart`, `app/test/l10n/error_l10n_test.dart` | Forbidden text corpus and ARB/generated getters | Raw strings leak or l10n keys absent |
| 6 | UT-005, UT-008 | `app/test/shared/errors/sentry_redaction_test.dart`, `sentry_breadcrumb_test.dart` | Allowlisted/forbidden contexts and breadcrumb candidates | Sanitizer missing or keeps forbidden fields |
| 7 | UT-006, IT-003 | `app/test/observability/log_bridge_test.dart` | `LogRecord` levels, stack traces, promoted warning flag | Warnings/severe logs not filtered correctly |
| 8 | IT-004 | `app/test/shared/api/providers/error_mapping_interceptor_test.dart` | Dio failures with AppView envelopes and forbidden body/header data | `ApiException` lacks safe diagnostic details |
| 9 | UT-010, IT-005 | `app/test/shared/riverpod/retry_policy_test.dart`, `app/test/bootstrap/provider_logger_test.dart` | Failing providers, attempt counters, fake reporter | Production/test retry helper and reporter-aware observer absent |
| 10 | AT-001, IT-001 | `app/test/observability/sentry_bootstrap_test.dart` | Fake bootstrap adapter with configured/unconfigured Sentry | Startup path cannot inject fake reporter or prove order |
| 11 | AT-002, IT-002, REG-009 | `app/test/observability/error_handlers_test.dart` | Fake reporter plus log capture; trigger framework/root-zone failures | Existing handlers only log locally |
| 12 | REG-001 | `app/test/app_test.dart` | Boot failure containing forbidden text; Retry recovery | Initialization screen still shows raw exception |
| 13 | REG-002 | `app/test/router/router_error_screen_test.dart` | Router error with forbidden text | Router screen still shows raw error |
| 14 | REG-003 | `app/test/settings/clear_image_cache_tile_test.dart` | `StateError('disk full')`, recording messenger | Cache error copy not fully shared/safe |
| 15 | REG-004 | `app/test/profile/widgets/profile_projects_tab_test.dart` | Failing `userProjectsProvider(handle)` | Profile tab still shows `error.toString()` |
| 16 | AT-004, UT-009 | `app/test/shared/errors/support_reference_test.dart` plus chosen widget tests | Non-empty/empty Sentry event IDs | Support reference model/UI absent |
| 17 | IT-006, REG-010 | `app/test/test_support/app_harness_test.dart` and representative provider/widget tests | Harness without DSN, fake reporter, retry defaults | Tests still hand-roll retry and reporter setup |
| 18 | REG-006, REG-007, REG-008, REG-011, IT-007 | `app/test/observability/import_boundary_test.dart`, `secret_scan_test.dart`, `sentry_release_config_test.dart` | File scans over `app/lib`, `app/pubspec.yaml`, docs | Static enforcement missing |
| 19 | MAN-001, MAN-002 | Manual staging checklist | Staging DSN/build env upload token; controlled reportable test error/native crash | Not automated; execute before production release |

Focused command for most loops: `cd app && flutter test <target>`. Run `cd app && flutter gen-l10n` after ARB edits and `cd app && dart run build_runner build --delete-conflicting-outputs` only if provider/codegen files are added or changed.

## 10. Sequencing And Guardrails
- First TDD step: write `UT-001` in `app/test/observability/sentry_options_test.dart` for DSN/environment/local-opt-in behavior and disabled tracing/profiling/metrics/session replay.
- Dependencies between work items: Sentry config and reporter interface should land before startup handlers; taxonomy and mapper should land before UI replacements; redaction/reportability should land before provider/API/log bridge capture; l10n keys should land before widget regression updates.
- Guardrails: no direct Sentry imports outside `app/lib/shared/observability/sentry_*` and approved observability tests; no `sentry_dio` in this slice; no raw user identifiers/content/tokens/URLs/bodies/headers in Sentry context; no user-visible raw error strings/codes/request IDs/status; no Sentry tracing/profiling/metrics/session replay; keep AppView `/v1/*` contract unchanged; keep Flutter app holding only Craftsky session tokens.
- Release-symbolication decision: use the minimal Sentry Dart Plugin path with `sentry_dart_plugin` as a dev dependency, `upload_debug_symbols`, `upload_source_maps`, `upload_sources`, and `symbols_path` documented. Source upload/source context is approved for app source only; upload auth token must come from `SENTRY_AUTH_TOKEN` in the build environment. Commit no auth token and no DSN secret.
- Static checks: implement as Dart/Flutter tests that scan repository files so they run under `cd app && flutter test` without real Sentry credentials.
- Out of scope: AppView/PDS/lexicon changes, Sentry tracing/profiling/metrics/session replay, `sentry_dio`, analytics funnels, user correlation, privacy-policy/legal copy, offline retry queues, broad feature-message rewrites beyond touched raw fallbacks.

## 11. Risks And Open Questions
| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Non-blocking | Exact Sentry Flutter package version may change before implementation | API names for logs/options/plugin config may need small adjustments | Use latest compatible `sentry_flutter`/plugin versions during implementation and keep behavior behind app-owned interfaces |
| CPQ-002 | Non-blocking | Native crashes bypass Dart `beforeSend` filtering | Some native event fields depend on SDK defaults rather than app sanitizer | Keep `sendDefaultPii=false`, do not set user scope, avoid custom native context, and validate MAN-002 |
| CPQ-003 | Non-blocking | `sentry_logging` may look attractive but can forward raw log bodies if used carelessly | Privacy risk and noisy issues | Prefer custom `LogForwarder`; use `sentry_logging` only if tests prove sanitized severe/error behavior |
| CPQ-004 | Non-blocking | Sentry org/project slugs for plugin config are not known in this planning stage | Symbol upload docs/config may need final deployment values | Document placeholders or environment-supplied CI values; never commit upload auth token |
| CPQ-005 | Non-blocking | Requirements doc still says `Status: Draft` although review approved it | Workflow metadata may look stale | Leave source docs unchanged in this stage; optionally add a manual note or revise requirements later |
| CPQ-006 | Non-blocking | No blocking implementation questions identified | The TDD builder can start from `UT-001` | No action needed |

## 12. Handoff To TDD Builder
- Coding plan: `docs/changes/2026-07-03-flutter-error-handling-sentry/04-coding-plan.md`
- TDD execution plan: `05-implementation-plan.md`
- Start with test: `UT-001` in `app/test/observability/sentry_options_test.dart`
- Focused command: `cd app && flutter test test/observability/sentry_options_test.dart`
- Notes: Keep Sentry optional and disabled without DSN, keep all Sentry SDK imports behind `shared/observability`, use fakes/no-op reporter in tests, and treat redaction/reportability as core behavior before wiring provider/API/log call sites.
