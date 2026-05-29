# Implementation Review: Timeline Feed Flutter

## Verdict
Status: Changes required
Reviewer: gpt-5.5 implementation reviewer
Date: 2026-05-29
Risk level: Medium

## Summary

The implementation is broadly aligned with the approved Flutter timeline scope: it adds AppView timeline API/repository support, a focused Riverpod timeline provider, a paginated `FeedPage`, localized empty/error copy, top-level compose, optimistic timeline insertion/dedupe, and timeline cache updates for create/delete/like/repost/reply paths. I did not find AppView, lexicon, migration, dependency, PDS-read-through, generic feed-framework, recommendation, ranking, or durable-cache scope creep.

Focused timeline tests, planned regression tests, and analyzer checks passed during review. However, one Must acceptance-test path is not actually verified: the timeline reply/comment scenario does not assert that the live timeline root row's `replyCount` and `viewerHasReplied` state are updated after a reply is created. The source appears to implement this in `FeedPage`, but the TDD traceability gap should be fixed before handoff.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-001 | Important | Tests / Traceability | `AT-009` / `UT-009` are marked completed, but the implemented tests do not assert the required timeline-row reply-state update. `FeedPage reply opens focused thread and updates root row` verifies composer submission and focused thread navigation, while `CreatePost` verifies reply creates are not prepended as top-level timeline rows. No test observes that the live timeline root post has `replyCount + 1` and `viewerHasReplied == true` after replying from the timeline. | `02-acceptance-tests.md` AT-009, UT-009; `01-requirements.md` FR-012, AC-012, RULE-003; `04-coding-plan.md` timeline row behavior; `05-implementation-plan.md` lines 43-50 and 173-181; `app/test/feed/feed_page_test.dart` reply test; `app/test/feed/providers/timeline_provider_test.dart` | Add/adjust a failing test that observes `timelineProvider` or the visible row after timeline reply creation and verifies `replyCount` increments, `viewerHasReplied` becomes true, and the reply URI is not present as a top-level timeline row. Rerun the focused timeline/widget tests and update `05-implementation-plan.md` if needed. |
| IR-002 | Suggestion | Tests | The delete widget test proves an owner row can delete and is removed, but it does not explicitly open/check the non-owned row menu to prove `Delete post` is absent there. The implementation gates `onDelete` by signed-in DID, so this is a non-blocking strengthening for a Should requirement. | `02-acceptance-tests.md` AT-010; `01-requirements.md` FR-013 / AC-015; `app/test/feed/feed_page_test.dart` delete test; `app/lib/feed/pages/feed_page.dart` | Consider extending the delete acceptance test to assert that the other-author row does not expose the delete action. |
| IR-003 | Suggestion | Test Evidence | The implementation log records full `flutter test` failing on an unrelated/stale profile-page expectation. Focused timeline and planned regression suites are green, and this implementation did not modify `app/lib/profile/**` or `app/test/profile/profile_page_test.dart`; still, the broader suite should be reconciled before final merge confidence. | `05-implementation-plan.md` final verification; `git diff --name-only HEAD~1..HEAD` | Track or fix the unrelated `ProfilePage visitor profile renders Follow + Share actions` failure separately; rerun full suite when that baseline is clean. |

## Requirement And Test Traceability

- Requirements implemented:
  - `FR-001` / `FR-002` / `NFR-001` / `NFR-002` / `RULE-001`: `PostApiClient.listTimeline`, `PostRepository.listTimeline`, and `ApiPostRepository.listTimeline` call `/v1/feed/timeline` through the existing post stack with no handle/DID route parameter.
  - `FR-003` through `FR-007`, `RULE-002`: `timelineProvider` accumulates cursor-paginated `PostPage` data, uses `timelinePageLimit = 20`, treats cursors opaquely, preserves previous data on load-more errors, and guards terminal/concurrent load-more calls.
  - `FR-008` through `FR-014`, `NFR-003` / `NFR-004`: `FeedPage` replaces the placeholder with localized loading/error/empty/loaded states, lazy sliver rendering, `PostCard` rows, thread navigation, interactions, delete gating, and a top-level composer entry.
  - `FR-015`, `RULE-003`, `RULE-004`: top-level create prepends into live timeline state; reply creates do not become top-level timeline rows; timeline merge/prepend dedupes by `Post.uri`.
- Tests implemented:
  - API/client and repository tests for `IT-001` through `IT-004`.
  - Provider tests for first load, pagination, load-more error/retry, guards, prepend/dedupe, delete removal, and timeline interaction patch/rollback.
  - Widget tests for loading, loaded, empty, initial error retry, pagination, load-more retry, thread navigation, like/repost, reply route, delete, and compose/prepend.
  - Planned regression suites for post cards, composer, interaction providers, profile tabs, auth/device/Dio providers, router, and app initialization.
- Unplanned behavior:
  - None identified in the reviewed diff.
- Remaining gaps:
  - Blocking: add actual verification for `AT-009` / `UT-009` reply-state update in the live timeline row.
  - Non-blocking: strengthen non-owned delete-menu assertion and reconcile the unrelated full-suite profile test baseline.

## Test Evidence

- Commands reviewed:
  - `git status --short`
  - `git show --stat --oneline --decorate --summary HEAD`
  - `git diff --name-only HEAD~1..HEAD`
  - Analyzer check on changed Dart source/test files via Dart MCP `analyze_files`
  - `flutter test test/feed/feed_page_test.dart test/feed/providers/timeline_provider_test.dart test/feed/providers/create_post_provider_test.dart test/feed/providers/toggle_post_interactions_provider_test.dart test/feed/data/post_api_client_test.dart test/feed/data/post_repository_test.dart`
  - `flutter test test/feed/widgets/post_card_test.dart test/feed/widgets/post_image_carousel_test.dart test/feed/widgets/post_image_gallery_test.dart test/feed/widgets/post_composer_sheet_discard_test.dart test/feed/providers/create_post_provider_test.dart test/feed/providers/toggle_post_interactions_provider_test.dart test/profile/widgets/profile_posts_tab_test.dart test/profile/widgets/profile_comments_tab_test.dart test/feed/providers/user_posts_provider_test.dart test/shared/api/providers/session_auth_interceptor_test.dart test/shared/device/device_id_provider_test.dart test/shared/api/providers/dio_provider_test.dart test/router/router_redirect_test.dart test/app_test.dart`
- Passing evidence:
  - Analyzer check: no errors on reviewed changed files.
  - Focused timeline/API/provider/widget suite: `+61 All tests passed`.
  - Planned regression suite: `+106 All tests passed`.
- Failing or skipped tests:
  - Reviewer did not rerun full `flutter test`; `05-implementation-plan.md` records one unrelated full-suite failure in `test/profile/profile_page_test.dart` for an existing profile expectation.
  - `AT-009` / `UT-009` reply-state update coverage is missing as described in `IR-001`.

## Risk Review

- Risk level: Medium.
- Risk notes:
  - User-facing home feed, cursor pagination, optimistic cache updates, and shared interaction providers make this a medium-risk Flutter change.
  - Source-level behavior is consistent with the approved plan, and focused/regression tests are green.
  - The remaining blocking risk is TDD traceability: one Must reply-state assertion is absent from the automated tests.
- Approval notes:
  - Not approved until `IR-001` is addressed.

## UI Polish Recommendation

- Recommendation: Optional
- Reason: The new Feed UI is coherent and covered by widget tests, but this is a new user-facing screen with simple compose, empty, error, and pagination states. A small visual/accessibility smoke pass could refine spacing/copy/semantics without changing behavior.
- Suggested polish notes: If run, limit polish to copy, spacing, visual states, and accessibility labels for the compose entry, empty state, error state, and load-more retry affordance.

## Handoff Back To TDD Builder

- Required fixes:
  - Address `IR-001` by adding/adjusting an automated test for timeline reply/comment state updates.
- Suggested next failing test:
  - In `app/test/feed/feed_page_test.dart` or `app/test/feed/providers/timeline_provider_test.dart`, create a live timeline with root post `replyCount: 3`, submit a reply from the timeline, then assert the live timeline root row has `replyCount: 4`, `viewerHasReplied == true`, and does not contain the created reply URI as a top-level item.
- Verification to rerun:
  - `flutter test test/feed/feed_page_test.dart test/feed/providers/timeline_provider_test.dart test/feed/providers/create_post_provider_test.dart`
  - `flutter test test/feed/providers/toggle_post_interactions_provider_test.dart test/feed/data/post_api_client_test.dart test/feed/data/post_repository_test.dart`
  - Planned regression command from `05-implementation-plan.md`.
