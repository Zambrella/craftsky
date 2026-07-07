# Requirements: Flutter Error Handling And Sentry Reporting

## 1. Initial Request

After the AppView observability work, add proper error handling and logging in the Flutter app. Continue using Sentry as the main app error logging/reporting mechanism. Add a user-facing error UX strategy so errors and warnings are shown safely without error codes, raw exception text, or sensitive information. User-facing messages should be internationalized and backed by a finite set of typed error/exception cases that can be enumerated. Do not add Sentry tracing or profiling at this time. Disable Riverpod 3 automatic retry for now.

## 2. Current Codebase Findings

- Relevant files:
  - `app/pubspec.yaml` uses Flutter, `dio`, `go_router`, `flutter_riverpod` 3.3.1, `riverpod_annotation`, `logging`, `flutter_localizations`, and generated l10n. It does not currently include `sentry_flutter`, `sentry_logging`, or `sentry_dio`.
  - `app/lib/main.dart` configures `package:logging`, wraps startup in `runZonedGuarded`, registers `FlutterError.onError`, `PlatformDispatcher.instance.onError`, and a release `ErrorWidget.builder`.
  - `app/lib/bootstrap.dart` creates production `ProviderScope(observers: [ProviderLogger()], child: App())` and a startup `ProviderContainer`. `ProviderLogger.providerDidFail` currently logs provider failures through `logging`.
  - `app/lib/app.dart` listens for `appDependenciesProvider` initialization errors and logs them once per error transition.
  - `app/lib/initialization_error_screen.dart` and `app/lib/router/error_screen.dart` currently render `error.toString()` to users.
  - `app/lib/shared/messaging/*` provides `AppMessenger` with info/warning/error severities and sticky warning/error snackbar behavior.
  - `app/lib/shared/api/api_exception.dart`, `app/lib/shared/api/providers/error_mapping_interceptor.dart`, and `app/lib/shared/api/api_unwrap.dart` already map Dio failures into a sealed `ApiException` hierarchy.
  - Feature screens already use localized generic messages for many failures, for example sign-in, feed loading, notifications loading, profile loading, compose failures, and search failures.
  - Some remaining UI paths still expose raw errors, including initialization, routing, profile projects, and settings cache-clear failures.
- Existing patterns:
  - User-facing strings go through `app/lib/l10n/app_en.arb` and generated `AppLocalizations`.
  - Flutter UI uses Riverpod listeners to surface transition-based failures.
  - Diagnostics should use `package:logging`, not `print`, except for the existing debug-only root logger sink in `main.dart`.
  - API reads/writes go through the AppView; the Flutter app must not hold PDS OAuth tokens or talk to the PDS directly in the happy path.
  - AppView `/v1/*` errors use a JSON envelope with `{error, message, requestId}` and camelCase keys.
- Current behavior:
  - App errors are logged locally in debug mode but are not exported to Sentry.
  - Uncaught Flutter/Dart errors are handled by local logging handlers, not Sentry.
  - Provider failures are logged locally by `ProviderLogger` but not reported to Sentry.
  - Dio failures are normalized into `ApiException`, but there is no app-wide user-safe message mapper.
  - Some UI still renders raw exception strings, which can expose technical details or sensitive values.
  - Riverpod 3 automatic retry is not disabled at the production root `ProviderScope`, although several tests already disable it explicitly with `retry: (_, _) => null`.
- Constraints discovered:
  - Sentry Flutter SDK setup should run early in app lifecycle. Official Sentry Flutter docs describe error monitoring as baseline, with optional logs, session replay, tracing, and profiling. Official Sentry logs docs require `enableLogs: true` and SDK support for logs in `sentry_flutter` 9.x.
  - `sentry_logging` can bridge the existing Dart `logging` package to Sentry. The Dart logging package requires explicit stack traces for accurate error reporting.
  - `sentry_dio` can integrate with Dio; request/response body and header privacy must be controlled if used.
  - Riverpod 3 docs state automatic retry can be disabled globally by passing `retry: (retryCount, error) => null` to `ProviderScope` and `ProviderContainer`.
  - The project roadmap already lists Flutter error-handling UX and crash reporting as open items.
- Test/build commands discovered:
  - App tests are under `app/test/`.
  - Focused app verification should use `cd app && flutter test`.
  - Existing root `just test` is Go/AppView-only and does not run Flutter tests.

## 3. Clarifying Questions And Decisions

### Q1: Should this requirements pass ask for Sentry tracing, profiling, metrics, or session replay?
Answer: No blocking question asked. The prompt explicitly excludes tracing and profiling; this requirements pass also excludes metrics and session replay to keep the app scope focused on error reporting, safe logging, and user error UX.
Decision / implication: Sentry configuration should capture errors/crashes and structured logs only. Tracing, profiling, metrics, and session replay are non-goals unless a later change scopes them.

### Q2: Should user-facing messages display AppView error codes, request IDs, status codes, or raw exception text?
Answer: No. The prompt says user-visible errors/warnings should not include error codes or sensitive information.
Decision / implication: UI copy must come from localized safe message keys. Diagnostic context may be logged or sent to Sentry only after redaction and allowlisting, and must not be displayed to users.

### Q3: Should the app send raw DIDs, handles, emails, tokens, device IDs, request bodies, or response bodies to Sentry?
Answer: No blocking question asked.
Decision / implication: Default requirements forbid sensitive identifiers and payloads in Sentry events/logs. If user correlation is needed later, it needs a separate privacy decision.

### Q4: Should Sentry be required in local development?
Answer: No blocking question asked.
Decision / implication: Sentry should be disabled when no DSN is supplied. Local/debug logging remains available through `package:logging`.

### Q5: Should Riverpod automatic retry be disabled per provider or globally?
Answer: The prompt says turn it off "for now"; Riverpod docs support disabling it globally.
Decision / implication: Production app `ProviderScope` and startup `ProviderContainer` should set `retry: (_, _) => null`, while preserving explicit user retry buttons.

### Q6: Does Sentry expose an event ID that can be shown to users for support?
Answer: Yes. The Dart SDK's capture APIs return a `SentryId`, and the SDK exposes `Sentry.lastEventId`. `SentryId` is a 32-character lowercase hexadecimal UUID string.
Decision / implication: Reportable user-visible failures should be able to expose a copyable "support reference" only when Sentry successfully returns a non-empty event ID. The reference should be presented as a diagnostic support value, not as an error code or explanation, and should be hidden when Sentry is disabled or capture fails.

### Q7: Which environments should send Sentry events in the first implementation?
Answer: Staging and production should report by default when configured. Local/debug builds should not report unless a developer explicitly opts in with build-time configuration.
Decision / implication: Sentry environment handling must support `production`, `staging`, and explicit local opt-in. Local development remains private and quiet by default, while still allowing SDK validation when needed.

### Q8: Should Sentry events include user correlation?
Answer: No, not in this first slice.
Decision / implication: Sentry events must remain user-anonymous by default. They must not include DID, handle, email, raw device ID, AppView session ID, PDS token material, or a stable generated user identifier. Useful context should come from event ID, release, environment, platform, feature area, error kind, coarse auth state, and other bounded diagnostics.

### Q9: Should AppView `requestId` be attached to Sentry events?
Answer: Yes, but only as allowlisted diagnostic context and never in user-facing UI.
Decision / implication: AppView `requestId` may be sent to Sentry for cross-service debugging. Users should only see a Sentry support reference when available, not AppView request IDs.

### Q10: Which user-visible failures should receive a Sentry support reference?
Answer: Only failures already classified as reportable.
Decision / implication: The app should not create Sentry events solely to produce a copyable reference for routine validation errors, cancellations, expected auth/session states, empty states, or ordinary transient errors.

### Q11: What counts as a reportable handled failure?
Answer: Report handled failures when they indicate a possible app/backend defect or persistent degraded state.
Decision / implication: Examples include initialization failure, router failure, unexpected `ApiException.unknown`, JSON/schema parse failure, secure storage failure, cache-clear failure, provider failure outside expected empty/offline states, and repeated AppView 5xx failures. Expected validation errors, auth cancellation, image-picker cancellation, normal not-found states, and single transient connectivity failures should not be reported unless they break a critical flow.

### Q12: Should AppView `requestId` be shown as part of the user support reference?
Answer: No.
Decision / implication: Users should only receive the Sentry support reference. The Sentry event can contain the AppView request ID so developers can pivot into backend logs.

### Q13: Should Sentry capture breadcrumbs for recent user actions/navigation?
Answer: Yes, but only minimal safe breadcrumbs.
Decision / implication: Breadcrumbs may include route name/category, feature area, lifecycle/app state, and coarse actions such as "retry tapped". Breadcrumbs must not include identifiers, typed input, handles, post/project text, raw URLs, query parameters, payloads, or analytics-style behavior trails.

### Q14: Should Sentry capture `package:logging` warnings automatically?
Answer: No. Automatic Sentry forwarding should default to severe/error records only.
Decision / implication: Warnings remain local unless deliberately promoted through the app error reporter as a classified reportable warning. This reduces noise and privacy risk from less-reviewed warning messages.

### Q15: Should the finite error taxonomy be user-message oriented or technical-cause oriented?
Answer: User-message oriented, with separate safe technical classification fields for Sentry.
Decision / implication: Error cases should model user outcomes such as network unavailable, service unavailable, session expired, permission denied, content unavailable, storage unavailable, and unexpected. Sentry can receive bounded classifications such as `api.timeout`, `api.5xx`, `json.decode`, or `router.unknown`.

### Q16: Should unknown errors all map to one generic user message?
Answer: No. Use a generic fallback plus a few surface-specific fallbacks where recovery differs.
Decision / implication: Initialization, navigation, save/action, and background/load failures may have separate fallback cases so each can carry the right display surface and recovery policy while keeping copy safe.

### Q17: Should AppView's `message` field ever be displayed to users?
Answer: No.
Decision / implication: AppView `{error, message, requestId}` is diagnostic input only. The Flutter app maps failures to localized app-owned copy and must not display backend `message` directly.

### Q18: Should AppView's `error` code be sent to Sentry?
Answer: Yes, as bounded diagnostic classification only.
Decision / implication: AppView `error` may be attached to Sentry when present, but it must not be displayed to users and must not be treated as arbitrary text for tags or grouping.

### Q19: Should HTTP status codes be sent to Sentry?
Answer: Yes, as numeric diagnostic context only.
Decision / implication: HTTP status codes may help debugging but must not be displayed to users and must not by themselves make an event reportable.

### Q20: Should request URLs be sent to Sentry?
Answer: No raw URLs. Send only normalized route templates or endpoint categories.
Decision / implication: Sentry context may include values such as `GET /v1/feed` or `appview.feed.list`, but not full URLs, query strings, cursors, handles, IDs, search terms, or user data.

### Q21: Should the app use Sentry's Dio integration directly?
Answer: Prefer reporting through the existing `ApiException` mapping layer first.
Decision / implication: `sentry_dio` may be used only if it can be configured to produce the same redacted endpoint/status/error-code context without raw request data. It must not bypass app-owned reportability and privacy rules.

### Q22: Should expected auth/session failures be reportable?
Answer: No, not by default.
Decision / implication: Session expiry, user-cancelled sign-in, invalid credentials/handle input, and normal authorization denial should map to safe UX without Sentry error capture. Unexpected auth machinery failures such as malformed callback state, secure storage failure, impossible state transitions, or repeated backend 5xx during auth may be reportable.

### Q23: Should provider failures always be captured by the provider observer?
Answer: No.
Decision / implication: The provider observer should route failures through the shared reportability classifier rather than blindly capturing every provider error.

### Q24: Should Riverpod automatic retry be disabled in tests?
Answer: App-owned test harnesses should disable it by default, while individual tests may opt into retry when specifically testing retry behavior.
Decision / implication: Provider failure states should be deterministic in production and tests unless retry behavior is explicitly under test.

### Q25: Should the app test that tracing, profiling, metrics, and session replay stay disabled?
Answer: Yes.
Decision / implication: Sentry option construction should have a lightweight test or static assertion that verifies these features are not enabled in this slice.

### Q26: Should Sentry initialization be hidden behind an app-owned abstraction?
Answer: Yes.
Decision / implication: `main.dart` may call an early bootstrap abstraction, but UI, provider, and API code should depend on an app-owned reporter/interface rather than importing Sentry directly. This centralizes configuration, redaction, reportability, test fakes, and capture policy.

## 4. Candidate Approaches

### Option A: Central typed error UX plus Sentry-backed reporting
Summary: Add a finite app error taxonomy for user-facing outcomes, route all user messages through localized copy, replace raw error rendering, initialize Sentry for error monitoring and structured logs, bridge existing `logging` output safely, wire Riverpod/Dio error reporting through bounded adapters, and disable Riverpod auto-retry globally.
Pros:
- Matches the prompt's typed, localizable error UX goal.
- Builds on existing `ApiException`, `AppMessenger`, l10n, Riverpod listener, and `logging` patterns.
- Keeps Sentry focused on app errors/logs without tracing/profiling overhead.
- Gives tests a finite set of cases to verify exhaustively.
Cons:
- Requires touching cross-cutting app startup, error mapping, l10n, and several UI fallbacks.
- Needs strict privacy rules so Sentry logs do not become a data leak.
Risks:
- A too-broad taxonomy may be hard to maintain; a too-narrow one may collapse actionable failures into generic messages.
- Logging integration can create duplicate Sentry issues if severity thresholds are not chosen carefully.

### Option B: Sentry SDK only, leave current UI error handling mostly unchanged
Summary: Initialize Sentry and capture uncaught errors/provider failures, but leave feature-level error copy and raw error fallbacks mostly as-is.
Pros:
- Faster and smaller implementation.
- Improves developer visibility for crashes quickly.
Cons:
- Does not solve the prompt's main UX concern.
- Leaves raw exception strings visible in some screens.
- Does not create an enumerable localization-friendly error model.
Risks:
- Users continue seeing technical or sensitive details even though maintainers see better Sentry events.

### Option C: Feature-by-feature error cleanup without central taxonomy
Summary: Update each screen's error copy manually and add Sentry reporting at selected call sites, but avoid a shared app error taxonomy.
Pros:
- Allows feature-specific messages without an up-front model.
- Avoids introducing a cross-cutting abstraction.
Cons:
- Hard to verify exhaustively.
- New screens can regress to raw `error.toString()` without a shared rule.
- Sentry context and privacy handling may become inconsistent.
Risks:
- Error UX drifts across features and translations become harder to keep complete.

## 5. Recommended Direction

Recommended approach: Option A: Central typed error UX plus Sentry-backed reporting.

Why: The app already has the right building blocks: localized strings, a sealed `ApiException`, `AppMessenger`, Riverpod observers/listeners, and standard Dart logging. A narrow app error taxonomy can turn these into a consistent UX contract while Sentry receives safe diagnostic events and logs. This meets the request without expanding into tracing, profiling, session replay, metrics, analytics, or PDS/AppView architecture changes.

## 6. Problem / Opportunity

The Flutter app currently has partial local logging and many localized feature messages, but it lacks a consistent end-to-end policy for deciding what users see, what developers receive in Sentry, and what data must never leave the device. Some paths still render raw exception text. Adding Sentry now is useful only if it is paired with safe, finite, translatable user-facing error handling.

## 7. Goals

- G-001: Capture Flutter app errors and important failure logs in Sentry without enabling tracing or profiling.
- G-002: Define a finite, enumerable app error taxonomy for user-facing errors and warnings.
- G-003: Ensure every user-facing error/warning message comes from localized copy and excludes raw exception text, error codes, request IDs, tokens, and sensitive values.
- G-004: Preserve explicit user retry UX while disabling Riverpod 3 automatic retry.
- G-005: Keep local development usable when Sentry is not configured.
- G-006: Provide enough traceability for acceptance test design.

## 8. Non-Goals

- NG-001: Enable Sentry tracing, profiling, metrics, session replay, product analytics, or user behavior funnels.
- NG-002: Add AppView routes, change the `/v1/*` error envelope, or change AppView observability.
- NG-003: Store PDS OAuth tokens, AppView session tokens, DPoP material, or private user data in Sentry.
- NG-004: Display AppView error codes, HTTP status codes, request IDs, stack traces, or raw exception text to users.
- NG-005: Add a user-facing locale picker or new non-English locale.
- NG-006: Replace all feature-specific error messages with one generic fallback.
- NG-007: Add offline-first retry queues, optimistic write reconciliation, or background sync behavior.
- NG-008: Add legal consent flows or a privacy policy update in this slice, beyond requirements that avoid sending sensitive data by default.
- NG-009: Generate commits, implement source changes, or add tests during this requirements stage.
- NG-010: Add user correlation, user analytics, stable pseudonymous user IDs, or broad behavior tracking to Sentry events.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Craftsky user | Person using the Flutter app on Android, iOS, or Web | Clear, non-technical, localized error/warning messages with useful recovery actions |
| App maintainer | Developer/operator investigating app failures | Sentry issues/logs with stack traces, release/environment/platform context, and safe bounded error categories |
| App contributor | Developer adding screens, providers, or API clients | A finite typed error UX model and rules that prevent raw error leakage |
| Translator | Future contributor translating app copy | Stable localization keys and descriptions for each user-facing error/warning message |

## 10. Current Behavior

The app logs through `package:logging` and debug prints root log records locally. Top-level Flutter/Dart error handlers log severe errors locally. Riverpod provider failures are logged by `ProviderLogger`. Dio failures are mapped to sealed `ApiException` cases for API clients. Many feature screens already show localized generic messages, but some global and feature fallbacks still display `error.toString()`. Sentry is not wired into the Flutter app. Riverpod automatic retry is not globally disabled in production app scopes.

## 11. Desired Behavior

The app should initialize Sentry early when a DSN is provided for staging or production, and should allow explicit local opt-in for SDK validation. It should capture uncaught Flutter/Dart errors, report selected provider/API/action failures with stack traces, export safe severe/error logs, and record only minimal safe breadcrumbs. Sentry must be disabled cleanly when no DSN is configured. The app should define a finite app error taxonomy that maps technical failures to localized user messages, severities, display surfaces, and optional recovery actions, with separate bounded diagnostics for Sentry. UI must never display raw exception details, AppView messages, AppView error codes, status codes, request IDs, tokens, identifiers, raw URLs, or payloads. Existing explicit retry buttons remain, but Riverpod automatic retry is disabled globally so errors remain stable until the user or code explicitly retries.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | The Flutter app shall use Sentry as the primary app-level error reporting and failure logging destination when configured. | Maintainers need app failures visible outside local debug logs. | Prompt, Sentry Flutter docs | AC-001, AC-002, AC-003 |
| BR-002 | Business | Must | User-facing errors and warnings shall be safe, localized, and non-technical. | Users should not see codes, stack traces, raw exception strings, or sensitive values. | Prompt, codebase findings | AC-004, AC-005, AC-006 |
| BR-003 | Business | Must | The app shall expose a finite, enumerable user-facing error/warning taxonomy. | Enables exhaustive tests, predictable UX, and translation coverage. | Prompt | AC-004, AC-007 |
| BR-004 | Business | Must | The error handling strategy shall preserve explicit recovery actions such as Retry while disabling implicit Riverpod automatic retry. | Users should control retries and tests should see stable error states. | Prompt, Riverpod docs, codebase tests | AC-008, AC-009 |
| FR-001 | Functional | Must | The system shall add Sentry Flutter initialization that runs early in app startup when `SENTRY_DSN` is supplied by build-time configuration for staging or production, leaves Sentry disabled when the DSN is absent or empty, and supports explicit local/development opt-in for SDK validation. | Sentry setup requires early lifecycle initialization, but local dev should not require external reporting. | Sentry Flutter docs, codebase findings, user answer | AC-001, AC-010, AC-019 |
| FR-002 | Functional | Must | The system shall capture uncaught Flutter framework errors, uncaught Dart async/root-zone errors, and release native crashes supported by the Sentry Flutter SDK. | These are baseline error monitoring responsibilities. | Sentry Flutter docs, codebase findings | AC-002 |
| FR-003 | Functional | Must | The system shall preserve or replace the existing `FlutterError.onError`, `PlatformDispatcher.instance.onError`, `runZonedGuarded`, and `ErrorWidget.builder` behavior so errors are both locally logged and reported to Sentry without losing Flutter's normal debug presentation. | Current startup handlers encode app behavior that should not regress. | Codebase findings | AC-002, AC-010 |
| FR-004 | Functional | Must | The system shall bridge severe/error app logs from `package:logging` to Sentry structured logs or an equivalent safe Sentry logging adapter, with stack traces attached where available. Warning logs shall remain local by default unless deliberately promoted through the app error reporter as classified reportable warnings. | The app already uses `logging`; Sentry logs should reuse that pipeline while avoiding noisy warning export. | Codebase findings, Sentry logs docs, user answer | AC-003 |
| FR-005 | Functional | Must | The system shall filter Sentry-bound events and logs to remove or avoid tokens, authorization headers, cookies, request/response bodies, raw AppView session tokens, PDS OAuth material, DPoP material, raw device IDs, emails, and user-entered content. | Observability must not leak credentials or private data. | AGENTS.md, prompt, Sentry privacy guidance | AC-011 |
| FR-006 | Functional | Must | The system shall include safe bounded diagnostic context in Sentry where useful, such as app error kind, severity, feature area, platform, environment, release/build, coarse auth state, AppView `requestId`, AppView `error`, HTTP status code, normalized endpoint category, and safe technical classification. | Sentry events need enough context to be actionable without sensitive data. | Prompt, Sentry Flutter docs, codebase findings, user answer | AC-003, AC-011, AC-012, AC-020 |
| FR-007 | Functional | Should | The system should report Dio/AppView failures through the existing `ApiException` mapping layer first, using redacted endpoint categories, status codes, and AppView error classifications with no request or response bodies. Direct `sentry_dio` integration may be used only if it follows the same privacy and reportability rules. | Network failures are important app errors, and the app already normalizes Dio into `ApiException`. | Codebase findings, Sentry ecosystem guidance, user answer | AC-012 |
| FR-008 | Functional | Must | The system shall route Riverpod provider failures through a Sentry-aware provider observer or equivalent adapter and the shared reportability classifier while keeping local provider logging. | Provider failures currently have a central observer, but not all provider failures should become Sentry issues. | Codebase findings, Sentry ecosystem guidance, user answer | AC-013 |
| FR-009 | Functional | Must | The system shall disable Riverpod automatic retry globally in production app scopes by configuring `ProviderScope` and startup `ProviderContainer` retry behavior to return `null`. | The prompt asks to turn off Riverpod 3 retry for now. | Prompt, Riverpod docs, codebase tests | AC-008 |
| FR-010 | Functional | Must | The system shall keep explicit user-initiated retry actions, such as invalidating providers from retry buttons, working after automatic retry is disabled. | Disabling implicit retry must not remove intentional recovery UX. | Codebase findings | AC-009 |
| FR-011 | Functional | Must | The system shall define a finite user-message-oriented app error/warning model that maps technical errors into user-facing message keys, severity, display surface, optional retry/action metadata, reportability, and separate safe Sentry classification fields. | Gives UI, logging, Sentry, and tests one shared error contract without making UX copy mirror implementation details. | Prompt, codebase findings, user answer | AC-004, AC-007, AC-021 |
| FR-012 | Functional | Must | The system shall map existing `ApiException` cases, AppView error envelopes, and known feature exceptions into the app error/warning model, with generic and surface-specific unknown fallbacks for unexpected failures. | Existing API normalization should feed the new UX model rather than be duplicated, and different surfaces need different recovery actions. | Codebase findings, user answer | AC-004, AC-005, AC-012, AC-021 |
| FR-013 | Functional | Must | The system shall replace user-visible `error.toString()` paths in initialization, router, settings cache-clear, profile project loading, and any other touched fallback surfaces with localized safe messages. | Current raw error rendering violates the desired UX strategy. | Codebase findings | AC-005, AC-006 |
| FR-014 | Functional | Must | The system shall add ARB localization keys and generated localizations for every new user-facing error/warning message and recovery action label. | The prompt requires internationalized/translatable messages. | Prompt, i18n scaffold | AC-006 |
| FR-015 | Functional | Should | The system should provide a small UI consumption API so screens can display app errors through existing full-screen, inline, or `AppMessenger` warning/error patterns without manually inspecting exception internals. | Reduces future drift and raw error leakage. | Codebase findings | AC-004, AC-005 |
| FR-016 | Functional | Must | The system shall not enable Sentry tracing, profiling, metrics, or session replay in this slice, and shall include a lightweight test or static assertion around option construction to guard that behavior. | The prompt excludes tracing/profiling, and the app does not need broader Sentry features yet. | Prompt, Sentry Flutter docs, user answer | AC-014 |
| FR-017 | Functional | Must | The system shall configure the Sentry Dart Plugin path for release symbolication, including `sentry_dart_plugin`, debug-symbol upload, source-map upload where relevant, source context where approved, release/dist configuration, and build documentation for `--split-debug-info` and obfuscation-map generation. | Error reports are not sufficiently useful without readable stack traces, source maps, and obfuscated issue-title support. | User feedback, Sentry Dart Plugin docs | AC-015 |
| FR-018 | Functional | Should | The system should expose a copyable localized support reference for reportable user-visible failures when Sentry returns a non-empty event ID, without showing AppView request IDs or other backend identifiers to users. | A support reference lets users send developers a precise diagnostic pointer without exposing technical error details. | User question, Sentry Dart API docs, user answer | AC-018 |
| FR-019 | Functional | Must | The system shall route Sentry initialization, capture, redaction, reportability decisions, and test fakes through a small app-owned error reporting abstraction instead of importing Sentry directly from UI, provider, or API feature code. | Centralizes policy, keeps Sentry optional in tests, and prevents call sites from bypassing privacy/reportability rules. | User answer, codebase findings | AC-016, AC-022 |
| FR-020 | Functional | Should | The system should attach only minimal safe Sentry breadcrumbs for coarse navigation, feature area, app lifecycle, and explicit recovery actions such as retry taps. | Breadcrumbs improve diagnosis but must not become behavior analytics or leak user content. | User answer, Sentry Flutter docs | AC-020 |
| FR-021 | Functional | Must | The system shall disable Riverpod automatic retry by default in app-owned Flutter test harnesses, while allowing individual tests to opt into retry when retry behavior is under test. | Test behavior should match production error stability unless retry is the subject of the test. | User answer, Riverpod docs, codebase tests | AC-008, AC-016 |
| NFR-001 | Non-functional | Must | Error and log reporting shall not block app startup, rendering, or user actions when Sentry is unavailable, disabled, rate-limited, or misconfigured. | Observability must not degrade product reliability. | Codebase findings | AC-010 |
| NFR-002 | Non-functional | Must | Tests shall be able to run without contacting Sentry and without requiring real Sentry credentials. | CI/local test determinism and secret safety. | Codebase findings | AC-010, AC-016 |
| NFR-003 | Non-functional | Must | User-facing error copy shall be understandable without technical codes and shall not rely on English-only hard-coded strings in Dart. | Supports future translation and safe UX. | Prompt, i18n scaffold | AC-006 |
| NFR-004 | Non-functional | Should | Sentry issue volume should be controlled so expected validation errors, user cancellations, and routine recoverable states do not create noisy error issues. | High noise makes error reporting less useful. | Sentry logging guidance, codebase findings | AC-003, AC-012 |
| RULE-001 | Business rule | Must | UI code must not display `Object.toString()`, `Exception.toString()`, `DioException.message`, AppView `message`, AppView `error`, AppView `requestId`, HTTP status code, stack trace, token, identifier, raw URL, or payload text as user-facing error copy. | Prevents leakage of technical or sensitive details and keeps backend diagnostic text out of localized UX. | Prompt, codebase findings, user answer | AC-005 |
| RULE-002 | Business rule | Must | Sentry-bound data must be allowlisted; unknown object payloads, headers, request bodies, response bodies, raw URLs, query strings, breadcrumbs, and user-entered text must be dropped or replaced with bounded classifications. | Privacy-safe observability requires explicit boundaries. | AGENTS.md, Sentry privacy guidance, user answer | AC-011, AC-020 |
| RULE-003 | Business rule | Must | User cancellations, expected validation failures, expected auth/session states, ordinary not-found states, and routine transient connectivity failures shall not be reported to Sentry as errors unless they reveal an app/backend bug or persistent degraded state. | Avoids noisy and unactionable reporting. | Codebase findings, user answer | AC-003, AC-012, AC-013 |
| RULE-004 | Business rule | Must | Sentry DSNs and build upload tokens shall not be committed to source; runtime DSN/environment/release values must come from build-time configuration, and upload auth tokens must remain build-environment only. | Protects secrets and deployment hygiene. | Sentry Flutter docs, security practice | AC-017 |
| RULE-005 | Business rule | Must | Sentry events shall remain user-anonymous by default and must not include DID, handle, email, raw device ID, AppView session ID, PDS token material, or a stable generated user identifier. | Keeps the first observability slice privacy-bounded and avoids implicit user tracking. | User answer, AGENTS.md | AC-011, AC-017 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-001 | Given `SENTRY_DSN` is supplied at build time for staging or production, when the app starts, then Sentry initialization is invoked before `runApp` and receives environment/release configuration where available. |
| AC-002 | BR-001, FR-002, FR-003 | Given a Flutter framework error, root-zone async error, or supported native crash path occurs, when Sentry is configured, then the failure is captured by Sentry and local logging behavior remains intact. |
| AC-003 | BR-001, FR-004, FR-006, NFR-004, RULE-003 | Given app log records are emitted, when records are severe/error or deliberately promoted reportable warnings, then Sentry receives safe structured logs/events with stack traces for severe failures and does not create error issues for expected validation, cancellation, auth/session, not-found, or routine transient connectivity paths. |
| AC-004 | BR-002, BR-003, FR-011, FR-012, FR-015 | Given a known API, auth, network, storage, image-picker, routing, initialization, or generic unknown failure occurs, when the app maps it for UX, then it resolves to one finite app error/warning case with severity, localization key, safe Sentry classification, and optional recovery action metadata. |
| AC-005 | BR-002, FR-012, FR-013, FR-015, RULE-001 | Given any app screen displays an error or warning, when the user sees the message, then the text does not include raw exception output, AppView `message`, AppView error codes, request IDs, HTTP status codes, stack traces, tokens, identifiers, raw URLs, or payload values. |
| AC-006 | BR-002, FR-014, NFR-003 | Given new error/warning cases exist, when localization generation runs, then every user-facing message and action label has an ARB key, description, and generated `AppLocalizations` getter. |
| AC-007 | BR-003, FR-011 | Given the error taxonomy is compiled and tested, when tests enumerate all defined cases, then each case has a user message key, severity, safe Sentry category, and intended display surface or action policy. |
| AC-008 | BR-004, FR-009, FR-021 | Given a provider fails, when the production `ProviderScope`, startup `ProviderContainer`, or app-owned test harness handles the failure, then Riverpod automatic retry does not rerun the provider by default unless a test explicitly opts into retry. |
| AC-009 | BR-004, FR-010 | Given a screen shows a retry action for a failed provider, when the user taps Retry, then the provider is explicitly retried or invalidated and can recover. |
| AC-010 | FR-001, FR-003, NFR-001, NFR-002 | Given no Sentry DSN is configured or Sentry initialization fails safely, when the app starts and tests run, then the app still renders, local logging remains available, and no network call to Sentry is required. |
| AC-011 | FR-005, FR-006, RULE-002, RULE-005 | Given a Sentry-bound event/log includes context, when filtering is applied, then forbidden fields, user correlation fields, unknown object payloads, raw URLs, and user-entered text are absent and only allowlisted bounded fields remain. |
| AC-012 | FR-007, FR-012, NFR-004, RULE-003 | Given a Dio/AppView request fails, when it is reported to Sentry, then the event/log includes safe request classification such as normalized endpoint, status code, AppView error, and AppView request ID when present, without authorization headers, cookies, request/response bodies, raw URLs, backend messages, or routine expected validation/cancellation/auth errors as issues. |
| AC-013 | FR-008, RULE-003 | Given a provider transitions to failure, when `ProviderLogger.providerDidFail` or its replacement runs, then the failure is locally logged and routed through the reportability classifier before any Sentry capture, with provider identity bounded to a safe provider name/category. |
| AC-014 | FR-016 | Given Sentry options are configured, when tests or code review inspect option construction, then tracing, profiling, metrics, and session replay are not enabled and the assertion fails if they become enabled. |
| AC-015 | FR-017 | Given release builds use obfuscation or web source maps, when build documentation/config is reviewed, then there is a documented path for uploading readable debug symbols/source maps without committing Sentry auth tokens. |
| AC-016 | FR-019, FR-021, NFR-002 | Given Flutter tests run, when Sentry-related behavior is tested, then fake/no-op transports or dependency overrides are used, app-owned test harnesses disable Riverpod retry by default, and no real DSN, auth token, or networked Sentry project is required. |
| AC-017 | RULE-004 | Given the repository is scanned, when Sentry configuration is present, then no DSN secret, Sentry auth token, PDS token, AppView session token, or OAuth material is committed. |
| AC-018 | FR-018 | Given a reportable user-visible failure is captured by Sentry and returns a non-empty event ID, when the error UI or message is shown, then the user can copy a localized support reference; when Sentry is disabled, capture fails, or the event ID is empty, then no fake reference is shown. |
| AC-019 | FR-001 | Given a local/debug build has no explicit Sentry opt-in configuration, when the app starts, then no Sentry events are sent; given a developer supplies an explicit local DSN/environment opt-in, then Sentry can be initialized for validation. |
| AC-020 | FR-006, FR-020, RULE-002 | Given breadcrumbs are attached to Sentry events, when filtering is applied, then they contain only minimal coarse navigation/feature/lifecycle/recovery-action labels and exclude identifiers, typed input, handles, post/project text, raw URLs, query strings, and payloads. |
| AC-021 | FR-011, FR-012 | Given unexpected failures occur in initialization, navigation, save/action, or background/load surfaces, when they are mapped for UX, then each resolves to a finite safe fallback with the correct display surface and recovery action policy. |
| AC-022 | FR-019 | Given UI, provider, or API feature code needs to report an error, when imports and tests are inspected, then feature code uses the app-owned reporter/interface or provider and does not bypass policy by importing Sentry directly. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | Sentry DSN omitted in debug/local builds | App starts normally, logs locally, sends no Sentry events. | FR-001, NFR-001 |
| EC-002 | Sentry initialization throws or SDK is unavailable on a platform | App degrades to local logging and safe UI; startup is not blocked. | NFR-001 |
| EC-003 | Error occurs before `MaterialApp`/l10n is available | Last-resort fallback remains safe and generic; no raw sensitive values are shown in release. | FR-003, RULE-001 |
| EC-004 | App dependencies fail before the main app is ready | Initialization error screen shows a localized safe message and Retry, not `error.toString()`. | FR-013, FR-014 |
| EC-005 | GoRouter receives an unexpected routing error | Routing error screen shows localized generic copy and Go Home, not raw route exception details. | FR-013, RULE-001 |
| EC-006 | AppView returns `{error, message, requestId}` for a failed `/v1/*` call | UI maps to a safe localized app error; Sentry may include only allowlisted classification, not displayed codes. | FR-012, RULE-001, RULE-002 |
| EC-007 | User cancels image picking, browser auth handoff, or another expected flow | UI may show no message or a safe info/warning; Sentry does not receive an error issue. | RULE-003 |
| EC-008 | Provider fails repeatedly after auto-retry is disabled | Error state stays visible until explicit user/code retry; no hidden background retry loop occurs. | FR-009, FR-010 |
| EC-009 | Severe log record has no stack trace | Adapter captures the best available context but tests should prefer paths that pass stack traces for reportable failures. | FR-004 |
| EC-010 | A log/event contains a token-like or payload-like attribute | Filtering removes or redacts the attribute before Sentry export. | FR-005, RULE-002 |
| EC-011 | Web build runs with Sentry configured | Dart/Flutter errors and logs are captured where supported; native crash/session replay expectations are not required. | FR-002, FR-016 |
| EC-012 | App is offline when an error occurs | User sees safe copy; Sentry SDK behavior must not block the app and may buffer/drop according to SDK support. | NFR-001 |
| EC-013 | Sentry capture returns an empty event ID or no event ID | User still sees safe localized error copy, but no support reference is displayed. | FR-018 |
| EC-014 | Local developer wants to validate Sentry setup | Developer can explicitly opt in with DSN/environment configuration; local builds without opt-in remain no-reporting. | FR-001 |
| EC-015 | AppView response includes a useful `message` | Message may inform internal mapping only if needed, but is not displayed to users or forwarded as arbitrary Sentry text. | FR-012, RULE-001 |
| EC-016 | Provider failure is an expected auth/session or transient network state | Failure is locally logged or shown through safe UX as appropriate, but the observer does not automatically create a Sentry issue. | FR-008, RULE-003 |
| EC-017 | Safe breadcrumb candidate contains a handle, content title, query, cursor, or raw path | Breadcrumb is dropped or reduced to a bounded category before Sentry export. | FR-020, RULE-002 |

## 15. Data / Persistence Impact

- New fields: None in AppView, PDS, or local user data stores.
- Changed fields: None.
- Migration required: No database or PDS migration.
- Backwards compatibility:
  - Existing AppView error envelopes remain unchanged.
  - Existing user sessions and secure storage remain unchanged.
  - New localization keys are additive.
  - New package dependencies may change `app/pubspec.lock`.

## 16. UI / API / CLI Impact

- UI:
  - Replace raw-error fallback text with localized generic/specific safe messages.
  - Use existing full-screen error, inline error, and `AppMessenger` severity patterns.
  - Preserve explicit Retry actions for recoverable failures.
  - Where a reportable failure has a non-empty Sentry event ID, provide a localized copyable support reference.
  - Show only the Sentry support reference to users; never show AppView request IDs, backend messages, status codes, or error codes.
- API:
  - No AppView API changes.
  - Existing `/v1/*` error envelope is consumed through safe mapping.
- CLI:
  - No CLI impact.
- Background jobs:
  - No app background job changes.

## 17. Security / Privacy / Permissions

- Authentication:
  - Flutter app continues to hold only Craftsky session tokens.
  - No PDS OAuth tokens or DPoP material are added to the device.
- Authorization:
  - No permission model changes.
- Sensitive data:
  - Sentry-bound data must not include authorization headers, cookies, AppView session tokens, PDS tokens, DPoP keys, device IDs, emails, request/response bodies, user-entered text, raw identifiers, or payload dumps.
  - Sentry DSN is supplied by build-time configuration; Sentry auth token for debug-symbol upload is build-environment only.
  - User-visible Sentry event IDs are support references only. They must not be treated as secrets, must not replace safe localized error messages, and must not include additional raw diagnostic payloads.
  - AppView `requestId`, AppView `error`, HTTP status code, and normalized endpoint category may be attached to Sentry as allowlisted diagnostics but must not be shown to users.
  - Sentry events remain user-anonymous by default and must not include stable user correlation identifiers in this slice.
- Abuse cases:
  - Error messages must not reveal whether a private resource exists unless existing product behavior intentionally does so.
  - Logs must not turn AppView error envelope data into a user-data exfiltration path.

## 18. Observability

- Events:
  - Sentry captures reportable app errors/crashes with stack traces and safe classification fields.
  - Expected validation, cancellation, auth/session, not-found, and routine transient connectivity paths are not Sentry error issues unless they reveal an app/backend bug or persistent degraded state.
  - Safe diagnostic fields may include AppView request ID, AppView error, HTTP status, normalized endpoint category, feature area, environment, release, platform, coarse auth state, and safe technical classification.
  - Minimal breadcrumbs may include coarse navigation, feature area, lifecycle/app state, and explicit recovery actions, after allowlist filtering.
- Logs:
  - Local logs remain available.
  - Sentry structured logs are enabled only for safe, filtered severe/error records when Sentry is configured, plus deliberately promoted reportable warnings.
- Metrics:
  - None in this slice.
- Alerts:
  - No Sentry alert rules or dashboards in this slice.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Sentry logging exports sensitive data from existing log messages. | Tokens, identifiers, or user content could leave the device. | Use allowlisted Sentry-bound context, filtering hooks, and tests that reject forbidden attributes. |
| RISK-002 | Error taxonomy becomes too broad or too implementation-specific. | Translations and tests become hard to maintain. | Start with stable user-actionable categories and keep technical details in safe Sentry classifications. |
| RISK-003 | Disabling Riverpod retry exposes errors that previously self-healed. | Users may see more stable error states until they tap Retry. | Preserve explicit retry actions and classify retryable failures clearly. |
| RISK-004 | Sentry integration changes startup order or breaks existing error handlers. | App startup or debug diagnostics could regress. | Test startup with and without DSN, and verify existing handler behavior remains. |
| RISK-005 | Duplicate capture paths create noisy Sentry issues. | Maintainer may see the same failure multiple times. | Define capture ownership: top-level handlers for uncaught errors, provider/Dio adapters for handled reportable failures, and filtering for expected paths. |
| RISK-006 | Debug symbol/source-map upload is missed. | Production Sentry stack traces may be hard to read. | Document release build upload configuration and keep auth tokens outside source. |
| RISK-007 | User-facing copy becomes too generic. | Users may not know what to do after failures. | Include severity and action metadata so recoverable errors can show Retry or sign-in actions without exposing technical details. |
| RISK-008 | Users or support copy the Sentry event ID without enough context. | Developers may still need surrounding user steps to reproduce the issue. | Label it as a support reference and keep it alongside normal user feedback/retry flows rather than replacing them. |
| RISK-009 | Direct Sentry SDK imports bypass app redaction/reportability policy. | Feature code could accidentally send sensitive or noisy events. | Use an app-owned reporter abstraction and add tests/static checks around imports and reporter behavior. |
| RISK-010 | Breadcrumbs drift into behavior tracking or leak content. | Sentry data could exceed the intended privacy scope. | Keep breadcrumbs minimal, allowlisted, and covered by redaction tests. |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | Sentry should be configured with build-time `--dart-define` values rather than committed config files containing secrets. | Requirements may need adjustment for a different deployment/config strategy. |
| ASM-002 | The first implementation should not send raw DIDs, handles, emails, device IDs, session identifiers, or stable generated user identifiers to Sentry. | If user correlation is required, a privacy review and revised allowlist are needed. |
| ASM-003 | Session replay is out of scope alongside tracing/profiling/metrics. | If session replay is desired, requirements must add explicit privacy, masking, and platform rules. |
| ASM-004 | English remains the only shipped locale for now, but all new strings must use the existing l10n pipeline. | If another locale is added in this slice, acceptance criteria must include translated ARB coverage. |
| ASM-005 | Sentry package versions available at implementation time support Flutter SDK logs and the chosen `sentry_logging`/Dio integration APIs. | Coding plan may need to adapt package versions or use a small local adapter. |

## 21. Open Questions

- [ ] Non-blocking: Should a later privacy-policy/legal task explicitly document app crash/error reporting before production release?

## 22. Review Status

Status: Draft
Risk level: Medium
Review recommended: Yes
Reviewer:
Date: 2026-07-03
Notes: Medium risk because this is cross-cutting startup, logging, privacy, localization, and UI behavior work. No blocking open questions for acceptance-test design, but the later privacy-policy/legal follow-up should be reviewed before production release.

## 23. Handoff To Test Design

- Requirements file: `docs/changes/2026-07-03-flutter-error-handling-sentry/01-requirements.md`
- Next test specification: `02-acceptance-tests.md`
- Must-cover requirement IDs:
  - `BR-001`, `BR-002`, `BR-003`, `BR-004`
  - `FR-001` through `FR-021`
  - `NFR-001`, `NFR-002`, `NFR-003`, `NFR-004`
  - `RULE-001`, `RULE-002`, `RULE-003`, `RULE-004`, `RULE-005`
- Suggested test levels:
  - Unit tests for error taxonomy exhaustiveness, technical-to-user error mapping, reportability classification, Sentry redaction/filter helpers, safe breadcrumb filtering, support-reference formatting, and Riverpod retry config helpers.
  - Widget tests for initialization, routing, feature fallback surfaces, and `AppMessenger` safe copy behavior.
  - Provider tests for provider failure capture and explicit retry behavior with auto-retry disabled.
  - Integration-style Flutter tests using fake/no-op Sentry transport or injectable reporter to verify startup with and without DSN, staging/production configuration, and explicit local opt-in.
  - Static/regression checks for forbidden user-facing `error.toString()`, direct feature imports of Sentry SDK APIs, committed Sentry secrets, disabled tracing/profiling/metrics/session replay, and required Sentry Dart Plugin release-symbolication configuration.
- Blocking open questions: None.
