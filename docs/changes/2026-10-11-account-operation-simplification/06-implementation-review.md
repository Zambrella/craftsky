# Implementation Review: Account Operation Simplification

## Verdict

Status: Approved with notes
Reviewer: Codex
Date: 2026-10-11
Risk level: Medium

## Summary

No blocking correctness, account-isolation, retry-contract or diagnostic regression was identified in the reviewed diff. The shared retry component removes substantive duplication while preserving feature-specific outcomes; captured repositories prevent retries from selecting a replacement dependency. Active authority remains distinct from retained-session authority, request revisions and domain fences.

The implementation follows the approved outline's selective migration strategy. Existing account families and invalidation/cleanup mechanisms were retained where an additional session family would add complexity. The inventory records dispositions for all 80 baseline files and additional specialized areas. Source, tests and implementation notes were not edited during this review.

Two non-blocking follow-ups remain: the planned signed-in-device checks have not been performed, and a few composer changes add indirection or duplicate checks without reducing responsibility.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-001 | Suggestion | Tests / Risk | The approved plan includes manual switching during a slow upload and retrying mutation. These were not performed on a signed-in device. Deterministic completer/provider/widget tests cover the ownership boundaries, but do not establish device-level cancellation and navigation behavior. The limitation is accurately disclosed. | INV-02–03, INV-05; V-02–03, V-06; 04-coding-plan.md section 9; 05-implementation-plan.md Limitations And Handoff | Non-blocking follow-up: exercise standard/project upload and ambiguous retry while switching, including switch-back, and record request, cancellation, navigation and origin-draft outcomes before device-validation sign-off. |
| IR-002 | Suggestion | Code Quality | In both _runSubmission methods, withGuard wraps the existing ownership predicate solely to expose it through another closure; no other context capability is used there. Several scheduling branches also repeat if (!mounted) immediately after a guard that already includes !mounted. These additions are safe but work against the simplification goal. | 04-coding-plan.md sections 2, 5, 7; app/lib/feed/widgets/post_composer_sheet.dart:1307; app/lib/projects/widgets/project_composer_sheet.dart:2209 and :2448 | Optional cleanup: keep the direct ownership callback in the submission adapter unless the context has a concrete shared responsibility there, and remove only the adjacent, identical mounted checks with no intervening async boundary. Preserve the authority check before navigation and the separate widget checks required after awaits. |

Neither finding requires a correction stage before handoff. IR-002 is a code cleanup, not a recommendation to remove account checks generally.

## Requirement And Test Traceability

| Contract | Reviewed implementation and evidence | Assessment |
|---|---|---|
| INV-01; multi-account FR-008; V-01–02, V-04 | ActiveAccountOperation captures the originating lease. Guarded async acquisition rejects session replacement; accountDio continues to select fixed token/session targets. Like/repost/follow/block and bespoke PDS loops capture their repository once. Instagram acceptance captures its repository future. Existing API tests and the new repository-replacement test cover binding and reuse. | Satisfied for the migrated boundaries. No mutable token indirection or new client cache was added. |
| INV-02; FR-007, NFR-001; V-02–03 | Session and activation generations remain in the existing lease predicate. Ref/widget lifetime guards remain effective. The new switch-away/back and disposed-owner tests reject obsolete authority; profile-save tests stop both record retries after switch or same-DID session replacement. | Satisfied. Existing cancellation checkpoints remain; dispatched remote work is not rolled back. |
| INV-03; FR-009, NFR-001; V-03, V-05 | Synchronous publish replaces guarded state assignments. Callers retain continuation guards before dispatch, state changes and navigation. Instagram keeps its typed discard contract; draft save maps only its internal discard to null and preserves original failures. | Satisfied in the reviewed changes. Guarded helper completion is not treated as permission for an unguarded later continuation. |
| INV-04; PDS mutation contract; V-04–05 | PdsMutationRetry reuses the existing budget, clocks, jitter, delay, immutable token and controller.canRetry. requireCurrentToken preserves the prior difference in markAmbiguous handling. Feature adapters keep exhaustion behavior. Request generations, entity revisions, overlays and controller reset were not collapsed into account authority. | Satisfied. Repositories are pinned while immutable keys/inputs remain unchanged. |
| INV-05; V-03, V-06 | Lazy final repository capture keeps the first dependency read inside the original error boundary. Profile records still start independently and produce separate outcomes. Raw ambiguous rethrow versus unresolved exception is preserved. Composer assertions preserve DiagnosticStateError and its message. New tests verify dependency-error cause and late failure stack preservation. | Satisfied. No loading, optimistic presentation, route or visual design change was identified. |
| INV-06; V-07 | Notification recipient/session authority, billing owner/target/SDK/foreground rules, account deletion fences and saved-post boundary epochs remain specialized. Onboarding adopts only the compatible active/retry mechanics. Existing specialized suites are included in the complete run. | Satisfied. No domain guard was replaced with a weaker active-only predicate. |

Unplanned behavior: none identified beyond the explicitly intended rejection of obsolete continuations/dependency replacement covered by the outline.

Remaining gaps: IR-001 and the reported suite skips. No new backend, API, lexicon, schema, dependency or generated-provider change is present. No Riverpod declaration changed that would require regeneration. The linked older feature documents provide preservation contracts; their original full feature implementation/release checklists are not additional scope for this client refactor.

## Test Evidence

The review checked the existing implementation-run logs and inspected the new tests and affected call sites. No source or test changed, so the already successful full run was not repeated solely for the review.

- Complete Flutter command: flutter test --no-pub, equivalent to just app-test with dependency resolution skipped. /private/tmp/craftsky-account-operation-tests-final.log ends with 2,850 passed, 38 skipped, All tests passed. The implementation run returned exit 0.
- Static analysis command: flutter analyze --no-pub. /private/tmp/craftsky-account-operation-analyze-final.log reports No issues found.
- Context/reference-mutation run: /private/tmp/craftsky-account-operation-boundaries.log ends with 26 passed, All tests passed.
- Additional focused run evidence is recorded in 05-implementation-plan.md: 534 passed / 9 skipped for affected features, and 174 passed for retry/outcome adapters.
- git diff --check was rerun during review and passed.
- No final failing test was identified. The 38 skips remain skips and are not included in the pass count. Earlier fixture/import/lint failures are disclosed in the implementation notes; the final logs show successful runs.
- Manual signed-in-device checks were not run; see IR-001. Backend tests were not rerun because no backend or transport contract changed.

The new tests assert meaningful effects: acquisition rejection, no subsequent dispatch/publication after activation changes, the initiating repository's requests across a provider replacement, both profile records stopping after authority loss, and original failure cause/stack preservation. The callback-only continuation test proves publish's contract; existing provider/widget tests supply the integrated account-switch coverage.

## Risk Review

- Risk level: Medium. The diff touches broad asynchronous client flows, so boundary coverage matters even though the changes are local refactoring.
- The original registry/lease validity rules remain authoritative. Null ownership still supports isolated fixtures and does not add a new production anonymous fallback.
- No credentials or private payloads were added to the context, exceptions or diagnostics. Existing fixed-token clients, lease-scoped 401 invalidation and close/cancel behavior remain unchanged.
- The shared retry decision object is part of existing mutation infrastructure, not a competing retry framework. Outcome adapters are intentionally retained where original contracts differ.
- Private storage cleanup, inactive-account work, billing and permanent deletion retain their specialized boundaries. No production infrastructure/data mutation is involved.
- Approval notes: automated verification supports handoff; complete manual device verification before claiming that validation. This approval is not authorization to commit, push or deploy.

## UI Polish Recommendation

- Recommendation: Not needed.
- Reason: The diff changes ownership plumbing and asynchronous control flow, not copy, spacing, styling, layout or accessibility. IR-002 is source cleanup and does not need a visual polish stage.

## Handoff Back To TDD Builder

- Required fixes: None.
- Optional work: IR-002 cleanup, if requested; preserve the same guards and failure contracts.
- Suggested next failing test: None required. If additional boundary behavior is changed, first add an integrated regression for the affected flow rather than another assertion against the guard implementation.
- Verification to rerun for optional composer cleanup: composer coordinator/uploader suites, standard/project composer and scheduling widget suites, Flutter analysis and git diff --check. Broaden again only if further shared behavior changes or a failure warrants it.
- Manual follow-up: IR-001.
- Stage state: review artifact only; no source/test/implementation-document edits, commit, push, deployment or UI polish performed.
