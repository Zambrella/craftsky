# UI Polish Notes: Local Post Drafts And Submit-Time Media Uploads

## Summary

Replaced the stock Material draft-close alert with CraftSky's existing paper-cutout dialog treatment. The close flow keeps the same copy, choices, save eligibility, and non-dismissible barrier while using the branded modal transition, red destructive text action, and chunky primary save action. Corrected the full-screen submission overlay so its status text inherits themed Material typography when mounted beside the composer `Scaffold`, and replaced its stock spinner with CraftSky's stitched progress indicator. Moved the standard composer action and every project composer primary action out of the app bar into a full-width floating CraftSky CTA, with bottom-safe scroll space so the button cannot obscure the last composer content at maximum scroll.

## Polish Items

| ID | Request / Source | Change Made | Files | Status |
|---|---|---|---|---|
| UIP-001 | User requested a CraftSky-themed modal when exiting a changed post composer | Rebuilt the shared draft-close dialog with `CraftskyDialog`, `showCraftskyModal`, and `ChunkyButton`; retained Save draft/Save changes, Discard/Discard changes, and Keep editing behavior for both standard and project composers | `app/lib/drafts/widgets/draft_close_dialog.dart`, `app/test/drafts/widgets/draft_close_dialog_test.dart` | Done |
| UIP-002 | Simulator screenshot showed fallback red/yellow debug text on the submission overlay; user requested the themed spinner | Wrapped the sibling overlay in a transparent `Material`, applied themed title typography with bounded horizontal padding, replaced `CircularProgressIndicator` with a 48 px `StitchProgressIndicator`, and changed the regression harness to match the production sibling-of-`Scaffold` structure | `app/lib/feed/widgets/submission_blocking_overlay.dart`, `app/test/feed/widgets/submission_overlay_test.dart` | Done |
| UIP-003 | User requested primary composer actions floating at the bottom, with enough trailing scroll space to keep content unobscured; follow-up requested the same treatment for project Next | Replaced the standard composer app-bar submit action with a full-width floating `ChunkyButton`; the project composer uses the same floating CTA for Next on pages one and two and Post/Schedule on page three; both scroll bodies reserve 96 px plus the device bottom inset | `app/lib/feed/widgets/post_composer_sheet.dart`, `app/lib/projects/widgets/project_composer_sheet.dart`, affected composer interaction tests | Done |

## Verification

- Commands run:
  - `flutter test test/drafts/widgets/draft_close_dialog_test.dart --reporter expanded`
  - `flutter test test/drafts/widgets/draft_close_dialog_test.dart test/feed/widgets/post_composer_sheet_discard_test.dart test/projects/widgets/project_composer_discard_test.dart --reporter compact`
  - `flutter test test/feed/widgets/submission_overlay_test.dart --reporter expanded`
  - `flutter test test/feed/widgets/submission_overlay_test.dart test/projects/widgets/project_composer_submit_test.dart test/scheduled_posts/scheduled_post_submission_test.dart --reporter compact`
  - `flutter test test/scheduled_posts/post_composer_scheduling_test.dart test/projects/widgets/project_composer_sheet_test.dart --reporter expanded`
  - Broad 13-file standard/project/scheduled composer regression group
  - `flutter test test/profile/widgets/profile_posts_tab_test.dart --reporter compact`
  - Focused 11-file project navigation, validation, submission, draft, and scheduling regression group
  - `flutter analyze`
  - `dart format lib/drafts/widgets/draft_close_dialog.dart test/drafts/widgets/draft_close_dialog_test.dart`
  - `dart format lib/feed/widgets/submission_blocking_overlay.dart test/feed/widgets/submission_overlay_test.dart`
- Passing evidence:
  - The focused dialog test failed against the previous stock `AlertDialog`, then passed after the themed implementation.
  - The combined standard composer, project composer, and dialog group passed all 16 tests.
  - The overlay regression failed against the previous stock spinner. After correction, both publishing and scheduling cases prove the themed spinner, a Material ancestor, themed default text colour, exact copy, modal barrier, and live-region semantics.
  - The combined overlay, project submission, and scheduled submission group passed all 8 tests.
  - The floating-CTA regressions failed against the previous app-bar `TextButton`, then passed with a full-width `ChunkyButton` and a trailing spacer taller than the CTA in both composer types.
  - The broad standard, project, scheduled-post, feed, and comment composer group passed 81 tests; the affected profile reply-composer suite passed another 7 tests.
  - The project Next refinement regression failed against the remaining app-bar action, then the focused project/scheduled group passed all 56 tests with the floating primary action on every project page.
  - Static analysis reported no issues and formatting was clean.
- Skipped checks and reason:
  - The full Flutter suite was not repeated because the change is isolated to a shared presentation widget and its focused standard/project composer suites pass.
  - The corrected overlay has not yet been re-inspected on a device in light and dark themes.

## Scope Guardrails

- Requirement behavior changed: No
- Business logic changed: No
- APIs, data models, migrations, permissions, or dependencies changed: No
- Notes: Existing dialog and overlay copy, action outcomes, disabled-save behavior, submission lifecycle, and barrier behavior are unchanged.

## Follow-ups

- [ ] Optionally inspect the modal on a device in both light and dark themes.
- [ ] Recheck both publishing and scheduling overlays on a device in light and dark themes.
- [ ] Inspect the floating CTA on a device in both themes, with the keyboard open and at maximum scroll on each composer page.
