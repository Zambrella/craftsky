# Acceptance Test Specification: Post Interaction Details

## 1. Test Strategy

Use a layered, test-first approach for the three interaction-list endpoints and their Flutter presentation:

- Go store integration tests establish list membership, viewer-policy filtering, totals, deterministic keyset pagination, quote hydration, and query-plan behavior against PostgreSQL.
- Go handler and route tests establish authenticated `/v1/*` contracts, body/query validation, error envelopes, route inventory, access/rate policy, and the no-PDS read boundary.
- Dart model, API-client, and provider tests establish wire decoding, exact paths/query parameters, cursor traversal, retained-items retry, refresh, and active-account fencing.
- Flutter widget and router acceptance tests establish the nonzero-only detail summary, dedicated navigation, comment/reply `View likes` menu behavior, account/profile navigation, quote-card reuse, page states, semantics, and responsive shell behavior.
- Regression tests protect existing like/repost mutations, combined share controls, response-card scope, one-level quote behavior, and surfaces that must not gain interaction summaries.
- Manual checks are limited to final visual polish and real assistive-technology behavior that widget semantics tests cannot fully represent.

Risk level remains **Medium**. The highest-risk areas are viewer-policy count/list alignment, cursor tie handling, response-specific route identity, and avoiding shared `PostCard` regressions.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-002, AC-003, AC-004 | AT-001, AT-002, AT-003, AT-004 | Acceptance | Yes |
| BR-002 | AC-001, AC-003, AC-004 | AT-001, AT-003, AT-004, IT-002, IT-003 | Acceptance / Integration | Yes |
| BR-003 | AC-006, AC-007, AC-008 | AT-003, AT-004, AT-006 | Acceptance | Yes |
| BR-004 | AC-024 | AT-005 | Acceptance | Yes |
| FR-001 | AC-001, AC-005 | AT-001, AT-008, UT-001 | Acceptance / Unit | Yes |
| FR-002 | AC-002, AC-009 | AT-002, AT-009, IT-007 | Acceptance / Integration | Yes |
| FR-003 | AC-003, AC-010 | AT-003, IT-001, IT-010 | Acceptance / Integration | Yes |
| FR-004 | AC-003, AC-010, AC-011 | AT-003, IT-002, IT-010 | Acceptance / Integration | Yes |
| FR-005 | AC-003, AC-006, AC-012 | AT-003, UT-007, IT-004 | Acceptance / Unit / Integration | Yes |
| FR-006 | AC-004, AC-007, AC-010 | AT-004, IT-003, IT-010 | Acceptance / Integration | Yes |
| FR-007 | AC-008, AC-013 | AT-006, UT-004 | Acceptance / Unit | Yes |
| FR-008 | AC-014, AC-015, AC-024 | AT-005, IT-005, IT-007 | Acceptance / Integration | Yes |
| FR-009 | AC-015, AC-016 | IT-005, IT-006, IT-011 | Integration | Yes |
| FR-010 | AC-011, AC-012, AC-017 | IT-002, IT-004, IT-012 | Integration | Yes |
| FR-011 | AC-005 | AT-008 | Acceptance | Yes |
| FR-012 | AC-018 | AT-008, REG-001, REG-002 | Acceptance / Regression | Yes |
| FR-013 | AC-024 | AT-005, UT-006 | Acceptance / Unit | Yes |
| NFR-001 | AC-019 | IT-008, IT-013 | Integration | Yes |
| NFR-002 | AC-010, AC-016 | UT-002, UT-008, IT-010 | Unit / Integration | Yes |
| NFR-003 | AC-020 | AT-007, MAN-001, MAN-002 | Acceptance / Manual | Partial |
| NFR-004 | AC-014, AC-021 | IT-007, REG-004 | Integration / Regression | Yes |
| NFR-005 | AC-022 | IT-014 | Integration | Yes |
| NFR-006 | AC-023 | AT-001 through AT-009, UT-001 through UT-008, IT-001 through IT-014, REG-001 through REG-006 | All automated levels | Yes |
| RULE-001 | AC-003, AC-004, AC-011 | IT-001, IT-002, IT-003 | Integration | Yes |
| RULE-002 | AC-004, AC-011 | IT-002, IT-003, REG-003 | Integration / Regression | Yes |
| RULE-003 | AC-003, AC-004 | IT-001, IT-002, IT-003 | Integration | Yes |
| RULE-004 | AC-001, AC-024 | AT-001, AT-005, UT-001, UT-006 | Acceptance / Unit | Yes |
| RULE-005 | AC-024 | AT-005, REG-002 | Acceptance / Regression | Yes |

## 3. Acceptance Scenarios

### AT-001: Show Only Nonzero Detail Interaction Links
Requirement IDs: BR-001, BR-002, FR-001, RULE-004

Acceptance Criteria: AC-001

Priority: Must

Level: Acceptance

Automation Target: `app/test/feed/pages/post_thread_page_test.dart`

```gherkin
Feature: Post detail interaction summary
  Scenario Outline: Only nonzero interaction types are shown
    Given an available top-level post with <likes> likes, <reposts> reposts, and <quotes> quotes
    When the post detail page renders
    Then the summary appears immediately below the top-level post card when any count is nonzero
    And the summary shows an exact localized link for each count greater than zero
    And visible links remain in Likes, Reposts, Quotes order and wrap independently without punctuation separators
    And the summary does not show a label or link for each zero count
    And the card action row remains separate from the summary

    Examples:
      | likes | reposts | quotes |
      | 3     | 2       | 1      |
      | 3     | 0       | 0      |
      | 0     | 2       | 0      |
      | 0     | 0       | 1      |

  Scenario: Omit the summary when every count is zero
    Given an available top-level post with zero likes, zero reposts, and zero quotes
    When the post detail page renders
    Then no interaction summary is rendered
    And no text equivalent to "0 likes", "0 reposts", or "0 quotes" is present
```

### AT-002: Navigate From Each Detail Count
Requirement IDs: BR-001, FR-002

Acceptance Criteria: AC-002

Priority: Must

Level: Acceptance

Automation Target: `app/test/feed/pages/post_thread_page_test.dart`

```gherkin
Feature: Interaction summary navigation
  Scenario Outline: A visible count opens its dedicated route
    Given a top-level post at DID "did:plc:alice" and record key "root"
    And its <interaction> count is greater than zero
    When the viewer activates the <interaction> summary link
    Then the router location is <location>
    And no repost, quote, like, or unlike mutation is triggered

    Examples:
      | interaction | location                                      |
      | Likes       | /posts/did:plc:alice/root/likes                |
      | Reposts     | /posts/did:plc:alice/root/reposts              |
      | Quotes      | /posts/did:plc:alice/root/quotes               |
```

### AT-003: Browse Liker And Reposter Accounts
Requirement IDs: BR-001, BR-002, BR-003, FR-003, FR-004, FR-005

Acceptance Criteria: AC-003, AC-006

Priority: Must

Level: Acceptance

Automation Target: `app/test/feed/pages/post_interaction_accounts_page_test.dart`

```gherkin
Feature: Account interaction lists
  Scenario Outline: Render hydrated accounts in server order
    Given the <interaction> repository returns Dana followed by Carol with total count 2
    When the <page> destination loads
    Then Dana appears before Carol
    And each row shows its existing display-name and handle presentation
    And the page displays the authoritative total count

    Examples:
      | interaction | page    |
      | like        | Likes   |
      | repost      | Reposts |

  Scenario: Open an account profile
    Given Dana appears in an interaction account list
    When the viewer selects Dana
    Then Dana's existing compact profile presentation opens
```

### AT-004: Browse Quote Posts
Requirement IDs: BR-001, BR-002, BR-003, FR-006

Acceptance Criteria: AC-004, AC-007

Priority: Must

Level: Acceptance

Automation Target: `app/test/feed/pages/post_quotes_page_test.dart`

```gherkin
Feature: Quote-post list
  Scenario: Render each quote as a normal post card
    Given the quote repository returns two quote posts from Dana followed by one from Carol
    When the Quotes destination loads
    Then all three posts render separately in repository order
    And both of Dana's posts remain present
    And the app-bar title is the plain localized "Quotes" title without a total
    And each item uses normal post-card author, quote-preview, action, and post navigation behavior
```

### AT-005: View Comment And Reply Likers From More Actions
Requirement IDs: BR-004, FR-008, FR-013, RULE-004, RULE-005

Acceptance Criteria: AC-024

Priority: Must

Level: Acceptance

Automation Target: `app/test/feed/pages/post_thread_page_test.dart`

```gherkin
Feature: Response liker navigation
  Scenario Outline: A liked response exposes its own liker list
    Given the thread contains a <responseType> by "did:plc:bob" with record key <rkey> and 2 likes
    When the viewer opens that response's more-actions menu
    Then a localized "View likes" action is shown
    And it is first in a separate non-destructive menu group
    When the viewer activates "View likes"
    Then the router opens <location>
    And the root post's identity is not used

    Examples:
      | responseType | rkey   | location                                    |
      | comment      | c1     | /posts/did:plc:bob/c1/likes                |
      | nested reply | r1     | /posts/did:plc:bob/r1/likes                |

  Scenario: An unliked response has no liker action
    Given a comment has zero likes
    When the viewer opens that comment's more-actions menu
    Then "View likes" is absent
    And no inline interaction summary is present
    And no repost-list or quote-list action is present
```

### AT-006: Handle Loading Empty And Retry States
Requirement IDs: BR-003, FR-007

Acceptance Criteria: AC-008, AC-013

Priority: Must

Level: Acceptance

Automation Target: `app/test/feed/pages/post_interaction_accounts_page_test.dart`, `app/test/feed/pages/post_quotes_page_test.dart`

```gherkin
Feature: Interaction destination states
  Scenario Outline: Initial loading and empty results
    Given the <page> first-page request is pending
    When the destination renders
    Then the established loading indicator is shown
    And the app-bar title does not present a pending count as zero
    When the request completes with no items
    Then the localized <page> empty state is shown

    Examples:
      | page    |
      | Likes   |
      | Reposts |
      | Quotes  |

  Scenario: Initial failure can be retried
    Given the first-page request fails once and then succeeds
    When the viewer activates Retry
    Then a fresh first-page request is made
    And the successful items replace the error state

  Scenario: An unavailable target is not retryable
    Given the first-page request returns "post_not_found"
    When the destination renders
    Then a localized "Post unavailable" state and Back action are shown
    And Retry is absent

  Scenario: Incremental failure retains prior items
    Given the first page contains items and a continuation cursor
    And the continuation request fails once
    When the failure is shown
    Then first-page items remain visible
    And Retry is available
    When the viewer activates Retry
    Then the same cursor is requested
    And successful new items append exactly once
```

### AT-007: Support Accessible Responsive Interaction Details
Requirement IDs: NFR-003

Acceptance Criteria: AC-020

Priority: Must

Level: Acceptance

Automation Target: `app/test/feed/pages/post_interaction_accessibility_test.dart`

```gherkin
Feature: Accessible interaction details
  Scenario: Summary and menu actions expose meaningful semantics
    Given a post has 12 likes, 3 reposts, and 2 quotes
    And a comment has 4 likes
    When semantics are enabled
    Then each summary link announces its exact value, interaction type, and button/link action
    And the comment's "View likes" menu item is focusable and clearly labelled
    And "View likes" is first in a separate non-destructive menu group
    And no meaning depends only on color or an icon

  Scenario Outline: Content remains usable at supported layouts and text scales
    Given the interaction destination is rendered at <size> with text scale <scale>
    When a full page and a pagination state are displayed
    Then count links, titles, account rows, post cards, status feedback, and retry controls do not clip or overlap
    And the authenticated shell uses its expected compact or large presentation

    Examples:
      | size      | scale |
      | 390x844   | 1.0   |
      | 390x844   | 2.0   |
      | 1200x800  | 1.0   |
      | 1200x800  | 2.0   |
```

### AT-008: Keep Summary Counts And Existing Mutations Consistent
Requirement IDs: FR-001, FR-011, FR-012

Acceptance Criteria: AC-005, AC-018

Priority: Must

Level: Acceptance

Automation Target: `app/test/feed/pages/post_thread_page_test.dart`, `app/test/feed/widgets/post_card_test.dart`

```gherkin
Feature: Detail summary and mutation controls
  Scenario Outline: A completed mutation updates the model-backed summary
    Given the top-level detail model has <before> <interaction>
    When the existing <action> mutation succeeds with a post model containing <after> <interaction>
    Then the summary reflects <after>
    And no independent interaction-summary counter is retained

    Examples:
      | interaction | action   | before | after |
      | likes       | like     | 0      | 1     |
      | likes       | unlike   | 1      | 0     |
      | reposts     | repost   | 0      | 1     |
      | reposts     | unrepost | 1      | 0     |

  Scenario: Existing controls keep their behavior
    Given a top-level detail card has likes, reposts, and quotes
    When the viewer activates the like control
    Then it performs the like/unlike mutation rather than list navigation
    When the viewer activates the combined share control
    Then the repost/quote choice menu opens rather than an interaction list
```

### AT-009: Support Direct Routes Back Navigation And Authentication
Requirement IDs: FR-002, NFR-004

Acceptance Criteria: AC-009, AC-021

Priority: Must

Level: Acceptance

Automation Target: `app/test/router/post_interaction_routes_test.dart`, `app/test/router/router_redirect_test.dart`

```gherkin
Feature: Interaction detail routes
  Scenario Outline: Authenticated direct navigation builds the correct page
    Given an authenticated onboarded member
    When the app starts at <location>
    Then the <page> destination is built with DID "did:plc:alice" and record key "root"
    And no in-memory route extra is required
    And large and compact shells follow existing detail-route presentation

    Examples:
      | location                                      | page    |
      | /posts/did:plc:alice/root/likes               | Likes   |
      | /posts/did:plc:alice/root/reposts             | Reposts |
      | /posts/did:plc:alice/root/quotes              | Quotes  |

  Scenario: Back returns to the originating post detail
    Given the viewer pushed Likes from a post detail
    When the viewer navigates back
    Then the originating post detail is visible with its existing thread state

  Scenario Outline: Signed-out direct navigation is protected
    Given the viewer is signed out
    When the app starts at <location>
    Then the welcome page is shown

    Examples:
      | location                                      |
      | /posts/did:plc:alice/root/likes               |
      | /posts/did:plc:alice/root/reposts             |
      | /posts/did:plc:alice/root/quotes              |
```

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-001, RULE-004 | AC-001 | Classify visible detail-summary entries without compacting or inventing zero values. | Count triples `(0,0,0)`, `(3,0,0)`, `(0,2,1)`, `(3,2,1)`. | Empty, Likes only, Reposts+Quotes, or all three respectively; each label uses the exact integer. | `app/test/feed/widgets/post_interaction_summary_test.dart` |
| UT-002 | NFR-002 | AC-016 | Decode and validate opaque interaction cursors. | Valid likes cursor; malformed encoding; valid cursor with wrong type; valid cursor with wrong target. | Valid cursor yields ordering tuple; every malformed/mismatched cursor returns the invalid-cursor classification. | `appview/internal/api/post_interaction_cursor_test.go` |
| UT-003 | RULE-001, RULE-002, RULE-003 | AC-003, AC-004, AC-011 | Classify indexed records by interaction-list membership. | Active/deleted likes, active/deleted reposts, quote posts, same actor with repost+quotes, repeated quote author. | Only active records contribute; repost and quote membership stays independent; repeated quotes remain separate. | `appview/internal/api/post_interaction_list_test.go` |
| UT-004 | FR-007 | AC-008, AC-013 | Provider state transitions preserve loaded items and cursor ownership. | Initial pending/success/empty/transient error/`post_not_found`; load-more success/error/retry; refresh while continuation is in flight. | Pending account titles omit a count; target-unavailable state has Back but no Retry; transient retry uses the correct cursor; stale continuation completion cannot overwrite refresh; loaded items survive incremental failure. | `app/test/feed/providers/post_interaction_lists_provider_test.dart` |
| UT-005 | FR-008, FR-009 | AC-014, AC-015, AC-016 | API client builds exact endpoint paths and query parameters and maps errors. | Three list methods with DID/rkey, absent/present cursor, absent/present limit, 400/404 envelope. | Correct URL encoding and GET method; optional query omission; account/post page decode; mapped `invalid_cursor` and `post_not_found`. | `app/test/feed/data/post_api_client_test.dart` |
| UT-006 | FR-013, RULE-004, RULE-005 | AC-024 | Determine response more-actions liker entry visibility, grouping, and route identity. | Comment/reply with 0 or positive likes and distinct root/response DID+rkey. | Action omitted at zero; shown first in a separate non-destructive group when positive; callback carries the response's own DID/rkey; no repost/quote list entries. | `app/test/feed/widgets/post_card_test.dart` |
| UT-007 | FR-005, FR-008 | AC-006, AC-015 | Decode account and quote list response shapes. | Account page with `items`, `totalCount`, optional/absent cursor; post page with optional/absent cursor and no pin metadata. | Existing `ProfileAccountSummary` and `Post` models decode all required fields; absent cursor remains null; no interaction-record model is required. | `app/test/profile/models/profile_account_page_test.dart`, `app/test/feed/models/post_page_test.dart` |
| UT-008 | NFR-002 | AC-010, AC-016 | Compare deterministic interaction ordering tuples. | Equal and unequal `created_at` values with unique URIs. | Rows and seek bounds use `(created_at DESC, uri DESC)` exactly. | `appview/internal/api/post_interaction_cursor_test.go` |

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | FR-003, RULE-001, RULE-003 | AC-003, AC-010 | List active liker accounts for top-level posts, comments, and nested replies. | Insert targets plus active/deleted likes, repeated actor history, and tied timestamps; insert hydrated profiles. | Call the store list method for each target across pages. | Only one active entry per eligible account/subject; exact `(created_at DESC, uri DESC)` order; exact total; deterministic complete traversal. | `appview/internal/api/post_interaction_store_test.go` |
| IT-002 | BR-002, FR-004, FR-010, RULE-001, RULE-002, RULE-003 | AC-003, AC-010, AC-011 | List straight reposter accounts without treating quotes as reposts. | Insert active/deleted reposts, quote-only author, repost-only author, and actor who did both. | Traverse Reposts pages. | Active reposters appear once in `(created_at DESC, uri DESC)` order; quote-only author is absent; dual actor appears once; total is correct. | `appview/internal/api/post_interaction_store_test.go` |
| IT-003 | BR-002, FR-006, RULE-001, RULE-002, RULE-003 | AC-004, AC-010, AC-011 | List and hydrate quote posts independently. | Insert multiple visible quotes from one author, another quote, a deleted/hidden quote, and nested quote preview state. | Traverse Quotes pages. | Every eligible quote post appears separately in `(created_at DESC, uri DESC)` order; ineligible quotes are absent; normal bounded one-level post hydration is returned. | `appview/internal/api/post_interaction_store_test.go` |
| IT-004 | FR-005, FR-010 | AC-012, AC-017 | Apply account/post visibility policy consistently to items and totals. | Create current, former/non-member, terminal, muted, viewer-blocked, actor-author-blocked, warned, hide/takedown, and language-ineligible actors/posts for two viewers. | Request each list and corresponding post counts as each viewer. | Muted account rows and revealable muted quotes remain included; former/non-member, terminal, blocked, hide/takedown, and language-ineligible quote records are excluded; warning-only records remain; totals/detail counts match unchanged eligible data. | `appview/internal/api/post_interaction_policy_test.go` |
| IT-005 | FR-008, FR-009 | AC-014, AC-015 | Verify successful endpoint wire contracts. | Fake/store data for account and post pages plus authenticated DID context. | GET all endpoints with default and explicit limits/cursors. | 200 camelCase bare bodies; likes/reposts include `items`, `totalCount`, optional cursor; quotes include `items`, optional cursor and no pin metadata; final cursor key is omitted. | `appview/internal/api/post_interaction_handler_test.go` |
| IT-006 | FR-009, NFR-002 | AC-015, AC-016 | Verify identifier, target, bounded-limit, and cursor failures. | Handler with resolvable/missing/hidden/muted targets, response targets, and valid target/type-bound cursors. | Send malformed DID/rkey, malformed/nonpositive/oversized limits, malformed cursor, wrong-target cursor, wrong-type cursor, and response-target Reposts/Quotes requests. | Missing/hidden and response-target Reposts/Quotes return `404 post_not_found`; muted targets remain eligible; malformed/nonpositive limits use the established default and oversized limits use the established cap; cursor misuse is `400 invalid_cursor`; no items leak. | `appview/internal/api/post_interaction_handler_test.go` |
| IT-007 | FR-002, FR-008, NFR-004 | AC-009, AC-014, AC-021 | Register and protect all three routes. | Build production route mux with fakes/recorders. | Exercise each route with missing auth, missing device ID, valid current-member auth, and unexpected body. | Route inventory includes each GET; access class is current member, rate class is read, body kind is no-body; middleware rejects invalid requests; successful reads make no PDS effect call. | `appview/internal/routes/routes_test.go`, `inventory_test.go`, `architecture_test.go` |
| IT-008 | NFR-001 | AC-019 | Verify subject/quote indexes support list scans. | Populate enough likes, reposts, posts, and profiles; disable sequential scans where existing query-plan convention requires it. | `EXPLAIN` each list query with a target and ordering bound. | Plans use the active subject or quote URI indexes and indexed joins; no full interaction/post scan is required. | `appview/internal/api/post_interaction_query_plan_test.go` |
| IT-009 | FR-008 | AC-014, AC-015 | Exercise Flutter API client against mocked HTTP contracts. | Dio adapter returns account and post pages with first/final cursors. | Invoke all repository methods for top-level and response likes. | Correct method/path/query and existing model decode; response likes use the selected response identity. | `app/test/feed/data/post_api_client_test.dart`, `post_repository_test.dart` |
| IT-010 | FR-003, FR-004, FR-006, NFR-002 | AC-010 | Traverse all endpoints with timestamp ties. | Create more than two pages per type with tied times and deterministic unique keys. | Follow every opaque cursor at several valid limits. | Limits are honored, order is stable, all eligible items appear once, and cursor is omitted only at exhaustion. | `appview/internal/api/post_interaction_pagination_test.go` |
| IT-011 | FR-009 | AC-016 | Hide or remove the target without exposing its interactions. | Create a target with interactions, then delete/hide it for the viewer. | Request all three endpoints. | Every endpoint returns indistinguishable `404 post_not_found` and no item/total data. | `appview/internal/api/post_interaction_policy_test.go` |
| IT-012 | FR-010 | AC-017 | Keep detail counts and first-page totals aligned. | Fixed unchanged dataset with active/inactive and viewer-filtered interactions. | Fetch target post and each first interaction page in the same viewer context. | Like/repost account totals and eligible quote item count match the corresponding viewer-specific detail values. | `appview/internal/api/post_interaction_policy_test.go` |
| IT-013 | NFR-001 | AC-019 | Bound account and post hydration work. | Full pages containing many actors/posts and a counting resolver/query observer. | List each interaction type once. | Hydration uses bounded batch calls independent of item count; no per-item directory/profile/post query pattern occurs. | `appview/internal/api/post_interaction_hydration_test.go` |
| IT-014 | NFR-005 | AC-022 | Keep observability bounded and identity-free. | Instrument route and DB observers for success, validation failure, not found, and internal failure. | Exercise all three route templates. | Existing route/status/latency and bounded DB operation labels are emitted; DIDs, handles, post URIs, cursors, and items are absent from labels/attributes/log fields. | `appview/internal/api/post_interaction_observability_test.go` |

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Test |
|---|---|---|---|
| REG-001 | Like button mutates like state; combined share button opens straight-repost/quote choices. | FR-012 | Extend `app/test/feed/widgets/post_card_test.dart` and `toggle_post_interactions_provider_test.dart` to assert list callbacks/routes never replace existing mutation callbacks. |
| REG-002 | Only top-level detail gets the interaction summary; comments/replies get only the approved nonzero `View likes` menu action. | FR-012, FR-013, RULE-005 | Render feed, profile, search, project, notification, top-level detail, comment, and nested reply cards; assert summary/menu scope exactly. |
| REG-003 | Straight repost and quote semantics remain independent; repeated quote posts remain normal authored posts. | RULE-002, RULE-003 | Preserve existing repost/quote tests and add assertions that interaction lists do not alter `viewerHasReposted`, unrepost, profile, timeline, or quote creation behavior. |
| REG-004 | Flutter reads through AppView and read endpoints never create PDS effects. | NFR-004 | Route test injects a panic/fail recorder as the PDS executor and confirms all three successful GETs complete without invoking it; client tests assert only `/v1/` URLs. |
| REG-005 | Existing one-level quote preview and moderation placeholders remain bounded. | FR-006, FR-010 | Render/list a quote whose outer post quotes another quote and assert no recursive hydration beyond the established post contract. |
| REG-006 | Existing thread sorting, replies, pin actions, report/delete actions, and responsive shell remain usable. | FR-012, NFR-003 | Run/extend `post_thread_page_test.dart`, `post_comment_section_page_test.dart`, `post_card_test.dart`, and router tests after adding the summary/menu callback. |

## 7. Test Data

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Detail count visibility | Root posts with `(likeCount,repostCount,quoteCount)` of `(0,0,0)`, each positive singleton, mixed zero/nonzero, and all positive. | AT-001, AT-002, AT-008, UT-001 |
| TD-002 | Interaction actors | Alice target author; Dana/Carol visible members; muted, viewer-blocked, actor-author-blocked, warning-only, hide/takedown, terminal, and former/non-member actors with complete profile summaries/customisation. | AT-003, IT-001, IT-002, IT-004 |
| TD-003 | Response identity | Root `at://did:plc:alice/.../root`, comment `at://did:plc:bob/.../c1`, nested reply `at://did:plc:carol/.../r1`, with zero and positive like variants. | AT-005, UT-006, IT-001, IT-009 |
| TD-004 | Interaction lifecycle | Active and deleted like/repost records for the same subject; one active record per actor/subject; created/indexed timestamps including exact ties and unique URIs. | UT-003, UT-008, IT-001, IT-002, IT-010 |
| TD-005 | Quote semantics | Two visible quote posts by Dana, one by Carol, deleted/hidden and language-ineligible quotes, a revealable muted-author quote, one quote with unavailable preview, and one quote-of-quote. | AT-004, IT-003, IT-004, REG-003, REG-005 |
| TD-006 | Pagination | At least 2× maximum test page size plus one item for each type, with records before/at/after cursor bounds. | IT-010, UT-004, AT-006 |
| TD-007 | Cursor misuse | Valid cursors bound to likes/root, reposts/root, quotes/root, and likes/comment plus malformed/tampered strings. | UT-002, IT-006, IT-010 |
| TD-008 | API errors | Missing, deleted, viewer-hidden, and malformed target identifiers; missing auth/device headers; unexpected GET body; invalid limits. | IT-006, IT-007, IT-011 |
| TD-009 | Responsive semantics | Long localized labels/display names, large total values, 390×844 and 1200×800 viewports, text scales 1.0 and 2.0, semantics enabled. | AT-007, MAN-001, MAN-002 |
| TD-010 | Concurrent list state | First page with cursor, pending continuation, failed continuation, refreshed first page, stale completion, and active-account switch. | AT-006, UT-004 |

## 8. Manual Checks

| ID | Requirement IDs | Check | Steps | Expected Result |
|---|---|---|---|---|
| MAN-001 | NFR-003 | Final responsive visual smoke | Run the app with seeded positive/mixed/zero counts. Inspect top-level detail, each destination, and comment/reply menus on a narrow phone and large desktop/tablet at normal and enlarged system text. | Summary spacing is clearly associated with the root post; zero links leave no awkward gaps; account/post rows, menus, empty/error/loading states, rail/full-screen shell, and pagination remain readable without overlap. |
| MAN-002 | NFR-003 | Real screen-reader and keyboard smoke | On one supported mobile screen reader and desktop keyboard path, focus each visible summary link, comment/reply `View likes`, list row, Retry, and Back. | Spoken labels include value/type/action where relevant; focus order is logical; activation opens the intended response/root route; hidden zero actions are not focusable; returning restores a usable thread context. |

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | Widget semantics cannot fully validate VoiceOver/TalkBack phrasing, focus restoration, or platform menu behavior. | NFR-003 | Flutter semantics tests model the tree but not every platform assistive-technology behavior. | Complete MAN-002 before merge or handoff; record platform/device used. |
| GAP-002 | Mutable keyset pagination cannot promise a transactionally frozen snapshot across requests. | NFR-002 | Requirements explicitly allow concurrent creates/deletes and require deterministic behavior only for unchanged data. | Test unchanged traversal exhaustively and test changed-data resilience without asserting snapshot semantics. |
| GAP-003 | Query planner choices can vary with PostgreSQL statistics and tiny fixtures. | NFR-001 | A plan test may choose a sequential scan for legitimately small tables. | Follow existing query-plan test conventions with representative rows and planner settings; assert usable indexes and bounded query shape rather than brittle complete plan text. |
| GAP-004 | Count/list equality can change naturally between independent post-detail and list requests. | FR-010 | No cross-request snapshot token is in scope. | Assert equality on unchanged fixtures and policy predicates; acceptance/UI tests tolerate refreshed empty or changed totals after concurrent activity. |

Blocking gaps: None.

## 10. Out Of Scope

- Tests for changing like, unlike, repost, unrepost, or quote-post creation semantics beyond regression protection.
- Repost or quote list behavior for comments/replies.
- Inline interaction summaries on feed, profile, search, project, notification, comment, or reply cards.
- Tabs, filtering, search, sorting controls, bulk actions, follow actions, moderation actions, or product analytics on interaction pages.
- PDS records, lexicons, migrations, background jobs, or direct Flutter-to-PDS reads.
- Frozen-snapshot pagination under concurrent writes.
- Performance/load testing beyond query plans, bounded hydration-call assertions, and normal integration-suite timing.

## 11. Handoff To Document Review

- Requirements file: `01-requirements.md`
- Test specification: `02-acceptance-tests.md`
- Next review artifact: `03-document-review.md`
- External Plannotator review, if the user initiates it outside this skill: `docs/changes/2026-09-06-post-interaction-details/`
- Recommended first failing test for implementation: `IT-001` in `appview/internal/api/post_interaction_store_test.go`, starting with active liker accounts for a top-level post and a comment/reply, newest-first ordering, exact `totalCount`, and final cursor omission.
- Suggested test order for implementation:
  - `IT-001`, `IT-002`, `IT-003`: establish store contracts for likes, reposts, and quotes.
  - `UT-002`, `UT-008`, `IT-010`: establish cursor encoding and deterministic traversal.
  - `IT-004`, `IT-011`, `IT-012`, `IT-013`: establish visibility, target protection, count alignment, and bounded hydration.
  - `IT-005`, `IT-006`, `IT-007`, `IT-008`, `IT-014`: establish HTTP, route, query-plan, and observability contracts.
  - `UT-005`, `UT-007`, `IT-009`: establish Flutter model/repository/API plumbing.
  - `UT-004`, `AT-003`, `AT-004`, `AT-006`: establish provider and list-page behavior.
  - `UT-001`, `AT-001`, `AT-002`, `AT-008`: establish detail summary and mutation separation.
  - `UT-006`, `AT-005`: establish comment/reply more-actions behavior and response-specific routing.
  - `AT-009`, `AT-007`, REG-001 through REG-006: finish routing, accessibility, responsive, and regression coverage.
- Commands discovered:
  - Focused AppView tests: `cd appview && go test ./internal/api ./internal/routes`
  - Full AppView development suite: `just test` with compose services running.
  - Release-equivalent AppView gate: `just appview-check`.
  - Go format/vet: `just fmt`.
  - Focused Flutter tests: `cd app && flutter test test/feed test/router`
  - Full Flutter tests: `just app-test`.
  - Flutter analysis: `just app-analyze`.
- Blocking gaps: None.
