# Implementation Review: AppView Facet Endpoints And Plain Profile Bios

## Verdict
Status: Changes required
Reviewer: OpenAI gpt-5.5 implementation reviewer
Date: 2026-06-04
Risk level: Medium

## Summary

The implementation covers the main planned surface area: authenticated AppView `/v1/facets/*` handlers, a separate `atproto_identity_cache` table, mention/hashtag store logic, `identity-cache backfill`, profile-initialization cache upsert wiring, AppView-backed Flutter facet repositories, removal of profile `descriptionFacets`, and render-time plain-bio token parsing. Focused Go and Flutter tests reviewed during this stage pass, and the working tree was clean before the review artifact was written.

One blocking correctness issue remains in the identity-cache write path. `atproto_identity_cache.handle_lower` is globally unique, but `IdentityCacheStore.Upsert` only handles conflicts on `did`. If a stale cache row still owns a handle and exact resolution discovers that the handle now belongs to a different Craftsky DID, the upsert fails with a unique-constraint error instead of refreshing or invalidating the stale entry. That contradicts the stale-handle/exact-resolve correctness requirements and leaves manually typed mention resolution unable to recover from handle reassignment.

There is also a non-blocking API hardening gap: suggestion SQL treats `%` and `_` in `q` as SQL wildcard syntax rather than literal autocomplete input.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-001 | Important | Behavior / Risk | Identity-cache upsert cannot recover when a handle is reassigned to a different DID. The migration makes `handle_lower` unique, but `Upsert` uses `ON CONFLICT (did)` only. A stale row such as `(oldDid, alice.craftsky.social)` causes exact resolve for `alice.craftsky.social -> newDid` to fail with a unique-constraint error rather than refresh/invalidate the old cache row. The handler then reports `404 mention_not_found`, so valid manually typed mentions can fail after handle changes. | `01-requirements.md` FR-014, AC-013, AC-018, EC-005; `04-coding-plan.md` lines 130-134 and 195-210; `appview/migrations/000015_identity_handle_cache.up.sql` lines 3-8; `appview/internal/api/identity_cache_store.go` lines 74-88; `appview/internal/api/facet_store.go` lines 183-190 | Update the identity-cache upsert/refresh logic to handle handle reassignments atomically, without storing handles on `bluesky_profiles`. Add a store test where a stale cached handle belongs to `oldDid`, resolver returns `newDid`, and exact resolve updates the cache to `newDid` and returns success. |
| IR-002 | Suggestion | API / Risk | Suggestion queries are interpolated into `LIKE` patterns without escaping SQL wildcard characters. Authenticated requests such as `q=%` or `q=_` can match broad sets rather than literal query text. Limits cap the response size, but this weakens autocomplete semantics and increases enumeration/load risk. | `01-requirements.md` NFR-002 and §17 abuse cases; `appview/internal/api/facet_store.go` lines 91-94 and 137 | Escape `%`, `_`, and the escape character before using user input in `LIKE`, or explicitly validate suggestion query characters. Add focused tests for literal `%`/`_` behavior if changed. |

## Requirement And Test Traceability

- Requirements implemented: BR-001 through BR-003, FR-001 through FR-016, NFR-001 through NFR-003, RULE-001 through RULE-003 are represented in code and tests across AppView API/store/CLI/auth wiring and Flutter repository/profile/rich-text layers.
- Tests implemented: The implementation added or updated AppView handler/request/response/store/CLI/auth/route tests and Flutter repository/facet-generator/profile-bio/profile-edit/profile-API tests corresponding to the planned AT/UT/IT/REG coverage.
- Unplanned behavior: None identified beyond the API wildcard semantics called out in IR-002.
- Remaining gaps: IR-001 requires a new stale-handle reassignment test and implementation fix before approval.

## Test Evidence

- Commands reviewed:
  - `git status --short` and `git diff --stat` before review: clean working tree / no current diff.
  - `git log --oneline -10` and `git show --stat --name-status HEAD` for implementation commit `a099c53 feat: implement appview facet endpoints`.
  - `go test ./...` from `appview/`.
  - `flutter test test/shared/rich_text/facet_suggestion_repository_test.dart test/profile/widgets/profile_bio_test.dart test/shared/rich_text/facet_generator_test.dart` from `app/`.
- Passing evidence:
  - `go test ./...` passed for all AppView packages.
  - Focused Flutter rich-text/repository/profile-bio tests passed: `+11: All tests passed!`.
- Failing or skipped tests:
  - No failing tests observed during this review.
  - Manual checks MAN-001 through MAN-004 were documented as not run in `05-implementation-plan.md`.

## Risk Review

- Risk level: Medium.
- Risk notes: The implementation spans AppView API contracts, persistence, identity resolution, OAuth/profile initialization side effects, CLI operations, and user-facing Flutter composer/profile surfaces. The remaining blocker is limited to identity-cache consistency during handle reassignment, but it affects exact mention resolution correctness.
- Approval notes: Not ready to approve until IR-001 is fixed and covered by a focused test. IR-002 can be fixed in the same pass or tracked as a non-blocking hardening note if the team accepts the current constrained autocomplete input behavior.

## UI Polish Recommendation

- Recommendation: Not needed
- Reason: The UI changes reviewed are primarily behavioral simplifications (plain bio editor, render-time clickable bio ranges, AppView-backed autocomplete repositories). Existing focused widget tests show coherent rendering/styling for bio tokens, and no polish-only rough edge was identified.
- Suggested polish notes: None.

## Handoff Back To TDD Builder

- Required fixes:
  - Fix IR-001 by making identity-cache upsert/refresh handle stale handle ownership conflicts across DIDs and adding a regression test for handle reassignment.
- Suggested next failing test:
  - Add an AppView store test in `appview/internal/api/identity_cache_store_test.go` or `facet_store_test.go`: seed `atproto_identity_cache` with `oldDid -> alice.craftsky.social` older than 24 hours, seed `craftsky_profiles` with `newDid`, have the fake resolver return `alice.craftsky.social -> newDid` and canonical handle `alice.craftsky.social`, then assert `ResolveMention` succeeds and the cache row now belongs to `newDid` with no duplicate stale owner.
- Verification to rerun:
  - Focused AppView store test for the new failing case.
  - `go test ./internal/api -run 'TestFacetStoreResolveMention|TestIdentityCache' -count=1` from `appview/`.
  - `go test ./...` from `appview/`.
  - Relevant Flutter tests do not need rerun unless the fix changes Flutter-facing API behavior; otherwise rerun the focused Flutter command from this review for confidence.
