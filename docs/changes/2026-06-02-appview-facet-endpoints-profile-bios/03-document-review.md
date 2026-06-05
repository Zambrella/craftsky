# Document Review: AppView Facet Endpoints And Plain Profile Bios

## Verdict

Status: Approved with notes
Reviewer: OpenAI gpt-5.5 document reviewer
Date: 2026-06-04
Risk level: Medium

## Summary

The requirements and acceptance-test specification are consistent and ready for coding planning. The selected direction, Option A with dedicated `/v1/facets/*` endpoints plus a separate identity/handle cache, is reflected in the goals, non-goals, Must requirements, acceptance criteria, test matrix, and implementation handoff. A post-review manual update added the missing requirement that newly created/initialized Craftsky users are immediately upserted into the identity cache, so backfill is only needed for existing profiles. No blocking coverage or traceability gaps were identified.

This remains a medium-risk change because it crosses AppView API routes, persistence, identity resolution, Flutter data repositories, composer behavior, and profile bio edit/render behavior. Coding planning should preserve the documented boundaries: Craftsky-only mention suggestions/resolution, root-post-only hashtag counts, no profile `descriptionFacets`, and no lexicon changes.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-001 | Suggestion | Tests / API design | Over-limit suggestion requests are intentionally left as either clamped or rejected, while still requiring no more than 25 results. This is acceptable for review, but implementation planning must choose one behavior before writing final assertions. | `01-requirements.md` AC-014, EC-007; `02-acceptance-tests.md` GAP-001, UT-001 | Coding planner should pick clamp or validation error for `limit > 25` and make tests assert that exact behavior. |
| DR-002 | Suggestion | CLI / Operations | The identity-cache population path is required and tested, but the concrete command name and operator-facing defaults are not locked. This is non-blocking because the behavior is specified. | `01-requirements.md` FR-015, AC-019, §16; `02-acceptance-tests.md` IT-005, MAN-002, GAP-002 | Coding planner should name the bounded command/path and define limits/defaults without adding network work to SQL migrations. |
| DR-003 | Suggestion | UI / Parser behavior | Plain bio parsing deliberately targets supported token rules rather than full Bluesky parity. This is aligned with the requirements, but fixtures must be explicit to avoid accidental behavior drift. | `01-requirements.md` FR-011, NFR-003, EC-003, EC-004, RISK-004; `02-acceptance-tests.md` UT-009, GAP-003, TD-005 | Implementation should centralize or clearly mirror post facet token fixtures and document malformed/overlapping token expectations in tests. |
| DR-004 | Suggestion | Test clarity | Exact mention resolution tests cover both AppView endpoint behavior and Flutter facet-generator behavior. The distinction between endpoint `404 mention_not_found` and Flutter's “emit no facet but continue” behavior should stay explicit. | `01-requirements.md` FR-003, FR-004, FR-008, AC-006, AC-016; `02-acceptance-tests.md` AT-002, UT-004, UT-007, IT-003 | Coding planner should keep handler tests and Flutter generator tests separate enough that both contracts are independently verified. |
| DR-005 | Manual update | Identity cache / onboarding | Post-review feedback identified that bounded backfill covers existing Craftsky profiles but not the first-created/new-user path. | `01-requirements.md` FR-016, AC-021; `02-acceptance-tests.md` IT-009, REG-006 | Coding planner should wire identity-cache upsert into the AppView profile creation/initialization path and keep handles out of `bluesky_profiles`. |

## Resolution Notes

| Finding ID | Resolution |
|---|---|
| DR-001 | Resolved to reject `limit > 25` with `400 validation_error` using the standard AppView `{error, message, requestId}` envelope. Empty or whitespace-only `q` still returns `{items:[]}`. `01-requirements.md` and `02-acceptance-tests.md` were updated to assert this exact behavior. |
| DR-002 | Resolved to name the bounded identity-cache population command `cli identity-cache backfill`, run in dev as `docker compose exec appview /app/cli identity-cache backfill`. The command defaults to batch limit 100 and supports `--limit <n>`. `01-requirements.md` and `02-acceptance-tests.md` were updated accordingly. |
| DR-003 | Resolved to defer full Bluesky parser parity. Implementation should centralize or explicitly mirror Craftsky's existing post facet generator token rules and lock fixtures for dotted handles, Unicode hashtags, HTTP/S URLs, bare domains normalized to HTTPS, unsupported schemes, malformed handles/URLs, and overlapping URL fragment/hashtag cases. |
| DR-004 | Resolved by keeping two separate test contracts: AppView handler/store tests assert success objects or `404 mention_not_found`; Flutter facet-generation tests assert `mention_not_found` maps to no emitted mention facet while submission continues. |
| DR-005 | Resolved by adding FR-016 / AC-021 and IT-009 / REG-006: when the AppView creates or initializes a Craftsky profile for an authenticated user, it must resolve and upsert that user's current DID/handle into the separate identity cache. Existing-profile backfill remains `cli identity-cache backfill`. |

## Traceability Review

- Planning to requirements: The clarified decisions in `01-requirements.md` Q1-Q28 are carried into the recommended direction, goals/non-goals, requirements, edge cases, risks, and data/API/UI impacts. The workflow folder does not contain `00-initial-prompt.md`; the initial request, planning decisions, and post-review manual update are embedded in `01-requirements.md` §§1-5.
- Requirements to acceptance criteria: Every Must `BR`, `FR`, `NFR-001`, and `RULE` requirement in `01-requirements.md` §12 links to acceptance criteria in §13. Should requirements `NFR-002` and `NFR-003` also have acceptance criteria and tests.
- Acceptance criteria to tests: `02-acceptance-tests.md` §2 maps all requirement IDs to acceptance criteria and test IDs. Acceptance scenarios, unit tests, integration tests, regression tests, manual checks, and test data all reference requirement IDs and/or acceptance criteria.

## Coverage Review

- Must requirements covered: Yes. `BR-001` through `BR-003`, `FR-001` through `FR-016`, `NFR-001`, and `RULE-001` through `RULE-003` are covered by automated acceptance/unit/integration/regression tests, with manual smoke checks only as supplemental validation.
- Missing or weak coverage: No blocking gaps identified. DR-001 and DR-002 are now resolved design decisions in `02-acceptance-tests.md`; remaining non-blocking gaps/risks are disclosed as GAP-003 and GAP-004.
- Manual-only coverage: None for Must behavior. MAN-001 through MAN-004 are justified as smoke/operator checks after automated tests pass, especially for end-to-end UX, new-user identity-cache insertion, and the identity-cache backfill operation.

## Risk And Approval Review

- Risk level: Medium, matching both documents. The risk is appropriate because the change spans AppView API contracts, database schema/backfill, identity-cache freshness, Flutter repositories, composer UX, and profile bio rendering.
- Review requirement: Document review is recommended before implementation and has been completed by this artifact.
- Approval notes: Implementation may proceed. The non-blocking design choices and manual update called out in DR-001 through DR-005 have been resolved or explicitly deferred in the workflow documents.

## Coding Plan Readiness

- Ready for coding planning: Yes
- Recommended first step: Start with `IT-001` for authenticated `GET /v1/facets/mentions` returning `{items:[...]}` and enforcing session/device/error-envelope conventions, then continue with `UT-001` through `UT-004` for AppView request validation, response shaping, ranking, and exact resolution contracts.
- Blocking issues: None identified.

## Notes For Next Stage

- Treat `01-requirements.md`, `02-acceptance-tests.md`, and this review as source of truth.
- Preserve the documented scope boundaries: no lexicon changes, no profile `descriptionFacets`, no general account search, no non-Craftsky mention targets, and no PDS tokens in Flutter.
- Implement over-limit behavior as `400 validation_error` for `limit > 25`.
- Implement the bounded identity-cache backfill path as `cli identity-cache backfill`, default limit 100, configurable with `--limit <n>`, and keep network resolution out of SQL migrations.
- Implement identity-cache upsert on the AppView profile creation/initialization path for newly created/initialized Craftsky users; this complements backfill and must not store handles on `bluesky_profiles`.
- Keep AppView handler tests separate from Flutter facet-generator tests where endpoint errors map to client-side “no facet emitted” behavior.
