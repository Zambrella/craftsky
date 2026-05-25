# Requirements: Follow / Unfollow MVP

## 1. Initial Request

The user asked for requirements for scope 1 from the recommended next feature: a Follow/Unfollow MVP that wires the existing profile Follow button, indexes the follow graph, exposes follow state/counts, and prepares the project for a later home timeline.

After review and grilling, the scope was amended: follows must be interoperable `app.bsky.graph.follow` relationships, Craftsky must allow following/unfollowing non-Craftsky atproto accounts, Craftsky profiles should count indexed follows regardless of which app/client created them, and non-Craftsky profiles should be viewable with Bluesky profile information plus a visible `Non Craftsky profile` marker. For MVP, follower/following counts for non-Craftsky profiles are not required.

## 2. Current Codebase Findings

- Relevant files:
  - `docs/roadmap.md` lists follow/unfollow interactions and timeline consumption as open v1 work.
  - `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md` already names `POST /v1/profiles/@{handleOrDid}/follows` and `DELETE /v1/profiles/@{handleOrDid}/follows`, with `app.bsky.graph.follow` as the PDS record type.
  - `docs/superpowers/specs/2026-04-23-profile-onboarding-design.md` explicitly defers profile counts and viewer relationship fields until graph indexing exists.
  - `docs/superpowers/specs/2026-04-24-bluesky-backfill-ordering-race-design.md` documents the current backfill race for `app.bsky.actor.profile` and the existing `BlueskyBackfiller` pattern.
  - `docker-compose.yml` configures Tap collection filters and `TAP_SIGNAL_COLLECTION=social.craftsky.actor.profile`; adding `app.bsky.graph.follow` requires widening the filter and registering a matching indexer.
  - `appview/internal/api/profile.go`, `profile_response.go`, and `profile_store.go` define current profile read behavior and response shape.
  - `appview/internal/api/profile_store.go` currently treats absence from `craftsky_profiles` as `ErrProfileNotFound`.
  - `appview/internal/index/bluesky_profile.go` currently gates `app.bsky.actor.profile` indexing on Craftsky membership.
  - `appview/internal/routes/routes.go` has no follow routes today.
  - `appview/internal/auth/pds_client.go` already exposes `CreateRecord` and `DeleteRecord`, which can support follow writes.
  - `app/lib/profile/pages/profile_page.dart` renders a visitor Follow button but currently shows `profileFollowComingSoon`.
  - `app/lib/profile/widgets/profile_meta_section.dart` hardcodes following/follower/project counts.
  - `app/lib/profile/widgets/profile_stats.dart` already accepts nullable counts and renders `—` for missing values.
  - `app/lib/profile/models/profile.dart` currently has no follow counts, viewer relationship fields, or Craftsky/non-Craftsky marker.
- Existing patterns:
  - AppView `/v1/*` routes require authentication and `X-Craftsky-Device-Id`.
  - API responses use camelCase JSON and the shared error envelope `{error, message, requestId}` for errors.
  - PDS writes are mediated by the AppView; Flutter never receives PDS access/refresh tokens.
  - Indexers are registered by NSID in `appview/internal/app/deps.go`; handlers should treat the firehose-backed database as the durable read-side source of truth while write endpoints may return synthetic responses for responsiveness.
  - Flutter data access follows `ApiClient` → `Repository` → Riverpod provider/notifier patterns.
- Current behavior:
  - Users can view and edit Craftsky profiles, create/delete posts, upload images, like/repost posts, and view post/thread/profile post surfaces.
  - Visitor profile Follow UI is non-functional and reports “Follow coming soon.”
  - Profile counts are placeholder values.
  - Non-Craftsky accounts are not viewable through the profile API because profile reads require a `craftsky_profiles` row.
  - The AppView does not store or expose follow graph state.
- Constraints discovered:
  - No new Craftsky lexicon should be introduced for follows; `app.bsky.graph.follow` is the planned interoperable follow record.
  - Reads must continue to come from the AppView; writes must go through the AppView to the user’s PDS.
  - Follow/unfollow should be interoperable with Bluesky/atproto follows. A follow made in Craftsky is a normal public atproto follow; a follow made elsewhere can affect Craftsky state when indexed.
  - Follow/unfollow targets are not limited to Craftsky profiles. Any resolvable atproto account may be followed/unfollowed through Craftsky, subject to self-follow/self-unfollow rejection.
  - For MVP, follower/following counts are required for Craftsky profiles but are not required for non-Craftsky profile pages.
  - Directly asking a target user's PDS cannot provide globally authoritative `followerCount`, because follower records are authored in other users' repos. Accurate global follower counts require a graph index or external AppView/graph service; that broader count source is out of MVP scope for non-Craftsky profiles.
  - Home timeline, notifications, follow-list screens, blocks, mutes, reports, and global non-Craftsky count sourcing are outside this scope.
- Test/build commands discovered:
  - AppView: `just test` runs Go tests on the host against compose Postgres.
  - Flutter: existing app tests live under `app/test`; exact command is typically `flutter test` from `app/`.

## 3. Clarifying Questions And Decisions

### Q1: For the original Follow/Unfollow MVP, what should the endpoint allow as a follow target?

Answer: Initially, Craftsky profiles only.

Decision / implication: This decision was superseded by the later amendment. Follow/unfollow may now target non-Craftsky atproto accounts.

### Q2: Should requirements use the full vertical Option A scope?

Answer: Yes, Option A.

Decision / implication: Requirements cover backend persistence/indexing, AppView APIs, profile response changes, and Flutter UI/data-layer wiring. Backend-only or UI-only staging is out of scope for this requirements document.

### Q3: What is the source of truth immediately after a follow/unfollow write?

Answer: The firehose index is the durable source of truth, but the app is optimistically updated.

Decision / implication: Follow/unfollow endpoints should return updated profile information for the target so Flutter can update local state without waiting for a refetch. Subsequent reads converge to Tap/firehose-indexed state.

### Q4: Should follows created outside Craftsky count?

Answer: Yes. Follows created through Bluesky or other atproto clients should count when Tap picks them up and updates the AppView database.

Decision / implication: Follow indexing must not be limited to Craftsky-created records. It should index `app.bsky.graph.follow` events delivered by Tap regardless of the client/app that created them.

### Q5: Should follow/unfollow support non-Craftsky accounts?

Answer: Yes. It should be possible to follow/unfollow non-Craftsky accounts.

Decision / implication: The previous Craftsky-only target rule is replaced by a resolvable-atproto-account rule. The AppView must be able to return enough non-Craftsky profile data for the target profile surface.

### Q6: How should non-Craftsky profiles appear in Craftsky?

Answer: Visiting a non-Craftsky account through Craftsky should show the usual profile information from the Bluesky profile shape, plus a UI element that says `Non Craftsky profile`.

Decision / implication: Profile responses need a marker such as `isCraftskyProfile` so Flutter can render the non-Craftsky indicator. Craftsky-specific fields may be empty/default for non-Craftsky profiles.

### Q7: Should follow/unfollow endpoints return `204 No Content`?

Answer: No. They should return updated profile information.

Decision / implication: `POST /follows` and `DELETE /follows` should return `200 OK` with the updated target `ProfileResponse` after successful or idempotent operations.

### Q8: What should happen for self-unfollow?

Answer: Self-unfollow is not allowed.

Decision / implication: Both self-follow and self-unfollow are validation errors.

### Q9: Do old handles need to resolve?

Answer: No. Old handles fail.

Decision / implication: `handleOrDid` resolution uses current handle/DID semantics only; no historical handle lookup is required.

### Q10: What should happen if profile count calculation fails?

Answer: It should error if calculation fails.

Decision / implication: Required count calculation failures for Craftsky profiles should fail the profile response rather than silently returning zero or stale placeholder counts.

### Q11: Is public follow disclosure required in the UI?

Answer: No.

Decision / implication: Security/privacy notes should record that follows are public interoperable records, but no new UI warning is required.

### Q12: What should happen while follow/unfollow is in flight?

Answer: Disable/show a loading state; the existing button has this functionality.

Decision / implication: Flutter should prevent duplicate taps during an in-flight follow/unfollow request.

### Q13: What should the followed-state button label be?

Answer: `Unfollow`.

Decision / implication: The visitor action should show `Follow` when `viewerIsFollowing=false` and `Unfollow` when `viewerIsFollowing=true`.

### Q14: Should MVP require follower/following counts for non-Craftsky accounts?

Answer: No. For MVP, do not worry about follower/following counts for non-Craftsky accounts, but users can still navigate to and follow/unfollow these accounts.

Decision / implication: Non-Craftsky profile pages may omit/null follower and following counts or render them as unknown. Craftsky profile count behavior remains required.

## 4. Candidate Approaches

### Option A: Full Vertical Follow/Unfollow MVP With Craftsky-Only Targets

Summary: Add AppView storage/indexing for `app.bsky.graph.follow`, expose follow/unfollow API endpoints and profile relationship/count fields, and wire the existing Flutter Follow button and counts only for Craftsky profile targets.

Pros:
- Delivers visible user value in one coherent slice.
- Removes existing UI placeholders for Craftsky-to-Craftsky follows.
- Establishes the graph foundation needed by the later home timeline.
- Matches the already documented API route names.

Cons:
- No longer matches the amended interoperable product direction.
- Prevents users from following valid atproto accounts through Craftsky.

Risks:
- Product semantics diverge from atproto/Bluesky social graph expectations.

### Option B: Full Vertical Interoperable Follow/Unfollow MVP

Summary: Add AppView storage/indexing for `app.bsky.graph.follow`, allow follow/unfollow of any resolvable atproto account, expose profile relationship/count fields, support non-Craftsky profile display with an explicit marker, and wire Flutter UI/data-layer behavior.

Pros:
- Matches the amended product direction.
- Treats Craftsky follows as normal interoperable atproto follows.
- Lets Craftsky users follow non-Craftsky accounts before those accounts create Craftsky profiles.
- Establishes graph data that can later support a chronological followed-account feed.

Cons:
- Touches persistence, indexing, API, profile hydration, and Flutter UI/data layers.
- Requires clear count semantics for non-Craftsky profiles and partially indexed external graph data.

Risks:
- Users may expect global Bluesky-grade counts for non-Craftsky accounts, but MVP does not require those counts.
- Current `bluesky_profiles` membership gate must be revisited for non-Craftsky profile display.

### Option C: Split Non-Craftsky Profiles And Global Counts Into A Later Slice

Summary: Implement follow/unfollow only for Craftsky profiles now, then separately design non-Craftsky profile discovery and globally accurate graph counts.

Pros:
- Lower implementation risk.
- Allows a fuller graph/count architecture decision later.

Cons:
- Does not satisfy the amended requirement to follow/unfollow non-Craftsky accounts.
- Delays the desired interoperability behavior.

Risks:
- Requires reworking target validation and profile response behavior later.

## 5. Recommended Direction

Recommended approach: Option B, full vertical interoperable Follow/Unfollow MVP, with MVP-limited count scope for non-Craftsky profile pages.

Why: The user explicitly changed the scope from Craftsky-only targeting to interoperable atproto follow behavior. The existing codebase already has profile UI, AppView auth/PDS-write primitives, API conventions, and post interaction patterns. A vertical slice removes placeholders, enables public atproto follows, supports non-Craftsky account navigation, and creates graph foundations needed for `GET /v1/feed/timeline`, without requiring globally accurate non-Craftsky counts in this MVP.

## 6. Problem / Opportunity

Craftsky currently lets users create profiles and posts, but it lacks a real social graph. Visitor profiles display a non-functional Follow button and fake stats, so users cannot build relationships required for a useful chronological home feed. Implementing interoperable atproto follows closes that gap while allowing Craftsky users to interact with the wider Bluesky/atproto network.

## 7. Goals

- G-001: Let an authenticated Craftsky user follow another resolvable atproto account.
- G-002: Let an authenticated Craftsky user unfollow a previously followed atproto account.
- G-003: Show real follower/following counts on Craftsky profile screens.
- G-004: Show whether the authenticated viewer follows a visited profile.
- G-005: Preserve Craftsky’s architecture: PDS-backed public records, AppView-backed reads, and no PDS tokens in Flutter.
- G-006: Establish graph persistence and indexing that can be reused by the later home timeline.
- G-007: Bring a joining Craftsky member's existing atproto follow graph into the AppView so relationships can be recognized from historical `app.bsky.graph.follow` records.
- G-008: Allow navigation to non-Craftsky atproto profiles with Bluesky profile information and an explicit non-Craftsky marker.

## 8. Non-Goals

- NG-001: No `GET /v1/feed/timeline` implementation in this scope.
- NG-002: No follower list or following list screens.
- NG-003: No notifications for new followers.
- NG-004: No blocks, mutes, reports, or moderation workflow changes.
- NG-005: No new Craftsky follow lexicon.
- NG-006: No globally authoritative follower/following count source for non-Craftsky profile pages in this MVP.
- NG-007: No profile project-count implementation; project counts may remain placeholder/future work.
- NG-008: No changes to PDS token storage or Flutter possession of PDS credentials.
- NG-009: No UI for importing, reviewing, or selectively approving an existing Bluesky/atproto social graph; historical follow ingestion is backend graph state only.
- NG-010: No historical handle resolution; old handles may fail.
- NG-011: No new public-follow warning UI.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Authenticated Craftsky user | A signed-in user with a Craftsky session and indexed Craftsky profile | Follow/unfollow Craftsky and non-Craftsky atproto accounts and see relationship state. |
| Visited Craftsky profile owner | A Craftsky user whose profile is viewed by another user | Have their follower/following counts reflect active indexed atproto follow relationships. |
| Visited non-Craftsky account | A resolvable atproto account without a Craftsky profile row | Be visible with Bluesky profile information and clearly marked as non-Craftsky. |
| Flutter client | The Craftsky mobile app | Render profile relationship state, counts where available, non-Craftsky marker, and trigger follow/unfollow API calls without holding PDS tokens. |
| AppView | Go service mediating reads, writes, profile hydration, and indexing | Write follow records to PDS, index follow graph events, expose graph-derived profile state, and serve profile data for Craftsky and non-Craftsky accounts. |
| User PDS | User-owned atproto data server | Store `app.bsky.graph.follow` records authored by the follower. |

## 10. Current Behavior

Visitor profile pages render a Follow button, but tapping it only shows a “coming soon” message. The profile response does not include `viewerIsFollowing`, `followingCount`, `followerCount`, or a Craftsky/non-Craftsky marker. The profile meta section hardcodes fake following/follower/project stats. The AppView has no follow graph migration, indexer, store, routes, or API handlers. Non-Craftsky accounts are not returned by the profile API because reads require a `craftsky_profiles` row.

## 11. Desired Behavior

An authenticated Craftsky user visiting a profile can tap Follow. The AppView resolves the target as a current handle or DID, rejects self-targets, writes an interoperable `app.bsky.graph.follow` record to the caller’s PDS, and returns an updated profile response for the target. The Flutter UI updates from that response, showing Follow or Unfollow, a loading state during requests, real Craftsky profile counts where required, and a `Non Craftsky profile` marker for non-Craftsky accounts.

The same user can tap Unfollow to remove the active atproto follow, with the AppView deleting the known active PDS follow record and returning an updated target profile response. Firehose/Tap indexing remains the durable source of truth for subsequent reads, and external Bluesky/atproto follow changes update Craftsky state when delivered through Tap.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | Craftsky users must be able to follow and unfollow resolvable atproto accounts from the profile surface. | Following is the social graph foundation for timeline and community discovery, and the amended scope requires non-Craftsky targets. | Prompt, user amendment | AC-001, AC-002, AC-010, AC-020 |
| BR-002 | Business | Must | Craftsky profile screens must display real follow relationship state and real follower/following counts. | Removes placeholders and makes relationship state visible to users. | Codebase, profile spec | AC-003, AC-004, AC-011 |
| BR-003 | Business | Should | The implementation should create graph data reusable by the future chronological followed-account feed. | Scope 2 depends on reliable follow graph data. | Discovery, roadmap | AC-005 |
| BR-004 | Business | Must | Craftsky should recognize existing and external atproto follow relationships when those `app.bsky.graph.follow` records are delivered by Tap. | Users should not have to rebuild their social graph if they already follow accounts through Bluesky/atproto. | User amendment | AC-016, AC-017, AC-025 |
| BR-005 | Business | Must | Non-Craftsky atproto profiles must be viewable through Craftsky with Bluesky profile information and a visible non-Craftsky marker. | Users need context for non-Craftsky accounts they can follow/unfollow. | User amendment | AC-020, AC-021 |
| FR-001 | Functional | Must | The AppView shall persist active `app.bsky.graph.follow` records in Postgres with enough data to derive follower counts, following counts, and viewer relationship state. | The AppView read model needs graph state for profiles and later timeline queries. | Discovery | AC-005, AC-006, AC-007 |
| FR-002 | Functional | Must | The AppView shall index `app.bsky.graph.follow` create, update, and delete/tombstone events from Tap idempotently, without requiring either side to be a Craftsky profile. | Follow records are public interoperable PDS records; Craftsky must consume delivered graph events regardless of client/app origin. | Architecture, user amendment | AC-006, AC-007, AC-016, AC-017, AC-025 |
| FR-003 | Functional | Must | The AppView shall expose `POST /v1/profiles/@{handleOrDid}/follows` for an authenticated viewer to follow a target atproto account, returning `200 OK` with the updated target profile response on successful follow or idempotent already-following no-op. | Enables Flutter follow actions and lets the client update local state from the AppView response. | API architecture spec, user answer | AC-001, AC-008, AC-012, AC-013, AC-020, AC-022 |
| FR-004 | Functional | Must | The AppView shall expose `DELETE /v1/profiles/@{handleOrDid}/follows` for an authenticated viewer to unfollow a target atproto account, returning `200 OK` with the updated target profile response on successful unfollow or idempotent no-active-follow no-op. | Enables Flutter unfollow actions and lets the client update local state from the AppView response. | API architecture spec, user answer | AC-002, AC-009, AC-012, AC-013, AC-020, AC-022 |
| FR-005 | Functional | Must | Follow and unfollow handlers shall write/delete `app.bsky.graph.follow` records through the AppView PDS client factory using the caller’s OAuth session. | Preserves Craftsky’s write-through-PDS model and avoids PDS tokens on the client. | AGENTS.md, API architecture | AC-001, AC-002, AC-014 |
| FR-006 | Functional | Must | Profile API responses shall include `viewerIsFollowing`, an `isCraftskyProfile` marker, and count fields using camelCase JSON; `followingCount` and `followerCount` are required for Craftsky profiles and may be omitted or null for non-Craftsky profiles in MVP. | Flutter needs relationship fields and profile-type state while count scope differs by profile type. | User amendment, profile spec | AC-003, AC-004, AC-011, AC-018, AC-021, AC-023 |
| FR-007 | Functional | Must | Flutter shall extend the profile data layer/model to consume the new profile relationship/count/profile-type fields and call the follow/unfollow endpoints through the existing API client/repository/provider pattern. | Keeps client implementation aligned with established patterns. | Codebase patterns | AC-010, AC-011, AC-014, AC-021 |
| FR-008 | Functional | Must | Flutter shall replace the visitor profile “coming soon” Follow action with a real Follow/Unfollow toggle that updates button label/state and relevant counts from the AppView response after successful follow/unfollow. | Delivers the user-visible MVP. | Codebase placeholder, user answer | AC-010, AC-011, AC-015, AC-022, AC-024 |
| FR-009 | Functional | Should | Flutter should preserve a usable profile screen if a follow/unfollow request fails, surfacing an error message and leaving or restoring the last confirmed state. | Prevents misleading relationship state under network or PDS failure. | Existing messaging patterns | AC-015 |
| FR-010 | Functional | Must | When the AppView/Tap begins tracking a repo, the follow graph indexer shall consume historical and live `app.bsky.graph.follow` events authored by that repo, subject to Tap's supported backfill/replay behavior. | Existing Bluesky/atproto follows are part of the user's social graph and should be available to Craftsky when indexed. | User answer, Tap docs | AC-016, AC-017 |
| FR-011 | Functional | Must | The AppView profile read path shall be able to return a non-Craftsky atproto profile when identity resolves and Bluesky profile information can be hydrated or read from cache. | Non-Craftsky accounts must be navigable and followable through Craftsky. | User amendment, codebase discovery | AC-020, AC-021 |
| FR-012 | Functional | Must | The AppView shall index or hydrate `app.bsky.actor.profile` data for non-Craftsky accounts needed by visited/followed profile surfaces, without requiring a `craftsky_profiles` row. | Current membership-gated profile indexing cannot support non-Craftsky profile display. | Codebase discovery, user amendment | AC-020, AC-021 |
| NFR-001 | Non-functional | Must | New `/v1/*` follow APIs must follow existing Craftsky API conventions for authentication, device ID, camelCase JSON, opaque request IDs, and error envelopes. | Maintains API consistency and testability. | AGENTS.md, API architecture | AC-008, AC-009, AC-012, AC-013 |
| NFR-002 | Non-functional | Must | Follow graph indexing must be idempotent and safe for repeated firehose events for the same record URI/CID. | Firehose consumers may observe retries or duplicate events. | Indexer conventions | AC-006, AC-007 |
| NFR-003 | Non-functional | Must | The Flutter app must not hold PDS access or refresh tokens for follow/unfollow operations. | Architectural security rule. | AGENTS.md | AC-014 |
| NFR-004 | Non-functional | Should | Profile count and viewer-state reads for Craftsky profiles should avoid N+1 profile or graph queries for a single profile response. | Keeps profile rendering scalable enough for v1. | Existing profile/post hydration patterns | AC-003, AC-004 |
| RULE-001 | Business rule | Must | A follow target must resolve as a current atproto handle or DID; non-Craftsky targets are allowed. | User amended scope to support non-Craftsky accounts while old handles may fail. | User amendment | AC-012, AC-020 |
| RULE-002 | Business rule | Must | A user must not be allowed to follow or unfollow themself through the follow endpoints. | Self-follow has no product value and would distort relationship semantics. | User answer | AC-013 |
| RULE-003 | Business rule | Must | At most one active relationship may contribute to counts or viewer state for a follower DID and target DID pair. Repeating follow should be idempotent from the caller’s perspective. | Prevents duplicate counts and supports retry behavior. | Discovery, user answer | AC-001, AC-006 |
| RULE-004 | Business rule | Must | Unfollowing an already-unfollowed or never-followed target should be idempotent from the caller’s perspective, except self-targets remain validation errors. | Supports safe retries and simple UI behavior. | Existing delete/idempotency patterns, user answer | AC-002, AC-009, AC-013 |
| RULE-005 | Business rule | Must | Craftsky profile follower/following counts must count active indexed `app.bsky.graph.follow` relationships regardless of whether the record was created by Craftsky, Bluesky, or another atproto client, and must exclude deleted/inactive follows. | Amended count semantics are app/client-agnostic for the indexed graph. | User amendment | AC-003, AC-004, AC-005, AC-007, AC-025 |
| RULE-006 | Business rule | Must | `viewerIsFollowing` must be present and `false` when the authenticated viewer fetches their own profile. | Self-follow is prohibited, and a stable non-null boolean avoids client branching. | Plannotator feedback, user answer | AC-018 |
| RULE-007 | Business rule | Must | AppView follow graph storage must represent currently active follows; delete/tombstone processing must remove the active row rather than preserving deleted follow history with `deletedAt`. | Deleted follow records do not persist on the PDS, and the MVP has no product need for retained follow history. | Plannotator feedback | AC-007, AC-019 |
| RULE-008 | Business rule | Must | Non-Craftsky profile pages do not need follower/following counts in MVP; if unavailable, the UI must not render fake counts. | User explicitly scoped out non-Craftsky counts for MVP. | User plan feedback | AC-023 |
| RULE-009 | Business rule | Must | Profile count calculation failures for required Craftsky profile counts must fail the profile read rather than silently returning fake, zero, or stale placeholder counts. | The user requested an error if count calculation fails. | User answer | AC-026 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-003, FR-005, RULE-003 | Given authenticated user A and resolvable atproto profile B, when A follows B through `POST /v1/profiles/@{handleOrDid}/follows`, then the AppView writes an `app.bsky.graph.follow` record to A’s PDS and returns `200 OK` with B’s updated profile response. |
| AC-002 | BR-001, FR-004, FR-005, RULE-004 | Given authenticated user A actively follows resolvable atproto profile B, when A unfollows B through `DELETE /v1/profiles/@{handleOrDid}/follows`, then the AppView deletes the known active PDS follow record and returns `200 OK` with B’s updated profile response. |
| AC-003 | BR-002, FR-001, FR-006, NFR-004, RULE-005 | Given Craftsky profile B has active indexed followers, when an authenticated viewer fetches B’s profile, then `followerCount` equals the number of active indexed `app.bsky.graph.follow` relationships targeting B. |
| AC-004 | BR-002, FR-006, NFR-004, RULE-005 | Given Craftsky profile A actively follows indexed targets, when an authenticated viewer fetches A’s profile, then `followingCount` equals the number of active indexed `app.bsky.graph.follow` relationships authored by A. |
| AC-005 | BR-003, FR-001, RULE-005 | Given the follow graph contains active follow records, when future feed work queries active followed DIDs, then the persistence model can identify the active targets followed by a viewer without consulting the PDS directly. |
| AC-006 | FR-001, FR-002, NFR-002, RULE-003 | Given the follow indexer receives the same active follow event more than once, when it handles the repeated event, then only one active relationship contributes to counts and viewer state. |
| AC-007 | FR-001, FR-002, NFR-002, RULE-005, RULE-007 | Given the follow indexer receives a delete/tombstone event for a previously indexed follow URI, when profile counts are read, then the relationship no longer exists as an active stored follow and does not contribute to follower or following counts. |
| AC-008 | FR-003, NFR-001 | Given a follow request is missing authentication or required device ID, when it reaches the AppView, then it is rejected using existing `/v1/*` authentication/device-id error behavior. |
| AC-009 | FR-004, NFR-001, RULE-004 | Given authenticated user A does not actively follow resolvable atproto profile B, when A sends `DELETE /v1/profiles/@B/follows`, then the endpoint succeeds with an updated profile response without creating a follow or decrementing counts below the active graph state. |
| AC-010 | BR-001, FR-007, FR-008 | Given a visitor views another profile in Flutter, when `viewerIsFollowing` is false, then the profile action shows `Follow` and tapping it calls the follow endpoint. |
| AC-011 | BR-002, FR-006, FR-007, FR-008 | Given a Craftsky profile response includes `viewerIsFollowing`, `followerCount`, `followingCount`, and `isCraftskyProfile=true`, when Flutter renders the profile, then the Follow/Unfollow label and stats reflect those response fields rather than placeholder values. |
| AC-012 | FR-003, FR-004, NFR-001, RULE-001 | Given a target handle/DID cannot be parsed or currently resolved, when an authenticated user attempts follow or unfollow through Craftsky, then the endpoint rejects the target with a documented error envelope and does not write/delete a follow. |
| AC-013 | FR-003, FR-004, NFR-001, RULE-002, RULE-004 | Given authenticated user A targets A’s own profile, when A sends a follow or unfollow request, then the AppView rejects the request with a documented validation error and does not write/delete a PDS follow. |
| AC-014 | FR-005, FR-007, NFR-003 | Given Flutter initiates follow/unfollow, when the operation is performed, then Flutter sends only the Craftsky session-authenticated API request and never receives or stores PDS tokens. |
| AC-015 | FR-008, FR-009 | Given Flutter attempts follow/unfollow and the AppView returns an error or the request fails, when the operation completes, then the user sees an error message and the profile UI does not falsely persist an unconfirmed relationship/count state. |
| AC-016 | BR-004, FR-010 | Given user A already has historical `app.bsky.graph.follow` records on their PDS before creating a Craftsky profile, when the AppView/Tap begins tracking A's repo, then those historical follow records are eligible to be indexed without A manually re-following through Craftsky. |
| AC-017 | BR-004, FR-010 | Given user A has a historical active follow to user B, when both profile/identity data and graph indexing are available, then the relationship contributes to `viewerIsFollowing` and applicable Craftsky profile counts according to the same active indexed graph rules as newly created follows. |
| AC-018 | FR-006, RULE-006 | Given authenticated user A fetches A's own profile, when the profile response is returned, then `viewerIsFollowing` is present and `false`. |
| AC-019 | RULE-007 | Given the stored active follow row for A following B is removed by an unfollow delete/tombstone, when storage is inspected, then there is no retained deleted follow row for that relationship in the MVP follow graph table. |
| AC-020 | BR-001, BR-005, FR-003, FR-004, FR-011, FR-012, RULE-001 | Given B is a resolvable non-Craftsky atproto account, when authenticated user A visits or follows/unfollows B through Craftsky, then the AppView can return a profile response for B without requiring B to have a `craftsky_profiles` row. |
| AC-021 | BR-005, FR-006, FR-007, FR-011, FR-012 | Given a profile response represents a non-Craftsky account, when Flutter renders it, then the screen shows available Bluesky profile information and a visible `Non Craftsky profile` indicator. |
| AC-022 | FR-003, FR-004, FR-008 | Given follow/unfollow succeeds, when Flutter receives the AppView response, then it updates local profile state from that response rather than waiting for a separate profile refetch. |
| AC-023 | FR-006, RULE-008 | Given a non-Craftsky profile has unavailable follower/following counts in MVP, when Flutter renders the profile stats, then it omits them or renders them as unknown and does not show fake numeric values. |
| AC-024 | FR-008 | Given a follow/unfollow request is in flight, when the profile action is rendered, then the button is disabled or shows loading state so duplicate taps are prevented. |
| AC-025 | BR-004, FR-002, RULE-005 | Given an `app.bsky.graph.follow` record is created or deleted outside Craftsky and Tap delivers the event, when the follow indexer handles it, then the AppView graph state updates according to the same active/deleted rules as Craftsky-created follows. |
| AC-026 | RULE-009 | Given a Craftsky profile requires follower/following counts and the AppView cannot calculate them, when the profile is requested, then the profile request fails with a documented error instead of returning fake, zero, or stale placeholder counts. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | Target handle cannot be parsed as a current handle or DID | Return existing-style `invalid_identifier` error envelope; no PDS write/delete. | FR-003, FR-004, NFR-001, RULE-001 |
| EC-002 | Target handle resolution fails due to identity service/PDS lookup problem | Return existing-style identity unavailable error; no PDS write/delete. | FR-003, FR-004, NFR-001, RULE-001 |
| EC-003 | Target resolves but has no Craftsky profile | Allow profile display and follow/unfollow if Bluesky profile/identity hydration succeeds; mark as non-Craftsky. | BR-005, FR-011, FR-012, RULE-001 |
| EC-004 | Authenticated viewer attempts to follow self | Reject with validation error; no PDS write. | RULE-002 |
| EC-005 | Authenticated viewer attempts to unfollow self | Reject with validation error; no PDS delete. | RULE-002, RULE-004 |
| EC-006 | User repeats follow request after already following target | Return success with updated profile response and keep one active relationship contributing to counts/state. | RULE-003, FR-003 |
| EC-007 | User repeats unfollow request after no active follow exists | Return success with updated profile response and keep counts unchanged. | RULE-004, FR-004 |
| EC-008 | PDS follow write fails after request validation | Return `pds_write_failed` or equivalent documented error; Flutter surfaces failure and preserves/restores last confirmed state. | FR-005, FR-009 |
| EC-009 | PDS delete fails during unfollow | Return `pds_write_failed` or equivalent documented error; Flutter surfaces failure and preserves/restores last confirmed state. | FR-005, FR-009 |
| EC-010 | Firehose indexing lags after successful follow/unfollow | API/UI update from the successful AppView response, but eventual profile reads converge to indexed graph state. | FR-001, FR-008 |
| EC-011 | Follow delete/tombstone arrives for an unknown follow URI | Indexer handles safely without corrupting counts or failing the stream. | FR-002, NFR-002 |
| EC-012 | Historical follow targets a non-Craftsky account | Store/index the active relationship if delivered by Tap; include it in the author’s Craftsky following count if the author has a Craftsky profile, but non-Craftsky target profile counts remain optional/unknown in MVP. | BR-004, RULE-005, RULE-008 |
| EC-013 | Target becomes a Craftsky member after an already-indexed historical follow | Once profile and graph reads converge, the active relationship may contribute to applicable Craftsky counts/state without a new follow write. | BR-004, FR-010 |
| EC-014 | Same follow record URI arrives with a new CID | Indexer treats update as an upsert for that URI and converges counts/state to the latest active record. | FR-002, NFR-002 |
| EC-015 | Delete event includes URI/rkey but no original record body | Indexer deletes by URI/rkey using stored active row data and does not require the tombstone to include target DID. | FR-002, RULE-007 |
| EC-016 | Non-Craftsky profile count fields are missing/null | Flutter does not crash and does not show fake numbers. | FR-006, RULE-008 |
| EC-017 | Profile count query fails for a Craftsky profile | AppView returns a documented error rather than a partial profile with fake counts. | RULE-009 |

## 15. Data / Persistence Impact

- New fields:
  - A new AppView follow graph persistence model is required, expected to store at least active follow record URI, follower DID, target/subject DID, rkey, CID, and created/indexed timestamps.
  - The graph table should support deletion by URI/rkey alone because tombstone events may not carry the original follow record body.
  - Profile API responses add `viewerIsFollowing`, `isCraftskyProfile`, `followingCount`, and `followerCount`.
- Changed fields:
  - Existing profile response shape is additive; existing fields remain camelCase and retain existing semantics.
  - Flutter `Profile` model gains additive count/relationship/profile-type fields.
  - `bluesky_profiles` indexing or hydration must no longer be limited solely to `craftsky_profiles` membership for non-Craftsky profile display use cases.
- Migration required:
  - Yes. A new migration is required for follow graph storage and indexes supporting active relationship lookup by follower DID, target DID, `(follower DID, target DID)`, and record URI.
- Backwards compatibility:
  - Additive API fields should not break existing clients.
  - The Flutter app version implementing this feature will require the new fields to render real state; non-Craftsky counts must be handled as nullable/unknown.

## 16. UI / API / CLI Impact

- UI:
  - Visitor profile Follow/Unfollow button becomes functional.
  - Button shows `Follow` when not following and `Unfollow` when following.
  - Button is disabled or shows loading state while a follow/unfollow request is in flight.
  - Craftsky profile stats use real `followingCount` and `followerCount`; project count remains placeholder/future work.
  - Non-Craftsky profile pages show available Bluesky profile information and a `Non Craftsky profile` marker.
  - Non-Craftsky profile follower/following counts may be omitted or rendered as unknown in MVP.
  - Follow/unfollow failures surface through existing app messaging patterns.
- API:
  - Add `POST /v1/profiles/@{handleOrDid}/follows`, returning `200 OK` with target profile response.
  - Add `DELETE /v1/profiles/@{handleOrDid}/follows`, returning `200 OK` with target profile response.
  - Add profile response fields `viewerIsFollowing`, `isCraftskyProfile`, `followingCount`, and `followerCount`.
  - Allow `GET /v1/profiles/@{handleOrDid}` to return non-Craftsky atproto profiles when resolvable/hydratable.
  - All new routes remain authenticated and device-id protected.
- CLI:
  - No dedicated CLI feature required. Existing `cli request` may be used for smoke testing if implementation supports it.
- Background jobs:
  - Add/register an indexer for `app.bsky.graph.follow` firehose events.
  - Extend Tap collection filters to include `app.bsky.graph.follow`.
  - Ensure profile hydration/backfill can support non-Craftsky profile display and followed targets.

## 17. Security / Privacy / Permissions

- Authentication:
  - Follow/unfollow APIs require the same Craftsky session authentication as existing `/v1/*` profile/post APIs.
- Authorization:
  - The follower is always the authenticated DID. Clients cannot specify a different follower DID.
  - Self-follow and self-unfollow are prohibited.
- Sensitive data:
  - Follow records are public interoperable PDS data. No private data is written to PDS by this feature beyond the user’s intentional public follow/unfollow action.
  - Flutter never receives PDS access/refresh tokens.
  - No new UI warning about public follows is required for MVP per user decision.
- Abuse cases:
  - Rate limiting is not implemented in this scope, but repeated follow/unfollow behavior should be idempotent and should not create duplicate active relationships in Craftsky state.
  - Blocks/mutes/reporting interactions are out of scope and should be considered in later moderation work.

## 18. Observability

- Events:
  - No product analytics events are required by this requirements slice.
- Logs:
  - AppView should log follow/unfollow request start, target resolution failures, profile hydration failures, PDS write/delete failures, and indexer errors using existing structured logging patterns and request IDs where available.
- Metrics:
  - No new metrics are required, but test design should consider whether existing health/indexer observability can surface graph indexer failures.
- Alerts:
  - No new alerts are required for MVP.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | PDS write succeeds but firehose indexing lags. | Profile reads may temporarily show stale counts/state. | Return updated profile response for local UI update; document eventual consistency and verify convergence through indexer tests. |
| RISK-002 | Duplicate follow events or repeated follow requests inflate counts. | Profile counts become wrong and timeline queries later duplicate authors. | Enforce/collapse one active relationship per follower-target pair for counts/state and make indexer idempotent. |
| RISK-003 | Users expect global counts for non-Craftsky profiles. | Non-Craftsky profile pages may look incomplete versus Bluesky. | Explicitly scope non-Craftsky counts out of MVP and render unknown/omitted counts rather than fake values. |
| RISK-004 | Unfollow needs to know the active PDS record rkey/URI. | Delete may fail or delete the wrong record if active follow lookup is wrong. | Persist active follow URI/rkey and define idempotent no-active-follow behavior. |
| RISK-005 | API response change and Flutter model change ship out of sync. | Flutter may fail to decode or display profile responses. | Keep API additions backward-compatible and test Flutter model decoding with new nullable/non-nullable fields. |
| RISK-006 | This feature touches migration, firehose indexing, API writes, profile hydration, and UI. | Higher regression surface across appview and app. | Require review before test design/implementation and cover with acceptance, integration, and widget tests. |
| RISK-007 | Tap historical event delivery semantics are misunderstood or insufficient for existing follow graph import. | Existing Bluesky/atproto relationships may not appear automatically for new Craftsky members. | Verify Tap repo-tracking/backfill behavior during test design and implementation; make historical import acceptance criteria explicit. |
| RISK-008 | Non-Craftsky profile hydration broadens reads beyond current membership-gated profile model. | Profile reads may become slower or introduce new failure modes. | Cache/index hydrated Bluesky profile data through AppView and define error behavior for unavailable profile data. |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | `app.bsky.graph.follow` is the correct record type for Craftsky follow relationships. | A lexicon/ADR process would be required before implementation. |
| ASM-002 | Current `PDSClient.CreateRecord` and `DeleteRecord` primitives are sufficient for follow/unfollow writes. | Requirements may need to add PDS client capability changes. |
| ASM-003 | Tap can provide historical and live `app.bsky.graph.follow` events for tracked repos, consistent with the existing Tap-backed architecture. | If Tap cannot provide this, scope must add a separate import/backfill design before implementation can meet historical graph requirements. |
| ASM-004 | Existing Flutter messaging/snackbar patterns are sufficient for follow/unfollow failure UX. | Additional UI design requirements may be needed. |
| ASM-005 | A non-Craftsky account's Bluesky profile data can be hydrated through AppView-side atproto/PDS reads or indexed Tap data without Flutter talking to the PDS directly. | Non-Craftsky profile display would require a separate profile hydration design. |
| ASM-006 | MVP does not require globally authoritative follower/following counts for non-Craftsky profile pages. | A new external graph/AppView count source would need to be added before implementation. |

## 21. Open Questions

- [ ] Blocking before implementation: Confirm during test design whether Tap delivers historical `app.bsky.graph.follow` records for newly tracked Craftsky member repos as assumed. The user believes Tap can do it, but requested verification.
- [ ] Non-blocking for MVP: Decide in a later architecture/design slice whether Craftsky should use an external AppView/graph source for globally authoritative counts on non-Craftsky profiles.

## 22. Review Status

Status: Reviewed and amended

Risk level: High

Review recommended: Required

Reviewer: Plannotator and user feedback

Date: 2026-05-25

Notes: Plannotator feedback from 2026-05-25 was applied. A subsequent grilling/review changed the scope from Craftsky-only follows to interoperable atproto follow/unfollow with non-Craftsky profile navigation. The amendment plan was approved with the constraint that MVP does not need follower/following counts for non-Craftsky accounts.

## 23. Handoff To Test Design

- Requirements file: `docs/changes/2026-05-25-follow-unfollow-mvp/01-requirements.md`
- Next test specification: `02-acceptance-tests.md`
- Must-cover requirement IDs:
  - Business: BR-001, BR-002, BR-004, BR-005
  - Functional: FR-001, FR-002, FR-003, FR-004, FR-005, FR-006, FR-007, FR-008, FR-010, FR-011, FR-012
  - Non-functional: NFR-001, NFR-002, NFR-003
  - Rules: RULE-001, RULE-002, RULE-003, RULE-004, RULE-005, RULE-006, RULE-007, RULE-008, RULE-009
- Suggested test levels:
  - AppView handler tests for follow/unfollow success, validation, idempotency, auth/device-id failures, PDS failures, self-target rejection, current-handle resolution failures, Craftsky targets, and non-Craftsky targets.
  - AppView profile tests for Craftsky profile counts, non-Craftsky profile responses, `isCraftskyProfile`, nullable/unknown non-Craftsky counts, viewer relationship state, and count calculation failure behavior.
  - AppView store/integration tests for counts, viewer relationship state, active relationships, hard deletion, historical follows, update/upsert, deletion by URI, and uniqueness/collapsing by follower-target pair.
  - Indexer tests for create, update, duplicate create, delete/tombstone, unknown delete, externally-created follows, and historical follows delivered by Tap/backfill.
  - Flutter model/API client/repository/provider tests for new fields, nullable non-Craftsky counts, endpoint calls, and response-driven local state updates.
  - Flutter widget tests for Follow/Unfollow button state, loading/disabled state, Craftsky count rendering, non-Craftsky marker, unknown non-Craftsky counts, success updates, and failure messaging.
- Blocking open questions: Verify Tap historical follow delivery behavior before implementation.
- Review gate: Because risk level is High and review is required, run Plannotator or equivalent document review before moving to implementation unless the user explicitly accepts the risk and approves continuation.
