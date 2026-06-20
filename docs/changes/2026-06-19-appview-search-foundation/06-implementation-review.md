# Implementation Review: AppView Search Foundation

## Verdict
Status: Approved with notes
Reviewer: OpenAI gpt-5.5 implementation reviewer
Date: 2026-06-20
Risk level: Medium

## Summary
The re-review confirms the previous blocking implementation-review findings were addressed in commit `841f906 fix: address appview search review findings`. Project popularity ordering, profile pagination, typed recent-search payload normalization, seeded store coverage, and FTS-backed keyword search now have implementation and focused tests. `just test` passed during re-review.

No remaining blocking issues were identified. A few non-blocking follow-ups are noted below for API polish/performance evidence before broader rollout.

## Findings
| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-001 | Suggestion | API Contract | Profile search rows carry avatar CID/MIME data, but `BuildProfileSearchSummary` does not populate the embedded `ProfileAccountSummary.Avatar` field. Existing profile summary conventions include `avatar` when available, and the coding-plan example includes it. This is additive and can be fixed without changing behavior. | `04-coding-plan.md` §5.2; `appview/internal/api/search_store.go` `BuildProfileSearchSummary`; `appview/internal/api/profile_response.go` `BuildProfileAccountSummary` | Consider synthesizing avatar URLs in search profile summaries before Flutter UI consumption. |
| IR-002 | Suggestion | Risk / Performance | `MAN-001` EXPLAIN review was attempted, but local migrations had not applied `000019_search_foundation`, so planner use of the added indexes was not confirmed. The SQL now uses FTS expressions aligned with the migration indexes, and tests pass. | `02-acceptance-tests.md` MAN-001; `05-implementation-plan.md` lines 188-194; `appview/migrations/000019_search_foundation.up.sql` | Re-run representative EXPLAIN checks after applying migration `000019_search_foundation` in local/dev database. |
| IR-003 | Suggestion | Risk / Performance | Search post response hydration still resolves handles per result through `HandleResolver`. This follows existing AppView post/timeline patterns and uses indigo's directory cache in production, but it is worth revisiting if search result sizes grow or cache misses cause external identity lookups. | `01-requirements.md` NFR-004 / AC-020; `04-coding-plan.md` §6.3; `appview/internal/api/search.go` `buildSearchPostResponses` | Consider a local/batched identity-cache lookup path for search result hydration in a future performance pass. |

## Requirement And Test Traceability
- Requirements implemented: Dedicated authenticated `/v1/search/*` routes; exact hashtag equality and canonical metadata; post/project keyword search through local FTS expressions; project filtering and browse-all semantics; chronological/popularity ordering with stable cursors; profile search with followed-first relevance and pagination; grouped top hashtags; recent-search save/list/delete with typed normalized payloads, de-duplication, pruning, DID scoping, and hard delete; moderation filtering before result limiting/ranking.
- Tests implemented: Route auth/device tests; request/normalization tests; cursor tests; ranking tests; response wrapper tests; seeded store tests for hashtag equality, profile pagination, keyword search, project filters, top hashtags, popularity, moderation; recent-search payload and lifecycle/privacy tests.
- Unplanned behavior: None identified. The work remains AppView-only with no Flutter UI, lexicon, or PDS-write changes.
- Remaining gaps: No blocking gaps. Non-blocking notes are listed in Findings.

## Test Evidence
- Commands reviewed:
  - `git status --short`: clean working tree before review artifact update.
  - `git diff --stat b273c58..HEAD` / `git diff --name-status b273c58..HEAD`: reviewed fix-pass changes.
  - `go test ./internal/api -run 'TestSearchStore_SearchProjectsPopularOrdersBrowseAllAndFilteredProjects|TestSearchStore_SearchProfilesPaginatesByRankTuple|TestSearchStore_SearchPostsAndProjectsUseFTSFields|TestSearchStore_RecentSearchLifecycleDedupesPrunesAndHardDeletes' -count=1`: passed.
  - `just test`: passed (`go test -race ./...` under `appview/`).
  - `05-implementation-plan.md` fix-pass evidence reports focused, package, broader, `just fmt`, and `just test` verification passed.
- Passing evidence:
  - Focused search fix tests passed during re-review.
  - Full AppView race suite passed during re-review.
- Failing or skipped tests:
  - No failing tests observed.
  - `MAN-001` EXPLAIN review remains to be rerun after local migrations are applied.

## Risk Review
- Risk level: Medium
- Risk notes: The previously high-risk gaps are now covered by focused automated tests. Remaining risk is primarily operational/performance evidence around query plans and optional API polish for profile avatar summaries.
- Approval notes: Approved with notes; non-blocking follow-ups can be handled before or alongside Flutter search UI integration.

## UI Polish Recommendation
- Recommendation: Not needed
- Reason: This stage changed AppView API, storage, migrations, and tests only; no Flutter UI was changed.
- Suggested polish notes: None.

## Handoff Back To TDD Builder
- Required fixes: None blocking.
- Suggested next failing test: If choosing to address notes now, start with a response-contract test proving profile search summaries include `avatar` when `avatar_cid`/`avatar_mime` are present.
- Verification to rerun: `go test ./internal/api -count=1`, `just test`, and `MAN-001` EXPLAIN checks after applying migration `000019_search_foundation`.
