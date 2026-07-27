# Requirements: Flutter Saved Posts

## 1. Initial Request

Implement the Flutter experience for the private saved-post and folder capabilities delivered by `docs/changes/2026-07-20-appview-saved-posts/`. Add a bookmark action to posts; prompt an unsaved user to choose a folder, defaulting to no folder and allowing folder creation; immediately unsave an already-saved post without confirmation; let users view and manage saved posts and folders from Settings; offer both non-destructive folder deletion and deletion of the saves in that folder; and introduce a reusable compact post-summary widget for quote previews, notification subjects, and saved-post rows.

## 2. Current Codebase Findings

- Relevant files:
  - Implemented AppView contract: `docs/changes/2026-07-20-appview-saved-posts/01-requirements.md`, `appview/internal/api/saved_post.go`, `appview/internal/api/saved_post_request.go`, `appview/internal/routes/routes.go`.
  - Post wire model and full post UI: `app/lib/feed/models/post.dart`, `app/lib/feed/widgets/post_card.dart`.
  - Post data/mutation patterns: `app/lib/feed/data/post_api_client.dart`, `app/lib/feed/data/post_repository.dart`, `app/lib/feed/providers/toggle_like_post_provider.dart`, `app/lib/feed/providers/timeline_provider.dart`.
  - Existing compact post presentations: private `_QuotePreviewCard` in `post_card.dart` and inline subject text in `app/lib/notifications/widgets/notification_row.dart`.
  - Navigation/settings: `app/lib/router/router.dart`, `app/lib/router/route_locations.dart`, `app/lib/settings/pages/settings_page.dart`.
  - Existing placeholders: `app/lib/profile/pages/saved_page.dart` and `ProfileTab.saved` in `app/lib/profile/widgets/profile_tab_bar.dart` / `app/lib/profile/pages/profile_page.dart`.
- Existing AppView behavior:
  - Canonical authenticated post responses now contain `viewerHasSaved` and nullable `viewerSavedFolderId`.
  - `POST /v1/posts/{did}/{rkey}/saves` saves or moves one post; `DELETE` on the same path unsaves it idempotently.
  - `GET /v1/saved-posts` lists all, one folder, or unfiled saves with newest/oldest opaque-cursor pagination.
  - Folder list/create/rename/delete endpoints exist. Current folder deletion is non-destructive and moves its saves to unfiled.
  - Folder IDs are opaque; duplicate names are valid; folder responses contain no save count.
  - Saved-list items return `{post, savedAt, folderId}` and exact saved replies retain root/parent references.
- Existing Flutter behavior:
  - `Post` does not yet decode the two saved viewer-state fields, and `PostCard` has no bookmark action.
  - `SavedPage` is only a title/body placeholder at `/profile/saved`; Settings has no saved-post entry.
  - A Saved tab placeholder is displayed in every profile tab set, including visited profiles, although saved data is private.
  - Quote previews and notification subject snippets duplicate compact post presentation concerns and are not reusable.
  - Like/repost mutations use Riverpod, optimistic state, active-account operation guards, and explicit updates to live caches. There is no normalized post cache shared by every surface.
- Constraints discovered:
  - Saves and folder names are private AppView data. No Flutter flow may write them to a PDS or expose them on another user's profile.
  - A client-side loop cannot reliably remove every save in a folder: policy-hidden items may be absent from hydrated list pages, pagination can change, and a sequence of deletes can partially fail or be rate-limited.
  - The API architecture requires authenticated `/v1/` JSON routes, camelCase keys, opaque cursors, no request body on `DELETE`, and the standard error envelope.
  - App account switching must not let a late operation from one account update another account's UI.
- Test/build commands discovered:
  - Flutter analysis: `just app-analyze`.
  - Flutter tests: `just app-test` or targeted `flutter test` from `app/`.
  - AppView formatting and tests for the small API extension: `just fmt` and `just test`.

## 3. Clarifying Questions And Decisions

### Q1: Does “delete all posts inside a folder” delete public posts?

Answer: No. It removes the owner's private saved-post records in that folder, then deletes the folder. It must never delete the source posts or contact a PDS.

Decision / implication: User-facing copy should say “remove saved posts” or “unsave posts,” not “delete posts.”

### Q2: Should Flutter recursively call the per-post delete endpoint?

Answer: No.

Decision / implication: Extend folder deletion additively with `DELETE /v1/saved-post-folders/{folderId}?deleteSaves=true`. The AppView performs the folder/save deletion atomically for all rows, including saves currently omitted by content-policy hydration. Omitting the parameter preserves the existing unfile-on-delete behavior.

### Q3: Where should the private saved-post surface live?

Answer: Under Settings as a typed, full-screen route rather than as a tab on a public-profile surface.

Decision / implication: Use a canonical `/profile/settings/saved` route, add a Settings tile, replace the existing `/profile/saved` placeholder, and remove the `ProfileTab.saved` placeholder so visited profiles do not imply that private saves are profile content.

### Q4: What happens when the bookmark is already selected?

Answer: Tapping it immediately unsaves the post with no confirmation.

Decision / implication: The folder chooser is shown only when the post is not saved. Duplicate taps are prevented while a mutation is pending, and failures restore the prior state with localized feedback.

### Q5: How does folder creation inside the save dialog behave?

Answer: Creating a folder is an immediate folder operation. On success the dialog retains the new folder, selects it, and lets the user confirm the post save. Canceling the post save does not roll back a successfully created folder.

Decision / implication: Folder creation and post saving have independent error/retry states and must not be represented as one server transaction.

### Q6: What are the safe defaults?

Answer: New saves default to No folder. Folder deletion defaults to preserving its saves as unfiled; removing the contained saves requires an explicit destructive choice.

Decision / implication: Cancel or the non-destructive choice must be the safest keyboard/focus path in destructive UI.

### Q7: What does the shared post summary cover in this slice?

Answer: The visible compact content pattern shared by quote previews, notification subject previews, and saved-post list items.

Decision / implication: Build one presentational widget/data adapter that supports those three consumers while preserving each surface's navigation, metadata, moderation, and surrounding actions. Do not turn the full interactive `PostCard` into a large matrix of modes.

### Q8: Does the Settings route replace the profile Saved tab?

Answer: Yes.

Decision / implication: Settings is the sole saved-post entry point. Remove both `ProfileTab.saved` and the `/profile/saved` placeholder rather than maintaining duplicate private collection surfaces.

### Q9: What does the Saved overview contain?

Answer: All folders first, followed by all unfiled saved posts. Posts assigned to a folder do not appear on the overview and become visible only after that folder is opened.

Decision / implication: “All” means the collection overview, not a flattened all-saves feed. Folder rows remain alphabetically ordered and show no counts. Newest/Oldest affects the visible unfiled post list, not folder order. The overview always opens in this root state rather than restoring a prior folder.

### Q10: How does a folder open and expose management actions?

Answer: Tapping a folder opens a separate folder screen. Its name is the title, and Rename/Delete live in the app-bar overflow menu.

Decision / implication: Back navigation restores the overview and its scroll position. Navigation/diagnostics use a generic folder-screen route name and must not record the private folder ID or name. Folder contents support Newest/Oldest ordering.

### Q11: Where can folders be created and searched?

Answer: The overview has an app-bar Add folder action. In the save/move dialog, New folder expands an inline name field; no folder search is included in this slice.

Decision / implication: Successful inline creation persists independently, collapses the form, and selects the new folder. Folder selection uses incremental alphabetical pagination; client-only partial search is prohibited.

### Q12: How are bookmark actions positioned and scoped?

Answer: The bookmark sits on the right of a full post card immediately before the overflow menu. It appears on eligible top-level posts, projects, quote posts, comments, and nested replies, but not inside compact quote/notification summaries or protected placeholders.

Decision / implication: Private saving remains visually separated from public engagement actions. `PostSummary` contains no bookmark or engagement controls; users open compact previews to reach the full post action.

### Q13: What feedback does saving use?

Answer: The save dialog requires a separate Save button and remains open with that button busy until AppView confirms the mutation. On success it closes without a success snackbar; on failure it stays open with an inline recoverable error.

Decision / implication: Saving is confirmation-driven rather than optimistic. A folder-list failure does not block saving to No folder; the dialog shows an inline Retry for the folder list.

### Q14: What feedback does moving use?

Answer: Moving uses the same confirmation pattern: the chooser opens on the current assignment, remains open while the move runs, closes on confirmation, and retains the selection with an inline error on failure.

Decision / implication: Folder assignment changes do not optimistically remove an item from the current list.

### Q15: What feedback does unsaving use?

Answer: Unsaving remains confirmation-free and optimistic. The filled bookmark becomes outlined immediately, duplicate taps are blocked, and failure restores the exact prior saved/folder state with localized error feedback.

Decision / implication: No Undo action is offered because re-saving would create a new `savedAt` and would not truly restore chronology.

### Q16: How does folder deletion present its choices?

Answer: Always show Cancel, Delete folder and keep saved posts, and Delete folder and remove saved posts, even when the visible folder appears empty. Cancel receives default keyboard focus; remove-saves receives the strongest destructive styling.

Decision / implication: Flutter cannot infer true emptiness because there are no folder counts and policy-hidden saves may exist.

### Q17: Which empty sections are visible on the overview?

Answer: If folders exist but no unfiled posts exist, hide the empty Unfiled section. Show the full Nothing saved yet state only when there are neither folders nor unfiled saves.

Decision / implication: The overview stays compact without obscuring a genuinely empty collection.

## 4. Candidate Approaches

### Option A: Saved-post feature area plus a dedicated reusable summary and atomic AppView delete mode

Summary: Add account-scoped Flutter models, API/repository/provider state, dialogs, and the Settings route. Add a small reusable `PostSummary` presentation with adapters for `Post` and `QuotePreviewPost`. Extend folder deletion with an optional server-side `deleteSaves` query parameter.

Pros:

- Keeps private saved state in one account-aware feature boundary.
- Preserves simple full-post and compact-summary responsibilities.
- Makes delete-with-contents complete, atomic, and retry-safe.
- Reuses existing AppView contracts and adds only one backward-compatible API option.
- Lets the same summary behavior and accessibility tests cover three surfaces.

Cons:

- Requires coordinated Flutter and small AppView changes.
- Requires deliberate synchronization between saved-state mutations and already-loaded post surfaces.

Risks:

- A stale account operation could update the wrong bookmark state without account-keyed state and operation guards.
- Over-generalizing `PostSummary` could make it harder to use than the current small presentations.

### Option B: Reuse `PostCard` modes and recursively unsave folder contents from Flutter

Summary: Add flags to `PostCard` for quote/notification/saved variants and have Flutter page through a folder, deleting each visible saved item before deleting the folder.

Pros:

- Avoids a new AppView parameter.
- Reuses an existing post widget by name.

Cons:

- `PostCard` carries engagement actions, relationship state, media galleries, menus, and layout that compact consumers do not need.
- A recursive client delete is slow, rate-limit-sensitive, non-atomic, and cannot see policy-hidden saves.
- Partial failures leave the folder in an unclear state.

Risks:

- Users may choose “remove all” but retain hidden saves or only some visible saves.
- Mode flags can create regressions across unrelated full-post surfaces.

### Option C: Keep saved posts as a profile tab

Summary: Replace the existing `ProfileTab.saved` placeholder and keep `/profile/saved` as the main collection surface.

Pros:

- Reuses an existing placeholder and path.

Cons:

- Saved posts are private account settings, not profile content.
- The current tab exists on visited profiles and creates a misleading privacy affordance.
- Folder management is a better fit with other private account controls in Settings.

Risks:

- Users may infer that their saves or folder organization are visible to profile visitors.

## 5. Recommended Direction

Recommended approach: Option A.

Why: It gives Flutter a coherent private saved-post feature, makes the destructive folder option truthful and atomic, and extracts only the genuinely shared compact presentation. It also removes the current public-profile ambiguity while preserving the implemented AppView privacy and one-folder rules.

## 6. Problem / Opportunity

The AppView can now store and organize private saves, but the Flutter app neither exposes the behavior nor decodes saved viewer state. Users need a fast bookmark action, lightweight folder choice, and a private management surface. The work is also an opportunity to stop duplicating compact post presentation as quote previews, notification snippets, and saved lists grow.

## 7. Goals

- G-001: Let the active account save and unsave eligible posts directly from full post cards.
- G-002: Let a user choose no folder, an existing folder, or a newly created folder when saving.
- G-003: Let the active account browse folders and unfiled saves, sort folder contents/unfiled saves, open exact posts, move saves, and unsave from Settings.
- G-004: Let the active account create, rename, and delete flat folders, optionally removing every save in a deleted folder.
- G-005: Keep saved viewer state consistent across loaded Flutter surfaces and isolated across accounts.
- G-006: Reuse one accessible compact post-summary presentation for quote, notification, and saved-post contexts.
- G-007: Preserve the AppView's privacy, content-policy, pagination, timestamp, and reply-navigation contracts.

## 8. Non-Goals

- NG-001: Do not create or change atproto lexicons, PDS records, firehose behavior, or public engagement counts.
- NG-002: Do not delete public posts when deleting folders or saved records.
- NG-003: Do not add nested folders, tags, smart folders, folder sharing, notes, pinning, manual ordering, search-within-saves, bulk selection, or drag-and-drop organization.
- NG-004: Do not add folder or saved-post counts; the current AppView contract does not provide policy-consistent counts.
- NG-005: Do not add offline mutation queues or durable local copies of saved-post/folder data.
- NG-006: Do not expose saved content on another user's profile or retain the current Saved profile tab as a second collection UI.
- NG-007: Do not redesign full `PostCard`, notifications, or quote policy beyond extracting and adopting the common compact summary.
- NG-008: Do not migrate composer reply/quote target previews or unrelated post snippets to `PostSummary` in this slice.
- NG-009: Do not add product analytics events, new dependencies, or a new alert solely for this feature.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Saving user | Signed-in member bookmarking content for private later use. | Fast save/unsave, folder choice, truthful feedback, and private management. |
| Multi-account user | User with more than one Craftsky account on one device. | Strict per-account state, no stale cross-account mutations, and correct refresh after switching. |
| Post author | Author of content another member saves. | No notification, count, or disclosure that a save occurred. |
| Flutter client | Consumer of the saved-post AppView API. | Typed models, stable errors, opaque pagination, and one consistent mutation seam. |
| AppView | Privacy and persistence boundary. | Atomic owner-scoped folder deletion semantics and no PDS side effects. |

## 10. Current Behavior

Flutter renders canonical posts without decoding their saved viewer state. Full post cards expose like, reply, repost/quote, and overflow actions but no bookmark. The Saved page and profile Saved tab are placeholders, Settings has no entry, and there are no saved-post data providers. Quote cards and notification rows each own a separate compact representation of post content. AppView folder deletion always keeps contained saves by moving them to unfiled.

## 11. Desired Behavior

Every eligible full `PostCard`, including comment/reply cards, shows a bookmark on the right immediately before overflow. An outlined bookmark opens a localized chooser whose initial selection is No folder. The chooser loads folders incrementally, permits one selection, exposes inline folder creation, and requires a separate Save button. It remains open and busy until save confirmation, closes silently on success, and retains an inline retryable error on failure. If folder loading fails, No folder remains usable. A filled bookmark immediately and optimistically unsaves without confirmation or Undo, restoring its exact prior state only if deletion fails.

Settings contains a Saved posts tile opening a typed, full-screen `/profile/settings/saved` overview. It always shows all alphabetically ordered folders first, followed by unfiled saved posts; foldered posts are omitted until their folder is opened. Folder rows have no counts. Newest/Oldest sorts only the visible post list. Tapping a folder opens a separate screen titled with the folder name, with Rename/Delete in app-bar overflow and paginated folder contents beneath. The overview app bar creates folders. Empty Unfiled is hidden when folders exist; the whole-page empty state appears only when both sections are empty. Saved items can open the exact post/focused reply, move through a confirmation-driven chooser, or unsave immediately.

Folder deletion always asks whether to keep its saves as unfiled or explicitly remove every save, with Cancel as the safest default. The latter uses one authenticated AppView deletion with `deleteSaves=true`; Flutter never loops through hydrated list items.

A reusable `PostSummary` renders compact shared content—author, bounded text, optional time/metadata, optional representative media/project title, and unavailable/protected state—while accepting callbacks/slots owned by its parent surface. Quote previews, notification subject previews, and saved-post items adopt it without losing their existing tap destinations, visibility rules, reveal behavior, notification action context, or saved metadata/actions.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | Craftsky shall let the active signed-in account privately save and unsave eligible posts from the Flutter post UI. | The implemented AppView capability needs a direct user affordance. | Prompt; AppView contract | AC-001, AC-002, AC-006 |
| BR-002 | Business | Must | Craftsky shall let the owner privately view and manage its saved posts and folders from Settings. | A bookmark collection needs a discoverable private management surface. | Prompt | AC-007, AC-008, AC-011 |
| BR-003 | Business | Must | Craftsky shall let folder deletion either preserve contained saves as unfiled or remove all contained saves without deleting source posts. | The user requested both cleanup behaviors, and data-loss intent must be explicit. | Prompt; Architecture | AC-013, AC-014 |
| BR-004 | Business | Must | Compact post summaries shall use one reusable Flutter presentation across quote previews, notification subjects, and saved-post items. | Repeated compact post UI should remain visually and behaviorally consistent. | Prompt; Codebase | AC-020, AC-021, AC-022 |
| BR-005 | Business | Must | Saved-post state and folder organization shall remain private and isolated to the authenticated account. | Saves are private-by-intent data and the app supports multiple accounts. | Architecture; Codebase | AC-025, AC-026 |
| FR-001 | Functional | Must | The Flutter `Post` wire model shall decode `viewerHasSaved` and nullable `viewerSavedFolderId` from canonical authenticated post responses, preserve them through copy/update operations, and supply safe false/null defaults when the existing protected-post placeholder legitimately omits viewer fields. | The server is the source of truth for initial viewer state, while protected placeholders intentionally contain less data. | AppView contract; Codebase | AC-001, AC-024 |
| FR-002 | Functional | Must | Every eligible full `PostCard`, including top-level ordinary/project/quote posts, comments, and nested replies, shall expose a bookmark on the right immediately before overflow: outlined when unsaved and filled/selected when saved, with localized semantics and no public count. Protected placeholders and compact `PostSummary` instances shall expose no bookmark. | Users need a consistent private affordance separated from public engagement without bloating compact previews. | Prompt; User grilling | AC-001, AC-023, AC-032 |
| FR-003 | Functional | Must | Tapping an unsaved bookmark shall open a folder chooser; tapping a saved bookmark shall immediately start an optimistic unsave without confirmation or Undo. | This is the confirmed interaction contract, and re-saving cannot restore the original `savedAt`. | Prompt; User grilling | AC-002, AC-006, AC-029 |
| FR-004 | Functional | Must | The save chooser shall default to No folder, list folders with opaque-cursor incremental loading, distinguish duplicate names by opaque ID, allow exactly one selection, and require a separate Save button. It shall include no partial client-side folder search. If folder loading fails, it shall show inline Retry while keeping No folder usable. | The full folder collection must be selectable without blocking the core unfiled save action or offering misleading partial search. | Prompt; AppView contract; User grilling | AC-003, AC-004, AC-017, AC-030 |
| FR-005 | Functional | Must | New folder in the save/move chooser shall expand an inline name field. A successful create shall persist independently, collapse the inline form, be inserted in server order, and become selected; creation failure shall remain editable/retryable without closing the chooser. | Folder creation is a distinct resource action inside the confirmed single-dialog flow. | Prompt; User grilling | AC-005, AC-017, AC-030 |
| FR-006 | Functional | Must | Save, move, and unsave operations shall call the exact AppView resource, use server-returned state, and prevent duplicate requests. Save/move dialogs shall remain open with their confirm button busy until success, close without a success snackbar after confirmation, and retain selection with inline error on failure. Unsave shall update optimistically and restore the exact prior state with localized error feedback on failure. | Each mutation needs feedback appropriate to whether it has a dialog and recoverable assignment choice. | AppView contract; Existing Flutter pattern; User grilling | AC-006, AC-018, AC-024, AC-029, AC-030 |
| FR-007 | Functional | Must | Flutter shall provide typed saved-post/folder models, page models, API-client methods, repository interfaces/implementations, and Riverpod providers for all consumed AppView operations. | UI code should not decode JSON or construct network calls directly. | Codebase convention | AC-027 |
| FR-008 | Functional | Must | Settings shall contain a localized Saved posts tile that opens a typed, full-screen `/profile/settings/saved` route and pops back to Settings. The previous `/profile/saved` placeholder and `ProfileTab.saved` shall be removed. | Private account data belongs in Settings and should not appear as public-profile content. | Prompt; Decision Q3 | AC-007, AC-026 |
| FR-009 | Functional | Must | The Saved route shall always enter on a collection overview that renders every owned folder first in AppView alphabetical order, then all unfiled saved posts. Foldered posts shall not be flattened into the overview. Folder rows shall show no save counts. Newest/Oldest shall sort the visible unfiled posts only and shall not affect folder order. | This is the user-confirmed meaning of the root Saved view. | User grilling; AppView contract | AC-008, AC-009, AC-028 |
| FR-010 | Functional | Must | Folder rows, overview unfiled posts, and folder-screen posts shall each support their applicable opaque-cursor pagination without duplicates, plus refresh, first/incremental loading, recoverable errors, and invalid-cursor restart. The overview shall preserve folders-before-unfiled rendering as folder pages load. After a successful folder create, rename, or delete, Flutter shall reconcile or safely restart the affected folder collection from server order, invalidate an unsafe prior cursor, and deduplicate by opaque folder ID. | Both resource types can exceed one page, and folder mutations can invalidate a partially loaded alphabetical cursor without changing the confirmed hierarchy. | AppView contract; Existing list patterns; User grilling; Document review | AC-009, AC-010, AC-019, AC-028, AC-031 |
| FR-011 | Functional | Must | Every visible unfiled/folder item shall show `PostSummary` plus saved time and allow exact-item opening, moving through a chooser preselected to the current folder/Unfiled, and confirmation-free unsave. Moving shall keep the chooser open until AppView confirms and remove/reposition the item only after confirmation. | “View/manage” includes navigation and individual organization/removal without optimistic assignment drift. | Prompt; AppView contract; User grilling | AC-011, AC-012, AC-030 |
| FR-012 | Functional | Must | Opening a saved comment or reply shall navigate to its root thread with the exact saved URI focused; opening a top-level post shall open that post normally. | The saved item is the exact record, not its root. | AppView contract; Existing navigation | AC-011 |
| FR-013 | Functional | Must | The overview app bar shall expose Add folder. Tapping a folder shall open a separate screen titled with its name; Rename and Delete shall live in that screen's app-bar overflow. Folder input shall match server validation and allow duplicate names. A confirmed rename shall update the open screen title and move the row identified by opaque ID to its server-ordered alphabetical position; a failed rename shall retain the last confirmed name. Folder navigation/diagnostics shall use a generic route name without recording the private folder ID or name. | Creation must work from an empty overview, while management belongs to a clear folder context, alphabetical state must remain server-confirmed, and private identifiers stay out of breadcrumbs. | AppView contract; User grilling; Document review | AC-005, AC-013, AC-017, AC-031 |
| FR-014 | Functional | Must | Folder deletion shall always offer Cancel, Delete folder and keep saved posts, and Delete folder and remove saved posts, even when no saves are visible. Cancel shall receive default keyboard focus; remove-saves shall receive the strongest destructive styling. | Flutter cannot prove emptiness, and users must understand/control data-loss scope. | Prompt; User grilling | AC-013, AC-023 |
| FR-015 | Functional | Must | AppView shall extend `DELETE /v1/saved-post-folders/{folderId}` with optional `deleteSaves=true`. Omitted/false shall retain current atomic unfile behavior; true shall atomically delete every save owned by the requester in that folder and then delete the folder. Missing/other-owner IDs shall remain idempotent `204` no-ops, and invalid values/unknown query parameters shall use the standard validation envelope. | Only the server can completely and atomically remove all folder saves. | Discovery; API architecture | AC-014, AC-015, AC-016, AC-025 |
| FR-016 | Functional | Must | The delete-with-saves AppView path shall modify only private saved rows/folders and shall not delete indexed post rows, call a PDS, emit a record/event, or expose another owner's resource existence. | “Remove saves” must never become post deletion or a privacy leak. | Architecture | AC-014, AC-025 |
| FR-017 | Functional | Must | Flutter shall never implement delete-with-saves by enumerating hydrated saved-list items and issuing per-post deletes. | Hidden items, pagination, partial failure, and rate limits make recursion incomplete. | Discovery | AC-015 |
| FR-018 | Functional | Must | Saved viewer state shall have one account-scoped client synchronization seam keyed by canonical post URI, seeded/reconciled from server state, so loaded post surfaces and the Saved collection converge after save, move, unsave, and folder deletion without manually forking business rules per screen. | There is no global normalized post cache, and duplicated mutation logic will drift. | Codebase; Recommended direction | AC-018, AC-024, AC-026 |
| FR-019 | Functional | Must | Every asynchronous saved/folder operation shall capture the initiating account and ignore/cancel stale completion after account switch; loaded list, folder, dialog, mutation, and optimistic overlay state shall be account-scoped or invalidated on switch. | Cross-account UI mutation would breach privacy and corrupt state. | Multi-account architecture; Existing pattern | AC-026 |
| FR-020 | Functional | Must | A reusable presentational `PostSummary` shall accept a common compact post representation and parent-owned callbacks/slots, support author identity and optional time/metadata, render bounded text, optional first image/project title, and explicit protected/unavailable presentation, and contain no engagement or bookmark controls. | Compact summaries navigate to full posts rather than duplicating actions. | Prompt; Codebase; User grilling | AC-020, AC-021, AC-022, AC-023, AC-032 |
| FR-021 | Functional | Must | Quote previews shall use `PostSummary` while preserving visible/hidden/muted/blocked/unavailable states, reveal behavior, author tap, post tap, one-level rendering, project title, representative image, and current policy copy. | Extraction must not regress quote behavior. | Codebase | AC-020 |
| FR-022 | Functional | Must | Post-bearing notification rows shall use `PostSummary` for the subject preview while preserving actor/action title, category icon/color, timestamp, follow control, notification filtering, and exact existing destination/focus behavior. | The shared summary must not erase notification context or routing. | Prompt; Codebase | AC-021 |
| FR-023 | Functional | Must | Saved-post rows shall use `PostSummary` while keeping save-specific metadata and actions outside the reusable summary's core content. | Saved management should reuse presentation without coupling the widget to folders. | Prompt; Recommended direction | AC-022 |
| FR-024 | Functional | Must | All new visible strings, semantic labels/hints, empty/error states, and destructive copy shall use app localization resources; interactive summary/bookmark/folder controls shall expose accessible names, selected/busy state, adequate tap targets, text scaling, and keyboard/focus order. | The feature must be usable beyond a pointer-driven English happy path. | Codebase quality standard | AC-023 |
| FR-025 | Functional | Must | Recoverable API failures shall retain the last confirmed state, show localized user feedback, and provide retry where the failed operation is not safely repeated automatically. Server error messages and private identifiers shall not be shown directly to the user. | Private operations need clear but sanitized failure handling. | API contract; Privacy | AC-019, AC-025 |
| FR-026 | Functional | Must | Folder selection and management shall always use opaque folder IDs for mutations and filters; display names shall never be treated as unique identity. | Duplicate names are allowed by the implemented contract. | AppView contract | AC-004, AC-017, AC-025 |
| NFR-001 | Non-functional | Must | Flutter and AppView changes shall preserve authenticated owner scoping, private saved data, author non-disclosure, and active-account isolation in UI state, requests, responses, logs, error reports, traces, and analytics dimensions. | Saved behavior and folder names are sensitive. | Architecture | AC-025, AC-026 |
| NFR-002 | Non-functional | Must | Saved/folder lists shall remain bounded and cursor-paginated; UI rendering and state updates shall avoid per-item network requests or folder-name lookups. | Large collections must not create N+1 behavior. | AppView contract | AC-009, AC-010 |
| NFR-003 | Non-functional | Must | Save, move, folder-create/rename/delete controls shall disable or serialize conflicting duplicates and show busy state while awaiting confirmation; unsave shall show immediate optimistic state. No operation shall allow double submission. | Rapid taps and slow networks must not produce ambiguous state. | Existing Flutter pattern; User grilling | AC-006, AC-018, AC-029, AC-030 |
| NFR-004 | Non-functional | Should | `PostSummary` should remain small and surface-agnostic, with visual variants/slots limited to differences demonstrated by quote, notification, and saved consumers. | A speculative widget API would recreate complexity in a different place. | Recommended direction | AC-020, AC-021, AC-022 |
| NFR-005 | Non-functional | Must | Automated coverage shall include model decoding, API/repository contracts, provider/account behavior, bookmark/dialog interactions, route/settings behavior, saved/folder pagination and management, delete modes, `PostSummary` consumers, accessibility semantics, and regressions for quote/notification navigation and visibility. | The feature crosses private data, routing, async state, and three existing UI surfaces. | Workflow quality standard | AC-027 |
| NFR-006 | Non-functional | Should | No new runtime dependency shall be introduced unless test design proves existing Flutter/Dart facilities cannot meet the requirements and the addition is explicitly approved. The completion gate shall inspect Flutter and Go dependency manifests/lockfiles rather than relying on runtime tests alone. | The repository already has the required networking, state, routing, and UI primitives, and dependency absence is a source-diff property. | Codebase; Document review | AC-027 |
| RULE-001 | Business rule | Must | An unsaved post begins the chooser with No folder selected; one save may have at most one folder; no save occurs until the user presses Save. | Saving should be low-friction but explicitly confirmed. | Prompt; AppView contract; User grilling | AC-003, AC-030 |
| RULE-002 | Business rule | Must | A selected bookmark unsaves immediately and optimistically without confirmation or Undo; a deselected bookmark never saves until the chooser is confirmed. | This is the confirmed interaction rule, and Undo cannot preserve chronology. | Prompt; User grilling | AC-002, AC-006, AC-029 |
| RULE-003 | Business rule | Must | Deleting a folder without `deleteSaves=true` preserves saves as unfiled. Deleting with `deleteSaves=true` removes saved records only, never source posts. | The two choices need exact, safe semantics. | Prompt; Architecture | AC-013, AC-014 |
| RULE-004 | Business rule | Must | Successfully creating a folder from the chooser persists it even if the later post-save action is canceled; the new folder becomes the chooser selection. | Folder creation and saving are distinct resource operations. | Decision Q5 | AC-005 |
| RULE-005 | Business rule | Must | Moving a saved post or deleting its folder does not change `savedAt`; unsaving and later saving again receives a new server timestamp. | UI ordering must continue to mean save chronology. | AppView contract | AC-008, AC-012, AC-013 |
| RULE-006 | Business rule | Must | Saved posts are private account settings, not profile content; no profile visitor can browse or infer them. | Placement must reinforce the privacy model. | Decision Q3; Architecture | AC-007, AC-025 |
| RULE-007 | Business rule | Must | Folder names may duplicate and differ only by case; folder ID is the sole identity. | This is the implemented server contract. | AppView contract | AC-004, AC-017 |
| RULE-008 | Business rule | Must | Visible unfiled/folder post lists sort by `savedAt`, default to newest first, and remain independent of post time/folder edits; folder rows remain alphabetically ordered. | Sorting must reflect the resource currently displayed. | AppView contract; User grilling | AC-008, AC-028, AC-031 |
| RULE-009 | Business rule | Must | A completion belonging to a no-longer-active account shall not mutate the active account's bookmark, folder, list, navigation, or message state. | Late responses are expected during account switches. | Multi-account architecture | AC-026 |
| RULE-010 | Business rule | Must | The Saved overview is folders followed by unfiled posts; foldered posts appear only inside their folder screen. If folders exist but unfiled is empty, the Unfiled section is hidden; the full empty state appears only when both are empty. | This is the confirmed collection hierarchy and empty-state behavior. | User grilling | AC-028 |
| RULE-011 | Business rule | Must | Folder rows expose no counts, and the feature includes no folder search in this slice. | Counts/search cannot be derived correctly from partial or policy-filtered client data. | User grilling; AppView contract | AC-028, AC-031 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-001, FR-002 | Given authenticated post JSON with saved fields, when Flutter decodes and renders an eligible full card, then an unsaved post shows an outlined localized bookmark and a saved post shows a selected filled bookmark with no save count; a protected placeholder that omits viewer fields decodes safely and exposes no bookmark. |
| AC-002 | BR-001, FR-003, RULE-002 | Given an unsaved post, when its bookmark is tapped, then the chooser opens and no save request is sent before Save; given a saved post, when its bookmark is tapped, then optimistic unsave begins immediately with no confirmation or Undo. |
| AC-003 | FR-004, RULE-001 | Given a new save and any folder collection, when the chooser opens, then No folder is selected, exactly one option can be selected, and only pressing the separate Save button sends an explicit unfiled/foldered save. |
| AC-004 | FR-004, FR-026, RULE-007 | Given more than one page of folders including duplicate names, when the user scrolls/selects, then every folder can be loaded and each selection/mutation uses its opaque ID without collapsing same-named entries. |
| AC-005 | FR-005, FR-013, RULE-004 | Given the save/move chooser, when New folder is selected, then an inline name field expands; valid creation persists, collapses the form, inserts/selects the folder, and survives later chooser cancellation, while invalid/failed creation remains editable and retryable. |
| AC-006 | BR-001, FR-003, FR-006, NFR-003, RULE-002 | Given save/move is in flight, then its chooser stays open with one busy confirm action until confirmed and retains selection with inline error on failure; given unsave is in flight, then the bookmark updates optimistically once and restores the exact prior state with localized error feedback if deletion fails. |
| AC-007 | BR-002, FR-008, RULE-006 | Given a signed-in user in Settings, when they tap Saved posts, then a typed full-screen `/profile/settings/saved` route opens and pops back to Settings; no Saved tab appears on own or visited profiles and `/profile/saved` is no longer canonical. |
| AC-008 | BR-002, FR-009, RULE-005, RULE-008 | Given folders plus foldered/unfiled saves, when the overview opens, then folders remain alphabetical and only unfiled posts appear beneath them; Newest/Oldest orders visible posts by server `savedAt`, not post time, and folder screens apply the same post sort. |
| AC-009 | FR-009, FR-010, NFR-002 | Given folders, unfiled saves, or one folder's saves longer than a page, when load-more runs, then each resource's opaque cursor is round-tripped, items append once in server order, and the overview preserves folders-before-unfiled rendering. |
| AC-010 | FR-010, NFR-002 | Given first-page, incremental, refresh, invalid-cursor, and retry states for folder rows, unfiled posts, or folder contents, when each occurs, then the relevant screen/section shows correct recoverable UI and invalid cursor restarts that list safely without parsing it. Given a partially loaded folder collection, when create, rename, or delete succeeds, then the affected folder collection reconciles or restarts from server order, discards any unsafe cursor, and contains each opaque folder ID at most once. |
| AC-011 | BR-002, FR-011, FR-012 | Given a visible saved top-level post, comment, and nested reply in Unfiled or a folder, when each summary is rendered/tapped, then it shows saved time; top-level content opens normally and each reply opens its root thread focused on the exact saved URI. |
| AC-012 | FR-011, RULE-005 | Given a saved item, when its move chooser opens, then its current folder or Unfiled is preselected; moving leaves one assignment and does not change `savedAt`, while the chooser closes and the item repositions only after confirmation; unsave remains confirmation-free. |
| AC-013 | BR-003, FR-013, FR-014, RULE-003, RULE-005 | Given any folder, even one with no visible saves, when delete is requested, then the dialog offers Cancel, keep saves as unfiled, or remove saved posts; Cancel has default focus, remove-saves has strongest destructive styling, and keep preserves every save/timestamp. |
| AC-014 | BR-003, FR-015, FR-016, RULE-003 | Given a folder containing visible and policy-hidden saves, when delete-with-saves is confirmed, then one authenticated `DELETE ...?deleteSaves=true` atomically removes all owned saves in that folder and the folder, returns `204`, and leaves all public/indexed/PDS posts unchanged. |
| AC-015 | FR-015, FR-017 | Given a missing/other-owner folder or a failure during delete-with-saves, when the endpoint runs, then missing/other-owner remains an indistinguishable `204` no-op and a real failure commits neither partial save deletion nor partial folder deletion; Flutter issued no recursive per-post deletes. |
| AC-016 | FR-015 | Given folder delete query input is absent or false, true, invalid, or unknown, when the handler parses it, then absent/false uses unfile behavior, true uses delete-saves behavior, and invalid/unknown input receives the standard camelCase validation error envelope. |
| AC-017 | FR-004, FR-005, FR-013, FR-026, RULE-007 | Given create/rename/select/list operations with whitespace, Unicode, slash, control characters, duplicate names, and page boundaries, when Flutter validates and the server responds, then accepted names/IDs match the AppView contract and no client uniqueness rule is invented. |
| AC-018 | FR-006, FR-018, NFR-003 | Given the same URI is visible in multiple loaded post surfaces, when save, move, unsave, or folder deletion succeeds/fails, then every surface that reads the account-scoped saved-state seam converges to the same confirmed bookmark/folder state without duplicate requests. |
| AC-019 | FR-010, FR-025 | Given saved/folder API errors, when the UI handles them, then confirmed content remains usable where possible, localized retry/error feedback is shown, and raw server messages, folder IDs/names, post URIs, and owner-target pairs are not displayed. |
| AC-020 | BR-004, FR-020, FR-021, NFR-004 | Given visible and protected quote-view fixtures, when `PostCard` renders its quote, then action-free `PostSummary` supplies compact content while all existing quote states, reveal action, author/post taps, text truncation, first image/project title, and one-level behavior remain intact. |
| AC-021 | BR-004, FR-020, FR-022, NFR-004 | Given every post-bearing notification category, when a row renders and is tapped, then `PostSummary` renders the subject while the actor/action context, category treatment, timestamp, filtering, and exact existing destination/focus behavior are unchanged. |
| AC-022 | BR-004, FR-020, FR-023, NFR-004 | Given a saved item, when it renders, then the same action-free `PostSummary` core is used and saved-time/move/unsave controls remain parent-owned rather than embedded as saved-specific widget logic. |
| AC-023 | FR-002, FR-014, FR-020, FR-024 | Given screen-reader, keyboard, large-text, and touch interaction, when bookmark, chooser, folder, destructive, and summary controls are used, then labels, selected/busy/destructive semantics, Cancel-first destructive focus order, wrapping/truncation, and tap targets remain understandable and operable. |
| AC-024 | FR-001, FR-006, FR-018 | Given a canonical post is refetched after an optimistic mutation or appears with stale cached data, when server viewer state is reconciled, then the account-scoped seam adopts the confirmed `viewerHasSaved`/folder state without changing another account. |
| AC-025 | BR-005, FR-015, FR-016, FR-025, FR-026, NFR-001, RULE-006 | Given Alice and Bob plus a post author, when every save/folder operation succeeds or fails, then only the authenticated owner's state is readable/mutable, the author receives no signal, another user's folder existence is not disclosed, no PDS write occurs, and private values are absent from diagnostics/analytics. |
| AC-026 | BR-005, FR-008, FR-018, FR-019, NFR-001, RULE-009 | Given Alice starts an operation and the app switches to Bob before completion, when Alice's response arrives, then Bob's bookmark/list/folder/navigation/message state does not change; switching back/refetching shows each account's own server state. |
| AC-027 | FR-007, NFR-005, NFR-006 | Given the feature test gates run, then model, API/repository, provider, route, widget, accessibility, AppView handler/store/privacy/concurrency, quote/notification regression, `flutter analyze`, Flutter tests, Go formatting, and focused/full AppView tests pass without an unapproved dependency. |
| AC-028 | FR-009, FR-010, RULE-008, RULE-010, RULE-011 | Given folders, foldered saves, and unfiled saves, when the overview renders, then every folder appears first without a count, foldered posts are absent, unfiled posts follow, sort changes only visible posts, empty Unfiled is hidden when folders exist, and the full empty state appears only when both sections are empty. |
| AC-029 | FR-003, FR-006, NFR-003, RULE-002 | Given a saved bookmark, when unsave begins, then it becomes outlined immediately, blocks duplicate taps, offers no Undo, and either remains unsaved on `204` or restores the exact previous folder/saved state on failure. |
| AC-030 | FR-004, FR-005, FR-006, FR-011, NFR-003, RULE-001 | Given save or move, when its chooser is used, then it has one explicit confirm button, inline New folder, no folder search, and current/default selection; a folder-list failure remains retryable while No folder works, success closes silently, and failure stays inline without closing. |
| AC-031 | FR-010, FR-013, RULE-008, RULE-011 | Given the overview or a folder screen, when the user creates/opens/manages a folder, then Add folder is available in the overview app bar, tapping a folder opens a separately navigable titled screen, Rename/Delete are in overflow, contents sort by save time, and back restores the overview. A confirmed rename updates the open title and alphabetically positioned overview row without changing opaque identity; failure retains the confirmed name. Successful create/rename/delete reconciles partial folder pagination without duplicates or a dangling selection, and private folder ID/name are absent from navigation diagnostics. |
| AC-032 | FR-002, FR-020 | Given eligible full top-level/project/quote/comment/reply cards plus quote/notification summaries and protected placeholders, when rendered, then only full eligible cards show the bookmark at the right immediately before overflow; compact summaries and placeholders contain no bookmark. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | Same post rendered in multiple screens/caches | All bookmark affordances use/reconcile through the same active-account URI state. | FR-018 |
| EC-002 | Rapid repeated bookmark taps | One mutation runs; the action is busy/disabled until reconciliation. | FR-006, NFR-003 |
| EC-003 | Save dialog dismissed | No post save occurs; a folder successfully created earlier in that dialog remains. | FR-004, FR-005, RULE-004 |
| EC-004 | Folder list exceeds 50/100 entries | Dialog and management surfaces load subsequent opaque-cursor pages. | FR-004, FR-010 |
| EC-005 | Duplicate folder names | Both remain visible as distinct entries and ID selects the intended one. | FR-026, RULE-007 |
| EC-006 | Current folder is deleted while chooser/page is open | Refresh/reconciliation removes the folder, uses server-confirmed unfile/deleted state, and avoids a dangling selection. | FR-018, FR-025 |
| EC-007 | Saved item moved out of the current folder or Unfiled | After AppView confirms, the item disappears from the current post list, appears only in its destination folder/Unfiled section, and preserves `savedAt`. | FR-011, RULE-005, RULE-010 |
| EC-008 | Saved item unsaved from Saved route | Item is removed from the current visible post list without confirmation; a later new save gets a new server timestamp. | FR-011, RULE-005 |
| EC-009 | Saved reply | Root-plus-focus navigation opens the exact saved reply. | FR-012 |
| EC-010 | Policy-hidden save in a folder | It may be absent from Flutter's hydrated page but is still removed by atomic delete-with-saves. | FR-015, FR-017 |
| EC-011 | Delete missing or cross-owner folder | AppView returns `204` without revealing existence or affecting either owner. | FR-015, NFR-001 |
| EC-012 | Delete-with-saves fails transactionally | Folder and every contained save retain their prior state; UI refreshes/reconciles and reports failure. | FR-015, FR-025 |
| EC-013 | Invalid/stale saved-list cursor | Flutter restarts the selected scope/sort from page one after `invalid_cursor`. | FR-010 |
| EC-014 | Account switches during dialog or request | Dialog/provider result cannot commit into the new account's state. | FR-019, RULE-009 |
| EC-015 | Quote is muted/revealable, blocked, hidden, or unavailable | Shared summary preserves the current placeholder/reveal behavior and never reveals absent data. | FR-021 |
| EC-016 | Notification subject is a comment/reply | Summary remains compact and row navigation preserves current root/focus inference. | FR-022 |
| EC-017 | Large text or long Unicode folder/post text | Controls remain reachable; summary text is bounded; folder input length follows Unicode-character rules. | FR-013, FR-024 |
| EC-018 | Save/move or unsave failure | Save/move keeps the chooser open on the attempted selection; unsave restores prior confirmed state; neither surfaces private raw error details. | FR-006, FR-025 |
| EC-019 | Many folder pages precede Unfiled | Folder pages load incrementally and remain above the independently paginated unfiled section; foldered posts never leak into the overview. | FR-009, FR-010, RULE-010 |
| EC-020 | Folder list fails in save chooser | No folder remains selectable/savable while the folder area offers inline Retry. | FR-004 |
| EC-021 | Folder appears empty but has hidden saves | The delete dialog still offers both keep-saves and remove-saves; only the atomic AppView choice determines hidden-row behavior. | FR-014, FR-015 |
| EC-022 | Folder create, rename, or delete succeeds while the folder collection is only partially loaded | The affected folder collection discards any unsafe cursor and reconciles or restarts from page one in server order, deduplicating by opaque ID. The selected new folder remains selected when still valid, a confirmed rename updates the open title, a deleted selection is cleared, and overview scroll restoration is preserved where the route remains open. | FR-005, FR-010, FR-013, FR-018, FR-026 |

## 15. Data / Persistence Impact

- Flutter models:
  - Add `viewerHasSaved` and nullable `viewerSavedFolderId` to `Post` and generated mapping/copy support.
  - Add typed `SavedPostState`, `SavedPostFolder`, `SavedPostItem`, saved/folder page, scope, and sort models as needed.
  - Add a small internal compact-summary representation/adapter; it is not a new wire format.
- Flutter state:
  - Add account-scoped folder/list/mutation state and a canonical URI saved-state synchronization seam.
  - No durable local saved-post database or content snapshot.
- AppView persistence:
  - No migration or new table is required.
  - Extend the existing folder deletion transaction to optionally delete owned `saved_posts` rows before deleting the owned folder.
- Backwards compatibility:
  - `deleteSaves` is additive; absent/false preserves current behavior.
  - The app is pre-production, so replacing the placeholder saved route/profile tab requires no compatibility alias.

## 16. UI / API / CLI Impact

- UI:
  - Right-aligned bookmark action on eligible full post cards, including comment/reply cards; none on compact summaries/protected placeholders.
  - Save/move chooser with No folder/current selection, explicit confirmation, paginated folders, inline folder creation, no search, loading, and inline errors.
  - Settings tile and full-screen Saved overview with alphabetical no-count folders followed by unfiled posts, separate folder screens, saved-time sort, pagination, refresh, summaries, and item actions.
  - Overview Add folder action; folder-screen title plus Rename/Delete overflow actions.
  - Destructive folder dialog with keep-saves versus remove-saves choices.
  - Reusable `PostSummary` adopted by quotes, post-bearing notifications, and saved rows.
  - Remove the current Saved profile tab and `/profile/saved` placeholder.
- API:
  - Consume existing save/unsave, saved-list, and folder list/create/rename/delete endpoints.
  - Add optional `deleteSaves=true` to `DELETE /v1/saved-post-folders/{folderId}`; the route remains authenticated, camelCase, no-body, write-rate-class, and returns `204` on success/idempotent absence.
  - No other AppView response-shape change is required.
- CLI: None.
- Background jobs: None.

## 17. Security / Privacy / Permissions

- Authentication: Every consumed/extended saved endpoint uses the existing Craftsky session and device-ID middleware.
- Authorization: Owner always comes from authenticated context. Flutter sends no owner DID, and opaque folder ID never grants access.
- Sensitive data: Saved URIs, folder IDs/names, assignments, timestamps, and owner-target relationships are private behavioral data.
- Multi-account: State, optimistic overlays, in-flight completions, dialogs, messages, pagination cursors, and repository actions must remain account-scoped.
- Author privacy: No notification, count, event, PDS write, or UI existence signal is sent to the post author.
- Destructive scope: Delete-with-saves removes private saved records only. It cannot call public post deletion or PDS endpoints.
- Content safety: Flutter renders only the post/quote availability shape supplied by AppView and must not reconstruct hidden content.
- Diagnostics: Folder names/IDs, saved URIs, and owner-target pairs must not be logged or attached to Sentry/analytics. Bounded operation/result/error-class values are acceptable.

## 18. Observability

- Events: No new product analytics event is required.
- Logs/error reporting: Existing sanitized Flutter/API error handling may record bounded operation/stage/error class and request correlation, not private saved/folder identifiers or names.
- Metrics: The AppView extension may reuse existing bounded HTTP/store metrics for success/failure; `deleteSaves` must not become an identifier-bearing dimension.
- Alerts: None added for this pre-production feature.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Client recursion misses hidden saves or partially deletes a folder. | “Remove all” is false and private data remains unexpectedly. | Require one atomic AppView delete mode and prohibit client enumeration. |
| RISK-002 | Saved state is copied independently into many post providers. | Bookmark icons drift across feed, profile, search, thread, notification, and saved views. | Use one account-scoped URI synchronization seam and reconcile server state. |
| RISK-003 | Late async completion crosses an account switch. | Privacy breach or incorrect active-account UI. | Capture account ownership, key providers by account, and ignore/cancel stale completion. |
| RISK-004 | Folder deletion copy is ambiguous. | User believes source posts are deleted or unintentionally loses saves. | Use “remove saved posts” wording, three explicit choices, and safe default focus. |
| RISK-005 | `PostSummary` becomes an over-generalized replacement for `PostCard`. | Complex API and regressions across unrelated surfaces. | Limit it to compact common content and keep surface metadata/actions in parent widgets. |
| RISK-006 | Quote/notification extraction changes policy or routing. | Hidden content exposure or wrong post/thread destination. | Preserve current state/destination inputs and add regression tests for every state/category. |
| RISK-007 | Duplicate folder names are treated as unique in Flutter. | Wrong folder is selected, renamed, filtered, or deleted. | Use opaque ID for all state/mutations and test duplicate/case-variant names across pages. |
| RISK-008 | Pending save/move state or optimistic unsave is not reconciled with server responses. | Saved lists and bookmark state remain wrong after failure or concurrent change. | Keep confirmed versus pending state distinct and refetch/invalidate on ambiguous failure. |
| RISK-009 | Private values enter diagnostics. | Sensitive behavioral data disclosure. | Sanitize client/server errors and test sentinel values across representative failures. |
| RISK-010 | Folder mutations leave a stale alphabetical cursor or duplicate folder row. | Later pagination can omit, duplicate, or misorder folders, and a chooser can retain a deleted selection. | Reconcile or restart the affected folder collection after confirmed create/rename/delete, invalidate unsafe cursors, and deduplicate solely by opaque folder ID. |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-005 | `PostSummary` includes one representative image and project title when available, matching the existing quote preview, rather than full galleries/project cards. | A richer saved-list design would require additional layout/performance requirements. |

## 21. Open Questions

None blocking. ASM-005 is the only remaining presentation assumption; every information-architecture, mutation, dialog, and destructive-action decision was confirmed during requirements grilling.

## 22. Review Status

Status: Reviewed
Risk level: Medium
Review recommended: Completed
Reviewer: Product owner; Codex document review
Date: 2026-07-21
Notes: Grilling confirmed the Settings-only route, folder-overview hierarchy, separate folder screens, no counts/search, app-bar management actions, bookmark placement/scope, explicit save/move confirmation, optimistic unsave without Undo, folder-list degradation, silent success, and always-explicit folder deletion choices. Document review made folder-mutation cursor reconciliation and dependency-diff verification explicit without changing product scope or IDs. Medium risk remains because of private multi-account state, atomic delete-with-saves, dual-resource overview pagination, and regression-sensitive summary extraction.

## 23. Handoff To Test Design

- Requirements file: `01-requirements.md`
- Next test specification: `02-acceptance-tests.md`
- Must-cover requirement IDs: BR-001–BR-005, FR-001–FR-026, NFR-001–NFR-003, NFR-005, RULE-001–RULE-011.
- Suggested test levels:
  - Unit/model: saved wire decoding, folder/name validation parity, post-summary adapters, route locations, exact reply destination, confirmed/pending saved-state reducer.
  - API/repository: every existing saved/folder request and response, explicit nullable `folderId`, opaque cursors/IDs, `deleteSaves` query serialization, standard errors.
  - Provider/state: independent folder/unfiled/folder-content pagination, saved-time sort, confirmation-driven save/move, optimistic unsave rollback, invalid cursor, duplicate folders, create/rename/delete cursor reconciliation and ID deduplication, live URI overlay, account switch and stale completion.
  - Widget: right-aligned full-card bookmark coverage, action-free summaries, chooser/create/degraded-folder flow, Settings overview hierarchy/empty states, separate folder screen/actions, folder delete choices/focus, large text/keyboard/screen reader, duplicate names.
  - Regression widget: visible/protected quote summaries, every post-bearing notification category and destination, full `PostCard` action layout, removal of profile Saved tab.
  - AppView unit/integration: query parsing, handler/status/error contract, atomic keep-versus-delete transaction, hidden saves, concurrency/idempotency, missing/cross-owner privacy, no public/PDS deletion, diagnostic redaction.
  - Gates: targeted Flutter tests, `just app-analyze`, `just app-test`, `just fmt`, focused AppView tests, `just test`, and explicit Flutter/Go dependency manifest and lockfile diff review.
- Blocking open questions: None.
