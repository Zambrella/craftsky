# Implementation Review: AppView Saved Posts

## Verdict

Status: Changes required  
Reviewer: Codex implementation review  
Date: 2026-07-21  
Risk level: Medium

## Summary

The implementation follows the approved private-AppView architecture and the full Go test gate passes. Core persistence, owner scoping, tri-state save behavior, folder CRUD, canonical viewer-state hydration, current-policy filtering, descendant cleanup, concurrency control, route registration, and additive response fields are present and covered by substantial automated tests.

The change is not ready for handoff because the implemented evidence does not satisfy several approved Must-level test cases, and one new saved-cleanup error path includes the event post URI despite the explicit diagnostic-redaction requirement. The implementation record currently overstates completion by marking all planned Must tests complete. The next TDD pass should close the findings below without broadening product scope.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-001 | Important | Tests / Traceability | Session and device retention is tested through a new decision enum that has no runtime consumer. The test proves only that the helper returns the values encoded beside it; it does not run sign-out, logout-all, device removal, token expiry, reinstall, or account-switch lifecycle paths against Alice and Bob saved rows. This leaves AC-019, IT-009, and REG-005 without the required integration/regression shield. | FR-014, NFR-006, AC-019, AC-030, IT-009, REG-005; `appview/internal/api/saved_post.go:199-228`; `appview/internal/api/saved_post_lifecycle_test.go:8-36` | Add a real lifecycle/auth regression that seeds independent Alice/Bob saved state, executes the existing session/device/account lifecycle entry points, and proves those events retain both owners' membership-owned state while permanent Alice membership deletion removes only Alice's rows. Remove the test-only production enum/helper unless it becomes a justified runtime policy boundary. |
| IR-002 | Important | Tests | The approved pagination-density scenarios are only partially implemented. Saved-list tests create 103 static saves but do not add newer rows between page requests or include hidden rows in the traversal, and the folder test pages only three Alice folders rather than the required more-than-100 duplicate/dense fixture. Static ordering passes, but the keyset drift and dense folder-boundary cases in IT-003, IT-005, EC-013, EC-018, and TD-008 are not proven. | FR-007, FR-009, NFR-003, NFR-006, AC-011, AC-013, AC-027, AC-030, IT-003, IT-005, TD-008; `appview/internal/api/saved_post_store_test.go:205-275`; `appview/internal/api/saved_post_store_test.go:416-555` | Add red tests that insert newer saves after page one and prove fixed-scope traversal has the documented once-only keyset behavior, include policy-hidden candidates without cursor stalls or leakage, and page more than 100 duplicate/case-variant folders exactly once in `lower(name), id` order. |
| IR-003 | Important | Tests / Risk | IT-014 does not exercise both sort directions or the production saved-list SQL shape. It explains only simplified literal `DESC` queries with sequential scans disabled, while `ListSavedRefs` uses a parameterized scope `OR`, cursor comparator, and dynamic direction. The test therefore does not prove that the actual all/folder/unfiled queries—including oldest/backward traversal—use compatible bounded indexes. | FR-019, NFR-002, AC-026, IT-014; `appview/internal/api/saved_post_store.go:241-332`; `appview/internal/api/saved_post_query_plan_test.go:47-124` | Exercise the production query shape, or a single extracted query builder used by production and tests, for all/folder/unfiled scopes, cursor and first-page forms, and newest/oldest directions. Assert the intended index or compatible backward scan and retain the bounded shared viewer-state query assertion. |
| IR-004 | Important | Risk / Tests | Diagnostic redaction is incomplete. The new descendant-save cleanup error interpolates `ev.URI` into an error explicitly labeled as a save cleanup, contrary to NFR-001's prohibition on saved URIs in error text. UT-011/IT-013 cover one identity-resolution failure response and generic HTTP failure metrics, but do not capture success telemetry, store/indexer failures, logs, traces, wrapped database errors, or fixed operation/result/stage/error-class fields. | NFR-001, NFR-005, AC-025, AC-029, UT-011, IT-013, TD-010; `appview/internal/index/craftsky_post.go:743-746`; `appview/internal/api/saved_post_observability_test.go:33-100` | First add a failing sentinel test for the cleanup/error path, then remove private identifiers from saved-operation error text. Extend observability tests across representative successful and failed save/folder/store/indexer operations, capturing logs, metrics, traces, and wrapped errors; assert only bounded fields and request correlation remain. If generic HTTP telemetry is the intended NFR-005 implementation, document that deviation and prove both success and failure; otherwise add the planned bounded saved-operation instrumentation. |
| IR-005 | Important | Tests / Traceability | REG-007's migration pre-state is narrower than the approved fixture. The test preserves profiles, posts, and a synthetic sentinel, but it does not snapshot representative likes/reposts, mutes, blocks, notifications, and their data as required by REG-007 and TD-011. The migration SQL is narrow, but the claimed unrelated public/private schema regression test was not implemented as specified. | FR-019, NFR-006, AC-024, AC-030, REG-007, TD-011; `appview/internal/db/saved_posts_migration_test.go:14-29`; `appview/internal/db/saved_posts_migration_test.go:175-190` | Expand the version-current pre-feature fixture with representative unrelated public/private tables and rows, snapshot them, run up/down/up, and prove their schema/data are unchanged while only saved tables and indexes disappear and reappear. |

## Requirement And Test Traceability

- Requirements implemented: The production diff maps cleanly to the approved backend-only scope: reversible private tables; owner/post and owner/folder constraints; save/folder mutations; scoped keyset reads; canonical `viewerHasSaved` hydration through `EngagementSummaries`; current membership/moderation/block policy; transactional exact/descendant cleanup; and seven authenticated routes. No lexicon, Flutter, PDS-write, Tap-filter, notification, dependency, or config expansion was introduced.
- Tests implemented: Unit, handler, real-Postgres store, migration, policy/context, index lifecycle, concurrency, route-policy, privacy, query-plan, and canonical-surface tests exist and pass. The strongest implemented cases cover tri-state assignment, owner isolation, duplicate names, both saved-list ordering directions, reply-context deletion, author non-disclosure, and `-race` mutation outcomes.
- Unplanned behavior: `savedPostLifecycleEvent` and `savedPostLifecycleDeletes` are production declarations used only by their adjacent unit test; they do not participate in real lifecycle behavior.
- Remaining gaps: IR-001 through IR-005. `05-implementation-plan.md` should not claim all planned Must tests are passing until these cases are implemented or the approved test specification is explicitly revised.

## Test Evidence

- Commands reviewed:
  - `git status --short`
  - `git diff --stat`
  - `git diff --check`
  - `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test -race ./internal/api ./internal/db ./internal/index ./internal/routes -run 'TestSaved|SavedPost|CraftskyPost.*Delete|TestAddRoutes_AllV1PoliciesEnforcedThroughMux' -count=1`
  - `just test`
- Passing evidence:
  - `git diff --check` passed.
  - The focused uncached `-race` command passed for `internal/api`, `internal/db`, `internal/index`, and `internal/routes` after running with permission to reach local PostgreSQL.
  - `just test` passed across the complete AppView package set with `go test -race ./...` and compose PostgreSQL available.
  - The implementation record reports `just fmt` (`gofmt` plus `go vet ./...`) passed before this review; the review did not rerun a formatting command because this stage is source-read-only.
- Failing or skipped tests:
  - The first sandboxed focused run failed only because the sandbox denied TCP access to `localhost:5433`; the permitted rerun passed.
  - No implemented test is failing or skipped. Findings identify approved tests that are absent or materially narrower than specified.

## Risk Review

- Risk level: Medium.
- Risk notes: Core database ownership and mutation behavior is well constrained and currently green under `-race`. Remaining risk is concentrated in unverified session/account retention, cursor behavior under concurrent inserts and hidden candidates, production-query index use in both directions, saved-operation diagnostic redaction, and migration regression breadth.
- Approval notes: Changes required. No source fix was applied during review. No commit, push, or PR action was taken.

## UI Polish Recommendation

- Recommendation: Not needed.
- Reason: This implementation is AppView-only and contains no user-facing Flutter changes.
- Suggested polish notes: None.

## Handoff Back To TDD Builder

- Required fixes: Close IR-001 through IR-005 with focused red-green-refactor loops, then update `05-implementation-plan.md` to record the actual correction evidence.
- Suggested next failing test: Start with IR-004 by injecting a private URI sentinel into the saved-descendant cleanup failure path and asserting the returned/captured error, log, trace, and metric output contains no sentinel.
- Verification to rerun: Focused privacy/observability, lifecycle/auth, pagination, query-plan, and migration suites with real PostgreSQL and `-race`; route-policy tests; `git diff --check`; `just fmt`; and full `just test`.
