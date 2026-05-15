# Discovery Notes: Reply Comment Section

## Initial Request

Update the implementation of replies based on draft requirements and brainstorm the right shape before implementation. The desired UX is a deep-linkable comment/reply experience with comment ordering, lazy loading, expandable replies, and a maximum of two visible levels.

## Current Codebase Findings

- Relevant files:
  - `lexicon/social/craftsky/feed/post.json` defines replies as regular `social.craftsky.feed.post` records with `reply.root` and `reply.parent` strong refs.
  - `appview/migrations/000010_craftsky_posts.up.sql` stores `reply_root_uri`, `reply_root_cid`, `reply_parent_uri`, and `reply_parent_cid`, with partial indexes on root and parent URI.
  - `appview/internal/api/post.go` exposes `GET /v1/posts/{did}/{rkey}/replies` and `GET /v1/posts/{did}/{rkey}/thread`.
  - `appview/internal/api/post_store.go` implements direct-reply pagination and recursive thread loading.
  - `app/lib/feed/pages/post_thread_page.dart` renders a recursive thread page with ancestors, anchor post, replies, and continuation links.
  - `app/lib/router/router.dart` currently routes `/posts/:did/:rkey` to `PostThreadPage`.
- Existing patterns:
  - Reads come from AppView JSON/HTTP APIs under `/v1/`.
  - Replies are posts, not a separate record type.
  - API pagination uses opaque cursors and a `limit` query param.
  - Current direct replies endpoint returns oldest-first direct children.
  - Current thread endpoint has recursive caps: `threadMaxDepth = 6` and `threadMaxPosts = 500`.
- Current behavior:
  - Opening a post route renders a thread-style view, not a comment-section-style view.
  - The thread view can show ancestors above the selected post and recursive descendants below it.
  - Direct replies can be fetched separately but only oldest-first.
  - The app is not in production, so breaking API/client changes and removing unused routes are acceptable.
- Constraints discovered:
  - No real follows graph is present yet; current profile UI has follow placeholders only. `follows` ordering cannot be implemented for real in this change without adding follow indexing/storage first.
  - Lexicon changes are not needed for this feature because existing reply root/parent refs already preserve full reply structure.
  - Backend should retain full reply parentage even though UI visually caps nesting at two levels.
- Test/build commands discovered:
  - Repository guidance says `just test` runs Go tests against the compose Postgres.
  - Flutter-specific test commands were not inspected in this discovery pass.

## Terminology

- **Comment**: A direct/top-level reply to the root post. This is the product, API, client, and user-facing name for direct replies to a root post.
- **Reply**: A reply to a comment. Deeper backend replies are still displayed as replies under their nearest comment ancestor.
- Backend storage may continue using `reply_*` field names where those names describe atproto record refs, but AppView read models, Flutter models/providers/widgets, API response fields, tests, and UI copy should use comment-section terminology.

## Clarifying Questions

### Q1: For the two-level maximum, should users be prevented from replying to a reply?

Answer: No. If the user clicks reply on a reply, the composer should include an `@user123` mention and the backend reply should still link to that actual post as parent. From the frontend standpoint, it is displayed under the existing reply list for the nearest comment rather than as a third indentation level.

Decision / implication: The backend keeps exact parent/root reply refs. The UI flattens deeper replies into the second visual level under the appropriate top-level branch.

### Q2: Should `follows` ordering be fully implemented now?

Answer: No. The dropdown should include `follows`, but it is stubbed/no-op for now.

Decision / implication: Requirements should call out visible `follows` sort as a placeholder. Until follow data exists, `follows` should behave like `oldest`.

### Q3: Which deep-link shape should be used for replies?

Answer: Use the root post route with a `focus` query parameter, e.g. `/posts/{rootDid}/{rootRkey}?focus={replyUriOrRef}`.

Decision / implication: Deep links should always open the root post comment section and focus/expand/scroll to the target comment or reply. Backend/client must ensure the focused branch is included even if it is outside the first comment page.

Follow-up decision: The query parameter name is `focus`, and it carries a URL-encoded AT-URI. `focus` may target either a comment or a reply.

Focus handling decisions:
- The route root remains authoritative. If `focus` points to a record under a different root, the app does not redirect.
- Malformed `focus` is a hard `400 invalid_focus` API error.
- Well-formed but missing/deleted/unindexed `focus` returns a normal root-post comment-section response with `focus.status = "notFound"`.
- Indexed focus under a different root returns a normal root-post comment-section response with `focus.status = "mismatchedRoot"`.
- Available focus under the route root returns `focus.status = "included"` and includes the focused branch in the normal rendered comment structure.
- Focus statuses use exact camelCase strings: `included`, `notFound`, `mismatchedRoot`.
- The focus object always echoes the requested `uri` when `focus` is present.
- For `included`, the focus object includes `kind: "comment" | "reply"`; reply focus also includes `commentUri` for the owning comment branch.
- If a focus is not found, deleted, or unavailable, the page content loads exactly as if the root post had been clicked normally, with an optional lightweight unavailable-focus message.

### Q4: How should a newly-created comment behave when the current sort is not newest?

Answer: Viewer-authored comments should always appear in a top group. Newly-created comments appear in that group and scroll into view, even if selected ordering is oldest or follows.

Decision / implication: Client/API state needs viewer-authored comment grouping with de-duplication against paginated results. The selected sort applies inside the viewer-authored group and inside the remaining-comments group.

Platform pattern note: common platforms vary here. Some highlight or prioritize the viewer's own comment after posting, some rank creator/verified/relevant comments above chronological order, and strictly chronological views usually keep comments in sort order after the immediate post confirmation affordance. Craftsky is making a deliberate product exception for viewer-authored comment grouping.

Follow-up decision: Viewer-authored comments are a deliberate product exception and always appear before other non-focused comments. Newly-created comments appear in that viewer-authored group and scroll into view.

### Q5: When a user taps “view replies” on a comment, how many replies should load initially?

Answer: Load 10.

Decision / implication: Expanded reply lists page independently with page size 10.

### Q6: Should the comment ordering dropdown apply to replies too?

Answer: No. Ordering applies only to the comment list.

Decision / implication: Replies use a fixed oldest-first conversation order, while comments support oldest/newest/follows.

### Q7: Should the current `/thread` endpoint remain?

Answer: No. Since the app is not in production, delete `/thread` and replace the thread page model with the new comment-section model.

Decision / implication: Requirements can include removal of `/v1/posts/{did}/{rkey}/thread`, the Flutter thread API client method/provider/model usage, and migration of `/posts/:did/:rkey` to the new comment-section experience.

### Q8: How should loading behavior differ between levels?

Answer: Top-level replies should be lazy-loaded as the user scrolls. Second-level replies should be user-actioned: “view replies”, then “load more”. Once replies are loaded, show “hide replies” where “view replies” was.

Decision / implication: Client state needs per-top-level-reply expansion state, cursor state, loaded children, and hide/show controls. Top-level list needs scroll-driven pagination.

## Candidate Approaches

### Option A: Adapt Current Recursive Thread Endpoint

Summary: Keep `/thread`, fetch recursive data, and have Flutter flatten/limit the recursive tree into a two-level comment view.

Pros:
- Reuses existing recursive backend code.
- Less immediate backend route churn.
- Existing deep-link-to-post behavior can continue to use ancestors and descendants.

Cons:
- Recursive response does not match the desired comment-section UX.
- Harder to paginate comments 10 at a time.
- Harder to order comments independently from replies.
- More client-side transformation and edge cases.

Risks:
- The implementation could preserve thread-view complexity while still failing to deliver predictable comment-section behavior.

### Option B: Replace Thread View With Comment-Section API

Summary: Delete `/thread` and introduce a root-post comment-section API/surface that returns the root post, a lazy-loadable comment page, reply counts, sort metadata, and optional focused branch inclusion. Expanded replies are loaded separately in pages of 10.

Pros:
- Best fit for the desired UX: root post + comments + expandable replies.
- Clean separation between top-level lazy loading and user-actioned child loading.
- Supports top-level ordering without recursive sorting ambiguity.
- Allows focused reply deep links to be handled deliberately.
- Removes obsolete thread-view concepts before production.

Cons:
- Requires API/client response shape changes.
- Requires replacing existing Flutter `PostThread` state and page behavior.
- Requires careful focused-reply inclusion when the focused item is not in the first page.

Risks:
- Medium risk due to coordinated backend and frontend changes, route removal, pagination behavior, and scroll/focus UX.

### Option C: Compose Existing Direct Reply Endpoint Client-Side

Summary: Keep `/posts/{did}/{rkey}/replies`, remove/ignore `/thread`, and have Flutter orchestrate root fetch, comments, expanded replies, and focused context through multiple existing calls plus small endpoint additions only if needed.

Pros:
- Smaller backend changes than a full new response shape.
- Direct-reply endpoint already exists and matches child-list loading.
- Could be implemented incrementally.

Cons:
- Focused deep links are difficult if the focused top-level branch is outside the first page.
- More round trips and more orchestration in Flutter.
- Top-level ordering still needs endpoint changes.
- The line between root comment-section state and generic direct replies may remain blurry.

Risks:
- Client complexity grows quickly, especially around focus/scroll and pagination race conditions.

## Recommendation

Recommended approach: Option B, replace the current thread view with a comment-section-oriented API and UI.

Why:
- The requested behavior is no longer a recursive thread reader; it is a root-post comment section.
- A purpose-built comment-section shape makes comment sorting, lazy loading, focused branch inclusion, and two-level visual display explicit.
- Keeping full reply parentage in storage avoids losing atproto/threading fidelity while still giving the product a simple two-level UI.
- The app is not in production, so removing `/thread` now is acceptable and avoids supporting two overlapping mental models.

## Scope Boundaries

In scope:
- Replace `/posts/:did/:rkey` UI behavior with a root post comment section.
- Remove the `/v1/posts/{did}/{rkey}/thread` backend route and corresponding Flutter thread API/model/provider usage.
- Support comment/reply deep links as root post route plus `focus` query parameter.
- Comment ordering dropdown with `oldest`, `newest`, and stubbed `follows`.
- Comment lazy loading as the user scrolls, in pages of 10.
- Expandable replies with “view replies”, “load more”, and “hide replies”, loaded 10 at a time.
- Two-level visual maximum while preserving full backend parentage.
- New reply focus/scroll behavior:
  - focused comment branches appear first on initial deep-link load;
  - viewer-authored comments appear before other non-focused comments;
  - newly-created comments appear in the viewer-authored comment group and are scrolled into view;
  - replies to another reply are displayed/scrolled within the relevant comment's reply list;
  - replies deeper than second-level are displayed as flattened replies under the nearest comment ancestor.

Out of scope:
- Real follows graph indexing/storage or true follows-based ranking.
- Lexicon changes.
- Infinite visible nesting.
- Production backward compatibility for `/thread`.
- Push notification infrastructure itself, beyond ensuring the route/API can support a focused reply deep link.

## Risks And Review Recommendation

Risk level: Medium

Review recommended: Yes

Reason: This is a coordinated user-visible behavior and API change across AppView and Flutter. It removes an existing route, changes pagination and scroll behavior, and introduces deep-link focus semantics. The risk is not high because there are no auth, billing, destructive, or privacy changes, and the app is not in production.

## Open Questions

- [ ] Exact backend route name for the comment-section endpoint.
- [ ] If deeper replies are flattened under the nearest top-level ancestor, the exact visual treatment for showing the true parent context, such as an `@user` mention, "replying to" label, or no extra label beyond the composer mention.

## Decision Summary

- Use root-post deep links with a `focus` query parameter rather than making the comment/reply itself the primary route target.
- Use `focus` as a URL-encoded AT-URI query parameter for both comment and reply targets.
- Treat top-level replies as **comments** and replies to comments as **replies** across product/API/client/docs/tests.
- Delete the existing `/thread` endpoint and replace thread UI with comment-section UI.
- Preserve full backend reply parentage even though frontend displays a maximum of two levels.
- Replying to a reply creates a real reply to that target post but displays it under the nearest comment's reply list.
- Include `follows` in the ordering dropdown as a stub/no-op that behaves like oldest-first until follow data exists.
- Comments lazy-load on scroll in pages of 10.
- Replies load only by user action, 10 at a time, can be hidden after loading, and are always oldest-first.
- Comment sort applies only to comments; replies have a fixed oldest-first order.
- Focus statuses are `included`, `notFound`, and `mismatchedRoot`; malformed focus is `400 invalid_focus`.
- Focused branches are promoted above viewer-authored comments and remaining comments until the user changes sort/filter.
- The comment-section response should use one ordered `comments.items` array with required `placement` metadata: `focused`, `viewerAuthored`, or `normal`.
- Every comment item should include a `replies` object with explicit `loaded` state and `items`.
- Reply items should include backend structural metadata for flattened replies: `flattened` and, when flattened, `replyingTo` with `uri`, `did`, `handle`, and optional `displayName`.

## Handoff To Requirements

- Inputs the requirements agent should use:
  - This discovery note.
  - Existing AppView API conventions in `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md`.
  - Existing API wire conventions in `docs/superpowers/specs/2026-04-22-api-wire-alignment-design.md`.
  - Current reply/thread files listed in Current Codebase Findings.
- Requirements areas likely needed:
  - Backend route removal and new/updated comment-section endpoints.
  - Query params and response shape for comment sort, pagination, focus status, focus promotion, and focused reply inclusion.
  - Flutter routing with `focus` query parameter.
  - Client state for comment lazy loading, per-comment reply pagination, expansion/hide state, viewer-authored comment grouping, focus promotion, and focused child slices.
  - UX strings for ordering dropdown, view replies, load more, hide replies, and focused reply states.
  - Test coverage expectations for backend API, response ordering/pagination, focused branch inclusion, and Flutter UI behavior.
- Acceptance criteria areas likely needed:
  - Deep link to a comment or reply opens the root post and scrolls/focuses the target.
  - Opening a post shows only comments initially.
  - Comments lazy-load 10 more on scroll.
  - “View replies” loads 10 replies and changes to “hide replies”.
  - “Load more” on expanded replies loads 10 more.
  - Creating comments and replies scrolls the created item into view.
  - Ordering dropdown supports oldest/newest and displays stubbed follows.
  - No third indentation level is displayed.
  - `/thread` route/client usage is removed.
