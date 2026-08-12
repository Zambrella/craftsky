# Implementation Review: Settings Page And Lean Account Deletion

## Verdict

Status: Changes required
Reviewer: Codex
Date: 2026-08-11
Risk level: High

## Summary

The expanded Settings, About, Account, Notifications, switching, cache, version, Sign out, fresh-OAuth confirmation, owner/namespace/blob boundaries, durable worker, pending-login denial, and immediate client handoff are substantially implemented. Static analysis and both complete automated suites are green, and the approved simplification removes most of the prior status/receipt/checkpoint/audit design.

The implementation is not ready to merge yet. The PDS loop can delete the membership profile before records skipped by an earlier paginated collection are removed, contradicting the explicit membership-last rule. Deletion-specific metric APIs also remain despite the approved removal, and two Must-level evidence claims rely on deleted or test-only artifacts rather than exercising the production paths. These are focused corrections; they do not require restoring indexer receipts, status UI, checkpoints, or an audit.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-001 | Important | Behavior | The repeated full-scan algorithm processes the membership collection at the end of every pass, not after every other collection is proven empty. With three posts, a batch size of two, and a deletion-mutated cursor, the first pass can skip the third post, delete `social.craftsky.actor.profile`, and only delete the skipped post on the next pass. The existing test detects the rescan but asserts only that the final *list call* is for the profile, so it does not protect membership-last deletion order. | FR-015; RULE-005; AC-015, AC-030, AC-037; UT-009, UT-010, IT-010; `appview/internal/accountdeletion/pds_deleter.go:30-73`; `appview/internal/accountdeletion/pds_deleter_test.go:34-71` | Add a failing paginated test that records delete order and requires the membership profile to be the final successful deletion. Converge all non-membership collections to empty first, then delete membership, then perform the required final empty scan across the complete registry. |
| IR-002 | Important | Traceability | The approved design and implementation log say deletion-specific metrics were removed, but the shared `MetricRecorder` still exposes `AccountDeletion`, and both in-memory and Sentry implementations still emit `craftsky_appview_account_deletion_*` metrics. The code is currently unused, but it is retained API and implementation surface that directly contradicts the simplification contract and the checked completion claim. | NG-006; FR-025; `01-requirements.md` sections 16 and 18; `04-coding-plan.md` sections 3 and 4; `05-implementation-plan.md:97-114`; `appview/internal/observability/metric_recorder.go:51,85,268-278,495-505,669-680` | Remove the deletion-specific recorder method, implementations, metric names, and attribute helper. Keep the worker's ordinary redacted structured logs. Correct the implementation-plan evidence after the residue is gone. |
| IR-003 | Important | Tests | UT-014 does not exercise the production local cleanup implementation. `account_deletion_local_cleanup_test.dart` checks a standalone enum/set that is never read by production, while `accountProductDataCleanerProvider`—the code that actually deletes draft/staged files, Instagram verification state, and both image caches—has no focused test. Coordinator tests prove only that an injected callback is invoked. A green test can therefore coexist with a broken confirming-device cleanup path. | FR-022; AC-036, AC-043; UT-014; `app/lib/settings/models/account_deletion_local_cleanup.dart`; `app/test/settings/account_deletion_local_cleanup_test.dart`; `app/lib/settings/services/account_product_data_cleaner.dart`; `app/test/settings/account_deletion_controller_test.dart` | Replace the test-only cleanup plan with production-bound tests using injectable file/storage/cache adapters. Prove every cleanup is attempted, failures do not prevent later cleanup attempts or ordinary-session removal, and the first error is reported only after the accepted account is safely removed. Remove the unused plan model if it has no runtime purpose. |
| IR-004 | Important | Traceability | The acceptance specification and execution log overstate the surviving test evidence. UT-012 and UT-024 still name deleted test targets; IT-016 and IT-032 still name deleted `worker_retry_test.go`; IT-033 claims obsolete status/retry/recovery routes are explicitly unregistered, but surviving route tests do not request those paths; and UT-011 primarily tests an `OAuthBinding` helper used only by its own test rather than the production `Store`/lifecycle path. The full suites pass, but those passes do not establish every recorded Must assertion. | AC-038–AC-042, AC-046; UT-011, UT-012, UT-024; IT-016, IT-032, IT-033; `02-acceptance-tests.md:124-153`; `05-implementation-plan.md:93-100`; `appview/internal/accountdeletion/oauth_binding.go`; `appview/internal/accountdeletion/oauth_binding_test.go`; `appview/internal/routes/account_deletion_test.go` | Move the required assertions into surviving production-bound tests: exercise Store/worker OAuth authorization and capped failure scheduling, request every removed endpoint and require 404, and point MRU/removal evidence at the coordinator/controller tests. Remove dead helper/test abstractions and update `02-acceptance-tests.md` plus `05-implementation-plan.md` to name the tests that actually provide coverage. |

## Requirement And Test Traceability

- Requirements implemented: Settings identity and hierarchy, canonical destinations, Notifications, About/legal/cache/version, Account, error-coloured Sign out, fresh deletion OAuth, exact-handle confirmation, atomic operation/OAuth binding, immediate local account handoff, private cleanup composition, automatic retry, pending-login denial, owner/namespace/blob restrictions, independent indexer convergence, and terminal operation/OAuth removal.
- Tests implemented: Focused Settings/router/accessibility tests; migration/store/auth/private-cleanup/PDS/worker tests; controller/coordinator/repository/401 tests; full Go and Flutter suites.
- Unplanned behavior: No eager AppView purge, whole-account deletion, direct blob deletion, non-CraftSky record deletion, status UI, manual Retry, checkpoint, receipt, or audit behavior was identified. The remaining deletion-metric API is superseded residue rather than an active product behavior.
- Remaining gaps: IR-001 through IR-004. Disposable real OAuth/PDS validation and responsive visual QA remain manual release gates.

## Test Evidence

- Commands reviewed: `go test ./... -count=1`; `dart analyze`; focused Settings/copy/AuthComplete tests; `flutter test --reporter compact`; changed-file `gofmt -l`; `git diff --check`.
- Passing evidence: All AppView packages passed; Dart analysis reported no issues; all 1,449 Flutter tests passed; changed Go files were formatted; `git diff --check` was clean.
- Failing or skipped tests: No automated command is currently failing. MAN-001 through MAN-003 and the disposable real-PDS/OAuth release smoke were not run. IR-001, IR-003, and IR-004 identify missing counterexample/production-path coverage despite the green suites.

## Risk Review

- Risk level: High.
- Risk notes: The feature permanently deletes owner-scoped federated and private data. The owner/DID/namespace/blob boundary remains narrow, but membership ordering is an explicit destructive-sequencing rule and must be corrected before handoff. Test evidence must bind to production paths because test-only policy objects can mask regressions in irreversible cleanup.
- Approval notes: Not approved for merge or release handoff until IR-001 through IR-004 are addressed. The simplification itself remains approved; none of the findings requires reintroducing indexer convergence receipts, status credentials/UI, manual recovery, checkpoints, an audit, or detailed metrics.

## UI Polish Recommendation

- Recommendation: Optional
- Reason: The user-facing information architecture, destructive colouring, disclosure affordances, localization, and accessibility tests are coherent. Responsive visual QA was not run, so a small polish pass may still be useful after behavioral corrections.
- Suggested polish notes: Check the long deletion warning and exact-handle dialog at large text sizes, compact heights, dark theme, and RTL; verify the identity header and switch-account row align with the existing drawer/navigation-rail identity treatment.

## Handoff Back To TDD Builder

- Required fixes: Address IR-001 through IR-004 without expanding the approved lean architecture.
- Suggested next failing test: Start with the smallest behavioral counterexample: seed a membership profile plus three earlier-collection records, use a deletion-mutating paginated fake with batch size two, and assert the profile is the final delete call. Then add production-bound local-cleanup failure/continuation coverage and removed-route 404 coverage before deleting metric/dead-test residue and correcting traceability documents.
- Verification to rerun: Focused PDS deleter, worker/store/auth/routes, and Flutter cleanup/controller tests; `go test ./... -count=1`; `dart analyze`; `flutter test --reporter compact`; changed-file `gofmt -l`; `git diff --check`. Keep real destructive testing restricted to a disposable development PDS after automated review is green.
