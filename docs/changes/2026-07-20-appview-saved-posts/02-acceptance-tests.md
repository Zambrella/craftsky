# Acceptance Test Specification: AppView Saved Posts

## 1. Test Strategy

This is a medium-risk AppView persistence, privacy, lifecycle, and API-contract change. The principal risks are exposing one account's private saved state, losing saves during folder operations, returning content that current policy forbids, leaving orphaned saved replies after permanent thread deletion, and introducing unstable or inefficient pagination. The automated design therefore uses these layers:

- Unit tests define folder-name validation, tri-state `folderId` request decoding, cursor compatibility, lifecycle decisions, response shaping, status selection, and bounded observability.
- Real-Postgres integration tests verify the reversible migration, owner/folder constraints, idempotent mutations, duplicate folder names, non-destructive folder deletion, saved-list ordering and pagination, target/ancestor cleanup, account lifecycle, concurrency, and indexed query plans.
- Handler and route tests exercise the complete `/v1/` JSON contract, authentication/device/body/rate policies, standard errors, immediate read-after-write, author non-disclosure, and absence of any PDS or Tap dependency.
- Regression tests protect existing post/reply/quote responses, root-plus-focus comment navigation, moderation and relationship policy, public interaction counts, session isolation, and unrelated schema.

All requirements can be automated using existing Go test conventions: table-driven package tests, `httptest`, fake stores/observers, and isolated real-Postgres schemas through `internal/testdb.WithSchema`. Real-Postgres cases must run with `TEST_DATABASE_URL`; they must not silently count a skipped local run as verification. No Flutter implementation or manual UI check is part of this slice.

The risk level remains **Medium**. No blocking test gap was found, but document review is recommended before coding because private owner state, destructive lifecycle cleanup, and pagination contracts are persistent and difficult to change after clients depend on them.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-002, AC-006 | AT-001, AT-005, IT-002, IT-006, IT-007, IT-009 | Acceptance / Integration | Yes |
| BR-002 | AC-003, AC-004, AC-005 | AT-002, IT-002, IT-003, IT-004, IT-006 | Acceptance / Integration | Yes |
| BR-003 | AC-011, AC-012, AC-013 | AT-003, AT-004, IT-003, IT-005 | Acceptance / Integration | Yes |
| BR-004 | AC-021, AC-025, AC-028 | AT-007, AT-009, IT-010, IT-013 | Acceptance / Integration | Yes |
| FR-001 | AC-001, AC-002 | AT-001, IT-006, IT-007 | Acceptance / Integration | Yes |
| FR-002 | AC-003, AC-004, AC-023, AC-031 | AT-001, AT-002, AT-010, UT-002, IT-002, IT-011 | Acceptance / Unit / Integration | Yes |
| FR-003 | AC-006 | AT-001, IT-002, IT-006 | Acceptance / Integration | Yes |
| FR-004 | AC-007, AC-008 | AT-003, UT-001, IT-003, IT-006 | Acceptance / Unit / Integration | Yes |
| FR-005 | AC-009, AC-033 | AT-003, UT-007, IT-003 | Acceptance / Unit / Integration | Yes |
| FR-006 | AC-010, AC-023, AC-032 | AT-003, AT-010, IT-004, IT-011 | Acceptance / Integration | Yes |
| FR-007 | AC-011, AC-028 | AT-003, AT-007, UT-004, IT-003, IT-010 | Acceptance / Unit / Integration | Yes |
| FR-008 | AC-012, AC-020, AC-032 | AT-004, AT-009, IT-005, IT-006, IT-012 | Acceptance / Integration | Yes |
| FR-009 | AC-013, AC-027 | AT-004, UT-003, IT-005 | Acceptance / Unit / Integration | Yes |
| FR-010 | AC-002, AC-014, AC-028 | AT-001, AT-005, AT-007, UT-009, IT-007, IT-010 | Acceptance / Unit / Integration | Yes |
| FR-011 | AC-015, AC-026, AC-028 | AT-007, UT-009, IT-010, IT-014, IT-015 | Acceptance / Unit / Integration | Yes |
| FR-012 | AC-016, AC-017 | AT-006, UT-008, IT-008 | Acceptance / Unit / Integration | Yes |
| FR-013 | AC-016, AC-017, AC-018 | AT-005, AT-006, AT-008, UT-005, IT-008, IT-009 | Acceptance / Unit / Integration | Yes |
| FR-014 | AC-019, AC-028 | AT-008, UT-005, IT-009, IT-010, REG-005 | Acceptance / Unit / Integration / Regression | Yes |
| FR-015 | AC-020, AC-021 | AT-009, IT-006, IT-012, IT-013 | Acceptance / Integration | Yes |
| FR-016 | AC-021, AC-022 | AT-009, AT-010, IT-006, IT-013 | Acceptance / Integration | Yes |
| FR-017 | AC-005, AC-009, AC-012, AC-025, AC-032 | AT-002, AT-003, AT-004, AT-007, UT-010, IT-003, IT-004, IT-005, IT-013 | Acceptance / Unit / Integration | Yes |
| FR-018 | AC-023 | AT-010, IT-011 | Acceptance / Integration | Yes |
| FR-019 | AC-024, AC-026 | IT-001, IT-010, IT-014 | Integration | Yes |
| FR-020 | AC-022, AC-023, AC-034 | AT-001, AT-002, AT-010, UT-006, IT-002, IT-004, IT-006, IT-011 | Acceptance / Unit / Integration | Yes |
| NFR-001 | AC-025, AC-029 | AT-007, AT-009, UT-011, IT-013 | Acceptance / Unit / Integration | Yes |
| NFR-002 | AC-026 | IT-010, IT-014 | Integration | Yes |
| NFR-003 | AC-027 | AT-004, UT-003, UT-004, IT-005 | Acceptance / Unit / Integration | Yes |
| NFR-004 | AC-028 | AT-007, IT-002, IT-003, IT-004, IT-005, IT-010 | Acceptance / Integration | Yes |
| NFR-005 | AC-029 | UT-011, IT-013 | Unit / Integration | Yes |
| NFR-006 | AC-024, AC-030 | IT-001, IT-015, REG-001–REG-007 | Integration / Regression | Yes |
| RULE-001 | AC-003, AC-004 | AT-001, AT-002, IT-002, IT-011 | Acceptance / Integration | Yes |
| RULE-002 | AC-003, AC-012, AC-031 | AT-002, AT-004, UT-002, IT-002, IT-005 | Acceptance / Unit / Integration | Yes |
| RULE-003 | AC-004, AC-006, AC-010, AC-013 | AT-001, AT-002, AT-003, AT-004, UT-007, IT-002, IT-004, IT-005 | Acceptance / Unit / Integration | Yes |
| RULE-004 | AC-007 | AT-003, IT-003 | Acceptance / Integration | Yes |
| RULE-005 | AC-021 | AT-009, IT-013, REG-003 | Acceptance / Integration / Regression | Yes |
| RULE-006 | AC-010, AC-016, AC-017, AC-018 | AT-003, AT-005, AT-006, AT-008, UT-005, IT-004, IT-008, IT-009 | Acceptance / Unit / Integration | Yes |
| RULE-007 | AC-007, AC-008, AC-009 | AT-003, UT-001, IT-003 | Acceptance / Unit / Integration | Yes |
| RULE-008 | AC-002, AC-014, AC-018 | AT-001, AT-005, IT-007, IT-009, REG-002 | Acceptance / Integration / Regression | Yes |

## 3. Acceptance Scenarios

### AT-001: Save And Unsave Every Eligible Post Type

Requirement IDs: BR-001, FR-001, FR-002, FR-003, FR-010, FR-020, RULE-001, RULE-003, RULE-008
Acceptance Criteria: AC-001, AC-002, AC-003, AC-006, AC-034
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/saved_post_test.go`

```gherkin
Feature: Private saved posts
  Scenario Outline: Save and unsave an eligible indexed post
    Given Alice is authenticated
    And an eligible indexed <postType> exists at its canonical URI
    When Alice saves that exact DID and rkey without a folder
    Then the AppView returns 201 with one unfiled save and a server-assigned savedAt
    And the returned post is the exact target rather than a root, parent, or quoted subject
    When Alice repeats the same save
    Then the AppView returns 200 with the same savedAt and no duplicate
    When Alice unsaves it twice
    Then both deletes return 204 and no save remains
    When Alice later saves the eligible target again
    Then it receives a later savedAt

    Examples:
      | postType |
      | ordinary top-level post |
      | project post |
      | quote post |
      | direct comment |
      | nested reply |
```

### AT-002: Assign, Move, And Unfile One Save

Requirement IDs: BR-002, FR-002, FR-017, FR-020, RULE-001, RULE-002, RULE-003
Acceptance Criteria: AC-003, AC-004, AC-005, AC-031, AC-034
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/saved_post_test.go`

```gherkin
Feature: Single-folder organization
  Scenario: Existing save distinguishes omitted folder from explicit null
    Given Alice has folders A and B
    And Alice saved a post in folder A
    When Alice repeats the save with no body or with folderId omitted
    Then the response is 200 and folder A and savedAt are preserved
    When Alice saves it with folder B's ID
    Then the response is 200 and only folder B remains assigned
    And savedAt is unchanged
    When Alice saves it with folderId explicitly null
    Then the save becomes unfiled without changing savedAt
    When Alice supplies a missing or Bob-owned folder ID
    Then the response is the same 404 saved_post_folder_not_found
    And the save remains unchanged
```

### AT-003: Manage Flat Duplicate-Named Folders

Requirement IDs: BR-003, FR-004, FR-005, FR-006, FR-007, FR-017, RULE-003, RULE-004, RULE-006, RULE-007
Acceptance Criteria: AC-007, AC-008, AC-009, AC-010, AC-011, AC-032, AC-033
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/saved_post_folder_test.go`, `appview/internal/api/saved_post_store_test.go`

```gherkin
Feature: Saved-post folders
  Scenario: Create, rename, list, and delete duplicate-named folders
    Given Alice is authenticated
    When Alice creates folders named Ideas, IDEAS, and " Ideas "
    Then all three creations return 201 with distinct opaque IDs
    And the trimmed third display name is Ideas
    And no folder response includes nesting, sharing, or saved-post-count fields
    When folders are listed across page boundaries
    Then they are ordered case-insensitively by display name and then folder ID
    When Alice renames one folder to another existing display name
    Then the rename succeeds, its ID and createdAt remain, and updatedAt advances
    When saves are added, moved, unfiled, or removed
    Then the folder updatedAt does not change
    When Alice deletes a non-empty folder
    Then its saves become unfiled with unchanged savedAt
    And repeating the delete, including with a missing or Bob-owned ID, returns 204 without changing Bob's data
```

### AT-004: List All, Foldered, And Unfiled Saves In Either Direction

Requirement IDs: BR-003, FR-008, FR-009, FR-017, NFR-003, RULE-002, RULE-003
Acceptance Criteria: AC-012, AC-013, AC-027, AC-032
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/saved_post_store_test.go`, `appview/internal/api/saved_post_test.go`

```gherkin
Feature: Saved-post listing
  Scenario: Page through each scope and sort direction
    Given Alice has foldered and unfiled saves whose post createdAt values differ from savedAt values
    When Alice lists all saves, one owned folder, and unfiled saves
    Then each response contains only eligible saves in that scope
    And the default and newest orders use savedAt descending with URI descending as the tie-breaker
    And oldest uses savedAt ascending with URI ascending as the tie-breaker
    And folder edits do not affect either order
    When a cursor is reused with another scope or sort, or is malformed
    Then the AppView returns 400 invalid_cursor
    When folderId and unfiled=true are combined
    Then validation rejects the request
    When the folder scope is missing or belongs to Bob
    Then both requests return the same 404 saved_post_folder_not_found
```

### AT-005: Preserve Exact Reply Identity And Valid Thread Context

Requirement IDs: BR-001, FR-010, FR-013, RULE-006, RULE-008
Acceptance Criteria: AC-002, AC-014, AC-018
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/saved_post_test.go`, `appview/internal/index/craftsky_post_test.go`

```gherkin
Feature: Saved reply context
  Scenario: Save a nested reply and react to context lifecycle
    Given a root, comment, parent reply, and nested target reply are indexed
    When Alice saves the nested target
    Then the saved-list item is the exact target URI
    And its canonical PostResponse retains root and parent strong references
    And no private snapshot of its parent chain is stored
    When required context is temporarily unavailable
    Then the item is hidden but the save, folderId, and savedAt are retained
    When the root or a required ancestor is permanently deleted
    Then the now-unnavigable saved reply is removed in the same AppView indexer deletion transaction
    And the still-indexed descendant reply row is retained unless it receives its own delete event
```

### AT-006: Apply Current Policy Without Discarding Temporary Saves

Requirement IDs: FR-012, FR-013, RULE-006
Acceptance Criteria: AC-016, AC-017
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/saved_post_policy_test.go`, `appview/internal/api/saved_post_store_test.go`

```gherkin
Feature: Saved-post policy
  Scenario Outline: A saved target's current policy changes
    Given Alice saved Bob's otherwise eligible post
    When Bob's state becomes <state>
    Then the saved list <visible outcome>
    And the private save <persistence outcome>

    Examples:
      | state | visible outcome | persistence outcome |
      | muted by Alice | returns the post with viewer-relative mute state | remains unchanged |
      | blocked in either direction | exposes no forbidden post payload | remains retained while the block is temporary policy state |
      | hidden or taken down | exposes no forbidden post payload | remains retained |
      | not a current member | exposes no forbidden post payload | remains retained |
      | eligible again at the same URI | returns the post again | keeps its original folderId and savedAt |
```

### AT-007: Isolate Private State Between Accounts

Requirement IDs: BR-004, FR-007, FR-010, FR-011, FR-017, NFR-001, NFR-004
Acceptance Criteria: AC-015, AC-025, AC-028
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/saved_post_test.go`, `appview/internal/api/saved_post_store_test.go`

```gherkin
Feature: Owner-private saved state
  Scenario: Alice and Bob request the same post on one device
    Given Alice saved the post in her folder and Bob did not
    When each account reads the post, feeds, folders, and saved lists
    Then Alice sees viewerHasSaved true and only her nullable viewerSavedFolderId
    And Bob sees viewerHasSaved false and null
    And neither response embeds a saved folder name or another owner's metadata
    When Bob submits Alice's folder ID to any operation
    Then Bob cannot read or mutate Alice's save or folder
    And errors, logs, traces, and metrics expose no private owner-target pair, folder ID, folder name, or saved URI
    And Bob cannot use Alice's cursor to cross the authenticated owner boundary
```

### AT-008: Apply Owner And Content Lifecycle Rules

Requirement IDs: FR-013, FR-014, RULE-006
Acceptance Criteria: AC-018, AC-019
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/saved_post_store_test.go`, `appview/internal/relationships/lifecycle_test.go`, `appview/internal/index/craftsky_post_test.go`

```gherkin
Feature: Saved-state lifecycle
  Scenario Outline: A lifecycle event occurs
    Given Alice has folders and saves and Bob has independent saved state
    When <event>
    Then <outcome>
    And Bob's state is unchanged

    Examples:
      | event | outcome |
      | Alice signs out, logs out all sessions, removes a device, expires a token, reinstalls, or switches account | Alice's server folders and saves remain |
      | Alice's Craftsky membership is permanently removed | only Alice's folders and saves are deleted |
      | the exact saved PDS record is permanently deleted | every save of that URI is deleted |
      | a required saved-reply root or ancestor is permanently deleted | every affected orphaned reply save is deleted |
```

### AT-009: Keep Saved Operations Private And AppView-Local

Requirement IDs: BR-004, FR-008, FR-015, FR-016, NFR-001, NFR-005, RULE-005
Acceptance Criteria: AC-020, AC-021, AC-025, AC-029
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/routes/routes_test.go`, `appview/internal/api/saved_post_test.go`, `appview/internal/api/saved_post_observability_test.go`

```gherkin
Feature: Private AppView API boundary
  Scenario: Exercise every saved-post and folder route
    Given the routes use the normal Craftsky session and device middleware
    When authenticated and unauthenticated clients send valid, malformed, oversized, unknown-field, and invalid-filter requests
    Then auth, device, body, read/write rate, camelCase JSON, and standard error-envelope policies apply
    And successful writes change only private AppView tables
    And no PDS client, lexicon record, Tap event, notification, author-visible state, or public interaction count is produced
    And telemetry contains only bounded operation, result, stage, and error-class values
```

### AT-010: Preserve Atomicity And Immediate Read-After-Write

Requirement IDs: FR-002, FR-006, FR-016, FR-018, FR-020
Acceptance Criteria: AC-022, AC-023
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/saved_post_store_test.go`

```gherkin
Feature: Concurrent saved-state mutation
  Scenario: Duplicate saves, moves, unfiling, unsave, and folder deletion race
    Given Alice owns a save and two folders
    When coordinated concurrent transactions repeat saves, move the save, delete a folder, unfile, and unsave
    Then every successful response reflects committed state immediately
    And at most one Alice/save row remains
    And it has at most one valid Alice-owned folder assignment
    And no dangling folder reference or partially deleted folder state exists
    And the final result matches one valid transaction ordering
```

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-004, RULE-007 | AC-007, AC-008 | Table-test folder display-name trimming and validation without uniqueness. | Empty/whitespace; 1, 100, and 101 Unicode characters; emoji; punctuation; `/`; `\`; Unicode control characters; duplicate/case-variant names. | Valid names preserve accepted casing after trimming; invalid boundaries fail `validation_failed`; duplicate names do not conflict. | `appview/internal/api/saved_post_folder_request_test.go` |
| UT-002 | FR-002, RULE-002 | AC-003, AC-031 | Decode the optional save body and preserve the three request states. | No body, `{}`, `{"folderId":null}`, valid ID, malformed JSON, unknown field, trailing JSON. | New-save no body/omission/null means unfiled; existing-save no body/omission means preserve; explicit null means unfile; malformed/unknown/trailing input is rejected. | `appview/internal/api/saved_post_request_test.go` |
| UT-003 | FR-009, NFR-003 | AC-027 | Encode and validate saved-list cursors against scope and direction using the existing envelope cursor contract. | All/folder/unfiled scopes; newest/oldest; timestamp/URI key; malformed, tampered, and cross-scope/sort cursors. | Valid base64url-JSON cursor round-trips as a client-opaque token; its decoded payload may contain owner-visible folder/post scope and keyset values but omits owner DID; incompatible or malformed input returns `invalid_cursor`; no encryption or confidentiality behavior is asserted. | `appview/internal/api/saved_post_cursor_test.go` |
| UT-004 | FR-007, NFR-003 | AC-011, AC-027 | Encode deterministic folder cursors with duplicate/case-variant names. | `Ideas`, `IDEAS`, duplicate `Ideas`, distinct opaque IDs, page boundaries. | Case-insensitive name plus ID creates stable once-only order and opaque continuation. | `appview/internal/api/saved_post_cursor_test.go` |
| UT-005 | FR-013, FR-014, RULE-006 | AC-016, AC-017, AC-018, AC-019 | Table-test saved-state lifecycle decisions. | Owner membership removal; sign-out/device/token/account events; exact target deletion; root/ancestor deletion; temporary policy/context/member ineligibility; restoration. | Only owner membership and permanent target/required-ancestor deletion remove applicable rows; session/device events and temporary ineligibility retain them. | `appview/internal/api/saved_post_lifecycle_test.go` |
| UT-006 | FR-020 | AC-034 | Select success status from mutation outcome. | Created, unchanged existing, moved, unfiled, save delete, folder delete. | Create is 201; existing results are 200; committed deletes are 204. | `appview/internal/api/saved_post_response_test.go` |
| UT-007 | FR-005, RULE-003 | AC-009, AC-033 | Decide timestamp effects for folder and save mutations. | Rename; add/move/unfile/remove save; folder delete; save move; unsave/resave. | Only rename advances folder `updatedAt`; organization edits preserve `savedAt`; unsave/resave creates a new `savedAt`. | `appview/internal/api/saved_post_lifecycle_test.go` |
| UT-008 | FR-012 | AC-016, AC-017 | Apply saved-list policy as an explicit/direct-access surface. | Eligible, muted, blocking, blocked-by, hidden, takedown, non-member, temporarily unavailable, restored. | Muted eligible post remains shaped and visible; stricter states expose no forbidden payload; restoration is eligible without changing private metadata. | `appview/internal/api/saved_post_policy_test.go` |
| UT-009 | FR-010, FR-011 | AC-002, AC-014, AC-015 | Serialize saved-list and canonical viewer fields. | Top-level post and nested reply; saved/unsaved; foldered/unfiled; different viewer. | Exact canonical post and reply refs remain; `viewerHasSaved` and nullable folder ID use camelCase; folder name and other-owner data are absent. | `appview/internal/api/saved_post_response_test.go`, `appview/internal/api/post_response_test.go` |
| UT-010 | FR-017 | AC-005, AC-009, AC-012, AC-032 | Map owned, missing, and other-owner folder outcomes by operation. | Assignment, rename, scoped list, delete. | Assignment/rename/list map missing and foreign to identical 404; delete maps both to 204 no-op. | `appview/internal/api/saved_post_error_test.go` |
| UT-011 | NFR-001, NFR-005 | AC-025, AC-029 | Verify saved-operation diagnostics use bounded, redacted fields. | Success/failure logs, wrapped errors, metric/trace attributes containing private sentinel DIDs, URI, folder ID/name, owner-target pair. | Sentinels are absent; only bounded operation/result/stage/error-class and request correlation remain. | `appview/internal/api/saved_post_observability_test.go`, `appview/internal/observability/error_classifier_test.go` |

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | FR-019, NFR-006 | AC-024, AC-030 | Exercise the saved-post migration up/down/up and schema contract. | Version-current pre-feature DDL with profiles and posts; migration files loaded into an isolated Postgres schema. | Apply up; inspect tables, constraints, FKs, and indexes; insert valid/invalid ownership and duplicate-name fixtures; apply down and up again. | Only new tables disappear on down; unique owner/post and folder owner integrity hold; duplicate names remain valid; owner/post/folder delete actions and ordering indexes match requirements. The exact-post FK cleans up only exact-target saves; ancestor cleanup is explicitly outside that FK. Any UUID storage choice is not asserted as a JSON contract. | `appview/internal/db/saved_posts_migration_test.go` |
| IT-002 | FR-002, FR-003, FR-020, NFR-004, RULE-001, RULE-002, RULE-003 | AC-003, AC-004, AC-006, AC-023, AC-031, AC-034 | Persist one owner-scoped save through idempotent create, tri-state assignment, move, unfile, delete, and resave. | Real Postgres; Alice and Bob; eligible post; two Alice folders and one Bob folder; controllable clock. | Execute each mutation repeatedly and read after every commit. | One `(ownerDid, postUri)` row; at most one owned folder; omission/null semantics, status outcomes, timestamp preservation, 204 delete without target lookup, and later new `savedAt` all hold; Bob is unchanged. | `appview/internal/api/saved_post_store_test.go`, `appview/internal/api/saved_post_test.go` |
| IT-003 | FR-004, FR-005, FR-007, FR-017, NFR-004, RULE-004, RULE-007 | AC-005, AC-007, AC-008, AC-009, AC-011, AC-028, AC-033 | Create, rename, and list owner-private folders including duplicates. | Real Postgres; Alice and Bob; valid/invalid names; duplicate/case-variant fixtures; fixed timestamps. | Create/rename/list as both owners across pages and mutate folder contents. | Opaque string IDs distinguish duplicates and round-trip without UUID parsing; validation and trimming hold; alphabetical/ID order is stable; rename-only `updatedAt`; no count/nesting/sharing fields; foreign read/rename is indistinguishable from missing. | `appview/internal/api/saved_post_store_test.go`, `appview/internal/api/saved_post_folder_test.go` |
| IT-004 | FR-006, FR-017, FR-020, NFR-004, RULE-006 | AC-010, AC-023, AC-032, AC-034 | Delete folders non-destructively and idempotently. | Alice folder with several saves plus Bob folder/save; fixed save timestamps. | Delete Alice's folder repeatedly, then issue deletes for missing and Bob-owned IDs as Alice. | Alice saves atomically become unfiled with unchanged timestamps; every delete returns 204 after commit; Bob's rows never change; no dangling FK remains. | `appview/internal/api/saved_post_store_test.go`, `appview/internal/api/saved_post_folder_test.go` |
| IT-005 | FR-008, FR-009, FR-017, NFR-003, NFR-004, RULE-002, RULE-003 | AC-012, AC-013, AC-027, AC-032 | List eligible saves across all/folder/unfiled scopes and both directions. | More than two pages of Alice saves with tied saved times, differing post times, foldered/unfiled rows, hidden rows, and Bob rows. | Traverse every scope/direction; introduce newer saves between pages; replay malformed and incompatible cursors. | Stable keyset pages contain every eligible Alice row once in saved-time/URI order; scopes do not mix; incompatible cursor is 400; missing/foreign folder is identical 404. Cursors follow the existing base64url-JSON opacity contract, may encode Alice's scope/keyset values, omit owner DID, and cannot authorize access outside the authenticated owner. | `appview/internal/api/saved_post_store_test.go`, `appview/internal/api/saved_post_test.go` |
| IT-006 | FR-001, FR-003, FR-004, FR-008, FR-015, FR-016, FR-020 | AC-001, AC-006, AC-008, AC-020, AC-022, AC-031, AC-032, AC-034 | Verify complete handler contracts and validation. | Fake saved store with call recording and failures; authenticated request helpers; fixed request ID/clock. | Exercise every route with valid/no/invalid bodies, path IDs, filters, limits, cursors, store outcomes, and immediate follow-up reads. | CamelCase response shapes, 201/200/204, standard errors, max/default limits, unknown-field/trailing/body-limit rejection, idempotent unsave, and read-after-write all match the API contract. | `appview/internal/api/saved_post_test.go`, `appview/internal/api/saved_post_folder_test.go` |
| IT-007 | FR-001, FR-010, RULE-008 | AC-002, AC-014 | Hydrate every supported exact post type through the saved list. | Indexed ordinary, project, quote, direct comment, and nested reply rows with canonical CIDs and reply refs. | Save and list each target. | Returned `post.uri` is the exact saved URI; project/quote fields remain canonical; reply root/parent refs remain; no parent/root substitution or private content snapshot occurs. | `appview/internal/api/saved_post_store_test.go`, `appview/internal/api/saved_post_test.go` |
| IT-008 | FR-012, FR-013, RULE-006 | AC-016, AC-017 | Apply current relationship, moderation, membership, and availability policy while retaining temporary saves. | Alice saves Bob posts; mute/block directions; hide/takedown; target-member removal/restoration; temporary missing context. | List while each policy state is active, inspect storage, then restore eligibility. | Muted eligible post follows direct-access shaping; forbidden payload never appears for stricter states; private row remains; restored item returns with original `savedAt` and folder assignment. | `appview/internal/api/saved_post_policy_test.go`, `appview/internal/api/saved_post_store_test.go` |
| IT-009 | FR-013, FR-014, RULE-006, RULE-008 | AC-018, AC-019 | Enforce destructive target/context/owner lifecycle and non-destructive session lifecycle. | Alice and Bob saved-state fixtures; root/comment/parent/target reply graph; membership/session events. | Delete exact target, root, intermediate ancestor, or Alice membership; simulate sign-out/logout-all/device/token/reinstall/account-switch events. | The post-indexer deletion transaction determines affected descendants before removing the event URI and atomically deletes exact/affected save rows. Still-indexed descendant public post rows remain, owner membership removes only Alice state, session/device events retain Alice state, no content snapshot remains, and Bob is unchanged. | `appview/internal/index/craftsky_post_test.go`, `appview/internal/api/saved_post_store_test.go`, `appview/internal/relationships/lifecycle_test.go` |
| IT-010 | FR-007, FR-010, FR-011, FR-014, FR-019, NFR-002, NFR-004 | AC-015, AC-026, AC-028 | Extend the shared `EngagementSummaries` seam to batch-hydrate viewer saved state without cross-owner or N+1 access. | Full-size post/feed/search/profile/saved pages; Alice foldered/unfiled saves; Bob differing saves; query-recording fake plus real Postgres. | Hydrate each surface as Alice and Bob through the extended engagement-summary path. | The shared seam performs one bounded set-based saved-state lookup per result set; no canonical surface adds an independent or per-item save query; booleans/folder IDs reflect only the requesting DID; no folder names appear; empty/duplicate URI inputs remain bounded. | `appview/internal/api/saved_post_response_test.go`, `appview/internal/api/saved_post_store_test.go` |
| IT-011 | FR-002, FR-006, FR-018, FR-020, RULE-001 | AC-023 | Coordinate concurrent duplicate saves, moves, folder deletion, unfiling, and unsave. | Real Postgres; explicit transaction barriers/channels; one owner/post and two folders. | Release competing mutations in controlled orderings and repeat under `-race`. | Constraints and transactions yield one valid serial outcome, never duplicate saves, cross-owner/dangling folders, partial unfiling, or success before commit. | `appview/internal/api/saved_post_store_test.go` |
| IT-012 | FR-008, FR-015 | AC-020 | Register and enforce saved route policies through the real mux. | `routes.testDeps`, normal auth/device middleware, low test rate limits. | Probe all seven routes without auth/device, with wrong body kind/oversize, and beyond rate limits. | Every route is registered; GET/DELETE/body-optional POST/PATCH policies are correct; auth and device checks precede handler work; errors use the standard envelope. | `appview/internal/routes/routes_test.go` |
| IT-013 | BR-004, FR-015, FR-016, FR-017, NFR-001, NFR-005, RULE-005 | AC-021, AC-025, AC-029 | Prove private AppView-only mutation, author non-disclosure, and diagnostic redaction. | Recording PDS/Tap/notification collaborators that fail the test if called; Alice saves Bob; captured logs/metrics/traces with private sentinels. | Run successful and failed save/folder operations and read Bob's author-facing/public shapes. | Only private DB state changes; no external/public interaction occurs; Bob sees no signal/count; all diagnostic fields remain bounded and sentinels absent. | `appview/internal/api/saved_post_test.go`, `appview/internal/api/saved_post_observability_test.go` |
| IT-014 | FR-011, FR-019, NFR-002 | AC-026 | Guard indexed set-based query plans for saved lists and the extended engagement-summary viewer state. | Real Postgres with representative cardinality; `ANALYZE`; all/folder/unfiled and shared `EngagementSummaries` URI-batch queries. | Run `EXPLAIN (FORMAT JSON)` and inspect the central store call shape. | Plans use intended owner/scope/order indexes without per-item statements, parallel per-surface helpers, or unbounded private-table scans; both sort directions use compatible access paths. | `appview/internal/api/saved_post_query_plan_test.go` |
| IT-015 | FR-011, NFR-006 | AC-015, AC-030 | Exercise additive saved viewer state from the shared `EngagementSummaries` path on every canonical post-shaped surface. | Existing post, timeline, profile content, project, search, comment/reply, notification, quote, and saved-list fixtures. | Request each surface as saved and unsaved viewers while recording hydration calls. | Existing fields and ordering are unchanged; saved viewer fields are correct and additive everywhere promised; every surface uses the central batch seam rather than a separate saved-state query; existing suites continue to pass. | Existing `appview/internal/api/*_test.go` suites plus `appview/internal/api/saved_post_response_test.go` |

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Test |
|---|---|---|---|
| REG-001 | Canonical post, project, quote, comment, and reply JSON remains backward-compatible. | FR-010, FR-011, NFR-006 | Extend `post_response_test.go` and affected list tests to assert existing fields/counts/refs remain unchanged while the two viewer save fields are additive camelCase fields. |
| REG-002 | Comment deep links still route through the root with the exact reply as focus. | FR-010, RULE-008 | Keep `post_test.go` focused-comment/reply/ancestor coverage passing and assert saved response refs supply the same root/focus inputs; a reply used as route root still returns `invalid_post_role`. |
| REG-003 | Likes, reposts, replies, quotes, notifications, and public counts do not treat saves as public interactions. | FR-016, RULE-005 | Save/unsave while asserting engagement summaries, notification events, push deliveries, PDS fakes, and Tap/indexer calls remain unchanged. |
| REG-004 | Existing mute, block, moderation, takedown, and membership policy retains its precedence on other surfaces. | FR-012 | Run existing relationship/moderation response and pagination suites; saved-list direct-access shaping must not weaken feed, search, thread, or profile policy. |
| REG-005 | Session/device/account operations remain account-scoped and do not delete membership-owned state. | FR-014 | Extend lifecycle/auth regression coverage so sign-out, logout-all, device removal, token expiry, and account switching retain saves and never clear another DID. |
| REG-006 | Every registered `/v1/` route remains covered by one valid policy and standard middleware. | FR-015 | Keep `TestV1RoutePoliciesCoverRegisteredRoutes` and mux-wide enforcement tests passing after all seven routes are added. |
| REG-007 | Saved migration reversal leaves unrelated public/private schema intact. | FR-019, NFR-006 | Migration test snapshots pre-existing profiles/posts/interactions/mutes/blocks, runs up/down/up, and verifies only saved tables/indexes are removed and recreated. |

## 7. Test Data

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Owner isolation | Alice, Bob, and Carol current-member DIDs with separate authenticated sessions and shared/distinct device IDs. | AT-002, AT-007–AT-010, IT-002–IT-013 |
| TD-002 | Supported post graph | Ordinary post, project post, quote post, root, direct comment, parent reply, and nested reply with canonical URIs/CIDs/root/parent refs. | AT-001, AT-005, IT-007–IT-009, REG-001–REG-003 |
| TD-003 | Folder identity/order | Fixed opaque IDs with `Ideas`, `IDEAS`, duplicate `Ideas`, emoji, punctuation, and cross-owner same-name folders. | AT-003, UT-001, UT-004, IT-001, IT-003–IT-005 |
| TD-004 | Time ordering | Fixed post `createdAt`, `indexedAt`, folder timestamps, tied/distinct `savedAt`, and controllable resave clock values. | AT-001–AT-004, UT-007, IT-002–IT-005 |
| TD-005 | Folder validation | Empty/whitespace, 1/100/101 Unicode characters, composed/decomposed Unicode, emoji, slash, backslash, newline, tab, NUL, and other control characters. | UT-001, IT-003, IT-006 |
| TD-006 | Request tri-state | No body, `{}`, `{"folderId":null}`, valid/foreign/missing folder IDs, malformed JSON, trailing JSON, unknown fields, and oversized body. | AT-002, UT-002, IT-002, IT-006, IT-012 |
| TD-007 | Policy lifecycle | Muted, blocking, blocked-by, hide, takedown, non-member, temporarily unavailable, restored, exact-delete, root-delete, and ancestor-delete states. | AT-005, AT-006, AT-008, UT-005, UT-008, IT-008, IT-009 |
| TD-008 | Pagination density | At least 103 saves and more than 100 folders, including hidden saves, tied timestamps, duplicate names, and rows arriving between requests. | AT-003, AT-004, UT-003, UT-004, IT-003, IT-005, IT-014 |
| TD-009 | Concurrency | Transaction barriers for duplicate insert, move-to-A, move-to-B, explicit unfile, folder delete, and unsave. | AT-010, IT-011 |
| TD-010 | Privacy sentinels | Unique owner DID, target DID, saved URI, opaque folder ID string, and folder name embedded in candidate errors/logs/metrics/traces. | AT-007, AT-009, UT-011, IT-013 |
| TD-011 | Migration pre-state | Current pre-feature profiles/posts plus unrelated likes, reposts, mutes, blocks, notifications, and representative owner/content rows. | IT-001, REG-007 |

## 8. Manual Checks

None identified. This slice has no UI, device-only behavior, or external service whose required behavior cannot be exercised with existing Go unit, handler, route, and real-Postgres integration tests.

## 9. Test Gaps And Risks

None identified. Implementation must still ensure that real-Postgres tests actually run with `TEST_DATABASE_URL`; a skipped database suite is not acceptable evidence for migration, concurrency, lifecycle, pagination, or query-plan completion.

## 10. Out Of Scope

- Flutter saved-post screens, buttons, folder pickers, navigation widgets, local caching, optimistic state, and accessibility tests.
- PDS, lexicon, Tap synchronization, Bluesky bookmark import/export, or cross-AppView portability tests.
- Nested folders, multi-folder membership, manual ordering, counts, quotas, notes, search, sharing, recommendations, and notifications for saves.
- Load/soak benchmarks or a numeric latency SLA. Indexed query-plan and bounded-query-count tests are required instead.
- Manual visual checks, because this slice changes no user interface.

## 11. Handoff To Document Review

- Requirements file: `01-requirements.md`
- Test specification: `02-acceptance-tests.md`
- Next review artifact: `03-document-review.md`
- External Plannotator review, if the user initiates it outside this skill: `docs/changes/2026-07-20-appview-saved-posts/`
- Recommended first failing test for implementation: `IT-001` in `appview/internal/db/saved_posts_migration_test.go`, defining the reversible tables, owner/folder constraints, duplicate-name allowance, exact-target foreign-key actions, and indexes before store code depends on them. The separate ancestor-cleanup behavior begins with `IT-009` because it belongs to the post-indexer transaction rather than the exact-post foreign key.
- Suggested test order for implementation:
  1. `IT-001` migration contract.
  2. `UT-001`–`UT-007` validation, tri-state requests, cursors, lifecycle, statuses, and timestamps.
  3. `IT-002`–`IT-005` core save/folder persistence and pagination.
  4. `UT-008`–`UT-011` policy, response, authorization/error mapping, and observability.
  5. `IT-006`, `IT-007`, `IT-010`, `IT-012`, and `IT-013` handlers, hydration, routes, and privacy boundary.
  6. `IT-008`, `IT-009`, `IT-011`, and `IT-014` policy lifecycle, destructive cleanup, concurrency, and query plans.
  7. `IT-015` and `REG-001`–`REG-007` full response and behavior regression pass.
- Commands discovered:
  - Focused unit/handler tests: `cd appview && go test ./internal/api ./internal/routes -run 'TestSaved|TestV1RoutePolicies|TestAddRoutes'`
  - Focused real-Postgres tests: `cd appview && TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test -race ./internal/db ./internal/api ./internal/index ./internal/relationships`
  - Full AppView gate from repository root, with compose Postgres running: `just test`
  - Formatting/vet after implementation: `just fmt`
- Blocking gaps: None.
