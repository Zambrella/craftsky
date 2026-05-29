# Acceptance Test Specification: Notifications MVP

## 1. Test Strategy

The Notifications MVP should be driven test-first from the AppView read model outward to the Flutter screen. The highest-risk behavior is the derived, heterogeneous notification feed: selection must be scoped to the authenticated viewer, assembled from existing indexed AppView data, ordered deterministically, and paginated without duplicates or skips. AppView store tests should therefore lead implementation, followed by handler/route contract tests, then Flutter API/model/provider tests, and finally widget/router tests for user-visible states and navigation.

Risk level carried forward from requirements: **Medium**. Review is recommended before implementation because this adds a new authenticated API, new Dart models/state, and a user-visible Notifications tab.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-010, AC-011 | AT-001, AT-002, AT-003, IT-001, IT-008, UT-010, UT-014, UT-015, MAN-001 | Acceptance / Integration / Unit / Manual | Yes, except MAN-001 |
| BR-002 | AC-002, AC-003, AC-006 | AT-001, AT-006, IT-001, IT-002, IT-003, IT-006, REG-001 | Acceptance / Integration / Regression | Yes |
| FR-001 | AC-001, AC-002, AC-014 | AT-001, IT-008, IT-010, UT-008, REG-002 | Acceptance / Integration / Unit / Regression | Yes |
| FR-002 | AC-003 | AT-001, IT-001 | Acceptance / Integration | Yes |
| FR-003 | AC-004, AC-006 | AT-001, IT-002, IT-006 | Acceptance / Integration | Yes |
| FR-004 | AC-005, AC-006 | AT-001, IT-003, IT-006 | Acceptance / Integration | Yes |
| FR-005 | AC-007 | AT-001, IT-004, UT-015 | Acceptance / Integration / Unit | Yes |
| FR-006 | AC-008 | IT-005 | Integration | Yes |
| FR-007 | AC-009 | IT-007, IT-008 | Integration | Yes |
| FR-008 | AC-009, AC-015, AC-016, AC-018 | AT-005, IT-008, IT-009, IT-011, UT-001, UT-002, UT-003, UT-004, UT-007, UT-012, UT-013 | Acceptance / Integration / Unit | Yes |
| FR-009 | AC-010, AC-011, AC-012 | AT-002, AT-003, IT-001, IT-002, IT-003, IT-004, UT-009, UT-014, MAN-001 | Acceptance / Integration / Unit / Manual | Yes, except MAN-001 |
| FR-010 | AC-004, AC-005, AC-007, AC-012 | IT-002, IT-003, IT-004, UT-009 | Integration / Unit | Yes |
| FR-011 | AC-007, AC-011 | AT-003, IT-004, UT-015 | Acceptance / Integration / Unit | Yes |
| FR-012 | AC-012 | UT-009, UT-010 | Unit | Yes |
| FR-013 | AC-013, AC-015, AC-016 | AT-004, AT-005, UT-011, UT-012, UT-013 | Acceptance / Unit | Yes |
| FR-014 | AC-010, AC-013, AC-016 | AT-002, AT-004, AT-005, UT-014 | Acceptance / Unit | Yes |
| FR-015 | AC-011 | AT-003, UT-015, MAN-001 | Acceptance / Unit / Manual | Yes, except MAN-001 |
| NFR-001 | AC-017 | AT-006, REG-001, MAN-002 | Acceptance / Regression / Manual | Yes, except MAN-002 |
| RULE-001 | AC-006 | IT-006, REG-003 | Integration / Regression | Yes |
| RULE-002 | AC-001, AC-019 | AT-001, IT-001, IT-012 | Acceptance / Integration | Yes |
| RULE-003 | AC-020 | IT-011, UT-005 | Integration / Unit | Yes |

## 3. Acceptance Scenarios

### AT-001: Authenticated User Sees Only Directed Notifications

Requirement IDs: BR-001, BR-002, FR-001, FR-002, FR-003, FR-004, FR-005, RULE-002  
Acceptance Criteria: AC-001, AC-002, AC-003, AC-004, AC-005, AC-007, AC-019  
Priority: Must  
Level: Acceptance  
Automation Target: `appview/internal/api/notification_store_test.go`, `appview/internal/api/notifications_test.go`

```gherkin
Feature: Notifications feed
  Scenario: Viewer receives directed social activity only
    Given the AppView has indexed a follow of the viewer
    And the AppView has indexed active likes, reposts, and direct replies against viewer-authored posts
    And the AppView has indexed similar activity directed at another account
    When the authenticated viewer requests GET /v1/notifications
    Then the response includes follow, like, repost, and reply notifications directed at the viewer
    And the response excludes activity directed at other accounts
    And adding a request-supplied DID query parameter does not change the viewer scope
```

### AT-002: Notifications Page Renders Mixed Rows

Requirement IDs: BR-001, FR-009, FR-014  
Acceptance Criteria: AC-010  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/notifications/notifications_page_test.dart`

```gherkin
Feature: Notifications tab
  Scenario: Mixed notification rows are understandable
    Given the Notifications page provider returns follow, like, repost, and reply notifications
    When the signed-in user opens the Notifications tab
    Then the page shows a Notifications title
    And each row displays actor identity fallback-safe copy
    And like, repost, and reply rows include enough subject context to understand the activity
```

### AT-003: Notification Rows Navigate To Relevant Destinations

Requirement IDs: BR-001, FR-011, FR-015  
Acceptance Criteria: AC-011  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/notifications/notifications_page_test.dart`, `app/test/router/router_redirect_test.dart`

```gherkin
Feature: Notification navigation
  Scenario: User taps notification rows
    Given the Notifications page renders follow, like, repost, and reply notifications
    When the user taps a follow notification
    Then the app navigates to the actor profile
    When the user taps a like or repost notification
    Then the app navigates to the subject post thread
    When the user taps a reply notification with focus data
    Then the app navigates to the subject thread focused on the reply
    When reply focus data cannot be represented
    Then the app opens the subject thread without failing navigation
```

### AT-004: Initial Error Can Be Retried

Requirement IDs: FR-013, FR-014  
Acceptance Criteria: AC-013  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/notifications/notifications_page_test.dart`, `app/test/notifications/providers/notifications_provider_test.dart`

```gherkin
Feature: Notifications loading
  Scenario: Initial load failure recovers after retry
    Given the first notifications request fails
    When the user opens the Notifications tab
    Then the page shows an initial error state with a retry affordance
    When the user retries and the request succeeds
    Then the page renders the returned notifications
```

### AT-005: Load More Preserves Existing Rows On Failure

Requirement IDs: FR-008, FR-013, FR-014  
Acceptance Criteria: AC-015, AC-016  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/notifications/notifications_page_test.dart`, `app/test/notifications/providers/notifications_provider_test.dart`

```gherkin
Feature: Notifications pagination
  Scenario: Loading the next page fails after rows are visible
    Given the first notifications page is visible and includes an opaque cursor
    When the user reaches the load-more trigger and the next request fails
    Then the existing rows remain visible
    And a load-more retry affordance is shown
    And concurrent load-more requests are not issued
    When the user retries load-more successfully
    Then the next page is appended in order using the original opaque cursor unchanged
```

### AT-006: Notifications Stay Read-Only And AppView-Sourced

Requirement IDs: BR-002, NFR-001  
Acceptance Criteria: AC-002, AC-017  
Priority: Must  
Level: Acceptance  
Automation Target: `appview/internal/api/notification_store_test.go`, `app/test/notifications/data/notification_api_client_test.dart`, code review checklist

```gherkin
Feature: AppView read architecture
  Scenario: Notifications are derived from indexed AppView data
    Given no notification table, PDS write path, or Flutter PDS client is added for this MVP
    When notifications are requested by the AppView or Flutter app
    Then the AppView derives rows from existing indexed follow, like, repost, and post tables
    And the Flutter app calls only GET /v1/notifications through the AppView API client
```

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-008 | AC-009, AC-015 | Encode/decode notification opaque seek cursor without exposing cursor internals to callers. | Indexed timestamp plus deterministic tie-break fields. | Decoding returns the original seek tuple; malformed values return the standard invalid cursor error. | `appview/internal/api/envelope/cursor_test.go` or notification cursor helper tests |
| UT-002 | FR-008 | AC-018 | Parse notification limit defaults. | Missing `limit`. | Handler/store receives documented default page size, expected to mirror timeline default `20` unless implementation documents otherwise. | `appview/internal/api/notifications_test.go` |
| UT-003 | FR-008 | AC-018 | Cap high notification limits. | `limit=999`. | Handler/store receives documented cap, expected to mirror timeline cap `50` unless implementation documents otherwise. | `appview/internal/api/notifications_test.go` |
| UT-004 | FR-008 | AC-018 | Treat invalid notification limits as default. | `limit=abc`, `limit=0`, negative limit if accepted by parser. | Handler/store receives default bounded limit and does not return unbounded rows. | `appview/internal/api/notifications_test.go` |
| UT-005 | RULE-003 | AC-020 | Unknown query parameters do not affect handler inputs. | `?foo=bar&actorDid=did:plc:other&limit=2`. | Handler/store receives only authenticated viewer DID, parsed limit, and cursor; unknown params are ignored. | `appview/internal/api/notifications_test.go` |
| UT-006 | FR-009 | AC-010, AC-012 | Serialize notification page shape with camelCase fields. | Fake follow, like, repost, reply rows and `next-cursor`. | JSON contains `items`, optional `cursor`, notification `type`, `actor`, event timestamps, subject fields, and no `totalCount`. | `appview/internal/api/notifications_test.go` |
| UT-007 | FR-008, NFR-002 | AC-014 | Invalid cursor maps to standard API error envelope. | Store returns `envelope.ErrInvalidCursor`. | HTTP status `400`; envelope contains `error: invalid_cursor`, non-empty `message`, and `requestId` if standard helper emits one. | `appview/internal/api/notifications_test.go` |
| UT-008 | FR-001, NFR-002 | AC-014 | Store failure maps to standard server error envelope. | Store returns unexpected error. | HTTP status is standard 5xx envelope, no raw database details leak. | `appview/internal/api/notifications_test.go` |
| UT-009 | FR-009, FR-010, FR-012 | AC-012 | Decode mixed notification item types in Flutter models. | JSON page containing `follow`, `like`, `repost`, and `reply` items with nested actor/post/reply identity. | Dart model maps every item type, nested actor and subject post data, event timestamps, and optional cursor correctly. | `app/test/notifications/models/notification_test.dart` |
| UT-010 | BR-001, FR-012 | AC-012 | Flutter API client requests notifications endpoint and passes cursor opaquely. | `listNotifications(limit: 20, cursor: 'opaque:abc')`. | Client sends `GET /v1/notifications?limit=20&cursor=opaque%3Aabc` or equivalent query encoding and decodes response. | `app/test/notifications/data/notification_api_client_test.dart` |
| UT-011 | FR-013, FR-014 | AC-013 | Provider initial load retries after failure. | Fake repository throws, then returns page. | Initial state is error; retry fetches first page again and transitions to loaded state. | `app/test/notifications/providers/notifications_provider_test.dart` |
| UT-012 | FR-008, FR-013 | AC-015, AC-016 | Provider appends next page and marks terminal cursor. | First page has cursor; second page has no cursor or empty items. | Provider sends cursor unchanged, appends items in order, and `hasMore` becomes false when cursor is absent. | `app/test/notifications/providers/notifications_provider_test.dart` |
| UT-013 | FR-008, FR-013, FR-014 | AC-016 | Provider preserves visible rows and cursor on load-more failure and guards concurrency. | First page loaded; second request throws; duplicate `loadMore()` call while in flight. | Existing items remain, retry uses preserved cursor, and only one in-flight request is issued. | `app/test/notifications/providers/notifications_provider_test.dart` |
| UT-014 | BR-001, FR-009, FR-014 | AC-010 | Notifications page renders loading, empty, loaded, initial error, and load-more states. | Provider overrides for each state. | Expected progress indicator, empty copy, row copy, retry affordance, load-more progress, and load-more retry are visible. | `app/test/notifications/notifications_page_test.dart` |
| UT-015 | FR-011, FR-015 | AC-011 | Notification row tap maps to the correct route intent. | Follow, like, repost, reply with focus, reply without focus. | Follow routes to actor profile; like/repost route to subject thread; reply routes to focused thread when possible, otherwise subject thread. | `app/test/notifications/notifications_page_test.dart` or route-focused widget tests |
| UT-016 | FR-009 | AC-010 | Actor display falls back safely when optional profile fields are missing. | Actor with DID only, actor with handle only, actor with display name/avatar. | Row still renders stable identity text and does not crash. | `app/test/notifications/notifications_page_test.dart`, `app/test/notifications/models/notification_test.dart` |

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | BR-001, BR-002, FR-001, FR-002, FR-009, RULE-002 | AC-001, AC-002, AC-003, AC-019 | Store derives follow notifications scoped to authenticated viewer. | Seed `atproto_follows` for Alice -> viewer and Bob -> someone else; seed actor profiles where useful. | Call notification store for viewer DID with no cursor. | Includes Alice follow notification, excludes Bob follow, uses authenticated viewer only. | `appview/internal/api/notification_store_test.go` |
| IT-002 | BR-002, FR-003, FR-009, FR-010 | AC-004 | Store derives active like notifications for viewer-authored posts. | Seed viewer post; seed active Alice like; seed like of another author's post. | List notifications for viewer. | Includes `like` item with Alice as actor and viewer post as subject; excludes other-author subject. | `appview/internal/api/notification_store_test.go` |
| IT-003 | BR-002, FR-004, FR-009, FR-010 | AC-005 | Store derives active repost notifications for viewer-authored posts. | Seed viewer post; seed active Alice repost; seed repost of another author's post. | List notifications for viewer. | Includes `repost` item with Alice as actor and subject post; excludes unrelated repost. | `appview/internal/api/notification_store_test.go` |
| IT-004 | FR-005, FR-009, FR-010, FR-011 | AC-007 | Store derives direct reply notifications with reply focus identity. | Seed viewer parent post; seed Alice direct reply whose parent is viewer post; seed nested/deeper case if implementation intentionally excludes it. | List notifications for viewer. | Includes `reply` item with Alice actor, parent/subject post, reply URI/rkey/CID or focus fields; excludes out-of-scope deeper replies unless documented otherwise. | `appview/internal/api/notification_store_test.go` |
| IT-005 | FR-006 | AC-008 | Store excludes self-generated notifications. | Seed viewer self-follow, self-like, self-repost, and self-reply against viewer content. | List notifications for viewer. | No self-generated notification rows are returned. | `appview/internal/api/notification_store_test.go` |
| IT-006 | BR-002, FR-003, FR-004, RULE-001 | AC-006 | Active-only policy excludes deleted likes/reposts. | Seed like/repost rows with `deleted_at` set plus active rows. | List notifications for viewer. | Deleted like/repost rows are absent; active rows remain present. | `appview/internal/api/notification_store_test.go` |
| IT-007 | FR-007, FR-008 | AC-009 | Mixed notification types sort newest-first with deterministic tie-break. | Seed follow, like, repost, and reply rows with staggered and tied indexed/event timestamps. | List notifications with large limit. | Rows are newest-first; ties are stable according to documented tie-break fields. | `appview/internal/api/notification_store_test.go` |
| IT-008 | BR-001, FR-007, FR-008 | AC-009 | Opaque pagination across mixed notification types has no duplicates or skips. | Seed at least five mixed rows including timestamp ties. | Fetch pages with limit 2 until terminal. | Combined pages equal full ordered list; cursors are non-empty only while more rows exist; no duplicates/skips. | `appview/internal/api/notification_store_test.go` |
| IT-009 | FR-008 | AC-009 | Exact-full final page omits cursor by checking one additional row. | Seed exactly `limit` notification rows. | List notifications with that limit. | Returns all rows and omits `cursor`. | `appview/internal/api/notification_store_test.go` |
| IT-010 | FR-001, NFR-002 | AC-014 | Route registration protects `GET /v1/notifications` with auth/device middleware. | Build routes with test deps. | Request without auth; request with auth but no `X-Craftsky-Device-Id`. | Missing auth returns unauthorized; missing device returns standard missing-device response. | `appview/internal/routes/routes_test.go` |
| IT-011 | FR-008, RULE-003 | AC-018, AC-020 | Handler ignores unknown query parameters and applies default/capped limits. | Fake notification store records viewer DID, limit, cursor. | Request `GET /v1/notifications` with unknown params, invalid limit, and high limit variants. | Unknown params do not alter selection; invalid/high limits are bounded; cursor is passed only from `cursor`. | `appview/internal/api/notifications_test.go` |
| IT-012 | RULE-002 | AC-001, AC-019 | Handler scopes to session viewer rather than request-supplied DID. | Fake notification store records viewer DID from request context. | Authenticated request includes `?did=did:plc:other&viewerDid=did:plc:other`. | Store is called with authenticated DID from context, not query values. | `appview/internal/api/notifications_test.go` |
| IT-013 | FR-009, FR-010 | AC-012 | Unavailable subject post does not crash endpoint. | Seed a like/repost/reply whose subject join cannot be hydrated, if schema allows. | List notifications. | Endpoint succeeds and either omits unavailable notification or returns documented unavailable-safe shape. | `appview/internal/api/notification_store_test.go` |

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Acceptance Criteria | Test |
|---|---|---|---|---|
| REG-001 | Flutter app reads craft data through AppView, not PDS, and does not hold PDS tokens. | BR-002, NFR-001 | AC-002, AC-017 | Verify new notification repository/client depends on existing AppView Dio/session stack only; no PDS client or token storage is introduced for notifications. |
| REG-002 | `/v1/*` protected route behavior requires bearer session and device ID. | FR-001 | AC-014 | Add route tests mirroring timeline/profile route protection for `/v1/notifications`. |
| REG-003 | Active-only like/repost semantics remain aligned with indexed interaction deletion behavior. | RULE-001 | AC-006 | Store tests seed `deleted_at` interactions and assert absence from notifications. |
| REG-004 | Existing timeline, profile, post, and error-envelope tests continue to pass after adding notification query/route code. | FR-001, FR-008, NFR-002 | AC-014, AC-015, AC-018 | Run focused AppView package tests and avoid changing existing endpoint JSON/cursor behavior. |
| REG-005 | Existing shell routing keeps the Notifications tab reachable. | BR-001, FR-014 | AC-010, AC-013 | Existing app shell/router widget tests plus updated notifications page tests still find the Notifications branch/title. |

## 7. Test Data

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Authenticated viewer identity | `did:plc:viewer`, handle `viewer.craftsky.social`, valid Craftsky session context, `X-Craftsky-Device-Id: dev-test`. | AT-001, IT-001, IT-010, IT-012, UT-010 |
| TD-002 | Notification actors | Alice `did:plc:alice`, Bob `did:plc:bob`, Carol `did:plc:carol`; include display name/avatar for Alice and missing optional profile fields for Bob/Carol. | AT-002, IT-001-IT-007, UT-009, UT-016 |
| TD-003 | Viewer-authored subject posts | Root post URI `at://did:plc:viewer/social.craftsky.feed.post/root`, CID, rkey, text, created/indexed timestamps, optional images/tags as existing `PostResponse` supports. | IT-002, IT-003, IT-004, UT-009, UT-015 |
| TD-004 | Follow rows | Alice follows viewer; Bob follows someone else; viewer follows viewer for self-exclusion case. | IT-001, IT-005 |
| TD-005 | Like/repost rows | Active rows for Alice -> viewer post; active rows against other authors; rows with `deleted_at` set; self-like/self-repost rows. | IT-002, IT-003, IT-005, IT-006 |
| TD-006 | Reply rows | Alice direct reply with root/parent pointing to viewer post; reply focus identity fields; self-reply; optional deeper descendant for exclusion/fallback. | IT-004, IT-005, UT-009, UT-015 |
| TD-007 | Ordering and pagination rows | Mixed follow/like/repost/reply events with indexed times: newest, tied-high, tied-low, older, oldest. | IT-007, IT-008, IT-009 |
| TD-008 | Flutter notification JSON page | `items` array with one each of `follow`, `like`, `repost`, `reply`, nested `actor`, nested subject `post`, reply focus fields, and `cursor: opaque-next`. | UT-009, UT-010, UT-014, AT-002 |
| TD-009 | Error and empty states | Empty `items` with no `cursor`; API error envelope for invalid cursor; thrown repository exceptions for network/server failures. | AT-004, AT-005, UT-007, UT-011, UT-013, UT-014 |

## 8. Manual Checks

| ID | Requirement IDs | Acceptance Criteria | Check | Steps | Expected Result |
|---|---|---|---|---|---|
| MAN-001 | BR-001, FR-009, FR-015 | AC-010, AC-011 | Human review of notification row copy and navigation feel. | Run the Flutter app with seeded activity; open Notifications; inspect follow/like/repost/reply rows; tap each row type. | Copy is understandable, fallback identity is acceptable, and navigation destinations match requirements. |
| MAN-002 | NFR-001 | AC-017 | Architecture review for no direct PDS reads/tokens in Flutter notification path. | Review notification Dart data layer and AppView handler/store code before implementation merge. | Flutter calls only AppView `/v1/notifications`; no PDS tokens, PDS clients, PDS writes, or new lexicons are introduced. |

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | No dedicated load/performance benchmark for the derived union query. | NFR-003 | Performance is a Should requirement and exact data volumes are not defined for MVP. | Keep endpoint limit-capped; add query plan/performance tests in a later scale-focused slice if needed. |
| GAP-002 | Exact unavailable-subject behavior is intentionally implementation-defined. | FR-010 | Requirements allow omission or unavailable-safe handling as long as endpoint does not crash. | Implementation must document its choice and cover it with IT-013. |
| GAP-003 | Manual visual review remains useful for copy quality and navigation feel. | BR-001, FR-014, FR-015 | Widget tests verify behavior, but product copy/readability benefits from human review. | Complete MAN-001 before release or during document review if UI copy is debated. |

## 10. Out Of Scope

- Push notification delivery, device registration, push permission prompts, and push token tests.
- Unread/read state, badges, mark-all-read, notification preferences, and read receipts.
- Notification grouping/aggregation tests such as “Alice and 3 others liked your post.”
- Persisted notification table/indexer fan-out/idempotency tests.
- New lexicon validation tests or PDS write-path tests.
- Moderation, blocking, muting, report, and safety filtering tests.
- Search, rich text rendering, quote-post notifications, algorithmic ranking, and durable history for inactive likes/reposts.
- Full performance/load testing beyond bounded limit/cursor behavior in this MVP.

## 11. Handoff To Document Review

- Requirements file: `docs/changes/2026-05-29-notifications-mvp/01-requirements.md`
- Test specification: `docs/changes/2026-05-29-notifications-mvp/02-acceptance-tests.md`
- Next review artifact: `03-document-review.md`
- Preferred Plannotator review: `/plannotator-annotate docs/changes/2026-05-29-notifications-mvp/`
- Recommended first failing test for implementation: `IT-001` in `appview/internal/api/notification_store_test.go` proving follow notifications are derived from indexed data and scoped to the authenticated viewer.
- Suggested test order for implementation:
  1. AppView store tests `IT-001` through `IT-009`, plus `IT-013` for unavailable subjects once the chosen behavior is documented.
  2. AppView handler tests `UT-002` through `UT-008`, `IT-011`, and `IT-012` for JSON contract, limits, cursors, unknown params, and viewer scoping.
  3. AppView route test `IT-010` for route registration and auth/device protection.
  4. Flutter model/API tests `UT-009` and `UT-010`.
  5. Flutter provider tests `UT-011` through `UT-013`.
  6. Flutter widget/router tests `UT-014` through `UT-016`, then acceptance scenarios `AT-002` through `AT-005`.
  7. Regression checks `REG-001` through `REG-005` and manual checks `MAN-001`, `MAN-002`.
- Commands discovered:
  - AppView focused tests: from `appview/`, `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes`
  - Full AppView verification: from repo root, `just test` with compose Postgres available.
  - Flutter focused tests: from `app/`, `flutter test test/notifications test/feed/providers/timeline_provider_test.dart test/feed/data/post_api_client_test.dart` adjusted to actual new notification test paths.
  - Flutter analyzer: run analyzer checks over changed Dart source/tests according to existing project practice.
- Blocking gaps: None. GAP-001 through GAP-003 are non-blocking risk notes.
