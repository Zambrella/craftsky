# TDD Implementation Plan: Reply Comment Section

## Inputs
- Requirements: `02-requirements.md`
- Tests: `03-acceptance-tests.md`
- Discovery: `01-discovery-notes.md`
- Document review: not present in this folder at implementation start.

## Implementation Rules
- Do not implement behavior without a linked requirement ID.
- Write or update a failing test before implementation.
- Run the smallest relevant test first.
- Refactor only after tests pass.
- Keep traceability updated.
- Preserve comment/reply terminology in API/client/test names except backend storage fields that intentionally model atproto `reply_*` refs.
- Do not change lexicon files for this feature.

## Concrete Implementation Choices
- Backend route: `GET /v1/posts/{did}/{rkey}/comments` is the root comment-section read surface.
- Existing `GET /v1/posts/{did}/{rkey}/replies` remains the actioned per-comment reply loader.
- `GET /v1/posts/{did}/{rkey}/thread` is removed.
- Comment page size defaults to 10 and caps at 10 for comment-section and reply-loading behavior.
- Sort values are `oldest`, `newest`, and `follows`; `follows` uses oldest-first semantics until follow data exists.

## Test Order
Mirrors the approved suggested order from `03-acceptance-tests.md` §11.

| Step | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| 1 | IT-001 | FR-002, RULE-002 | AC-004, AC-008 | Fails until comment-section endpoint exists. |
| 2 | IT-014 | FR-022 | AC-022, AC-024 | Fails until comment placement is required. |
| 3 | IT-015 | FR-023 | AC-023 | Fails until replies object is always present. |
| 4 | IT-004 | FR-006, NFR-002 | AC-005, AC-009 | Fails until comment pagination is bounded and cursor-based. |
| 5 | IT-005 | FR-007, FR-008, RULE-005, RULE-006 | AC-007, AC-020 | Fails until sort and viewer grouping are implemented. |
| 6 | IT-002 | BR-001, FR-003 | AC-001 | Fails until focus query is accepted. |
| 7 | IT-003 | BR-001, FR-004 | AC-002, AC-003 | Fails until focused branch outside page one is included. |
| 8 | IT-006 | FR-004, RULE-007 | AC-003, AC-021 | Fails until focused reply branch expands with bounded slice. |
| 9 | IT-011 | NFR-002, RULE-007 | AC-021 | Fails until focused reply slice is bounded. |
| 10 | IT-013 | FR-019, FR-020, FR-021 | AC-025 | Fails until full focus status contract exists. |
| 11 | IT-016 | FR-024 | AC-026 | Fails until flattened reply metadata is returned. |
| 12 | IT-007 | FR-010, FR-011, RULE-004 | AC-006, AC-009, AC-010 | Existing behavior may need cap/default updates. |
| 13 | IT-008 | FR-014, RULE-001 | AC-015, AC-019 | Existing create-reply behavior should be preserved/pinned. |
| 14 | IT-009 | FR-001 | AC-013 | Fails until `/thread` route is removed. |
| 15 | IT-010 | NFR-001 | AC-017 | Fails until comment endpoints match API conventions. |
| 16 | UT-007 | FR-002, FR-022, FR-023 | AC-008, AC-022, AC-023 | Fails until Flutter model decodes comment-section response. |
| 17 | UT-006 | FR-003 | AC-001 | Fails until route passes decoded focus. |
| 18 | UT-001 | FR-007, FR-008, RULE-005, RULE-006 | AC-007, AC-020 | Fails until client state sorts/groups comments. |
| 19 | UT-002 | FR-007, FR-010, FR-011, RULE-004 | AC-010 | Fails until replies are sorted oldest-first in state. |
| 20 | UT-003 | FR-013, FR-014, RULE-003 | AC-014 | Fails until deeper replies flatten to comment branch. |
| 21 | UT-004 | FR-005, RULE-002 | AC-004 | Fails until initial state excludes reply lists. |
| 22 | UT-005 | FR-009, FR-012 | AC-006 | Fails until expansion/collapse state exists. |
| 23 | UT-008 | FR-006, NFR-002 | AC-005, AC-009 | Fails until lazy-load state guards duplicate loads. |
| 24 | UT-009 | FR-011, NFR-002 | AC-009 | Fails until reply cursors are per comment branch. |
| 25 | UT-010 | FR-017 | AC-012 | Fails until new nested replies insert into nearest branch. |
| 26 | UT-011 | FR-016, NFR-003, RULE-006 | AC-011, AC-018, AC-020 | Fails until viewer-authored duplicates are removed. |
| 27 | UT-012 | FR-015 | AC-015 | Fails until composer prefill mentions target handle. |
| 28 | UT-013 | FR-018 | AC-016 | Fails until l10n strings exist. |
| 29 | UT-014 | RULE-008, FR-007 | AC-024 | Fails until sort change clears focus promotion. |
| 30 | UT-015 | FR-022 | AC-022 | Fails until placement is enum-backed/required. |
| 31 | UT-016 | FR-023 | AC-023 | Fails until replies object is required in client model. |
| 32 | UT-017 | FR-024 | AC-026 | Fails until flattened reply metadata decodes. |
| 33 | AT-001..AT-011 | BR-001..FR-024 | AC-001..AC-026 | Fails until Flutter comment-section page workflows exist. |
| 34 | REG-001..REG-005 | FR-001, NFR-001, RULE-001 | AC-013, AC-017, AC-019 | Fails until stale thread usage is removed and regressions pass. |

## Implementation Steps

### Step 1: IT-001
- Write failing test: add AppView handler/store coverage for `GET /v1/posts/{did}/{rkey}/comments` returning root post and comments only.
- Run command: `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api`.
- Confirmed failure: `go test ./internal/api` failed to compile because `GetPostCommentsHandler`, `CommentSectionResponse`, and fake comment-list fields did not exist.
- Implement: added `CommentSectionResponse`/comment item/reply page wire structs, `GetPostCommentsHandler`, `PostReader.ListRootComments`, and the initial Postgres `ListRootComments` query for direct replies to the root.
- Run command: `go test ./internal/api -run TestGetPostComments_ReturnsRootAndCommentsOnly` passed; `go test ./internal/api` passed.
- Refactor: none.
- Notes: Concrete route is `GET /v1/posts/{did}/{rkey}/comments`; response includes root post, `comments.items`, selected sort default `oldest`, normal placement, and collapsed `replies` objects by default. Sort/viewer grouping remain for later tests.

### Step 2: IT-014
- Write failing test: add handler coverage that each returned comment item carries required placement metadata.
- Run command: `go test ./internal/api -run TestGetPostComments_CommentItemsIncludePlacement`.
- Confirmed failure: this test was already green because Step 1 returned `placement = "normal"` to satisfy the initial response metadata contract.
- Implement: no code change required beyond Step 1.
- Run command: `go test ./internal/api -run TestGetPostComments_CommentItemsIncludePlacement` passed.
- Refactor: none.
- Notes: Focused and viewer-authored placement variants remain for later focus/sort tests.

### Step 3: IT-015
- Write failing test: add handler coverage that every comment item includes `replies.loaded`, `replies.items`, and omits `replies.cursor` when replies are not loaded.
- Run command: `go test ./internal/api -run TestGetPostComments_CommentItemsAlwaysIncludeRepliesObject`.
- Confirmed failure: this test was already green because Step 1 returned a collapsed reply state for every comment item.
- Implement: no code change required beyond Step 1.
- Run command: `go test ./internal/api -run TestGetPostComments_CommentItemsAlwaysIncludeRepliesObject` passed.
- Refactor: none.
- Notes: Loaded reply pages and reply cursors remain for later child-reply tests.

### Step 4: IT-004
- Write failing test: add PostStore coverage for `ListRootComments` paging direct root comments 10 at a time with an opaque cursor and excluding nested replies.
- Run command: `go test ./internal/api -run TestPostStore_ListRootComments_PaginatesOpaqueCursorOldestFirst`.
- Confirmed failure: this test was already green because Step 1 introduced an oldest-first direct-comment query with cursor pagination.
- Implement: no code change required beyond Step 1.
- Run command: `go test ./internal/api -run TestPostStore_ListRootComments_PaginatesOpaqueCursorOldestFirst` passed.
- Refactor: none.
- Notes: Handler-level limit capping to 10 was introduced in Step 1; broader pagination and sort interactions continue in later tests.

### Step 5: IT-005
- Write failing test: add PostStore coverage for viewer-authored comments sorted before normal comments with `oldest`, `newest`, and `follows` (`follows` = oldest) ordering within each group.
- Run command: `go test ./internal/api -run TestPostStore_ListRootComments_GroupsViewerAndSortsWithinGroups -count=1 -v`.
- Confirmed failure: initial local run skipped because `TEST_DATABASE_URL`/`DATABASE_URL` were unset; explicit `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable` failed before `just dev-d` because the dev Postgres was not running (`connection refused`). After starting the compose stack, the behavior test passed with the implementation below.
- Implement: updated `ListRootComments` ordering to place viewer-authored rows first and apply `oldest`/`follows` ascending or `newest` descending order within each group.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api -run TestPostStore_ListRootComments_GroupsViewerAndSortsWithinGroups -count=1 -v` passed; `go test ./internal/api` passed locally.
- Refactor: none.
- Notes: Cursor semantics for grouped pagination will be revisited in later pagination/focus de-duplication tests if needed.

### Step 6: IT-002
- Write failing test: add handler coverage for `focus=<url-encoded AT-URI>` identifying an indexed comment under the route root.
- Run command: `go test ./internal/api -run TestGetPostComments_FocusQueryIdentifiesIncludedComment`.
- Confirmed failure: compile failed because `CommentSectionResponse.Focus` and focus-resolution plumbing did not exist.
- Implement: added `FocusContext`, `PostReader.ReadPostByURI`, Postgres `ReadPostByURI`, focus AT-URI parsing, included comment detection, and focused placement for loaded focused comments.
- Run command: `go test ./internal/api -run TestGetPostComments_FocusQueryIdentifiesIncludedComment` passed; `go test ./internal/api` passed.
- Refactor: none.
- Notes: Focused branch inclusion outside page one and reply focus expansion remain for later tests.

### Step 7: IT-003
- Write failing test: add handler coverage for a focused comment that is not in the current comment page.
- Run command: `go test ./internal/api -run TestGetPostComments_FocusedCommentOutsidePageIsIncludedFirst`.
- Confirmed failure: response contained only the normal page item; the focused comment was not included or promoted.
- Implement: when focus resolves to a comment under the root and it is absent from the normal page, hydrate it, prepend it to `comments.items`, and mark `placement = "focused"` without duplicating normal page items.
- Run command: `go test ./internal/api -run 'TestGetPostComments_FocusedCommentOutsidePageIsIncludedFirst|TestGetPostComments_FocusQueryIdentifiesIncludedComment'` passed; `go test ./internal/api` passed.
- Refactor: none.
- Notes: Focused reply branch expansion and bounded reply slices remain for later tests.

### Step 8: IT-006
- Write failing test: add handler coverage for focus on a reply whose comment branch is outside the current comment page.
- Run command: `go test ./internal/api -run TestGetPostComments_FocusedReplyExpandsCommentBranch`.
- Confirmed failure: focus was reported as `included`/`reply` but `commentUri` was empty and the owning comment branch/reply item were not included.
- Implement: when focus resolves to a reply under the root, read its direct comment parent, set `focus.commentUri`, promote/include the comment branch, and return a loaded reply slice containing the focused reply.
- Run command: `go test ./internal/api -run 'TestGetPostComments_FocusedReplyExpandsCommentBranch|TestGetPostComments_FocusedCommentOutsidePageIsIncludedFirst'` passed; `go test ./internal/api` passed.
- Refactor: none.
- Notes: Current slice contains only the focused reply; explicit bounded-size assertions continue in IT-011.

### Step 9: IT-011
- Write failing test: add handler coverage that a focused reply branch returns a bounded reply slice containing only the focused item rather than loading arbitrary earlier replies.
- Run command: `go test ./internal/api -run TestGetPostComments_FocusedReplySliceIsBounded`.
- Confirmed failure: test was already green because IT-006 implemented the focused reply slice as a single focused reply item.
- Implement: no code change required beyond Step 8.
- Run command: `gofmt -w internal/api/post_test.go && go test ./internal/api -run TestGetPostComments_FocusedReplySliceIsBounded` passed.
- Refactor: none.
- Notes: Future UX can expand the bounded slice to include neighboring replies while remaining capped; current implementation is the smallest bounded slice satisfying the requirement.

### Step 10: IT-013
- Write failing test: add handler coverage for malformed focus, not-found focus, and mismatched-root focus status behavior.
- Run command: `go test ./internal/api -run TestGetPostComments_FocusStatusContract`.
- Confirmed failure: test was already green because IT-002 implemented malformed/not-found and IT-006 implemented mismatched-root handling.
- Implement: no code change required beyond earlier focus steps.
- Run command: `gofmt -w internal/api/post_test.go && go test ./internal/api -run TestGetPostComments_FocusStatusContract` passed.
- Refactor: none.
- Notes: Included comment and reply focus cases are covered by IT-002 and IT-006 tests.

### Step 11: IT-016
- Write failing test: add handler coverage for a deeper focused reply (`root -> comment -> reply -> deeper`) returning `flattened = true` and `replyingTo` metadata for the true backend parent.
- Run command: `go test ./internal/api -run TestGetPostComments_DeeperFocusedReplyIncludesFlattenedMetadata`.
- Confirmed failure: focus stayed `notFound` because only direct reply parents were resolved to comment branches.
- Implement: added one-level ancestor resolution from a focused deeper reply to its nearest comment branch, hydrated the true parent row, and added `buildFocusedReplyItem` to emit `flattened` plus `replyingTo { uri, did, handle, displayName? }` when the visual reply's true parent is not the comment branch.
- Run command: `go test ./internal/api -run 'TestGetPostComments_DeeperFocusedReplyIncludesFlattenedMetadata|TestGetPostComments_FocusedReplyExpandsCommentBranch'` passed; `go test ./internal/api` passed.
- Refactor: extracted focused reply item construction.
- Notes: Current ancestor resolution covers the tested deeper backend chain in the accepted test design. More arbitrary-depth ancestor walking can be added if future requirements require it.

### Step 12: IT-007
- Write failing test: add handler coverage that direct reply expansion caps page size at 10.
- Run command: `go test ./internal/api -run TestListDirectReplies_CapsPageSizeAtTen`.
- Confirmed failure: handler passed `limit=20` through to the store instead of capping at 10.
- Implement: changed direct replies handler to use the comment/reply page limit cap of 10; updated stale tests that expected the generic 50 default / 100 cap for this endpoint.
- Run command: `go test ./internal/api -run 'TestListDirectReplies_CapsPageSizeAtTen|TestListDirectReplies_HappyPath_PaginatesEngagementAndAuthorHandles'` passed; `go test ./internal/api` passed.
- Refactor: renamed stale cap test to `TestListDirectReplies_LimitCapsAt10`.
- Notes: Existing store pagination test already pins oldest-first opaque cursor behavior.

### Step 13: IT-008
- Write failing test: reuse existing `TestCreatePost_WithReply_PassesThroughToPDS` coverage for reply root/parent strong refs.
- Run command: `go test ./internal/api -run TestCreatePost_WithReply_PassesThroughToPDS`.
- Confirmed failure: test was already green; existing create-post behavior preserves the request's actual root and parent refs.
- Implement: no code change required.
- Run command: `go test ./internal/api -run TestCreatePost_WithReply_PassesThroughToPDS` passed.
- Refactor: none.
- Notes: No lexicon files were changed.

### Step 14: IT-009
- Write failing test: update route coverage so `GET /v1/posts/{did}/{rkey}/thread` returns 404 instead of resolving to the old thread handler.
- Run command: `go test ./internal/routes -run TestAddRoutes_PostThreadRouteRemoved`.
- Confirmed failure: route still resolved to `GetPostThreadHandler` and panicked in the route test because the nil test DB was reached.
- Implement: removed `/thread` from route registration and registered `GET /v1/posts/{did}/{rkey}/comments` for the comment-section handler.
- Run command: `go test ./internal/routes -run TestAddRoutes_PostThreadRouteRemoved` passed; `go test ./internal/routes ./internal/api` passed.
- Refactor: replaced stale thread auth/device route tests with the route-removal test.
- Notes: Thread handler/types still exist for now because backend tests call them directly; client and broader stale-thread removal remain in later Flutter/regression loops.

### Step 15: IT-010
- Write failing test: add handler coverage that invalid comment-section cursors use the standard error envelope.
- Run command: `go test ./internal/api -run TestGetPostComments_InvalidCursorUsesStandardEnvelope`.
- Confirmed failure: initial assertion incorrectly required a non-empty request ID in a direct handler unit test; `envelope.WriteError` explicitly allows empty request IDs in direct tests/pre-logging paths. The meaningful contract fields (`error`, `message`, camelCase `requestId` in the struct tag) were already present.
- Implement: no production code change required; corrected the test expectation to assert the standard error code/message envelope while allowing empty direct-test request IDs.
- Run command: `go test ./internal/api -run TestGetPostComments_InvalidCursorUsesStandardEnvelope` passed; `go test ./internal/api ./internal/routes` passed.
- Refactor: none.
- Notes: Success responses use camelCase struct tags (`comments`, `items`, `placement`, `loaded`, `commentUri`, `replyingTo`) and cursors are produced by the existing opaque envelope cursor helper.

### Step 16: UT-007
- Write failing test: add Flutter model test for decoding the comment-section response shape with `post`, `comments.items`, required `placement`, `replies.loaded`, reply cursor/items, focus metadata, and flattened `replyingTo` metadata.
- Run command: `flutter test test/feed/models/post_comment_section_test.dart`.
- Confirmed failure: compilation failed because `post_comment_section.dart`, mapper, and enums did not exist.
- Implement: added `PostCommentSection` Flutter model and nested models/enums, registered the mapper in `initializeMappers`, and regenerated Dart mapper outputs with `dart run build_runner build --delete-conflicting-outputs`.
- Run command: `flutter test test/feed/models/post_comment_section_test.dart` passed.
- Refactor: none.
- Notes: Generated mapper files changed broadly because build_runner refreshed existing outputs.

### Step 17: UT-006
- Write failing test: add router widget coverage that `/posts/{did}/{rkey}?focus=<encoded AT-URI>` decodes and passes the focus value into the post page.
- Run command: `flutter test test/router/router_redirect_test.dart --plain-name 'post route decodes focus query parameter'`.
- Confirmed failure: compilation failed because the page/route did not expose a `focus` field.
- Implement: added optional `focus` to `PostThreadRoute` and `PostThreadPage`, passed it through route build, and regenerated go_router output.
- Run command: `flutter test test/router/router_redirect_test.dart --plain-name 'post route decodes focus query parameter'` passed.
- Refactor: none.
- Notes: Page is still named `PostThreadPage`; full UI replacement/stale thread cleanup remains for later widget/regression loops.

### Step 18: UT-001
- Write failing test: add Flutter state/model coverage that comment sorting groups viewer-authored comments first and treats `follows` as oldest-first.
- Run command: `flutter test test/feed/models/post_comment_section_state_test.dart`.
- Confirmed failure: compilation failed because `sortCommentItemsForViewer` did not exist.
- Implement: added `sortCommentItemsForViewer` helper to `post_comment_section.dart`, grouping by viewer DID and applying oldest/follows ascending or newest descending within viewer and normal groups.
- Run command: `flutter test test/feed/models/post_comment_section_state_test.dart` passed.
- Refactor: none.
- Notes: Focus promotion is handled separately in UT-014.

### Step 19: UT-002
- Write failing test: add Flutter state/model coverage that reply ordering remains oldest-first for every comment sort.
- Run command: `flutter test test/feed/models/post_comment_section_state_test.dart --plain-name 'orders replies oldest-first regardless of comment sort'`.
- Confirmed failure: compilation failed because `sortReplyItems` did not exist.
- Implement: added `sortReplyItems` helper that sorts by created time and URI ascending while ignoring the selected comment sort.
- Run command: `flutter test test/feed/models/post_comment_section_state_test.dart` passed.
- Refactor: none.
- Notes: This is a pure state helper until provider/page state integration lands.

### Step 20: UT-003
- Write failing test: add Flutter state/model coverage that a deeper reply is flattened into the nearest comment branch with no third visual branch.
- Run command: `flutter test test/feed/models/post_comment_section_state_test.dart --plain-name 'flattens deeper replies to nearest comment branch'`.
- Confirmed failure: compilation failed because `ReplyTreeEdge` and `flattenRepliesToCommentBranches` did not exist.
- Implement: added `ReplyTreeEdge` and `flattenRepliesToCommentBranches`, which walks parent URI links to find the owning comment branch and marks deeper replies as flattened.
- Run command: `flutter test test/feed/models/post_comment_section_state_test.dart` passed.
- Refactor: none.
- Notes: Structural `replyingTo` metadata is decoded separately by UT-017 and already covered in backend IT-016.

### Step 21: UT-004
- Write failing test: add Flutter state/model coverage that initial comment-section state keeps per-comment reply lists collapsed.
- Run command: `flutter test test/feed/models/post_comment_section_state_test.dart --plain-name 'initial comment section state keeps reply lists collapsed'`.
- Confirmed failure: compilation failed because `initialCommentSectionState` did not exist.
- Implement: added `initialCommentSectionState`, which preserves root/comment metadata while resetting each comment's replies to `loaded = false` and empty items.
- Run command: `flutter test test/feed/models/post_comment_section_state_test.dart` passed.
- Refactor: none.
- Notes: Focus-loaded reply branches may bypass this helper; this helper models the unfocused initial state.

### Step 22: UT-005
- Write failing test: add Flutter state/model coverage for collapsed/expanded branch controls and collapse behavior.
- Run command: `flutter test test/feed/models/post_comment_section_state_test.dart --plain-name 'branch expansion and collapse state changes controls'`.
- Confirmed failure: compilation failed because `BranchControl`, `branchControlFor`, `setCommentReplies`, and `collapseCommentReplies` did not exist.
- Implement: added branch control enum and state helpers for setting loaded replies and collapsing a comment branch.
- Run command: `flutter test test/feed/models/post_comment_section_state_test.dart` passed.
- Refactor: none.
- Notes: UI label localization and actual widget controls remain in later widget/l10n loops.

### Step 23: UT-008
- Write failing test: add Flutter provider coverage that top-level comment `loadMoreComments` uses the current comment cursor, blocks concurrent duplicate loads, appends the next page, and updates the cursor.
- Run command: `flutter test test/feed/providers/post_comment_section_provider_test.dart`.
- Confirmed failure: compilation failed because `post_comment_section_provider.dart`, repository `commentSection` plumbing, and fake repository `onCommentSection` did not exist.
- Implement: added `PostCommentSection` Riverpod async notifier for initial comment-section fetch and guarded `loadMoreComments`, added `PostRepository.commentSection`, `ApiPostRepository`/`PostApiClient.getCommentSection`, fake repository support, and generated provider code.
- Run command: `flutter test test/feed/providers/post_comment_section_provider_test.dart` passed; nearby `flutter test test/feed/providers/post_comment_section_provider_test.dart test/feed/models/post_comment_section_state_test.dart test/feed/models/post_comment_section_test.dart` passed.
- Refactor: kept provider state merging minimal and cursor-specific; no unrelated UI integration yet.
- Notes: The provider currently uses the loaded section sort/focus context and appends only top-level comments; per-branch reply load-more remains for UT-009.

### Step 24: UT-009
- Write failing test: add Flutter provider coverage that loading more replies for one expanded comment branch calls the direct-replies endpoint with that branch cursor, appends only that branch's replies, and leaves another branch's cursor/items unchanged.
- Run command: `flutter test test/feed/providers/post_comment_section_provider_test.dart --plain-name 'reply load more keeps branch cursors and items independent'`.
- Confirmed failure: compilation failed because `PostCommentSection` notifier did not expose `loadMoreReplies`.
- Implement: added `loadMoreReplies(commentUri)`, which finds the target comment branch, uses its reply cursor with `listDirectReplies`, converts returned posts into non-flattened `ReplyItem`s, and replaces only that comment branch in provider state.
- Run command: focused reply load-more test passed; nearby `flutter test test/feed/providers/post_comment_section_provider_test.dart test/feed/models/post_comment_section_state_test.dart test/feed/models/post_comment_section_test.dart` passed.
- Refactor: added a private branch replacement helper in the provider; no broader UI integration yet.
- Notes: Initial “view replies” loading without a cursor is still represented by existing `setCommentReplies` state helper/UI follow-up; UT-009 covers load-more cursor isolation.

### Step 25: UT-010
- Write failing test: add Flutter state/model coverage that a newly-created reply whose parent is an existing reply is inserted into the nearest comment branch rather than creating another visual level.
- Run command: `flutter test test/feed/models/post_comment_section_state_test.dart --plain-name 'inserts a new nested reply into the nearest comment branch'`.
- Confirmed failure: compilation failed because `insertCreatedReplyIntoNearestBranch` did not exist.
- Implement: added `insertCreatedReplyIntoNearestBranch`, which locates the branch by either direct comment parent URI or an already-loaded reply parent URI, appends the created reply to that branch, and marks it flattened when its backend parent is not the comment itself.
- Run command: focused nested-reply insertion test passed; nearby `flutter test test/feed/models/post_comment_section_state_test.dart test/feed/providers/post_comment_section_provider_test.dart` passed.
- Refactor: none.
- Notes: Scroll/focus side effects remain for page/widget integration, but the branch-targeting state update is pinned here.

### Step 26: UT-011
- Write failing test: add Flutter state/model coverage that a viewer-authored comment already surfaced in the current list is not duplicated when a later paginated page contains the same URI as a normal item.
- Run command: `flutter test test/feed/models/post_comment_section_state_test.dart --plain-name 'de-duplicates viewer-authored comments from later pages'`.
- Confirmed failure: compilation failed because `appendCommentPageDeduplicating` did not exist.
- Implement: added `appendCommentPageDeduplicating`, which appends a comment page while preserving the first item for each URI and advancing the comment cursor, and updated `loadMoreComments` to use it.
- Run command: focused de-duplication test passed; nearby `flutter test test/feed/models/post_comment_section_state_test.dart test/feed/providers/post_comment_section_provider_test.dart` passed.
- Refactor: reused the pure state helper in the provider to keep pagination merging behavior consistent.
- Notes: This covers duplicate viewer-authored comments encountered through later pages; focused promotion clearing is covered separately by UT-014.

### Step 27: UT-012
- Write failing test: add composer widget coverage that a reply-to-reply target pre-populates the text field with `@<target handle>`.
- Run command: `flutter test test/feed/widgets/post_composer_sheet_test.dart --plain-name 'replying to a reply prefills target author mention'`.
- Confirmed failure: text field controller was empty for a reply target that itself had reply refs.
- Implement: initialized the composer controller/text state with `@${replyTarget.author.handle} ` when the target is itself a reply, and placed the cursor after the mention.
- Run command: focused composer mention test passed; full `flutter test test/feed/widgets/post_composer_sheet_test.dart` passed.
- Refactor: none.
- Notes: Top-level comment replies remain unprefilled; the prefill is limited to replying to a reply, matching FR-015.

### Step 28: UT-013
- Write failing test: add Flutter localization coverage for comment-section sort labels, view/load/hide reply controls, and focus-state messages.
- Run command: `flutter test test/feed/pages/post_comment_section_page_test.dart --plain-name 'comment section labels are exposed through localizations'`.
- Confirmed failure: compilation failed because the `AppLocalizations` accessors for comment-section labels did not exist.
- Implement: added the English ARB entries and regenerated `AppLocalizations` files with `flutter gen-l10n`.
- Run command: focused localization test passed; nearby `flutter test test/feed/pages/post_comment_section_page_test.dart test/feed/widgets/post_composer_sheet_test.dart` passed.
- Refactor: none.
- Notes: Widget usage of these labels is still part of the later page workflow tests; this step pins the l10n contract.

### Step 29: UT-014
- Write failing test: add Flutter state/model coverage that changing comment sort clears focus context/promotion and reapplies viewer-authored grouping with the selected sort.
- Run command: `flutter test test/feed/models/post_comment_section_state_test.dart --plain-name 'sort change clears focus promotion and preserves viewer grouping'`.
- Confirmed failure: compilation failed because `changeCommentSortClearingFocus` did not exist.
- Implement: added `changeCommentSortClearingFocus`, which normalizes focused placement back to viewer-authored or normal, clears `focus`, updates the selected sort, and reuses the existing viewer grouping/sort helper.
- Run command: focused sort/focus-clearing test passed; nearby `flutter test test/feed/models/post_comment_section_state_test.dart test/feed/providers/post_comment_section_provider_test.dart` passed.
- Refactor: none.
- Notes: The helper intentionally resets the top-level cursor for a new sort/filter intent, matching EC-006/RULE-008.

### Step 30: UT-015
- Write failing test: add Flutter model decode coverage that missing `placement` and unknown placement strings fail, while a valid `viewerAuthored` value decodes to the enum.
- Run command: `flutter test test/feed/models/post_comment_section_test.dart --plain-name 'requires enum-backed comment placement'`.
- Confirmed failure: the test was already green because the Step 16 model used a required `CommentPlacement` enum generated by dart_mappable.
- Implement: no production code change required.
- Run command: focused placement decode test passed; full `flutter test test/feed/models/post_comment_section_test.dart` passed.
- Refactor: none.
- Notes: This pins the existing enum-backed/required behavior against future mapper changes.

### Step 31: UT-016
- Write failing test: add Flutter model decode coverage that missing `replies` fails and that unloaded, loaded-empty, and loaded-with-cursor variants decode distinctly.
- Run command: `flutter test test/feed/models/post_comment_section_test.dart --plain-name 'requires replies object and decodes loaded states'`.
- Confirmed failure: the test was already green because the Step 16 model made `ReplyPage replies` required and preserved `loaded`, `items`, and optional `cursor`.
- Implement: no production code change required.
- Run command: focused replies decode test passed; full `flutter test test/feed/models/post_comment_section_test.dart` passed.
- Refactor: none.
- Notes: This pins the loaded-state contract for future UI/provider changes.

### Step 32: UT-017
- Write failing test: add Flutter model decode coverage that direct replies decode without `replyingTo`, while flattened replies decode `replyingTo { uri, did, handle, displayName? }` metadata.
- Run command: `flutter test test/feed/models/post_comment_section_test.dart --plain-name 'decodes flattened reply metadata structurally'`.
- Confirmed failure: the test was already green because the Step 16 model included required `flattened` and optional `ReplyingToAuthor` metadata.
- Implement: no production code change required.
- Run command: focused flattened metadata test passed; broader `flutter test test/feed/models/post_comment_section_test.dart test/feed/models/post_comment_section_state_test.dart test/feed/providers/post_comment_section_provider_test.dart test/feed/pages/post_comment_section_page_test.dart test/feed/widgets/post_composer_sheet_test.dart` passed.
- Refactor: none.
- Notes: Backend structural truth for flattened replies was already pinned by IT-016; this pins client decode behavior.

### Step 33: AT-001
- Write failing test: add Flutter page coverage that a focused root-post route passes the focus URI to the comment-section provider and renders the root post, focused comment branch, and focused reply target.
- Run command: `flutter test test/feed/pages/post_comment_section_page_test.dart --plain-name 'focused link renders root context and focused reply'`.
- Confirmed failure: the route page still watched the stale thread provider, so the comment-section repository callback was never called.
- Implement: changed `PostThreadPage` to watch `postCommentSectionProvider(did, rkey, focus: focus)` and render a root comment-section body with root post, comments, loaded reply items, and a `focused-comment-target` key on the focused item.
- Run command: focused page deep-link test passed; full `flutter test test/feed/pages/post_comment_section_page_test.dart` passed.
- Refactor: kept legacy thread helper widgets in place temporarily for pending regression cleanup; new page path no longer uses the thread provider in the main build.
- Notes: The page class name remains `PostThreadPage` until the stale thread cleanup/regression loop; behavior now uses comment-section data.

### Step 34: AT-002
- Write failing test: add Flutter page coverage that a focused reply branch returned first by the API renders before normal comments and shows the focused reply without intermediate page loads.
- Run command: `flutter test test/feed/pages/post_comment_section_page_test.dart --plain-name 'focused reply branch is promoted before normal comments'`.
- Confirmed failure: initial assertion could not see the normal comment within the default test viewport; after sizing the widget test viewport to cover the rendered section, the behavior was green from Step 33's comment-section page renderer.
- Implement: no production code change required beyond Step 33; adjusted the page-test pump helper to use a stable large viewport for comment-section assertions.
- Run command: focused branch promotion widget test passed.
- Refactor: none.
- Notes: Backend pagination/focus inclusion remains authoritative; the widget test verifies that promoted render order and loaded focused replies are honored by the page.

### Step 35: AT-003
- Write failing test: add Flutter page coverage that an unfocused root post renders the root post and top-level comments while not rendering reply items from an unloaded branch.
- Run command: `flutter test test/feed/pages/post_comment_section_page_test.dart --plain-name 'root post initially shows comments only without replies'`.
- Confirmed failure: test was already green because Step 33's page renderer only renders branch replies when `replies.loaded = true`.
- Implement: no production code change required.
- Run command: focused initial comments-only test passed; full `flutter test test/feed/pages/post_comment_section_page_test.dart` passed.
- Refactor: none.
- Notes: The test intentionally includes a reply item in an unloaded branch to guard against accidentally rendering nested replies before expansion.

### Step 36: AT-004
- Write failing test: add Flutter page coverage that scrolling near the end of the loaded comments requests the next comment page with the current cursor and appends the returned comment.
- Run command: `flutter test test/feed/pages/post_comment_section_page_test.dart --plain-name 'scrolling near the end loads the next comment page'`.
- Confirmed failure: only the initial comment-section request was made; the page did not trigger `loadMoreComments` from scroll position.
- Implement: added near-end scroll handling to the comment-section page using the provider notifier's guarded `loadMoreComments`; kept provider-level duplicate-load protection from UT-008.
- Run command: focused scroll pagination test passed; full `flutter test test/feed/pages/post_comment_section_page_test.dart` passed.
- Refactor: converted the comment-section body to a stateful widget with a `ScrollController` so near-end detection is deterministic in tests and runtime.
- Notes: This covers the widget/user workflow; the provider and backend still enforce bounded page sizes and cursor semantics.

### Step 37: AT-005
- Write failing test: add Flutter page coverage for “view replies”, “load more replies”, and “hide replies” controls under a comment branch.
- Run command: `flutter test test/feed/pages/post_comment_section_page_test.dart --plain-name 'expands, loads more, and hides child replies'`.
- Confirmed failure: the page did not render a “View replies” control for collapsed comment branches with replies.
- Implement: added comment-branch controls using localized labels; changed provider reply loading so an unloaded branch can fetch its first reply page with `cursor = null`, an expanded branch can fetch subsequent pages with its own cursor, and a branch can collapse via `collapseReplies`.
- Run command: focused reply-controls widget test passed; nearby `flutter test test/feed/pages/post_comment_section_page_test.dart test/feed/providers/post_comment_section_provider_test.dart` passed.
- Refactor: reused provider/model state helpers for branch collapse and reply append behavior.
- Notes: Replies remain oldest-first by backend/provider contract; UI now exposes the actioned branch controls required by AC-006/AC-009/AC-010.

### Step 38: AT-006
- Write failing test: add Flutter page coverage that selecting a comment sort option re-fetches the comment section with that sort and renders the backend-ordered viewer-authored and normal comment groups.
- Run command: `flutter test test/feed/pages/post_comment_section_page_test.dart --plain-name 'selecting comment sort rerenders backend ordered comments'`.
- Confirmed failure: the page did not render any sort selector.
- Implement: converted `PostThreadPage` to hold selected sort state, pass it to `postCommentSectionProvider`, and render a localized `DropdownButton<CommentSort>` with `oldest`, `newest`, and `follows`; changing the sort rekeys the provider and fetches the sorted backend response.
- Run command: focused sort-selection widget test passed; nearby `flutter test test/feed/pages/post_comment_section_page_test.dart test/feed/providers/post_comment_section_provider_test.dart` passed.
- Refactor: no unrelated UI styling; backend/provider remains source of truth for viewer grouping and follows-as-oldest behavior.
- Notes: AT-011 will separately pin focus-promotion clearing on sort change.

### Step 39: AT-007
- Write failing test: add Flutter page coverage that creating a top-level comment from the root comment section inserts the synthetic create result into the visible viewer-authored group without changing the selected sort.
- Run command: `flutter test test/feed/pages/post_comment_section_page_test.dart --plain-name 'new top-level comment appears in viewer group after create'`.
- Confirmed failure: the create action initially was not called until the test pumped after text entry; once meaningful, the page had no comment-section cache update for successful top-level comment creation.
- Implement: added `prependCreatedComment` state/provider helpers and updated `CreatePost` to prepend successful top-level replies into any live root `postCommentSectionProvider` entries for each sort, preserving focus-first placement and the selected sort.
- Run command: focused new-comment widget test passed; nearby `flutter test test/feed/pages/post_comment_section_page_test.dart test/feed/providers/post_comment_section_provider_test.dart test/feed/providers/create_post_provider_test.dart` passed.
- Refactor: kept existing create-post profile cache behavior and reply invalidation behavior intact; comment-section cache update is limited to top-level comments where root and parent refs match.
- Notes: Widget test verifies the composer closes, the new comment is visible, and the sort label remains unchanged.

### Step 40: AT-008
- Write failing test: add Flutter page coverage that replying to a visible reply pre-fills the target author mention, sends create refs with the original root and actual reply parent, and inserts the synthetic create result into the nearest comment branch.
- Run command: `flutter test test/feed/pages/post_comment_section_page_test.dart --plain-name 'replying to a reply inserts created reply into comment branch'`.
- Confirmed failure: tapping the reply text did not open the composer; after targeting the reply action, the created reply was not inserted into the visible comment branch.
- Implement: added `PostCommentSection` provider support for inserting created replies by backend parent URI and updated `CreatePost` to insert successful nested replies into any live root comment-section provider entries while preserving actual root/parent refs.
- Run command: focused nested-reply create widget test passed; nearby `flutter test test/feed/pages/post_comment_section_page_test.dart test/feed/providers/post_comment_section_provider_test.dart test/feed/providers/create_post_provider_test.dart test/feed/widgets/post_composer_sheet_test.dart` passed.
- Refactor: reused the existing `insertCreatedReplyIntoNearestBranch` state helper.
- Notes: This covers the composer mention, root/parent preservation, and two-level branch insertion for reply-to-reply creation.

### Step 41: AT-009
- Write failing test: add Flutter page coverage that a focused deep backend reply renders as a second-level visual reply under its comment branch, with no additional indentation level.
- Run command: `flutter test test/feed/pages/post_comment_section_page_test.dart --plain-name 'focused deep reply renders without a third visual level'`.
- Confirmed failure: test was already green because the Step 33/AT-001 renderer only has root/comment/reply visual levels and uses the same indentation for flattened replies as direct replies.
- Implement: no production code change required.
- Run command: focused visual-depth widget test passed; full `flutter test test/feed/pages/post_comment_section_page_test.dart` passed.
- Refactor: none.
- Notes: This pairs with backend IT-016 and Flutter model UT-003/UT-017 to ensure deeper reply data is flattened structurally and visually.

### Step 42: AT-010
- Write failing test: reuse the localization widget test added in UT-013 as the acceptance coverage for comment-section labels coming from `AppLocalizations`.
- Run command: `flutter test test/feed/pages/post_comment_section_page_test.dart --plain-name 'comment section labels are exposed through localizations'`.
- Confirmed failure: test was already green because UT-013 added ARB entries and regenerated localization accessors.
- Implement: no production code change required.
- Run command: focused localization acceptance test passed.
- Refactor: none.
- Notes: Page controls added in AT-005/AT-006 consume the localized sort and reply-control labels.

### Step 43: AT-011
- Write failing test: add Flutter page coverage that a focused branch renders before a viewer-authored comment, then changing sort clears the focus promotion and lets the viewer-authored comment render first.
- Run command: `flutter test test/feed/pages/post_comment_section_page_test.dart --plain-name 'focus promotion clears on sort change'`.
- Confirmed failure: test was already green because AT-006 rekeys the provider on sort changes and the backend response supplies unfocused placement/order after the sort change.
- Implement: no production code change required beyond AT-006.
- Run command: focused focus-clearing widget test passed; full `flutter test test/feed/pages/post_comment_section_page_test.dart` passed.
- Refactor: none.
- Notes: Unit-level `changeCommentSortClearingFocus` remains available for pure state merges, while the page path relies on backend response state after sort/filter changes.

### Step 44: REG-001
- Write failing test: add route regression coverage that the new `GET /v1/posts/{did}/{rkey}/comments` route requires both authentication and device ID, replacing the stale thread-route auth coverage.
- Run command: `go test ./internal/routes -run 'TestAddRoutes_PostCommentsRequiresAuthenticatedDevice|TestAddRoutes_PostCommentsRequiresDeviceID'`.
- Confirmed failure: tests were already green because the comment-section route was registered under the same authenticated + device-ID middleware stack as other `/v1/posts` routes in IT-009.
- Implement: no production code change required; added the regression tests to pin the route middleware behavior.
- Run command: focused route auth/device tests passed.
- Refactor: none.
- Notes: This covers FR-001/NFR-001 for the replacement comment-section route while preserving the existing `/thread` 404 route-removal test.

### Step 45: REG-002
- Write failing test: add Flutter regression coverage that `lib/feed` no longer exposes the stale thread API/model/provider surface (`getThread`, `PostThreadMapper`, `postThreadProvider`, and deleted thread model/provider files).
- Run command: `flutter test test/feed/regression/no_stale_thread_usage_test.dart`.
- Confirmed failure: regression test reported stale offenders in `post_thread_provider`, `post_thread` model/mapper, `PostApiClient.getThread`, `ApiPostRepository.thread`, and `CreatePost`'s stale `postThreadProvider` invalidation path.
- Implement: removed `PostThread` model/mapper/provider files, removed `getThread`/`thread` from the API client and repository contracts, removed stale thread-provider invalidation from create-post success handling, deleted stale thread-specific tests, updated affected router/composer/create tests to use the comment-section repository surface, and kept the route/page entry point backed by `postCommentSectionProvider`.
- Run command: `flutter test test/feed/regression/no_stale_thread_usage_test.dart` passed; nearby `flutter test test/router/router_redirect_test.dart test/feed/widgets/post_composer_sheet_test.dart test/feed/providers/create_post_provider_test.dart test/feed/data/post_api_client_test.dart test/feed/regression/no_stale_thread_usage_test.dart test/feed/pages/post_comment_section_page_test.dart` passed.
- Refactor: removed dead legacy thread rendering helpers from `post_thread_page.dart`; the remaining route/page class names stay as navigation compatibility names while no stale thread API/model/provider surface remains.
- Notes: This completes the FR-001 Flutter-side stale API/model/provider removal without changing lexicon or backend storage semantics.

### Step 46: REG-003
- Write failing test: run existing post create/read/delete/like/repost backend and Flutter client/provider regression coverage after the comment-section replacement.
- Run command: `go test ./internal/api -run 'Test(CreatePost|GetPost|DeletePost|LikePost|UnlikePost|RepostPost|UnrepostPost)'` and `flutter test test/feed/data/post_api_client_test.dart test/feed/providers/create_post_provider_test.dart`.
- Confirmed failure: no failure; the existing post API behavior remained green after stale thread removal and create-post cache update changes.
- Implement: no production code change required.
- Run command: backend focused post API regressions passed; Flutter post API client/create-provider regressions passed.
- Refactor: none.
- Notes: This guards NFR-001 outside the intentional `/thread` removal and comment-section additions.

### Step 47: REG-004
- Write failing test: run existing reply-ref response/create/indexer regression coverage and inspect git status for unintended `lexicon/` changes.
- Run command: `go test ./internal/api -run 'Test(PostResponse|CreatePost_WithReply|ListDirectReplies|GetPostComments_DeeperFocusedReplyIncludesFlattenedMetadata)'`; initially also ran `go test ./internal/index ./internal/app -run 'Test'` without a database URL.
- Confirmed failure: no behavior failure; the first `./internal/app` run failed setup with a malformed/empty DB connection (`lookup x: no such host`). Re-running with `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable` passed.
- Implement: no production code change required.
- Run command: API reply-ref regressions passed; `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/app ./internal/index -run 'Test'` passed.
- Refactor: none.
- Notes: `git status --short` showed no `lexicon/` changes, preserving RULE-001/AC-019.

### Step 48: REG-005
- Write failing test: run backend direct-replies pagination/cap regressions and Flutter provider branch-cursor regression used by the comment-section reply loader.
- Run command: `go test ./internal/api -run 'TestPostStore_ListDirectReplies_PaginatesOpaqueCursorOldestFirst|TestListDirectReplies'` and `flutter test test/feed/providers/post_comment_section_provider_test.dart --plain-name 'reply load more keeps branch cursors and items independent'`.
- Confirmed failure: no failure; direct replies still paginate oldest-first and remain capped/defaulted at 10 after comment-section integration.
- Implement: no production code change required.
- Run command: backend direct-replies regressions passed; focused Flutter provider reply-load regression passed.
- Refactor: none.
- Notes: This preserves the existing direct replies endpoint as the actioned per-comment reply loader for FR-010/FR-011/RULE-004.

## Final Verification
- Backend broader verification: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes ./internal/app ./internal/index` passed.
- Flutter broader verification: `flutter test test/feed test/router/router_redirect_test.dart` passed.
- Diff review: `git status --short` confirmed no `lexicon/` changes and no unrelated `app/lib/shared/messaging` formatting changes remain.
- Coverage gaps: OS-level deep-link launch/push notification delivery remains manual/out of scope per `03-acceptance-tests.md` MAN-001/GAP-003; visual copy clarity for stubbed `follows` remains MAN-003.

## Completion Checklist
- [x] All Must requirements covered by tests or documented gaps
- [x] All planned Must tests passing
- [x] Relevant regression tests passing
- [x] No unlinked behavior implemented
- [x] Docs updated
- [ ] Review completed or explicitly skipped
