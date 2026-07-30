# Document Review: Post And Content Languages

## Verdict

Status: Approved

Reviewer: Codex

Date: 2026-07-29

Risk level: High

## Summary

The revised requirements and acceptance-test specification are ready for coding-plan work. The documents preserve the confirmed Bluesky-inspired language model, private per-account AppView preferences, device-locale first-use initialisation, exact base-tag matching, and the complete browse/discovery versus deliberate/contextual visibility boundary.

All 33 Must requirements link to acceptance criteria and automated tests. All 28 acceptance criteria are represented in the test specification. The test design gives each of the eight filtered endpoints a database-backed verification path and independently protects direct posts, complete threads, notifications, Saved Posts, quoted context, and viewer-authored content.

The user chose to revise both source documents after the first review. DR-001 through DR-003 are now resolved: the TDD handoff starts with schema/persistence and names IT-008 as the first authoritative database-backed filter test; the preference routes and strict request behavior are fixed; and AT-015 is explicitly a composed Flutter flow using fakes rather than live end-to-end evidence. No blocking or non-blocking findings remain.

## Findings

Open findings: None identified. Resolved findings are retained below as the review audit trail.

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-001 | Important | Tests / Coding readiness | Resolved. The first implementation test is now migration/persistence case IT-028, and IT-008 is explicitly the first authoritative filtering test against executed Postgres SQL before limit/cursor. UT-003 is only a test-data truth table. | `01-requirements.md` section 23; `02-acceptance-tests.md` UT-003 and section 11 | None. Preserve this sequencing in the coding plan. |
| DR-002 | Important | Tests / Security | Resolved. The contract now fixes `GET`/`PUT /v1/languages/preferences` and `POST /v1/languages/preferences/initialize`, complete PUT semantics, absent-row behavior, strict JSON, and deterministic `400 invalid_request` rejection of account selectors or query parameters. | `01-requirements.md` FR-013, AC-016, AC-019, AC-020, AC-022, EC-023, sections 16–17; `02-acceptance-tests.md` UT-011, IT-002, IT-004, IT-026 | None. Implement the fixed contract without accepting an alternate account source. |
| DR-003 | Suggestion | Tests / Verification claims | Resolved. AT-015 is renamed and written as a composed Flutter client flow with fake repositories, while Go/Postgres and optional compose-stack evidence are reported separately. | `01-requirements.md` sections 2 and 23; `02-acceptance-tests.md` section 1, AT-015, GAP-003, section 11 | None. Keep verification reporting separated by layer. |

## Traceability Review

- Planning to requirements:
  - The recommended direction is preserved: App language is device-local; Primary and Content languages are private per-DID AppView preferences; post tags remain public PDS metadata.
  - The agreed filtering scope covers the home timeline, Projects browse, post/project/hashtag search, and other users' profile post/project/comment lists.
  - The agreed exceptions are explicit for direct posts, complete thread context, notifications, Saved Posts, quoted context, and viewer-authored posts.
  - Empty Content languages, untagged behavior, any-match multilingual posts, repost/quote semantics, composer reset behavior, device defaults, and exact base-tag matching are resolved.
  - The two remaining open questions are marked non-blocking future work.
- Requirements to acceptance criteria:
  - All 36 requirements link to acceptance criteria.
  - All 33 Must requirements have at least one externally verifiable acceptance criterion.
  - AC-001 through AC-028 are defined once with no undefined requirement references.
- Acceptance criteria to tests:
  - AC-001 through AC-028 all appear in the test specification.
  - Every Must requirement has automated coverage.
  - Unit, integration, acceptance, regression, manual, data, and gap IDs are defined without duplicates or unresolved references.
  - Regression and manual rows include both requirement IDs and acceptance-criteria IDs.

## Coverage Review

- Must requirements covered:
  - 33 of 33.
  - Business behavior, UI, API, indexing, persistence, security/privacy, pagination, account isolation, and exception boundaries all have automated paths.
- Missing or weak coverage:
  - None identified.
  - DR-001 through DR-003 were resolved without removing or renumbering requirement, acceptance, or test IDs.
- Manual-only coverage:
  - None of the Must behavior is manual-only.
  - Physical VoiceOver/TalkBack, hardware-keyboard, maximum-text-scale, narrow-screen, and representative query-performance checks supplement automated semantics, layout, integration, and query-plan tests.

## Risk And Approval Review

- Risk level:
  - High, unchanged.
  - The main risks are inconsistent eligibility across eight list endpoints, filtering after pagination, stale Flutter caches, preference/account leakage, and exceptions being applied too broadly or narrowly.
- Review requirement:
  - Formal document review was required and is now complete with an `Approved` verdict after re-reviewing both revised source documents.
  - A separate explicit implementation approval remains required after the coding plan.
- Approval notes:
  - The user approved progression from requirements to test design on 2026-07-29 and selected formal document review.
  - The requirements/test pair is approved for coding-plan work.
  - DR-001 through DR-003 are resolved and retained above as an audit trail.

## Coding Plan Readiness

- Ready for coding planning: Yes.
- Recommended first step:
  - Identify the next migration numbers and map the fixed preference contract plus shared language predicate into the existing storage, route-policy, and SQL query families.
  - Follow IT-028 with the private preference route/store cases, then establish IT-008 as the first authoritative language-filter slice before expanding the shared corpus.
- Blocking issues:
  - None.

## Notes For Next Stage

- Preserve stable requirement, acceptance, and test IDs in the coding plan.
- Implement the fixed `GET`/`PUT /v1/languages/preferences` and `POST /v1/languages/preferences/initialize` contract, including strict `400 invalid_request`, absent-row `404`, and initialise `200` behavior.
- Treat the AppView's stored Content languages as authoritative. Do not add `Accept-Language`, query parameters, or client-provided account identifiers as alternate filter sources.
- Keep language eligibility inside each candidate query before limit/cursor. A shared Go helper is useful only for test data or query construction if it cannot drift from the executed SQL.
- Design the shared visibility corpus once, then adapt it to timeline, Projects, post search, project search, hashtag results, profile posts, profile projects, and profile comments.
- Keep direct/thread, notification, Saved Posts, quote-context, and ownership exception tests separate so a broad shared predicate cannot silently capture them.
- Preserve existing surface order, cursor tie-breakers, moderation, relationship, membership, reply, and import policies.
- No Lexicon change is planned. If implementation discovery changes that conclusion, stop and use the repository's Lexicon-plus-ADR workflow before editing `lexicon/`.
- Verification claims must distinguish Flutter tests with fakes, Go/Postgres integration tests, and any optional compose-stack smoke test.
