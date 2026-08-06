# UI Polish Notes: Pinned Profile Posts

## Summary

Pin and unpin feedback now uses the existing CraftSky themed message system instead of constructing a plain local `SnackBar`. Successful pin/unpin confirmations use the transient themed info variant; failed pin/unpin attempts use the themed sticky error variant. All localized copy and mutation behavior remain unchanged.

## Polish Items

| ID | Request / Source | Change Made | Files | Status |
|---|---|---|---|---|
| UIP-001 | User request: use the CraftSky themed toast for pin/unpin confirmation | Routed `Post pinned` and `Post unpinned` through `context.showInfo(...)`, and routed the existing retry messages through `context.showError(...)`. Updated widget tests to assert messenger severity and exact copy; aligned direct `PostCard` and thread test harnesses with the production `MessengerScope`. | `app/lib/feed/widgets/post_card.dart`; `app/test/feed/widgets/post_card_test.dart`; `app/test/feed/feed_page_test.dart`; `app/test/feed/pages/post_thread_page_test.dart`; `app/test/profile/widgets/profile_posts_tab_test.dart` | Done |

## Verification

- Commands run:
  - Focused red/green: `flutter test test/feed/feed_page_test.dart test/feed/pages/post_thread_page_test.dart test/profile/widgets/profile_posts_tab_test.dart --plain-name 'pin'`.
  - Affected widget regression: `flutter test test/feed/widgets/post_card_test.dart test/feed/feed_page_test.dart test/feed/pages/post_thread_page_test.dart test/profile/widgets/profile_posts_tab_test.dart`.
  - Static analysis: `dart analyze`.
- Passing evidence:
  - The focused test first failed because the recording CraftSky messenger received no calls, then passed with six selected tests after the presentation change.
  - The broader affected widget suite passed 98 tests.
  - `dart analyze` reported no issues.
- Skipped checks and reason:
  - The full Flutter suite was not repeated because this polish changes one feedback presentation seam and the shared card plus every pin-enabled surface were covered by the affected widget suite.
  - Physical device rendering was not repeated; MAN-001 and MAN-002 remain the existing external accessibility and visual checks.

## Scope Guardrails

- Requirement behavior changed: No
- Business logic changed: No
- APIs, data models, migrations, permissions, or dependencies changed: No
- Notes: Exact success and retry copy, authoritative reconciliation, pending-slot behavior, cache invalidation, and profile ordering are unchanged. Only the existing UI message delivery component and semantic severity changed.

## Follow-ups

- [ ] None.
