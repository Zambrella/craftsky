# Post, Comment, and Reply Polish Amendments

**Date:** 2026-05-15  
**Status:** Draft for review  
**Parent plan:** `docs/changes/2026-05-15-post-comment-reply-polish/plan.md`  
**Scope:** Follow-up UI behavior and reply ordering amendments after the profile comments tab implementation.

## Goals

1. Scroll focused comments/replies into view when opening a post thread with `focus`.
2. Visually highlight focused comments/replies for a few seconds.
3. Hide reply counts on reply cards while preserving reply counts on top-level comments.
4. Verify and fix reply ordering so replies under a comment render oldest-first.
5. Add a tooltip to the post-card relative time label that shows the full timestamp.

## Current State

- `PostThreadPage` already passes `focus` to `postCommentSectionProvider`, and focused items get `ValueKey('focused-comment-target')` in `app/lib/feed/pages/post_thread_page.dart`.
- There is no code that calls `Scrollable.ensureVisible`, so focused comments/replies may be present in the widget tree but remain off-screen.
- There is no focused-state visual treatment. Focused items are only discoverable by tests through the key.
- `PostCard` always displays `post.replyCount` next to the comment/reply action. Thread comment cards and nested reply cards both use `PostCard`.
- `PostStore.ListCommentBranchReplies` currently orders flattened branch replies by `created_at ASC, uri ASC`, and the previous acceptance spec says oldest-first should be preserved.
- `PostCard` renders relative time in `_PostCardTime`, but that text has no tooltip.

## Decisions

- Reply branch order remains **oldest-first**.
- Top-level comments keep reply counts because they indicate whether branch replies are available.
- Nested replies hide reply counts even if the backend still returns `replyCount`.
- The highlight should be temporary, lasting a few seconds, and should not mutate post/comment data.
- The full timestamp tooltip should use the viewer's local time, with a stable, readable format including date and time.

## Implementation Plan

### 1. Focus Scroll Into View

- In `_CommentSectionBodyState`, track the latest focus URI that has been scrolled to, so the app does not repeatedly force-scroll on rebuilds.
- Introduce one `GlobalKey` for the currently focused visible target.
- Pass that key to the focused comment card or focused reply card instead of relying only on `ValueKey('focused-comment-target')`.
- After a frame where `widget.section.focus?.status == FocusStatus.included`, call `Scrollable.ensureVisible` on the focused target context:
  - use a short duration such as 250-350ms
  - use a middle-ish alignment so the focused card is not hidden under the app bar or bottom composer
  - guard against missing context because not every focus status yields a visible target
- Keep the existing `ValueKey('focused-comment-target')` for tests and continuity.

### 2. Temporary Focus Highlight

- Add a transient highlighted-focus URI in `_CommentSectionBodyState`.
- When an included focus target becomes visible, set the highlighted URI to the focus URI.
- Clear it after a short delay, such as 3 seconds, if the widget is still mounted and the highlighted URI has not changed.
- Add `PostCard` support for a highlighted state, preferably as an optional `isHighlighted` boolean with default `false`.
- Implement the visual treatment in or around `PostCard` using the existing design system:
  - keep the chunky card shape
  - use a soft brand-colored background/border/shadow change
  - animate back to the normal surface rather than disappearing abruptly
- Pass `isHighlighted: true` only to the focused comment/reply while the transient highlight is active.

### 3. Hide Reply Counts On Replies

- Add a `showReplyCount` option to `PostCard`, defaulting to `true`.
- Keep `showReplyCount: true` for root posts and top-level comments.
- Pass `showReplyCount: false` for nested reply cards in `PostThreadPage`.
- In `ProfileCommentsTab`, derive whether a row is a top-level comment or nested reply from `post.reply`:
  - top-level comment: `post.reply.root.uri == post.reply.parent.uri`
  - nested reply: `post.reply.root.uri != post.reply.parent.uri`
- In `ProfileCommentsTab`, pass `showReplyCount: false` for nested replies and keep counts for top-level comments.
- Keep backend `replyCount` unchanged because comment branch controls still need it.

### 4. Verify And Fix Reply Ordering

- Preserve oldest-first branch order as the intended behavior.
- Add/adjust backend store tests for `ListCommentBranchReplies`:
  - direct replies ordered by `created_at ASC, uri ASC`
  - nested/flattened descendants also appear in oldest-first order across the visible branch
  - pagination cursor continues oldest-first without duplicates or skips
- Add/adjust `ListCommentBranchRepliesAround` tests so the focused bounded page is returned oldest-first after the descending selection window is re-ordered.
- Add/adjust Flutter thread tests to assert visible reply card order after expanding a branch.
- If tests reveal a mismatch, fix the SQL/order or client append behavior to match oldest-first.
- Re-check the created-reply insertion helper so newly-created replies append to loaded branches only when that preserves oldest-first order; otherwise sort the branch by `createdAt` after insertion.

### 5. Full Timestamp Tooltip

- Wrap `_PostCardTime`'s relative time text in a `Tooltip`.
- Tooltip message should show the full local timestamp for `postedAt`.
- Prefer `MaterialLocalizations` for readable local date/time formatting, with the time zone name appended if available.
- Preserve existing relative text (`now`, `5m`, `3h`, `2d`) and layout.
- Add a widget test that verifies hovering/long-pressing the time label exposes the full timestamp tooltip text.

## Test Plan

- AppView:
  - `go test ./internal/api ./internal/routes`
  - `just test` now that local Postgres is running
- Flutter focused tests:
  - `flutter test test/feed/pages/post_comment_section_page_test.dart`
  - `flutter test test/feed/widgets/post_card_test.dart`
  - `flutter test test/profile/profile_page_test.dart`
- Static checks:
  - `dart analyze`

## Risks And Notes

- Auto-scrolling too early can fail if the focused reply branch has not been laid out. Use post-frame scheduling after `section.focus` is included.
- Highlight timing should be resilient to rebuilds and sort changes; compare focus URI before clearing delayed state.
- Hiding reply counts should be presentation-only. The model and AppView response should continue to include `replyCount`.
- `Scrollable.ensureVisible` may interact with the sticky bottom composer on small screens; test at a small viewport size.
