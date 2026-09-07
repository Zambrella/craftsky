# Requirements: Post Interaction Details

## 1. Initial Request

Let people viewing a post's detail screen see separate clickable nonzero counts for likes, reposts, and quote reposts immediately beneath the post card. Selecting a count navigates to a dedicated route: likes and reposts show account lists, while quote reposts show a post list. A viewer must also be able to inspect who liked a comment or reply through that response card's more-actions menu. Reuse existing Flutter widgets and patterns where practical, and add three AppView endpoints to supply the lists.

The phrase "quite reposted" in the prompt is interpreted as "quote reposted" based on the surrounding request and Craftsky's existing quote-post terminology.

## 2. Current Codebase Findings

- Relevant files:
  - `app/lib/feed/pages/post_thread_page.dart` renders the detail/thread page and its top-level `PostCard`.
  - `app/lib/feed/widgets/post_card.dart` renders separate `likeCount` and combined `repostCount + quoteCount` action feedback. The combined share control opens the repost/quote mutation menu.
  - The same `PostCard` more-actions menu is available on comment/reply cards and already composes contextual owner/report and relationship actions.
  - `app/lib/router/router.dart` and `app/lib/router/route_locations.dart` define typed authenticated routes, including `/posts/:did/:rkey`.
  - `app/lib/settings/pages/follow_list_page.dart`, `app/lib/profile/models/profile_account_summary.dart`, and `profile_account_page.dart` provide the current account-list model and presentation pattern.
  - `app/lib/shared/widgets/auto_paginated_list_view.dart` provides shared automatic cursor-pagination presentation with loading and retry states.
  - `app/lib/feed/models/post.dart`, `post_page.dart`, `app/lib/feed/data/post_api_client.dart`, and `post_repository.dart` provide the reusable post-list data contract.
  - `appview/internal/api/post_interactions.go` and `post_interactions_store.go` contain existing like/repost mutation handlers and indexed-interaction access.
  - `appview/internal/routes/routes_scheduled_post.go`, `policy.go`, and `inventory_test.go` register and classify the existing post-interaction routes.
  - `appview/migrations/000010_craftsky_posts.up.sql` indexes `quote_uri`; existing interaction migrations/index maintenance provide active subject indexes for `craftsky_likes` and `craftsky_reposts`.
  - `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md` and `2026-04-22-api-wire-alignment-design.md` govern `/v1/` authentication, errors, camelCase JSON, and opaque-cursor pagination.
  - `docs/changes/2026-07-08-reposts-and-quotes/01-requirements.md` established distinct `repostCount` and `quoteCount` semantics but deliberately excluded list screens at that time; this feature supersedes that earlier non-goal for post detail.
- Existing patterns:
  - Flutter reads interaction data from AppView using the Craftsky session; it does not query PDS repositories directly.
  - Authenticated list endpoints use `limit` and opaque `cursor`, return `items` and an omitted cursor at exhaustion, and use standard route policy/middleware.
  - Account summaries already support display name, handle, avatar/customisation, and viewer relationship flags.
  - Full post responses already support quote previews, moderation state, action callbacks, and navigation through `PostCard`.
- Current behavior:
  - The post detail screen has no interaction-summary row and no navigation to liker, reposter, or quote-post lists. Comment/reply more-actions menus do not provide liker-list navigation.
  - The card action row hides zero counts and combines repost and quote counts into one button whose tap opens mutation choices.
  - Likes and straight reposts are indexed as active interaction records; quote reposts are indexed normal posts whose `quote_uri` references the subject.
  - No GET handlers currently exist for the three requested interaction lists.
- Constraints discovered:
  - The existing share mutation control must retain its current purpose; turning its combined count into list navigation would make repost and quote destinations ambiguous and affect every `PostCard` consumer.
  - New endpoints must be additive authenticated `/v1/*` GET routes with camelCase JSON, standard errors, route-policy coverage, and opaque cursors.
  - Returned items and displayed counts must honor the resolved current-member, terminal-owner, block, mute, moderation, and content-visibility rules. Mutes remain revealable rather than existence-hiding.
  - No lexicon change or PDS write is needed because all three lists can be derived from indexed AppView data.
- Test/build commands discovered:
  - AppView release-equivalent verification: `just appview-check`.
  - AppView development test suite: `just test` with the compose services running.
  - Go formatting/vetting: `just fmt`.
  - Flutter tests: `just app-test` or focused `flutter test` from `app/`.
  - Flutter analysis: `just app-analyze`.

## 3. Clarifying Questions And Decisions

### Q1: Does "quite reposted" mean quote reposted?

Answer: Inferred yes from the requested list of quote posts and existing product terminology.

Decision / implication: The third interaction type is a quote repost, represented by an authored post whose `quote_uri` points to the viewed post.

### Q2: Should the existing action-row counts become the navigation controls?

Answer: No. The prompt places the new clickable counts underneath the post card, and the existing combined repost/quote control is needed for mutation choices.

Decision / implication: Add a detail-only summary row below the top-level card with independent nonzero Likes, Reposts, and Quotes links. Preserve the existing card action row on detail and all other surfaces. Comment/reply liker navigation is a separate more-actions menu item, not a summary row.

### Q3: What routes and response shapes best fit existing conventions?

Answer: Use separate sub-resources for each interaction and reuse the existing account-page and post-page item shapes.

Decision / implication: Flutter routes and AppView endpoints use `/posts/{did}/{rkey}/likes`, `/reposts`, and `/quotes`; AppView paths include the `/v1` prefix. Likes/reposts return hydrated account summaries plus `totalCount`; quotes return hydrated post items. All use opaque cursors.

### Q4: Are zero counts visible and actionable?

Answer: No. The user confirmed that zero counts must not show a link; for example, `0 likes` is never displayed.

Decision / implication: Each detail-summary link is independently omitted when its count is zero. If all three counts are zero, no interaction summary is shown. Comment/reply more-actions menus likewise omit liker-list navigation when `likeCount` is zero.

### Q5: How are results ordered?

Answer: No explicit order was supplied; reverse interaction chronology matches existing social-list expectations and stable cursor pagination.

Decision / implication: Likes and reposts order by interaction `created_at DESC, uri DESC`; quote reposts order by quote-post `created_at DESC, uri DESC`. SQL seek comparisons and opaque cursor fields use that exact tuple and direction.

### Q6: How are likes on comments and replies accessed?

Answer: The user requires access through the comment/reply card's more-actions menu.

Decision / implication: A comment or reply with a nonzero `likeCount` exposes a localized `View likes` action in its existing more-actions menu. Selecting it opens the same dedicated Likes route/page used by top-level posts. No summary row, repost list, or quote list is added to comments/replies.

### Q7: How do mute, membership, and block relationships affect membership?

Answer: Muted actors remain visible with existing relationship state. Former/non-members, terminal owners, and actors hidden by account moderation are excluded. An actor-author block in either direction severs the interaction, as does a viewer-actor block in either direction.

Decision / implication: Likes/Reposts require a current Craftsky profile and include muted actors. Quotes by muted authors remain in the count/list as revealable muted-post placeholders. Eligible totals use the same predicates as list membership.

### Q8: Do quote lists use viewer content-language preferences?

Answer: Yes. Quotes are authored content and follow the same language visibility rules as other post lists.

Decision / implication: The Quotes list and detail `quoteCount` exclude language-ineligible quote posts for that viewer. Likes/Reposts are account lists and are not language-filtered.

### Q9: How should destination titles and unavailable targets behave?

Answer: Likes/Reposts omit a pending zero while initially loading, then show the successful `totalCount`. Quotes keeps a plain title and no `totalCount`. A target `404` gets a non-retryable unavailable state; transient failures remain retryable.

Decision / implication: Do not render `Likes (0)` or `Reposts (0)` before a response. Keep Quotes `PostPage`-compatible. Reposts/Quotes requests for comments/replies return `404 post_not_found`.

### Q10: What are the responsive summary and response-menu layouts?

Answer: Summary links wrap independently in fixed Likes, Reposts, Quotes order. `View likes` is the first item in its own non-destructive menu group.

Decision / implication: Use a left-aligned wrapping row with spacing and no punctuation separators. Keep liker navigation visually separated from ownership, relationship, report, and delete actions.

## 4. Candidate Approaches

### Option A: Detail Summary Row And Three Dedicated Routes

Summary: Add independent nonzero count links beneath the detail card, a nonzero comment/reply liker action in the existing more-actions menu, three typed Flutter routes/pages, and three corresponding AppView list endpoints. Reuse account summaries, post cards, and pagination presentation.

Pros: Directly matches the prompt, preserves mutation controls, creates shareable/deep-linkable destinations, and keeps each response type simple.

Cons: Adds three routes and page states even though two pages have nearly identical account-list behavior.

Risks: Count/list visibility drift, cursor errors during concurrent interaction changes, and duplicated account-row code if reuse is handled poorly.

### Option B: Convert Existing Action Counts Into Navigation

Summary: Make action-row counts navigate while icons retain like/repost mutation behavior.

Pros: Uses less vertical space and resembles some established social applications.

Cons: The existing repost count combines straight reposts and quotes, creates ambiguous tap targets, complicates semantics, and changes shared-card behavior beyond detail.

Risks: Accidental mutations or navigation, reduced accessibility, and regressions across every `PostCard` surface.

### Option C: One Interaction Route With Tabs

Summary: Use one route and a tabbed page for Likes, Reposts, and Quotes.

Pros: One page shell and easy switching among interaction types.

Cons: Does not follow the request for navigation based on the chosen count as cleanly, mixes account and post item types in one screen, and introduces tab/deep-link state not otherwise needed.

Risks: More UI state and less direct route semantics for little benefit.

## 5. Recommended Direction

Recommended approach: Option A, a detail-only nonzero interaction summary plus comment/reply liker access, backed by three dedicated routes and AppView endpoints.

Why: It is the smallest change that gives each distinct nonzero count an unambiguous destination and lets response viewers inspect likers without adding another inline row. It preserves the established repost/share mutation control and reuses the existing account summary, post response, `PostCard` menu, profile presentation, and automatic pagination patterns without introducing a new aggregate response or tab system.

## 6. Problem / Opportunity

Engagement counts currently show that a post received activity but do not let a viewer understand who engaged or what people said when quoting it. Exposing the underlying visible accounts and quote posts makes engagement transparent, supports discovery of people and conversation, and completes a common post-detail interaction pattern.

## 7. Goals

- G-001: Show separate, understandable like, repost, and quote counts on every available top-level post detail.
- G-002: Let a viewer open a dedicated, paginated destination for each interaction type.
- G-003: Reuse established account, post, profile-navigation, error, empty-state, and pagination patterns.
- G-004: Keep interaction details consistent with AppView visibility policy and the counts shown to the viewer.
- G-005: Preserve existing like, repost, quote, thread, and shared-card mutation behavior.
- G-006: Let viewers inspect the visible accounts that liked a comment or reply without adding repost/quote affordances to responses.

## 8. Non-Goals

- NG-001: Do not change like, unlike, repost, unrepost, or quote-post creation semantics.
- NG-002: Do not add inline interaction summaries to feed, profile, search, project-discovery, notification, comment, or reply cards; comment/reply more-actions liker navigation is the sole response-card exception.
- NG-003: Do not combine the three destinations into tabs, filters, search, sorting controls, or a single polymorphic endpoint.
- NG-004: Do not expose interaction-record URIs, CIDs, timestamps, or other internal record metadata to Flutter unless already part of a reused public model.
- NG-005: Do not add follow/unfollow controls, bulk actions, moderation actions, or new profile-card behavior to account rows.
- NG-006: Do not add public PDS records, lexicon changes, direct Flutter-to-PDS reads, notifications, or analytics tracking.
- NG-007: Do not provide repost or quote lists for comments/replies because those interactions are not supported on responses in the current product rules.
- NG-008: Do not guarantee a snapshot that remains immutable while later pages are fetched; deterministic keyset behavior is sufficient.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Post detail viewer | Authenticated member viewing an available top-level post and its thread. | See distinct engagement counts and open the corresponding visible interactions. |
| Response viewer | Authenticated member viewing a comment or nested reply with likes. | Open the visible liker list from the response's existing more-actions menu. |
| Account-list viewer | Viewer of a post's likes or reposts. | Recognize accounts and open their existing profile presentation. |
| Quote-list viewer | Viewer of quote reposts for a post. | Read each visible quote post and navigate/interact through existing post behavior. |
| AppView | Trusted read, identity-hydration, and policy boundary. | Return stable paginated lists without leaking hidden or terminal content. |
| Flutter client | Consumer of the three list contracts. | Render responsive, accessible loading, content, empty, retry, and pagination states using existing components. |

## 10. Current Behavior

On post detail, the top-level `PostCard` shows action controls followed immediately by thread sorting and replies. The like action displays a nonzero like count; the share action displays the nonzero sum of straight repost and quote counts and opens a repost/quote action menu. There is no separate summary below the card, no typed interaction-detail routes, no Flutter repository methods for these lists, and no AppView GET routes for them.

## 11. Desired Behavior

Immediately beneath the top-level post card, the post detail screen shows a localized, accessible link for each interaction type whose exact count is greater than zero: Likes, Reposts, and Quotes. Zero-count links are omitted independently, and no summary is rendered when all three are zero. Each visible link is independent from the card's mutation controls. Activating one pushes a typed authenticated route tied to the post DID and record key.

A comment or nested reply with one or more likes exposes a localized `View likes` action in its existing more-actions menu. Activating it pushes the same typed Likes route for that response's DID and record key. A response with zero likes does not show the action, and responses do not gain inline summaries, repost lists, or quote lists.

The Likes and Reposts pages show cursor-paginated account rows, newest interaction first. Each row uses the established profile account summary presentation and opens the account's existing profile card/presentation. Their titles omit a count until the first successful response, then show the authoritative `totalCount`. The Quotes page has a plain title and shows cursor-paginated, fully hydrated quote posts newest first using the established `PostCard` behavior. All pages provide initial loading, empty, non-retryable target-unavailable, retryable transient initial error, incremental loading, and incremental-error retry states and remain usable on supported mobile and large layouts.

The AppView resolves the target post, applies the requesting viewer's existing visibility policy, and serves only active visible interactions. Its count and list calculations use the same eligibility rules so the summary and returned total do not knowingly diverge. Concurrent creates/deletes may change later pages, but stable newest-first keyset cursors prevent ordering ties from causing duplicate or skipped records within an unchanged data set.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | An authenticated viewer of an available top-level post shall be able to see each nonzero like, straight-repost, and quote-repost count separately and inspect the visible activity behind it. | Engagement should be transparent and useful for account/conversation discovery without displaying zero-value links. | Prompt; User review feedback | AC-001, AC-002, AC-003, AC-004 |
| BR-002 | Business | Must | Interaction-detail presentation shall preserve the distinction between straight reposts and quote posts. | They are different record types and lead to different content. | Prompt; Existing repost/quote requirements | AC-001, AC-003, AC-004 |
| BR-003 | Business | Should | The feature should feel consistent with existing Craftsky account lists, post lists, and profile/post navigation. | Reuse reduces learning cost and implementation divergence. | Prompt; Codebase | AC-006, AC-007, AC-008 |
| BR-004 | Business | Must | An authenticated viewer shall be able to inspect the visible accounts that liked an available comment or nested reply. | Liker transparency is also required for responses in a thread. | User review feedback | AC-024 |
| FR-001 | Functional | Must | The top-level post detail shall render a detail-only, left-aligned wrapping summary immediately beneath the `PostCard`, in fixed Likes, Reposts, Quotes order, with a separate localized link only when that post model count is greater than zero. Each visible link shall show the exact count; links use spacing without punctuation separators, zero-count links are omitted independently, and the whole summary is omitted when all counts are zero. | The action row combines repost/quote counts and cannot provide distinct destinations, while the user explicitly rejected zero-count links. | Prompt; Codebase; User review feedback | AC-001, AC-005 |
| FR-002 | Functional | Must | Activating each summary count shall push its dedicated typed authenticated route for the current post: `/posts/{did}/{rkey}/likes`, `/posts/{did}/{rkey}/reposts`, or `/posts/{did}/{rkey}/quotes`. Direct navigation and back navigation shall retain the normal authenticated-shell behavior. | Each selected count needs an unambiguous destination and deep-linkable identity. | Prompt; Routing pattern | AC-002, AC-009 |
| FR-003 | Functional | Must | The Likes destination shall display a cursor-paginated list of visible accounts with active likes for the target post, ordered by `(created_at DESC, uri DESC)`. | Viewers need to see who liked the post with deterministic pagination. | Prompt; API pagination convention; Codebase convention | AC-003, AC-010 |
| FR-004 | Functional | Must | The Reposts destination shall display a cursor-paginated list of visible accounts with active straight reposts for the target post, ordered by `(created_at DESC, uri DESC)`, excluding quote-post authors unless they also straight-reposted. | Straight reposts must remain distinct from quote posts. | Prompt; Existing semantics; Codebase convention | AC-003, AC-010, AC-011 |
| FR-005 | Functional | Must | Likes and Reposts account items shall use the existing `ProfileAccountSummary`-compatible shape, include `totalCount`, and open the account's existing profile presentation when selected. Their title shall remain count-free during initial loading and show the successful authoritative total afterward, including zero for a genuine empty response. | Reuses current identity hydration and account navigation without presenting a pending value as real data. | Prompt; Codebase; User decision | AC-003, AC-006, AC-012 |
| FR-006 | Functional | Must | The Quotes destination shall display cursor-paginated, visible quote posts whose indexed `quote_uri` references the target post, ordered by `(created_at DESC, uri DESC)`. Items shall use the existing hydrated post response shape and `PostCard` behavior. The response remains `PostPage`-compatible without `totalCount`, and the page title remains plain. | Quote reposts contain authored content and should be read as posts without creating another post-page contract. | Prompt; Codebase; User decision | AC-004, AC-007, AC-010 |
| FR-007 | Functional | Must | Each destination shall support initial loading, a type-specific empty state, a non-retryable localized target-unavailable state with Back for `404 post_not_found`, retryable transient initial failure, automatic incremental loading, and retryable incremental failure without discarding already loaded items. | Network and pagination states must be complete without encouraging futile retries for unavailable targets. | Codebase; Recommended direction; User decision | AC-008, AC-013 |
| FR-008 | Functional | Must | The AppView shall add authenticated GET endpoints at `/v1/posts/{did}/{rkey}/likes`, `/v1/posts/{did}/{rkey}/reposts`, and `/v1/posts/{did}/{rkey}/quotes`; each accepts bounded `limit` and opaque `cursor`, returns camelCase JSON with `items` and an omitted cursor at exhaustion, and uses the standard error envelope. The Likes endpoint accepts any available indexed post, including comments/replies; Reposts and Quotes remain top-level-post surfaces. Likes/reposts return `ProfileAccountPage`-compatible responses; quotes return `PostPage`-compatible responses without pin metadata. | Supplies the exact three requested read surfaces and the reviewed response-liker behavior using established contracts. | Prompt; User review feedback; API architecture; Codebase | AC-014, AC-015, AC-024 |
| FR-009 | Functional | Must | Each endpoint shall validate the DID and record key, return `404 post_not_found` when the target is missing or unavailable to the viewer, and return `400 invalid_cursor` for malformed or endpoint/target-mismatched cursors. Reposts/Quotes shall also return `404 post_not_found` for comment/reply targets. | Prevents ambiguous empty results, unsupported response-share surfaces, and unsafe cursor reuse. | API architecture; Existing handler conventions; User decision | AC-015, AC-016 |
| FR-010 | Functional | Must | Likes/Reposts shall require current Craftsky actors; exclude terminal, account-hide/takedown, viewer-actor blocked, and actor-author blocked relationships in either direction; and include muted actors with existing relationship state. Quotes shall exclude terminal, blocked, account/post-hide/takedown, and viewer-language-ineligible posts, while including muted authors as revealable placeholders. List totals and detail counts shall use the same eligibility predicates. | Lists and counts must not leak hidden activity or contradict one another, while mute remains revealable rather than existence-hiding. | Architecture; Existing policies; User decisions | AC-011, AC-012, AC-017 |
| FR-011 | Functional | Must | Successful like/unlike and repost/unrepost updates already reflected in the detail post model shall update the corresponding summary count without requiring separate interaction-list state, while quote count continues to follow normal post refresh/cache behavior. | Keeps the new summary consistent with the authoritative post state and avoids duplicate count ownership. | Codebase; Minimal direction | AC-005 |
| FR-012 | Functional | Must | The existing like and repost/share action controls shall retain their current mutation behavior and combined share-count behavior; the new navigation summary shall appear only below the top-level detail card and not on comment/reply cards or other `PostCard` consumers. | Avoids breaking established controls and prevents broad shared-widget regressions. | Codebase; Recommended direction | AC-018 |
| FR-013 | Functional | Must | A comment or nested reply with `likeCount > 0` shall expose a localized `View likes` action as the first item in a separate non-destructive group in its existing more-actions menu. Activating it shall push the same Likes route for that response; the action shall be omitted at zero, and no inline interaction summary shall be added to response cards. | Provides response-liker access while separating navigation from ownership and moderation actions. | User review feedback | AC-024 |
| NFR-001 | Non-functional | Must | All three list queries shall use bounded, indexed, set-based access and avoid per-item database or identity lookups for normal page hydration. | These lists may be large and are user-facing read paths. | Codebase; Existing indexes | AC-019 |
| NFR-002 | Non-functional | Must | Pagination shall be deterministic for a fixed data set, use `(created_at DESC, uri DESC)` seek ordering and opaque target/type-bound cursors, respect the requested bounded limit, and return each eligible interaction at most once during unchanged traversal. | Prevents duplicates, gaps, and cross-endpoint cursor misuse. | API architecture; Codebase convention | AC-010, AC-016 |
| NFR-003 | Non-functional | Must | Count links, comment/reply `View likes` menu actions, account rows, post rows, loading/error feedback, and empty states shall be accessible by touch, keyboard/focus navigation, and screen reader, and remain readable without clipping at supported text scales and mobile/large layouts. | Interaction details must not depend on sight, color, or one input method. | UI quality standard | AC-020 |
| NFR-004 | Non-functional | Must | Flutter shall obtain all interaction details through AppView using the current Craftsky session and shall never receive PDS OAuth credentials or query a PDS directly. | Preserves the Token Mediating Backend boundary. | AGENTS.md architecture | AC-014, AC-021 |
| NFR-005 | Non-functional | Should | The three routes should use existing bounded HTTP/database observability and error reporting without logging post URIs, account DIDs, handles, cursor contents, or response items as telemetry dimensions. | Enables diagnosis without high-cardinality or identity leakage. | Existing observability patterns | AC-022 |
| NFR-006 | Non-functional | Must | The change shall have AppView store/handler/route-contract tests and Flutter data/provider/router/widget/accessibility regression tests covering all three interaction types, comment/reply liker access, and existing mutation controls. | The feature crosses API, policy, pagination, routing, and shared UI boundaries. | Workflow quality standard | AC-023 |
| RULE-001 | Business rule | Must | An active like contributes one liker account; an active straight repost contributes one reposter account; a quote post contributes one post item. Deleted/inactive records do not contribute. | Defines list membership consistently with indexed record semantics. | Existing data model | AC-003, AC-004, AC-011 |
| RULE-002 | Business rule | Must | A quote post is not a straight repost, even when authored by the same account for the same subject; it appears only in Quotes unless that account also has an active straight repost. | Preserves established semantic and count separation. | Existing repost/quote requirements | AC-004, AC-011 |
| RULE-003 | Business rule | Must | Multiple quote posts by the same author for the same subject shall appear as separate post items when each is otherwise visible and active; likes/reposts list an account at most once because existing invariants allow at most one active record per account and subject. | Quote posts are authored posts, while likes/reposts are toggle interactions. | Existing data model and rules | AC-003, AC-004 |
| RULE-004 | Business rule | Must | A zero interaction count shall not produce a detail-summary link or a comment/reply `View likes` menu action. | The user explicitly does not want `0 likes` or equivalent zero-count links shown. | User review feedback | AC-001, AC-024 |
| RULE-005 | Business rule | Must | Comment/reply liker access uses the Likes list only; response cards shall not gain repost or quote list actions. | Repost and quote interactions remain unsupported for responses. | User review feedback; Existing response rules | AC-024 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, BR-002, FR-001, RULE-004 | Given an available top-level post detail with any combination of zero and nonzero interaction counts, when the page renders, then the immediately-under-card summary shows separate exact labelled links in fixed Likes, Reposts, Quotes order only for nonzero values; links wrap independently without punctuation separators, no zero value is shown, and the whole summary is absent when all three are zero. |
| AC-002 | BR-001, FR-002 | Given any visible summary link, when the viewer activates it by a supported input method, then the corresponding post-specific route opens. |
| AC-003 | BR-001, BR-002, FR-003, FR-004, FR-005, RULE-001, RULE-003 | Given active visible likes and straight reposts, when their destinations load, then each eligible account appears once in the correct newest-first list with hydrated account presentation, and selecting an account opens its existing profile presentation. |
| AC-004 | BR-001, BR-002, FR-006, RULE-001, RULE-002, RULE-003 | Given visible quote posts for the target, including two from one author, when Quotes loads, then each quote post appears separately newest first as a normal hydrated post card and no quote-only author is added to Reposts. |
| AC-005 | FR-001, FR-011 | Given the detail post model changes after a successful like/unlike or repost/unrepost, when Flutter rebuilds the detail, then the corresponding new summary value matches the model's count and is not maintained by a conflicting local counter. |
| AC-006 | BR-003, FR-005 | Given an account returned by Likes or Reposts, when its row renders, then existing account-summary fields and profile presentation behavior are reused rather than introducing an incompatible interaction-only account contract. |
| AC-007 | BR-003, FR-006 | Given a quote list item, when it renders and the viewer uses its existing author, quote-preview, action, or post navigation affordances, then normal `PostCard` behavior is preserved. |
| AC-008 | BR-003, FR-007 | Given each destination is loading or fails before returning a first page, then it shows progress, a retryable transient-error presentation, or a non-retryable localized target-unavailable state with Back for `post_not_found`; after a successful empty response, it shows interaction-type-specific empty copy. |
| AC-009 | FR-002 | Given the viewer opens an interaction route directly while authenticated and then navigates back, then typed DID/rkey parsing, authenticated-shell presentation, and return to the originating detail/navigation history work without requiring in-memory route extras. |
| AC-010 | FR-003, FR-004, FR-006, NFR-002 | Given more results than one page and no data changes during traversal, when the client follows every cursor at a fixed limit, then pages remain newest first, no page exceeds the limit, every eligible item appears exactly once, and exhaustion omits `cursor`. |
| AC-011 | FR-004, FR-010, RULE-001, RULE-002 | Given an account only quote-posted the target, an account only straight-reposted it, and an account did both, when lists load, then Quotes contains the quote post(s), Reposts contains only accounts with active straight repost records, and the account that did both participates independently in both results. |
| AC-012 | FR-005, FR-010 | Given account interactions include current, former/non-member, terminal, viewer-blocked, actor-author-blocked, muted, account-hide/takedown, warned, or otherwise non-returnable actors, when Likes or Reposts is requested, then muted and warning-only current actors are returned and counted with existing relationship fields, while all defined ineligible actors are neither returned nor counted. |
| AC-013 | FR-007 | Given a destination has no eligible items, including after visibility changes or direct navigation, or a later page fails, then the initial empty state is scroll-safe/readable and a later-page failure preserves loaded items while offering retry. |
| AC-014 | FR-008, NFR-004 | Given valid authentication and device headers, when each of the three GET endpoints is called, then it returns the specified account or post page; without valid authentication it is rejected by existing middleware, and no PDS request or credential handoff occurs. |
| AC-015 | FR-008, FR-009 | Given valid or invalid endpoint requests, when contracts are inspected, then successes use camelCase `items`, optional `cursor`, and account `totalCount` where applicable; failures use `{error, message, requestId}` with the expected status and no success-response wrapper. |
| AC-016 | FR-009, NFR-002 | Given a malformed cursor or a valid cursor copied between interaction types or target posts, when it is submitted, then AppView returns `400 invalid_cursor`; an invalid DID/rkey returns the existing identifier error behavior, and a missing/unavailable target returns `404 post_not_found`. |
| AC-017 | FR-010 | Given the detail counts and each first-page result are evaluated for the same viewer against unchanged indexed data, when membership, block, moderation, mute, and quote-language policy is applied, then each summary count/returned account `totalCount` equals the number of eligible active records for that type; muted actors and revealable muted quote placeholders remain included, and no hidden record is exposed through an item or count. |
| AC-018 | FR-012 | Given the feature is enabled, when post cards render in detail, feed, profile, search, project, notification, comment, and reply contexts, then only the top-level detail has the new summary; existing like controls still mutate likes, the combined share control still opens repost/quote choices, and the response more-actions exception does not alter those controls. |
| AC-019 | NFR-001 | Given a full page from any endpoint, when database and hydration behavior are inspected, then subject/quote indexes and bounded set-based hydration are used without one query or external identity lookup per returned item. |
| AC-020 | NFR-003 | Given touch, keyboard, screen-reader, narrow viewport, large layout, and supported enlarged text use, when the viewer navigates the summary, a comment/reply `View likes` menu action, and the destinations, then each count announces its value/type/destination, menu and row actions have clear semantics, status changes are understandable, and content does not clip or overlap. |
| AC-021 | NFR-004 | Given Flutter loads any interaction destination, when network traffic is inspected, then requests go only to the AppView `/v1/` endpoint with Craftsky session/device credentials and no PDS OAuth credential is present in memory handoff, storage, URL, or logs. |
| AC-022 | NFR-005 | Given successful, rejected, and failed list requests, when telemetry is inspected, then route/status/latency and bounded database-operation observations are available without post URIs, DIDs, handles, cursor contents, or list items as dimensions. |
| AC-023 | NFR-006 | Given the feature verification suite runs, then it covers active/deleted interactions, straight-versus-quote separation, repeated quote authors, ordering/ties, limits/cursors/exhaustion/cursor misuse, missing targets, auth/policy filtering/count alignment, all Flutter page states, deep links/back navigation, accessibility/responsiveness, account/profile navigation, quote-card reuse, positive/zero count visibility, comment/reply liker menu access, count updates, and unchanged mutation controls on all existing card surfaces. |
| AC-024 | BR-004, FR-008, FR-013, RULE-004, RULE-005 | Given an available comment or nested reply, when `likeCount` is greater than zero, then its existing more-actions menu shows a localized `View likes` action that opens the response-specific Likes route and returns the visible liker account list; when `likeCount` is zero, the action is absent. In both cases no inline summary, repost-list action, or quote-list action is added to the response card. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | One or all top-level detail counts are zero. | Omit each zero-count link independently. If all are zero, omit the entire interaction summary. | FR-001, RULE-004 |
| EC-002 | Interaction is deleted between page requests. | A later traversal may contain fewer items; no inactive record is returned. The client remains stable and refresh starts a new traversal. | FR-003, FR-004, NFR-002, RULE-001 |
| EC-003 | New interactions arrive between page requests. | Keyset pagination continues from the opaque cursor; refreshing from page one reveals newer activity. No immutable snapshot guarantee is implied. | FR-003, FR-004, FR-006, NFR-002 |
| EC-004 | Like/repost records have equal timestamps. | Stable unique tie-breakers produce deterministic ordering without duplicate or skipped items in unchanged data. | FR-003, FR-004, NFR-002 |
| EC-005 | Two quote posts by one author reference the same target. | Both visible posts appear separately in Quotes. | FR-006, RULE-003 |
| EC-006 | One account both reposted and quote-posted. | The account appears once in Reposts and each visible quote post appears in Quotes; neither action substitutes for the other. | FR-004, RULE-002, RULE-003 |
| EC-007 | Quoting post is hidden, language-ineligible, or authored by a muted account. | Hidden/language-ineligible posts are omitted from Quotes and `quoteCount`; muted-author posts remain counted and render through the existing revealable placeholder behavior. | FR-006, FR-010 |
| EC-008 | Quoting post contains an unavailable nested quote preview or is itself a quote of a quote. | Reuse normal one-level post/quote-preview moderation behavior; do not recursively hydrate beyond the existing post contract. | FR-006, FR-010 |
| EC-009 | Target post is deleted, hidden, or otherwise unavailable to the viewer. | All three endpoints return `404 post_not_found`; the routes do not expose interaction existence. | FR-009, FR-010 |
| EC-010 | A cursor is replayed against another post or interaction endpoint. | Return `400 invalid_cursor`; do not reinterpret or partially honor it. | FR-009, NFR-002 |
| EC-011 | Active account changes while a list request is in flight. | Existing account/session fencing prevents the old completion from populating the newly active account's state. | NFR-004 |
| EC-012 | Initial request succeeds but load-more fails. | Keep rendered items, announce/show the failure, and allow retry with the same cursor. | FR-007, NFR-003 |
| EC-013 | Count changes while an interaction destination remains open. | The page may retain its current traversal until refresh; returning to detail uses the current post-model count. | FR-011, NFR-002 |
| EC-014 | Comment/reply like count changes while its more-actions menu is open or after navigation. | Menu availability follows the latest rendered post model. The Likes destination remains authoritative and may validly show a changed or empty result after concurrent unlikes or policy changes. | FR-007, FR-013, RULE-004 |
| EC-015 | Reposts or Quotes is requested for a comment/reply. | Return `404 post_not_found`; do not return an empty success or introduce a response-share list. | FR-008, FR-009, RULE-005 |
| EC-016 | An account handle cannot be resolved while hydrating Likes/Reposts. | Fail the request with retryable `502 identity_unavailable`; do not silently omit the account or return a mismatched total. | FR-005, FR-007, FR-010 |

## 15. Data / Persistence Impact

- New fields: No persistent fields expected. Likes/reposts responses reuse `ProfileAccountSummary` items and `totalCount`; quote responses reuse post items.
- Changed fields: Existing post response fields remain unchanged. The detail UI begins consuming already separate `likeCount`, `repostCount`, and `quoteCount` values in a new presentation.
- Migration required: None expected. Existing active subject indexes for likes/reposts and the partial `craftsky_posts.quote_uri` index cover the access paths; implementation must verify query plans and add a reversible index migration only if evidence shows an existing index is insufficient.
- Backwards compatibility: The three GET routes and Flutter routes are additive. Existing POST/DELETE routes at the likes/reposts paths remain method-distinct and unchanged. No PDS or lexicon data changes.

## 16. UI / API / CLI Impact

- UI:
  - Add a left-aligned wrapping interaction summary immediately below only the top-level post card in `PostThreadPage`, containing nonzero links in Likes, Reposts, Quotes order.
  - Add `View likes` as a separate first menu group in a comment/reply's existing more-actions menu only when its like count is nonzero.
  - Add dedicated Likes, Reposts, and Quotes pages with localized titles and empty-state copy.
  - Reuse/extract the existing account-row presentation for Likes/Reposts, existing profile presentation on account tap, `AutoPaginatedListView`, and `PostCard` for Quotes.
  - Preserve the existing card action row and comment/reply layout.
- API:
  - Add `GET /v1/posts/{did}/{rkey}/likes` returning `{items: [ProfileAccountSummary], totalCount, cursor?}`.
  - Add `GET /v1/posts/{did}/{rkey}/reposts` returning `{items: [ProfileAccountSummary], totalCount, cursor?}`.
  - Add `GET /v1/posts/{did}/{rkey}/quotes` returning `{items: [Post], cursor?}`.
  - All routes use authenticated/device middleware, read-rate/no-body policy, bounded `limit`, opaque `cursor`, visibility policy, and standard error envelopes.
  - The Likes route accepts top-level posts, comments, and nested replies; Reposts and Quotes remain top-level-only.
- CLI: None.
- Background jobs: None. Existing Tap indexers continue to maintain likes, reposts, and quote-post references.

## 17. Security / Privacy / Permissions

- Authentication: All detail and list routes require an authenticated current-member Craftsky session and device ID under existing middleware.
- Authorization: A viewer may inspect interactions only when the target post is returnable; a muted/revealable target remains eligible. Likes/Reposts additionally require current-member actors and no viewer-actor or actor-author block in either direction. Quotes apply normal post moderation/block/language policy while preserving revealable mute behavior.
- Sensitive data: Return only existing public account summaries and public post responses. Do not expose private AppView state, PDS credentials, raw interaction records, or hidden-account identifiers.
- Abuse cases:
  - Counts and pages must not become an oracle for blocked, moderated, terminal, former/non-member, language-ineligible, or otherwise hidden actors/posts. Muted actors/posts remain deliberately revealable under the rules above.
  - Invalid or cross-target cursors must fail without disclosing whether hidden interactions exist.
  - Direct API requests must not bypass target-post or item visibility checks.

## 18. Observability

- Events: No product analytics events required.
- Logs: Reuse request-ID-correlated route and error logging; avoid raw post/account identifiers, cursor contents, or response data.
- Metrics: Reuse bounded HTTP status/latency and database operation timing for the three route templates and list-query operations.
- Alerts: None specific to this pre-production feature.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Detail counts and list eligibility use different visibility rules. | Counts could reveal hidden activity or fail to match visible lists. | Centralize/reuse existing policy predicates and test count/list alignment for the same viewer. |
| RISK-002 | Shared `PostCard` changes alter mutation behavior or leak summary/menu actions to unrelated surfaces. | Users may navigate when intending to mutate, or UI may regress broadly. | Keep the summary owned by the detail page, make liker-menu exposure explicit for comments/replies only, and add shared-card surface regression tests. |
| RISK-003 | Account hydration introduces N+1 database or identity calls. | Large interaction pages become slow or dependency-heavy. | Use set-based profile hydration and verify bounded query behavior. |
| RISK-004 | Cursor ordering is based only on non-unique timestamps. | Tied or concurrent interactions can duplicate or disappear across pages. | Include a stable unique record key in ordering/cursor state and test timestamp ties. |
| RISK-005 | Quote-list post hydration bypasses existing moderation or recursively expands quote data. | Hidden content may leak and responses may grow unexpectedly. | Route quote items through the established bounded post hydration/visibility path. |
| RISK-006 | The account list model's `totalCount` and an independently computed post count drift under concurrent changes. | Page title/list metadata can briefly disagree with the originating detail. | Define both as request-time eligible totals, accept normal concurrency changes, and refresh rather than attempting a cross-screen snapshot. |
| RISK-007 | Three similar pages duplicate state and presentation code. | Fixes and accessibility behavior may diverge. | Share account-list page behavior between Likes/Reposts and reuse the existing pagination component; do not force account/post pages into one polymorphic abstraction. |
| RISK-008 | Comment/reply menu navigation uses the root post identity instead of the selected response identity. | The viewer sees the wrong liker list. | Build the route from the response card's own DID/rkey and cover comments and nested replies separately. |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | "Quite reposted" means quote reposted. | The third list's meaning and endpoint would need clarification and revision. |
| ASM-005 | Existing subject and quote indexes are sufficient for expected pre-production data volume. | A reversible index migration and migration tests may be required. |

## 21. Open Questions

- None blocking. The assumptions above are non-blocking and can be revised before test design if product intent differs.

## 22. Review Status

Status: Reviewed

Risk level: Medium

Review recommended: Yes

Reviewer: User

Date: 2026-09-06

Notes: User review and grilling feedback was applied on 2026-09-06. It resolves exact ordering, mute/membership/block/language policy, destination totals/loading/unavailable states, response-target errors, responsive summary layout, and response-menu grouping. Further review remains recommended because this is an authenticated full-stack API, pagination, visibility-policy, routing, and shared-UI change.

## 23. Handoff To Test Design

- Requirements file: `01-requirements.md`
- Next test specification: `02-acceptance-tests.md`
- Must-cover requirement IDs: BR-001, BR-002, BR-004, FR-001 through FR-013, NFR-001 through NFR-004, NFR-006, RULE-001 through RULE-005.
- Suggested test levels:
  - AppView store tests for active/deleted membership, top-level and response likes, viewer policy, count alignment, newest-first ordering, timestamp ties, limits, cursor traversal/misuse, and bounded hydration.
  - AppView handler/route tests for the three response shapes, identifier/target errors, authentication/device middleware, policy classification, camelCase JSON, omitted exhausted cursors, and standard envelopes.
  - Flutter model/API-client tests for account and post page decoding, exact endpoint paths/query parameters, missing cursors, errors, and session isolation.
  - Flutter provider/page tests for first load, empty, initial retry, automatic load-more, retained-items retry, repeated quote authors, and account switching.
  - Flutter router/widget/accessibility tests for direct routes, back navigation, omitted zero-count links, exact nonzero summary presentation, comment/reply `View likes` menu visibility and response-specific routing, account/profile taps, quote `PostCard` reuse, responsive/text-scale behavior, and unchanged mutation controls across existing card surfaces.
- Blocking open questions: None.
