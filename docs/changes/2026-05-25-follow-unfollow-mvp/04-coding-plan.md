# Coding Plan: Follow / Unfollow MVP

## 1. Inputs

- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md`

## 2. Implementation Strategy

Implement the AppView follow graph first, then profile reads, follow/unfollow handlers, and finally Flutter UI wiring. This matches the accepted test order and the existing codebase shape: AppView uses raw SQL migrations, small handler-specific interfaces, typed atproto identifiers at boundaries, `net/http` route registration, and indexers registered from `appview/internal/app/deps.go`; Flutter uses `ProfileApiClient` -> `ProfileRepository` -> Riverpod providers -> profile widgets.

Counts and relationship state split deliberately:

- Store all active `app.bsky.graph.follow` relationships needed for relationship state and future feed lookup, including non-Craftsky targets.
- Count only relationships where both follower and target have `craftsky_profiles` rows.
- Return nullable/omitted counts for non-Craftsky profile pages.
- Never let Flutter talk to a PDS or receive PDS tokens.

Document-review decisions resolved here:

- Follow validation error codes: `invalid_identifier`, `identity_unavailable`, `self_follow_not_allowed`, `pds_unavailable`, `pds_write_failed`.
- Craftsky count calculation error code: `profile_counts_unavailable`.
- Non-Craftsky profile hydration: first read `bluesky_profiles`; if absent, AppView hydrates `app.bsky.actor.profile/self` through the existing anonymous PDS client and stores the same normalized fields in `bluesky_profiles` without requiring `craftsky_profiles` membership.

## 3. Affected Areas

| Area | Existing Pattern | Planned Change | Requirement IDs | Test IDs |
|---|---|---|---|---|
| Follow graph persistence | Raw SQL migrations and Postgres integration tests | Add active follow table without `craftsky_profiles` FKs; hard-delete on tombstone/unfollow | FR-001, RULE-003, RULE-007 | UT-004, UT-005, UT-006, IT-001 |
| Follow indexer | One indexer per NSID, registered in `deps.go` | Add `app.bsky.graph.follow` indexer and Tap filter registration | FR-002, FR-010, NFR-002 | AT-008, IT-003, IT-008, IT-012, MAN-001 |
| Profile read model | `ProfileStore.Read` joins `craftsky_profiles` to `bluesky_profiles` | Read Craftsky and non-Craftsky profiles, calculate counts/viewer state, hydrate non-members | BR-002, BR-005, FR-006, FR-011, FR-012, RULE-005, RULE-008, RULE-009 | AT-004, AT-005, IT-002, IT-006, IT-007, IT-011, UT-008, UT-009, UT-012, UT-013 |
| Follow API handlers | Handler factories take narrow dependencies and PDS factory | Add POST/DELETE profile follow endpoints returning updated `ProfileResponse` | BR-001, FR-003, FR-004, FR-005, NFR-001, NFR-003 | AT-001, AT-002, AT-003, IT-004, IT-005, IT-009, IT-010, UT-001, UT-002, UT-003, UT-010, UT-011 |
| Routes and deps | `routes.AddRoutes` wires handlers from `app.Deps` | Add follow store/deps and route auth/device protection | NFR-001 | IT-009, REG-001, REG-002 |
| Flutter profile model/API/repository | `ProfileApiClient`, `ApiProfileRepository`, `ProfileRepository` | Add follow fields and follow/unfollow methods returning `Profile` | FR-007, NFR-003 | UT-014, UT-015, REG-004 |
| Flutter profile state | `UserProfile` is keyed profile cache; mutation providers handle loading/errors elsewhere | Add follow toggle mutation provider that updates `userProfileProvider(handle)` from AppView response | FR-008, FR-009 | AT-001, AT-002, AT-006, UT-017, UT-018 |
| Flutter UI | `ProfileActions`, `ProfileMetaSection`, `ProfileStats` | Render real labels/counts, loading state, and `Non Craftsky profile` marker | FR-008, RULE-008 | AT-001, AT-005, AT-007, UT-016, REG-005, REG-006 |

## 4. Files And Modules

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `appview/migrations/000012_atproto_follows.up.sql` | Create | Active follow graph table and indexes | FR-001, NFR-004, RULE-003, RULE-007 | IT-001, IT-002, MAN-003 |
| `appview/migrations/000012_atproto_follows.down.sql` | Create | Drop follow graph table | FR-001 | IT-001 |
| `appview/internal/index/bluesky_follow.go` | Create | Index `app.bsky.graph.follow` create/update/delete events | FR-002, FR-010, NFR-002 | UT-004, UT-005, UT-006, IT-003, IT-008 |
| `appview/internal/index/bluesky_follow_test.go` | Create | Unit tests for follow indexer behavior | FR-002, NFR-002 | UT-004, UT-005, UT-006, IT-003, IT-008 |
| `appview/internal/app/deps.go` | Change | Register follow indexer and expose anonymous PDS/profile store dependencies | FR-002, FR-012 | IT-012 |
| `appview/internal/app/indexer_wiring_test.go` | Change | Cover follow dispatcher registration | FR-002 | IT-012, REG-007 |
| `docker-compose.yml` | Change | Add `app.bsky.graph.follow` to `TAP_COLLECTION_FILTERS` | FR-010 | IT-012, MAN-001 |
| `appview/internal/api/profile_store.go` | Change | Add profile fields, counts, viewer state, non-Craftsky read/hydrate path, active followed lookup helpers if kept here | FR-001, FR-006, FR-011, FR-012, RULE-005, RULE-008, RULE-009 | IT-002, IT-006, IT-007, UT-007, UT-008, UT-009, UT-012, UT-013 |
| `appview/internal/api/profile_response.go` | Change | Add `viewerIsFollowing`, `isCraftskyProfile`, nullable count fields | FR-006 | UT-008, UT-012 |
| `appview/internal/api/profile.go` | Change | Pass viewer DID into reads; return count/hydration errors with chosen envelopes | FR-006, RULE-006, RULE-009 | IT-006, IT-007 |
| `appview/internal/api/follow.go` | Create | POST/DELETE follow handlers and validation | FR-003, FR-004, FR-005, NFR-001, NFR-003, RULE-001, RULE-002, RULE-003, RULE-004 | UT-001, UT-002, UT-003, UT-010, UT-011, IT-004, IT-005 |
| `appview/internal/api/follow_store.go` | Create | Handler-side active follow lookup/upsert/delete methods | FR-001, FR-003, FR-004, RULE-003, RULE-004, RULE-007 | UT-007, UT-010, UT-011, IT-001 |
| `appview/internal/api/follow_test.go` | Create | Handler and follow store tests | FR-003, FR-004, FR-005, NFR-001, NFR-003 | UT-001, UT-002, UT-003, UT-010, UT-011, IT-004, IT-005, IT-010 |
| `appview/internal/routes/routes.go` | Change | Register POST/DELETE follow endpoints under auth/device middleware | NFR-001 | IT-009, REG-001 |
| `appview/internal/routes/routes_test.go` | Change | Assert follow routes require auth/device and keep error envelope conventions | NFR-001 | IT-009, REG-001, REG-002 |
| `appview/internal/index/bluesky_profile.go` | Change | Stop dropping non-member `app.bsky.actor.profile` events | FR-012 | UT-013, IT-011 |
| `appview/internal/index/bluesky_profile_test.go` | Change | Replace non-member drop expectation with non-member cache expectation | FR-012 | UT-013, IT-011 |
| `app/lib/profile/models/profile.dart` and `profile.mapper.dart` | Change | Add follow/count/profile-type fields and regenerate mapper | FR-007 | UT-015 |
| `app/lib/profile/data/profile_api_client.dart` | Change | Add `followProfile` and `unfollowProfile` | FR-007, NFR-003 | UT-014 |
| `app/lib/profile/data/profile_repository.dart` | Change | Add follow/unfollow contract | FR-007 | UT-014 |
| `app/lib/profile/data/api_profile_repository.dart` | Change | Delegate follow/unfollow to API client | FR-007 | UT-014 |
| `app/lib/profile/providers/toggle_follow_profile_provider.dart` and generated file | Create | Mutation provider for loading/error and cache updates | FR-008, FR-009 | UT-017, UT-018 |
| `app/lib/profile/providers/user_profile_provider.dart` | Change | Keep `setCached`; use it from follow toggle provider | FR-008 | UT-017, UT-018 |
| `app/lib/profile/widgets/profile_actions.dart` | Change | Support disabled/loading state and `Unfollow` label | FR-008 | UT-016, UT-017 |
| `app/lib/profile/widgets/profile_meta_section.dart` | Change | Use real follower/following counts; render non-Craftsky marker | FR-006, FR-008, RULE-008 | UT-016, REG-006 |
| `app/lib/profile/widgets/profile_stats.dart` | Change only if needed | Existing nullable counts already render unknown; keep API if sufficient | FR-006, RULE-008 | UT-016 |
| `app/lib/profile/pages/profile_page.dart` | Change | Wire follow toggle provider, error messaging, and response-driven cache updates | FR-008, FR-009 | AT-001, AT-002, AT-006, UT-017, UT-018 |
| `app/lib/l10n/app_en.arb` and generated localization files | Change | Change followed label to `Unfollow`; add non-Craftsky marker if needed | FR-008, BR-005 | UT-016 |
| `app/test/profile/*` | Change/Create | Flutter model/API/provider/widget coverage | FR-007, FR-008, FR-009 | UT-014, UT-015, UT-016, UT-017, UT-018, AT-001..AT-007 |

## 5. Services, Interfaces, And Data Flow

### AppView Follow Graph

Use a table named `atproto_follows` to avoid implying Craftsky-only records.

```text
atproto_follows
- uri TEXT PRIMARY KEY
- did TEXT NOT NULL                 // follower DID / record repo
- rkey TEXT NOT NULL
- cid TEXT NOT NULL
- subject_did TEXT NOT NULL          // app.bsky.graph.follow subject
- record JSONB NOT NULL
- created_at TIMESTAMPTZ NOT NULL
- indexed_at TIMESTAMPTZ NOT NULL DEFAULT now()

Indexes/constraints:
- UNIQUE (did, rkey)
- UNIQUE (did, subject_did)
- INDEX (did)
- INDEX (subject_did)
- INDEX (did, subject_did)
- INDEX (uri, rkey) if query plans need it; uri is already primary key
```

No foreign keys to `craftsky_profiles`: follows involving non-Craftsky accounts must be stored when needed for relationship state.

Indexer record shape:

```text
type bskyFollowRecord struct {
  Subject string `json:"subject"`
  CreatedAt string `json:"createdAt"`
}
```

Indexer behavior:

```text
Handle(create/update):
  if collection != app.bsky.graph.follow: return nil
  parse subject DID and createdAt
  begin tx
  delete other local active rows for same follower DID + subject DID where uri != event uri
  insert/update uri row with cid, rkey, subject DID, record, createdAt, indexedAt now
  commit

Handle(delete):
  delete from atproto_follows where uri = event uri
```

### AppView Profile Reads

Change the read interface to include viewer DID:

```text
type ProfileReader interface {
  Read(ctx, profileDID, viewerDID string) (*ProfileRow, error)
}
```

Extend `ProfileRow`:

```text
ProfileRow:
- DID string
- Crafts []string
- CreatedAt *time.Time
- DisplayName/Description/Avatar/Banner fields
- IsCraftskyProfile bool
- FollowerCount *int
- FollowingCount *int
- ViewerIsFollowing bool
```

Read behavior:

```text
Read(profileDID, viewerDID):
  load craftsky_profiles left join bluesky_profiles
  if craftsky row exists:
    count followers by joining atproto_follows.did to craftsky_profiles.did
    count following by joining atproto_follows.subject_did to craftsky_profiles.did
    viewerIsFollowing = exists atproto_follows where did=viewerDID and subject_did=profileDID, except self false
    return counts as non-null pointers, isCraftskyProfile=true

  load bluesky_profiles for profileDID
  if absent, hydrate app.bsky.actor.profile/self through anonymous PDS client and upsert bluesky_profiles
  if still absent/unhydratable, return ErrProfileNotFound or wrapped hydrate error
  viewerIsFollowing = exists atproto_follows where did=viewerDID and subject_did=profileDID, except self false
  return empty crafts, nil createdAt, nil counts, isCraftskyProfile=false
```

Count-query failures for Craftsky profiles should return an error the handler maps to `profile_counts_unavailable`. Non-Craftsky missing counts are not an error.

### AppView Follow Handlers

Create `api.FollowStore` methods for handler-side active graph operations:

```text
type FollowRow struct {
  URI string
  DID string
  Rkey string
  CID string
  SubjectDID string
  CreatedAt time.Time
}

FindActiveFollow(ctx, followerDID, subjectDID string) (*FollowRow, error)
UpsertActiveFollow(ctx, row FollowRow, record json.RawMessage) error
DeleteActiveFollowByURI(ctx, uri string) error
ListActiveFollowedDIDs(ctx, followerDID string) ([]string, error)
```

POST data flow:

```text
resolve target handle/DID
reject self with self_follow_not_allowed
if local active row exists: return updated target ProfileResponse
build app.bsky.graph.follow record with subject target DID and createdAt UTC
new PDS client from authenticated DID + OAuth session ID
CreateRecord(repo=viewerDID, collection=app.bsky.graph.follow, record)
upsert returned uri/cid into atproto_follows before responding
read target profile with viewer DID
return 200 ProfileResponse
```

DELETE data flow:

```text
resolve target handle/DID
reject self with self_follow_not_allowed
find active local row by viewer DID + target DID
if none: return updated target ProfileResponse
new PDS client
DeleteRecord(repo=viewerDID, collection=app.bsky.graph.follow, rkey=active.rkey)
if ErrRecordNotFound: treat as success
delete local active row by uri
read target profile with viewer DID
return 200 ProfileResponse
```

### Flutter Data Flow

Add to `Profile`:

```text
final bool viewerIsFollowing;
final bool isCraftskyProfile;
final int? followingCount;
final int? followerCount;
```

Add API/repository calls:

```text
Future<Profile> followProfile(String handleOrDid)
  POST /v1/profiles/@$handleOrDid/follows

Future<Profile> unfollowProfile(String handleOrDid)
  DELETE /v1/profiles/@$handleOrDid/follows
```

Mutation provider sketch:

```text
@riverpod
class ToggleFollowProfile extends _$ToggleFollowProfile {
  FutureOr<Profile?> build() => null;

  Future<void> toggle({required String cacheKey, required Profile profile}) async {
    optimistic = profile.copyWith(viewerIsFollowing: !profile.viewerIsFollowing, count adjusted only if non-null and isCraftskyProfile)
    ref.read(userProfileProvider(cacheKey).notifier).setCached(optimistic)
    try:
      updated = profile.viewerIsFollowing ? repo.unfollow(cacheKey) : repo.follow(cacheKey)
      setCached(updated)
      state = AsyncData(updated)
    catch:
      setCached(profile)
      state = AsyncError(error, stack)
  }
}
```

The final state must come from the AppView response. The optimistic local count is only a transient loading affordance and should be skipped when counts are null.

## 6. State, Providers, Controllers, Or DI

### AppView DI

- Add `FollowStore *api.FollowStore` to `app.Deps`.
- Add `AnonymousPDS auth.PDSClient` or a narrower profile-hydrator dependency to `app.Deps` if `ProfileStore` needs hydration outside `newDeps`.
- Construct `api.NewProfileStore(pool, anonPDS)` or `api.NewProfileStore(pool, profileHydrator)`.
- Keep handler constructors narrow: pass `ProfileStore`, `FollowStore`, resolver, PDS factory, logger.

### Flutter Providers

Provider graph:

```text
profileApiClientProvider -> profileRepositoryProvider -> userProfileProvider(handleOrDid)
profileRepositoryProvider -> toggleFollowProfileProvider
toggleFollowProfileProvider -> userProfileProvider(handleOrDid).notifier.setCached
```

Use the existing generated Riverpod pattern:

- Add `toggle_follow_profile_provider.dart` with `part 'toggle_follow_profile_provider.g.dart';`.
- Run code generation after model/provider changes.
- In `ProfilePage`, watch `toggleFollowProfileProvider` for loading and listen for `AsyncError` to show existing messenger/snackbar behavior.

## 7. UI, Widgets, Routes, Or User-Facing Surfaces

### AppView Routes

Register under existing auth/device middleware:

```text
POST   /v1/profiles/{handleOrDid}/follows
DELETE /v1/profiles/{handleOrDid}/follows
```

The client still calls the existing convention with an `@` prefix in the path segment.

### Flutter UI

- `ProfileActions`: add `isBusy` or equivalent to `VisitorProfileActionSet`; disable the follow button and optionally show the existing button loading affordance while mutation is in flight.
- Button labels: `Follow` when `viewerIsFollowing=false`; `Unfollow` when true. Update `profileFollowingAction` from `Following` to `Unfollow` or add a new localization key.
- `ProfileMetaSection`: pass `profile.followingCount` and `profile.followerCount` to `ProfileStats`; keep project-count behavior explicitly separate from follow work.
- Non-Craftsky marker: render exact text `Non Craftsky profile` when `profile.isCraftskyProfile == false`, near identity/meta content so it is visible before tab content.
- Self profile: continue to render edit/settings actions; `viewerIsFollowing` is present and false but not shown as a follow action.

## 8. Error, Loading, Empty, And Edge States

| State / Case | Planned Handling | Requirement IDs | Test IDs |
|---|---|---|---|
| Invalid handle/DID | `400 invalid_identifier`; no PDS call | RULE-001, NFR-001 | UT-001, UT-003, IT-004, IT-005 |
| Resolver failure/current handle unavailable | `502 identity_unavailable`; no PDS call | RULE-001, NFR-001 | UT-001, UT-003, IT-004, IT-005 |
| Self follow/unfollow | `400 self_follow_not_allowed`; no PDS call | RULE-002 | AT-007, UT-002, IT-004, IT-005 |
| Missing auth/device ID | Existing middleware behavior | NFR-001 | IT-009, REG-001 |
| PDS client creation failure | `502 pds_unavailable` | FR-005 | UT-003, IT-004, IT-005 |
| PDS create/delete failure | `502 pds_write_failed`; Flutter rolls back | FR-005, FR-009 | AT-006, UT-003, UT-018, IT-004, IT-005 |
| PDS delete says record missing | Treat as idempotent success; delete local row | RULE-004 | UT-011, IT-005 |
| Duplicate indexer events | Upsert/idempotent; one active row | NFR-002, RULE-003 | UT-004, UT-005, IT-001 |
| Follow delete/tombstone | Hard-delete local row; unknown delete is no-op | RULE-007 | UT-006, IT-001 |
| Craftsky profile count query failure | `500 profile_counts_unavailable`; no fake counts | RULE-009 | UT-009, IT-006 |
| Non-Craftsky profile without counts | Return nil/omitted count fields; UI shows unknown/omits, no fake values | RULE-008 | AT-005, UT-013, UT-016, IT-007 |
| Non-Craftsky profile hydration unavailable | Return documented profile-load error; Flutter uses existing profile error UI | FR-011, FR-012 | IT-007, GAP-004 |
| Follow mutation in flight | Disable/loading button; prevent duplicate taps | FR-008 | AT-001, AT-002, UT-017 |
| Follow mutation fails in Flutter | Restore last confirmed profile and show existing error messaging | FR-009 | AT-006, UT-018 |

## 9. Test Implementation Plan

| Order | Test ID | Target | Setup / Fixture | Initial Expected Failure |
|---|---|---|---|---|
| 1 | UT-004 | `appview/internal/index/bluesky_follow_test.go` | Synthetic create event for `app.bsky.graph.follow` | No follow indexer/table exists |
| 2 | UT-005 | `appview/internal/index/bluesky_follow_test.go` | Update event for same URI with new CID/subject | Upsert behavior missing |
| 3 | UT-006 | `appview/internal/index/bluesky_follow_test.go` | Delete and unknown delete events | Hard-delete behavior missing |
| 4 | IT-001 | `appview/internal/api/follow_store_test.go` or migration/store test | Real Postgres schema and duplicate/delete rows | Migration/store missing |
| 5 | IT-002 | `appview/internal/api/profile_store_test.go` | Craftsky Alice/Bob/Carol plus non-Craftsky Dana/Erin | Counts include wrong relationships or fields missing |
| 6 | UT-007 | `appview/internal/api/follow_store_test.go` | Active follows for a viewer | Active followed DID query missing |
| 7 | UT-008 | `appview/internal/api/profile_response_test.go`, `profile_store_test.go` | Craftsky profile with viewer and graph rows | Response lacks count/viewer fields |
| 8 | UT-009 | `appview/internal/api/profile_test.go` | Inject count-store failure | Handler returns generic/fake success |
| 9 | IT-012 | `appview/internal/app/indexer_wiring_test.go`, config inspection | Dispatcher and compose filters | Follow NSID not registered/filter absent |
| 10 | UT-012, UT-013 | `profile_response_test.go`, `bluesky_profile_test.go`, hydration tests | Non-member cached/hydrated Bluesky profile | Membership gate drops non-member profile |
| 11 | IT-007, IT-011 | `profile_test.go`, `profile_store_test.go`, `bluesky_profile_test.go` | Non-Craftsky profile read/hydration | GET returns 404 for hydratable non-member |
| 12 | UT-001, UT-002, UT-003 | `appview/internal/api/follow_test.go` | Invalid, resolver failure, self target, PDS errors | Handler absent/error codes missing |
| 13 | UT-010, UT-011 | `appview/internal/api/follow_test.go` | Fake PDS create/delete and active row lookup | PDS write/delete behavior missing |
| 14 | IT-004, IT-005, IT-009, IT-010 | `follow_test.go`, `routes_test.go` | Handler with fake deps and route mux | Routes/handlers absent |
| 15 | UT-014, UT-015 | `app/test/profile/data/profile_api_client_test.dart`, `app/test/profile/models/profile_test.dart` | Dio adapter and profile JSON fixtures | Client/model fields missing |
| 16 | UT-016 | `app/test/profile/profile_page_test.dart`, widget tests | Craftsky and non-Craftsky profile models | UI still shows placeholder/fake stats/coming soon |
| 17 | UT-017, UT-018 | `app/test/profile/providers/toggle_follow_profile_provider_test.dart` or `profile_page_test.dart` | Fake repository delayed/error futures | No loading/rollback provider exists |
| 18 | AT-001..AT-008 | Acceptance-level widget/API/indexer tests | End-to-end component flows | Scenario assertions fail until all slices wired |
| 19 | MAN-001, MAN-002, MAN-003 | Dev stack/manual checks | `just dev-d` plus test DIDs/follow rows | Runtime Tap/filter/query-plan issues, if any |

## 10. Sequencing And Guardrails

- First TDD step: `UT-004` for follow indexer create idempotency against a temporary test schema containing `atproto_follows`.
- Dependencies between work items: migration/table before store/profile counts; indexer before dispatcher/Tap wiring; profile response fields before follow handlers return updated profiles; Flutter model before API/repository/provider/widget tests.
- Guardrails: do not add a Craftsky follow lexicon; do not store PDS tokens in Flutter; do not add `craftsky_profiles` FKs to follow graph; do not count non-Craftsky relationships in Craftsky profile counts; do not use deleted-history rows for follows in MVP; do not use fake numeric counts when AppView has no count.
- Out of scope: follower/following lists, timeline endpoint, notifications, blocks/mutes/reports, rate limiting, public-follow warning UI, global atproto count sourcing.

Implementation commands:

- Go focused: `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/index ./internal/api ./internal/routes ./internal/app`
- Go full: `just test` after `just dev-d` is running.
- Flutter codegen: `cd app && dart run build_runner build --delete-conflicting-outputs`
- Flutter focused: `cd app && flutter test test/profile test/shared/api`
- Flutter full: `cd app && flutter test`

## 11. Risks And Open Questions

| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Non-blocking | Multiple active PDS follow records can exist for the same follower/target pair outside Craftsky. | Unfollow may delete only the locally canonical active URI. | Collapse local counts/state to one row; keep duplicate cleanup as GAP-003/future work. |
| CPQ-002 | Non-blocking | Non-Craftsky profile hydration can fail if DID resolution/PDS read fails or profile record is absent. | A valid atproto account may be followable only after profile data becomes available. | Use `profile_not_found`/existing profile error UI; keep exact Flutter copy in implementation if changed. |
| CPQ-003 | Non-blocking | Optimistic Flutter count adjustment may momentarily differ from AppView response under Craftsky-only count rules. | Brief UI mismatch during mutation. | Always replace local state with the AppView response; skip optimistic count changes when counts are null. |
| CPQ-004 | Non-blocking | Removing the Bluesky profile membership gate increases cached profile rows. | More indexed/cache data than today. | Store only public Bluesky profile fields already in `bluesky_profiles`; no PDS tokens or private data. |

## 12. Handoff To TDD Builder

- Coding plan: `docs/changes/2026-05-25-follow-unfollow-mvp/04-coding-plan.md`
- TDD execution plan: `05-implementation-plan.md`
- Start with test: `UT-004` in `appview/internal/index/bluesky_follow_test.go`
- Focused command: `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/index`
- Notes: Keep commits small by vertical layer if possible: follow graph/indexer, profile reads, handlers/routes, Flutter model/data, Flutter provider/UI.
