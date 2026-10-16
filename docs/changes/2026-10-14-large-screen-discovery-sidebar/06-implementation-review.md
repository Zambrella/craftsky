# Implementation Review: Large-screen discovery sidebar

## Verdict

Status: Changes required

Reviewer: Codex

Date: 2026-10-15

Risk level: Medium

## Summary

The implementation follows the planned scope and has substantial passing automated evidence. Preferred craft navigation, account isolation, independent section states, deterministic member selection, server-side exclusions and the narrow API response are implemented. One relationship-consistency defect prevents approval: accepting a block only invalidates the member read, so a stale AppView response can immediately redisplay the blocked member while Tap catches up.

Reviewed inputs: `01-requirements.md` through `05-implementation-plan.md`, tracked diffs, new untracked source/tests and generated companions, existing mutation/overlay paths, recorded verification logs and a temporary review reproduction. This stage changes only this review artifact; no implementation fixes, commits, push or deployment were performed.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-001 | Important | Behavior / Tests | A successfully blocked recent member remains visible when the subsequent discovery read precedes block projection. `recentMembers` returns repository rows unchanged and the sidebar renders every returned row. Block publication invalidates the read but does not compose accepted block state. The existing test conceals this race by changing its fake discovery result synchronously inside `onBlock`. | RULE-002, FR-007; AC-007/AC-009; AT-007, IT-003; `04-coding-plan.md` section 6; `app/lib/profile/providers/recent_members_provider.dart:22`; `app/lib/router/widgets/desktop_discovery_sidebar.dart:120`; `app/lib/profile/providers/toggle_block_profile_provider.dart:108`; `app/test/router/app_shell_discovery_sidebar_test.dart:542` | Add a failing accepted-block test whose refreshed discovery response still contains the target, then honor the active account's accepted block state until projection converges. Preserve server-side eligibility before limit, lease isolation and reconciliation behavior. Do not treat omitted relationship flags in this API as authoritative confirmation that the block is absent. Record any necessary design adjustment in the next authorized TDD pass. |

### IR-001 reproduction and cause

1. Display a recent member and accept a block through the existing `toggleBlockProfileProvider` path.
2. Keep the next discovery response unchanged, modeling a PDS-accepted block that has not reached the AppView aggregate.
3. Expect the blocked member to disappear. The review probe fails: one `ProfileAccountListTile` remains instead of zero.

The probe copies the current widget suite into `/tmp/discovery_review_stale_block_test.dart`, resolves its relative test imports and changes only the block fake so it no longer synchronously hides the target from discovery. It runs the existing `AT-007 accepted local block refreshes current member eligibility` assertion. Repository source and tests are untouched. Output: `/tmp/discovery-review-stale-block.log`, exit 1, expected no matching tiles / found one. This demonstrates the client behavior under a delayed projection; it is not a measurement of production ingestion latency.

The modeled delay is supported by the actual mutation path: `appview/internal/api/block_commands.go:194` constructs an accepted response from the authoritative PDS record; `appview/internal/pdscommands/set_command_service.go:391` completes the command after that read; `appview/internal/pdscommands/store.go:509` persists dispatch/command completion. Discovery instead reads `pds_set_aggregates` in `recent_members_store.go`. The client already has accepted block overlays in `block_profile_overlay.dart` and the shared operation controller, but this new surface does not consume them. Scheduled invalidations alone cannot prevent the stale response from being displayed.

The correction must retain SQL selection of five eligible members, rather than replacing server eligibility with filtering a fixed five-row response. If a temporary client overlay is used to bridge projection lag, define refill and convergence behavior explicitly and keep the authoritative server selection intact.

## Requirement And Test Traceability

- Requirements implemented: BR-001/FR-002 via preferred-only catalog links; FR-003 via optional typed route query, mounted tab updates and explicit selection preservation; FR-004 via the existing editor/save overlay; FR-001 via the unchanged shared-shell breakpoint and destinations.
- Member behavior: BR-002/FR-005/RULE-001 via fixed-five SQL ordered by onboarding completion and C-collated DID; missing completion metadata is omitted. Lifecycle/onboarding integration tests exercise repeat completion, index replay, reactivation and private cleanup without altering those systems.
- Safety/API: RULE-002 is implemented for indexed server state, with the accepted-block gap above. RULE-003 uses authenticated current-member reads, a dedicated public-field DTO, standard envelopes and no-store success responses. Full-mux tests check the allowlist after decorators; serialized diagnostic tests check permitted causes and private canaries.
- Async/UI: FR-006/FR-007 have independent loading/error/retry, disabled spinning retry controls, current-lease member reads, previous-account suppression and profile-save updates. NFR-001 has automated keyboard, semantics, enlarged-text and scrolling coverage.
- Tests implemented: AT-001–AT-008, UT-001–UT-002, IT-001–IT-006 and REG-001–REG-004 have corresponding feature/regression coverage. The local block variant needs the delayed-projection case; current passing evidence does not cover it.
- Plan alignment: Retaining filters/sort on route-driven craft changes is explicitly planned in section 7, so it is not a regression finding. Tests placed in the new real-router suite instead of modifying the existing Projects suite still cover route validation/reordering. Settings unmute reuses its existing guarded publication path. The endpoint-specific presentation-hydrator exception protects the planned field allowlist while retaining authorization middleware.
- Unplanned behavior: None identified. No dependency, migration, lexicon, infrastructure, onboarding or lifecycle source changes are included.
- Remaining gaps: MAN-001 live iPad/browser/platform gestures and MAN-002 iPad VoiceOver remain pending. They are explicitly disclosed in `05` and classified as non-blocking platform evidence in CPQ-004. Synthetic renders and widget semantics do not prove either manual check. Legacy production completion coverage remains unverified under ASM-002/CPQ-001; no production inspection or backfill is implied.

## Test Evidence

- Reviewed passing logs from the completed implementation: `/tmp/discovery-final-flutter-tests.log` reports **160 passed** across sidebar, preferred links, Projects, profile client, shell/navigation/account-switcher, block/mute and settings suites; `/tmp/discovery-analyze-final.log` reports **No issues found**.
- Backend: `/tmp/discovery-whitespace-green.log` records focused recent-member route/store/diagnostic tests on the local migrated database. `/tmp/craftsky-discovery-appview-check-verified.log` ends with **all release gates passed**, superseding earlier checkpoints. The complete gate includes real-schema service checks, race/shuffle/static checks, Docker/smoke/health and the pinned Tap handshake. No synthetic Tap event injection is claimed by that gate.
- Commands reviewed: `flutter test --no-pub` with the ten suites listed in `05`; `flutter analyze`; focused `go -C appview test ./internal/routes ./internal/api -run '^TestRecentMembers' -count=1` with the local required database; `APPVIEW_CHECK_ARTIFACT_DIR=/tmp/craftsky-discovery-appview-check-verified just appview-check`.
- Executed in this review: the temporary stale-block reproduction above, using `flutter test --no-pub /tmp/discovery_review_stale_block_test.dart --plain-name 'AT-007 accepted local block refreshes current member eligibility'`; **failed meaningfully** at the retained-row assertion. `git diff --check` passed.
- Existing green suites and the full backend gate were inspected, not rerun in this source-unchanged review. The new failing probe adds a missing scenario; it does not contradict the recorded existing-suite results.

## Risk Review

- Risk level: Medium.
- Blocking risk: A newly blocked account can reappear in a safety-sensitive discovery surface after the app has accepted the user's action. Resolve IR-001 before merge/handoff approval.
- Other risks: Real-device presentation/VoiceOver, legacy metadata coverage and query performance at future scale retain their documented evidence limits. No current production performance claim is made.
- Approval notes: Prior implementation authorization remains valid; this review does not authorize a production mutation. No commits are enabled for this workflow.

## UI Polish Recommendation

- Recommendation: Not needed.
- Reason: The two-card composition, existing craft/profile components and independent state copy are coherent. Automated constrained-layout checks and the inspected short-height enlarged-text render provide no concrete polish-only defect to assign. Actual platform quality remains a manual evidence gap.
- Suggested polish notes: None. IR-001 is behavioral and belongs in TDD, not polish.

## Handoff Back To TDD Builder

- Required fixes: IR-001 only. Correct accepted-block visibility under delayed projection and update `05` with the red/green evidence and the narrowed safety claim. Amend `04` only if the chosen consistency mechanism changes its design.
- Suggested next failing test: Preserve the current member response after accepted block; verify the row remains hidden across repeated stale reads, then projection convergence. Include failed block, unblock/reblock and account-switch controls to protect existing overlay ownership and prevent cross-account suppression. If the correction changes list selection/refill, test a sixth eligible member filling the vacated slot and preserve the endpoint allowlist.
- Verification to rerun: Focused new race scenario; sidebar, block overlay/controller, profile relationship and account-switch regressions; Flutter analysis. Rerun recent-member database/route tests and the full `just appview-check` gate if backend behavior changes. Repeat broader checks only as justified by the correction.
- Stage exit: Stop for the user's choice to address required changes, add a manual comment, run UI polish or do nothing. No automatic correction pass follows this review.
