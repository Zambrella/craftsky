# Acceptance Test Specification: Search Refinements Before UI Slice

## 1. Test Strategy

This test design covers the non-UI AppView and Flutter data/logic refinements described in `01-requirements.md`. The strategy is test-first and contract-focused:

- Use AppView unit tests for request parsing, normalization, ranking helpers, cursor validation, recent-search payload validation, and project/search boundary rules.
- Use AppView store/handler/route integration tests for authenticated API contracts, Postgres-backed ranking/counting, pagination, visibility predicates, and compatibility wrappers.
- Use Flutter model, API-client, repository, and Riverpod provider tests for JSON mapping, shared authenticated Dio usage, independent pagination state, blank-search data, and feature-boundary enforcement.
- Use regression tests to protect existing facet autocomplete, exact hashtag behavior, private recent-search behavior, and the no-rendered-UI boundary.
- Keep manual checks limited to source-diff review, architectural/privacy inspection, and bounded/index-aware query review where automated tests cannot fully prove production-scale performance.

Read-only test context discovered from the repository:

- AppView tests already live under `appview/internal/api/*_test.go` and `appview/internal/routes/routes_test.go`.
- Flutter search tests already live under `app/test/search/**`; rich-text facet compatibility tests live under `app/test/shared/rich_text/**`.
- Flutter project discovery data code exists under `app/lib/projects/data/**` and `app/lib/projects/providers/**`, with no current `app/test/projects/data/**` suite.
- Existing command hints: `just dev-d`, `just test`, focused `cd appview && go test ./internal/api ./internal/routes -count=1`, `cd app && dart run build_runner build --delete-conflicting-outputs`, `cd app && flutter test test/search test/shared/rich_text test/projects`, `cd app && flutter analyze`, and `cd app && flutter test`.

Risk-based review recommendation: retain the requirements risk level of **Medium**. A short document review before implementation is recommended because this slice crosses AppView API contracts, Flutter data providers, recents payloads, ranking semantics, and project/search boundaries. Review may be skipped by the user, but implementation should not begin without an explicit decision.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-006, AC-012 | AT-001, AT-004, IT-013, REG-005, MAN-001 | Acceptance / Integration / Regression / Manual | Mostly; source-diff manual |
| BR-002 | AC-011, AC-012, AC-013 | AT-001, AT-007, AT-008, UT-011, IT-007, IT-013 | Acceptance / Unit / Integration | Yes |
| BR-003 | AC-002, AC-003, AC-004 | AT-002, AT-003, UT-002, UT-003, IT-001, IT-002, IT-003, REG-001 | Acceptance / Unit / Integration / Regression | Yes |
| BR-004 | AC-005, AC-006, AC-007, AC-014 | AT-004, AT-005, AT-009, UT-006, IT-004, IT-005, IT-011, IT-012, IT-013 | Acceptance / Unit / Integration | Yes |
| BR-005 | AC-009, AC-010 | AT-006, UT-008, IT-008, IT-009, IT-014, REG-007 | Acceptance / Unit / Integration / Regression | Yes |
| BR-006 | AC-011 | AT-007, UT-007, IT-010, REG-004 | Acceptance / Unit / Integration / Regression | Yes |
| FR-001 | AC-002, AC-014, AC-017 | AT-002, AT-009, UT-001, IT-001, IT-011 | Acceptance / Unit / Integration | Yes |
| FR-002 | AC-003, AC-004, AC-017 | AT-002, AT-003, UT-002, IT-002, IT-012, REG-001 | Acceptance / Unit / Integration / Regression | Yes |
| FR-003 | AC-002, AC-004, AC-005 | AT-002, AT-003, AT-005, UT-003, IT-003, IT-004, REG-001 | Acceptance / Unit / Integration / Regression | Yes |
| FR-004 | AC-004 | AT-003, IT-002, IT-003, IT-015, REG-001 | Acceptance / Integration / Regression | Yes |
| FR-005 | AC-005, AC-014 | AT-005, AT-009, UT-003, IT-004, IT-011, IT-012 | Acceptance / Unit / Integration | Yes |
| FR-006 | AC-006, AC-014, AC-018 | AT-004, AT-009, UT-006, UT-010, IT-005, IT-011, IT-013 | Acceptance / Unit / Integration | Yes |
| FR-007 | AC-007 | AT-004, UT-006, IT-005, REG-003 | Acceptance / Unit / Integration / Regression | Yes |
| FR-008 | AC-008, AC-021 | AT-005, UT-004, UT-013, IT-006, IT-017, REG-002 | Acceptance / Unit / Integration / Regression | Yes |
| FR-009 | AC-009, AC-010, AC-014 | AT-006, AT-009, UT-005, UT-008, IT-008, IT-011, IT-014, MAN-003 | Acceptance / Unit / Integration / Manual | Mostly; index review manual |
| FR-010 | AC-009, AC-010, AC-019 | AT-006, UT-008, IT-008, IT-009, IT-014, REG-007 | Acceptance / Unit / Integration / Regression | Yes |
| FR-011 | AC-010 | AT-006, IT-014, REG-007 | Acceptance / Integration / Regression | Yes |
| FR-012 | AC-001, AC-002, AC-005, AC-006, AC-008, AC-012, AC-021 | AT-001, AT-002, AT-004, AT-005, UT-009, UT-010, UT-013, IT-012, IT-013, IT-017, REG-005 | Acceptance / Unit / Integration / Regression | Yes |
| FR-013 | AC-011 | AT-007, UT-007, UT-012, IT-010, REG-004 | Acceptance / Unit / Integration / Regression | Yes |
| FR-014 | AC-013 | AT-008, UT-005, IT-007, IT-008 | Acceptance / Unit / Integration | Yes |
| FR-015 | AC-013 | AT-008, UT-009, UT-011, IT-007, IT-008 | Acceptance / Unit / Integration | Yes |
| FR-016 | AC-011, AC-020 | AT-007, UT-007, IT-010, IT-012, REG-004 | Acceptance / Unit / Integration / Regression | Yes |
| NFR-001 | AC-001 | AT-001, REG-005, MAN-001 | Acceptance / Regression / Manual | Partly manual |
| NFR-002 | AC-014, AC-015 | AT-009, UT-009, IT-011, IT-012, IT-014, REG-006, MAN-002 | Acceptance / Unit / Integration / Regression / Manual | Mostly; architecture inspection manual |
| NFR-003 | AC-014, AC-021 | AT-009, UT-001, UT-004, UT-008, UT-013, IT-011, IT-012, IT-017 | Acceptance / Unit / Integration | Yes |
| NFR-004 | AC-016 | AT-010, IT-004, IT-008, IT-016, MAN-003, GAP-002 | Acceptance / Integration / Manual / Gap | Partly manual |
| NFR-005 | AC-016 | AT-010, UT-001 through UT-013, IT-001 through IT-017, REG-001 through REG-007 | Acceptance / Unit / Integration / Regression | Yes |
| RULE-001 | AC-011, AC-015 | AT-007, UT-007, IT-010, REG-004, REG-006, MAN-002 | Acceptance / Unit / Integration / Regression / Manual | Mostly |
| RULE-002 | AC-011 | AT-007, UT-012, IT-010, REG-004 | Acceptance / Unit / Integration / Regression | Yes |
| RULE-003 | AC-003, AC-004 | AT-002, AT-003, UT-002, IT-002, REG-001 | Acceptance / Unit / Integration / Regression | Yes |
| RULE-004 | AC-005, AC-008, AC-021 | AT-005, UT-003, UT-004, UT-013, IT-004, IT-006, IT-017, REG-002 | Acceptance / Unit / Integration / Regression | Yes |
| RULE-005 | AC-009, AC-010 | AT-006, UT-008, IT-008, IT-009, IT-014, REG-007 | Acceptance / Unit / Integration / Regression | Yes |
| RULE-006 | AC-007 | AT-004, UT-006, IT-005, REG-003 | Acceptance / Unit / Integration / Regression | Yes |
| RULE-007 | AC-002, AC-017 | AT-002, UT-001, IT-001 | Acceptance / Unit / Integration | Yes |
| RULE-008 | AC-006, AC-018 | AT-004, UT-006, IT-005, IT-013 | Acceptance / Unit / Integration | Yes |
| RULE-009 | AC-011, AC-020 | AT-007, UT-007, IT-010, IT-014, REG-004 | Acceptance / Unit / Integration / Regression | Yes |

## 3. Acceptance Scenarios

### AT-001: Non-UI contracts provide blank search data

Requirement IDs: BR-001, BR-002, FR-012, NFR-001  
Acceptance Criteria: AC-001, AC-012  
Priority: Must  
Level: Acceptance  
Automation Target: `app/test/search/providers/blank_search_provider_test.dart`, `app/test/search/search_page_test.dart`, and source-diff review

```gherkin
Feature: Non-UI search contracts
  Scenario: Future blank search logic can fetch recents and craft-grouped top hashtags without rendered UI work
    Given the slice is limited to AppView and Flutter data/logic layers
    And recent-search and top-hashtag repositories are available through UI-agnostic providers
    When future blank SearchPage logic requests its initial data
    Then recent searches are fetched through the search repository
    And top hashtags are fetched for the supported default craft tokens
    And no rendered search tabs, management pages, project layouts, cards, or navigation behavior are added by this slice
```

### AT-002: Unified typeahead suggestions return bounded profile and hashtag previews

Requirement IDs: BR-003, FR-001, FR-002, FR-003, RULE-003, RULE-007  
Acceptance Criteria: AC-002, AC-003, AC-017  
Priority: Must  
Level: Acceptance  
Automation Target: `appview/internal/api/search_request_test.go`, `appview/internal/api/facet_suggestion_test.go`, `appview/internal/api/search_store_test.go`, `app/test/search/data/search_api_client_test.dart`

```gherkin
Feature: Search typeahead suggestions
  Scenario: Authenticated typeahead receives bounded top-N suggestions with hasMore metadata
    Given an authenticated user with indexed Craftsky profiles, follows, crafts, regular posts, and project posts
    And more profile and hashtag matches exist than the requested per-section limit
    When the user types a non-empty query and the unified suggestion contract is requested for profiles and hashtags
    Then AppView returns profile and hashtag sections with camelCase fields
    And each section contains no more than the requested top-N items
    And each section reports hasMore accurately without returning a pagination cursor
    And profile summary data includes craft metadata
    And hashtag values are normalized canonical tags
```

### AT-003: Existing facet autocomplete remains compatible with shared suggestion logic

Requirement IDs: BR-003, FR-004, RULE-003  
Acceptance Criteria: AC-003, AC-004  
Priority: Must  
Level: Acceptance / Regression  
Automation Target: `appview/internal/api/facet_test.go`, `appview/internal/api/facet_suggestion_test.go`, `app/test/shared/rich_text/facet_suggestion_repository_test.dart`

```gherkin
Feature: Facet autocomplete compatibility
  Scenario: Composer/profile rich-text suggestions keep working after suggestion unification
    Given existing rich-text code requests mention and hashtag suggestions through `/v1/facets/*`
    When the shared suggestion core is used by search and facets
    Then mention and hashtag facet responses keep the fields expected by current Flutter callers
    And overlapping profile suggestions are ranked equivalently between search typeahead and facet mention autocomplete
    And suggestion failures remain tolerant for rich-text autocomplete where current behavior is tolerant
```

### AT-004: Submitted result tabs page independently and keep posts/projects disjoint

Requirement IDs: BR-004, FR-006, FR-007, RULE-006, RULE-008  
Acceptance Criteria: AC-006, AC-007, AC-018  
Priority: Must  
Level: Acceptance  
Automation Target: `appview/internal/api/search_store_test.go`, `app/test/search/providers/post_search_provider_test.dart`, `app/test/search/providers/project_search_provider_test.dart`, `app/test/search/providers/profile_search_provider_test.dart`, `app/test/search/providers/hashtag_result_search_provider_test.dart`

```gherkin
Feature: Submitted search result tabs
  Scenario: Four result categories fetch independently with relevance-first post and project results
    Given indexed regular posts, project posts, profiles, and hashtags match a submitted text query
    And some project posts also match the text that regular posts match
    When Flutter reads the submitted-search providers for Posts, Projects, Profiles, and Hashtags
    Then each provider fetches through its own repository/API path
    And each provider exposes UI-agnostic async state and paginates with its own opaque cursor
    And regular posts appear only in Posts results
    And project posts appear only in Projects results
    And post/project text search defaults to relevance-first ordering with deterministic tie-breakers
```

### AT-005: Hashtag query results and exact hashtag feeds remain separate modes

Requirement IDs: BR-004, FR-005, FR-008, FR-012, NFR-003, RULE-004  
Acceptance Criteria: AC-005, AC-008, AC-014, AC-021  
Priority: Must  
Level: Acceptance  
Automation Target: `appview/internal/api/search_request_test.go`, `appview/internal/api/search_store_test.go`, `app/test/search/data/search_api_client_test.dart`, `app/test/search/providers/hashtag_search_provider_test.dart`

```gherkin
Feature: Hashtag search modes
  Scenario: Submitted Hashtags tab uses substring query while exact hashtag navigation uses exact tag matching
    Given hashtag entities include exact, prefix, and substring matches
    And top-level regular posts, top-level project posts, replies, and text-only hashtag mentions are indexed
    When the committed hashtag-query endpoint is requested with q, limit, and cursor
    Then it returns hashtag result items ranked exact match first, prefix matches next, then 28-day count descending, then tag ascending
    And it returns an opaque next cursor when more hashtag items exist
    When exact hashtag results are requested for a selected hashtag with optional leading # or mixed casing
    Then the tag is normalized safely
    And only top-level regular posts and project posts with that exact canonical tag are returned
    And replies, comments, substring tags, and text-only visual hashtag mentions are excluded
    When the exact hashtag feed is requested with chronological and popular sort choices
    Then each response preserves exact-tag matching and orders the combined feed by the requested sort
    And unsupported exact hashtag sort values return the standard validation error
```

### AT-006: Project browse/filtering stays under the Projects API and data layer

Requirement IDs: BR-005, FR-009, FR-010, FR-011, RULE-005  
Acceptance Criteria: AC-009, AC-010, AC-019  
Priority: Must  
Level: Acceptance  
Automation Target: `appview/internal/api/search_request_test.go`, `appview/internal/api/search_store_test.go`, `app/test/projects/data/project_api_client_test.dart`, `app/test/projects/providers/project_feed_provider_test.dart`

```gherkin
Feature: Project browsing boundary
  Scenario: Project filters are served by /v1/projects rather than search UI endpoints
    Given future Projects UI code supplies craft type, filter families, sort, limit, and cursor
    When Flutter reads the project browse provider
    Then Flutter calls the project repository/data layer and `/v1/projects`
    And AppView returns paginated project posts matching the browse filters
    And chronological and popular sorting are supported
    And popular sorting uses the existing deterministic engagement plus recency-decay formula
    And `/v1/search/projects` remains for committed text-search Projects results only
    And rich browse filters sent to `/v1/search/projects` are rejected or documented as removed from that public contract
```

### AT-007: Recent/saved searches are one explicit private history surface

Requirement IDs: BR-006, FR-013, FR-016, RULE-001, RULE-002, RULE-009  
Acceptance Criteria: AC-011, AC-015, AC-020  
Priority: Must  
Level: Acceptance  
Automation Target: `appview/internal/api/search_request_test.go`, `appview/internal/api/search_recent_store_test.go`, `app/test/search/models/recent_search_test.dart`, `app/test/search/providers/recent_searches_provider_test.dart`

```gherkin
Feature: Private search history
  Scenario: Recents change only through explicit save/delete mutations and use future Search payloads
    Given a signed-in user has an AppView-backed recent-search list
    When typeahead suggestions, exact hashtag results, and submitted result fetches are performed
    Then recent searches are unchanged
    When explicit save mutations are sent for a free-text query, selected hashtag, and selected profile
    Then the query payload contains q only
    And the hashtag payload contains canonical tag only
    And the profile payload contains stable selected-profile identity for direct navigation
    And project browse/filter interactions do not generate Search recents in this slice
    And no separate saved-search table, route family, Flutter repository/provider, PDS record, or local persistent Flutter history store is added
```

### AT-008: Craft-type inputs canonicalize to full tokens and responses expose full tokens

Requirement IDs: BR-002, FR-014, FR-015  
Acceptance Criteria: AC-013  
Priority: Must  
Level: Acceptance  
Automation Target: `appview/internal/api/search_request_test.go`, `appview/internal/api/search_store_test.go`, `app/test/search/models/top_hashtags_test.dart`, `app/test/projects/options/project_option_catalogs_test.dart`

```gherkin
Feature: Craft token normalization
  Scenario: Project and top-hashtag APIs accept aliases but return canonical full craft tokens
    Given project records store `social.craftsky.feed.defs#...` craft tokens
    And callers provide a mix of full tokens and supported bare aliases
    When project browse, project search, and top-hashtag requests are handled
    Then comparisons use canonical full craft tokens
    And duplicate equivalent inputs do not duplicate results
    And responses expose canonical full craft tokens for craft groups
    And supported default craft groups are returned, including empty groups when requested or included by default
```

### AT-009: New and changed AppView APIs follow /v1 auth, validation, and cursor conventions

Requirement IDs: BR-004, FR-001, FR-005, FR-006, FR-009, NFR-002, NFR-003  
Acceptance Criteria: AC-014  
Priority: Must  
Level: Acceptance  
Automation Target: `appview/internal/routes/routes_test.go`, `appview/internal/api/search_request_test.go`, `app/test/search/data/search_api_client_error_test.dart`

```gherkin
Feature: AppView API conventions
  Scenario: Search, suggestion, and project endpoints use existing authenticated /v1 behavior
    Given a new or changed `/v1/*` search, suggestion, recent, or project endpoint
    When the endpoint is called without an authenticated session or without `X-Craftsky-Device-Id`
    Then AppView returns the existing authentication or missing-device error envelope
    When the endpoint is called with invalid limits, invalid cursors, malformed tags, or unsupported parameters
    Then AppView returns the standard camelCase error envelope with requestId
    When Flutter calls valid endpoints
    Then it uses the shared authenticated Dio stack and preserves opaque cursors without parsing them
```

### AT-010: Focused tests prove the refined contracts or document explicit gaps

Requirement IDs: NFR-004, NFR-005  
Acceptance Criteria: AC-016  
Priority: Should  
Level: Acceptance / Regression  
Automation Target: AppView and Flutter focused suites listed in Section 11

```gherkin
Feature: Test coverage for the refinement slice
  Scenario: Search, facet, and project focused tests cover high-risk contract changes
    Given the slice is complete
    When focused AppView and Flutter tests run for search, facets, and projects
    Then tests cover ranking consistency, craft-token normalization, hashtag-query pagination, project browse filters, provider pagination, and compatibility regressions
    And any remaining performance or UI coverage limitations are documented as explicit gaps rather than treated as covered
```

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-001, NFR-003, RULE-007 | AC-002, AC-014, AC-017 | Parse and validate unified suggestion requests. | `q`, optional type selection, per-type limits, empty query, over-limit values, invalid cursor parameter. | Non-empty `q` is trimmed; type/limit values are bounded; no cursor is accepted; invalid input maps to standard validation errors. | `appview/internal/api/search_request_test.go` |
| UT-002 | FR-002, RULE-003 | AC-003, AC-004, AC-017 | Rank profile suggestions through one shared helper and include crafts in summary data. | Viewer follows, Craftsky profiles, non-Craftsky profile, prefix/non-prefix handle matches, crafts arrays. | Search and facet callers receive equivalent ranking for overlaps; non-Craftsky profiles are excluded where required; summaries carry craft tokens. | `appview/internal/api/facet_suggestion_test.go`, `appview/internal/api/search_profile_rank_test.go` |
| UT-003 | FR-003, FR-005, RULE-004 | AC-002, AC-004, AC-005 | Normalize and rank hashtag suggestions/query results. | Rows with mixed-case tags, leading `#`, duplicate tags, negative counts, exact/prefix/substring matches, old vs recent post counts. | Tags lower-case/canonicalize; counts aggregate distinct visible top-level regular/project posts in the last 28 days; ranking is exact, prefix, count desc, tag asc. | `appview/internal/api/facet_suggestion_test.go`, `appview/internal/api/search_ranking_test.go` |
| UT-004 | FR-008, NFR-003, RULE-004 | AC-008, AC-014 | Validate exact hashtag path/query normalization separately from hashtag-query substring matching. | `#SockKAL`, `SockKAL`, empty, only `#`, spaces, slashes, control characters, overlong tags. | Valid values normalize to canonical tag; invalid values return standard validation errors and are safe for one path segment. | `appview/internal/api/search_request_test.go`, `app/test/search/data/search_api_client_test.dart` |
| UT-005 | FR-009, FR-014, FR-015 | AC-009, AC-013 | Canonicalize supported craft-type aliases to full lexicon tokens. | Full tokens and bare aliases for knitting/crochet/sewing/embroidery/quilting, duplicate equivalent inputs, unknown craft types. | Supported inputs compare as full tokens; duplicate equivalents de-dupe; unknown values reject rather than widening to all projects; responses use full tokens. | `appview/internal/api/search_request_test.go`, `app/test/projects/options/project_option_catalogs_test.dart` |
| UT-006 | FR-006, FR-007, RULE-006, RULE-008 | AC-006, AC-007, AC-018 | Score submitted post/project text relevance and apply disjoint tab filters. | Regular posts and project posts with title/text/material matches, varying creation times and engagement. | Higher textual relevance beats newer weak matches; deterministic tie-breakers apply; Posts excludes project posts; Projects includes only project posts. | `appview/internal/api/search_ranking_test.go`, `appview/internal/api/search_store_test.go` |
| UT-007 | BR-006, FR-013, FR-016, RULE-001, RULE-009 | AC-011, AC-015, AC-020 | Validate refined recent-search save payloads. | `query` with `q`, `hashtag` with `tag`, `profile` with DID/handle/display metadata, blank/overlong values, legacy project filter payload from Flutter. | Query payload stores q only; hashtag stores canonical tag only; profile stores stable identity; invalid values reject; Flutter does not serialize project browse/filter recents. | `appview/internal/api/search_request_test.go`, `app/test/search/models/recent_search_test.dart` |
| UT-008 | FR-009, FR-010, NFR-003, RULE-005 | AC-009, AC-010, AC-019 | Parse project browse filters under `/v1/projects` and reject browse filters under `/v1/search/projects`. | Craft type, color, material, design tag, project tag, pattern difficulty, project type, sort, limit, cursor, unsupported keys. | `/v1/projects` accepts supported browse filters and cursors; `/v1/search/projects` accepts text-search params only and rejects/removes rich browse filter params per final contract. | `appview/internal/api/search_request_test.go` |
| UT-009 | FR-012, FR-015, NFR-002 | AC-001, AC-002, AC-005, AC-006, AC-008, AC-012, AC-013, AC-014 | Map Flutter search/project models for suggestions, hashtag-result pages, full craft tokens, and selected-entity recents. | Mock JSON for suggestions, hashtag query page, top hashtag groups with full tokens, profile crafts, recent payloads. | Models decode camelCase fields, keep cursors opaque, preserve full tokens, and remain UI-agnostic; clients use shared AppView Dio. | `app/test/search/models/*_test.dart`, `app/test/search/data/search_api_client_test.dart` |
| UT-010 | FR-006, FR-012 | AC-006, AC-016 | Keep Flutter provider pagination independent, duplicate-safe, and concurrency-safe. | Separate query objects for posts/projects/profiles/hashtags, load-more calls with same/different cursors, duplicate items, concurrent loads. | Each tab owns state/cursor; loadMore appends unique items; no-op at end; duplicate concurrent load is suppressed. | `app/test/search/providers/*_provider_test.dart` |
| UT-011 | BR-002, FR-015 | AC-012, AC-013 | Build top-hashtag craft groups from project posts only and include empty supported groups. | Project posts for some crafts, regular posts with same hashtags, hidden/deleted data, no rows for one default craft. | Counts are distinct visible project-post counts by craft for last 28 days; default supported groups include empty `items`; response craftType values are full tokens. | `appview/internal/api/search_store_test.go`, `app/test/search/models/top_hashtags_test.dart` |
| UT-012 | FR-013, RULE-002 | AC-011 | Ensure non-mutating search/suggestion repository paths do not save recents. | Typeahead fetch, post/project/profile/hashtag fetch, exact hashtag fetch, explicit save. | Fetch paths do not call save; only explicit recent save/delete mutations change recent state. | `app/test/search/data/search_repository_test.dart`, `app/test/search/providers/recent_searches_provider_test.dart` |
| UT-013 | FR-008, FR-012, NFR-003, RULE-004 | AC-008, AC-014, AC-021 | Parse and propagate exact hashtag result sort choices without changing exact-match normalization. | Exact hashtag requests with omitted sort, `sort=chronological`, `sort=popular`, invalid sort, limit, cursor, leading-`#` tag, mixed-case tag. | Omitted sort uses the existing/default exact-hashtag ordering; chronological and popular are accepted; invalid sort returns standard validation; normalized tag and opaque cursor handling are preserved. | `appview/internal/api/search_request_test.go`, `app/test/search/data/search_api_client_test.dart`, `app/test/search/models/search_sort_test.dart` |

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | FR-001, RULE-007 | AC-002, AC-014, AC-017 | Unified suggestions endpoint returns grouped top-N sections with `hasMore`. | Seed viewer, followed/unfollowed Craftsky profiles, hashtagged regular/project posts, more matches than limit. | `GET /v1/search/suggestions?q=sock&types=profiles,hashtags&profileLimit=2&hashtagLimit=2`. | 200 response has profile/hashtag sections, no cursor, accurate `hasMore`, camelCase fields, normalized tags, bounded items. | `appview/internal/routes/routes_test.go`, `appview/internal/api/search_store_test.go` |
| IT-002 | FR-002, FR-004, RULE-003 | AC-003, AC-004 | Search profile suggestions and facet mentions share ranking and crafts. | Seed same viewer/profile/follow/crafts data for both routes. | Call unified suggestions and `/v1/facets/mentions` with same query. | Overlapping profile order is equivalent; search suggestion/search summary includes crafts; facet response remains compatible. | `appview/internal/api/facet_test.go`, `appview/internal/api/search_store_test.go` |
| IT-003 | FR-003, FR-004 | AC-002, AC-004, AC-005 | Hashtag suggestion core powers search and facet compatibility. | Seed regular and project posts with hashtag duplicates, hidden rows, old rows, and mixed-case tags. | Call unified suggestions and `/v1/facets/hashtags`. | Both use normalized/count logic; facet callers still decode expected fields; search suggestions include per-section `hasMore`. | `appview/internal/api/facet_test.go`, `appview/internal/api/search_store_test.go` |
| IT-004 | FR-005, NFR-004, RULE-004 | AC-005, AC-014, AC-016 | Committed hashtag-query endpoint ranks and paginates deterministically. | Seed exact, prefix, substring, equal-count tags; enough rows for multiple pages. | `GET /v1/search/hashtags?q=sock&limit=2`, then with returned cursor. | Page 1/2 ordering follows exact, prefix, count desc, tag asc; cursor is opaque; invalid cursor returns standard error. | `appview/internal/api/search_store_test.go`, `appview/internal/routes/routes_test.go` |
| IT-005 | FR-006, FR-007, RULE-006, RULE-008 | AC-006, AC-007, AC-018 | Submitted post/project search is relevance-first and disjoint. | Seed regular post, project title/material matches, newer weak match, hidden/reply rows. | Call `/v1/search/posts?q=alpaca` and `/v1/search/projects?q=alpaca` without sort override. | Posts returns only non-project top-level matches; Projects returns only project posts; ranking is relevance-first with stable tie-breakers. | `appview/internal/api/search_store_test.go` |
| IT-006 | FR-008, NFR-003, RULE-004 | AC-008, AC-014 | Exact hashtag feed returns combined top-level regular/project posts only. | Seed exact regular post, exact project post, substring tag, reply/comment, text-only visual hashtag. | `GET /v1/search/hashtags/{tag}/posts` with mixed-case/leading-hash input. | Response hashtag is canonical; exact regular/project posts returned; replies/comments/substrings/text-only mentions excluded; invalid path values error. | `appview/internal/api/search_store_test.go`, `app/test/search/data/search_api_client_test.dart` |
| IT-007 | BR-002, FR-014, FR-015 | AC-012, AC-013 | Top hashtags use canonical full craft groups and include empty defaults. | Seed project posts with full craft tokens, bare alias requests, default craft with no tags. | Call `/v1/search/hashtags/top` with no craftTypes and with mixed full/bare craftTypes. | Responses expose full craft tokens; supported default groups are included; counts use distinct visible project posts from last 28 days. | `appview/internal/api/search_store_test.go`, `app/test/search/data/search_api_client_test.dart` |
| IT-008 | FR-009, FR-010, FR-014, RULE-005 | AC-009, AC-010, AC-013, AC-014 | `/v1/projects` supports browse filters, full/bare craft inputs, pagination, chronological/popular sort. | Seed projects across craft tokens, filter families, engagement, and dates. | Call `/v1/projects` with craftType, filter families, sort, limit, cursor. | Matching project posts page correctly; popular formula reused; full token and bare alias inputs match same rows; cursors and validation follow `/v1/` conventions. | `appview/internal/api/search_store_test.go`, `appview/internal/routes/routes_test.go` |
| IT-009 | FR-010, RULE-005 | AC-009, AC-019 | `/v1/search/projects` is text-search-only after filters move to `/v1/projects`. | Use request parser/route with rich browse filter params. | Call `/v1/search/projects?q=sock&craftType=...&material=...`. | Unsupported browse filters are rejected or documented as removed; text-only Projects search remains available. | `appview/internal/api/search_request_test.go`, `appview/internal/routes/routes_test.go` |
| IT-010 | BR-006, FR-013, FR-016, RULE-001, RULE-002, RULE-009 | AC-011, AC-015, AC-020 | Recent-search persistence supports refined private payloads and no automatic mutations. | Existing recent-search table, viewer DID, explicit save/delete calls, intervening suggestion/result fetches. | Save/list/delete `query`, `hashtag`, and selected `profile` recents; perform suggestion/result fetches before and after. | List changes only after explicit saves/deletes; payloads normalize as required; records scoped to viewer; no saved-search table or PDS write path exists. | `appview/internal/api/search_recent_store_test.go`, `appview/internal/api/search_request_test.go` |
| IT-011 | NFR-002, NFR-003 | AC-014 | New/changed routes enforce auth/device headers, bounded limits, and error envelopes. | Route mux with auth middleware and representative new endpoints. | Call endpoints without auth/device, with invalid limits/cursors/unsupported params, and with valid requests. | 401/400 validation behavior matches existing `/v1/` conventions; error envelope is `{error, message, requestId}` with camelCase JSON. | `appview/internal/routes/routes_test.go`, `appview/internal/api/search_request_test.go` |
| IT-012 | FR-001, FR-005, FR-012, FR-016, NFR-002 | AC-002, AC-005, AC-006, AC-008, AC-014, AC-020 | Flutter SearchApiClient and repository cover new contracts. | Mock Dio with ErrorMappingInterceptor and AppView base URL. | Call unified suggestions, hashtag-query results, exact hashtag posts, top hashtags, recents save/list/delete, and submitted result methods. | Correct paths/query params/body shape; full tokens/crafts decoded; cursors opaque; API errors mapped; no PDS client involved. | `app/test/search/data/search_api_client_test.dart`, `app/test/search/data/search_repository_test.dart` |
| IT-013 | BR-001, BR-002, FR-006, FR-012, RULE-008 | AC-006, AC-012, AC-016, AC-018 | Flutter search providers expose UI-agnostic blank-search and independent tab states. | ProviderContainer with fake search repository. | Read blank-search, posts, projects, profiles, hashtag-query, and exact hashtag providers; call loadMore per tab. | Providers delegate to intended repository methods; pagination is independent; state is UI-agnostic; default text search uses relevance-oriented query params. | `app/test/search/providers/*_provider_test.dart` |
| IT-014 | BR-005, FR-011, RULE-005, RULE-009 | AC-010, AC-011, AC-014 | Flutter project browse stays in project feature data layer. | Mock project repository/client and search repository spy. | Read project feed/provider with craft type, filters, sort, cursor; perform project browse interactions. | Project provider calls project repository/client and `/v1/projects`; search repository is not called; no project browse recent is serialized. | `app/test/projects/data/project_api_client_test.dart`, `app/test/projects/providers/project_feed_provider_test.dart` |
| IT-015 | FR-004 | AC-004 | Existing Flutter rich-text facet repositories remain compatible. | Mock `/v1/facets/mentions`, `/v1/facets/mentions/resolve`, `/v1/facets/hashtags`. | Run existing rich-text repository/controller tests. | Current mention/hashtag decode behavior and tolerant error behavior remain passing. | `app/test/shared/rich_text/facet_suggestion_repository_test.dart`, `app/test/shared/rich_text/*autocomplete*_test.dart` |
| IT-016 | NFR-004, NFR-005 | AC-016 | Migration/index/query checks are covered where implementation adds persistence/index changes. | Any migration for recent type constraint and any supporting indexes for hashtag/project filters. | Run AppView migration/test suite and focused store tests against Postgres. | Schema permits refined recents; queries remain bounded; any performance limitation is documented as a gap. | `appview/migrations/*`, `appview/internal/api/*_test.go` |
| IT-017 | FR-008, FR-012, NFR-003, RULE-004 | AC-008, AC-021 | Exact hashtag feed sorts the combined regular/project feed by chronology or popularity. | Seed at least three exact-tag top-level posts/projects with different creation times and deterministic engagement/recency popularity scores, plus substring/reply rows that must remain excluded. | Call `/v1/search/hashtags/{tag}/posts?sort=chronological` and `?sort=popular`, paginate at least one sort, and call with an invalid sort. | Chronological order follows creation time with stable tie-breakers; popular order follows the existing deterministic popularity formula; exact matching and exclusions are unchanged; cursors are sort-specific/opaque; invalid sort returns standard validation. | `appview/internal/api/search_store_test.go`, `appview/internal/routes/routes_test.go`, `app/test/search/data/search_api_client_test.dart`, `app/test/search/providers/hashtag_search_provider_test.dart` |

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Acceptance Criteria | Test |
|---|---|---|---|---|
| REG-001 | Composer/profile rich-text autocomplete keeps `/v1/facets/mentions`, `/v1/facets/mentions/resolve`, and `/v1/facets/hashtags` compatibility. | BR-003, FR-002, FR-003, FR-004, RULE-003 | AC-003, AC-004 | Re-run existing AppView facet tests and Flutter rich-text repository/controller tests after shared suggestion refactor. |
| REG-002 | Exact hashtag result feed excludes substring tags, replies/comments, and text-only hashtag mentions while including regular and project top-level posts and preserving supported chronology/popularity sorts. | FR-008, FR-012, NFR-003, RULE-004 | AC-008, AC-021 | Keep/extend existing `SearchHashtagPosts` tests with project post, invalid tag, chronological sort, and popular sort cases. |
| REG-003 | Submitted Posts and Projects tabs do not duplicate project posts. | FR-007, RULE-006 | AC-007 | Seed a project post matching the query and assert it appears only in `/v1/search/projects`, not `/v1/search/posts`. |
| REG-004 | Recent-search list/save/delete remains explicit, private, viewer-scoped, and idempotent on already-deleted/not-owned IDs. | BR-006, FR-013, FR-016, RULE-001, RULE-002, RULE-009 | AC-011, AC-015, AC-020 | Extend recent store tests to cover refined payloads and existing delete semantics. |
| REG-005 | SearchPage/ProjectsPage rendered UI stubs and route/navigation behavior are not expanded by this non-UI slice. | BR-001, FR-012, NFR-001 | AC-001 | Keep UI tests limited to compile/stub compatibility and manually inspect source diff for no rendered search/project UI. |
| REG-006 | Flutter reads continue through authenticated AppView HTTP and do not call PDS or persist PDS tokens/local search history. | NFR-002, RULE-001 | AC-014, AC-015 | Search/project clients use shared Dio providers; no direct PDS dependency or local persistent history storage is introduced. |
| REG-007 | Project browse/filtering belongs to `app/lib/projects/**`, not `app/lib/search/**`, while text search Projects stays under search. | BR-005, FR-010, FR-011, RULE-005 | AC-009, AC-010, AC-019 | Provider/client tests assert project browse calls project repository and rich filters are not exposed through search-projects. |

## 7. Test Data

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Profile suggestion ranking and crafts | Viewer DID, followed and unfollowed Craftsky profiles, one non-Craftsky profile, handles/display names/descriptions matching query, craft token arrays such as `social.craftsky.feed.defs#knitting`. | AT-002, AT-003, UT-002, IT-001, IT-002 |
| TD-002 | Hashtag suggestion/query/exact modes | Tags `sock`, `sockkal`, `sockmending`, mixed-case duplicates, prefix/substrings, old rows outside 28 days, hidden rows, regular posts, project posts, replies/comments. | AT-002, AT-005, UT-003, UT-004, IT-003, IT-004, IT-006, REG-002 |
| TD-003 | Submitted text-search relevance and disjoint tabs | Regular post text match, project title/material match, newer weak match, project post that also matches post text, hidden/reply rows. | AT-004, UT-006, IT-005, REG-003 |
| TD-004 | Project browse filters and craft tokens | Projects across full craft tokens for knitting, crochet, sewing, embroidery, quilting; bare alias requests; color/material/designTag/projectTag/patternDifficulty/projectType values; empty default craft group. | AT-006, AT-008, UT-005, UT-008, UT-011, IT-007, IT-008, IT-009 |
| TD-005 | Recent-search payloads | `query` recent with q only, `hashtag` recent with canonical tag, selected `profile` recent with DID/handle/display metadata, invalid blank/overlong payloads, legacy post/project payloads for compatibility decisions. | AT-007, UT-007, UT-012, IT-010, IT-012, REG-004 |
| TD-006 | API validation and auth errors | Missing auth, missing device ID, invalid limits, invalid cursors, unsupported filter keys, malformed hashtags, whitespace-only q. | AT-009, UT-001, UT-004, UT-008, IT-011 |
| TD-007 | Flutter mocked HTTP/provider state | DioAdapter responses with camelCase JSON, opaque cursors containing punctuation, fake repositories with duplicate items and delayed futures. | IT-012, IT-013, IT-014, UT-009, UT-010 |
| TD-008 | Existing facet compatibility | Current `/v1/facets/mentions`, `/v1/facets/mentions/resolve`, and `/v1/facets/hashtags` response shapes expected by rich-text code. | AT-003, IT-015, REG-001 |
| TD-009 | Exact hashtag sort ordering | Exact-tag regular posts and project posts with controlled `createdAt`, like/repost/reply counts, and recency values, plus substring/reply/comment rows that should not affect chronology or popularity results. | AT-005, UT-013, IT-017, REG-002 |

## 8. Manual Checks

| ID | Requirement IDs | Acceptance Criteria | Check | Steps | Expected Result |
|---|---|---|---|---|---|
| MAN-001 | BR-001, FR-012, NFR-001 | AC-001 | Verify no rendered UI or visual navigation behavior was added. | Review the implementation diff for `app/lib/search/pages/**`, `app/lib/projects/pages/**`, `app/lib/router/**`, and widget/card/layout files. | Only compile/build-compatible stubs or non-rendered data wiring changed; no search UI, project UI, tab UI, management page, card, or visual route behavior is implemented. |
| MAN-002 | NFR-002, RULE-001 | AC-014, AC-015 | Verify AppView-only reads and private recents architecture. | Inspect Flutter clients/providers and AppView recents code for direct PDS reads, PDS token/device storage, PDS recent records, or local persistent search-history storage. | Search/project/facet reads use shared authenticated AppView Dio; recents remain AppView Postgres state only; no PDS/local persistent history path is introduced. |
| MAN-003 | FR-009, NFR-004 | AC-009, AC-016 | Review bounded/index-aware query shape for hashtag query and project browse filters. | Inspect SQL/query plans or run `EXPLAIN` against representative seeded data if implementation adds new queries/indexes. | Queries enforce bounded limits, use materialized/indexed columns where practical, and any missing index/performance follow-up is documented. |
| MAN-004 | BR-001-BR-006, FR-001-FR-016, NFR-001-NFR-005, RULE-001-RULE-009 | AC-016 | Perform document review before implementation or record an explicit skip decision. | Have a reviewer compare `01-requirements.md` and this test specification for coverage, ambiguity, and overfit. | Review feedback is folded into this document, or the user explicitly elects to skip document review before coding plan. |

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | No rendered UI end-to-end tests are possible in this slice. | BR-001, FR-012, NFR-001 | Rendered search/project UI is explicitly out of scope, so full user interaction tests for tabs, cards, recent-management screens, and filter controls must wait. | Cover provider/API contracts now; add widget/E2E tests in the later UI slice. |
| GAP-002 | Production-scale performance cannot be fully proven by focused test data. | FR-005, FR-009, NFR-004 | Store tests can prove bounded deterministic behavior, but real hashtag/project filter cardinality may need profiling. | Add indexes if query plans require them; record any deferred performance work in implementation notes. |
| GAP-003 | Route names are assumed from requirements recommendations. | FR-001, FR-005 | Requirements recommend `/v1/search/suggestions` and `/v1/search/hashtags`; implementation could choose equivalent names only if requirements are revised first. | Keep route names as specified unless the user explicitly revises requirements and this test spec. |
| GAP-004 | Profile recent display freshness is not solved here. | FR-016 | Requirements allow selected-profile recents to store stable identity plus display metadata; refreshing stale handle/display labels is future work. | Test stable DID/direct navigation now; add profile hydration tests only if future requirements demand fresh display data. |
| GAP-005 | Legacy `post`/`project` recent payload migration policy is not fully prescribed. | FR-016, RULE-001, RULE-009 | Requirements require future Flutter Search recents to generate `query`, `hashtag`, and selected `profile`, while existing pre-UI payloads may be retained or migrated intentionally. | Implementation should choose retain-or-migrate behavior deliberately and test it; do not generate project browse/filter recents from Flutter. |

Blocking gaps: None identified for test design.

## 10. Out Of Scope

- Rendered search UI, search tabs, search box widgets, recent-management screens, project tab UI, project filters UI, cards, layouts, scroll behavior, or visual navigation changes.
- atproto lexicon changes and any PDS persistence for recents/saved searches.
- Public unauthenticated search endpoints.
- Semantic search, embeddings, typo tolerance, recommendations, analytics, telemetry, push notifications, or background polling.
- Full production load testing; this spec targets bounded/index-aware automated tests plus manual query review.
- Changing requirements or route names without an explicit requirements revision.

## 11. Handoff To Document Review

- Requirements file: `01-requirements.md`
- Test specification: `02-acceptance-tests.md`
- Next review artifact: `03-document-review.md`
- External Plannotator review, if the user initiates it outside this agent: `docs/changes/2026-06-21-search-refinements/`
- Recommended first failing test for implementation: `UT-001` for parsing/validating `GET /v1/search/suggestions` as a bounded, non-paginated, authenticated top-N suggestion request. This is the smallest failing test that anchors the new shared suggestion contract.
- Suggested test order for implementation:
  1. AppView request/normalization/payload unit tests: `UT-001`, `UT-004`, `UT-005`, `UT-007`, `UT-008`, `UT-013`.
  2. Shared ranking/counting unit tests: `UT-002`, `UT-003`, `UT-006`, `UT-011`.
  3. AppView store/route integration tests: `IT-001` through `IT-011`, `IT-017`, plus `IT-016` if migrations/indexes are added.
  4. Flutter model/API-client/repository/provider tests: `UT-009`, `UT-010`, `UT-012`, `IT-012`, `IT-013`, `IT-014`.
  5. Compatibility and boundary regressions: `IT-015`, `REG-001` through `REG-007`, then manual checks `MAN-001` through `MAN-004`.
- Commands discovered:
  - Start dependencies: `just dev-d`
  - Full AppView tests: `just test`
  - Focused AppView tests: `cd appview && go test ./internal/api ./internal/routes -count=1`
  - Flutter code generation after provider/model changes: `cd app && dart run build_runner build --delete-conflicting-outputs`
  - Focused Flutter tests: `cd app && flutter test test/search test/shared/rich_text test/projects`
  - Flutter broader checks: `cd app && flutter analyze` and `cd app && flutter test`
- Blocking gaps: None identified.
- Review recommendation before implementation: Medium-risk document review is recommended because API, persistence, ranking, provider, and project/search boundary decisions must remain aligned. The user may explicitly skip document review, but the skip should be recorded before moving to coding plan.
