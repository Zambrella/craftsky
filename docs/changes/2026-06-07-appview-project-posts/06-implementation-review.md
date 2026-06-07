# Implementation Review: AppView Project Posts

## Verdict
Status: Changes required
Reviewer: gpt-5.5 implementation-reviewer
Date: 2026-06-07
Risk level: High

## Summary
The implementation covers much of the planned AppView project-post slice: schema migration, project indexing/materialization, project-bearing create requests, shared post response hydration, profile `projectCount`, and `GET /v1/profiles/{handleOrDid}/projects` are present. The independently runnable no-database Go suite passes, and the implementation plan records focused TDD evidence.

Changes are required before approval because the project detail materialization currently copies detail fields into every craft-specific column family regardless of the `details.$type`, which corrupts query dimensions. Several Must acceptance tests are also missing or incomplete, especially route auth coverage and full DB coverage for project update/removal/delete convergence and detail-column materialization. Full `just test` evidence is still unavailable in this review environment because the compose Postgres was not running.

## Findings
| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-001 | Important | Behavior | Craft-specific detail columns are populated without checking `project.details.$type`. `upsertProjectMaterialization` passes the same `projectType`, `projectSubtype`, `yarnWeight`, and other keys into knitting, crochet, quilting, and sewing columns whenever those keys exist, so a knitting project can also populate crochet/sewing/quilting dimensions. This violates the intended per-craft materialized read model and creates false positives for future craft filters. | `01-requirements.md` FR-003, AC-003, NFR-003; `04-coding-plan.md` persistence shape/detail DTOs; `appview/internal/index/craftsky_post.go` `upsertProjectMaterialization` | Branch materialization by `details_type` and only populate the matching craft column family; leave unrelated craft columns NULL. Add tests that prove a known knitting/crochet/quilting/sewing payload populates only its own columns plus raw details. |
| IR-002 | Important | Tests / Traceability | Must acceptance coverage is incomplete. The implementation lacks explicit DB-backed tests for project update removing stale child rows, project delete/cascade removing child rows, unknown future details through the full indexer path, and materialized craft detail columns. There is also no route test proving `GET /v1/profiles/{handleOrDid}/projects` is registered under the same auth/device stack. | `02-acceptance-tests.md` AT-004, IT-005, IT-013, UT-002, IT-003; `05-implementation-plan.md` Steps 4, 5, 13; `appview/internal/index/craftsky_post_test.go`; `appview/internal/routes/routes_test.go` | Add the missing failing tests, then implement/fix until they pass with a real test database. Update `05-implementation-plan.md` if prior coverage claims need correction. |
| IR-003 | Important | Test Evidence | Full repository verification was not completed in this review environment. `go test ./...` from `appview/` passed with DB-backed tests skipped/cached when `TEST_DATABASE_URL` was unset, but `just test` failed because Postgres at `localhost:5433` was not running. | `02-acceptance-tests.md` commands; `05-implementation-plan.md` Verification; review command output | After the implementation fixes, start the compose database (`just dev-d` or equivalent repo workflow) and rerun `just test`; record passing evidence or any remaining failures. |

## Requirement And Test Traceability
- Requirements implemented: FR-001 through FR-012 are represented in code paths for migration/schema, indexer materialization, create request/PDS body, `PostResponse.project`, profile counts, profile project list, and route registration.
- Tests implemented: Migration/schema test; tag merge unit tests; project detection/extraction tests; create request/handler tests; response inclusion/omission tests; profile count/list store tests; profile projects handler test.
- Unplanned behavior: None broad in scope, but IR-001 introduces incorrect cross-craft detail materialization.
- Remaining gaps: Missing Must tests and full DB-backed verification listed in IR-002 and IR-003.

## Test Evidence
- Commands reviewed:
  - `git status --short` — clean before review artifact.
  - `git show --stat --oneline HEAD` / changed-file inspection for implementation commit `f8ee2ac`.
  - `go test ./...` from `appview/`.
  - `just test` from repo root.
- Passing evidence:
  - `go test ./...` from `appview/` passed in this environment with no `TEST_DATABASE_URL` set.
- Failing or skipped tests:
  - `just test` failed before exercising DB-backed assertions because compose Postgres at `localhost:5433` refused connections.
  - DB-backed tests are not sufficient evidence until rerun against the expected compose database.

## Risk Review
- Risk level: High
- Risk notes: The slice changes persistence, indexing convergence, PDS write shape, public API responses, profile counts, and route surface. Incorrect project detail columns can compromise future project filtering/search and query correctness.
- Approval notes: Not ready for merge/handoff as complete until IR-001 and IR-002 are fixed and `just test` is rerun with the test database available.

## UI Polish Recommendation
- Recommendation: Not needed
- Reason: This AppView-only slice has no Flutter or user-facing UI changes beyond API responses.
- Suggested polish notes: None.

## Handoff Back To TDD Builder
- Required fixes:
  - Fix craft-specific detail materialization to respect `details.$type`.
  - Add missing Must tests for detail columns, update/removal/delete convergence, unknown details through the full indexer path, and projects route auth/device registration.
  - Rerun `just test` with compose Postgres running and document the result.
- Suggested next failing test: Add an indexer DB test with a knitting details payload that asserts knitting columns are populated while crochet/quilting/sewing columns remain NULL.
- Verification to rerun: `go test ./internal/index -run 'TestCraftskyPost_.*Project' -count=1`, `go test ./internal/routes -count=1`, `go test ./...`, and `just test` after `just dev-d`/compose Postgres is available.
