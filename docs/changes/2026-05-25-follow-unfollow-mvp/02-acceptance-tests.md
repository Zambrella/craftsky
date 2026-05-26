# Acceptance Test Specification: Follow / Unfollow MVP

## 1. Test Strategy

This is a high-risk vertical slice touching AppView migrations, Tap/firehose indexing, PDS write mediation, profile reads, API contracts, and Flutter UI state. Use a test-first path that starts with the AppView follow graph store/indexer, then API handlers, then Flutter model/client/provider/widget behavior.

Primary automation targets:

- **AppView unit tests** for validation, response building, follow record decode/upsert/delete, and count/viewer-state logic.
- **AppView integration tests** against the existing Postgres test harness for migrations, active graph persistence, counts, profile hydration, API handlers, routes, and dispatcher wiring.
- **Flutter unit/widget tests** for model decoding, API client paths, repository/provider state transitions, Follow/Unfollow rendering, loading state, error recovery, non-Craftsky marker, and unknown non-Craftsky counts.
- **Manual checks** only for live Tap/PDS interoperability and end-to-end behavior that is impractical to prove in isolated tests.

Tap historical delivery note: existing project docs and `docker-compose.yml` state that Tap can add repos and backfill existing records when a repo is tracked; the user confirmed during document review follow-up that Tap will deliver historical data. This test plan still includes an explicit live Tap/manual smoke check because isolated tests can prove indexer behavior for synthetic historical events (`Live=false`) but cannot prove the deployed Tap sidecar's end-to-end runtime wiring without running the stack.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-002, AC-010, AC-020 | AT-001, AT-002, AT-003, IT-004, IT-005, UT-010, UT-011 | Acceptance / Integration / Unit | Yes |
| BR-002 | AC-003, AC-004, AC-011 | AT-004, IT-002, IT-006, UT-008, UT-012 | Acceptance / Integration / Unit | Yes |
| BR-003 | AC-005 | IT-001, UT-007 | Integration / Unit | Yes |
| BR-004 | AC-016, AC-017, AC-025 | AT-008, IT-003, IT-008, MAN-001 | Acceptance / Integration / Manual | Yes |
| BR-005 | AC-020, AC-021 | AT-003, AT-005, IT-007, UT-012, UT-013 | Acceptance / Integration / Unit | Yes |
| FR-001 | AC-005, AC-006, AC-007 | IT-001, IT-002, UT-004, UT-005, UT-006, UT-007 | Integration / Unit | Yes |
| FR-002 | AC-006, AC-007, AC-016, AC-017, AC-025 | IT-003, IT-008, UT-004, UT-005, UT-006, MAN-001 | Integration / Unit / Manual | Yes |
| FR-003 | AC-001, AC-008, AC-012, AC-013, AC-020, AC-022 | AT-001, AT-003, IT-004, IT-009, UT-001, UT-002, UT-010 | Acceptance / Integration / Unit | Yes |
| FR-004 | AC-002, AC-009, AC-012, AC-013, AC-020, AC-022 | AT-002, AT-003, IT-005, IT-009, UT-001, UT-002, UT-011 | Acceptance / Integration / Unit | Yes |
| FR-005 | AC-001, AC-002, AC-014 | IT-004, IT-005, IT-010, UT-010, UT-011, REG-004 | Integration / Unit / Regression | Yes |
| FR-006 | AC-003, AC-004, AC-011, AC-018, AC-021, AC-023 | AT-004, AT-005, AT-007, IT-006, IT-007, UT-008, UT-012, UT-013 | Acceptance / Integration / Unit | Yes |
| FR-007 | AC-010, AC-011, AC-014, AC-021 | AT-001, AT-003, AT-005, UT-014, UT-015, UT-016, REG-005 | Acceptance / Unit / Regression | Yes |
| FR-008 | AC-010, AC-011, AC-015, AC-022, AC-024 | AT-001, AT-002, AT-006, UT-016, UT-017, UT-018 | Acceptance / Unit | Yes |
| FR-009 | AC-015 | AT-006, UT-018 | Acceptance / Unit | Yes |
| FR-010 | AC-016, AC-017 | AT-008, IT-003, IT-008, MAN-001 | Acceptance / Integration / Manual | Yes |
| FR-011 | AC-020, AC-021 | AT-003, AT-005, IT-007, UT-012, UT-013 | Acceptance / Integration / Unit | Yes |
| FR-012 | AC-020, AC-021 | AT-003, AT-005, IT-007, IT-011, UT-013 | Acceptance / Integration / Unit | Yes |
| NFR-001 | AC-008, AC-009, AC-012, AC-013 | IT-009, REG-001, REG-002 | Integration / Regression | Yes |
| NFR-002 | AC-006, AC-007 | UT-004, UT-005, UT-006, IT-001 | Unit / Integration | Yes |
| NFR-003 | AC-014 | IT-010, UT-014, REG-004 | Integration / Unit / Regression | Yes |
| NFR-004 | AC-003, AC-004 | IT-006, MAN-003 | Integration / Manual | Partial |
| RULE-001 | AC-012, AC-020 | AT-003, IT-004, IT-005, IT-007, UT-001 | Acceptance / Integration / Unit | Yes |
| RULE-002 | AC-013 | AT-007, IT-004, IT-005, UT-002 | Acceptance / Integration / Unit | Yes |
| RULE-003 | AC-001, AC-006 | IT-001, IT-004, UT-004, UT-010 | Integration / Unit | Yes |
| RULE-004 | AC-002, AC-009, AC-013 | IT-005, UT-002, UT-011 | Integration / Unit | Yes |
| RULE-005 | AC-003, AC-004, AC-005, AC-007, AC-025 | AT-004, AT-008, IT-001, IT-002, IT-003, UT-007, UT-008 | Acceptance / Integration / Unit | Yes |
| RULE-006 | AC-018 | AT-007, IT-006, UT-008 | Acceptance / Integration / Unit | Yes |
| RULE-007 | AC-007, AC-019 | IT-001, UT-006 | Integration / Unit | Yes |
| RULE-008 | AC-023 | AT-005, UT-013, UT-016 | Acceptance / Unit | Yes |
| RULE-009 | AC-026 | IT-006, UT-009 | Integration / Unit | Yes |

## 3. Acceptance Scenarios

### AT-001: Follow a Craftsky profile from Flutter

Requirement IDs: BR-001, FR-003, FR-005, FR-007, FR-008, RULE-001, RULE-003  
Acceptance Criteria: AC-001, AC-010, AC-011, AC-022, AC-024  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/profile/profile_page_test.dart` plus `app/test/profile/providers/*follow*_test.dart`

```gherkin
Feature: Follow profiles
  Scenario: Visitor follows a Craftsky profile
    Given Alice is signed in to Craftsky
    And Bob is a Craftsky profile with viewerIsFollowing false
    When Alice opens Bob's profile
    Then the profile action shows "Follow"
    When Alice taps "Follow"
    Then the action enters a loading or disabled state
    And Flutter calls POST /v1/profiles/@bob.example/follows
    When the AppView returns Bob's updated profile with viewerIsFollowing true
    Then the profile action shows "Unfollow"
    And Bob's visible Craftsky follower count reflects the AppView response
```

### AT-002: Unfollow a profile from Flutter

Requirement IDs: BR-001, FR-004, FR-005, FR-007, FR-008, RULE-004  
Acceptance Criteria: AC-002, AC-009, AC-022, AC-024  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/profile/profile_page_test.dart` plus `app/test/profile/providers/*follow*_test.dart`

```gherkin
Feature: Unfollow profiles
  Scenario: Visitor unfollows a profile they already follow
    Given Alice is signed in to Craftsky
    And Bob's profile has viewerIsFollowing true
    When Alice opens Bob's profile
    Then the profile action shows "Unfollow"
    When Alice taps "Unfollow"
    Then the action enters a loading or disabled state
    And Flutter calls DELETE /v1/profiles/@bob.example/follows
    When the AppView returns Bob's updated profile with viewerIsFollowing false
    Then the profile action shows "Follow"
    And Bob's visible counts reflect the AppView response
```

### AT-003: Follow a non-Craftsky atproto account

Requirement IDs: BR-001, BR-005, FR-003, FR-004, FR-011, FR-012, RULE-001  
Acceptance Criteria: AC-001, AC-002, AC-020, AC-021, AC-022  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/profile/profile_page_test.dart`, `appview/internal/api/profile_test.go`, `appview/internal/api/follow_test.go`

```gherkin
Feature: Non-Craftsky profiles
  Scenario: Visitor follows and unfollows a non-Craftsky account
    Given Alice is signed in to Craftsky
    And Carol is a resolvable atproto account without a Craftsky profile row
    And Carol has an app.bsky.actor.profile display name and avatar
    When Alice opens Carol's profile in Craftsky
    Then Carol's Bluesky profile information is shown
    And the page shows "Non Craftsky profile"
    And the profile action shows "Follow"
    When Alice follows Carol
    Then the AppView writes an app.bsky.graph.follow record through Alice's PDS session
    And the AppView returns Carol's updated profile response
    When Alice unfollows Carol
    Then the AppView deletes the known active follow record through Alice's PDS session
    And the AppView returns Carol's updated profile response
```

### AT-004: Craftsky profile counts and relationship state come from indexed graph

Requirement IDs: BR-002, FR-001, FR-006, RULE-005  
Acceptance Criteria: AC-003, AC-004, AC-011  
Priority: Must  
Level: Acceptance  
Automation Target: `appview/internal/api/profile_store_test.go`, `appview/internal/api/profile_test.go`, `app/test/profile/profile_page_test.dart`

```gherkin
Feature: Craftsky profile stats
  Scenario: Profile stats show active indexed atproto follows
    Given Bob is a Craftsky profile
    And Alice and Dana actively follow Bob in the indexed graph
    And Bob actively follows Carol in the indexed graph
    When an authenticated viewer opens Bob's profile
    Then Bob's followerCount is 2
    And Bob's followingCount is 1
    And Flutter renders those values rather than placeholder stats
```

### AT-005: Non-Craftsky profile counts are unknown rather than fake

Requirement IDs: BR-005, FR-006, FR-011, RULE-008  
Acceptance Criteria: AC-021, AC-023  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/profile/profile_page_test.dart`, `app/test/profile/widgets/profile_stats_test.dart` if added

```gherkin
Feature: Non-Craftsky profile stats
  Scenario: Non-Craftsky profile omits unavailable counts
    Given Carol is a non-Craftsky atproto account
    And Carol's profile response has isCraftskyProfile false
    And Carol's followerCount and followingCount are absent or null
    When Flutter renders Carol's profile
    Then the screen shows "Non Craftsky profile"
    And the follower/following stats are omitted or shown as unknown
    And no fake numeric follower or following count is shown
```

### AT-006: Follow/unfollow failure preserves last confirmed UI state

Requirement IDs: FR-008, FR-009  
Acceptance Criteria: AC-015  
Priority: Should  
Level: Acceptance  
Automation Target: `app/test/profile/profile_page_test.dart`, `app/test/profile/providers/*follow*_test.dart`

```gherkin
Feature: Follow failure handling
  Scenario: Follow request fails
    Given Alice is viewing Bob's profile with viewerIsFollowing false
    When Alice taps "Follow"
    And the AppView request fails
    Then Flutter shows an error message
    And Bob's profile action returns to "Follow"
    And no unconfirmed follower count increase remains visible
```

### AT-007: Self profile cannot be followed or unfollowed

Requirement IDs: FR-003, FR-004, FR-006, RULE-002, RULE-006  
Acceptance Criteria: AC-013, AC-018  
Priority: Must  
Level: Acceptance  
Automation Target: `appview/internal/api/follow_test.go`, `appview/internal/api/profile_test.go`, `app/test/profile/profile_page_test.dart`

```gherkin
Feature: Self profile follow rules
  Scenario: Viewer opens their own profile
    Given Alice is signed in
    When Alice fetches her own profile
    Then the response includes viewerIsFollowing false
    And the Flutter screen shows self-profile actions, not Follow or Unfollow

  Scenario: Viewer targets self through follow endpoints
    Given Alice is signed in
    When Alice sends POST or DELETE /v1/profiles/@alice.example/follows
    Then the AppView rejects the request with a validation error
    And no PDS follow record is written or deleted
```

### AT-008: External and historical follows converge through Tap

Requirement IDs: BR-004, FR-002, FR-010, RULE-005  
Acceptance Criteria: AC-016, AC-017, AC-025  
Priority: Must  
Level: Acceptance  
Automation Target: `appview/internal/index/follow_test.go`, `appview/internal/app/indexer_wiring_test.go`, manual `just dev-d` Tap check

```gherkin
Feature: Follow graph indexing
  Scenario: Historical or external follow is delivered by Tap
    Given Alice has an app.bsky.graph.follow record authored outside Craftsky
    And Tap delivers the record to the AppView follow indexer
    When the AppView handles the event
    Then the active relationship is stored in the follow graph
    And applicable Craftsky profile counts and viewerIsFollowing state use that active relationship
    When Tap later delivers a delete event for the same follow URI
    Then the active relationship is removed from graph state
```

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-003, FR-004, RULE-001 | AC-012, AC-020 | Validate current handle/DID target parsing for follow/unfollow. | `did:plc:*`, valid handle, malformed handle, old/unresolvable handle fake. | Valid identifiers resolve; malformed returns `invalid_identifier`; resolver errors map to identity error; non-Craftsky target is not rejected solely for missing `craftsky_profiles`. | `appview/internal/api/follow_test.go` |
| UT-002 | FR-003, FR-004, RULE-002, RULE-004 | AC-013 | Reject self-targets for both follow and unfollow before PDS calls. | Auth DID equals target DID for POST and DELETE. | Validation error envelope; fake PDS create/delete call count remains zero. | `appview/internal/api/follow_test.go` |
| UT-003 | FR-003, FR-004, NFR-001 | AC-008, AC-012, AC-013 | Define stable follow error codes. | Missing device/auth handled by middleware; invalid target; identity unavailable; self target; PDS failure. | Error envelopes use documented codes such as `invalid_identifier`, `identity_unavailable`, `self_follow_not_allowed` or chosen equivalent, and `pds_write_failed`. | `appview/internal/api/follow_test.go`, `appview/internal/routes/routes_test.go` |
| UT-004 | FR-001, FR-002, NFR-002, RULE-003 | AC-006 | Follow indexer create is idempotent. | Same `app.bsky.graph.follow` create event delivered twice with same URI/CID. | One active row; count/viewer state contributes once. | `appview/internal/index/follow_test.go` |
| UT-005 | FR-002, NFR-002 | AC-006 | Follow indexer update upserts by URI. | Update event for existing URI with new CID and same/different subject. | Stored row reflects latest CID/subject; no duplicate active URI row. | `appview/internal/index/follow_test.go` |
| UT-006 | FR-001, FR-002, RULE-007 | AC-007, AC-019 | Follow indexer delete removes by URI/rkey and tolerates unknown deletes. | Delete event for existing URI; delete event for unknown URI. | Existing active row is hard-deleted; unknown delete is no-op; no deleted-history row is retained. | `appview/internal/index/follow_test.go` |
| UT-007 | FR-001, BR-003, RULE-005 | AC-005 | Store exposes active followed target DIDs for timeline. | Follower DID with active follows, deleted follows, duplicate event attempts. | Returns only active target DIDs once, without PDS calls. | `appview/internal/api/follow_store_test.go` or `appview/internal/api/profile_store_test.go` |
| UT-008 | FR-006, RULE-005, RULE-006 | AC-003, AC-004, AC-018 | Build/read Craftsky profile response with counts and viewer state. | Craftsky profile row, active follows, viewer DID equal/not equal profile DID. | `followerCount`, `followingCount`, `viewerIsFollowing`, and `isCraftskyProfile=true` are correct; self profile has `viewerIsFollowing=false`. | `appview/internal/api/profile_response_test.go`, `appview/internal/api/profile_store_test.go` |
| UT-009 | RULE-009 | AC-026 | Profile count calculation failure is surfaced. | Fake store/count dependency returns error while reading Craftsky profile. | Handler returns documented error; no fake zero/placeholder counts. | `appview/internal/api/profile_test.go` |
| UT-010 | FR-003, FR-005, RULE-003 | AC-001 | Follow handler writes correct PDS record shape. | Auth DID, target DID, fake PDS create response URI/CID. | Calls `CreateRecord` on auth DID repo, collection `app.bsky.graph.follow`, subject target DID, returns 200 profile response. | `appview/internal/api/follow_test.go` |
| UT-011 | FR-004, FR-005, RULE-004 | AC-002, AC-009 | Unfollow handler deletes active PDS record or no-ops idempotently. | Active graph row with URI/rkey; no-active-row case. | Calls `DeleteRecord` with auth DID repo and active rkey when present; no-active case returns 200 without PDS delete. | `appview/internal/api/follow_test.go` |
| UT-012 | BR-005, FR-006, FR-011 | AC-020, AC-021 | Build/read non-Craftsky profile response. | Bluesky profile row/cache without Craftsky row. | Response includes DID, handle, display fields, empty/default crafts, `isCraftskyProfile=false`, nullable/omitted counts. | `appview/internal/api/profile_response_test.go`, `appview/internal/api/profile_store_test.go` |
| UT-013 | FR-012, RULE-008 | AC-021, AC-023 | Non-Craftsky profile hydration does not require membership. | Fake anonymous/PDS profile record for non-member; missing profile; cache hit. | Hydratable account returns profile; unavailable profile returns documented error; counts remain unknown without fake values. | `appview/internal/index/bluesky_profile_test.go`, new hydration tests |
| UT-014 | FR-007, NFR-003 | AC-014 | Flutter API client uses Craftsky API only. | Follow/unfollow calls with `DioAdapter`. | POST/DELETE `/v1/profiles/@handle/follows`; no PDS URL/token fields exposed in request/response model. | `app/test/profile/data/profile_api_client_test.dart` |
| UT-015 | FR-007 | AC-011, AC-021 | Flutter `Profile` model decodes new fields. | Craftsky JSON with counts; non-Craftsky JSON with null/missing counts. | Model exposes `viewerIsFollowing`, `isCraftskyProfile`, nullable counts, and existing fields. | `app/test/profile/models/profile_test.dart` |
| UT-016 | FR-007, FR-008, RULE-008 | AC-010, AC-011, AC-021, AC-023 | Flutter widgets render Follow/Unfollow, counts, and non-Craftsky marker from model. | Profile models with following true/false, Craftsky/non-Craftsky, counts/null counts. | Correct labels, stats, and `Non Craftsky profile`; no fake count for null non-Craftsky counts. | `app/test/profile/profile_page_test.dart`, profile widget tests |
| UT-017 | FR-008 | AC-024 | Flutter disables/loading button during in-flight mutation. | Fake repository with delayed follow/unfollow Future. | Button cannot be tapped twice and shows existing loading/disabled affordance. | `app/test/profile/profile_page_test.dart` or provider test |
| UT-018 | FR-008, FR-009 | AC-015 | Flutter restores last confirmed state on follow/unfollow failure. | Fake repository throws; initial following false/true. | Error message recorded; button/counts return to prior state. | `app/test/profile/profile_page_test.dart`, provider test |

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | FR-001, RULE-003, RULE-007 | AC-005, AC-006, AC-007, AC-019 | Follow graph migration/store enforces active graph semantics. | Real Postgres test DB with follow table migration. | Insert/upsert active follows, repeat same event, delete by URI. | Unique active relationship for counts; deleted row removed; active followed DID query works. | `appview/internal/api/follow_store_test.go` or migration/store tests |
| IT-002 | FR-001, FR-002, RULE-005 | AC-003, AC-004, AC-007 | Counts reflect active indexed app-agnostic follows. | Craftsky profile rows for Alice/Bob; follow rows from Craftsky and external clients; deleted row. | Fetch profile counts through store. | Counts include active indexed follows regardless of client/app and exclude deleted/inactive rows. | `appview/internal/api/profile_store_test.go` |
| IT-003 | BR-004, FR-002, RULE-005 | AC-017, AC-025 | External follow create/delete events update graph. | Real DB; follow indexer; synthetic Tap events with `Collection=app.bsky.graph.follow`. | Handle create from external client, then delete. | Graph state and profile counts converge on create/delete. | `appview/internal/index/follow_test.go` |
| IT-004 | FR-003, FR-005, RULE-001, RULE-002, RULE-003 | AC-001, AC-012, AC-013, AC-020, AC-022 | POST follow endpoint contract. | Handler with fake resolver, fake PDS, profile store/hydrator, authenticated DID/session context. | POST Craftsky target, non-Craftsky target, invalid target, self target, already-following target. | 200 updated profile for valid/idempotent cases; correct PDS create behavior; documented errors for invalid/self. | `appview/internal/api/follow_test.go` |
| IT-005 | FR-004, FR-005, RULE-001, RULE-002, RULE-004 | AC-002, AC-009, AC-012, AC-013, AC-020, AC-022 | DELETE unfollow endpoint contract. | Handler with fake active follow lookup, fake PDS, resolver/profile hydrator, auth context. | DELETE active, no-active, non-Craftsky, invalid, and self targets. | Active delete calls PDS with stored rkey; no-active returns 200; self/invalid errors; updated profile response returned for valid targets. | `appview/internal/api/follow_test.go` |
| IT-006 | BR-002, FR-006, NFR-004, RULE-006, RULE-009 | AC-003, AC-004, AC-011, AC-018, AC-026 | Profile GET includes required follow fields for Craftsky profiles and fails on count errors. | Real store or fake dependency; Craftsky profile row; viewer DID context; graph rows; count-error injection. | GET `/v1/profiles/@bob`, GET `/v1/profiles/me`. | Response has counts, `viewerIsFollowing`, `isCraftskyProfile=true`; self has false; count failure returns documented error. | `appview/internal/api/profile_test.go`, `profile_store_test.go` |
| IT-007 | BR-005, FR-011, FR-012, RULE-001, RULE-008 | AC-020, AC-021, AC-023 | Non-Craftsky profile read/hydration. | No `craftsky_profiles` row for Carol; cached or fake-hydrated Bluesky profile exists. | GET `/v1/profiles/@carol.example`. | 200 profile response with Bluesky fields, `isCraftskyProfile=false`, null/omitted counts allowed, no fake values. | `appview/internal/api/profile_test.go`, `profile_store_test.go` |
| IT-008 | FR-010 | AC-016 | Historical follow event is accepted by indexer path. | Synthetic Tap event with `Live=false` for `app.bsky.graph.follow`; real follow indexer/DB. | Dispatch event through indexer/dispatcher. | Historical event creates same active graph row as live event. | `appview/internal/index/follow_test.go`, `appview/internal/app/indexer_wiring_test.go` |
| IT-009 | NFR-001 | AC-008, AC-012, AC-013 | Route/middleware conventions for follow routes. | Registered routes with auth/device middleware. | POST/DELETE without auth, without device ID, invalid/self target. | Existing 401/400 behavior for auth/device; camelCase JSON and `{error,message,requestId}` errors. | `appview/internal/routes/routes_test.go`, `appview/internal/api/follow_test.go` |
| IT-010 | FR-005, NFR-003 | AC-014 | PDS tokens remain server-side. | Fake Flutter/Dio client and AppView handler with fake PDS factory. | Initiate follow/unfollow from Flutter-side client tests and handler tests. | Flutter sends Craftsky request only; AppView constructs PDS client from server-side session; response contains no PDS token fields. | `app/test/profile/data/profile_api_client_test.dart`, `appview/internal/api/follow_test.go` |
| IT-011 | FR-012 | AC-020, AC-021 | Bluesky profile indexer/hydrator supports non-members. | Non-member DID `app.bsky.actor.profile` event or fake PDS hydrate response. | Handle event or hydrate profile. | `bluesky_profiles`/cache row can exist without `craftsky_profiles`; Craftsky membership marker remains false. | `appview/internal/index/bluesky_profile_test.go`, hydration tests |
| IT-012 | FR-002 | AC-006, AC-007, AC-025 | Dispatcher and Tap filter wiring include follow NSID. | App deps dispatcher and compose/config inspection test if practical. | Dispatch `app.bsky.graph.follow`; inspect registered collection/filter config. | Follow event reaches follow indexer; unregistered fallback is not used; Tap filter includes `app.bsky.graph.follow`. | `appview/internal/app/indexer_wiring_test.go`, config test |

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Test |
|---|---|---|---|
| REG-001 | `/v1/*` auth and `X-Craftsky-Device-Id` middleware behavior stays consistent. | NFR-001 | Existing route tests plus new follow route tests assert 401 without auth and 400 without device ID. |
| REG-002 | Error envelope shape remains `{error, message, requestId}`. | NFR-001 | New follow/profile error tests decode `envelope.Error`; existing envelope tests remain passing. |
| REG-003 | Existing profile GET by Craftsky handle/DID still returns display fields, crafts, avatar/banner, and createdAt. | FR-006, FR-011 | Extend `profile_test.go` / `profile_response_test.go` with new fields while preserving existing assertions. |
| REG-004 | Flutter never receives or stores PDS tokens. | FR-005, NFR-003 | Flutter API client tests assert follow/unfollow request/response bodies contain no token fields; existing auth storage tests remain unchanged. |
| REG-005 | Existing profile edit, avatar/banner display, post tabs, and share behavior are not broken by follow UI changes. | FR-007, FR-008 | Existing `profile_page_test.dart`, `profile_api_client_test.dart`, and profile widget tests continue to pass; update only the obsolete “Follow coming soon” expectation. |
| REG-006 | Project count remains placeholder/future work and is not silently converted to a fake follow-derived value. | RULE-008 | Profile UI tests assert project stat behavior remains explicitly placeholder/unknown as implemented. |
| REG-007 | Post like/repost indexers and dispatcher routing are not affected by adding follow indexer registration. | FR-002 | Existing `indexer_wiring_test.go`, `craftsky_interaction_test.go`, and dispatcher tests continue to pass after adding follow registration. |

## 7. Test Data

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Authenticated Craftsky viewer | DID `did:plc:alice`, handle `alice.craftsky.social`, OAuth session ID `sess-alice`, device ID UUID, Craftsky profile row. | AT-001, AT-002, AT-003, IT-004, IT-005, IT-009 |
| TD-002 | Craftsky target profile | DID `did:plc:bob`, handle `bob.craftsky.social`, Craftsky row with crafts, Bluesky display fields, initial follower/following counts. | AT-001, AT-002, AT-004, IT-002, IT-006 |
| TD-003 | Non-Craftsky target profile | DID `did:plc:carol`, handle `carol.bsky.social`, no `craftsky_profiles` row, Bluesky profile display name/avatar/description. | AT-003, AT-005, IT-007, IT-011 |
| TD-004 | External followed account | DID `did:plc:dana`, handle `dana.example`, may or may not have Craftsky row. | IT-002, IT-003, IT-008 |
| TD-005 | Active follow record | URI `at://did:plc:alice/app.bsky.graph.follow/follow1`, rkey `follow1`, CID `bafyfollow1`, subject DID `did:plc:bob`, createdAt ISO timestamp. | UT-004, UT-006, IT-001, IT-004, IT-005 |
| TD-006 | Duplicate/update follow record | Same URI/rkey as TD-005 with same CID for replay and new CID `bafyfollow2` for update/upsert. | UT-004, UT-005, IT-003 |
| TD-007 | Tombstone event | Delete Tap event for TD-005 URI/rkey with empty record body. | UT-006, IT-001, IT-003 |
| TD-008 | Historical Tap event | Tap event for `app.bsky.graph.follow` with `Live=false`. | AT-008, IT-008, MAN-001 |
| TD-009 | Error fixtures | Invalid identifier `NOT VALID`, resolver error, self target, fake PDS create/delete error, count-store error. | UT-001, UT-002, UT-003, UT-009, IT-004, IT-005, IT-006 |
| TD-010 | Flutter profile JSON | Craftsky JSON with new fields; non-Craftsky JSON with `isCraftskyProfile=false` and missing/null counts. | UT-014, UT-015, UT-016 |

## 8. Manual Checks

| ID | Requirement IDs | Check | Steps | Expected Result |
|---|---|---|---|---|
| MAN-001 | BR-004, FR-010 | Smoke-check live Tap historical follow delivery after adding `app.bsky.graph.follow`. | Start stack with `just dev-d`; ensure Tap filter includes `app.bsky.graph.follow`; add a dev DID with existing follow records through `/repos/add` or signal collection; inspect AppView graph rows/logs after backfill. | Historical follow records are delivered with `Live=false` or equivalent backfill semantics and indexed into active graph state. |
| MAN-002 | BR-001, BR-005, FR-003, FR-004 | End-to-end follow/unfollow against dev PDS/Tap stack. | Sign in as a dev user; visit Craftsky and non-Craftsky profile routes; follow and unfollow; watch UI and AppView logs. | UI updates immediately from AppView response; graph converges after Tap events; no PDS tokens appear in Flutter logs/storage. |
| MAN-003 | NFR-004 | Check profile count query plan after migration. | With representative follow rows in dev Postgres, run `EXPLAIN` for follower count, following count, viewerIsFollowing, and active-follow lookup queries. | Queries use indexes on follower DID, subject DID, `(follower DID, subject DID)`, and URI/rkey; no obvious N+1 behavior for a single profile response. |

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | Globally authoritative non-Craftsky counts are intentionally out of MVP. | RULE-008, ASM-006 | Requirements explicitly defer external graph/AppView count source. | Future design slice for global non-Craftsky profile counts if product needs them. |
| GAP-002 | Exact error code names are not all fixed in requirements. | NFR-001, FR-003, FR-004 | Requirements name examples and require documented envelopes but leave some code names to implementation. | Implementation plan should choose stable codes before writing handler tests; tests should lock them once chosen. |
| GAP-003 | PDS duplicate follow records are assumed rare/invalid but can exist in the wider network. | RULE-003, RISK-002 | Requirements require one active relationship contributes to counts/state, but do not require deleting all duplicate PDS records on unfollow. | Store/indexer tests should collapse duplicates for counts; implementation plan should decide canonical delete behavior if multiple active URIs exist for same pair. |
| GAP-004 | Non-Craftsky profile hydration failure UX is only specified at API behavior level. | FR-011, FR-012 | Requirements require hydratable profiles but do not specify exact Flutter error copy for unavailable profiles. | Use existing profile-load error UI; document exact copy during implementation if changed. |

## 10. Out Of Scope

- Automated global Bluesky/atproto follower/following count verification for non-Craftsky profiles; MVP explicitly excludes globally authoritative non-Craftsky counts.
- Follower/following list screen tests; list screens are non-goals.
- Timeline tests for `GET /v1/feed/timeline`; this feature only prepares graph data.
- Notifications, blocks, mutes, reports, moderation workflow, and rate-limit tests.
- New lexicon validation tests for a Craftsky follow type; follows use existing `app.bsky.graph.follow`.
- Public-follow warning UI tests; user explicitly decided no new disclosure UI is required for MVP.

## 11. Handoff To Document Review

- Requirements file: `docs/changes/2026-05-25-follow-unfollow-mvp/01-requirements.md`
- Test specification: `docs/changes/2026-05-25-follow-unfollow-mvp/02-acceptance-tests.md`
- Next review artifact: `03-document-review.md`
- Preferred Plannotator review: `/plannotator-annotate docs/changes/2026-05-25-follow-unfollow-mvp/`
- Recommended first failing test for implementation: `UT-004` — follow indexer create is idempotent and stores one active relationship from an `app.bsky.graph.follow` create event. This establishes the graph foundation before API/UI work depends on it.
- Suggested test order for implementation:
  1. `UT-004`, `UT-005`, `UT-006` follow indexer create/update/delete behavior.
  2. `IT-001`, `IT-002`, `UT-007`, `UT-008` follow graph store, counts, viewer state, and active followed DID lookup.
  3. `IT-012` dispatcher/Tap filter registration for `app.bsky.graph.follow`.
  4. `UT-012`, `UT-013`, `IT-007`, `IT-011` non-Craftsky profile response/hydration.
  5. `UT-010`, `UT-011`, `IT-004`, `IT-005`, `IT-009`, `IT-010` follow/unfollow API handlers and route conventions.
  6. `UT-014`, `UT-015` Flutter model/API client support.
  7. `UT-016`, `UT-017`, `UT-018`, `AT-001` through `AT-007` Flutter UI/provider behavior.
  8. `IT-008`, `AT-008`, `MAN-001`, `MAN-002` historical/external Tap convergence checks.
- Commands discovered:
  - AppView full suite: `just test` from repo root after `just dev-d` is running.
  - Focused AppView examples: `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/index ./internal/api ./internal/routes ./internal/app`.
  - Flutter full suite: `cd app && flutter test`.
  - Focused Flutter examples: `cd app && flutter test test/profile test/shared/api`.
- Blocking gaps:
  - None. Tap historical delivery has been confirmed; `MAN-001` remains as an end-to-end smoke check.
- Risk-based review recommendation: risk remains **High**. Run document review before coding unless the user explicitly skips review.
