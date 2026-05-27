# Coding Plan: Profile Social Summary

## 1. Inputs
- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` (`Approved with notes`)
- Review notes resolved here:
  - DR-001 / GAP-001: route names are finalized in §5.
  - DR-002 / GAP-002: response field names are finalized in §5.
  - DR-003 / GAP-003: Flutter account-age thresholds are finalized in §7.
  - DR-004 / GAP-004: query shapes and indexes are called out in §5 and §10.

## 2. Implementation Strategy
Implement this as an additive full-stack profile-summary change using the existing AppView-read / Flutter-render architecture.

On the AppView, extend the existing profile read model with scalar summary fields (`mutualFollowerCount`, `postCount`, `postsLast7Days`, `projectCount`) and add authenticated paginated social-graph list endpoints. Keep existing `followerCount` and `followingCount` in the profile response for compatibility, but do not expose them in profile/settings entry UI.

On Flutter, extend profile models/repositories/providers for the new fields and graph list pages, replace the current follower/following stats widget with account/activity/project stats, add a visitor-only mutual-follower link that opens a 90%-height bottom sheet, and add settings-owned followers/following list pages.

The first TDD slice should lock AppView summary semantics (`IT-001`, `UT-002`, `UT-003`) before list endpoints and Flutter UI, because the client will depend on the chosen API contract.

## 3. Affected Areas
| Area | Existing Pattern | Planned Change | Requirement IDs | Test IDs |
|---|---|---|---|---|
| AppView profile summary | `ProfileStore.Read` builds `ProfileRow`; `BuildProfileResponse` serializes camelCase JSON | Add top-level post counts, data-driven project count, and viewer-scoped mutual follower count | FR-003, FR-005, FR-006, FR-012, FR-015, RULE-001, RULE-003, RULE-005 | IT-001, IT-002, IT-003, UT-001, UT-002, UT-003, UT-005 |
| AppView social graph lists | Profile subresources use `/v1/profiles/{handleOrDid}/...`; lists use `items` + opaque `cursor` | Add mutual followers, followers, and following list read methods/handlers/routes | FR-008, FR-009, FR-010, FR-011, FR-016, NFR-001, RULE-004 | IT-004, IT-005, IT-006, IT-007, IT-008 |
| AppView persistence/indexes | Existing `atproto_follows` indexes by `did`, `subject_did`, `(did, subject_did)`; posts index by `(did, indexed_at DESC)` | Add ordered indexes for follow recency and root-post summary queries if integration/performance tests show current indexes insufficient | NFR-002 | IT-008, MAN-002 |
| Flutter profile model/client/repository | `Profile`, `ProfileApiClient`, `ProfileRepository`, Riverpod repository provider | Add profile summary fields and paginated account-summary list methods | FR-004, FR-006, FR-010, FR-011, FR-015, FR-016 | UT-011, UT-012, AT-003, AT-004 |
| Flutter profile UI | `ProfileMetaSection` renders `ProfileStats` with following/follower/project cells | Render new profile summary stats; hide old count cells; add mutual link and bottom sheet | BR-001, BR-002, FR-001, FR-002, FR-004, FR-005, FR-006, FR-013, FR-014, FR-017 | AT-001, AT-002, AT-003, AT-004, AT-008, AT-009, UT-004, UT-006, UT-007 |
| Flutter settings/navigation | Settings has clear-cache and sign-out tiles only; typed `go_router` routes under profile branch | Add followers/following settings entries and root-navigator list routes/pages | BR-003, FR-007, FR-008, FR-009, FR-014 | AT-005, AT-006, AT-007, AT-008, UT-008, UT-009, UT-010, REG-005 |
| Test fixtures/fakes | Existing Go `fakeStore`, Flutter `FakeProfileRepository`, `FakePostRepository` | Extend fakes for summary and graph lists; add seed helpers where useful | All Must requirements | All listed automated tests |

## 4. Files And Modules
| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `appview/internal/api/profile_store.go` | Change | Extend `ProfileRow`; add summary-count and graph-list methods | FR-003, FR-005, FR-006, FR-010, FR-011, FR-015, FR-016, RULE-001, RULE-003, RULE-004, RULE-005 | IT-001, IT-002, IT-004, IT-005, IT-006, IT-008, UT-001, UT-002, UT-003 |
| `appview/internal/api/profile_response.go` | Change | Add scalar profile fields and account-summary/list response DTOs | FR-006, FR-010, FR-012, FR-015 | IT-003, UT-005 |
| `appview/internal/api/profile.go` | Change | Add graph-list handler interfaces/handlers and route helper reuse | FR-016, NFR-001 | IT-004, IT-005, IT-006, IT-007 |
| `appview/internal/routes/routes.go` | Change | Register finalized social graph endpoints behind `authN(deviceID(...))` | NFR-001 | IT-007, REG-007 |
| `appview/migrations/000013_profile_social_summary_indexes.up.sql` / `.down.sql` | Create if needed | Add supporting indexes for ordered follows/root-post counts | NFR-002 | IT-008, MAN-002 |
| `appview/internal/api/profile_store_test.go` | Change | Add summary-count, mutual, ordering, pagination, and bounded-query tests | FR-003, FR-005, FR-006, FR-011, FR-016, NFR-002, RULE-001, RULE-003, RULE-004, RULE-005 | IT-001, IT-002, IT-004, IT-005, IT-006, IT-008, UT-001, UT-002, UT-003 |
| `appview/internal/api/profile_test.go` | Change | Add handler contract tests for profile response and graph list endpoints | FR-010, FR-011, FR-015, FR-016, NFR-001 | IT-003, IT-004, IT-005, IT-006 |
| `appview/internal/api/profile_response_test.go` | Change | Assert new scalar fields, old count fields, and no mutual preview array | FR-012, FR-015 | UT-005, REG-008 |
| `appview/internal/routes/routes_test.go` | Change | Assert new endpoints require auth and device ID | NFR-001 | IT-007, REG-007 |
| `app/lib/profile/models/profile.dart` + generated mapper | Change | Add `mutualFollowerCount`, `postCount`, `postsLast7Days`, `projectCount` | FR-004, FR-005, FR-006, FR-015 | UT-011, AT-004 |
| `app/lib/profile/models/profile_account_summary.dart` + generated mapper | Create | Display-ready account row model for graph lists | FR-010 | UT-011 |
| `app/lib/profile/models/profile_account_page.dart` + generated mapper | Create | Paginated graph-list response (`items`, optional `cursor`, `totalCount`) | FR-011, FR-016 | UT-012 |
| `app/lib/profile/data/profile_api_client.dart` | Change | Add `listMutualFollowers`, `listFollowersMe`, `listFollowingMe` | FR-010, FR-011, FR-016 | UT-012, AT-003 |
| `app/lib/profile/data/profile_repository.dart` / `api_profile_repository.dart` | Change | Add repository methods for graph list pages | FR-010, FR-011, FR-016 | UT-012 |
| `app/lib/profile/providers/profile_account_list_provider.dart` + generated provider | Create | Cursor-accumulating Riverpod list state for mutual/followers/following | FR-008, FR-009, FR-013, FR-014, FR-016 | UT-010, AT-003, AT-006, AT-007, AT-008 |
| `app/lib/profile/models/profile_account_list_state.dart` + generated mapper | Create | Shared items/cursor/total-count state object | FR-008, FR-009, FR-011 | UT-010, UT-012 |
| `app/lib/profile/widgets/profile_stats.dart` | Change | Replace old count cells with joined/recent/total/projects rendering | FR-001, FR-004, FR-005, FR-006, FR-017 | UT-004, UT-006, AT-001, AT-004, AT-009 |
| `app/lib/profile/widgets/profile_mutual_followers_link.dart` | Create | Visitor-only clickable mutual follower text | FR-002, FR-013, FR-014 | UT-007, AT-002 |
| `app/lib/profile/widgets/profile_account_list.dart` | Create | Reusable account list body/rows/empty/loading/pagination UI | FR-010, FR-014 | UT-008, UT-010, AT-003, AT-006, AT-007, AT-008 |
| `app/lib/profile/widgets/profile_mutual_followers_sheet.dart` | Create | 90%-height bottom sheet shell using account list provider | FR-013, FR-016 | AT-003, MAN-001 |
| `app/lib/profile/pages/profile_page.dart` | Change | Pass `isOwnProfile` into meta section; preserve follow/unfollow behavior | FR-001, FR-002, FR-013 | AT-001, AT-002, AT-003, REG-001, REG-002 |
| `app/lib/profile/widgets/profile_meta_section.dart` | Change | Compose non-Craftsky marker, bio/crafts, mutual link, and new stats | FR-001, FR-002, FR-004, FR-005, FR-006, FR-014, FR-017 | AT-001, AT-002, AT-004, AT-008, AT-009 |
| `app/lib/settings/pages/settings_page.dart` | Change | Add followers/following entries without counts | FR-007, FR-012 | AT-005, UT-009, REG-005 |
| `app/lib/settings/pages/followers_page.dart` / `following_page.dart` or one parameterized page | Create | Settings-owned list pages with counts in app bar titles | FR-008, FR-009, FR-014 | AT-006, AT-007, AT-008, UT-010 |
| `app/lib/router/route_locations.dart`, `app/lib/router/router.dart`, generated route file | Change | Add typed routes under settings/profile branch for follow-list pages | FR-008, FR-009 | AT-006, AT-007, UT-010 |
| `app/lib/l10n/app_en.arb` + generated localizations | Change | Add labels/copy for stats, mutuals, settings entries, empty states | FR-002, FR-004, FR-005, FR-006, FR-007, FR-014 | UT-004, UT-006, UT-008, UT-009, UT-010 |
| `app/test/profile/**`, `app/test/settings/**` | Change/Create | Flutter model/client/provider/widget tests from `02-acceptance-tests.md` | All UI/client requirements | AT-001-AT-009, UT-004, UT-006-UT-012, REG-001-REG-005 |

## 5. Services, Interfaces, And Data Flow
### Final AppView routes
Use these concrete `/v1/` endpoint paths:

```text
GET /v1/profiles/@{handleOrDid}/mutual-followers?limit=&cursor=
GET /v1/profiles/me/followers?limit=&cursor=
GET /v1/profiles/me/following?limit=&cursor=
```

Rationale:
- Mutuals are target-profile scoped, so they live under the existing profile subresource pattern and accept handle or DID.
- Followers/following access in this slice is settings-owned and scoped to the authenticated user, so `me` avoids embedding a stale self handle/DID.
- All three routes are authenticated and device-ID protected in `routes.go`.

### Final JSON field names
Extend `ProfileResponse` additively:

```text
ProfileResponse {
  did, handle, viewerIsFollowing, isCraftskyProfile,
  followingCount?, followerCount?,
  mutualFollowerCount?,
  postCount?,
  postsLast7Days?,
  projectCount?,
  displayName?, description?, avatar?, banner?, crafts, createdAt?
}
```

Notes:
- Keep `followerCount` and `followingCount`; Flutter must not render them on profile pages or settings entry page.
- `mutualFollowerCount` is set only when a signed-in viewer is reading another Craftsky profile; otherwise omit or decode to `0` client-side.
- `postCount`, `postsLast7Days`, and `projectCount` are Craftsky-profile summary fields. `projectCount` is data-driven and may be `0` until project persistence exists.
- Do not add a mutual preview array to `ProfileResponse`.

Graph-list response shape:

```text
ProfileAccountPage {
  items: ProfileAccountSummary[],
  cursor?: string,
  totalCount: int
}

ProfileAccountSummary {
  did: string,
  handle: string,
  displayName?: string,
  description?: string,
  avatar?: string,
  isCraftskyProfile: bool
}
```

`totalCount` is included on list responses so settings list pages can put counts in app-bar titles without fetching `/v1/profiles/me` first. The response still follows the opaque pagination convention: `items` plus omitted `cursor` when exhausted.

### AppView store interface sketch
Extend or split the profile read surfaces as follows:

```text
type ProfileReader interface {
  Read(ctx, profileDID, viewerDID string) (*ProfileRow, error)
}

type ProfileGraphReader interface {
  ListMutualFollowers(ctx, viewerDID, profileDID string, limit int, cursor string) ([]*ProfileAccountRow, string, int, error)
  ListFollowers(ctx, subjectDID string, limit int, cursor string) ([]*ProfileAccountRow, string, int, error)
  ListFollowing(ctx, did string, limit int, cursor string) ([]*ProfileAccountRow, string, int, error)
}
```

`ProfileStore.Read` should populate:
- `FollowerCount` / `FollowingCount`: existing Craftsky-restricted count behavior.
- `MutualFollowerCount`: count `mutual.did` where `viewer -> mutual` and `mutual -> profile`, excluding self profile and non-Craftsky target profiles.
- `PostCount`: `COUNT(*) FROM craftsky_posts WHERE did=$1 AND reply_root_uri IS NULL AND reply_parent_uri IS NULL`.
- `PostsLast7Days`: same root-post predicate with `created_at >= now() - interval '7 days'`.
- `ProjectCount`: `0` from the server for now; keep the code path explicit so Flutter no longer hardcodes it.

### Query shapes and cursor keys
Reuse the existing opaque cursor helpers (`envelope.EncodeCursor`, `envelope.DecodeCursor`) and the keyset pattern from `PostStore`.

```text
// followers: accounts that follow subjectDID, newest follow first
WHERE f.subject_did = $subjectDID
  AND (cursor empty OR (f.created_at, f.uri) < ($createdAt, $uri))
ORDER BY f.created_at DESC, f.uri DESC
LIMIT $limit
cursor keys: { "createdAt": RFC3339Nano, "uri": followURI }

// following: accounts did follows, newest follow first
WHERE f.did = $did
  AND (cursor empty OR (f.created_at, f.uri) < ($createdAt, $uri))
ORDER BY f.created_at DESC, f.uri DESC
LIMIT $limit

// mutuals: viewer follows mutual, mutual follows profile
FROM atproto_follows viewer_follow
JOIN atproto_follows mutual_follow ON mutual_follow.did = viewer_follow.subject_did
WHERE viewer_follow.did = $viewerDID
  AND mutual_follow.subject_did = $profileDID
ORDER BY mutual_follow.created_at DESC, mutual_follow.uri DESC
```

Account rows should join `bluesky_profiles` for display fields and `craftsky_profiles` for `isCraftskyProfile`. Handle resolution can be performed per unique DID in the handler using `HandleResolver`, matching existing response-building patterns.

### Index migration guidance
If the initial implementation uses these ordered predicates, create `000013_profile_social_summary_indexes` with indexes such as:

```text
CREATE INDEX atproto_follows_subject_created_uri_desc_idx
  ON atproto_follows (subject_did, created_at DESC, uri DESC);

CREATE INDEX atproto_follows_did_created_uri_desc_idx
  ON atproto_follows (did, created_at DESC, uri DESC);

CREATE INDEX craftsky_posts_root_did_created_idx
  ON craftsky_posts (did, created_at DESC)
  WHERE reply_root_uri IS NULL AND reply_parent_uri IS NULL;
```

Guardrail: do not add tables or project persistence in this slice.

## 6. State, Providers, Controllers, Or DI
### AppView DI
- Reuse `deps.ProfileStore` for both summary reads and graph list reads.
- Add handler constructors in `profile.go` that accept the narrow interfaces above for testability.
- Register routes in `routes.go` using the same `authN(deviceID(...))` middleware order as existing profile routes.

### Flutter Riverpod provider graph
Use existing repository-provider style and generated Riverpod providers.

```text
dioProvider
  -> profileApiClientProvider
    -> profileRepositoryProvider
      -> userProfileProvider(handleOrDid)        // existing summary/profile
      -> profileAccountListProvider(request)    // new paginated graph lists

authSessionProvider
  -> ProfilePage.isOwnProfile decisions          // existing
  -> settings follow-list route/page labels       // signed-in self only
```

Suggested model/provider shapes:

```text
enum ProfileAccountListKind { mutualFollowers, followersMe, followingMe }

class ProfileAccountListRequest {
  ProfileAccountListKind kind;
  String? targetHandleOrDid; // required only for mutualFollowers
}

@riverpod
class ProfileAccountList extends _$ProfileAccountList {
  Future<ProfileAccountListState> build(ProfileAccountListRequest request)
  Future<void> loadMore()
}
```

Provider behavior should mirror `UserPosts`:
- First page loads in `build`.
- `loadMore()` no-ops when loading, no data, or no cursor.
- Use `AsyncLoading<T>().copyWithPrevious(state)` so lists stay visible while appending or retrying.
- Keep cursor opaque; do not parse it client-side.

## 7. UI, Widgets, Routes, Or User-Facing Surfaces
### Profile page composition
Change `ProfileMetaSection` to accept `isOwnProfile` and compose:

```text
ProfileMetaSection(profile, isOwnProfile)
  non-Craftsky marker if !profile.isCraftskyProfile
  ProfileBio if present
  ProfileCraftChips if present
  ProfileMutualFollowersLink if !isOwnProfile && profile.mutualFollowerCount > 0
  ProfileStats(profile)
```

`ProfileStats` should no longer accept `followingCount` or `followerCount`. It should render only available new stats:
- `Joined <age> ago` for Craftsky profiles with `createdAt`.
- `<X> posts in the last 7 days` from `postsLast7Days`.
- `<X> posts` from `postCount`.
- `<X> projects` from `projectCount`.

For non-Craftsky profiles, hide account age. If post/project summary values are omitted, do not invent Craftsky stats.

### Account-age formatter thresholds
Create a small formatter near profile widgets (or a lightweight utility) with testable thresholds:

```text
Duration age = now - createdAt
if age < 24h: "Joined less than 1 day ago"
else if days < 30: "Joined N day(s) ago"
else if days < 365: "Joined N month(s) ago" where N = max(1, floor(days / 30))
else: "Joined N year(s) ago" where N = max(1, floor(days / 365))
```

Use injectable/fixed `now` in unit tests rather than relying on wall-clock timing. Keep pluralization deterministic in tests.

### Mutual followers bottom sheet
`ProfileMutualFollowersLink` should render uncapped text such as `12 mutual followers` only when `mutualFollowerCount > 0`. On tap:

```text
showModalBottomSheet(
  context: context,
  isScrollControlled: true,
  builder: (_) => FractionallySizedBox(
    heightFactor: 0.9,
    child: ProfileMutualFollowersSheet(targetHandleOrDid: profile.handle.toString()),
  ),
)
```

The sheet uses `profileAccountListProvider(ProfileAccountListRequest.mutualFollowers(...))` and renders account rows via the reusable `ProfileAccountList` widget.

### Settings and routes
Add settings entries without counts:
- `Followers`
- `Following`

Add typed routes under the existing profile/settings route area. Suggested locations:

```text
RouteLocations.followersChild = 'followers'
RouteLocations.followingChild = 'following'

/profile/settings/followers
/profile/settings/following
```

Use root navigator keys like `SettingsRoute` so list pages cover the shell bottom navigation and back returns to settings.

List page app-bar titles should include `totalCount`, for example:
- `Followers (12)`
- `Following (7)`

Rows should display `displayName ?? handle`, `@handle`, avatar placeholder using existing `ProfileAvatar`, and optionally navigate to `UserProfileRoute(handle: row.handle)` on tap.

## 8. Error, Loading, Empty, And Edge States
| State / Case | Planned Handling | Requirement IDs | Test IDs |
|---|---|---|---|
| Profile fetch loading/error | Preserve existing `StitchProgressIndicator` and `ProfilePageError` behavior | REG-001 | REG-001 |
| Follower/following counts in profile response | Keep decoding and serializing, but remove visible profile/settings-entry rendering | FR-001, FR-012 | AT-001, AT-005, IT-003, UT-005, UT-006, REG-008 |
| Zero mutual followers | Do not render mutual follower link/section | FR-014 | AT-008, UT-007 |
| Self profile | Do not render mutual follower section even if response includes `0` or omits field | FR-002 | AT-002 |
| Mutual list loading | Bottom sheet shows centered `StitchProgressIndicator` on first page load | FR-013, FR-016 | AT-003 |
| Mutual/list pagination loading | Keep existing rows visible and show bottom spinner/retry, mirroring `UserPosts` | FR-011, FR-016 | UT-010, UT-012 |
| Empty followers | Show `No one follows you yet` | FR-014 | AT-008, UT-008 |
| Empty following | Show `You are not following anyone` | FR-014 | AT-008, UT-008 |
| Non-Craftsky profile | Show marker; hide account age and any omitted Craftsky-only stats | FR-017 | AT-009, UT-006, REG-003 |
| Invalid graph-list cursor | AppView returns `400 invalid_cursor` using existing error envelope | NFR-001 | IT-004, IT-005, IT-006 |
| Missing auth/device header | Routes use existing middleware and error envelope | NFR-001 | IT-007, REG-007 |
| List account lacks optional display data | Render stable DID/handle with null display/avatar handled gracefully | FR-010 | AC-011, UT-011 |
| Posts exactly at 7-day boundary | Count with `created_at >= now() - interval '7 days'`; seed deterministic test timestamps | RULE-003 | UT-003, IT-001 |

## 9. Test Implementation Plan
| Order | Test ID | Target | Setup / Fixture | Initial Expected Failure |
|---|---|---|---|---|
| 1 | IT-001, UT-002, UT-003 | `appview/internal/api/profile_store_test.go` | Extend `profileStoreDDL` with `craftsky_posts`; seed root posts, replies, outside-window rows | `ProfileRow` lacks `postCount`, `postsLast7Days`, `projectCount`; query does not count roots |
| 2 | IT-002, UT-001 | `appview/internal/api/profile_store_test.go` | Seed viewer/profile/mutual/non-mutual `craftsky_profiles` and `atproto_follows` rows | No mutual-count field/query |
| 3 | IT-003, UT-005, REG-008 | `appview/internal/api/profile_response_test.go`, `profile_test.go` | Build rows with old and new counts | JSON lacks new fields or accidentally embeds mutual previews |
| 4 | IT-004 | `appview/internal/api/profile_store_test.go`, `profile_test.go` | Seed > limit mutuals with display rows and follow times | No mutual followers endpoint/list method |
| 5 | IT-005, IT-006 | `appview/internal/api/profile_store_test.go`, `profile_test.go` | Seed followers/following with distinct `created_at`; include > limit | No self graph endpoints; no ordering/cursor support |
| 6 | IT-007, REG-007 | `appview/internal/routes/routes_test.go` | Call new routes unauthenticated and without device ID | Routes not registered/protected |
| 7 | IT-008 | `appview/internal/api/profile_store_test.go` | Request defaults/max limits; inspect deterministic bounded behavior | Limit cap/query may be missing |
| 8 | UT-011 | `app/test/profile/models/profile_test.dart`, `app/test/profile/data/profile_api_client_test.dart` | JSON with `mutualFollowerCount`, `postCount`, `postsLast7Days`, `projectCount`, account summaries | Flutter models do not decode new fields |
| 9 | UT-012 | `app/test/profile/data/profile_api_client_test.dart` | Mock Dio adapter; call each list method with limit/cursor | API client lacks routes/query params/page decode |
| 10 | UT-004, UT-006, AT-001, AT-004, AT-009 | `app/test/profile/widgets/profile_stats_test.dart` or `profile_page_test.dart` | Craftsky/non-Craftsky `Profile` models with fixed dates and counts | Old stats render followers/following and hardcoded projects |
| 11 | UT-007, AT-002, AT-003 | `app/test/profile/widgets/profile_mutual_followers_test.dart`, `profile_page_test.dart` | Visitor profile with count 12 and fake list repository | No clickable mutual link/bottom sheet/list load |
| 12 | UT-009, AT-005, REG-005 | `app/test/settings/settings_page_test.dart` | Render settings | Social entries absent or counts accidentally shown |
| 13 | UT-008, UT-010, AT-006, AT-007, AT-008 | `app/test/settings/followers_page_test.dart`, `following_page_test.dart` | Fake repository returns ordered rows, total counts, empty pages | Pages/providers/routes do not exist |
| 14 | REG-001-REG-006 | Existing profile/post/follow tests | Existing fakes extended for new interface methods | Existing identity/actions/tabs/follow writes regress |

Focused commands:
```text
# AppView focused
cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes

# AppView full repo wrapper (requires compose Postgres/dev stack as documented)
just test

# Flutter focused
cd app && flutter test test/profile/profile_page_test.dart test/profile/data/profile_api_client_test.dart test/settings/settings_page_test.dart

# Flutter full
cd app && flutter test
```

## 10. Sequencing And Guardrails
- First TDD step: `IT-001` in `appview/internal/api/profile_store_test.go` for root-post counts and data-driven `projectCount`.
- Backend sequence:
  1. Extend AppView store row/queries and response DTOs.
  2. Add graph list store methods with keyset cursors and account-summary rows.
  3. Add handlers/routes/auth tests.
  4. Add optional index migration if query shapes require ordered support.
- Flutter sequence:
  1. Extend profile and account-page models plus generated mappers.
  2. Extend client/repository/fakes and providers.
  3. Update profile stats/mutual UI.
  4. Add settings entries/routes/list pages.
  5. Update l10n and generated files.
- Dependencies between work items:
  - Flutter model/client tests depend on finalized JSON field names and route paths in §5.
  - Widget tests depend on repository/fake extensions for account pages.
  - Settings list pages depend on `ProfileAccountList` provider and route generation.
- Guardrails:
  - Do not query the PDS from Flutter for graph, profile, or post summary data.
  - Do not store PDS tokens on the device.
  - Do not change lexicon schemas or generated lexicon Go types.
  - Do not remove `followerCount` or `followingCount` from AppView responses.
  - Do not render follower/following counts on profile pages or settings entry page.
  - Do not include replies/comments, reposts, likes, or follows in `postCount` or `postsLast7Days`.
  - Keep cursors opaque and omit `cursor` when exhausted.
  - Keep list limits bounded; prefer default 50/max 100 unless a narrower per-endpoint cap is intentionally documented.
- Out of scope:
  - Project persistence beyond returning a real `projectCount` value of `0`.
  - Blocks/mutes filtering.
  - Public unauthenticated graph access.
  - Discovery CTAs for empty lists.
  - Follow/unfollow write semantics changes.

## 11. Risks And Open Questions
| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Non-blocking | Mutual/follower list queries may be expensive for large graphs | Slow profile/list loads | Use bounded pagination and ordered indexes in §5; verify with IT-008/MAN-002 |
| CPQ-002 | Non-blocking | List response includes `totalCount` beyond base `items/cursor` shape | Slight API surface expansion | Chosen here to satisfy app-bar counts without extra profile fetch; keep camelCase and opaque cursor semantics |
| CPQ-003 | Non-blocking | Account-age copy thresholds were unspecified | Test drift if not fixed | Thresholds finalized in §7; lock with UT-004 |
| CPQ-004 | Non-blocking | Non-Craftsky stats can be misinterpreted | UI could invent Craftsky data for external profiles | Omit/hide Craftsky-only stats when response fields are absent; explicitly test non-Craftsky age hiding |
| CPQ-005 | Non-blocking | Route generation and mapper/l10n generation must be kept in sync | Compile failures if generated files are stale | TDD builder should run project generation commands used locally after model/router/l10n changes |

Blocking open questions: None.

## 12. Handoff To TDD Builder
- Coding plan: `docs/changes/2026-05-27-profile-social-summary/04-coding-plan.md`
- TDD execution plan: `05-implementation-plan.md`
- Start with test: `IT-001` in `appview/internal/api/profile_store_test.go`.
- Focused command: `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes`
- Notes:
  - Treat §5 route paths and JSON field names as the API contract for tests.
  - Preserve old profile count fields in AppView while removing their visible Flutter profile/settings-entry presentation.
  - Add generated Flutter artifacts (`*.mapper.dart`, `*.g.dart`, localizations, route generated file) only as required by source changes during implementation.
