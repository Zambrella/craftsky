# Discovery Notes: Threads UX

## Initial Request
Improve app thread behavior so opening a post anchors the selected post at the top of the screen, replies render below top-level posts, reply targets keep parent context available above the selected post, the selected post cannot be tapped into itself, and reply composition shows the post being replied to above the text input.

## Current Codebase Findings
- Relevant files:
  - `app/lib/feed/pages/post_thread_page.dart` renders `PostThreadPage`, ancestors, selected post, reply prompt, and nested replies.
  - `app/lib/feed/models/post_thread.dart` already models `ancestors`, selected `post`, nested `replies`, and `truncated`.
  - `app/lib/feed/widgets/post_composer_sheet.dart` accepts `replyTarget` and creates correct reply refs, but currently shows only the reply title/hint and text field.
  - `app/lib/feed/widgets/post_card.dart` accepts nullable `onTap`, so selected/preview posts can be made non-navigable without changing the shared card API.
  - `app/lib/feed/data/post_api_client.dart` calls `GET /v1/posts/{did}/{rkey}/thread`.
  - `appview/internal/api/post.go` and related tests already return a thread response with root-to-parent `ancestors` for reply targets.
- Existing patterns:
  - Flutter uses Riverpod providers and GoRouter typed route helpers for thread navigation.
  - Small screens use a sticky reply prompt; large screens show an inline reply prompt.
  - AppView reads remain JSON/HTTP under `/v1/`; no API change appears necessary for this UX improvement.
- Current behavior:
  - `PostThreadPage` renders ancestors before the selected post in a plain `ListView`, so a reply target may appear below the initial viewport top instead of just under the app bar.
  - `_ThreadPostCard` always wires `PostCard.onTap` to push `PostThreadRoute`, including for the currently selected anchor post, allowing recursive navigation to the same route/post.
  - Replies appear under the selected post, with continuation controls for deeper branches.
  - The composer computes reply root/parent refs correctly but does not display the replied-to content above the input.
- Constraints discovered:
  - This should remain a Flutter UX change if possible; the current AppView response already supplies ancestors and replies.
  - The selected post should be initially positioned just below the app bar even when it is a reply. Ancestors should remain above it in scrollback.
  - The reply preview should be compact: author + handle and a concise content preview, without full post-card action buttons.
- Test/build commands discovered:
  - App widget/provider tests exist under `app/test/feed/...`.
  - Repository guidance uses `just test` for broader testing; Flutter-specific tests likely run from `app/` via Flutter tooling.

## Clarifying Questions
### Q1: For a reply target, should the initial viewport place the selected reply at the top or start at the parent/root?
Answer: Selected at top.
Decision / implication: Ancestors should remain above the selected post in the scrollable content, but the initial scroll position should place the selected post's top just below the app bar.

### Q2: What kind of replied-to preview should the composer show above the input?
Answer: Compact preview.
Decision / implication: Add a lightweight reply-target preview with author/handle and text, not a full action-enabled `PostCard`.

### Q3: Should the API/AppView be changed for this work?
Answer: No. The confirmed direction is to keep the existing thread API/model and document a Flutter-focused UX change.
Decision / implication: Requirements and tests should focus on `PostThreadPage`, route behavior, and `PostComposerSheet`.

## Candidate Approaches
### Option A: Flutter-focused anchored thread UX
Summary: Keep the current `PostThread` API/model, update the thread page scroll behavior to initially anchor the selected post below the app bar, disable navigation on the selected post, and add a compact reply preview to the composer.
Pros:
- Minimal scope and aligns with current data shape.
- Avoids backend/API churn.
- Directly addresses all stated UX issues.
- Can be covered by widget tests.
Cons:
- Initial scroll anchoring must be implemented carefully to avoid jank after async load/layout.
- Ancestors above the selected post may not be visible until the user scrolls upward.
Risks:
- Medium user-visible UI risk around scroll positioning across device sizes and text scaling.

### Option B: Split reply context from selected thread body in UI
Summary: Render parent context in a fixed/collapsible header area, selected post below it, and replies in a separate list.
Pros:
- Parent context is more visibly present.
- Could make reply context easier to understand.
Cons:
- Conflicts with the confirmed requirement that the selected post starts at the top.
- More bespoke layout complexity and potential nested scroll issues.
Risks:
- Higher chance of accessibility/scroll regressions.

### Option C: Backend-curated display sections
Summary: Change the AppView thread response to explicitly separate display ancestors, anchor, and reply sections with presentation hints.
Pros:
- Moves thread-shaping decisions server-side if multiple clients need identical behavior.
- Could support future pagination/context policies.
Cons:
- Unnecessary for current app-only UX requirements because ancestors/replies already exist.
- Requires API and test changes in Go and Flutter.
Risks:
- Larger blast radius and potential API compatibility concerns.

## Recommendation
Recommended approach: Option A, Flutter-focused anchored thread UX.
Why: The AppView already returns the needed structure, and existing widgets expose enough hooks to disable anchor navigation and add a compact reply preview. This keeps the change narrow while satisfying the confirmed behavior.

## Scope Boundaries
In scope:
- Thread page initial scroll positioning so the selected post starts just below the app bar.
- Keep ancestors above the selected post in the same scrollable thread content.
- Keep replies below the selected post for top-level posts and replies.
- Disable tapping/navigating on the currently selected post.
- Preserve navigation for ancestors, replies, and continuation controls where they point to different posts.
- Add compact replied-to content preview above the composer text input.
- Update/extend Flutter widget tests for these behaviors.

Out of scope:
- AppView/API response changes.
- Lexicon changes.
- New pagination behavior for replies or ancestors.
- Redesigning `PostCard` globally.
- Changing reply record semantics or PDS writes.

## Risks And Review Recommendation
Risk level: Medium.
Review recommended: Yes.
Reason: This is user-visible navigation and composition behavior with scroll-position edge cases across form factors, but it does not touch auth, data models, migrations, or public API contracts.

## Open Questions
- [ ] Exact visual styling of the compact reply preview should be finalized during design/implementation, using existing theme spacing/type tokens.
- [ ] Requirements should specify how much of a long replied-to post is shown before truncation.

## Decision Summary
- Use the existing thread API and model; do not add backend work for this change.
- Initial viewport should anchor the selected post below the app bar, with ancestors above in scrollback.
- The selected post must not navigate to itself.
- The reply composer should show a compact non-actionable preview of the reply target above the text input.

## Handoff To Requirements
- Inputs the requirements agent should use:
  - User's initial requirements.
  - Confirmed answers above.
  - Current implementation notes for `PostThreadPage` and `PostComposerSheet`.
- Requirements areas likely needed:
  - Thread route initial positioning.
  - Ancestor/reply ordering.
  - Anchor post interaction disabling.
  - Reply composer target preview.
  - Accessibility semantics for disabled/non-actionable preview and selected post.
- Acceptance criteria areas likely needed:
  - Top-level post opens with selected post just below the app bar and replies below.
  - Reply post opens with selected post just below the app bar, ancestors available above via upward scroll, and replies below.
  - Tapping selected post does not push another route or reload the same route.
  - Tapping other posts/continuation controls still navigates appropriately.
  - Reply composer displays compact target content above input and still submits correct root/parent refs.
