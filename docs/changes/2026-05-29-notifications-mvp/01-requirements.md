# Requirements: Notifications MVP

## 1. Initial Request

After the timeline implementation, identify and proceed with the next best slice of work. The approved direction is a read-only in-app notifications MVP that makes existing social activity visible in the Flutter Notifications tab.

## 2. Current Codebase Findings

- Relevant files:
  - AppView routing: `appview/internal/routes/routes.go` currently has no `GET /v1/notifications` route.
  - AppView indexed data sources: `craftsky_posts`, `craftsky_likes`, `craftsky_reposts`, and `atproto_follows` are already present through migrations `000010`, `000011`, and `000012`.
  - AppView response patterns: post/feed handlers use authenticated/device-id middleware, opaque cursor pagination, camelCase JSON, and existing `PostResponse` hydration.
  - Flutter route/UI: `app/lib/notifications/pages/notifications_page.dart` is currently a placeholder scaffold in the shell's Notifications branch.
  - Flutter data/provider patterns: timeline and profile list slices already provide API client, repository, provider, pagination, retry, and widget-test examples.
- Existing patterns:
  - `/v1/*` endpoints require `Authorization: Bearer <craftsky-token>` and `X-Craftsky-Device-Id`, except explicitly public/login routes.
  - List endpoints accept `limit` and opaque `cursor`; response `cursor` is omitted when no more results exist.
  - Flutter app talks only to the AppView for app reads/writes and must not read craft data directly from a PDS.
- Current behavior:
  - Users can post, reply, like, repost, follow, unfollow, view profile data, and consume the chronological timeline.
  - The Notifications tab renders only static placeholder text.
  - There is no push notification registration, unread state, notification table, or notification API.
- Constraints discovered:
  - No new lexicon is needed for this slice.
  - No PDS write path should be added.
  - Active-only likes/reposts are acceptable for MVP; historical notifications after unlike/unrepost are out of scope.
  - Search, moderation, push delivery, grouping, unread badges, and notification preferences are separate future slices.
- Test/build commands discovered:
  - AppView focused tests are typically run from `appview/` with `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes`.
  - Full AppView verification is exposed by `just test` from the repo root when the compose Postgres is available.
  - Flutter focused tests are run from `app/` with `flutter test ...`; analyzer checks should cover changed Dart source and tests.

## 3. Clarifying Questions And Decisions

### Q1: Which next slice should follow timeline?

Answer: The approved plan selects a read-only in-app notifications MVP.

Decision / implication: Scope centers on deriving notification rows from already-indexed social activity and rendering them in Flutter. Push, unread state, grouping, and persisted notification records are not included.

### Q2: Should MVP notifications be derived or persisted?

Answer: Use derived notifications for MVP.

Decision / implication: The AppView should query existing indexed tables at request time. This avoids new notification persistence/indexer fan-out work, but active-only likes/reposts may disappear when the actor unlikes/unreposts.

## 4. Candidate Approaches

### Option A: Derived read-only notifications feed

Summary: Add `GET /v1/notifications` as a union query over follows, likes, reposts, and replies, then render it in Flutter.

Pros:
- Smallest coherent slice after timeline.
- Uses existing indexed tables and route/provider patterns.
- Requires no new lexicons, PDS writes, push infrastructure, or migration for read/unread state.
- Validates notification semantics before committing to durable notification storage.

Cons:
- Like/repost notifications are active-only in this MVP.
- More complex SQL than a single event table.
- No unread badge/read tracking.

Risks:
- Ordering and pagination across heterogeneous events need careful tests.
- Duplicate/self-notification rules must be explicit.

### Option B: Persisted notifications table populated by indexers

Summary: Add a notification table and create notification records during follow/like/repost/reply indexing.

Pros:
- Preserves historical notifications even if interactions are later removed.
- Provides a foundation for read/unread state, push fan-out, and grouping.

Cons:
- Larger migration and indexer slice.
- Requires event idempotency and delete/retraction semantics now.
- Delays user-visible notification UI.

Risks:
- Harder to change notification semantics after persistence ships.
- More failure modes around indexer convergence and duplicate generation.

### Option C: Flutter placeholder polish only

Summary: Improve the Notifications tab empty/coming-soon UI without adding an API.

Pros:
- Very low risk.
- Quick visual improvement.

Cons:
- Does not close the product feedback loop.
- Leaves roadmap `GET /v1/notifications` untouched.

Risks:
- Gives the appearance of progress while social activity remains invisible.

## 5. Recommended Direction

Recommended approach: Option A, derived read-only notifications feed.

Why: Timeline made the main feed usable; notifications is the next highest-leverage social-loop slice. A derived feed is appropriately sized for MVP, aligns with existing AppView indexed-read architecture, and avoids prematurely designing push/unread/persistence mechanics.

## 6. Problem / Opportunity

Users can now create and receive social activity, but inbound activity is not visible in one place. This weakens the social feedback loop because users must manually revisit posts and profiles to notice replies, likes, reposts, and follows.

## 7. Goals

- G-001: Let signed-in users view a chronological list of social activity directed at them.
- G-002: Reuse existing AppView indexed data and Flutter pagination patterns.
- G-003: Keep the slice small enough to validate notification semantics before adding push, unread, grouping, or persistence.
- G-004: Maintain Craftsky architectural boundaries: reads from AppView, no PDS tokens in Flutter, no direct PDS reads in the app.

## 8. Non-Goals

- NG-001: Push notification delivery or device registration.
- NG-002: Unread/read state, badges, mark-all-read, or notification preferences.
- NG-003: Notification grouping or aggregation copy such as “Alice and 3 others liked your post.”
- NG-004: Persisted notification records or notification indexer fan-out.
- NG-005: New lexicons, new PDS writes, or changes to existing atproto record schemas.
- NG-006: Muting, blocking, reports, moderation filtering, or safety tooling.
- NG-007: Search, rich text rendering, quote-post notifications, or algorithmic ranking.
- NG-008: Durable history for inactive likes/reposts after unlike/unrepost.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Signed-in Craftsky user | A user with a Craftsky app session. | See follows, likes, reposts, and replies directed at them. |
| Notification actor | Another account whose action creates activity directed at the signed-in user. | Be shown with enough identity context for the viewer to understand the activity. |
| AppView | Server-side read model and API. | Derive notification rows from indexed data without PDS read-through. |
| Flutter app | Mobile client. | Fetch, paginate, display, and navigate from notifications using existing AppView API patterns. |

## 10. Current Behavior

The AppView has no notifications route. The Flutter Notifications tab is a placeholder that renders an app bar titled `Notifications` and a centered static `Notifications` label. Existing social activity is visible only in local contexts such as timeline rows, post threads, and profiles.

## 11. Desired Behavior

The AppView exposes an authenticated `GET /v1/notifications` endpoint that returns a reverse-chronological, cursor-paginated list of notifications derived from indexed follows, likes, reposts, and replies directed at the authenticated user. The Flutter Notifications page consumes this endpoint, renders loading/empty/error/list/pagination states, and lets users navigate from rows to relevant profiles or post threads.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | Signed-in users shall be able to see social activity directed at them in the Notifications tab. | Completes the social feedback loop after timeline, posting, replies, likes, reposts, and follows. | Prompt / approved plan | AC-001, AC-010, AC-011 |
| BR-002 | Business | Must | The notifications MVP shall remain read-only and derived from existing indexed AppView data. | Keeps the slice small and avoids premature push/unread/persistence design. | Approved plan | AC-002, AC-003, AC-006 |
| FR-001 | Functional | Must | The AppView shall expose `GET /v1/notifications` under the existing `/v1/` authenticated app API. | Roadmap and API architecture identify this as the notifications endpoint. | API architecture / roadmap | AC-001, AC-002, AC-014 |
| FR-002 | Functional | Must | The AppView shall include follow notifications where another account follows the authenticated user. | Follows are an existing social action directed at the viewer. | Discovery | AC-003 |
| FR-003 | Functional | Must | The AppView shall include like notifications where another account actively likes a post authored by the authenticated user. | Likes are an existing social action directed at viewer-authored content. | Discovery | AC-004, AC-006 |
| FR-004 | Functional | Must | The AppView shall include repost notifications where another account actively reposts a post authored by the authenticated user. | Reposts are an existing social action directed at viewer-authored content. | Discovery | AC-005, AC-006 |
| FR-005 | Functional | Must | The AppView shall include reply notifications where another account authors a reply whose parent post is authored by the authenticated user. | Replies are the highest-value conversation notification after thread support. | Discovery | AC-007 |
| FR-006 | Functional | Must | The AppView shall exclude self-generated notifications. | Users should not be notified about their own follows, likes, reposts, or replies. | Approved plan | AC-008 |
| FR-007 | Functional | Must | The AppView shall return notification items in reverse chronological order using indexed event time, with deterministic tie-breaking. | The product principle favors chronological feeds and the API must paginate predictably. | AGENTS / discovery | AC-009 |
| FR-008 | Functional | Must | The AppView shall support opaque cursor pagination with `limit` and `cursor`, including omitting `cursor` entirely when no more results exist. | Matches existing `/v1/*` list contract and prevents client cursor coupling. | API architecture / timeline lessons | AC-009, AC-015, AC-016 |
| FR-009 | Functional | Must | Each notification item shall identify its type, actor, event timestamps, and relevant subject data needed for display and navigation. | Flutter needs enough data to render rows and route users without N+1 API calls. | Approved plan | AC-010, AC-011, AC-012 |
| FR-010 | Functional | Must | Like, repost, and reply notifications shall include the viewer-authored subject post using the existing post response semantics where practical. | Reusing post hydration keeps wire shape consistent and supports navigation to threads. | Discovery | AC-004, AC-005, AC-007, AC-012 |
| FR-011 | Functional | Must | Reply notifications shall include enough reply identity to focus or identify the reply from the subject thread. | Users should land in the relevant conversation context. | Approved plan | AC-007, AC-011 |
| FR-012 | Functional | Must | The Flutter data layer shall add notification models and API/repository methods for `GET /v1/notifications`. | The UI must consume the new endpoint through existing app architecture. | Discovery | AC-012 |
| FR-013 | Functional | Must | The Flutter provider/state layer shall load the first notification page, append additional pages, preserve existing items on load-more failure, and guard concurrent/terminal load-more calls. | Matches timeline pagination behavior and protects user-visible state. | Existing Flutter patterns | AC-013, AC-015, AC-016 |
| FR-014 | Functional | Must | The Flutter Notifications page shall render loading, empty, initial error/retry, loaded mixed rows, load-more progress, and load-more retry states. | Replaces the placeholder with a usable screen. | Approved plan | AC-010, AC-013, AC-016 |
| FR-015 | Functional | Must | Notification rows shall navigate to relevant destinations: actor profile for follows, subject post thread for likes/reposts, and subject thread focused on the reply when possible. | Notifications should be actionable. | Approved plan | AC-011 |
| NFR-001 | Non-functional | Must | The notification read path shall not read craft data directly from a PDS. | Preserves AppView read architecture and keeps Flutter away from PDS tokens. | AGENTS architectural rules | AC-017 |
| NFR-002 | Non-functional | Should | The implementation should reuse existing pagination, error envelope, JSON casing, model mapping, and visual patterns. | Reduces inconsistency and test burden. | Existing patterns | AC-012, AC-013, AC-014 |
| NFR-003 | Non-functional | Should | The derived query should remain bounded by endpoint limit caps and avoid unnecessary N+1 profile/post lookups. | Notifications may become a high-traffic screen. | Discovery risk | AC-009, AC-018 |
| RULE-001 | Business rule | Must | Active-only interaction policy: likes and reposts shall appear only while the indexed interaction row is active. | Approved MVP tradeoff avoids persisted notification history. | Approved plan | AC-006 |
| RULE-002 | Business rule | Must | Notification responses shall be scoped to the authenticated viewer DID from the Craftsky session, not to a request-supplied DID. | Prevents users from querying other users' notification feeds. | API auth model | AC-001, AC-019 |
| RULE-003 | Business rule | Must | Unknown query parameters shall not change notification selection semantics. | Aligns with existing API handler behavior for list endpoints. | Existing API patterns | AC-020 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-001, RULE-002 | Given an authenticated request with a valid device ID, when `GET /v1/notifications` is called, then the response contains only notifications directed at the authenticated viewer. |
| AC-002 | BR-002, FR-001 | Given no notification table exists for this MVP, when notifications are requested, then the AppView derives rows from existing indexed follow, like, repost, and post tables. |
| AC-003 | BR-002, FR-002 | Given Alice follows the viewer and Bob follows someone else, when the viewer requests notifications, then Alice's follow is included and Bob's follow is excluded. |
| AC-004 | FR-003, FR-010 | Given Alice actively likes a viewer-authored post, when the viewer requests notifications, then a `like` notification is returned with Alice as actor and the liked post as subject. |
| AC-005 | FR-004, FR-010 | Given Alice actively reposts a viewer-authored post, when the viewer requests notifications, then a `repost` notification is returned with Alice as actor and the reposted post as subject. |
| AC-006 | BR-002, FR-003, FR-004, RULE-001 | Given a like or repost row has `deleted_at` set, when the viewer requests notifications, then that inactive interaction does not appear. |
| AC-007 | FR-005, FR-010, FR-011 | Given Alice replies to a viewer-authored parent post, when the viewer requests notifications, then a `reply` notification is returned with Alice as actor, the parent post as subject, and reply identity for focus/navigation. |
| AC-008 | FR-006 | Given the viewer follows, likes, reposts, or replies to their own content, when the viewer requests notifications, then no self-generated notification is returned. |
| AC-009 | FR-007, FR-008, NFR-003 | Given mixed notification types with different indexed times and ties, when notifications are listed across pages, then rows are ordered newest-first with deterministic tie-breaking and no duplicates or skips. |
| AC-010 | BR-001, FR-009, FR-014 | Given the Flutter Notifications page loads a mixed notification page, then it renders understandable rows for follow, like, repost, and reply notifications. |
| AC-011 | BR-001, FR-011, FR-015 | Given a user taps a notification row, then follow rows navigate to the actor profile, like/repost rows navigate to the subject thread, and reply rows navigate to the subject thread focused on the reply when focus data is available. |
| AC-012 | FR-009, FR-010, FR-012, NFR-002 | Given the Flutter API client receives a valid notification response, then it decodes all notification item types and relevant nested actor/post data without inspecting opaque cursors. |
| AC-013 | FR-013, FR-014, NFR-002 | Given initial notification loading fails, when the user retries, then the provider/page requests the first page again and renders the resulting state. |
| AC-014 | FR-001, NFR-002 | Given a request lacks authentication or a required device ID, when `GET /v1/notifications` is called, then existing middleware returns the standard unauthorized or missing-device response. |
| AC-015 | FR-008, FR-013 | Given the first page response includes a cursor, when Flutter requests more, then it sends the opaque cursor unchanged and appends the next page in order. |
| AC-016 | FR-008, FR-013, FR-014 | Given load-more fails after existing items are rendered, then existing rows remain visible, a retry affordance is shown, and concurrent load-more requests are not issued. |
| AC-017 | NFR-001 | Given notifications are requested, then the AppView serves them from indexed AppView data and the Flutter app does not call a PDS directly. |
| AC-018 | NFR-003 | Given a notification page is requested with a high or invalid limit, then the handler applies documented defaults/caps and returns bounded results. |
| AC-019 | RULE-002 | Given a malicious client attempts to include another DID in query parameters, when notifications are requested, then notification scope remains the authenticated viewer. |
| AC-020 | RULE-003 | Given unknown query parameters are present, when notifications are requested, then they are ignored and do not alter selection or ordering. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | User has no notifications. | AppView returns an empty `items` list with no cursor; Flutter renders an empty state. | BR-001, FR-014 |
| EC-002 | Final page exactly fills requested limit. | AppView omits `cursor` by detecting whether an additional row exists, not by checking page length alone. | FR-008 |
| EC-003 | Actor has no hydrated Bluesky display profile. | Notification still renders with DID/handle fallback if resolvable through existing handle resolution behavior. | FR-009 |
| EC-004 | Subject post is no longer indexed or cannot be joined. | Notification is omitted or handled as unavailable according to implementation choice documented in test design; it must not crash the endpoint. | FR-010 |
| EC-005 | Multiple events have identical indexed timestamps. | Deterministic tie-breaker yields stable pagination. | FR-007, FR-008 |
| EC-006 | User taps a reply notification whose focus cannot be represented. | Flutter opens the subject thread without focus rather than failing navigation. | FR-015 |
| EC-007 | Invalid cursor. | AppView returns a standard `400 invalid_cursor` style response consistent with existing list endpoints. | FR-008, NFR-002 |
| EC-008 | Load-more returns an empty page. | Flutter clears terminal cursor state and keeps existing items. | FR-013 |

## 15. Data / Persistence Impact

- New fields: None required for MVP persistence.
- Changed fields: None.
- Migration required: No, unless implementation proves an index is necessary for acceptable query performance.
- Backwards compatibility: Additive API and Flutter UI change; no breaking changes to existing endpoints expected.
- Data source policy: Notifications are derived from `atproto_follows`, `craftsky_likes`, `craftsky_reposts`, and `craftsky_posts`.

## 16. UI / API / CLI Impact

- UI:
  - Replace `NotificationsPage` placeholder with a real paginated notifications screen.
  - Add localized or otherwise testable copy for loading, empty, error, row labels, and retry states according to existing Flutter localization practices.
- API:
  - Add `GET /v1/notifications`.
  - Response uses camelCase JSON and opaque pagination cursor semantics.
  - Errors use the existing `{error, message, requestId}` envelope through existing helpers/middleware.
- CLI: None.
- Background jobs: None.

## 17. Security / Privacy / Permissions

- Authentication: `GET /v1/notifications` requires an authenticated Craftsky session.
- Authorization: Notification scope is always the authenticated viewer DID; clients cannot request another user's notification feed.
- Sensitive data: This slice returns activity derived from public atproto records already indexed by the AppView. It must not expose private server-side state, OAuth tokens, or PDS credentials.
- Abuse cases:
  - Unknown query parameters and request-supplied DIDs must not bypass viewer scoping.
  - Self-notifications are excluded to avoid misleading activity.
  - Push/spam/rate limiting is out of scope, but endpoint should remain bounded by limit caps.

## 18. Observability

- Events: None required.
- Logs: Use existing request logging/error logging patterns if present; avoid logging sensitive session tokens.
- Metrics: None required for MVP.
- Alerts: None required for MVP.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Unioning multiple event sources may create ordering/pagination bugs. | Users could see duplicate, skipped, or unstable notifications. | Require mixed-type pagination tests, deterministic tie-breaker tests, and exact-full final page tests. |
| RISK-002 | Active-only like/repost policy may surprise users if notifications disappear after unlike/unrepost. | MVP behavior differs from most persisted notification systems. | Record policy explicitly as `RULE-001`; revisit with persisted notifications if user feedback demands history. |
| RISK-003 | Derived query may become expensive as activity grows. | Notifications screen could become slow. | Keep page sizes capped, test query shape, and add indexes in a later focused performance slice if needed. |
| RISK-004 | Flutter navigation from notification rows may conflict with existing typed routes or thread focus semantics. | Taps may route incorrectly or fail. | Cover row navigation with widget/router tests for each notification type. |
| RISK-005 | Missing actor/profile hydration could degrade row display. | Rows could show poor or broken identity text. | Require fallback display behavior and tests for missing optional profile fields. |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | Derived active-only notifications are acceptable for MVP. | A persisted notification table/indexer slice would be needed before implementation. |
| ASM-002 | Existing indexed tables contain enough data to identify actors, subjects, and reply focus targets. | API shape may need additional nested fields or joins. |
| ASM-003 | Existing Flutter post-thread/profile routes can support the required notification navigation. | Scope may need route adjustments. |
| ASM-004 | No moderation/block/mute filtering is required for this MVP. | Notification selection would need additional policy and data dependencies. |
| ASM-005 | The current timeline implementation and review fixes are treated as complete enough to build from. | If timeline remains unstable, stabilization should precede notifications. |

## 21. Open Questions

- [ ] Non-blocking: What exact empty-state copy should the Notifications page use?
- [ ] Non-blocking: Should reply notifications target only direct replies to viewer-authored posts, or also deeper descendants in threads authored by others? MVP requirements use direct parent authored by viewer.
- [ ] Non-blocking: Should inactive follow/unfollow history ever produce notifications? MVP includes only currently indexed follows because `atproto_follows` represents active graph state.

## 22. Review Status

Status: Approved direction / requirements draft

Risk level: Medium

Review recommended: Yes

Reviewer: User direction approved via plan review; formal requirements review pending user choice.

Date: 2026-05-29

Notes: This is a user-visible API + Flutter feature touching AppView queries, route auth, Dart data models, provider pagination, and UI navigation. Review is recommended before test design, but not required by risk policy.

## 23. Handoff To Test Design

- Requirements file: `docs/changes/2026-05-29-notifications-mvp/01-requirements.md`
- Next test specification: `02-acceptance-tests.md`
- Must-cover requirement IDs:
  - `BR-001`, `BR-002`
  - `FR-001` through `FR-015`
  - `NFR-001`
  - `RULE-001` through `RULE-003`
- Suggested test levels:
  - AppView store tests for notification derivation, scoping, event types, self-exclusion, active-only policy, ordering, pagination, invalid cursor, and limit handling.
  - AppView handler tests for JSON response shape, auth/device middleware behavior, unknown query handling, and error envelope behavior.
  - AppView route tests for route registration and protected access.
  - Flutter API client/model tests for decoding mixed notification types and cursor handling.
  - Flutter provider tests for initial load, empty state, pagination, terminal cursor, load-more failure, retry, and concurrent guard behavior.
  - Flutter widget/router tests for loading, empty, initial error, mixed rows, row navigation, load-more progress/retry, and fallback actor display.
- Blocking open questions: None.
- Review recommendation before test design: Recommended due to medium risk and new user-visible API/UI surface.
