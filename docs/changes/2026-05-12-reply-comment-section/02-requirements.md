# Business Requirements: Reply Comment Section

## 1. Summary

Replace the existing recursive post thread experience with a root-post comment section for replies. The new experience must support deep links to individual replies, top-level reply sorting and lazy loading, action-expanded second-level replies, and a maximum of two visible reply levels while preserving full backend reply parentage.

## 2. Problem / Opportunity

The current `/posts/:did/:rkey` screen behaves like a recursive thread reader: it can show ancestors above the selected post and nested descendants below it. The desired product experience is a comment section under a root post: opening a post shows only top-level replies, users can expand child replies when desired, and reply links from shares or push notifications can focus a specific comment inside that root-post context.

## 3. Goals

- G-001: Make every reply comment deep-linkable from shares and future push notifications.
- G-002: Present replies as a predictable two-level comment section under the root post.
- G-003: Support top-level reply ordering and efficient incremental loading.
- G-004: Preserve exact atproto reply parentage in backend data even when the UI flattens deeper visual nesting.
- G-005: Remove the pre-production `/thread` route and thread-specific client model to avoid parallel reply paradigms.

## 4. Non-Goals

- NG-001: Implementing a real follows graph, follow indexing, or true follows-based reply ranking.
- NG-002: Changing the `social.craftsky.feed.post` lexicon or creating a separate comment record type.
- NG-003: Displaying infinite or third-level-and-deeper visual nesting.
- NG-004: Building push notification registration or delivery infrastructure.
- NG-005: Maintaining production backward compatibility for `/v1/posts/{did}/{rkey}/thread`.
- NG-006: Loading every intermediate reply before a focused second-level reply that is outside the first child page.

## 5. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Viewer | Signed-in Craftsky user reading a post and its comments. | See top-level replies first, expand child replies intentionally, choose top-level ordering, and load more replies without losing place. |
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

Opening a root post route shows the root post and top-level replies only. Top-level replies load 10 at a time as the user scrolls and can be ordered by `oldest`, `newest`, or `follows`, where `follows` is visible but behaves like `oldest` until real follow data exists. Viewer-authored top-level replies always appear in a top group ahead of other top-level replies, with the selected sort applied inside each group. If a top-level reply has child replies, the UI shows “view replies”; tapping it loads 10 second-level replies oldest-first, then changes the control to “hide replies” and shows “load more” when more child replies exist.

Reply deep links use the root post route plus a focused reply query parameter. Opening such a link includes the focused reply branch even if it is outside the first top-level page, expands the relevant branch if needed, and scrolls/focuses the target reply. If a focused second-level visual reply is outside the first child page, the focused branch includes a bounded focused slice around the target rather than loading every earlier child reply. The UI never shows a third indentation level. Replies to second-level or deeper posts still record the actual target post as the backend parent but are displayed flattened under the nearest top-level ancestor.

## 8. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | Reply comments must be deep-linkable from a share or future push notification. | Users need links/notifications to land on the intended conversation context. | Initial prompt; discovery Q3 | AC-001, AC-002, AC-003 |
| BR-002 | Business | Must | The post reply experience must behave as a root-post comment section rather than a recursive thread reader. | The desired UX is top-level comments with optional child expansion. | Discovery recommendation | AC-004, AC-005, AC-006 |
| BR-003 | Business | Must | Users must be able to choose top-level reply ordering options for oldest, newest, and follows. | The prompt requires an ordering dropdown. | Initial prompt; discovery Q2/Q6 | AC-007 |
| BR-004 | Business | Must | New replies must be brought into view after creation. | Posting should provide immediate confirmation and context. | Initial prompt; discovery Q4 | AC-011, AC-012 |
| FR-001 | Functional | Must | The system shall remove `GET /v1/posts/{did}/{rkey}/thread` and thread-specific Flutter API/model/provider usage. | The app is pre-production and the thread route no longer matches the product model. | Discovery Q7 | AC-013 |
| FR-002 | Functional | Must | The system shall provide a comment-section read surface for a root post that returns the root post, a top-level reply page, top-level pagination state, top-level sort state, and focused-reply context when requested. | The client needs one root-post-oriented contract for initial render and deep-link focus. | Discovery recommendation | AC-001, AC-004, AC-008 |
| FR-003 | Functional | Must | The root post route shall support a focused reply query parameter, assumed to be a URL-encoded AT-URI in `focus`, for opening a specific reply inside the root post comment section. | Query-param focus keeps the root post as the primary destination while identifying the target reply. | Discovery Q3; API URL conventions | AC-001, AC-002, AC-003 |
| FR-004 | Functional | Must | Opening a focused reply link shall include and render the focused reply branch even when the focused top-level ancestor would not appear in the first top-level reply page. | Deep links must be reliable independent of pagination. | Discovery Q3 | AC-002, AC-003, AC-014 |
| FR-005 | Functional | Must | Opening a post without a focus parameter shall initially render only the root post and top-level replies. | The comment section should not expand child replies by default. | Initial prompt; discovery | AC-004 |
| FR-006 | Functional | Must | Top-level replies shall load in pages of 10 and request additional pages lazily as the user scrolls near the end of the loaded list. | The prompt requires loading 10 more and discovery chose scroll-driven top-level loading. | Initial prompt; discovery Q8 | AC-005, AC-009 |
| FR-007 | Functional | Must | Top-level reply ordering shall support `oldest`, `newest`, and `follows`, with ordering applied only to top-level replies after viewer-authored grouping. | Sorting child replies would conflict with conversation order and was explicitly excluded. | Discovery Q2/Q6; Plannotator update | AC-007, AC-010, AC-020 |
| FR-008 | Functional | Must | The `follows` ordering option shall be visible but behave like `oldest` until follow data exists. | A real follows graph is out of scope. | User answer; Plannotator-applied discovery update | AC-007 |
| FR-009 | Functional | Must | A top-level reply with child replies shall show a “view replies” control before child replies are loaded. | Users need an explicit expansion affordance. | Initial prompt; discovery Q8 | AC-006 |
| FR-010 | Functional | Must | Activating “view replies” shall load the first 10 direct child replies oldest-first under that top-level reply. | Nested replies are user-actioned and bounded. | Discovery Q5/Q6/Q8 | AC-006, AC-010 |
| FR-011 | Functional | Must | Expanded child replies shall provide “load more” when additional child replies are available, loading 10 more oldest-first per activation. | The prompt requires loading more comments in increments of 10. | Initial prompt; discovery Q8 | AC-009, AC-010 |
| FR-012 | Functional | Must | Expanded child replies shall provide “hide replies” where “view replies” was, and hiding shall collapse the loaded child list without deleting reply data. | Users requested a hide control replacing the view control. | User answer | AC-006 |
| FR-013 | Functional | Must | The UI shall display no more than two visual reply levels: top-level replies and one indented second-level list. | The prompt requires maximum two levels. | Initial prompt; discovery Q1 | AC-014 |
| FR-014 | Functional | Must | When replying to a second-level or deeper reply, the created record shall preserve the actual target reply as backend parent while the UI displays the created reply flattened under the nearest top-level ancestor. | Preserve atproto threading fidelity without third-level UI nesting. | Discovery Q1; Plannotator update | AC-012, AC-014, AC-015 |
| FR-015 | Functional | Must | The composer for a reply to a second-level or deeper reply shall include an `@handle` mention for the target author. | The mention supplies visible context when the UI flattens deeper replies. | User answer | AC-015 |
| FR-016 | Functional | Must | A newly-created top-level reply shall appear in the viewer-authored top-level reply group and be scrolled into view without changing the selected sort. | User-authored top-level replies always appear at the top under the updated product rule. | Discovery Q4; Plannotator update | AC-011, AC-020 |
| FR-017 | Functional | Must | A newly-created reply to another reply shall be inserted/displayed within the relevant second-level list and scrolled into view. | Reply creation should visibly confirm the correct branch. | Initial prompt; discovery Q1/Q4 | AC-012 |
| FR-018 | Functional | Should | The comment-section UI should expose clear localized labels for ordering, “view replies”, “load more”, “hide replies”, and focused reply states. | The Flutter UI already uses generated localization patterns. | Discovery handoff | AC-016 |
| NFR-001 | Non-functional | Must | Comment list APIs shall use existing `/v1/` API conventions: authenticated requests, camelCase JSON, error envelopes, and opaque cursors. | Maintains AppView API consistency. | AGENTS.md; API specs | AC-017 |
| NFR-002 | Non-functional | Must | Reply loading shall be bounded to avoid unbounded recursive traversal or unbounded response sizes. | Protects performance and keeps UX predictable. | Discovery; current capped implementation | AC-005, AC-006, AC-009 |
| NFR-003 | Non-functional | Should | Focus, grouping, and lazy-loading behavior should avoid duplicate rendered replies when a focused branch overlaps with a loaded page or viewer-authored top group. | Prevents confusing duplicate comments. | Requirements analysis; Plannotator update | AC-018 |
| RULE-001 | Business rule | Must | Replies remain `social.craftsky.feed.post` records with existing root and parent reply refs; this change must not require lexicon changes. | Existing lexicon already supports required parentage. | Discovery constraints | AC-015, AC-019 |
| RULE-002 | Business rule | Must | Top-level replies are replies whose parent is the root post. | Defines what appears in the root comment list. | Current data model; discovery | AC-004, AC-007 |
| RULE-003 | Business rule | Must | Second-level visual replies are replies displayed under a top-level reply, including direct children and deeper focused/newly-created replies flattened to the nearest top-level branch. | Keeps visual nesting capped while preserving parentage. | Discovery Q1; Plannotator update | AC-012, AC-014 |
| RULE-004 | Business rule | Must | Second-level reply lists are always ordered oldest-first. | Discovery explicitly fixed nested ordering. | Discovery Q6 | AC-010 |
| RULE-005 | Business rule | Must | Top-level `follows` sort behaves as `oldest` until follow data exists. | Avoids fake ranking and keeps the dropdown stable. | Discovery Q2 update | AC-007 |
| RULE-006 | Business rule | Must | Viewer-authored top-level replies always appear before other top-level replies, with the selected top-level sort applied within the viewer-authored group and within the remaining-replies group. | Updated product rule from review. | Plannotator update | AC-007, AC-011, AC-020 |
| RULE-007 | Business rule | Must | A focused second-level visual reply outside the first child page is loaded as a bounded focused slice for its branch, not by loading every earlier child reply up to the focused reply. | Keeps focus behavior reliable without unbounded child traversal. | Plannotator feedback | AC-003, AC-021 |

## 9. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-002, FR-003 | Given a valid root post route with a URL-encoded `focus` AT-URI for an indexed reply, when the route is opened, then the root post comment section loads and identifies the focused reply target. |
| AC-002 | BR-001, FR-004 | Given the focused reply's top-level ancestor is outside the first top-level reply page, when the comment section loads, then the focused reply branch is included without requiring the user to scroll/load intermediate top-level pages first. |
| AC-003 | BR-001, FR-003, FR-004, RULE-007 | Given the focused reply is nested under a top-level reply, when the comment section loads, then the relevant branch is expanded and the focused reply is scrolled or otherwise brought into view. |
| AC-004 | BR-002, FR-005, RULE-002 | Given a user opens a root post without a focus parameter, when the initial comment section renders, then only the root post and top-level replies are visible. |
| AC-005 | BR-002, FR-006, NFR-002 | Given more than 10 top-level replies exist, when the user scrolls near the end of loaded top-level replies, then the next page of up to 10 top-level replies is requested and appended. |
| AC-006 | BR-002, FR-009, FR-010, FR-012, NFR-002 | Given a top-level reply has child replies, when child replies are not loaded, then “view replies” is shown; when activated, up to 10 child replies load and the control changes to “hide replies”; when “hide replies” is activated, the child list collapses. |
| AC-007 | BR-003, FR-007, FR-008, RULE-002, RULE-005, RULE-006 | Given the ordering dropdown is used, when `oldest`, `newest`, or `follows` is selected, then viewer-authored top-level replies remain grouped first, each top-level group is ordered oldest-first, newest-first, or oldest-first respectively, and second-level reply order is unaffected. |
| AC-008 | FR-002 | Given the root comment-section read surface succeeds, when the client decodes the response, then it has the root post, top-level reply items, an omitted/opaque next cursor when applicable, selected sort, and any focused reply metadata needed for rendering. |
| AC-009 | FR-006, FR-011, NFR-002 | Given more top-level or expanded child replies are available, when more replies are requested, then each request loads no more than 10 additional replies and preserves already-loaded replies. |
| AC-010 | FR-007, FR-010, FR-011, RULE-004 | Given child replies are expanded or more child replies are loaded, when they render, then they are ordered oldest-first regardless of the selected top-level sort. |
| AC-011 | BR-004, FR-016, RULE-006 | Given a top-level reply is successfully created, when the comment section updates, then the new reply appears in the viewer-authored top-level group and is scrolled into view without changing the selected sort. |
| AC-012 | BR-004, FR-014, FR-017, RULE-003 | Given a reply to another reply is successfully created, when the comment section updates, then the new reply is displayed in the relevant second-level list and scrolled into view. |
| AC-013 | FR-001 | Given the app and AppView are updated, when code references and routes are inspected, then `/v1/posts/{did}/{rkey}/thread` and Flutter thread-specific API/model/provider usage are removed or replaced by comment-section equivalents. |
| AC-014 | FR-004, FR-013, FR-014, RULE-003 | Given replies exist deeper than two backend levels, when they are displayed due to focus or new reply insertion, then no third visual indentation level is rendered. |
| AC-015 | FR-014, FR-015, RULE-001 | Given a user replies to a second-level reply, when the create request is made, then the reply parent ref targets the actual second-level post, the root ref remains the root post, and the composer includes the target author mention. |
| AC-016 | FR-018 | Given the comment section renders controls, when labels are shown, then ordering, view, load more, hide, and focus-related text use the app localization mechanism rather than hard-coded one-off strings. |
| AC-017 | NFR-001 | Given comment-section endpoints return success or error responses, when the wire bodies are inspected, then JSON keys are camelCase, errors use the standard envelope, and pagination cursors are opaque client-round-tripped strings. |
| AC-018 | NFR-003 | Given a focused reply or viewer-authored top-level reply also appears in a loaded page, when the list renders, then the reply appears only once in the visible comment section. |
| AC-019 | RULE-001 | Given the feature is implemented, when lexicon files are inspected, then no reply/comment lexicon change is required for this behavior. |
| AC-020 | FR-007, FR-016, RULE-006 | Given the viewer has authored one or more top-level replies on the root post, when the top-level reply list renders or paginates, then those replies appear before non-viewer-authored top-level replies without duplicating entries in later pages. |
| AC-021 | FR-004, NFR-002, RULE-007 | Given a focused second-level visual reply is outside the first 10 child replies for its top-level branch, when the comment section loads, then the response renders a bounded focused child slice containing the target and preserves pagination controls for loading additional child replies predictably. |

## 10. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | Focused reply is the root post itself. | Treat as an unfocused root post view or focus the root post without expanding replies. | FR-003, FR-004 |
| EC-002 | Focused reply AT-URI is malformed. | Return or surface a validation error using the standard API error envelope / client error handling. | FR-003, NFR-001 |
| EC-003 | Focused reply is not indexed or no longer available. | Load the root post if valid and show a non-blocking “reply unavailable” state, or return not found if root cannot be resolved. | BR-001, FR-004 |
| EC-004 | Focused reply belongs to a different root than the route root. | Do not render misleading context; reject focus as invalid for that root or redirect/reload using the actual root if available. | FR-004, RULE-001 |
| EC-005 | User hides child replies after loading several pages. | Collapse the child list while preserving enough state to re-show loaded replies or reload predictably; no replies are deleted. | FR-012 |
| EC-006 | User changes top-level sort after pages have loaded. | Reset/reload top-level pagination under the new sort and avoid mixing cursors from different sorts. | FR-007, NFR-001 |
| EC-007 | User selects `follows`. | Display/select the option but order top-level replies oldest-first. | FR-008, RULE-005 |
| EC-008 | Newly-created top-level reply is later encountered in a fetched page. | De-duplicate against the viewer-authored top group item. | FR-016, NFR-003 |
| EC-009 | Direct child reply count changes while a branch is expanded. | Preserve currently-loaded replies and allow subsequent load-more/refresh to converge with server state. | FR-011, NFR-002 |
| EC-010 | Reply is authored by a user whose handle cannot be resolved. | Use existing identity error behavior for API failures or existing fallback display behavior if available; do not invent a new token/storage flow. | NFR-001 |
| EC-011 | Focused second-level reply is beyond the first 10 child replies. | Load a bounded focused child slice containing the focused reply rather than all preceding child replies, and expose enough pagination state for predictable load-more behavior. | FR-004, RULE-007 |
| EC-012 | Viewer-authored top-level reply would naturally sort into a later page. | Surface it in the viewer-authored top group and de-duplicate it from later paginated results. | FR-007, FR-016, RULE-006, NFR-003 |

## 11. Data / Persistence Impact

- New fields: None required in lexicon or existing post storage.
- Changed fields: None required for `craftsky_posts`; existing `reply_root_*` and `reply_parent_*` fields remain source of truth.
- Migration required: No migration is expected for reply structure. If implementation adds optimization-only indexes or denormalized counters, that must be justified separately during design/implementation.
- Backwards compatibility: `/v1/posts/{did}/{rkey}/thread` may be removed because the app is not in production. Existing app code must be updated in the same change.

## 12. UI / API / CLI Impact

- UI:
  - Replace the recursive thread page with a root post comment section at `/posts/:did/:rkey`.
  - Add support for a focused reply query parameter on the post route.
  - Add top-level ordering dropdown: `oldest`, `newest`, `follows`.
  - Add top-level scroll-driven lazy loading.
  - Add per-top-level-reply `view replies`, `load more`, and `hide replies` controls.
  - Add viewer-authored top-level reply grouping and scroll/focus behavior for new/focused replies.
- API:
  - Remove `GET /v1/posts/{did}/{rkey}/thread`.
  - Add or update root comment-section read surface under `/v1/` for root post plus top-level replies, sort, cursor, and focus context.
  - Include viewer-authored top-level grouping and bounded focused child slices in the comment-section response contract.
  - Keep or update direct replies loading for child reply expansion, bounded to 10 per page and oldest-first for second-level use.
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
| RISK-001 | Focused reply inclusion may complicate backend queries and response shape. | Deep links could fail or require multiple round trips. | Make focused branch inclusion an explicit API requirement and test it with replies outside the first page. |
| RISK-002 | Removing `/thread` could leave stale client references. | Runtime route/API failures. | Include route/client removal in acceptance criteria and regression tests. |
| RISK-003 | Viewer-authored grouping plus pagination may duplicate comments. | Confusing UI and incorrect perceived counts. | Require visible de-duplication and test overlap cases. |
| RISK-004 | `follows` as no-op may confuse users. | Users may expect personalized sorting. | Treat `follows` as a visible stub only for this pre-follow-graph phase; consider copy or disabled styling during UX implementation. |
| RISK-005 | Flattening deeper replies can hide true parent context. | Conversation context may be unclear. | Include composer mention and keep an open product/design question for additional “replying to” treatment. |

## 16. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | The focused reply query parameter will be named `focus` and will carry a URL-encoded AT-URI. | Route/API requirements and tests must be updated if a structured `{did}/{rkey}` pair is preferred. |
| ASM-002 | Second-level reply lists are direct-child pages for normal expansion; deeper replies only appear flattened when focused or newly created from a deeper target. | If all deeper descendants must appear during normal expansion, backend query and UI rules become more complex. |
| ASM-003 | Viewer identity is available to the comment-section API/client so viewer-authored top-level replies can be grouped before other top-level replies. | If viewer identity is unavailable at the ordering layer, the API/client contract must add or expose it. |
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
  - `FR-001` through `FR-018`
  - `NFR-001`, `NFR-002`, `NFR-003`
  - `RULE-001` through `RULE-007`
- Suggested test levels:
  - Backend handler/store tests for comment-section response, viewer-authored grouping, sorting, pagination, focus inclusion, child reply loading, and `/thread` removal.
  - API contract tests for camelCase JSON, standard errors, opaque cursor behavior, and invalid focus/cursor handling.
  - Flutter provider/state tests for top-level lazy loading, per-branch expansion/collapse, de-duplication, sort changes, and viewer-authored grouping.
  - Flutter widget tests for initial top-level-only render, view/load/hide controls, no third indentation level, ordering dropdown, and focus/scroll behavior where feasible.
  - Regression tests to ensure existing post create/delete/like/repost behavior remains unaffected.
- Blocking open questions: None for test design if `ASM-001` is accepted. Non-blocking product/design questions remain around exact flattened-reply context labeling.
