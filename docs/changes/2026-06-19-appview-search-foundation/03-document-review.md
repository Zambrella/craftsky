# Document Review: AppView Search Foundation

## Verdict

Status: Approved
Reviewer: OpenAI gpt-5.5 document reviewer
Date: 2026-06-19
Risk level: Medium

## Summary

The requirements and acceptance-test specification are consistent enough to proceed to coding planning. The requested AppView-only search foundation is reflected in the requirements, and the acceptance tests cover the Must business, functional, non-functional, and rule requirements with appropriate API, store/integration, unit, regression, and manual checks.

No blocking contradictions or missing Must-requirement coverage were identified. Follow-up edits to `01-requirements.md` and `02-acceptance-tests.md` addressed the non-blocking traceability and contract-precision notes. The original findings are retained below for audit context, with resolution status recorded after the findings table.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-001 | Important | Traceability | Several `Source` entries in the requirements table reference clarifying-question IDs that do not exist in Section 3 (`Q17+`, such as `Q18`, `Q19`, `Q20`, `Q21`, `Q22`, `Q23`, `Q25`, `Q26`, `Q27`, `Q28`, `Q29`, `Q31`). This weakens audit traceability from planning decisions to requirements, although the requirement text and tests are otherwise clear. | `01-requirements.md` §§3, 12; examples include `FR-007`, `FR-008`, `FR-009`, `FR-012`, `FR-013`, `FR-014`, `FR-017`, `FR-020`, `FR-021`, `FR-022`, `NFR-006`, `RULE-006` | Treat requirement text, acceptance criteria, and test IDs as source of truth for coding planning. If strict planning traceability is needed before external review, revise the `Source` column in `01-requirements.md` to point to existing decisions or remove stale IDs. |
| DR-002 | Important | Tests / API Contract | One acceptance scenario shows `GET /v1/search/hashtags/#Sock/posts`. A literal `#` in an HTTP URL begins a fragment and will not reach the server path unless URL-encoded. The requirements correctly allow stripping one leading `#`, and another criterion uses the canonical no-`#` path, but test implementation must not copy the raw-fragment example. | `01-requirements.md` `FR-002`, `FR-003`, `AC-021`; `02-acceptance-tests.md` `AT-002`, `UT-001`, `IT-002` | In coding planning/tests, use canonical path examples without `#` or explicitly encode the leading hash as `%23` when testing normalization. Keep the server-side normalization behavior for decoded path/query input. |
| DR-003 | Suggestion | Requirements / Tests | Recent-search duplicate display-label behavior is not fully explicit when the same normalized search is saved again with a different display label. The docs require storing display labels and refreshing duplicates, and `AC-027` says the display label is “preserved,” but it is not clear whether the original label or latest submitted label wins. | `01-requirements.md` `FR-014`, `FR-021`, `AC-027`; `02-acceptance-tests.md` `AT-004`, `UT-004`, `IT-008`, `GAP-003` | Coding plan should pin the intended duplicate-label rule before implementing recent-search normalization/store tests. If product intent is different from “preserve the existing stored label,” revise requirements first. |
| DR-004 | Suggestion | Risk / Non-functional Coverage | Search performance guardrails are covered by bounded limits, local indexed paths, and manual query-plan review, but no concrete data-volume or latency threshold is specified. This is already acknowledged as `GAP-001`. | `01-requirements.md` `NFR-002`, `NFR-005`, `NFR-006`, `RISK-003`; `02-acceptance-tests.md` `AT-011`, `MAN-001`, `GAP-001` | Coding plan should choose concrete default/max limits and identify indexes/generated columns. Add benchmark/load targets later when expected AppView scale is defined. |

## Resolution Status

| Finding ID | Status | Resolution |
|---|---|---|
| DR-001 | Addressed | `01-requirements.md` source references now point only to existing clarifying decisions, discovery/codebase findings, or review feedback. |
| DR-002 | Addressed | `02-acceptance-tests.md` now uses `GET /v1/search/hashtags/Sock/posts` in `AT-002`; future tests should use canonical no-`#` paths or URL-encoded `%23` when deliberately testing leading-hash normalization. |
| DR-003 | Addressed | `01-requirements.md` now states duplicate recent-search saves refresh `updatedAt` while preserving the existing stored display label, and `02-acceptance-tests.md` reflects that rule in `AT-004`, `UT-004`, and `IT-008`. |
| DR-004 | Addressed | `01-requirements.md` now defines v1 default/max limits, text/payload/filter bounds, expected response wrappers, and expected search-supporting index paths; `02-acceptance-tests.md` aligns `AT-011`, `MAN-001`, and `GAP-001`. |

## Traceability Review

- Planning to requirements: The initial request and recorded decisions are carried into goals, non-goals, requirements, risks, and assumptions. The chosen Option A AppView-only `/v1/search/*` approach is preserved. The stale/nonexistent `Q` references documented in `DR-001` have been corrected.
- Requirements to acceptance criteria: All Must `BR`, `FR`, `NFR`, and `RULE` entries link to at least one acceptance criterion. Acceptance criteria are externally verifiable through API behavior, persistence state, ordering, pagination, moderation filtering, and privacy checks.
- Acceptance criteria to tests: The coverage matrix maps all acceptance criteria to acceptance, unit, integration, regression, or manual checks. Manual checks are justified for query-plan/index review, popularity formula review, log redaction, and API contract review.

## Coverage Review

- Must requirements covered: Yes. `BR-001` through `BR-006`, `FR-001` through `FR-022`, Must `NFR-001` and `NFR-002`, and `RULE-001` through `RULE-006` all have acceptance criteria and test coverage.
- Missing or weak coverage: No blocking Must coverage gaps. Follow-up edits resolved the raw `#` hashtag path example, duplicate recent-search display-label rule, search-specific response wrapper names, and concrete v1 request bounds. Remaining performance scale work is limited to future benchmark/load targets once expected AppView data volume is defined.
- Manual-only coverage: Manual checks are reasonable for `MAN-001` query-plan/index review, `MAN-002` popularity formula review, `MAN-003` privacy/log redaction review, and `MAN-004` API contract review. These should be scheduled in the implementation plan rather than treated as optional cleanup.

## Risk And Approval Review

- Risk level: Medium.
- Review requirement: Review is appropriate because this slice adds several authenticated endpoints, private recent-search persistence, deterministic ranking, full-text/search-index decisions, and future Flutter-facing contracts.
- Approval notes: Approved to proceed to coding planning. The coding plan should use the documented canonical hashtag path behavior, recent-search duplicate label semantics, response examples, migration/index expectations, and concrete v1 bounds.

## Coding Plan Readiness

- Ready for coding planning: Yes.
- Recommended first step: Start with `IT-001` / `AT-001` route registration and auth/device enforcement for the `/v1/search/*` family, alongside validation-envelope scaffolding from `UT-002` and regression coverage from `REG-001`.
- Blocking issues: None.

## Notes For Next Stage

- Treat `01-requirements.md`, `02-acceptance-tests.md`, and this review as source of truth; do not infer Flutter UI work into this AppView-only slice.
- Use the documented response examples for hashtag metadata, top-hashtag groups, profile summaries, recent-search payloads, and post/project list wrappers.
- Use canonical no-`#` hashtag path examples or URL-encoded `%23` examples in tests; never use a raw `#` in request URLs.
- Centralize the popularity formula and cursor seek values early so chronological and popularity pagination share deterministic tie-breakers.
- Plan migrations deliberately: recent-search persistence is required; FTS/trigram/search-supporting indexes or generated columns should be decided with `MAN-001` in mind.
- Keep recent searches AppView-private, scoped by authenticated DID, hard-deleted, idempotent on not-owned/nonexistent delete, and excluded from PDS writes and verbose logs.
