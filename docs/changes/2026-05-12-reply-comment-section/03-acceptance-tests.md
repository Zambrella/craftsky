# Acceptance Test Specification: Reply Comment Section

## 1. Test Strategy

Use a test-first path that starts at the AppView read contract, then adds Flutter data/model/provider coverage, then widget-level user workflows. Backend tests should pin the comment-section API shape, comment grouping/sorting/pagination, focus status and promotion, required comment placement metadata, reply loaded-state metadata, flattened reply metadata, bounded focused reply slices, reply pagination, and `/thread` removal. Flutter tests should verify route parsing, state transitions, de-duplication, expansion/collapse controls, new comment/reply scroll/focus behavior, and the two-level visual cap.

Risk level remains **Medium** because this is a coordinated API and UI behavior change. Most coverage should be automated. Manual checks are limited to real scroll feel/deep-link launch behavior that is hard to fully assert in unit/widget tests.

Terminology note: top-level replies are **comments**. Replies under a comment are **replies**. This is a hard product/API/client/docs/tests convention. Backend storage may retain `reply_*` names where they refer to atproto reply refs, but API response fields, Flutter read models/providers/widgets, UI labels, and implementation names should prefer `comment` for direct replies to a root post and `reply` for replies under comments.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-002, AC-003 | AT-001, AT-002, IT-002, IT-003, UT-006 | Acceptance / Integration / Unit | Yes |
| BR-002 | AC-004, AC-005, AC-006 | AT-003, AT-004, AT-005, IT-001, IT-004, UT-004 | Acceptance / Integration / Unit | Yes |
| BR-003 | AC-007 | AT-006, IT-005, UT-001 | Acceptance / Integration / Unit | Yes |
| BR-004 | AC-011, AC-012 | AT-007, AT-008, UT-010, UT-011 | Acceptance / Unit | Yes |
| FR-001 | AC-013 | REG-001, REG-002, IT-009 | Regression / Integration | Yes |
| FR-002 | AC-001, AC-004, AC-008 | IT-001, IT-002, UT-007 | Integration / Unit | Yes |
| FR-003 | AC-001, AC-002, AC-003 | AT-001, IT-002, UT-006 | Acceptance / Integration / Unit | Yes |
| FR-004 | AC-002, AC-003, AC-014 | AT-002, AT-009, IT-003, IT-006 | Acceptance / Integration | Yes |
| FR-005 | AC-004 | AT-003, UT-004 | Acceptance / Unit | Yes |
| FR-006 | AC-005, AC-009 | AT-004, IT-004, UT-008 | Acceptance / Integration / Unit | Yes |
| FR-007 | AC-007, AC-010, AC-020 | AT-006, IT-005, UT-001, UT-002 | Acceptance / Integration / Unit | Yes |
| FR-008 | AC-007 | AT-006, IT-005, UT-001 | Acceptance / Integration / Unit | Yes |
| FR-009 | AC-006 | AT-005, UT-005 | Acceptance / Unit | Yes |
| FR-010 | AC-006, AC-010, AC-026 | AT-005, IT-007, UT-002, IT-016 | Acceptance / Integration / Unit | Yes |
| FR-011 | AC-009, AC-010 | AT-005, IT-007, UT-009 | Acceptance / Integration / Unit | Yes |
| FR-012 | AC-006 | AT-005, UT-005 | Acceptance / Unit | Yes |
| FR-013 | AC-014 | AT-009, UT-003 | Acceptance / Unit | Yes |
| FR-014 | AC-012, AC-014, AC-015 | AT-008, AT-009, IT-008, UT-003, UT-012 | Acceptance / Integration / Unit | Yes |
| FR-015 | AC-015 | AT-008, UT-012 | Acceptance / Unit | Yes |
| FR-016 | AC-011, AC-020 | AT-007, UT-010, UT-011 | Acceptance / Unit | Yes |
| FR-017 | AC-012 | AT-008, UT-010 | Acceptance / Unit | Yes |
| FR-018 | AC-016 | AT-010, UT-013 | Acceptance / Unit | Yes |
| NFR-001 | AC-017 | IT-010, REG-003 | Integration / Regression | Yes |
| NFR-002 | AC-005, AC-006, AC-009, AC-021 | IT-004, IT-007, IT-011, UT-008, UT-009 | Integration / Unit | Yes |
| NFR-003 | AC-018 | UT-011, IT-012 | Unit / Integration | Yes |
| RULE-001 | AC-015, AC-019 | IT-008, REG-004 | Integration / Regression | Yes |
| RULE-002 | AC-004, AC-007 | IT-001, IT-005, UT-004 | Integration / Unit | Yes |
| RULE-003 | AC-006, AC-010, AC-012, AC-014, AC-026 | AT-005, AT-008, AT-009, IT-007, UT-003 | Acceptance / Integration / Unit | Yes |
| RULE-004 | AC-010 | IT-007, UT-002 | Integration / Unit | Yes |
| RULE-005 | AC-007 | IT-005, UT-001 | Integration / Unit | Yes |
| RULE-006 | AC-007, AC-011, AC-020 | AT-006, AT-007, IT-005, UT-001, UT-011 | Acceptance / Integration / Unit | Yes |
| RULE-007 | AC-003, AC-021 | AT-002, IT-006, IT-011 | Acceptance / Integration | Yes |
| RULE-008 | AC-024 | AT-011, UT-014 | Acceptance / Unit | Yes |
| FR-019 | AC-025 | IT-013 | Integration | Yes |
| FR-020 | AC-025 | IT-013 | Integration | Yes |
| FR-021 | AC-001, AC-003, AC-025 | IT-013, UT-006 | Integration / Unit | Yes |
| FR-022 | AC-022, AC-024 | IT-014, UT-015 | Integration / Unit | Yes |
| FR-023 | AC-023 | IT-015, UT-016 | Integration / Unit | Yes |
| FR-024 | AC-026 | IT-016, UT-017 | Integration / Unit | Yes |
| NFR-004 | AC-018, AC-024 | IT-012, UT-011, UT-014 | Integration / Unit | Yes |

## 3. Acceptance Scenarios

### AT-001: Deep Link Opens Root Comment Section With Focus

Requirement IDs: BR-001, FR-003
Acceptance Criteria: AC-001, AC-003
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/pages/post_comment_section_page_test.dart`

```gherkin
Feature: Reply deep links
  Scenario: Opening a focused comment or reply link
    Given a root post has an indexed comment or reply
    And the app route is /posts/{rootDid}/{rootRkey}?focus={encodedReplyAtUri}
    When the post comment section loads
    Then the root post is displayed as the page context
    And the focused target is identified in the loaded comment state
    And the focused target is scrolled or highlighted into view
```

### AT-002: Focused Reply Branch Is Included Outside Initial Pages

Requirement IDs: BR-001, FR-004, RULE-007
Acceptance Criteria: AC-002, AC-003, AC-021
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/pages/post_comment_section_page_test.dart`

```gherkin
Feature: Reply deep links
  Scenario: Focused nested reply is outside the first page
    Given a root post has more than 10 comments
    And the focused reply belongs to a comment branch outside the first comment page
    And the focused reply is outside the first 10 replies for that branch
    When the focused route is opened
    Then the focused comment branch is promoted to the top without loading all intermediate comment pages
    And a bounded reply slice containing the focused reply is displayed
    And child pagination controls remain available for predictable loading
```

### AT-003: Root Post Initially Shows Top-Level Replies Only

Requirement IDs: BR-002, FR-005, RULE-002
Acceptance Criteria: AC-004
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/pages/post_comment_section_page_test.dart`

```gherkin
Feature: Root post comment section
  Scenario: Opening a post without focus
    Given a root post has comments and replies
    When the user opens /posts/{rootDid}/{rootRkey} without a focus parameter
    Then the root post is visible
    And comments are visible below it
    And replies are not visible until the user expands a comment branch
```

### AT-004: Top-Level Replies Lazy Load On Scroll

Requirement IDs: BR-002, FR-006, NFR-002
Acceptance Criteria: AC-005, AC-009
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/pages/post_comment_section_page_test.dart`

```gherkin
Feature: Top-level reply pagination
  Scenario: Scrolling loads more comments
    Given a root post has more than 10 comments
    And the first 10 replies are displayed
    When the user scrolls near the end of the comment list
    Then the app requests the next page using the current cursor and sort
    And up to 10 additional comments are appended
    And existing replies remain visible without duplication
```

### AT-005: User Expands, Loads More, And Hides Child Replies

Requirement IDs: BR-002, FR-009, FR-010, FR-011, FR-012
Acceptance Criteria: AC-006, AC-009, AC-010
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/pages/post_comment_section_page_test.dart`

```gherkin
Feature: Child reply controls
  Scenario: Managing replies under a comment
    Given a comment has more than 10 replies
    And replies are not loaded
    When the user taps "view replies"
    Then the first 10 visual replies for that comment branch are shown oldest-first
    And deeper descendants are flattened into the same reply list
    And the branch control changes to "hide replies"
    And a "load more" control is shown
    When the user taps "load more"
    Then up to 10 more replies are appended oldest-first
    When the user taps "hide replies"
    Then the reply list collapses without deleting reply data
```

### AT-006: Top-Level Sort Groups Viewer Replies First

Requirement IDs: BR-003, FR-007, FR-008, RULE-005, RULE-006
Acceptance Criteria: AC-007, AC-020
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/pages/post_comment_section_page_test.dart`

```gherkin
Feature: Reply ordering
  Scenario Outline: Selecting comment ordering
    Given the viewer and other users have authored comments
    When the user selects <sort> from the ordering dropdown
    Then viewer-authored comments appear before normal comments
    And replies inside each group use <effectiveOrder>
    And expanded replies remain oldest-first

    Examples:
      | sort    | effectiveOrder |
      | oldest  | oldest-first   |
      | newest  | newest-first   |
      | follows | oldest-first   |
```

### AT-007: New Top-Level Reply Appears In Viewer Group And Scrolls Into View

Requirement IDs: BR-004, FR-016, RULE-006
Acceptance Criteria: AC-011, AC-020
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/pages/post_comment_section_page_test.dart`

```gherkin
Feature: Reply creation feedback
  Scenario: Creating a comment
    Given the user is viewing a root post with any comment sort selected
    When the user creates a comment
    Then the new comment appears in the viewer-authored comment group
    And the selected sort does not change
    And the new reply is scrolled into view
```

### AT-008: Replying To A Reply Preserves Parent And Displays In Second Level

Requirement IDs: BR-004, FR-014, FR-015, FR-017, RULE-003
Acceptance Criteria: AC-012, AC-015
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/pages/post_comment_section_page_test.dart`, `app/test/feed/widgets/post_composer_sheet_test.dart`

```gherkin
Feature: Reply creation feedback
  Scenario: Replying to a reply
    Given a reply under a comment is visible
    When the user taps reply on that reply
    Then the composer includes an @handle mention for the target author
    When the reply is submitted successfully
    Then the create request uses the actual target reply as parent
    And the created reply is displayed in the nearest comment branch's reply list
    And the created reply is scrolled into view
```

### AT-009: Deeper Replies Never Render A Third Visual Level

Requirement IDs: FR-004, FR-013, FR-014, RULE-003
Acceptance Criteria: AC-014
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/pages/post_comment_section_page_test.dart`

```gherkin
Feature: Two-level visual cap
  Scenario: Deep backend reply is displayed from focus
    Given a backend reply chain is deeper than two levels
    When a focused deep reply is rendered in the comment section
    Then the root post is displayed at the root level
    And the nearest comment ancestor is displayed as a comment
    And the focused deep reply is displayed in the reply list
    And no third indentation level is present
```

### AT-010: Comment Section Labels Are Localized

Requirement IDs: FR-018
Acceptance Criteria: AC-016
Priority: Should
Level: Acceptance
Automation Target: `app/test/feed/pages/post_comment_section_page_test.dart`

```gherkin
Feature: Comment section labels
  Scenario: Controls use localized text
    Given the comment section renders reply controls
    When ordering, view replies, load more, hide replies, or focused-reply states are shown
    Then the visible labels come from the app localization mechanism
```

### AT-011: Focus Promotion Precedes Viewer Group And Clears On Sort

Requirement IDs: FR-007, FR-022, RULE-006, RULE-008
Acceptance Criteria: AC-022, AC-024
Priority: Must
Level: Acceptance
Automation Target: `app/test/feed/pages/post_comment_section_page_test.dart`

```gherkin
Feature: Focus promotion
  Scenario: Focused branch ordering and clearing
    Given a focused comment branch is included
    And the viewer has authored other comments
    When the focused route is opened
    Then the focused comment branch is first with placement "focused"
    And viewer-authored comments follow with placement "viewerAuthored"
    And normal comments follow with placement "normal"
    When the user changes the comment sort
    Then focus promotion is cleared
    And viewer-authored comments appear before normal comments under the selected sort
```

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-007, FR-008, RULE-005, RULE-006 | AC-007, AC-020 | Sort comparator groups viewer-authored comments first and maps `follows` to oldest. | Mixed viewer/non-viewer comments with timestamps and sort values. | Viewer group first; `oldest`/`follows` oldest-first; `newest` newest-first inside each group. | `app/test/feed/models/post_comment_section_state_test.dart` or backend pure sort helper test. |
| UT-002 | FR-007, FR-010, FR-011, RULE-004 | AC-010 | Reply ordering ignores comment sort. | Replies with shuffled timestamps under each comment sort. | Replies render oldest-first for every comment sort. | `app/test/feed/models/post_comment_section_state_test.dart` |
| UT-003 | FR-013, FR-014, RULE-003 | AC-014 | Tree-flattening logic maps deeper replies to nearest comment branch. | Root, comment, reply, deeper reply chain. | Output contains only comment and reply visual nodes. | `app/test/feed/models/post_comment_section_state_test.dart` |
| UT-004 | FR-005, RULE-002 | AC-004 | Initial render state excludes reply lists. | Root post response with comments carrying reply counts. | State contains root and comments; branch expansion states are collapsed. | `app/test/feed/providers/post_comment_section_provider_test.dart` |
| UT-005 | FR-009, FR-012 | AC-006 | Branch expansion/collapse state changes controls. | Top-level reply with child count and loaded child page. | Collapsed branch shows view; expanded branch shows hide and optional load more. | `app/test/feed/models/post_comment_section_state_test.dart` |
| UT-006 | FR-003 | AC-001 | Route/parser decodes `focus` query parameter. | `/posts/did:plc:alice/root?focus=<encoded AT-URI>`. | Route passes decoded focus AT-URI to page/provider. | `app/test/router/router_redirect_test.dart` or new route test. |
| UT-007 | FR-002, FR-022, FR-023 | AC-008, AC-022, AC-023 | Client model decodes comment-section response shape. | JSON with `post`, `comments.items`, `placement`, `replies.loaded`, `cursor`, `sort`, focus metadata. | Model fields decode with camelCase keys, required placement, required replies object, and optional cursor/focus fields. | `app/test/feed/models/post_comment_section_test.dart` |
| UT-008 | FR-006, NFR-002 | AC-005, AC-009 | Top-level lazy-load state prevents concurrent duplicate loads and tracks cursor. | State with loading flag, cursor, scroll trigger. | One page request is queued; cursor updates after success. | `app/test/feed/providers/post_comment_section_provider_test.dart` |
| UT-009 | FR-011, NFR-002 | AC-009 | Reply load-more state is per comment branch. | Two expanded comment branches with independent cursors. | Loading one branch does not mutate the other branch cursor/items. | `app/test/feed/providers/post_comment_section_provider_test.dart` |
| UT-010 | FR-017 | AC-012 | New nested reply insertion targets relevant comment branch. | Created reply whose parent is a reply/deeper reply. | Reply appears in nearest comment branch reply list. | `app/test/feed/models/post_comment_section_state_test.dart` |
| UT-011 | FR-016, NFR-003, RULE-006 | AC-011, AC-018, AC-020 | De-duplicates viewer-authored group against paginated pages. | Viewer-authored reply in top group and later page. | Visible list contains one instance. | `app/test/feed/models/post_comment_section_state_test.dart` |
| UT-012 | FR-015 | AC-015 | Composer mention prefill uses target author's handle. | Second-level target post with handle `bobbin.craftsky.social`. | Composer text starts with or includes `@bobbin.craftsky.social` per UX implementation. | `app/test/feed/widgets/post_composer_sheet_test.dart` |
| UT-013 | FR-018 | AC-016 | Localizations expose required strings. | Generated l10n accessors for sort/view/load/hide/focus labels. | Strings are available through `AppLocalizations`. | `app/test/feed/pages/post_comment_section_page_test.dart` or l10n-focused widget tests. |
| UT-014 | RULE-008, FR-007 | AC-024 | Sort change clears focus promotion. | State with focused placement and selected sort change. | Focused placement is removed; viewer-authored grouping remains. | `app/test/feed/models/post_comment_section_state_test.dart` |
| UT-015 | FR-022 | AC-022 | Placement is required and enum-backed. | Comment item JSON missing placement or with unknown placement. | Decode/validation fails; valid placements decode. | `app/test/feed/models/post_comment_section_test.dart` |
| UT-016 | FR-023 | AC-023 | Replies object is required and distinguishes loaded/unloaded. | Comment JSON with loaded false/true, empty/items/cursor variants. | State distinguishes unloaded, loaded empty, loaded with next cursor. | `app/test/feed/models/post_comment_section_test.dart` |
| UT-017 | FR-024 | AC-026 | Flattened reply metadata decodes structurally. | Reply item JSON with `flattened: true` and `replyingTo`; direct reply with `flattened: false`. | Flattened reply exposes parent metadata; direct reply does not require `replyingTo`. | `app/test/feed/models/post_comment_section_test.dart` |

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | FR-002, RULE-002 | AC-004, AC-008 | Comment-section endpoint returns root and comments only. | Seed root, comments, and replies in Postgres fixture. | GET comment-section endpoint without focus. | Response includes root and comment page; no reply items are expanded by default. | `appview/internal/api/post_test.go` handler test + `post_store_test.go`. |
| IT-002 | BR-001, FR-003 | AC-001 | Focus query is accepted as URL-encoded AT-URI. | Seed root and indexed reply. | GET comment-section endpoint with valid focus. | 200 response identifies focused target and root context. | `appview/internal/api/post_test.go`. |
| IT-003 | BR-001, FR-004 | AC-002, AC-003 | Focused comment ancestor outside first page is included and promoted. | Seed >10 comments; focused branch is after first page. | GET first page with focus. | Focused branch appears first with `placement = "focused"` without intermediate comment pages. | `appview/internal/api/post_store_test.go`. |
| IT-004 | FR-006, NFR-002 | AC-005, AC-009 | Comments page in chunks of 10 with opaque cursor. | Seed 12 comments. | Request limit 10, then cursor. | Page 1 has 10 normal-page comments plus allowed promoted extras; page 2 has remaining normal comments; cursor is based on normal ordering. | `appview/internal/api/post_store_test.go`. |
| IT-005 | FR-007, FR-008, RULE-005, RULE-006 | AC-007, AC-020 | Comment sorting and viewer grouping. | Seed viewer-authored and other comments with different timestamps. | Request `sort=oldest`, `sort=newest`, `sort=follows`. | Viewer-authored group first when no focus promotion; sort applies within groups; follows equals oldest. | `appview/internal/api/post_store_test.go`. |
| IT-006 | FR-004, RULE-007 | AC-003, AC-021 | Focused reply outside first reply page uses bounded focused slice. | Seed comment with >10 replies and focus reply after first 10. | GET comment section with focus. | Expanded branch contains target in bounded slice and exposes reply pagination state. | `appview/internal/api/post_store_test.go`. |
| IT-007 | FR-010, FR-011, RULE-003, RULE-004 | AC-006, AC-009, AC-010, AC-026 | Comment branch replies load 10 visual replies at a time oldest-first, flattening deeper descendants. | Seed comment with direct replies and deeper descendants. | GET replies with limit 10, then cursor. | Direct replies and deeper descendants for the comment branch are returned oldest-first; deeper descendants have `flattened = true` and `replyingTo` metadata. | `appview/internal/api/post_store_test.go`, `appview/internal/api/post_test.go`, and API client decode tests. |
| IT-008 | FR-014, RULE-001 | AC-015, AC-019 | Creating reply preserves actual parent and root refs. | Existing create-post handler with reply target parent/root refs. | POST reply to a reply target. | PDS record body contains actual parent ref and root ref; no lexicon change needed. | `appview/internal/api/post_test.go`, existing create reply tests. |
| IT-009 | FR-001 | AC-013 | `/thread` route is removed. | AppView routes registered. | GET `/v1/posts/{did}/{rkey}/thread` with auth/device headers. | Route no longer resolves to thread handler; stale route tests are removed/replaced. | `appview/internal/routes/routes_test.go`. |
| IT-010 | NFR-001 | AC-017 | API contract follows `/v1/` conventions. | Success and error responses from comment endpoints. | Inspect JSON bodies. | camelCase keys, standard error envelope, opaque cursor string semantics. | `appview/internal/api/post_test.go`, routes contract tests. |
| IT-011 | NFR-002, RULE-007 | AC-021 | Focused reply slice is bounded. | Seed many replies before focused target. | Request focus. | Response does not include every earlier reply and remains within documented bound. | `appview/internal/api/post_store_test.go`. |
| IT-012 | NFR-003 | AC-018 | Backend/client model de-duplicates overlaps. | Focused reply also appears in loaded page, or viewer reply appears in page and viewer group. | Decode/merge response into state. | Visible list contains single instance per URI. | `app/test/feed/models/post_comment_section_state_test.dart` and/or backend response test. |
| IT-013 | FR-019, FR-020, FR-021 | AC-025 | Focus status contract. | Malformed focus, valid missing focus, valid mismatched-root focus, and valid included focus. | GET comment-section endpoint with each focus. | Returns `400 invalid_focus`, `200 notFound`, `200 mismatchedRoot`, and `200 included` with `uri`, `kind`, and `commentUri` when required. | `appview/internal/api/post_test.go`. |
| IT-014 | FR-022 | AC-022, AC-024 | Ordered comments array includes placement. | Seed focused, viewer-authored, and normal comments. | GET comment section with focus. | `comments.items` is render order and each item has required placement. | `appview/internal/api/post_test.go`. |
| IT-015 | FR-023 | AC-023 | Comment replies object is always present. | Seed comments with no replies, unloaded replies, and focused loaded replies. | GET comment section. | Every comment item includes `replies.loaded` and `replies.items`; cursor only when more replies exist. | `appview/internal/api/post_test.go`. |
| IT-016 | FR-024 | AC-026 | Flattened reply metadata is returned for deeper replies. | Seed root -> comment -> reply -> deeper focused reply. | GET comment section with focus on deeper reply. | Deeper reply item has `flattened: true` and `replyingTo` with `uri`, `did`, `handle`, optional `displayName`. | `appview/internal/api/post_test.go`. |

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Test |
|---|---|---|---|
| REG-001 | Route registration continues to require auth and device ID for post reply/comment endpoints. | FR-001, NFR-001 | Update `appview/internal/routes/routes_test.go` to assert new comment-section route auth/device behavior and remove old thread auth tests. |
| REG-002 | Flutter no longer calls stale `/thread` API or decodes `PostThread`. | FR-001 | Replace `PostApiClient.getThread`, `postThreadProvider`, and `PostThreadPage` tests with comment-section equivalents; stale test names should fail if code remains. |
| REG-003 | Existing post create/read/delete/like/repost API behavior remains unchanged. | NFR-001 | Keep existing `post_api_client_test.dart`, `post_test.go`, and interaction tests passing outside intentional thread/comment changes. |
| REG-004 | Reply records remain posts with root/parent refs; lexicon-derived behavior remains unchanged. | RULE-001 | Keep existing `post_request_test.go`, `post_response_test.go`, and indexer reply-column tests passing; verify no lexicon file changes are needed. |
| REG-005 | Existing replies endpoint pagination oldest-first remains valid for comment-branch reply loading. | FR-010, FR-011, RULE-003, RULE-004 | Preserve/update `TestPostStore_ListCommentBranchReplies_PaginatesBranchOldestFirst` and handler/client reply-loading tests with limit 10 expectations and flattened descendant metadata. |

## 7. Test Data

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Basic root comment section | Root post by Alice; 3 comments by Bob/Carol/Dave; each has reply counts. | AT-003, IT-001, UT-004 |
| TD-002 | Comment pagination | Root post with 12 comments, timestamps one minute apart. | AT-004, IT-004 |
| TD-003 | Viewer grouping and sort | Viewer-authored comments at early/late timestamps plus non-viewer comments at interleaved timestamps. | AT-006, AT-007, IT-005, UT-001, UT-011 |
| TD-004 | Reply pagination | One comment with direct replies and deeper descendants ordered oldest-first as visual branch replies. | AT-005, IT-007, UT-009 |
| TD-005 | Focus outside pages | >10 comments; focused branch outside page 1; branch has >10 replies; focused reply after first 10. | AT-002, IT-003, IT-006, IT-011 |
| TD-006 | Deeper backend chain | Root -> comment -> reply -> deeper reply target, with actual root/parent refs. | AT-008, AT-009, IT-008, UT-003, UT-010 |
| TD-007 | Invalid focus cases | Malformed AT-URI, missing reply URI, focus belonging to a different root. | IT-010, GAP-001 follow-up tests |
| TD-008 | Localization labels | English l10n strings for `oldest`, `newest`, `follows`, `view replies`, `load more`, `hide replies`, focused reply state. | AT-010, UT-013 |

## 8. Manual Checks

| ID | Requirement IDs | Check | Steps | Expected Result |
|---|---|---|---|---|
| MAN-001 | BR-001, FR-003, FR-004 | Real app deep-link launch and scroll feel. | Launch the Flutter app from a route containing `?focus=<encoded AT-URI>` on a simulator/device. | App opens root post, expands focused branch, and the focused reply is visibly brought into view without jarring scroll jumps. |
| MAN-002 | FR-006, FR-011 | Scroll and load-more UX feel. | Use a seeded/development account with >10 comments and >10 replies under a comment. Scroll the root list and use reply load more. | Comment lazy loading and reply action loading feel predictable, with no duplicate or disappearing records. |
| MAN-003 | FR-018 | Copy/label clarity for stubbed `follows`. | Select each sort option in the UI. | `follows` is understandable as currently behaving like oldest, or follow-up UX copy/styling work is filed. |

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | Exact error behavior for invalid/mismatched focus is under-specified. | FR-003, FR-004, NFR-001 | Requirements allow reject or redirect/reload for some edge cases. | Implementation should choose a concrete behavior, then add handler and Flutter error-state tests. |
| GAP-002 | Exact visual treatment for flattened deeper reply parent context remains non-blocking. | FR-014, FR-015, FR-024, RULE-003 | Requirements mandate structural `replyingTo` metadata and composer mention but leave optional visible “replying to” label undecided. | Test required metadata, required mention, and two-level cap now; add visual-label tests if product chooses additional context labeling. |
| GAP-003 | Full OS-level push notification deep-link delivery is out of scope. | BR-001 | Push infrastructure is not part of this change. | Cover route/deep-link handling with widget/manual checks; add notification integration tests when push exists. |
| GAP-004 | Backend response shape is not named in requirements. | FR-002 | Requirements define contents, not endpoint name/body fields. | TDD builder should choose a concrete route/model consistent with `/v1/` conventions and pin it in first API tests. |

## 10. Out Of Scope

- Real follows graph ranking or storage tests; `follows` must be tested only as an oldest-first stub.
- Lexicon migration tests; requirements say no lexicon change is needed.
- Push notification registration/delivery tests; only deep-link route behavior is in scope.
- Cross-client backward compatibility tests for `/thread`; the route is pre-production and intentionally removed.
- Full end-to-end tests against a live PDS for reply creation; existing handler/PDS fake tests are the appropriate automation target for this stage.

## 11. Handoff To Document Review

- Requirements file: `docs/changes/2026-05-12-reply-comment-section/02-requirements.md`
- Test specification: `docs/changes/2026-05-12-reply-comment-section/03-acceptance-tests.md`
- Next review artifact: `04-document-review.md`
- Preferred Plannotator review: `/plannotator-annotate docs/changes/2026-05-12-reply-comment-section/`
- Recommended first failing test for implementation: `IT-001` for the new AppView comment-section read surface returning root post plus comments only, with required `placement` and `replies` metadata. This pins the API contract before Flutter work depends on it.
- Suggested test order for implementation:
  1. `IT-001`, `IT-014`, `IT-015`, `IT-004`, `IT-005` — backend comment-section contract, placement, replies loaded-state, pagination, sort/viewer grouping.
  2. `IT-002`, `IT-003`, `IT-006`, `IT-011`, `IT-013`, `IT-016` — focus handling/status, bounded focused reply slices, and flattened reply metadata.
  3. `IT-007`, `IT-008`, `IT-009`, `IT-010` — reply loading, parent refs, route removal, API conventions.
  4. `UT-007`, `UT-006`, `UT-001` through `UT-017` — Flutter models/providers/state merging.
  5. `AT-001` through `AT-010` — Flutter widget/user workflows.
  6. `REG-001` through `REG-005` — stale thread removal and unaffected post behavior.
- Commands discovered:
  - Backend: `just test` from repo root after `just dev-d` is running. This runs `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test -race ./...`.
  - Backend focused examples: `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api ./internal/routes`.
  - Flutter tests exist under `app/test/**`; a likely focused command is `cd app && flutter test test/feed`.
- Blocking gaps: None. `GAP-001` and `GAP-004` require implementation choices but do not block useful TDD because first tests can define the concrete API behavior.
