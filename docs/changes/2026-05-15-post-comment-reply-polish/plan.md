# Post, Comment, and Reply Polish Plan

**Date:** 2026-05-15  
**Status:** Draft for review  
**Scope:** Flutter UI and AppView API/read-model changes for profile comment separation, comment-section loading behavior, reply controls, wording, counts, and repost-button visibility.

## Goals

1. Keep authored comments and replies out of the profile `Posts` tab.
2. Add a profile `Comments` tab next to `Posts` that shows the user's authored top-level comments and nested replies.
3. Navigate from a profile comment/reply row to the root post route with that row selected as the `focus`.
4. Move the `Hide replies` control above the expanded reply list.
5. Keep the root post visible when changing comment sort, with a loading indicator only in the comments area.
6. Rename the root-post reply CTA to `Comment` in user-facing UI.
7. Make `replyCount` represent total descendant replies, not only direct children.
8. Hide repost actions on comments and replies.

## Current State

- `GET /v1/profiles/{handleOrDid}/posts` is handled by `ListPostsByAuthorHandler` in `appview/internal/api/post.go` and uses `PostStore.ListByAuthor` in `appview/internal/api/post_store.go`.
- `PostStore.ListByAuthor` currently returns every authored `craftsky_posts` row, including roots, comments, and nested replies.
- The profile UI has `ProfileTab { posts, projects, saved, reposts, about }` in `app/lib/profile/widgets/profile_tab_bar.dart`; `ProfilePostsTab` consumes `userPostsProvider(handle)`.
- The comment thread UI is in `app/lib/feed/pages/post_thread_page.dart` and currently keys `postCommentSectionProvider` by sort. Changing sort creates a new provider instance, so the entire `Scaffold` body falls back to the full-page spinner.
- Expanded replies are rendered before the controls row in `_CommentCard`, so `Hide replies` appears below replies.
- `PostStore.CountDirectReplies` feeds `EngagementSummaries`, so `replyCount` only counts direct children.
- `PostCard` always renders comment, like, and repost actions. Comments and replies receive the same repost control as root posts.

## Assumptions

- The new profile tab should be named `Comments` and include both authored top-level comments and nested replies, ordered newest-first by `indexed_at`, using the same `PostPage` response shape as profile posts.
- The existing `/v1/profiles/{handleOrDid}/posts` route can change to roots-only without an API version bump because this feature is still pre-release.
- The new endpoint should be additive: `GET /v1/profiles/{handleOrDid}/comments`.
- The existing `PostResponse.reply.root.uri` field is enough for profile comment navigation. No new AppView response field is needed to find the root post.
- `replyCount` should be total descendants for any post-shaped response. For a root post, that means all top-level comments plus all nested replies. For a comment, that means all replies anywhere under that comment branch.
- Repost write endpoints can remain available server-side. This plan only removes repost UI affordances from comments and replies.
- Root-post CTAs should say `Comment`; branch-level actions may still say `Reply` where the user is replying to an existing comment/reply.

## AppView Plan

### 1. Split Profile Posts From Comments

- Change `PostStore.ListByAuthor` to filter root posts only:
  - `reply_root_uri IS NULL`
  - `reply_parent_uri IS NULL`
- Add `PostStore.ListCommentsByAuthor(ctx, did, limit, cursor)`:
  - filter authored rows with both reply columns present
  - order by `indexed_at DESC, uri DESC`
  - reuse the existing opaque cursor shape with `indexedAt` and `uri`
- Extend `PostReader` with `ListCommentsByAuthor` only if handlers continue to use the shared interface; otherwise introduce a smaller interface for the new handler.
- Add `ListCommentsByAuthorHandler` in `appview/internal/api/post.go`:
  - resolve `{handleOrDid}` the same way `ListPostsByAuthorHandler` does
  - call `ListCommentsByAuthor`
  - hydrate handles and engagement summaries like the posts handler
  - return `{items, cursor}` with camelCase keys
- Register `GET /v1/profiles/{handleOrDid}/comments` in `appview/internal/routes/routes.go` with the same authenticated + device-id middleware stack.

### 2. Count All Descendant Replies

- Replace `CountDirectReplies` with a batch-friendly descendant count.
- Use a recursive CTE seeded by the requested post URIs:
  - start with direct children where `reply_parent_uri = subject_uri`
  - recursively join children where `reply_parent_uri = previous_child_uri`
  - cap depth at the existing branch traversal limit of 64
  - count descendants per original subject URI
- Keep the public response field named `replyCount`; only its semantics change.
- Keep like/repost counts unchanged.

### 3. Server Tests

- Add/adjust `appview/internal/api/post_store_test.go` coverage:
  - `ListByAuthor` excludes comments and nested replies
  - `ListCommentsByAuthor` returns only authored comments/replies and paginates
  - descendant `replyCount` includes all nested replies for root posts and comment branches
- Add/adjust `appview/internal/api/post_test.go` handler coverage:
  - profile posts endpoint returns roots only
  - profile comments endpoint resolves handle/DID, returns authored comments/replies, handles invalid cursor, and hydrates engagement summaries
- Add/adjust `appview/internal/routes/routes_test.go` for `GET /v1/profiles/{handleOrDid}/comments` auth/device behavior.

## Flutter Plan

### 1. Add Profile Comments Data Path

- Add `PostApiClient.listCommentsByAuthor(...)` for `GET /v1/profiles/@{handleOrDid}/comments`.
- Add `PostRepository.listCommentsByAuthor(...)` and implement it in `ApiPostRepository`.
- Extend `FakePostRepository` for tests.
- Add a `userCommentsProvider` mirroring `userPostsProvider` pagination behavior, or extract a small shared paginated-author-list helper if it stays simple.
- Add `ProfileCommentsTab` under `app/lib/profile/widgets/profile_tabs/`, reusing most of `ProfilePostsTab` list rendering but without the new-post composer button.
- Add `ProfileTab.comments` immediately after `ProfileTab.posts` and wire it in `ProfilePage`.
- On tap, comments-tab rows should navigate to `PostThreadRoute` for the root post and pass the tapped row URI as `focus`:
  - read `post.reply.root.uri` from the tapped comment/reply
  - parse the root AT-URI into `{did, rkey}` for `PostThreadRoute(did: ..., rkey: ..., focus: post.uri)`
  - if a malformed or missing root reference is encountered, fail safely by disabling navigation for that row or routing to the row itself only if it can be proven to be a root post
- Add a small AT-URI parsing helper near the feed/router layer if no existing helper is available; keep it narrow to Craftsky post URIs (`at://{did}/social.craftsky.feed.post/{rkey}`).
- Add l10n strings:
  - `profileTabComments`: `Comments`
  - `profileCommentsEmpty`: `No comments yet.`
  - `profileCommentsLoadError`: `Comments didn't load.`
  - `profileCommentsLoadMore`: `Load more comments`

### 2. Hide Repost Actions On Comments And Replies

- Add a presentation option to `PostCard`, for example `showRepostAction = true`.
- Keep repost visible for root posts in feed/profile/thread root cards.
- Pass `showRepostAction: false` for comment cards and nested reply cards in `post_thread_page.dart`.
- Do not call `toggleRepostPostProvider` from comment/reply render paths.

### 3. Move Hide Replies Above Replies

- In `_CommentCard`, when `item.replies.loaded` is true:
  - render a controls row above the reply list containing `Hide replies`
  - render loaded replies below that row
  - keep `Load more replies` below the reply list so it remains near the pagination boundary
- Preserve existing loading behavior for `View replies` and `Load more replies`.

### 4. Comments-Only Spinner On Sort Change

- Keep the last successful `PostCommentSection` in `_PostThreadPageState` when the user changes sort.
- While the newly keyed `postCommentSectionProvider(... sort: _sort ...)` is loading:
  - render the cached root post card and sort control
  - replace only the comments list area with `StitchProgressIndicator`
  - keep the full-page spinner only for the initial load when no cached section exists
- Clear any focused-link promotion when switching sort by continuing to request the new sort without the focus parameter after the first focused render if needed. If keeping the current focus behavior is preferred, document it in the test expectation.

### 5. Rename Root-Post Reply CTA To Comment

- Add/adjust l10n so root-post actions can say `Comment`:
  - `postCommentAction`: `Comment`
  - `postCommentOnAuthor`: `Comment on {author}`
- Use `Comment` for the sticky and inline root-post prompt on `PostThreadPage`.
- Use `Comment` as the root post card action tooltip where the action creates a top-level comment.
- Keep branch-level composer language as `Reply` unless product wants a global wording change for every reply-shaped write.

### 6. Flutter Tests

- Add/adjust `app/test/feed/data/post_api_client_test.dart`:
  - new `listCommentsByAuthor` route and query params
  - existing `listPostsByAuthor` still points to `/posts`
- Add provider tests for `userCommentsProvider` if added.
- Add/adjust profile widget tests:
  - `Comments` tab appears next to `Posts`
  - posts tab reads roots-only data from `listByAuthor`
  - comments tab reads `listCommentsByAuthor`
  - tapping a comment/reply opens `/posts/{rootDid}/{rootRkey}?focus={commentOrReplyUri}`
- Add focused unit coverage for the AT-URI parsing helper if one is introduced.
- Add/adjust `post_comment_section_page_test.dart`:
  - `Hide replies` appears above the first loaded reply
  - changing sort keeps the root post visible and shows a spinner in the comments section only
  - comment and reply cards do not render repost icons/buttons
  - root post prompt says `Comment`
- Add/adjust `post_card_test.dart` for the new `showRepostAction` option.

## Implementation Order

1. AppView store and handler changes for profile posts/comments split.
2. AppView descendant reply count change.
3. Flutter API/repository/provider additions for profile comments.
4. Profile `Comments` tab UI and l10n.
5. Thread UI polish: hide-replies placement, comments-only spinner, root CTA wording, repost hiding.
6. Run generation steps for Flutter codegen/l10n if needed.
7. Run focused Go and Flutter tests, then broader project test commands if practical.

## Verification Commands

- `just test` from the repo root for AppView Go tests.
- `flutter test test/feed/data/post_api_client_test.dart test/feed/pages/post_comment_section_page_test.dart test/profile/profile_page_test.dart test/feed/widgets/post_card_test.dart` from `app/` for focused Flutter coverage.
- Run `dart run build_runner build --delete-conflicting-outputs` from `app/` if new Riverpod providers or mappable classes are added.
- Run Flutter l10n generation if generated localization files are not updated automatically by the normal build flow.

## Risks And Notes

- The recursive descendant count is more expensive than direct counts. Keep it batch-based, depth-capped, and covered by tests. Existing indexes on `reply_parent_uri` and `reply_root_uri` should support this, but query plans should be checked if counts become slow.
- Profile comments expose reply-shaped posts in a profile context. The UI should make the tab label clear enough without adding parent-context previews in this pass.
- Hiding repost UI on comments/replies does not prevent existing indexed reposts from contributing to counts if they exist. This is acceptable unless product wants server-side rejection for comment/reply reposts.
- The root CTA wording needs careful use of context: `Comment` for root posts, `Reply` for replies to comments/replies.
