# Coding Plan: Timeline Feed AppView

## 1. Inputs

- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md`
- Verdict: Approved with notes; no blocking issues.
- Plannotator/user feedback incorporated: timeline pagination should default to `20` and cap at `50` items, even though existing shared pagination helpers may default/cap differently.

## 2. Implementation Strategy

Implement the home timeline as an AppView-only additive endpoint backed by existing indexed Postgres data. Add a narrow timeline query boundary on `*api.PostStore`, add a handler that reuses existing `PostResponse`, engagement summary, handle-resolution, cursor, and error-envelope conventions, then wire `GET /v1/feed/timeline` through existing authenticated-device middleware.

This plan avoids a materialised feed table, PDS read-through, lexicon changes, Flutter work, generic feed/search abstractions, moderation filters, expanded quote cards, and repost feed reasons. Future feed variants should remain possible because timeline author eligibility, root-post eligibility, pagination, and response assembly stay behind named store/handler boundaries rather than being embedded directly in route registration.

## 3. Affected Areas

| Area | Existing Pattern | Planned Change | Requirement IDs | Test IDs |
|---|---|---|---|---|
| Store/query | `PostStore` owns SQL over `craftsky_posts`; list methods use seek cursors and `PostRow`. | Add `ListTimeline(ctx, viewerDID, limit, cursor)` using current follows and top-level post predicates. | BR-001, BR-002, FR-002-FR-007, FR-009, FR-013, NFR-002, NFR-003, RULE-001-RULE-005 | IT-001-IT-007, IT-015, IT-016, UT-001, UT-002, UT-004 |
| API handler | Handlers parse pagination, map `envelope.ErrInvalidCursor`, batch engagement summaries, resolve handles, and encode JSON. | Add `ListTimelineHandler` using a `TimelineReader` interface and existing post response assembly. | FR-001, FR-006-FR-012, NFR-001 | AT-005, AT-007-AT-009, IT-004, IT-010-IT-014, UT-003, UT-005 |
| Routes | `routes.AddRoutes` composes `Authenticated` + `DeviceID` for `/v1/*` routes. | Register `GET /v1/feed/timeline` with the same auth/device stack. | BR-001, FR-001, NFR-001 | AT-001, IT-008, IT-009, REG-005 |
| Persistence/indexes | Existing migrations define `craftsky_posts` and `atproto_follows`. | No migration expected. Review query plan; add only a narrow supporting index if evidence shows existing indexes are insufficient. | FR-009, NFR-002 | IT-016, MAN-001 |
| Regression surfaces | Profile post/comment lists and post response helpers are already tested. | Keep shared helper changes minimal; run existing post/profile suites. | FR-004, FR-008, RULE-003 | REG-001-REG-005 |

## 4. Files And Modules

| Path / Module | Create / Change | Purpose | Requirement IDs | Test IDs |
|---|---|---|---|---|
| `appview/internal/api/timeline_store.go` | Create | Timeline-specific query method on `PostStore`; reuse `postSelectColumns`, `scanPostRow`, `decodeSeekCursor`, and `envelope.EncodeCursor`. | FR-002-FR-007, FR-009, FR-013, RULE-001-RULE-005, NFR-002, NFR-003 | IT-001-IT-007, IT-015, IT-016, UT-001, UT-002, UT-004 |
| `appview/internal/api/timeline.go` | Create | Handler, timeline page response struct, `TimelineReader` interface, timeline-specific limit parsing, and post-response hydration flow. | FR-001, FR-006-FR-012, NFR-001 | AT-005, AT-007-AT-009, IT-004, IT-010-IT-014, UT-003, UT-005 |
| `appview/internal/routes/routes.go` | Change | Register `GET /v1/feed/timeline` with `authN(deviceID(...))`, using existing `postStore`. | FR-001, NFR-001 | IT-008, IT-009, REG-005 |
| `appview/internal/api/timeline_store_test.go` | Create | Store/query integration tests against isolated Postgres schemas. | BR-001, FR-002-FR-007, FR-009, FR-013, RULE-001-RULE-005, NFR-002 | IT-001-IT-007, IT-015, IT-016 |
| `appview/internal/api/timeline_test.go` | Create | Handler tests with fake `TimelineReader` and fake resolver. | FR-006-FR-012, NFR-001, RULE-003 | AT-005, AT-007-AT-009, IT-004, IT-010-IT-014, UT-003, UT-005 |
| `appview/internal/api/timeline_query_test.go` | Optional create | Only if pure helpers/options are introduced; otherwise cover eligibility and cursor behavior through store/handler tests. | BR-002, FR-004, FR-007, RULE-002, NFR-003 | UT-001, UT-002, UT-004 |
| `appview/internal/routes/routes_test.go` | Change | Add auth/device route tests and optional route-registration/positive wiring assertion. | FR-001, NFR-001 | AT-001, IT-008, IT-009, REG-005 |
| `appview/migrations/<next>_*.up.sql` / `.down.sql` | Avoid unless proven | Optional narrow supporting index only after query-plan evidence. | NFR-002 | IT-016, MAN-001 |

## 5. Services, Interfaces, And Data Flow

### Store/query boundary

Use `PostStore` as the concrete store to avoid a new repository stack while still adding a feed-specific method and interface.

```text
interface TimelineReader {
  ListTimeline(ctx, viewerDID string, limit int, cursor string) ([]*PostRow, string, error)
  EngagementSummaries(ctx, viewerDID string, postURIs []string) (map[string]EngagementSummary, error)
}

func (s *PostStore) ListTimeline(ctx, viewerDID string, limit int, cursor string) ([]*PostRow, string, error)
```

Query shape sketch:

```text
SELECT <postSelectColumns>
FROM craftsky_posts p
LEFT JOIN bluesky_profiles bp ON bp.did = p.did
WHERE p.reply_root_uri IS NULL
  AND p.reply_parent_uri IS NULL
  AND (
    p.did = $viewerDID
    OR EXISTS (
      SELECT 1 FROM atproto_follows f
      WHERE f.did = $viewerDID AND f.subject_did = p.did
    )
  )
  AND ($cursorIndexedAt IS NULL
       OR (p.indexed_at, p.uri) < ($cursorIndexedAt, $cursorURI))
ORDER BY p.indexed_at DESC, p.uri DESC
LIMIT $limit
```

Important query properties:

- Query `craftsky_posts` once, with `EXISTS` for follow eligibility, so self-follow cannot duplicate rows.
- Root-post predicate is exactly `reply_root_uri IS NULL AND reply_parent_uri IS NULL`.
- Project posts and quote posts pass through naturally because they are still top-level `craftsky_posts` rows.
- Reposts do not appear because the query never reads `craftsky_reposts` as feed items.
- Non-Craftsky follows do not contribute because only `craftsky_posts` rows are selected.
- Cursor payload should match existing indexed order: `{indexedAt: <RFC3339Nano>, uri: <uri>}`.

### Handler data flow

```text
GET /v1/feed/timeline
  -> Authenticated middleware injects viewer DID
  -> DeviceID middleware validates device header
  -> ListTimelineHandler
      limit := parseTimelineLimit(query.limit)  // default 20, cap 50
      cursor := query.cursor                    // ignore all unknown params
      rows, nextCursor := store.ListTimeline(viewerDID, limit, cursor)
      summaries := store.EngagementSummaries(viewerDID, rowURIs)
      handles := resolveHandlesForRows(rows, resolver)
      items := BuildPostResponse(row, handles[row.DID]) + applyEngagementSummary
      write {items, cursor?}
```

Handle resolution should fail the whole request with `502 identity_unavailable`, matching existing post/profile behavior.

## 6. State, Providers, Controllers, Or DI

None identified for Flutter/Riverpod or client state; this is AppView-only.

Server-side DI remains handler-constructor based:

```text
routes.AddRoutes
  postStore := api.NewPostStore(deps.DB)
  mux.Handle("GET /v1/feed/timeline",
    authN(deviceID(api.ListTimelineHandler(postStore, deps.HandleResolver, deps.Logger))))
```

Avoid adding a broad `Deps.TimelineStore` unless tests prove route wiring cannot be covered otherwise. The handler itself should take the narrow `TimelineReader` interface so API tests can inject fakes without constructing Postgres.

## 7. UI, Widgets, Routes, Or User-Facing Surfaces

- Add API route: `GET /v1/feed/timeline?limit=<n>&cursor=<opaque>`.
- Success response: `{ "items": [PostResponse...], "cursor": "..." }`, with `cursor` omitted when empty/final page.
- Empty response: `{ "items": [] }`.
- Default timeline page size: `20`.
- Maximum timeline page size: `50`; overlarge values should be silently capped.
- No Flutter screens, widgets, repositories, providers, routing, or app models in this chunk.
- No CLI or background job changes.

## 8. Error, Loading, Empty, And Edge States

| State / Case | Planned Handling | Requirement IDs | Test IDs |
|---|---|---|---|
| Missing auth | Existing `Authenticated` middleware returns `401 unauthorized`. | FR-001, NFR-001 | AT-001, IT-008 |
| Missing device ID | Existing `DeviceID` middleware returns `400 missing_device_id`. | FR-001, NFR-001 | AT-001, IT-009 |
| Invalid cursor | Store/helper returns `envelope.ErrInvalidCursor`; handler maps to `400 invalid_cursor`. | FR-007, FR-010, NFR-001 | AT-007, IT-011, UT-002 |
| Empty timeline | Return `200` with `items: []`, no `cursor`, no suggestions. | FR-006, FR-011 | AT-005, IT-004, UT-003 |
| No more pages | Return final page with cursor omitted. | FR-006, FR-007 | IT-003, UT-003 |
| Unknown query params | Do not read them; only `limit` and `cursor` affect store call. | FR-007 | IT-012, UT-003 |
| Overlarge/invalid limit | Use timeline-specific parsing: default `20`, cap `50`, silent default on invalid/non-positive. | FR-007, DR-001 | Add to IT-012 or handler unit tests |
| Author handle resolution fails | Fail entire request with `502 identity_unavailable`; no partial page. | FR-012, NFR-001 | AT-009, IT-014 |
| Engagement lookup fails | Follow existing post-list behavior: `500 internal_error`. | FR-008, NFR-001 | IT-010 plus negative handler test if added |
| Comments/replies | Exclude any row with either reply field present. | FR-004, RULE-002 | AT-003, UT-001, IT-007 |
| Self-follow duplicate risk | Use single post query plus `EXISTS`, not a join that can duplicate rows. | RULE-005 | IT-006 |
| Just-created unindexed post | Do not synthesize; only rows returned by store appear. | FR-009 | IT-013 |

## 9. Test Implementation Plan

| Order | Test ID | Target | Setup / Fixture | Initial Expected Failure |
|---|---|---|---|---|
| 1 | IT-001 | `appview/internal/api/timeline_store_test.go` | Extend isolated DDL with `atproto_follows`; seed viewer, followed A, unfollowed B, root/project/quote rows. | `PostStore` has no `ListTimeline`; no timeline query exists. |
| 2 | IT-007 | `timeline_store_test.go` | Followed author with root, quote, top-level comment, nested reply, active repost row. | Conversation rows/reposts not yet filtered or no method exists. |
| 3 | UT-001 | `timeline_query_test.go` or `timeline_store_test.go` | If helper introduced, pass rows with reply-field combinations. | Helper missing or wrong classification. |
| 4 | IT-002 | `timeline_store_test.go` | Seed eligible rows with distinct and tied `indexed_at`. | Ordering not implemented. |
| 5 | IT-003 | `timeline_store_test.go` | Page size 2 over five ordered rows, then use returned cursor. | Cursor encoding/seek not implemented. |
| 6 | IT-006 | `timeline_store_test.go` | Viewer own post with and without self-follow. | Own-post eligibility/dedup not implemented. |
| 7 | IT-015 | `timeline_store_test.go` | Viewer follows DID with no `craftsky_posts` rows. | Should remain empty for that follow. |
| 8 | IT-013 | `timeline_test.go` | Fake timeline store lacks newly-created row; no PDS fake in handler. | Handler should only return store rows. |
| 9 | IT-010 / AT-008 | `timeline_test.go` | Fake rows with images/tags/quote, engagement summaries, fake resolver. | Handler missing or not using `PostResponse`/engagement. |
| 10 | IT-014 | `timeline_test.go` | Resolver returns error for returned author DID. | Handler must map to `identity_unavailable`. |
| 11 | IT-011 / AT-007 | `timeline_test.go` | Store returns `envelope.ErrInvalidCursor` or bad cursor reaches store. | Error mapping missing. |
| 12 | IT-012 / DR-001 | `timeline_test.go` | Request unknown params; include default/max-limit cases with default `20`, cap `50`. | Handler may pass wrong limit or inspect filters. |
| 13 | IT-004 / AT-005 | `timeline_test.go` | Fake store returns zero rows. | Empty page shape missing. |
| 14 | IT-008 / IT-009 | `routes_test.go` | Existing test deps; request timeline without auth / without device ID. | Route missing, wrong middleware, or route not registered. |
| 15 | DR-002 optional | `routes_test.go` | Pattern-registration assertion or valid request using a real test DB/fake-safe route setup. | Positive wiring not directly asserted. |
| 16 | IT-016 / MAN-001 | Store test/review | Ensure query uses `LIMIT`; optionally inspect `EXPLAIN` in dev DB. | Query may be unbounded or require index review. |
| 17 | REG-001-REG-005 | Existing suites | Run focused API/routes packages, then full AppView suite when DB is up. | Shared helper or route changes break existing behavior. |

Focused commands:

```text
cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api
cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/routes
just test
```

## 10. Sequencing And Guardrails

- First TDD step: `IT-001` — `TestTimelineStore_ListTimeline_ReturnsOwnAndFollowedEligiblePostsOnly` in `appview/internal/api/timeline_store_test.go`.
- Dependencies between work items:
  1. Store query and cursor behavior before handler assembly.
  2. Handler response/error behavior before route wiring.
  3. Route tests after handler constructor exists.
  4. Regression suite after shared helpers/routes are stable.
- Guardrails:
  - Do not modify `lexicon/` or generated lexicon Go types.
  - Do not add Flutter UI/data-layer work.
  - Do not fetch timeline posts from PDSes in handler/store.
  - Do not synthesize just-created posts into timeline responses.
  - Do not introduce total counts, suggestions, filters, ranking, materialised feed tables, repost feed reasons, or moderation filters in this chunk.
  - Keep unknown query params ignored.
  - Use timeline-specific limit parsing with default `20` and max `50`.
  - Use typed DID parsing at HTTP/identity boundaries where existing helpers require it; do not revalidate internal DB fields unnecessarily.
  - Stage any optional migration only if query-plan evidence justifies it.

## 11. Risks And Open Questions

| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Non-blocking | Existing indexes may not be ideal for the exact timeline predicate and order. | Could cause slow reads for large follow graphs. | Review query plan during implementation; add a narrow partial index only with evidence. |
| CPQ-002 | Non-blocking | Positive route-to-handler assertion may need a real test DB because `routes.AddRoutes` currently constructs `PostStore` from `deps.DB`. | Route tests may cover auth/device negatives and registration but not full positive dispatch cheaply. | Prefer handler tests for behavior; add route registration assertion or real-DB positive route test if practical. |
| CPQ-003 | Non-blocking | Timeline page-size policy differs from current shared `parseLimit`. | Reusing `parseLimit` would incorrectly default to 50 and cap at 100. | Add timeline-specific parser/tests: default 20, max 50. |
| CPQ-004 | Non-blocking | Future feed variants remain unspecified. | Over-abstraction now could be wrong; under-abstraction could cause rewrites. | Keep only a small timeline store/query boundary and avoid generic feed framework. |

## 12. Handoff To TDD Builder

- Coding plan: `04-coding-plan.md`
- TDD execution plan: `05-implementation-plan.md`
- Start with test: `IT-001` — `TestTimelineStore_ListTimeline_ReturnsOwnAndFollowedEligiblePostsOnly`
- First target file: `appview/internal/api/timeline_store_test.go`
- First production target: `appview/internal/api/timeline_store.go`
- Focused command: `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api`
- Notes: Treat workflow docs as source of truth; ignore prior conversation unless captured in those documents. Keep TDD red-green-refactor discipline and avoid implementing out-of-scope feed variants.
