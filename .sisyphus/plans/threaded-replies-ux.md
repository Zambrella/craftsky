# Threaded Replies UX Implementation

## TL;DR
> **Summary**: Build a Bluesky-inspired anchored thread experience for Craftsky: ancestors above, selected post as anchor, direct replies below, focused branch continuation rows, and responsive reply composer behavior. The current AppView thread API only returns target + descendants, so the implementation must extend the API contract to include ancestor context before building the Flutter screen.
> **Deliverables**:
> - AppView `/v1/posts/{did}/{rkey}/thread` response includes root-to-parent `ancestors`.
> - Flutter post thread data model/API client parses ancestors and supports reply creation payloads.
> - Dedicated thread route/page at `/posts/:did/:rkey`.
> - Focused branch UI with Craftsky styling, local reply actions, collapsed continuation rows, and responsive composer.
> - Unit/widget tests plus agent-executed QA evidence.
> **Effort**: Medium
> **Parallel**: YES - 4 waves
> **Critical Path**: Task 1 → Task 2 → Task 4 → Task 5 → Task 6

## Context

### Original Request
- User wanted to think through replies from a UX perspective.
- User likes Bluesky's reply UX because it feels intuitive.
- User selected **Option 1: Bluesky-style anchored thread**.
- User selected **Focused Branches**.
- User selected **Responsive Composer**.
- User then requested an implementation plan.

### Interview Summary
- Primary priority: readable threaded conversations, not flat comments.
- Selected UX: tapping a post opens a dedicated thread screen where the selected post is the anchor.
- Ancestor context appears above the anchor.
- Direct replies appear below the anchor.
- Side/deeper branches are not expanded indefinitely; they use explicit continuation rows.
- Composer behavior: inline composer below the anchor on wide screens; sticky bottom reply prompt on mobile/tablet.

### Metis Review (gaps addressed)
- **UI-only vs reply creation**: include reply creation, because composer behavior is meaningless without posting replies.
- **Thread data shape**: existing API lacks ancestor content; extend AppView response with `ancestors`.
- **Missing parent/deleted/unindexed context**: use root-to-parent ancestors only when indexed; do not invent placeholders in this pass. Show ancestor context as best-effort and keep anchor usable.
- **Route shape**: use `/posts/:did/:rkey`, not encoded AT URI, because repository methods already use DID + rkey and AT URIs contain slash-separated segments.
- **Re-anchor behavior**: tapping any post/reply in the thread navigates to `/posts/{author.did}/{rkey}` and makes that post the new anchor.
- **Wide screen breakpoint**: use existing `FormFactor.isLarge`, i.e. laptop/desktop (`width > 900`) from `app/lib/theme/form_factor.dart:14-21`.
- **Nesting rule**: render ancestors as a compact path, render only direct replies under the anchor, and collapse deeper descendants behind continuation rows.

## Work Objectives

### Core Objective
Implement threaded replies as a readable, focused, Bluesky-inspired conversation screen that respects Craftsky's AppView-read/PDS-write architecture and existing Flutter design system.

### Deliverables
- AppView thread response has `ancestors` with root-to-parent `PostResponse` objects.
- Flutter `PostThread` model has `ancestors`, `post`, `replies`, and `truncated`.
- Flutter create-post flow accepts optional reply target `{root, parent}` and sends it to `POST /v1/posts`.
- Root-navigator Flutter route `/posts/:did/:rkey` displays a thread detail page.
- Feed/profile post cards navigate to the thread screen; reply buttons open targeted composer.
- Thread UI renders ancestor context, anchor, direct replies, collapsed deeper branches, loading/error/empty states, and responsive composer.

### Definition of Done (verifiable conditions with commands)
- From repo root: `just test` passes.
- From `app/`: `dart run build_runner build --delete-conflicting-outputs` completes and generated files are committed.
- From `app/`: `flutter test test/feed test/router` passes.
- From `app/`: `flutter analyze` passes.
- Manual/agent QA evidence exists under `.sisyphus/evidence/` for each task.

### Must Have
- Ancestors above anchor; anchor is visually prominent; replies below anchor are oldest-first.
- Direct replies visible by default; nested descendants collapsed behind continuation rows.
- Tapping a reply re-anchors to that reply's thread route.
- Mobile/tablet uses sticky bottom reply prompt; laptop/desktop uses inline composer below anchor.
- Reply creation sends lexicon-shaped strong refs using the target post's `uri` and `cid`.
- All JSON remains camelCase.
- The Flutter app continues to read via AppView and write through AppView only.

### Must NOT Have
- No direct Flutter reads/writes to a PDS.
- No lexicon changes.
- No generic OAuth/token changes.
- No infinite Reddit-style indentation.
- No algorithmic ranking or non-chronological reply sorting.
- No placeholder fake parent posts for missing/unindexed ancestors.
- No second API serialization format.

## Verification Strategy
> ZERO HUMAN INTERVENTION - all verification is agent-executed.
- Test decision: tests-after, using existing Go tests (`just test`) and Flutter `flutter_test`/Riverpod/http_mock_adapter tests.
- QA policy: Every task has agent-executed scenarios.
- Evidence: `.sisyphus/evidence/task-{N}-{slug}.{ext}`

## Execution Strategy

### Parallel Execution Waves
> Target: 5-8 tasks per wave. <3 per wave (except final) = under-splitting.
> Extract shared dependencies as Wave-1 tasks for max parallelism.

Wave 1: Task 1 backend API ancestors; Task 3 reply create contract can start after reading existing request shape but must coordinate response model names.
Wave 2: Task 2 Flutter data model/API after Task 1 contract is fixed; Task 4 route/navigation can proceed in parallel after Task 2 model names are known.
Wave 3: Task 5 thread page UI; Task 6 responsive composer and reply actions; Task 7 polish/accessibility/localization.
Wave 4: Task 8 integration verification and cleanup.

### Dependency Matrix (full, all tasks)
- Task 1: blocks Task 2 and Task 5.
- Task 2: blocked by Task 1; blocks Task 5 and Task 6.
- Task 3: blocks Task 6.
- Task 4: blocks Task 5 navigation entry and QA.
- Task 5: blocked by Tasks 1, 2, 4; blocks Task 7 and Task 8.
- Task 6: blocked by Tasks 2, 3, 5; blocks Task 8.
- Task 7: blocked by Tasks 5 and 6; blocks Task 8.
- Task 8: blocked by all implementation tasks.

### Agent Dispatch Summary (wave → task count → categories)
- Wave 1 → 2 tasks → `unspecified-high`, `quick`
- Wave 2 → 2 tasks → `unspecified-high`, `quick`
- Wave 3 → 3 tasks → `visual-engineering`, `unspecified-high`, `quick`
- Wave 4 → 1 task → `unspecified-high`

## TODOs
> Implementation + Test = ONE task. Never separate.
> EVERY task MUST have: Agent Profile + Parallelization + QA Scenarios.

- [x] 1. Extend AppView thread response with ancestor context

  **What to do**: Add root-to-parent ancestor support to `GET /v1/posts/{did}/{rkey}/thread`. Keep existing `post`, `replies`, and `truncated` fields. Add `ancestors` as a JSON array of `PostResponse` objects ordered from root parent down to immediate parent. Use existing `PostRow.ReplyRootURI` / `ReplyParentURI` fields to walk indexed parents only. If any ancestor is missing/unindexed, return the ancestors that can be resolved and still return the target post. Do not add placeholders. Preserve oldest-first descendant ordering.
  **Must NOT do**: Do not change the route path, do not add pagination in this pass, do not change lexicon schemas, and do not alter reply write semantics.

  **Recommended Agent Profile**:
  - Category: `unspecified-high` - Reason: crosses API response shape, store query logic, and Go tests.
  - Skills: [] - No special skill needed; no lexicon change.
  - Omitted: [`atproto-lexicon`] - No `lexicon/` files are edited.

  **Parallelization**: Can Parallel: YES | Wave 1 | Blocks: [2, 5] | Blocked By: []

  **References** (executor has NO interview context - be exhaustive):
  - API route: `appview/internal/routes/routes.go:49-58` - existing authenticated post/thread handlers.
  - Handler: `appview/internal/api/post.go:389-456` - resolves target, root URI, and descendants.
  - Current tree builder: `appview/internal/api/post.go:1013-1072` - descendants-only tree construction.
  - Response type: `appview/internal/api/post_response.go:56-67` - add `Ancestors []*PostResponse json:"ancestors"` to `ThreadResponse`.
  - Store query: `appview/internal/api/post_store.go:508-557` - current `LoadThreadCandidates` returns target + descendants only.
  - Existing tests: `appview/internal/api/post_test.go:1122-1168` - nested descendants behavior and ordering.
  - API contract: `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md:7-11` and `docs/superpowers/specs/2026-04-22-api-wire-alignment-design.md:11-13` - `/v1/`, JSON, auth/device conventions.

  **Acceptance Criteria** (agent-executable only):
  - [ ] `ThreadResponse` JSON includes `ancestors` for reply targets and `[]` for root targets.
  - [ ] Existing descendants-only tests still pass unchanged except for accepting the new `ancestors` field.
  - [ ] New Go tests cover root thread (`ancestors=[]`), reply anchor with root-to-parent ancestors, and missing parent best-effort behavior.
  - [ ] `just test` passes from repo root.

  **QA Scenarios** (MANDATORY - task incomplete without these):
  ```
  Scenario: Reply anchor includes ancestors
    Tool: Bash
    Steps: Run `just test` from repo root after adding a test where root -> replyA -> replyB and `/v1/posts/{replyB.did}/{replyB.rkey}/thread` returns ancestors `[root, replyA]`.
    Expected: Go test passes and decoded JSON has ancestors ordered root-to-parent.
    Evidence: .sisyphus/evidence/task-1-appview-ancestors.txt

  Scenario: Missing parent does not break thread
    Tool: Bash
    Steps: Run the new missing-parent Go test where target has a parent URI absent from indexed rows.
    Expected: Handler returns 200 with target post, descendants if any, and no fabricated ancestor placeholder.
    Evidence: .sisyphus/evidence/task-1-missing-parent.txt
  ```

  **Commit**: YES | Message: `feat(appview): include ancestors in post threads` | Files: [`appview/internal/api/post.go`, `appview/internal/api/post_response.go`, `appview/internal/api/post_store.go`, `appview/internal/api/post_test.go`]

- [x] 2. Update Flutter thread/reply data contracts

  **What to do**: Update Flutter models and API client to match the backend contract. `PostThread` must parse `ancestors` as `List<Post>` plus existing `post`, `replies`, and `truncated`. Add a small reply-create request shape using existing `PostRef`/`PostReply` concepts so `PostApiClient.createPost` can send optional `reply: {root: {uri,cid}, parent: {uri,cid}}`. Regenerate dart_mappable code.
  **Must NOT do**: Do not build UI in this task. Do not introduce a second model hierarchy for post refs when `PostRef` already exists.

  **Recommended Agent Profile**:
  - Category: `unspecified-high` - Reason: generated models, API client, tests.
  - Skills: [`dart-add-unit-test`] - Add focused data/API tests.
  - Omitted: [`flutter-add-widget-test`] - No widgets in this task.

  **Parallelization**: Can Parallel: NO | Wave 2 | Blocks: [5, 6] | Blocked By: [1]

  **References**:
  - Current model: `app/lib/feed/models/post_thread.dart:6-17` - add `ancestors`.
  - Post refs: `app/lib/feed/models/post.dart:75-87` - reuse `PostRef` and `PostReply` semantics.
  - API client: `app/lib/feed/data/post_api_client.dart:17-28` and `app/lib/feed/data/post_api_client.dart:58-64` - create and thread endpoints.
  - Repository contract: `app/lib/feed/data/post_repository.dart:10-29` - extend `create` signature with optional reply.
  - Production repository: `app/lib/feed/data/api_post_repository.dart:14-33` - forward optional reply.
  - API tests: `app/test/feed/data/post_api_client_test.dart:35-68` - add reply payload and ancestors parsing coverage.
  - Fake repository: `app/test/feed/fakes/fake_post_repository.dart:24-49` - update callback signature.
  - Provider tests: `app/test/feed/providers/post_thread_provider_test.dart:59-78` - update sample thread model.

  **Acceptance Criteria**:
  - [ ] `PostThreadMapper.fromMap` parses `ancestors` root-to-parent.
  - [ ] `PostApiClient.createPost(text:, reply:)` sends exactly `{text, reply: {root: {uri, cid}, parent: {uri, cid}}}` when reply is provided and omits `reply` otherwise.
  - [ ] Existing post creation tests pass.
  - [ ] `dart run build_runner build --delete-conflicting-outputs` completes from `app/`.
  - [ ] `flutter test test/feed/data test/feed/providers` passes from `app/`.

  **QA Scenarios**:
  ```
  Scenario: Thread model parses ancestors
    Tool: Bash
    Steps: Run `flutter test test/feed/models test/feed/data test/feed/providers` from `app/`.
    Expected: Test fixture with `ancestors` decodes into `PostThread.ancestors` in root-to-parent order.
    Evidence: .sisyphus/evidence/task-2-thread-model.txt

  Scenario: Reply create payload is camelCase and lexicon-shaped
    Tool: Bash
    Steps: Run `flutter test test/feed/data/post_api_client_test.dart` from `app/`.
    Expected: http_mock_adapter observes `POST /v1/posts` with `reply.root.uri`, `reply.root.cid`, `reply.parent.uri`, `reply.parent.cid` nested under camelCase `reply`.
    Evidence: .sisyphus/evidence/task-2-reply-payload.txt
  ```

  **Commit**: YES | Message: `feat(app): model post thread ancestors` | Files: [`app/lib/feed/models/post_thread.dart`, `app/lib/feed/models/post_thread.mapper.dart`, `app/lib/feed/data/post_api_client.dart`, `app/lib/feed/data/post_repository.dart`, `app/lib/feed/data/api_post_repository.dart`, `app/test/feed/**`]

- [x] 3. Add reply-aware create mutation and composer inputs

  **What to do**: Extend the existing create-post provider/composer so it can create either a top-level post or a reply. Add a `ReplyTarget` value object or equivalent helper that derives `root` and `parent` refs from the target post: if target has `post.reply != null`, use `post.reply!.root` as root; otherwise use the target post's own `{uri,cid}` as root. Parent is always the target post's own `{uri,cid}`. Add composer title/hint/submit labels for reply mode. On reply success, invalidate `postThreadProvider(target.author.did, target.rkey)` and `directRepliesProvider(target.author.did, target.rkey)`; do not implement optimistic insertion in this pass.
  **Must NOT do**: Do not duplicate the entire composer. Do not send replies directly to a PDS. Do not change top-level post behavior except for optional reply parameters.

  **Recommended Agent Profile**:
  - Category: `quick` - Reason: bounded provider/widget extension with tests.
  - Skills: [`dart-add-unit-test`, `flutter-add-widget-test`] - Provider and composer widget tests.
  - Omitted: [`flutter-add-integration-test`] - No full app automation needed for this provider-level task.

  **Parallelization**: Can Parallel: YES | Wave 1 | Blocks: [6] | Blocked By: []

  **References**:
  - Composer launcher: `app/lib/feed/widgets/post_composer_sheet.dart:10-20` - accept optional reply target.
  - Composer state/listener: `app/lib/feed/widgets/post_composer_sheet.dart:49-74` - success/error handling.
  - Submit path: `app/lib/feed/widgets/post_composer_sheet.dart:79-144` - adapt labels and call create with optional reply target.
  - Create provider: `app/lib/feed/providers/create_post_provider.dart:10-51` - extend `create` signature and invalidation behavior.
  - Existing composer tests: `app/test/feed/widgets/post_composer_sheet_test.dart` - add reply mode tests.
  - Existing create provider tests: `app/test/feed/providers/create_post_provider_test.dart` - add reply payload/invalidation coverage.

  **Acceptance Criteria**:
  - [ ] Top-level composer still sends no `reply` field.
  - [ ] Reply composer sends root/parent refs derived from target post.
  - [ ] Reply mode has distinct accessible copy: title `Reply`, hint `Write your reply`, submit `Reply`.
  - [ ] Success/error states reset the provider exactly like top-level compose.
  - [ ] `flutter test test/feed/widgets/post_composer_sheet_test.dart test/feed/providers/create_post_provider_test.dart` passes from `app/`.

  **QA Scenarios**:
  ```
  Scenario: Reply to root post builds root=parent=target
    Tool: Bash
    Steps: Run provider test where target has no `reply`; call create in reply mode.
    Expected: Fake repository receives reply root and parent both equal to target `{uri,cid}`.
    Evidence: .sisyphus/evidence/task-3-root-reply.txt

  Scenario: Reply to reply preserves original root
    Tool: Bash
    Steps: Run provider test where target has `reply.root` from another post; call create in reply mode.
    Expected: Fake repository receives root from `target.reply.root` and parent from target `{uri,cid}`.
    Evidence: .sisyphus/evidence/task-3-reply-to-reply.txt
  ```

  **Commit**: YES | Message: `feat(app): support creating replies` | Files: [`app/lib/feed/providers/create_post_provider.dart`, `app/lib/feed/providers/create_post_provider.g.dart`, `app/lib/feed/widgets/post_composer_sheet.dart`, `app/lib/l10n/**`, `app/test/feed/**`]

- [x] 4. Add thread route and navigation entry points

  **What to do**: Add a root-navigator typed route at `/posts/:did/:rkey` that builds a new `PostThreadPage(did: did, rkey: rkey)`. Add `RouteLocations.postThread = '/posts/:did/:rkey'` or equivalent typed route path. Add a reusable navigation helper or typed route call so feed/profile cards can open the thread. Update `PostCard` to support separate `onTap` for opening the thread and `onReply` for composer. Replace the profile tab's current `_loadThread` preload-only behavior with actual navigation/composer behavior. Feed cards must navigate on tap and open reply composer from the reply action.
  **Must NOT do**: Do not make the thread route a child of only the feed branch; it must be reachable from profile/search/notifications later. Do not remove existing like/repost/delete actions.

  **Recommended Agent Profile**:
  - Category: `quick` - Reason: route and callback plumbing.
  - Skills: [`flutter-add-widget-test`] - Widget/router tests.
  - Omitted: [`agent-browser`] - No browser automation required.

  **Parallelization**: Can Parallel: YES | Wave 2 | Blocks: [5] | Blocked By: [2]

  **References**:
  - Route constants: `app/lib/router/route_locations.dart:1-20` - add canonical post route path.
  - Router shell/root routes: `app/lib/router/router.dart:122-178` and `app/lib/router/router.dart:266-280` - add root-navigator typed route over shell.
  - Current feed route class: `app/lib/router/router.dart:205-209` - pattern for typed route build methods.
  - Shell behavior: `app/lib/router/app_shell.dart:52-73` - root route should cover bottom navigation for focused reading.
  - Post card callback: `app/lib/feed/widgets/post_card.dart:14-29` and `app/lib/feed/widgets/post_card.dart:80-99` - add `onTap`, keep reply action separate.
  - Existing profile usage: `app/lib/profile/widgets/profile_tabs/profile_posts_tab.dart:115-127` and `_loadThread` around `app/lib/profile/widgets/profile_tabs/profile_posts_tab.dart:182` - replace preload with navigation.
  - Router tests: `app/test/router/router_redirect_test.dart` - add signed-in route coverage.

  **Acceptance Criteria**:
  - [ ] Signed-in user can navigate to `/posts/did:plc:alice/root` without redirect.
  - [ ] Signed-out user is redirected to welcome for the same route.
  - [ ] Tapping a `PostCard` body opens the thread route.
  - [ ] Tapping the reply icon opens reply composer and does not also navigate.
  - [ ] `dart run build_runner build --delete-conflicting-outputs` completes from `app/` after route changes.
  - [ ] `flutter test test/router test/feed/widgets/post_card_test.dart` passes from `app/`.

  **QA Scenarios**:
  ```
  Scenario: Signed-in routing reaches thread page
    Tool: Bash
    Steps: Run router test that pumps signed-in state and navigates to `/posts/did:plc:alice/root`.
    Expected: `PostThreadPage` is built with did `did:plc:alice` and rkey `root`.
    Evidence: .sisyphus/evidence/task-4-route-signed-in.txt

  Scenario: Reply icon does not trigger card navigation
    Tool: Bash
    Steps: Run `post_card_test.dart` with both `onTap` and `onReply`; tap `Icons.chat_bubble_outline`.
    Expected: reply callback count increments; navigation/tap callback remains zero.
    Evidence: .sisyphus/evidence/task-4-reply-hit-test.txt
  ```

  **Commit**: YES | Message: `feat(app): route posts to thread screen` | Files: [`app/lib/router/route_locations.dart`, `app/lib/router/router.dart`, `app/lib/router/router.g.dart`, `app/lib/feed/widgets/post_card.dart`, `app/lib/feed/pages/feed_page.dart`, `app/lib/profile/widgets/profile_tabs/profile_posts_tab.dart`, `app/test/router/**`, `app/test/feed/widgets/post_card_test.dart`]

- [x] 5. Build anchored thread page and focused branch renderer

  **What to do**: Create `app/lib/feed/pages/post_thread_page.dart` plus focused widgets under `app/lib/feed/widgets/thread/`. The page watches `postThreadProvider(did, rkey)`. Loading uses `StitchProgressIndicator`. Error state uses existing retry copy/patterns. Render ancestors as a compact vertical path above anchor. Render anchor with a visually stronger `PostCard` variant or wrapper. Render direct replies below anchor. For each direct reply, do not recursively render unlimited descendants; instead show a continuation row when `reply.replies.isNotEmpty` or `reply.post.replyCount > 0`, labelled `Continue thread` for a single visible child and `Show more replies` for multiple/unknown. Tapping a reply card or continuation row navigates to that reply as the new anchor.
  **Must NOT do**: Do not render all nested replies recursively. Do not invent missing ancestor placeholders. Do not change feed card styling globally beyond reusable props required for thread rendering.

  **Recommended Agent Profile**:
  - Category: `visual-engineering` - Reason: main UX implementation with responsive, accessible visual hierarchy.
  - Skills: [`flutter-add-widget-test`, `flutter-build-responsive-layout`] - Widget coverage and responsive behavior.
  - Omitted: [`frontend-ui-ux`] - Built-in skill not needed if visual-engineering category is used.

  **Parallelization**: Can Parallel: NO | Wave 3 | Blocks: [6, 7, 8] | Blocked By: [1, 2, 4]

  **References**:
  - Provider: `app/lib/feed/providers/post_thread_provider.dart:8-21` - thread data source.
  - Thread model: `app/lib/feed/models/post_thread.dart:6-17` - update from Task 2 includes ancestors.
  - Card visual baseline: `app/lib/feed/widgets/post_card.dart:38-136` - reuse Craftsky card/action style.
  - Theme extensions: `app/lib/theme/app_theme.dart` and `app/lib/theme/form_factor.dart:14-21` - spacing, colors, form factor.
  - Existing loading indicator: `app/lib/theme/stitch_progress_indicator.dart` - loading state.
  - Bluesky inspiration: anchored thread screen with parent context, anchor, replies, and explicit expansion rows; do not copy code.

  **Acceptance Criteria**:
  - [ ] Loading, error+retry, empty replies, ancestors, anchor, replies, and truncated states have widget tests.
  - [ ] Ancestors render above anchor in root-to-parent order.
  - [ ] Direct replies render below anchor in API order.
  - [ ] Nested descendants render as continuation rows, not full recursive cards.
  - [ ] Tapping a reply card re-anchors via `/posts/{reply.author.did}/{reply.rkey}`.
  - [ ] `flutter test test/feed/pages test/feed/widgets/thread` passes from `app/`.

  **QA Scenarios**:
  ```
  Scenario: Thread screen renders focused conversation
    Tool: Bash
    Steps: Run widget test with ancestors `[root, parent]`, anchor `replyB`, and two direct replies.
    Expected: Root and parent appear before anchor; anchor appears before both replies; no nested grandchild card is rendered inline.
    Evidence: .sisyphus/evidence/task-5-focused-thread.txt

  Scenario: Continuation row re-anchors
    Tool: Bash
    Steps: Run widget test with a direct reply that has nested replies; tap `Continue thread`.
    Expected: Router receives navigation to `/posts/{directReply.author.did}/{directReply.rkey}`.
    Evidence: .sisyphus/evidence/task-5-continuation-nav.txt
  ```

  **Commit**: YES | Message: `feat(app): render focused post threads` | Files: [`app/lib/feed/pages/post_thread_page.dart`, `app/lib/feed/widgets/thread/**`, `app/test/feed/pages/**`, `app/test/feed/widgets/thread/**`]

- [x] 6. Wire responsive composer and local reply actions into thread page

  **What to do**: Add responsive composer entry points to the thread page. For `FormFactor.isLarge`, render an inline reply composer/prompt below the anchor that targets the anchor post. For small form factors, render a sticky bottom reply prompt that opens `PostComposerSheet` in reply mode for the anchor. Each ancestor/reply/anchor post card's reply icon opens the composer targeted to that specific post. After successful reply creation, invalidate the active thread provider and keep the user on the current thread.
  **Must NOT do**: Do not show both inline and sticky composer on the same form factor. Do not make reply buttons target the wrong post. Do not add optimistic insertion in this pass.

  **Recommended Agent Profile**:
  - Category: `visual-engineering` - Reason: responsive behavior and composer UX.
  - Skills: [`flutter-add-widget-test`, `flutter-build-responsive-layout`] - Required for breakpoint tests.
  - Omitted: [`dart-add-unit-test`] - Provider tests were handled in Task 3.

  **Parallelization**: Can Parallel: NO | Wave 3 | Blocks: [7, 8] | Blocked By: [2, 3, 5]

  **References**:
  - Form factors: `app/lib/theme/form_factor.dart:14-21` - `isLarge` means laptop/desktop (`width > 900`).
  - Shell form factor use: `app/lib/router/app_shell.dart:48-73` - established responsive pattern.
  - Composer launcher: `app/lib/feed/widgets/post_composer_sheet.dart:10-20` - extend with reply target.
  - Existing post card action: `app/lib/feed/widgets/post_card.dart:92-99` - reply icon callback.
  - Create provider invalidation from Task 3: `app/lib/feed/providers/create_post_provider.dart`.

  **Acceptance Criteria**:
  - [ ] At width 390, thread page shows sticky bottom reply prompt and no inline composer.
  - [ ] At width 1024, thread page shows inline anchor composer/prompt and no sticky bottom prompt.
  - [ ] Reply icon on an ancestor targets that ancestor.
  - [ ] Reply icon on direct reply targets that direct reply.
  - [ ] Successful reply invalidates current `postThreadProvider`.
  - [ ] `flutter test test/feed/pages/post_thread_page_test.dart test/feed/widgets/post_composer_sheet_test.dart` passes from `app/`.

  **QA Scenarios**:
  ```
  Scenario: Mobile sticky prompt only
    Tool: Bash
    Steps: Run widget test with `MediaQuery` width 390 and find text `Reply` in the sticky bottom area.
    Expected: Sticky prompt exists; inline composer/prompt below anchor does not exist.
    Evidence: .sisyphus/evidence/task-6-mobile-composer.txt

  Scenario: Desktop inline prompt only
    Tool: Bash
    Steps: Run widget test with `MediaQuery` width 1024 and find inline anchor reply prompt below the anchor card.
    Expected: Inline prompt exists; sticky bottom prompt does not exist.
    Evidence: .sisyphus/evidence/task-6-desktop-composer.txt
  ```

  **Commit**: YES | Message: `feat(app): add responsive thread reply composer` | Files: [`app/lib/feed/pages/post_thread_page.dart`, `app/lib/feed/widgets/thread/**`, `app/lib/feed/widgets/post_composer_sheet.dart`, `app/test/feed/**`]

- [x] 7. Add localization, accessibility, and Craftsky visual polish

  **What to do**: Add generated localization strings for thread page title, retry, empty replies, reply labels, `Continue thread`, `Show more replies`, and reply composer copy. Add semantics labels for continuation rows and reply buttons that include target author/display name where available. Ensure thread connector/indent visuals use Craftsky theme colors and spacing, not hardcoded Bluesky styling. Verify text scale and narrow widths do not overflow.
  **Must NOT do**: Do not use hardcoded user-facing English in widgets after l10n is added. Do not copy Bluesky colors/visual identity.

  **Recommended Agent Profile**:
  - Category: `quick` - Reason: polish/localization/test pass.
  - Skills: [`flutter-add-widget-test`] - Accessibility/widget assertions.
  - Omitted: [`avoid-ai-writing`] - UI copy is short product text, not prose.

  **Parallelization**: Can Parallel: YES | Wave 3 | Blocks: [8] | Blocked By: [5, 6]

  **References**:
  - Existing localization delegates in tests: `app/test/feed/widgets/post_card_test.dart:38-45`.
  - Composer l10n usage: `app/lib/feed/widgets/post_composer_sheet.dart:51-58` and `app/lib/feed/widgets/post_composer_sheet.dart:79-144`.
  - Theme card/divider patterns: `app/lib/feed/widgets/post_card.dart:4-8` and `app/lib/feed/widgets/post_card.dart:38-136`.
  - Brand/form factor theme: `app/lib/theme/app_theme.dart`, `app/lib/theme/form_factor.dart:14-21`.

  **Acceptance Criteria**:
  - [ ] No new thread/composer user-facing strings are hardcoded in Dart widgets.
  - [ ] Semantics tests cover reply action and continuation rows.
  - [ ] Text scale 2.0 widget test does not overflow in the thread screen.
  - [ ] `flutter test test/feed/pages test/feed/widgets` passes from `app/`.
  - [ ] `flutter analyze` passes from `app/`.

  **QA Scenarios**:
  ```
  Scenario: Continuation row is accessible
    Tool: Bash
    Steps: Run semantics widget test for a continuation row with target author `Alice`.
    Expected: Semantics label includes action and context, e.g. `Continue thread from Alice` or localized equivalent.
    Evidence: .sisyphus/evidence/task-7-semantics.txt

  Scenario: Large text does not overflow
    Tool: Bash
    Steps: Run thread page widget test with `MediaQuery.textScaler` equivalent of 2.0 at width 390.
    Expected: Test completes without Flutter overflow exceptions.
    Evidence: .sisyphus/evidence/task-7-large-text.txt
  ```

  **Commit**: YES | Message: `chore(app): polish thread accessibility copy` | Files: [`app/lib/l10n/**`, `app/lib/feed/pages/post_thread_page.dart`, `app/lib/feed/widgets/thread/**`, `app/test/feed/**`]

- [x] 8. Run integrated verification and fix regressions

  **What to do**: Run the full agreed verification set after all implementation tasks. Fix any failures in the smallest relevant files. Confirm generated files are up to date. Capture evidence logs. Verify the plan's architectural guardrails: Flutter reads from AppView only, reply writes go through AppView only, no lexicon change, no recursive full-thread sprawl.
  **Must NOT do**: Do not skip failing tests. Do not mark final verification complete without user approval after F1-F4.

  **Recommended Agent Profile**:
  - Category: `unspecified-high` - Reason: cross-stack verification and regression cleanup.
  - Skills: [] - Commands and focused fixes only.
  - Omitted: [`git-master`] - Commit handling is already explicit per task; no history surgery required.

  **Parallelization**: Can Parallel: NO | Wave 4 | Blocks: [] | Blocked By: [1, 2, 3, 4, 5, 6, 7]

  **References**:
  - Dev workflow: `AGENTS.md` Dev Workflow section - `just test` against compose Postgres; appview runs in Docker for dev.
  - Flutter dependencies: `app/pubspec.yaml:37-50` - build runner, go_router_builder, flutter_test, analysis.
  - Architecture rules: `AGENTS.md` Architectural Rules 1-3 - AppView reads, AppView-mediated writes, no PDS tokens in Flutter.

  **Acceptance Criteria**:
  - [ ] `just test` passes from repo root.
  - [ ] `dart run build_runner build --delete-conflicting-outputs` produces no unstaged generated drift after commit.
  - [ ] `flutter test` passes from `app/`.
  - [ ] `flutter analyze` passes from `app/`.
  - [ ] Evidence files exist for Tasks 1-8.
  - [ ] No files under `lexicon/` changed.

  **QA Scenarios**:
  ```
  Scenario: Full cross-stack verification
    Tool: Bash
    Steps: Run `just test` from repo root, then from `app/` run `dart run build_runner build --delete-conflicting-outputs`, `flutter test`, and `flutter analyze`.
    Expected: All commands exit 0.
    Evidence: .sisyphus/evidence/task-8-full-verification.txt

  Scenario: Architecture guardrail check
    Tool: Bash
    Steps: Inspect final diff and verify no `lexicon/` changes, no Flutter PDS client write path, and reply creation still calls `/v1/posts` through `PostApiClient`.
    Expected: Guardrails satisfied; any violations are fixed before final review.
    Evidence: .sisyphus/evidence/task-8-guardrails.txt
  ```

  **Commit**: YES | Message: `test: verify threaded replies implementation` | Files: [only files needed for final fixes/evidence if repo tracks evidence]

## Final Verification Wave (MANDATORY — after ALL implementation tasks)
> 4 review agents run in PARALLEL. ALL must APPROVE. Present consolidated results to user and get explicit "okay" before completing.
> **Do NOT auto-proceed after verification. Wait for user's explicit approval before marking work complete.**
> **Never mark F1-F4 as checked before getting user's okay.** Rejection or user feedback -> fix -> re-run -> present again -> wait for okay.
- [x] F1. Plan Compliance Audit — oracle
- [x] F2. Code Quality Review — unspecified-high
- [x] F3. Real Manual QA — unspecified-high (+ interactive Flutter app run if available)
- [x] F4. Scope Fidelity Check — deep

## Commit Strategy
- Use one focused commit per TODO task when possible.
- Conventional commit messages are specified on each task.
- Do not commit `.env`, credentials, local IDE state, or unrelated files.
- Generated Flutter files from `go_router_builder`, `riverpod_generator`, `dart_mappable_builder`, and l10n must be committed with their source changes.

## Success Criteria
- Thread route `/posts/:did/:rkey` works for signed-in users and redirects signed-out users.
- Thread screen shows ancestor context above anchor and direct replies below anchor.
- Deeper branches are collapsed behind clear continuation rows.
- Reply composer targets the correct post on mobile and desktop.
- Reply creation payload includes correct root/parent strong refs.
- Backend thread response supports ancestors without breaking existing descendants behavior.
- Full Go and Flutter verification passes.
