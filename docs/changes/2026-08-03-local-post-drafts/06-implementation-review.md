# Implementation Review: Local Post Drafts And Submit-Time Media Uploads

## Verdict

Status: Approved with notes
Reviewer: Codex
Date: 2026-08-04
Risk level: Medium

## Summary

The implementation is ready for handoff. The third correction pass resolves the final Must-level blocker: a failing screen-awake disable call no longer masks the submission result or leaves the lifecycle permanently running. Release is best-effort, retained ownership remains available for a later run or disposal to retry, and the running guard clears regardless of release outcome. Focused lifecycle coverage now proves successful-result preservation, original-operation-error preservation, retry availability, disposal retry, and the existing success/failure/disposal paths.

Post-review iOS simulator testing exposed one production-only provider-lifecycle bug: valid manifests were written, but the auto-dispose save controller was not retained by either composer and could disappear before returning success. Both standard and project composer widget trees now listen to the active account's save controller. A production-shaped delayed-save widget test failed before that listener and passes afterward.

A subsequent immediate-post failure exposed a second runtime issue: the shared Dio client's 15-second response timeout canceled a valid 7,251,552-byte media upload before the composer's approved one-minute per-transfer timeout. Immediate blob uploads and scheduled media staging now disable that inherited request timeout while retaining their existing one-minute outer timers and cancellation tokens. A real-network delayed-response regression failed before the change and passes afterward; scheduled staging request options are covered directly.

The earlier filesystem-containment, account-ownership, damaged-media recovery, retry revisioning, pre-submit network, privacy, retention, scheduled-wire, and integration-evidence findings remain resolved. The complete Flutter suite passes 1,287 tests, the 58-test focused media/submission regression group passes, static analysis reports no issues, and diff hygiene is clean.

One non-blocking Should-level coverage note remains: the four-media asynchronous test covers provider save progress but not the full Drafts-page widget save/open path and visible progress treatment. `05-implementation-plan.md` now records that limitation explicitly under `MAN-003` rather than claiming complete `IT-018` coverage.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-012 | Suggestion | Tests / Traceability | The four-media asynchronous evidence remains provider-level. It does not reopen through delayed media reads or exercise the complete widget save/open path with visible progress and continued frame pumping. This is non-blocking because `NFR-004` is a Should requirement and the limitation is now accurately recorded. | NFR-004, AC-020, AT-020, IT-018; `app/test/drafts/providers/draft_save_controller_test.dart`; `05-implementation-plan.md` R8 and remaining `MAN-003` note | Optionally add the planned four-media Drafts-page widget save/open harness during polish or release hardening. No correction is required for approval. |

## Requirement And Test Traceability

- Requirements implemented: All Must requirements are represented by implementation and passing automated evidence. Draft persistence, account isolation, explicit save/resume, submit-time immediate and scheduled media transfer, atomic recovery behavior, the blocking overlay, screen-awake cleanup, retry state, privacy, retention, and stale-account fencing match the approved requirements.
- Tests implemented: Unit, repository, provider, widget, API-contract, real-network client, and regression coverage exists for the Must paths. `IT-013` includes disable-failure terminal-result and retained-release-retry coverage in addition to success, failure, and mid-run disposal. The post-review widget regression covers the real unobserved auto-dispose mutation failure that the provider-only harness missed. The post-review HTTP regression proves that immediate uploads can outlive the shared client timeout, while scheduled request inspection proves the same override is applied to staging.
- Unplanned behavior: None identified. No AppView route, database, worker, lexicon, PDS draft record, remote draft-storage surface, or new serialization format was added.
- Remaining gaps: `IR-012` is a documented non-blocking Should-level coverage gap. `MAN-001` through `MAN-004` remain physical-device/release checks.

## Test Evidence

- Commands reviewed:
  - `flutter test test/feed/composer/submission_screen_awake_test.dart --reporter compact`
  - Draft save controller plus standard/project composer regression suites
  - `flutter test test/feed/data/post_api_client_test.dart test/scheduled_posts/scheduled_post_api_client_test.dart --reporter expanded`
  - Immediate API client/uploader/coordinator and scheduled client/edit regression group
  - Coordinator, screen-awake, overlay, standard composer, project composer, and scheduled-submission regression group recorded in `05-implementation-plan.md`
  - `just app-test --reporter compact`
  - `just app-analyze`
  - `git diff --check`
- Passing evidence: The focused lifecycle suite passed 4 tests during this review. The post-review controller/composer group passed 13 tests. The submission regression group passed 22 tests. The media-timeout API client group passed 37 tests, and its broader media/submission group passed 58 tests. The full Flutter suite passed 1,287 tests. Static analysis reported no issues and current diff hygiene is clean.
- Failing or skipped tests: No executed automated command is failing. The optional complete `IT-018` widget scenario is not implemented. `MAN-001` through `MAN-004` have not been run.

## Risk Review

- Risk level: Medium because the change introduces private persistent media storage and alters every composer submission boundary, with physical-device filesystem and wakelock behavior still requiring release checks.
- Risk notes: Automated coverage now protects filesystem containment, immutable/atomic draft updates, unpublished-content privacy, account binding, source-loss recovery, submit-time transfer, partial-upload retry, one-minute transfer budgets at both the coordinator and HTTP-client layers, scheduled wire compatibility, overlay blocking, and cleanup failures. Screen-awake adapter failure is localized and handled without changing the authoritative submission result.
- Approval notes: No code correction is required before handoff. Complete `MAN-001` through `MAN-004` before release; consider the optional `IR-012` widget scenario while performing `MAN-003`.

## UI Polish Recommendation

- Recommendation: Optional
- Reason: The Drafts page, damaged-media recovery controls, explicit save/close actions, and blocking overlay are coherent in widget coverage. This is a broad new visible surface, so a focused visual/accessibility pass could still improve confidence without changing behavior.
- Suggested polish notes: Check Drafts rows, `Image unavailable` and Replace image treatment, maximum-media composer layout, overlay text scaling, both themes, and screen-reader announcements while completing `MAN-003`.

## Handoff Back To TDD Builder

- Required fixes: None.
- Suggested next failing test: Optional only — mount the Drafts-page/composer flow with four media items, delay persistence and media reads, verify visible bounded progress while frames continue to pump, then reopen and verify all four ordered items.
- Verification to rerun: No additional automated correction gate is required. Before release, run `MAN-001` through `MAN-004`; rerun the full Flutter suite, analyzer, and diff hygiene if any further source or test changes are made.
