# Document Review: AppView Project Posts

## Verdict
Status: Approved with notes
Reviewer: gpt-5.5 document-reviewer
Date: 2026-06-07
Risk level: High

## Summary
`01-requirements.md` and `02-acceptance-tests.md` are consistent and ready for coding planning after the 2026-06-07 clarification that historical migration backfill is out of scope. The selected Option B schema direction is carried through requirements, acceptance criteria, and tests; every Must business, functional, rule, and non-functional requirement has traceable acceptance coverage and at least one proposed automated test. No blocking contradictions or missing Must coverage were found.

The approval is with notes because the work remains high risk: the migration/schema path, unknown open-union project details, and cross-surface `PostResponse` hydration can each cause subtle data or API regressions. The coding plan should preserve the suggested first failing test and front-load those risks.

## Findings
| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-001 | Important | Risk | The unknown `details` open-union behavior is correctly identified as risky and covered by tests, but it is implementation-sensitive because typed generated unmarshalling may reject unknown variants before raw/common project fields are preserved. | `01-requirements.md` FR-005, EC-003, RISK-003; `02-acceptance-tests.md` AT-004, UT-003, IT-005, GAP-002 | Coding planner should schedule UT-003/IT-005 early and require a pause for requirements/design review if the current generated types cannot preserve common/raw project data for unknown details. |
| DR-002 | Important | Tests / Risk | Migration schema coverage is mandatory for readiness, but historical backfill is out of scope after clarification. The test spec notes uncertainty about an existing migration-chain harness; this is not a blocker because IT-001 and GAP-003 give a concrete fallback path. | `01-requirements.md` FR-001, FR-002, Data / Persistence Impact; `02-acceptance-tests.md` IT-001, GAP-003, Handoff recommended first failing test | Coding planner should make IT-001 the first failing test and explicitly choose the migration-chain or isolated SQL-test strategy before implementation starts. |
| DR-003 | Suggestion | Risk | Query-plan/index verification for NFR-003 is appropriately mixed automated/manual, but acceptance of this Should requirement depends on documenting deferred filters and index rationale. | `01-requirements.md` NFR-003, AC-013, RISK-004; `02-acceptance-tests.md` AT-013, MAN-002, GAP-001 | Coding planner should include an implementation-review checkpoint for schema indexes and any intentionally deferred project filters. |
| DR-004 | Suggestion | API compatibility | Requirements and tests imply project responses should be lexicon-shaped and preserve authored project metadata, while `craftsky_posts.tags` is normalized for search. This distinction should stay explicit during implementation to avoid leaking normalized search tags back as authored `project.common.tags`. | `01-requirements.md` FR-006, FR-009, AC-007, AC-011, Open Question 2; `02-acceptance-tests.md` AT-003, AT-007, AT-011, MAN-003 | Coding planner should prefer response hydration from raw project JSON or equivalent authored data, and keep normalized merged tags scoped to search/index columns. |

## Traceability Review
- Planning to requirements: The confirmed decision to use Option B is reflected in `01-requirements.md` Q1, Recommended Direction, FR-001, and Data / Persistence Impact. Scope boundaries from the initial request are preserved in goals and non-goals: AppView persistence, indexing/storage, create/read/list/count API behavior are in scope; Flutter UI, lexicon changes, search/discovery, and separate project collections are out of scope.
- Requirements to acceptance criteria: Every Must BR, FR, RULE, and NFR has at least one linked acceptance criterion in `01-requirements.md` section 12. The Should requirement NFR-003 is linked to AC-013 and treated as a mixed automated/manual performance-readiness check.
- Acceptance criteria to tests: Every AC-001 through AC-013 is represented in the `02-acceptance-tests.md` requirement coverage matrix and has one or more AT/UT/IT/MAN/REG tests. Tests consistently reference requirement IDs and acceptance criteria IDs.

## Coverage Review
- Must requirements covered: BR-001 through BR-002, FR-001 through FR-012, RULE-001 through RULE-003, and NFR-001, NFR-002, NFR-004 all have acceptance criteria and proposed tests.
- Missing or weak coverage: No missing Must coverage found. Weak/implementation-sensitive areas are already identified as GAP-002 for unknown open-union details and GAP-003 for migration-chain harness availability.
- Manual-only coverage: No Must requirement is manual-only. Manual checks support NFR-001, NFR-003, API shape review, and cross-surface hydration review; these are justified because they involve architecture/path inspection or query-plan behavior that may be brittle as pure automated assertions.

## Risk And Approval Review
- Risk level: High, matching both source documents. The risk comes from schema migration, Tap indexing convergence, PDS write payload validation, public `/v1/*` response shape changes, and response hydration across multiple existing post-shaped surfaces.
- Review requirement: Satisfied for document review. No blocking issues were found, but the coding plan should keep explicit risk checkpoints for migration schema, unknown details, indexes, and hydration coverage.
- Approval notes: Approved to proceed to coding planning with the findings above carried forward. Implementation completion should not be accepted until MAN-001 through MAN-004 or equivalent review steps are addressed.

## Coding Plan Readiness
- Ready for coding planning: Yes
- Recommended first step: Start with `IT-001` — a migration/schema/index test proving minimal base project flags, `craftsky_project_posts`, FK/cascade behavior, no required historical backfill, and supporting indexes.
- Blocking issues: None identified.

## Notes For Next Stage
- Treat `01-requirements.md`, `02-acceptance-tests.md`, and this review as source of truth.
- Keep the implementation sequence from `02-acceptance-tests.md` section 11: migration/schema first, then core indexing/materialization, then convergence/unknown details, then create API, then response hydration, then profile counts/projects endpoint, then interactions/regressions/manual checks.
- Front-load unknown details tests before broad API work so parser limitations are discovered early.
- Prefer shared post response hydration so single post reads, profile lists, timeline, comments/replies, notifications, and create responses cannot drift.
- Preserve architecture rules: writes go through PDS, reads come from AppView Postgres, `/v1/*` JSON stays camelCase, and public project data remains PDS-backed with AppView materialization as the read model.

## 2026-06-09 Clarification Addendum

The grill-me review after implementation changed the product rules for profile surfaces. The updated requirements and tests supersede any earlier review language that implied profile post lists or comments/replies should hydrate project metadata.

- Project posts are standalone only: no reply pointer and no quote embed.
- Profile `postCount` and profile post lists exclude projects.
- Profile `projectCount` and `/v1/profiles/{handleOrDid}/projects` include visible standalone project posts only.
- Timeline/feed surfaces still include project posts.
