# Implementation Review: Profile Social Summary

## Verdict
Status: Changes required
Reviewer: OpenCode implementation reviewer
Date: 2026-05-27
Risk level: Medium

## Summary
The implementation covers most of the AppView profile-summary contract and the basic Flutter profile/settings UI changes, and focused AppView/Flutter verification commands pass. However, the Flutter list UI only loads the first page for mutual followers, followers, and following. That means users cannot view complete graph lists when the API returns a cursor, which conflicts with the Must requirement to preserve complete follower/following access from settings and the paginated mutuals requirement.

## Findings
| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-001 | Important | Behavior / Tests | The Flutter mutual followers bottom sheet and settings follower/following pages fetch exactly one `ProfileAccountPage` and never use the returned `cursor` to load additional pages. This truncates graph lists at the API default page size and does not satisfy the “complete” follower/following list or paginated mutual-list behavior. The implementation plan also records this gap as “pagination UI currently loads the first page only in this slice.” Existing Flutter tests only assert first-page rendering and do not fail when `cursor` is ignored. | `01-requirements.md` BR-003, FR-011, FR-016, AC-008, AC-009, AC-010, AC-012, AC-015; `02-acceptance-tests.md` UT-010, UT-012, IT-004-IT-006; `04-coding-plan.md` §6 provider behavior and §8 pagination loading; `05-implementation-plan.md` lines 203-206; `app/lib/profile/widgets/profile_mutual_followers_sheet.dart`; `app/lib/settings/pages/follow_list_page.dart`; `app/test/profile/profile_page_test.dart`; `app/test/settings/follow_list_page_test.dart` | Add cursor-aware pagination/loading for mutual followers, followers, and following UI, or an equivalent mechanism that lets users view all available pages. Add/extend Flutter tests that return a non-null cursor from page 1 and assert the UI requests and appends page 2 while preserving order and total count. |
| IR-002 | Suggestion | Risk / Performance | No supporting index migration was added for the new ordered follow/root-post query shapes. This is not blocking because bounded keyset queries and limit caps are implemented and verified, and the coding plan made indexes conditional, but large-list performance should remain on the follow-up radar. | `01-requirements.md` NFR-002, RISK-001; `04-coding-plan.md` §5 index migration guidance; `05-implementation-plan.md` lines 203-204 | Revisit with manual large-list checks or query plans after the pagination UI gap is fixed; add indexes if dev data shows slow profile/list reads. |

## Requirement And Test Traceability
- Requirements implemented: AppView scalar fields for `mutualFollowerCount`, `postCount`, `postsLast7Days`, and `projectCount`; preservation of `followerCount`/`followingCount`; authenticated graph endpoints; profile UI hides follower/following stats; settings exposes follower/following entries; non-Craftsky age hiding; basic empty states.
- Tests implemented: AppView store/handler/route tests for summary counts, mutual/list endpoints, ordering, cursor contracts, and auth/device behavior; Flutter model/client/profile/settings tests for new fields and first-page rendering.
- Unplanned behavior: Settings list pages use local `MaterialPageRoute` instead of generated typed GoRouter routes. This appears acceptable behaviorally and is documented in `05-implementation-plan.md`.
- Remaining gaps: Flutter graph-list pagination UI and tests are missing for mutual followers, followers, and following (IR-001).

## Test Evidence
- Commands reviewed:
  - `go test ./internal/api ./internal/routes` from `appview/`
  - `flutter test test/profile/profile_page_test.dart test/profile/widgets/profile_stats_test.dart test/settings/settings_page_test.dart test/settings/follow_list_page_test.dart` from `app/`
  - Dart analyzer via MCP on `app/lib`, `app/test/profile`, and `app/test/settings`
- Passing evidence:
  - AppView focused tests passed.
  - Flutter focused tests passed.
  - Dart analyzer reported no errors.
- Failing or skipped tests:
  - No failing tests observed.
  - Plannotator review was attempted but the command was aborted before feedback was returned.
  - Missing test coverage for cursor-returning Flutter graph list pages/sheet is called out in IR-001.

## Risk Review
- Risk level: Medium.
- Risk notes: The main risk is a user-visible truncation of graph lists despite the API supporting pagination. Backend performance/index risk remains non-blocking but should be checked with larger data.
- Approval notes: Not ready for final handoff until graph-list pagination can be exercised from the Flutter UI and covered by tests.

## UI Polish Recommendation
- Recommendation: Optional
- Reason: After the functional pagination gap is fixed, a small polish pass could improve copy/plurals (for example `1 mutual follower`), row affordances, avatar placeholders, and loading/error visual states. These are polish-level unless they affect the pagination behavior in IR-001.
- Suggested polish notes: Review singular/plural labels, account-row spacing/avatar presentation, and bottom-sheet/list error states.

## Handoff Back To TDD Builder
- Required fixes:
  - Address IR-001 by adding cursor-aware pagination to `ProfileMutualFollowersSheet` and `FollowListPage` or a shared account-list provider/widget.
  - Add Flutter tests that fail when a returned cursor is ignored.
- Suggested next failing test:
  - Add a widget/provider test where `listFollowersMe` returns page 1 with `cursor: 'next'`; after scrolling/tapping load-more, assert `listFollowersMe(cursor: 'next')` is called and page 2 rows are appended. Mirror for mutual followers or cover via a shared list component.
- Verification to rerun:
  - `flutter test test/profile/profile_page_test.dart test/settings/follow_list_page_test.dart test/profile/data/profile_api_client_test.dart`
  - `go test ./internal/api ./internal/routes`
  - Dart analyzer on `app/lib`, `app/test/profile`, and `app/test/settings`
