# Acceptance Test Specification: AppView Search Foundation

## 1. Test Strategy

This specification verifies the AppView-only search foundation from route/API behavior through storage, ranking, pagination, moderation filtering, and private recent-search persistence. The requirements are **medium risk** because the slice adds multiple authenticated `/v1/search/*` endpoints, new private persistence, search-supporting indexes, deterministic ranking rules, and response contracts intended for later Flutter UI consumption. Review is recommended before implementation continues.

Test design emphasizes Go automation in the AppView codebase:

1. API handler and route tests for `/v1/search/*` registration, session/device enforcement, validation, camelCase response bodies, standard error envelopes, and list response shapes.
2. Store/integration tests for exact hashtag equality, profile search ranking, full-text post/project keyword search, project filter semantics, top hashtag grouping, recent-search persistence, moderation filtering, stable pagination, and popularity ordering.
3. Unit tests for input normalization, validation/parsing, cursor handling, recent-search normalization/de-duplication, profile relevance classification, and centralized popularity score calculation.
4. Regression tests that protect existing `/v1/facets/*` autocomplete semantics, existing post/timeline/profile response contracts, and existing moderation behavior.
5. Manual checks limited to database query-plan/index review, log redaction review, and final contract review for API documentation because those are not fully proven by unit assertions.

Discovered commands:

- `just dev-d` from the repo root to start the compose database before integration tests.
- `just test` from the repo root; runs `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test -race ./...`.
- `just fmt` from the repo root for Go formatting and vetting.

Existing relevant test conventions:

- Route auth/device wrapping tests live in `appview/internal/routes/routes_test.go`.
- API handler tests use fake stores/clients in `appview/internal/api/*_test.go`.
- API store tests use inline DDL fixtures in `appview/internal/api/post_store_test.go`, `timeline_store_test.go`, `profile_store_test.go`, and related store tests.
- Cursor helpers are covered under `appview/internal/api/envelope/cursor_test.go`.
- Response-shape tests live in `appview/internal/api/post_response_test.go`, `profile_response_test.go`, and `facet_response_test.go`.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-002, AC-016 | AT-002, AT-009, IT-002, IT-003, REG-003 | Acceptance / Integration / Regression | Yes |
| BR-002 | AC-003, AC-004 | AT-003, UT-006, IT-004 | Acceptance / Unit / Integration | Yes |
| BR-003 | AC-005, AC-006, AC-007 | AT-004, UT-004, IT-008, IT-014 | Acceptance / Unit / Integration | Yes |
| BR-004 | AC-008, AC-009 | AT-005, UT-007, IT-007 | Acceptance / Unit / Integration | Yes |
| BR-005 | AC-010, AC-011 | AT-006, UT-003, IT-006 | Acceptance / Unit / Integration | Yes |
| BR-006 | AC-012, AC-013 | AT-008, UT-005, IT-010, IT-011 | Acceptance / Unit / Integration | Yes |
| FR-001 | AC-014, AC-015 | AT-001, AT-009, IT-001, REG-001 | Acceptance / Integration / Regression | Yes |
| FR-002 | AC-001, AC-002, AC-012, AC-015, AC-021 | AT-002, AT-008, AT-009, UT-001, IT-002, IT-009 | Acceptance / Unit / Integration | Yes |
| FR-003 | AC-001, AC-021, EC-001, EC-002 | AT-002, UT-001, UT-002, IT-002 | Acceptance / Unit / Integration | Yes |
| FR-004 | AC-003, AC-004, EC-003 | AT-003, UT-006, IT-004 | Acceptance / Unit / Integration | Yes |
| FR-005 | AC-004 | AT-003, UT-006, IT-004 | Acceptance / Unit / Integration | Yes |
| FR-006 | AC-012, AC-013, AC-016, AC-022 | AT-007, AT-008, AT-009, UT-002, UT-005, IT-005, IT-010, IT-012 | Acceptance / Unit / Integration | Yes |
| FR-007 | AC-010, AC-011, AC-013, AC-023 | AT-006, AT-008, UT-003, IT-006, IT-011 | Acceptance / Unit / Integration | Yes |
| FR-008 | AC-010, AC-011 | AT-006, UT-003, IT-006 | Acceptance / Unit / Integration | Yes |
| FR-009 | AC-016, AC-024 | AT-009, UT-008, IT-012, REG-003 | Acceptance / Unit / Integration / Regression | Yes |
| FR-010 | AC-017, EC-004 | AT-010, IT-013, REG-004 | Acceptance / Integration / Regression | Yes |
| FR-011 | AC-008, AC-009 | AT-005, IT-007 | Acceptance / Integration | Yes |
| FR-012 | AC-008, AC-009, AC-025 | AT-005, UT-007, IT-007 | Acceptance / Unit / Integration | Yes |
| FR-013 | AC-005, AC-006, AC-007 | AT-004, UT-004, IT-008 | Acceptance / Unit / Integration | Yes |
| FR-014 | AC-005, AC-006, AC-027, EC-005 | AT-004, UT-004, IT-008 | Acceptance / Unit / Integration | Yes |
| FR-015 | AC-006, AC-007, AC-018 | AT-004, IT-008, IT-014 | Acceptance / Integration | Yes |
| FR-016 | AC-015, EC-010 | AT-009, UT-009, IT-009 | Acceptance / Unit / Integration | Yes |
| FR-017 | AC-013, AC-026 | AT-008, UT-005, IT-010 | Acceptance / Unit / Integration | Yes |
| FR-018 | AC-019, EC-006 | AT-006, AT-007, UT-002, UT-003, IT-001 | Acceptance / Unit / Integration | Yes |
| FR-019 | AC-022 | AT-007, IT-005 | Acceptance / Integration | Yes |
| FR-020 | AC-023, EC-012 | AT-006, IT-006 | Acceptance / Integration | Yes |
| FR-021 | AC-027 | AT-004, UT-004, IT-008 | Acceptance / Unit / Integration | Yes |
| FR-022 | AC-006, AC-007 | AT-004, UT-004, IT-008, IT-014 | Acceptance / Unit / Integration | Yes |
| NFR-001 | AC-014, AC-015, AC-019 | AT-001, AT-009, UT-002, IT-001 | Acceptance / Unit / Integration | Yes |
| NFR-002 | AC-015, AC-020 | AT-009, AT-011, UT-002, MAN-001, GAP-001 | Acceptance / Unit / Manual | Mixed |
| NFR-003 | AC-013 | AT-008, UT-005, MAN-002 | Acceptance / Unit / Manual | Mixed |
| NFR-004 | AC-020 | AT-011, IT-015, MAN-001 | Acceptance / Integration / Manual | Mixed |
| NFR-005 | AC-004, AC-020 | AT-003, AT-011, UT-006, MAN-001 | Acceptance / Unit / Manual | Mixed |
| NFR-006 | AC-020, AC-022 | AT-007, AT-011, IT-005, MAN-001 | Acceptance / Integration / Manual | Mixed |
| RULE-001 | AC-018 | AT-004, IT-014, REG-005 | Acceptance / Integration / Regression | Yes |
| RULE-002 | AC-001, AC-002 | AT-002, UT-001, IT-002, IT-003 | Acceptance / Unit / Integration | Yes |
| RULE-003 | AC-004, EC-007 | AT-003, UT-002, IT-001 | Acceptance / Unit / Integration | Yes |
| RULE-004 | AC-005, EC-008 | AT-004, IT-015 | Acceptance / Integration | Yes |
| RULE-005 | AC-007, AC-018, EC-013 | AT-004, IT-008, IT-014 | Acceptance / Integration | Yes |
| RULE-006 | AC-012, AC-023 | AT-006, AT-008, IT-006, IT-009 | Acceptance / Integration | Yes |

## 3. Acceptance Scenarios

### AT-001: Search Routes Require Authenticated Devices

Requirement IDs: FR-001, NFR-001
Acceptance Criteria: AC-014, AC-019
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/routes/routes_test.go`, `appview/internal/api/search_test.go`

```gherkin
Feature: Authenticated AppView search routes
  Scenario Outline: Search endpoint rejects missing session or device headers
    Given the AppView route mux has registered /v1/search routes
    When a request to <endpoint> is missing <required_header>
    Then the response status is the existing auth or device middleware error status
    And the response body uses the standard AppView error envelope with error, message, and requestId

    Examples:
      | endpoint | required_header |
      | GET /v1/search/hashtags/sock/posts | Authorization |
      | GET /v1/search/profiles?q=ali | X-Craftsky-Device-Id |
      | GET /v1/search/posts?q=sock | Authorization |
      | GET /v1/search/projects | X-Craftsky-Device-Id |
      | GET /v1/search/hashtags/top?craftTypes=knitting | Authorization |
      | GET /v1/search/recent | X-Craftsky-Device-Id |
      | POST /v1/search/recent | Authorization |
      | DELETE /v1/search/recent/recent_123 | X-Craftsky-Device-Id |
```

### AT-002: Exact Hashtag Search Uses Canonical Equality

Requirement IDs: BR-001, FR-002, FR-003, RULE-002
Acceptance Criteria: AC-001, AC-002, AC-021
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/search_store_test.go`, `appview/internal/api/search_test.go`

```gherkin
Feature: Exact hashtag search
  Scenario: Hashtag results only include top-level visible posts with the exact normalized stored tag
    Given visible top-level posts and projects have stored tags sock, sockknitting, and SOCK normalized as indexed values
    And another visible post text visually contains #sock but its indexed tags do not contain sock
    And a reply contains the stored tag sock
    When an authenticated user calls GET /v1/search/hashtags/Sock/posts
    Then only visible top-level posts and projects whose stored normalized tag equals sock are returned
    And posts with sockknitting are not returned
    And display-text-only #sock matches are not returned
    And replies are not returned
    And response metadata identifies the canonical hashtag as sock without a leading #
```

### AT-003: Profile Search Ranks Followed And Relevant Craftsky Profiles

Requirement IDs: BR-002, FR-004, FR-005, RULE-003, NFR-005
Acceptance Criteria: AC-003, AC-004, AC-020
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/search_store_test.go`, `appview/internal/api/search_test.go`

```gherkin
Feature: Profile search
  Scenario: Search returns Craftsky profiles ordered by followed-first relevance
    Given Craftsky profiles match query ali by exact handle, prefix handle, handle substring, display name, and bio text
    And one non-Craftsky profile also matches the query
    And the viewer follows one weaker textual match
    When the viewer calls GET /v1/search/profiles?q=ali
    Then only Craftsky profiles are returned with profile summary fields
    And followed matches are ordered before non-followed matches
    And within each followed or non-followed group exact and prefix handle matches rank before handle substring, display-name, and bio matches
    And deterministic tie-breakers keep the order stable
    When the viewer sends sort=popular or sort=chronological to profile search
    Then the endpoint returns a validation error using the standard error envelope
```

### AT-004: Recent Searches Are Explicit, Private, Deduplicated, And Hard Deleted

Requirement IDs: BR-003, FR-013, FR-014, FR-015, FR-021, FR-022, RULE-001, RULE-004, RULE-005
Acceptance Criteria: AC-005, AC-006, AC-007, AC-018, AC-027
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/search_recent_store_test.go`, `appview/internal/api/search_test.go`

```gherkin
Feature: Private recent searches
  Scenario: A user explicitly saves, lists, refreshes, prunes, and deletes recent searches
    Given Alice and Bob are authenticated Craftsky users
    When Alice commits hashtag, profile, post, and project-filter searches and POSTs each to /v1/search/recent
    Then Alice's recent-search list returns newest-first entries with opaque IDs, type metadata, display labels, and rerunnable normalized payloads
    And Bob's recent-search list does not include Alice's entries
    When Alice saves the same normalized search again with a display label
    Then the existing recent search is refreshed to the top instead of duplicated
    And updatedAt is refreshed while the existing stored display label remains unchanged
    When Alice saves more than 50 distinct searches
    Then only the latest 50 remain after pruning
    When Alice deletes a recent-search ID
    Then the row is hard deleted and no longer appears in Alice's list
    And deleting an already deleted, nonexistent, or not-owned opaque ID returns idempotent success without leaking ownership
```

### AT-005: Blank Search Top Hashtags Are Grouped By Craft Type

Requirement IDs: BR-004, FR-011, FR-012
Acceptance Criteria: AC-008, AC-009, AC-025
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/search_store_test.go`, `appview/internal/api/search_test.go`

```gherkin
Feature: Top hashtags for blank search
  Scenario: Recent project hashtags are counted per requested craft group
    Given recent top-level project posts across knitting and crochet have materialized hashtags
    And one project repeats the same hashtag across multiple materialized tag sources
    And one requested craft type has no recent hashtag activity
    When the app calls GET /v1/search/hashtags/top?craftTypes=knitting&craftTypes=crochet&craftTypes=quilting
    Then the response contains separate craft groups for knitting, crochet, and quilting
    And tag counts use the 28-day v1 window
    And each tag count represents distinct project posts, not repeated occurrences inside one project
    And the inactive requested craft group is present with items as an empty array
```

### AT-006: Project Search Applies Filters And Browse-All Semantics

Requirement IDs: BR-005, FR-007, FR-008, FR-018, FR-020, RULE-006
Acceptance Criteria: AC-010, AC-011, AC-019, AC-023
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/search_store_test.go`, `appview/internal/api/search_test.go`

```gherkin
Feature: Project search and filters
  Scenario: Project search supports case-insensitive filters, AND/OR semantics, and browse-all
    Given visible top-level project posts have different craft types, project types, pattern difficulties, colors, materials, design tags, and project tags
    When the app calls GET /v1/search/projects with repeated values inside one filter family
    Then projects matching any repeated value in that family may be returned
    When the app combines different filter families
    Then only projects matching every requested filter family are returned
    And user-facing string values match case-insensitively while response data preserves stored display values
    When no filters and no q are provided
    Then all visible top-level projects are returned in default chronological order
    When unsupported filter fields or invalid filter values are provided
    Then the endpoint returns a documented validation error rather than silently broadening results
```

### AT-007: General Post And Project Keyword Search Use Indexed Local Text

Requirement IDs: FR-006, FR-018, FR-019, NFR-006
Acceptance Criteria: AC-019, AC-022
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/search_store_test.go`, `appview/internal/api/search_test.go`

```gherkin
Feature: Post and project keyword search
  Scenario: Keyword search finds visible top-level records through documented local indexed fields
    Given visible top-level posts and project posts contain matches in post text, project title, pattern name, material text, project tags, and design tags
    And non-matching replies also contain the query text
    When the app calls GET /v1/search/posts?q=alpaca
    Then matching visible top-level regular and project posts are returned
    And replies are not returned as result items
    When the app calls GET /v1/search/projects?q=alpaca
    Then matching visible top-level projects are returned using post text plus core materialized project common fields
    And missing, empty, or whitespace-only q for /v1/search/posts returns a validation error rather than a global all-posts feed
```

### AT-008: Post And Project Sort Orders Are Deterministic

Requirement IDs: BR-006, FR-002, FR-006, FR-007, FR-017, RULE-006, NFR-003
Acceptance Criteria: AC-012, AC-013, AC-024, AC-026
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/search_store_test.go`, `appview/internal/api/search_ranking_test.go`

```gherkin
Feature: Search result ordering
  Scenario: Chronological and popularity sort produce stable post/project ordering
    Given matching post and project results have different authored creation times, URIs, active likes, active reposts, visible replies, deleted interactions, and hidden replies
    When the app requests sort=chronological or omits sort on post/project/hashtag result endpoints
    Then results are ordered by created_at descending and URI descending
    When the app requests sort=popular
    Then results are ordered by score = (likes + (2 * replies) + (3 * reposts)) / pow(1 + ageHours / 72, 1.5)
    And only active likes, active reposts, and visible descendant replies/comments contribute to the score
    And score ties sort by created_at descending and URI descending
    And response items include engagement counts but do not expose popularityScore
```

### AT-009: Search List Responses Use AppView Pagination And Post Contracts

Requirement IDs: FR-001, FR-002, FR-006, FR-009, FR-016, NFR-001, NFR-002
Acceptance Criteria: AC-015, AC-016, AC-024
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/search_test.go`, `appview/internal/api/search_store_test.go`, `appview/internal/api/post_response_test.go`

```gherkin
Feature: Search list response contracts
  Scenario: Paginated search responses are object-wrapped and stable across pages
    Given a search result set spans more than one page
    When the app calls a search list endpoint with a valid limit
    Then the response body contains an items array
    And an opaque cursor is present only when more results are available
    When the app follows the cursor
    Then the next page continues the same deterministic order without duplicates or omissions
    When the app sends an invalid cursor
    Then the response is 400 invalid_cursor using the standard error envelope
    And post-shaped search items use the same core PostResponse contract as timeline/profile post list items, including project fields when present
```

### AT-010: Moderation Filtering Happens Before Ranking And Limiting

Requirement IDs: FR-010
Acceptance Criteria: AC-017
Priority: Must
Level: Acceptance
Automation Target: `appview/internal/api/search_store_test.go`, `appview/internal/api/moderation_store_test.go`

```gherkin
Feature: Moderated content is absent from search
  Scenario Outline: Hidden or taken-down content does not appear in search results
    Given matching content would otherwise rank highly for <endpoint>
    And the post or author has an active hide or takedown moderation output
    When the app requests <endpoint>
    Then moderated rows are filtered before limiting and ranking
    And the returned page contains only visible results

    Examples:
      | endpoint |
      | GET /v1/search/hashtags/sock/posts |
      | GET /v1/search/posts?q=sock |
      | GET /v1/search/projects?q=sock |
      | GET /v1/search/profiles?q=alice |
      | GET /v1/search/hashtags/top?craftTypes=knitting |
```

### AT-011: Search Uses Bounded Local Indexed AppView Paths

Requirement IDs: NFR-002, NFR-004, NFR-005, NFR-006
Acceptance Criteria: AC-020, AC-022
Priority: Should
Level: Acceptance
Automation Target: `appview/internal/api/search_store_test.go`, migration/index tests under `appview/internal/db` or `appview/cmd/cli/migrate_test.go`

```gherkin
Feature: Search scalability guardrails
  Scenario: Search paths are bounded and local to AppView data
    Given a representative seeded AppView database
    When search endpoints are exercised with default and maximum limits
    Then result-list limits default to 25 and reject values above 100
    And top-hashtag group limits default to 10 and reject values above 50 per craft group
    And free-text queries, hashtag path values, recent-search display labels, recent-search payload sizes, filter-family counts, total filter counts, and cursors are bounded by the documented v1 limits
    And post/project keyword search uses the documented PostgreSQL full-text or local indexed strategy
    And profile search can use pg_trgm-backed local matching without changing followed-first deterministic ranking
    And normal result hydration does not perform per-result PDS or identity-service network calls
```

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-003, RULE-002 | AC-001, AC-021 | Normalize exact hashtag request path values. | `" #SockKAL "`, `"#sock"`, `"sock"`, `"##sock"` | One leading `#` is removed, whitespace trimmed, canonical lowercase tag returned; empty/invalid values are rejected. | `appview/internal/api/search_request_test.go` |
| UT-002 | FR-003, FR-006, FR-018, RULE-003, NFR-001, NFR-002 | AC-019 | Validate search query params. | Missing/blank post `q`, invalid `sort`, profile `sort=popular`, over-limit values, overlong query strings, invalid cursors. | Documented 400/422 standard error envelope; no silent broadening. | `appview/internal/api/search_request_test.go` |
| UT-003 | FR-007, FR-008, FR-018 | AC-010, AC-011, AC-019 | Parse project filter query parameters and normalize user-facing values. | Repeated `craftType`, `color`, `material`, `designTag`, `projectTag`, invalid key/value. | OR-within-family and AND-across-family representation; case-insensitive normalized comparisons; validation errors for unsupported inputs. | `appview/internal/api/search_request_test.go` |
| UT-004 | FR-014, FR-021, FR-022 | AC-005, AC-006, AC-027 | Normalize recent-search payloads and produce de-duplication keys. | Hashtag/profile/post/project recent payloads with different casing/ordering and duplicate saves with different display labels. | Stable normalized payload; opaque ID remains server generated; equal normalized payloads map to one per-user de-dupe key; duplicate saves preserve the existing stored display label while refreshing `updatedAt`. | `appview/internal/api/search_recent_test.go` |
| UT-005 | FR-017, BR-006, NFR-003 | AC-013, AC-026 | Calculate centralized popularity scores and tie-breakers. | Likes/replies/reposts, age in hours, equal scores. | Formula matches requirements; deleted/hidden inputs excluded by caller contract; ties sort `created_at DESC, uri DESC`. | `appview/internal/api/search_ranking_test.go` |
| UT-006 | FR-004, FR-005, BR-002, NFR-005 | AC-003, AC-004 | Classify profile match relevance. | Exact handle, prefix handle, handle substring, display-name, bio match, followed flag. | Followed group outranks non-followed; relevance class order is deterministic inside groups. | `appview/internal/api/search_profile_rank_test.go` |
| UT-007 | FR-012, BR-004 | AC-008, AC-009, AC-025 | Count top hashtags by distinct project per craft group. | Project rows with duplicate tag sources and requested empty craft groups. | Counts distinct project posts once per tag; requested empty groups return `items: []`. | `appview/internal/api/search_top_hashtags_test.go` |
| UT-008 | FR-009 | AC-016, AC-024 | Ensure search response DTOs wrap existing post responses and omit internal scores. | Post/project search rows with engagement counts and internal popularity score. | JSON contains normal post/project and engagement fields; `popularityScore` is absent. | `appview/internal/api/search_response_test.go`, `post_response_test.go` |
| UT-009 | FR-016 | AC-015 | Encode/decode stable opaque search cursors. | Chronological and popularity seek values, malformed cursor. | Valid cursors round-trip; malformed cursor maps to `invalid_cursor`. | `appview/internal/api/envelope/cursor_test.go` or `search_cursor_test.go` |

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | FR-001, FR-018, NFR-001 | AC-014, AC-019 | Registered route family enforces auth/device and validation envelope. | Build mux with `routes.AddRoutes`; seed fake deps. | Request every `/v1/search/*` route without auth/device and with malformed params. | Middleware errors and validation errors use standard envelopes and camelCase JSON. | `appview/internal/routes/routes_test.go`, `appview/internal/api/search_test.go` |
| IT-002 | BR-001, FR-002, FR-003, RULE-002 | AC-001, AC-021 | Exact hashtag search equality and canonical metadata. | Seed posts/projects with normalized tags `sock`, `sockknitting`, and `sockkal` casing variants. | `GET /v1/search/hashtags/SockKAL/posts`. | Only exact stored tag matches returned; metadata canonical tag is `sockkal`. | `appview/internal/api/search_store_test.go` |
| IT-003 | BR-001, FR-002, RULE-002 | AC-002 | Exclude text-only hashtag appearances and replies. | Seed top-level post text containing `#sock` without indexed tag and reply with indexed `sock`. | `GET /v1/search/hashtags/sock/posts`. | Neither text-only fallback nor reply appears. | `appview/internal/api/search_store_test.go` |
| IT-004 | BR-002, FR-004, FR-005, NFR-005 | AC-003, AC-004 | Profile search returns Craftsky profiles and followed-first relevance. | Seed identity cache, Craftsky profiles, Bluesky profile summaries, follow graph, non-Craftsky candidate. | `GET /v1/search/profiles?q=ali`. | Only Craftsky profile summaries return in followed-first relevance order; invalid sort returns validation error. | `appview/internal/api/search_store_test.go` |
| IT-005 | FR-006, FR-019, NFR-006 | AC-022 | Post/project keyword search covers post text and core project fields. | Seed visible top-level regular posts and project posts with matches across text/title/pattern/material/tags. | `GET /v1/search/posts?q=alpaca` and `GET /v1/search/projects?q=alpaca`. | Matching visible top-level records found through local FTS/indexed fields with deterministic tie-breakers. | `appview/internal/api/search_store_test.go` |
| IT-006 | BR-005, FR-007, FR-008, FR-020, RULE-006 | AC-010, AC-011, AC-023 | Project filters and browse-all project search. | Seed projects across craft type, project type, difficulty, color, material, design tag, project tag, and hidden/non-project rows. | Query repeated same-family filters, combined families, no filters, and `sort=popular`. | OR/AND filter semantics hold; no-filter browse returns all visible top-level projects chronological by default and popular when requested. | `appview/internal/api/search_store_test.go` |
| IT-007 | BR-004, FR-011, FR-012 | AC-008, AC-009, AC-025 | Top hashtags are grouped and distinct within 28-day window. | Seed project posts inside/outside 28 days across craft types with duplicate tag sources. | `GET /v1/search/hashtags/top` with and without `craftTypes`. | Returned groups and counts match requested/all craft groups; empty requested group included. | `appview/internal/api/search_store_test.go` |
| IT-008 | BR-003, FR-013, FR-014, FR-021, FR-022 | AC-005, AC-006, AC-007, AC-027 | Recent-search save/list/delete lifecycle. | Migrate recent-search table; authenticate one user. | Save all supported recent types, save duplicate with a different display label, exceed 50, delete one. | List is newest-first with opaque IDs and rerunnable payloads; duplicate refreshed with existing stored display label preserved; older entries pruned; delete hard removes row. | `appview/internal/api/search_recent_store_test.go` |
| IT-009 | FR-016, FR-002, RULE-006 | AC-012, AC-015 | Pagination preserves chronological order. | Seed more matching hashtag/post/project results than one page with equal timestamps requiring URI tiebreaker. | Request first page with limit and follow cursor. | Pages continue `created_at DESC, uri DESC` without duplicates; invalid cursor returns `400 invalid_cursor`. | `appview/internal/api/search_store_test.go` |
| IT-010 | BR-006, FR-006, FR-017 | AC-013, AC-026 | Popularity sort for general post search. | Seed active/deleted likes/reposts and visible/hidden descendant replies across differently aged posts. | `GET /v1/search/posts?q=sock&sort=popular`. | Results follow decayed score using active visible inputs only and deterministic tie-breakers. | `appview/internal/api/search_store_test.go` |
| IT-011 | BR-006, FR-007, FR-017 | AC-013, AC-023, AC-026 | Popularity sort for project search including browse-all. | Seed projects with different ages and engagement counts. | `GET /v1/search/projects?sort=popular`. | All visible top-level projects are popularity ordered; score ties stable. | `appview/internal/api/search_store_test.go` |
| IT-012 | FR-009 | AC-016, AC-024 | Search post items match existing PostResponse contract. | Seed regular and project posts with author data and engagement summaries. | Hit hashtag, post, and project search endpoints. | Items decode as existing core post response including project data when present and no `popularityScore`. | `appview/internal/api/search_test.go`, `post_response_test.go` |
| IT-013 | FR-010 | AC-017 | Moderation filters before limit/rank across search surfaces. | Seed visible and hidden/taken-down posts/accounts where moderated rows would sort first. | Query all result endpoints. | Moderated content/authors absent and do not consume page limit. | `appview/internal/api/search_store_test.go` |
| IT-014 | FR-015, RULE-001, RULE-005 | AC-018 | Recent searches are scoped to authenticated DID. | Seed Alice and Bob recent searches. | Alice lists/deletes Bob ID; Bob lists after Alice actions. | Cross-user content is never returned; not-owned delete is idempotent success and does not leak existence. | `appview/internal/api/search_recent_store_test.go` |
| IT-015 | RULE-004, NFR-004 | AC-020, EC-008 | Search result endpoints do not auto-save and avoid per-result external calls. | Seed result data and recent table; use fake PDS/identity clients with call counters if handlers expose dependencies. | Call result endpoints while typing-style queries, then list recents. | Recents unchanged unless explicit save endpoint called; no per-result external client calls in normal path. | `appview/internal/api/search_test.go`, `search_store_test.go` |

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Acceptance Criteria | Test |
|---|---|---|---|---|
| REG-001 | Existing non-search `/v1/*` routes keep auth/device wrapping and are not made public by search route additions. | FR-001, NFR-001 | AC-014 | Extend or preserve `routes_test.go` cases for `/v1/whoami`, `/v1/feed/timeline`, post replies/comments, and notifications while adding search route tests. |
| REG-002 | `/v1/facets/hashtags` remains autocomplete by substring and 28-day root-post counts; it is not replaced by exact hashtag search. | NG-004, FR-001 | AC-014 | Existing facet tests continue to pass; add a regression asserting facet autocomplete can still return substring suggestions independently of `/v1/search/hashtags/{tag}/posts`. |
| REG-003 | Timeline/profile/post list item response contracts remain compatible with search's reuse of `PostResponse`. | FR-009, BR-001 | AC-016, AC-024 | Existing `post_response_test.go`, timeline, profile post tests continue to decode expected author/project/engagement fields; search does not introduce `popularityScore`. |
| REG-004 | Existing moderation hide/takedown filtering semantics still apply to timeline/profile surfaces and are reused by search. | FR-010 | AC-017 | Existing moderation/timeline tests continue to pass; search store tests use the same active policy semantics. |
| REG-005 | Private-by-intent state is stored in AppView only, not written as atproto records. | RULE-001, FR-015 | AC-018 | Recent-search handler/store tests use AppView Postgres only and assert save/delete does not call PDS clients. |

## 7. Test Data

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Exact hashtag matching and response shape | Top-level regular posts and project posts tagged `sock`, `SOCK`, `sockknitting`; post text with visual `#sock` but no tag; reply with tag `sock`; hidden/taken-down match. | AT-002, AT-009, AT-010, IT-002, IT-003, IT-013 |
| TD-002 | Profile ranking | Craftsky profiles and identity cache rows for exact handle `ali.craftsky.social`, prefix handle `alice...`, handle substring, display-name match, bio match, followed weaker match, non-Craftsky profile. | AT-003, UT-006, IT-004 |
| TD-003 | Recent searches | Alice and Bob DIDs; hashtag/profile/post/project recent payloads; duplicate normalized payloads with different display labels; 51+ generated entries; nonexistent/not-owned opaque IDs. | AT-004, UT-004, IT-008, IT-014 |
| TD-004 | Top hashtag grouping | Project posts across knitting/crochet/quilting inside and outside 28 days, duplicate hashtag across materialized tag sources, requested craft group with no tags. | AT-005, UT-007, IT-007 |
| TD-005 | Project filters | Projects across craft type, craft-specific project type, pattern difficulty, colors, materials, design tags, project tags, keyword fields, plus regular non-project posts. | AT-006, IT-006 |
| TD-006 | Keyword search | Regular and project top-level posts with query matches in post text, project title, pattern name, material text, project tags, and design tags; replies with matching text. | AT-007, IT-005 |
| TD-007 | Popularity ordering | Posts/projects with controlled creation times, URIs, active/deleted likes, active/deleted reposts, visible replies, and hidden/taken-down replies. | AT-008, UT-005, IT-010, IT-011 |
| TD-008 | Pagination and validation | Result sets larger than default page size; equal timestamps/scores; valid and invalid opaque cursors; over-limit values and overlong query strings. | AT-009, UT-002, UT-009, IT-009 |

## 8. Manual Checks

| ID | Requirement IDs | Acceptance Criteria | Check | Steps | Expected Result |
|---|---|---|---|---|---|
| MAN-001 | NFR-002, NFR-004, NFR-005, NFR-006 | AC-020, AC-022 | Query-plan/index review for search paths. | After implementation migrations, run representative `EXPLAIN`/local development checks for hashtag equality, profile search, post/project FTS, project filters, top hashtags, and recent-search list using the documented default and maximum limits. | Queries use bounded limits and appropriate local indexes/materialized columns; no per-result network dependency is needed. |
| MAN-002 | BR-006, FR-017, NFR-003 | AC-013, AC-026 | Popularity formula review. | Review implementation docs/tests for the centralized formula and tie-breakers. | Formula is implemented once or through a shared helper; score is not part of public response JSON. |
| MAN-003 | RULE-001, FR-015 | AC-018 | Recent-search privacy/log redaction review. | Inspect logs and handler/store error paths for recent-search save/list/delete. | Full recent-search payloads and long free-text queries are not logged at high verbosity; recent state remains AppView-private. |
| MAN-004 | NFR-001, FR-009 | AC-014, AC-015, AC-016 | API contract review before Flutter UI work. | Review generated examples or handler responses for camelCase JSON, object-wrapped lists, optional cursors, and error envelope consistency. | Response shapes are stable and suitable for future Flutter search page consumption. |

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | No concrete latency/row-count performance threshold is specified. | NFR-002, NFR-005, NFR-006 | Requirements now define v1 request bounds and local indexed-path expectations, but not target data volume or max response time. | Use query-plan/index manual review now; add benchmark or load-test requirements when expected AppView scale is defined. |
| GAP-002 | `pg_trgm` adoption threshold for profile search is not quantified. | NFR-005 | Requirement says preferred once data size requires it, but not exactly when. | Implementation plan should decide whether to add `pg_trgm` immediately or document threshold; preserve ranking tests either way. |
| GAP-003 | Full response DTO field names for nested existing `PostResponse` and profile-summary objects are inherited rather than restated. | FR-002, FR-009, FR-011, FR-013, NFR-001 | Requirements pin minimum search-specific wrapper fields and rely on existing response contracts for nested post/profile objects. | Coding plan should reuse existing response builders and include handler tests for the documented wrapper fields. |

## 10. Out Of Scope

- Flutter search UI, state management, repositories, navigation, and widget tests.
- Lexicon changes or any PDS record type for recent searches.
- Public unauthenticated search endpoints, third-party XRPC search APIs, and moderation/admin search tooling.
- Typo-tolerant edit-distance search, semantic search, embeddings, recommendations, or algorithmic feed ranking.
- Block/mute filtering unless already indexed and enforced elsewhere.
- Backfilling missing hashtag materialization from raw display text; exact hashtag search uses indexed/materialized tags only.
- Broad performance/load testing beyond query-plan/index review because concrete targets are not in the requirements.

## 11. Handoff To Document Review

- Requirements file: `docs/changes/2026-06-19-appview-search-foundation/01-requirements.md`
- Test specification: `docs/changes/2026-06-19-appview-search-foundation/02-acceptance-tests.md`
- Next review artifact: `03-document-review.md`
- External Plannotator review, if the user initiates it outside this agent: `docs/changes/2026-06-19-appview-search-foundation/`
- Risk-based review recommendation: **Medium risk; review recommended before implementation.** The broad AppView API/store surface, private recent-search data, ranking semantics, and search-index performance concerns should be reviewed before coding.
- Recommended first failing test for implementation: `IT-001` / `AT-001` route registration and auth/device enforcement for `/v1/search/*`, because it establishes the endpoint family and AppView API conventions before store behavior is added.
- Suggested test order for implementation:
  1. `IT-001`, `UT-002`, `REG-001` for route family, auth/device middleware, and validation envelopes.
  2. `UT-001`, `IT-002`, `IT-003`, `AT-002` for exact hashtag normalization/equality.
  3. `UT-009`, `IT-009`, `AT-009` for pagination/cursor shape shared by result endpoints.
  4. `UT-005`, `IT-010`, `IT-011`, `AT-008` for centralized popularity ordering.
  5. `IT-012`, `REG-003` for post response contract reuse.
  6. `UT-006`, `IT-004`, `AT-003` for profile search and ranking.
  7. `IT-005`, `AT-007` for post/project keyword search and required post `q` validation.
  8. `UT-003`, `IT-006`, `AT-006` for project filtering and browse-all projects.
  9. `UT-007`, `IT-007`, `AT-005` for grouped top hashtags.
  10. `UT-004`, `IT-008`, `IT-014`, `AT-004` for recent-search persistence/privacy.
  11. `IT-013`, `AT-010`, `REG-004` for moderation filtering across all result surfaces.
  12. `IT-015`, `AT-011`, `MAN-001`, `MAN-003` for local indexed paths, no auto-save, and privacy/log review.
- Commands discovered:
  - `just dev-d`
  - `just test`
  - `just fmt`
- Blocking gaps: None. Non-blocking gaps are listed in Section 9.
