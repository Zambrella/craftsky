# Implementation Review: Sentry-led Logging and Error Reporting

## Verdict

Status: Changes required
Reviewer: Codex
Date: 2026-10-07
Risk level: Medium

## Summary

The SDK-001–SDK-006 implementation follows the approved simplification: native exception chains/stacks, official logging integrations, SDK-owned automatic Flutter capture, selected breadcrumbs, reviewed static explanations and a smaller reporter interface. Source selection and final privacy hooks remain, and the existing broad suites pass.

One required correction remains: SDK-owned automatic Flutter events bypass the existing expected-failure classification. Cancellation, offline and expired-session errors now produce issues although explicit capture suppresses them. Keep SDK ownership and apply the existing policy at the automatic SDK boundary; no ownership flags or custom forwarding layer need to return.

This review changed only this artifact. A temporary actual-SDK probe under `/private/tmp` reproduced the regression without editing repository source or tests. No commit, push, deployment or production operation was performed.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-007 | Important | Behavior / Tests | Automatic SDK capture bypasses expected-failure filtering. `SentryErrorReporter.captureException` checks `isExpectedDiagnostic`, but SDK framework/platform/zone integrations capture directly and `beforeSend` only sanitizes. Actual PlatformDispatcher capture exported three issues for ApiCanceled, AppError(networkUnavailable), and ApiUnauthorized; the same explicit ApiCanceled capture correctly returned null with no event. Expected failures become issue noise and undermine the genuine-error goal. | SDK-001, SDK-005, SIM-002 / AC-009; SDK-T04/T05; `app/lib/shared/observability/sentry_error_reporter.dart:241–246`, `app/lib/shared/observability/diagnostic_outcome.dart:10–46`, `app/lib/main.dart:97–110` | Apply expected classification to SDK-owned automatic events using their original typed throwable. Preserve explicit terminal/reportability overrides, normal unexpected capture, local output and rendering fallback. Add real-SDK framework/platform/unhandled regressions for cancellation, offline/expiry, reportable overrides and unexpected failures. |
| IR-008 | Suggestion | Code Quality / Diagnostics | Go log breadcrumbs always use LevelInfo even for warning/error records. Structured Logs retain their correct severity, but an issue's breadcrumb timeline mislabels those records. | SDK-004 / SDK-T03/T05; `appview/internal/observability/logs.go:75–77` | Preserve the source log severity when constructing the breadcrumb. Add a focused actual-SDK assertion for a warning/error breadcrumb. This is non-blocking. |

### IR-007 reproduction

Temporary probe: `/private/tmp/sdk_review_expected_probe_test.dart`. It initializes the actual SentryFlutter SDK with production `configureDiagnosticOptions`, a serialized transport and the local callbacks installed first. Native platform initialization is disabled to avoid device/plugin dependencies, matching the permanent SDK-T04 fixture.

1. Explicit reporter capture of `const ApiCanceled()` returns null and leaves the transport empty.
2. Invoke the installed PlatformDispatcher SDK callback with ApiCanceled, AppError(networkUnavailable), and ApiUnauthorized, supplying a caught upload frame.
3. Allow asynchronous SDK capture to settle, close/flush, decode serialized events and assert no issues.
4. The assertion fails: three events contain the concrete types and `PlatformDispatcher.onError` mechanisms. They remain protected, but should not have become issues.

Command from `app/`:

```sh
flutter test --no-pub /private/tmp/sdk_review_expected_probe_test.dart
```

Result: meaningful behavioral failure. Transcript: `/private/tmp/sdk-review-expected.log`. This uses synthetic errors and an injected transport; no real collector receives events. Permanent SDK-T04 tests cover two distinct unexpected failures only, so they do not expose this regression.

## Requirement And Test Traceability

- SDK-001: SDK ownership and local-only callbacks are implemented; local handlers remain installed through SDK chaining. Expected-failure policy is incomplete at the new automatic owner (IR-007).
- SDK-002: native cause extractors/Go SetException preserve chain structure and attached/supplied stacks. Go stackless errors lose the SDK capture-time stack; panic recovery remains separate. Cyclic graphs have tested bounded fallback. Parallel SDK cause/stack JSON is removed.
- SDK-003: reviewed static DiagnosticStateError/WrapError messages survive selection. Source changes retain StateError subtype catches/Go Unwrap and use literal explanations; reviewed auth/billing/lease guards preserve their conditions and outcomes. Unknown dependency prose stays generic.
- SDK-004: root/shell/tab observers remove arguments before SDK formatting; transactions stay disabled. Selected log, connectivity, lifecycle and HTTP/operation breadcrumbs survive. Go hubs isolate request histories without tracing. Go breadcrumb severity is a non-blocking loss (IR-008).
- SDK-005: official matching-version logging packages are added; Dart issue threshold is OFF, Go handler exports Logs only; custom LogForwarder and unused reporter methods/context cause fields are removed. Explicit expected filtering remains, but automatic filtering lacks required coverage (IR-007).
- SDK-006: approved amendments, implementation/evidence notes and guide agree. Broad tests and analysis pass. The new review probe demonstrates that passing broad evidence does not cover all required automatic expected outcomes.
- Unplanned business/API/lexicon/schema/production changes: none identified. Generated changes match affected providers/router; the regenerated user-profile comments reflect its existing source rather than new write behavior.
- Remaining accepted gaps: device/release console retrieval and live symbolication remain pending. Native platform crash/device behavior is not proved by Dart SDK mock transports. These were explicitly deferred by SDK-006.

## Test Evidence

Commands and actual transcripts reviewed:

- `just test`: all packages passed with PostgreSQL/MinIO and race detector; `/private/tmp/sdk-go-all-green.log`.
- `cd app && flutter test --no-pub`: 2,707 passed / 38 skipped; `/private/tmp/sdk-flutter-full-final.log`.
- Final Flutter capture/sanitizer tests: 19 passed after the snapshot/language correction; `/private/tmp/sdk-flutter-last-focused.log`.
- Flutter observability/router/breadcrumb suite: 243 passed / 8 skipped; `/private/tmp/sdk-flutter-final-focused.log`.
- Final Flutter analyzer: No issues found; `/private/tmp/sdk-analyze-last.log`.
- Go focused observability/middleware race suite: passed; `/private/tmp/sdk-go-complete-focused.log`.
- Reviewer `git diff --check`: passed.
- Reviewer actual-SDK expected-failure probe: FAILED as documented in IR-007. Broad suites were not repeated during review because source/tests were unchanged and their transcripts are available.

Prior failed broad runs and their corrections are recorded in 05/06-validation-evidence: legacy navigation message, cyclic fallback, retry scalar error context, native numeric status and stack metadata. This review does not hide or relabel the separate automatic-classification failure.

## Risk Review

- Risk level: Medium. SDK ownership is simpler, but moves policy enforcement to a boundary not exercised by the current expected-failure tests.
- Privacy: source-selected private/public fields and credential/private-payload canaries remain; attachment transport guard and Go thread exclusion remain intact. Static error subtype changes do not admit arbitrary StateError/dependency prose. No new privacy bypass was reproduced in this review.
- Business controls: lease/auth/purchase/retry/ACK conditions are preserved. No API, lexicon, persistence or infrastructure changes.
- Approval: the maintainer approved the SDK refinement; no extra implementation approval is required for the narrow IR-007 correction once the correction stage is selected.
- Historical findings IR-005/IR-006: addressed in the preceding correction pass; thread and scope-attachment regressions remain covered. This verdict supersedes the old implementation-review verdict for the current worktree.

## UI Polish Recommendation

- Recommendation: Not needed.
- Reason: changes affect diagnostics and preserve existing recovery/rendering behavior; no new UI needs visual polish.

## Handoff Back To TDD Builder

- Required fix: IR-007 only. IR-008 is an optional severity improvement.
- Suggested next failing test: promote the temporary actual-SDK automatic expected-failure probe into permanent SDK-T04 coverage. Assert no issues for canceled/offline/expiry errors, one issue for an explicitly reportable override/unexpected error, retained cause/stack and protected local output. Also retain explicit terminal retry behavior and always suppress cancellation.
- Implementation direction: keep SDK automatic capture and official logging. Filter at the SDK-owned event boundary using the existing expected classifier, without suppressing explicit captures already admitted with terminal context.
- Verification to rerun: actual framework/platform/unhandled ownership cases, expected/terminal outcome matrix, serialized privacy/cause/stack tests, affected Flutter regression suite, Flutter analysis and full suite. Go checks only if Go code changes. For optional IR-008, run Go observability tests with serialized breadcrumb severity assertions.
