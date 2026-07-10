# TDD Implementation Plan: Reposts And Quote Posts

## Inputs
- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md`
- Coding plan: `04-coding-plan.md`

## Implementation Rules
- Do not implement behavior without a linked requirement ID.
- Write or update a failing test before implementation.
- Run the smallest relevant test first.
- Refactor only after tests pass.
- Keep traceability updated after each TDD loop.
- Do not change lexicon files unless implementation uncovers a concrete need.
- Do not touch migrations without explicit approval.

## Test Order
| Step | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| 1 | IT-001 | BR-001, FR-005, FR-007, RULE-004 | AC-003, AC-009, AC-029 | Fails because `ListTimeline` returns post rows only and excludes repost activity. |
| 2 | IT-004 | BR-003, NFR-001 | AC-005, AC-020 | Fails because mixed post/repost cursor identity does not exist. |
| 3 | IT-005 | FR-017, NFR-002 | AC-032, AC-021 | Fails because the home timeline handler returns bare posts. |
| 4 | UT-004 | FR-007, FR-010, FR-011, RULE-004 | AC-009, AC-013, AC-014 | Fails because response builders do not expose quote count or repost reason shape. |
| 5 | UT-006 | FR-010 | AC-013, AC-031 | Fails because Flutter `Post` lacks quote-count/quote-preview support. |
| 6 | UT-012 / IT-012 | NFR-002 | AC-021 | Fails if new response/request fields or validation errors are not camelCase/enveloped. |
| 7 | IT-003 | FR-008, FR-009, RULE-005 | AC-011, AC-012, AC-030 | Fails because quote previews are not hydrated. |
| 8 | IT-010 | RULE-006, RULE-010 | AC-012, AC-026, AC-039 | Fails if repost subjects or quote previews bypass moderation filtering. |
| 9 | IT-016 | RULE-011, FR-010 | AC-040, AC-013 | Fails because quote counts are not implemented and count filtering is incomplete. |
| 10 | IT-006 | FR-001, RULE-002 | AC-001, AC-007 | Expected to protect existing repost write semantics while adding eligibility. |
| 11 | IT-007 | FR-002, FR-004, RULE-008 | AC-002, AC-008, AC-028, AC-037 | Fails because quote/repost target eligibility is not enforced. |
| 12 | IT-008 | FR-011, FR-012 | AC-014, AC-015 | Fails if quote posts affect straight repost state or unrepost behavior. |
| 13 | IT-009 | RULE-003 | AC-025 | Expected to pass after quote create follows normal post semantics. |
| 14 | IT-015 | RULE-009 | AC-038 | Fails if share eligibility rejects self-share. |
| 15 | UT-007 | FR-003, FR-018, RULE-008 | AC-006, AC-027, AC-033 | Fails because `PostCard` directly toggles repost and replies expose current action UI. |
| 16 | UT-008 | FR-010 | AC-031 | Fails because action-row count renders repost count only. |
| 17 | UT-009 | FR-019, NFR-005 | AC-024, AC-034 | Fails if optimistic repost cache code inserts/removes timeline items. |
| 18 | UT-010 | FR-020, NFR-005 | AC-024, AC-035 | Fails if create-post provider cannot pass quote refs through normal cache behavior. |
| 19 | AT-006 through AT-010 | FR-003, FR-004, FR-008, FR-019, FR-020 | AC-006, AC-011, AC-024, AC-034, AC-035 | Fails until UI/provider flows are complete. |
| 20 | REG-001 through REG-007 | FR-013, FR-014, FR-015, FR-016, FR-021 | AC-016, AC-017, AC-018, AC-019, AC-036 | Expected to catch accidental scope expansion. |

## Implementation Steps
### Step 1: IT-001
- Write failing test: Extend `appview/internal/api/timeline_store_test.go` so followed straight reposts appear as distinct feed items with `reason.type == "repost"` and reposter attribution.
- Run command: `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test -run 'TestTimelineStore_ListTimeline' ./internal/api`
- Confirmed failure: `go test -run 'TestTimelineStore_ListTimeline_IncludesFollowedRepostActivityWithReasonAndExcludesReplies' ./internal/api` first failed because `api.TimelineFeedItemRow`, item keys, and repost reason fields did not exist.
- Implement: Added `TimelineFeedItemRow` / `TimelineRepostReasonRow`, changed `PostStore.ListTimeline` to return mixed authored-post and straight-repost activity rows, and updated the timeline handler fake boundary to compile against feed items.
- Run command: `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test -run 'TestTimelineStore' ./internal/api`
- Refactor: Kept handler JSON output unchanged for the later `IT-005` loop; only the store boundary and query were changed.
- Notes: `just dev-d` was required because compose Postgres was not running on `localhost:5433`.

### Step 2: IT-004
- Write failing test: Added `TestTimelineStore_ListTimeline_PaginatesMixedPostsAndRepostsWithFeedItemCursor` covering tied post/repost activity, feed-item cursor identity, and no duplicate across pages.
- Run command: `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test -run 'TestTimelineStore_ListTimeline_PaginatesMixedPostsAndRepostsWithFeedItemCursor' ./internal/api`
- Confirmed failure: The cursor payload lacked `itemKey` because timeline pagination still encoded the old `indexedAt`/`uri` pair.
- Implement: Added `decodeTimelineCursor` and changed timeline cursor payloads to `activityAt` + `itemKey`.
- Run command: `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test -run 'TestTimelineStore' ./internal/api`
- Refactor: Corrected the new test to treat an omitted cursor as exhausted pagination rather than issuing another request with an empty cursor.
- Notes: Existing post-only pagination tests continue to pass with feed-item cursor identity.

### Step 3: IT-005
- Write failing test: Added `TestTimelineHandler_ReturnsFeedItemsWithRepostReason` to require `items: [{itemKey, post, reason}]` with hydrated reposter actor metadata.
- Run command: `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test -run 'TestTimelineHandler_ReturnsFeedItemsWithRepostReason' ./internal/api`
- Confirmed failure: The handler returned bare post objects in `items`, with no `itemKey`, `post`, or `reason` wrapper.
- Implement: Added `TimelineFeedItemResponse` and `TimelineReasonRepost`, wrapped timeline posts in feed items, resolved reposter handles, and kept repost attribution off `PostResponse`.
- Run command: `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test -run 'TestTimelineHandler' ./internal/api`
- Refactor: Updated existing handler tests to assert post fields through `item.post`.
- Notes: Only home timeline handler shape changed; non-home post-shaped surfaces are untouched in this loop.

### Step 4: UT-004
- Write failing test: Added `TestBuildPostResponse_JSONIncludesQuoteCount` to require the separate `quoteCount` response field.
- Run command: `cd appview && go test -run 'TestBuildPostResponse_JSONIncludesQuoteCount' ./internal/api`
- Confirmed failure: Marshaled `PostResponse` contained `repostCount` and `replyCount` but no `quoteCount`.
- Implement: Added `PostResponse.QuoteCount`, `EngagementSummary.QuoteCount`, and assignment in `applyEngagementSummary`.
- Run command: `cd appview && go test -run 'TestBuildPostResponse|TestTimelineHandler_ReturnsFeedItemsWithRepostReason' ./internal/api`
- Refactor: None.
- Notes: Store-side quote counting is still pending under `IT-016`; this loop only created the response/model field and repost reason response coverage.

### Step 5: UT-006
- Write failing test: Added Flutter model tests for `quoteCount`, `quoteView`, visible quote-preview data, and absent-field defaults in `app/test/feed/models/post_test.dart`.
- Run command: `cd app && flutter test test/feed/models/post_test.dart`
- Confirmed failure: Compilation failed because `Post.quoteCount` and `Post.quoteView` did not exist.
- Implement: Added `Post.quoteCount`, `Post.quoteView`, `QuoteView`, and `QuotePreviewPost`; regenerated `post.mapper.dart`.
- Run command: `cd app && flutter test test/feed/models/post_test.dart`
- Refactor: Removed unrelated generated churn from build_runner output; kept only `post.mapper.dart`.
- Notes: `quoteCount` defaults to `0` for older payloads, while AppView now emits `quoteCount` in new responses.

### Step 6: UT-012 / IT-012
- Write failing test: Added `TestTimelineFeedItemResponse_JSONUsesCamelCase` for `itemKey`, `quoteCount`, `displayName`, `createdAt`, and `indexedAt`.
- Run command: `cd appview && go test -run 'TestTimelineFeedItemResponse_JSONUsesCamelCase' ./internal/api`
- Confirmed failure: The test passed on first run because Steps 3 and 4 had already introduced the needed camelCase tags.
- Implement: No code change required in this loop.
- Run command: `cd appview && go test -run 'TestBuildPostResponse|TestTimelineFeedItemResponse|TestTimelineHandler' ./internal/api`
- Refactor: None.
- Notes: Error envelope coverage remains in the existing timeline invalid-cursor handler test; quote validation error casing will be revisited when write validation is implemented.

### Step 7: IT-003
- Write failing test: Added store, response-builder, and post-detail handler tests for visible, hidden, unavailable, and quote-of-quote preview behavior.
- Run command: `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test -run 'TestPostStore_QuoteViewRows|TestBuildQuoteView|TestGetPost_WithQuote_AttachesCompactQuoteView' ./internal/api`
- Confirmed failure: `QuoteViewRows`, `QuoteViewRow`, and `BuildQuoteView` were missing; after adding them, the post-detail handler still did not request quote views.
- Implement: Added quote-view row/response types, batched `PostStore.QuoteViewRows`, compact `BuildQuoteView`, `PostResponse.quoteView`, and `GetPostHandler` quote-view attachment.
- Run command: `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api`
- Refactor: Kept quote hydration handle resolution outside `PostStore` so storage remains independent of identity resolution.
- Notes: Compact previews intentionally do not include nested quote views; hidden indexed targets return `state: "hidden"`, missing/unindexed refs return `state: "unavailable"`.

### Step 8: IT-010
- Write failing test: Added `TestTimelineStore_ListTimeline_OmitsRepostsOfHiddenSubjects`; reused `TestPostStore_QuoteViewRows_ReturnsVisibleHiddenAndUnavailableStates` for hidden quote preview state.
- Run command: `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test -run 'TestTimelineStore_ListTimeline_OmitsRepostsOfHiddenSubjects|TestPostStore_QuoteViewRows' ./internal/api`
- Confirmed failure: The tests passed on first run because Step 1's repost timeline query already used `postVisibleModerationPredicate`, and Step 7 implemented hidden quote preview state.
- Implement: No code change required in this loop.
- Run command: `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test -run 'TestTimelineStore|TestPostStore_QuoteViewRows' ./internal/api`
- Refactor: None.
- Notes: Straight reposts of hidden subjects are omitted entirely; quote posts can remain renderable with a hidden placeholder for their target.

### Step 9: IT-016
- Write failing test: Extended `TestPostStore_EngagementSummaries_ActiveOnlyAndViewerStates` with visible quote posts and a hidden quote post.
- Run command: `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test -run 'TestPostStore_EngagementSummaries_ActiveOnlyAndViewerStates' ./internal/api`
- Confirmed failure: `post1` returned `QuoteCount: 0` instead of the two visible quote posts.
- Implement: Added `CountVisibleQuotes` and wired `QuoteCount` into `EngagementSummaries`.
- Run command: `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api`
- Refactor: None.
- Notes: Quote counts currently count visible top-level quote posts whose own post rows pass AppView moderation filtering.

### Step 10: IT-006
- Write failing test: Existing tests already covered `social.craftsky.feed.repost` PDS write shape and duplicate active repost idempotency.
- Run command: `cd appview && go test -run 'TestRepostPost_CreatesPDSRepostRecord|TestRepostPost_AlreadyRepostedReturnsExistingIdentity|TestUnrepostPost_AbsentActiveRepostIsIdempotent' ./internal/api`
- Confirmed failure: No failure; this loop protected existing behavior before target-eligibility changes.
- Implement: No code change required in this loop.
- Run command: Same focused command.
- Refactor: None.
- Notes: The next loop changes validation behavior while preserving these repost write/idempotency contracts.

### Step 11: IT-007
- Write failing test: Added store and handler tests for share-target eligibility: visible root/project targets resolve, hidden targets are treated as not found, reply targets are rejected before any PDS write for both straight reposts and quote posts.
- Run command: `cd appview && go test -run 'TestRepostPost_RejectsReplyTargetBeforePDSWrite|TestCreatePost_QuoteRejectsReplyTargetBeforePDSWrite|TestPostStore_ResolveShareTarget_ReturnsEligibilityAndFiltersHidden' ./internal/api`
- Confirmed failure: Initial compile/test failure because `ShareTargetRef`, `ResolveShareTarget`, and reply-target validation did not exist.
- Implement: Added `ShareTargetRef` and `PostStore.ResolveShareTarget`; wired repost and quote-create validation through it; rejected reply targets with `validation_failed` before PDS calls; preserved like/unlike and unrepost lookup behavior on the existing `ResolvePostTarget` path.
- Run command: `cd appview && go test -run 'TestPostStore_ResolveShareTarget|TestRepostPost_RejectsReplyTargetBeforePDSWrite|TestCreatePost_QuoteRejectsReplyTargetBeforePDSWrite|TestCreatePost_QuoteEmbed_TranslatedToLexiconShape|TestRepostPost_CreatesPDSRepostRecord|TestLikePost_CreatesPDSLikeRecord|TestRepostPost_TargetLookupFailureReturns500' ./internal/api`
- Refactor: Kept share eligibility separate from generic post target resolution so delete/idempotency paths do not become dependent on moderation visibility or reply eligibility.
- Notes: Focused Step 11 tests pass. A broad `go test ./internal/api` attempt was interrupted after hanging and is not counted as final verification.

### Step 12: IT-008
- Write failing test: Added store coverage that a quote-only viewer has `quoteCount` without `viewerHasReposted`, and handler coverage that unrepost deletes only the active `social.craftsky.feed.repost` record even when the viewer also has quote activity for the subject.
- Run command: `cd appview && go test -run 'TestPostStore_EngagementSummaries_QuoteOnlyViewerDoesNotSetReposted|TestUnrepostPost_WithAuthoredQuoteDeletesOnlyStraightRepost|TestUnrepostPost_ExistingDeletesPDSRecord|TestUnrepostPost_AbsentActiveRepostIsIdempotent' ./internal/api`
- Confirmed failure: The focused tests passed on first run because quote posts and straight repost interactions were already stored/read through separate paths.
- Implement: No production code change required.
- Run command: Same focused command.
- Refactor: None.
- Notes: This locks down `FR-011`/`FR-012`: quote state does not toggle `viewerHasReposted`, and unrepost remains scoped to the straight repost record.

### Step 13: IT-009
- Write failing test: Added `TestCreatePost_AllowsMultipleQuotePostsForSameSubject` to submit two quote posts against the same target and require two independent PDS create calls.
- Run command: `cd appview && go test -run 'TestCreatePost_AllowsMultipleQuotePostsForSameSubject|TestCreatePost_QuoteEmbed_TranslatedToLexiconShape|TestCreatePost_QuoteRejectsReplyTargetBeforePDSWrite' ./internal/api`
- Confirmed failure: The focused tests passed on first run because quote posts already use the normal create-post path instead of repost-style idempotency.
- Implement: No production code change required.
- Run command: Same focused command.
- Refactor: None.
- Notes: This locks down `RULE-003`: quote posts are authored posts and may be created multiple times for the same subject.

### Step 14: IT-015
- Write failing test: Added handler tests for self-reposting and self-quoting eligible own posts.
- Run command: `cd appview && go test -run 'TestRepostPost_AllowsSelfRepost|TestCreatePost_AllowsSelfQuote|TestRepostPost_CreatesPDSRepostRecord|TestCreatePost_QuoteEmbed_TranslatedToLexiconShape' ./internal/api`
- Confirmed failure: The focused tests passed on first run because the eligibility path does not reject matching caller/target DIDs.
- Implement: No production code change required.
- Run command: Same focused command.
- Refactor: None.
- Notes: This locks down `RULE-009`: self-reposts and self-quotes are allowed when the target is otherwise eligible.

### Step 15: UT-007
- Write failing test: Added `PostCard` widget tests for opening a share menu with repost and quote choices, opening the same menu from the share count, and hiding the share action for reply posts.
- Run command: `cd app && flutter test test/feed/widgets/post_card_test.dart`
- Confirmed failure: Compilation failed because `PostCard` had no `onQuote` parameter and the repost control still mapped directly to one callback.
- Implement: Added optional `onQuote`, replaced the direct repost action with a share menu backed by `showCraftskyContextMenu`, and hid the share control when `post.reply` is present. Added `postQuoteAction` and `postShareAction` localization keys.
- Run command: `cd app && flutter test test/feed/widgets/post_card_test.dart`
- Refactor: Moved compact count formatting to shared private helpers so both normal actions and the new share action use identical labels.
- Notes: The repost callback now fires from the share menu item; direct icon/count taps open the choice menu.

### Step 16: UT-008
- Write failing test: Added a `PostCard` widget test requiring the share count to render `repostCount + quoteCount` while preserving separate model fields.
- Run command: `cd app && flutter test test/feed/widgets/post_card_test.dart`
- Confirmed failure: The share control rendered only `repostCount`, so `repostCount: 2, quoteCount: 3` did not display `5`.
- Implement: Changed the share action count to `post.repostCount + post.quoteCount`; selection color and menu label still use `viewerHasReposted`.
- Run command: `cd app && flutter test test/feed/widgets/post_card_test.dart`
- Refactor: None.
- Notes: The API/model still exposes separate counts; only the visible share-control count is combined.

### Step 17: UT-009
- Write failing test: Added a toggle-repost provider test proving an optimistic repost does not insert the target into a live timeline cache when the target is not already present.
- Run command: `cd app && flutter test test/feed/providers/toggle_post_interactions_provider_test.dart`
- Confirmed failure: The focused tests passed on first run because `updateLiveTimelineCache` replaces matching posts only and does not prepend repost activity.
- Implement: No production code change required.
- Run command: Same focused command.
- Refactor: None.
- Notes: This preserves `FR-019`/`NFR-005`: optimistic straight reposts patch existing post cards but do not synthesize feed items.

### Step 18: UT-010
- Write failing test: Added provider coverage that quote creation passes a quote ref through the normal create path and prepends the resulting post into timeline/profile caches; added API-client coverage for the `embed.quote` request body.
- Run command: `cd app && flutter test test/feed/providers/create_post_provider_test.dart test/feed/data/post_api_client_test.dart`
- Confirmed failure: Compilation failed because `CreatePost.create`, `PostApiClient.createPost`, and the fake repository had no `quote` parameter/capture field.
- Implement: Threaded `PostRef? quote` through `CreatePost`, `PostRepository`, `ApiPostRepository`, `PostApiClient`, and `FakePostRepository`; API writes `embed: {quote: {uri, cid}}`; successful synthetic quote responses are patched with the quote ref when omitted.
- Run command: `cd app && flutter test test/feed/providers/create_post_provider_test.dart test/feed/data/post_api_client_test.dart`
- Refactor: Kept quote creation on the existing top-level post cache path and added assertions preventing quote+reply and quote+project combinations.
- Notes: Quote posts now use normal post creation and cache insertion; there is no quote-specific optimistic timeline branch.

### Step 19: AT-006 through AT-010
- Write failing test: Added widget/page/provider coverage for quote preview rendering, hidden/unavailable placeholders, FeedPage quote-menu composer launch, composer quote submission, and existing feed repost menu routing.
- Run command: `cd app && flutter test test/feed/widgets/post_card_test.dart test/feed/widgets/post_composer_sheet_facets_test.dart test/feed/feed_page_test.dart test/feed/providers/create_post_provider_test.dart test/feed/providers/toggle_post_interactions_provider_test.dart`
- Confirmed failure: Quote previews were not rendered; `PostComposerSheet` had no `quoteTarget`; FeedPage did not pass `onQuote`, so the share menu only exposed straight repost.
- Implement: Added compact quote preview rendering to `PostCard`; added quote placeholder localization; added `quoteTarget` support to `showPostComposerSheet`/`PostComposerSheet`; submitted quote refs through `CreatePost`; wired FeedPage `onQuote` to open the quote composer; updated the Flutter timeline client to unwrap AppView feed-item `{post, reason}` wrappers into existing post pages.
- Run command: Same focused command.
- Refactor: Reused the existing composer and create-post provider path; quote and reply targets are asserted mutually exclusive.
- Notes: This covers the core AT-006 through AT-010 paths without adding a quote-specific timeline insertion branch.

### Step 20: REG-001 through REG-007
- Write failing test: Added backend regression coverage for authored quote posts in profile-style post lists, project-plus-quote rejection before PDS writes, and Dart repository project-plus-quote rejection. Ran existing search/profile/notification/repost regression slices.
- Run command: `cd appview && go test -run 'TestPostStore_ListByAuthor_IncludesAuthoredQuotePostsAndExcludesReposts|TestCreatePost_ProjectQuoteRejectedBeforePDSWrite|TestUnrepostPost_AbsentActiveRepostIsIdempotent' ./internal/api`; `cd appview && go test -run 'TestSearchStore|TestSearchSuggestions|TestNotifications|TestNotification|TestListNotifications' ./internal/api`; `cd app && flutter test test/feed/data/post_repository_test.dart test/search/providers/post_search_provider_test.dart test/profile/widgets/profile_posts_tab_test.dart`
- Confirmed failure: The Flutter profile regression initially failed because the repost action now opens a share menu and the test still expected direct repost on icon tap.
- Implement: Updated profile post rows to pass `onQuote` into `PostCard` and updated the profile repost test to select the `Repost` menu item.
- Run command: Re-ran the same focused Flutter regression command; all focused regression commands pass.
- Refactor: None.
- Notes: Search and notification tests passed unchanged; profile post lists keep authored-post semantics while straight reposts remain interaction records.

### Review Fix 1: IR-001 / AT-003 / IT-005
- Write failing test: Updated Flutter API, provider, and feed-page tests so home timeline items preserve `itemKey`, `post`, and repost `reason`; added duplicate same-post repost fixtures with distinct attribution.
- Run command: `cd app && flutter test test/feed/data/post_api_client_test.dart test/feed/providers/timeline_provider_test.dart test/feed/feed_page_test.dart`
- Confirmed failure: Tests failed because `TimelinePage`/`TimelineItem` models did not exist, `listTimeline` still returned `PostPage`, and `TimelineState.items` still held bare `Post` rows deduped by `post.uri`.
- Implement: Added typed Flutter `TimelinePage`, `TimelineItem`, and `RepostReason` models; changed repository/API/provider timeline contracts to feed items; changed timeline dedupe/append to use `itemKey`; changed cache updates to patch every item with the same `post.uri`; rendered "Reposted by {name}" attribution in `FeedPage`.
- Run command: `cd app && flutter test test/feed/data/post_api_client_test.dart test/feed/providers/timeline_provider_test.dart test/feed/feed_page_test.dart`
- Refactor: Kept profile/search/post-detail surfaces on `PostPage`; updated only home-timeline fakes and tests to `TimelinePage`.
- Notes: This fixes duplicate followed repost activities for the same original post and preserves repost attribution for rendering.

### Review Fix 2: IR-002 / AT-004 / IT-003
- Write failing test: Added `TestTimelineHandler_AttachesQuoteViewsToTimelinePosts` requiring timeline quote posts to include visible quote previews and hidden placeholders, with quote refs batched across the page.
- Run command: `cd appview && go test -run 'TestTimelineHandler_AttachesQuoteViewsToTimelinePosts' ./internal/api`
- Confirmed failure: The test failed because the timeline handler never called `QuoteViewRows`, so `quoteView` was absent on timeline quote posts.
- Implement: Extended the timeline reader boundary with `QuoteViewRows`, refactored post-detail quote hydration into a batched `attachQuoteViews` helper, and attached quote views before encoding timeline feed items.
- Run command: `cd appview && go test -run 'TestTimelineHandler_AttachesQuoteViewsToTimelinePosts|TestGetPost_WithQuote_AttachesCompactQuoteView|TestTimelineHandler' ./internal/api`
- Refactor: Shared one hydration path for post-detail and timeline responses.
- Notes: Quote previews remain one-level compact views; missing quote refs become `state: "unavailable"`.

### Review Fix 3: IR-003 / AT-002 / IT-007
- Write failing test: Added `TestCreatePost_QuoteEmbed_UsesResolvedTargetCID` with a valid quote target URI and stale caller-supplied CID.
- Run command: `cd appview && go test -run 'TestCreatePost_QuoteEmbed_UsesResolvedTargetCID' ./internal/api`
- Confirmed failure: The PDS record used the stale request CID instead of the indexed target CID.
- Implement: Changed quote share-target validation to return the resolved `ShareTargetRef` and canonicalized `req.Embed.Quote` to the resolved `{uri,cid}` before building the PDS record and synthetic response.
- Run command: `cd appview && go test -run 'TestCreatePost_QuoteEmbed_UsesResolvedTargetCID|TestCreatePost_QuoteEmbed_TranslatedToLexiconShape|TestCreatePost_QuoteRejectsReplyTargetBeforePDSWrite|TestCreatePost_AllowsSelfQuote|TestCreatePost_AllowsMultipleQuotePostsForSameSubject' ./internal/api`
- Refactor: Kept straight repost write validation unchanged; this fix only affects quote-post create strongRefs.
- Notes: AppView canonicalizes stale quote CIDs rather than rejecting otherwise valid requests.

### Review Fix 4: IR-001 / FR-008 / FR-009
- Write failing test: Add handler coverage proving post-shaped profile and comment/thread responses attach compact `quoteView` previews while staying post-shaped.
- Run command: `cd appview && go test -run 'TestListPosts_AttachesQuoteViewsToAuthoredQuotePosts|TestGetPostComments_AttachesQuoteViewsToPostShapedResponses' ./internal/api`
- Confirmed failure: Both tests failed with empty `QuoteViewRows` refs, proving profile authored-post and comment-section response builders returned quote posts without invoking quote-preview hydration.
- Implement: Reused the existing batched `attachQuoteViews` helper for author post/project/comment lists, comment sections, branch reply lists, and search post/project/hashtag result builders; added `SearchStore.QuoteViewRows` delegation to the underlying `PostStore`.
- Run command: `cd appview && go test -run 'TestListPosts_AttachesQuoteViewsToAuthoredQuotePosts|TestGetPostComments_AttachesQuoteViewsToPostShapedResponses' ./internal/api`; `cd appview && go test ./internal/api`; `cd appview && go test -timeout 120s ./...`
- Refactor: Added small collectors for post-shaped comment-section and reply-page responses so hydration stays batched and the response shapes remain unchanged.
- Notes: Profile, search, thread/comment, post-detail, and timeline quote posts now share the same compact one-level quote-view hydration path; home timeline remains the only feed-item wrapper surface.

### Review Fix 5: IR-001 / FR-008 / FR-009
- Write failing test: Add `TestCreatePost_QuoteEmbed_AttachesCompactQuoteView` proving a successful quote create returns `quoteView.state == "visible"` with quoted author attribution.
- Run command: `cd appview && go test -run 'TestCreatePost_QuoteEmbed_AttachesCompactQuoteView' ./internal/api`
- Confirmed failure: The focused test failed with empty `QuoteViewRows` refs, proving the quote-create synthetic `PostResponse` encoded the raw strongRef without compact preview hydration.
- Implement: Called `attachQuoteView` on the synthetic create response before encoding `201 Created`, reusing the same compact one-level quote-view hydration path as post detail and list responses.
- Run command: `cd appview && go test -run 'TestCreatePost_QuoteEmbed_AttachesCompactQuoteView|TestCreatePost_QuoteEmbed_UsesResolvedTargetCID|TestListPosts_AttachesQuoteViewsToAuthoredQuotePosts|TestGetPostComments_AttachesQuoteViewsToPostShapedResponses' ./internal/api`; `cd appview && go test -timeout 120s ./...`; `cd app && flutter test test/feed/data/post_api_client_test.dart test/feed/providers/create_post_provider_test.dart`
- Refactor: None.
- Notes: Quote create stays on the normal post-create cache path; the returned post is now hydrated for immediate rendering and still preserves the resolved canonical quote `{uri,cid}`.

## Verification Log
- `cd appview && go test -timeout 120s ./...` — pass.
- `cd app && flutter test test/feed/models/post_test.dart test/feed/widgets/post_card_test.dart test/feed/widgets/post_composer_sheet_facets_test.dart test/feed/feed_page_test.dart test/feed/providers/create_post_provider_test.dart test/feed/providers/toggle_post_interactions_provider_test.dart test/feed/data/post_api_client_test.dart test/feed/data/post_repository_test.dart test/search/providers/post_search_provider_test.dart test/profile/widgets/profile_posts_tab_test.dart` — pass.
- `cd app && dart analyze` — pass.
- Note: an earlier `cd appview && go test -timeout 60s ./...` exposed stale fake-store fixtures in missing-subject handler tests; those were corrected and the suite passed with `-timeout 120s`.
- Review-fix verification:
  - `cd appview && go test -timeout 120s ./...` — pass.
  - `cd app && flutter test test/feed/models/post_test.dart test/feed/data/post_api_client_test.dart test/feed/data/post_repository_test.dart test/feed/providers/timeline_provider_test.dart test/feed/providers/create_post_provider_test.dart test/feed/providers/toggle_post_interactions_provider_test.dart test/feed/widgets/post_card_test.dart test/feed/widgets/post_composer_sheet_facets_test.dart test/feed/feed_page_test.dart test/feed/pages/feed_page_composer_entry_test.dart test/search/providers/post_search_provider_test.dart test/profile/widgets/profile_posts_tab_test.dart test/router/router_redirect_test.dart` — pass.
  - `cd app && dart analyze` — pass.
  - Note: an attempted Flutter verification command included non-existent `test/feed/models/timeline_state_test.dart`; rerunning without that path passed.
- Review-fix 4 verification:
  - `cd appview && go test -run 'TestListPosts_AttachesQuoteViewsToAuthoredQuotePosts|TestGetPostComments_AttachesQuoteViewsToPostShapedResponses' ./internal/api` — pass.
  - `cd appview && go test ./internal/api` — pass.
  - `cd appview && go test -timeout 120s ./...` — pass.
- Review-fix 5 verification:
  - `cd appview && go test -run 'TestCreatePost_QuoteEmbed_AttachesCompactQuoteView|TestCreatePost_QuoteEmbed_UsesResolvedTargetCID|TestListPosts_AttachesQuoteViewsToAuthoredQuotePosts|TestGetPostComments_AttachesQuoteViewsToPostShapedResponses' ./internal/api` — pass.
  - `cd appview && go test -timeout 120s ./...` — pass.
  - `cd app && flutter test test/feed/data/post_api_client_test.dart test/feed/providers/create_post_provider_test.dart` — pass.

## Completion Checklist
- [x] All Must requirements covered by tests or documented gaps
- [x] All planned Must tests passing
- [x] Relevant regression tests passing
- [x] No unlinked behavior implemented
- [x] Docs updated
- [ ] Review completed or explicitly skipped
