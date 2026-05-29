# TDD Implementation Plan: Timeline Feed AppView

## Inputs
- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md`
- Coding plan: `04-coding-plan.md`

## Implementation Rules
- Do not implement behavior without a linked requirement ID.
- Write or update a failing test before implementation.
- Run the smallest relevant test first.
- Refactor only after tests pass.
- Keep traceability updated.
- Use timeline-specific pagination defaults from `04-coding-plan.md`: default `20`, max `50`.

## Test Order
| Step | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| 1 | IT-001 | BR-001, FR-002, FR-003, RULE-001, RULE-003 | AC-002, AC-004 | Fails because timeline store/query does not exist. |
| 2 | IT-007 + UT-001 | FR-003, FR-004, RULE-002, RULE-004 | AC-004 | Fails until conversation/repost exclusion is covered. |
| 3 | IT-002 | BR-001, FR-005 | AC-005 | Fails until timeline ordering is covered. |
| 4 | IT-003 | FR-005, FR-006, FR-007 | AC-006 | Fails until cursor pagination is covered. |
| 5 | IT-006 | FR-002, RULE-001, RULE-005 | AC-014 | Fails until self inclusion/deduplication is covered. |
| 6 | IT-015 | FR-013 | AC-015 | Fails until non-Craftsky follow behavior is covered. |
| 7 | IT-013 | FR-009 | AC-018 | Fails until handler proves no synthetic rows. |
| 8 | IT-010 + AT-008 | FR-006, FR-008, FR-009, RULE-003 | AC-007, AC-011, AC-016 | Fails until handler and PostResponse assembly exist. |
| 9 | IT-014 | FR-012, NFR-001 | AC-012 | Fails until identity failure maps correctly. |
| 10 | IT-011 | FR-007, FR-010, NFR-001 | AC-009 | Fails until invalid cursor maps correctly. |
| 11 | IT-012 | FR-007 | AC-017 | Fails until unknown params are ignored and timeline limits default/cap correctly. |
| 12 | IT-004 | FR-011 | AC-008 | Fails until empty handler response is covered. |
| 13 | IT-008 + IT-009 | FR-001, NFR-001 | AC-001 | Fails until route is registered through auth/device middleware. |
| 14 | IT-016 | NFR-002 | AC-013 | Review/query test verifies bounded/index-oriented query. |
| 15 | REG-001 through REG-005 | FR-004, FR-008, NFR-001, RULE-003 | AC-007, AC-010 | Existing tests continue passing. |

## Implementation Steps

### Review Remediation: IR-001 and IR-002
- Source review: `06-implementation-review.md` found two blocking issues to fix: `IR-001` exact-full final pages returned a cursor, and `IR-002` `IT-005` current-follow-graph pagination coverage was missing.
- Remediation test order:
  1. `IT-003 / FR-006, FR-007, AC-006, AC-016`: add an exact-full final page cursor omission test, then fix pagination detection.
  2. `IT-005 / FR-002, RULE-001, AC-003`: add current-follow-graph-on-each-page coverage.
- Out of scope: `IR-003` roadmap edit is unrelated and will not be staged or changed by this remediation.

#### Remediation Step A: IT-003 exact-full final page cursor omission
- Write failing test: Added `TestTimelineStore_ListTimeline_OmitsCursorWhenExactFullFinalPage` in `appview/internal/api/timeline_store_test.go`.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api -run TestTimelineStore_ListTimeline_OmitsCursorWhenExactFullFinalPage`.
- Confirmed failure: The store returned an encoded cursor for a two-row final page requested with `limit=2`.
- Implement: Changed `PostStore.ListTimeline` to fetch `limit + 1` rows, return only the requested `limit`, and encode a next cursor only when the extra row proves more eligible rows exist.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api -run 'TestTimelineStore_ListTimeline_(OmitsCursorWhenExactFullFinalPage|PaginatesWithOpaqueSeekCursor)'`.
- Refactor: Ran `gofmt` on touched Go files.
- Notes: Focused pagination tests passed. This resolves `IR-001` for `FR-006`, `FR-007`, `AC-006`, and `AC-016`.

#### Remediation Step B: IT-005 current follow graph on each page
- Write failing test: Added `TestTimelineStore_ListTimeline_UsesCurrentFollowGraphOnEachPage` in `appview/internal/api/timeline_store_test.go`.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api -run TestTimelineStore_ListTimeline_UsesCurrentFollowGraphOnEachPage`.
- Confirmed failure: The test passed immediately because the existing query already evaluates `atproto_follows` on each call; the review issue was missing coverage rather than missing behavior.
- Implement: No additional production code required for current-follow-graph semantics.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api -run 'TestTimelineStore_ListTimeline_(OmitsCursorWhenExactFullFinalPage|PaginatesWithOpaqueSeekCursor|UsesCurrentFollowGraphOnEachPage)'`.
- Refactor: Ran `gofmt` on touched Go files.
- Notes: Focused remediation tests passed. This resolves `IR-002` for `FR-002`, `RULE-001`, and `AC-003`.

### Step 1: IT-001
- Write failing test: Added `TestTimelineStore_ListTimeline_ReturnsOwnAndFollowedEligiblePostsOnly` in `appview/internal/api/timeline_store_test.go`.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api -run TestTimelineStore_ListTimeline_ReturnsOwnAndFollowedEligiblePostsOnly`.
- Confirmed failure: Build failed because `*api.PostStore` had no `ListTimeline` method.
- Implement: Added `appview/internal/api/timeline_store.go` with `PostStore.ListTimeline`, selecting root posts authored by the viewer or current active follows through `EXISTS`, ordered by `(indexed_at DESC, uri DESC)`, with opaque indexed seek cursor support.
- Run command: Same focused command after starting `just dev-d`.
- Refactor: None.
- Notes: Focused test passed with database available. This covers own/followed/unfollowed author eligibility and includes root/project/quote-as-root rows without special feed item types.

### Step 2: IT-007 + UT-001
- Write failing test: Added `TestTimelineStore_ListTimeline_ExcludesConversationAndRepostActivity` for root/quote inclusion and comment/nested reply/repost exclusion.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api -run 'TestTimelineStore_ListTimeline_(ExcludesConversationAndRepostActivity|ReturnsOwnAndFollowedEligiblePostsOnly)'`.
- Confirmed failure: The test passed immediately because Step 1's minimum implementation already used the required root-post predicate and never reads repost rows as feed items.
- Implement: No additional production code required.
- Run command: Same focused command passed.
- Refactor: None.
- Notes: `UT-001` pure helper test was not added because no standalone eligibility helper was introduced; eligibility is covered through store integration tests.

### Step 3: IT-002
- Write failing test: Added `TestTimelineStore_ListTimeline_OrdersByIndexedAtThenURIDesc`.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api -run TestTimelineStore_ListTimeline_OrdersByIndexedAtThenURIDesc`.
- Confirmed failure: The test passed immediately because Step 1's query already used the specified order.
- Implement: No additional production code required.
- Run command: Focused command passed.
- Refactor: None.
- Notes: Covers `indexed_at DESC` and deterministic `uri DESC` tie-breaker.

### Step 4: IT-003
- Write failing test: Added `TestTimelineStore_ListTimeline_PaginatesWithOpaqueSeekCursor`.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api -run TestTimelineStore_ListTimeline_PaginatesWithOpaqueSeekCursor`.
- Confirmed failure: The test passed immediately because Step 1's store method reused existing indexed seek cursor behavior.
- Implement: No additional production code required.
- Run command: Focused command passed.
- Refactor: None.
- Notes: Covers page continuation, tied timestamp ordering, and final cursor omission.

### Step 5: IT-006
- Write failing test: Added `TestTimelineStore_ListTimeline_IncludesOwnPostWithoutSelfFollowAndDeduplicatesSelfFollow`.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api -run TestTimelineStore_ListTimeline_IncludesOwnPostWithoutSelfFollowAndDeduplicatesSelfFollow`.
- Confirmed failure: The test passed immediately because the store query selects viewer-authored rows directly and uses `EXISTS` instead of a duplicating join.
- Implement: No additional production code required.
- Run command: Focused command passed.
- Refactor: None.
- Notes: Covers own-post eligibility with and without self-follow and URI-level deduplication for self-follow.

### Step 6: IT-015
- Write failing test: Added `TestTimelineStore_ListTimeline_NonCraftskyFollowsDoNotContributeContent`.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api -run TestTimelineStore_ListTimeline_NonCraftskyFollowsDoNotContributeContent`.
- Confirmed failure: The test passed immediately because the timeline query only reads `craftsky_posts` rows.
- Implement: No additional production code required.
- Run command: Focused command passed.
- Refactor: None.
- Notes: A follow row for a DID with no Craftsky post rows contributes no content.

### Step 7: IT-013
- Write failing test: Added `TestTimelineHandler_DoesNotSynthesizeUnindexedPosts`.
- Run command: `go test ./internal/api -run TestTimelineHandler_DoesNotSynthesizeUnindexedPosts`.
- Confirmed failure: Build failed because `api.ListTimelineHandler` was undefined.
- Implement: Added `appview/internal/api/timeline.go` with `TimelineReader`, `TimelinePage`, `ListTimelineHandler`, and timeline-specific limit parsing.
- Run command: Focused command passed.
- Refactor: None.
- Notes: Handler constructor accepts no PDS dependency and only renders rows returned by the timeline store.

### Step 8: IT-010 + AT-008
- Write failing test: Added `TestTimelineHandler_ReturnsPostResponseItemsWithEngagementAndNoTotalCount`.
- Run command: `go test ./internal/api -run TestTimelineHandler_ReturnsPostResponseItemsWithEngagementAndNoTotalCount`.
- Confirmed failure: The test passed after Step 7 handler implementation because response assembly already reused `BuildPostResponse`, `applyEngagementSummary`, and `resolveHandlesForRows`.
- Implement: No additional production code required.
- Run command: Focused command passed.
- Refactor: None.
- Notes: Covers `PostResponse` shape, image rendering, quote strong-ref-only behavior, engagement fields, cursor, and absence of `totalCount`.

### Step 9: IT-014
- Write failing test: Added `TestTimelineHandler_HandleResolutionFailureFailsRequest`.
- Run command: `go test ./internal/api -run TestTimelineHandler_HandleResolutionFailureFailsRequest`.
- Confirmed failure: Test passed immediately because Step 7 handler already mapped row handle-resolution failures to `502 identity_unavailable`.
- Implement: No additional production code required.
- Run command: Focused command passed.
- Refactor: None.
- Notes: Confirms no partial page is returned when author identity cannot be resolved.

### Step 10: IT-011
- Write failing test: Added `TestTimelineHandler_InvalidCursorUsesStandardErrorEnvelope`.
- Run command: `go test ./internal/api -run TestTimelineHandler_InvalidCursorUsesStandardErrorEnvelope`.
- Confirmed failure: Test passed after Step 7 handler implementation because invalid cursor errors were already mapped to `400 invalid_cursor`.
- Implement: No additional production code required.
- Run command: Focused command passed.
- Refactor: None.
- Notes: Confirms standard error envelope contains `error`, `message`, and request ID field from shared writer.

### Step 11: IT-012
- Write failing test: Added `TestTimelineHandler_IgnoresUnknownParamsAndUsesTimelineLimits`.
- Run command: `go test ./internal/api -run TestTimelineHandler_IgnoresUnknownParamsAndUsesTimelineLimits`.
- Confirmed failure: Test passed because Step 7 handler already reads only `limit`/`cursor` and uses timeline-specific default `20` / max `50` parsing.
- Implement: No additional production code required.
- Run command: Focused command passed.
- Refactor: None.
- Notes: Covers DR-001 feedback and unknown query parameter behavior.

### Step 12: IT-004
- Write failing test: Added `TestTimelineHandler_EmptyTimelineReturnsEmptyItemsAndOmitsCursor`.
- Run command: `go test ./internal/api -run TestTimelineHandler_EmptyTimelineReturnsEmptyItemsAndOmitsCursor`.
- Confirmed failure: Test passed because Step 7 handler already initializes `items` as an empty slice and omits empty cursor.
- Implement: No additional production code required.
- Run command: Focused command passed.
- Refactor: None.
- Notes: Confirms no empty-state suggestions are included.

### Step 13: IT-008 + IT-009
- Write failing test: Added `TestAddRoutes_TimelineRequiresAuthenticatedDevice` and `TestAddRoutes_TimelineRequiresDeviceID` in `appview/internal/routes/routes_test.go`.
- Run command: `go test ./internal/routes -run 'TestAddRoutes_TimelineRequires(AuthenticatedDevice|DeviceID)'`.
- Confirmed failure: Both tests returned `404` because `GET /v1/feed/timeline` was not registered.
- Implement: Registered `GET /v1/feed/timeline` in `routes.AddRoutes` using `authN(deviceID(api.ListTimelineHandler(...)))`.
- Run command: Same focused route command passed.
- Refactor: Ran `gofmt` on touched Go files.
- Notes: Confirms route is under the existing authenticated-device middleware.

### Step 14: IT-016
- Write failing test: Not added; test specification allows code review/query inspection for this bounded-query concern.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api`; `just test`.
- Confirmed failure: Not applicable.
- Implement: The store query uses SQL `LIMIT $4`, seek pagination on `(indexed_at, uri)`, root-post predicates, and indexed follow lookup through `EXISTS` on `(did, subject_did)`.
- Run command: Both final package/full-suite commands passed.
- Refactor: None.
- Notes: No migration added because existing indexes support the current v1 query well enough for this slice; production-scale plans remain a documented follow-up risk.

### Step 15: REG-001 through REG-005
- Write failing test: Existing regression tests were reused.
- Run command: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api`; `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/routes`; `just test`.
- Confirmed failure: No regression failures.
- Implement: No additional production code required.
- Run command: All focused and full-suite commands passed.
- Refactor: None.
- Notes: Existing post/profile response, conversation, and auth/device tests remain green.

## Verification Results
- Focused API package: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api` — passed.
- Focused routes package: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/routes` — passed.
- Full AppView suite: `just test` — passed.
- Supporting dev stack: `just dev-d` was started to provide the compose Postgres required for integration tests.

## Remediation Verification Results
- Focused remediation tests: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api -run 'TestTimelineStore_ListTimeline_(OmitsCursorWhenExactFullFinalPage|PaginatesWithOpaqueSeekCursor|UsesCurrentFollowGraphOnEachPage)'` — passed.
- Focused API package: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api` — passed.
- Focused routes package: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/routes` — passed.
- Full AppView suite: `just test` — passed.
- Review findings addressed: `IR-001` and `IR-002` fixed. `IR-003` was explicitly excluded by the user and remains unstaged/unmodified by this remediation.

## Completion Checklist
- [x] All Must requirements covered by tests or documented gaps
- [x] All planned Must tests passing
- [x] Relevant regression tests passing
- [x] No unlinked behavior implemented
- [x] Docs updated
- [x] Review completed or explicitly skipped
