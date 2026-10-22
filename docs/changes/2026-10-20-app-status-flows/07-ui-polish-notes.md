# UI Polish Notes: App Status Flows

## Summary

Polished the announcement dialog following the user's iPhone screenshot and spacing/action hierarchy feedback on 2026-10-22.

## Polish Items

| ID | Request / Source | Change Made | Files | Status |
|---|---|---|---|---|
| UIP-001 | More spacing around the image | Added 24px above and 8px below using existing spacing tokens. Together with the dialog's existing action gap, the image has 32px before the actions. | `app/lib/service_status/widgets/announcement_modal.dart` | Done |
| UIP-002 | Column of buttons; primary chunky main action and text dismissal | Stacked full-width actions, kept the main CTA as the primary ChunkyButton, and changed Dismiss to TextButton with a 12px inter-action gap. | Same widget | Done |

## Verification

- `flutter test --no-pub test/service_status/status_message_image_test.dart test/service_status/announcement_dismissal_test.dart test/service_status/service_status_accessibility_test.dart test/service_status/message_cta_test.dart`: 28 tests passed. Includes narrow/wide screens, light/dark themes, increased text scaling, image loading/failure, dismissal, CTA and accessibility coverage.
- `flutter analyze lib/service_status/widgets/announcement_modal.dart`: no issues.
- `git diff --check`: passed.
- Initial test invocation named a nonexistent CTA test file; reran successfully with the existing `message_cta_test.dart` path.
- No physical-device visual inspection or new screenshot was captured. Existing widget tests provide layout/interaction coverage.

## Scope Guardrails

- Requirement behavior changed: No.
- Business logic changed: No.
- APIs, data models, migrations, permissions, or dependencies changed: No.
- Standard Image.network loading, callbacks, disabled/loading states, autofocus and message eligibility remain intact. Existing unrelated worktree changes were preserved. No commit or deployment.

## Follow-ups

- Optional final implementation review or further user visual feedback at the polish stage exit.
