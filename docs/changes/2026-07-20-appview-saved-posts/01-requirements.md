# Requirements: AppView Saved Posts

## 1. Initial Request

Implement the AppView slice of private saved posts for Craftsky. The behavior should broadly match Bluesky bookmarks, but Craftsky users may optionally organize saved posts into one-level folders. A saved post may belong to at most one folder, or remain unfiled. Any indexed Craftsky post, including a top-level post, project post, comment, or nested reply, may be saved. Saved-post lists must sort by the time the user saved the post, in either newest-first or oldest-first order. Saved posts and folders are private AppView data and must not be written to the user's PDS.

## 2. Current Codebase Findings

- Relevant files:
  - Architecture and privacy boundary: `atproto-craft-social-app-reference.md`, `AGENTS.md`.
  - API conventions: `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md`, `docs/superpowers/specs/2026-04-22-api-wire-alignment-design.md`.
  - Route registration and route policies: `appview/internal/routes/routes.go`, `appview/internal/routes/policy.go`.
  - Canonical post handlers, storage, and response model: `appview/internal/api/post.go`, `appview/internal/api/post_store.go`, `appview/internal/api/post_response.go`.
  - Indexed post schema: `appview/migrations/000010_craftsky_posts.up.sql` plus later post migrations.
  - Existing private owner-scoped precedent: `appview/migrations/000023_mutes_blocks.up.sql`, `appview/internal/relationships/store.go`, `appview/internal/relationships/lifecycle.go`.
- Existing patterns:
  - Every applicable `/v1/*` request uses Craftsky session authentication, `X-Craftsky-Device-Id`, route body limits, read/write rate classes, camelCase JSON, and the standard `{error, message, requestId}` error envelope.
  - List APIs return object-wrapped `items` and an optional opaque `cursor`; default limits are normally 50 and maximum limits are 100.
  - `PostResponse` is the canonical post-shaped response for top-level posts, project posts, comments, and replies. Replies are rows in `craftsky_posts`, distinguished by `replyRootUri` and `replyParentUri` relationships rather than a separate record type.
  - Canonical reply responses already carry root and parent strong references. Existing comment deep links open the root post route with the exact comment/reply AT-URI as `focus`; the comments handler rejects a reply used as the route root and reconstructs the focused branch from its required ancestors.
  - Viewer-relative post state is already hydrated for likes, reposts, replies, mutes, and blocks.
  - Private mutes are keyed to an owning DID in AppView Postgres, are retained across session/device events, and are deleted when the owning Craftsky membership is permanently removed.
- Current behavior:
  - Craftsky has no saved-post or saved-post-folder tables, handlers, routes, viewer state, or list API.
  - Saving a post is not represented on the PDS or in the AppView.
  - All indexed ordinary posts, project posts, comments, and replies share a canonical AT-URI identity in `craftsky_posts`.
- Constraints discovered:
  - Private-by-intent data must remain in AppView Postgres; this slice requires no lexicon change and no PDS write.
  - Saved-post APIs must not bypass current membership, moderation, mute, or block policy when hydrating referenced posts.
  - Multi-account state must be keyed to the authenticated DID and must never be inferred from a client-supplied owner identifier.
  - The current highest migration is `000023`; implementation must re-check the migration number before adding the saved-post migration.
- Test/build commands discovered:
  - AppView test gate: `just test` from the repository root with the compose Postgres available.
  - Formatting: `just fmt`.
  - Local stack: `just dev`.

## 3. Clarifying Questions And Decisions

### Q1: May one saved post appear in more than one folder?

Answer: No. A saved post may belong to one folder or no folder.

Decision / implication: Folder membership is a nullable single assignment on the unique owner/post save. Moving a save replaces its prior folder assignment; no many-to-many join or duplicate saved-list entries are required.

### Q2: Which timestamp controls chronological ordering?

Answer: The time the user saved the post.

Decision / implication: Saved lists order by server-assigned `savedAt`, not post `createdAt` or AppView `indexedAt`. Moving a save between folders does not change `savedAt`; removing and later saving it again creates a new `savedAt`.

### Q3: How does a saved comment or reply retain navigable thread context?

Answer: Save the exact comment or reply URI. Use its canonical `PostResponse.reply.root` and `PostResponse.reply.parent` references to open the root thread with the saved URI as the focused item; do not duplicate parent content in private saved-post storage.

Decision / implication: The saved item remains the exact selected post. Temporary inability to resolve its required thread context hides but retains the save. Permanent deletion of its root or any required ancestor removes the now-orphaned saved reply because the existing thread route can no longer open it meaningfully.

### Q4: What happens to saves when a folder is deleted?

Answer: They become unfiled.

Decision / implication: Folder deletion removes only the organization container, preserves each save and its `savedAt`, and never implicitly unsaves content.

### Q5: Must folder names be unique per owner?

Answer: No. Folders behave as independently identified resources, so exact duplicate names and names differing only by case are allowed.

Decision / implication: Opaque folder ID is the sole folder identity. There is no normalized-name uniqueness constraint or `folder_name_conflict` error.

### Q6: How are folders ordered?

Answer: Alphabetically by display name, case-insensitively, with folder ID ascending as the stable tie-breaker.

Decision / implication: Duplicate names paginate deterministically without manual ordering.

### Q7: What does an omitted `folderId` mean when saving?

Answer: For a new save, an absent body or omitted `folderId` creates an unfiled save. For an existing save, either preserves its current folder. Explicit `"folderId": null` moves an existing save to unfiled.

Decision / implication: Retried or repeated save actions cannot silently discard an existing folder assignment, while the request still supports an explicit unfile operation.

### Q8: Are otherwise eligible posts from a muted author visible in saved posts?

Answer: Yes.

Decision / implication: Saved posts are an explicit owner-curated surface and retain current direct-access mute shaping. Blocks, hides, takedowns, non-membership, and other stricter availability decisions still prevent content exposure.

### Q9: How does a folder-scoped read handle a missing or other-owner folder ID?

Answer: Return the same `404 saved_post_folder_not_found` response for both.

Decision / implication: The API detects stale or invalid folder references without exposing another owner's folder existence.

### Q10: How does folder deletion handle a missing or other-owner folder ID?

Answer: Return `204` as an idempotent no-op for both.

Decision / implication: Delete retries remain safe and disclose no ownership or existence signal. Rename and folder-scoped reads still require an existing owned folder and return the indistinguishable 404 otherwise.

### Q11: What folder-name validation applies?

Answer: Trim surrounding whitespace, require 1–100 Unicode characters after trimming, and reject `/`, `\\`, and control characters. Other printable Unicode, punctuation, and emoji are allowed.

Decision / implication: Folder names remain expressive but cannot be blank, impractically large, unsafe to display, or misleadingly path-like in a flat folder model.

### Q12: Do folder responses include save counts in this slice?

Answer: No.

Decision / implication: Folder resources contain identity, name, and timestamps only. Policy-consistent counts may be added later as an additive contract.

### Q13: Are there hard per-account folder or saved-post quotas in this slice?

Answer: No.

Decision / implication: Existing authenticated rate limits and bounded pagination apply; product quotas remain future work informed by real storage and abuse needs.

### Q14: Does a normal post response embed a saved folder's name?

Answer: No. It exposes only `viewerHasSaved` and nullable `viewerSavedFolderId`.

Decision / implication: Mutable private folder names are retrieved from the folder resource rather than duplicated throughout post responses.

### Q15: When does a folder's `updatedAt` change?

Answer: Only when the folder itself is renamed.

Decision / implication: Adding, moving, unfiling, or removing contained saves does not mutate folder metadata or create unnecessary folder-row contention.

### Q16: Which success statuses do saved-post mutations return?

Answer: Follow the existing interaction contract: creating a new save returns `201`; an idempotent repeat or folder change returns `200`; save and folder deletes return `204`.

Decision / implication: The API distinguishes creation from an existing-resource result while retaining retry-safe mutation semantics.

### Q17: How are saved replies cleaned up when a root or intermediate ancestor is permanently deleted?

Answer: The post-deletion indexer transaction must determine the exact target and every still-indexed descendant reply whose required parent chain contains the deleted URI, then remove the affected saves in that same transaction. The exact-target foreign key is not sufficient for ancestor deletion, and descendant public post rows are not deleted merely because their context was deleted.

Decision / implication: Root deletion can use the indexed root relationship, while intermediate-ancestor deletion may require recursive parent traversal. The affected save set is calculated before or within the transaction that removes the indexed post so no orphaned saved reply is committed between those steps.

### Q18: Where is viewer saved state hydrated?

Answer: Extend the existing shared `EngagementSummaries` batch-hydration path with `viewerHasSaved` and `viewerSavedFolderId` rather than adding per-surface or per-item save lookups.

Decision / implication: Post, timeline, profile-content, project, search, comment/reply, notification, quote, and saved-list consumers use one central set-based seam, preserving consistent viewer fields and bounded query counts.

### Q19: Are saved-post cursors confidential or encrypted?

Answer: No. They follow the existing AppView cursor contract: base64url-encoded JSON that clients must treat as opaque, not an encrypted or confidential token.

Decision / implication: A cursor may encode the requesting owner's scope and keyset values, including a folder ID or post URI, when needed for deterministic pagination. It must not encode an owner DID, and the server must reject malformed cursors or reuse with an incompatible scope or sort direction.

### Q20: Is a UUID-shaped folder ID part of the API contract?

Answer: No. Folder IDs are opaque JSON strings. The storage implementation may choose UUIDs, but clients and contract tests must not parse or validate a UUID wire format.

Decision / implication: Folder identity remains stable and independent of duplicate display names without unnecessarily freezing its representation.

## 4. Candidate Approaches

### Option A: Normalized private AppView folders and saves

Summary: Add owner-scoped folder and saved-post tables. Each save is unique on owner DID plus post URI and has one nullable folder reference. AppView APIs mutate and list this private state while hydrating current canonical post responses.

Pros:

- Directly matches the confirmed single-folder model.
- Enforces owner isolation and referential behavior in Postgres.
- Supports efficient folder, unfiled, and all-saves pagination.
- Keeps private organization separate from public post records.
- Allows post response viewer state to be batch-hydrated without PDS access.

Cons:

- Adds a migration and a new AppView API/storage area.
- Requires explicit handling when folders or indexed posts are deleted.

Risks:

- Incorrect owner predicates could expose private saved state.
- Ad hoc per-post lookups could create list-response N+1 queries.

### Option B: Store folder names directly on saved rows

Summary: Store a nullable folder-name string on each saved row without a folder resource table.

Pros:

- Fewer tables and simpler initial writes.

Cons:

- Folder rename becomes a bulk update.
- Folder identity, uniqueness, timestamps, and future metadata become awkward.
- Concurrent rename/move operations are harder to make consistent.

Risks:

- String drift can create duplicate or partially renamed folders.

### Option C: Public PDS records for saves and folders

Summary: Define Craftsky lexicons and store saves/folders in user repositories.

Pros:

- Data would travel with the user's public repository.

Cons:

- Saved-post choices and organization would be publicly readable and broadcast.
- Requires load-bearing lexicon decisions and PDS writes.
- Conflicts with the explicit privacy requirement and current atproto public-repository model.

Risks:

- Irreversible privacy harm and public disclosure of user behavior.

## 5. Recommended Direction

Recommended approach: Option A — normalized private AppView folders and saves.

Why: It satisfies the confirmed privacy and one-folder constraints, follows the existing private-mute ownership precedent, gives Flutter a conventional owner-scoped REST contract for a later slice, and keeps public post identity separate from private organization.

## 6. Problem / Opportunity

Craftsky users need a private way to keep useful posts, projects, comments, and replies for later without publicly liking or reposting them. A flat list becomes difficult to use as it grows, so optional one-level folders provide lightweight organization while an unfiled state preserves a low-friction save action. This AppView slice establishes the private persistence, authorization, and API contracts before Flutter UI work begins.

## 7. Goals

- G-001: Let a signed-in Craftsky user privately save or unsave any currently eligible indexed Craftsky post, comment, or reply.
- G-002: Let the owner organize each save into at most one optional one-level folder.
- G-003: Let the owner create, rename, list, and delete saved-post folders without losing saves when a folder is deleted.
- G-004: Let the owner list all, one folder's, or unfiled saved posts in newest-saved-first or oldest-saved-first order.
- G-005: Expose owner-relative saved state on canonical AppView post responses for later Flutter consumption.
- G-006: Preserve strict account isolation, existing content-policy boundaries, and retry-safe mutations.

## 8. Non-Goals

- NG-001: Do not implement Flutter UI, local caching, providers, repositories, optimistic updates, or navigation in this slice.
- NG-002: Do not add or change atproto lexicons, create PDS records, or synchronize Bluesky bookmarks.
- NG-003: Do not support nested folders, subfolders, multi-folder membership, tags, smart folders, folder sharing, collaboration, or public collections.
- NG-004: Do not support folder ordering/reordering, manual saved-post ordering, pinning, notes, annotations, or full-text search within saves.
- NG-005: Do not archive or snapshot another user's post body or media in private saved-post storage.
- NG-006: Do not make saved-post data portable between independent AppViews in this slice.
- NG-007: Do not add saved-post notifications, recommendations, popularity ranking, or background export jobs.
- NG-008: Do not change existing post authoring, indexing, moderation, membership, mute, block, like, repost, reply, or deletion policy except to project saved state safely through it.
- NG-009: Do not include saved-post counts on folder resources in this slice.
- NG-010: Do not impose product-level per-account folder or saved-post quotas in this slice; existing authentication, rate limits, body limits, and bounded pagination still apply.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Saving user | Signed-in Craftsky member saving content for personal reference. | Private, reliable save/unsave behavior and optional folder organization. |
| Folder owner | Saving user managing their own one-level folders. | Create, rename, list, delete, and move saves without cross-account effects. |
| Post author | Member whose public post, project, comment, or reply is saved by someone else. | No notification or disclosure that the save occurred. |
| Flutter client | Future consumer of the AppView saved-post API. | Stable authenticated camelCase contracts, pagination, viewer state, and predictable errors. |
| AppView | Trusted privacy, persistence, and content-policy boundary. | Enforce owner scoping, hydrate eligible current posts, and avoid PDS writes. |

## 10. Current Behavior

Craftsky indexes top-level posts, project posts, comments, and replies in `craftsky_posts` and serves them through canonical `PostResponse` objects. Users can like or repost publicly through their PDS, but there is no private save action or folder organization. Post responses do not indicate whether the authenticated viewer has saved the post, and no AppView route lists private saved content.

## 11. Desired Behavior

An authenticated user can save any currently eligible row in `craftsky_posts`, optionally assigning it to one folder they own. Saving is private, silent, AppView-only, and idempotent. On a new save, omitted or null `folderId` creates an unfiled save. On an existing save, omitted `folderId` preserves the current assignment, a folder ID replaces it, and explicit null moves it to unfiled; none of these existing-save operations changes `savedAt`. Unsaving and later saving again creates a new `savedAt`.

The owner can create, rename, list, and delete flat folders. Folders are identified by opaque IDs, and duplicate display names are allowed. Names are trimmed, bounded, and restricted to printable non-path-like characters. Deleting a folder is non-destructive: contained saves become unfiled atomically. Saved-post lists can show all saves, a selected owned folder, or unfiled saves and can sort by `savedAt` newest-first or oldest-first with opaque cursor pagination.

Each saved-list item contains private save metadata plus the current canonical post view. A saved comment or reply remains that exact item and carries its existing root/parent references so a future client can open the root thread with the saved URI focused. Normal post-shaped responses expose only the authenticated viewer's own `viewerHasSaved` and nullable `viewerSavedFolderId`, never an embedded folder name. Existing membership, moderation, mute, and block rules still control whether and how referenced content can be returned. Temporarily unavailable items and thread contexts retain hidden saves; permanently deleted targets or required thread ancestors remove saves that can no longer be opened. Private save metadata never reaches the PDS, firehose, another user, logs, or telemetry dimensions.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | Craftsky shall let an authenticated member privately save and unsave any currently eligible indexed ordinary post, project post, quote post, comment, or reply. | Users need a non-public alternative to likes/reposts for keeping useful content. | Prompt | AC-001, AC-002, AC-006 |
| BR-002 | Business | Must | Craftsky shall let the owner optionally organize each saved post into one and only one flat folder. | Lightweight organization is the differentiating product requirement. | Prompt; User answer | AC-003, AC-004, AC-005 |
| BR-003 | Business | Must | Craftsky shall let the owner retrieve saved posts by all, folder, or unfiled scope in either chronological direction based on save time. | Saved content must remain usable as the collection grows. | Prompt; User answer | AC-011, AC-012, AC-013 |
| BR-004 | Business | Must | A user's saved posts, folders, folder assignments, and saved timestamps shall remain private to that authenticated user and the trusted AppView. | Saving behavior is private-by-intent data. | Prompt; Architecture | AC-021, AC-025, AC-028 |
| FR-001 | Functional | Must | The AppView shall save a target only when `{did}/{rkey}` resolves to a currently eligible indexed `craftsky_posts` row, using the canonical post URI as logical identity and a server-assigned `savedAt`. | Every supported post/comment/reply already has one stable AppView identity; the client must not choose its ordering timestamp. | Codebase; User answer | AC-001, AC-002 |
| FR-002 | Functional | Must | The AppView shall maintain at most one save per `(ownerDid, postUri)`. For a new save, an absent body, omitted `folderId`, or null `folderId` means unfiled. For an existing save, an absent body or omitted `folderId` preserves the current assignment, an owned folder ID atomically replaces it, and explicit null moves it to unfiled; all existing-save results preserve `savedAt`. | Save requests must be retry-safe, distinguish omission from explicit unfiling, and conform to the one-folder decision. | User answer; API precedent | AC-003, AC-004, AC-023, AC-031 |
| FR-003 | Functional | Must | The AppView shall unsave only the authenticated owner's matching post idempotently and return `204` even when no save or currently indexed target exists. A later new save of that post, if it is eligible again, shall receive a new `savedAt`. | DELETE retries must remain safe across target deletion, and chronology must reflect a later new save action. | User answer; API conventions | AC-006 |
| FR-004 | Functional | Must | The AppView shall create an owner-scoped folder with a stable ID represented on the JSON wire as an opaque string, server-assigned `createdAt` and `updatedAt`, and a display name trimmed to 1–100 Unicode characters that rejects `/`, `\`, and control characters. Duplicate names, including exact and case-only duplicates, shall be allowed for one owner. Storage may use UUIDs, but clients and contract tests shall not require or validate UUID-shaped IDs. | Folders need stable identity and predictable validation, while the confirmed resource model makes name uniqueness unnecessary and should not freeze an implementation-specific wire representation. | User answer; Recommended direction; Document review | AC-007, AC-008 |
| FR-005 | Functional | Must | The AppView shall rename only an authenticated owner's folder, preserve its ID and `createdAt`, advance `updatedAt`, apply the same name validation rules, and leave contained saves and their `savedAt` values unchanged. Adding, moving, unfiling, or removing saves shall not change the folder's `updatedAt`. | Folder metadata time must describe folder metadata changes rather than contained-save activity. | User answer; Recommended direction | AC-009, AC-033 |
| FR-006 | Functional | Must | Deleting an authenticated owner's folder shall atomically set every contained save's folder assignment to unfiled without changing or deleting those saves. Deleting a missing or other-owner folder ID shall be an idempotent `204` no-op. | Folder cleanup should not destroy saved content, and delete retries must disclose no resource-existence signal. | User answer; API conventions | AC-010, AC-023, AC-032 |
| FR-007 | Functional | Must | The AppView shall list only the authenticated owner's folders using opaque-cursor pagination, default limit 50 and maximum 100, ordered case-insensitively by name ascending with folder ID ascending as the stable tie-breaker. | Folder management must be deterministic, bounded, and owner-private. | API conventions; User answer | AC-011, AC-028 |
| FR-008 | Functional | Must | The AppView shall list only the authenticated owner's saved posts across all saves by default, within one owned `folderId` when provided, or within unfiled saves when `unfiled=true`; `folderId` and `unfiled=true` together shall be rejected. A missing or other-owner list `folderId` shall return the same `404 saved_post_folder_not_found`. | The API must represent all three confirmed organizational scopes without ambiguity or cross-account existence disclosure. | Prompt; User answer | AC-012, AC-020, AC-032 |
| FR-009 | Functional | Must | Saved-post lists shall accept `sort=newest` or `sort=oldest`, default to `newest`, order by `savedAt` in that direction, and use post URI in the same direction as a deterministic tie-breaker. | Pagination needs exact chronological semantics and stable ordering. | User answer; API conventions | AC-013, AC-027 |
| FR-010 | Functional | Must | Each saved-post list item shall return `{post, savedAt, folderId}`, where `post` uses the current canonical `PostResponse`, `folderId` is nullable, and save metadata belongs only to the authenticated owner. For a comment or reply, `post` shall retain its canonical root and parent references so the exact saved URI can be focused within its root thread without storing duplicate context. | Future Flutter work should reuse existing post rendering and deep-link behavior while receiving organization metadata. | Codebase; User answer | AC-002, AC-014, AC-028 |
| FR-011 | Functional | Must | Every authenticated canonical post-shaped response shall expose additive viewer-relative `viewerHasSaved` and nullable `viewerSavedFolderId` fields derived only from the requesting DID. The AppView shall extend the existing shared `EngagementSummaries` batch-hydration path so every canonical consumer receives this state through one set-based seam, without per-surface or per-item save lookups. It shall not embed a saved folder name. | The client must know whether and where the active account saved a displayed post without query drift, N+1 access, or duplicated mutable folder metadata. | Existing viewer-state pattern; User answer; Document review | AC-015, AC-026, AC-028 |
| FR-012 | Functional | Must | Saved-post hydration shall apply current membership, moderation, mute, block, and availability policy and shall never reveal content the requester is not otherwise permitted to receive. The owner-curated saved list shall follow current explicit/direct-access semantics: otherwise-eligible content from a muted author may remain available with viewer-relative mute state, while block, hide, takedown, non-member, and unavailable decisions remain stricter. | Private organization must not become a policy bypass, while a personal collection is not unsolicited discovery. | Architecture; Existing policy; User answer | AC-016, AC-017 |
| FR-013 | Functional | Must | A save shall reference the logical post URI rather than store a private snapshot. Updates at the same URI shall hydrate the current indexed version. Deletion of the saved post's indexed PDS record shall delete matching saves. Before or within that same post-indexer deletion transaction, permanent deletion of a saved comment/reply's root or required ancestor shall identify all still-indexed descendant replies whose parent chain contains the deleted URI and delete their now-unnavigable saves, without deleting those descendant public post rows. Temporary content or thread-context ineligibility shall hide without deleting the save so it can reappear if eligible again. | The exact-target foreign key cannot clean up saves of still-indexed descendants after ancestor deletion; explicit atomic cleanup avoids dead thread links without inventing public-record deletion or retaining private content copies. | Architecture; User answer; Document review | AC-016, AC-017, AC-018 |
| FR-014 | Functional | Must | Permanently removing the owning DID's Craftsky membership shall delete that DID's folders and saves. Sign-out, logout-all, device removal, token expiry, app reinstall, and account switching shall retain server-side saved state and shall not affect another account. | Private state belongs to membership, not a session or device, and multi-account isolation is mandatory. | Existing mute lifecycle; Multi-account architecture | AC-019, AC-028 |
| FR-015 | Functional | Must | The AppView shall expose the saved-post REST surface described in Section 16, using existing `/v1/` authentication/device middleware, camelCase JSON, body limits, read/write rate classes, and standard error envelopes. | New routes must conform to the governing API contract. | AGENTS.md; API architecture | AC-020, AC-021 |
| FR-016 | Functional | Must | No saved-post or folder operation shall write to a PDS, emit an atproto record, require a lexicon change, or depend on Tap convergence before becoming visible to the owner. | Saves are private AppView state and should be immediately consistent after successful mutation. | Prompt; Architecture | AC-021, AC-022 |
| FR-017 | Functional | Must | A folder assignment, rename, or folder-scoped read shall succeed only when the folder exists and is owned by the authenticated DID; a missing or other-owner folder shall return the same `404 saved_post_folder_not_found` response. Folder deletion is the sole exception and shall return `204` for either case. | Folder identifiers must not permit cross-account mutation or enumeration while deletes remain idempotent. | Privacy boundary; User answer | AC-005, AC-009, AC-012, AC-025, AC-032 |
| FR-018 | Functional | Must | Folder creation, rename, deletion, save creation/move, and unsave operations shall be transactionally atomic under concurrency and preserve the uniqueness and one-folder invariants. | Concurrent retries must not create duplicates, partial moves, or destructive folder deletion races. | Recommended direction | AC-023 |
| FR-019 | Functional | Must | The AppView shall provide a reversible migration with owner, URI, folder, and ordering constraints/indexes sufficient for owner-scoped mutations, duplicate folder names, deterministic folder ordering, and both saved-list sort directions. | Persistence and rollback must enforce the product contract and avoid unindexed private-list scans. | Codebase conventions; User answer | AC-024, AC-026 |
| FR-020 | Functional | Must | Saving or moving a post shall return the current saved state synchronously after commit: `201` when a new save is created and `200` for an idempotent existing save or folder-assignment result. Deleting a save or folder shall return `204` only after its transaction commits. | Private AppView writes do not need firehose eventual consistency and should follow existing interaction status conventions. | Architecture; Codebase | AC-022, AC-023, AC-034 |
| NFR-001 | Non-functional | Must | Saved-post data, folder names, folder IDs, and owner/target pairs shall not appear in another user's response or in log messages, trace attributes, metric labels, or error text; telemetry may include only bounded operation, result, stage, and error-class values. | Saved behavior and folder naming are sensitive private data. | Prompt; Privacy precedent | AC-025, AC-029 |
| NFR-002 | Non-functional | Must | Saved-post list and viewer-state queries shall avoid per-item database lookups and use bounded, indexed, set-based access paths. Canonical post surfaces shall obtain saved viewer state by extending the existing shared `EngagementSummaries` batch query rather than adding independent hydration paths. | Saved state will be hydrated across feeds and lists; one central seam prevents N+1 access and response drift. | Codebase; Discovery; Document review | AC-026 |
| NFR-003 | Non-functional | Must | Saved-post pagination shall be deterministic and cursor-based for each fixed scope and sort direction, using the existing base64url-JSON envelope cursor contract. Cursors are opaque to clients but are not encrypted or confidential; their payload may contain owner-visible scope and keyset values such as folder ID or post URI, but shall not contain owner DID. Malformed cursors or cursors reused with incompatible scope/sort parameters shall return `400 invalid_cursor`. | Clients must not skip, duplicate, or reinterpret pages silently, while the feature must not invent a stronger cursor-confidentiality contract than the rest of the API. | API conventions; Document review | AC-027 |
| NFR-004 | Non-functional | Must | Every store query and mutation shall derive the owner from authenticated request context and include the owner predicate or owner-enforcing join in the database operation. | Client-provided ownership would create a critical cross-account privacy flaw. | Security boundary | AC-028 |
| NFR-005 | Non-functional | Should | Saved-post operations shall emit bounded success/failure metrics and sanitized error logs consistent with existing AppView observability, without introducing a new alert solely for this pre-production feature. | Operators need basic diagnostics without leaking private state or adding premature alert noise. | Existing observability; Pre-production status | AC-029 |
| NFR-006 | Non-functional | Must | The migration shall be testable through an up/down/up cycle, and the feature shall be covered by store, handler, route-contract, privacy-isolation, lifecycle, policy, pagination, and regression tests. | Private persistent state requires strong regression protection. | Workflow quality standard | AC-024, AC-030 |
| RULE-001 | Business rule | Must | One owner may have only one save for a given post URI, and that save may reference at most one folder. | This is the confirmed organization model. | User answer | AC-003, AC-004 |
| RULE-002 | Business rule | Must | A save does not require a folder; null folder assignment means unfiled. | Saving must remain low-friction. | Prompt | AC-003, AC-012 |
| RULE-003 | Business rule | Must | `savedAt` is assigned when an unsaved post becomes saved; idempotent saves, folder moves, and folder renames/deletion do not change it, but unsave followed by save does. | Sorting must represent save chronology rather than organization edits. | User answer | AC-004, AC-006, AC-010, AC-013 |
| RULE-004 | Business rule | Must | Folders are flat owner-private resources and cannot have parents, children, collaborators, visibility settings, or multi-folder links. | The slice is intentionally limited to one level. | Prompt; User answer | AC-007 |
| RULE-005 | Business rule | Must | Saving is silent: it creates no notification, public interaction count, author-visible state, or PDS/firehose event. | A save must remain private and must not behave like a like or repost. | Prompt | AC-021 |
| RULE-006 | Business rule | Must | Folder deletion is non-destructive to saves. Permanent deletion of a saved post or required reply-thread ancestor is destructive to affected saves; temporary visibility, context, or membership changes are not. | Each deletion/eligibility event needs deterministic persistence semantics and saved replies must remain meaningfully navigable. | User answer | AC-010, AC-016, AC-017, AC-018 |
| RULE-007 | Business rule | Must | Folder display names preserve accepted casing and may duplicate any other folder name owned by the same DID. After trimming, names must contain 1–100 Unicode characters and no `/`, `\`, or control character. | Opaque IDs provide identity while validation preserves a safe flat-folder presentation. | User answer | AC-007, AC-008, AC-009 |
| RULE-008 | Business rule | Must | Saving a comment or reply saves and lists that exact post URI, not its root or parent. Its canonical root/parent references provide navigation context; the AppView stores no duplicate parent-content snapshot. | Every post depth is an independent save target while existing thread navigation requires root-plus-focus context. | Prompt; Codebase; User answer | AC-002, AC-014, AC-018 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-001 | Given an eligible indexed post and an authenticated member, when the member saves it without a folder, then exactly one private save is committed with a server-assigned `savedAt` and returned as unfiled. |
| AC-002 | BR-001, FR-001, FR-010, RULE-008 | Given an eligible top-level ordinary post, project post, quote post, direct comment, or nested reply, when each exact `{did}/{rkey}` is saved, then the AppView saves that exact canonical post URI without substituting its root or parent, and a saved comment/reply response retains its canonical root and parent references. |
| AC-003 | BR-002, FR-002, RULE-001, RULE-002 | Given an unfiled save, when the same save request is repeated, then one row remains, `folderId` remains null, and `savedAt` is unchanged. |
| AC-004 | BR-002, FR-002, RULE-001, RULE-003 | Given a saved post and two owned folders, when it is assigned first to one folder and then the other, then only the second assignment remains, no duplicate save is created, and `savedAt` is unchanged. |
| AC-005 | BR-002, FR-017 | Given a missing folder ID or a folder owned by another DID, when the user tries to save or move a post into it, then the AppView returns `404 saved_post_folder_not_found`, makes no change, and does not distinguish the cases. |
| AC-006 | BR-001, FR-003, RULE-003 | Given a saved post, when its owner unsaves it twice, or retries after the indexed target has been deleted, then every delete returns `204` and no save remains; when the post is eligible and saved later, its new `savedAt` is later than the deleted save's timestamp. |
| AC-007 | FR-004, RULE-004, RULE-007 | Given valid folder input, when the owner creates a folder, then the response contains a stable opaque-string ID, trimmed display name, `createdAt`, and `updatedAt`, and contains no nesting, sharing, or saved-post-count fields; clients can round-trip the ID without assuming a UUID format. |
| AC-008 | FR-004, RULE-007 | Given an owner already has folders named `Ideas` and `IDEAS`, when that owner creates another ` Ideas ` folder, then creation succeeds with display name `Ideas` and a distinct opaque ID; blank names, names longer than 100 Unicode characters, names containing `/` or `\`, and names containing control characters fail validation. |
| AC-009 | FR-005, FR-017, RULE-007 | Given a folder containing saves and another same-named folder, when its owner renames it to that duplicate valid name, then its ID, `createdAt`, contents, and save timestamps remain, `updatedAt` advances, and a non-owner receives the same not-found response as a missing folder. |
| AC-010 | FR-006, RULE-003, RULE-006 | Given a folder containing saved posts, when its owner deletes the folder, then deletion commits atomically, the folder is gone, every contained save is unfiled with unchanged `savedAt`, and repeating deletion succeeds. |
| AC-011 | BR-003, FR-007 | Given multiple folders, including duplicate and case-variant names, owned by two users, when one user lists folders with pagination, then only that user's folders appear in deterministic case-insensitive name order with folder ID as the tie-breaker and an optional opaque next cursor. |
| AC-012 | BR-003, FR-008, FR-017, RULE-002 | Given foldered and unfiled saves, when the owner requests all saves, one owned folder, or `unfiled=true`, then each result contains exactly the eligible saves in that scope, simultaneous `folderId` plus `unfiled=true` is rejected, and a missing or other-owner folder scope returns the same `404 saved_post_folder_not_found`. |
| AC-013 | BR-003, FR-009, RULE-003 | Given saves with distinct `savedAt` values, when listed with default/`newest` and then `oldest`, then the orders are exact reverses by save time with the documented URI tie-breaker, independent of post creation time and folder edits. |
| AC-014 | FR-010, RULE-008 | Given an eligible saved post, comment, or reply, when it appears in a saved list, then the item contains the exact canonical current `PostResponse`, the save's `savedAt`, and its nullable `folderId`; comments/replies retain root/parent references, and no duplicate parent snapshot or private metadata from another owner is returned. |
| AC-015 | FR-011 | Given Alice saved a post in a folder and Bob did not, when each requests the same post or a list containing it, then Alice sees `viewerHasSaved: true` and her folder ID without an embedded folder name while Bob sees `viewerHasSaved: false` and null, with saved state supplied through the shared `EngagementSummaries` batch path and no per-item query behavior. |
| AC-016 | FR-012, FR-013, RULE-006 | Given a saved post's author or required thread context becomes muted, blocked, moderated, temporarily unavailable, or not a current member, when the saved list is read, then an otherwise-eligible muted-author item follows current explicit/direct-access shaping, stricter block/hide/takedown/non-member/unavailable decisions do not expose forbidden content, and non-destructive policy/context cases retain the private save. |
| AC-017 | FR-012, FR-013, RULE-006 | Given a temporarily ineligible saved post later becomes eligible again at the same URI, when the owner lists saves, then it reappears with its original `savedAt` and folder assignment. |
| AC-018 | FR-013, RULE-006, RULE-008 | Given a saved post record is deleted from its PDS, or a saved comment/reply's root or required ancestor is permanently deleted, when AppView deletion converges, then the exact target's saves or affected descendant-reply saves are removed in the same AppView indexer deletion transaction, descendant public post rows remain indexed unless separately deleted, and no post or thread snapshot is stored. |
| AC-019 | FR-014 | Given a user has folders and saves, when a session expires, a device is removed, the user signs out, or the client switches accounts, then the server state remains; when the owning Craftsky membership is permanently removed, that owner's folders and saves are deleted without affecting another owner. |
| AC-020 | FR-008, FR-015 | Given malformed list filters, invalid sort/limit values, an oversized JSON body, or a body on a no-body route, when the request is made, then the governing route middleware and standard validation envelope reject it with a stable code. |
| AC-021 | BR-004, FR-015, FR-016, RULE-005 | Given any successful save/folder mutation, when storage and external activity are inspected, then only AppView private tables changed: no PDS call, lexicon record, Tap event, notification, author-visible state, or public interaction count was produced. |
| AC-022 | FR-016, FR-020 | Given a successful private mutation, when the owner immediately reads the corresponding post, folder, or saved list, then the committed state is visible without waiting for Tap or another background process. |
| AC-023 | FR-002, FR-006, FR-018, FR-020 | Given concurrent duplicate saves, moves, folder deletion, and retries, when transactions complete, then database constraints and atomic operations leave one save per owner/URI, at most one valid folder assignment, and no partially deleted folder state. |
| AC-024 | FR-019, NFR-006 | Given a pre-feature database, when the saved-post migration runs up/down/up, then both tables, owner/post and owner/folder constraints, foreign-key actions, and ordering indexes are created, removed, and recreated cleanly without changing unrelated schema, and multiple same-named folders for one owner remain valid. |
| AC-025 | BR-004, FR-017, NFR-001 | Given operations involving private folder/save data succeed or fail, when another user observes API responses, logs, traces, metrics, and errors, then no folder name/ID, saved URI, owner-target pair, or existence signal is exposed. This does not prohibit an authenticated owner's own opaque cursor from encoding that owner's scope/keyset values under NFR-003. |
| AC-026 | FR-011, FR-019, NFR-002 | Given a full page of post or saved-list results, when viewer save state is hydrated, then the shared `EngagementSummaries` seam performs a bounded set-based indexed query rather than one query per item or an independent query path per canonical surface. |
| AC-027 | FR-009, NFR-003 | Given a saved-list cursor created for one scope and sort, when it is reused with a different folder/unfiled scope or sort direction, or is malformed, then the AppView returns `400 invalid_cursor`; unchanged parameters continue deterministically. Decoding may reveal the owner-visible scope/keyset fields required by the existing cursor helper, but no owner DID is encoded and no encryption guarantee is asserted. |
| AC-028 | BR-004, FR-007, FR-010, FR-011, FR-014, NFR-004 | Given Alice and Bob are both signed in on the same device or in separate sessions, when each performs any saved-post/folder operation, then every read and write is scoped from the authenticated DID and neither account can read, mutate, or clear the other's state. |
| AC-029 | NFR-001, NFR-005 | Given saved-post operations succeed and fail, when telemetry is inspected, then only bounded operation/result/stage/error-class values are recorded and private identifiers or names are absent. |
| AC-030 | NFR-006 | Given the feature test suite runs, then it covers storage, handlers, routes, privacy isolation, membership/session lifecycle, reply-context lifecycle, current content policy, both sort directions, all/folder/unfiled pagination, duplicate folder names, request omission versus explicit null, concurrency/idempotency, migration reversal, and existing post-response regressions. |
| AC-031 | FR-002, RULE-002, RULE-003 | Given a new save request has no body, omits `folderId`, or sends `"folderId": null`, then it creates an unfiled save; given an existing foldered save, no body or omitted `folderId` preserves that folder, while explicit null moves it to unfiled, and all existing-save behaviors preserve the original `savedAt`. |
| AC-032 | FR-006, FR-008, FR-017 | Given a missing folder ID or one owned by another DID, when it is used for assignment, rename, or folder-scoped listing, then the API returns the same `404 saved_post_folder_not_found`; when it is deleted, the API returns `204` and changes no data. |
| AC-033 | FR-005, RULE-003 | Given a folder's `updatedAt`, when saves are added, moved into it, moved out, unfiled, or removed, then `updatedAt` remains unchanged; when the folder is renamed, it advances. |
| AC-034 | FR-020 | Given save mutations, when a new save is committed, then the API returns `201` plus current saved state; when an existing save is repeated, moved, or unfiled, it returns `200` plus current saved state; committed save and folder deletes return `204`. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | Save the same post repeatedly | Return the one current save; do not change `savedAt` unless it was unsaved first. | FR-002, RULE-003 |
| EC-002 | Save a nested reply | Save that reply URI, return its canonical root/parent references, and use root-plus-focus navigation rather than substituting the root as the saved item. | FR-001, FR-010, RULE-008 |
| EC-003 | Move a save to its current folder | Succeed idempotently with no timestamp or row change. | FR-002 |
| EC-004 | Move a save to unfiled | Set `folderId` to null and preserve `savedAt`. | FR-002, RULE-002 |
| EC-005 | Delete a non-empty folder | Atomically retain its saves as unfiled. | FR-006, RULE-006 |
| EC-006 | Duplicate folder name with the same or different casing | Allow it and distinguish each folder by opaque ID; trim surrounding whitespace on input. | FR-004, RULE-007 |
| EC-007 | Folder ID belongs to another account | Return the same not-found error as an unknown ID for assignment, rename, and scoped reads; return `204` for delete. | FR-006, FR-017, NFR-004 |
| EC-008 | Saved post is updated at the same URI | Return the current indexed post version while retaining original save metadata. | FR-013 |
| EC-009 | Saved post or required thread context is temporarily hidden | Apply existing policy and retain the save for possible restoration. Otherwise-eligible muted-author content remains directly accessible. | FR-012, FR-013 |
| EC-010 | Saved post or required reply ancestor is permanently deleted | Remove matching target saves or affected orphaned-reply saves; do not retain a private content/thread snapshot. | FR-013, RULE-006 |
| EC-011 | Owner signs out or changes device/account | Retain the owner's AppView saved state and isolate it from the active alternate account. | FR-014 |
| EC-012 | Owner membership is permanently removed | Delete only that owner's folders/saves. | FR-014 |
| EC-013 | New saves arrive during pagination | Keyset cursor semantics preserve the fixed scope/order without offset-based page drift. | FR-009, NFR-003 |
| EC-014 | Cursor used with another folder or sort | Reject as `invalid_cursor`. | NFR-003 |
| EC-015 | Folder delete races with move into folder | Transaction/FK behavior yields either a committed move before non-destructive unfiling or a not-found folder result; never a dangling reference. | FR-018 |
| EC-016 | Repeated save omits `folderId` | Preserve an existing assignment; require explicit null to move to unfiled. | FR-002, RULE-002 |
| EC-017 | Folder contents change | Preserve the folder's `updatedAt`; only rename advances it. | FR-005 |
| EC-018 | Duplicate folder names cross a page boundary | Order case-insensitively by display name and then folder ID so every distinct folder paginates deterministically. | FR-007, NFR-003 |

## 15. Data / Persistence Impact

- New private persistence:
  - An owner-scoped saved-post-folders table with opaque ID, owner DID, display name, `createdAt`, and `updatedAt` equivalents. Display names are not unique.
  - An owner-scoped saved-posts table keyed uniquely by owner DID plus canonical post URI, with nullable folder ID and `savedAt`.
- Referential behavior:
  - Owner DID references current Craftsky membership with owner deletion cascading to owned folders/saves.
  - Folder deletion uses `ON DELETE SET NULL` or transactionally equivalent behavior.
  - Indexed post deletion cascades to saves of that exact URI; ordinary post updates at the same URI do not.
  - The post indexer's existing deletion transaction explicitly determines affected still-indexed descendants before deleting the event URI. Root deletion can select by the indexed root relationship; intermediate-ancestor deletion traverses the indexed parent chain. That same transaction removes saves for every descendant reply that can no longer be resolved by the root-plus-focus thread flow, while retaining descendant public post rows. The exact-target foreign key alone is not sufficient for this cleanup.
  - Temporary policy or membership ineligibility does not perform destructive save cleanup.
- Indexes/constraints:
  - Unique owner/post URI.
  - Owner plus case-insensitive folder display name plus folder ID ordering index; no folder-name uniqueness constraint.
  - Owner plus `savedAt`/URI indexes supporting both scan directions.
  - Owner plus folder plus `savedAt`/URI index and an unfiled partial/equivalent index.
  - Folder owner-integrity constraint so a save cannot reference another owner's folder.
- Migration required: Yes; reversible up/down files with migration-number verification at implementation time.
- Backwards compatibility: Additive AppView tables, endpoints, and JSON fields only. No public record or lexicon migration.

## 16. UI / API / CLI Impact

- UI: None in this AppView-only slice.
- API:
  - `POST /v1/posts/{did}/{rkey}/saves` — idempotently create or update a save. On create, an absent body, omitted `folderId`, or null `folderId` means unfiled. On an existing save, an absent body or omitted `folderId` preserves its assignment, a folder ID moves it, and explicit null unfiles it. Return current saved state with `201` for create or `200` for an existing-save result.
  - `DELETE /v1/posts/{did}/{rkey}/saves` — idempotently unsave for the authenticated owner without requiring the target to remain indexed; return 204 when the save or target is already absent.
  - `GET /v1/saved-posts?folderId=&unfiled=&sort=&limit=&cursor=` — list owner saves; absent folder scope means all, default sort is newest, and a missing/other-owner folder scope returns `404 saved_post_folder_not_found`.
  - `GET /v1/saved-post-folders?limit=&cursor=` — list owner folders.
  - `POST /v1/saved-post-folders` — create `{name}`; duplicate names are valid; return 201 plus the folder without a saved-post count.
  - `PATCH /v1/saved-post-folders/{folderId}` — rename using `{name}`; duplicate names are valid; return the updated folder.
  - `DELETE /v1/saved-post-folders/{folderId}` — idempotently delete and unfile contained saves; return 204 for owned, missing, or other-owner IDs.
  - Folder IDs are opaque JSON strings. A UUID is an allowed storage choice, not a public wire-format guarantee.
  - Canonical post responses add `viewerHasSaved` and nullable `viewerSavedFolderId`, but no embedded folder name. The fields extend the shared `EngagementSummaries` batch-hydration path used by every canonical post consumer. Saved replies retain canonical root/parent references for root-plus-focus navigation.
  - List cursors use the existing base64url-JSON envelope helper. Clients treat them as opaque, but the server does not promise encryption or confidentiality; owner-visible scope/keyset values may be encoded, while owner DID is derived from authentication rather than cursor data.
  - Expected feature errors include `post_not_found`, `saved_post_folder_not_found`, `validation_failed`, and `invalid_cursor`, all in the standard envelope.
- CLI: None.
- Background jobs: None.

## 17. Security / Privacy / Permissions

- Authentication: All saved-post and folder routes require the existing Craftsky session and device-ID middleware.
- Authorization: The authenticated DID is the sole owner source. Folder IDs and target post identifiers never grant access by themselves.
- Sensitive data: Saved URIs, folder IDs/names, assignments, and timestamps are private behavioral data. They are stored only in AppView Postgres and returned only to the owner.
- Cross-account behavior: Multi-account sessions must never share saved state, cursors, responses, mutations, or cleanup.
- Author privacy: Post authors receive no notification, count, viewer state, or existence signal when another user saves their content.
- Content safety: Saved lists and viewer-state hydration do not override membership, moderation, mute, block, or takedown policy.
- Abuse cases: Existing authenticated read/write rate limits and body limits apply. Folder IDs are opaque and non-owner/missing IDs are indistinguishable.

## 18. Observability

- Events: No product analytics event is required in this AppView-only slice.
- Logs: Sanitized failures may include request ID, bounded operation/stage/error class, and wrapped internal error, but not folder names/IDs, post URI, or owner/target pairs.
- Metrics: Reuse bounded AppView operation duration/outcome patterns for save, unsave, folder create/rename/delete, and list operations if instrumentation is added; identifiers and names are forbidden dimensions.
- Alerts: None added for this pre-production slice; existing AppView health/error monitoring remains applicable.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Missing owner predicates expose or mutate another user's saved state. | Critical privacy breach. | Derive owner from auth context, enforce owner-aware constraints/joins, use opaque IDs, and add cross-account tests for every operation. |
| RISK-002 | Viewer saved-state hydration adds N+1 queries across feeds/search/profile lists. | Latency and database-load growth. | Require set-based batch hydration and supporting indexes; test query shape. |
| RISK-003 | Folder deletion accidentally deletes saved posts. | User data loss. | Use explicit non-destructive `SET NULL` semantics plus migration/store/concurrency tests. |
| RISK-004 | Saved lists bypass existing moderation or relationship policy. | Hidden or blocked content exposure. | Reuse canonical response hydration and current policy decisions; add policy regression tests. |
| RISK-005 | Cursor/filter/sort mismatch causes skipped or repeated saved items. | Confusing collection browsing. | Encode scope and direction into opaque keyset cursors and reject incompatible reuse. |
| RISK-006 | Folder name validation or case-insensitive ordering differs across application and database code. | Invalid names may be accepted or duplicate-name pagination may become unstable. | Centralize trimming/validation and ordering behavior; cover whitespace, Unicode, slash, control-character, duplicate-name, and page-boundary fixtures. |
| RISK-007 | Post, required reply-ancestor, or membership lifecycle leaves unintended private rows or deletes them too early. | Orphaned saved replies or premature private-data loss. | Encode separate destructive target/ancestor deletion, temporary eligibility/context, and owner-membership rules in store and lifecycle tests. |

## 20. Assumptions

None identified. Folder deletion, naming, ordering, counts, quotas, mute shaping, reply-context lifecycle, and mutation semantics were confirmed during requirements grilling.

## 21. Open Questions

None.

## 22. Review Status

Status: Reviewed
Risk level: Medium
Review recommended: No (completed through requirements grilling)
Reviewer: Product owner
Date: 2026-07-20
Notes: The privacy and one-folder model, reply navigation/context lifecycle, folder deletion, duplicate-name behavior, validation, ordering, request semantics, unavailable-content policy, counts, quotas, response metadata, and timestamp behavior were reviewed and confirmed. Document-review findings were resolved by specifying transactional descendant-save cleanup, the shared `EngagementSummaries` hydration seam, existing non-confidential cursor semantics, and opaque-string folder IDs. There are no blocking questions; the medium risk remains because the feature creates private persistent-data, lifecycle, pagination, and account-isolation contracts.

## 23. Handoff To Test Design

- Requirements file: `01-requirements.md`
- Next test specification: `02-acceptance-tests.md`
- Must-cover requirement IDs: BR-001–BR-004, FR-001–FR-020, NFR-001–NFR-004, NFR-006, RULE-001–RULE-008.
- Suggested test levels:
  - Unit: folder-name normalization/validation, cursor encoding/compatibility, saved ordering, error mapping, lifecycle decisions.
  - Store/integration: migration up/down/up, owner constraints, duplicate folder names, idempotent save/unsave, omitted-versus-null folder assignment, single-folder moves, folder CRUD, non-destructive delete, folder timestamp behavior, concurrency, target/ancestor/member deletion behavior, both sort directions and scopes.
  - Handler/route contract: auth/device/body/rate policies, camelCase responses, standard errors, create/update/delete response codes, folder-name validation, missing/cross-owner folder behavior, list filters, and opaque cursors.
  - Privacy/regression: two-owner isolation, multi-account session behavior, author non-disclosure, viewer-state batch hydration without folder names, reply root-plus-focus context, temporary/permanent content lifecycle, moderation/mute/block/membership shaping, and unchanged like/repost/reply/post behavior.
- Blocking open questions: None.
