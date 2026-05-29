# Implementation Review: Timeline Feed AppView

## Verdict
Status: Changes required
Reviewer: gpt-5.5 implementation reviewer
Date: 2026-05-29
Risk level: Medium

## Summary

The implementation adds the requested AppView timeline endpoint, store boundary, route wiring, and focused tests. The main timeline semantics are present: authenticated `/v1/feed/timeline`, current AppView-indexed posts/follows, own-post inclusion, followed-author selection, comment/reply/repost exclusion, `indexed_at DESC, uri DESC` ordering, existing `PostResponse` hydration, engagement summaries, and no PDS read-through.

However, two requirements-to-test gaps remain before this stage should be approved. First, the store returns a next cursor whenever a page is exactly full, even when that page contains the final eligible row, which contradicts the documented `{items, cursor}` contract that omits `cursor` when there are no more results. Second, the acceptance spec's automated current-follow-graph pagination test (`IT-005`) was not implemented or documented as intentionally skipped. A minor unreviewed working-tree roadmap edit is also present and should be kept out of this stage unless handled separately.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-001 | Important | Behavior / Tests | `PostStore.ListTimeline` returns a cursor whenever `len(out) == limit`; therefore an exact-full final page still includes a cursor even though no additional eligible rows exist. The tests only cover a final page smaller than `limit`, so this contract edge case is unprotected. | `01-requirements.md` FR-006, AC-006, AC-016; `02-acceptance-tests.md` IT-003; `appview/internal/api/timeline_store.go:58-69`; `appview/internal/api/timeline_store_test.go:160-208` | Add a failing test for an exact-full final page and update pagination detection, likely by fetching `limit + 1` rows internally and returning only `limit` items with a cursor only when an extra row exists. |
| IR-002 | Important | Tests / Traceability | Planned automated coverage for current follow-graph evaluation on each page request (`IT-005`) is missing from the implemented tests and is not documented as a gap. The query appears to use the current `atproto_follows` state per call, but the required follow/unfollow-between-pages behavior is not protected. | `01-requirements.md` FR-002, RULE-001, Q8, AC-003; `02-acceptance-tests.md` IT-005; `05-implementation-plan.md` Steps 1-15; `appview/internal/api/timeline_store_test.go` | Add the `IT-005` store integration test or explicitly document an accepted gap with rationale. Prefer adding the test because it is planned automated coverage for Must behavior. |
| IR-003 | Suggestion | Risk / Working tree | The current working tree contains an unrelated unstaged edit to `docs/roadmap.md` marking follow/unfollow interactions complete. This does not appear to be part of the timeline-feed implementation commit or workflow documents. | `git status --short`; `git diff -- docs/roadmap.md` | Do not include this edit in the implementation-review stage commit. Handle it separately or revert it in a later appropriate stage if it was accidental. |

## Requirement And Test Traceability

- Requirements implemented:
  - `FR-001` / `NFR-001`: `GET /v1/feed/timeline` is registered under authenticated-device middleware.
  - `FR-002`, `RULE-001`, `RULE-005`: store query includes viewer-authored rows and followed authors via `EXISTS`, avoiding self-follow duplication.
  - `FR-003`, `FR-004`, `RULE-002`, `RULE-003`, `RULE-004`: timeline selects top-level Craftsky post rows and naturally excludes comments/replies/repost rows.
  - `FR-005`: ordering uses `indexed_at DESC, uri DESC`.
  - `FR-007`, `FR-010`: handler accepts `limit`/`cursor`, ignores unknown params, applies timeline-specific default/cap, and maps invalid cursors to `400 invalid_cursor`.
  - `FR-008`, `FR-012`: handler reuses `PostResponse`, engagement hydration, and handle-resolution failure behavior.
  - `FR-009`, `FR-013`: reads are backed by `craftsky_posts`/indexed AppView data and do not synthesize unindexed rows.
- Tests implemented:
  - Store tests cover `IT-001`, `IT-002`, `IT-003`, `IT-006`, `IT-007`, and `IT-015`.
  - Handler tests cover `IT-004`, `IT-010`, `IT-011`, `IT-012`, `IT-013`, and `IT-014`, plus DR-001 default/cap behavior.
  - Route tests cover `IT-008` and `IT-009`.
  - Existing regression suites passed under `just test`.
- Unplanned behavior:
  - None identified in committed source/test changes.
  - Unrelated unstaged `docs/roadmap.md` edit exists outside the committed implementation.
- Remaining gaps:
  - Exact-full final page cursor omission behavior and test coverage (`IR-001`).
  - Current-follow-graph pagination test coverage (`IR-002`).

## Test Evidence

- Commands reviewed:
  - Implementation plan reports `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api` — passed.
  - Implementation plan reports `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/routes` — passed.
  - Implementation plan reports `just test` — passed.
  - Reviewer reran `just test` — passed.
- Passing evidence:
  - Full AppView suite passed with race-enabled `go test -race ./...` via `just test`.
- Failing or skipped tests:
  - No command failures observed during review.
  - Missing planned `IT-005` current-follow-graph test remains unimplemented.

## Risk Review

- Risk level: Medium.
- Risk notes:
  - The endpoint is additive and follows existing API/store patterns, but timeline semantics are foundational for later Flutter feed work.
  - The exact-full-page cursor issue could make clients perform unnecessary follow-up requests and violates the documented cursor omission contract.
  - Missing `IT-005` weakens protection around follow/unfollow drift behavior explicitly accepted by the requirements.
- Approval notes:
  - Most implementation pieces are traceable and the full suite passes, but Must-level behavior/test gaps should be addressed before approval.

## UI Polish Recommendation

- Recommendation: Not needed
- Reason: This AppView-only change adds an HTTP endpoint and Go tests; it does not include user-facing Flutter UI changes.
- Suggested polish notes: None.

## Handoff Back To TDD Builder

- Required fixes:
  - Add exact-full final page cursor test and fix `ListTimeline` pagination to omit `cursor` when no additional row exists.
  - Add `IT-005` coverage for current follow-graph evaluation between page requests, or document a deliberate accepted gap.
- Suggested next failing test:
  - `TestTimelineStore_ListTimeline_OmitsCursorWhenExactFullFinalPage` in `appview/internal/api/timeline_store_test.go`.
  - Then `TestTimelineStore_ListTimeline_UsesCurrentFollowGraphOnEachPage` for `IT-005`.
- Verification to rerun:
  - `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/api`
  - `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/routes`
  - `just test`
