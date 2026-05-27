# Implementation Review: Follow / Unfollow MVP

## Verdict
Status: Approved with notes
Reviewer: implementation-reviewer
Date: 2026-05-27
Risk level: Medium

## Summary

The implementation satisfies the approved Follow / Unfollow MVP scope. It uses interoperable `app.bsky.graph.follow` records, keeps PDS writes server-side, indexes active follow graph state, supports non-Craftsky profile display/hydration, exposes profile relationship/count fields, and wires Flutter follow/unfollow UI with optimistic loading/error behavior.

The prior review gaps around durable graph ownership, real `viewerIsFollowing` reads, non-Craftsky hydration, duplicate follow collapse, and failing broader tests have been addressed. Automated AppView and Flutter profile test evidence is green. The remaining notes are non-blocking operational/manual verification items rather than implementation defects.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-001 | Suggestion | Tests / Release | Live dev-stack smoke checks are still documented as manual rather than evidenced in the implementation log. Automated tests cover handler, store, indexer, route, and Flutter behavior, but `MAN-001`, `MAN-002`, and `MAN-003` remain useful before release or demo because they validate Tap/PDS runtime wiring and query plans outside isolated tests. | `02-acceptance-tests.md` MAN-001, MAN-002, MAN-003; `05-implementation-plan.md` lines 55-57 | Run the manual smoke checks before release/signoff if live Tap/PDS confidence is required. No code change required for this review. |
| IR-002 | Suggestion | Traceability | `05-implementation-plan.md` records completed tests and review follow-ups, but the final completion checklist remains unchecked. This is documentation hygiene only; the commands and implementation notes above it provide sufficient evidence. | `05-implementation-plan.md` Completion Checklist | Optionally mark the checklist in a later docs-only cleanup. No implementation change required. |

## Requirement And Test Traceability

- Requirements implemented:
  - `BR-001`, `FR-003`, `FR-004`, `FR-005`: follow/unfollow handlers resolve current handles/DIDs, reject self targets, write/delete `app.bsky.graph.follow` records through the server-side PDS client, and return updated profile responses.
  - `BR-002`, `FR-006`, `RULE-005`, `RULE-006`, `RULE-009`: profile responses include `viewerIsFollowing`, `isCraftskyProfile`, and Craftsky-account-only counts with count-error handling.
  - `BR-004`, `FR-001`, `FR-002`, `FR-010`, `RULE-003`, `RULE-007`: active follow graph persistence and Tap/indexer handling cover create/update/delete, duplicate collapse, historical events, and future followed-DID lookup.
  - `BR-005`, `FR-011`, `FR-012`, `RULE-008`: non-Craftsky profile cache/hydration and UI marker are implemented; non-Craftsky counts remain nullable/unknown.
  - `FR-007`, `FR-008`, `FR-009`, `NFR-003`: Flutter model/client/repository/provider/UI are wired without PDS tokens, with loading, optimistic update, server-response replacement, and rollback/error messaging.
- Tests implemented:
  - AppView follow store/indexer/handler/route/profile tests cover the planned UT/IT cases, including duplicate collapse and Tap ownership follow-up tests.
  - Flutter model/API/provider/profile-page tests cover the planned UI/data behavior, including non-Craftsky marker, unknown counts, loading, rollback, and token-boundary checks.
  - Broader AppView package tests and full Flutter profile tests pass.
- Unplanned behavior:
  - `PostStore.ViewerReplyStates` and one invalid Craftsky post image test fixture were corrected while fixing broader failing tests. These are regression-aligned repairs, not feature behavior expansions.
- Remaining gaps:
  - Manual dev-stack checks `MAN-001`, `MAN-002`, and `MAN-003` are not evidenced in `05-implementation-plan.md`.

## Test Evidence

- Commands reviewed:
  - `plannotator review --git` → review session closed without feedback.
  - `cd appview && TEST_DATABASE_URL="postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable" go test ./internal/api ./internal/index ./internal/routes ./internal/app`
  - `cd app && flutter test test/profile`
- Passing evidence:
  - AppView command passed for `internal/api`, `internal/index`, `internal/routes`, and `internal/app`.
  - Flutter profile suite passed all 49 tests.
  - `05-implementation-plan.md` records additional focused commands for follow handlers, profile store, follow indexer, and focused Flutter model/client/provider/page tests.
- Failing or skipped tests:
  - No current automated failures observed.
  - Manual checks are not recorded as run.

## Risk Review

- Risk level: Medium
- Risk notes:
  - The slice touches migration, Tap indexing, AppView PDS writes, profile hydration, and Flutter UI. Automated coverage is strong, but live Tap/PDS behavior still benefits from manual smoke testing.
  - Durable graph state correctly belongs to Tap/indexer convergence, with handler response overlays limited to immediate UI responsiveness.
  - PDS tokens remain server-side; Flutter only calls Craftsky `/v1/*` endpoints.
- Approval notes:
  - No blocking behavior, test, traceability, or security issues were identified.

## UI Polish Recommendation

- Recommendation: Optional
- Reason:
  - User-facing UI changed for Follow/Unfollow, loading/disabled states, real counts, unknown non-Craftsky counts, and the `Non Craftsky profile` marker. The implemented UI satisfies requirements and tests, but a small polish pass could improve marker presentation or accessibility copy without changing behavior.
- Suggested polish notes:
  - Consider styling the non-Craftsky marker as a small badge/chip and reviewing semantics for loading/disabled follow state. Do not change copy, behavior, API shape, or acceptance criteria.

## Handoff Back To TDD Builder

- Required fixes:
  - None.
- Suggested next failing test:
  - None required. If additional assurance is desired, run `MAN-001`/`MAN-002` against the dev stack and document results.
- Verification to rerun:
  - `cd appview && TEST_DATABASE_URL="postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable" go test ./internal/api ./internal/index ./internal/routes ./internal/app`
  - `cd app && flutter test test/profile`
