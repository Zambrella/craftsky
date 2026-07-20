# UI Polish Notes: Mutes and Blocks

## Summary

Updated the visitor-profile action hierarchy so Mute/Unmute is immediately available, Share lives in the existing responsive CraftSky context menu, and the same hierarchy remains available when the profile header collapses.

## Polish Items

| ID | Request / Source | Change Made | Files | Status |
|---|---|---|---|---|
| UIP-001 | User request: replace the visible Share action with Mute/Unmute | Added a state-aware, disabled-while-busy Mute/Unmute icon action and removed Mute/Unmute from the profile More menu. | `app/lib/profile/widgets/profile_actions.dart`, `app/lib/theme/chunky_icon_button.dart` | Done |
| UIP-002 | User request: move Share to the context menu | Added Share as the first profile More action, followed by a separate safety-action group containing Block/Unblock and Report. | `app/lib/profile/widgets/profile_actions.dart` | Done |
| UIP-003 | User request: use the existing CraftSky responsive modal | Replaced the raw `PopupMenuButton` with `CraftskyContextMenuButton`; compact screens use its bottom sheet and larger screens use its anchored popup. Added an enabled state so an in-flight relationship mutation disables the complete menu consistently. | `app/lib/profile/widgets/profile_actions.dart`, `app/lib/theme/craftsky_context_menu.dart` | Done |
| UIP-004 | Profile-page consistency | Replaced the collapsed visitor app-bar Share shortcut with the current Mute/Unmute action, so Share is not exposed as a second standalone action. | `app/lib/profile/widgets/profile_sliver_app_bar.dart` | Done |
| UIP-005 | TDD and workflow alignment | Added compact/wide adaptive-menu coverage, direct mute mutation coverage, disabled-menu coverage, and aligned FR-022, AC-033, AT-001, and the coding plan with the user-approved hierarchy. | `app/test/profile/widgets/profile_actions_test.dart`, `app/test/profile/profile_page_test.dart`, `app/test/theme/craftsky_context_menu_test.dart`, `01-requirements.md`, `02-acceptance-tests.md`, `04-coding-plan.md` | Done |

## Verification

- Commands run:
  - `flutter test test/profile/widgets/profile_actions_test.dart test/profile/profile_page_test.dart test/theme/craftsky_context_menu_test.dart`
  - `flutter analyze`
  - `git diff --check`
- Passing evidence:
  - All 34 focused widget tests passed.
  - Flutter analysis completed with no issues.
  - Git whitespace validation passed.
- Skipped checks and reason:
  - The complete repository test suite was not rerun because the changes are confined to Flutter profile actions and two backward-compatible shared-widget parameters; the affected profile and shared-menu suites were run directly.

## Scope Guardrails

- Requirement behavior changed: Yes
- Business logic changed: No
- APIs, data models, migrations, permissions, or dependencies changed: No
- Notes: The user explicitly revised the profile action hierarchy after the original FR-022/AC-033 decision. The workflow contract was updated to match. Existing mute, unmute, block, report, confirmation, optimistic-state, and blocked-profile rules are unchanged.

## Follow-ups

- [ ] Run the final implementation review against the updated action hierarchy and current worktree.
