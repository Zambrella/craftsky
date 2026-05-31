# Implementation Review: Moderation Flow MVP Without Live Ozone/PDS Report Submission

## Verdict
Status: Changes required
Reviewer: gpt-5.5 implementation reviewer
Date: 2026-05-31
Risk level: High

## Summary
Re-reviewed the implementation after the review-fix commit `553a038 fix: address moderation flow review`. The previous blocking findings are largely addressed: the dev synthetic moderation route now persists valid trusted outputs, handle-based profile reports for indexed Craftsky profiles are covered, profile-authored post/comment and thread/comment filtering gained tests, and the focused AppView, full AppView, and focused Flutter test commands pass.

However, a Must profile-enforcement gap remains. `ProfileStore.Read` applies account hide/takedown filtering only to the `craftsky_profiles` query, then falls back to `readNonCraftsky` when that filtered query returns no rows. A hidden/taken-down Craftsky account that also has a `bluesky_profiles` cache row, or a hidden/taken-down non-Craftsky account, can therefore still be returned as a non-Craftsky profile. The same fallback path also misses account-level warn metadata. Related profile report eligibility is still limited to `craftsky_profiles`, even though the profile page can display non-Craftsky profiles from `bluesky_profiles`/hydration.

Because this affects required direct profile hide/takedown behavior and account-level profile warnings, the implementation is not ready to merge as complete.

## Findings
| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-005 | Critical | Behavior / Risk | Direct profile moderation can be bypassed through the non-Craftsky fallback path. `ProfileStore.Read` filters hidden/taken-down `craftsky_profiles`, but on `pgx.ErrNoRows` it calls `readNonCraftsky`; if a hidden Craftsky account has a `bluesky_profiles` row, or an account-level output targets a non-Craftsky profile, the profile can be returned instead of `404 profile_not_found`. Warn-only account outputs also are not attached to non-Craftsky profile rows. | `BR-004`, `BR-005`, `FR-016`, `FR-018`, `AC-016`, `AC-018`, `AC-039`; `appview/internal/api/profile_store.go` `Read`, `readNonCraftsky`, `readNonCraftskyCached`; existing `TestProfileStore_ReadByDID_HiddenAccountReturnsNotFound` lacks a `bluesky_profiles` row/fallback case. | Check active account moderation before falling back to non-Craftsky profile reads/hydration. Return `ErrProfileNotFound` for active hide/takedown regardless of Craftsky-vs-Bluesky profile source, and attach generic `profile` warning metadata for warn-only non-Craftsky profiles. Add tests for hidden Craftsky profile with a Bluesky cache row, hidden/taken-down non-Craftsky profile, and warn-only non-Craftsky profile metadata. |
| IR-006 | Important | Behavior / Traceability | Profile report target resolution remains narrower than the profile surface: `ResolveAccountReportTarget` only accepts DIDs present in `craftsky_profiles`. A user can view a non-Craftsky profile through the existing profile read fallback, but reporting that profile by handle/DID will still return `profile_not_found`. | `BR-001`, `FR-002`, `FR-006`, `RULE-005`, `AC-002`, `AC-008`, `AC-044`; `appview/internal/api/profile_store.go` `ResolveAccountReportTarget`, `ProfileReportTargetResolver`; `04-coding-plan.md` §5.2. | Expand report-target validation to the same account-existence sources the profile page can display, while still bypassing moderation visibility for stale/race reports and preserving the submitted handle snapshot. Add tests for reporting a cached/hydratable non-Craftsky profile by handle and DID, plus unresolvable identity failure behavior. |

## Requirement And Test Traceability
- Requirements implemented: Private report persistence, report validation/detail normalization, placeholder forwarding metadata, report route middleware, minimal report responses, dev moderation config/token gating, valid synthetic route persistence, moderation output store/policy semantics, post/timeline/thread/notification hide filtering, warning metadata for Craftsky post/profile responses, Flutter report flows, duplicate-submit prevention, and generic warning rendering.
- Tests implemented: The review-fix loops added coverage for prior findings IR-001 through IR-004: valid synthetic route-to-store persistence, profile report handle resolution for indexed Craftsky profiles, and additional profile-authored/thread/comment filtering tests.
- Unplanned behavior: None identified as a deliberate product expansion.
- Remaining gaps: Account-level profile moderation is incomplete for `readNonCraftsky` fallback paths, and profile reports do not cover profiles that are readable only through the non-Craftsky profile path.

## Test Evidence
- Commands reviewed and run:
  - From `appview/`: `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/api ./internal/routes ./internal/app`
  - From repo root: `just test`
  - From `app/`: `flutter test test/feed test/profile test/moderation test/notifications`
  - Repository state/history: `git status --short`, `git log --oneline -10`, `git show --stat HEAD`
- Passing evidence:
  - Focused AppView packages passed.
  - Full AppView `just test` race suite passed.
  - Focused Flutter feed/profile/moderation/notifications tests passed.
- Failing or skipped tests:
  - No automated test failed during review.
  - Coverage is missing for the profile fallback moderation gaps above.
  - Manual checks `MAN-001` through `MAN-004` remain not run by this agent and should stay as human/local follow-up.

## Risk Review
- Risk level: High
- Risk notes: This feature controls private safety reports, dev-only moderation mutation, and server-side content suppression. The previous synthetic ingestion and post-list filtering blockers are fixed, but direct profile enforcement still has a bypass for cached/hydrated non-Craftsky profile rows.
- Approval notes: Do not approve until account-level hide/takedown and warn behavior is applied consistently across Craftsky and non-Craftsky profile read paths, and profile report eligibility is aligned with the readable profile surface.

## UI Polish Recommendation
- Recommendation: Optional
- Reason: The feature includes user-facing report sheets and warning banners. The current blocking issues are AppView behavior/traceability issues, not polish-level UI defects.
- Suggested polish notes: After behavioral fixes, a small polish pass could review report sheet spacing, button affordance while submitting, long reason-list scrolling, and warning banner color/contrast. Do not use polish to change report behavior or acceptance criteria.

## Handoff Back To TDD Builder
- Required fixes:
  1. Apply account-level hide/takedown and warn policy before or inside `readNonCraftsky` fallback so direct profile reads cannot bypass moderation.
  2. Add tests for hidden/taken-down Craftsky profiles with Bluesky cache rows, hidden/taken-down non-Craftsky profiles, and warn-only non-Craftsky profiles.
  3. Expand profile report target resolution/tests to cover profiles readable through the non-Craftsky profile path.
- Suggested next failing test: `TestProfileStore_ReadByDID_HiddenAccountWithBlueskyProfileReturnsNotFound`, followed by a non-Craftsky warn metadata test.
- Verification to rerun:
  - From `appview/`: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes ./internal/app`
  - From repo root: `just test`
  - From `app/`: `flutter test test/feed test/profile test/moderation test/notifications`
