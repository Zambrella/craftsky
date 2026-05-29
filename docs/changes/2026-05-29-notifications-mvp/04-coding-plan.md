# Coding Plan: Notifications MVP

## 1. Inputs

- Requirements: `docs/changes/2026-05-29-notifications-mvp/01-requirements.md`
- Tests: `docs/changes/2026-05-29-notifications-mvp/02-acceptance-tests.md`
- Document review: `docs/changes/2026-05-29-notifications-mvp/03-document-review.md`
- Review verdict: Approved with notes; no blocking issues.

## 2. Implementation Strategy

Implement the MVP as a derived read-only feed from existing AppView indexed tables, then consume it through a new Flutter notifications feature slice. This matches the existing architecture: AppView owns indexed reads; Flutter talks to AppView through Dio + session/device interceptors; paginated state uses Riverpod `AsyncNotifier` patterns already present in the timeline.

Key design choices from document review:

- **Notification response schema:** define a concrete notification page with `items` and optional `cursor`. Each item has event identity (`uri`, `cid`, `rkey`), `type`, `actor`, `createdAt`, `indexedAt`, optional `subjectPost`, and optional `reply` focus identity.
- **Unavailable subject-post behavior:** omit like/repost/reply notifications when the subject post cannot be joined or hydrated. This follows existing foreign-key-backed interaction data and keeps the MVP UI simple; the endpoint must not crash.
- **Pagination:** use reverse chronological ordering by event `indexed_at DESC, event_uri DESC`; encode opaque cursors with `indexedAt` and `uri`, mirroring timeline cursor semantics.
- **No persistence or writes:** do not create notification tables, migrations, PDS write paths, push tokens, unread state, grouping, or lexicons.

## 3. Affected Areas

| Area | Existing Pattern | Planned Change | Requirement IDs | Test IDs |
|---|---|---|---|---|
| AppView store | `PostStore` has timeline/profile post read methods with SQL, seek cursors, and real-Postgres tests. | Add `ListNotifications` to derive follow/like/repost/reply rows from existing tables. | BR-002, FR-002-FR-008, FR-010, FR-011, RULE-001, RULE-002 | IT-001-IT-009, IT-013, REG-003 |
| AppView API handler | `ListTimelineHandler` parses `limit`/`cursor`, gets viewer DID from middleware, writes JSON/error envelopes. | Add `ListNotificationsHandler` with the same auth-context, limit, cursor, error-envelope, handle-resolution, and post-engagement patterns. | FR-001, FR-008, FR-009, RULE-002, RULE-003 | UT-002-UT-008, IT-011, IT-012, REG-002, REG-004 |
| AppView routing | `routes.AddRoutes` registers `/v1/*` with `authN(deviceID(...))`. | Register `GET /v1/notifications` under authenticated + device middleware. | FR-001 | IT-010, REG-002 |
| Flutter data layer | Feature-specific API clients/repositories use Dio, `unwrapApi`, Riverpod keepAlive providers, dart_mappable models. | Add notifications API client, repository, page/item models, provider bindings, and mapper initialization. | FR-012, NFR-001 | UT-009, UT-010, REG-001 |
| Flutter state | Timeline uses `@riverpod class Timeline extends _$Timeline` with cursor state and load-more guard. | Add `Notifications` `AsyncNotifier` with initial load, append, retry, terminal cursor, load-more error preservation, and concurrency guard. | FR-013 | UT-011-UT-013, AT-004, AT-005 |
| Flutter UI/navigation | `NotificationsPage` is a placeholder; routes already include `NotificationsRoute`, `UserProfileRoute`, and `PostThreadRoute(focus)`. | Replace placeholder with paginated sliver UI, row widget, l10n strings, and row tap navigation. | BR-001, FR-014, FR-015 | AT-002-AT-005, UT-014-UT-016, REG-005 |
| Generated Dart files | Existing Riverpod/dart_mappable/go_router generated files are checked in. | TDD builder should regenerate affected providers/mappers/l10n after source changes. | FR-012-FR-014 | UT-009-UT-016 |

## 4. Files And Modules

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `appview/internal/api/notifications.go` | Create | Notification DTOs, handler interface, `ListNotificationsHandler`, limit parsing, response building. | FR-001, FR-008, FR-009, RULE-002, RULE-003 | UT-002-UT-008, IT-011, IT-012 |
| `appview/internal/api/notification_store.go` | Create | `PostStore.ListNotifications` derived union query and notification cursor encoding. | BR-002, FR-002-FR-008, FR-010, FR-011, RULE-001 | IT-001-IT-009, IT-013 |
| `appview/internal/api/notification_store_test.go` | Create | Store/integration tests from the acceptance spec. | BR-001, BR-002, FR-002-FR-008, RULE-001, RULE-002 | IT-001-IT-009, IT-013 |
| `appview/internal/api/notifications_test.go` | Create | Handler JSON/error/limit/scoping tests. | FR-001, FR-008, FR-009, RULE-002, RULE-003 | UT-002-UT-008, IT-011, IT-012 |
| `appview/internal/routes/routes.go` | Change | Register `GET /v1/notifications` with existing authenticated/device middleware. | FR-001 | IT-010, REG-002 |
| `appview/internal/routes/routes_test.go` | Change | Add route protection tests for `/v1/notifications`. | FR-001 | IT-010, REG-002 |
| `app/lib/notifications/models/craftsky_notification.dart` | Create | Dart notification item, actor, reply-focus models with mappers. | FR-009, FR-011, FR-012 | UT-009, UT-016 |
| `app/lib/notifications/models/notification_page.dart` | Create | Paginated response model with opaque cursor. | FR-008, FR-012 | UT-009, UT-010 |
| `app/lib/notifications/models/notifications_state.dart` | Create | Provider state: items + cursor + `hasMore`. | FR-013 | UT-011-UT-013 |
| `app/lib/notifications/data/notification_api_client.dart` | Create | Dio client for `GET /v1/notifications`. | FR-012, NFR-001 | UT-010, REG-001 |
| `app/lib/notifications/data/notification_repository.dart` | Create | Abstract repository interface for providers/tests. | FR-012 | UT-011-UT-013 |
| `app/lib/notifications/data/api_notification_repository.dart` | Create | Production repository wrapping the API client. | FR-012 | UT-010 |
| `app/lib/notifications/providers/notification_api_client_provider.dart` | Create | KeepAlive provider backed by shared `dioProvider`. | FR-012, NFR-001 | UT-010, REG-001 |
| `app/lib/notifications/providers/notification_repository_provider.dart` | Create | KeepAlive repository provider. | FR-012 | UT-011-UT-013 |
| `app/lib/notifications/providers/notifications_provider.dart` | Create | `AsyncNotifier` for initial page and load-more behavior. | FR-013 | UT-011-UT-013 |
| `app/lib/notifications/pages/notifications_page.dart` | Change | Replace placeholder with loading/empty/error/list/load-more UI. | BR-001, FR-014, FR-015 | AT-002-AT-005, UT-014-UT-016 |
| `app/lib/notifications/widgets/notification_row.dart` | Create | Type-specific row copy, actor fallback display, and tap target. | FR-009, FR-014, FR-015 | UT-014-UT-016 |
| `app/lib/bootstrap.dart` | Change | Register new dart_mappable model mappers. | FR-012 | UT-009 |
| `app/lib/l10n/app_en.arb` and generated l10n | Change | Add notification title, empty, error, row, and retry/load-more copy. | FR-014 | UT-014, MAN-001 |
| Generated Dart (`*.g.dart`, `*.mapper.dart`, `generated/app_localizations*.dart`) | Create / Change | Riverpod, mappers, l10n outputs after source changes. | FR-012-FR-014 | UT-009-UT-016 |
| `app/test/notifications/models/notification_test.dart` | Create | Mixed-type model decode tests. | FR-009, FR-010, FR-012 | UT-009 |
| `app/test/notifications/data/notification_api_client_test.dart` | Create | Endpoint path/query/cursor and decode tests. | FR-012, NFR-001 | UT-010 |
| `app/test/notifications/fakes/fake_notification_repository.dart` | Create | Provider/widget fake repository. | FR-013, FR-014 | UT-011-UT-016 |
| `app/test/notifications/providers/notifications_provider_test.dart` | Create | Provider state/pagination/retry/concurrency tests. | FR-013 | UT-011-UT-013 |
| `app/test/notifications/notifications_page_test.dart` | Change | Replace placeholder test with page state, row, fallback, and navigation tests. | BR-001, FR-014, FR-015 | AT-002-AT-005, UT-014-UT-016 |

## 5. Services, Interfaces, And Data Flow

### AppView data contract

Use a concrete JSON shape that is easy to decode and navigate from Flutter:

```text
NotificationPage {
  items: []NotificationItem,
  cursor?: string
}

NotificationItem {
  uri: string,          // event record URI: follow/like/repost/reply URI
  cid: string,
  rkey: string,
  type: "follow" | "like" | "repost" | "reply",
  actor: NotificationActor,
  createdAt: time,
  indexedAt: time,
  subjectPost?: PostResponse,       // required for like/repost/reply
  reply?: NotificationReplyRef      // present for reply, used as thread focus
}

NotificationActor {
  did: string,
  handle: string,
  displayName?: string,
  avatarCid?: string
}

NotificationReplyRef {
  uri: string,
  cid: string,
  rkey: string
}
```

Do not include `totalCount`, unread state, grouping metadata, or notification persistence IDs.

### AppView store shape

Add a read interface and store row in `appview/internal/api/notifications.go` / `notification_store.go`:

```text
type NotificationReader interface {
  ListNotifications(ctx context.Context, viewerDID string, limit int, cursor string) ([]*NotificationRow, string, error)
  EngagementSummaries(ctx context.Context, viewerDID string, postURIs []string) (map[string]EngagementSummary, error)
}

type NotificationRow struct {
  Type NotificationType
  URI, CID, Rkey string
  ActorDID string
  ActorDisplayName *string
  ActorAvatarCID *string
  CreatedAt, IndexedAt time.Time
  SubjectPost *PostRow
  Reply *NotificationReplyRef
}
```

Derived query strategy:

```text
WITH events AS (
  -- follow notifications where f.subject_did = viewerDID and f.did != viewerDID
  UNION ALL
  -- active likes where subject post did = viewerDID and l.did != viewerDID
  UNION ALL
  -- active reposts where subject post did = viewerDID and r.did != viewerDID
  UNION ALL
  -- direct replies where reply_parent_uri joins a viewer-authored post and reply.did != viewerDID
)
SELECT event columns,
       actor bluesky profile columns,
       subject post columns and subject author profile columns
FROM events
LEFT JOIN bluesky_profiles actor_bp ON actor_bp.did = actor_did
LEFT/INNER JOIN subject post/profile as appropriate
WHERE cursor seek predicate on (indexed_at, event_uri)
ORDER BY indexed_at DESC, event_uri DESC
LIMIT limit + 1
```

Important store decisions:

- For like/repost/reply rows, use an inner join to `craftsky_posts` for the subject. If subject hydration is unavailable, omit the notification row and pass `IT-013` by asserting endpoint success and omission.
- For follows, no subject post exists; only actor data is returned.
- For likes/reposts, require `deleted_at IS NULL`.
- For replies, include only direct replies where `reply.reply_parent_uri` is a viewer-authored post URI. Deeper descendants are out of MVP scope unless already represented as direct parent authored by the viewer.
- Exclude all rows where `actor_did = viewerDID`.
- Cursor payload should be `{ "indexedAt": <RFC3339Nano>, "uri": <event URI> }`.

### AppView handler data flow

```text
request -> auth/device middleware -> ListNotificationsHandler
  -> viewer DID from middleware context
  -> parse limit default/cap = 20/50; read cursor; ignore unknown params
  -> store.ListNotifications(viewerDID, limit, cursor)
  -> collect unique actor DIDs and subject-post author DIDs
  -> resolve handles through HandleResolver (fail with identity_unavailable on resolver error)
  -> batch EngagementSummaries for subjectPost URIs
  -> BuildNotificationResponse rows
  -> JSON NotificationPage{items, cursor}
```

Add a small DID-handle helper rather than bending `resolveHandlesForRows` to notification rows:

```text
resolveHandlesForDIDs(ctx, didStrings, resolver) (map[string]syntax.Handle, error)
```

### Flutter data flow

```text
NotificationsPage
  watches notificationsProvider
    -> notificationRepositoryProvider
      -> ApiNotificationRepository
        -> NotificationApiClient(dioProvider)
          -> GET /v1/notifications?limit=20&cursor=<opaque>
```

Use a feature-local repository rather than extending `PostRepository`; notifications combine social events and posts, and keeping the dependency separate helps `REG-001` prove no PDS path is introduced.

## 6. State, Providers, Controllers, Or DI

Flutter Riverpod provider graph:

```text
dioProvider (existing, session + device interceptors)
  -> notificationApiClientProvider (keepAlive)
    -> notificationRepositoryProvider (keepAlive)
      -> notificationsProvider (AsyncNotifier<NotificationsState>)
```

Provider sketch:

```text
const notificationsPageLimit = 20

@riverpod
class Notifications extends _$Notifications {
  Future<NotificationsState> build()
  Future<void> loadMore()
}

NotificationsState {
  List<CraftskyNotification> items
  String? cursor
  bool get hasMore => cursor != null
}
```

Use the timeline provider behavior as the reference:

- Initial load calls `repo.list(limit: notificationsPageLimit)`.
- `loadMore` no-ops when current state is absent, `hasMore` is false, or the provider is already loading.
- During load-more, use previous-value-preserving `AsyncLoading().copyWithPrevious(state)` pattern.
- On load-more success, append rows in order and dedupe by event `uri`.
- On load-more failure, preserve visible rows and cursor for retry.

AppView DI:

- No `app.Deps` field is needed for notifications. `routes.AddRoutes` can reuse `postStore := api.NewPostStore(deps.DB)` as it does for timeline/posts.
- No new PDS client or OAuth dependency is introduced.

## 7. UI, Widgets, Routes, Or User-Facing Surfaces

### Notifications page

Replace the placeholder `NotificationsPage` with the same broad shape as `FeedPage`:

```text
Scaffold
  AppBar(title: l10n.notificationsTitle)
  CustomScrollView
    loading -> SliverFillRemaining(StitchProgressIndicator)
    initial error -> retry sliver
    loaded empty -> empty sliver
    loaded items -> SliverList(NotificationRow)
    load-more loading/error -> bottom progress or retry row
```

Use an `_autoLoadMoreThreshold` equivalent (same threshold as feed is acceptable) to trigger pagination near the end.

### Row composition

Create `NotificationRow` with these display rules:

- Actor display: `displayName ?? handle.toString() ?? did.toString()`; handle is expected from API but still keep a defensive fallback for tests.
- `follow`: “Alice followed you” and tap opens actor profile.
- `like`: “Alice liked your post” plus subject text preview; tap opens subject post thread.
- `repost`: “Alice reposted your post” plus subject text preview; tap opens subject post thread.
- `reply`: “Alice replied to your post” plus subject text preview; tap opens subject post thread with `focus=reply.uri` when available.

Preferred navigation:

```text
follow -> UserProfileRoute(handle: item.actor.handle).push(context)
like/repost -> PostThreadRoute(did: item.subjectPost.author.did, rkey: item.subjectPost.rkey).push(context)
reply -> PostThreadRoute(did: item.subjectPost.author.did, rkey: item.subjectPost.rkey, focus: item.reply?.uri).push(context)
```

No router path changes are expected; `NotificationsRoute`, `UserProfileRoute`, and `PostThreadRoute(focus)` already exist. Regenerate router code only if route APIs change unexpectedly.

### Localization

Add l10n keys rather than hard-coded strings. Suggested keys:

- `notificationsTitle`: `Notifications`
- `notificationsEmpty`: `No notifications yet.`
- `notificationsLoadError`: `Notifications didn't load.`
- `notificationFollowRow`: `{actor} followed you`
- `notificationLikeRow`: `{actor} liked your post`
- `notificationRepostRow`: `{actor} reposted your post`
- `notificationReplyRow`: `{actor} replied to your post`
- Optional accessibility labels for row tap targets if widget tests need stable semantics.

## 8. Error, Loading, Empty, And Edge States

| State / Case | Planned Handling | Requirement IDs | Test IDs |
|---|---|---|---|
| Empty notifications | AppView returns `items: []` and omits `cursor`; Flutter renders `notificationsEmpty`. | BR-001, FR-014 | UT-014, AT-002 |
| Initial AppView store error | Handler logs with run ID and returns standard 5xx error envelope. | FR-001, NFR-002 | UT-008, REG-004 |
| Invalid cursor | Store returns `envelope.ErrInvalidCursor`; handler returns `400 invalid_cursor`. | FR-008 | UT-001, UT-007 |
| Missing auth or device ID | Existing middleware handles unauthorized/missing-device response. | FR-001 | IT-010, REG-002 |
| High/invalid limit | Handler applies default/cap `20/50`; unknown query params ignored. | FR-008, RULE-003 | UT-002-UT-005, IT-011 |
| Request-supplied DID | Handler never reads DID query params for scope; store gets middleware DID. | RULE-002 | IT-012, AT-001 |
| Self-generated activity | Store excludes actor DID equal to viewer DID across all event types. | FR-006 | IT-005 |
| Deleted like/repost | Store filters `deleted_at IS NULL`. | RULE-001 | IT-006, REG-003 |
| Subject post unavailable | Omit like/repost/reply notification through subject-post inner join; endpoint succeeds. | FR-010 | IT-013, GAP-002 |
| Actor profile missing display fields | API still includes DID/handle; Flutter row falls back display safely. | FR-009 | UT-016 |
| Initial Flutter load error | Page shows error state and retry invalidates `notificationsProvider`. | FR-013, FR-014 | UT-011, UT-014, AT-004 |
| Load-more failure | Preserve visible rows/cursor, show bottom retry, no concurrent duplicate requests. | FR-013, FR-014 | UT-013, UT-014, AT-005 |
| Terminal cursor | `cursor == null` makes `hasMore` false; provider/page stop load-more. | FR-008, FR-013 | UT-012 |
| Reply focus unavailable | Open subject thread without `focus` rather than failing navigation. | FR-015 | UT-015, AT-003 |

## 9. Test Implementation Plan

| Order | Test ID | Target | Setup / Fixture | Initial Expected Failure |
|---|---|---|---|---|
| 1 | IT-001 | `appview/internal/api/notification_store_test.go` | Add `notificationStoreDDL = timelineStoreDDL`; seed viewer/Alice/Bob follows and profiles. | `PostStore.ListNotifications` does not exist. |
| 2 | IT-002-IT-004 | `notification_store_test.go` | Seed viewer subject posts, active likes/reposts, and direct reply rows. | Store returns no like/repost/reply rows or row type fields missing. |
| 3 | IT-005-IT-006 | `notification_store_test.go` | Seed self-generated rows and deleted interactions. | Store includes self/deleted rows. |
| 4 | IT-007-IT-009, UT-001 | `notification_store_test.go` and cursor helper coverage | Seed mixed rows with tied times and page size boundaries. | Ordering/cursor implementation missing or unstable. |
| 5 | IT-013 | `notification_store_test.go` | If schema allows, remove/unjoin subject or construct test around omission semantics. | Endpoint/store crashes or behavior undocumented. |
| 6 | UT-002-UT-008, IT-011-IT-012 | `appview/internal/api/notifications_test.go` | Fake notification reader and fake resolver. | Handler/types do not exist. |
| 7 | IT-010 | `appview/internal/routes/routes_test.go` | Route mux with `testDeps()`, auth/device header variants. | `/v1/notifications` returns 404. |
| 8 | UT-009 | `app/test/notifications/models/notification_test.dart` | Mixed notification JSON using `TD-008`. | Dart models/mappers do not exist. |
| 9 | UT-010 | `app/test/notifications/data/notification_api_client_test.dart` | Dio mock adapter expects `/v1/notifications` with `limit` and opaque `cursor`. | API client does not exist. |
| 10 | UT-011-UT-013 | `app/test/notifications/providers/notifications_provider_test.dart` | Fake notification repository with success, failure, cursor, and Completer gates. | Provider/state classes do not exist. |
| 11 | UT-014-UT-016, AT-002-AT-005 | `app/test/notifications/notifications_page_test.dart` | Provider overrides and test router harness for profile/thread navigation. | Placeholder page lacks states/rows/navigation. |
| 12 | REG-001-REG-005 | Existing and new focused suites | Run focused AppView/Flutter tests; review no PDS dependencies. | Regressions in auth/cursor/routing or direct PDS dependency introduced. |

Focused commands for the TDD builder:

```text
# AppView, from appview/
TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes

# Flutter codegen, from app/ when providers/mappers/l10n are added
dart run build_runner build --delete-conflicting-outputs

# Flutter focused tests, from app/
flutter test test/notifications/models/notification_test.dart \
  test/notifications/data/notification_api_client_test.dart \
  test/notifications/providers/notifications_provider_test.dart \
  test/notifications/notifications_page_test.dart

# Flutter analyzer, from app/
flutter analyze
```

## 10. Sequencing And Guardrails

- First TDD step: create failing `IT-001` for `PostStore.ListNotifications` follow derivation and viewer scoping.
- Dependencies between work items:
  1. Store row/query and cursor behavior before handler response construction.
  2. Handler DTO shape before Flutter model/API tests.
  3. Flutter models/API client before repository/provider.
  4. Provider before widget states.
  5. UI rows before navigation widget/router tests.
- Guardrails:
  - Do not add notification persistence, migrations, unread state, push delivery, grouping, or preferences.
  - Do not read craft data directly from a PDS in Flutter or AppView notification read path.
  - Do not modify lexicons.
  - Do not add PDS write paths or store PDS tokens on device.
  - Keep `/v1/notifications` protected by both auth and device middleware.
  - Keep all response JSON camelCase and all errors in the standard `{error, message, requestId}` envelope.
  - Use typed atproto identifiers at Go HTTP boundaries when parsing user input; internally follow existing `PostStore` string conventions for stored rows.
  - Preserve existing tests for timeline/profile/posts; notification code must not change their wire contracts.
- Out of scope:
  - Push, unread/read, badges, grouping, notification preferences, moderation/block/mute filtering, search, quote-post notifications, and durable historical like/repost notifications.

## 11. Risks And Open Questions

| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Non-blocking | Exact empty-state copy was non-blocking in requirements. | Copy can be adjusted without changing behavior. | Use `No notifications yet.` initially via l10n; manual review `MAN-001` can request changes. |
| CPQ-002 | Non-blocking | Reply notification scope is direct replies only. | Deeper thread descendants may not notify unless their direct parent is viewer-authored. | Plan follows `FR-005`/requirements open question: direct parent authored by viewer for MVP. |
| CPQ-003 | Non-blocking | Derived query could become expensive with scale. | Future performance work may need indexes. | Keep page size capped; no migration unless implementation proves an index is necessary and separately justified. |
| CPQ-004 | Non-blocking | Handle resolution failure policy may hide otherwise indexed notifications. | Existing timeline behavior returns 502 on resolver failure. | Reuse existing `identity_unavailable` behavior for consistency. |
| CPQ-005 | Non-blocking | Flutter route tests may need a small typed-router test harness. | Navigation assertions can be brittle. | Prefer checking `GoRouter.location`/matched path after taps, with provider overrides and test auth state as existing router tests do. |

## 12. Handoff To TDD Builder

- Coding plan: `docs/changes/2026-05-29-notifications-mvp/04-coding-plan.md`
- TDD execution plan: implementation should proceed directly from this plan and `02-acceptance-tests.md`; create any separate `05-implementation-plan.md` only if the next-stage workflow requires it.
- Start with test: `IT-001` in `appview/internal/api/notification_store_test.go` proving follow notifications are derived from indexed data and scoped to the authenticated viewer.
- Focused command: from `appview/`, `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes`
- Notes:
  - Treat `01-requirements.md`, `02-acceptance-tests.md`, `03-document-review.md`, and this coding plan as source of truth.
  - Implement red-green-refactor in the order listed above.
  - Regenerate Dart generated files only during implementation, not before tests require them.
  - Keep unrelated `docs/roadmap.md` changes unstaged unless explicitly asked to handle them.
