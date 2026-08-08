# Implementation Review: Responsive App Navigation Drawer And Extended Rail

## Verdict

Status: Changes required
Reviewer: Codex
Date: 2026-08-06
Risk level: Medium

## Summary

The implementation establishes the intended responsive shell structure, relocates Saved posts, Scheduled posts, and Drafts beneath Profile, removes those entries from Settings, adds localized legal and Feedback actions, and wires the hamburger into all five real branch-root app bars. The focused test suite and full static analysis pass.

The change is not ready for handoff because the compact drawer does not preserve two explicitly required navigation affordances, repeated selection is not guarded, the hamburger does not expose the required expanded/collapsed semantics, and the Must-level automated verification matrix remains materially incomplete. These are requirements and accessibility gaps rather than optional visual polish.

The user explicitly approved skipping `02-acceptance-tests.md`, `03-document-review.md`, `04-coding-plan.md`, and `05-implementation-plan.md` and proceeding directly from `01-requirements.md` with TDD. Their absence is therefore an accepted workflow deviation, not an implementation finding.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-001 | Important | Behavior / Accessibility | The compact drawer renders the Notifications badge through `_DestinationIcon` without the full-count semantic label used by the bottom bar and rail, so a count such as 137 is exposed only as the capped `99+`. Its Profile row renders a plain `AccountAvatar` and supplies no long-press or keyboard account-switcher action. The requirements explicitly preserve the full accessible count and Profile account-switcher affordance when the drawer renders. | FR-008; AC-006; AC-017; `app/lib/router/app_shell.dart` lines 344-360 and 667-676; `app/test/notifications/app_shell_notification_badge_test.dart` lines 13-43; `app/test/router/app_shell_account_switcher_test.dart` lines 469-540 | Give the drawer Notifications row the localized full-count semantics. Add the established Profile account-switcher interaction to the drawer without turning it into a prominent profile section. Add drawer-specific regression tests for the full count, active-account avatar isolation, long press, and keyboard activation. |
| IR-002 | Important | Behavior | `_selectAndClose` has no selection-in-progress guard. Every activation immediately calls `Navigator.pop` and then dispatches navigation, while the drawer remains mounted during its closing animation. Rapid repeated input can therefore dispatch more than one branch action and can invoke `Navigator.pop` again, contrary to the explicit at-most-once requirement. No test exercises the race. | FR-005; FR-011; AC-005; AC-011; EC-002; `app/lib/router/app_shell.dart` lines 421-424 | Add a failing rapid-activation test, then make destination selection idempotent for one drawer-open lifecycle so only the first activation closes and navigates. Verify both a core destination and a root-navigator secondary destination. |
| IR-003 | Important | Accessibility | `AppShellDrawerButton` is an `IconButton` with a localized “Open navigation menu” tooltip, but it exposes no expanded/collapsed state. This does not satisfy the explicit hamburger state requirement, and the current tests only prove that tapping invokes the callback. | NFR-001; AC-014; `app/lib/router/app_shell_drawer.dart` lines 23-33; `app/test/router/top_level_drawer_button_test.dart` lines 42-79 | Model and expose the drawer state through semantics, with a localized accessible label/state as appropriate, and add a semantics test for closed and open menu states plus keyboard activation and focus return. |
| IR-004 | Important | Tests / Traceability | NFR-004 and AC-017 require automated coverage of every destination action, both form-factor selections, dismissal modes, active state, resize behavior, semantics, and overflow. Current feature tests check one secondary action (Saved posts), successful legal launches, an inert compact Feedback button, menu presence, a default-scale 500-pixel-high rail, and five hamburger callbacks. They do not cover core destination selection/reselection, Scheduled posts/Drafts/Settings actions, large-rail actions, scrim/Back/Escape dismissal, rapid activation, resize across 900 pixels, RTL, increased text scaling, drawer/rail selected semantics, failed legal launch feedback, former-path rejection, or Back from Scheduled posts and Drafts to Profile. | NFR-004; AC-002; AC-003-AC-007; AC-010-AC-018; EC-001-EC-010; `app/test/router/app_shell_navigation_menu_test.dart` lines 13-134; `app/test/router/top_level_drawer_button_test.dart` lines 20-80; `app/test/router/saved_posts_route_test.dart` | Add the missing public-behavior tests. At minimum, cover every core and secondary action on its applicable presentation, current-branch reselection, all standard drawer dismissal paths, breakpoint resize with an open drawer, RTL and large-text short-height layouts, selected/accessibility semantics, failed legal launches, negative former routes, and Profile Back behavior for all three relocated personal-content pages. Record physical edge-swipe and VoiceOver/TalkBack checks as manual gates. |

## Requirement And Test Traceability

- Requirements implemented:
  - BR-001 through BR-004 are structurally represented.
  - FR-001 through FR-007 and FR-009 through FR-014 are substantially implemented, subject to IR-002 and the missing verification in IR-004.
  - FR-008 is only partially implemented because its drawer-specific accessibility and account-switcher behavior is missing.
  - RULE-001 through RULE-005 are reflected in the implementation and generated route hierarchy.
  - Saved posts, Scheduled posts, and Drafts resolve as Profile siblings, and Settings no longer renders them.
- Tests implemented:
  - Compact edge-drag opening and hamburger opening.
  - Presence/order by source list for compact and large menu labels.
  - Compact Saved posts navigation and dismissal.
  - Exact successful Terms and Privacy URLs.
  - Compact Feedback route/menu-state no-op.
  - Five real top-level app-bar hamburger integrations.
  - Saved route/folder hierarchy and Settings-row removal.
  - Existing bottom-bar/rail notification semantics and account-switcher regressions.
- Unplanned behavior:
  - None identified.
- Remaining gaps:
  - IR-001 through IR-004.
  - Native device edge-swipe behavior and physical VoiceOver/TalkBack behavior remain manual checks.

## Test Evidence

- Commands reviewed:
  - `flutter test test/router/app_shell_navigation_menu_test.dart test/router/top_level_drawer_button_test.dart test/router/saved_posts_route_test.dart test/router/settings_routes_test.dart test/settings/settings_page_test.dart test/notifications/app_shell_notification_badge_test.dart test/router/app_shell_account_switcher_test.dart`
  - `dart analyze`
  - `git diff --check`
- Passing evidence:
  - 32 focused tests passed.
  - Full app static analysis completed with no issues.
  - Diff whitespace validation passed.
- Failing or skipped tests:
  - No implemented automated test failed.
  - The required automated cases listed in IR-004 were not implemented.
  - Native edge-swipe, physical screen-reader, and manual visual checks were not run.

## Risk Review

- Risk level: Medium
- Risk notes:
  - Route relocation and Settings ownership are low risk and supported by generated-route and widget evidence.
  - App-wide navigation and accessibility are high-frequency surfaces; the missing drawer semantics and repeated-input guard can affect every compact branch.
  - No persistence, API, permission, billing, security, or data-migration changes were introduced.
- Approval notes:
  - Approval is blocked on IR-001 through IR-004.
  - The passing focused suite and static analysis provide a sound base for a narrow correction pass.

## UI Polish Recommendation

- Recommendation: Optional
- Reason: The current grouping, simple profile treatment, footer separation, and localized labels align with the requested direction. A device-level visual pass may still improve spacing and footer proportions, but polish should follow the required behavior and accessibility corrections.
- Suggested polish notes:
  - Review drawer and extended-rail spacing at narrow widths, short landscape heights, and large text scales.
  - Confirm the footer remains visually separated while all actions remain reachable.
  - Check the custom Profile header hamburger alignment through its collapsed and expanded states.

## Handoff Back To TDD Builder

- Required fixes:
  - Restore drawer-specific full notification semantics and the Profile account-switcher affordance.
  - Guard drawer destination activation against rapid repeated input.
  - Expose and test hamburger expanded/collapsed semantics.
  - Complete the Must-level navigation, responsive, accessibility, negative-route, and failure-path test matrix.
- Suggested next failing test:
  - Open the compact drawer with a notification count of 137 and multiple accounts, then assert that Notifications announces the localized full count and that the drawer Profile avatar exposes the existing long-press/keyboard account-switcher action.
- Verification to rerun:
  - The focused navigation, route, Settings, badge, and account-switcher suite above.
  - New rapid-input, dismissal, resize, RTL, large-text, failure, negative-route, and semantics tests.
  - `dart analyze`.
  - `git diff --check`.
  - Manual native edge-swipe and VoiceOver/TalkBack checks.
