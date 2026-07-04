# Implementation Review: Flutter Error Handling And Sentry Reporting

## Verdict
Status: Approved with notes
Reviewer: Codex
Date: 2026-07-03
Risk level: Medium

## Summary
Re-review confirms the previous blocking findings were addressed. The implementation now wires severe/error `package:logging` records into the app-owned reporter, preserves severe log errors and stack traces through the Sentry reporter boundary, carries safe API diagnostics into provider failure captures, normalizes dynamic AppView paths to bounded endpoint categories, replaces the profile-page raw error surface with localized safe copy, and bounds unnamed provider identity.

The implementation is acceptable for handoff. Remaining notes are non-blocking: real Sentry ingestion/native crash behavior still requires the documented manual staging checks, and the worktree still contains an unrelated `docs/roadmap.md` edit that should not be included accidentally if this change is committed.

## Findings
None identified.

## Requirement And Test Traceability
- Requirements implemented: Sentry startup/configuration and disabled local default (`FR-001`, `AC-001`, `AC-010`, `AC-019`); uncaught Flutter/platform/root-zone capture (`FR-002`, `FR-003`, `AC-002`); logging bridge and safe severe-log reporting (`FR-004`, `AC-003`); privacy-bounded diagnostics, breadcrumbs, and endpoint categories (`FR-005`, `FR-006`, `FR-007`, `FR-020`, `RULE-002`, `RULE-005`, `AC-011`, `AC-012`, `AC-020`); provider reporting and retry disablement (`FR-008`, `FR-009`, `FR-010`, `FR-021`, `AC-008`, `AC-009`, `AC-013`, `AC-016`); finite localized error taxonomy and safe UI surfaces (`BR-002`, `BR-003`, `FR-011` through `FR-015`, `AC-004` through `AC-007`, `AC-021`); support references (`FR-018`, `AC-018`); app-owned reporter/import boundary (`FR-019`, `AC-022`); disabled tracing/profiling/metrics/session replay (`FR-016`, `AC-014`); release symbolication docs/config (`FR-017`, `AC-015`, `AC-017`).
- Tests implemented: Focused tests cover Sentry config/options, reporter guard behavior, Sentry log capture behavior, root log forwarding, error handlers, taxonomy, mapping, copy safety, redaction, breadcrumbs, support references, retry policy, provider logging, API endpoint normalization, startup bootstrap, import boundary, secret scan, release config, l10n, router/settings/profile/profile-project surfaces, and the app harness.
- Unplanned behavior: `docs/roadmap.md` has an unrelated `standard.site` change present in the worktree and was not reviewed as part of this implementation.
- Remaining gaps: Real staging Sentry delivery and native crash symbolication remain manual checks as planned.

## Test Evidence
- Commands reviewed: Implementation plan reports `cd app && dart analyze`, focused settings widget tests, initial full `cd app && flutter test --reporter=compact` with 759 tests, post-review focused regression tests, post-review `dart analyze`, and post-review full Flutter test run with 766 tests.
- Passing evidence: Reviewer reran `cd app && flutter test test/observability/log_bridge_test.dart test/observability/sentry_error_reporter_test.dart test/bootstrap/provider_logger_test.dart test/shared/api/providers/error_mapping_interceptor_test.dart test/profile/profile_page_test.dart --reporter=compact`; all tests passed. Reviewer also reran `cd app && dart analyze`; no issues found.
- Failing or skipped tests: No failing tests observed during re-review. Full suite was not rerun by the reviewer in this pass; the implementation notes report the full suite passing after fixes.

## Risk Review
- Risk level: Medium.
- Risk notes: Privacy-sensitive Sentry context is now centralized and covered by static/unit regressions. Remaining risk is mostly integration risk around real Sentry project configuration, release symbol upload, and native crash ingestion, which cannot be fully proven by local fake-reporter tests.
- Approval notes: Prior blocking issues `IR-001` through `IR-006` are resolved. Commit/stage carefully because unrelated `docs/roadmap.md` work remains in the worktree.

## UI Polish Recommendation
- Recommendation: Not needed
- Reason: UI-facing changes are localized safe-copy substitutions and do not introduce visible rough edges requiring a polish pass.
- Suggested polish notes: None.

## Handoff Back To TDD Builder
- Required fixes: None.
- Suggested next failing test: None required for this stage.
- Verification to rerun: Before merge or PR handoff, rerun `cd app && dart analyze` and `cd app && flutter test --reporter=compact`; perform the documented staging Sentry/manual symbolication smoke check before production release.
