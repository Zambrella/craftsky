# Business Requirements: Reply Comment Section

## 1. Summary

Replace the existing recursive post thread experience with a root-post comment section. The new experience must support deep links to individual comments or replies, comment sorting and lazy loading, action-expanded replies, and a maximum of two visible levels while preserving full backend reply parentage.

Terminology is load-bearing for this change: a direct/top-level reply to the root post is a **comment**; a reply under a comment is a **reply**. Product copy, API response fields, Flutter model/provider/widget names, tests, and implementation names should follow that convention. Backend storage fields may retain `reply_*` names where they refer to atproto reply refs.

## 2. Problem / Opportunity

The current `/posts/:did/:rkey` screen behaves like a recursive thread reader: it can show ancestors above the selected post and nested descendants below it. The desired product experience is a comment section under a root post: opening a post shows only comments, users can expand replies when desired, and links from shares or push notifications can focus a specific comment or reply inside that root-post context.

## 3. Goals

- G-001: Make every reply comment deep-linkable from shares and future push notifications.
- G-002: Present replies as a predictable two-level comment section under the root post.
- G-003: Support comment ordering and efficient incremental loading.
- G-004: Preserve exact atproto reply parentage in backend data even when the UI flattens deeper visual nesting.
- G-005: Remove the pre-production `/thread` route and thread-specific client model to avoid parallel reply paradigms.

## 4. Non-Goals

- NG-001: Implementing a real follows graph, follow indexing, or true follows-based reply ranking.
- NG-002: Changing the `social.craftsky.feed.post` lexicon or creating a separate comment record type.
- NG-003: Displaying infinite or third-level-and-deeper visual nesting.
- NG-004: Building push notification registration or delivery infrastructure.
- NG-005: Maintaining production backward compatibility for `/v1/posts/{did}/{rkey}/thread`.
- NG-006: Loading every intermediate reply before a focused reply that is outside the first reply page.

## 5. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Viewer | Signed-in Craftsky user reading a post and its comments. | See comments first, expand replies intentionally, choose comment ordering, and load more replies without losing place. |
| Comment author | Signed-in user creating a reply. | See their newly-created reply appear and scroll into view, including replies to existing replies. |
| Deep-link recipient | User opening a shared link or future notification. | Land on the root post with the intended reply visible, expanded if needed, and focused/scrolled into view. |
| AppView API client | Flutter app code consuming `/v1/*` JSON APIs. | Receive camelCase JSON, opaque cursors, bounded page sizes, and enough context to render/focus comments. |

## 6. Current Behavior

- `GET /v1/posts/{did}/{rkey}/thread` returns a recursive thread tree with ancestors and descendants.
- Flutter routes `/posts/:did/:rkey` to `PostThreadPage`, which renders ancestors, an anchor post, and recursive replies.
- `GET /v1/posts/{did}/{rkey}/replies` returns direct child replies oldest-first with cursor pagination.
- The backend stores full reply refs using `reply_root_uri`, `reply_root_cid`, `reply_parent_uri`, and `reply_parent_cid`.
- There is no implemented follows graph available for true follows-based reply ordering.

## 7. Desired Behavior

Opening a root post route shows the root post and comments only. Comments load 10 at a time as the user scrolls and can be ordered by `oldest`, `newest`, or `follows`, where `follows` is visible but behaves like `oldest` until real follow data exists. Viewer-authored comments always appear ahead of other non-focused comments, with the selected sort applied inside each placement group. If a comment has replies, the UI shows “view replies”; tapping it loads 10 visual replies oldest-first for that comment branch, including deeper descendants flattened under the comment, then changes the control to “hide replies” and shows “load more” when more replies exist.

Comment/reply deep links use the root post route plus `focus=<url-encoded AT-URI>`. Opening such a link includes the focused comment branch even if it is outside the first comment page, promotes that branch to the top of the displayed comments list, expands the relevant reply list if needed, and scrolls/focuses the target. Focus promotion outranks viewer-authored comment grouping and remains while the user scrolls, but clears when the user explicitly changes sort/filter. If a focused reply is outside the first reply page, the focused branch includes a bounded focused slice around the target rather than loading every earlier reply. The UI never shows a third indentation level. Replies to replies still record the actual target post as the backend parent but are displayed flattened under the nearest comment ancestor.

## 8. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | Reply comments must be deep-linkable from a share or future push notification. | Users need links/notifications to land on the intended conversation context. | Initial prompt; discovery Q3 | AC-001, AC-002, AC-003 |
| BR-002 | Business | Must | The post reply experience must behave as a root-post comment section rather than a recursive thread reader. | The desired UX is comments with optional reply expansion. | Discovery recommendation | AC-004, AC-005, AC-006 |
| BR-003 | Business | Must | Users must be able to choose comment ordering options for oldest, newest, and follows. | The prompt requires an ordering dropdown. | Initial prompt; discovery Q2/Q6; terminology decision | AC-007 |
| BR-004 | Business | Must | New replies must be brought into view after creation. | Posting should provide immediate confirmation and context. | Initial prompt; discovery Q4 | AC-011, AC-012 |
| FR-001 | Functional | Must | The system shall remove `GET /v1/posts/{did}/{rkey}/thread` and thread-specific Flutter API/model/provider usage. | The app is pre-production and the thread route no longer matches the product model. | Discovery Q7 | AC-013 |
| FR-002 | Functional | Must | The system shall provide a comment-section read surface for a root post that returns the root post, a comment page, comment pagination state, comment sort state, placement metadata, reply loaded-state metadata, and focus context when requested. | The client needs one root-post-oriented contract for initial render and deep-link focus. | Discovery recommendation; grill-me decisions | AC-001, AC-004, AC-008, AC-022, AC-023 |
| FR-003 | Functional | Must | The root post route shall support `focus` as a URL-encoded AT-URI query parameter for opening a specific comment or reply inside the root post comment section. | Query-param focus keeps the root post as the primary destination while identifying the target item. | Discovery Q3; grill-me Q2 | AC-001, AC-002, AC-003 |
| FR-004 | Functional | Must | Opening a focused link shall include and render the focused comment branch even when the focused comment or focused reply's comment ancestor would not appear in the first comment page. | Deep links must be reliable independent of pagination. | Discovery Q3; grill-me Q12/Q13 | AC-002, AC-003, AC-014, AC-021 |
| FR-005 | Functional | Must | Opening a post without a focus parameter shall initially render only the root post and comments. | The comment section should not expand replies by default. | Initial prompt; discovery; terminology decision | AC-004 |
| FR-006 | Functional | Must | Comments shall load in pages of 10 and request additional pages lazily as the user scrolls near the end of the loaded list. | The prompt requires loading 10 more and discovery chose scroll-driven comment loading. | Initial prompt; discovery Q8; terminology decision | AC-005, AC-009 |
| FR-007 | Functional | Must | Comment ordering shall support `oldest`, `newest`, and `follows`, with ordering applied only to comments after focus promotion and viewer-authored grouping. | Sorting replies would conflict with conversation order and was explicitly excluded. | Discovery Q2/Q6; Plannotator update; grill-me Q13/Q14 | AC-007, AC-010, AC-020, AC-024 |
| FR-008 | Functional | Must | The `follows` ordering option shall be visible but behave like `oldest` until follow data exists. | A real follows graph is out of scope. | User answer; Plannotator-applied discovery update | AC-007 |
| FR-009 | Functional | Must | A comment with replies shall show a “view replies” control before replies are loaded. | Users need an explicit expansion affordance. | Initial prompt; discovery Q8; terminology decision | AC-006 |
| FR-010 | Functional | Must | Activating “view replies” shall load the first 10 visual replies oldest-first under that comment branch, including direct replies and deeper descendants flattened into that branch. | Replies are user-actioned, bounded, and always displayed within the two-level comment-section model. | Discovery Q5/Q6/Q8; terminology decision; implementation review decision on 2026-05-15 | AC-006, AC-010, AC-026 |
| FR-011 | Functional | Must | Expanded replies shall provide “load more” when additional replies are available, loading 10 more oldest-first per activation. | The prompt requires loading more comments/replies in increments of 10. | Initial prompt; discovery Q8 | AC-009, AC-010 |
| FR-012 | Functional | Must | Expanded replies shall provide “hide replies” where “view replies” was, and hiding shall collapse the loaded reply list without deleting reply data. | Users requested a hide control replacing the view control. | User answer | AC-006 |
| FR-013 | Functional | Must | The UI shall display no more than two visual levels: comments and one indented reply list. | The prompt requires maximum two levels. | Initial prompt; discovery Q1; terminology decision | AC-014 |
| FR-014 | Functional | Must | When replying to a reply, the created record shall preserve the actual target reply as backend parent while the UI displays the created reply flattened under the nearest comment ancestor. | Preserve atproto threading fidelity without third-level UI nesting. | Discovery Q1; Plannotator update; terminology decision | AC-012, AC-014, AC-015, AC-026 |
| FR-015 | Functional | Must | The composer for a reply to a reply shall include an `@handle` mention for the target author. | The mention supplies visible context when the UI flattens deeper replies. | User answer | AC-015 |
| FR-016 | Functional | Must | A newly-created comment shall appear in the viewer-authored comment group and be scrolled into view without changing the selected sort. | User-authored comments always appear at the top under the updated product rule. | Discovery Q4; Plannotator update; terminology decision | AC-011, AC-020 |
| FR-017 | Functional | Must | A newly-created reply shall be inserted/displayed within the relevant comment's reply list and scrolled into view. | Reply creation should visibly confirm the correct branch. | Initial prompt; discovery Q1/Q4; terminology decision | AC-012 |
| FR-018 | Functional | Should | The comment-section UI should expose clear localized labels for ordering, “view replies”, “load more”, “hide replies”, and focused reply states. | The Flutter UI already uses generated localization patterns. | Discovery handoff | AC-016 |
| FR-019 | Functional | Must | Focus validation/status shall be performed by the backend comment-section response. | Backend has indexed root/parent refs and should provide one consistent focus contract. | Grill-me Q4 | AC-025 |
| FR-020 | Functional | Must | Malformed `focus` shall return `400 invalid_focus`; well-formed unavailable focus shall return `200` root comment-section content with `focus.status = "notFound"`; well-formed focus under another root shall return `200` root comment-section content with `focus.status = "mismatchedRoot"`. | Separates bad link syntax from distributed indexing/deletion cases. | Grill-me Q5-Q8 | AC-025 |
| FR-021 | Functional | Must | Included focus shall return `focus.status = "included"`, echo `focus.uri`, include `kind: "comment" | "reply"`, and include `commentUri` when `kind = "reply"`. | Client needs placement metadata for scroll/focus without reconstructing thread membership. | Grill-me Q9-Q11 | AC-001, AC-003, AC-025 |
| FR-022 | Functional | Must | The comment-section response shall expose one ordered `comments.items` array, and every comment item shall include required `placement: "focused" | "viewerAuthored" | "normal"`. | One render source of truth avoids client-side merge ambiguity. | Grill-me Q17-Q18 | AC-022, AC-024 |
| FR-023 | Functional | Must | Every comment item shall include a `replies` object with explicit `loaded` state and `items`; `cursor` is present only when more replies can be loaded. | Avoids ambiguous null/empty reply state. | Grill-me Q19 | AC-023 |
| FR-024 | Functional | Must | Reply items shall include backend structural metadata: `flattened` on every reply item and `replyingTo` with `uri`, `did`, `handle`, and optional `displayName` when `flattened = true`. | Backend owns structural truth for deeper replies flattened into the two-level UI. | Grill-me Q20-Q21 | AC-026 |
| NFR-001 | Non-functional | Must | Comment list APIs shall use existing `/v1/` API conventions: authenticated requests, camelCase JSON, error envelopes, and opaque cursors. | Maintains AppView API consistency. | AGENTS.md; API specs | AC-017 |
| NFR-002 | Non-functional | Must | Reply loading shall be bounded to avoid unbounded recursive traversal or unbounded response sizes. | Protects performance and keeps UX predictable. | Discovery; current capped implementation | AC-005, AC-006, AC-009, AC-021 |
| NFR-003 | Non-functional | Should | Focus, grouping, and lazy-loading behavior should avoid duplicate rendered comments/replies when a focused branch overlaps with a loaded page or viewer-authored group. | Prevents confusing duplicate comments. | Requirements analysis; Plannotator update | AC-018 |
| NFR-004 | Non-functional | Must | Focus promotion and viewer-authored grouping shall not duplicate comments when promoted comments are encountered later through normal cursor pagination. | The focused/comment grouping model intentionally promotes items out of normal order. | Grill-me Q16 | AC-018, AC-024 |
| RULE-001 | Business rule | Must | Replies remain `social.craftsky.feed.post` records with existing root and parent reply refs; this change must not require lexicon changes. | Existing lexicon already supports required parentage. | Discovery constraints | AC-015, AC-019 |
| RULE-002 | Business rule | Must | Comments are replies whose parent is the root post. | Defines what appears in the root comment list. | Current data model; discovery; terminology decision | AC-004, AC-007 |
| RULE-003 | Business rule | Must | Visual replies are records displayed under a comment, including direct replies and deeper descendants flattened to the nearest comment branch. | Keeps visual nesting capped while preserving parentage. | Discovery Q1; Plannotator update; terminology decision; implementation review decision on 2026-05-15 | AC-006, AC-010, AC-012, AC-014, AC-026 |
| RULE-004 | Business rule | Must | Reply lists are always ordered oldest-first. | Discovery explicitly fixed nested ordering. | Discovery Q6 | AC-010 |
| RULE-005 | Business rule | Must | Comment `follows` sort behaves as `oldest` until follow data exists. | Avoids fake ranking and keeps the dropdown stable. | Discovery Q2 update | AC-007 |
| RULE-006 | Business rule | Must | Display precedence is focused comment branch first, then viewer-authored comments, then remaining comments; selected comment sort applies within viewer-authored and remaining groups. | Deep-link intent outranks viewer grouping; viewer grouping outranks normal comments. | Plannotator update; grill-me Q13 | AC-007, AC-011, AC-020, AC-024 |
| RULE-007 | Business rule | Must | A focused visual reply outside the first reply page is loaded as a bounded focused slice for its comment branch, not by loading every earlier reply up to the focused reply. | Keeps focus behavior reliable without unbounded reply traversal. | Plannotator feedback; grill-me Q13 | AC-003, AC-021 |
| RULE-008 | Business rule | Must | Focus promotion persists while the user scrolls but is cleared by explicit sort/filter changes. | Focus is an entry affordance; sort/filter is a new ordering intent. | Grill-me Q14-Q15 | AC-024 |

## 9. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-002, FR-003, FR-021 | Given a valid root post route with a URL-encoded `focus` AT-URI for an indexed comment or reply, when the route is opened, then the root post comment section loads and identifies the focused target. |
| AC-002 | BR-001, FR-004 | Given the focused comment or focused reply's comment ancestor is outside the first comment page, when the comment section loads, then the focused comment branch is included without requiring the user to scroll/load intermediate comment pages first. |
| AC-003 | BR-001, FR-003, FR-004, FR-021, RULE-007 | Given the focused item is a reply under a comment, when the comment section loads, then the relevant comment branch is expanded and the focused reply is scrolled or otherwise brought into view. |
| AC-004 | BR-002, FR-005, RULE-002 | Given a user opens a root post without a focus parameter, when the initial comment section renders, then only the root post and comments are visible. |
| AC-005 | BR-002, FR-006, NFR-002 | Given more than 10 comments exist, when the user scrolls near the end of loaded comments, then the next page of up to 10 comments is requested and appended. |
| AC-006 | BR-002, FR-009, FR-010, FR-012, NFR-002, RULE-003 | Given a comment has replies, when replies are not loaded, then “view replies” is shown; when activated, up to 10 visual replies for that comment branch load, including flattened descendants when present, and the control changes to “hide replies”; when “hide replies” is activated, the reply list collapses. |
| AC-007 | BR-003, FR-007, FR-008, RULE-002, RULE-005, RULE-006 | Given the ordering dropdown is used, when `oldest`, `newest`, or `follows` is selected, then viewer-authored comments remain grouped before normal comments, each group is ordered oldest-first, newest-first, or oldest-first respectively, and reply order is unaffected. |
| AC-008 | FR-002 | Given the root comment-section read surface succeeds, when the client decodes the response, then it has the root post, comment items, an omitted/opaque next cursor when applicable, selected sort, required placement metadata, reply loaded-state metadata, and any focus metadata needed for rendering. |
| AC-009 | FR-006, FR-011, NFR-002 | Given more comments or expanded replies are available, when more items are requested, then each request loads no more than 10 additional records and preserves already-loaded records. |
| AC-010 | FR-007, FR-010, FR-011, RULE-003, RULE-004 | Given replies are expanded or more replies are loaded, when they render, then direct replies and flattened descendants for that comment branch are ordered oldest-first regardless of the selected comment sort. |
| AC-011 | BR-004, FR-016, RULE-006 | Given a comment is successfully created, when the comment section updates, then the new comment appears in the viewer-authored comment group and is scrolled into view without changing the selected sort. |
| AC-012 | BR-004, FR-014, FR-017, RULE-003 | Given a reply to another reply is successfully created, when the comment section updates, then the new reply is displayed in the relevant comment's reply list and scrolled into view. |
| AC-013 | FR-001 | Given the app and AppView are updated, when code references and routes are inspected, then `/v1/posts/{did}/{rkey}/thread` and Flutter thread-specific API/model/provider usage are removed or replaced by comment-section equivalents. |
| AC-014 | FR-004, FR-013, FR-014, RULE-003 | Given replies exist deeper than two backend levels, when they are displayed due to focus or new reply insertion, then no third visual indentation level is rendered. |
| AC-015 | FR-014, FR-015, RULE-001 | Given a user replies to a reply, when the create request is made, then the reply parent ref targets the actual target post, the root ref remains the root post, and the composer includes the target author mention. |
| AC-016 | FR-018 | Given the comment section renders controls, when labels are shown, then ordering, view, load more, hide, and focus-related text use the app localization mechanism rather than hard-coded one-off strings. |
| AC-017 | NFR-001 | Given comment-section endpoints return success or error responses, when the wire bodies are inspected, then JSON keys are camelCase, errors use the standard envelope, and pagination cursors are opaque client-round-tripped strings. |
| AC-018 | NFR-003, NFR-004 | Given a focused comment branch or viewer-authored comment also appears in a loaded page, when the list renders, then the comment appears only once in the visible comment section. |
| AC-019 | RULE-001 | Given the feature is implemented, when lexicon files are inspected, then no reply/comment lexicon change is required for this behavior. |
| AC-020 | FR-007, FR-016, RULE-006 | Given the viewer has authored one or more comments on the root post, when the comment list renders or paginates, then those comments appear before non-viewer-authored normal comments without duplicating entries in later pages. |
| AC-021 | FR-004, NFR-002, RULE-007 | Given a focused visual reply is outside the first 10 replies for its comment branch, when the comment section loads, then the response renders a bounded focused reply slice containing the target and preserves pagination controls for loading additional replies predictably. |
| AC-022 | FR-022 | Given the comment-section response contains comments, when `comments.items` is inspected, then every item has required `placement` set to `focused`, `viewerAuthored`, or `normal`, and array order is the render order. |
| AC-023 | FR-023 | Given the comment-section response contains comments, when each comment item is inspected, then every item has a `replies` object with `loaded` and `items`; `cursor` appears only when additional replies can be loaded. |
| AC-024 | FR-007, FR-022, NFR-004, RULE-006, RULE-008 | Given focus promotion is active, when the comment section renders, then the focused comment branch appears before viewer-authored comments and normal comments; when the user changes sort/filter, focus promotion clears and normal viewer-authored grouping applies. |
| AC-025 | FR-019, FR-020, FR-021 | Given `focus` is present, when it is malformed, unavailable, mismatched, or included, then the API returns `400 invalid_focus`, `200` with `focus.status = "notFound"`, `200` with `focus.status = "mismatchedRoot"`, or `200` with `focus.status = "included"` and placement metadata respectively. |
| AC-026 | FR-014, FR-024 | Given a deeper backend reply is displayed as a visual reply under a comment, when the reply item is inspected, then `flattened = true` and `replyingTo` contains `uri`, `did`, `handle`, and optional `displayName`; direct replies have `flattened = false`. |

## 10. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | Focused item is the root post itself. | Treat as an unfocused root post view or focus the root post without expanding replies. | FR-003, FR-004 |
| EC-002 | Focus AT-URI is malformed. | Return `400 invalid_focus` using the standard API error envelope. | FR-003, FR-020, NFR-001 |
| EC-003 | Focused comment/reply is not indexed, deleted, or no longer available. | Load the root post comment section as if clicked normally and include `focus.status = "notFound"`; return not found only if root cannot be resolved. | BR-001, FR-004, FR-020 |
| EC-004 | Focused comment/reply belongs to a different root than the route root. | Load the requested root post comment section, include `focus.status = "mismatchedRoot"`, and do not redirect. | FR-004, FR-020, RULE-001 |
| EC-005 | User hides replies after loading several pages. | Collapse the reply list while preserving enough state to re-show loaded replies or reload predictably; no replies are deleted. | FR-012 |
| EC-006 | User changes comment sort after pages have loaded. | Reset/reload comment pagination under the new sort, clear focus promotion, and avoid mixing cursors from different sorts. | FR-007, RULE-008, NFR-001 |
| EC-007 | User selects `follows`. | Display/select the option but order comments oldest-first within placement groups. | FR-008, RULE-005 |
| EC-008 | Newly-created comment is later encountered in a fetched page. | De-duplicate against the viewer-authored comment group item. | FR-016, NFR-003, NFR-004 |
| EC-009 | Reply count changes while a comment branch is expanded. | Preserve currently-loaded replies and allow subsequent load-more/refresh to converge with server state. | FR-011, NFR-002 |
| EC-010 | Reply is authored by a user whose handle cannot be resolved. | Use existing identity error behavior for API failures or existing fallback display behavior if available; do not invent a new token/storage flow. | NFR-001 |
| EC-011 | Focused visual reply is beyond the first 10 replies. | Load a bounded focused reply slice containing the focused reply rather than all preceding replies, and expose enough pagination state for predictable load-more behavior. | FR-004, RULE-007 |
| EC-012 | Viewer-authored comment would naturally sort into a later page. | Surface it in the viewer-authored group and de-duplicate it from later paginated results. | FR-007, FR-016, RULE-006, NFR-003, NFR-004 |
| EC-013 | Focused comment is also viewer-authored. | Render it once with `placement = "focused"`, not again as `viewerAuthored`. | FR-022, NFR-004, RULE-006 |

## 11. Data / Persistence Impact

- New fields: None required in lexicon or existing post storage.
- Changed fields: None required for `craftsky_posts`; existing `reply_root_*` and `reply_parent_*` fields remain source of truth.
- Migration required: No migration is expected for reply structure. If implementation adds optimization-only indexes or denormalized counters, that must be justified separately during design/implementation.
- Backwards compatibility: `/v1/posts/{did}/{rkey}/thread` may be removed because the app is not in production. Existing app code must be updated in the same change.

## 12. UI / API / CLI Impact

- UI:
  - Replace the recursive thread page with a root post comment section at `/posts/:did/:rkey`.
  - Add support for `focus=<url-encoded AT-URI>` on the post route.
  - Add comment ordering dropdown: `oldest`, `newest`, `follows`.
  - Add comment scroll-driven lazy loading.
  - Add per-comment `view replies`, `load more`, and `hide replies` controls.
  - Add focused comment branch promotion, viewer-authored comment grouping, and scroll/focus behavior for new/focused comments and replies.
- API:
  - Remove `GET /v1/posts/{did}/{rkey}/thread`.
  - Add or update root comment-section read surface under `/v1/` for root post plus comments, sort, cursor, placement, replies loaded-state, and focus context.
  - Include focused branch promotion, viewer-authored comment grouping, bounded focused reply slices, and flattened reply metadata in the comment-section response contract.
  - Keep or update comment-branch reply loading for reply expansion, bounded to 10 visual replies per page, oldest-first, with deeper descendants flattened under the nearest comment branch.
  - Use `limit`, `cursor`, and camelCase response keys per API conventions.
- CLI: None identified.
- Background jobs: None identified.

## 13. Security / Privacy / Permissions

- Authentication: Comment-section read endpoints remain authenticated `/v1/*` endpoints requiring the existing Craftsky session token and device ID unless the broader API policy changes separately.
- Authorization: No new authoring permissions beyond existing post creation/deletion rules are introduced. Reply creation continues through existing authenticated PDS write flow.
- Sensitive data: No private data is added; replies remain public atproto records.
- Abuse cases: This change does not add moderation, blocks, mutes, reports, rate limiting, or spam controls. Those remain out of scope and should be considered in future moderation work.

## 14. Observability

- Events: None required for this requirements stage. Implementation may add client analytics only if a separate analytics policy exists.
- Logs: Backend failures for comment-section reads should follow existing AppView handler logging patterns with request IDs, without logging sensitive tokens.
- Metrics: None required. Useful future metrics include comment-section load latency, focus-resolution failures, and pagination error counts.
- Alerts: None required.

## 15. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Focused comment/reply inclusion may complicate backend queries and response shape. | Deep links could fail or require multiple round trips. | Make focused branch inclusion an explicit API requirement and test it with comments/replies outside the first page. |
| RISK-002 | Removing `/thread` could leave stale client references. | Runtime route/API failures. | Include route/client removal in acceptance criteria and regression tests. |
| RISK-003 | Focus promotion, viewer-authored grouping, and pagination may duplicate comments. | Confusing UI and incorrect perceived counts. | Require visible de-duplication and test overlap cases. |
| RISK-004 | `follows` as no-op may confuse users. | Users may expect personalized sorting. | Treat `follows` as a visible stub only for this pre-follow-graph phase; consider copy or disabled styling during UX implementation. |
| RISK-005 | Flattening deeper replies can hide true parent context. | Conversation context may be unclear. | Include composer mention and keep an open product/design question for additional “replying to” treatment. |

## 16. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | The focus query parameter is named `focus` and carries a URL-encoded AT-URI. | Route/API requirements and tests must be updated if a structured `{did}/{rkey}` pair is preferred. |
| ASM-002 | Reply lists are visual comment-branch pages for normal expansion; deeper descendants appear flattened under the nearest comment branch. | If only direct children are desired later, backend query and UI tests must be changed to exclude descendants from normal expansion. |
| ASM-003 | Viewer identity is available to the comment-section API/client so viewer-authored comments can be grouped before other normal comments. | If viewer identity is unavailable at the ordering layer, the API/client contract must add or expose it. |
| ASM-004 | The app remains pre-production, so removing `/thread` within `/v1/` is acceptable without a compatibility period. | If external clients depend on `/thread`, deprecation or versioning would be required. |
| ASM-005 | No lexicon or persistence migration is needed for reply parentage. | If query performance requires denormalized depth/top-level ancestor fields, additional schema work may be needed. |

## 17. Review Status

Status: Draft
Risk level: Medium
Review recommended: Yes
Reviewer: Unassigned
Date: 2026-05-12
Notes: Review is recommended before test design because this changes both AppView API behavior and Flutter UI state/navigation behavior.

## 18. Handoff To Test Design

- Requirements file: `02-requirements.md`
- Must-cover requirement IDs:
  - `BR-001` through `BR-004`
  - `FR-001` through `FR-024`
  - `NFR-001` through `NFR-004`
  - `RULE-001` through `RULE-008`
- Suggested test levels:
  - Backend handler/store tests for comment-section response, focus status/promotion, placement metadata, replies loaded-state metadata, flattened reply metadata, viewer-authored grouping, sorting, pagination, reply loading, and `/thread` removal.
  - API contract tests for camelCase JSON, standard errors, opaque cursor behavior, and invalid focus/cursor handling.
  - Flutter provider/state tests for comment lazy loading, per-comment expansion/collapse, de-duplication, sort changes, focus promotion clearing, and viewer-authored grouping.
  - Flutter widget tests for initial comments-only render, view/load/hide controls, no third indentation level, ordering dropdown, and focus/scroll behavior where feasible.
  - Regression tests to ensure existing post create/delete/like/repost behavior remains unaffected.
- Blocking open questions: None for test design. Non-blocking product/design questions remain around exact flattened-reply context labeling.
