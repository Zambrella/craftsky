# Acceptance Test Specification: Timeline Feed Flutter

## 1. Test Strategy

This feature is a medium-risk Flutter UI/data-layer change. The test strategy is to start at the lowest boundary that proves the new AppView timeline contract, then move upward through repository plumbing, Riverpod timeline state, and Feed tab widget behavior.

- **Integration-style API/client tests** verify `GET /v1/feed/timeline`, query parameters, `PostPage` parsing, and error mapping with `http_mock_adapter`.
- **Unit/provider tests** verify cursor accumulation, opaque-cursor pass-through, load-more guards, retry-preserving state, optimistic prepend, URI dedupe, and cache update helpers.
- **Acceptance/widget tests** verify user-visible Feed tab behavior: loading, loaded timeline, empty state, first-load retry, pagination, compose, thread navigation, interactions, replies, and own-post deletion.
- **Regression tests** protect existing profile list, post-card, composer, interaction-provider, route, generated-code, and full-suite behavior.
- **Manual checks** are limited to visual/accessibility smoke checks that are better validated on a running Flutter target than in widget tests.

Recommended first failing test: `IT-001` in `app/test/feed/data/post_api_client_test.dart`, proving `PostApiClient.listTimeline` calls `/v1/feed/timeline` and parses `PostPage`.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-004, AC-005 | IT-001, AT-001, AT-002, AT-003 | Integration / Acceptance | Yes |
| BR-002 | AC-013 | REG-001, MAN-002 | Regression / Manual review | Partial |
| FR-001 | AC-001, AC-002 | IT-001, IT-002, IT-003 | Integration | Yes |
| FR-002 | AC-002 | IT-004, UT-001 | Integration / Unit | Yes |
| FR-003 | AC-003, AC-006, AC-007 | UT-002, UT-003, UT-004, UT-005, AT-005 | Unit / Acceptance | Yes |
| FR-004 | AC-003 | UT-002 | Unit | Yes |
| FR-005 | AC-006, AC-014 | UT-003, IT-002, AT-005 | Unit / Integration / Acceptance | Yes |
| FR-006 | AC-008 | AT-004 | Acceptance | Yes |
| FR-007 | AC-007 | UT-004, AT-006 | Unit / Acceptance | Yes |
| FR-008 | AC-009 | AT-003 | Acceptance | Yes |
| FR-009 | AC-004, AC-013 | AT-002, REG-002, MAN-001 | Acceptance / Regression / Manual | Yes |
| FR-010 | AC-010 | AT-007 | Acceptance | Yes |
| FR-011 | AC-011 | UT-008, AT-008, REG-004 | Unit / Acceptance / Regression | Yes |
| FR-012 | AC-012 | UT-009, AT-009, REG-003 | Unit / Acceptance / Regression | Yes |
| FR-013 | AC-015 | UT-010, AT-010 | Unit / Acceptance | Yes |
| FR-014 | AC-016 | AT-011, REG-003 | Acceptance / Regression | Yes |
| FR-015 | AC-017, AC-018 | UT-006, UT-007, AT-012 | Unit / Acceptance | Yes |
| NFR-001 | AC-001 | IT-001, REG-007 | Integration / Regression | Yes |
| NFR-002 | AC-002, AC-013 | IT-001, REG-001 | Integration / Regression | Yes |
| NFR-003 | AC-006, AC-019 | AT-005, MAN-001 | Acceptance / Manual | Partial |
| NFR-004 | AC-020 | AT-003, AT-004, MAN-001, REG-006 | Acceptance / Manual / Regression | Partial |
| NFR-005 | AC-021 | REG-006 | Regression command | Yes |
| RULE-001 | AC-001, AC-002 | IT-004, REG-007 | Integration / Regression | Yes |
| RULE-002 | AC-006, AC-014 | UT-003, IT-002 | Unit / Integration | Yes |
| RULE-003 | AC-012, AC-017 | UT-006, UT-009, AT-009, AT-012 | Unit / Acceptance | Yes |
| RULE-004 | AC-018 | UT-007, AT-012 | Unit / Acceptance | Yes |

## 3. Acceptance Scenarios

### AT-001: Feed tab replaces placeholder with timeline loading state
Requirement IDs: BR-001  
Acceptance Criteria: AC-005  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/feed/feed_page_test.dart`

```gherkin
Feature: Flutter home timeline
  Scenario: Feed tab begins loading timeline content
    Given the user is signed in and opens the Feed tab
    And the timeline repository request has not completed
    When FeedPage is rendered
    Then the app bar shows "Feed"
    And the body shows a timeline loading indicator
    And the old static placeholder body is not the primary content
```

### AT-002: Loaded timeline renders post cards
Requirement IDs: BR-001, FR-009  
Acceptance Criteria: AC-004  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/feed/feed_page_test.dart`

```gherkin
Feature: Flutter home timeline
  Scenario: Loaded timeline displays post-shaped rows
    Given the timeline repository returns posts with text, images, author fields, timestamps, and engagement state
    When FeedPage settles after loading
    Then each post text is visible
    And the author identity is visible
    And the row uses the existing post-card action affordances for comment, like, and repost
```

### AT-003: Empty timeline shows empty-feed state only
Requirement IDs: FR-008, NFR-004  
Acceptance Criteria: AC-009, AC-020  
Priority: Should  
Level: Acceptance  
Automation Target: `app/test/feed/feed_page_test.dart`

```gherkin
Feature: Flutter home timeline
  Scenario: Empty timeline has no suggestions
    Given the timeline repository returns items [] and no cursor
    When FeedPage renders
    Then a localized empty-feed message is shown
    And no onboarding card is shown
    And no discovery or recommendation card is shown
```

### AT-004: Initial load failure can retry first page
Requirement IDs: FR-006, NFR-004  
Acceptance Criteria: AC-008, AC-020  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/feed/feed_page_test.dart`

```gherkin
Feature: Flutter home timeline
  Scenario: Initial timeline error is recoverable
    Given the first timeline repository request fails
    When FeedPage renders the error state
    Then a localized feed error message is shown
    And a Retry action is available
    When the user taps Retry
    Then the first timeline page is requested again
```

### AT-005: Scrolling near the end appends next timeline page
Requirement IDs: FR-003, FR-005, NFR-003, RULE-002  
Acceptance Criteria: AC-006, AC-019  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/feed/feed_page_test.dart`

```gherkin
Feature: Flutter home timeline pagination
  Scenario: Timeline auto-loads the next page
    Given the first timeline page contains enough posts to scroll and cursor "c1"
    When the user scrolls near the end of the visible timeline
    Then Flutter requests the next page with cursor "c1"
    And the returned posts are appended after the first page
    And previously visible posts remain visible
```

### AT-006: Load-more failure preserves visible posts and retries same cursor
Requirement IDs: FR-007  
Acceptance Criteria: AC-007  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/feed/feed_page_test.dart`

```gherkin
Feature: Flutter home timeline pagination
  Scenario: Load-more error does not blank the feed
    Given the timeline shows posts from the first page and stores cursor "c1"
    And the next-page request fails
    When FeedPage renders the load-more error state
    Then the first-page posts remain visible
    And a retry affordance is shown near the bottom of the list
    When the user taps Retry
    Then Flutter requests the next page with cursor "c1" again
```

### AT-007: Tapping a timeline row opens the thread route
Requirement IDs: FR-010  
Acceptance Criteria: AC-010  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/feed/feed_page_test.dart`

```gherkin
Feature: Flutter home timeline navigation
  Scenario: Timeline row opens post thread
    Given a timeline row for author did "did:plc:alice" and rkey "post1" is visible
    When the user taps the row body
    Then Flutter navigates to the existing post thread route
    And the route path parameters contain did "did:plc:alice" and rkey "post1"
```

### AT-008: Timeline like and repost actions update the row
Requirement IDs: FR-011  
Acceptance Criteria: AC-011  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/feed/feed_page_test.dart`

```gherkin
Feature: Flutter home timeline interactions
  Scenario: User likes and reposts from the timeline
    Given a timeline row is visible and the repository succeeds for like and repost writes
    When the user taps Like
    Then the row shows the liked state and incremented like count
    When the user taps Repost
    Then the row shows the reposted state and incremented repost count
```

### AT-009: Timeline comment action uses composer and opens focused thread
Requirement IDs: FR-012, RULE-003  
Acceptance Criteria: AC-012  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/feed/feed_page_test.dart`

```gherkin
Feature: Flutter home timeline comments
  Scenario: User comments from a timeline row
    Given a timeline row is visible
    And creating a reply succeeds with a created reply post
    When the user taps Comment and submits text
    Then the existing reply composer flow is used
    And Flutter opens the thread route focused on the created reply
    And the timeline root row's reply count and viewer-replied state are updated
    And the reply is not inserted as a top-level timeline row
```

### AT-010: Only own timeline rows expose delete and delete removes row
Requirement IDs: FR-013  
Acceptance Criteria: AC-015  
Priority: Should  
Level: Acceptance  
Automation Target: `app/test/feed/feed_page_test.dart`

```gherkin
Feature: Flutter home timeline deletion
  Scenario: Own post can be deleted from timeline
    Given the signed-in viewer DID is "did:plc:viewer"
    And the timeline has one viewer-authored post and one other-authored post
    When the timeline rows render
    Then only the viewer-authored post exposes Delete post
    When the user confirms deletion of the viewer-authored post
    Then the delete repository method is called
    And the deleted row is removed from the timeline
```

### AT-011: Feed compose creates top-level post
Requirement IDs: FR-014  
Acceptance Criteria: AC-016  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/feed/feed_page_test.dart`

```gherkin
Feature: Flutter home timeline composition
  Scenario: User creates a top-level post from Feed
    Given FeedPage is loaded
    When the user activates the New post compose entry
    And submits valid post text
    Then the existing top-level post composer creates the post without a reply target
```

### AT-012: Created top-level post is optimistically prepended and deduped
Requirement IDs: FR-015, RULE-003, RULE-004  
Acceptance Criteria: AC-017, AC-018  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/feed/feed_page_test.dart`

```gherkin
Feature: Flutter home timeline composition
  Scenario: Created top-level post appears immediately once
    Given FeedPage is showing a live timeline
    When top-level post creation succeeds with URI "at://did:plc:viewer/social.craftsky.feed.post/new"
    Then the created post appears at the top of the visible timeline
    When a later fetched timeline page includes a post with the same URI
    Then the timeline still shows only one row for that URI
```

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-002, RULE-001 | AC-002 | Fake and abstract repository expose a timeline method with no handle/DID input. | Compile-time fake implementation plus a call to `repo.listTimeline(limit: 20)`. | Test double records only cursor/limit; no profile ID argument exists. | `app/test/feed/fakes/fake_post_repository.dart` usage in provider tests |
| UT-002 | FR-003, FR-004 | AC-003 | First timeline build stores items/cursor and uses bounded page size. | Repository returns two posts and cursor `next`. | Timeline state has two items, cursor `next`, `hasMore == true`, request limit equals chosen timeline page limit. | `app/test/feed/providers/timeline_provider_test.dart` |
| UT-003 | FR-003, FR-005, RULE-002 | AC-006, AC-014 | `loadMore` passes the exact opaque cursor and appends next page. | First page cursor `opaque:abc`, second page one post. | Repository receives cursor `opaque:abc`; state items are first-page then second-page posts; cursor advances. | `app/test/feed/providers/timeline_provider_test.dart` |
| UT-004 | FR-007 | AC-007 | Load-more failure preserves previous data and cursor. | First page item `a` cursor `c1`; second call throws; third succeeds. | After failure, `AsyncError` retains item `a` and cursor `c1`; retry uses `c1`. | `app/test/feed/providers/timeline_provider_test.dart` |
| UT-005 | FR-003 | AC-006 | Concurrent or terminal load-more calls are guarded. | State is loading-more, or state has `cursor == null`. | Additional `loadMore` call does not call repository. | `app/test/feed/providers/timeline_provider_test.dart` |
| UT-006 | FR-015, RULE-003 | AC-017 | Top-level create prepends to live timeline; reply create does not. | Live timeline with `old`; created top-level `new`; created reply `reply`. | Top-level yields `[new, old]`; reply does not create a top-level timeline row. | `app/test/feed/providers/create_post_provider_test.dart` plus timeline provider tests |
| UT-007 | FR-015, RULE-004 | AC-018 | Timeline state dedupes by URI during prepend and page merge. | Existing item URI `u1`; prepend/fetched page also contains URI `u1`. | Timeline contains one `u1`; order remains deterministic with new non-duplicate items appended/prepended appropriately. | `app/test/feed/providers/timeline_provider_test.dart` |
| UT-008 | FR-011 | AC-011 | Like/repost providers patch live timeline entries and roll back on failure. | Live timeline contains post `a`; like/repost succeeds and separately fails. | Success updates viewer flags/counts; failure restores prior row. | `app/test/feed/providers/toggle_post_interactions_provider_test.dart` |
| UT-009 | FR-012, RULE-003 | AC-012 | Reply creation from timeline updates root row without inserting reply row. | Timeline root with `replyCount: 0`; created reply with `reply` field. | Root row has `replyCount: 1` and `viewerHasReplied: true`; reply URI absent as top-level item. | `app/test/feed/providers/timeline_provider_test.dart` or feed page provider helper tests |
| UT-010 | FR-013 | AC-015 | Delete cache helper removes by stable post key and ignores missing rows. | Timeline `[a, b]`; delete `a`; delete `missing`. | After delete `a`, state `[b]`; missing delete leaves state unchanged. | `app/test/feed/providers/timeline_provider_test.dart` |

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | FR-001, NFR-001, NFR-002, BR-001 | AC-001, AC-002 | API client reads timeline with no cursor. | `Dio` with `ErrorMappingInterceptor`; `DioAdapter` mocks `/v1/feed/timeline` returning one `samplePost` and cursor. | Call `PostApiClient(dio).listTimeline()`. | GET path is `/v1/feed/timeline`; parsed `PostPage.items.length == 1`; cursor matches response. | `app/test/feed/data/post_api_client_test.dart` |
| IT-002 | FR-001, FR-005, RULE-002 | AC-006, AC-014 | API client passes cursor and limit as query parameters. | Mock `/v1/feed/timeline` expecting `{'cursor': 'c1', 'limit': '20'}`. | Call `listTimeline(cursor: 'c1', limit: 20)`. | Request matches expected query parameters exactly; cursor is not transformed. | `app/test/feed/data/post_api_client_test.dart` |
| IT-003 | FR-001 | AC-002 | API client handles empty page and errors consistently. | Mock successful `{'items': []}` and an error envelope response. | Call `listTimeline()` for each case. | Empty response parses with `cursor == null`; error maps through existing `ApiException` path. | `app/test/feed/data/post_api_client_test.dart` |
| IT-004 | FR-002, RULE-001 | AC-002 | Production repository delegates timeline call without handle/DID input. | Fake/stub `PostApiClient` or repository-targeted test seam. | Call `ApiPostRepository.listTimeline(limit: 20)`. | API receives only cursor/limit; returned `PostPage` is passed through. | `app/test/feed/data/api_post_repository_test.dart` or provider tests if no seam exists |

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Test |
|---|---|---|---|
| REG-001 | App continues using existing `Post`/`PostPage` wire models and does not add a competing feed-item envelope. | BR-002, NFR-002 | Existing `app/test/feed/models/post_test.dart`, `post_page_test.dart`, and new API tests continue parsing AppView post-shaped responses. |
| REG-002 | `PostCard` rendering and image/gallery behavior remain unchanged for profile/thread contexts. | FR-009 | Run `flutter test test/feed/widgets/post_card_test.dart test/feed/widgets/post_image_carousel_test.dart test/feed/widgets/post_image_gallery_test.dart`. |
| REG-003 | Existing top-level and reply composer behavior remains unchanged outside the Feed tab. | FR-012, FR-014 | Run `flutter test test/feed/widgets/post_composer_sheet_test.dart test/feed/providers/create_post_provider_test.dart`. |
| REG-004 | Existing profile post/comment interaction providers still update profile caches correctly. | FR-011 | Run `flutter test test/feed/providers/toggle_post_interactions_provider_test.dart test/profile/widgets/profile_posts_tab_test.dart test/profile/widgets/profile_comments_tab_test.dart`. |
| REG-005 | Existing profile post/comment pagination still works after any shared helper extraction. | FR-003, FR-007 | Run `flutter test test/feed/providers/user_posts_provider_test.dart test/profile/widgets/profile_posts_tab_test.dart test/profile/widgets/profile_comments_tab_test.dart`. |
| REG-006 | Generated Riverpod/mappable/l10n files are up to date. | NFR-004, NFR-005 | Run `dart run build_runner build --delete-conflicting-outputs` from `app/`, then inspect generated-file diff; run focused tests that import generated providers/localizations. |
| REG-007 | Shared AppView `Dio` auth/device interceptors are not bypassed by timeline calls. | NFR-001, RULE-001 | Run `flutter test test/shared/api/providers/session_auth_interceptor_test.dart test/shared/device/device_id_provider_test.dart test/shared/api/providers/dio_provider_test.dart`. |
| REG-008 | Router shell still lands signed-in users on Feed and thread routes still parse DID/rkey. | FR-010 | Run `flutter test test/router/router_redirect_test.dart test/app_test.dart` plus Feed row navigation widget test. |

## 7. Test Data

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Basic timeline posts | `Post` maps for Alice/Bob with `uri`, `cid`, `rkey`, `text`, `tags`, counts, viewer flags, `createdAt`, `indexedAt`, and author `{did, handle, displayName}`. | AT-002, AT-005, UT-002, IT-001 |
| TD-002 | Empty timeline | `PostPage(items: const [], cursor: null)`. | AT-003, UT-005, IT-003 |
| TD-003 | Pagination sequence | First page: 10+ posts, cursor `c1`; second page: one post, cursor null; failing second-page variant throws once then succeeds with same cursor. | AT-005, AT-006, UT-003, UT-004 |
| TD-004 | Optimistic create and dedupe | Existing timeline `[old]`; created top-level post `new` with unique URI; duplicate fetched post with same URI; created reply with `reply` field. | AT-012, UT-006, UT-007, UT-009 |
| TD-005 | Interactions | Post with `likeCount`, `repostCount`, `viewerHasLiked`, `viewerHasReposted`; `InteractionWriteResponse` for like/repost; failing repository calls. | AT-008, UT-008, REG-004 |
| TD-006 | Ownership/delete | Signed-in viewer DID `did:plc:viewer`; one viewer-authored post and one other-authored post. | AT-010, UT-010 |
| TD-007 | Rich post-card rendering | Post with images and optional quote strong reference, using existing `PostImage` fields. | AT-002, REG-002, MAN-001 |

## 8. Manual Checks

| ID | Requirement IDs | Check | Steps | Expected Result |
|---|---|---|---|---|
| MAN-001 | FR-009, NFR-003, NFR-004 | Visual and accessibility smoke check for the Feed tab. | Run the app with a fake/seeded AppView timeline containing text, image, long-text, empty, and paginated states; inspect small and large form factors; use screen reader/semantics inspection where available. | Feed is readable, lazy scrolling feels stable, post actions have accessible labels/tooltips, empty/error/retry copy is localized and understandable. |
| MAN-002 | BR-002, NFR-002 | Scope review for no speculative feed framework. | Inspect changed files after implementation. | Timeline code uses a focused timeline provider/API path and existing post/list models; no generic feed-item envelope or project/search/list framework is introduced. |

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | No full-device E2E test against a live AppView is required for this stage. | BR-001, NFR-001 | Existing Flutter suite appears unit/widget focused; full stack E2E would require running AppView and authenticated session setup. | Cover with mocked API/widget tests now; consider an integration-test harness after app test infrastructure matures. |
| GAP-002 | Performance/laziness is only partially automated. | NFR-003 | Widget tests can verify bounded calls and sliver-style behavior, but smoothness and memory use are better observed manually. | Use MAN-001 now; consider performance tests when timeline volume grows. |
| GAP-003 | Localization code generation is command-verified rather than asserted by a dedicated unit test. | NFR-004, NFR-005 | Flutter localization generation is build tooling, not app logic. | Require generated files to be committed and run focused widget tests importing generated localizations. |

## 10. Out Of Scope

- AppView endpoint, database, Go route, and Go test changes; those belong to `2026-05-28-timeline-feed-appview` and are already implemented.
- PDS read-through tests; Flutter must not read timeline craft data directly from PDSes.
- Feed ranking, recommendations, onboarding suggestions, search, project filters, list/custom feeds, and discovery surfaces.
- Repost feed reasons and nested quote-card expansion.
- Durable local timeline cache, offline sync, background refresh, push/live updates, or analytics events.
- Lexicon, dependency, and migration checks beyond ensuring this Flutter change does not touch them.

## 11. Handoff To Document Review

- Requirements file: `docs/changes/2026-05-29-timeline-feed-flutter/01-requirements.md`
- Test specification: `docs/changes/2026-05-29-timeline-feed-flutter/02-acceptance-tests.md`
- Next review artifact: `docs/changes/2026-05-29-timeline-feed-flutter/03-document-review.md`
- Preferred Plannotator review: `/plannotator-annotate docs/changes/2026-05-29-timeline-feed-flutter/`
- Recommended first failing test for implementation: `IT-001` — add `PostApiClient.listTimeline` API-client test in `app/test/feed/data/post_api_client_test.dart`.
- Suggested test order for implementation:
  1. `IT-001` API client no-cursor timeline parsing.
  2. `IT-002` API client `cursor`/`limit` query params and opaque pass-through.
  3. `IT-003` API empty page and error mapping.
  4. `IT-004` repository/fake timeline method plumbing.
  5. `UT-002` first timeline provider build and bounded page size.
  6. `UT-003`, `UT-004`, `UT-005` pagination append, failure retry, and guards.
  7. `UT-006`, `UT-007` optimistic top-level prepend and URI dedupe.
  8. `AT-001` through `AT-006` FeedPage loading, loaded, empty, initial-error, pagination, load-more retry.
  9. `AT-007` through `AT-011` row navigation, like/repost, comment, delete, and top-level compose.
  10. `AT-012` end-to-end optimistic create/dedupe widget behavior.
  11. `UT-008` through `UT-010` cache updates for interactions, replies, and delete if not already covered by shared helpers.
  12. `REG-001` through `REG-008` focused regression suites and generated-code verification.
- Commands discovered:
  - `cd app && flutter test test/feed/data/post_api_client_test.dart`
  - `cd app && flutter test test/feed/providers/timeline_provider_test.dart`
  - `cd app && flutter test test/feed/feed_page_test.dart`
  - `cd app && flutter test test/feed/providers/create_post_provider_test.dart test/feed/providers/toggle_post_interactions_provider_test.dart`
  - `cd app && flutter test test/profile/widgets/profile_posts_tab_test.dart test/profile/widgets/profile_comments_tab_test.dart`
  - `cd app && dart run build_runner build --delete-conflicting-outputs`
  - `cd app && flutter test`
- Blocking gaps: None.
- Risk-based review recommendation: Medium risk; document review is recommended before coding plan/implementation because the test plan touches user-visible feed UI, pagination, optimistic cache updates, and shared post interaction behavior.
