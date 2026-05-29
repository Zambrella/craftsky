# TDD Implementation Plan: Timeline Feed Flutter

## Inputs

- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` — Approved with notes, no blockers
- Coding plan: `04-coding-plan.md` — Plannotator approved

## Implementation Rules

- Do not implement behavior without linked requirement IDs.
- Write or update one failing test before implementation.
- Run the smallest relevant test first.
- Refactor only after tests pass.
- Keep traceability updated with requirement IDs, acceptance criteria, test IDs, red failures, and green results.
- Preserve architecture boundaries: Flutter reads timeline from AppView via the existing Dio/repository stack; no PDS read-through, AppView/Go, lexicon, migration, dependency, or endpoint-contract changes.
- Reuse existing `Post`, `PostPage`, `PostCard`, composer, route, and interaction-provider patterns; no generic feed framework or feed-item envelope.
- Keep timeline cursors opaque and match live timeline cache rows by stable `Post.uri`.

## Test Order

| Step | Test ID | Requirement IDs | Acceptance Criteria | Focused Target | Expected Initial State | Status |
|---|---|---|---|---|---|---|
| 1 | IT-001 | FR-001, NFR-001, NFR-002, BR-001 | AC-001, AC-002 | `app/test/feed/data/post_api_client_test.dart` | `PostApiClient.listTimeline` missing | Completed |
| 2 | IT-002 | FR-001, FR-005, RULE-002 | AC-006, AC-014 | `app/test/feed/data/post_api_client_test.dart` | Method missing or query params absent | Completed |
| 3 | IT-003 | FR-001 | AC-002 | `app/test/feed/data/post_api_client_test.dart` | Empty/error behavior missing | Completed |
| 4 | IT-004 | FR-002, RULE-001 | AC-002 | repository/fake/provider tests | Repository lacks timeline method | Completed |
| 5 | UT-001 | FR-002, RULE-001 | AC-002 | fake repository usage | Fake lacks timeline method without handle/DID | Completed |
| 6 | UT-002 | FR-003, FR-004 | AC-003 | `app/test/feed/providers/timeline_provider_test.dart` | `TimelineState`/`timelineProvider` missing | Completed |
| 7 | UT-003 | FR-003, FR-005, RULE-002 | AC-006, AC-014 | `app/test/feed/providers/timeline_provider_test.dart` | `loadMore` missing or cursor not passed | Completed |
| 8 | UT-004 | FR-007 | AC-007 | `app/test/feed/providers/timeline_provider_test.dart` | Previous data/cursor not preserved on error | Completed |
| 9 | UT-005 | FR-003 | AC-006 | `app/test/feed/providers/timeline_provider_test.dart` | Terminal/concurrent guards missing | Completed |
| 10 | UT-006 | FR-015, RULE-003 | AC-017 | create provider + timeline tests | Top-level create updates only profile caches | Completed |
| 11 | UT-007 | FR-015, RULE-004 | AC-018 | `app/test/feed/providers/timeline_provider_test.dart` | Timeline duplicates URI rows | Completed |
| 12 | AT-001 | BR-001 | AC-005 | `app/test/feed/feed_page_test.dart` | FeedPage still placeholder/static body | Completed |
| 13 | AT-002 | BR-001, FR-009 | AC-004 | `app/test/feed/feed_page_test.dart` | Loaded timeline rows not rendered as PostCards | Completed |
| 14 | AT-003 | FR-008, NFR-004 | AC-009, AC-020 | `app/test/feed/feed_page_test.dart` | Empty feed copy/state missing | Completed |
| 15 | AT-004 | FR-006, NFR-004 | AC-008, AC-020 | `app/test/feed/feed_page_test.dart` | Initial error retry missing | Completed |
| 16 | AT-005 | FR-003, FR-005, NFR-003, RULE-002 | AC-006, AC-019 | `app/test/feed/feed_page_test.dart` | Scroll pagination missing | Completed |
| 17 | AT-006 | FR-007 | AC-007 | `app/test/feed/feed_page_test.dart` | Load-more error blanks list/cannot retry | Completed |
| 18 | AT-007 | FR-010 | AC-010 | `app/test/feed/feed_page_test.dart` | Row tap not wired to thread route | Completed |
| 19 | AT-008 | FR-011 | AC-011 | FeedPage + interaction provider tests | Timeline row not updated by like/repost | Completed |
| 20 | AT-009 | FR-012, RULE-003 | AC-012 | FeedPage + timeline helper tests | Reply does not update root or inserts as row | Completed |
| 21 | AT-010 | FR-013 | AC-015 | FeedPage + timeline helper tests | Delete visibility/removal missing | Completed |
| 22 | AT-011 | FR-014 | AC-016 | `app/test/feed/feed_page_test.dart` | Top-level compose entry missing | Completed |
| 23 | AT-012 | FR-015, RULE-003, RULE-004 | AC-017, AC-018 | `app/test/feed/feed_page_test.dart` | Created post not prepended/deduped | Completed |
| 24 | UT-008 | FR-011 | AC-011 | `app/test/feed/providers/toggle_post_interactions_provider_test.dart` | Like/repost providers do not patch/rollback timeline | Completed |
| 25 | UT-009 | FR-012, RULE-003 | AC-012 | timeline provider/helper tests | Reply helper missing | Completed |
| 26 | UT-010 | FR-013 | AC-015 | timeline provider/helper tests | Delete helper missing | Completed |
| 27 | REG-001..REG-008 | BR-002, FR-009, FR-011, FR-012, FR-014, NFR-001, NFR-002, NFR-004, NFR-005 | AC-013, AC-020, AC-021 | listed regression commands | Shared/generated regressions surface | Completed with unrelated full-suite failure noted |

## Implementation Steps

### Step 1: IT-001

- Write failing test: Added `PostApiClient.listTimeline` no-cursor test for `GET /v1/feed/timeline` parsing `{items, cursor}` into `PostPage`.
- Run command: `flutter test test/feed/data/post_api_client_test.dart --plain-name "PostApiClient.listTimeline"`
- Confirmed failure: Compile failure: `The method 'listTimeline' isn't defined for the type 'PostApiClient'.`
- Implement: Added `PostApiClient.listTimeline({String? cursor, int? limit})` using existing `unwrapApi`, shared `Dio`, `/v1/feed/timeline`, and `PostPageMapper.fromMap`.
- Green command: `flutter test test/feed/data/post_api_client_test.dart --plain-name "PostApiClient.listTimeline"` → `+1 All tests passed`.
- Refactor: None.
- Notes: Covers AppView timeline read path and post-shaped response parsing without PDS read-through or a new feed-item model.

### Step 2: IT-002

- Write failing test: Added `PostApiClient.listTimeline` test expecting exact query parameters `cursor=opaque:abc` and `limit=20`.
- Run command: `flutter test test/feed/data/post_api_client_test.dart --plain-name "passes cursor and limit as query params"`
- Confirmed failure: Not red; the minimal Step 1 implementation already matched existing client patterns and included optional cursor/limit pass-through.
- Implement: No additional production change required.
- Green command: `flutter test test/feed/data/post_api_client_test.dart --plain-name "passes cursor and limit as query params"` → `+3 All tests passed` (matched existing same-name author/comment tests too).
- Refactor: None.
- Notes: Cursor remains an opaque string and is forwarded unchanged.

### Step 3: IT-003

- Write failing test: Added timeline empty-page and server-error mapping tests in `PostApiClient.listTimeline` group.
- Run command: `flutter test test/feed/data/post_api_client_test.dart --plain-name "PostApiClient.listTimeline"`
- Confirmed failure: Not red for production behavior; existing `unwrapApi` + `PostPageMapper` behavior already handles empty pages and shared error mapping once `listTimeline` existed. Test-placement/expectation mistakes were corrected before green.
- Implement: No additional production change required.
- Green command: `flutter test test/feed/data/post_api_client_test.dart --plain-name "PostApiClient.listTimeline"` → `+4 All tests passed`; nearby command `flutter test test/feed/data/post_api_client_test.dart` → `+23 All tests passed`.
- Refactor: None.
- Notes: For 5xx responses, current shared error mapping exposes `ApiServerError('http_500')`; timeline uses the same path.

### Step 4: IT-004 / UT-001

- Write failing test: Added `post_repository_test.dart` proving `PostRepository.listTimeline({cursor, limit})` is available through `FakePostRepository` with no handle/DID input.
- Run command: `flutter test test/feed/data/post_repository_test.dart`
- Confirmed failure: Compile failure: `onListTimeline` named parameter missing and `PostRepository.listTimeline` undefined.
- Implement: Added `listTimeline` to `PostRepository`, delegated in `ApiPostRepository`, and added `onListTimeline` to `FakePostRepository`.
- Green command: `flutter test test/feed/data/post_repository_test.dart` → `+1 All tests passed`.
- Refactor: None.
- Notes: No new `PostApiClient` abstraction was added; follows DR-001.

### Step 5: UT-002

- Write failing test: Added `timeline_provider_test.dart` first-build test for bounded page size, returned items/cursor, and `hasMore`.
- Run command: `flutter test test/feed/providers/timeline_provider_test.dart`
- Confirmed failure: Compile failure: missing `timeline_provider.dart`, `timelineProvider`, and `timelinePageLimit`.
- Implement: Added `TimelineState` mappable model and generated `timelineProvider` using `repo.listTimeline(limit: timelinePageLimit)` with `timelinePageLimit = 20`.
- Green command: `flutter test test/feed/providers/timeline_provider_test.dart` → `+1 All tests passed`.
- Refactor: None.
- Notes: Ran `dart run build_runner build --delete-conflicting-outputs` to create generated mapper/provider outputs; build_runner warned the option is ignored by current tooling.

### Step 6: UT-003

- Write failing test: Added `loadMore` test that records exact opaque cursor and expects appended rows.
- Run command: `flutter test test/feed/providers/timeline_provider_test.dart --plain-name "passes opaque cursor and appends next page"`
- Confirmed failure: Compile failure: `Timeline.loadMore` missing.
- Implement: Added `Timeline.loadMore` with previous-state preservation pattern and append-dedupe merge.
- Green command: same focused command → passed.
- Refactor: None.
- Notes: Cursor is passed through unchanged.

### Step 7: UT-004 / UT-005 / UT-007

- Write failing tests: Added provider tests for load-more failure preserving previous data/cursor, terminal/concurrent no-op guards, prepend duplicate protection, and fetched-page URI dedupe.
- Run command: `flutter test test/feed/providers/timeline_provider_test.dart`
- Confirmed failure: Prepend test failed at compile time until `Timeline.prepend` existed; other guard/error behavior was already covered by the `loadMore` implementation.
- Implement: Added `Timeline.prepend` and `Timeline.replace`; existing `loadMore` guarded and preserved previous state via `copyWithPrevious`.
- Green command: `flutter test test/feed/providers/timeline_provider_test.dart` → `+7 All tests passed`.
- Refactor: None.
- Notes: URI-based dedupe is enforced for initial build, prepend, and next-page append.

### Step 8: UT-006

- Write failing test: Added `CreatePost` tests for top-level success prepending into live timeline and reply success not inserting a timeline row.
- Run command: `flutter test test/feed/providers/create_post_provider_test.dart --plain-name "top-level success prepends into live timeline provider"`
- Confirmed failure: Timeline stayed `['old']`; `CreatePost` updated only profile caches.
- Implement: Added `prependLiveTimelineCache` helper and called it for top-level creates only.
- Green command: focused top-level and reply commands passed.
- Refactor: None.
- Notes: Reply path remains non-row insertion per `RULE-003`.

### Step 9: AT-001 through AT-004

- Write failing tests: Replaced placeholder FeedPage test with loading, loaded post-card, empty state, and initial-error retry tests using `FakePostRepository`.
- Run command: `flutter test test/feed/feed_page_test.dart`
- Confirmed failure: Initial error test failed until FeedPage preferred `timelineAsync.hasError` over loading when Riverpod retained an error during retry state.
- Implement: Replaced placeholder `FeedPage` with timeline-backed `CustomScrollView`, loading/error/empty/loaded slivers, top compose button, post-card rows, and load-more affordance. Added `feedEmpty` and `feedLoadError` localized strings and generated l10n outputs.
- Green command: `flutter test test/feed/feed_page_test.dart` → `+4 All tests passed`.
- Refactor: None.
- Notes: Row navigation/interactions/delete are still pending in later planned AT/UT loops.

### Step 10: AT-005 / AT-006

- Write failing tests: Added FeedPage widget tests for scroll-triggered append and load-more failure retry with same cursor.
- Run command: focused `flutter test test/feed/feed_page_test.dart --plain-name ...` commands.
- Confirmed failure: AT-005 initially did not show appended page until the test scrolled to the appended row; AT-006 initially could not find bottom retry until the test scrolled to the bottom affordance.
- Implement: Existing `Timeline.loadMore` and FeedPage bottom affordance covered behavior; adjusted tests to assert visible lazy-list behavior correctly.
- Green command: focused AT-005/AT-006 commands passed.
- Refactor: None.
- Notes: UI remains lazy/sliver-based; tests assert cursor `c1` is reused.

### Step 11: AT-007

- Write failing test: Added FeedPage router test for tapping a row and capturing `/posts/:did/:rkey` params.
- Run command: `flutter test test/feed/feed_page_test.dart --plain-name "FeedPage row tap opens thread route"`
- Confirmed failure: Thread route was not reached.
- Implement: Wired `PostCard.onTap` to `PostThreadRoute(did: post.author.did, rkey: post.rkey)`.
- Green command: focused AT-007 command passed.
- Refactor: None.

### Step 12: AT-008 / UT-008

- Write failing tests: Added interaction-provider tests for live timeline patch/rollback and FeedPage like/repost action test.
- Run command: `flutter test test/feed/providers/toggle_post_interactions_provider_test.dart --plain-name "patches and rolls back live timeline entries"`
- Confirmed failure: Timeline row remained unliked/unreposted; FeedPage actions were not wired.
- Implement: Added `updateLiveTimelineCache` calls to like/repost providers on optimistic update and rollback; wired FeedPage `onLike`/`onRepost` to existing providers.
- Green command: focused provider and FeedPage interaction commands passed.
- Refactor: None.

### Step 13: AT-009 / UT-009

- Write failing test: Added FeedPage reply composer test that opens focused thread with created reply as route extra.
- Run command: `flutter test test/feed/feed_page_test.dart --plain-name "FeedPage reply opens focused thread and updates root row"`
- Confirmed failure: No reply composer text field appeared because timeline rows had no `onReply`.
- Implement: Wired `PostCard.onReply`; after composer success, replace the root timeline row with incremented `replyCount` and `viewerHasReplied: true`, then push focused thread route.
- Green command: focused AT-009 command passed.
- Refactor: None.
- Notes: Reply is not inserted as a top-level timeline row; existing create-provider test also verifies reply creates do not prepend into timeline.

### Step 14: AT-010 / UT-010

- Write failing tests: Added `Timeline.removeByUri` provider test and FeedPage delete visibility/removal test for signed-in owner vs other author.
- Run command: focused timeline remove and FeedPage delete commands.
- Confirmed failure: `Timeline.removeByUri` was undefined and FeedPage own row menu had no delete item.
- Implement: Added `Timeline.removeByUri`, `removeFromLiveTimelineCache`, delete-provider timeline removal after repository success, FeedPage auth-gated delete callback, delete confirmation, and delete feedback listener.
- Green command: focused remove/delete commands passed.
- Refactor: None.

### Step 15: AT-011 / AT-012

- Write failing test: Added FeedPage compose test that opens top-level composer from `New post`, verifies `reply == null`, and observes created post prepended into the visible timeline.
- Run command: `flutter test test/feed/feed_page_test.dart --plain-name "FeedPage compose creates top-level post and prepends it"`
- Confirmed failure: Composer success needed a `MessengerScope` in the test harness; FeedPage already exposed the compose entry and create-provider timeline prepend was in place.
- Implement: Updated test harness to include `MessengerScope`; no extra production change needed.
- Green command: focused AT-011/AT-012 compose command passed.
- Refactor: None.
- Notes: Widget test covers top-level create/prepend; provider tests cover URI dedupe across prepend and fetched pages.

### Step 16: Focused timeline suite

- Run command: `flutter test test/feed/feed_page_test.dart test/feed/providers/timeline_provider_test.dart test/feed/providers/create_post_provider_test.dart test/feed/providers/toggle_post_interactions_provider_test.dart test/feed/data/post_api_client_test.dart test/feed/data/post_repository_test.dart`
- Result: `+61 All tests passed`.

### Step 17: REG-001 through REG-008 and final verification

- Run command: `flutter test test/feed/widgets/post_card_test.dart test/feed/widgets/post_image_carousel_test.dart test/feed/widgets/post_image_gallery_test.dart test/feed/widgets/post_composer_sheet_discard_test.dart test/feed/providers/create_post_provider_test.dart test/feed/providers/toggle_post_interactions_provider_test.dart test/profile/widgets/profile_posts_tab_test.dart test/profile/widgets/profile_comments_tab_test.dart test/feed/providers/user_posts_provider_test.dart test/shared/api/providers/session_auth_interceptor_test.dart test/shared/device/device_id_provider_test.dart test/shared/api/providers/dio_provider_test.dart test/router/router_redirect_test.dart test/app_test.dart`
- Result: `+106 All tests passed`.
- Run command: `dart run build_runner build --delete-conflicting-outputs`
- Result: built successfully, wrote `0 outputs`. Tooling warned that `--delete-conflicting-outputs` is ignored by the current build_runner and that SDK language version `3.12.0` is newer than analyzer language version `3.11.0`.
- Run command: `flutter test test/feed/feed_page_test.dart test/feed/providers/timeline_provider_test.dart test/feed/providers/create_post_provider_test.dart test/feed/providers/toggle_post_interactions_provider_test.dart test/feed/data/post_api_client_test.dart test/feed/data/post_repository_test.dart`
- Result: `+61 All tests passed`.
- Run command: `flutter test`
- Result: failed with one unrelated pre-existing/stale profile expectation: `test/profile/profile_page_test.dart` / `ProfilePage visitor profile renders Follow + Share actions` expected text `2 posts in the last 7 days`, while current `ProfileStats` renders the metric as separate `2 posts` and `7 days` labels. No `app/lib/profile/**` or `app/test/profile/profile_page_test.dart` files were changed by this implementation stage, and the planned profile widget regressions passed.
- Scope review: Changed files are limited to Flutter timeline API/repository/provider/UI, generated provider/l10n outputs, timeline tests/fakes, router test stubbing, and this implementation log. No AppView/Go, lexicon, migration, dependency, PDS-read-through, generic feed-framework, recommendation, ranking, or durable-cache changes were introduced.

## Execution Notes

- Created from approved coding plan on 2026-05-29.
- `IT-002` and `IT-003` did not produce production red failures after the minimum `IT-001` implementation because the initial method already followed existing optional query-parameter and shared error-mapping patterns; tests still provide regression coverage.
- Some widget test adjustments were test-harness corrections rather than production changes: lazy sliver assertions scroll to relevant rows/affordances, and feed compose tests include `MessengerScope` because the existing composer reports through the messenger stack.
- Full-suite verification currently has one failing profile-page test unrelated to this change; focused timeline and planned regression suites are green.

## Implementation Review Fixes

### Review Fix 1: IR-001 / AT-009 / UT-009

- Requirement IDs: `FR-012`, `RULE-003`
- Acceptance Criteria: `AC-012`
- Target: `app/test/feed/feed_page_test.dart`
- Planned failing test: Strengthen the existing timeline reply widget test to inspect live `timelineProvider` state after reply creation and assert the root row has `replyCount + 1`, `viewerHasReplied == true`, and the created reply URI is absent as a top-level timeline row.
- Run command: `flutter test test/feed/feed_page_test.dart --plain-name "FeedPage reply opens focused thread and updates root row"`
- Confirmed failure: No production red failure; the source already updated the timeline row correctly. This was the review-identified missing assertion/traceability gap.
- Implement: Strengthened the widget test to keep and inspect the same `ProviderContainer`, assert the focused thread route still opens, and verify live `timelineProvider` contains only the root row with `replyCount` incremented from `3` to `4` and `viewerHasReplied == true`.
- Green command: `flutter test test/feed/feed_page_test.dart --plain-name "FeedPage reply opens focused thread and updates root row"` → `+1 All tests passed`.
- Notes: Covers `AT-009` / `UT-009` without adding production behavior; reply URI absence is asserted against top-level timeline items.

### Review Fix 2: IR-002 / AT-010

- Requirement IDs: `FR-013`
- Acceptance Criteria: `AC-015`
- Target: `app/test/feed/feed_page_test.dart`
- Planned failing test: Strengthen the existing delete widget test to explicitly open/check the non-owned row menu and verify `Delete post` is absent before deleting the owned row.
- Run command: `flutter test test/feed/feed_page_test.dart --plain-name "FeedPage only exposes delete for own rows and removes row"`
- Confirmed failure: No production red failure; the existing `FeedPage` DID gate already hid delete for non-owned rows. This was a non-blocking assertion-strengthening request.
- Implement: Extended the delete widget test to open the non-owned row menu first and assert `Delete post` is absent, then dismiss it and continue through owned-row delete confirmation/removal.
- Green command: `flutter test test/feed/feed_page_test.dart --plain-name "FeedPage only exposes delete for own rows and removes row"` → `+1 All tests passed`.
- Notes: No production code changed.

### Review Fix Verification

- Focused command: `flutter test test/feed/feed_page_test.dart test/feed/providers/timeline_provider_test.dart test/feed/providers/create_post_provider_test.dart`
- Result: `+29 All tests passed`.
- Broader command: `flutter test test/feed/providers/toggle_post_interactions_provider_test.dart test/feed/data/post_api_client_test.dart test/feed/data/post_repository_test.dart`
- Result: `+32 All tests passed`.
- Analyzer: Dart MCP `analyze_files` on `test/feed/feed_page_test.dart` → no errors.
- Scope review: Changes are limited to `app/test/feed/feed_page_test.dart` test coverage and this implementation log. No production code, generated files, dependencies, AppView/Go, lexicon, migration, API contract, PDS-read-through, or generic feed-framework changes were made for these review fixes.

## Completion Checklist

- [x] All Must requirements covered by tests or documented gaps.
- [x] All planned Must tests passing.
- [x] Should tests (`FR-008`, `FR-013`, `NFR-003`) passing or documented if skipped.
- [x] Relevant regression tests passing.
- [x] Generated files updated and verified.
- [x] No unlinked behavior implemented.
- [x] No AppView/Go, lexicon, migration, dependency, PDS-read-through, or generic feed-framework changes.
- [x] `05-implementation-plan.md` updated with actual red/green commands and deviations.
- [x] Stage completion commit created before the exit gate.
