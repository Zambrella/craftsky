# Implementation Review: Notifications MVP

## Verdict
Status: Changes required  
Reviewer: gpt-5.5 implementation reviewer  
Date: 2026-05-29  
Risk level: Medium

## Summary

The AppView derived store, handler, and protected route are broadly aligned with the approved read-only Notifications MVP direction, and focused AppView plus Flutter notification test commands pass. However, the implementation is not ready to merge because a reply-notification API/Flutter contract bug can break decoding, and several Must UI acceptance behaviors/tests from `02-acceptance-tests.md` and `04-coding-plan.md` are missing.

The current working tree also contains an unrelated `docs/roadmap.md` change. It was not reviewed as part of the notifications implementation and should remain unstaged unless handled separately.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-001 | Critical | API / Flutter Contract | Reply notification focus identity is encoded by Go without JSON tags, so the API response will use `URI`, `CID`, and `Rkey` while the Flutter decoder expects `uri`, `cid`, and `rkey`. A page containing a reply notification can fail to decode or lose focus-navigation data. Handler tests only cover a like notification, so this contract mismatch is untested. | `appview/internal/api/notification_store.go:23-27`; `appview/internal/api/notifications.go:37-38`; `app/lib/notifications/models/craftsky_notification.dart:47-51`, `151-156`; `01-requirements.md` `FR-011`, `FR-012`; `02-acceptance-tests.md` `UT-006`, `UT-009`, `AT-003` | Add camelCase JSON tags or a separate response DTO for reply focus refs, add handler coverage for a reply item JSON shape, and rerun Flutter model/page decode tests with AppView-shaped reply JSON. |
| IR-002 | Important | UI Behavior / Tests | The Notifications page does not implement the required load-more progress and load-more retry states. It renders only top-level loading/error/data branches plus a static `Load more` button; there is no loaded-list UI that shows bottom progress or bottom retry while preserving rows after load-more failure. | `app/lib/notifications/pages/notifications_page.dart:14-45`; `app/lib/feed/pages/feed_page.dart:46-152` as the planned pattern; `01-requirements.md` `FR-013`, `FR-014`; `02-acceptance-tests.md` `AC-016`, `AT-005`, `UT-014`; `04-coding-plan.md` §7-§8 | Mirror the feed loaded-state pattern for notifications: show existing rows, bottom progress while loading more, and a bottom retry affordance on load-more failure; add widget tests proving these states. |
| IR-003 | Important | Tests / Navigation | Required widget/router coverage for mixed rows and row navigation is incomplete. `notifications_page_test.dart` only verifies the title and one follow row; it does not cover like/repost/reply rows, actor fallback display, follow profile navigation, subject-thread navigation, reply focus navigation, or reply-without-focus fallback. | `app/test/notifications/notifications_page_test.dart:14-44`; `app/lib/notifications/widgets/notification_row.dart:15-60`; `01-requirements.md` `FR-014`, `FR-015`; `02-acceptance-tests.md` `AT-002`, `AT-003`, `UT-014`, `UT-015`, `UT-016` | Add widget/router tests for all notification row types and navigation outcomes, including reply focus and no-focus fallback; fix behavior if those tests expose route issues. |
| IR-004 | Suggestion | UI / Code Quality | Notification page and row copy is hard-coded instead of using the existing localization surface planned for this slice. This is not the primary blocker, but it diverges from the coding plan and existing Feed/Profile patterns. | `app/lib/notifications/pages/notifications_page.dart:13`, `20-23`, `30`, `39`; `app/lib/notifications/widgets/notification_row.dart:16-27`; `04-coding-plan.md` §7 Localization; `app/lib/l10n/app_en.arb` | Move notification title, empty/error/retry/load-more, and row copy into app localization if touching the UI for IR-002/IR-003. |

## Requirement And Test Traceability

- Requirements implemented:
  - AppView read-only route and derived store work cover `BR-002`, `FR-001` through `FR-008`, `FR-010`, `RULE-001`, `RULE-002`, and `RULE-003` in the core backend path.
  - Flutter model/API/provider foundations cover part of `FR-012` and `FR-013`.
- Tests implemented:
  - AppView store tests cover follows, likes, reposts, replies, self-exclusion, active-only likes/reposts, ordering, pagination, terminal cursor, and invalid cursor.
  - AppView handler/route tests cover default/capped/invalid limits, unknown/request-supplied DID query handling, authenticated viewer scoping, a like-shaped JSON page, invalid cursor envelope, store failure envelope, and protected route registration.
  - Flutter tests cover mixed model decoding using expected camelCase JSON, API cursor forwarding, provider retry/pagination/load-more failure state preservation, and a minimal page title/follow-row render.
- Unplanned behavior:
  - None found in source code, but `docs/roadmap.md` remains modified in the working tree and is unrelated to this stage.
- Remaining gaps:
  - Reply JSON shape is not aligned between AppView and Flutter (`IR-001`).
  - Page-level load-more and retry states are missing (`IR-002`).
  - Required widget/router coverage for mixed rows, fallback identity, and navigation is missing (`IR-003`).
  - `IT-013` unavailable-subject coverage was cancelled in `05-implementation-plan.md` with a schema-constraint rationale; accepted as a documented gap for this review, but it should remain visible as MVP risk.

## Test Evidence

- Commands reviewed:
  - Implementation plan reports focused AppView, focused Flutter notification tests, and `flutter analyze`.
  - Reviewer reran: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes` from `appview/`.
  - Reviewer reran: `flutter test test/notifications/models/notification_test.dart test/notifications/data/notification_api_client_test.dart test/notifications/providers/notifications_provider_test.dart test/notifications/notifications_page_test.dart` from `app/`.
  - Reviewer reran: `flutter analyze` from `app/`.
- Passing evidence:
  - Focused AppView command passed for `social.craftsky/appview/internal/api` and `social.craftsky/appview/internal/routes`.
  - Focused Flutter notification command passed all 7 current tests.
- Failing or skipped tests:
  - `flutter analyze` still fails on an existing `dummy_profile_repository.dart` abstract-member error; notification code also introduces analyzer info-level findings.
  - The passing focused Flutter suite is insufficient for the Must UI/navigation acceptance criteria listed in `IR-002` and `IR-003`.
  - `IT-013` unavailable-subject behavior was skipped/cancelled with rationale in `05-implementation-plan.md`.

## Risk Review

- Risk level: Medium
- Risk notes:
  - The backend query path is reasonably covered, but the reply JSON mismatch is a real cross-layer contract risk.
  - The user-facing Notifications tab does not yet meet all planned load-more/error/navigation behaviors.
  - The Flutter analyzer remains red because of an existing profile repository error, so analyzer evidence is weaker than desired.
- Approval notes:
  - Changes required before merge/handoff completion. The required fixes are focused and can be handled in another TDD loop.

## UI Polish Recommendation

- Recommendation: Not needed
- Reason: The visible gaps are behavioral/test-completeness issues, not polish-only copy/spacing/color refinements.
- Suggested polish notes: After required fixes land, a separate polish pass could still evaluate copy and spacing, but it should not precede the required TDD fixes.

## Handoff Back To TDD Builder

- Required fixes:
  - Fix reply focus JSON casing and add AppView/Flutter contract tests (`IR-001`).
  - Implement and test loaded-list load-more progress and retry states (`IR-002`).
  - Add widget/router tests for all row types, fallback identity, and navigation (`IR-003`).
- Suggested next failing test:
  - Start with an AppView handler test that returns a `reply` notification and asserts `reply.uri`, `reply.cid`, and `reply.rkey` are camelCase in JSON; then add/adjust Flutter model/page tests against that response.
- Verification to rerun:
  - `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes`
  - `flutter test test/notifications/models/notification_test.dart test/notifications/data/notification_api_client_test.dart test/notifications/providers/notifications_provider_test.dart test/notifications/notifications_page_test.dart`
  - `flutter analyze` once the known profile repository analyzer blocker is resolved or explicitly waived.
