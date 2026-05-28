# Acceptance Test Specification: Profile Social Summary

## 1. Test Strategy
This change is medium risk because it spans AppView API contracts, indexed graph/post queries, Flutter profile UI, Flutter settings navigation, and existing follow/profile regressions. Use a TDD sequence that starts with AppView read-model integration tests for the new summary/list semantics, then API contract tests, then Flutter model/client/widget tests.

Primary automation targets:

- AppView integration/unit tests for profile summary counts, top-level post filtering, mutual-follower count/list behavior, follower/following recency, pagination, auth/device enforcement, and response shape.
- Flutter model/API client tests for new fields and list endpoints.
- Flutter widget tests for profile stats, clickable mutual count, 90%-height bottom sheet, settings entries without counts, app-bar counts, empty states, non-Craftsky age hiding, and follower/following count non-display on profile pages.
- Regression tests for existing follow/unfollow behavior, profile fetches, profile tabs, posts/comments tabs, and settings page baseline behavior.

Manual checks are limited to visual polish that is awkward to assert precisely in widget tests, especially bottom-sheet height/scroll behavior on device sizes.

## 2. Requirement Coverage Matrix
| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001 | AT-001, UT-006, REG-001 | Acceptance / Unit / Regression | Yes |
| BR-002 | AC-002, AC-003, AC-004, AC-015, AC-019 | AT-002, AT-003, IT-002, IT-004, UT-001, UT-006, MAN-001 | Acceptance / Integration / Unit / Manual | Yes, except MAN-001 |
| BR-003 | AC-008, AC-009, AC-010 | AT-005, AT-006, IT-005, IT-006, UT-009 | Acceptance / Integration / Unit | Yes |
| FR-001 | AC-001 | AT-001, UT-006, REG-001 | Acceptance / Unit / Regression | Yes |
| FR-002 | AC-002, AC-003, AC-004, AC-015 | AT-002, AT-003, IT-002, IT-004, UT-006, UT-007 | Acceptance / Integration / Unit | Yes |
| FR-003 | AC-003 | IT-002, IT-004, REG-006 | Integration / Regression | Yes |
| FR-004 | AC-005, AC-018 | AT-004, UT-004, UT-006 | Acceptance / Unit | Yes |
| FR-005 | AC-006 | AT-004, IT-001, UT-003 | Acceptance / Integration / Unit | Yes |
| FR-006 | AC-007, AC-020 | AT-004, IT-001, UT-002, UT-006 | Acceptance / Integration / Unit | Yes |
| FR-007 | AC-008 | AT-005, UT-009, REG-005 | Acceptance / Unit / Regression | Yes |
| FR-008 | AC-009 | AT-006, IT-005, UT-010 | Acceptance / Integration / Unit | Yes |
| FR-009 | AC-010 | AT-007, IT-006, UT-010 | Acceptance / Integration / Unit | Yes |
| FR-010 | AC-011 | AT-006, AT-007, IT-004, IT-005, IT-006, UT-011 | Acceptance / Integration / Unit | Yes |
| FR-011 | AC-012 | IT-004, IT-005, IT-006, UT-012 | Integration / Unit | Yes |
| FR-012 | AC-001, AC-008, AC-016 | AT-001, AT-005, IT-003, UT-005, UT-006 | Acceptance / Integration / Unit | Yes |
| FR-013 | AC-015 | AT-003, UT-007, MAN-001 | Acceptance / Unit / Manual | Yes, except MAN-001 |
| FR-014 | AC-017 | AT-008, UT-008, UT-010 | Acceptance / Unit | Yes |
| FR-015 | AC-019 | IT-003, UT-005, UT-011 | Integration / Unit | Yes |
| FR-016 | AC-012, AC-015, AC-019 | AT-003, IT-004, UT-012 | Acceptance / Integration / Unit | Yes |
| FR-017 | AC-018 | AT-009, UT-006 | Acceptance / Unit | Yes |
| NFR-001 | AC-012, AC-013 | IT-004, IT-005, IT-006, IT-007, REG-007 | Integration / Regression | Yes |
| NFR-002 | AC-014 | IT-008, MAN-002 | Integration / Manual | Partial |
| RULE-001 | AC-003 | IT-002, IT-004, UT-001 | Integration / Unit | Yes |
| RULE-002 | AC-005 | AT-004, UT-004 | Acceptance / Unit | Yes |
| RULE-003 | AC-006 | IT-001, UT-003 | Integration / Unit | Yes |
| RULE-004 | AC-009, AC-010 | IT-005, IT-006, UT-010 | Integration / Unit | Yes |
| RULE-005 | AC-007 | IT-001, UT-002 | Integration / Unit | Yes |

## 3. Acceptance Scenarios
### AT-001: Profile pages do not show follower or following counts
Requirement IDs: BR-001, FR-001, FR-012
Acceptance Criteria: AC-001, AC-016
Priority: Must
Level: Acceptance
Automation Target: `app/test/profile/profile_page_test.dart`

```gherkin
Feature: Profile social summary
  Scenario: Follower and following counts are hidden on profile pages
    Given a signed-in viewer opens a profile whose API response contains followerCount 9 and followingCount 7
    When the profile page renders
    Then the profile stats area does not show follower or following count cells
    And the text "followers" is not shown as a profile stat label
    And the text "following" is not shown as a profile stat label
```

### AT-002: Visitor profile shows clickable uncapped mutual follower count
Requirement IDs: BR-002, FR-002
Acceptance Criteria: AC-002, AC-004
Priority: Must
Level: Acceptance
Automation Target: `app/test/profile/profile_page_test.dart`

```gherkin
Feature: Visitor profile mutuals
  Scenario: Visitor sees mutual follower count text
    Given Alice is signed in
    And Alice opens Bob's Craftsky profile
    And Bob's profile response has mutualFollowerCount 12
    When the profile page renders
    Then the profile shows clickable text "12 mutual followers"
    And the profile does not show Bob's followerCount or followingCount as profile stats

  Scenario: Self profile does not show mutual followers
    Given Alice is signed in
    And Alice opens her own profile
    When the profile page renders
    Then no mutual-follower section is shown
```

### AT-003: Mutual follower count opens paginated 90%-height bottom sheet
Requirement IDs: BR-002, FR-013, FR-016
Acceptance Criteria: AC-015, AC-019
Priority: Must
Level: Acceptance
Automation Target: `app/test/profile/profile_page_test.dart`, `app/test/profile/data/profile_api_client_test.dart`

```gherkin
Feature: Mutual follower list
  Scenario: Tapping mutual count opens the mutual followers bottom sheet
    Given Alice opens Bob's profile with "12 mutual followers" visible
    When Alice taps "12 mutual followers"
    Then a bottom sheet opens at approximately 90 percent height
    And the app requests Bob's mutual followers from the separate paginated endpoint
    And the bottom sheet displays mutual account rows returned by AppView
```

### AT-004: Craftsky profile stats show joined age and top-level post/project counts
Requirement IDs: FR-004, FR-005, FR-006, RULE-002, RULE-003, RULE-005
Acceptance Criteria: AC-005, AC-006, AC-007, AC-020
Priority: Must
Level: Acceptance
Automation Target: `app/test/profile/profile_page_test.dart`, `app/test/profile/models/profile_test.dart`

```gherkin
Feature: Profile stats
  Scenario: Craftsky profile displays data-driven social summary stats
    Given a Craftsky profile response includes createdAt, postsLast7Days, postCount, and projectCount
    When the profile page renders
    Then it displays account age as "Joined <age> ago"
    And it displays "X posts in the last 7 days" from top-level post count data
    And it displays total top-level posts
    And it displays projects from profile response data rather than a hardcoded value
```

### AT-005: Settings entries are tappable and do not show graph counts
Requirement IDs: BR-003, FR-007, FR-012
Acceptance Criteria: AC-008
Priority: Must
Level: Acceptance
Automation Target: `app/test/settings/settings_page_test.dart`

```gherkin
Feature: Settings social graph entry points
  Scenario: Settings links to followers and following without counts
    Given Alice is signed in
    When Alice opens Settings
    Then Settings shows tappable entries for Followers and Following
    And Settings does not show follower or following counts next to those entries
```

### AT-006: Followers list is ordered by newest follow and shows count in app bar
Requirement IDs: BR-003, FR-008, FR-010, RULE-004
Acceptance Criteria: AC-009, AC-011
Priority: Must
Level: Acceptance
Automation Target: `app/test/settings/followers_page_test.dart` or equivalent new suite

```gherkin
Feature: Followers list
  Scenario: Followers page shows newest followers first
    Given Alice has followers Carol, Bob, and Dana with different follow createdAt values
    When Alice opens Settings and taps Followers
    Then the followers page app bar title includes the follower count
    And follower rows are ordered by newest follow first
    And each row displays account identity from AppView data
```

### AT-007: Following list is ordered by newest follow and shows count in app bar
Requirement IDs: BR-003, FR-009, FR-010, RULE-004
Acceptance Criteria: AC-010, AC-011
Priority: Must
Level: Acceptance
Automation Target: `app/test/settings/following_page_test.dart` or equivalent new suite

```gherkin
Feature: Following list
  Scenario: Following page shows newest followed Craftsky accounts first
    Given Alice follows Craftsky accounts Bob, Carol, and Dana at different times
    And Alice follows non-Craftsky account Erin
    When Alice opens Settings and taps Following
    Then the following page app bar title includes the following count for Craftsky accounts only
    And following rows are ordered by newest follow first
    And Erin is not shown
    And each row displays account identity from AppView data
```

### AT-008: Empty graph states use simple empty copy and zero mutuals render nothing
Requirement IDs: FR-014
Acceptance Criteria: AC-017
Priority: Must
Level: Acceptance
Automation Target: `app/test/profile/profile_page_test.dart`, `app/test/settings/followers_page_test.dart`, `app/test/settings/following_page_test.dart`

```gherkin
Feature: Social graph empty states
  Scenario: Empty graph pages and zero mutuals render expected empty states
    Given Alice follows nobody
    When Alice opens the Following page
    Then the page shows "You are not following anyone"
    When Alice opens the Followers page and has no followers
    Then the page shows "No one follows you yet"
    When Alice opens Bob's profile and mutualFollowerCount is 0
    Then no mutual-follower section is rendered
```

### AT-009: Non-Craftsky profile hides account age
Requirement IDs: FR-017, FR-004
Acceptance Criteria: AC-018
Priority: Must
Level: Acceptance
Automation Target: `app/test/profile/profile_page_test.dart`

```gherkin
Feature: Non-Craftsky profile summary
  Scenario: Non-Craftsky profile does not show Craftsky account age
    Given Alice opens Carol's non-Craftsky profile
    When the profile page renders
    Then the non-Craftsky marker is shown
    And no "Joined <age> ago" account-age stat is shown
```

## 4. Unit Test Cases
| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | RULE-001, FR-003 | AC-003 | Mutual predicate includes only DIDs where viewer follows mutual and mutual follows profile. | Follow graph with viewer→m1, m1→profile, viewer→m2 only, m3→profile only. | Only m1 qualifies as mutual. | `appview/internal/api/profile_store_test.go` or new graph helper test |
| UT-002 | FR-006, RULE-005 | AC-007 | Total post count filters to top-level authored posts. | Root posts, replies/comments, repost interactions for same DID. | Count includes roots only. | `appview/internal/api/profile_store_test.go` or profile summary helper test |
| UT-003 | FR-005, RULE-003 | AC-006 | Recent activity filters to top-level posts in trailing 7 days. | Root posts inside/outside window, replies, repost interactions. | Count includes only top-level roots within window. | `appview/internal/api/profile_store_test.go` |
| UT-004 | FR-004, RULE-002 | AC-005 | Flutter account-age formatting from `createdAt`. | Fixed current clock and createdAt values. | Displays `Joined <age> ago` using expected units. | `app/test/profile/widgets/profile_stats_test.dart` or model/formatter suite |
| UT-005 | FR-012, FR-015 | AC-016, AC-019 | Profile response preserves follower/following counts and includes mutualFollowerCount only. | Profile row with followerCount, followingCount, mutualFollowerCount. | JSON has those count fields and no mutual preview array. | `appview/internal/api/profile_response_test.go` |
| UT-006 | FR-001, FR-004, FR-006, FR-017 | AC-001, AC-005, AC-007, AC-018, AC-020 | Profile stats widget renders new stats and hides old counts/non-Craftsky age. | Craftsky and non-Craftsky `Profile` models with counts/stats. | No follower/following stat cells; Craftsky age shown; non-Craftsky age hidden; projectCount from model. | `app/test/profile/profile_page_test.dart` or `app/test/profile/widgets/profile_stats_test.dart` |
| UT-007 | FR-002, FR-013 | AC-002, AC-015 | Mutual count widget is clickable only when count > 0. | mutualFollowerCount 12 and 0. | Count 12 renders clickable text; 0 renders shrink/absent. | `app/test/profile/widgets/profile_mutual_followers_test.dart` |
| UT-008 | FR-014 | AC-017 | Empty state copy selection. | `followers` empty and `following` empty modes. | Followers copy is `No one follows you yet`; following copy is `You are not following anyone`. | `app/test/settings/follow_list_empty_state_test.dart` |
| UT-009 | FR-007, FR-012 | AC-008 | Settings entries omit counts. | Settings page with known follower/following counts. | Entries are tappable and do not render numeric counts. | `app/test/settings/settings_page_test.dart` |
| UT-010 | FR-008, FR-009, RULE-004 | AC-009, AC-010, AC-017 | Follow list page title and ordering presentation. | Account summaries ordered newest-first from repository plus total count. | App bar contains count; rows preserve repository order; empty copy displayed when no rows. | `app/test/settings/followers_page_test.dart`, `app/test/settings/following_page_test.dart` |
| UT-011 | FR-010, FR-015 | AC-011, AC-019 | Flutter/API account summary model decodes list rows and profile mutual count. | JSON account rows with did, handle, displayName/avatar; profile JSON with mutualFollowerCount. | Models decode display-ready data; no PDS-only fields required. | `app/test/profile/models/profile_test.dart`, `app/test/profile/data/profile_api_client_test.dart` |
| UT-012 | FR-011, FR-016, NFR-001 | AC-012, AC-015 | Flutter API client sends/decodes list pagination. | limit/cursor for mutuals/followers/following endpoints. | Requests include query params; response cursor is decoded/retained opaquely. | `app/test/profile/data/profile_api_client_test.dart` |

## 5. Integration Test Cases
| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | FR-005, FR-006, RULE-003, RULE-005 | AC-006, AC-007, AC-020 | AppView profile summary counts top-level posts and project count field. | Seed `craftsky_posts` with root posts in/out of 7-day window, replies/comments, and no projects table/source. | Read profile summary. | `postsLast7Days` and `postCount` count roots only; `projectCount` is data-driven and can be 0. | `appview/internal/api/profile_store_test.go` |
| IT-002 | BR-002, FR-003, RULE-001 | AC-003 | Mutual follower count uses active indexed follow graph. | Seed viewer, profile, mutual, non-mutual accounts and `atproto_follows`. | Read profile as viewer. | Mutual count includes only accounts satisfying viewer→mutual and mutual→profile. | `appview/internal/api/profile_store_test.go` |
| IT-003 | FR-012, FR-015 | AC-016, AC-019 | Profile response contract includes graph counts and excludes embedded mutual preview. | Fake/store row with followerCount, followingCount, mutualFollowerCount. | GET `/v1/profiles/@{handle}`. | JSON has followerCount, followingCount, mutualFollowerCount; no preview array. | `appview/internal/api/profile_test.go`, `profile_response_test.go` |
| IT-004 | FR-010, FR-011, FR-016, NFR-001 | AC-011, AC-012, AC-015, AC-019 | Mutual followers endpoint returns paginated display-ready rows. | Seed > limit mutuals with handles/profile display data and follow edges. | GET mutual followers endpoint with limit/cursor. | Rows include DID, handle, optional display fields; next cursor is opaque; second page works. | `appview/internal/api/profile_store_test.go`, `appview/internal/api/profile_test.go` |
| IT-005 | BR-003, FR-008, FR-010, FR-011, RULE-004 | AC-009, AC-011, AC-012 | Followers endpoint orders by newest follower first and paginates. | Seed followers with different `created_at`; > limit rows. | GET followers endpoint with limit/cursor. | Page order is `created_at DESC`; account rows display-ready; cursor paginates. | `appview/internal/api/profile_store_test.go`, `appview/internal/api/profile_test.go` |
| IT-006 | BR-003, FR-009, FR-010, FR-011, RULE-004 | AC-010, AC-011, AC-012 | Following endpoint orders followed Craftsky profiles by newest follow first, excludes non-Craftsky followed accounts, and paginates. | Seed signed-in DID follows with different `created_at`, including at least one non-Craftsky followed account; > limit Craftsky rows. | GET following endpoint with limit/cursor. | Page order is `created_at DESC` for Craftsky rows; non-Craftsky followed accounts are excluded; total count and cursor pagination use Craftsky rows only. | `appview/internal/api/profile_store_test.go`, `appview/internal/api/profile_test.go` |
| IT-007 | NFR-001 | AC-013 | New social graph endpoints enforce auth and device ID. | Unauthenticated requests and authenticated requests missing device ID. | Call mutual/followers/following endpoints. | Existing 401/400 error behavior applies with standard error envelope. | `appview/internal/routes/routes_test.go` |
| IT-008 | NFR-002 | AC-014 | Summary/list queries remain bounded. | Seed large follow and post sets; request default and max limits. | Call profile summary and graph list endpoints. | List responses respect bounded limit; no unbounded list returned; query order/filter behavior is deterministic. | `appview/internal/api/profile_store_test.go` |

## 6. Regression Tests
| ID | Existing Behavior Protected | Requirement IDs | Test |
|---|---|---|---|
| REG-001 | Existing profile identity, bio, crafts, avatar/banner rendering remains intact while stats change. | FR-001, FR-004, FR-006 | Extend `ProfilePage` widget tests to assert identity/bio/craft chips still render with new stats. |
| REG-002 | Follow/unfollow buttons and optimistic/repository update behavior continue to work. | NG-005 | Keep/extend existing `tapping Follow updates profile` and `tapping Unfollow updates profile` tests. |
| REG-003 | Non-Craftsky marker still renders for non-Craftsky profiles. | FR-017 | Update existing non-Craftsky profile test to assert marker remains and age is hidden. |
| REG-004 | Profile posts/comments tabs still load from existing post repository paths. | FR-005, RULE-005 | Existing `profile_posts_tab_test.dart` and `profile_comments_tab_test.dart` continue passing; add no reliance on profile summary counts for tab contents. |
| REG-005 | Settings still renders existing Clear Image Cache and Sign Out tiles. | FR-007 | Extend `settings_page_test.dart` so new social entries are additive and existing settings tiles remain. |
| REG-006 | Follow/unfollow write endpoints and viewerIsFollowing behavior remain unchanged. | FR-003, NG-005 | Existing AppView follow/unfollow handler tests continue passing; profile reads still return `viewerIsFollowing`. |
| REG-007 | `/v1/` API auth/device/error-envelope conventions remain consistent. | NFR-001 | Add route tests for new endpoints patterned after existing auth/device tests. |
| REG-008 | Profile response keeps existing `followerCount` and `followingCount` fields available. | FR-012 | Extend `profile_response_test.go` to assert old fields remain serialized. |

## 7. Test Data
| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Mutual graph inclusion/exclusion | DIDs: viewer Alice, profile Bob, mutual Carol; non-mutuals Dana (viewer follows only), Erin (follows Bob only). Follow edges Alice→Carol, Carol→Bob, Alice→Dana, Erin→Bob. | AT-002, AT-003, IT-002, IT-004, UT-001 |
| TD-002 | Follower recency order | Bob, Carol, Dana follow Alice at T-1h, T-2h, T-3h. | AT-006, IT-005, UT-010 |
| TD-003 | Following recency order | Alice follows Craftsky accounts Bob, Carol, Dana at T-1h, T-2h, T-3h, and non-Craftsky Erin at T-30m. | AT-007, IT-006, UT-010 |
| TD-004 | Post summary filtering | Alice has top-level posts at now-1d and now-8d, replies/comments at now-1d, and repost interactions at now-1d. | AT-004, IT-001, UT-002, UT-003 |
| TD-005 | Profile response scalar fields | Craftsky profile with `createdAt`, followerCount 9, followingCount 7, mutualFollowerCount 12, postCount 5, postsLast7Days 2, projectCount 0. | AT-001, AT-002, AT-004, IT-003, UT-005, UT-006, UT-011 |
| TD-006 | Non-Craftsky profile | Bluesky-only profile with display name/avatar and no Craftsky profile row. | AT-009, REG-003, UT-006 |
| TD-007 | Empty states | Empty followers page, empty following page, visitor profile with mutualFollowerCount 0. | AT-008, UT-007, UT-008 |
| TD-008 | Pagination | More graph rows than endpoint limit with deterministic `created_at`/tie-breaker values and opaque next cursor. | IT-004, IT-005, IT-006, UT-012 |

## 8. Manual Checks
| ID | Requirement IDs | Check | Steps | Expected Result |
|---|---|---|---|---|
| MAN-001 | FR-013, BR-002 | Bottom sheet visual behavior on device sizes. | Run app, open a visitor profile with mutualFollowerCount > 0, tap mutual count on small and large screens. | Bottom sheet appears around 90% height, is scrollable, and does not obscure navigation unexpectedly. |
| MAN-002 | NFR-002 | Large-list perceived performance. | Seed enough follows to require pagination, open mutuals/followers/following lists in dev. | Initial load and pagination feel bounded; no obvious UI freeze or huge payload. |

## 9. Test Gaps And Risks
| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | Exact AppView route names for mutual/followers/following endpoints are not specified in requirements. | FR-016, NFR-001 | Requirements define behavior and conventions, not final URL paths. | Coding plan should name routes before implementation; tests should lock them once chosen. |
| GAP-002 | Exact JSON field names for post counts are not specified beyond camelCase and examples. | FR-005, FR-006 | Requirements name semantics; implementation may choose `postCount`/`postsLast7Days` or similar. | Coding plan should settle field names before test implementation. |
| GAP-003 | Account-age formatter exact thresholds are not specified. | FR-004, RULE-002 | Requirement fixes copy pattern but not unit thresholds (days/months/years). | Unit tests should document chosen thresholds in implementation plan. |
| GAP-004 | Automated verification of actual database index use is limited. | NFR-002 | Unit/integration tests can verify bounded query behavior, not query planner quality portably. | Add manual/performance review or query-plan checks if large graph performance becomes a concern. |

## 10. Out Of Scope
- Tests for block/mute filtering in mutuals or graph lists; blocks/mutes are explicitly out of scope.
- Tests for a discovery CTA on empty follow lists; this slice requires plain empty text only.
- Tests for project persistence creation; project count is data-driven but may remain `0` until project records exist.
- Tests for public unauthenticated graph access; social graph endpoints remain authenticated.
- Tests for lexicon changes; no lexicon changes are in scope.

## 11. Handoff To Document Review
- Requirements file: `docs/changes/2026-05-27-profile-social-summary/01-requirements.md`
- Test specification: `docs/changes/2026-05-27-profile-social-summary/02-acceptance-tests.md`
- Next review artifact: `03-document-review.md`
- Preferred Plannotator review: `/plannotator-annotate docs/changes/2026-05-27-profile-social-summary/`
- Recommended first failing test for implementation: `IT-001` — AppView profile summary counts top-level posts only and exposes data-driven project count.
- Suggested test order for implementation:
  1. `IT-001`, `UT-002`, `UT-003` for post/project summary semantics.
  2. `IT-002`, `UT-001`, `IT-003` for mutual count and profile response contract.
  3. `IT-004`, `IT-005`, `IT-006`, `IT-007` for graph list endpoints, pagination, ordering, and auth/device behavior.
  4. `UT-011`, `UT-012` for Flutter API/model decoding and list clients.
  5. `AT-001` through `AT-009` for Flutter profile/settings behavior.
  6. `REG-001` through `REG-008` to protect existing profile, settings, and follow behavior.
- Commands discovered:
  - AppView: `just test` from repo root after `just dev-d` is running.
  - AppView focused examples: `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes`.
  - Flutter: `flutter test` from `app/`.
  - Flutter focused examples: `flutter test test/profile/profile_page_test.dart test/settings/settings_page_test.dart` from `app/`.
- Blocking gaps: None. GAP-001 through GAP-004 should be resolved during coding plan or implementation test naming.
