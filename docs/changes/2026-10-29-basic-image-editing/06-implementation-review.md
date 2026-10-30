# Implementation Review: Basic Image Editing and Avatar Preview

## Verdict
Status: Approved with notes
Reviewer: Codex
Date: 2026-10-29
Risk level: Medium

## Summary

The implementation satisfies the approved crop/rotate and avatar editing scope. Re-review confirms that IR-001 is corrected: a stale avatar operation now releases its matching temporary busy state, preserves the previously accepted avatar and unrelated form edits, and cannot update a disposed form. No required corrections remain.

The maintainer's no-new-integration-tests decision remains in force. Package geometry/export/platform behavior is trusted; the remaining notes concern unperformed manual checks and optional UI polish.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-002 | Suggestion | Validation | Brief finished-app manual checks remain unperformed. Host test evidence does not establish device/browser appearance, gallery interactions, physical accessibility or manual draft save/reopen behavior. This remains a non-blocking evidence limitation under the approved reliance on package behavior. | `05-implementation-plan.md`; MAN-001–MAN-003; NFR-001/002, FR-007 | Record available manual checks when performed. Do not add an integration suite, browser/device harness or package conformance gate. |

### Resolved finding: IR-001

The initial review found that `_pickProfileImage` returned on stale ownership without releasing `_avatarDraft.isUploading`. A form surviving an activation away/back could retain a disabled Save button and reject another avatar selection.

The correction in `app/lib/profile/pages/edit_profile_dialog.dart` retains the immutable pending draft as its operation identity. Its `finally` block restores the previous accepted draft only when the form is still mounted and that exact pending draft remains active. Normal success, cancellation and error handling already replace the pending draft, so the cleanup does not undo accepted output. It changes only avatar operation state and preserves typed form fields.

Three owning-widget regressions in `app/test/profile/edit_profile_dialog_test.dart` now cover:

- Activation away/back while the form remains mounted: the old result is stale, Save recovers, typed text survives, and another selection can start.
- A previous accepted avatar followed by a stale successful response: the accepted preview/blob survive, no profile write occurs before Save, retry works, and Save receives the accepted blob rather than late output.
- Disposal during a pending selection: the guard becomes false, and late completion produces neither a framework exception nor a restored UI or issue.

The implementation record documents the meaningful red failure and green correction. IR-001 is closed. This re-review changes only the review artifact.

## Requirement And Test Traceability

- Requirements implemented: scoped composer Edit actions and local-byte eligibility (FR-001/RULE-001); restricted shared package tools and circular avatar preview with square export (FR-002/005); atomic accepted output preserving attachment identity/order/alt and fresh draft media mapping (FR-003/007); cancellation/failure preservation and profile save semantics (FR-004/006); existing preparation/upload validation (RULE-002); operation/account fencing and submission guards (RULE-003); localized accessibility/reduced-motion controls (NFR-002); local editing and protected diagnostics (NFR-003). These jointly implement BR-001.
- Tests implemented: attachment entry/eligibility/busy behavior; composer acceptance/cancellation/failure/validation/lease fencing/draft mapping; real-package restricted configuration smoke checks; duplicate completion and export-error route handoff; keyboard, semantics, enlarged text and reduced motion; avatar picker cancellation/local/stale-result boundaries; owning composer/onboarding/Edit Profile wiring; onboarding save guards; serialized diagnostic privacy checks. The correction adds the previously missing owning Edit Profile lifecycle tests under AT-009/UT-005/REG-003.
- Remaining required gaps: none identified.
- Accepted validation gaps: package geometry/export/platform behavior is trusted; MAN-001–MAN-003 remain unperformed. Existing preparation/repository/snapshot tests supply codec and storage coverage; no new cross-layer journey is required.
- Plan deviations: a single transient composer busy boolean replaces an editing-ID set because only one editor can be active; documented and reasonable. A small toolbar awaits package export and forwards errors to the consuming owner. Avatar replacement is displayed after upload succeeds, preserving the last accepted selection during processing. These choices stay within the approved behavior.
- Unplanned behavior: no unrelated backend, API, Lexicon, permission, infrastructure, record-deletion or authentication implementation changes identified. The correction remains confined to avatar operation cleanup and its tests.

## Test Evidence

- Commands reviewed: focused red/green commands and correction notes in `05-implementation-plan.md`; `just app-analyze`; relevant broader `just app-test` commands; `git diff --check`.
- Current correction evidence: all three new owning-widget regressions pass. `just app-test test/profile test/onboarding test/observability/profile_image_error_reporting_test.dart` reports **393 passed**, no failures. The log `/tmp/craftsky-image-editing-ir001-tests.log` ends with that result. Final analysis reports **No issues found**; diff whitespace checks pass.
- Earlier full-suite evidence: before the focused correction, `just app-test` reported **3,523 passed, 38 skipped, no failures**. This is historical evidence, not a claim that the entire app suite was rerun after correction. The relevant broader profile/onboarding/diagnostic gate verifies the correction.
- Initial failures are explained in the implementation record: the original missing-listener test setup was corrected; the IR-001 disabled-Save regression failed meaningfully before the cleanup fix. No current failing relevant test is concealed.
- Review checks: inspected the cleanup and controlled picker fixtures, assertions for accepted preview/save payload, retained typed values, stale activation and disposal, plus the approved requirement/test contract and implementation evidence. No additional test run was needed against unchanged code during this re-review.
- No physical-device, browser-export, manual gallery, screen-reader or live-publication proof claimed. No new integration tests requested or run.

## Risk Review

- Risk level: Medium, unchanged.
- Risk notes: local image handling retains existing bounded normalization/preparation and managed upload clients. Composer changes apply atomically by stable ID and source phase; lease checks include activation generation. Edited media clears old stored origin. Editor navigation uses a static route name and local bytes, with redacted request/result strings. Existing diagnostic owners remain in place.
- Cleanup safety: identity and mounted checks release only the matching temporary avatar state, without introducing a new provider, persistence field, account mutation or cross-operation state reset.
- Dependency: only `pro_image_editor` 14.21.1 was added and pinned; unrelated dependency versions, platform configuration and desktop targets remain untouched.
- Approval notes: no required correction stage remains. Retain the reduced testing scope and accurately report manual validation limits. No commits or push are enabled.

## UI Polish Recommendation

- Recommendation: Optional.
- Reason: a brief visual pass could assess the 48-pixel Edit control over the 64-pixel summary thumbnail and comfortable access to photo details. The tested target is usable; no blocking accessibility or layout issue is identified.
- Suggested polish notes: thumbnail/button balance, contrast and loading-state clarity on narrow screens. Do not change crop behavior, validation or account ownership during a styling pass.

## Handoff Back To TDD Builder

- Required fixes: none. IR-001 is resolved and covered by the owning-widget regressions.
- Suggested next failing test: none required for this review.
- Verification to rerun: none required while implementation remains unchanged; rerun affected unit/widget tests and analysis if a later change warrants it. Manual checks may be recorded separately when available.
- Next stage: optional UI polish, a manual comment, or stop. No implementation or test files were changed by this re-review, and no commits or push were created.
