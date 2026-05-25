# Requirements: Follow / Unfollow MVP

## 1. Initial Request

The user asked for requirements for scope 1 from the recommended next feature: a Follow/Unfollow MVP that wires the existing profile Follow button, indexes the follow graph, exposes follow state/counts, and prepares the project for a later home timeline.

## 2. Current Codebase Findings

- Relevant files:
  - `docs/roadmap.md` lists follow/unfollow interactions and timeline consumption as open v1 work.
  - `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md` already names `POST /v1/profiles/@{handleOrDid}/follows` and `DELETE /v1/profiles/@{handleOrDid}/follows`, with `app.bsky.graph.follow` as the PDS record type.
  - `docs/superpowers/specs/2026-04-23-profile-onboarding-design.md` explicitly defers profile counts and viewer relationship fields until graph indexing exists.
  - `appview/internal/api/profile.go`, `profile_response.go`, and `profile_store.go` define current profile read/write behavior and response shape.
  - `appview/internal/routes/routes.go` has no follow routes today.
  - `appview/internal/auth/pds_client.go` already exposes `CreateRecord` and `DeleteRecord`, which can support follow writes.
  - `app/lib/profile/pages/profile_page.dart` renders a visitor Follow button but currently shows `profileFollowComingSoon`.
  - `app/lib/profile/widgets/profile_meta_section.dart` hardcodes following/follower/project counts.
  - `app/lib/profile/models/profile.dart` currently has no follow counts or viewer relationship fields.
- Existing patterns:
  - AppView `/v1/*` routes require authentication and `X-Craftsky-Device-Id`.
  - API responses use camelCase JSON and the shared error envelope `{error, message, requestId}` for errors.
  - PDS writes are mediated by the AppView; Flutter never receives PDS access/refresh tokens.
  - Indexers are registered by NSID in `appview/internal/app/deps.go`; handlers should treat the firehose-backed database as the read-side source of truth while write endpoints may return synthetic responses for responsiveness.
  - Flutter data access follows `ApiClient` → `Repository` → Riverpod provider/notifier patterns.
- Current behavior:
  - Users can view and edit profiles, create/delete posts, upload images, like/repost posts, and view post/thread/profile post surfaces.
  - Visitor profile Follow UI is non-functional and reports “Follow coming soon.”
  - Profile counts are placeholder values.
  - The AppView does not store or expose follow graph state.
- Constraints discovered:
  - No new Craftsky lexicon should be introduced for follows; `app.bsky.graph.follow` is the planned standard follow record.
  - Reads must continue to come from the AppView; writes must go through the AppView to the user’s PDS.
  - The MVP must only allow following indexed Craftsky profiles.
  - Home timeline, notifications, follow-list screens, blocks, mutes, and reports are outside this scope.
- Test/build commands discovered:
  - AppView: `just test` runs Go tests on the host against compose Postgres.
  - Flutter: existing app tests live under `app/test`; exact command is typically `flutter test` from `app/`.

## 3. Clarifying Questions And Decisions

### Q1: For the Follow/Unfollow MVP, what should the endpoint allow as a follow target?

Answer: Craftsky profiles only.

Decision / implication: Follow and unfollow endpoints must resolve the target identity and require an indexed `craftsky_profiles` row for the target. The graph, counts, and viewer state for this MVP are scoped to Craftsky members, even though the PDS record type is the reusable `app.bsky.graph.follow` record.

### Q2: Should requirements use the full vertical Option A scope?

Answer: Yes, Option A.

Decision / implication: Requirements cover backend persistence/indexing, AppView APIs, profile response changes, and Flutter UI/data-layer wiring. Backend-only or UI-only staging is out of scope for this requirements document.

## 4. Candidate Approaches

### Option A: Full Vertical Follow/Unfollow MVP

Summary: Add AppView storage/indexing for `app.bsky.graph.follow`, expose follow/unfollow API endpoints and profile relationship/count fields, and wire the existing Flutter Follow button and counts.

Pros:
- Delivers visible user value in one coherent slice.
- Removes existing UI placeholders.
- Establishes the graph foundation needed by the later home timeline.
- Matches the already documented API architecture.

Cons:
- Touches persistence, indexing, API, and Flutter UI/data layers.
- Requires careful handling of PDS/firehose eventual consistency.

Risks:
- Inconsistent state if PDS writes succeed but firehose indexing lags or misses an event.
- Count semantics could become confusing if non-Craftsky accounts are accidentally included.

### Option B: Backend/API First Only

Summary: Add graph storage/indexing and API endpoints, but leave Flutter profile UI as “coming soon.”

Pros:
- Easier to test in isolation.
- Lower UI regression risk.

Cons:
- Does not deliver user-visible value.
- Requires another planning/implementation slice before the product behavior works.

Risks:
- Backend contracts may drift from eventual UI needs.

### Option C: UI-First Optimistic Follow State Without Full Indexing

Summary: Wire the button to issue PDS writes before adding robust graph indexing and counts.

Pros:
- Fastest path to a visible demo.

Cons:
- Counts and relationship state would be unreliable.
- Does not cleanly unlock the later timeline.
- Conflicts with the profile onboarding spec’s note that counts/viewer relationship fields require graph indexing.

Risks:
- User-visible state could diverge from the AppView read model.

## 5. Recommended Direction

Recommended approach: Option A, full vertical Follow/Unfollow MVP.

Why: The existing codebase already has profile UI, AppView auth/PDS-write primitives, API conventions, and post interaction patterns. A vertical slice removes placeholders and creates the graph foundation needed for the next planned feature (`GET /v1/feed/timeline`) while avoiding a new lexicon or broad product expansion.

## 6. Problem / Opportunity

Craftsky currently lets users create profiles and posts, but it lacks a real social graph. Visitor profiles display a non-functional Follow button and fake stats, so users cannot build the relationships required for a useful chronological home feed. Implementing follows between Craftsky profiles closes that gap and creates the prerequisite data for timeline scope 2.

## 7. Goals

- G-001: Let an authenticated Craftsky user follow another Craftsky profile.
- G-002: Let an authenticated Craftsky user unfollow a previously followed Craftsky profile.
- G-003: Show real follower/following counts on profile screens.
- G-004: Show whether the authenticated viewer follows a visited profile.
- G-005: Preserve Craftsky’s architecture: PDS-backed public records, AppView-backed reads, and no PDS tokens in Flutter.
- G-006: Establish graph persistence and indexing that can be reused by the later home timeline.

## 8. Non-Goals

- NG-001: No `GET /v1/feed/timeline` implementation in this scope.
- NG-002: No follower list or following list screens.
- NG-003: No notifications for new followers.
- NG-004: No blocks, mutes, reports, or moderation workflow changes.
- NG-005: No new Craftsky follow lexicon.
- NG-006: No ability to follow non-Craftsky profiles through Craftsky in this MVP.
- NG-007: No profile project-count implementation; project counts may remain separate future work.
- NG-008: No changes to PDS token storage or Flutter possession of PDS credentials.
- NG-009: No automatic backfill requirement for historical follows unless the test-design/implementation stages explicitly identify an existing supported backfill pattern for this graph.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Authenticated Craftsky user | A signed-in user with a Craftsky session and indexed Craftsky profile | Follow/unfollow other Craftsky profiles and see accurate relationship state. |
| Visited Craftsky profile owner | A Craftsky user whose profile is viewed by another user | Have their follower/following counts reflect active Craftsky follow relationships. |
| Flutter client | The Craftsky mobile app | Render profile relationship state and trigger follow/unfollow API calls without holding PDS tokens. |
| AppView | Go service mediating reads, writes, and indexing | Write follow records to PDS, index follow graph events, and expose graph-derived profile state. |
| User PDS | User-owned atproto data server | Store `app.bsky.graph.follow` records authored by the follower. |

## 10. Current Behavior

Visitor profile pages render a Follow button, but tapping it only shows a “coming soon” message. The profile response does not include `viewerIsFollowing`, `followingCount`, or `followerCount`. The profile meta section hardcodes fake following/follower/project stats. The AppView has no follow graph migration, indexer, store, routes, or API handlers.

## 11. Desired Behavior

An authenticated Craftsky user visiting another Craftsky profile can tap Follow. The AppView writes an `app.bsky.graph.follow` record to the caller’s PDS, returns success using Craftsky API conventions, and the Flutter UI reflects the new relationship and count. The same user can tap Following/Unfollow to remove the active follow, with the AppView deleting the PDS record and the Flutter UI reflecting the change. Profile reads expose real Craftsky follower/following counts and whether the authenticated viewer follows the profile.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | Craftsky users must be able to follow and unfollow other Craftsky profiles from the profile surface. | Following is the social graph foundation for timeline and community discovery. | Prompt, user confirmation | AC-001, AC-002, AC-010 |
| BR-002 | Business | Must | Profile screens must display real follow relationship state and real follower/following counts. | Removes placeholders and makes relationship state visible to users. | Codebase, profile spec | AC-003, AC-004, AC-011 |
| BR-003 | Business | Should | The implementation should create graph data reusable by the future chronological followed-account feed. | Scope 2 depends on reliable follow graph data. | Discovery, roadmap | AC-005 |
| FR-001 | Functional | Must | The AppView shall persist active and deleted `app.bsky.graph.follow` records in Postgres with enough data to derive follower counts, following counts, and viewer relationship state. | The AppView read model needs graph state for profiles and later timeline queries. | Discovery | AC-005, AC-006, AC-007 |
| FR-002 | Functional | Must | The AppView shall index `app.bsky.graph.follow` create and delete/tombstone events from the firehose idempotently. | Follow records are public PDS records; the AppView must consume them through the atproto indexing path. | Architecture, discovery | AC-006, AC-007 |
| FR-003 | Functional | Must | The AppView shall expose `POST /v1/profiles/@{handleOrDid}/follows` for an authenticated viewer to follow a target Craftsky profile. | Matches the existing API architecture and enables Flutter follow actions. | API architecture spec | AC-001, AC-008, AC-012, AC-013 |
| FR-004 | Functional | Must | The AppView shall expose `DELETE /v1/profiles/@{handleOrDid}/follows` for an authenticated viewer to unfollow a target Craftsky profile. | Matches the existing API architecture and enables Flutter unfollow actions. | API architecture spec | AC-002, AC-009, AC-012 |
| FR-005 | Functional | Must | Follow and unfollow handlers shall write/delete `app.bsky.graph.follow` records through the AppView PDS client factory using the caller’s OAuth session. | Preserves Craftsky’s write-through-PDS model and avoids PDS tokens on the client. | AGENTS.md, API architecture | AC-001, AC-002, AC-014 |
| FR-006 | Functional | Must | Profile API responses shall include `followingCount`, `followerCount`, and `viewerIsFollowing` using camelCase JSON. | Flutter needs these fields to render counts and button state. | Profile spec, user request | AC-003, AC-004, AC-011 |
| FR-007 | Functional | Must | Flutter shall extend the profile data layer/model to consume the new profile relationship/count fields and call the follow/unfollow endpoints through the existing API client/repository/provider pattern. | Keeps client implementation aligned with established patterns. | Codebase patterns | AC-010, AC-011, AC-014 |
| FR-008 | Functional | Must | Flutter shall replace the visitor profile “coming soon” Follow action with a real toggle that updates button label/state and relevant counts after successful follow/unfollow. | Delivers the user-visible MVP. | Codebase placeholder | AC-010, AC-011, AC-015 |
| FR-009 | Functional | Should | Flutter should preserve a usable profile screen if a follow/unfollow request fails, surfacing an error message and leaving or restoring the last confirmed state. | Prevents misleading relationship state under network or PDS failure. | Existing messaging patterns | AC-015 |
| NFR-001 | Non-functional | Must | New `/v1/*` follow APIs must follow existing Craftsky API conventions for authentication, device ID, camelCase JSON, opaque request IDs, and error envelopes. | Maintains API consistency and testability. | AGENTS.md, API architecture | AC-008, AC-009, AC-012, AC-013 |
| NFR-002 | Non-functional | Must | Follow graph indexing must be idempotent and safe for repeated firehose events for the same record URI/CID. | Firehose consumers may observe retries or duplicate events. | Indexer conventions | AC-006, AC-007 |
| NFR-003 | Non-functional | Must | The Flutter app must not hold PDS access or refresh tokens for follow/unfollow operations. | Architectural security rule. | AGENTS.md | AC-014 |
| NFR-004 | Non-functional | Should | Profile count and viewer-state reads should avoid N+1 profile or graph queries for a single profile response. | Keeps profile rendering scalable enough for v1. | Existing profile/post hydration patterns | AC-003, AC-004 |
| RULE-001 | Business rule | Must | A follow target must resolve to an indexed Craftsky profile; non-Craftsky targets must not be followed through Craftsky in this MVP. | User explicitly chose Craftsky profiles only. | User answer | AC-012 |
| RULE-002 | Business rule | Must | A user must not be allowed to follow themself through the follow endpoint. | Self-follow has no product value and would distort counts. | Product rule | AC-013 |
| RULE-003 | Business rule | Must | At most one active follow may exist per follower DID and target DID. Repeating follow should be idempotent from the caller’s perspective. | Prevents duplicate counts and supports retry behavior. | Discovery | AC-001, AC-006 |
| RULE-004 | Business rule | Must | Unfollowing an already-unfollowed or never-followed target should be idempotent from the caller’s perspective. | Supports safe retries and simple UI behavior. | Existing delete/idempotency patterns | AC-002, AC-009 |
| RULE-005 | Business rule | Must | Follower and following counts must count only active Craftsky-to-Craftsky follow relationships and must exclude deleted/inactive follows. | Keeps v1 count semantics clean and aligned with Craftsky-only targeting. | User answer, discovery | AC-003, AC-005, AC-007 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-003, FR-005, RULE-003 | Given authenticated user A and indexed Craftsky profile B, when A follows B through `POST /v1/profiles/@{handleOrDid}/follows`, then the AppView writes an `app.bsky.graph.follow` record to A’s PDS and returns a successful response. |
| AC-002 | BR-001, FR-004, FR-005, RULE-004 | Given authenticated user A actively follows indexed Craftsky profile B, when A unfollows B through `DELETE /v1/profiles/@{handleOrDid}/follows`, then the AppView deletes or deactivates the active PDS follow record and returns a successful response. |
| AC-003 | BR-002, FR-001, FR-006, NFR-004, RULE-005 | Given profile B has active Craftsky followers, when an authenticated viewer fetches B’s profile, then `followerCount` equals the number of active Craftsky-to-Craftsky follows targeting B. |
| AC-004 | BR-002, FR-006, NFR-004 | Given profile A actively follows other Craftsky profiles, when an authenticated viewer fetches A’s profile, then `followingCount` equals the number of active Craftsky-to-Craftsky follows authored by A. |
| AC-005 | BR-003, FR-001, RULE-005 | Given the follow graph contains active and deleted follow records, when future feed work queries active followed DIDs, then the persistence model can identify the active targets followed by a viewer without consulting the PDS directly. |
| AC-006 | FR-001, FR-002, NFR-002, RULE-003 | Given the follow indexer receives the same active follow event more than once, when it handles the repeated event, then only one active relationship contributes to counts. |
| AC-007 | FR-001, FR-002, NFR-002, RULE-005 | Given the follow indexer receives a delete/tombstone event for a previously indexed follow, when profile counts are read, then the deleted relationship no longer contributes to follower or following counts. |
| AC-008 | FR-003, NFR-001 | Given a follow request is missing authentication or required device ID, when it reaches the AppView, then it is rejected using existing `/v1/*` authentication/device-id error behavior. |
| AC-009 | FR-004, NFR-001, RULE-004 | Given authenticated user A does not actively follow indexed Craftsky profile B, when A sends `DELETE /v1/profiles/@B/follows`, then the endpoint succeeds without creating a follow or decrementing counts below the active graph state. |
| AC-010 | BR-001, FR-007, FR-008 | Given a visitor views another Craftsky profile in Flutter, when `viewerIsFollowing` is false, then the profile action shows Follow and tapping it calls the follow endpoint. |
| AC-011 | BR-002, FR-006, FR-007, FR-008 | Given a profile response includes `viewerIsFollowing`, `followerCount`, and `followingCount`, when Flutter renders the profile, then the Follow/Following label and stats reflect those response fields rather than placeholder values. |
| AC-012 | FR-003, FR-004, NFR-001, RULE-001 | Given a target handle/DID resolves but has no indexed Craftsky profile, when an authenticated user attempts follow or unfollow through Craftsky, then the endpoint rejects the target with a documented error envelope rather than writing a follow. |
| AC-013 | FR-003, NFR-001, RULE-002 | Given authenticated user A targets A’s own profile, when A sends a follow request, then the AppView rejects the request with a documented validation error and does not write a PDS follow. |
| AC-014 | FR-005, FR-007, NFR-003 | Given Flutter initiates follow/unfollow, when the operation is performed, then Flutter sends only the Craftsky session-authenticated API request and never receives or stores PDS tokens. |
| AC-015 | FR-008, FR-009 | Given Flutter attempts follow/unfollow and the AppView returns an error or the request fails, when the operation completes, then the user sees an error message and the profile UI does not falsely persist an unconfirmed relationship/count state. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | Target handle cannot be parsed as a handle or DID | Return existing-style `invalid_identifier` error envelope; no PDS write. | FR-003, FR-004, NFR-001 |
| EC-002 | Target handle resolves fails due to identity service/PDS lookup problem | Return existing-style identity unavailable error; no PDS write. | FR-003, FR-004, NFR-001 |
| EC-003 | Target resolves but is not an indexed Craftsky profile | Reject as non-Craftsky target; no PDS write. | RULE-001 |
| EC-004 | Authenticated viewer attempts to follow self | Reject with validation error; no PDS write. | RULE-002 |
| EC-005 | User repeats follow request after already following target | Return success/idempotent result and keep one active relationship. | RULE-003 |
| EC-006 | User repeats unfollow request after no active follow exists | Return success/idempotent result and keep counts unchanged. | RULE-004 |
| EC-007 | PDS follow write fails after request validation | Return `pds_write_failed` or equivalent documented error; Flutter surfaces failure and preserves/restores last confirmed state. | FR-005, FR-009 |
| EC-008 | PDS delete fails during unfollow | Return `pds_write_failed` or equivalent documented error; Flutter surfaces failure and preserves/restores last confirmed state. | FR-005, FR-009 |
| EC-009 | Firehose indexing lags after successful follow/unfollow | API/UI may update optimistically after successful write, but eventual profile reads converge to indexed graph state. | FR-001, FR-008 |
| EC-010 | Follow delete/tombstone arrives for an unknown follow URI | Indexer handles safely without corrupting counts or failing the stream. | FR-002, NFR-002 |

## 15. Data / Persistence Impact

- New fields:
  - A new AppView follow graph persistence model is required, expected to store at least follow record URI, follower DID, target DID/subject DID, rkey, CID, created/indexed timestamps, and deleted/inactive state.
  - Profile API responses add `followingCount`, `followerCount`, and `viewerIsFollowing`.
- Changed fields:
  - Existing profile response shape is additive; existing fields remain camelCase and retain existing semantics.
  - Flutter `Profile` model gains additive count/relationship fields.
- Migration required:
  - Yes. A new migration is required for follow graph storage and indexes that support active relationship lookup by follower and target.
- Backwards compatibility:
  - Additive API fields should not break existing clients.
  - The Flutter app version implementing this feature will require the new fields to render real counts/state; test design should specify defaults or failure behavior if the API is older.

## 16. UI / API / CLI Impact

- UI:
  - Visitor profile Follow/Following button becomes functional.
  - Profile stats use real `followingCount` and `followerCount`; project count remains out of scope.
  - Follow/unfollow failures surface through existing app messaging patterns.
- API:
  - Add `POST /v1/profiles/@{handleOrDid}/follows`.
  - Add `DELETE /v1/profiles/@{handleOrDid}/follows`.
  - Add profile response fields `followingCount`, `followerCount`, `viewerIsFollowing`.
  - All new routes remain authenticated and device-id protected.
- CLI:
  - No dedicated CLI feature required. Existing `cli request` may be used for smoke testing if implementation supports it.
- Background jobs:
  - Add/register an indexer for `app.bsky.graph.follow` firehose events.

## 17. Security / Privacy / Permissions

- Authentication:
  - Follow/unfollow APIs require the same Craftsky session authentication as existing `/v1/*` profile/post APIs.
- Authorization:
  - The follower is always the authenticated DID. Clients cannot specify a different follower DID.
  - Self-follow is prohibited.
- Sensitive data:
  - Follow records are public PDS data. No private data is written to PDS by this feature beyond the user’s intentional public follow action.
  - Flutter never receives PDS access/refresh tokens.
- Abuse cases:
  - Rate limiting is not implemented in this scope, but repeated follow/unfollow behavior should be idempotent and should not create duplicate active relationships.
  - Blocks/mutes/reporting interactions are out of scope and should be considered in later moderation work.

## 18. Observability

- Events:
  - No product analytics events are required by this requirements slice.
- Logs:
  - AppView should log follow/unfollow request start, target resolution failures, PDS write/delete failures, and indexer errors using existing structured logging patterns and request IDs where available.
- Metrics:
  - No new metrics are required, but test design should consider whether existing health/indexer observability can surface graph indexer failures.
- Alerts:
  - No new alerts are required for MVP.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | PDS write succeeds but firehose indexing lags. | Profile reads may temporarily show stale counts/state. | Permit UI/API optimistic update after successful write; document eventual consistency and verify convergence through indexer tests. |
| RISK-002 | Duplicate follow events or repeated follow requests inflate counts. | Profile counts become wrong and timeline queries later duplicate authors. | Enforce one active follow per follower-target pair and make indexer idempotent. |
| RISK-003 | Non-Craftsky follows leak into counts. | Counts/timeline semantics become unclear. | Require target Craftsky profile existence and count only Craftsky-to-Craftsky active follows. |
| RISK-004 | Unfollow needs to know the active PDS record rkey/URI. | Delete may fail or delete the wrong record if active follow lookup is wrong. | Persist active follow URI/rkey and define idempotent no-active-follow behavior. |
| RISK-005 | API response change and Flutter model change ship out of sync. | Flutter may fail to decode or display profile responses. | Keep API additions backward-compatible and test Flutter model decoding with the new fields. |
| RISK-006 | This feature touches migration, firehose indexing, API writes, and UI. | Higher regression surface across appview and app. | Require review before test design/implementation and cover with acceptance, integration, and widget tests. |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | `app.bsky.graph.follow` is the correct record type for Craftsky follow relationships. | A lexicon/ADR process would be required before implementation. |
| ASM-002 | Current `PDSClient.CreateRecord` and `DeleteRecord` primitives are sufficient for follow/unfollow writes. | Requirements may need to add PDS client capability changes. |
| ASM-003 | Craftsky-only follow targeting is acceptable for v1 even though atproto follows can target any account. | API behavior would need to widen and profile/count semantics would need redesign. |
| ASM-004 | Existing Flutter messaging/snackbar patterns are sufficient for follow/unfollow failure UX. | Additional UI design requirements may be needed. |
| ASM-005 | Historical follow backfill is not required for MVP; newly observed firehose events and direct API writes are sufficient. | If existing follow records must be imported, scope expands to include backfill design and tests. |

## 21. Open Questions

- [ ] Non-blocking: Should the follow/unfollow success response return a small relationship/count payload, the full updated profile, or no body? Test design should force a contract before implementation.
- [ ] Non-blocking: Should `viewerIsFollowing` be `false` or omitted when the viewer fetches their own profile? Requirements prefer a stable boolean field, but exact self-profile semantics should be finalized in API tests.
- [ ] Non-blocking: Should deleted follows remain as historical rows with `deletedAt` or be removed from storage? Requirements only require active counts and idempotent behavior; implementation can choose the storage strategy if tests preserve behavior.

## 22. Review Status

Status: Draft

Risk level: High

Review recommended: Required

Reviewer: TBD

Date: 2026-05-25

Notes: Review is required before test design/implementation because this feature includes a database migration, firehose indexing, PDS write/delete operations, auth-protected API changes, and user-visible Flutter behavior.

## 23. Handoff To Test Design

- Requirements file: `docs/changes/2026-05-25-follow-unfollow-mvp/01-requirements.md`
- Next test specification: `02-acceptance-tests.md`
- Must-cover requirement IDs:
  - Business: BR-001, BR-002
  - Functional: FR-001, FR-002, FR-003, FR-004, FR-005, FR-006, FR-007, FR-008
  - Non-functional: NFR-001, NFR-002, NFR-003
  - Rules: RULE-001, RULE-002, RULE-003, RULE-004, RULE-005
- Suggested test levels:
  - AppView handler tests for follow/unfollow success, validation, idempotency, auth/device-id failures, PDS failures, and non-Craftsky targets.
  - AppView store/integration tests for counts, viewer relationship state, active/deleted relationships, and uniqueness.
  - Indexer tests for create, duplicate create, delete/tombstone, and unknown delete events.
  - Flutter model/API client/repository/provider tests for new fields and endpoint calls.
  - Flutter widget tests for Follow/Following button state, count rendering, success updates, and failure messaging.
- Blocking open questions: None identified.
- Review gate: Because risk level is High and review is required, run Plannotator or equivalent document review before moving to test design unless the user explicitly accepts the risk and approves continuation.
