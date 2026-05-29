# Requirements: Timeline Feed Flutter

## 1. Initial Request

The AppView timeline endpoint has just been implemented in `docs/changes/2026-05-28-timeline-feed-appview/`. Move on to the Flutter side so the app consumes the home timeline through the AppView and replaces the placeholder feed screen with the user-facing timeline experience.

## 2. Current Codebase Findings

- Relevant files:
  - `docs/changes/2026-05-28-timeline-feed-appview/01-requirements.md` defines `GET /v1/feed/timeline?limit=<n>&cursor=<opaque>` returning `{items: PostResponse[], cursor?: string}` in AppView index order, with no total count and no empty-state suggestions.
  - `docs/changes/2026-05-28-timeline-feed-appview/05-implementation-plan.md` records that timeline pagination uses default `20` and max `50`, and that the endpoint reuses the existing post response shape.
  - `docs/roadmap.md` lists Flutter `Feed screen (timeline consumption + pagination)` as the open app-side task.
  - `app/lib/feed/pages/feed_page.dart` is currently a placeholder `ConsumerWidget` that renders only the localized title.
  - `app/lib/feed/data/post_api_client.dart`, `post_repository.dart`, and `api_post_repository.dart` already expose post CRUD, profile post/comment lists, comments, likes, and reposts through AppView-backed methods.
  - `app/lib/feed/models/post.dart` and `post_page.dart` already model the AppView post response and paginated `{items, cursor}` shape used by profile lists.
  - `app/lib/feed/providers/user_posts_provider.dart` and `app/lib/profile/widgets/profile_tabs/profile_posts_tab.dart` provide an existing Riverpod cursor-accumulation and paginated `PostCard` list pattern.
  - `app/lib/feed/widgets/post_card.dart` renders post text, images, author identity, relative time, engagement counts, like/comment/repost actions, and optional delete menu.
  - `app/lib/feed/widgets/post_composer_sheet.dart` and `app/lib/feed/providers/create_post_provider.dart` already support top-level post creation, reply creation, and optimistic insertion into live profile post caches.
  - `app/lib/router/router.dart` already routes the feed branch to `FeedPage` and routes post cards to `PostThreadPage` via `PostThreadRoute`.
  - `app/lib/l10n/app_en.arb` has `feedTitle` but no feed-specific loading, empty, or error copy beyond generic `retryButton` and `loading`.
- Existing patterns:
  - Flutter reads Craftsky social data from AppView through `Dio` clients and repositories; it must not read timeline data directly from PDSes.
  - Auth/device headers are installed centrally on the shared `Dio` stack; endpoint-specific clients only call `/v1/*` paths and parse responses.
  - Riverpod generated providers are used for async state, mutations, and repository bindings.
  - Paginated profile lists preserve visible data during load-more errors via `AsyncLoading().copyWithPrevious(state)` and retry with the same cursor.
  - Sliver-based UI is used for long, paginated post lists, and `PostCard` is the reusable post row component.
  - New provider/model/l10n changes usually require generated Dart output through `build_runner` and Flutter localization generation.
- Current behavior:
  - The bottom navigation has a Feed tab, but `FeedPage` is not connected to AppView timeline data.
  - Users can create, view, delete, like, repost, and comment on posts through existing screens/providers, especially profile tabs and thread pages.
  - Profile tabs can list and paginate a single author's posts/comments, but no Flutter provider or repository method consumes the authenticated home timeline.
  - Top-level create optimistically prepends to live `userPostsProvider` entries but has no timeline cache to update.
- Constraints discovered:
  - This change is Flutter-side only; AppView endpoint behavior, lexicons, database migrations, and Go code are out of scope.
  - The Flutter client should consume the AppView timeline endpoint and preserve the existing AppView read architecture.
  - The timeline response uses the existing post-shaped wire contract, so a new feed-item envelope is not required for this slice.
  - Reposts as separate feed reasons, quote-card expansion, project-specific fields, search/list/project feeds, moderation filters, and onboarding/discovery suggestions remain out of scope unless already present in the returned post shape.
- Test/build commands discovered:
  - Focused Flutter tests run from `app/`, for example `flutter test test/feed/data/post_api_client_test.dart` and `flutter test test/feed/feed_page_test.dart`.
  - Provider/widget focused tests are already present under `app/test/feed/**` and `app/test/profile/widgets/**`.
  - Full Flutter suite: `flutter test` from `app/`.
  - Generated code after provider/model/router changes: `dart run build_runner build --delete-conflicting-outputs` from `app/`.
  - Flutter analysis is available via `flutter analyze` from `app/`, with prior docs noting existing info-level findings may exist.

## 3. Clarifying Questions And Decisions

### Q1: Should the Flutter timeline requirements include the recommended scope: reusable timeline API/provider + paginated FeedPage + top-level compose entry with optimistic prepend into the live timeline cache?

Answer: Recommended scope.

Decision / implication: Requirements cover the app-side timeline API/repository/provider, replacement of the placeholder Feed tab with a paginated timeline, a top-level compose entry on the feed, and optimistic insertion of newly created top-level posts into the live timeline cache. Generic feed framework work and read-only-only scope are not selected.

## 4. Candidate Approaches

### Option A: Reuse Existing Post Stack And Build Home Timeline Screen (Recommended)

Summary: Add AppView timeline consumption to the existing post API/repository layer, introduce a Riverpod timeline provider/state, and replace `FeedPage` with a paginated `PostCard` list that includes top-level compose and optimistic prepend behavior.

Pros:
- Delivers the user-facing home feed that the AppView endpoint was built for.
- Reuses existing `Post`, `PostPage`, `PostCard`, composer, interaction providers, thread route, and pagination patterns.
- Keeps response parsing aligned with the AppView post-shaped timeline contract.
- Keeps future feed variants possible without building speculative abstractions now.

Cons:
- Larger than a read-only client slice because it includes top-level compose integration and timeline cache updates.
- Requires careful cache coordination so synthetic create responses do not duplicate later indexed timeline rows.

Risks:
- Feed-page UI can regress existing post-card actions if the implementation forks profile tab behavior instead of extracting/reusing patterns where appropriate.

### Option B: Read-Only Timeline Consumption Only

Summary: Add the timeline API/repository/provider and render paginated timeline rows, but leave feed composition and optimistic insertion for a later change.

Pros:
- Smallest app-side slice.
- Maps directly to `GET /v1/feed/timeline` without new write/cache behavior.

Cons:
- The Feed tab remains less useful as the primary home surface.
- The known post-create/read-after-indexing gap is deferred despite already having a profile-cache pattern.

Risks:
- A later compose integration could require reworking the timeline provider state and tests soon after this slice.

### Option C: Generic Feed Framework Now

Summary: Introduce generalized feed-source abstractions that can support home timeline, profile posts/comments, future project feeds, list/custom feeds, and search results in one pass.

Pros:
- Makes future feed surfaces a first-class concern.
- Could reduce duplication across timeline and profile-list providers later.

Cons:
- Larger design surface than the current product request.
- Future project/search/list feed semantics are not specified enough to justify a framework now.
- Risks abstracting around the wrong differences.

Risks:
- Over-generalization could slow delivery of the basic Feed tab and make tests harder to understand.

## 5. Recommended Direction

Recommended approach: Option A — Reuse Existing Post Stack And Build Home Timeline Screen.

Why: The Flutter app already has nearly all rendering, post model, interaction, composer, repository, and pagination building blocks needed for the timeline. The AppView endpoint intentionally returns the existing post-shaped list contract, so the safest and fastest client path is to add one timeline read path and a dedicated timeline provider while reusing proven `PostCard` and profile-list interaction patterns. The confirmed scope includes feed composition and optimistic prepend because the Feed tab should become the primary home timeline surface, not just a passive read endpoint demo.

## 6. Problem / Opportunity

Craftsky now has an AppView home timeline endpoint, but the Flutter Feed tab does not consume it. Signed-in users land on a placeholder instead of a chronological home feed of their own and followed-account posts. Wiring the Flutter side completes the first usable home-feed loop and gives users a central place to read, create, and interact with posts while preserving the AppView read architecture.

## 7. Goals

- G-001: Replace the placeholder Feed tab with a paginated home timeline backed by `GET /v1/feed/timeline`.
- G-002: Reuse existing post models, list envelope, post-card UI, interaction providers, and thread navigation wherever practical.
- G-003: Provide clear loading, empty, initial-error, and load-more-error states for the timeline.
- G-004: Add a top-level compose entry on the Feed tab and optimistically show newly created top-level posts in the live timeline cache.
- G-005: Keep this change limited to Flutter timeline consumption and UI behavior, without changing AppView endpoint semantics or introducing speculative generic feed frameworks.

## 8. Non-Goals

- NG-001: Do not change AppView route behavior, response shape, pagination semantics, database queries, migrations, or Go code.
- NG-002: Do not read timeline posts directly from PDSes or store PDS tokens on the Flutter device.
- NG-003: Do not add a generic multi-source feed framework for project feeds, custom/list feeds, search, or discovery feeds in this chunk.
- NG-004: Do not add reposts as separate feed items, feed reasons, or repost attribution cards beyond the existing `Post.viewerHasReposted` and `repostCount` fields.
- NG-005: Do not expand quoted posts into nested quote cards; render only the fields already supported by the existing `Post`/`PostCard` stack.
- NG-006: Do not add craft/project-specific timeline filters, hashtag search, ranking, algorithmic recommendations, or onboarding/discovery suggestions.
- NG-007: Do not implement offline-first storage, durable local timeline caching, background refresh, push updates, or analytics events in this slice.
- NG-008: Do not change lexicons, generated AppView lexicon types, or Flutter dependencies.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Signed-in viewer | Authenticated Craftsky user using the Feed tab. | See a chronological home timeline of their own and followed-account posts, paginate it, and interact with posts. |
| Posting viewer | Signed-in viewer creating a new top-level post from the Feed tab. | See the created post appear immediately in the live feed while AppView indexing catches up. |
| Followed author | User followed by the viewer whose indexed posts appear in the timeline. | Have their eligible posts shown using the same post card and interaction affordances as elsewhere. |
| Test designer | Next workflow agent writing `02-acceptance-tests.md`. | Receive stable requirement IDs, edge cases, and test-level handoff notes for Flutter API/provider/widget coverage. |

## 10. Current Behavior

The app's Feed tab is routed and localized but remains a placeholder. The app can render post cards in profile and thread contexts, can page profile post/comment lists, and can create, delete, like, repost, and comment on posts through existing providers. There is no Flutter API client method, repository method, timeline provider/state, or Feed page UI that calls `GET /v1/feed/timeline`.

## 11. Desired Behavior

When a signed-in user opens the Feed tab, Flutter requests the authenticated AppView timeline through the existing API stack, parses the response as the existing `PostPage`/`Post` shape, and renders a lazily paginated chronological list of `PostCard` rows. The screen handles initial loading, empty timeline, initial load failure, load-more failure, retry, and end-of-list states. Users can navigate from timeline posts to thread pages, comment/reply through the existing composer flow, like and repost posts, and delete their own posts where the existing permissions and delete provider allow. The Feed tab also exposes a top-level compose entry; when creation succeeds, the new top-level post is inserted at the head of any live timeline state without duplicating the same post when later fetched from AppView.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | Craftsky must provide a usable Flutter home timeline backed by the AppView timeline endpoint. | The AppView timeline work is only useful to users once the Feed tab consumes it. | Prompt, roadmap, AppView timeline docs | AC-001, AC-004, AC-005 |
| BR-002 | Business | Must | The Flutter timeline design must preserve room for future feed variants without introducing a speculative generic framework now. | The prior AppView requirements explicitly protected future project/list/search feed variants while keeping the current slice simple. | AppView timeline requirements, discovery | AC-013 |
| FR-001 | Functional | Must | `PostApiClient` shall expose a timeline read method that calls `GET /v1/feed/timeline` with optional `limit` and `cursor` query parameters and parses the response as `PostPage`. | Gives the app data layer access to the new endpoint using existing AppView conventions. | AppView timeline contract, codebase | AC-001, AC-002 |
| FR-002 | Functional | Must | `PostRepository` and the production `ApiPostRepository` shall expose a home-timeline list method that does not require a handle or DID route parameter. | Timeline scope is the authenticated viewer, not an arbitrary profile. | AppView timeline requirements, codebase | AC-002 |
| FR-003 | Functional | Must | Flutter shall maintain timeline state as an async, cursor-accumulating list of `Post` items with a derived “has more” state. | Matches the existing profile-list state pattern and supports infinite scrolling. | Existing `UserPosts` pattern | AC-003, AC-006, AC-007 |
| FR-004 | Functional | Must | The first timeline load shall request the first page from the repository using a bounded page size compatible with the AppView timeline endpoint. | Prevents unbounded loads and aligns client pagination with server limits. | AppView implementation plan, existing list patterns | AC-003 |
| FR-005 | Functional | Must | Loading the next timeline page shall pass the opaque cursor returned by the previous page and append the returned items without inspecting or decoding the cursor. | Preserves API cursor semantics and avoids client coupling to server internals. | API architecture spec, AppView timeline requirements | AC-006, AC-014 |
| FR-006 | Functional | Must | If an initial timeline load fails, `FeedPage` shall show a feed-specific error state with a retry action that reattempts the first page. | Users need a recoverable state when the home feed cannot load. | Existing profile error patterns, discovery | AC-008 |
| FR-007 | Functional | Must | If a load-more request fails after items are visible, the timeline shall keep the existing visible items and cursor available for retry. | Avoids blanking the feed after pagination errors and matches existing profile-list behavior. | Existing `UserPosts` pattern | AC-007 |
| FR-008 | Functional | Should | An empty successful timeline should render a clear empty-feed state and must not embed onboarding, discovery suggestions, or recommendation content in this slice. | Empty timelines are normal; suggestions are explicitly out of scope from the AppView slice. | AppView timeline requirements, discovery | AC-009 |
| FR-009 | Functional | Must | `FeedPage` shall render timeline posts using the existing `PostCard` component or an extracted equivalent that preserves current post-card behavior for text, images, author display, timestamps, engagement counts, and viewer engagement state. | Reuses the established post UI and avoids a second post rendering contract. | Codebase findings | AC-004, AC-013 |
| FR-010 | Functional | Must | Tapping a timeline post shall navigate to the existing thread route for that post using the post author's DID and record key. | Timeline rows need the same detail navigation as profile post rows. | Existing router/profile patterns | AC-010 |
| FR-011 | Functional | Must | Timeline post like and repost actions shall use the existing interaction providers and update the visible timeline row after success. | Keeps engagement actions consistent across feed and profile contexts. | Existing provider patterns | AC-011 |
| FR-012 | Functional | Must | Timeline comment/reply actions shall use the existing composer/thread flow and update the visible timeline row's reply-related viewer/count state when a reply is created from the timeline. | Users should be able to comment from the home feed without losing the thread-focused workflow. | Existing profile post tab and thread patterns | AC-012 |
| FR-013 | Functional | Should | Timeline rows for posts authored by the signed-in viewer should expose the existing delete action; non-owned rows should not expose delete. Successful deletion should remove the row from the live timeline state. | Home timeline includes the viewer's own posts and should respect existing delete affordances and permissions. | Existing profile delete pattern, AppView timeline self-inclusion | AC-015 |
| FR-014 | Functional | Must | The Feed tab shall expose a top-level compose entry that uses the existing post composer to create top-level posts. | Confirmed scope includes feed composition as part of making the Feed tab useful. | Q1 user answer | AC-016 |
| FR-015 | Functional | Must | When top-level post creation from any live app surface succeeds, the timeline cache shall optimistically prepend the created top-level post to any live timeline state, deduplicated by stable post URI. | AppView timeline reads only show indexed rows; optimistic client insertion bridges the indexing lag without duplicate rows. | Q1 user answer, existing create provider pattern, AppView Q11 | AC-017, AC-018 |
| NFR-001 | Non-functional | Must | Timeline reads must use the existing AppView `Dio` stack with Craftsky session and device headers; the Flutter app must not fetch timeline craft data directly from PDSes. | Preserves the project architecture and token-boundary rules. | AGENTS.md, API architecture spec | AC-001 |
| NFR-002 | Non-functional | Must | Timeline code must reuse existing post-shaped models and list envelope conventions rather than introducing a separate feed-item wire model in this slice. | The AppView timeline returns existing post responses and no feed reasons. | AppView timeline requirements | AC-002, AC-013 |
| NFR-003 | Non-functional | Should | The feed UI should remain lazy and bounded, avoiding eager rendering or fetching of all timeline pages. | Protects app performance for long feeds. | Existing sliver/infinite-list patterns | AC-006, AC-019 |
| NFR-004 | Non-functional | Must | New user-facing feed copy shall be localized through the existing Flutter localization pipeline, and loading/retry controls shall remain accessible. | Maintains app i18n/accessibility conventions. | Codebase findings | AC-020 |
| NFR-005 | Non-functional | Must | Any new Riverpod providers, mappable classes, or localization outputs introduced by the implementation shall have generated files updated consistently. | Prevents build/test drift in a generated-code-heavy Flutter app. | Codebase findings | AC-021 |
| RULE-001 | Business rule | Must | The Flutter home timeline is always scoped to the authenticated viewer implied by the current Craftsky session; the client shall not send another user's DID/handle to request a different home timeline. | The AppView endpoint has no viewer path parameter and is auth-scoped. | AppView timeline requirements | AC-001, AC-002 |
| RULE-002 | Business rule | Must | Timeline cursors are opaque client tokens; Flutter may store and pass them back but must not parse, mutate, or derive ordering from them. | Maintains server-owned pagination semantics. | API architecture spec | AC-006, AC-014 |
| RULE-003 | Business rule | Must | Only top-level created posts are optimistically prepended to the home timeline; reply/comment creations are not inserted as separate timeline rows. | AppView timeline excludes comments/replies as rows, while replies update thread/comment state. | AppView timeline requirements, Q1 | AC-012, AC-017 |
| RULE-004 | Business rule | Must | A post URI may appear at most once in the live timeline state, including across optimistic inserts and later server-fetched pages. | Prevents duplicated rows when a synthetic create response later appears through AppView indexing. | Existing profile cache pattern, Q1 | AC-018 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-001, NFR-001, RULE-001 | Given an authenticated app session with the shared `Dio` client configured, when the timeline API method is called without a cursor, then Flutter requests `GET /v1/feed/timeline` through the AppView stack and does not call any PDS endpoint. |
| AC-002 | FR-001, FR-002, NFR-002, RULE-001 | Given AppView returns `{items, cursor}` from `/v1/feed/timeline`, when the API/repository methods complete, then callers receive a `PostPage` containing existing `Post` items and the optional next cursor, with no handle or DID argument required. |
| AC-003 | FR-003, FR-004 | Given the timeline provider is first watched, when the first page succeeds, then state contains the returned posts, stores the returned cursor, and reports whether another page is available. |
| AC-004 | BR-001, FR-009 | Given timeline items include text, images, author display fields, timestamps, and engagement fields, when `FeedPage` renders the loaded timeline, then each row displays through the existing post-card behavior rather than a placeholder feed title. |
| AC-005 | BR-001 | Given the user selects the Feed tab after sign-in, when the first page is loading, then the page shows a loading state for timeline content rather than the old static placeholder body. |
| AC-006 | FR-003, FR-005, NFR-003, RULE-002 | Given a loaded timeline state has a non-null cursor, when the user scrolls near the end or otherwise triggers loading more, then Flutter requests the next page with that exact cursor and appends the returned items to the visible list. |
| AC-007 | FR-003, FR-007 | Given visible timeline items and a non-null cursor, when the next-page request fails, then visible items remain on screen, a retry affordance is shown, and retry uses the same cursor. |
| AC-008 | FR-006 | Given the first timeline request fails, when `FeedPage` renders the error state and the user taps retry, then the first page request is attempted again. |
| AC-009 | FR-008 | Given AppView returns `200` with `items: []` and no cursor, when `FeedPage` renders, then the page shows a clear empty-feed message and no onboarding, discovery, or recommendation cards. |
| AC-010 | FR-010 | Given a timeline row is visible, when the user taps the row, then Flutter navigates to the existing post thread route using the row post author's DID and rkey. |
| AC-011 | FR-011 | Given a timeline row is visible, when the user successfully likes/unlikes or reposts/unreposts it through existing actions, then the visible row reflects the updated viewer engagement/count state. |
| AC-012 | FR-012, RULE-003 | Given a timeline row is visible, when the user creates a reply/comment from that row, then the existing composer/thread flow is used, the user is taken to the relevant thread/focus when appropriate, and the timeline row's reply state is updated without inserting the reply as its own timeline row. |
| AC-013 | BR-002, FR-009, NFR-002 | Given the implementation is reviewed, when timeline rendering and state are inspected, then it reuses the existing post/list contracts and does not introduce a generic feed-item envelope or future-feed framework. |
| AC-014 | FR-005, RULE-002 | Given a non-empty cursor value from AppView, when Flutter stores and sends it for pagination, then the cursor is treated as an opaque string and not parsed or modified by client code. |
| AC-015 | FR-013 | Given a timeline includes one post authored by the signed-in viewer and one by another author, when rows render, then only the viewer-authored row exposes the delete action; after a successful delete, that row is removed from the live timeline list. |
| AC-016 | FR-014 | Given the Feed tab is loaded, when the user activates the top-level compose entry and submits a valid post, then the existing top-level composer flow creates the post. |
| AC-017 | FR-015, RULE-003 | Given a top-level post creation succeeds while timeline state is live, when the create provider receives the created post, then the post appears at the head of the visible timeline without waiting for a timeline refetch. |
| AC-018 | FR-015, RULE-004 | Given a post was optimistically prepended and later appears in a fetched timeline page or refresh with the same URI, when timeline state is merged, then only one row for that URI remains visible. |
| AC-019 | NFR-003 | Given a timeline with more than one page of data, when the page renders and paginates, then it uses lazy list/sliver-style rendering and bounded page requests instead of fetching or building all pages at once. |
| AC-020 | NFR-004 | Given new feed empty/error/retry/loading labels are visible or exposed to assistive technologies, when localization is generated, then the labels come from the existing localization pipeline and interactive retry/compose controls have accessible text or semantics. |
| AC-021 | NFR-005 | Given new Riverpod providers, mappable state, or localizations are introduced, when generated-code checks/tests run, then required generated Dart files are present and consistent with the source declarations. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | First page returns no items and no cursor. | Show the empty-feed state and do not attempt load-more. | FR-008 |
| EC-002 | First page returns items and no cursor. | Show items and no bottom load-more/progress affordance. | FR-003, FR-005 |
| EC-003 | First page returns items and a cursor. | Show items and trigger/request the next page only when the user nears the end or explicitly retries/loads more. | FR-003, FR-005 |
| EC-004 | Load-more request fails. | Preserve existing items and cursor; show retry without resetting to initial-error state. | FR-007 |
| EC-005 | Initial request fails. | Show full feed error with retry; no stale placeholder text should appear as the primary body. | FR-006 |
| EC-006 | AppView returns an invalid or unexpected post item shape. | Surface the error through the same API/error handling path as other post list parsing failures. | FR-001, FR-006 |
| EC-007 | Timeline includes a quote post. | Render using existing `Post`/`PostCard` capabilities; do not require nested quote-card expansion in this slice. | NFR-002 |
| EC-008 | Timeline includes image posts. | Render image carousel/gallery behavior consistently with existing `PostCard` tests. | FR-009 |
| EC-009 | User creates a top-level post while timeline is not live. | No new timeline provider instance is created solely for optimistic insertion; the post appears when the timeline is later loaded/refetched by AppView or normal provider lifecycle. | FR-015 |
| EC-010 | User creates a reply from a timeline row. | Reply updates row/thread behavior but is not inserted as a top-level timeline item. | FR-012, RULE-003 |
| EC-011 | Optimistically inserted post later appears in server response. | Dedupe by URI; do not show duplicate rows. | FR-015, RULE-004 |
| EC-012 | User taps like/repost rapidly or while a mutation is in flight. | Use existing mutation-provider safeguards/behavior; timeline should not invent a separate interaction model. | FR-011 |
| EC-013 | Signed-in viewer changes due to sign-out/sign-in. | Existing auth/router/provider invalidation should prevent one user's timeline state from being shown as another user's home timeline. | RULE-001, NFR-001 |
| EC-014 | Delete action is considered for non-owned post. | Do not expose delete callback/menu item for non-owned rows. | FR-013 |

## 15. Data / Persistence Impact

- New fields: No durable app persistence fields required.
- Changed fields: Existing `Post` and `PostPage` wire models are expected to be reused unchanged unless implementation discovers a missing field already present in the AppView response.
- Migration required: None.
- Backwards compatibility: Additive Flutter data-layer and UI behavior. Existing profile post/comment lists, post cards, composer, thread view, and interaction providers should remain compatible.
- Local cache: Only in-memory Riverpod provider state is in scope. No SQLite, shared preferences, or durable timeline cache is required.

## 16. UI / API / CLI Impact

- UI: Replaces the placeholder Feed tab body with a real timeline; adds feed-specific loading, empty, error, retry, pagination, top-level compose, and post interaction behavior.
- API: Flutter adds client/repository support for `GET /v1/feed/timeline`; no AppView API contract changes.
- CLI: None.
- Background jobs: None.
- Localization: New feed-specific strings are expected for empty/error and possibly compose affordance copy if existing strings are insufficient.

## 17. Security / Privacy / Permissions

- Authentication: Timeline requests use the existing signed-in Craftsky session and device ID headers through the shared AppView `Dio` stack.
- Authorization: The home timeline is scoped by the authenticated viewer on the server; Flutter does not provide a viewer DID/handle query or path parameter.
- Sensitive data: Flutter must not access or store PDS OAuth tokens for timeline reads. No private-by-intent data is introduced.
- Abuse cases: Client-side UI does not implement blocks, mutes, reports, moderation labels, rate limiting, or recommendations in this slice. It should render only what AppView returns and remain compatible with future server-side filtering.

## 18. Observability

- Events: No product analytics events are required in this requirements slice.
- Logs: No new app logs are required. If implementation logs timeline errors, logs must not include bearer tokens or device secrets.
- Metrics: None required.
- Alerts: None required.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Timeline UI duplicates profile-list pagination logic instead of reusing/extracting common patterns. | Higher maintenance cost and inconsistent behavior across profile and feed lists. | Requirements call for reuse of existing post/list conventions; test design should include regression coverage for profile lists. |
| RISK-002 | Optimistic create insertion duplicates rows after AppView indexing catches up. | Users see the same post twice in the feed. | Require URI-level deduplication when prepending and merging fetched pages. |
| RISK-003 | Feed interactions update profile caches but not timeline cache, or vice versa. | Visible engagement state diverges across screens. | Test timeline interaction updates directly and consider shared cache-update helpers during implementation. |
| RISK-004 | Load-more errors accidentally blank the whole feed. | Poor user experience on intermittent network failures. | Preserve previous timeline state during load-more errors, matching existing `UserPosts` behavior. |
| RISK-005 | Existing `PostCard` lacks some desired home-feed-specific presentation. | The first feed may feel basic. | Keep polish beyond existing post-card behavior out of scope; document as future UI polish if needed. |
| RISK-006 | Generated files drift after adding providers/localizations. | Build or tests fail in later workflow stages. | Require generated-code updates and test-design coverage for generated outputs. |
| RISK-007 | Feed page becomes a place for discovery/recommendation scope creep. | Delays the basic chronological feed and conflicts with prior AppView non-goals. | Explicitly exclude recommendations, ranking, search, and onboarding suggestions in this slice. |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | The AppView timeline endpoint from `2026-05-28-timeline-feed-appview` is available to the Flutter app with the documented `PostPage` shape. | Flutter requirements would need to change if the endpoint path or response shape changed. |
| ASM-002 | Existing `Post`, `PostPage`, and `PostCard` are sufficient to render timeline rows for this slice. | Additional model/UI requirements would be needed for quote/project-specific presentation. |
| ASM-003 | A bounded timeline page size of about the AppView default (`20`) is acceptable for the Flutter Feed tab. | Requirements/tests would need to specify a different client limit or rely entirely on server defaults. |
| ASM-004 | The existing top-level composer can be reused from the Feed tab without a separate feed-specific composer design. | Composer UX requirements would need expansion. |
| ASM-005 | Existing auth routing prevents `FeedPage` from being used by signed-out users in normal app flow. | Feed page would need explicit signed-out handling requirements. |
| ASM-006 | Current profile-list pagination behavior is an acceptable model for timeline pagination UX. | Timeline-specific scroll/load behavior would need separate product decisions. |

## 21. Open Questions

- [ ] Non-blocking: Should future timeline rows show explicit feed reasons, such as “reposted by” or “followed because of list,” once repost/list/custom feeds are designed?
- [ ] Non-blocking: Should quote posts eventually render nested quote cards in the feed, or remain strong-reference-only until quote-post UX is separately specified?
- [ ] Non-blocking: Should empty Feed eventually include follow suggestions or onboarding content once discovery/onboarding work is scoped?
- [ ] Non-blocking: Should timeline state eventually be shared with profile post caches through a normalized post cache, or is targeted provider cache updating sufficient for v1?

## 22. Review Status

Status: Draft
Risk level: Medium
Review recommended: Yes
Reviewer:
Date: 2026-05-29
Notes: Medium risk because this is user-visible home-feed UI, app-side pagination, and cache coordination around optimistic creation and post interactions. Review is recommended before test design, but not required if the user accepts the documented scope and risks.

## 23. Handoff To Test Design

- Requirements file: `docs/changes/2026-05-29-timeline-feed-flutter/01-requirements.md`
- Next test specification: `02-acceptance-tests.md`
- Must-cover requirement IDs:
  - `BR-001`, `BR-002`
  - `FR-001` through `FR-007`
  - `FR-009` through `FR-012`
  - `FR-014`, `FR-015`
  - `NFR-001`, `NFR-002`, `NFR-004`, `NFR-005`
  - `RULE-001` through `RULE-004`
- Suggested test levels:
  - API client tests for `GET /v1/feed/timeline` path, optional `limit`/`cursor` query params, `PostPage` parsing, empty cursor handling, and error mapping.
  - Repository/fake tests proving timeline method plumbing does not require handle/DID input.
  - Riverpod provider tests for first load, empty page, pagination append, no-op after end, load-more failure preserving visible data/cursor, concurrent load-more guard, optimistic prepend, and URI deduplication.
  - Widget tests for `FeedPage` loading, loaded list, empty state, initial error retry, load-more retry, top-level compose entry, row tap to thread, like/repost updates, reply/comment flow, and own-post delete visibility/removal.
  - Regression tests for existing profile post/comment tabs, post card rendering, composer behavior, and interaction providers to ensure shared changes do not regress non-feed contexts.
  - Generated-code verification after provider/model/l10n changes.
- Blocking open questions: None.
- Review recommendation: Because this is medium-risk user-visible feed behavior with pagination and optimistic cache updates, review of `01-requirements.md` is recommended before test design.
