# Acceptance Test Specification: Timeline Feed AppView

## 1. Test Strategy

This AppView-only change should be driven by Go tests around three boundaries:

- **Store/query integration tests** against isolated Postgres schemas for timeline eligibility, ordering, pagination, and deduplication. This is the highest-value layer because most risk lives in SQL selection over `craftsky_posts` and `atproto_follows`.
- **Handler tests** using fakes for the timeline reader and handle resolver to verify auth context usage, response shape, engagement hydration, invalid cursor mapping, unknown query params, empty pages, no total count, no PDS calls, and identity failures.
- **Route tests** to verify `GET /v1/feed/timeline` is registered under `/v1/` and protected by the existing authenticated-device middleware.

Unit tests should cover pure helper behavior such as eligibility classification and cursor parsing if those helpers are introduced. Regression tests should protect existing post/profile response behavior and existing post list routes. Manual checks are optional and limited to a smoke request once the endpoint exists.

Risk level carried from requirements: **Medium**. The endpoint is additive, but feed semantics are foundational for later Flutter work.

## 2. Requirement Coverage Matrix

| Requirement ID | Acceptance Criteria | Test IDs | Test Level | Automated? |
|---|---|---|---|---|
| BR-001 | AC-001, AC-002, AC-005 | AT-001, AT-002, IT-001, IT-002, IT-003, IT-004 | Acceptance / Integration | Yes |
| BR-002 | AC-010 | AT-006, UT-004, REG-004 | Acceptance / Unit / Regression | Yes |
| FR-001 | AC-001 | AT-001, IT-008, IT-009 | Acceptance / Integration | Yes |
| FR-002 | AC-002, AC-003, AC-014 | AT-002, IT-001, IT-005, IT-006 | Acceptance / Integration | Yes |
| FR-003 | AC-004 | AT-003, IT-001, IT-007 | Acceptance / Integration | Yes |
| FR-004 | AC-004 | AT-003, UT-001, IT-007 | Acceptance / Unit / Integration | Yes |
| FR-005 | AC-005, AC-006 | AT-004, IT-002, IT-003 | Acceptance / Integration | Yes |
| FR-006 | AC-006, AC-008, AC-016 | AT-004, AT-005, IT-003, IT-010 | Acceptance / Integration | Yes |
| FR-007 | AC-006, AC-009, AC-017 | AT-004, AT-007, IT-003, IT-011, IT-012 | Acceptance / Integration | Yes |
| FR-008 | AC-007 | AT-008, IT-010, REG-001, REG-002 | Acceptance / Integration / Regression | Yes |
| FR-009 | AC-011, AC-018 | IT-010, IT-013 | Integration | Yes |
| FR-010 | AC-009 | AT-007, IT-011 | Acceptance / Integration | Yes |
| FR-011 | AC-008 | AT-005, IT-004 | Acceptance / Integration | Yes |
| FR-012 | AC-012 | AT-009, IT-014 | Acceptance / Integration | Yes |
| FR-013 | AC-015 | IT-015 | Integration | Yes |
| NFR-001 | AC-001, AC-006, AC-009, AC-012 | AT-001, AT-004, AT-007, AT-009, IT-008, IT-009, IT-011, IT-014 | Acceptance / Integration | Yes |
| NFR-002 | AC-013 | IT-016, MAN-001 | Integration / Manual | Partial |
| NFR-003 | AC-010 | UT-004, REG-004 | Unit / Regression | Yes |
| RULE-001 | AC-002, AC-003, AC-014 | IT-001, IT-005, IT-006 | Integration | Yes |
| RULE-002 | AC-004 | UT-001, IT-007 | Unit / Integration | Yes |
| RULE-003 | AC-004, AC-007 | IT-001, IT-010, REG-001 | Integration / Regression | Yes |
| RULE-004 | AC-004 | IT-007 | Integration | Yes |
| RULE-005 | AC-014 | IT-006 | Integration | Yes |

## 3. Acceptance Scenarios

### AT-001: Timeline Requires Authenticated Device
Requirement IDs: BR-001, FR-001, NFR-001  
Acceptance Criteria: AC-001  
Priority: Must  
Level: Acceptance  
Automation Target: `appview/internal/routes/routes_test.go`

```gherkin
Feature: Timeline feed AppView API
  Scenario: Authenticated-device protection
    Given the AppView routes are registered
    When a client requests GET /v1/feed/timeline without authentication
    Then the response is 401 using the existing auth error behavior
    When a client requests GET /v1/feed/timeline with auth but without X-Craftsky-Device-Id
    Then the response is 400 with error "missing_device_id"
```

### AT-002: Timeline Returns Own And Followed Authors Only
Requirement IDs: BR-001, FR-002, RULE-001  
Acceptance Criteria: AC-002, AC-003  
Priority: Must  
Level: Acceptance  
Automation Target: `appview/internal/api/timeline_store_test.go`

```gherkin
Feature: Timeline author eligibility
  Scenario: Own and followed author rows are eligible
    Given the authenticated viewer has an eligible own post
    And the viewer actively follows author A
    And the viewer does not follow author B
    And authors A and B have eligible indexed posts
    When the viewer requests the timeline
    Then the response includes the viewer's own post
    And the response includes author A's eligible post
    And the response excludes author B's post
```

### AT-003: Timeline Excludes Conversation And Repost Activity
Requirement IDs: FR-003, FR-004, RULE-002, RULE-003, RULE-004  
Acceptance Criteria: AC-004  
Priority: Must  
Level: Acceptance  
Automation Target: `appview/internal/api/timeline_store_test.go`

```gherkin
Feature: Timeline item eligibility
  Scenario: Only top-level post rows appear
    Given an eligible author has a root post, project post, quote post, top-level comment, nested reply, and repost record
    When the viewer requests the timeline
    Then the root post, project post, and quote post are returned
    And the top-level comment, nested reply, and repost record are not returned
```

### AT-004: Timeline Pagination Is Stable And Opaque
Requirement IDs: BR-001, FR-005, FR-006, FR-007, NFR-001  
Acceptance Criteria: AC-005, AC-006  
Priority: Must  
Level: Acceptance  
Automation Target: `appview/internal/api/timeline_store_test.go`

```gherkin
Feature: Timeline pagination
  Scenario: Pages continue after the previous page
    Given more eligible timeline rows exist than the requested limit
    And at least two rows share the same indexed_at timestamp
    When the viewer requests the first page
    Then items are ordered by indexed_at descending and URI descending
    And a cursor is returned
    When the viewer requests the next page with that cursor
    Then the next page continues after the first page without duplicate or skipped eligible rows
```

### AT-005: Empty Timeline Is A Normal Empty Page
Requirement IDs: FR-006, FR-011  
Acceptance Criteria: AC-008  
Priority: Should  
Level: Acceptance  
Automation Target: `appview/internal/api/timeline_test.go`

```gherkin
Feature: Empty timeline
  Scenario: No eligible rows
    Given the viewer has no eligible own rows
    And follows no accounts with eligible rows
    When the viewer requests the timeline
    Then the response status is 200
    And the response body has items as an empty array
    And the response omits cursor
    And the response contains no discovery suggestions
```

### AT-006: Timeline Design Leaves Future Feeds Extensible
Requirement IDs: BR-002, NFR-003  
Acceptance Criteria: AC-010  
Priority: Must  
Level: Acceptance  
Automation Target: `appview/internal/api/timeline_query_test.go` or implementation review checklist

```gherkin
Feature: Timeline extensibility
  Scenario: Timeline query boundary is reusable
    Given the timeline endpoint is implemented
    When the code is reviewed for feed-source boundaries
    Then author eligibility, post eligibility, pagination, and response assembly are separated enough that later project filters, list author sources, or search-backed sources do not require changing the public timeline response contract
```

### AT-007: Invalid Cursor Uses Standard Error Envelope
Requirement IDs: FR-007, FR-010, NFR-001  
Acceptance Criteria: AC-009  
Priority: Must  
Level: Acceptance  
Automation Target: `appview/internal/api/timeline_test.go`

```gherkin
Feature: Timeline cursor validation
  Scenario: Malformed cursor
    Given the viewer is authenticated
    When the viewer requests GET /v1/feed/timeline?cursor=not-a-valid-cursor
    Then the response is 400
    And the JSON error envelope has error "invalid_cursor"
    And the envelope includes message and requestId
```

### AT-008: Timeline Items Use Existing PostResponse Shape
Requirement IDs: FR-008, RULE-003  
Acceptance Criteria: AC-007  
Priority: Must  
Level: Acceptance  
Automation Target: `appview/internal/api/timeline_test.go`

```gherkin
Feature: Timeline item response shape
  Scenario: Post-shaped timeline items
    Given eligible timeline rows include author display data, image data, tags, quote strong references, and engagement state
    When the viewer requests the timeline
    Then every item uses the existing PostResponse field names
    And quote posts expose only quote.uri and quote.cid
    And nested quoted-post content is not expanded
```

### AT-009: Author Identity Failure Fails The Request
Requirement IDs: FR-012, NFR-001  
Acceptance Criteria: AC-012  
Priority: Should  
Level: Acceptance  
Automation Target: `appview/internal/api/timeline_test.go`

```gherkin
Feature: Timeline author identity resolution
  Scenario: Handle resolution fails
    Given the timeline page contains a row for an author whose handle cannot be resolved
    When the viewer requests the timeline
    Then the request fails with the existing identity_unavailable error behavior
```

## 4. Unit Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Inputs | Expected Result | Automation Target |
|---|---|---|---|---|---|---|
| UT-001 | FR-004, RULE-002 | AC-004 | Classify post rows as eligible top-level rows vs conversation rows if a helper is introduced. | Rows with no reply fields; rows with root=parent; rows with root!=parent; rows with one reply field present. | Only rows with both reply fields absent are eligible. | `appview/internal/api/timeline_query_test.go` |
| UT-002 | FR-007, FR-010 | AC-009 | Decode and validate timeline cursor payload if timeline gets its own cursor helper. | Empty cursor, valid `indexedAt`/`uri` cursor, malformed base64, missing keys, bad timestamp. | Empty cursor starts first page; malformed payloads map to `envelope.ErrInvalidCursor`. | `appview/internal/api/timeline_query_test.go` or `appview/internal/api/envelope/cursor_test.go` |
| UT-003 | FR-006, FR-007 | AC-006, AC-016, AC-017 | Verify timeline response struct omits cursor when empty, has no total-count field, and ignores unknown query params at handler boundary. | Empty next cursor; non-empty next cursor; request with `craftType` and `tag`. | Cursor omitted when empty; no total-count key; unknown params do not affect store call. | `appview/internal/api/timeline_test.go` |
| UT-004 | BR-002, NFR-003 | AC-010 | Verify feed query options or interface can represent author source and post eligibility separately if such a struct is introduced. | Timeline/default query options. | Timeline source is not hard-coded into route registration; future source/filter options can be added without changing response shape. | `appview/internal/api/timeline_query_test.go` |
| UT-005 | FR-008, RULE-003 | AC-007 | Verify timeline response assembly uses `BuildPostResponse` and does not expand quoted posts. | PostRow with `QuoteURI`/`QuoteCID`. | Item has `quote` strong ref only and otherwise matches PostResponse shape. | `appview/internal/api/timeline_test.go` or `post_response_test.go` |

## 5. Integration Test Cases

| ID | Requirement IDs | Acceptance Criteria | Description | Setup | Action | Expected Result | Automation Target |
|---|---|---|---|---|---|---|---|
| IT-001 | BR-001, FR-002, FR-003, RULE-001, RULE-003 | AC-002, AC-004 | Store returns own and followed eligible top-level/project/quote posts only. | Isolated Postgres schema with viewer, followed author A, unfollowed author B, craftsky profiles, posts for all authors, and follow viewer→A. | Call timeline store list for viewer. | Viewer and A posts appear; B posts do not. | `appview/internal/api/timeline_store_test.go` |
| IT-002 | BR-001, FR-005 | AC-005 | Store orders by `indexed_at DESC, uri DESC`. | Seed eligible rows with different `indexed_at` values and tied timestamps. | Call timeline store list with large limit. | Rows sorted newest indexed_at first, URI descending for ties. | `appview/internal/api/timeline_store_test.go` |
| IT-003 | FR-005, FR-006, FR-007 | AC-006 | Store paginates with opaque seek cursor. | Seed more eligible rows than limit, including tied timestamps. | Fetch page 1, then page 2 with returned cursor. | Page 2 continues after page 1 without duplicate/skipped rows under same dataset; cursor omitted on final page. | `appview/internal/api/timeline_store_test.go` |
| IT-004 | FR-011 | AC-008 | Empty timeline returns empty page. | Viewer with no own eligible posts and no followed eligible posts. | Call handler or store. | `items: []`, no cursor, 200 at handler layer. | `appview/internal/api/timeline_test.go` and/or `timeline_store_test.go` |
| IT-005 | FR-002, RULE-001 | AC-003 | Current follow graph controls eligibility on every page. | Page 1 fixtures with viewer→A active; then remove or change follow before next request. | Call page 1 then page 2 after graph change. | Page 2 uses current active follow state; no cursor snapshot dependency. | `appview/internal/api/timeline_store_test.go` |
| IT-006 | FR-002, RULE-001, RULE-005 | AC-014 | Own posts are included without self-follow and deduplicated with self-follow. | Viewer own post; run once without self-follow and once with viewer→viewer follow row. | Call timeline store list. | Own post appears in both cases and appears only once. | `appview/internal/api/timeline_store_test.go` |
| IT-007 | FR-003, FR-004, RULE-002, RULE-004 | AC-004 | Comments, nested replies, and repost records are excluded. | Followed author with root post, top-level comment, nested reply, repost interaction, and quote post. | Call timeline store list. | Root/quote rows appear; comment/reply rows and repost record do not. | `appview/internal/api/timeline_store_test.go` |
| IT-008 | FR-001, NFR-001 | AC-001 | Route is registered and requires auth. | Add routes with test deps. | Request `GET /v1/feed/timeline` without auth. | Existing 401 auth behavior. | `appview/internal/routes/routes_test.go` |
| IT-009 | FR-001, NFR-001 | AC-001 | Route requires device ID. | Add routes with test deps. | Request `GET /v1/feed/timeline` with auth but no `X-Craftsky-Device-Id`. | 400 with `missing_device_id`. | `appview/internal/routes/routes_test.go` |
| IT-010 | FR-006, FR-008, FR-009, RULE-003 | AC-007, AC-011, AC-016 | Handler returns PostResponse items with engagement and no total count/PDS fetch. | Fake timeline store rows, fake engagement summary, fake resolver, no PDS dependency. | Request timeline as authenticated viewer. | 200 with `items`; item fields match PostResponse; engagement state present; no `totalCount`; no PDS factory needed/called. | `appview/internal/api/timeline_test.go` |
| IT-011 | FR-007, FR-010, NFR-001 | AC-009 | Invalid cursor maps to standard error envelope. | Fake store returns `envelope.ErrInvalidCursor` or handler decodes bad cursor. | Request `?cursor=garbage`. | 400 `{error:"invalid_cursor", message, requestId}`. | `appview/internal/api/timeline_test.go` |
| IT-012 | FR-007 | AC-017 | Unknown query params are ignored. | Fake store captures limit/cursor and request includes `craftType`, `tag`, `authorList`. | Request timeline. | Store receives only defined limit/cursor effects; response matches unfiltered timeline. | `appview/internal/api/timeline_test.go` |
| IT-013 | FR-009 | AC-018 | Just-created but unindexed post is not synthesized. | Fake PDS/post-create path not involved; store lacks newly-created row. | Request timeline after create fixture not inserted into `craftsky_posts`. | Timeline omits unindexed post. | `appview/internal/api/timeline_test.go` or implementation-level test with fake store |
| IT-014 | FR-012, NFR-001 | AC-012 | Handle-resolution failure fails request. | Timeline row for author; resolver returns error for that DID. | Request timeline. | Standard `identity_unavailable` error behavior; no partial page returned. | `appview/internal/api/timeline_test.go` |
| IT-015 | FR-013 | AC-015 | Non-Craftsky follows do not contribute non-Craftsky posts. | Viewer follows non-Craftsky DID in `atproto_follows`; no `craftsky_posts` row for that DID, optionally seed unrelated app.bsky-like fixture if helper table exists. | Call timeline store list. | No non-Craftsky content returned. | `appview/internal/api/timeline_store_test.go` |
| IT-016 | NFR-002 | AC-013 | Query remains bounded and index-oriented. | Representative fixture larger than one page. | Run timeline store list with limit. | Query uses `LIMIT` and does not load unbounded rows for client-side filtering; optionally assert via code review or `EXPLAIN` helper if practical. | `appview/internal/api/timeline_store_test.go` / review checklist |

## 6. Regression Tests

| ID | Existing Behavior Protected | Requirement IDs | Test |
|---|---|---|---|
| REG-001 | Existing post endpoints continue using the same `PostResponse` shape. | FR-008, RULE-003 | Existing `appview/internal/api/post_response_test.go` and post handler tests should still pass unchanged or with only shared-helper-safe updates. |
| REG-002 | Profile post list remains author-scoped and excludes comments/replies as before. | FR-008 | Existing `ListByAuthor` and profile posts tests in `post_store_test.go` / `post_test.go` should continue to pass. |
| REG-003 | Profile comments and comment/reply endpoints continue returning conversation rows; timeline exclusion must not break conversation surfaces. | FR-004 | Existing comments/replies tests in `post_test.go` should continue to pass. |
| REG-004 | Feed extensibility boundary is not coupled to route registration or a one-off handler-only SQL string. | BR-002, NFR-003 | Review or unit test the introduced timeline store/query boundary; route test should only verify wiring. |
| REG-005 | Existing auth/device middleware behavior remains unchanged for other routes. | NFR-001 | Existing `routes_test.go` auth/device tests continue to pass. |

## 7. Test Data

| ID | Purpose | Data | Used By |
|---|---|---|---|
| TD-001 | Base authenticated viewer | `did:plc:viewer` Craftsky profile, Bluesky profile row, auth context or `X-Dev-DID` test request. | AT-001, AT-002, IT-001, IT-004, IT-006, IT-010 |
| TD-002 | Followed and unfollowed authors | `did:plc:alice` followed by viewer; `did:plc:bob` not followed; optional `did:plc:viewer` self-follow row. | AT-002, IT-001, IT-006 |
| TD-003 | Eligible post rows | Root/top-level posts with no reply fields; quote post with `quote_uri`/`quote_cid`; project post represented as stored record JSON while still top-level. | AT-003, AT-008, IT-001, IT-007, IT-010 |
| TD-004 | Excluded conversation rows | Top-level comment where `reply_root_uri = reply_parent_uri`; nested reply where `reply_parent_uri != reply_root_uri`; partial reply-field malformed row if helper supports it. | AT-003, UT-001, IT-007 |
| TD-005 | Excluded repost activity | `craftsky_reposts` active row referencing an indexed post. | AT-003, IT-007 |
| TD-006 | Ordering and pagination rows | At least five eligible rows with distinct and tied `indexed_at`, deterministic URIs, page size 2. | AT-004, IT-002, IT-003 |
| TD-007 | Engagement state | Active like/repost/reply state for viewer and counts for returned post URIs. | AT-008, IT-010 |
| TD-008 | Empty timeline | Viewer with no own eligible rows and no followed eligible rows. | AT-005, IT-004 |
| TD-009 | Non-Craftsky follow | `atproto_follows` row from viewer to `did:plc:external` without matching `craftsky_posts` rows. | IT-015 |
| TD-010 | Identity failure | Resolver fake returning error for one returned author DID. | AT-009, IT-014 |

## 8. Manual Checks

| ID | Requirement IDs | Check | Steps | Expected Result |
|---|---|---|---|---|
| MAN-001 | NFR-002 | Basic dev smoke and perceived query boundedness. | With `just dev-d` running and seed data available, request `GET /v1/feed/timeline?limit=2` using dev auth/device headers; optionally inspect logs for row count/latency. | Response is bounded to requested limit, returns expected JSON shape, and does not show obvious unbounded behavior. |

## 9. Test Gaps And Risks

| ID | Gap / Risk | Affected Requirement IDs | Reason | Follow-Up |
|---|---|---|---|---|
| GAP-001 | SQL query-plan/index performance may not be fully proven by automated tests. | NFR-002 | Unit/integration tests can assert bounded behavior but may not reliably validate production-scale plans. | Include query review during implementation; add narrow supporting index only if evidence shows existing indexes are insufficient. |
| GAP-002 | Future moderation filters are not tested. | NG-007, RISK-005 | Blocks, mutes, reports, and moderation labels are explicitly out of scope for this chunk. | Cover in future moderation/timeline filtering specs. |
| GAP-003 | Future repost feed-item shape is not tested. | RULE-004 | Repost timeline shape remains an explicit non-blocking open question. | Cover when repost feed reasons/attribution are designed. |
| GAP-004 | Flutter timeline consumption is not tested. | NG-001 | This is AppView-only; Flutter feed screen/data layer is separate work. | Add Flutter repository/provider/UI tests in the later Flutter feed change. |
| GAP-005 | Real PDS/network behavior is not tested for timeline reads. | FR-009 | Timeline happy path should not call PDS; PDS integration belongs to write/indexing tests. | Keep AppView timeline tests DB/fake based. |

## 10. Out Of Scope

- Flutter feed UI, API client, providers, infinite scroll, and empty-state UI.
- Materialised feed tables, fan-out-on-write, Redis/cache behavior, or background feed jobs.
- Algorithmic ranking, recommendation feeds, search, craft/project filters, hashtag feeds, and custom/list feeds.
- Blocks, mutes, reports, moderation labels, and rate limiting behavior.
- Expanded quoted-post cards and repost feed reasons.
- PDS read-through or synthetic just-created timeline rows before indexing.

## 11. Handoff To Document Review

- Requirements file: `docs/changes/2026-05-28-timeline-feed-appview/01-requirements.md`
- Test specification: `docs/changes/2026-05-28-timeline-feed-appview/02-acceptance-tests.md`
- Next review artifact: `03-document-review.md`
- Preferred Plannotator review: `/plannotator-annotate docs/changes/2026-05-28-timeline-feed-appview/`
- Recommended first failing test for implementation: `IT-001` — `TestTimelineStore_ListTimeline_ReturnsOwnAndFollowedEligiblePostsOnly` in `appview/internal/api/timeline_store_test.go`.
- Suggested test order for implementation:
  1. `IT-001` for core own/followed/unfollowed author eligibility.
  2. `IT-007` and `UT-001` for post/comment/reply/repost eligibility.
  3. `IT-002` and `IT-003` for ordering and cursor pagination.
  4. `IT-006`, `IT-015`, and `IT-013` for self deduplication, non-Craftsky follows, and index-backed visibility.
  5. `IT-010`, `AT-008`, and `IT-014` for response assembly and identity failure behavior.
  6. `IT-008`, `IT-009`, `IT-011`, `IT-012`, and `IT-004` for route/handler API edge cases.
  7. Regression suite (`REG-001` through `REG-005`).
- Commands discovered:
  - Full AppView suite: `just test` from repo root after `just dev-d` is running.
  - Focused API tests: `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api`.
  - Focused route tests: `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/routes`.
- Blocking gaps: None.
