# TDD Implementation Plan: Responsive App Navigation Drawer And Extended Rail Corrections

## Inputs

- Requirements: `01-requirements.md`
- Implementation review: `06-implementation-review.md`
- Tests: intermediate `02-acceptance-tests.md` was explicitly skipped by the user; correction tests are derived from Must acceptance criteria and review findings.
- Coding plan: intermediate `04-coding-plan.md` was explicitly skipped by the user.

## Implementation Rules

- Do not implement behavior without a linked requirement ID.
- Write or update one focused failing test before each implementation change.
- Run the smallest relevant test first.
- Refactor only after tests pass.
- Keep traceability updated.
- Preserve unrelated worktree changes.

## Test Order

| Step | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| 1 | CORR-001 | FR-008 | AC-006, AC-017 | Drawer exposes capped notification text but not the full accessible count. |
| 2 | CORR-002 | FR-008 | AC-006, AC-017 | Drawer Profile avatar has no account-switcher action. |
| 3 | CORR-003 | FR-005, FR-011 | AC-005, AC-011 | Rapid repeated selection can dispatch more than once. |
| 4 | CORR-004 | NFR-001 | AC-014 | Hamburger does not expose menu expanded/collapsed state or verified focus return. |
| 5 | CORR-005 | FR-003-FR-005, FR-011, NFR-004 | AC-003-AC-005, AC-011, AC-013, AC-017 | Core/secondary/large actions and standard dismissal paths are incompletely covered. |
| 6 | CORR-006 | FR-006, FR-012, NFR-002-NFR-004 | AC-007, AC-012, AC-015-AC-017 | Resize, RTL, increased text scale, short viewport, and legal failure behavior are incompletely covered. |
| 7 | CORR-007 | FR-013, RULE-005 | AC-018 | Former paths and Back from Scheduled posts/Drafts lack explicit production-router coverage. |

## Implementation Steps

### Step 1: CORR-001

- Write failing test: Open the compact drawer with notification newness above the visible cap and assert the localized full count in semantics.
- Run command: `flutter test test/notifications/app_shell_notification_badge_test.dart --plain-name '<drawer semantics test>'`
- Confirmed failure: The drawer subtree contained no semantic label matching `Notifications, 137 new activities`; the existing bottom-bar semantics could not satisfy the drawer-scoped finder.
- Implement: Excluded the decorative capped badge from drawer semantics and gave each primary drawer title its localized destination semantic label, including the uncapped notification count.
- Run command: Focused CORR-001 test passes.
- Refactor: Reused `_destinationSemanticsLabel` rather than duplicating count copy.
- Notes: Linked to IR-001. Completed.

### Step 2: CORR-002

- Write failing test: Open the compact drawer with multiple accounts and assert Profile exposes and opens the existing account switcher through long press and keyboard activation.
- Run command: `flutter test test/router/app_shell_account_switcher_test.dart --plain-name '<drawer account switcher test>'`
- Confirmed failure: The drawer `AccountAvatar` semantics had no long-press action and activation produced no switcher.
- Implement: Reused `_DestinationIcon` for the drawer Profile destination with a drawer-specific focus node and the existing compact account-switcher callback; opening the switcher first closes the drawer.
- Run command: Focused CORR-002 test passes for long press and `Alt+ArrowDown` keyboard activation.
- Refactor: Kept drawer and bottom-bar focus nodes separate so both mounted presentations never attach one `FocusNode` twice.
- Notes: Linked to IR-001. Completed.

### Step 3: CORR-003

- Write failing test: Activate one drawer destination twice before its close animation settles and assert one navigation action and no additional pop.
- Run command: `flutter test test/router/app_shell_navigation_menu_test.dart --plain-name '<rapid selection test>'`
- Confirmed failure: Two synchronous Saved posts activations produced two router redirects.
- Implement: Converted the drawer to stateful ownership and added a one-action claim for destination, account-switcher, and legal-link activations during one open lifecycle.
- Run command: Focused secondary and core rapid-selection tests pass with exactly one redirect and one close.
- Refactor: Centralized the guard in `_claimAction` and reused it across every drawer action that dismisses the drawer.
- Notes: Linked to IR-002. Completed.

### Step 4: CORR-004

- Write failing test: Inspect hamburger/menu semantics closed and open, activate by keyboard, dismiss, and verify focus return.
- Run command: `flutter test test/router/app_shell_navigation_menu_test.dart --plain-name 'CORR-004 hamburger exposes state and returns keyboard focus'`
- Confirmed failure: The hamburger semantic node reported no expanded/collapsed state; after adding state, focus still did not return to the button when Escape closed the drawer.
- Implement: Added compact shell drawer-open state, an explicit localized hamburger semantic node with expanded state and tap action, and a shell-owned hamburger focus node restored after drawer dismissal.
- Run command: Focused CORR-004 test passes for keyboard open, expanded/collapsed semantics, Escape dismissal, and focus return.
- Refactor: Kept state and focus ownership in the shell while exposing only the state and focus node through `AppShellDrawerScope` to each branch app bar.
- Notes: Linked to IR-003. Completed.

### Step 5: CORR-005

- Write failing tests: Cover core selection/reselection, all secondary route actions, large-rail actions, compact five-item bottom navigation, active selection, and scrim/Back/Escape dismissal.
- Run command: `flutter test test/router/app_shell_navigation_menu_test.dart`
- Confirmed failure: The large layout still rendered the branch app bar's disabled hamburger because `AppShellDrawerButton` remained visible when no compact drawer scope existed. Initial-route redirect accounting in the new Feed reselection tests was separated from the activation under test.
- Implement: Hide the shared hamburger entirely outside the compact drawer scope. No product changes were needed for the destination mapping, one-action selection, selected state, five-item bottom bar, inert Feedback, scrim dismissal, or system Back dismissal.
- Run command: The complete 29-test navigation-menu suite passes, including every core and secondary action on compact and large layouts.
- Refactor: Reused table-driven route cases for the two form-factor action matrices.
- Notes: Linked to IR-004. Completed.

### Step 6: CORR-006

- Write failing tests: Cover breakpoint resize with an open drawer, RTL start edge, increased text scaling and short height, and false/throwing legal launch feedback.
- Run command: `flutter test test/router/app_shell_navigation_menu_test.dart`
- Confirmed failure: No product behavior failed. The first stale-scrim assertion counted unrelated framework barriers and was refined to prove the resized rail is interactive by navigating through it.
- Implement: No product change was needed for breakpoint cleanup, RTL start-edge behavior, large-text/short-height reachability, or safe localized handling of both false and throwing external launchers.
- Run command: The complete 35-test navigation-menu suite passes.
- Refactor: Extended the shared shell harness with optional directionality, text scaling, and a recording messenger so the edge cases exercise the same shell.
- Notes: Linked to IR-004 and RISK-003. Completed.

### Step 7: CORR-007

- Write failing tests: Verify former `/profile/settings/...` paths use existing unknown-route behavior and Back from Saved posts, Scheduled posts, and Drafts reveals Profile.
- Run command: `flutter test test/router/saved_posts_route_test.dart`
- Confirmed failure: A secondary menu selection replaced the route configuration, so system Back from Scheduled posts remained on the page instead of revealing Profile. Scheduled posts and Drafts also lacked a visible Back affordance while their account state was loading.
- Implement: Secondary menu actions now reset the Profile branch and push the selected full-screen route. Saved posts, Scheduled posts, and Drafts expose an explicit Back button that returns to Profile, including their loading presentations.
- Run command: The focused menu Back test, the complete 36-test navigation-menu suite, and the 6-test production saved/personal-route suite pass.
- Refactor: Centralized the secondary location selection before the shared Profile-branch-and-push behavior; reused one production-router harness for positive Back and negative former-path checks.
- Notes: The production router rejects all former Settings-owned paths, including the old saved-folder descendant. Linked to IR-004 and RISK-008. Completed.

## Completion Checklist

- [x] All Must review findings corrected.
- [x] All planned correction tests passing.
- [x] Relevant regression tests passing.
- [x] No unlinked behavior implemented.
- [x] Full `dart analyze` passes.
- [x] `git diff --check` passes.
- [x] Implementation notes updated.
- [ ] Implementation review rerun or explicitly skipped.

## Final Verification

- `flutter test test/router/app_shell_navigation_menu_test.dart test/router/top_level_drawer_button_test.dart test/router/app_shell_account_switcher_test.dart test/notifications/app_shell_notification_badge_test.dart test/router/saved_posts_route_test.dart test/router/settings_routes_test.dart test/settings/settings_page_test.dart test/scheduled_posts/scheduled_posts_page_test.dart test/drafts/drafts_page_test.dart` — 72 tests passed.
- `dart analyze` — no issues found.
- `git diff --check` — passed.
- Manual device gates remain for native edge gestures and physical VoiceOver/TalkBack behavior, as specified by AC-017.
