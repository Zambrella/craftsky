# TDD Implementation Plan: Notifications MVP

## Inputs
- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md`
- Coding plan: `04-coding-plan.md`

## Implementation Rules
- Do not implement behavior without a linked requirement ID.
- Write or update a failing test before implementation.
- Run the smallest relevant test first.
- Refactor only after tests pass.
- Keep traceability updated.
- Keep unrelated `docs/roadmap.md` changes unstaged.

## Test Order
| Step | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| 1 | IT-001 | BR-001, BR-002, FR-001, FR-002, FR-009, RULE-002 | AC-001, AC-002, AC-003, AC-019 | `PostStore.ListNotifications` does not exist. |
| 2 | IT-002 | BR-002, FR-003, FR-009, FR-010 | AC-004 | Store does not derive like notifications. |
| 3 | IT-003 | BR-002, FR-004, FR-009, FR-010 | AC-005 | Store does not derive repost notifications. |
| 4 | IT-004 | FR-005, FR-009, FR-010, FR-011 | AC-007 | Store does not derive reply notifications. |
| 5 | IT-005 | FR-006 | AC-008 | Store may include self-generated notifications. |
| 6 | IT-006 | BR-002, FR-003, FR-004, RULE-001 | AC-006 | Store may include deleted like/repost rows. |
| 7 | IT-007 | FR-007, FR-008 | AC-009 | Mixed ordering/tie-break may be missing. |
| 8 | IT-008 | BR-001, FR-007, FR-008 | AC-009 | Pagination across mixed rows may be missing. |
| 9 | IT-009 | FR-008 | AC-009 | Exact-full terminal page cursor behavior may be missing. |
| 10 | UT-001 | FR-008 | AC-009, AC-015 | Cursor helper coverage missing. |
| 11 | IT-013 | FR-009, FR-010 | AC-012 | Unavailable subject omission untested. |
| 12 | UT-002-UT-008, IT-011-IT-012 | FR-001, FR-008, FR-009, RULE-002, RULE-003, NFR-002 | AC-014, AC-018, AC-020, AC-001, AC-019 | Handler/types do not exist. |
| 13 | IT-010 | FR-001, NFR-002 | AC-014 | Route not registered. |
| 14 | UT-009-UT-010 | BR-001, FR-009, FR-010, FR-012 | AC-012 | Flutter models/API client do not exist. |
| 15 | UT-011-UT-013 | FR-008, FR-013, FR-014 | AC-013, AC-015, AC-016 | Flutter provider/state do not exist. |
| 16 | UT-014-UT-016, AT-002-AT-005 | BR-001, FR-009, FR-011, FR-014, FR-015 | AC-010, AC-011, AC-013, AC-016 | Notifications page is placeholder. |
| 17 | REG-001-REG-005 | BR-002, NFR-001, FR-001, RULE-001, BR-001, FR-014 | AC-002, AC-017, AC-014, AC-006, AC-010 | Regression verification pending. |

## Implementation Steps

### Step 1: IT-001
- Write failing test: Added `TestNotificationStore_ListNotifications_DerivesFollowNotificationsScopedToViewer` in `appview/internal/api/notification_store_test.go`.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes`
- Confirmed failure: build failed because `api.NotificationRow`, `PostStore.ListNotifications`, and `api.NotificationTypeFollow` did not exist.
- Implement: Added notification store row/type definitions and follow-only `PostStore.ListNotifications` derived from `atproto_follows`, scoped to `viewerDID`, excluding self-follows, hydrating actor profile fields, and returning `limit + 1` cursor shape.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes`
- Green result: `ok social.craftsky/appview/internal/api`; `ok social.craftsky/appview/internal/routes`.
- Refactor: None.
- Notes: Covers follow derivation/scoping foundation only; later loops extend the same store method for likes, reposts, replies, and mixed pagination.

### Step 2: IT-002
- Write failing test: Added `TestNotificationStore_ListNotifications_DerivesActiveLikeNotificationsForViewerPosts`.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes`
- Confirmed failure: build failed because `api.NotificationTypeLike` did not exist and `ListNotifications` did not derive like rows.
- Implement: Extended notification row types and `ListNotifications` with active likes joined to viewer-authored subject posts, excluding other-author subjects and self likes.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes`
- Green result: `ok social.craftsky/appview/internal/api`; `ok social.craftsky/appview/internal/routes`.
- Refactor: Introduced nullable subject-post scan helper used by non-follow notification rows.
- Notes: Active-only `deleted_at IS NULL` filter was added for likes ahead of IT-006 because IT-002 is explicitly active-like behavior.

### Step 3: IT-003
- Write failing test: Added `TestNotificationStore_ListNotifications_DerivesActiveRepostNotificationsForViewerPosts`.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes`
- Confirmed failure: build failed because `api.NotificationTypeRepost` did not exist and repost derivation was missing.
- Implement: Extended `ListNotifications` with active repost rows joined to viewer-authored subject posts, excluding other-author subjects and self reposts.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes`
- Green result: `ok social.craftsky/appview/internal/api`; `ok social.craftsky/appview/internal/routes`.
- Refactor: None.
- Notes: Active-only `deleted_at IS NULL` filter was added for reposts ahead of IT-006 because IT-003 is explicitly active-repost behavior.

### Step 4: IT-004
- Write failing test: Added `TestNotificationStore_ListNotifications_DerivesDirectReplyNotificationsWithFocus`.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes`
- Confirmed failure: build failed because `api.NotificationTypeReply` did not exist and reply derivation was missing.
- Implement: Extended `ListNotifications` with direct reply rows where the reply parent is a viewer-authored post, excluding other-author subjects and deeper descendants, and added reply focus identity from the reply event URI/CID/rkey.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes`
- Green result: `ok social.craftsky/appview/internal/api`; `ok social.craftsky/appview/internal/routes`.
- Refactor: None.
- Notes: Direct-parent-only scope follows CPQ-002 and FR-005.

### Step 5: IT-005
- Write failing test: Added `TestNotificationStore_ListNotifications_ExcludesSelfGeneratedNotifications`.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes`
- Confirmed failure: No new failure; test passed because self-exclusion predicates were introduced during prior type derivation loops to satisfy active scoped behavior.
- Implement: No code change required in this loop.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes`
- Green result: `ok social.craftsky/appview/internal/api`; `ok social.craftsky/appview/internal/routes`.
- Refactor: None.
- Notes: Self-exclusion is covered explicitly for follow, like, repost, and reply rows.

### Step 6: IT-006
- Write failing test: Added `TestNotificationStore_ListNotifications_ExcludesDeletedLikesAndReposts`.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes`
- Confirmed failure: No new failure; test passed because active-only filters were introduced with IT-002/IT-003.
- Implement: No code change required in this loop.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes`
- Green result: `ok social.craftsky/appview/internal/api`; `ok social.craftsky/appview/internal/routes`.
- Refactor: None.
- Notes: Explicitly protects RULE-001 for both likes and reposts.

### Step 7: IT-007
- Write failing test: Added `TestNotificationStore_ListNotifications_OrdersMixedTypesByIndexedAtThenURIDesc`.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes`
- Confirmed failure: Initial failure was test setup related because seeded interactions used fixed old timestamps and a multi-statement update was invalid for pgx prepared execution. Fixed setup before implementation.
- Implement: No production code change required; existing query already orders by `e.indexed_at DESC, e.uri DESC`.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes`
- Green result: `ok social.craftsky/appview/internal/api`; `ok social.craftsky/appview/internal/routes`.
- Refactor: None.
- Notes: Deterministic tie-break is event URI descending across mixed notification types.

### Step 8: IT-008
- Write failing test: Added `TestNotificationStore_ListNotifications_PaginatesMixedTypesWithoutDuplicatesOrSkips`.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes`
- Confirmed failure: No new failure; pagination cursor support had been added in IT-001 and works across the mixed event query.
- Implement: No code change required.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes`
- Green result: `ok social.craftsky/appview/internal/api`; `ok social.craftsky/appview/internal/routes`.
- Refactor: None.
- Notes: Combined pages matched the full ordered list with no duplicate or skipped event URIs.

### Step 9: IT-009
- Write failing test: Added `TestNotificationStore_ListNotifications_OmitsCursorWhenExactFullFinalPage`.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes`
- Confirmed failure: No new failure; `limit + 1` cursor detection from IT-001 already omits terminal cursors correctly.
- Implement: No code change required.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes`
- Green result: `ok social.craftsky/appview/internal/api`; `ok social.craftsky/appview/internal/routes`.
- Refactor: None.
- Notes: Exact-full final page is protected for derived notification rows.

### Step 10: UT-001
- Write failing test: Added `TestNotificationStore_ListNotifications_InvalidCursorReturnsInvalidCursor`.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes`
- Confirmed failure: No new failure; existing `decodeSeekCursor` already returns `envelope.ErrInvalidCursor` for malformed cursors.
- Implement: No code change required.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes`
- Green result: `ok social.craftsky/appview/internal/api`; `ok social.craftsky/appview/internal/routes`.
- Refactor: None.
- Notes: The notification store uses the same opaque cursor helper as timeline with `indexedAt` and `uri` fields.

### Step 11: IT-013
- Write failing test: skipped.
- Run command: not run for this skipped loop.
- Confirmed failure: n/a.
- Implement: No code change required.
- Run command: n/a.
- Refactor: None.
- Notes: Cancelled because the current schema/test DDL has `craftsky_likes.subject_uri` and `craftsky_reposts.subject_uri` foreign keys with cascade, and replies reference the subject post through `craftsky_posts.reply_parent_uri`; the planned unavailable-subject case cannot be constructed without mutating schema constraints. The implementation behavior remains omission through inner joins to subject posts for like/repost/reply events.

### Step 12: UT-002-UT-008, IT-011-IT-012
- Write failing test: Added `appview/internal/api/notifications_test.go` covering default/capped/invalid limits, ignored unknown/request-supplied DID params, authenticated viewer scoping, camelCase JSON page shape, invalid cursor envelope, and store failure envelope.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes`
- Confirmed failure: build failed because `api.ListNotificationsHandler` and `api.NotificationPage` did not exist.
- Implement: Added `notifications.go` with `NotificationReader`, response DTOs, limit parsing, handler error mapping, handle resolution for actor/subject authors, subject-post response hydration with engagement summaries, and JSON page output.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes`
- Green result: `ok social.craftsky/appview/internal/api`; `ok social.craftsky/appview/internal/routes`.
- Refactor: Kept notification-specific handle resolution helper separate from timeline row helper.
- Notes: Handler never reads query DIDs for scope; only middleware DID is passed to the store.

### Step 13: IT-010
- Write failing test: Added route protection tests for `GET /v1/notifications` requiring auth and device ID.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes`
- Confirmed failure: route tests returned 404 because `/v1/notifications` was not registered.
- Implement: Registered `GET /v1/notifications` in `routes.AddRoutes` under the existing authenticated + device middleware stack using the shared `postStore`.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes`
- Green result: `ok social.craftsky/appview/internal/api`; `ok social.craftsky/appview/internal/routes`.
- Refactor: None.
- Notes: Protected route behavior mirrors timeline.

### Step 14: UT-009-UT-010
- Write failing test: Added Flutter model and API client tests for mixed notification decode and opaque cursor forwarding.
- Run command: `flutter test test/notifications/models/notification_test.dart test/notifications/data/notification_api_client_test.dart`
- Confirmed failure: tests could not pass until notification models and client existed.
- Implement: Added notification model/page parsing and `NotificationApiClient` for `GET /v1/notifications`.
- Run command: `flutter test test/notifications/models/notification_test.dart test/notifications/data/notification_api_client_test.dart`
- Green result: focused model/API tests passed.
- Refactor: Kept notification models manual to avoid unnecessary generated-file churn.
- Notes: Cursor remains opaque in the Dart API surface.

### Step 15: UT-011-UT-013
- Write failing test: Added provider tests for initial retry, append/terminal cursor, and load-more failure/concurrency.
- Run command: `flutter test test/notifications/models/notification_test.dart test/notifications/data/notification_api_client_test.dart test/notifications/providers/notifications_provider_test.dart`
- Confirmed failure: provider/state/repository bindings did not exist; initial retry test also exposed provider-lifetime setup issues that were fixed in the test harness.
- Implement: Added notification repository, production repository, provider bindings, `NotificationsState`, and `notificationsProvider` with pagination/dedupe/load-more guard behavior.
- Run command: `flutter test test/notifications/models/notification_test.dart test/notifications/data/notification_api_client_test.dart test/notifications/providers/notifications_provider_test.dart`
- Green result: focused Flutter model/API/provider tests passed.
- Refactor: None.
- Notes: Provider preserves previous value on load-more error and keeps cursor for retry.

### Step 16: UT-014-UT-016, AT-002-AT-005
- Write failing test: Updated `notifications_page_test.dart` to override repository and verify the page title plus rendered notification row copy.
- Run command: `flutter test test/notifications/models/notification_test.dart test/notifications/data/notification_api_client_test.dart test/notifications/providers/notifications_provider_test.dart test/notifications/notifications_page_test.dart`
- Confirmed failure: placeholder page did not consume provider/rows before implementation.
- Implement: Replaced placeholder with provider-backed loading/error/empty/list/load-more UI and added `NotificationRow` with type-specific copy and route intents.
- Run command: `flutter test test/notifications/models/notification_test.dart test/notifications/data/notification_api_client_test.dart test/notifications/providers/notifications_provider_test.dart test/notifications/notifications_page_test.dart`
- Green result: all focused Flutter notification tests passed.
- Refactor: None.
- Notes: Widget coverage is thinner than the full acceptance-test wish list for every state/navigation branch; row code implements route intents but deeper router assertions remain a follow-up candidate.

### Step 17: REG-001-REG-005
- Write failing test: n/a, regression verification step.
- Run command: AppView focused command and Flutter focused notification test command.
- Confirmed failure: n/a.
- Implement: n/a.
- Run command: `flutter analyze` also run.
- Refactor: n/a.
- Notes: Focused tests pass. `flutter analyze` remains blocked by an existing unrelated error in `lib/profile/data/dummy_profile_repository.dart` missing `ProfileRepository` methods, plus pre-existing infos/warnings; no new analyzer errors from notification code were observed.

## Completion Checklist
- [x] All Must requirements covered by tests or documented gaps
- [x] All planned Must tests passing or explicitly cancelled with reason
- [x] Relevant regression tests passing where runnable
- [x] No unlinked behavior implemented
- [x] Docs updated
- [ ] Review completed or explicitly skipped
