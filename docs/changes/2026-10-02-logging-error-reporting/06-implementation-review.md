> Correction update 2026-10-07: IR-005 and IR-006 have been addressed in TDD loops C8–C10, with permanent SDK-output regressions and passing focused/full suites. This artifact retains the pre-correction review verdict and reproduced findings; a new implementation review has not yet been performed. See `05-implementation-plan.md` and `06-validation-evidence.md` for correction evidence.

# Implementation Review: Logging and Error Reporting

## Verdict

Status: Changes required
Reviewer: Codex
Date: 2026-10-07
Risk level: High

## Summary

The approved SIM-001–SIM-006 refinement substantially simplifies the implementation: Flutter owns UI messages, logs export independently of explicit issue capture, console output uses one brief record, and SDK hooks filter native events without provenance registries or wholesale reconstruction. The implementation follows this direction. Two narrow privacy omissions remain in the final SDK boundaries: Go thread frames and Flutter scope attachments.

These findings do not require restoring the removed architecture. Fix the remaining fields/envelope items at the existing boundary and add serialized-output regressions. The existing focused suites pass, but their enrichment canaries do not exercise these channels.

This review changes only this artifact. Probes live in `/private/tmp`; the Go probe uses an overlay without modifying repository tests. No source correction, dependency/configuration change, production operation, commit or push was performed. Plannotator previously closed without feedback; that was not approval.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-005 | Important | Risk / Tests | Go `protectSDKEvent` sanitizes exception frames but leaves `event.Threads` untouched. Thread stack locals, source lines and absolute paths survive actual SDK capture and serialization. | SIM-004, SIM-005; FR-010 / RULE-001–003; SIM-T05, AT-007, IT-011; `appview/internal/observability/diagnostic_final.go:19–63`; `appview/internal/observability/diagnostic_test.go:273` | Omit thread data or sanitize every thread frame using the existing frame protection. Add a permanent SDK-transport regression with private locals/source/path canaries and retained exception type/useful frame/release positives. |
| IR-006 | Important | Risk / Tests | Flutter clears hint attachments in `beforeSend`, but the SDK assembles scope attachments afterwards. They survive into both issue and transaction envelopes. | SIM-004, SIM-006; FR-010 / RULE-001–003; SIM-T04, AT-007, IT-011; `app/lib/shared/observability/sentry_error_reporter.dart:275–297`; `app/test/observability/sdk_emission_test.dart:619`; installed Sentry 9.23.0 `sentry_client.dart:187–217,456–464` | Block scope attachment envelope items as well as hint screenshot/view-hierarchy/attachment items. Use a small SDK configuration or final envelope boundary appropriate to the installed SDK; do not rebuild events or restore provenance tracking. Add permanent actual-envelope tests for issues and transactions with allowed metadata/type/stack positives. |

### IR-005 reproduction

A temporary test creates the production Observer with `sentry.MockTransport`, then calls its SDK hub's `CaptureEvent`. The event has a `ReviewFailure` exception and a thread containing `readRecord`, an absolute path, a `draft` local and a source line. After flush, the transport events are JSON-serialized. `ReviewFailure` and `review-release` remain; exception prose is removed, but all three thread canaries remain:

- `private-thread-canary`
- `private-path-canary`
- `private-source-canary`

Command from `appview/`:

```sh
go test -overlay /private/tmp/refinement-review-overlay.json ./internal/observability -run TestReviewThreadPrivacy -count=1
```

Result: fails three protected-value assertions. Transcript: `/private/tmp/refinement-review-thread-probe.log`. Probe: `/private/tmp/refinement_review_probe_test.go`. This is synthetic test data, not a claim that existing production events contain private thread values.

### IR-006 reproduction

A temporary Flutter test initializes the real SDK with the production `configureDiagnosticOptions` callbacks and a serialized test transport. It adds one scope attachment containing `private-attachment-canary`, with `addToTransactions: true`, then captures a `StateError` with a supplied `readRecord` frame and finishes a `post.read` transaction. Both serialized envelopes contain the attachment bytes. The exception's prose is correctly replaced; exception type, stack, release and transaction operation survive.

The installed SDK collects `scope.attachments` after the event callback and separately collects transaction-enabled scope attachments after the transaction callback. Clearing `hint.attachments` cannot filter either collection.

Command from `app/`:

```sh
flutter test --no-pub /private/tmp/refinement_review_sdk_test.dart --plain-name 'review scope attachments excluded from SDK envelope'
```

Result: fails the attachment absence assertion; the canary appears twice, once after each event payload. Transcript: `/private/tmp/refinement-review-attachment-probe.log`. Probe: `/private/tmp/refinement_review_sdk_test.dart`. Transactions remain disabled by production Flutter bootstrap; the issue-envelope bypass also occurs with ordinary issue capture. This is a boundary coverage finding, not evidence of a production disclosure.

## Requirement And Test Traceability

- Requirements implemented: SIM-001 client scalar correlation and local messages; SIM-002 log/capture separation and explicit consumed owners; SIM-003 brief parseable console output; SIM-004 native SDK exception/stack filtering; SIM-005 static messages and local/source selection; SIM-006 guide and broad evidence are represented in source and the refined R1–R6 execution record.
- Tests implemented: mapper/code/correlation fixtures, actual SDK issue/log/trace serialization, provider/unhandled and consumed-failure owners, account-switch attribution, platform/product output, private worker/PDS boundaries and guide examples. Existing enrichment tests cover exception frames, request/user/custom contexts and selected attributes, but omit SDK thread frames and scope attachment items.
- Unplanned behavior: no API route/envelope, lexicon, schema, dependency, infrastructure or business-policy change identified. Injected guarded reporters retain existing storage, caption, device, picker and stale-account fallback behavior. Logs no longer capture implicitly, as explicitly approved.
- Remaining gaps: IR-005 and IR-006 violate retained private-data exclusions. SIM-T04/SIM-T05/SIM-T06 cannot yet be considered fully accepted despite completed implementation-loop labels.
- Superseded history: the earlier IR-003 complete chunk/6,500-byte parity finding is superseded by SIM-003 and the fixed brief schema. Earlier IR-001 product output, IR-002 scheduled PDS privacy and IR-004 original media causes remain covered by retained source/tests. No obsolete catalogue, ownership marker, generic Flutter retry-window or aggregate SDK-envelope parity requirement is reopened.
- Manual gaps: physical iOS/Android release-console retrieval and live production retrieval/access/retention remain pending as explicitly separated rollout checks. Local synthetic evidence and product AOT coverage do not close them.

## Test Evidence

### Current review execution

| Check | Result |
|---|---|
| `go test -race ./internal/observability ./internal/middleware -count=1` | Passed. Transcript `/private/tmp/refinement-review-go-focused.log`. Initial sandbox attempt could not bind an httptest loopback port; rerun with local-port permission passed. |
| Flutter SDK emission, platform diagnostics, product AOT adapter, ProviderLogger and error-mapping suites | 34 passed. Transcript `/private/tmp/refinement-review-flutter-focused.log`. |
| Go SDK thread privacy probe | Failed: three retained thread canaries; positive exception type/release assertions passed. |
| Isolated Flutter scope attachment probe | Failed: attachment bytes in issue and transaction envelopes; positive type/release/operation assertions passed; supplied frame observed in output. |
| `git diff --check` | Passed. |

Focused Flutter command:

```sh
flutter test --no-pub test/observability/sdk_emission_test.dart test/observability/platform_diagnostic_test.dart test/observability/product_platform_log_test.dart test/bootstrap/provider_logger_test.dart test/shared/api/providers/error_mapping_interceptor_test.dart
```

An exploratory map-form SDK context probe produced no event and is inconclusive; it is not an additional finding. Repository tests and source were not modified to run the probes.

### Implementation-stage evidence inspected

- Full Flutter suite: 2708 passed / 38 skipped, `/private/tmp/refine-flutter-final.log`.
- Flutter analysis: no issues found, `/private/tmp/refine-analyze.log`.
- Full Go race/integration suite with local PostgreSQL/MinIO: passed, `/private/tmp/refine-go-final.log`.
- Refined paired synthetic Go API envelope and Flutter transport artifacts retain request correlation independently of sampled tracing: `evidence/refined-2026-10-07/` and recorded paired commands.
- Broader suites were not repeated during this source-read-only review. Their passing evidence does not override the new counterexamples. Pending device/live checks and ordinary suite skips remain explicit.

## Risk Review

- Risk level: High because diagnostic privacy crosses both runtimes and multiple SDK payload channels.
- Risk notes: retaining native SDK structures is consistent with the approved simpler design, but fields outside the exception list and envelope items assembled after callbacks need explicit coverage. These defects have small correction scope.
- Approval notes: both findings require correction before merge/handoff. Keep source-selected privacy and existing business behavior intact; do not introduce a global registry, event reconstruction or runtime ownership flags. Production data/settings/deployment remain outside this task.

## UI Polish Recommendation

- Recommendation: Not needed.
- Reason: the refinement concerns diagnostics and client/server separation; no visible styling, layout or copy issue was identified.
- Suggested polish notes: none.

## Handoff Back To TDD Builder

- Required fixes: IR-005 Go thread-field protection; IR-006 Flutter scope/envelope attachment exclusion.
- Suggested next failing tests: promote the two confirmed temporary probes into permanent serialization regressions. Separate issue/transaction attachment assertions, retain native exception types and supplied useful frames, and verify normal metadata/correlation survives. Cover hint attachments/screenshots/view hierarchies alongside scope items where the SDK supports them.
- Verification to rerun: affected Go observability/middleware suites with race detection; affected Flutter SDK/bootstrap/ownership/platform tests and analysis. Then run full Go/Flutter suites after corrections and update implementation/validation evidence. No production telemetry is necessary.
- Return for implementation review after correction. This review does not authorize a commit, push or deployment.
