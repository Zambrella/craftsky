# Acceptance Test Specification: Reposts And Quote Posts

## 1. Test Strategy
This is a medium-risk full-stack API, feed, and Flutter UI change. The test design uses Go store and handler tests for AppView behavior, Go response-builder and indexer tests for data contracts, and Flutter model, API client, provider, and widget tests for client behavior. Acceptance scenarios focus on user-visible repost and quote workflows. Regression tests protect existing profile, search, project-post, notification, lexicon, and session-token contracts.

Manual checks are limited to final visual/UX confirmation because the core behavior can be automated in existing Go and Flutter suites.

## 2. Requirement Coverage Matrix
| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-003 | AT-001, AT-003, IT-001, IT-006 | Acceptance / Integration | Yes |
| BR-002 | AC-002, AC-004 | AT-002, AT-004, IT-002, IT-007 | Acceptance / Integration | Yes |
| BR-003 | AC-005 | AT-005, IT-004 | Acceptance / Integration | Yes |
| BR-004 | AC-006 | AT-006, MAN-001 | Acceptance / Manual | Partial |
| FR-001 | AC-001, AC-007 | AT-001, IT-006, REG-004 | Acceptance / Integration / Regression | Yes |
| FR-002 | AC-002, AC-008 | AT-002, UT-001, IT-007 | Acceptance / Unit / Integration | Yes |
| FR-003 | AC-006, AC-027 | AT-006, UT-007 | Acceptance / Unit | Yes |
| FR-004 | AC-002, AC-004, AC-028 | AT-002, UT-001, IT-007 | Acceptance / Unit / Integration | Yes |
| FR-005 | AC-003, AC-009, AC-029 | AT-003, IT-001, IT-005 | Acceptance / Integration | Yes |
| FR-006 | AC-004, AC-010 | AT-004, IT-002 | Acceptance / Integration | Yes |
| FR-007 | AC-009 | AT-003, UT-004, IT-001 | Acceptance / Unit / Integration | Yes |
| FR-008 | AC-011, AC-030 | AT-004, AT-007, UT-003, IT-003 | Acceptance / Unit / Integration | Yes |
| FR-009 | AC-012 | AT-007, IT-003, IT-010 | Acceptance / Integration | Yes |
| FR-010 | AC-013, AC-031 | AT-008, UT-004, UT-006, UT-008 | Acceptance / Unit | Yes |
| FR-011 | AC-014 | AT-008, UT-004, IT-008 | Acceptance / Unit / Integration | Yes |
| FR-012 | AC-015 | AT-008, IT-008 | Acceptance / Integration | Yes |
| FR-013 | AC-016 | REG-001 | Regression | Yes |
| FR-014 | AC-017 | REG-002 | Regression | Yes |
| FR-015 | AC-018 | REG-003 | Regression | Yes |
| FR-016 | AC-019 | REG-004 | Regression | Yes |
| FR-017 | AC-032 | AT-003, IT-005, REG-006 | Acceptance / Integration / Regression | Yes |
| FR-018 | AC-033 | AT-006, UT-007 | Acceptance / Unit | Yes |
| FR-019 | AC-034 | AT-009, UT-009 | Acceptance / Unit | Yes |
| FR-020 | AC-035 | AT-010, UT-010 | Acceptance / Unit | Yes |
| FR-021 | AC-036 | REG-005 | Regression | Yes |
| NFR-001 | AC-005, AC-020 | AT-005, IT-004 | Acceptance / Integration | Yes |
| NFR-002 | AC-021 | UT-012, IT-012 | Unit / Integration | Yes |
| NFR-003 | AC-022 | IT-013, GAP-001 | Integration | Partial |
| NFR-004 | AC-023 | UT-013, IT-014 | Unit / Integration | Yes |
| NFR-005 | AC-024, AC-034, AC-035 | AT-009, AT-010, UT-009, UT-010 | Acceptance / Unit | Yes |
| RULE-001 | AC-001, AC-002, AC-014, AC-015 | AT-001, AT-002, AT-008, IT-008 | Acceptance / Integration | Yes |
| RULE-002 | AC-007 | IT-006 | Integration | Yes |
| RULE-003 | AC-025 | IT-009 | Integration | Yes |
| RULE-004 | AC-009 | AT-003, UT-004 | Acceptance / Unit | Yes |
| RULE-005 | AC-011 | AT-004, AT-007, UT-003 | Acceptance / Unit | Yes |
| RULE-006 | AC-012, AC-026 | AT-007, IT-010 | Acceptance / Integration | Yes |
| RULE-007 | AC-019 | REG-004 | Regression | Yes |
| RULE-008 | AC-027, AC-037 | AT-006, IT-007 | Acceptance / Integration | Yes |
| RULE-009 | AC-038 | AT-011, IT-015 | Acceptance / Integration | Yes |
| RULE-010 | AC-012, AC-026, AC-039 | AT-007, IT-010 | Acceptance / Integration | Yes |
| RULE-011 | AC-040 | IT-016 | Integration | Yes |

## 3. Acceptance Scenarios
### AT-001: Straight Repost An Eligible Target
Requirement IDs: BR-001, FR-001, RULE-001
Acceptance Criteria: AC-001, AC-007
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/post_test.go`

```gherkin
Feature: Straight reposts
  Scenario: Reposter shares an eligible indexed post without commentary
    Given the viewer is authenticated with a Craftsky session
    And the target is a visible indexed top-level post or project post
    When the viewer chooses straight repost
    Then AppView writes a social.craftsky.feed.repost record to the viewer's PDS
    And the record subject references the target post uri and cid
    And repeating the request does not create a second active repost for the same subject
```

### AT-002: Quote Post An Eligible Target
Requirement IDs: BR-002, FR-002, FR-004, RULE-001
Acceptance Criteria: AC-002, AC-004, AC-008, AC-028
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/post_test.go`, `app/test/feed/data/post_api_client_test.dart`

```gherkin
Feature: Quote posts
  Scenario: Quote poster shares an eligible post with commentary
    Given the viewer is authenticated with a Craftsky session
    And the target is a visible indexed top-level post or project post
    And the viewer enters non-empty commentary
    When the viewer submits the quote
    Then AppView writes a social.craftsky.feed.post record
    And the record uses social.craftsky.feed.post#quoteEmbed with the target uri and cid
    And an empty or whitespace-only quote is rejected without a PDS write
```

### AT-003: Timeline Shows Followed Straight Reposts With Attribution
Requirement IDs: BR-001, FR-005, FR-007, FR-017, RULE-004
Acceptance Criteria: AC-003, AC-009, AC-029, AC-032
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/timeline_store_test.go`, `appview/internal/api/timeline_test.go`, `app/test/feed/feed_page_test.dart`

```gherkin
Feature: Home timeline repost activity
  Scenario: Viewer sees a followed user's repost as a distinct feed item
    Given Alice follows Bob
    And Bob straight-reposts Carol's visible post
    When Alice loads the home timeline
    Then Carol's original post appears as a feed item
    And the feed item reason says it was reposted by Bob
    And the reason includes Bob's actor summary, repost record identity, and repost timestamp
```

### AT-004: Timeline Shows Quote Posts As Authored Posts
Requirement IDs: BR-002, FR-006, FR-008, RULE-005
Acceptance Criteria: AC-004, AC-010, AC-011, AC-030
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/timeline_store_test.go`, `app/test/feed/widgets/post_card_test.dart`

```gherkin
Feature: Quote posts in the timeline
  Scenario: Viewer sees a followed user's quote post
    Given Alice follows Bob
    And Bob quote-posts Carol's visible post
    When Alice loads the home timeline
    Then Bob's quote post appears as Bob's authored post
    And the post renders a compact preview attributed to Carol
    And if Carol's post is itself a quote post, nested quote previews are not hydrated
```

### AT-005: Mixed Timeline Remains Chronological And Paginated
Requirement IDs: BR-003, NFR-001
Acceptance Criteria: AC-005, AC-020
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/timeline_store_test.go`, `app/test/feed/providers/timeline_provider_test.dart`

```gherkin
Feature: Chronological timeline
  Scenario: Authored posts and straight reposts are ordered by activity time
    Given a timeline contains followed authored posts and followed repost activity
    When the viewer requests pages using opaque cursors
    Then feed items are ordered newest-first by post or repost activity timestamp
    And deterministic tie-breakers keep pagination stable
    And no ranking score changes item order
    And no eligible item is skipped or duplicated across pages
```

### AT-006: Repost Share Control Offers Correct Actions
Requirement IDs: BR-004, FR-003, FR-018, RULE-008
Acceptance Criteria: AC-006, AC-027, AC-033
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/widgets/post_card_test.dart`, `app/test/feed/widgets/post_composer_sheet_test.dart`

```gherkin
Feature: Repost and quote UI
  Scenario: Viewer opens the share menu for an eligible post
    Given a post card is an eligible top-level post or project post
    When the viewer activates the repost/share control or combined count
    Then the UI offers separate straight repost and quote actions
    And selecting quote opens the quote composer
    And no reposts/quotes list screen opens
    And reply cards do not offer repost or quote actions
```

### AT-007: Hidden Or Unavailable Quote Targets Render As Placeholders
Requirement IDs: FR-009, RULE-005, RULE-006, RULE-010
Acceptance Criteria: AC-012, AC-026, AC-039
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/post_response_test.go`, `app/test/feed/widgets/post_card_test.dart`

```gherkin
Feature: Quote preview availability
  Scenario: Quoted content is unavailable but the quote post remains visible
    Given a visible quote post references a missing, deleted, unindexed, hidden, or blocked target
    When the quote post is served
    Then the quote post still renders its author's commentary
    And the quoted preview uses a stable unavailable or hidden placeholder
    And straight reposts of hidden or unavailable subjects are omitted completely
```

### AT-008: Repost And Quote Engagement Remain Distinct
Requirement IDs: FR-010, FR-011, FR-012, RULE-001
Acceptance Criteria: AC-013, AC-014, AC-015, AC-031
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/post_response_test.go`, `appview/internal/api/post_test.go`, `app/test/feed/widgets/post_card_test.dart`

```gherkin
Feature: Engagement semantics
  Scenario: Viewer has quoted and straight-reposted the same subject
    Given the viewer has authored one or more quote posts for a subject
    And the viewer has one active straight repost for the same subject
    When the subject post is returned
    Then repostCount and quoteCount are exposed separately
    And viewerHasReposted reflects only the active straight repost
    When the viewer unreposts the subject
    Then only the straight repost record is deleted
    And the authored quote posts remain
```

### AT-009: Straight Repost Optimistic UI Does Not Insert Timeline Items
Requirement IDs: FR-019, NFR-005
Acceptance Criteria: AC-024, AC-034
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/providers/toggle_post_interactions_provider_test.dart`, `app/test/feed/feed_page_test.dart`

```gherkin
Feature: Repost optimistic updates
  Scenario: Repost succeeds or fails from a loaded timeline
    Given the timeline has a visible post row
    When the viewer straight-reposts the post
    Then the action state and visible count update optimistically
    And no repost feed item is inserted into the local timeline cache
    When the API call fails
    Then the action state and count roll back to the prior visible state
    And an error is emitted through existing messaging patterns
```

### AT-010: Quote Creation Uses Normal Post Cache Behavior
Requirement IDs: FR-020, NFR-005
Acceptance Criteria: AC-024, AC-035
Priority: Should
Level: Acceptance
Automation Target: `app/test/feed/providers/create_post_provider_test.dart`, `app/test/feed/feed_page_test.dart`

```gherkin
Feature: Quote post cache updates
  Scenario: Quote post creation completes
    Given the viewer creates a quote post
    When the create request succeeds
    Then Flutter applies the same cache insertion or refresh behavior used by normal post creation
    And no quote-specific timeline insertion path is used
    When the create request fails after optimistic state
    Then affected visible state rolls back consistently with normal post creation
```

### AT-011: Users Can Share Their Own Eligible Content
Requirement IDs: RULE-009
Acceptance Criteria: AC-038
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/post_test.go`, `app/test/feed/widgets/post_card_test.dart`

```gherkin
Feature: Self sharing
  Scenario: Viewer reposts or quotes their own eligible content
    Given the target is the viewer's own visible top-level post or project post
    When the viewer straight-reposts or quote-posts the target
    Then the action succeeds using the same record semantics as sharing another user's post
```

## 4. Unit Test Cases
| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-002, FR-004 | AC-002, AC-008, AC-028 | Validate quote create request body and non-empty commentary rule before PDS write. | Quote create requests with valid text, empty text, whitespace text, target `{uri,cid}`, optional images. | Valid request builds Craftsky `post#quoteEmbed`; invalid text returns validation error and fake PDS create count remains zero. | `appview/internal/api/post_test.go` |
| UT-002 | FR-003, RULE-008 | AC-027, AC-037 | Determine whether a post card is eligible for repost/quote actions. | Post models for top-level post, project post, root comment, nested reply. | Top-level and project posts expose the share affordance; reply cards do not expose repost or quote choices. | `app/test/feed/widgets/post_card_test.dart` |
| UT-003 | FR-008, FR-009, RULE-005 | AC-011, AC-012, AC-030 | Build compact quote preview states without recursive nested hydration. | Post rows with visible quote target, hidden target, unavailable target, quote-of-quote target. | Response includes visible preview, hidden/unavailable placeholder, or one-level quote preview with no nested preview. | `appview/internal/api/post_response_test.go` |
| UT-004 | FR-007, FR-010, FR-011, RULE-004 | AC-009, AC-013, AC-014 | Build engagement and repost-reason response models. | Post rows with separate repost/quote counts, viewer straight repost, viewer quote only, repost reason row. | JSON exposes `repostCount`, `quoteCount`, correct `viewerHasReposted`, and repost reason actor/record/timestamp data in camelCase. | `appview/internal/api/post_response_test.go` |
| UT-005 | NFR-001 | AC-005, AC-020 | Encode and compare feed-item seek cursors for mixed post/repost activity. | Activity timestamps and stable item IDs with tied timestamps across post and repost item types. | Cursor comparison returns deterministic newest-first order and stable next-page boundaries. | `appview/internal/api/envelope/cursor_test.go`, `appview/internal/api/timeline_store_test.go` |
| UT-006 | FR-010 | AC-013, AC-031 | Decode Flutter `Post` with separate repost and quote counts. | Wire payload containing `repostCount`, `quoteCount`, and current engagement fields. | Model preserves both counts, round-trips JSON, and existing payloads without `quoteCount` use the intended default. | `app/test/feed/models/post_test.dart` |
| UT-007 | FR-003, FR-018 | AC-006, AC-027, AC-033 | Repost/share control opens a choice menu and count tap follows the same path. | Eligible post card, reply post card, taps on icon and count. | Eligible rows show straight repost and quote choices; reply rows hide them; count tap does not navigate to a list. | `app/test/feed/widgets/post_card_test.dart` |
| UT-008 | FR-010 | AC-031 | Render a combined share count from separate model values. | Post model with `repostCount: 2`, `quoteCount: 3`. | The action row may display `5`, while the model still exposes `2` and `3` separately. | `app/test/feed/widgets/post_card_test.dart` |
| UT-009 | FR-019, NFR-005 | AC-024, AC-034 | Optimistically patch straight repost state without timeline insertion. | Loaded timeline/profile/project caches, repost success and failure responses. | Count/state patch and rollback work; cache item count and feed item identities do not change because of optimistic repost. | `app/test/feed/providers/toggle_post_interactions_provider_test.dart` |
| UT-010 | FR-020, NFR-005 | AC-024, AC-035 | Confirm quote create uses normal post-create provider cache path. | Create post provider invoked with quote ref and normal post fixtures. | Quote creation follows existing create-post cache behavior with no quote-specific timeline insertion branch. | `app/test/feed/providers/create_post_provider_test.dart` |
| UT-011 | FR-021 | AC-036 | Ensure quote create does not map to a new notification type in client models. | Notification payload/model registrations before and after quote support. | No new quote-specific notification enum or mapper is required by this feature. | `app/test/notifications/models/notification_test.dart` |
| UT-012 | NFR-002 | AC-021 | Verify new API response/request structs use camelCase JSON tags. | Marshaled feed item, quote preview, quote count, repost reason, and validation error payloads. | JSON contains camelCase keys and does not leak snake_case fields. | `appview/internal/api/post_response_test.go`, `appview/internal/api/envelope/envelope_test.go` |
| UT-013 | NFR-004 | AC-023 | Verify Flutter repost, unrepost, and quote APIs use the session-auth path only. | API client tests with Dio interceptors and mocked endpoints. | Requests target AppView `/v1/*` endpoints and do not contain or decode PDS OAuth token fields. | `app/test/feed/data/post_api_client_test.dart`, `app/test/shared/api/providers/session_auth_interceptor_test.dart` |

## 5. Integration Test Cases
| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | BR-001, FR-005, FR-007 | AC-003, AC-009, AC-029 | Timeline store returns followed straight reposts as distinct feed items. | Seed Alice follows Bob; Bob reposts Carol; multiple followed users repost same original. | List Alice's timeline. | Each repost appears as a separate feed item with original post data and correct reposter reason. | `appview/internal/api/timeline_store_test.go` |
| IT-002 | BR-002, FR-006 | AC-004, AC-010 | Timeline includes quote posts by followed authors as normal authored posts. | Seed Alice follows Bob; Bob authors quote post of Carol's visible post. | List Alice's timeline. | Bob's quote post appears by followed-author rules with no repost reason. | `appview/internal/api/timeline_store_test.go` |
| IT-003 | FR-008, FR-009 | AC-011, AC-012, AC-030 | Quote hydration returns visible preview, placeholder state, and one-level depth. | Seed quote posts targeting visible, hidden, missing, and nested quote targets. | Read post detail and timeline rows that include quote posts. | Visible target hydrates compact preview; unavailable target returns stable state; nested target is not recursively hydrated. | `appview/internal/api/post_store_test.go`, `appview/internal/api/post_response_test.go` |
| IT-004 | BR-003, NFR-001 | AC-005, AC-020 | Mixed feed ordering and pagination are deterministic. | Seed authored posts and repost activity with tied timestamps and enough rows for multiple pages. | Page through the home timeline using returned cursors. | Items are reverse-chronological by activity timestamp with deterministic tie-breakers; no skip or duplicate occurs. | `appview/internal/api/timeline_store_test.go` |
| IT-005 | FR-017 | AC-032 | Only home timeline uses feed-item shape. | Handler tests for home timeline, profile, search, thread, and post detail. | Call each endpoint through `httptest`. | Home timeline returns `items: [{post, reason}]`; other surfaces remain post-shaped. | `appview/internal/api/timeline_test.go`, `appview/internal/api/post_test.go`, `appview/internal/api/search_response_test.go` |
| IT-006 | FR-001, RULE-002 | AC-001, AC-007 | Straight repost create remains idempotent and uses repost collection. | Fake authenticated request and fake PDS; existing active repost row for duplicate case. | POST `/v1/posts/{did}/{rkey}/reposts`. | PDS create uses `social.craftsky.feed.repost` with subject strongRef; duplicate active repost is preserved or returned without duplicate active DB state. | `appview/internal/api/post_test.go`, `appview/internal/index/craftsky_interaction_test.go` |
| IT-007 | FR-002, FR-004, RULE-008 | AC-002, AC-008, AC-028, AC-037 | Quote create validates target eligibility and write shape. | Seed visible top-level target, project target, reply target, missing target, and hidden target. | POST `/v1/posts` with quote ref and commentary. | Eligible targets write Craftsky quote embed; reply/missing/hidden targets return validation errors and no PDS write. | `appview/internal/api/post_test.go` |
| IT-008 | FR-011, FR-012 | AC-014, AC-015 | Viewer quote state is independent from straight repost state and unrepost. | Viewer has quote posts for subject with and without active straight repost. | Read subject, then DELETE `/v1/posts/{did}/{rkey}/reposts`. | Quote-only viewer has `viewerHasReposted: false`; unrepost deletes only active repost and quote posts remain indexed. | `appview/internal/api/post_test.go`, `appview/internal/api/post_store_test.go` |
| IT-009 | RULE-003 | AC-025 | Multiple quote posts for one subject are allowed. | Seed a user with one existing quote post for a target. | Submit another valid quote post for the same target. | Normal post creation may succeed; no toggle-style deduplication or active-unique constraint is applied to quotes. | `appview/internal/api/post_test.go`, `appview/internal/index/craftsky_post_test.go` |
| IT-010 | RULE-006, RULE-010 | AC-012, AC-026, AC-039 | Moderation filtering applies to repost subjects and quoted previews. | Seed hidden/taken-down original posts, hidden quote targets, hidden authors, and visible quote posts. | List timeline and read post detail as affected viewer. | Straight reposts of hidden/unavailable subjects are omitted; quote posts remain only when the quote post itself is visible and preview state reflects policy. | `appview/internal/api/timeline_store_test.go`, `appview/internal/api/moderation_policy_test.go` |
| IT-011 | FR-021 | AC-036 | Quote post creation does not emit a new quote notification type. | Existing notification store fixtures plus quote post create/index event. | Run notification generation or store read path used by current tests. | No new quote-specific notification row/type is created by this feature. | `appview/internal/api/notification_store_test.go`, `appview/internal/api/notifications_test.go` |
| IT-012 | NFR-002 | AC-021 | API casing and error envelope remain consistent for new/changed routes. | Handler tests for timeline feed items, quote validation errors, hidden target errors. | Call endpoints via `httptest`. | Success bodies use camelCase; errors use `{error, message, requestId}` with optional `fields`. | `appview/internal/api/timeline_test.go`, `appview/internal/api/post_test.go`, `appview/internal/api/envelope/envelope_test.go` |
| IT-013 | NFR-003 | AC-022 | Timeline and quote hydration use bounded/batched query plans where practical. | Seed a normal page containing posts, reposts, authors, and quote previews. | Exercise store method under DB query-count instrumentation or query-plan-specific test helpers. | Author/repost/quote hydration is performed with bounded batch queries rather than one unbounded query per item. | `appview/internal/api/timeline_store_test.go`, `appview/internal/api/post_store_test.go` |
| IT-014 | NFR-004 | AC-023 | AppView mediates PDS writes without returning PDS tokens to Flutter. | Authenticated repost, unrepost, and quote-post handler tests using fake PDS/session store. | Call write endpoints. | Flutter-facing responses contain Craftsky result data only; PDS OAuth tokens are not serialized. | `appview/internal/api/post_test.go`, `appview/internal/auth/craftsky_session_test.go` |
| IT-015 | RULE-009 | AC-038 | Self-reposts and self-quotes are accepted for eligible targets. | Seed viewer's own top-level post and project post. | Straight repost and quote each own target. | Requests succeed with the same semantics as sharing another user's eligible target. | `appview/internal/api/post_test.go` |
| IT-016 | RULE-011 | AC-040 | Counts exclude hidden records where policy can be evaluated. | Seed visible and hidden repost/quote records for a subject under viewer-specific policy. | Read post-shaped response as the viewer. | `repostCount` and `quoteCount` include only records visible under the applicable AppView policy. | `appview/internal/api/post_store_test.go`, `appview/internal/api/moderation_policy_test.go` |

## 6. Regression Tests
| ID | Existing Behavior Protected | Requirement IDs | Acceptance Criteria | Test |
|---|---|---|---|---|
| REG-001 | Profile post lists remain authored-post surfaces. | FR-013 | AC-016 | Extend `appview/internal/api/profile_store_query_test.go` and `app/test/profile/widgets/profile_posts_tab_test.dart` so authored quote posts may appear, while straight repost interaction records do not. |
| REG-002 | Search scopes do not broaden because of repost/quote work. | FR-014 | AC-017 | Extend `appview/internal/api/search_store_test.go` and `app/test/search/providers/post_search_provider_test.dart` to keep existing quote-post exclusion/inclusion rules unchanged. |
| REG-003 | Project posts cannot themselves be replies or quote posts. | FR-015 | AC-018 | Keep or extend `appview/internal/index/craftsky_post_test.go` and create-post handler validation tests for project plus quote/reply rejection. |
| REG-004 | Repost lexicon stays unchanged unless a backward-compatible optional change is intentionally made and regenerated. | FR-016, RULE-007 | AC-019 | If lexicon files change, use the atproto lexicon checklist and run `just lexgen`; otherwise verify no `lexicon/social/craftsky/feed/repost.json` diff is present for this feature. |
| REG-005 | No new quote-specific notification behavior is introduced. | FR-021 | AC-036 | Extend notification store/API tests to prove quote-post create does not add a dedicated notification type. |
| REG-006 | Profile, search, thread, and post detail remain post-shaped while only home timeline changes to feed items. | FR-017 | AC-032 | Add handler/model tests for each non-home surface to reject accidental `reason` wrapping outside the home timeline. |
| REG-007 | Existing straight repost/unrepost endpoints keep current path and idempotent delete behavior. | FR-001, FR-012 | AC-001, AC-007, AC-015 | Preserve `POST /v1/posts/{did}/{rkey}/reposts` and `DELETE /v1/posts/{did}/{rkey}/reposts` tests, including unrepost with no active repost returning the existing success behavior. |

## 7. Test Data
| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Standard actor graph | Alice viewer follows Bob and Dana; Carol is an original author; Eve is unfollowed. Include Craftsky member rows and Bluesky profile summaries. | AT-003, AT-004, IT-001, IT-002, IT-004 |
| TD-002 | Eligible share targets | Visible top-level general post and visible project post with stable `{uri,cid}` and indexed timestamps. | AT-001, AT-002, IT-006, IT-007, IT-015 |
| TD-003 | Ineligible share targets | Reply post, nested reply post, missing/unindexed URI, hidden post, hidden author post. | AT-006, AT-007, IT-007, IT-010 |
| TD-004 | Repost activity rows | Active `craftsky_reposts` rows with unique reposter/subject pairs, duplicate attempt data, deleted repost row, tied timestamps. | AT-003, AT-005, IT-001, IT-004, IT-006 |
| TD-005 | Quote post rows | Quote posts with visible target, project target, missing target, hidden target, quote-of-quote target, quote with top-level images. | AT-002, AT-004, AT-007, IT-002, IT-003 |
| TD-006 | Engagement/count data | Mix of visible and hidden reposts/quotes for the same subject plus viewer quote-only and viewer straight-repost states. | AT-008, IT-008, IT-016 |
| TD-007 | Flutter wire payloads | Home timeline feed-item JSON with `post` and nullable `reason`; post-shaped JSON with `quoteCount`; quote preview visible/unavailable/hidden states. | UT-006, UT-007, UT-008, IT-005 |
| TD-008 | Failure fixtures | Validation errors, PDS write failure, network failure, and hidden-target rejection using standard error envelope. | AT-009, AT-010, UT-009, UT-010, IT-012 |

## 8. Manual Checks
| ID | Requirement IDs | Acceptance Criteria | Check | Steps | Expected Result |
|---|---|---|---|---|---|
| MAN-001 | BR-004, FR-003, FR-004, FR-008 | AC-006, AC-011 | Final UX smoke check for Bluesky-familiar repost/quote flow and visual fit. | Run the Flutter app against seeded data; open eligible post, project post, reply, visible quote, unavailable quote, and straight repost feed item on mobile and desktop widths. | Action menu, quote composer preview, quoted-card rendering, hidden reply actions, and "reposted by" attribution are clear, non-overlapping, and visually consistent. |

## 9. Test Gaps And Risks
| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | Exact N+1 query prevention may need additional instrumentation. | NFR-003 | Existing tests exercise store behavior but do not appear to include a reusable query-count harness. | Prefer a bounded-query integration test if practical; otherwise document the query plan and add targeted store tests around batched author and quote hydration. |
| GAP-002 | Full real-PDS OAuth behavior is not covered by this acceptance stage. | NFR-004 | Handler tests can prove Flutter-facing responses do not expose tokens, but real OAuth/PDS integration is outside this feature's test scope. | Keep existing auth integration coverage and do not add PDS-token fields to any client model. |

## 10. Out Of Scope
- Algorithmic ranking, recommendations, paid reach, feed generators, and XRPC surfaces.
- Literal migration to Bluesky `app.bsky.embed.record` or `recordWithMedia`.
- Reposting or quoting reply posts.
- Quote-detach, postgate, quote-disable, anti-dogpile controls, and reposts/quotes list screens.
- Private reposts or private quote posts.
- End-to-end tests against a real production PDS.

## 11. Handoff To Document Review
- Requirements file: `01-requirements.md`
- Test specification: `02-acceptance-tests.md`
- Next review artifact: `03-document-review.md`
- External Plannotator review, if the user initiates it outside this skill: `docs/changes/2026-07-08-reposts-and-quotes/`
- Recommended first failing test for implementation: `IT-001` by changing the current `TestTimelineStore_ListTimeline_ExcludesConversationAndRepostActivity` coverage into a feed-item test that includes followed straight repost activity with reason attribution while still excluding replies.
- Suggested test order for implementation:
  1. `IT-001`, `IT-004`, `IT-005` for home timeline feed-item shape, ordering, and pagination.
  2. `UT-004`, `UT-006`, `UT-012` for response/model contract changes.
  3. `IT-003`, `IT-010`, `IT-016` for quote hydration, moderation filtering, and visibility-aware counts.
  4. `IT-006`, `IT-007`, `IT-008`, `IT-009`, `IT-015` for write semantics and validation.
  5. `UT-007`, `UT-008`, `UT-009`, `UT-010`, then `AT-006` through `AT-010` for Flutter UI and cache behavior.
  6. Regression tests `REG-001` through `REG-007`.
- Commands discovered:
  - `just dev-d` starts the compose stack for host Go tests.
  - `just test` runs Go tests against compose Postgres.
  - `just app-test` runs Flutter tests.
  - `just app-analyze` runs Flutter analysis.
  - `just lexgen` is required only if lexicon files change.
- Blocking gaps: None.
