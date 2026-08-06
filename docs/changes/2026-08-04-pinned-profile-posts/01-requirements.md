# Requirements: Pinned Profile Posts

## 1. Initial Request

Let users pin their own posts so pinned content appears first in the corresponding list on their profile. Each user may pin one project post and one top-level standard post. Pin state must exist only in the AppView, not as public PDS data. The owner pins from the post context menu; pinning a new post for an occupied slot replaces the old pin. On profile lists, every viewer sees a small pin icon and the exact text `Pinned post` at the top of the pinned card. For this slice, every authenticated current member may pin posts. It must contain no payment, plan-tier, or access-gating integration.

## 2. Current Codebase Findings

- Relevant files:
  - Product and architecture boundaries: `atproto-craft-social-app-reference.md`, `AGENTS.md`, and the linked product vision document, *Building a Social Network for Textile & Fibre Crafters*.
  - API conventions: `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md` and `docs/superpowers/specs/2026-04-22-api-wire-alignment-design.md`.
  - Shared card and context menu: `app/lib/feed/widgets/post_card.dart`.
  - Profile list presentation: `app/lib/profile/widgets/profile_tabs/profile_posts_tab.dart`, `profile_projects_tab.dart`, and `profile_post_feed_slivers.dart`.
  - Flutter post data seam: `app/lib/feed/models/post.dart`, `app/lib/feed/data/post_api_client.dart`, `user_posts_provider.dart`, and `app/lib/projects/providers/user_projects_provider.dart`.
  - AppView post routes and list hydration: `appview/internal/routes/routes.go`, `policy.go`, `appview/internal/api/post.go`, `post_store.go`, and `post_response.go`.
  - Private owner-scoped persistence precedent: `appview/migrations/000024_saved_posts.up.sql` and the saved-post handlers/store.
- Existing patterns:
  - `PostCard` is shared by feed, thread, profile, project, and search surfaces. Its menu already distinguishes owner actions such as Delete from other-viewer actions such as Report, Mute, and Block.
  - The profile Posts and Projects tabs use separate cursor-paginated providers and endpoints but share `ProfilePostFeedSlivers` and `PostCard`.
  - `GET /v1/profiles/{handleOrDid}/posts` returns top-level non-project posts. This includes top-level quote posts and excludes replies/comments.
  - `GET /v1/profiles/{handleOrDid}/projects` returns top-level project posts and excludes replies/comments and quote posts.
  - Both profile lists currently sort by `(profile_sort_at DESC, uri DESC)` and apply authentication, relationship policy, moderation policy, and the requesting viewer's content-language preferences before pagination.
  - Private saved-post state is keyed by authenticated owner DID and canonical post URI, remains out of the PDS, and cascades away when the owner membership or indexed target is permanently deleted.
- Current behavior:
  - There is no pin table, pin mutation, pin-aware list ordering, pin response field, pin menu action, or pinned-card annotation.
  - Profile Posts and Projects are chronological and paginated; an older highlighted post cannot currently appear before newer posts.
  - There is no existing access-tier gate in the Flutter client or AppView, and this slice does not add one.
- Constraints discovered:
  - Private-by-intent state belongs in AppView Postgres and requires no lexicon or PDS write.
  - New `/v1/*` routes must use authenticated DID context, device middleware, camelCase JSON, rate/body policies, and the standard `{error, message, requestId}` envelope.
  - Pin promotion must preserve opaque-cursor pagination without duplicating the pin on a later page, omitting another item, or exceeding the requested page limit.
  - A pin must not bypass existing membership, block, moderation, or language-visibility policy.
  - Every authenticated current member has the same pinning capability in this slice. No external account status or plan state participates in menu visibility, mutation authorization, storage, or profile-list presentation.
  - The current highest migration is `000034`; implementation must re-check this before allocating a migration number.
- Test/build commands discovered:
  - AppView gate: `just test` with the checkout's compose services available.
  - Go formatting/vetting: `just fmt`.
  - Flutter tests: `just app-test` or focused `flutter test` from `app/`.
  - Flutter analysis: `just app-analyze`.

## 3. Clarifying Questions And Decisions

### Q1: What occupies each pin slot?

Answer: The prompt establishes one project slot and one top-level standard-post slot. Existing profile-list definitions provide the concrete boundary.

Decision / implication: A standard pin may target a top-level non-project post, including a top-level quote post, but not a comment or nested reply. A project pin may target a top-level project post. The two slots are independent.

### Q2: What happens when the selected slot already has a pin?

Answer: The old post is unpinned and the new post is pinned.

Decision / implication: Replacement is one atomic AppView mutation. No observer may see two pins or an empty intermediate state for the slot.

### Q3: Can the current pin be removed without choosing a replacement?

Answer: Yes. The user confirmed that the current pinned post's context menu offers `Unpin post`.

Decision / implication: A user can return either slot to empty without choosing a replacement or deleting the post.

### Q4: Where is the pin action offered?

Answer: The user confirmed that the action appears on owner-authored cards that match a pin slot on surfaces that already expose owner actions: profile lists, timeline, and the top-level thread card. It does not add owner actions to search or general project-discovery cards solely for this feature.

Decision / implication: The shared menu can expose a consistent action without making the pinned annotation global. Only profile Posts and Projects lists show the annotation and pin-first ordering.

### Q5: Does a pin override content visibility or language preferences?

Answer: No. The user confirmed that existing profile-list policies continue to apply.

Decision / implication: A stored pin is promoted and labelled only when that post may be returned to that viewer in the corresponding profile list. A temporarily filtered or hidden pin remains stored and can reappear when visible again. It never bypasses a block, moderation decision, membership boundary, or content-language preference.

### Q6: Does this slice distinguish between categories of users when authorizing pinning?

Answer: No. The user explicitly confirmed that every authenticated current member may pin posts in this slice.

Decision / implication: Flutter and AppView shall not read, model, call, or reserve an integration seam for payment status, account plans, feature access, or future user categories. Pin authorization depends only on existing authentication/current-membership, post ownership, target type, and content-policy rules.

### Q7: Should future access policy be designed in this slice?

Answer: No. Future access policy is outside this change.

Decision / implication: Requirements, code, database schema, API contracts, UI copy, tests, and telemetry shall describe pinning only as a capability available to every authenticated current member. Pinning remains constrained to owner-controlled ordering on the owner's profile and does not boost timelines, search, discovery, notifications, or interaction counts.

### Q8: How should mutation interaction and feedback behave?

Answer: The user confirmed immediate replacement without confirmation, server-confirmed rather than optimistic reordering, disabled repeat actions while pending, and explicit feedback.

Decision / implication: Selecting `Pin post` replaces the occupied slot without a confirmation dialog. Flutter keeps the last confirmed card order while the request is pending and reconciles from the authoritative response. Pending state is scoped by active account and inferred slot: all Pin/Unpin actions for the affected slot are disabled to prevent competing replacements, while the independent slot remains usable. Success messages are `Post pinned` and `Post unpinned`; failures are `Couldn’t pin post. Try again.` and `Couldn’t unpin post. Try again.`

### Q9: How should the pinned-card annotation look and behave?

Answer: The user confirmed that it reuses the existing “Reposted by…” attribution slot and styling.

Decision / implication: The row uses the same typography, spacing, and subdued colour, replacing the repeat icon/text with a pin icon and exact text `Pinned post`. It is informational and has no independent tap action.

### Q10: How should first-page limits and cursor changes work?

Answer: The user confirmed that the pin consumes one normal page position and that cursors are bound to the pin state used to start traversal.

Decision / implication: At limit 10, page one contains the visible pin plus up to nine chronological posts. If that slot's pin changes before a later-page request, the AppView returns `400 invalid_cursor`; Flutter restarts from page one. This prevents silent duplication or omission.

### Q11: What AppView contract exposes and mutates pin state?

Answer: The user confirmed a private current-state read, target-specific mutations, authoritative mutation responses, page-level profile metadata, and no public timestamps.

Decision / implication: `GET /v1/profiles/me/pins` returns the authenticated member's two selected URIs. Bodyless `PUT /v1/posts/{did}/{rkey}/pin` and `DELETE /v1/posts/{did}/{rkey}/pin` pin or target-conditionally unpin; the authenticated owner and target path fully identify each operation. As a deliberate feature-specific exception to the API architecture's generic verb examples, all three endpoints return `200 OK` with authoritative two-slot state rather than requiring a PUT body or returning an empty DELETE `204`. This prevents a follow-up read race and lets stale no-ops reconcile correctly. A profile page includes page-level `pinnedPostUri` only on page one when a visible pin is promoted; it omits the key when there is no visible pin and on every later page. Flutter accepts absent or explicit null when decoding for compatibility but encodes the absent state by omitting the key. The shared canonical post model has no `isPinned` field, and pin timestamps remain internal.

### Q12: How are structural changes, moderation, invalid requests, and concurrency handled?

Answer: The user confirmed automatic cleanup for permanent/structural invalidity, retention for temporary policy hiding, explicit errors, last-committed replacement, and idempotent same-target pinning.

Decision / implication: Permanent post deletion, membership removal, or an indexed post update that makes the target structurally invalid clears the pin; type changes never auto-move or replace the other slot. A currently hidden/taken-down post cannot be newly pinned, but an existing pin remains private and may reappear after reversal. Another author's post returns `403 forbidden`; a missing/deleted/hidden target returns `404 post_not_found`; a structurally non-pinnable post returns `422 pin_not_allowed`. Concurrent replacements are last-commit-wins, same-target pin is an idempotent success, and stale target-specific unpin cannot clear a newer replacement.

## 4. Candidate Approaches

### Option A: Dedicated AppView profile-pin state

Summary: Add an owner-scoped AppView table with one row per owner and slot (`standard` or `project`), referencing the selected indexed post URI. Add authenticated pin/unpin mutations and integrate pin selection into the existing profile Posts and Projects list responses.

Pros:

- Matches the private AppView boundary.
- Enforces one pin per slot directly and supports atomic replacement.
- Keeps public indexed post fields and chronological sort keys unchanged.
- Can cascade on permanent owner or target deletion.
- Requires no lexicon or PDS write.

Cons:

- Adds a migration, mutation surface, and pin-aware pagination logic.
- Requires the server and Flutter caches to coordinate replacement of the old marker.

Risks:

- Incorrect first-page/cursor handling could duplicate or omit items.
- Incorrect ownership or current-membership checks could allow invalid mutations.

### Option B: Store pin flags or sort overrides on `craftsky_posts`

Summary: Add pin ownership/ordering fields directly to the indexed public-post table or change `profile_sort_at` when a post is pinned.

Pros:

- Profile list queries could order on fields already joined to posts.

Cons:

- Mixes private curation state with the indexed public snapshot.
- A post row cannot naturally represent owner-and-slot lifecycle as clearly as a dedicated resource.
- Changing `profile_sort_at` would conflate import chronology with profile curation and could affect cursor semantics.

Risks:

- Pin state could leak into unrelated post surfaces or corrupt established chronological behavior.

### Option C: Public PDS pin record

Summary: Add a lexicon record that points to the user's selected posts.

Pros:

- Pin choices would be portable with the user's repository.

Cons:

- Makes the AppView-owned curation state publicly readable and usable by any AppView.
- Requires a load-bearing lexicon decision, ADR, code generation, PDS writes, and firehose convergence.
- Directly contradicts the requested AppView-only boundary.

Risks:

- Irreversible public disclosure and loss of the requested AppView-only boundary.

## 5. Recommended Direction

Recommended approach: Option A — dedicated AppView profile-pin state.

Why: It directly models two independent owner-scoped slots, permits transactional replacement, follows the existing private saved-post lifecycle precedent, and lets the two existing profile list endpoints add pin-first presentation without changing public post records, lexicons, or chronological ordering elsewhere.

## 6. Problem / Opportunity

Profile visitors currently see only chronological lists, so a creator cannot keep one representative standard post and one representative project visible above newer content. Private AppView pin state creates a profile-curation feature while allowing every profile visitor to understand why a card is out of chronological order.

## 7. Goals

- G-001: Let every authenticated current member select one top-level standard post and one project post as profile pins.
- G-002: Place each visible pin first in its corresponding profile list for every viewer allowed to see it.
- G-003: Make replacement and removal predictable, atomic, and immediately reflected in active profile UI.
- G-004: Keep pin state, authorization, and lifecycle entirely in the AppView.
- G-005: Preserve existing content policy, pagination, and chronological behavior outside the pinned first position.

## 8. Non-Goals

- NG-001: Do not write pin records or fields to a PDS, add a lexicon, emit a Tap event, or make pin choices portable across independent AppViews.
- NG-002: Do not add timeline boosting, search ranking, discovery ranking, notifications, recommendation signals, or interaction-count changes.
- NG-003: Do not support more than one pin per slot, arbitrary pin ordering, cross-slot ordering, folders, collections, or scheduled pin changes.
- NG-004: Do not allow users to pin another person's post, a comment, a nested reply, or a post that does not match the selected slot.
- NG-005: Do not add payment processing, account-plan concepts, user-tier distinctions, feature-access gates, or integration seams for any of them. For this slice, pinning is available to every authenticated current member.
- NG-006: Do not show the `Pinned post` annotation on timelines, post threads, search results, notifications, saved-post lists, or general project discovery.
- NG-007: Do not change the chronological order of the non-pinned remainder of either profile list.
- NG-008: Do not add a separate pin-management page in this slice.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Post owner | Authenticated current member viewing a post they authored. | Pin, replace, and unpin their own qualifying posts from familiar context menus. |
| Profile visitor | Any authenticated viewer opening another member's profile or their own. | See visible pins first with a clear `Pinned post` explanation. |
| AppView | Trusted persistence and authorization boundary. | Enforce current membership, ownership, slot rules, visibility policy, and atomicity without PDS writes. |
| Flutter client | UI and cache consumer. | Stable pin mutations, pin-aware profile responses, accessible menu state, and deterministic refresh behavior. |

## 10. Current Behavior

The AppView serves separate chronological lists for a profile's top-level standard posts and project posts. Flutter renders both through the shared `PostCard`, whose context menu can expose owner-specific actions. There is no pin state, and all visible profile items retain their indexed chronological order.

## 11. Desired Behavior

An authenticated current member opens the context menu for their qualifying top-level post on a profile list, timeline, or top-level thread card and chooses `Pin post`. The AppView derives ownership from the authenticated DID, confirms current membership, validates that the target exists, is currently returnable, and matches its inferred `standard` or `project` slot, and transactionally makes it the sole pin for that slot. If another post occupied the slot, it is replaced immediately in the same commit without confirmation. Opening the menu for the current pin offers `Unpin post`; target-conditional removal is idempotent and cannot clear a newer replacement. No payment status, account plan, or separate feature-access decision is consulted.

Flutter does not reorder optimistically. While a mutation is pending, it disables every Pin/Unpin action for that active account and inferred slot, but leaves the independent slot usable. It then reconciles the current account's two slots from the authoritative response and refreshes the affected profile list. It shows `Post pinned` or `Post unpinned` after success, and the agreed retry message after failure while preserving confirmed state.

The first page of the owner's corresponding profile list returns the visible current pin as the first item and identifies it with page-level `pinnedPostUri`. When no visible pin is promoted, and on every later page, the response omits `pinnedPostUri` rather than returning it as null. The attribution reuses the existing “Reposted by…” slot's typography, spacing, and subdued colour, with a pin icon and exact non-interactive text `Pinned post`. The pin consumes one normal page position; the remainder stays newest-first. It never appears again on later pages and does not cause duplicates, omissions, or oversized pages. A later-page cursor is invalid if its starting pin state changed. Other surfaces render the same post normally without the annotation or ranking change, while the private two-slot read lets owner menus show the correct Pin/Unpin action.

Pin state remains in AppView Postgres. It survives session/device changes but is removed when the selected indexed post or owner membership is permanently deleted, or when a post update makes the target structurally invalid for its slot. Structural changes never auto-move a pin into the other slot. Temporary policy or language filtering hides but does not erase an existing pin. No pin operation writes to the PDS, changes public interaction data, or waits for firehose convergence.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | Craftsky shall let every authenticated current member curate one standard post and one project post at the top of the corresponding lists on their own profile. | Gives creators durable profile highlights across both core post types. | Prompt; User clarification | AC-001, AC-002, AC-004, AC-008 |
| BR-002 | Business | Must | Profile visitors shall be able to distinguish a promoted profile pin from chronological content through a visible, understandable annotation. | Explains why the first card may be older than the cards below it. | Prompt | AC-010, AC-018 |
| BR-003 | Business | Must | Pinning shall be an AppView-controlled capability available equally to every authenticated current member, with no payment, account-plan, user-tier, or separate feature-access behavior in this slice. | Preserves the clarified scope and prevents unrelated access-policy work. | User clarification | AC-007, AC-011, AC-015 |
| FR-001 | Functional | Must | From a qualifying owner-authored top-level post's existing owner-action context menu on profile, timeline, and top-level thread surfaces, Flutter shall offer `Pin post`; for the current pin it shall offer `Unpin post`. Search and general project-discovery cards shall not gain owner actions in this slice. | Uses the requested interaction surface and confirmed scope. | Prompt; User grilling decision | AC-001, AC-002, AC-005, AC-006 |
| FR-002 | Functional | Must | A successful pin mutation shall atomically make the target the sole pin for its inferred slot, replacing any existing pin in that slot without affecting the other slot. | Prevents transient or persistent multiple-pin states. | Prompt | AC-003, AC-004, AC-014 |
| FR-003 | Functional | Must | Unpinning shall target the selected post URI, remove it only when it is the current pin for its slot, and otherwise succeed as an idempotent no-op returning the authoritative two-slot state, so stale UI cannot clear a newer replacement. | Makes retries safe and avoids stale-action data loss. | Recommended direction; User grilling decision | AC-005, AC-014, AC-017 |
| FR-004 | Functional | Must | The AppView shall create or replace a pin only for the authenticated DID's own currently indexed, currently returnable top-level post whose type matches the inferred slot. Another author's post shall return `403 forbidden`; a missing, deleted, or currently hidden/taken-down target shall return `404 post_not_found`; and a reply/comment or otherwise structurally non-pinnable target shall return `422 pin_not_allowed`, without changing either slot. These new-pin validity checks shall not prevent a target-specific unpin of a retained pin. | Enforces ownership, current visibility, and the two-slot product boundary while preserving explicit removal. | Prompt; Codebase; User grilling decision | AC-006, AC-016 |
| FR-005 | Functional | Must | Pin state shall be persisted only in owner-scoped AppView Postgres state and shall require no PDS call, lexicon change, Tap event, or firehose convergence. | Pinning must remain private AppView value and server-immediate state. | Prompt; Architecture | AC-007, AC-013 |
| FR-006 | Functional | Must | The first page of each profile Posts or Projects list shall place its visible current pin first, identify it with page-level `pinnedPostUri`, count it toward the requested page limit, and return every other item in the existing chronological order. The response shall omit `pinnedPostUri` when no visible pin is promoted and on every later page. Flutter shall tolerate absent or explicit null input but shall encode the absent state by omitting the key. The shared canonical post model shall not gain an `isPinned` field. | Implements an exact profile-only page contract without leaking state into shared post responses. | Prompt; Codebase; User grilling decision; DR-002 resolution | AC-008, AC-010, AC-012, AC-017 |
| FR-007 | Functional | Must | Pin-aware pagination shall honor the requested limit, include the pin at most once, return every visible non-pinned post exactly once across traversal, and bind cursors to the pin state used to start traversal. If that pin state changes, a later-page request shall return `400 invalid_cursor` and Flutter shall restart from page one. | Avoids duplicates, gaps, page-size drift, and ambiguous traversal after pin replacement. | API architecture; User grilling decision | AC-009 |
| FR-008 | Functional | Must | Flutter shall key pending mutations by active account and inferred slot. While one slot is pending, every Pin/Unpin action for that account and slot shall be disabled to prevent competing replacements, while the independent slot remains usable. Flutter shall not reorder or change confirmed menu state optimistically. After the authoritative response, it shall reconcile the account-scoped two-slot state and refresh the affected profile list. It shall show `Post pinned` or `Post unpinned` on success and `Couldn’t pin post. Try again.` or `Couldn’t unpin post. Try again.` on failure while retaining confirmed state. | Prevents duplicate/racing same-slot actions without unnecessarily serializing the independent slot, avoids disruptive rollback, and gives exact understandable feedback. | User grilling decision; Codebase; DR-003 resolution | AC-003, AC-005, AC-013, AC-018 |
| FR-009 | Functional | Must | The profile-only pin attribution shall reuse the existing “Reposted by…” attribution slot's typography, spacing, and subdued colour, replacing its content with a small pin icon and exact localized English copy `Pinned post`. The attribution shall be informational and non-interactive. | Preserves the requested copy and confirmed visual consistency. | Prompt; User grilling decision | AC-010, AC-011, AC-018 |
| FR-010 | Functional | Must | Every authenticated current member shall be able to create, replace, and remove their own valid pins. Flutter and AppView shall not consult or model payment status, account plans, user tiers, or a separate feature-access decision for any pin behavior. | Keeps the slice focused on universal pinning and prevents speculative access infrastructure. | User clarification | AC-015, AC-016 |
| FR-011 | Functional | Must | A stored pin shall be promoted and labelled only when the target may be returned to that requesting viewer under existing membership, block, moderation, and content-language policy; temporary filtering shall not delete the pin. | Pinning must not bypass established safety or preference policy. | Codebase; Architecture | AC-012 |
| FR-012 | Functional | Must | Permanent deletion of a pinned indexed post, permanent removal of its owner's Craftsky membership, or an indexed update that makes the target structurally invalid for its slot shall delete the affected pin state. A structural type change shall not auto-move or replace the other slot. Session expiry, sign-out, device removal, reinstall, account switching, and temporary policy/language filtering shall not delete pins. | Gives permanent and temporary lifecycle changes deterministic semantics. | Private-state precedent; User grilling decision | AC-013, AC-016 |
| FR-013 | Functional | Must | The AppView shall expose `GET /v1/profiles/me/pins`, bodyless `PUT /v1/posts/{did}/{rkey}/pin`, and bodyless `DELETE /v1/posts/{did}/{rkey}/pin` using existing `/v1/` middleware, camelCase JSON, no-body route policies, and standard error envelopes. The authenticated owner and target path fully identify each mutation. As a deliberate feature-specific exception to the API architecture's generic PUT/DELETE examples, each successful read or mutation shall return `200 OK` with the authoritative nullable `standardPostUri` and `projectPostUri`; mutations shall neither require a body nor return an empty `204` response. Timestamps shall remain internal. Profile page one shall include `pinnedPostUri` only for a visible promoted pin and otherwise omit it; later pages shall always omit it. | Gives all owner-action surfaces one bounded account-scoped state source, avoids a follow-up-read race, and makes the intentional wire-contract exception explicit. | API architecture; User grilling decision; DR-001 and DR-002 resolutions | AC-017 |
| NFR-001 | Non-functional | Must | Concurrent pin, replacement, unpin, deletion, and retry operations shall preserve at most one valid row per owner/slot and shall never change another owner or the other slot. Concurrent replacements shall use last-committed-wins semantics, and a stale target-specific unpin shall not clear a newer selection. | Private state must remain atomic and account-isolated. | Recommended direction; User grilling decision | AC-004, AC-014, AC-016 |
| NFR-002 | Non-functional | Must | Pin-aware profile queries shall use bounded, indexed, set-based access and shall not add per-item pin lookups. | Profile lists are paginated hot paths. | Codebase | AC-009, AC-019 |
| NFR-003 | Non-functional | Must | Pin selection shall not be exposed through PDS records, firehose events, another owner's mutation response, log fields, trace attributes, or metric labels. The resulting profile order and `Pinned post` annotation are intentionally visible to allowed profile viewers. | Distinguishes private control state from its public-in-app presentation. | Prompt; Privacy boundary | AC-007, AC-016, AC-019 |
| NFR-004 | Non-functional | Must | Pin and unpin controls and the pinned annotation shall be accessible by screen reader, keyboard/focus navigation, and sufficient icon/text contrast; the icon shall not be the sole indication. | The state and action must not depend on sight or color alone. | UI quality standard | AC-018 |
| NFR-005 | Non-functional | Should | Pin operations shall emit bounded success/failure telemetry by operation, slot, result, and error class, without recording owner DID or post URI; no new alert is required for this pre-production feature. | Provides diagnostics without leaking identifiers or adding premature alert noise. | Existing observability | AC-019 |
| NFR-006 | Non-functional | Must | The migration shall be reversible and the feature shall have store, handler, route-contract, pagination, membership, lifecycle, Flutter data/provider, widget, accessibility, and regression coverage. | Cross-layer private state and pagination require durable regression protection. | Workflow quality standard | AC-020 |
| RULE-001 | Business rule | Must | Each owner may have zero or one `standard` pin and zero or one `project` pin; the slots are independent. | This is the requested capacity. | Prompt | AC-003, AC-004, AC-005 |
| RULE-002 | Business rule | Must | The `standard` slot accepts top-level non-project posts, including top-level quote posts; the `project` slot accepts top-level project posts. Neither accepts a comment or nested reply. | Aligns the user wording with current profile-list membership. | Prompt; Codebase | AC-001, AC-002, AC-006 |
| RULE-003 | Business rule | Must | Pinning changes only profile-list prominence and annotation; it creates no notification, public interaction count, timeline/search/discovery boost, or author-visible event beyond the owner's own profile presentation. | Keeps the behavior limited to the requested profile lists. | Scope | AC-007, AC-011 |
| RULE-004 | Business rule | Must | Every viewer who is allowed to see the pinned post on the profile sees the same `Pinned post` annotation. | Pin presentation depends only on whether the viewer may see the post. | Prompt; User clarification | AC-010, AC-015 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-001, RULE-002 | Given an authenticated current member and their unpinned top-level standard post, when they choose `Pin post` from an existing owner-action menu, then the standard slot points to that exact URI and the project slot is unchanged. |
| AC-002 | BR-001, FR-001, RULE-002 | Given an authenticated current member and their unpinned top-level project post, when they choose `Pin post`, then the project slot points to that exact URI and the standard slot is unchanged. |
| AC-003 | FR-002, FR-008, RULE-001 | Given standard post A is pinned, when the owner pins standard post B, then no confirmation dialog appears, every standard-slot Pin/Unpin action for the active account is disabled during the request while project-slot actions remain usable, one commit replaces A with B, and only after success Flutter reconciles B first, updates both menu states, and shows `Post pinned`. |
| AC-004 | BR-001, FR-002, NFR-001, RULE-001 | Given one standard and one project pin, when either slot is replaced concurrently or retried, then exactly one final pin exists in that slot, the other slot is unchanged, and no intermediate two-pin state is committed. |
| AC-005 | FR-001, FR-003, FR-008, RULE-001 | Given a currently pinned post, when its owner chooses `Unpin post`, then actions for that active account and slot are disabled while pending, the independent slot remains usable, the slot becomes empty after authoritative confirmation, Flutter shows `Post unpinned`, and no replacement is required. Given stale UI after another device replaced the slot, the same target-specific request is a harmless no-op that returns the newer state without removing it. |
| AC-006 | FR-001, FR-004, RULE-002 | Given an invalid target, when a new pin is attempted directly or through UI, then another author's post returns `403 forbidden`, a missing/deleted/hidden post returns `404 post_not_found`, and a comment/reply or other structurally non-pinnable post returns `422 pin_not_allowed`; neither slot changes and non-qualifying cards do not offer `Pin post`. These restrictions do not prevent target-specific removal of an existing retained pin. |
| AC-007 | BR-003, FR-005, NFR-003, RULE-003 | Given any successful pin, replacement, or unpin, when storage and external activity are inspected, then only private AppView state changed: no PDS request, lexicon record, Tap event, notification, interaction count, or non-profile ranking signal was produced. |
| AC-008 | BR-001, FR-006 | Given a visible pinned post older than several unpinned posts and a page limit of 10, when any allowed viewer opens the corresponding profile tab without a cursor, then page-level `pinnedPostUri` identifies item one, the page contains at most nine chronological items after it, and those items retain newest-first order. |
| AC-009 | FR-007, NFR-002 | Given a pin older than the original first page and enough items for multiple pages, when every cursor page is traversed at a fixed limit without pin changes, then each response has at most that limit, the pin appears only as the first item of page one, every later page omits `pinnedPostUri`, every other visible post appears exactly once, and continuation is deterministic. If the slot's pin changes after traversal starts, the old cursor returns `400 invalid_cursor` and Flutter restarts from page one. |
| AC-010 | BR-002, FR-006, FR-009, RULE-004 | Given any allowed viewer opens a profile list with a visible pin, then only the card identified by page-level `pinnedPostUri` uses the existing repost-attribution slot's typography, spacing, and subdued colour to show a small pin icon and exact text `Pinned post`; the row has no independent tap action. |
| AC-011 | BR-003, FR-009, RULE-003 | Given the same pinned post appears in a timeline, thread, search result, notification, saved-post list, or general project list, then it has normal ordering and no pinned annotation. |
| AC-012 | FR-006, FR-011 | Given a stored pin is excluded for a viewer by language preference, block, moderation, or membership policy, when that viewer loads the profile list, then the post is neither promoted nor labelled, page one omits `pinnedPostUri`, and the list fills from visible chronological items; if a temporary exclusion later clears, the retained pin can reappear first. |
| AC-013 | FR-005, FR-008, FR-012 | Given a pin exists, when its indexed target or owner membership is permanently deleted or the target is updated into another type/non-top-level shape, then the pin is removed without a placeholder or automatic move. When only a session/device/account selection or temporary policy/language state changes, it persists. Successful mutations reconcile without waiting for Tap. |
| AC-014 | FR-002, FR-003, NFR-001 | Given racing replacements from two devices, an unpin for a stale target, same-target retries, and two occupied independent slots, when operations settle, then the last committed replacement wins, same-target pinning succeeds idempotently, the stale unpin cannot clear the winner, at most one row exists per owner/slot, and the other slot is unchanged. |
| AC-015 | BR-003, FR-010, RULE-004 | Given any authenticated current member, when they use valid pin, replacement, and unpin actions, then each action is available and succeeds without reading or requiring payment status, an account plan, a user tier, or a separate feature-access decision; every allowed profile viewer sees the resulting visible pin. |
| AC-016 | FR-004, FR-010, FR-012, NFR-001, NFR-003 | Given two current-member accounts on one device, another author's target, and owner/target deletion cases, when pin operations run, then authenticated owner scoping, current membership, target ownership/type, and lifecycle are enforced without exposing or mutating another account's private selection and without consulting any additional user category. |
| AC-017 | FR-003, FR-006, FR-013 | Given the pin read, target-specific mutations, and profile-list responses, when contract tests inspect requests, success bodies, failures, and middleware, then all routes are authenticated/device-bound, camelCase, idempotent, rate classified, and use `{error, message, requestId}`. PUT and DELETE reject request bodies under no-body policies. The read and both bodyless mutations return `200 OK` with authoritative nullable `standardPostUri` and `projectPostUri`, never an empty `204` or public timestamps. Profile page one includes `pinnedPostUri` exactly when a visible pin is promoted and otherwise omits it; later pages always omit it; Flutter tolerates absent/null decoding and omits the key when encoding no pin. No canonical post response contains `isPinned`. |
| AC-018 | BR-002, FR-008, FR-009, NFR-004 | Given touch, keyboard, and screen-reader use in supported text scales, when pin controls and the pinned card receive focus, then actions have clear labels, affected-slot pending actions are disabled while the independent slot remains usable, failures preserve confirmed state and announce the agreed retry copy, the non-interactive attribution is announced as `Pinned post`, the icon is not the sole signal, and the layout remains readable without clipping. |
| AC-019 | NFR-002, NFR-003, NFR-005 | Given full profile pages and successful/failed pin operations, when database access and telemetry are inspected, then pin lookups are bounded and indexed, no per-card query occurs, and only bounded operation/slot/result/error-class telemetry is recorded without DIDs or post URIs. |
| AC-020 | NFR-006 | Given the feature verification suite runs, then it covers migration up/down/up; slot/type/ownership/current-membership and current-visibility validation; universal member access; same-target idempotency, last-commit-wins replacement, and stale-unpin races; deletion/type-change/session lifecycle; pin-bound cursor invalidation and full traversal; policy/language filtering; private-state read and page-level metadata contracts; server-confirmed Flutter reconciliation and exact feedback copy; attribution styling/accessibility; and non-profile regressions. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | Pin the post already occupying its slot. | Succeed idempotently, keep one row and the same selected URI, return the authoritative two-slot state, and show the normal `Post pinned` feedback. | FR-002, FR-008, FR-013, RULE-001 |
| EC-002 | Replace only one of two occupied slots. | Replace that slot atomically and preserve the other. | FR-002, RULE-001 |
| EC-003 | Unpin from stale UI after another device replaced the slot. | Treat the target-specific removal as an idempotent no-op, do not clear the newer target, and return its authoritative state for reconciliation. | FR-003, FR-013, NFR-001 |
| EC-004 | Pin a comment, nested reply, other owner's post, missing/deleted/hidden target, or mismatched type. | Reject without changing either slot or disclosing private state: `403 forbidden` for another author's post, `404 post_not_found` for a missing/deleted/hidden target, and `422 pin_not_allowed` for a structurally non-pinnable target. | FR-004, RULE-002 |
| EC-005 | Pinned target is older than the first chronological page. | Move it into the first position of page one and exclude it from its original later position without a gap. | FR-006, FR-007 |
| EC-006 | Pin changes between page-one and later-page requests. | Bind the cursor to the slot's starting pin state, omit `pinnedPostUri` from later-page responses, return `400 invalid_cursor` after a change, and have Flutter restart from page one; never silently duplicate or omit an item. | FR-007 |
| EC-007 | Pinned target is hidden for one viewer but visible to another. | Fill the first viewer's page chronologically without the marker; promote and label for the allowed viewer. | FR-011 |
| EC-008 | Pinned target is temporarily language-filtered or moderated, then becomes visible again. | Retain the private pin and promote it again once visible. | FR-011 |
| EC-009 | Pinned target or owner membership is permanently deleted, or an indexed update makes the target structurally invalid for its slot. | Delete the affected pin through referential/indexer lifecycle handling without a placeholder, automatic move, or replacement of the other slot. | FR-012 |
| EC-010 | A current member with no other account classification uses pinning. | Allow pin, replace, and unpin based solely on current membership, ownership, and target validity; do not request or infer any additional account state. | FR-010 |
| EC-011 | Active account changes while a pin request is in flight. | Fence the completion to the initiating account and prevent it from updating the newly active account's caches or state. | NFR-001 |
| EC-012 | Text scale or narrow viewport makes the annotation tight. | Wrap or size the icon/text row without clipping card content or obscuring the context menu. | NFR-004 |
| EC-013 | Two devices concurrently pin different valid posts into the same slot. | The last committed replacement wins, exactly one row remains, and each response reports the authoritative state at its commit. | FR-002, FR-013, NFR-001 |
| EC-014 | Pin or unpin fails after Flutter starts the request. | Preserve the last confirmed order and menu state, re-enable every action for the affected active-account slot, leave the independent slot usable throughout, and show `Couldn’t pin post. Try again.` or `Couldn’t unpin post. Try again.` as applicable. | FR-008 |

## 15. Data / Persistence Impact

- New fields/tables:
  - Add a dedicated private AppView relation conceptually keyed by `(ownerDid, slot)` with a canonical `postUri` reference and server timestamps. `slot` is constrained to `standard` or `project`; timestamps are internal and are not returned by the pin APIs.
  - Keep public `craftsky_posts` records and `profile_sort_at` unchanged.
- Constraints:
  - Unique/primary key on owner plus slot.
  - Owner references Craftsky membership and selected post references `craftsky_posts`, both with deletion lifecycle that removes stale pins.
  - Pin creation/replacement validates current returnability, target author, top-level status, and post/project classification transactionally. Target-specific unpin remains possible for an existing retained pin.
  - Indexed target updates that invalidate top-level/type classification clear the affected pin; they do not move it between slots or alter the other slot. Read-time validation remains a defence against stale or racing index state.
- Migration required: Yes, reversible up/down migration; implementation must allocate the next available number after rechecking the tree.
- Backwards compatibility:
  - Existing accounts begin with both slots empty.
  - Existing clients ignore the additive profile-list marker and continue to receive valid chronological content, although they will not display the annotation.
  - No public PDS record or lexicon migration is required.

## 16. UI / API / CLI Impact

- UI:
  - Add `Pin post` / `Unpin post` to qualifying owner context menus on profile lists, timelines, and top-level thread cards. Do not add owner actions to search or general project-discovery cards solely for pinning.
  - Replace an occupied slot immediately without confirmation. Scope pending state by active account and slot: disable all actions for the affected slot, preserve independent-slot actions, and keep the last confirmed order rather than reordering optimistically.
  - After success, reconcile both slots and the affected profile list from authoritative AppView state, then show `Post pinned` or `Post unpinned`. On failure, preserve confirmed state and show `Couldn’t pin post. Try again.` or `Couldn’t unpin post. Try again.`.
  - In profile lists only, reuse the “Reposted by…” attribution position, typography, spacing, and subdued colour for a pin icon and exact text `Pinned post`; the annotation is informational and non-interactive.
- API:
  - Add `GET /v1/profiles/me/pins`, returning authoritative nullable `standardPostUri` and `projectPostUri` for the authenticated current member.
  - Add bodyless idempotent `PUT /v1/posts/{did}/{rkey}/pin` and bodyless target-conditional `DELETE /v1/posts/{did}/{rkey}/pin`. The AppView derives the owner from authenticated context and infers the slot from the indexed target; both routes use no-body policies.
  - The read and both mutations return `200 OK` with authoritative nullable `standardPostUri` and `projectPostUri`, including same-target success and stale-unpin no-op responses. This is an intentional exception to the API architecture's generic PUT-body and DELETE-204 examples: a path-complete mutation requires no body, and returning state avoids a follow-up-read race. No pin timestamp is exposed.
  - Profile page one includes page-level `pinnedPostUri` only when a visible pin is promoted. It omits the key when no pin is promoted and on every later page. Flutter tolerates absent/null input but omits the key when encoding no pin. The shared canonical post response does not gain `isPinned`. A visible pin consumes one requested page position.
  - Bind continuation cursors to the starting pin state. A changed pin makes an old cursor return `400 invalid_cursor`, after which Flutter restarts from page one.
  - Use `403 forbidden` for another author's target, `404 post_not_found` for a missing/deleted/currently hidden target, and `422 pin_not_allowed` for a structurally non-pinnable target, through the standard error envelope.
- CLI: None.
- Background jobs: None. Pin mutation and profile presentation are synchronous AppView decisions.

## 17. Security / Privacy / Permissions

- Authentication: All pin mutations and profile-list reads remain authenticated `/v1/*` requests using the current session and device middleware.
- Authorization:
  - Derive the owner from the authenticated DID, not a body field.
  - Require target author DID to equal authenticated DID for mutations.
  - Enforce current membership and target ownership. Enforce current target returnability and slot/type when creating or replacing a pin without blocking target-specific removal of a retained pin.
  - Allow all authenticated profile viewers to see the annotation only when they may see the underlying post.
- Sensitive data:
  - The stored selection is private AppView state.
  - The resulting order and annotation are intentionally visible in the profile UI.
  - Do not put owner/target identifiers into telemetry dimensions.
- Abuse cases:
  - Direct API calls must not bypass current membership, ownership, or slot limits.
  - Stale cross-device unpin must not clear a newer replacement.
  - Pinning must not bypass moderation, blocks, membership, or language preferences.
  - Pin state must not become an input to distribution ranking outside the profile.

## 18. Observability

- Events/logs: Reuse structured API request/error logging with bounded operation (`pin`, `replace`, `unpin`), slot, stage, result, and error class; exclude DIDs and post URIs.
- Metrics: Count successful, rejected, and failed operations by bounded operation/slot/result; observe latency with existing HTTP metrics.
- Alerts: None specific to this pre-production feature.
- Product analytics: No new analytics event is required by these requirements. If later requested, it must not contain post content or raw identifiers.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Speculative user-tier or access-policy infrastructure is added despite the clarified slice. | Scope expands and pinning behavior differs between users without a product requirement. | Make universal current-member access an acceptance criterion and reject payment, plan, tier, or separate access-gate dependencies in review. |
| RISK-002 | Flutter or AppView accidentally assumes a user category beyond current membership. | Some current members could lose access to pinning or receive inconsistent menu/API behavior. | Authorize only from existing authentication/current-membership, ownership, and target validity; test multiple ordinary member accounts. |
| RISK-003 | Naive first-page insertion or pin changes during traversal break cursor pagination. | Posts may duplicate, disappear, or page sizes may exceed their limit. | Treat the pin as one page item, exclude it from chronological traversal, bind cursors to starting pin state, return `invalid_cursor` after a change, and test full multi-page walks plus Flutter restart. |
| RISK-004 | Shared `PostCard` presentation leaks the annotation to non-profile surfaces or conflicts with repost attribution layout. | Timelines, search results, or threads could incorrectly show profile-only prominence, or card spacing could diverge. | Pass explicit page-level pin presentation into profile cards, reuse the existing attribution slot and styling, and add cross-surface/widget regression tests. |
| RISK-005 | Ownership or slot validation occurs only in Flutter. | Direct callers could pin another user's post, a reply, or more than the allowed capacity. | Enforce authenticated DID, target type, and unique owner/slot invariants in AppView transactions. |
| RISK-006 | A post is updated through another AT client so its type/top-level classification no longer matches its stored slot. | A stale pin could be promoted in the wrong profile tab. | Revalidate during indexing/read, clear structurally invalid state, and never auto-move it or alter the other slot. |
| RISK-007 | Active-account changes race with Flutter mutation completion. | One account's result could update another account's profile cache. | Use the established account-operation ownership guard and account-scoped cache invalidation. |
| RISK-008 | A future API cleanup applies generic PUT/DELETE conventions to the pin routes. | Requiring a PUT body or changing DELETE to `204` would break the agreed authoritative reconciliation contract. | Document the deliberate path-complete/no-body and `200` response exception in requirements and coding plan; protect it with handler, route-policy, and Flutter API contract tests. |
| RISK-009 | Server and Flutter disagree on absent `pinnedPostUri` representation. | Profile decoding or exact contract tests could drift between missing and null. | Require the server to omit the key when no visible first-page pin exists and on later pages; keep Flutter tolerant of absent/null input and omission on encode. |
| RISK-010 | Flutter uses one global pending flag for both slots. | A standard mutation unnecessarily blocks project actions or vice versa. | Key pending state by active account and slot, suppress all competing same-slot actions, and test that the other slot remains usable. |

## 20. Assumptions

None identified. Former assumptions ASM-001–ASM-006 were resolved during grilling on 2026-08-05 and are recorded as confirmed decisions in Section 3 and as requirements.

## 21. Open Questions

None.

## 22. Review Status

Status: Reviewed
Risk level: Medium
Review recommended: Yes — completed through decision grilling
Reviewer: User
Date: 2026-08-05
Notes: The user ended grilling after confirming all product, interaction, lifecycle, pagination, and API decisions in Section 3. Document-review findings DR-001–DR-003 were resolved on 2026-08-05 by making the feature-specific mutation contract explicit, standardizing omitted `pinnedPostUri`, and scoping pending state per active account and slot. There are no blocking questions. The requirements deliberately contain no payment, free/paid-user distinction, plan-tier behavior, or access-gating work; every authenticated current member may pin.

## 23. Handoff To Test Design

- Requirements file: `01-requirements.md`
- Next test specification: `02-acceptance-tests.md`
- Must-cover requirement IDs: BR-001–BR-003, FR-001–FR-013, NFR-001–NFR-004, NFR-006, RULE-001–RULE-004.
- Suggested test levels:
  - Store/migration: two-slot uniqueness, current-returnability/type validation, last-commit-wins replacement, target-specific unpin races, permanent deletion and structural-update cleanup, unaffected-slot preservation, and up/down/up.
  - Handler/route: exact private read and mutation response shapes, deliberate bodyless PUT/DELETE plus authoritative `200` exception, current membership, ownership, universal member access, same-target idempotency, exact error codes, middleware/policy, camelCase contract, and no timestamp exposure.
  - Integration: first-page promotion and exact `pinnedPostUri` omission within the requested limit, later-page omission, full cursor traversal, pin-bound cursor invalidation, Flutter-compatible restart semantics, policy/language visibility, and deletion/type-change lifecycle.
  - Flutter unit/provider: absent/null-tolerant page decoding with omission on encode, no optimistic reorder, active-account-and-slot pending state, independent-slot availability, exact success/error feedback, authoritative reconciliation, mutation fencing, cache invalidation, and multi-account isolation.
  - Widget/accessibility: eligible menu surfaces and labels/states, repost-attribution-slot styling, exact `Pinned post` copy, non-interactive semantics, icon semantics, text scaling, and profile-only annotation.
  - Regression: chronology and presentation in timeline, thread, search, saved posts, notifications, and general projects.
- Blocking open questions: None.
